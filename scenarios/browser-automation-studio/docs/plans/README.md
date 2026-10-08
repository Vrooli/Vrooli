# Historical implementation plans

Canonical imported records live in Plan Manager. These are historical sources,
not current acceptance evidence: old “Planning” labels and unchecked items can
coexist with implemented code. Importing them does not approve a new execution
or mark any requirement complete. Reconcile remaining work with current plans
before resuming an imported draft.

Read a record with `plan-manager plans get <slug>` or
`plan-manager plans render <slug>`.

## Historical source files

| Original filename | Plan Manager slug |
|---|---|
| `RECORD_MODE_IMPLEMENTATION_PLAN.md` | `record-mode-implementation-plan` |
| `ai-api-keys-and-credits-plan.md` | `ai-api-keys-credits-implementation-plan` |
| `browser-automation-engine.md` | `browser-automation-engine-refactor-plan` |
| `multi-tab-recording-implementation.md` | `multi-tab-recording-implementation-plan` |
| `playwright-driver-completion.md` | `playwright-driver-completion-plan` |
| `replay-export-implementation-plan.md` | `replay-export-implementation-plan` |
| `ux-metrics-architecture-plan.md` | `ux-metrics-foundation-architecture-proposal` |
| `vision-agent-implementation-plan.md` | `vision-agent-implementation-plan` |
| `workflow-scheduling-plan.md` | `workflow-scheduling-implementation-plan` |

The full original files and original heading anchors are preserved under the
protected runtime-home `plan_artifacts` entry:

```text
docs-cleanup-20260907-followup/scenarios/browser-automation-studio/docs/plans/
```

See the [preservation record](../../../../docs/internal/PROGRESS.md#documentation-cleanup-follow-up--2026-09-07)
for recovery and hashes. Put new plans and supplements in Plan Manager's
returned artifact directory, not this documentation directory.
