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

> **Status 2026-09-22:** still accurate; direct authoring was repeated for the v2.0 redesign (D-025).

**Symptom:** `PRD.md` is conformant and scenario-specific, but it was not produced by the canonical `business-health wizard` path that `docs/START-HERE.md` Gate 1 prescribes. The registry was authored directly from the PRD instead (`vrooli scenario requirements validate nutrition-planner` passes with the `requirements_registry` capability at L3).

**Root cause:** The PRD, requirements registry, concepts, and experience contract were written from the canonical product specification as one deliberate documentation/planning step rather than through the wizard interview. This is a **process divergence, not a product defect**; the resulting PRD validates, carries real `OT-P0/P1/P2` targets, and every target has at least one requirement.

**Workaround:** Treat `PRD.md` and the direct registry as authoritative. Keep them reconciled with `business-health validate scenario nutrition-planner --json` and `vrooli scenario requirements validate nutrition-planner --json`.

**Real fix:** Decide whether direct authoring remains the convention for PRD-spec-driven scenarios or whether to reconcile through `business-health wizard` / `business-health fix preview/apply`; record the decision in `docs/internal/DECISIONS.md`. No registry action is outstanding — the starter `requirements/01-foundation/` module has been removed.

**Owner:** unassigned (next implementation/orientation agent).

**Refs:** `PRD.md`, `requirements/index.json`, `requirements/README.md`, `docs/START-HERE.md` Gates 1–2.

### 2026-09-18 — UI, API, and CLI implementation has not started

> **Status 2026-09-22: superseded.** Code was written after this entry but has never worked end to end; see the 2026-09-22 entry below.

**Symptom:** The scenario idles at the scaffold state. Only the `health` domain and the fenced `notes` example domain exist; none of the product domains (profile/rules, food catalog, recipes, nutrition, planning, inventory/costs, intake/feedback, transfer, provider jobs) are implemented. `make orient` reports the first-real-vertical-slice and design-language gates open.

**Root cause:** This milestone is documentation/planning only. The canonical specification is now written into `PRD.md`, `requirements/`, `docs/concepts/`, the business/operations/internal docs, and the experience contract; no product code was in scope.

**Workaround:** None required. Do not represent any product capability as working. The reference specification, not this repository, is the current statement of intended behavior.

**Real fix:** Deliver R0 as staged milestones (spec §22.2), starting with the durable minimal-meal vertical slice: authenticated workspace → name-only meal draft → durable save → collection display → reopen/edit → native export (`BUILD-01`).

**Owner:** unassigned (next implementation agent).

**Refs:** `docs/reference/product-specification.md` §22, `.vrooli/orientation.json`, `docs/START-HERE.md` Gate 6.

### 2026-09-18 — Orientation gates: scaffold-health passed; several gates remain open

> **Status 2026-09-22:** the gates are still open in substance; see the status note at the top of [`../START-HERE.md`](../START-HERE.md).

**Symptom:** The scaffold starts, reports status, and passes the generated lifecycle test (Gate 0 / `scaffold-health`). The following orientation steps remain unsatisfied: `design-language` (the home placeholder and `DESIGN.md` rationale are untouched), `first-real-vertical-slice` (no real product domain), and `example-domain-removed`.

**Root cause:** Documentation work ran ahead of implementation. Replacing the home placeholder and building a real domain are implementation tasks, not doc tasks, and were intentionally not attempted.

**Workaround:** Use `make orient` as the current progress check. Read the documentation set as intent; read the code as the actual, still-scaffold state.

**Real fix:** Complete Gate 5 (design decision / home placeholder) and Gate 6 (first real vertical slice), then Gate 7 (`template-manager detemplate`). Do not flip `docs/manifest.json` maturity values to reflect implementation that does not exist.

**Owner:** unassigned (next implementation agent).

**Refs:** `.vrooli/orientation.json` step ids `scaffold-health`, `design-language`, `first-real-vertical-slice`, `example-domain-removed`, `progress-handoff`; `docs/concepts/EXPERIENCE.md`, `ui/src/pages/DashboardPage.tsx`.

### 2026-09-18 — The removable `notes` example domain is still present by design

