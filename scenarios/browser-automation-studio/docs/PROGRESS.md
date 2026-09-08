# Progress Log

## Documentation cleanup — 2026-09-07

Detailed historical entries were preserved byte-for-byte under the protected
runtime-home `plan_artifacts` entry:

```text
docs-cleanup-20260907-followup/scenarios/browser-automation-studio/docs/PROGRESS.md
```

The [repository preservation record](../../../docs/internal/PROGRESS.md#documentation-cleanup-follow-up--2026-09-07)
locates original hashes and recovery copies. This condensation does not close
any issue or change plan status. Consult the preserved entries for unresolved
handoffs that have not yet been reconciled; historical passing counts do not
establish current validation.

## Historical milestones

- November 2025: browser execution, replay/export, desktop integration, UI,
  and test infrastructure were iterated.
- January 2026: recording pipeline observability and unified action recording
  were introduced. These historical milestones do not certify today's build.

Current known issues: [problems ledger](PROBLEMS.md).
Read live plan state with `plan-manager plans get <slug>`.

## Agent usage and improvement implementation — 2026-09-04

### Summary

Implemented scenario-owned usage and improvement skills with bounded programs,
explicit workflow selection, verified promotion, preserved repair contracts,
and outcome-linked learning. Speed targets remain pending a comparable operator
baseline. Test evidence establishes implementation behavior, not household-device
coverage or a measured agent speedup.

The five usage programs now have distinct selection, execution, authoring,
navigation, and task-routing contracts. `author-flow` persists a candidate only
after successful validation and execution; repair preserves assertions, topology,
settings and metadata and uses the expected revision. Workflow API updates share
the project lock with filesystem reconciliation. `do-task` does not execute fuzzy
matches or duplicate the usage skill's recall. `learning-read` provides the new
effort/outcome board separately from the execution-quality board to fit the
runtime output bound. Friction samples with truncation or unknown attribution
remain unproven. Navigation-to-workflow drafting remains agent judgment; the
program validates and persists the resulting candidate.

Shared changes: Memory accepts optional first-action, tool-round-trip,
visual-reasoning and workflow-reuse observations. Missing values remain absent;
ambiguous or retry-only histories cannot earn first-action timing. Test attempts
are excluded from operator metrics. Program Runtime captures nested contract
envelopes without leaking child stdout and refreshes declared source for each
new kernel; refresh scans program directories rather than the entire source tree.

### Findings

| Capability | Primary path | Verification | Failure path |
|---|---|---|---|
| Orientation and selection | Usage skill and discovery program | Exact identity and context | Refuse ambiguity; inspect owner inventory |
| Workflow reuse | Exact saved revision | Owner terminal result and evidence ID | Preserve failed result; no speculative alternate |
| Candidate promotion | author-flow | Assertions, complete execution, durable identity | Failed/pending candidate remains unsaved |
| Repair | author-flow plus expected revision | Existing checks preserved | Refuse stale or weakened baseline |
| Improvement | Improve skill and learning sensor | Comparable windows and sample counts | Missing/unreliable evidence cannot meet a target |

### Divergence probe

| Instruction | Possible divergent reading | Resolution and evidence |
|---|---|---|
| Select a matching workflow | Execute the first fuzzy result | Explicit identity/version required; search-only probe performs no execution |
| Save a successful candidate | Treat pending, dry-run or AI completion as verified | Owner gate and negative promotion probes refuse these paths |
| Repair while preserving checks | Drop an assertion to make the repair pass | Owner regression and live BAS refusal preserve the acceptance contract |
| Reduce time to first action | Count retries or missing timestamps as faster samples | Memory tests exclude retry-only/ambiguous histories and preserve missing values |
| Run an improvement goal | Repeatedly file work without implementing | Improve skill explicitly routes implementation under existing task authority |

No divergence remains on these probed instructions. This is a bounded probe,
not a claim that every possible user task has been tested.

### Contract integrity and prose retirement map

Routine leaves use owning CLIs; programs use registered typed bindings. Structured
Attempt JSON and typed workflow JSON are necessary inputs, not output-parsing
workarounds. No raw ADB, browser-driver endpoint, database access, or shell-based
device control is part of the usage path. Ordinary usage invokes registered
programs; temporary sessions are used only by program validation.

| Repeated agent work retired | Durable replacement | Verification |
|---|---|---|
| Reconstructing orientation joins | Bounded discovery/prepare program | Ambiguous-target probes and live read |
| Hand-copying a successful draft into storage | Owner promotion plus author-flow | Persistence/replay regression and live browser test |
| Overwriting a broken saved flow | Revision-preserving owner update | Stale/weakened repair refusal |
| Informal claims that reuse is faster | Outcome-linked Memory cohorts | Optional-field and test-exclusion checks |
| Parsing nested program prints | Runtime returns a bounded envelope Handle | Kernel regression tests |

### Validation

- All 24 declared fixtures across both scenarios passed in fresh test-provenance
  sessions, with a preflight explain call for each fixture.
- All 11 program behavior probes passed. The owning Go tests invoke these probes.
- Device API `go test ./...`, device CLI control tests, BAS workflow handler and
  service tests, Memory learning tests, Runtime contract-index tests, and all 30
  kernel Handle tests passed.
- Live BAS: create/execute/save, exact replay, revision-2 repair, stale-revision
  refusal, weakened-assertion refusal, and original-revision replay all passed.
- Live device orientation read succeeded. A deterministic property adapter proved
  execution, assertion failure, durable restart/read, and saved revision replay.
- Memory accepted the new effort fields, returned the same entry on exact retry,
  and excluded the test attempt from operator measurements.
- Both scenario Test Genie program and skill-set phases passed. Broader unit
  runs have provider/UI failures; device business checks also report older
  unearned completion claims. These runs are not represented as green.

Machine-readable program IDs and live evidence are retained in
`.vrooli/program-runtime/tests/validation-evidence.json`. Behavioral probes are
in `.vrooli/program-runtime/tests/test_workflows.py`.

### Recommendations

A (recommended): use the registered usage skill for authorized real tasks and
capture comparable operator attempts; then use the improve skill to move the
first measured bottleneck through its owning implementation workflow.
B: continue fixture validation for an unavailable device or environment, retaining
that evidence class separately. This can improve correctness but cannot earn a
physical-device claim or operator speed baseline.

### Notes

No physical device was actuated for this implementation validation. Exact
screen/transport/app coverage and earned latency targets require authorized
representative device tasks. The self-improvement skills can guide sustained
implementation; they do not turn missing evidence into a completion claim.
