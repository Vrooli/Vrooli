# Component Library Gaps

## Purpose Of This Document

Record what Personal Planner needed from `react-component-library`
and could not link as-is. Every ejection and every missing asset is a debt the
library can pay back only if it is written down here; a silent local fork is
not. Keep this file short and factual, and raise each entry with the library
(`scenarios/react-component-library/docs/reference/`).

## Ejections

Components copied into this scenario with `react-component-library adoptions
eject <component-id> personal-planner --reason "..."`. One row per ejection.

**There are no real shell ejections yet.** No product UI code exists, so
nothing has been ejected from the navigated-console shell. The candidate
Observatory needs below are recorded as *potential* gaps to raise if the
shell cannot host them once the UI is built — they are not ejections.

| Component | Version | Reason | Revisit when |
|---|---|---|---|
| _none yet_ | | | |

## Missing Assets

Surfaces this scenario expects to need and that no current library asset
clearly fits. Named ahead of the UI build so the library can weigh them;
each becomes a real entry (or a scoped `shell-ejection` per
[EXPERIENCE.md](../concepts/EXPERIENCE.md)) only if the navigated-console
shell cannot host it.

| Need | Nearest library asset | Why it may not fit |
|---|---|---|
| Decorative full-bleed day/night scene band (the Observatory landscape occupying the lower working band, coordinated across Day/Night, never competing with text or inputs) | navigated-console shell background / theme surfaces | The archetype styles chrome and content surfaces; a full-bleed place-making art band below the work area may not be expressible as shell configuration alone. |
| Truthful proportional timeline / time-horizon component (block width ∝ real duration — a 30/60/90-min trio renders 1:2:3 — with a now-indicator and collision lanes for overlapping fixed events) | none (no honest time-geometry primitive identified) | Generic scheduler/agenda components typically size blocks to fit labels, which the Observatory design forbids (fixture F14); truthful geometry, a now marker, and collision lanes are load-bearing product requirements, not decoration. |

Both are candidate library gaps to raise with
`scenarios/react-component-library/docs/reference/` **if** the
navigated-console shell cannot carry them; do not fork silently.

## Known Validation Note

The removable `notes` example page
(`experience/pages/notes.json`) pins library component
`experience-surface@1.0.0`, whose canonical experience contract does not
currently resolve in this environment. This is **pre-existing
example-domain scaffold state**, not a defect in the authored specs: the
notes example is a copyable reference, and
`template-manager detemplate personal-planner` (a code-phase step) removes
`experience/pages/notes.json` along with every other fenced example, at
which point the unresolved pin disappears. No action is required from the
documentation; flagged here only so a validator sees it is expected and
transient.

## Cross-References

- `path:../guides/choosing-ui.md` — when to eject and when to record a gap
- `path:../concepts/UI-ARCHITECTURE.md` — where local components live
