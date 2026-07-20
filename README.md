# k3d-router

[![CI](https://github.com/foundry23/k3d-router/actions/workflows/ci.yml/badge.svg)](https://github.com/foundry23/k3d-router/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/foundry23/k3d-router)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

k3d-router is a reverse proxy that lets multiple k3d clusters share port `:80`/`:443`, preventing port collisions and bloat on your local machines.

All routing is auto-discovered by configured `Host` manifests, routing all traffic to the correct cluster, allowing standard HTTP and HTTPs ports to be used across multiple projects without having to remember or configure ports.

Running `up` / `reload` scans all running k3d clusters and configures a traefik container with the routing table.

## Requirements

- A docker API engine
- One (or more) k3d clusters
- [Emissary-ingress](https://github.com/emissary-ingress/emissary) Host manifests (see [Host sources](#host-sources))

## Install

```bash
go install github.com/foundry23/k3d-router@latest
```

## Usage

```bash
k3d-router up         # start the router + generate routing
k3d-router reload     # re-scan after adding/removing a cluster or Host CRD
k3d-router watch      # keep routing in sync as clusters start/stop (Ctrl-C to stop)
k3d-router status     # container state, attached networks, current routes
k3d-router logs       # follow Traefik logs
k3d-router down       # stop and remove the router
```

`watch` reacts to Docker container events — a cluster's loadbalancer starting or dying. Host CRD changes *inside* a running cluster aren't observed; run `reload` for those.

## Configuration

Every option is a flag with an env-var fallback (see `k3d-router --help`):

- `--image` / `K3D_ROUTER_IMAGE` (default `traefik:v3`)
- `--http-port` / `K3D_ROUTER_HTTP_PORT` (default `80`), `--https-port` / `K3D_ROUTER_HTTPS_PORT` (default `443`)
- `--home` / `K3D_ROUTER_HOME` (default `~/.config/k3d-router`)
- `--log-level` / `K3D_ROUTER_LOG_LEVEL` (default `info`; use `debug` to trace internal steps)

Command output goes to stdout; leveled diagnostics go to stderr.

## Host sources

Currently, only [Emissary-ingress](https://github.com/emissary-ingress/emissary) Host manifests are supported, if you need another mechanism please [open an issue](https://github.com/foundry23/k3d-router/issues).

## Development

```bash
just build            # go build -o k3d-router .
just fmt              # goimports + gofumpt
just check            # go vet + go test
just test             # go test ./...
just test-integration # integration tests (needs Docker; k3d tier needs a cluster)
just run up           # go run . up
```

`just --list` shows every recipe. CI runs build, vet, race tests, formatting, linting (golangci-lint), and an integration job against a real k3d cluster.

## Contributing

Contributions are welcome:

1. **Open an issue first** for anything non-trivial — bugs, new [host sources](#host-sources), or behaviour changes — so the approach can be agreed before you build it.
2. Fork and branch, and add tests where it makes sense.
3. Run `just check` and `just test-integration` before opening a PR.
4. Keep commits focused and the CI green.

AI-assisted contributions are welcome, but code **must** be reviewed by a human before being submitted.

Any contributions that have not obviously been reviewed by a human may be closed without comment or justification.

## License

[MIT](LICENSE).
