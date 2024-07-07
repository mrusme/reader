reader
------

*reader* is for your command line what the “readability” view is for modern
browsers: A lightweight tool offering better readability of web pages on the
CLI.

![reader](demo.gif)

`reader` parses a web page for its actual content and displays it in nicely
highlighted text on the command line. In addition, `reader` renders embedded
images from that page as colored block-renders on the terminal as well.


## Usage

```sh
reader https://xn--gckvb8fzb.com/superhighway84/
```

Don't render images:

```sh
reader --image-mode none https://xn--gckvb8fzb.com/superhighway84/
```

Output raw markdown, don't pretty print:

```sh
reader -o https://xn--gckvb8fzb.com/superhighway84/
```

Read from file:

```sh
reader ${HOME}/downloads/example.com.html
```

Read from stdin:

```sh
curl -o - https://superhighway84.com | reader -
```

Render images using the SIXEL graphics encoder:

```sh
reader --image-mode sixel https://xn--gckvb8fzb.com/travel-aruba/
```

![sixel](sixel.png)

More options:

```sh
reader -h
```

## Examples


### Using `reader` from within `w3m`

While on a web page in w3m, press `!` and enter the following:

```
reader $W3M_URL
```

This will open the current url with `reader`. `w3m` will wait for you to press
any key in order to resume browsing.

If you want to navigate through the page:

```
reader $W3M_URL | less -R
```


### Using `reader` from within `vim`/`neovim`

Add the following function/mapping to your `init.vim`:

```
function s:vertopen_url()
  normal! "uyiW
  let mycommand = "reader " . @u
  execute "vertical terminal " . mycommand
endfunction
noremap <Plug>vertopen_url : call <SID>vertopen_url()<CR>
nmap gx <Plug>vertopen_url
```

Open a document and place the cursor on a link, then press `g` followed by `x`.
Vim will open a new terminal and show you the output of `reader`.


## Contributing

[![GitHub repo Good Issues for newbies](https://img.shields.io/github/issues/mrusme/reader/good%20first%20issue?style=flat&logo=github&logoColor=green&label=Good%20First%20issues)](https://github.com/mrusme/reader/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22) [![GitHub Help Wanted issues](https://img.shields.io/github/issues/mrusme/reader/help%20wanted?style=flat&logo=github&logoColor=b545d1&label=%22Help%20Wanted%22%20issues)](https://github.com/mrusme/reader/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) [![GitHub Help Wanted PRs](https://img.shields.io/github/issues-pr/mrusme/reader/help%20wanted?style=flat&logo=github&logoColor=b545d1&label=%22Help%20Wanted%22%20PRs)](https://github.com/mrusme/reader/pulls?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22) [![GitHub repo Issues](https://img.shields.io/github/issues/mrusme/reader?style=flat&logo=github&logoColor=red&label=Issues)](https://github.com/mrusme/reader/issues?q=is%3Aopen)

👋 **Welcome, new contributors!**

Whether you're a seasoned developer or just getting started, your contributions are valuable to us. Don't hesitate to jump in, explore the project, and make an impact. To start contributing, please check out our [Contribution Guidelines](CONTRIBUTING.md). 
