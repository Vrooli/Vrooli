# Problems — Personal Planner

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

### 2026-09-18 — notes example pins an experience contract that does not resolve here

**Symptom:** `experience-manager spec validate` reports one error, scoped
to the removable `notes` example domain. The example page pins the library
component `experience-surface@1.0.0`, whose canonical experience contract
does not resolve in this environment.

**Root cause:** The generated `notes` worked example references a library
experience contract (`experience-surface@1.0.0`) that is not resolvable
here. The error is inherent to the shipped example, not to any real
product domain.

**Workaround:** None needed for docs-only work — the error is isolated to
the example domain and does not affect the real `health` domain or the
product design. Treat the single validation error as expected while the
example is still present.

**Real fix:** Remove the notes example. `template-manager detemplate
personal-planner` deletes the worked example (a code-phase step); the
validation error clears once it is gone.

**Owner:** unassigned (resolved during the example-domain-removal
code phase).

**Refs:** `experience-manager spec validate`; `template-manager detemplate`;
the "example-domain-removed" orientation gate below.

### 2026-09-19 — workspace availability exists; visual and richer scheduling release work remain deferred

**Symptom:** The product still has open release work after the initial
vertical slices: richer schedule operations, goal milestones/links, and the
responsive/accessibility visual sweep.

**Root cause:** The Observatory Today composition, `work` slice, persisted
`focus` session, `goals`, Review, Calendar allocation/range slices, the
revisioned workspace planning profile, and local weekday availability windows
with protected/extra date exceptions now exist, as do native fixed and flexible
routine definitions with deterministic bounded occurrence expansion. Provider
projections, provider recurrence semantics, richer Goals milestone/link depth,
and responsive/contrast evidence remain open. The visual oracle has had a
desktop Day/Night checkpoint, but the responsive and accessibility release
sweep is open.

**Workaround:** Today, Plan, and Review share the accepted Calendar read model
and fall back honestly when work, schedule, or focus data is unavailable.
Focus records real service transitions, Goals records explicit manual
progress, and Review reports measured planned/focus/goal facts without
inventing unrecorded coverage.

**Real fix:** Add read-only provider projections and recurrence adapters,
provider recurrence semantics, richer capacity accounting for
routine demand, richer goal milestone/link contracts, complete responsive and
accessibility evidence, then re-run the full scenario gate.

**Owner:** unassigned (product code phase).

**Refs:** `ui/src/pages/DashboardPage.tsx`; `template-manager detemplate`;
[`PROGRESS.md`](PROGRESS.md) for the remaining-work summary.

### 2026-09-19 — notes example removed after Work slice became green

**Symptom:** The fenced notes domain and its unresolved experience contract
were present during the initial implementation gate.

**Root cause:** The generated scenario retained the template example until a
real replacement domain had passing evidence.

**Workaround:** None; `template-manager detemplate personal-planner` was run
after the Work proto/API/CLI/UI slice passed its focused checks.

**Real fix:** Complete the remaining product-owned domains and re-run the
full experience and Test Genie gates.

**Owner:** product implementation.

**Refs:** `template-manager detemplate personal-planner`; `docs/internal/TESTING.md` outcome-evidence inventory.

### 2026-09-19 — correction provenance read path is now bounded and explicit

**Symptom:** The correction ledger originally had no read path, so users could
not inspect why an actual changed and Measures Health could not cover it.

**Root cause:** Focus correction wrote durable before/after provenance, but
the Focus contract only listed current manual actuals.

**Workaround:** The new bounded `focus corrections` read is available with
optional actual/date filters and a default limit of 100.

**Real fix:** Completed for the current manual-actual slice. Future work still
needs overlap resolution, segment correction, and export/retention semantics.

**Owner:** product implementation.

**Refs:** `api/internal/focus/schema.sql`, `api/internal/focus/sqlite.go`,
`packages/proto/schemas/personal-planner/v1/focus/focus.proto`,
`cli/manifest.json`.

### 2026-09-19 — portability is blocked by shared trusted-base closure

