# npm-go-check

A CLI tool to check the latest version of your dependencies in your `package.json` file.

## Installation

```bash
go install github.com/marcosvnmelo/npm-go-check@latest
```

## Usage

```bash
npm-go-check [options]
```

### Options

- `-p`, `--package <name>`: Check the latest version of a package, can be used multiple times
- `-u`, `--update`: Update all packages
- `-h`, `--help`: Show help message

## Example

This will check the latest version of all packages in your `package.json` file.
```bash
npm-go-check
```


This will check the latest version of `react` and `react-dom` packages.
```bash
npm-go-check -p react -p react-dom
```


This will update all packages in your `package.json` file.
```bash
npm-go-check -u
```


## Roadmap:

- [x] Add support for pnpm workspaces
- [ ] Follow .gitignore rules
- [ ] Implement the update option
- [ ] Add color support

## License

MIT
