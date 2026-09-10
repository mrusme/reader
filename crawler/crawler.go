package crawler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	scraper "github.com/memclutter/go-cloudflare-scraper"

	"github.com/go-shiori/go-readability"
)

const (
	DefaultUserAgent = "Mozilla/5.0 AppleWebKit/537.36 " +
		"(KHTML, like Gecko; compatible; " +
		"Googlebot/2.1; +http://www.google.com/bot.html)"

	DefaultTimeout = 30 * time.Second

	StdinLocation = "-"
)

const cycleTLSJa3 = "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-" +
	"49162-49161-49171-49172-51-57-47-53-10,0-23-65281-10-11-35-16-5-51-43-13-" +
	"45-28-21,29-23-24-25-256-257,0"

const cycleTLSUserAgent = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:87.0) " +
	"Gecko/20100101 Firefox/87.0"

var ErrNoLocation = errors.New("no source location set")

type StatusError struct {
	Location   string
	StatusCode int
}

func (e *StatusError) Error() string {
	status := strconv.Itoa(e.StatusCode)
	if text := http.StatusText(e.StatusCode); text != "" {
		status += " " + text
	}
	return fmt.Sprintf("%s: server returned HTTP %s", e.Location, status)
}

type Article struct {
	Title       string
	Author      string
	Excerpt     string
	SiteName    string
	Image       string
	ContentHTML string
	ContentText string
}

type Crawler struct {
	source      io.ReadCloser
	location    string
	locationURL *url.URL

	UserAgent string
	Timeout   time.Duration

	log *slog.Logger
}

func New(log *slog.Logger) *Crawler {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	return &Crawler{
		UserAgent: DefaultUserAgent,
		Timeout:   DefaultTimeout,
		log:       log,
	}
}

func (c *Crawler) Close() error {
	if c.source == nil {
		return nil
	}

	source := c.source
	c.source = nil
	return source.Close()
}

func (c *Crawler) SetLocation(location string) {
	c.location = location
	c.locationURL = nil

	if location == StdinLocation {
		return
	}

	if locationURL, err := url.Parse(location); err == nil {
		c.locationURL = locationURL
	}
}

func (c *Crawler) Source() io.ReadCloser {
	return c.source
}

func (c *Crawler) GetReadable(useCycleTLS bool) (Article, error) {
	if err := c.FromAuto(useCycleTLS); err != nil {
		return Article{}, err
	}

	article, err := readability.FromReader(c.source, c.locationURL)
	if err != nil {
		return Article{}, err
	}

	return Article{
		Title:       article.Title,
		Author:      article.Byline,
		Excerpt:     article.Excerpt,
		SiteName:    article.SiteName,
		Image:       article.Image,
		ContentHTML: article.Content,
		ContentText: article.TextContent,
	}, nil
}

func (c *Crawler) FromAuto(useCycleTLS bool) error {
	switch {
	case c.location == "":
		return ErrNoLocation
	case c.location == StdinLocation:
		return c.FromStdin()
	case c.isHTTP():
		if useCycleTLS {
			return c.FromHTTPCycleTLS()
		}
		return c.FromHTTP()
	default:
		return c.FromFile()
	}
}

func (c *Crawler) isHTTP() bool {
	u := c.locationURL
	if u == nil || u.Host == "" {
		return false
	}

	return u.Scheme == "http" || u.Scheme == "https"
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
		Timeout:   c.Timeout,
	}

	req, err := http.NewRequest(http.MethodGet, c.location, nil)
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

	c.log.Debug("Crawler.FromHTTP",
		"location", c.location,
		"userAgent", c.UserAgent,
	)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if !isSuccess(resp.StatusCode) {
		resp.Body.Close()
		return &StatusError{Location: c.location, StatusCode: resp.StatusCode}
	}

	c.log.Debug("Crawler.FromHTTP",
		"status", resp.StatusCode,
		"contentType", resp.Header.Get("Content-Type"),
	)

	if err := c.Close(); err != nil {
		resp.Body.Close()
		return err
	}

	c.source = resp.Body
	return nil
}

func (c *Crawler) FromHTTPCycleTLS() error {
	client := cycletls.Init()
	defer client.Close()

	c.log.Debug("Crawler.FromHTTPCycleTLS",
		"location", c.location,
		"userAgent", cycleTLSUserAgent,
	)

	resp, err := client.Do(c.location, cycletls.Options{
		Ja3:       cycleTLSJa3,
		UserAgent: cycleTLSUserAgent,
		Proxy:     c.proxyFromEnvironment(),
		Timeout:   c.timeoutSeconds(),
	}, http.MethodGet)
	if err != nil {
		return err
	}

	if !isSuccess(resp.Status) {
		return &StatusError{Location: c.location, StatusCode: resp.Status}
	}

	c.log.Debug("Crawler.FromHTTPCycleTLS",
		"status", resp.Status,
		"finalUrl", resp.FinalUrl,
	)

	if err := c.Close(); err != nil {
		return err
	}

	c.source = io.NopCloser(strings.NewReader(resp.Body))
	return nil
}

func (c *Crawler) FromFile() error {
	file, err := os.Open(c.location)
	if err != nil {
		return err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	if info.IsDir() {
		file.Close()
		return fmt.Errorf("%s: is a directory", c.location)
	}

	c.log.Debug("Crawler.FromFile",
		"location", c.location,
		"size", info.Size(),
	)

	if err := c.Close(); err != nil {
		file.Close()
		return err
	}

	c.source = file
	return nil
}

func (c *Crawler) FromStdin() error {
	c.log.Debug("Crawler.FromStdin")

	if err := c.Close(); err != nil {
		return err
	}

	c.source = io.NopCloser(os.Stdin)
	return nil
}

func (c *Crawler) proxyFromEnvironment() string {
	if c.locationURL == nil {
		return ""
	}

	proxyURL, err := http.ProxyFromEnvironment(&http.Request{
		URL: c.locationURL,
	})
	if err != nil || proxyURL == nil {
		return ""
	}

	return proxyURL.String()
}

func (c *Crawler) timeoutSeconds() int {
	if c.Timeout <= 0 {
		return 0
	}

	seconds := int(c.Timeout.Round(time.Second) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

func isSuccess(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}
