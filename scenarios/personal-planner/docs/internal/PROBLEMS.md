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

### 2026-09-19 — Observatory bespoke controls remain standards debt

**Symptom:** UI-health still reports project-standards advisory findings for
raw color literals in the Observatory stylesheet and may report the aggregate
of anchored detail surfaces as raw dialogs.

**Root cause:** The approved Today oracle needs custom scenic colors, a compact
appearance segmented control, and anchored timeline disclosure geometry. The
interactive controls and Today draft/capture surfaces now compose over shared
Button/Dialog primitives; timeline and allocation details intentionally remain
anchored Popovers, while the standards aggregate still needs a narrower
recognizer.

**Workaround:** Shared Button/Input/Select/RadioGroup/Switch/Textarea/FormField,
PageHeader, SettingsList, and EmptyState are used everywhere their interaction
contract fits. The current UI-health receipt passes with project standards at
L3; runtime, manifest, interop, freshness, and PWA capabilities are L5.

**Real fix:** Keep the anchored Popover geometry while narrowing the standards
recognizer, then map the remaining Observatory palette to named token variables
and rerun UI-health plus the visual experience gate.

**Owner:** product implementation.

**Refs:** `ui/src/pages/DashboardPage.tsx`, `ui/src/pages/PlanPage.tsx`,
`ui/src/styles.css`, UI-health receipt `20260919-174124-c14934ac`.

## Work ladder

- Rung: W3
- Evidence: focused API, CLI, UI coverage, Focus/Goals/Review/Plan UI, Today, Workspace profile, availability, profile-backed capacity, weekly Review, Focus actuals, reflection, correction history, and goal prerequisite/rollup tests pass; focused receipt `20260919-122802-b345ac2d` passes unit/contracts/experience, measures receipt `20260919-124657-216206ea` passes domain coverage, and comprehensive receipt `20260919-122840-a6789844` passes 26/27 with security green. The only failed phase is portability; narrow receipt `20260919-123437-8b8b6355` identifies the shared trusted-base closure error.
- Blocker: portability requires a control-plane repair for the shared trusted-base closure `agent-manager -> workspace-sandbox`; this is outside Personal Planner's declared resource surface. Scenario-owned maturity debt remains UI component-adoption/template reporting, documentation snippet debt, and measure tier fallback. Product work still has provider recurrence operations, richer routine-demand capacity accounting, and broader hosted authorization scoping remaining.
- Note: focused Test Genie runs `20260919-090516-29764305` and `20260919-092558-f15f9dd0` both terminated `provider_unavailable`; local API/CLI/UI/build and lifecycle evidence is retained separately and is not promoted to a Test Genie pass. Later scenario-owned receipts `20260919-142916-abab670a` and its predecessors pass the requested unit/contracts/experience/measures phases.
- Measured: 2026-09-19

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Today timeline/capacity | Today reads accepted Calendar allocations and measured capacity; native routine occurrences are generated projections with skip-once/reschedule overrides, while work-item demand now conserves accepted planned minutes but routine demand is not yet an accepted-block state. | W3 partial. | Add routine-demand conservation, provider recurrence semantics, split/setup policies, and prove responsive geometry. |
| Goal milestone depth | Goal milestones now persist criteria, due dates, validated optional work-item links, prerequisite edges, and revision-safe open/complete state; milestone-mode goals derive completed/total progress. | W3 partial. | Add richer prerequisite editing/visual evidence and re-run the full scenario gate. |
| UI health/template adoption | Runtime rendering and responsive behavior are healthy, but the provider still reports the scenario as L0 because legacy template-slot and standard-component adoption contracts remain. The mobile text-entry zoom risk was corrected by enforcing the 16px floor at the mobile breakpoint. | W3 partial. | Migrate/remap the remaining declared slots and adopt the governed component contracts where they materially improve the Observatory surface; retain the provider report as a release prerequisite. |

## UX Issues

### UI design-system migration brief

