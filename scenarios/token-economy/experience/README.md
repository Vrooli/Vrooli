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

Use `experience-manager spec validate token-economy --json` after edits.
Machine-tier claims should only be added when the UI has stable selectors and
the claim can be checked by the experience phase. Manual claims need
attestations with expiry; aspirational claims are useful intent but never gate.

Every independently meaningful async region should expose a stable
`data-experience-surface` and canonical lifecycle state. Passive UI primitives
inherit their parent state.
