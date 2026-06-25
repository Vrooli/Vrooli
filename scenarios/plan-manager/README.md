# Plan Manager

Plan Manager is the planning runtime and single authority for plan logic in
Vrooli: a guided wizard that makes **authoring** and **executing** implementation
plans cheap enough — in tokens and intelligence — for a local model. Plans become
first-class structured records with ordered phases, computed status and staleness,
and baseline-aware validation, instead of prose files a tired, max-context agent
has to reverse-engineer at the end of a run.

The mechanism is a deterministic section wizard plus validators that move judgment
into code, combined with just-in-time context injection during execution that holds
the procedural knowledge agents would otherwise carry in their heads.

See [`PRD.md`](PRD.md) for the full product framing and
[`docs/concepts/PLAN-MODEL.md`](docs/concepts/PLAN-MODEL.md) for the canonical
structured-plan + phase schema.

## Domains

Plan Manager is organized into four product domains, each a Connect-RPC service
with a matching CLI group:

- **plans** — the structured-plan SSOT. Create / read / update / archive plans,
  render the markdown view, manage first-class phases, instantiate per-surface
  templates (generic / cli / proto / ui), and track the supersession/dependency
  graph. Plans persist to the scenario-**independent** `~/.vrooli` home store so
  they stay readable with the server down.
- **authoring** — the guided composer wizard. Walk a plan's sections in order,
  validate structure as you go, and **autofill** the mechanical sections (regression
  anchor, required-reading, code references) behind seams so a small model only
  supplies genuine prose, then finalize into a structured plan.
- **execution** — the guided runner. `status`/`next` act as a just-in-time context
  server (current phase, what's next, phase-scoped reading + reminders, last
  validation results, staleness); transition phases; capture decisions and candidate
  findings in-flow; `complete` assembles the **canonical** handoff; read per-plan
  velocity.
- **validation** — plan health. Resolve code references against `code-facts`,
  compute staleness tiers (fresh / lightly-stale / definitely-stale), derive the
  exact baseline/validation command set for a plan's connected code, run it with the
  agent in the loop, and verify the Definition of Done against the regression anchor.

## Surfaces

- **CLI (`cli/`)** — the primary, agent- and operator-facing surface. Typed
  proto-JSON output; groups `plans`, `phase`, `template`, `author`, `exec`, and
  `validate`. This is the guided surface agents drive when authoring and executing a
  plan; the command list is the contract in [`cli/manifest.json`](cli/manifest.json).
- **API (`api/`)** — Go + Connect-RPC over the proto contracts in
  `packages/proto/schemas/plan-manager`. Storage is SQLite via api-core/storage,
  rooted at the durable home store.
- **UI (`ui/`)** — React + Vite operator console for viewing and managing plans,
  phase progress, staleness tiers, handoff records, candidate-finding triage, and
  velocity trends.

## Running

```bash
# Build API + UI, install pnpm deps, install the scenario CLI
make setup   # wraps `vrooli scenario setup`

# Start API + UI in the background
make start   # wraps `vrooli scenario start`
```

See [`docs/QUICKSTART.md`](docs/QUICKSTART.md) for the full flow. Run tests with
`make test` (which wraps `vrooli scenario test`), or invoke
`test-genie execute plan-manager --preset comprehensive` directly for finer-grained
presets.

## Integration posture

Plan Manager **composes** substrate it should not own and degrades gracefully when an
owner is down: `code-facts` (code references), the freshness engine (content-hash
staleness), `git-control-tower baseline` (regression anchor + diff), `test-genie` /
`scenario-validation` (validation results it consumes), `prompt-manager`
(`plan-skill-discovery` for required-reading autofill), and `meta-optimization-manager`
(velocity sink). It does **not** own project-level validation, read agent transcripts,
spawn agents, or promote candidate findings to real bugs — an operator triages those.

## Documentation Map

| Need | Start Here |
|---|---|
| The structured plan + phase model | [`docs/concepts/PLAN-MODEL.md`](docs/concepts/PLAN-MODEL.md) |
| Product framing and operational targets | [`PRD.md`](PRD.md) |
| Architecture | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Product domains | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Workflows, data, integrations | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md), [`docs/concepts/DATA.md`](docs/concepts/DATA.md), [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| UI design language | `DESIGN.md` at this scenario's root, [`docs/concepts/UI-ARCHITECTURE.md`](docs/concepts/UI-ARCHITECTURE.md) |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |
| Deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Write tests | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Seams and fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| Add API endpoints | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md) |
| Add CLI commands | [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |

## Working Rules

1. **Connect-RPC is the default transport.** Every domain endpoint goes through a
   proto service and generated Connect handlers/clients. Do not write literal
   `Path: "/api/v1/..."` strings in an `EndpointDescriptor`; codegen rejects literal
   Paths that lack an explicit `RESTException` tag.
2. **Update [`PRD.md`](PRD.md) and `requirements/`** before feature work; operational
   targets drive code + tests. Requirement status is auto-synced by Test Genie — tag
   tests with `[REQ:ID]` rather than hand-editing status.
3. **Update [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md)** before adding
   product code, and keep [`docs/manifest.json`](docs/manifest.json) accurate.
4. **Read the root `DESIGN.md` before UI work.** Tokens, motion, and status semantics
   are binding; preserve the i18n + accessibility seams.
5. **The agent never hand-edits plan markdown** — phase status is a typed transition,
   and the rendered markdown view reflects state but is never the source of truth.
6. **Append progress entries** to [`docs/internal/PROGRESS.md`](docs/internal/PROGRESS.md)
   when you land work, and keep edits within this scenario's directory.

## pnpm Everywhere

This scenario assumes pnpm. Scripts use `pnpm` directly (no `npm` fallbacks) to
reduce drift; convert lockfiles yourself if you use another package manager.
