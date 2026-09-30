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

- Rung: W1 (contract gate satisfied; implementation/evidence work remains)
- Compared goal: `release-ladder-offer-desk`; title `Offer Desk release-ladder contract`; description quotation: “Offer Desk is the typed commercial release-ladder catalog: it stores deliverables, release order, distribution ramps, metered streams, audiences, unlock relationships, and operator-owned promotion evidence.”
- Compared target: `serves_deliverable=web-console`.
- Evidence: the named-mention query now returns `release-ladder-offer-desk`; structural health remains supplemental evidence, not contract evidence.
- Measured: 2026-08-31 (23 of 26 requirements passing (88%), composite 65/100; maturity rung R0 unsatisfied, 0 test phases recorded)

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

### 2026-09-01 — intentionally open monetization vocabulary gaps

The release-ladder graph now reconciles the meters currently declared by the
fleet, but no `cloud_compute` meter exists for the planned cloud-compute
offering. The runtime `entitlement_tier` enum remains an orphaned fifth tier
vocabulary in the service schema. Both are outside this plan's ownership and
must not be silently invented or retired here.

**Owner:** monetization strategy / service-schema owner.

**Refs:** `packages/monetization-go/meter-inventory.json`,
`.vrooli/schemas/service.schema.json`, and `DECISIONS.md` D4.

### 2026-09-08 — persisted decision history has no read path

Five tables hold countable history that nothing outside the API can read, which
is also the five `measures.undeclared-substrate` findings holding this scenario
at measures L1:

| Table | Holds | Read path |
|---|---|---|
| `catalog_audit` | `node_id, actor, prior_status, next_status, reason, created_at` | none |
| `facts` | `value, observed_at, stale_after_days` | none (`gates-fact` writes only) |
| `triggers` | `fact_name, operator, threshold, expression` | none (`gates-trigger` writes only) |
| `migration_findings` | import reconciliation residue | none |
| `evaluations` | per-node verdicts | only the latest run summary, via `board-show` meta |

Consequence: the questions "what did we decide about this node, when, by whom
and why", "which facts are stale", and "which triggers are unmet" cannot be
answered by any command, though the data exists and is correct. This is why
`offer-desk.setpoint-read` reports `promotion-latency` as `pending_telemetry`:
the latency is derivable from `proposals.created_at` and `catalog_audit`, but
neither is readable.

Wanted, in dependency order: a `catalog-audit` read (with its governed binding),
then `gates-facts` and `gates-triggers` reads, then measures declared over all
five tables. Once `catalog_audit` is readable, `offer-desk.board-read` should
derive changes from the audit log instead of a caller-supplied baseline
snapshot, which removes caller state entirely.

**Owner:** offer-desk. Not a defect in the writes; a missing read surface.

**Refs:** `api/internal/catalog/schema.sql`; `measures-health validate scenario
offer-desk`; `.vrooli/program-runtime/setpoint-read.json` row
`promotion-latency`.

### 2026-09-08 — the requirement matrix is complete and unproven

`business-health matrix show offer-desk` reports 36 requirements and 16
operational targets with `complete=33, planned=3` — and **32 unproven claims**,
degraded with "no evidence artifacts yet (no suite runs, no snapshot)".
`scenarios/offer-desk/evals/` does not exist, so no suite and no approved floor
governs this scenario.

Consequence: `offer-desk-improve` §4 records that no corpus result may be
claimed. A cycle can report `setpoint-read: ok` while the contract behind it is
entirely unearned. Adding capability on top of this makes the reporting prettier
without making it truer.

Wanted: an `evals/` suite with an approved floor, then a requirements-sync
snapshot so the 33 completions are earned rather than asserted.

**Owner:** offer-desk, with target approval for any floor.

**Refs:** `business-health matrix show offer-desk --format markdown`;
`offer-desk-improve` §4 Golden corpora.

### 2026-09-08 — catalog-verify exceeds a usable budget

