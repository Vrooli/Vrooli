# `experience.capture_bindings_unjoined`

## Meaning

The accessibility capture completed, but zero of the document's active declared
bindings joined the captured tree. This is an authoring defect, not a transient
capture outage, so it blocks the experience phase for pages and components.

## Fix

Correct the document binding selectors, test ids, roles, or state-specific
bindings so the rendered UI joins the declared contract. A capture that never
runs remains `experience.capture_unavailable` at INFO and is retried.
