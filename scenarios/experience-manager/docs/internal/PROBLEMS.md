# Problems — Experience Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

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

### 2026-07-06 — BAS reconciliation captures only default state setup

**Symptom:** `state-covered`, `state-distinct`, and claim-level `x-` dimensions can be parsed and reported honestly, but non-default state or display-mode proof cannot be fully captured yet.

**Root cause:** Experience-manager now captures a viewport matrix (`desktop` 1280x720 and `mobile` 390x844) and persists evidence per viewport, but it still does not drive per-state transitions or per-dimension mode changes before capture.

**Workaround:** Default-state machine claims reconcile normally across captured viewports. Viewport-scoped claims reconcile when their viewport is in the matrix; scopes outside the matrix and non-default/extension-scoped claims stay unverifiable with explicit limitation reasons or must be manual/aspirational.

**Real fix:** Add a state setup runner that can execute per-state setup steps before capture and persist evidence per state/mode, using the existing viewport profile machinery.

**Owner:** unassigned.

**Refs:** `api/internal/reconcile/reconcile.go`, `docs/internal/DECISIONS.md` rows for display-mode scoping and extension-scope reconciliation.

### 2026-07-06 — Experience gate remains advisory by default

**Symptom:** Error-severity experience findings can produce an advisory-pass suite result unless `EXPERIENCE_ALIGNMENT_GATE=strict` is set.

**Root cause:** The test-genie `experience` phase launched shadow-first so existing scenarios without mature specs would not fail fleet-wide.

**Workaround:** Keep strict-mode assertions in unit tests and use provider output plus BAS evidence for acceptance. Do not promote the gate during this follow-up plan.

**Real fix:** Promote strict gating only after fleet adoption and after the cockpit dogfood is green in live suite runs.

**Owner:** unassigned.

**Refs:** `api/handlers/validation/connect_handler.go`, `api/handlers/validation/connect_handler_test.go`.

### 2026-07-06 — Perception tier remains deferred

**Symptom:** Visual saliency and pixel-side promises remain aspirational/manual; v1 checks only deterministic contract and accessibility-tree structure.

**Root cause:** P2 perception needs a separate engine decision and must stay off the CI hot path until deterministic enough to trust.

**Workaround:** Keep v1 claims focused on roles, accessible names, keyboard reachability, state coverage, evidence links, and behavior visible through the live UI.

**Real fix:** Start P2 with a quarantined advisory perception runner and explicit model/license decision.

**Owner:** unassigned.

**Refs:** PRD OT-P2-001, `docs/internal/DECISIONS.md` zero-ML and saliency decisions.

### 2026-07-04 — BAS lacks accessibility-tree capture (blocks reconciliation)

**Symptom:** OT-P0-003 (structure reconciliation) cannot be implemented: BAS execution timelines carry screenshots but no accessibility-tree snapshot per step.

**Root cause:** Feature does not exist in BAS yet; ui-health's `uiruntime` only consumes screenshot + console/network observations.

**Workaround:** None needed yet — build order puts the spec contract, provider, and studio first.

**Real fix:** File the cross-scenario feature request to BAS (a11y-tree snapshot alongside each step screenshot) before reconciliation work starts; the `bas-screenshot-api-audit` backlog item already touches this capture surface.

**RESOLVED 2026-07-04 — BAS now captures the accessibility tree.** BAS ships `CAPTURE_TYPE_ACCESSIBILITY` (capture proto value 7), which walks the Chromium AX tree via CDP `Accessibility.getFullAXTree` at a settled point, joins per-node geometry + `data-testid`, and emits `accessibility.json` normalized to the frozen contract **`bas-accessibility-snapshot/v1`** (role/name/description/value/states/bounds/`dom.{testid,tag}`/children; ignored nodes pruned, empty fields omitted, main-frame-only in v1). `inline_accessibility` returns it inline in `CaptureResponse.accessibility_json`. Validated live against a Vrooli UI: 355 nodes, roles+names+bounds present, real `data-testid`s surfaced. Reconciliation (OT-P0-003) can now consume this contract. **Scope note:** v1 delivers the *single-location* capture (one snapshot per capture, at the final settled page — the point the final screenshot fires). *Per-step timeline attachment* is deferred; BAS reserved the multi-step slot as `ARTIFACT_TYPE_ACCESSIBILITY_SNAPSHOT` (base proto value 8) but has not wired per-step emission — if reconciliation needs an AX snapshot on *every* step (not just the terminal state), that per-step wiring is a follow-up. Contract seam: `scenarios/browser-automation-studio/docs/SEAMS.md` §30.

