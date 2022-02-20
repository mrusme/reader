reader
------

A minimal command line reader offering better readability of web pages on the
CLI.

![reader](demo.gif)

`reader` parses a web page for its actual content and displays it in nicely
highlighted text on the command line. In addition, `reader` renders embedded
images from that page as colored block-renders on the terminal as well.


## Usage

```sh
reader https://xn--gckvb8fzb.com/superhighway84/
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
reader $W3M_URL | less
```

