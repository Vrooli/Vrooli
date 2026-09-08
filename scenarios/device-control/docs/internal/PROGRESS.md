# Progress — Device Control

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/device-control/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
The table retains selected dated milestones; consult the archive for the full sequence.

These are dated development milestones, not a current readiness verdict.
Detailed pre-cleanup entries are preserved beneath the protected runtime home:
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/device-control/docs/internal/PROGRESS.md`.
Use that original for command transcripts, measurements, and full qualifications.
Hashes and recovery instructions are in the project documentation cleanup record
(`docs/internal/PROGRESS.md`, "Documentation cleanup completion — 2026-09-07").
Current unresolved issues belong in this scenario's problem ledger; the historical
handoff notes below remain follow-up leads until checked against current evidence.

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append concise milestones when work lands. Keep detailed execution receipts
with the owning plan.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-18 | codex | Closed the locally actionable agent and documentation gaps: bounded multi-iteration planning now records typed world models and per-step evidence, supports planner goal completion and dry-run, and promotion creates a deterministic flow consumed by the existing export path. | Archived |
| 2026-08-18 | codex | Closed additional plan-level gaps: `packages/mdns-go` now exposes the typed `Browse(ctx, []string, Options)` contract, the required README, IPv4/IPv6 multicast joins, address-family queries, and cancellation-preserved observations. | Archived |
| 2026-08-18 | codex | Repaired the generic mDNS browser's Linux shared-port fallback, canonical DNS-SD queries, section-aware unknown-record skipping, and requested-service PTR filtering. | Archived |
| 2026-08-18 | codex | Completed targeted implementation validation for the reachable non-mobile device plan: API/CLI/mDNS suites, focused race checks, gofumpt, golangci-lint, UI 183 tests/type-check/lint/coverage, requirements/business-health validation, and experience-contract validation pass. | Archived |
| 2026-08-17 | codex | Repaired the UI test provider harness and route providers, expanded focused coverage, and declared the `agents_list` REST projection explicitly as an external JSON shape. | Archived |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map


## Agent usage and improvement implementation — 2026-09-04

### Summary

Implemented scenario-owned usage and improvement skills with bounded programs,
explicit workflow selection, verified promotion, preserved repair contracts,
and outcome-linked learning. Speed targets remain pending a comparable operator
baseline. Test evidence establishes implementation behavior, not household-device
coverage or a measured agent speedup.

Device programs are `prepare-task`, `replay-flow`, `author-flow`, and
`setpoint-read`. The FlowService owns a durable SQLite revision library, exact
device/context selection, atomic expected-version saves, idempotent retry, and
retained validation receipts. Candidate promotion requires every step to pass,
an explicit terminal assertion, and deterministic targeting. Dry-runs and vision
resolution cannot qualify. `property-assert` validates an adapter-observed value
without requiring a screen. Repair preflight uses the exact device adapter.
The old prompt-manager local-pack skill was retired in favor of the scenario source.

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

## Historical handoff leads retained during condensation

These clauses are preserved from dated entries; later work may have superseded
them. They do not assert that an old failure or gap still exists.

- **2026-08-18 — partial**: Full API/race/lint, mDNS live, Cast live, UI, requirements, and independent Avahi checks pass; owner-present PIN/event/UI walkthroughs and the provider-owned Test Genie/baseline gate remain deferred.

- **2026-08-17 — partial**: The bounded API fallback remains healthy and device-control API/UI/build checks pass; the remaining failures are the shared `@scenario/self` workflow resolver and repeated UI-health chrome captures.

- **2026-08-17 — partial**: The only remaining integrated failure from that run is the shared BAS `@scenario/self` workflow resolver.

- **2026-08-17 — partial**: Cleared the remaining local Go quality floor: four pre-existing gofumpt findings were normalized, and the full device-control API suite, `go vet ./...`, and `golangci-lint run --timeout=5m` now pass.

- **2026-08-17 — partial**: Retained run `20260817-044920-e9b29908` reflects pre-run status-bar/docs state and the shared workflow-provider failure, so fresh integrated evidence remains pending.

- **2026-08-16 — partial**: Final managed physical screenshot flow `98fdca8f-13ce-4117-94f5-b95928e4edc2` passed after deployment and retained screenshot evidence `4f3ff631-e5a3-40bf-86bc-84daa1aa1caf` at 720x1600 with only the declared `status_bar_identifiers` rule; the remaining narrow black band is the intentional status-bar privacy mask, not a geometry bar.

- **2026-08-15 — partial**: The remaining architecture warnings are pre-existing manifest maturity debt, not binding errors.

- **2026-08-13 — partial**: Follow-up live validation found the Galaxy A03s `screenrecord` path can produce a valid H.264 container whose display body is uniformly black even while ADB screenshot/state probes report an awake launcher.
