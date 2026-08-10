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

Use `experience-manager spec validate device-control --json` after edits.
Machine-tier claims should only be added when the UI has stable selectors and
the claim can be checked by the experience phase. Manual claims need
attestations with expiry; aspirational claims are useful intent but never gate.

## Current depth: L2 across all product pages

The eight product pages — `fleet`, `device-detail`, `session`, `flows`,
`run-review`, `strategies`, `agent`, `settings` — carry priorities, states,
elements, claims, and bindings, plus two journeys. The UX design they encode is
in [`../docs/concepts/UI-ARCHITECTURE.md`](../docs/concepts/UI-ARCHITECTURE.md).

Three conventions were established while writing them; keep them when adding a
page.

**Declare the generic state as well as the specific one.** DESIGN.md's UX-state
contract requires `loading`, `empty`, `partial`, `stale`, and `request-error` on
every page. Domain states such as `frame-unavailable`, `redaction-pending`, and
`bound-exhausted` are *better* product thinking and are kept — but they are
added alongside the generic vocabulary, never instead of it, so a user always
gets a legible answer and an operator gets a precise one.

**Every non-default state declares a `setup`.** A machine claim scoped to a
state with no deterministic setup reports `claim_unverifiable` and checks
nothing. Data-dependent states use the reserved `fixture` query parameter named
after the state id; the contract and its test-mode gating are documented in
[`../docs/concepts/UI-ARCHITECTURE.md`](../docs/concepts/UI-ARCHITECTURE.md#deterministic-state-capture).

**Pick the tier honestly.** Presence, order, keyboard reachability, and
state-distinctness are `machine`. Judgements that a checker cannot make —
whether three dispositions stay perceptually distinct, whether a gap report
*reads* as a successful answer rather than a failure, whether a disposition was
rendered verbatim rather than paraphrased — are `manual`, and say why in
`rationale`. Do not inflate a perceptual judgement to `machine` to make a
number go down.

Bindings name `data-testid` values the UI does not implement yet. That is
deliberate: they are the selector SSOT for the build and for `bas/` case
scaffolding. Until the markup exists, reconciliation reports
`capture_unavailable` at INFO — the contract is declared, not yet met.

The generated `notes` page is part of the removable example domain. Running
`template-manager detemplate device-control` removes its page spec and registry
entry with the rest of the notes example.

The Notes example also demonstrates the generated semantic foundation: its
`notes` region is bound to `data-experience-surface="notes"` and reports the
canonical lifecycle vocabulary. Keep this boundary for every independently
meaningful async region; passive UI primitives inherit their parent state.
