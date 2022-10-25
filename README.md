## devproxy

[![Build Status](https://github.com/romnn/devproxy/workflows/test/badge.svg)](https://github.com/romnn/devproxy/actions)
[![GitHub](https://img.shields.io/github/license/romnn/devproxy)](https://github.com/romnn/devproxy)

`devproxy` is a tiny command line reverse proxy for local development 
written in Go.

### Installation

```bash
go get -u github.com/romnn/devproxy/cmd/devproxy
go install github.com/romnn/devproxy/cmd/devproxy
```

### Example

Assume you have two services running:
- an API service running at http://localhost:8090
- a frontend service running at http://localhost:8080

When trying to access the API service from the frontend service directly,
you probably run into issues with CORS.
Using `devproxy`, you can start a reverse proxy that proxies
multiple services using rules of the format `/path/prefix@http://service-url`.

```bash
devproxy start --port 5000 /api@http://127.0.0.1:8090 /@http://127.0.0.1:8080
```

In this example, the reverse proxy will proxy both services on port 5000 such that all paths with prefix `/api` will be proxied to the API service 
at `http://127.0.0.1:8090` and path without a prefix (prefix `/`) will be
proxied to the frontend service at `/@http://127.0.0.1:8080`.

#### Development

##### Tooling

Before you get started, make sure you have installed the following tools:

    $ python3 -m pip install pre-commit bump2version
    $ go install golang.org/x/tools/cmd/goimports
    $ go install golang.org/x/lint/golint
    $ go install github.com/fzipp/gocyclo

Please check that all checks pass:

```bash
pre-commit run --all-files
```
