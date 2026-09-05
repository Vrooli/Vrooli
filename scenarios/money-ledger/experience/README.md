# experience/ - UX Contract

This folder is the scenario's generated **experience contract**. It is the
design-first sibling of `requirements/`: requirements say what the scenario
does, while `experience/` says what the UI must communicate.

This contract is authored to **L4**. Every non-deprecated page declares
identity, route, purpose and operational-target linkage (L0), communication
priorities (L1), elements, claims and bindings (L2), and explicit states with
`setup` blocks (L3); journeys connect them (L4).

That sentence was false for `settings` until 2026-08-13 — the page was `active`
in the registry with no elements, claims, bindings, or state `setup` blocks, so
it gated nothing while appearing complete. It is now authored to L3 like every
other active page. **If a page cannot be authored to L2, mark it `draft` rather
than leaving it `active`**; an active page with no claims is invisible to the
experience phase and reads as coverage that does not exist.

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

## Known gaps in this contract

Recorded so they are chosen rather than discovered.

- **Mobile is now explicitly scoped.** Every page carries desktop/mobile
  viewport coverage, the journal create-event claims are viewport-scoped, and
  `record-a-source-with-no-api-mobile` covers the standing cash-sale path that
  makes `OT-P0-006` a first-class adapter rather than a desktop fallback.
- **First run is explicit.** `first-run.json` covers the empty-to-first-event
  path so an installed scenario does not begin in an unexplained mid-product
  state.
- **No cross-scenario journey exists, and the schema cannot express one.**
  `JourneyStep.page` resolves within this scenario, so the pair's headline claim
  — "this offer is active and has earned nothing" — has no journey on either
  side. Neither contract is at fault; the capability is missing. Raise it against
  `experience-manager` rather than working around it with a page that pretends to
  own both halves.

The product surfaces use semantic regions such as `position`, `journal`, and
`offers`, each bound to the corresponding page contract. Keep this boundary
for every independently meaningful async region; passive UI primitives inherit
their parent state.