**Owner:** unassigned.

**Refs:** `docs/internal/DECISIONS.md` (single-capture-engine decision), `scenarios/ui-health/api/internal/uiruntime/`, `scenarios/browser-automation-studio/docs/SEAMS.md` §30 (`bas-accessibility-snapshot/v1`), `packages/proto/schemas/browser-automation-studio/v1/capture/capture.proto`.

### 2026-07-04 — Spec schema spike gate before broad rollout

**Symptom:** The claim vocabulary is designed but unproven against real pages; a too-abstract or too-rigid vocabulary would either block authoring or pigeonhole UX.

**Root cause:** Only this scenario's own `experience/` spec exists so far (authored 2026-07-04, schema-valid); the two external pages remain unexpressed.

**Workaround:** n/a — this is a deliberate gate, not a defect.

**Real fix:** Before promoting the phase beyond presence-keyed applicability, the schema must express three pages naturally at useful depth: business-health's Matrix (dense conventional), web-console's terminal/chat surface (hostile to descriptive schemas), and this scenario's own Studio (self-referential dogfood, OT-P0-005).

**RESOLVED 2026-07-04 — gate CLOSED 3/3, zero schema changes needed.** Studio (1/3), business-health Matrix (2/3: `scenarios/business-health/experience/`, authored to intent with 8 expected reconciliation failures as the detection-calibration list — see its `x-spike` block), web-console workspace (3/3: `scenarios/web-console/experience/`, the hostile case — routerless SPA, `x-terminal` custom role, no DESIGN.md, orthogonal display modes — all absorbed by open-world `x-` semantics). Two vocabulary learnings recorded as DECISIONS.md rows (display-mode `x-` scoping; DESIGN.md-absent graceful degradation). Rollout is no longer blocked on vocabulary. Entry retained only until the Go parser (OT-P0-001) reproduces the scratch validator's checks; delete it then.

**Owner:** unassigned.

**Refs:** DECISIONS rows on claim schema, open-world semantics, maturity ladder, display-mode scoping, DESIGN.md-absent degradation.

### 2026-07-04 — requirements auto-sync disabled until real validation refs exist

**Symptom:** `requirements/index.json` has `auto_sync_enabled: false`, so requirement statuses will not update automatically from suite runs.

**Root cause:** Known platform bug (filed as `requirements-sync-refless-validations-fake-complete` during the business-health build): comprehensive runs flip refless test-typed validations to complete, fabricating completion for unbuilt work. All 13 seeded requirements currently carry refless stubs.

**Workaround:** Keep auto-sync off; statuses stay honestly `planned`.

**Real fix:** Re-enable once requirements carry real validation refs (`[REQ:...]`-tagged tests) as implementation lands.

**Owner:** unassigned.

**Refs:** business-health PROBLEMS.md documents the same revert loop.

### 2026-07-04 — Requirements modules are wizard-tier-shaped, not thematic

**Symptom:** `requirements/` has `01-must-ship` / `02-post-launch` / `03-future` (one module per OT tier) rather than the eight thematic modules sketched during design (spec-contract, provider-integration, reconciliation, authoring-studio, render-workshop, scaffolding-autofix, attestation-fleet, ux).

**Root cause:** The business-health wizard seeds the registry 1:1 from OT tiers — conformant by construction.

**Workaround:** None needed; the registry validates green and requirement↔OT linkage is complete.

**Real fix:** Optionally restructure into thematic modules during implementation planning, when requirements gain real validation refs anyway. Not worth hand-editing before then.

**Owner:** unassigned.

**Refs:** `requirements/index.json` imports.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
