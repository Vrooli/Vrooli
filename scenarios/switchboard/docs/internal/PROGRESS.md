# Progress — Switchboard

## Progress Log

Newest first. Each entry records what changed, what proves it, and what the next
agent should pick up. Keep entries factual; aspirations belong in `PRD.md`.

### 2026-09-01 — Scenario generated and documented; no product code written

**What was done.**

- Generated from the `react-vite` template (version 2.0.0) with the default
  `vrooli-default` design kit: `template-manager generate react-vite --id switchboard`.
- Authored the charter through the `business-health` wizard — the canonical PRD
  path, non-interactive via an answers file. **40 operational targets**: 17 P0,
  15 P1, 8 P2.
- Wizard emitted `requirements/01-must-ship`, `02-post-launch`, `03-future` with
  `SWBD-*` identifiers linked to targets by `prd_ref`. Removed the orphaned
  `requirements/01-foundation/module.json` starter module, which the wizard
  flagged as describing the scaffold rather than this scenario.
- Declared nine scenario dependencies in `.vrooli/service.json`, all
  `runtime_only`, with `supervision_precedence: required` on the three required
  ones. Zero resource dependencies at P0, deliberately.
- Authored `docs/concepts/DOMAINS.md`, `ARCHITECTURE.md`, `FLOWS.md`, `DATA.md`,
  `INTEGRATIONS.md`; `docs/internal/DECISIONS.md`, `SECURITY.md`, `PROBLEMS.md`,
  `PERFORMANCE.md`, this file; `docs/operations/*`; `docs/business/*`; and the
  planned-surface sections of the reference documents.
- Replaced the `ORIENTATION-TODO: scenario-design-adaptation` marker in
  `DESIGN.md` with the rationale for adopting the design kit unmodified.
- Authored the experience contract: six product page specs (`dashboard`,
  `agents`, `agent-new`, `conversations`, `contacts`, `channels`) with
  priorities and states, and three journeys (`first-agent`, `attach-a-handle`,
  `govern-access`). All `status: draft` — intent authored ahead of the build.

**Evidence.**

- `vrooli scenario requirements validate switchboard --json` → `status: PASSED`.
- `template-manager orient switchboard` → **7 of 9 gates complete**. Passing:
  scaffold-health, charter, requirements-registry, domain-map,
  dependency-decisions, design-language, experience-contract.

**True frontier — where the next agent starts.**

- **Two gates remain, both requiring code**: `first-real-vertical-slice` (needs
  two `api/internal/*/service.go`, two `api/handlers/*/*_test.go`, two
  `ui/src/pages/*.test.tsx`) and `example-domain-removed` (needs
  `template-manager detemplate switchboard`, which must not run until one real
  domain is green).
- **Build order is fixed** in `DOMAINS.md`: `channels` → `conversations` →
  `agents` → `trust` → `turns`. Do not build horizontally.
- **The first slice is `channels`**, because everything else reads it and it
  reads nothing. Within it, follow the `ARCHITECTURE.md` extension order: proto,
  API, transport, CLI, UI.
- **Ship the in-app adapter and Telegram together**, not sequentially. Two
  adapters on day one is the only cheap proof the contract is channel-neutral
  before anything depends on it.
- Nothing is implemented. No proto tree exists under
  `packages/proto/schemas/switchboard/`. Every requirement carries a `manual`
  validation stub; a green requirements validation reflects a complete *claim
  set*, not working behavior.

**Deliberately not done.**

- No domain code, no proto, no UI beyond the generated scaffold — the operator
  asked for initialization and documentation only.
- No plan-manager plan. The `scenario-generation` skill treats a canonical plan
  as the execution contract for the build phases; authoring one is the natural
  next step before the first slice.
- No `make setup` / `make start`. The scenario has not been run.
