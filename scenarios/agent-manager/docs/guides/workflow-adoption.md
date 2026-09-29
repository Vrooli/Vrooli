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

## Fixed worker authority

For an implementation boundary, put the literal `sandboxConfig` on its Run node.
The workflow revision pins this configuration alongside the prompt and budgets;
worker output and prompt bindings cannot replace it. `scopePathTemplate` selects
the workspace, not its write permissions. Set `writePolicy.paths` to the existing
paths the worker may edit, or an empty array for read-only review. Set protected
mode, `networkMode: "none"`, `manualReview: true`, `autoApply: false`, and
`applyOnFailure: false` for a worker that cannot publish its own changes.

Dispatch passes the node's explicit network mode to both native runner controls
and the sandbox. Omitted fields retain ordinary profile defaults; omission is not
a restricted-worker policy. Continue nodes retain the source run's persisted
authority and offer no authority override. A new boundary requires a new reviewed
revision; it is not a worker-authored prompt change. Host path existence and alias
containment remain launch-time checks owned by Workspace Sandbox.

This pins run authority; it does not by itself authorize who may register a
workflow, prove all transports isolated, or qualify a candidate for publication.

## Retained reviewer input

On an independent reviewer Run node, set `reviewInput` to
`{"fromNode":"worker","paths":["src","tests"]}`. The paths are literal selections
relative to the worker sandbox scope. Include every changed file and the unchanged
context needed for review. Do not also set `scopePathTemplate`. Set explicit
protected mode, empty `writePolicy.paths`, `networkMode: "none"`, and
`autoApply: false` on the reviewer. Select its review profile explicitly; the
workflow's worker model/effort preferences do not override this profile.

The engine persists the completed source attempt before dispatch. Later worker
attempts cannot replace it. The launcher accepts only a completed protected,
non-applying, sandboxed codec-pipe child of the same execution. Workspace Sandbox
stops its tracked processes and requires terminal receipts before a new capture;
an empty inventory is unknown, not proof of drain. Interactive and imported
sources are not supported by this path.

The reviewer attempt UUID is the immutable capture key. WSS materializes
`before/`, `after/`, `changes.patch`, and `snapshot.json` from retained blobs.
The reviewer task uses this root instead of the canonical project. The run retains
the source run, request UUID and content digest in `VROOLI_REVIEW_*` provenance.
A lost dispatch response reattaches to the same child. A lost capture response
reuses the retained input; it does not require the old overlay to survive.

Declare a deterministic JSON-schema `resultSpec` with an object that requires
`accepted` (boolean) and `candidateSha256` (string). Add the review's required
reasoning and findings fields to that schema. At dispatch, AM narrows
`candidateSha256` to the retained digest with a schema constant. The run's existing
result receipt retains that bound schema digest and the validated verdict.
The authored workflow revision remains unchanged. A continuation or schema repair
keeps the bound contract; it cannot remove validation or substitute another
candidate. Do not use constrained extraction to infer the reviewer's decision.

Typed nodes require a present, valid result before successful routing. A success
flag, prose handoff or completed goal cannot substitute for it. This applies to
sequential nodes and parallel joins. A valid `accepted: false` is a rejection,
not an execution error; route it explicitly through the existing typed branch.

This bound verdict does not itself grant approval or qualification. Its source
implementation is tested; live verdict adoption is pending. Process groups left
by exited leaders, cross-restart process recovery, candidate-bound independent
acceptance and one-use qualification still need separate owner evidence before
campaign activation. The source-level deterministic checks do not constitute the
managed BAS pilot.

The production orchestrator constructor must install the existing qualification
owner when workflow repositories are configured, just as it installs child and
subworkflow owners. Tests must exercise that constructor, not only an engine with
a manually injected owner. This wiring does not grant execution: the pinned
independent verdict, retained qualification attempt and PRT admission-contract
checks still precede effects. Source readiness does not authorize campaign
activation; managed adoption and the representative pilot remain separate gates.

The effort owner must select the boundary and its required qualification rows
before dispatching the worker. Retain them in the original workflow input and
bind them with `source: workflow_input`, `renderAs: json`, and
`missingPolicy: error` in the immutable qualification node. Never bind acceptance
scope from a worker result, handoff, signal or caller override at qualification
time. The protected worker cannot change that input or revision. AM retains the
evaluated inputs in the qualification attempt; it need not inject a second copy
of scenario-specific rows beside the candidate. The BAS pilot must prove this
configured path rejects a worker's attempt to shrink the selected rows; merely
passing an arbitrary nonempty array to the program does not prove ownership.

Qualification usage must come from the matching runtime-owned `Program` receipt,
not its stdout envelope. Preserve all candidate, program, request and caller
identity checks. Program Runtime's [durable accounting target](../../../program-runtime/docs/operations/OBSERVABILITY.md#durable-declared-program-accounting--implementation-target)
owns settlement, measured-zero semantics and survival of session reclamation.
An absent or incomplete receipt remains unverified, even when stdout claims
complete measured accounting. This is an implementation target, not adopted proof.

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
