# DNSAQ — DNS Advanced Query Tool

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)

DNSAQ is a high-performance, bandwidth-efficient DNS resolution and subdomain enumeration tool written in Go. It is designed for security professionals, bug-bounty hunters, and penetration testers who need reliable DNS reconnaissance with minimal bandwidth consumption.

---

## Features

* ⚡ **High-Speed Resolution**: Asynchronous DNS resolution with configurable rate limiting.
* 🌐 **Bandwidth-Efficient**: Optimized to minimize internet usage while maintaining performance.
* 🎯 **Subdomain Bruteforcing**: Wordlist-based subdomain enumeration.
* 🛡️ **Wildcard Detection**: Automatic detection and filtering of wildcard DNS responses.
* 📁 **Resolver Files**: Support for custom resolver lists from files.
* 💾 **Output Options**: Save results to file while still printing to stdout.
* 🔌 **Tool Integration**: Seamless piping with other reconnaissance tools.
* 📊 **Verbose Mode**: Detailed logging for debugging and analysis.

---

## Installation

### Prerequisites

* Go 1.21 or later
* Git (for source installation)

### Quick Install (via `go install`)

```bash
# Install directly with Go
go install github.com/cristophercervantes/dnsaq@latest

# Ensure the Go bin directory is in your PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

### From Source

```bash
# Clone the repository
git clone https://github.com/cristophercervantes/dnsaq.git
cd dnsaq

# Build and install
go install

# Or build a binary
go build -o dnsaq main.go
```

### Pre-built Binaries

Download the latest release for your platform from the Releases page and add it to your `PATH`.

---

## Usage

```
dnsaq [options]
```

### Options

| Flag           |                                  Description | Default                 |
| -------------- | -------------------------------------------: | ----------------------- |
| `-d`           |                        Domain to brute-force | (none)                  |
| `-w`           |                     Wordlist for brute-force | (none)                  |
| `-r`           | File containing DNS resolvers (one per line) | (none)                  |
| `-resolvers`   |        Comma-separated list of DNS resolvers | `8.8.8.8:53,1.1.1.1:53` |
| `-rate`        |                           Queries per second | `10`                    |
| `-t`           |                           Timeout in seconds | `2`                     |
| `-no-wildcard` |                   Disable wildcard detection | `false`                 |
| `-v`           |                               Verbose output | `false`                 |
| `-o`           |                  Output file to save results | (none)                  |
| `-version`     |                     Show version information | (none)                  |

---

## Examples

### Subdomain Bruteforcing

```bash
# Basic bruteforcing
dnsaq -d example.com -w wordlist.txt -r resolvers.txt

# With output file and rate limiting
dnsaq -d example.com -w wordlist.txt -r resolvers.txt -o results.txt -rate 5

# With verbose output and disabled wildcard detection
dnsaq -d example.com -w wordlist.txt -r resolvers.txt -v -no-wildcard
```

### Domain Resolution

```bash
# Resolve domains from a file
cat domains.txt | dnsaq -r resolvers.txt

# Resolve domains and save to file
cat domains.txt | dnsaq -r resolvers.txt -o resolved.txt

# With custom resolvers and timeout
cat domains.txt | dnsaq -resolvers "9.9.9.9:53,208.67.222.222:53" -t 5
```

### Integration with Other Tools

```bash
# With subfinder
subfinder -d example.com -silent | dnsaq -r resolvers.txt -o subfinder_results.txt

# With amass
amass enum -passive -d example.com | dnsaq -r resolvers.txt -v

# With assetfinder
assetfinder example.com | dnsaq -r resolvers.txt -o assetfinder_results.txt

# Chaining multiple tools
subfinder -d example.com | dnsaq -r resolvers.txt | httpx -silent
```

---

## Resolver Files

Create a text file with one DNS resolver per line. Comments starting with `#` are supported.

**Example `resolvers.txt`:**

```
# Public DNS resolvers
8.8.8.8
1.1.1.1
9.9.9.9
208.67.222.222

# Additional resolvers
64.6.64.6
77.88.8.8
```

---

## Performance Tuning

### Rate Limiting

Adjust the `-rate` parameter based on your network capacity and target environment:

```bash
# Conservative rate (home networks)
dnsaq -d example.com -w wordlist.txt -rate 5

# Moderate rate (business networks)
dnsaq -d example.com -w wordlist.txt -rate 10-20

# Aggressive rate (dedicated scanning environments)
dnsaq -d example.com -w wordlist.txt -rate 50
```

### Timeout Settings

Adjust timeout based on network reliability:

```bash
# Fast timeout (reliable networks)
dnsaq -d example.com -w wordlist.txt -t 2

# Longer timeout (unreliable networks)
dnsaq -d example.com -w wordlist.txt -t 5
```

---

## Output Format

The tool outputs results in the format:

```
subdomain.example.com [192.168.1.1, 192.168.1.2]
```

When using the `-o` flag, results are simultaneously displayed on stdout and saved to the specified file.

---

## Building from Source

### Prerequisites

* Go 1.21 or later
* Git

### Steps

```bash
# Clone the repository
git clone https://github.com/yourusername/dnsaq.git
cd dnsaq

# Install dependencies
go mod download

# Build the binary
go build -o dnsaq main.go

# (Optional) Install to your GOPATH
go install
```

### Cross-Compilation

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o dnsaq-linux-amd64 main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o dnsaq-windows-amd64.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o dnsaq-darwin-amd64 main.go
```

---

## Contributing

We welcome contributions!

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m "Add some amazing feature"`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please follow the existing code style and include tests for new functionality where appropriate.

---

## License

This project is licensed under the **MIT License**. See the `LICENSE` file for details.

---

## Acknowledgments

Inspired by tools like `massdns`, `puredns`, and `shuffledns`.
Built with the `miekg/dns` Go library.

---

## Support

If you have any questions or issues:

* Check existing issues on GitHub
* Create a new issue
* Contact: [your-email@example.com](mailto:your-email@example.com)

---

## Disclaimer

This tool is intended for **educational and authorized testing purposes only**. Always ensure you have proper authorization before scanning any network or domain.
