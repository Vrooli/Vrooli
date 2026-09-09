# Problems — Template Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear.

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

### 2026-07-09 — comprehensive maturity gates remain red

**Symptom:** Focused Template Manager phase validations pass, but the default
`vrooli scenario test template-manager` comprehensive run still reports broader
maturity-provider failures.

**Root cause:** The current plan slices intentionally shipped domain behavior
before final hardening. Remaining findings include structure/profile evidence,
architecture ownership, docs snippets, duplication, security advisories, and
other provider-level maturity gaps.

**Workaround:** Use the focused phase validations recorded in the plan evidence
for Phase 1 through Phase 7 while continuing the remaining plan phases.

**Real fix:** Complete Phase 8 through Phase 11, then run and close the full
comprehensive suite and baseline diffs.

**Owner:** Template Manager implementation agents.

**Refs:** operator-local plan `template-manager-scenario-owning-the-template-domain.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| comprehensive suite | Focused plan slices are green while broader maturity gates remain unresolved. | Template Manager cannot claim final DoD until the default suite is green. | Finish hard cutover and final hardening phases, then update this entry with the closing run id. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## 2026-09-07 — Shared selector consolidation

W3 implementation review under the existing selector contract; no W0–W2 maturity promotion is claimed. Both React/Vite templates use @vrooli/ui-selectors and exporter scripts with governed package declarations and locks. Focused generation/copy checks pass. Unit run 20260907-213240-57f1e0c5 could not acquire unit-health. Full templateengine testing additionally found an invalid compose-service resource driver (knw-1788818286865080380), outside this selector change. Shared evidence and limitations: `packages/ui-selectors/README.md`.

## 2026-09-07 — Unused health fixture cleanup

W3 scoped implementation cleanup: removed 67 unused scenario health fixture
pairs after checking that builder and option symbols had no external Go
consumers. The canonical template example remains linked from the test-authoring
guide; manifest copy exclusions prevent its propagation to new scenarios.
Focused generation/copy checks and Proto Health handler/testutil tests pass.
Test Genie unit run: `20260907-222237-5e59d117` (terminal result recorded in
the work-record journal). No W0–W2 readiness claim is made.

## Work ladder — shared gamepad ownership

- Rung: W3 (scoped implementation change; W0–W2 not re-certified).
- Evidence: scenario entry points called `initSpatialNav()`, while local React
  hooks created independent managers; Swarm Manager's graph used both paths.
- Repair: shared application controller and React adapters, scoped action
  routing, modal registration cleanup, and template/provider migration.
- Measured: 2026-09-07. Package and scenario validation recorded in the work journal.

## Work ladder — shared binary boot harness

- Rung: W3, scoped consolidation under the existing binary-startup contract.
- Evidence: 67 scenario boot tests and the canonical template duplicated process
  setup and weak health/shutdown checks. `packages/api-core/boottest` now owns
  those mechanics; scenarios retain service identity and necessary startup inputs.
- Measured: 2026-09-07. Shared regression tests pass with the race detector. Real
  boot checks pass for Proto Health, Unit Health, Vrooli Bridge, and Template Manager.
- Limitation: Test Genie unit run `20260908-032406-c3fc9bec` returned FAIL with
  `UNIT_REQUIRED_ROLE_MISSING`: Code Facts did not observe the required UI role.
  This is separate from the passing focused boot checks; no W0–W2 claim is made.
