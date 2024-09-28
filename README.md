# garchetype

> This is a **Production Draft** ¯\\_(ツ)_/¯ , it has no tests, no CI, no docs,
> no examples, etc. It's just a proof of concept whit good intentions and charm.

This is an scaffold tool for adding functionality to Go projects.

It makes use of [go-archetype](https://github.com/rantav/go-archetype) for handling the templates.

## Requirements

- Go 1.23+

## Install

```shell
go install github.com/diegosz/garchetype@latest
```

Recommended: Pin the `garchetype` tool to your project using `bingo`. Do it via:

```shell
bingo get -l github.com/diegosz/garchetype@latest
```

## Usage

Add a feature using an archetype:

```shell
./garchetype add -f example-app -- --salutation 'Hi, punk!'
🌱 Adding 'example-app' feature using 'hello-world' archetype.
📦 Using transformation file: xarchetype_godev_default/archetypes/hello-world/transformations-default.yaml
🎉 Feature 'example-app' added.
```

## TODO

- [ ] Add tests.
- [ ] Add documentation.
- [ ] Add examples.
- [ ] Add continuous integration.

## Credits

- [go-archetype](https://github.com/rantav/go-archetype) by [rantav](https://github.com/rantav), many thanks!
- [skaphos](https://github.com/bmfs/skaphos) by [bmfs](https://github.com/bmfs), many thanks!
