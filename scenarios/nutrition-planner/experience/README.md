# experience/ — UX contract

This folder is the scenario's **experience contract**: the design-first sibling
of `requirements/`. Requirements say what Nooch does; `experience/` says what
each surface must communicate, which states it has, which elements carry that
meaning, and how those elements are bound to the UI. The prose companion is
[`../docs/concepts/EXPERIENCE.md`](../docs/concepts/EXPERIENCE.md); where the two
disagree, this folder wins because a validator reads it.

## What is here

- `index.json` registers every page and journey with its status.
- `pages/` holds one spec per surface of the v2.0 redesign: `today`,
  `week`, `meals`, `explore`, `recipe`, `cooking`, `groceries`, `kitchen`,
  `onboarding`, and `settings`. Each declares routes (R03.1 of
  [`../docs/reference/product-specification.md`](../docs/reference/product-specification.md)),
  purpose, operational-target links, ranked priorities, states, elements, claims,
  and `data-testid` bindings named `<page>-<element>`.
- `pages/nutrition.json`, `pages/data.json`, and `pages/dashboard.json` are
  **deprecated tombstones**. Nutrition analysis moved to the Week Nutrition view,
  the Today overview, and the recipe Nutrition tab; transfer moved to Settings ›
  Data & exports (decision D-031); the scaffold home was retired earlier. A
  generated BAS case still pins the `dashboard` id.
- `journeys/` holds nine journeys that connect pages: first useful plan, capture
  an existing routine, a tired evening, shopping and changes, cooking and
  logging, a new week, moving data, plan from Explore, and set up the kitchen.

## Current level — honest status

The contract is authored to **L4 depth** (priorities, elements, claims,
bindings, explicit states, journeys), but it describes surfaces that are **not
built yet**. The 2026-09-22 audit found none of the bound test ids in the UI and
no redesigned surface in code. Therefore:

- Every redesigned page and journey is `draft`, and every claim is
  `aspirational` (decision D-035). Draft reconciliation is advisory.
- When a surface is rebuilt to match its mockup and its bindings exist in the
  UI, flip that page to `active`, add a BAS case whose
  `metadata.labels.spec_entry_id` is the page id, and promote the claims the
  experience phase can check (`single-dominant-action`, `visible-without-scroll`,
  `keyboard-reachable`, `element-present`, `element-absent`, `accessible-name`,
  `dark-parity`, `state-distinct`, `reading-order`) to `machine` tier. Claims of
  type `custom` stay `aspirational` or become `manual` with an attestation;
  they are proven by domain and journey tests instead.
- Non-default states use URL setup only: real query parameters for navigable UI
  state (`view`, `tab`, `mode`, `step`, `date`, `slot`) and `fixture=<state-id>`
  for data conditions. `fixture` is a contract with the capture harness, not an
  application code path (decision D-038): the harness produces data conditions by
  seeding Test Genie's routed test database and client conditions (loading,
  saving, offline) by network control; ordinary UI and API code never branch on a
  `fixture` parameter. Routes with `:id` or `:sessionId` need the harness to
  substitute a seeded id.

Visual fidelity against the approved mockups is judged outside this contract,
per [`../docs/reference/mockups/README.md`](../docs/reference/mockups/README.md)
and the ledger in
[`../docs/internal/REDESIGN_LEDGER.md`](../docs/internal/REDESIGN_LEDGER.md).

## Editing

Validate after every edit:

```bash
experience-manager spec validate nutrition-planner --json
```

Keep element ids stable once a BAS case or test references them; change only
`bindings` for a pure refactor or restyle. Every independently meaningful async
region reports the lifecycle vocabulary in the root `DESIGN.md` UX-state
contract; passive primitives inherit their parent's state.
