# Problems — Nutrition Planner

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Entries are appended as they appear, so they are not rediscovered.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-09-18 — PRD was authored directly, not through the business-health wizard

**Symptom:** `PRD.md` is conformant and scenario-specific, but it was not produced by the canonical `business-health wizard` path that `docs/START-HERE.md` Gate 1 prescribes. The registry was authored directly from the PRD instead (`vrooli scenario requirements validate nutrition-planner` passes with the `requirements_registry` capability at L3).

**Root cause:** The PRD, requirements registry, concepts, and experience contract were written from the canonical product specification as one deliberate documentation/planning step rather than through the wizard interview. This is a **process divergence, not a product defect**; the resulting PRD validates, carries real `OT-P0/P1/P2` targets, and every target has at least one requirement.

**Workaround:** Treat `PRD.md` and the direct registry as authoritative. Keep them reconciled with `business-health validate scenario nutrition-planner --json` and `vrooli scenario requirements validate nutrition-planner --json`.

**Real fix:** Decide whether direct authoring remains the convention for PRD-spec-driven scenarios or whether to reconcile through `business-health wizard` / `business-health fix preview/apply`; record the decision in `docs/internal/DECISIONS.md`. No registry action is outstanding — the starter `requirements/01-foundation/` module has been removed.

**Owner:** unassigned (next implementation/orientation agent).

**Refs:** `PRD.md`, `requirements/index.json`, `requirements/README.md`, `docs/START-HERE.md` Gates 1–2.

### 2026-09-18 — UI, API, and CLI implementation has not started

**Symptom:** The scenario idles at the scaffold state. Only the `health` domain and the fenced `notes` example domain exist; none of the product domains (profile/rules, food catalog, recipes, nutrition, planning, inventory/costs, intake/feedback, transfer, provider jobs) are implemented. `make orient` reports the first-real-vertical-slice and design-language gates open.

**Root cause:** This milestone is documentation/planning only. The canonical specification is now written into `PRD.md`, `requirements/`, `docs/concepts/`, the business/operations/internal docs, and the experience contract; no product code was in scope.

**Workaround:** None required. Do not represent any product capability as working. The reference specification, not this repository, is the current statement of intended behavior.

**Real fix:** Deliver R0 as staged milestones (spec §22.2), starting with the durable minimal-meal vertical slice: authenticated workspace → name-only meal draft → durable save → collection display → reopen/edit → native export (`BUILD-01`).

**Owner:** unassigned (next implementation agent).

**Refs:** `docs/reference/product-specification.md` §22, `.vrooli/orientation.json`, `docs/START-HERE.md` Gate 6.

### 2026-09-18 — Orientation gates: scaffold-health passed; several gates remain open

**Symptom:** The scaffold starts, reports status, and passes the generated lifecycle test (Gate 0 / `scaffold-health`). The following orientation steps remain unsatisfied: `design-language` (the home placeholder and `DESIGN.md` rationale are untouched), `first-real-vertical-slice` (no real product domain), and `example-domain-removed`.

**Root cause:** Documentation work ran ahead of implementation. Replacing the home placeholder and building a real domain are implementation tasks, not doc tasks, and were intentionally not attempted.

**Workaround:** Use `make orient` as the current progress check. Read the documentation set as intent; read the code as the actual, still-scaffold state.

**Real fix:** Complete Gate 5 (design decision / home placeholder) and Gate 6 (first real vertical slice), then Gate 7 (`template-manager detemplate`). Do not flip `docs/manifest.json` maturity values to reflect implementation that does not exist.

**Owner:** unassigned (next implementation agent).

**Refs:** `.vrooli/orientation.json` step ids `scaffold-health`, `design-language`, `first-real-vertical-slice`, `example-domain-removed`, `progress-handoff`; `docs/concepts/EXPERIENCE.md`, `ui/src/pages/DashboardPage.tsx`.

### 2026-09-18 — The removable `notes` example domain is still present by design

**Symptom:** The `notes` domain still exists across proto, API, CLI, UI, schema, and docs (fenced with `EXAMPLE-DOMAIN` markers), and its page remains in `experience/index.json`. It carries placeholder data and is not product scope.

**Root cause:** The template ships one worked vertical slice to copy, then delete. It is removed by `template-manager detemplate nutrition-planner` only after a real domain is green (Gate 7), which has not happened.

**Workaround:** Treat every `notes` surface as reference only; do not extend it, and do not let its schema or UI patterns be mistaken for the product's domain model.

**Real fix:** Build the first real product domain beside it, prove that domain green across API/CLI/UI, then run `template-manager detemplate nutrition-planner` and confirm the `example-domain-removed` gate passes.

**Owner:** unassigned (next implementation agent).

**Refs:** `docs/START-HERE.md` Gates 6–7; `docs/concepts/DOMAINS.md`; `docs/concepts/DATA.md`; `experience/index.json`.

