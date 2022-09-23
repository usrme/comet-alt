# Comet Alt

An alternatively configured [Comet](https://github.com/liamg/comet) to suit my own preferences.

![comet-alt-demo](https://user-images.githubusercontent.com/5902545/191939072-0b91d07c-fd82-4d38-9a53-b546895df216.gif)

The way I've changed the original is for it to look and feel more like [Commitizen](https://github.com/commitizen-tools/commitizen) when invoking its sub-command `commit`. My only gripe was that the start-up speed was a tad on the slow side sometimes, given that it is Python, and that customizing the prompts wasn't as straight-forward as with Comet.

What I missed with Comet though was that Commitizen's `commit` by default keeps the values given for previous prompts on the screen, as seen in the [demo](https://github.com/commitizen-tools/commitizen/raw/master/docs/images/demo.gif), and that in and of itself was a major sticking point in continuing to use Comet.

Other minor changes include a fix to the prompt that asks for a commit message body that was misaligned and a check prior to running that confirms whether there are even any files that can be committed (i.e. are in the staging area).

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

## Acknowledgments

Couldn't have been possible without the work of [Liam Galvin](https://github.com/liamg).

## License

[MIT](/LICENSE)
