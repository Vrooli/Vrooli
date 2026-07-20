# SearXNG resource

SearXNG is Vrooli's privacy-conscious local metasearch resource. It is an
intentional `docker-service`: one manifest-declared SearXNG container exposes
an HTTP endpoint on port 8280. Docker is therefore an explicit host
requirement on Linux, macOS, and Windows.

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

- `RESOURCE_CONFIG_DIR` → `/etc/searxng` (contains `settings.yml`)
- `RESOURCE_DATA_DIR` → `/var/cache/searxng`

`config-apply` imports and backs up existing settings, preserves unknown
upstream YAML and existing secrets, and redacts secrets from output. The
consumer interface is HTTP JSON (`/search?format=json`), documented in
[docs/API.md](docs/API.md). Compose, Redis, shell scripts, and terminal search
convenience commands are deliberately unsupported.

See [operations](docs/OPERATIONS.md) and [configuration](docs/CONFIGURATION.md)
for the operator contract.