### 2026-09-18 — Generated reference docs still mirror the `notes` example and miss manifest headings

**Symptom:** `docs/reference/api-endpoints.md` and `docs/reference/cli-commands.md` document the example `notes` domain (correct for the current code) but their headings do not match the `docs/manifest.json` contract (`Notes (CRUD reference)` and `Scenario commands — notes (CRUD reference)`). Those two required headings are therefore unsatisfied.

**Root cause:** The reference docs are generated from the scaffold's example domain, and the manifest's required-heading expectations drifted from the generated heading text. Both are inherited template artifacts, not product decisions.

**Workaround:** Treat the reference docs as accurate to the current code and leave the heading mismatch until the first real domain replaces `notes`.

**Real fix:** Build the first real domain (Gate 6), run `template-manager detemplate`, then regenerate `.vrooli/endpoints.json`, `docs/reference/api-endpoints.md`, and `docs/reference/cli-commands.md` and reconcile their headings with the manifest in one pass.

**Owner:** unassigned (next implementation agent).

**Refs:** `docs/manifest.json` reference section; `docs/reference/api-endpoints.md`; `docs/reference/cli-commands.md`; `docs/START-HERE.md` Gates 6–7.

### 2026-09-18 — Baseline comprehensive test run fails on inherited scaffold debt (docs phase included)

**Symptom:** `test-genie execute nutrition-planner` (default preset, 27 phases; run `20260918-145731-c1c08d4f`) returns FAIL. Passing phases include `structure`, `api`, `architecture`, `quality`, `performance`, `business`, `experience`, `tidiness`, `security`, `programs`, `proto`, `branding`, `templates`, `code-facts`, and both code-graph phases. Failing phases: `portability`, `contracts`, `ui-health`, `dependencies`, `docs`, `unit`, `storage`, `workflow`, `measures`, `skill-set`.

**Root cause:** The failures are inherited scaffold/template conditions, not this documentation milestone. Examples: `ui-health` blocks on `standard_shell_ownership` / `standard_component_adoption_contracts` (the home placeholder has not been replaced and the shell has not been configured — intended Gate 5 work); `contracts` on `binding.scalar_bound_to_message`; `storage` on `STORAGE_ACCOUNTABILITY_UNDECLARED`; `unit` on `TEST_MISCONFIGURATION`; `docs` on `broken_command_snippet` in `bas/README.md` and `docs/QUICKSTART.md` plus `broken_marked_ref` in template guides and reference docs; `branding`, `portability`, `dependencies`, `workflow`, `measures`, and `skill-set` carry their own template debt.

**Workaround:** None required for the documentation milestone; none of the failing findings point at `PRD.md`, `requirements/`, `docs/concepts/`, `docs/business/`, `docs/operations/`, `docs/internal/`, or `experience/`. Validate the documentation contract directly with `business-health validate scenario nutrition-planner`, `vrooli scenario requirements validate nutrition-planner`, `experience-manager spec validate nutrition-planner`, and `knowledge-observatory docs audit nutrition-planner`.

**Real fix:** Address each inherited finding as part of the implementation milestones: Gate 5 for ui-health/design, Gate 6 for contracts/storage/unit, and the template-owned reference/snippet debt as the scaffold is replaced. The remaining `docs` reference debt (12 markers in `docs/guides/choosing-ui.md`, `docs/guides/troubleshooting.md`, `docs/reference/cli-commands.md`, `docs/reference/component-library-gaps.md`, and `docs/reference/configuration.md`) is fleet-wide: the `path:` markers resolve at repository root while the auditor resolves them scenario-relative, and mature scenarios carry the same pattern.

**Owner:** unassigned (next implementation agent).

**Refs:** test-genie run `20260918-145731-c1c08d4f`; `docs/START-HERE.md` Gates 5–7; `knowledge-observatory docs audit nutrition-planner --json`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Documentation vs. implementation | The PRD, requirements registry, concepts, business/operations/internal docs, and experience contract now describe the real product, but no product code exists yet — only the `health` domain and the fenced `notes` example. A reader who trusts the docs would assume capabilities that are not implemented. | Documentation is deliberately ahead of implementation; `docs/manifest.json` marks these documents `active` as authored contracts, not as working code. | Deliver the first real vertical slice (Gate 6), then detemplate (Gate 7), and keep every planned-versus-implemented statement explicit until then. |
| Design contract vs. implementation | `DESIGN.md` still carries the template `ORIENTATION-TODO: scenario-design-adaptation` rationale and the home surface is still marked `PLACEHOLDER:home-surface`; `docs/concepts/EXPERIENCE.md` states the product decision but the UI has not been configured to it. | The design-language gate cannot pass, so no UI implementation should be treated as final. | Complete Gate 5: configure the shell, replace the home placeholder, and record the design rationale in `DESIGN.md`. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
- [`../reference/product-specification.md`](../reference/product-specification.md) — canonical product/implementation specification
