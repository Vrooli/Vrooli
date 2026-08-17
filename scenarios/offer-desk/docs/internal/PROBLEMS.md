# Problems — Offer Desk

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Entries are appended as constraints appear, so they are not rediscovered.

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

## Work ladder

- Rung: W0
- Evidence: The required W0 goal query could not run because `swarm-manager` is stopped; lifecycle start fails during setup at `scenarios/swarm-manager/ui/src/components/settings/GoalDrainToggle.tsx:33` with a TypeScript `AutoDrainState`/`Updater` mismatch. Offer Desk's PRD P0 targets and this execution plan were read, but no goal comparison was inferred from the unavailable runtime.
- Blocker: W0 contract comparison is currently unverifiable; the unrelated `swarm-manager` UI build defect is outside this scenario's scope.
- Measured: 2026-08-16

### Scaffold health is complete; capture is an external boundary

The scenario starts through `make start`, reports healthy, and has API, CLI,
and UI evidence. Product-side validation is complete for the adopted scope;
direct experience capture remains an external provider boundary and is not an
Offer Desk completion blocker.

### The import is the riskiest step, and it is ordered

`OT-P0-006` moves 22 markdown files into this scenario. The ordering rule is absolute and easy to get wrong under time pressure: **import, verify per-file counts, then delete sources.** The source files are the importer's only input. The team that owns them must be paused first, because a running team writes to the surfaces the importer reads.

### The source catalog is already 19% broken

33 of 174 internal links in `docs/monetization/` do not resolve. `MIG-002` requires those to import as findings rather than be discarded, so the drift stays visible after the move. Expect the first import run to produce a substantial finding list; that is the correct outcome, not a failure.

### The trigger language will be asked to grow

`GATE-002` deliberately admits only declared facts, comparison operators and boolean composition. The first real trigger that wants something richer will feel like a small exception. It is not — that is how a rules engine starts. Route it to `OT-P2-004` (scenario-sourced facts) rather than widening the expression language.

### The generated shell fails two of its own accessibility floors on mobile

Discovered 2026-08-13 during experience validation, and **reproduced identically in both scenarios generated from `react-vite` v1.6.5**, so this is a template defect rather than anything either scenario did:

- `floor-tap-target-size` — the theme control renders ~79×30px on mobile, under the 44px minimum.
- `floor-safe-area-tap-targets` — an interactive target on the dashboard overlaps the mobile unsafe bottom area.

This matters more than a normal placeholder complaint because `docs/START-HERE.md` lists the shell's *"fixed safe-area bottom navigation on mobile"* as durable infrastructure to preserve, not as illustrative content. The floor it is supposed to guarantee is the one failing.

It is invisible until a scenario's experience contract is real enough to be checked against a running UI, which is why a freshly generated scenario reports clean. Fix it in the template rather than per-scenario, or every future scenario inherits it.

**Reproduce:** `make start`, then `experience-manager spec validate <scenario> --json`.

**Filed:** against `template-manager` (`react-vite` v1.6.5), not against this scenario. Do not patch the shell here — a per-scenario fix hides the defect from every future generation.

### 2026-08-15 — capability-gap plan execution

The current plan is repairing the importer, runtime integration, projection, evaluation condition, and experience coverage. The red-bar tests in `importer_test.go` prove the status-shape and silent-default defects before repair. Protected-tree input digests are recorded in `PROGRESS.md`; no source or team file is in scope for edits.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None recorded._ | | | |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

### 2026-08-13 — requirement evidence is not yet promoted

The implementation and focused suites are green, but the registry remains
`planned` until a fresh comprehensive suite sync and the twelve Level 3
behavioural drills produce durable evidence. The affected Test Genie rerun is
The fresh full Test Genie run terminated with only the `ui-health` provider
failing and no findings payload. Direct experience validation is now clean
after BAS stabilized. The inherited Phase 1 log did not contain the required
start-of-plan protected-tree hashes, so the current end hashes are recorded in
`PROGRESS.md` without claiming a historical comparison.

### 2026-08-14 — final validation boundary and intentional deferrals