> **Status 2026-09-22: mostly resolved.** The `notes` code was removed on 2026-09-18; stale `notes`/`attachments` tables, `notes` comments in Go, and the template proto README remain ([`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3.5).

**Symptom:** The `notes` domain still exists across proto, API, CLI, UI, schema, and docs (fenced with `EXAMPLE-DOMAIN` markers), and its page remains in `experience/index.json`. It carries placeholder data and is not product scope.

**Root cause:** The template ships one worked vertical slice to copy, then delete. It is removed by `template-manager detemplate nutrition-planner` only after a real domain is green (Gate 7), which has not happened.

**Workaround:** Treat every `notes` surface as reference only; do not extend it, and do not let its schema or UI patterns be mistaken for the product's domain model.

**Real fix:** Build the first real product domain beside it, prove that domain green across API/CLI/UI, then run `template-manager detemplate nutrition-planner` and confirm the `example-domain-removed` gate passes.

**Owner:** unassigned (next implementation agent).

**Refs:** `docs/START-HERE.md` Gates 6–7; `docs/concepts/DOMAINS.md`; `docs/concepts/DATA.md`; `experience/index.json`.

### 2026-09-18 — Generated reference docs still mirror the `notes` example and miss manifest headings

> **Status 2026-09-22:** the reference docs now carry a generic `<domain>` placeholder, and the manifest no longer requires the `notes` headings; regenerate both docs from the real API and CLI (ledger F-016).

**Symptom:** `docs/reference/api-endpoints.md` and `docs/reference/cli-commands.md` document the example `notes` domain (correct for the current code) but their headings do not match the `docs/manifest.json` contract (`Notes (CRUD reference)` and `Scenario commands — notes (CRUD reference)`). Those two required headings are therefore unsatisfied.

**Root cause:** The reference docs are generated from the scaffold's example domain, and the manifest's required-heading expectations drifted from the generated heading text. Both are inherited template artifacts, not product decisions.

**Workaround:** Treat the reference docs as accurate to the current code and leave the heading mismatch until the first real domain replaces `notes`.

**Real fix:** Build the first real domain (Gate 6), run `template-manager detemplate`, then regenerate `.vrooli/endpoints.json`, `docs/reference/api-endpoints.md`, and `docs/reference/cli-commands.md` and reconcile their headings with the manifest in one pass.

**Owner:** unassigned (next implementation agent).

**Refs:** `docs/manifest.json` reference section; `docs/reference/api-endpoints.md`; `docs/reference/cli-commands.md`; `docs/START-HERE.md` Gates 6–7.

### 2026-09-18 — Baseline comprehensive test run fails on inherited scaffold debt (docs phase included)

> **Status 2026-09-22: superseded** by the runs in the evidence index of [`REDESIGN_LEDGER.md`](REDESIGN_LEDGER.md).

**Symptom:** `test-genie execute nutrition-planner` (default preset, 27 phases; run `20260918-145731-c1c08d4f`) returns FAIL. Passing phases include `structure`, `api`, `architecture`, `quality`, `performance`, `business`, `experience`, `tidiness`, `security`, `programs`, `proto`, `branding`, `templates`, `code-facts`, and both code-graph phases. Failing phases: `portability`, `contracts`, `ui-health`, `dependencies`, `docs`, `unit`, `storage`, `workflow`, `measures`, `skill-set`.

**Root cause:** The failures are inherited scaffold/template conditions, not this documentation milestone. Examples: `ui-health` blocks on `standard_shell_ownership` / `standard_component_adoption_contracts` (the home placeholder has not been replaced and the shell has not been configured — intended Gate 5 work); `contracts` on `binding.scalar_bound_to_message`; `storage` on `STORAGE_ACCOUNTABILITY_UNDECLARED`; `unit` on `TEST_MISCONFIGURATION`; `docs` on `broken_command_snippet` in `bas/README.md` and `docs/QUICKSTART.md` plus `broken_marked_ref` in template guides and reference docs; `branding`, `portability`, `dependencies`, `workflow`, `measures`, and `skill-set` carry their own template debt.

**Workaround:** None required for the documentation milestone; none of the failing findings point at `PRD.md`, `requirements/`, `docs/concepts/`, `docs/business/`, `docs/operations/`, `docs/internal/`, or `experience/`. Validate the documentation contract directly with `business-health validate scenario nutrition-planner`, `vrooli scenario requirements validate nutrition-planner`, `experience-manager spec validate nutrition-planner`, and `knowledge-observatory docs audit nutrition-planner`.

**Real fix:** Address each inherited finding as part of the implementation milestones: Gate 5 for ui-health/design, Gate 6 for contracts/storage/unit, and the template-owned reference/snippet debt as the scaffold is replaced. The remaining `docs` reference debt (12 markers in `docs/guides/choosing-ui.md`, `docs/guides/troubleshooting.md`, `docs/reference/cli-commands.md`, `docs/reference/component-library-gaps.md`, and `docs/reference/configuration.md`) is fleet-wide: the `path:` markers resolve at repository root while the auditor resolves them scenario-relative, and mature scenarios carry the same pattern.

**Owner:** unassigned (next implementation agent).

**Refs:** test-genie run `20260918-145731-c1c08d4f`; `docs/START-HERE.md` Gates 5–7; `knowledge-observatory docs audit nutrition-planner --json`.

### 2026-09-22 — The app has never worked end to end; the redesign contract replaces the old one

**Symptom:** In the running scenario every workspace-scoped RPC returns `401 unauthenticated` (UI, UI proxy, and CLI), so every page renders its error state; the live database has 29 tables and zero rows. Beyond that, a fresh workspace hits Internal errors on profile, plan, and swap; Today and Week regenerate and overwrite each other's plan; ~1.1k lines of domain logic are reachable only from tests; the UI is a minified-style prototype with raw palette classes and no redesign surface. No requirement references a real test and none of the 143 experience-contract test ids exists in the UI.

**Root cause:** The scenario declares no authentication profile, so no principal reaches the handlers (B1); the remaining causes are listed as B2–B14 in [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3.2. Earlier progress entries recorded code that compiled and unit-tested, not behaviour that ran.

**Workaround:** None for users. Do not treat the 2026-09-18 "implemented" entries in [`PROGRESS.md`](PROGRESS.md) or the archived plan's "done" phases as evidence (D-035).

**Real fix:** Execute the redesign goal ([`REDESIGN_GOAL.md`](REDESIGN_GOAL.md)) starting with milestone D0 in [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §5; delete this entry when D0's exit proof and the redesign's definition of done are recorded in [`REDESIGN_LEDGER.md`](REDESIGN_LEDGER.md).

**Rung record (`scenario-work-ladder`, 2026-09-22):**
- **W0 contract** — repaired: `PRD.md` regenerated for the v2.0 redesign against the operator-supplied specification and mockups (D-025). `business-health validate` reports the contract shape clean.
- **W1 obligations** — repaired in the same change: requirement modules updated and 20–29 added so every operational target has requirements.
- **W2 evidence** — broken: zero validation refs to real tests; every status is `planned`. This is the rung the redesign goal must raise, surface by surface, through `[REQ:ID]`-tagged tests.
- **W3 implementation** — broken at the foundation (B1–B14) and absent for most redesign surfaces.

**Owner:** the redesign goal.

**Refs:** [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3; audit of 2026-09-22 summarized there; `api/main.go`, `.vrooli/service.json`, `api/internal/profile/sqlite.go`, `api/internal/planning/`, `ui/src/features/`.

### 2026-09-22 — Requirements sync promotes ref-less validations from a phase-level pass

**Symptom:** After the comprehensive Test Genie run `20260922-205742-7edb0e18`, 93 requirements became `complete`, 143 validation entries became `implemented`, and six PRD targets were checked, although no test references any requirement and the app does not work.

**Root cause:** test-genie's requirement matcher (`scenarios/test-genie/api/internal/requirements/enrichment/matcher.go`, `matchByPhase`) folds the phase-level `__phase__<phase>` record and every record without a requirement id into the candidates for any validation that declares that phase. The schema makes `phase` mandatory and `ref` optional, so every ref-less validation inherits its phase's overall pass.

**Workaround:** Statuses reverted to `planned` and PRD boxes cleared; auto-sync disabled for every module (D-037).

**Real fix:** test-genie must require per-requirement evidence (a `ref` whose test carries `[REQ:<ID>]`, a manual attestation, or a record naming the requirement) before promoting a validation. Here, re-enable sync per module once its validations carry real refs. Delete this entry when both hold.

**Owner:** test-genie (scenario-qa bug `knw-1790111426711964967`); the redesign goal for the per-module re-enable.

**Refs:** D-037; [`REDESIGN_LEDGER.md`](REDESIGN_LEDGER.md) F-015.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Documentation vs. implementation | As of 2026-09-22 the documentation describes the v2.0 redesign as intended behaviour and states the current state separately ([`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3). The code is a non-working prototype of the v1.0 direction. | Documentation is deliberately ahead of implementation; every planned-versus-existing statement must stay explicit. | Deliver the redesign goal; keep §3 of the plan current as defects close. |
| Design contract vs. implementation | `DESIGN.md` now defines the Warm Kitchen language (D-030); `design-tokens.css`, fonts, and every page still use the generic vrooli-default blue/cyan kit and raw palette classes. | No UI can be judged against the mockups until tokens, fonts, and the shell are rebuilt. | Milestone D1 in [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §5. |
| Experience contract vs. UI | `experience/` describes the redesigned pages with aspirational claims; none of its bound test ids exist in `ui/src`. | The experience phase cannot prove anything yet. | Render bound test ids as surfaces land and promote claims to machine tier (D-035). |

## Current-state clarification

### 2026-09-18 — Historical scaffold entries require supersession context

**Symptom:** Earlier entries in this file state that implementation had not started and that the notes example remained present. Those statements describe the documentation handoff state, not the current worktree.

**Root cause:** The implementation progressed substantially after the entries were authored, but the persistent problem register intentionally keeps historical entries unchanged.

**Workaround:** Read the current code, `PROGRESS.md`, and the active plan evidence as authoritative for present implementation state.

**Real fix:** Retire or annotate historical entries when the scenario’s documentation lifecycle permits a consolidated problem-register cleanup; do not use them as current product claims.

**Update 2026-09-22:** Superseded by the 2026-09-22 entry above; the active Plan Manager execution referenced below belonged to a plan that is now archived (D-026).

**Owner:** nutrition-planner implementation

**Refs:** `PROGRESS.md`, active Plan Manager execution `27e8a38e-18d9-47c4-9bd9-d5925c91c002`.

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
- [`../reference/product-specification.md`](../reference/product-specification.md) — canonical product/implementation specification
