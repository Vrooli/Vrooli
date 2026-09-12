# Template Manager

Template Manager is the single accountable owner of Vrooli's **template domain**.
Standing up a new scenario used to be a loose collection of `vrooli` CLI verbs with
no persistent evidence and no owner for quality. Template Manager makes that work
**measured, repeatable, and improvable**: scenario templates, design kits, and
resource templates each have one owner, a durable store of validation evidence,
and a programmatic surface other scenarios can call.

It owns the full lifecycle — generation, orientation, and detemplating of scenarios;
shallow and deep template validation; drift and version-lag detection; the inherited
template-debt ledger; design-kit and resource-template inspection; factory
documentation; and a recurring deep-validation monitor with federated measures.

See [`PRD.md`](PRD.md) for the full product framing and operational targets, and
[`docs/factory/TEMPLATE-FACTORY-GUIDE.md`](docs/factory/TEMPLATE-FACTORY-GUIDE.md)
for the maintainer's operating loop.

## What You Get

- A **template registry** cataloging scenario templates, design kits, and resource
  templates with version and manifest metadata, backed by a migration-owned SQLite
  store under the api-core storage resolver.
- **Validation runs** (shallow and deep), **drift snapshots**, version-lag tracking,
  and a deduplicated **debt ledger** — all persisted and queryable through API and CLI.
- A **test-genie provider** for the `templates` phase, plus a **recurring monitor**
  that runs scheduled deep validation for active scenario templates under capacity-aware,
  serialized execution.
- An **orientation guidance** surface that returns the next incomplete orientation gate
  and its contract as structured data, so execution agents know what work a freshly
  generated scenario still owes.
- **Factory documentation** for template maintenance, the generation contract, and the
  migration protocol — owned here and indexed into search-hub.
- Coordinated surfaces over generated proto contracts:
  - **API (`api/`)** — Go + Connect-RPC over the contracts in
    `packages/proto/schemas/template-manager`. Storage is SQLite via api-core/storage,
    rooted at the durable home store. Serves on **port 17093**.
  - **CLI (`cli/`)** — the primary agent- and operator-facing surface: a thin,
    manifest-driven wrapper (`cli/manifest.json` is the contract) that speaks
    proto-JSON to the API. Groups are listed below.
  - **UI (`ui/`)** — a React + Vite operations console for fleet standing, validation
    history, debt trends, registry state, drift, and monitor status. Serves on
    **port 21598**.

## CLI Command Groups

The CLI is the contract in [`cli/manifest.json`](cli/manifest.json); run
`template-manager help` or `template-manager <group> help` for the live surface.
All read commands accept `--json` for machine-readable output.

| Group | What it does | Example |
|---|---|---|
| **lifecycle** | Generate, orient, and detemplate scenarios | `template-manager lifecycle generate --template react-vite --id my-scenario` |
| **template** | Validate templates, report drift, clean validation workspaces | `template-manager template drift my-scenario` |
| **design** | List, show, and validate scenario design kits | `template-manager design list --json` |
| **guidance** | Return the next incomplete orientation gate for a scenario | `template-manager guidance next my-scenario` |
| **measures** | List and run declared template-manager measures | `template-manager measures list` |
| **monitor** | Inspect recurring deep-validation monitor state | `template-manager monitor status --json` |
| **registry** | Inspect governed template records | `template-manager registry list` |
| **resource-template** | List, show, validate, and generate from resource templates | `template-manager resource-template generate <name>` |
| **runs** | Run and inspect validation runs and drift snapshots | `template-manager runs run my-scenario --deep` |
| **debt** | Inspect inherited template-debt entries | `template-manager debt list --json` |

Top-level utility commands: `status` (API health; `--json` for the raw payload),
`configure` (view/update CLI settings), `version`, and the three template-lifecycle
verbs (`generate`, `orient`, `detemplate`) exposed both directly and under `lifecycle`.

> **Note:** the template-domain verbs used to live on the `vrooli` CLI as
> `vrooli scenario generate|orient|detemplate|template|design`. Those were removed
> when the domain moved here — use the `template-manager` CLI instead.

## The Operating Loops