`offer-desk offers catalog-verify --source-path <path>` did not return inside 90
seconds while every other read binding answered in well under a second. Filed as
`knw-1788899099098954026`. `offer-desk.setpoint-read` therefore declines to call
it and reports `catalog-conformance` with reason `kernel_invoke_budget`, so the
conformance obligation is preserved but unmeasured. Cause not diagnosed.

**Owner:** offer-desk.

**Refs:** bug `knw-1788899099098954026`;
`.vrooli/program-runtime/setpoint-read.py` row `catalog-conformance`.

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

### 2026-08-17 — obligation denominator remains `sketch`, deliberately

**Symptom:** `docs/spaces/offers.json` and `SpaceService.GetProjection` both
report `denominator_confidence: sketch`.

**Root cause:** not a defect. The rationale in that file names its own exit
condition — one observed reconciliation cycle between `infra-health` and the
monetization team against an independent obligation inventory. That cycle has
not run, so no external roster exists to weight the three cells against.

**Workaround:** none needed. Consumers read the confidence field and the
rationale together; the value is honest as it stands.

**Real fix:** run the reconciliation cycle, then raise the confidence to match
what was actually observed. Raising it without that evidence would be exactly
the fabricated-denominator failure the fleet contract exists to prevent, so this
entry exists to stop a future agent from "fixing" it by editing the value.

**Owner:** monetization team plus `infra-health`.

**Refs:** `docs/spaces/offers.json`, `handlers/offers/module.go` GetProjection.

### 2026-08-17 — five drill fixtures remain in the live catalog

**Symptom:** `catalog-list` returns 33 nodes, five of which are test artifacts:
`Level 3 drill active offer`, `Phase 5 live operator node`, `Phase 5 refusal
drill node`, `Phase 5 multi-clause trigger node`, `Phase 6 proposal live drill`.

**Root cause:** they were created by Level 3 behavioural drills against the live
database rather than a scratch variant.

**Workaround:** they are `RETIRED` and, since the board-ranking fix, sort last on
every board read, so they no longer occupy the operator's attention surface.
`catalog-verify` does not flag them because the sources are judgment-only and
therefore `comparable=false`.

**Real fix:** needs an operator decision, not an implementation. The decision log
records that `catalog-merge` is *the one operation permitted to delete a node
row*, and these have no surviving counterpart to merge into. Removing them means
either extending that delete authority to provably sourceless records or
accepting them permanently as retired history. Both are legitimate; neither
should be chosen silently by an agent.

**Owner:** operator.

**Refs:** `DECISIONS.md` (2026-08-17 catalog-merge row).

### 2026-08-18 — experience capture flake on `dirty-state-guard/prompt-open`

**Symptom:** The Test Genie `experience` phase intermittently fails with two
`experience.capture_bindings_unjoined` errors: the accessibility capture for
component `dirty-state-guard` state `prompt-open` "joined zero of 4 declared
bindings".

**Root cause:** capture-side, not a contract defect. The same provider run
directly — `experience-manager spec validate offer-desk --json` — reports
`PASSED` at **L3 with zero findings** against the same running UI. The phase
passed in runs `20260818-034654-1a00e50b` and `20260818-035751-975366f3` and
failed in `20260818-040653-ebb621f9` with no intervening UI change; this
repair touched only the API, CLI, and docs. `prompt-open` requires driving the
component into a dirty state before capture, which is the timing-sensitive step.

**Workaround:** trust the direct validation. Per the 2026-08-15 entry above, a
capture-provider result is recorded as an infrastructure boundary only when all
other scenario phases and direct experience validation pass — both hold here
(20/21 with `experience` the sole failure, direct validation clean).

**Real fix:** make the `prompt-open` capture await the rendered prompt rather
than a settle delay, in the capture provider. Do **not** weaken the four
declared bindings to make the phase green — they are correct, and the direct
validator proves they resolve.

**Owner:** experience-manager / BAS capture, not this scenario.

**Refs:** `experience/components/dirty-state-guard.json`; runs
`20260818-040653-ebb621f9` (failed) vs `20260818-035751-975366f3` (passed).