**Intent:** Make the shell feel like one calm Observatory product across every route: translucent, theme-aware, editorial, and quiet rather than library-default.

**References:** `DESIGN.md`, the Today Day/Night mockups, and the shared React Component Library `AppShell`, `SettingsList`, and `ChromeTheme` contracts.

**Constraints:** Preserve navigation selectors, keyboard/focus behavior, 44px touch targets, safe-area handling, Auto/Day/Night semantics, and mobile bottom navigation. Keep route composition in the scenario; do not fork shared primitives.

**Scope:** Scenario-level shell/theme layout refresh using semantic tokens and existing primitives. Shared-library changes are reserved for a demonstrated contract gap.

**Current debt:** the generated token file still contains its canonical palette literals, and ThemeProvider retains browser-safe RGB fallbacks; Observatory scenic overlays now consume scene-local semantic variables. Runtime adoption is strong but the visual oracle and responsive contrast sweep remain open.

### 2026-09-19 — remaining visual-oracle and responsive sweep

**Status:** reduced; secondary-surface review remains open

**Observed:** Today now has desktop oracle captures and governed 390×844 Day/Night runtime evidence. Secondary surfaces still need a separate adversarial visual review, and the broader product/monetization scope is not complete.

**Addressed in this loop:** fixed scenic background behavior, theme chrome propagation, sidebar collapse/resize, timeline lane reuse and disclosure, shared form controls, mobile secondary-surface spacing, and Settings desktop hierarchy.

**Next evidence:** review secondary routes by surface and preserve any residual geometry/contrast issue as a targeted follow-up instead of treating the runtime receipt as visual completion.

### 2026-09-20 — comprehensive gate limitations after chrome fix

**Status:** open; scenario-owned branding repair complete

**Observed:** Comprehensive Test Genie run `20260920-000129-85e490c3` completed 25/27. Portability and performance failed, while the branding phase identified and then the scoped rerun `20260920-001027-c46e8d58` closed the HTML/manifest theme-color mismatch and missing dark fallback.

**Root cause:** Portability/performance are broader release-gate findings; the branding defect was a stale install-surface contract rather than a React runtime defect. The requirements-sync snapshot is also older than the scoped branding run because Test Genie owns that artifact.

**Workaround:** Treat the scoped branding receipt and direct brand-manager validation as the authoritative proof for the chrome repair. Do not call the comprehensive run a release certification; retain portability/performance and evidence freshness as explicit follow-ups.

**Refs:** `ui/index.html`, `ui/public/public/site.webmanifest`, Test Genie runs above, and `coverage/requirements-sync/latest.json`.

### 2026-09-21 — final scenic certification remains open

**Status:** reduced; procedural sky and responsive structure are verified

**Observed:** The shared sky, route-start behavior, and short-landscape Plan chrome now have live evidence and regression coverage. A single final matrix covering every route in Day/Night desktop/mobile, active and overtime Focus, and long-content user scrolling has not yet been completed.

**Addressed in this loop:** removed decorative star swirls; introduced one deterministic natural star/Milky Way renderer; limited the date-selected zodiac figure to Goals; made the Day balloon visible; stabilized route-entry scroll during asynchronous layout; and replaced short-landscape overflow with compact governed view selection.

**Evidence:** 40 files / 259 UI tests; Test Genie `20260922-024852-e6e837d7` (unit + L3 experience pass) and `20260922-025005-9c6f4e82` (L5 UI-health pass); Browser Automation Studio `3ab9ae92-2837-4ed2-ac39-b8d764d95d98` at the former 844×390 failure viewport.

**Next evidence:** execute the final visual matrix and preserve only concrete route/state defects as targeted follow-ups.

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## 2026-10-02 — runtime UX defects for design review (unfixed)

### PP-UX-01: Default Countdown cannot start, with silent failure

