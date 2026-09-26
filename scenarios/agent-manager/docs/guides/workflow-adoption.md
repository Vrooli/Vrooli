# Workflow adoption

## Decision rule

Use this test before integrating an agent:

```text
human composes prompt + human reads reply  -> Run / conversation session
code composes input + code consumes result -> declared Workflow
no agent judgment                         -> deterministic domain action
```

This is an ownership boundary, not a complexity threshold. A one-turn,
code-owned interaction is a workflow because its contract, prompt provenance,
budget, and result validation must be reviewable before it runs.

## What a consumer keeps

A consumer scenario owns only its domain boundary:

1. Build a typed, bounded, immutable snapshot from authoritative state.
2. Authorize, correlate, validate evidence for, and apply the terminal result
   exactly once.

The workflow owns the middle: prompt resolution, Run creation, structured
result validation, retry and repair policy, looping, branching, waits, child
workflows, and execution provenance. A consumer-side prompt builder, prose
parser, failure classifier, polling loop, or review-routing state machine is
unmigrated workflow logic.

## Adoption checklist

1. Name the domain transition and decide whether it is conversational,
   programmatic, or deterministic.
2. Declare a workflow with an input schema, output schema, budgets, typed
   terminal outcomes, and bindings.
3. Put substantive reusable instructions in a Prompt Manager skill and use a
   `promptRef`; keep declaration-local framing limited to typed bindings and
   transition-specific facts.
4. Reconcile in validate-only mode. Prompt references resolve and pin during
   reconciliation, so an execution revision cannot silently change when a
   skill changes.
5. Implement only the consumer input-snapshot and exact-once apply adapters.
6. Test malformed output, abstention, cancellation, stale input, duplicate
   delivery, required evidence, and restart recovery as relevant to the
   transition.
7. Remove the old consumer-owned method seam in the same change. A wrapper
   around a surviving prompt parser or loop is not a completed migration.

## Prompt ownership

Author exactly one of `promptTemplate` or `promptRef` for a Run/Continue node.
Prefer `promptRef` when instructions are reusable or need Prompt Manager
governance. Agent Manager resolves the reference at reconcile time and embeds
the resolved prompt plus its provenance into the immutable workflow revision.
See [scenario declarations](../reference/scenario-declarations.md#referencing-a-prompt-manager-skill-promptref).

## Consumer example

```text
Swarm Manager: authorize work + snapshot backlog item
    -> Agent Manager: execute declared plan-repair workflow
    -> Plan Manager / Test Genie: supply authoritative validity/evidence
    -> Swarm Manager: verify correlation and apply typed terminal result once
```

The workflow never mutates the consumer's domain directly; it returns a typed
result for the consumer's explicit, authoritative apply action.
