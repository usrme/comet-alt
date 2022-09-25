# Comet Alt

An alternatively configured [Comet](https://github.com/liamg/comet) to suit my own preferences.

![comet-alt-demo](https://user-images.githubusercontent.com/5902545/192158617-2feef11e-1915-445a-92d8-1bab342c3da9.gif)

The way I've changed the original is for it to look and feel more like [Commitizen](https://github.com/commitizen-tools/commitizen) when invoking its sub-command `commit`. My only gripe was that the start-up speed was a tad on the slow side sometimes, given that it is Python, and that customizing the prompts wasn't as straight-forward as with Comet.

What I missed with Comet though was that Commitizen's `commit` by default keeps the values given for previous prompts on the screen, as seen in the [demo](https://github.com/commitizen-tools/commitizen/raw/master/docs/images/demo.gif), and that in and of itself was a major sticking point in continuing to use Comet.

Other minor changes include a fix to the prompt that asks for a commit message body that was misaligned and a check prior to running that confirms whether there are even any files that can be committed (i.e. are in the staging area). More improvements have been made in terms of customizing the character input limits for the scope, message, or setting a total one in general and having a visible character count for all limit types.

## Installation

- using `go install`:

```bash
go install github.com/usrme/comet-alt@latest
```

- download a binary from the [releases](https://github.com/usrme/comet-alt/releases) page

- build it yourself (requires Go 1.17+):

```bash
git clone https://github.com/usrme/comet-alt.git
cd comet-alt
go build
```

## Removal

```bash
rm -f "${GOPATH}/bin/comet-alt"
rm -rf "${GOPATH}/pkg/mod/github.com/usrme/comet-alt*"
```

## Usage

There is an additional `comet.json` file that includes the prefixes and descriptions that I most prefer myself, which can be added to either the root of a repository or to one's home directory as `.comet.json`. Omitting this means that the same defaults are used as in the original.

### Setting character limits

- To adjust the character limit of the scope, add the key `scopeInputCharLimit` into the `.comet.json` file with the desired limit
  - Omitting the key uses a default value of 16 characters
- To adjust the character limit of the message, add the key `commitInputCharLimit` into the `.comet.json` file with the desired limit
  - Omitting the key uses a default value of 100 characters
- To adjust the total limit of characters in the *resulting* commit message, add the key `totalInputCharLimit` into the `.comet.json` file with the desired limit
  - Adding this key overrides scope- and message-specific limits

## Acknowledgments

Couldn't have been possible without the work of [Liam Galvin](https://github.com/liamg).

## License

[MIT](/LICENSE)
