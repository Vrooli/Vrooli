# Problems — Money Ledger

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

### Scaffold health gate was superseded by the implementation run

The scenario now starts through `make start`, reports healthy, and has API,
CLI, and UI evidence. The remaining completion blocker is live experience
capture reconciliation, not scaffold boot.

### The reversing-entry workflow must exist before real data lands

`JRNL-004` makes the journal append-only, which is correct and also unforgiving: without a working correction path, the first mistyped amount has no remedy and the temptation to add an edit endpoint becomes overwhelming. Build the reversing entry in the same slice as the journal, not after it.

### Position is the least useful thing to build first, despite being the most interesting

Runway is the number the first user cares about, but its most important inputs are still unavailable — there are no subscribers, and recurring-revenue telemetry does not exist yet. Building `position` early would ship a calculator over hand-entered values and prove very little. The sequencing in the PRD puts it fourth for that reason.

### Adapter credentials are a real liability, not a checkbox

No adapter that requires a password may be built. Bank access is via aggregator API or file export only. This constraint is stated in the PRD non-goals and in INTEGRATIONS, and it should be re-read before any adapter work rather than rediscovered.

### The extraction trigger will arrive quietly

`ADP-001` names three conditions under which adapters become their own scenario. None will announce itself; the third adapter will simply feel like the second. Re-read that requirement at each adapter addition and record the decision in `DECISIONS.md` either way.

### The generated shell fails two of its own accessibility floors on mobile

Discovered 2026-08-13 during experience validation, and **reproduced identically in both scenarios generated from `react-vite` v1.6.5**, so this is a template defect rather than anything either scenario did:

- `floor-tap-target-size` — the theme control renders ~79×30px on mobile, under the 44px minimum.
- `floor-safe-area-tap-targets` — an interactive target on the dashboard overlaps the mobile unsafe bottom area.

This matters more than a normal placeholder complaint because `docs/START-HERE.md` lists the shell's *"fixed safe-area bottom navigation on mobile"* as durable infrastructure to preserve, not as illustrative content. The floor it is supposed to guarantee is the one failing.

It is invisible until a scenario's experience contract is real enough to be checked against a running UI, which is why a freshly generated scenario reports clean. Fix it in the template rather than per-scenario, or every future scenario inherits it.

**Reproduce:** `make start`, then `experience-manager spec validate <scenario> --json`.

**Filed:** against `template-manager` (`react-vite` v1.6.5), not against this scenario. Do not patch the shell here — a per-scenario fix hides the defect from every future generation.

### The financial cutover is ordered, and the source data exists nowhere else

`CTR-006` moves `shared/operator-inputs.json` and `FINANCIAL_MODEL.md`'s snapshot schema into this scenario. The ordering rule is the same one Offer Desk states for the catalog and is just as easy to get wrong under time pressure: **import, verify field-by-field with counts, then retire the source.**

Two things make this worse here than on the catalog side. The operator-inputs file is **operator-supplied and not regenerable** — a lost field is not recoverable from any other system, unlike a markdown catalog whose content is at least reconstructible from history. And the file is smaller and looks trivial, which is exactly the property that invites skipping the verification step.

Fields marked `pending-operator` in the source must import as **absent, not as zero**. A zero burn category is a confidently wrong figure that will propagate into runway and read as good news.

### The team must be paused for the financial cutover too

Offer Desk records this for the catalog import. It applies here for the same reason and is easy to miss because this cutover touches one JSON file rather than 22 markdown ones: `financial-tracker` writes to `operator-inputs.json`'s neighbourhood on its heartbeat, and a running team during the import produces a verification mismatch that looks like an importer bug.

### Goal thresholds must not be re-typed as prose

The default-alive rule (three consecutive months, 1.25 buffer), the services-capacity guardrail (30% for three consecutive weeks) and the services-trap signal (two consecutive months) are the clearest instances of the defect this scenario exists to remove. They must land as declared goals under `POS-002`. Copying them into a settings page description, a comment, or a README would reproduce the original problem inside the replacement — and would be nearly invisible in review, because the text would look like documentation rather than state.

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
behavioural drills produce durable evidence. The inherited Phase 1 log did not
contain the required start-of-plan protected-tree hashes, so the current end
hashes are recorded in `PROGRESS.md` without claiming a historical comparison.
Direct experience validation is now clean after BAS stabilized, but the full
Test Genie `ui-health` provider still fails without a findings payload.

### 2026-08-14 — final validation boundary and intentional deferrals

The requirement registry is now evidence-backed: `vrooli scenario requirements
validate money-ledger --json` passes at L3, with 16 requirements complete and 10
explicitly planned (including the future commerce adapter and tax-category work).
The fresh comprehensive run `20260814-034428-59a384b3` passed 20/21 phases. The
remaining failure is not a Money Ledger finding: the shared Test Genie
`ui-health` execution provider times out without returning a findings payload.
Static-only UI-health reports zero required findings, and direct experience
validation reports zero findings. The protected-tree end hashes are recorded in
`PROGRESS.md`; no historical start hash was available from Phase 1.
