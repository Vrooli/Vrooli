# `experience.affordance_missing`

## What it means

An active page made a machine-tier `affordance-present` claim, but the captured
accessibility tree did not contain one or more expected controls. The component
may exist, but users cannot perform the promised operation through perceivable
structure.

## How to fix it

Add or expose the missing affordance in the UI, then bind the relevant element
or component in the page spec. For example, a table claiming `sort`, `filter`,
and `search` should render accessible sort controls, filtering controls, and a
search input with stable roles, names, or test IDs. If the affordance is not
intended for this page, update the claim so the experience contract matches the
product intent.

Use a manual or aspirational `custom` claim only when the expected affordance is
real but not yet deterministically recognizable by the checker.

## Provenance

Emitted by experience-manager structure reconciliation when an
`affordance-present` machine claim fails against the BAS accessibility snapshot.
