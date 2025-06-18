## reader

[![Static Badge](https://img.shields.io/badge/Donate-Support_this_Project-orange?style=for-the-badge&logo=buymeacoffee&logoColor=%23ffffff&labelColor=%23333&link=https%3A%2F%2Fxn--gckvb8fzb.com%2Fsupport%2F)](https://xn--gckvb8fzb.com/support/) [![Static Badge](https://img.shields.io/badge/Join_on_Matrix-green?style=for-the-badge&logo=element&logoColor=%23ffffff&label=Chat&labelColor=%23333&color=%230DBD8B&link=https%3A%2F%2Fmatrix.to%2F%23%2F%2521PHlbgZTdrhjkCJrfVY%253Amatrix.org)](https://matrix.to/#/%21PHlbgZTdrhjkCJrfVY%3Amatrix.org)

_reader_ is for your command line what the “readability” view is for modern
browsers: A lightweight tool offering better readability of web pages on the
CLI.

![reader](demo.gif)

`reader` parses a web page (or an EML file) for its actual content and displays
it in nicely highlighted text on the command line. In addition, `reader` renders
embedded images from that page as colored block-renders on the terminal as well.

## Installation

```
go install github.com/mrusme/reader@latest
```

If the above fails, then the following should work:

```
git clone https://github.com/mrusme/reader.git
cd reader
go install
```

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

Render EML file:

```sh
reader --eml -i none my-email-file.eml
```

Output EML file raw:

```sh
reader --eml --raw my-email-file.eml
```

Parse an attribute out of a series of `*.eml` files using
[pup](https://github.com/ericchiang/pup):

```sh
$ /bin/ls -1 ./*@mail.uber.com.eml \
  | while read mail; do reader --raw --eml "$mail" \
    | pup 'td.Uber18_text_p1 span.Uber18_text_p2 text{}'; done
$3.80
$13.72
$17.90
$5.87
$15.90
$24.40
$23.00
$35.00
$27.19
$4.54
$5.07
$8.54
$2.60
$19.81
$25.61
$30.00
€5.90
$4.68
...
```

So let's say you want a `|` delimited CSV with all your Uber payments (based on 
the mails you received from Uber) you could do:

```sh
/bin/ls -1 ./*@mail.uber.com.eml \
  | while read mail; do reader --raw --eml "$mail" \
    | pup 'span.Uber18_text_p2, span.Uber18_text_p1 json{}' \
    | jq -r '"| \(.[1].text) | \(.[0].text) |"'; done > ./uber.csv
```

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
