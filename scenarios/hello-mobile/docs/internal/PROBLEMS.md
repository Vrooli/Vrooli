# Problems — Hello Mobile

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

### 2026-08-17 — Integrated Test Genie validation is queued without a terminal verdict

**Symptom:** Fresh run `20260817-035422-beba6cf8` remains `queued/active` beyond
its estimated duration, and the waiter returns without terminal JSON.

**Root cause:** Shared Test Genie scheduling/worker state is degraded while
another agent changes the test-genie scenario; this is not a hello-mobile
application failure.

**Workaround:** Use the passing targeted UI, API, CLI, build, and requirements
checks as current evidence. Do not infer an integrated PASS from the queued run.

**Real fix:** Restore Test Genie worker scheduling and complete the retained
server-owned run, then reconcile its findings.

**Owner:** Test Genie maintainers.

**Refs:** Run `20260817-035422-beba6cf8`; plan mobile-delivery-ramps.

### 2026-08-17 — Phase 9 retry is queued by shared Test Genie admission

**Symptom:** Fresh run `20260817-094636-3bbc63e0` remains queued with
`last_progress_at` equal to creation after its waiter was allowed to run; the
parallel Android and desktop attempts were rejected as admission-saturated.

**Root cause:** Test Genie caller-queued capacity is saturated during the
parallel Test Genie workstream. The fixture itself passes its targeted
workflow, API, UI, build, and requirements checks.

**Workaround:** Keep the targeted evidence as the current implementation gate
and preserve the queued run identity. Do not infer an integrated PASS.

**Real fix:** Restore Test Genie worker scheduling, let the retained run reach
terminal state, and reconcile any findings manifest.

**Owner:** Test Genie maintainers.

**Refs:** run `20260817-094636-3bbc63e0`; plan
`mobile-delivery-ramps-implement-scenario-to-ios-complete`.

### 2026-08-17 — Integrated routed-storage proof cannot acquire storage-manager

**Symptom:** Hello Mobile run `20260817-120004-44f9247e` reached 19/21 phases,
but the storage phase failed and the workflow phase reported that routed test
isolation could not resolve the `storage-manager` API port.

**Root cause:** The shared Test Genie/storage-manager runtime was unavailable
during the server-owned run. This is infrastructure admission/setup state, not
an application-owned Hello Mobile storage implementation defect.

**Workaround:** Keep the scenario-local API, UI, build, requirements, and
experience-spec validation as the actionable gate, and use the integrated run
as evidence that the notes region binding is now covered. Do not claim a full
integrated PASS until routed storage is available.

**Real fix:** Restore Test Genie's routed-storage setup and rerun the retained
mobile validation suite.

**Owner:** Test Genie/storage-manager maintainers.

**Refs:** Run `20260817-120004-44f9247e`; plan
`mobile-delivery-ramps-implement-scenario-to-ios-complete`.

### 2026-08-17 — Shared lifecycle cannot start the fixture

**Symptom:** Fresh Phase 9 run `20260817-135349-efb7c62c` failed before phase
execution because lifecycle startup could not allocate ports.

**Root cause:** The shared resource migration currently lacks
`resources/sherpa-onnx/resource.json`; lifecycle startup fails before the
hello-mobile API or UI can be exercised.

**Workaround:** Keep the targeted hello-mobile API, UI, build, requirements,
and experience checks as the actionable evidence. Do not repair the shared
resource manifest from this scenario.

**Real fix:** The control-plane/resource migration owner must restore the
managed-resource manifest and verify lifecycle startup, then rerun the
server-owned suite.

**Owner:** control-plane/resource migration maintainers.

**Refs:** Run `20260817-135349-efb7c62c`; missing
`resources/sherpa-onnx/resource.json`; plan
`mobile-delivery-ramps-implement-scenario-to-ios-complete`.

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
