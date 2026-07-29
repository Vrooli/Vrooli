# Landing Page Business Suite

Landing Page Business Suite is a SaaS landing-page and business-operations
scenario. It provides configurable public landing pages, A/B experimentation,
analytics, Stripe billing, subscriptions, credits, download delivery, and an
admin portal.

## Run locally

From this directory, use the Vrooli lifecycle commands:

```bash
make start
make logs
make stop
```

Vrooli assigns the API and UI ports during startup. Check the scenario status
or logs for the active endpoints, including the API health endpoint.

## Documentation

- [Quick start](docs/QUICKSTART.md)
- [Architecture](docs/concepts/ARCHITECTURE.md)
- [Security](docs/reference/SECURITY.md)
- [Configuration guide](docs/guides/CONFIGURATION_GUIDE.md)
- [API overview](docs/reference/api/README.md)
- [Documentation manifest](docs/manifest.json)

## Development checks

Run the comprehensive, server-owned suite with:

```bash
vrooli scenario test landing-page-business-suite
```

For focused Go checks, run `go test ./... -timeout=600s` from `api/` or
`cli/`. Do not start scenario binaries directly; use the lifecycle commands
above.
