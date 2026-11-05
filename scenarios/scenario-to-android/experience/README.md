# experience/ - UX Contract

This folder is the scenario's generated **experience contract**. It is the
design-first sibling of `requirements/`: requirements say what the scenario
does, while `experience/` says what the UI must communicate.

The template starts at **L0** on purpose. Each page spec declares identity,
route, purpose, and operational-target linkage only. As the product becomes
real, raise the depth:

- L1: add communication priorities.
- L2: add elements, claims, and bindings.
- L3: add explicit states.
- L4: add journeys that connect pages.

Use `experience-manager spec validate scenario-to-android --json` after edits.
Machine-tier claims should only be added when the UI has stable selectors and
the claim can be checked by the experience phase. Manual claims need
attestations with expiry; aspirational claims are useful intent but never gate.

The example `notes` domain has been detemplated; every page here is real.

Each page declares `elements` (intent-level, role + accessible name) and a
`bindings.elements` map from element id to `data-testid`. Bindings are the
volatile layer — the only section a pure restyle should touch — and they are the
selector source of truth for `bas/` case scaffolding. Keep an element's id stable
even when its testid changes.

Machine-tier claims require both an element and its binding; a `custom` claim can
only ever be `manual` or `aspirational`. Prefer a typed claim
(`element-present`, `element-absent`, `state-distinct`, `reading-order`,
`visible-without-scroll`) over prose whenever the property is observable in the
accessibility tree — a machine claim gates CI, a manual one does not.

Keep the `data-experience-surface` boundary for every independently meaningful
async region; passive UI primitives inherit their parent state.
