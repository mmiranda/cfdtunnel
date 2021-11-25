[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org)
[![codecov](https://codecov.io/gh/mmiranda/cfdtunnel/branch/main/graph/badge.svg?token=HAUMRJQ4OX)](https://codecov.io/gh/mmiranda/cfdtunnel)
[![https://goreportcard.com/report/github.com/mmiranda/cfdtunnel](https://goreportcard.com/badge/github.com/mmiranda/cfdtunnel)](https://goreportcard.com/report/github.com/mmiranda/markdown-index)
![[Test](https://github.com/mmiranda/cfdtunnel/actions/workflows/test-coverage.yml)](https://github.com/mmiranda/cfdtunnel/actions/workflows/test-coverage.yml/badge.svg)


# Cloudflared Tunnel Wrapper
**cfdtunnel** is a wrapper for [cloudflared](https://github.com/cloudflare/cloudflared) `access` tunnel, designed to access multiple tunnels without having to worry about your `cloudflared` process.


## Installation

The easiest way to install it is using Homebrew:

```bash
brew tap mmiranda/apps
brew install cfdtunnel
```

If you prefer, you also can download the latest binary on the [release section](https://github.com/mmiranda/cfdtunnel/releases)

## How does it work?

Basically this tool takes care of the `cloudflared` process initialization for you.

1. Runs cloudflared based on you config ini file
2. Runs the command you want
3. Kills the cloudflared proccess at the end


You can use any command on top of *cfdtunnel*:

### Kubectl
```bash
cfdtunnel --profile my-profile1 kubectl get namespaces
```
### K9S
```bash
cfdtunnel --profile my-profile1 k9s
```

### Configuration

Configuration is really simple, you just need to create your profiles in `~/.cfdtunnel/config`

Example:
```ini
[my-profile1]
host = https://kubernetes.foo.bar.com

[my-profile2]
host = sql.foo.bar.com
port = 1234
```

Defining a port is not required, if you don't specify, *cfdtunnel* will launch the tunnel using the ~~most random~~ port **5555**

## Contributing
Contributions, issues, and feature requests are welcome!

Give a ⭐️ if you like this project!

## License
[MIT](https://choosealicense.com/licenses/mit/)