- **Validation loop.** `runs run` executes shallow or deep validation against a
  scenario template and persists the result; `runs list`/`runs show` read the history;
  `template validate` and `template drift` cover ad-hoc checks. A template stays
  quarantined until its current deep-validation evidence is clean.
- **Drift loop.** `runs drift-record` runs fleet drift and persists a snapshot;
  `runs drift` / `template drift` surface which scenarios have diverged from their
  source template (manifest or content drift, plus version lag).
- **Debt loop.** Defects surfaced during validation are recorded as deduplicated
  entries in the debt ledger (`debt list` / `debt show`), giving inherited template
  debt one accountable, trend-able home instead of scattered TODOs.
- **Monitor loop.** A recurring, capacity-aware scheduler runs deep validation for
  active scenario templates in serialized order and persists scheduler-attributed
  results; `monitor status` reports its state.
- **Measures.** Typed measures (`measures list` / `measures run`) compute fleet
  standing, validation history, and debt trends from one path and federate to
  measures-health.

## Running

```bash
# Build API + UI, install pnpm deps, install the scenario CLI
make setup   # wraps `vrooli scenario setup`

# Start API + UI in the background
make start   # wraps `vrooli scenario start`

make logs    # tail API + UI logs
make stop    # stop the scenario
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full clone-to-running flow.
Run tests with `make test` (which wraps `vrooli scenario test template-manager`), or
invoke `test-genie execute template-manager --preset comprehensive` directly for
finer-grained presets.

## Documentation Map

| Need | Start Here |
|---|---|
| Product framing and operational targets | [`PRD.md`](PRD.md) |
| Maintain the template factory | [`docs/factory/TEMPLATE-FACTORY-GUIDE.md`](docs/factory/TEMPLATE-FACTORY-GUIDE.md) |
| The react-vite generation contract | [`docs/factory/TEMPLATE-GENERATION-CONTRACT.md`](docs/factory/TEMPLATE-GENERATION-CONTRACT.md) |
| Maintain the react-vite template | [`docs/factory/TEMPLATE-MAINTENANCE.md`](docs/factory/TEMPLATE-MAINTENANCE.md) |
| Architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Workflows, data, integrations | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md), [`docs/concepts/DATA.md`](docs/concepts/DATA.md), [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| UI design language | `DESIGN.md` at this scenario's root, [`docs/concepts/UI-ARCHITECTURE.md`](docs/concepts/UI-ARCHITECTURE.md) |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| Deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |
| Troubleshooting | [`docs/guides/troubleshooting.md`](docs/guides/troubleshooting.md) |

## Customize Safely

Template Manager is a real scenario, not scaffold to be replaced — extend it, don't
strip it. When you add product surface, keep these seams intact:

1. **Connect-RPC is the default transport.** Every domain endpoint goes through a
   proto service in `packages/proto/schemas/template-manager` and generated
   Connect handlers/clients. Do not write literal `Path: "/api/v1/..."` strings in an
   `EndpointDescriptor`; codegen rejects literal Paths that lack an explicit
   `RESTException` tag.
2. **The CLI stays a thin wrapper.** Logic lives in the API; the CLI is
   manifest-driven proto-JSON plumbing. Add commands by extending
   [`cli/manifest.json`](cli/manifest.json) and the API service, not by putting
   behavior in the CLI.
3. **Storage changes are migration-first.** The SQLite store is migrated, never
   recreated — add a migration rather than editing schema in place or dropping tables.
4. **Template content stays under `templates/`.** Template Manager operates on and
   validates template content; it does not house it.
5. **Update [`PRD.md`](PRD.md) and `requirements/`** before feature work — operational
   targets drive code and tests. Requirement status is auto-synced by Test Genie; tag
   tests with `[REQ:ID]` rather than hand-editing status.
6. **Keep [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) and
   [`docs/manifest.json`](docs/manifest.json) accurate**, read the root `DESIGN.md`
   before UI work, and keep edits within this scenario's directory.

## pnpm Everywhere

This scenario assumes pnpm. Scripts use `pnpm` directly (no `npm` fallbacks) to reduce
drift; convert lockfiles yourself if you use another package manager.
