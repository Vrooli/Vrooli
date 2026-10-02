# Recording Reconciliation Architecture

Recording has three separate responsibilities:

1. Workflow files are reconciled with their database index by the workflow sync service.
2. The Go live-capture `WorkflowGenerator` owns recorded-action merge semantics when creating workflows.
3. The UI projects each recorded action into the timeline in journal order. It does not combine, remove, or rewrite recorded actions. AI-step correlation is a separate display helper.

```text
raw recorded-action journal
  ├─ UI: ordered timeline projection; edit/delete/selection target raw indices
  └─ API: live-capture WorkflowGenerator
       └─ MergeConsecutiveActions()
            └─ workflow nodes

AI steps + raw actions ── mergeActionsWithAISteps() ── correlated timeline display
workflow files ── workflow sync ── database index
```

## Workflow sync

`api/services/workflow/sync.go` indexes filesystem workflow definitions for fast queries. The filesystem remains the source of truth; sync reconciles database metadata with the files that currently exist.

## Recorded-action workflow generation

`api/services/live-capture/workflow_generator.go` converts recorded actions into workflow nodes and calls `MergeConsecutiveActions()` before mapping those actions. This is the single recorded-action merge policy. Its tests cover input and keyboard sequences, scroll snapshots, focus handling, selectors, page and frame transitions, and timing semantics.

The UI passes raw actions to API generation and replay requests. The timeline preserves their order and identity, and editing, deleting, selecting, and replaying address the individual action. The Go generator applies coalescing only when it builds a workflow.

## AI-step correlation

`ui/src/domains/recording/types/timeline-unified.ts` keeps `mergeActionsWithAISteps()` for correlating AI reasoning with recorded actions. It is separate from recorded-action coalescing and does not define which actions the workflow generator combines.

## Key ownership

| Responsibility | Owner | Behavior |
|---|---|---|
| Workflow file and database index sync | `api/services/workflow/sync.go` | Files remain canonical; the database is a query index |
| Recorded-action merging for workflows | `api/services/live-capture/workflow_generator.go` | `MergeConsecutiveActions()` runs as workflow nodes are generated |
| Recording timeline | `ui/src/domains/recording/timeline/ActionTimeline.tsx` and `useUnifiedTimeline.ts` | Raw action projection in journal order |
| AI reasoning correlation | `ui/src/domains/recording/types/timeline-unified.ts` | Correlates AI steps and recorded actions for display |