**Symptom:** The comprehensive Test Genie gate remains 26/27; the narrow
portability run fails before scenario validation starts.

**Root cause:** Scenario Dependency Analyzer reports the shared trusted-base
closure `agent-manager` requires `workspace-sandbox`, which is invalid in the
repository environment. Personal Planner does not declare that dependency.

**Workaround:** Use the passing scenario-owned phases and record the exact
trusted-base error; do not hand-edit approved dependency state.

**Real fix:** Repair the shared trusted-base dependency closure through the
Scenario Dependency Analyzer owner, then rerun portability.

**Owner:** control-plane/dependency-governance owner.

**Refs:** Test Genie runs `20260919-122840-a6789844` and
`20260919-123437-8b8b6355`; `docs/package-governance.md`.

### 2026-09-19 — provider connection seam is real but intentionally fixture-only

**Symptom:** Settings and CLI can show, refresh, and disconnect a read-only
calendar connection, but no external provider account can be authorized.

**Root cause:** The scenario had no integrations domain. This slice adds the
provider-neutral persistence and revision contract plus a synthetic adapter so
the boundary is testable without inventing credentials or a user's provider.

**Workaround:** Use manual planning or the clearly labeled synthetic fixture
for UI and contract verification. Synthetic busy intervals now project into
Today capacity; the Settings copy explicitly says live OAuth and credential
storage are not configured.

**Real fix:** Wire the tested Google adapter to the credential-owner/OAuth
callback path, durable calendar/cursor persistence, and stale/outage semantics;
then verify it with a real read-only account before declaring the R1
integration gate complete.

**Owner:** product code phase; external credentials are a deployment concern.

**Refs:** `api/internal/integrations`, `ui/src/api/integrations.ts`,
`ui/src/pages/SettingsPage.tsx`, source plan §16.

## Work ladder

- Rung: W3
- Evidence: focused API, CLI, UI coverage, Focus/Goals/Review/Plan UI, Today, Workspace profile, availability, profile-backed capacity, weekly Review, Focus actuals, reflection, correction history, and goal prerequisite/rollup tests pass; focused receipt `20260919-122802-b345ac2d` passes unit/contracts/experience, measures receipt `20260919-124657-216206ea` passes domain coverage, and comprehensive receipt `20260919-122840-a6789844` passes 26/27 with security green. The only failed phase is portability; narrow receipt `20260919-123437-8b8b6355` identifies the shared trusted-base closure error.
- Blocker: portability requires a control-plane repair for the shared trusted-base closure `agent-manager -> workspace-sandbox`; this is outside Personal Planner's declared resource surface. Scenario-owned maturity debt remains UI component-adoption/template reporting, documentation snippet debt, and measure tier fallback. Product work still has provider recurrence operations, richer routine-demand capacity accounting, and broader hosted authorization scoping remaining.
- Note: focused Test Genie runs `20260919-090516-29764305` and `20260919-092558-f15f9dd0` both terminated `provider_unavailable`; local API/CLI/UI/build and lifecycle evidence is retained separately and is not promoted to a Test Genie pass.
- Measured: 2026-09-19

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Today timeline/capacity | Today reads accepted Calendar allocations and measured capacity; native routine occurrences are generated projections with skip-once/reschedule overrides, but are not yet demand/accepted-block state. | W3 partial. | Add routine-demand conservation, provider recurrence semantics, and prove responsive geometry. |
| Goal milestone depth | Goal milestones now persist criteria, due dates, validated optional work-item links, prerequisite edges, and revision-safe open/complete state; milestone-mode goals derive completed/total progress. | W3 partial. | Add richer prerequisite editing/visual evidence and re-run the full scenario gate. |
| UI health/template adoption | Runtime rendering and responsive behavior are healthy, but the provider still reports the scenario as L0 because legacy template-slot and standard-component adoption contracts remain. The mobile text-entry zoom risk was corrected by enforcing the 16px floor at the mobile breakpoint. | W3 partial. | Migrate/remap the remaining declared slots and adopt the governed component contracts where they materially improve the Observatory surface; retain the provider report as a release prerequisite. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
