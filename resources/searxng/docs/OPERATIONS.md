# SearXNG operations

SearXNG is a `docker-service` resource. Its one supported runtime is the
single image declared in `resource.json`; Docker must be installed and running
before lifecycle operations. Redis, Docker Compose, direct Docker commands,
and shell resource scripts are not supported operator paths.

Use the shared control plane for lifecycle and logs:

```bash
vrooli resource install searxng
resource-searxng start
resource-searxng status
resource-searxng logs
resource-searxng stop
```

The manifest declares two independent durable mounts: `RESOURCE_CONFIG_DIR`
maps to `/etc/searxng`, and `RESOURCE_DATA_DIR` maps to `/var/cache/searxng`.
Do not put settings in the cache directory.

`engine-health --json` is the resource diagnostic. It performs a bounded JSON
canary query and reports `healthy`, `degraded`, or `critical`; it does not
replace the normal `/stats` container health check.

```bash
resource-searxng engine-health --json
```

The resource intentionally has no terminal search, batch, lucky, headlines,
benchmark, engine-edit, or container-exec command. Consumers should call the
HTTP JSON interface directly, for example `GET /search?q=<query>&format=json`.