**Observed/rendered/executed:** isolated shadow Focus shows Countdown by
default; Start focus sends mode=countdown and returns400 invalid_argument.
Exact independent API response: `mode: must be open, pomodoro, timed, or untimed`.
Local Chrome repeats the failure with no visible error/AX state change. BAS
network telemetry records repeated rejected calls and independent DB reads
show no session. **Source:** FocusPage passes timerMode directly; focus/service.go
lines43–48 rejects countdown. Start-card branch has no mutation error renderer;
error copy is nested under the existing-session branch.

**Workaround verified:** select Open timer then Start focus. Supported loop
BAS `c4b066df-6b32-40c2-b9d4-367e89368f75` passes with DB readback.
**Impact:** primary default focus journey unusable; silent failure masks cause.
**Status:** open; no product fix authorized by this access/inspection task.

### PP-UX-02: Accept placement waits indefinitely without saving

**Observed/rendered/executed:** synthetic45min09:00 preview is feasible;
Accept stays Placing…/Checking…, no allocation persists. BAS
`e0a45fc2-687a-4222-a33a-f28720f0ef39` failed completion assertion; local Chrome
reproduces. `timeout 8 personal-planner --instance shadow calendar apply-placement
--proposal-id 7a1fa32d-f675-44eb-93a5-c623d3390071 --revision 1
--idempotency-key ux-20261002-independent --json` exited124 without output.

**Source hypothesis (not a stack-trace proof):** main.go configures
MaxOpenConns1; calendar/sqlite.go ApplyProposal begins a transaction then calls
r.routineBusy outside the transaction, which reaches ListRoutines through r.db.
The sole connection is held by the transaction. This can explain the self-wait.
**Impact:** daily planning acceptance, split-session and accepted-capacity probes
blocked; later broad BAS text assertions falsely looked successful.
**Status:** open; inspect transaction-bound reads in a separately authorized fix.

### Missing-path observations and visual questions

No explicit More to do or date-level allocation→child session split control
was found. FLOWS.md describes these as target semantics; absence in this bounded
inspection is a design/implementation coverage gap, not a demonstrated broken
existing control. Empty mobile Today puts the large empty timeline before the
capture card, while enabled Start focus is visible without work; evaluate
whether first-use guidance communicates spontaneous focus versus capture well.
Review floors55active seconds to0min; honest accounting persists, but presentation
of sub-minute activity may merit design review. No changes made.

## 2026-10-02 — second-sweep runtime findings (unfixed)

### PP-UX-03: Repeated manual capture creates duplicate tasks

Executed Today name-only capture → double Save task in BAS
2423970a-f966-4486-b628-b14bc21f6ef4. Independent SQLite read shows two open
unknown-effort rows with identical title Planner review second name-only:
59d3b767-7634-499a-88cf-fe690fd6c2bc and cfc64f14-9702-4bbd-a70c-b2eeb0b662f1,
created05:10:29.368Z. Expected one accepted submission; actual two records.
Impact medium: repeat clicks duplicate backlog demand. No live data affected.
Root cause not isolated; pending-button disable alone did not prevent this race.

### PP-UX-04: Placement duration's native validity contradicts default45

Runtime DOM reports45 invalid, nearest41/46; source PlanPage.tsx:172 combines
min1 with step5. Low-severity constraint inconsistency. Placement is a div with
type=button preview,so native form submission validation does NOT block its
preview; first-pass45minpreview was feasible. Do not describe this as the cause
of PP-UX-02 or as a demonstrated visible error banner.

### PP-UX-05: A persisted milestone makes its list wait indefinitely

Goal1ffb8222-988d-4084-bb5d-a4c00492c9ed and target date2026-10-09 persisted;
milestone066fba5a-15c4-4d9c-9156-b48dcf294d32 also persisted as open revision1.
BAS a3df673b-1726-4471-849f-1492e8302f26 failed its scoped list assertion; UI
showed Adding…/Proofs loading after reopen. Independent read-only POST
http://localhost:17778/vrooli.personal_planner.v1.goals.GoalsService/ListMilestones
with that goalId timed out6.02s. No duplicate creation retry performed.
High impact: a real stored proof point cannot be read through the normal list.
Source hypothesis: sqlite.go:105–125 keeps outer rows open while calling
loadPrerequisites:217–222 through r.db; main MaxOpenConns1. This explains a
connection self-wait but is not stack-trace proof. Browser disposal/cancellation
releases the request; shadow work API responds afterward. No restart needed.

