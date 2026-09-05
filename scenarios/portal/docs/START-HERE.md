# Start Here — Portal

Portal is no longer a template placeholder. It is the Vrooli ecosystem front
door: chat workspace, readiness registry, passive search loop, API, UI, and
CLI.

## Initialization Protocol

Start by verifying the scenario through lifecycle commands:

```bash
make setup
make start
vrooli scenario status portal
```

When you need the full scenario gate, use test-genie:

```bash
vrooli scenario test portal
test-genie runs wait --json portal 20260706-000000-example
```

Do not run `api/portal-api`, UI dev scripts, or CLI binaries directly for
lifecycle work. The Vrooli lifecycle owns process names, ports, health, and
cleanup.

## Architecture Rules

- API behavior lives in domain packages under `api/internal/<domain>/` and
  transport packages under `api/handlers/<domain>/`.
- Wire contracts live in `packages/proto/schemas/portal/v1/**`.
- UI behavior lives in `ui/src/features/**` with generated proto clients in
  `ui/src/api/**`.
- CLI commands live in `cli/domains/**` and are declared in
  `cli/manifest.json`.
- SQLite schema is declarative and domain-owned. Do not add migration ladders.
- Optional dependencies must fail soft and feed the readiness registry.

## Portal Domains

Read [`concepts/DOMAINS.md`](concepts/DOMAINS.md) before moving code. Current
domains are:

- health: lifecycle readiness and dependency health.
- chat: grouped conversations and message-tree persistence.
- message/completion: user messages, branch operations, LLM/agent streaming,
  usage capture, and search attachment events.
- integrations: readiness registry, rolling stats, behavior-mode policy, and
  overrides.
- search: Portal-mediated search-hub suggestions and passive attachment
  hydration.

## Working Sequence

1. Read `PRD.md` and `requirements/`.
2. Read the relevant concept doc: `DOMAINS.md`, `DATA.md`, `FLOWS.md`, or
   `INTEGRATIONS.md`.
3. Update proto first for wire-shape changes.
4. Implement API behavior and tests.
5. Update CLI/UI thin surfaces.
6. Run focused verification, then `cli-health`, `proto-health`, `ui-health`,
   and the relevant test-genie phase.
7. Append durable progress to `docs/internal/PROGRESS.md`.

## Cross-References

- [`../README.md`](../README.md) — operator overview
- [`../PRD.md`](../PRD.md) — operational targets
- [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) — system map
- [`reference/cli-commands.md`](reference/cli-commands.md) — CLI surface
- [`internal/SEAMS.md`](internal/SEAMS.md) — seams and fakes
