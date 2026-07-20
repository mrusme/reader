package crawler

import (
	"bufio"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	scraper "github.com/memclutter/go-cloudflare-scraper"

	"github.com/go-shiori/go-readability"
)

const defaultUserAgent = "Mozilla/5.0 AppleWebKit/537.36 " +
	"(KHTML, like Gecko; compatible; " +
	"Googlebot/2.1; +http://www.google.com/bot.html)"

const cycleTLSJa3 = "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-" +
	"49162-49161-49171-49172-51-57-47-53-10,0-23-65281-10-11-35-16-5-51-43-13-" +
	"45-28-21,29-23-24-25-256-257,0"

const cycleTLSUserAgent = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:87.0) " +
	"Gecko/20100101 Firefox/87.0"

type ItemCrawled struct {
	Title       string
	Author      string
	Excerpt     string
	SiteName    string
	Image       string
	ContentHtml string
	ContentText string
}

type Crawler struct {
	source            io.ReadCloser
	sourceLocation    string
	sourceLocationUrl *url.URL

	UserAgent string

	logger *zap.Logger
}

func New(logger *zap.Logger) *Crawler {
	return &Crawler{
		UserAgent: defaultUserAgent,
		logger:    logger,
	}
}

func (c *Crawler) Close() {
	if c.source != nil {
		c.source.Close()
		c.source = nil
	}
}

func (c *Crawler) SetLocation(sourceLocation string) {
	c.sourceLocation = sourceLocation
	c.sourceLocationUrl = nil

	if sourceLocation == "-" {
		return
	}

	if urlUrl, err := url.Parse(sourceLocation); err == nil {
		c.sourceLocationUrl = urlUrl
	}
}

func (c *Crawler) GetSource() io.ReadCloser {
	return c.source
}

func (c *Crawler) GetReadable(useCycleTLS bool) (ItemCrawled, error) {
	if err := c.FromAuto(useCycleTLS); err != nil {
		return ItemCrawled{}, err
	}

	article, err := readability.FromReader(c.source, c.sourceLocationUrl)
	if err != nil {
		return ItemCrawled{}, err
	}

	item := ItemCrawled{
		Title:       article.Title,
		Author:      article.Byline,
		Excerpt:     article.Excerpt,
		SiteName:    article.SiteName,
		Image:       article.Image,
		ContentHtml: article.Content,
		ContentText: article.TextContent,
	}

	return item, nil
}

func (c *Crawler) FromAuto(useCycleTLS bool) error {
	var err error

	scheme := ""
	if c.sourceLocationUrl != nil {
		scheme = c.sourceLocationUrl.Scheme
	}

	switch {
	case c.sourceLocation == "-":
		err = c.FromStdin()
	case scheme == "http", scheme == "https":
		if useCycleTLS {
			err = c.FromHTTPCycleTLS()
		} else {
			err = c.FromHTTP()
		}
	default:
		err = c.FromFile()
	}

	return err
}

func (c *Crawler) FromHTTP() error {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return err
	}

	transport, err := scraper.NewTransport(http.DefaultTransport)
	if err != nil {
		return err
	}

	client := &http.Client{
		Jar:       jar,
		Transport: transport,
	}

	req, err := http.NewRequest("GET", c.sourceLocation, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent",
		c.UserAgent)
	req.Header.Set("Accept",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,"+
			"image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language",
		"en-US,en;q=0.5")
	req.Header.Set("DNT",
		"1")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	c.Close()
	c.source = resp.Body
	return nil
}

func (c *Crawler) FromHTTPCycleTLS() error {
	client := cycletls.Init()

	resp, err := client.Do(c.sourceLocation, cycletls.Options{
		Body:      "",
		Ja3:       cycleTLSJa3,
		UserAgent: cycleTLSUserAgent,
	}, "GET")
	if err != nil {
		return err
	}

	c.Close()
	c.source = io.NopCloser(strings.NewReader(resp.Body))
	return nil
}

func (c *Crawler) FromFile() error {
	file, err := os.Open(c.sourceLocation)
	if err != nil {
		return err
	}

	c.Close()
	c.source = file
	return nil
}

func (c *Crawler) FromStdin() error {
	c.Close()
	c.source = io.NopCloser(bufio.NewReader(os.Stdin))
	return nil
}