The requirement registry is now evidence-backed: `vrooli scenario requirements
validate offer-desk --json` passes at L3, with 18 requirements complete and 6
explicitly planned (including source retirement and later catalog capabilities).
The fresh comprehensive run `20260814-035300-bf1ee069` passed 20/21 phases. The
remaining failure is not an Offer Desk finding: the shared Test Genie `ui-health`
execution provider times out without returning a findings payload. Static-only
UI-health reports zero required findings, and direct experience validation reports
zero findings. The protected-tree end hashes are recorded in `PROGRESS.md`; no
historical start hash was available from Phase 1.

### 2026-08-15 — capability-gap plan rehearsal boundaries

The catalog rehearsal is read-only against a copy of `docs/monetization/` and
does not authorize source retirement. The narrative files remain deliberately
unreplaced judgment prose; the sufficiency dossier names that boundary. The
cross-scenario journey gap is filed as Swarm Manager backlog item
`idea/experience-manager-cross-scenario-journeys` because the legacy issue
tracker is no longer available.

If the shared Test Genie `ui-health` provider again times out without a findings
payload, record it as the same infrastructure boundary only after all scenario
phases and direct experience validation pass.

The 2026-08-15 comprehensive rerun `20260815-072053-feebd711` passed 20/21
phases. CLI contracts passed after the importer flags were bound to
`source-path` and `source-mode`; the mobile tap-target and safe-area floor
findings are absent from this final run after the template/shell fix. The sole
remaining failure is experience reconciliation: 18 bindings are unresolved,
20 captures join zero bindings, and 16 claims cannot be proven from the
captured accessibility tree. `ui-health` itself completed with runtime render
success and only advisory debt. Do not promote the experience claims until the
capture provider can observe and reconcile the declared running surfaces.

The following requirements remain intentionally planned because this plan stops
before adoption or broader cross-scenario product work: `COMP-001` (per-offer
compliance review dates), `BOOK-001` (independent offer-book scoping), and
`GATE-007` (triggers reading another scenario's live state). The cross-scenario
journey contract is separately tracked as Swarm Manager item
`idea/experience-manager-cross-scenario-journeys`.

The post-fix run `20260815-090238-1e5e8a8f` passed experience reconciliation and
all product-side phases (20/21 overall). The remaining `ui-health` failure is a
provider `missing_dependency` result with zero findings; the ui-health API,
code-facts, and qdrant were subsequently confirmed healthy. This remains a
shared Test Genie/provider boundary rather than Offer Desk product debt.

### 2026-08-16 — current capture boundary

The historical template floor report is retained for provenance. The current
scenario shell now uses 44px compact controls, safe-area-aware bottom
navigation, and table-local mobile scrolling; both UI suites and accessibility
component tests pass. An earlier direct `experience-manager spec validate`
attempt hit the provider timeout, but the Phase 11 direct validations now pass
at L3 with zero required findings for both scenarios. Browser Automation
Studio capture remains unavailable because its CLI rebuild is blocked by
unrelated fleet-wide `golang.org/x/sys/unix` module drift; that boundary no
longer blocks the product-side experience contract.

### 2026-08-17 — remaining hand-rolled test clocks

**Symptom:** Several neighboring package and scenario tests still declare
private timers, tickers, or clock helpers instead of using the shared
`packages/api-core/schedule` contract.

**Root cause:** The fake-clock migration is intentionally scoped to Offer Desk
for this repair; the other call sites were discovered during the migration
and are follow-up work.

**Workaround:** Keep new time-dependent tests on `schedule.Clock` and use
`schedule.NewFake` where the test owns time. Do not add another local drill
clock or ticker abstraction.

**Real fix:** Migrate these six call sites in a dedicated follow-up, preserving
their existing behavioral assertions:

- `scenarios/search-hub/api/internal/evalsched/scheduler_test.go`
- `scenarios/meta-optimization-manager/api/internal/coverage/coverage_test.go`
- `packages/api-core/database/routed_lease_test.go`
- `packages/api-core/filerouting/routed_test.go`
- `packages/api-core/validationrun/lifecycle_test.go`
- `packages/api-core/retention/scheduler_test.go`

**Owner:** api-core maintainers and the owning scenario teams.

**Refs:** `packages/api-core/schedule/fake.go`,
`scenarios/offer-desk/api/handlers/offers/module_test.go`, and the fake-clock
migration phase in the monetization repair plan.
