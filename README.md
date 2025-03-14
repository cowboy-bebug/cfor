# cfor

![cfor](./cfor.gif)

`cfor` (command for) is an AI-powered terminal assistant that helps you find and
execute commands without digging through man pages. Simply ask what you want to
do in natural language, and `cfor` will suggest relevant commands with brief
explanations.

The name reflects its usage pattern: `cfor [what you want to do]` is like asking
"what's the command for [task]?" - making it intuitive to use for finding the
right commands for your tasks.

## Features

- **Natural Language Queries**: Ask for commands in plain English
- **Smart Command Suggestions**: Get multiple command variations with inline
  comments
- **Interactive Selection**: Choose the right command from a list of suggestions
- **Terminal Integration**: Selected commands are automatically inserted into
  your terminal prompt
- **OpenAI Integration**: Powered by OpenAI's language models (supports multiple
  models)

## Installation

### Using Homebrew (macOS and Linux)

```bash
brew install cowboy-bebug/tap/cfor
```

### From Source

Requirements:

- Go 1.24 or later

```bash
git clone https://github.com/cowboy-bebug/cfor.git && cd cfor
make install
```

## Usage

```bash
cfor [question]
```

### Examples

```bash
cfor "listing directories with timestamps"
cfor "installing a new package for a pnpm workspace"
cfor "applying terraform changes to a specific resource"
cfor "running tests in a go project"
```

## Configuration

`cfor` requires an OpenAI API key to function. You can set it up in one of two
ways:

```bash
# Use a general OpenAI API key
export OPENAI_API_KEY="sk-..."

# Or use a dedicated key for cfor (takes precedence)
export CFOR_OPENAI_API_KEY="sk-..."
```

### Model Selection

By default, `cfor` uses `gpt-4o`. You can switch to other supported models:

```bash
export CFOR_OPENAI_MODEL="gpt-4o"
```

## Building from Source

```bash
make build    # Build the binary
make install  # Install to your GOPATH
make clean    # Clean build artifacts
```

## Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

## License

MIT
