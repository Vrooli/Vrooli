# Device Control

Device Control is the governed control plane for owner-authorized physical and
virtual devices. It discovers devices, describes the capabilities of each
transport, acquires leases, executes bounded actions, and retains auditable
evidence. Phones are one supported device class alongside televisions,
speakers, lights, sensors, cameras, locks, and other endpoints.

The device identity is durable across endpoint changes and may have multiple
transports. Capabilities belong to the device/transport pair, so a Google TV
can expose ADB screenshots and semantic input while its Android TV Remote
transport exposes directional input and media control without a screen.

## Device authentication

Device authentication is a governed capability for owner-trusted phones. Use
`device-control auth create` to save reference-only policy, use
`device-control auth update` for metadata changes, pipe a credential to
`device-control auth provision <profile-id>`, acquire a device lease, and run
`device-control auth unlock` to obtain a typed, live-verified result. Delete
the authority-held value with `device-control auth delete-credential` before
revoking a temporary profile. The credential authority owns the secret;
device-control owns binding, method policy, bounded attempts, and postcondition
verification.

Promoted wireless Android devices retain their stable device id across service
restarts. Recover a stale saved endpoint with
`device-control device reconnect <device-id> --json`; the reconnect verifies
the original hardware serial before persisting a discovered TLS endpoint.

Drive owner-authorized physical and virtual devices through pluggable control
strategies, capability-derived actions, reusable automation flows, and
agent-driven goal completion.

## What You Get

- LAN discovery and pairing for Google TV transports.
- Durable multi-transport identities with evidence-bearing claims.
- Lease-owned actuation, state reads, events, audit records, and bounded flows.

## Quick use

```bash
device-control device list
device-control device screenshot "SmartTV 4K"
device-control device open "SmartTV 4K" --app netflix
```

One call each: the verb wakes the screen, does the work, verifies it, and on
failure prints the exact next command. See [`docs/START-HERE.md`](docs/START-HERE.md).

## Layout

- Go API (`api/`), Go CLI (`cli/`, surface declared in `cli/manifest.json`), React UI (`ui/`).
- Proto contracts in `packages/proto/schemas/device-control`; Connect-RPC is the
  default transport, REST only for the documented third-party shapes.
- Skills (`skills/`) and governed programs (`.vrooli/program-runtime/`).
- Executable behavior specifications: `api/internal/control/features/*.feature`.

## Running The Scenario

```bash
make setup   # build API, UI and CLI
make start   # start; make logs / make stop
make test    # vrooli scenario test device-control
```

## Documentation Map

| Need | Start Here |
|---|---|
| Do something with a device | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at this scenario's root |
| Author UX intent | [`experience/README.md`](experience/README.md) |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| Understand the architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Map product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Track workflows, data, and integrations | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md), [`docs/concepts/DATA.md`](docs/concepts/DATA.md), [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Capture monetization and launch strategy | [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md), [`docs/business/GO-TO-MARKET.md`](docs/business/GO-TO-MARKET.md) |
| Prepare deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Write tests | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Add or update seams/fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Configure env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Add API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Add CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |

## Working Rules

1. Capabilities are probed, never inferred; keep fail-closed dispositions, leases and evidence retention intact.
2. Record non-obvious choices in `docs/internal/DECISIONS.md` and keep `docs/manifest.json` truthful.
3. Express new behavior as Gherkin scenarios in `api/internal/control/features/` where it reads as a user journey.
4. Read root `DESIGN.md` before UI work; reuse `react-component-library` assets.
5. Keep changes inside this scenario unless a control-plane or shared piece must change, and say why.