### PP-UX-06: Stored focus notes and interruption reasons render blank

Daily Review displays FOCUS NOTES/PAUSE PATTERNS headings and empty content,
despite note Synthetic UX verification note and event reason interrupted in
shadow storage. Independent REST GET /api/v1/focus/notes?date=2026-10-02 returns
SessionID/LocalDate/Note/UpdatedAt; pause-reasons returns ID/Reason/etc.
focus.ts:80–100 reads session_id/local_date/note/reason instead. Go SessionNote
and PauseEvent in focus/types.go:31–39 have no JSON field tags. This source and
executed response mismatch directly supports the blank render diagnosis.
Medium impact: recorded reflection/interruption context is hidden in review.
The first pass proved note persistence and save acknowledgment, not correct
Review content display. No note loss/delete was observed.

### Wider rendered/design gaps, not newly proven implementation failures

Long title wraps through most of desktop NextActionCard and truncates on phone;
mobile Today puts its empty timeline before the useful next action. Global
capture tomorrow30m retains tomorrow in the title,shows30min but no interpreted
date; cancelled without applying. Outlook shows known45min but omits the three
unknown-effort task count from its observed explanation. Full task detail is a
small read-only description/source dialog; date-level split,More to do,first-use
setup,what-if,search,learning-controls,data-portability and share/guest surfaces
were absent/not reachable in this bounded six-route inspection. Domain code is
not proof that these user journeys exist. Use catalogue dispositions in TESTING.

### PP-UX-07: Duplicate Pause conserves state but gives ambiguous failure feedback

Original BAS c4b066df-6b32-40c2-b9d4-367e89368f75 trace, recovered without new
mutations: both PauseFocus requests used expectedRevision "1" and the same
session. At04:44:48.224Z the first returned200, paused revision2, active53/wall53.
At04:44:48.225Z the second returned409 aborted: focus session changed; refresh
before retrying. Exactly one interrupted pause event persisted. focus-paused.jpg
shows State paused / Evidence Server saved together with "That transition did
not save. Nothing was assumed." Reopening retained paused and cleared the alert.
Conservation passes this duplicate-command case; medium UX feedback defect:
the message does not identify the rejected repeated Pause and obscures the
successful transition. This is not evidence of a failed original save or lost time.
Exact requests/responses and original trace hash: first-double-pause-exact-responses.json.

### PP-UX-08: Fixed Open timer guidance conflicts with selected Countdown draft

Actual focus-ended.jpg and focus-note-saved.jpg show Countdown selected while
the guidance above says Open timer / Stop when the work stops. Source
ui/src/components/FocusStartGuidance.tsx hardcodes those words; FocusPage.tsx
initializes the new timer draft as countdown after reload. The ended server
session remained mode open, revision4. This is low-severity misleading start
mode guidance, not an executed backend mode change or a completed-session
summary. Note-save acknowledgment persists; Review display remains PP-UX-06.
No extra focus session, product repair or mutation was needed to establish this.

### Capability prerequisite and product correctness are separate results

Existing secured access, independently verified shadow routing, bounded fixtures,
BAS replay definitions, actual screenshots/video and reproducible readbacks are
established for a buyer/design audit. Open Planner defects do not by themselves
invalidate this access/isolation/capture prerequisite. Full core-loop product
acceptance remains blocked by placement/milestone waits; Countdown400, duplicate
capture and reflection display/feedback defects remain open. Native200% zoom,
full accessibility matrix and consuming-executor recording playback remain
unverified. P01–P42 dispositions are bounded evidence, never full-family passes.
