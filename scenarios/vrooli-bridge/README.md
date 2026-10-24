# Vrooli Bridge

Owner control plane for a trusted fleet of Vrooli nodes across Linux, macOS,
and Windows.

Bridge owns durable node identity, pairing, presence, scope-gated dispatch, run
history, provisioning gates, and the operator surfaces that use them:

- Go API (`api/`)
- React + TypeScript + Vite UI (`ui/`)
- CLI wrapper (`cli/`)
- Lifecycle + health wiring (`.vrooli/service.json`)
- Requirements registry + progress log (`requirements/`, `docs/internal/PROGRESS.md`)
- `agent/` is a standalone CGO-free Go module that cross-compiles for all six
  supported OS/architecture pairs. It resolves installed scenario CLIs by
  absolute path and executes typed argv without a shell.
- `packages/api-core/nodereach` is the shared typed client for downstream
  scenarios; they do not carry private Bridge wire adapters.

> **Start here:** open [`docs/START-HERE.md`](docs/START-HERE.md). It
> owns the first-session initialization protocol — charter, requirements,
> domain map, design language, placeholder replacement, and first real
> vertical slice. Run `make orient` for a machine-readable gate status.

## Onboarding

Use [`bootstrap/README.md`](bootstrap/README.md) for a fresh machine. The
manual code path remains available when the control plane is not discoverable.
On a trusted LAN, the agent can discover `_vrooli-bridge._tcp.local`; the
advertised control-plane URL is carried in DNS-SD TXT data.

The terminal-free request/approve path is used when an installed agent has a
control-plane URL but no node id. It submits public facts, displays three
key-derived confirmation words, and waits. The owner sees the same words,
chooses a catalog-derived permission preset (read-only by default), and must
confirm the match before approval. The agent persists its assigned id and the
pinned control-plane public key.

Presence survives a Bridge restart as durable registry state, while online and
dispatchable status require a fresh heartbeat and live channel. A stale row is
not presented as ready.

## Product surfaces

- Go API (`api/`), Go CLI (`cli/`), and React/Vite UI (`ui/`) coordinated
  through generated proto contracts for pairing, registry, dispatch, runs,
  machine lineage, readiness, and cross-platform gates.
- Lifecycle metadata, Makefile entrypoints, health checks, endpoint
  metadata, testing config, and CLI install wiring.
- The UI includes the fleet dashboard, pending-pairing approvals, permission
  presets, onboarding, run history, and trust/grant controls.
- SQLite by default. Add external resources to `.vrooli/service.json`
  only when this scenario actually needs them.
- The standalone `agent/` module cross-compiles for the supported OS/architecture
  pairs and resolves installed scenario CLIs by absolute path without a shell.
- Downstream scenarios use [`packages/api-core/nodereach`](../../packages/api-core/nodereach/)
  for Bridge discovery, authentication, typed relay arguments, durable runs,
  and interactive sessions. Browser payloads never contain Bridge owner or
  node credentials.

## Extension rules

The durable product surfaces are the Bridge domains and their generated
contracts. Keep these invariants when extending them:

- i18n and accessibility wiring, including stable selectors.
- Existing design tokens and feature-folder boundaries.
- The proto → API → CLI → UI vertical-slice shape.
- Owner/operator separation: Bridge holds credentials and nodes execute only
  admitted, typed verbs.

**Connect-RPC is the default transport.** Every domain endpoint goes
through a proto service and generated Connect handlers/clients. If
you find yourself writing `Path: "/api/v1/..."` as a literal string in
an `EndpointDescriptor`, stop — use a proto service method instead.
Codegen rejects literal Paths that lack an explicit `RESTException`
tag; the four allowed REST reasons (multipart upload, webhook
receiver, third-party shape, ops probe) are enumerated in
`api/internal/module/module.go`. The notes attachments endpoint is
the worked REST example.

[`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) documents the current
ownership boundaries and transport seams.

## Running and validating

```bash
vrooli scenario start vrooli-bridge
vrooli scenario test vrooli-bridge
vrooli scenario stop vrooli-bridge
```

Use [`bootstrap/README.md`](bootstrap/README.md) for a fresh node. The
installed agent can discover the control plane over DNS-SD, request pairing,
show key-derived confirmation words, and persist its assigned identity. An
owner approves the request with a catalog permission preset; read-only is the
default.

## Documentation Map

| Need | Start Here |
|---|---|
| Initialize after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Establish UI design language | `DESIGN.md` at this scenario's root |
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

1. **Read [`docs/START-HERE.md`](docs/START-HERE.md) first.** It owns the first implementation workflow.
2. **Run `make orient`** as a progress check — it reports initialization gates from `.vrooli/orientation.json`.
3. **Update `PRD.md` and `requirements/`** before feature work. Operational targets drive code + tests.
4. **Read root `DESIGN.md` before UI work.** Tokens, motion, and status semantics are binding; specific component lists in the design are illustrative — implement everything your scenario actually needs.
5. **Update `docs/concepts/DOMAINS.md`** before adding product code.
6. **Keep `docs/manifest.json` accurate.** Durable docs should be registered there with a truthful maturity value.
7. **Append progress entries** to `docs/internal/PROGRESS.md` whenever you land work.
8. **Add resources** to `.vrooli/service.json` only when needed; this scenario ships with no resource dependencies (SQLite is in-process).
9. **Keep boundaries**: only edit within this scenario's directory.

## pnpm Everywhere

This scenario assumes pnpm. If you run another package manager, convert
lockfiles yourself before committing. Scripts use `pnpm` directly (no
`npm` fallbacks) to reduce drift.

## Need Inspiration?

Open `scenarios/browser-automation-studio/` to see the same template
shape taken to completion.
