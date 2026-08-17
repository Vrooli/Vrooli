# Prose Studio experience audit

## Migration brief

- Intent: make the four core jobs feel measured, calm, and editorial rather than like one prop-switched scaffold.
- References: the scenario's token-driven primitives and the marketing voice canon; non-goal: inventing a new generation workflow.
- Constraints: preserve route paths, selector IDs, keyboard focus, declaration retry behavior, and score-free candidate cards.
- Scope: primitive/token-compatible surface and layout migration with explicit loading, error, and empty states.

## Purpose and users

Prose Studio helps an operator generate, compare, and assemble prose while preserving voice instructions and measurable evidence.

- Generator: create a candidate set and reroll when no candidate fits.
- Editor: inspect a document outline and continue one section at a time.
- Voice owner: inspect declared and operator-authored style inputs.
- Operator: validate consumer-owned declarations and diagnose invalid or stale files.

## Current versus ideal flows

| Job | Current | Ideal | Friction addressed |
| --- | --- | --- | --- |
| Compare candidates | Variation route → static empty board | Variation route → set basis → candidates → negative reroll | Empty and loading states now explain the next action. |
| Continue a document | Document route → static outline → no document | Document route → outline rail → active section state | The route now owns a distinct workspace surface. |
| Validate declarations | Declaration route → fetch → success/error | Same route → loading/error/retry/success | Retry and audit-preserving status are explicit. |

## Implemented in this loop

- Split variation, styles, document, and declarations into distinct route components.
- Kept set diversity as one named-basis value and removed per-card ordering language.
- Made “None of these — reroll” an enabled negative action only after a set exists.
- Added explicit loading, error, and empty rendering branches to the primary surfaces.

## Deferred

Connect the empty states to the typed generation/document clients after the service-backed query models are exposed. Add live candidate and section fixtures to BAS once those clients have stable fixture endpoints.
