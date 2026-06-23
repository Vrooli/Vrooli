# Network Manager

Local-first internet management, DNS filtering, network diagnostics, and safe optimization for home and small-office networks.

## What You Get

- React/Vite UI, Go API, and Go CLI scaffold generated from the `react-vite` template.
- `vrooli-default` design contract.
- PRD defining Network Manager as a greenfield scenario.
- Requirements registry covering health snapshots, AdGuard Home management, DNS filtering, device inventory, optimization experiments, adapter capabilities, Home Automation integration, and privacy defaults.
- Documentation for architecture, domains, data, flows, integrations, deployment, operations, security, performance, monetization, and testing.

Product implementation has not started. Generated example code remains only as scaffold until the first Network Manager domain is built.

## Documentation Map

| Need | Start Here |
|---|---|
| Product contract | [`PRD.md`](PRD.md) |
| Requirements | [`requirements/README.md`](requirements/README.md) |
| First-session guide | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Data and privacy | [`docs/concepts/DATA.md`](docs/concepts/DATA.md), [`docs/internal/SECURITY.md`](docs/internal/SECURITY.md) |
| Integrations | [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Operations | [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md) |
| Commercial framing | [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md) |

## Customize Safely

- Keep API business logic in the API; UI and CLI call API contracts.
- Use capability-reported adapters for host, resolver, router, and manual operations.
- Do not add router writes before approval and rollback workflows exist.
- Do not expose DNS query-level metadata without explicit retention and visibility settings.
- Use Scenario Dependency Analyzer for third-party dependency work.
- Build the first real domain beside the generated example, then remove the example with the detemplate flow.

## Running The Scaffold

```bash
cd scenarios/network-manager
make setup
make start
make status
```

Run tests with:

```bash
make test
```
