# experience/ - UX Contract

This folder is the scenario's generated **experience contract**. It is the
design-first sibling of `requirements/`: requirements say what the scenario
does, while `experience/` says what the UI must communicate.

This contract is authored to **L4**. Every non-deprecated page declares
identity, route, purpose and operational-target linkage (L0), communication
priorities (L1), elements, claims and bindings (L2), and explicit states with
`setup` blocks (L3); journeys connect them (L4).

Two consequences follow, and both are binding on implementation:

- **`bindings` is the selector contract, not a record of what was built.** The
  `testid` values were chosen before the UI existed. Implement them as written
  rather than renaming the spec to match the code — the bindings block is the
  one section a pure refactor or restyle may touch, and only to re-ground an
  element that moved.
- **Each state's `setup.query.fixture` names a fixture the UI must serve.** A
  state with no reachable fixture cannot be captured, so its claims silently
  never run. Add the fixture with the feature, not afterwards.

Use `experience-manager spec validate money-ledger --json` after edits.
Machine-tier claims should only be added when the UI has stable selectors and
the claim can be checked by the experience phase. Manual claims need
attestations with expiry; aspirational claims are useful intent but never gate.

The `statements` page carries operational targets that had no surface before it existed; check `prd_refs` coverage across the whole contract before adding another page.

The generated `notes` page is part of the removable example domain. Running
`template-manager detemplate money-ledger` removes its page spec and registry
entry with the rest of the notes example.

The Notes example also demonstrates the generated semantic foundation: its
`notes` region is bound to `data-experience-surface="notes"` and reports the
canonical lifecycle vocabulary. Keep this boundary for every independently
meaningful async region; passive UI primitives inherit their parent state.
