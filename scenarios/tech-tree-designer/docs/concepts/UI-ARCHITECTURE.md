# UI Architecture

## Shared contract

Read the [UI Architecture shared guide](../../../template-manager/docs/concepts/UI-ARCHITECTURE.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

Use this scenario’s UI manifest and source tree for its actual layout.
Record local layout or adoption constraints here; the current template guide
does not establish that an older scenario has migrated.

The target is an artifact-centered design workspace with graph-backed context, not a full-graph canvas as the only navigation. The existing routes /, /graph, /ontology, /planning and /settings anchor the draft experience contract. Review, conflicts and recovery are proposed planning states; implementation may refine route decomposition while preserving the journeys.

Use progressive disclosure, scoped graph queries, asynchronous layout and textual/keyboard alternatives. Show exact revisions and material changes before decorative graph detail. Independent source/validation/apply regions need distinct pending, partial, stale and failure states. Compact layouts must retain warnings, authority boundaries and recovery actions.

No new selectors or component bindings are asserted by this documentation update. Ground experience claims in actual component stories and browser evidence as implementation lands. Keep business rules in shared typed API behavior, not UI-only checks.
