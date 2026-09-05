# experience/ - UX Contract

This folder is the scenario's **experience contract**. It is the design-first
sibling of `requirements/`: requirements say what the scenario does, while
`experience/` says what the UI must communicate.

## Where these specs actually are

The generated template starts at L0, declaring identity, route, purpose, and
operational-target linkage only. **Compute Manager is past that.** All five
page specs are authored to **L4**: each declares `priorities`, `states`,
`elements`, `claims` and `bindings`, and `journeys/` connects the pages into
two flows.

| Depth | What it adds | Here |
|---|---|---|
| L0 | Identity, route, purpose, operational-target linkage | All five pages |
| L1 | Communication priorities | All five pages |
| L2 | Elements, claims, bindings | All five pages |
| L3 | Explicit states | All five pages |
| L4 | Journeys connecting pages | `acquire-capacity`, `account-for-a-surprise` |

The L0-L4 ladder above is authoring depth. It is a different axis from the
maturity ladder `experience-manager spec validate` reports, which runs L0-L3
and measures whether the contract parses and cross-references cleanly; this
contract currently sits at that ladder's top rung with zero findings.

What depth does **not** mean is implementation. Every page is status `draft`
and every claim is tier `aspirational`, because machine-tier claims gate CI and
there are no stable selectors to check against yet. The `bindings` declared
here are the `data-testid` selectors the UI is to be built to match, not
selectors that exist in `ui/src/`.

## Contents

| Path | Holds |
|---|---|
| `index.json` | The page and journey registry, and the contract version |
| `pages/dashboard.json` | Inventory, `/` |
| `pages/instance-detail.json` | Instance, `/instances/:id` |
| `pages/request-capacity.json` | Request capacity, `/request` |
| `pages/findings.json` | Findings, `/findings` |
| `pages/settings.json` | Settings, `/settings` |
| `journeys/acquire-capacity.json` | The primary loop: request, watch, drain or destroy |
| `journeys/account-for-a-surprise.json` | The safety loop: notice, read, quarantine, decide |

The template's generated `notes` page spec and its registry entry were removed
by `template-manager detemplate compute-manager`, along with the rest of the
notes example domain. Nothing in this folder refers to it.

## Working rules

- Use `experience-manager spec validate compute-manager --json` after edits.
- Add machine-tier claims only once the UI has stable selectors and the claim
  can be checked by the experience phase. Manual claims need attestations with
  expiry; aspirational claims are useful intent but never gate.
- A `state-covered` claim must list every state the page declares, or say in
  its statement which subset it covers. A list that silently omits a declared
  state reads as coverage the page does not have.
- An `element-absent` claim asserts that the elements it lists are missing. A
  claim about a control that must be **present** lists no elements, or uses
  `element-present` instead. Compare `instance-no-pause-affordance` and
  `settings-never-shows-a-credential`.
- A journey step's `via` names an element declared on that step's page. A `via`
  that matches nothing is an unrouted transition, and the validator cannot see
  the difference between prose and a typo.
- Keep every independently meaningful async region bound to a
  `data-experience-surface` value that reports the canonical lifecycle
  vocabulary. Passive UI primitives inherit their parent's state.
