# SearXNG resource

SearXNG is Vrooli's privacy-conscious local metasearch resource. It is a
manifest-declared managed service composed from a checksum-pinned portable
Python runtime, locked wheels, and a reviewed SearXNG source tree. The native
supervisor exposes an HTTP endpoint on port 8280 with no external runtime
daemon required.

`resource.json` is the runtime authority. The shared Go control plane owns
install, start, stop, restart, status, and logs; the resource Go CLI provides
only safe configuration and engine-level diagnostics.

```bash
vrooli resource install searxng
resource-searxng config-apply
resource-searxng start
resource-searxng engine-health --json
```

Configuration and cache are separate durable mounts:

- `RESOURCE_CONFIG_DIR` → the durable SearXNG settings directory
- `RESOURCE_DATA_DIR` → the regenerable SearXNG cache directory

`config-apply` imports and backs up existing settings, preserves unknown
upstream YAML and existing secrets, and redacts secrets from output. The
consumer interface is HTTP JSON (`/search?format=json`), documented in
[docs/API.md](docs/API.md). Compose, Redis, shell scripts, and terminal search
convenience commands are deliberately unsupported.

See [operations](docs/OPERATIONS.md) and [configuration](docs/CONFIGURATION.md)
for the operator contract.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
