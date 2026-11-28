# Git Control Tower maturity scope assessment

Date: 2026-09-05. Status: discussion input, not an implementation plan or an approved replacement PRD.

This assessment combines current source inspection, Swarm Manager goal/backlog reads, the scenario skill/program standards, and official host webhook documentation. It does not establish a readiness score. No scenario test suites, agent runs, repository mutations, or remote writes were executed. Existing tests contain real Git mutations, so running the suite requires a prior safe-test-boundary change.

## 1. Product direction and authority

Proposed direction: GCT owns repository/change identity, attributable change evidence, grounded summaries and drafts, advisory review, repository-host collaboration surfaces, and operator-facing repository controls. Skills carry judgment; programs compose bounded operations; the scenario owns durable facts and invariants.

The operator's constraint is authoritative: potentially destructive Git operations are human-only, including testing. Agents do not commit, push, merge, revert, reset, checkout/switch, stash, manipulate worktrees, stage/unstage, or arrange fixtures using those operations. A temporary repository or sandbox is not an exception. Existing human controls can remain a product capability; their implementation tests use fakes, supplied immutable data, and read-only inspection. Real actuation evidence must be human-produced and separately identified.

Repository reads must also avoid hidden effects: no implicit fetch, submodule update, hooks, external diff/textconv execution, or executing code from an untrusted PR. A local snapshot export can create a new artifact without modifying source Git state. Publishing comments, PRs, releases, or mirror updates is a separate external-write authority, not permission conferred by a request to draft text. No such publication was authorized in this assessment.

## 2. Reading map

Paths below are repository-relative. Registry skills are read through `prompt-manager skill read <id>`; their source paths can move.

### Core concepts

| Read | Why |
|---|---|
| `docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md`, especially section 4 | Three-speed stack and the requirement to simplify skills/programs after scenario promotion. |
| `docs/agent-system/SKILL_AUTHORING.md`, especially Scenario skill sets and Universal quality bars | Usage/improve/feature roles, step rungs, learning spine, programs as steps, conditioning defects and contract prompts. |
| `docs/agent-system/README.md#where-does-this-paragraph-go` and `docs/agent-system/README.md` | Separate authoritative truth, judgment, execution, and captured learning. |
| `docs/agent-system/SKILL_AUTHORING.md#promotion-ladder` | What earns promotion; what prose should be retired or retained. |
| `docs/agent-system/STANDING_TEAMS.md` and `RULES.md` | Setpoints, sensors, ownership, deadbands, and control-loop boundaries. |
| `docs/concepts/ECOSYSTEM.md` | Placement of integration infrastructure, interface enablers, products, and reusable capabilities. |

### Authoring and quality skills

| Skill | Use |
|---|---|
| `scenario-skill-setup` | Derive owed roles, inventory sensors and programs, declare the set and waivers, and set up measured improvement. |
| `skill-authoring-tools` | Usage decisions and operational feature skills. |
| `skill-authoring` | Skill-kind authoring guidance, including review practices and provenance retrieval. |
| `skill-validation` | Executability, real references, capability map, failure behavior, divergence probe, conditioning cost and compression candidates. |
| `program-runtime` | Governed composition, binding selection, typed inference/delegation, budgets and result interpretation. |
| `vrooli-memory` | Scoped recall, outcome-linked attempt capture, advice supersession, comparable learning measurements. |
| `measures-adoption`, `improvement-do-and-dont` | Real instruments and resistance to improving scores by weakening evidence. |
| `writing-standards` | Placement of controlled language, requirements and behavioral acceptance conventions. |

### Executable contracts

- `scenarios/program-runtime/docs/guides/program-construction.md`
- `scenarios/program-runtime/docs/guides/program-contracts.md`
- `scenarios/program-runtime/schemas/program-contract.schema.json`
- `.vrooli/schemas/service.schema.json` (skill declaration)
- `packages/proto/schemas/vrooli-memory/v1/learning/learning.proto`
- `scenarios/program-runtime/api/handlers/programs-validation/module.go`
- `docs/TESTING.md`

Some prose still describes programs validation as planned; the current provider module implements the `programs` phase and advertises execution support. Verify the actual provider contract during planning instead of carrying forward dated statements. Presence, static conformance, executed fixtures, and real-use efficacy are distinct claims.

### Maturity and future planning

Read `scenario-work-ladder`, `prd-authoring`, `requirements-traceability-steer`, `storage-steer`, `ecosystem-fit`, and `scenario-readiness-review`. The eventual planning workflow uses `implementation-plan-authoring`; execution uses `implementation-plan-execution`. This assessment does not run the maturity gates or claim to pass them.

Examples worth borrowing selectively: BAS and Device Control usage/improve skills and `author-flow` programs for exact identity and evidence-preserving repair; Deployment Manager for exact candidate evidence; Command Center for bounded briefings and operator feedback; Plan Manager for routing friction across all three layers; Web Search for structured advisory results and retained uncertainty. They are examples to validate, not blanket certifications.

## 3. Historical work inventory

Authoritative retrieval: `swarm-manager goals get --name <name>` and `swarm-manager backlog get --kind <kind> --name <name>`. Archived goals and dropped items are historical input, not automatic authorization to restore their original scope.

| Family | Records and desired outcomes | Reconciliation needed |
|---|---|---|
| Trailers | Archived goal `gct-commit-initiative-linking`; research `gct-trailer-system-design`; execute `gct-trailer-support`, `gct-swarm-manager-integration` | Parse/validate namespaced links, propose operator-editable trailers, preserve unresolved references. Reconcile initiative terminology with current work identities. |
| PR hosting | Active goal `gct-github-integration`; research `gct-github-api-design`; execute `gct-github-pr-api`, `gct-github-pr-ui` | Generalize host connection and PR identity; separate read/draft/publication/human merge. SSH credentials do not establish API authorization. |
| Release communication | Archived goal `gct-release-pipeline`; execute `gct-pr-description-generator`, `gct-release-notes-generator`, `gct-github-release-api`, `gct-release-authoring-ui` | Local drafting need not wait for hosting APIs. Generated PR prose is not sufficient release evidence by itself. Deployment Manager retains release/promotion authority. |
| Merge/conflict controls | Goal `gct-merge-and-conflicts`; research `gct-merge-architecture`; execute `gct-merge-api`, `gct-conflict-resolution-ui`, `gct-merge-time-checks` | Human-only operations. Shared readiness policy rather than a second gate configuration. Whether all new human controls belong in the first comprehensive scope remains an operator choice. |
| AI provenance | Archived goal `git-control-tower-ai-provenance`; execute `gct-pending-ai-provenance-hardening`, `gct-committed-ai-provenance-history` | Preserve pending/committed standing, run outcome, per-file state, origin and uncertainty; reuse Workspace Sandbox/Agent Manager authority. |
| Provenance search | idea `build-git-control-tower-git-provenance-search` | A real indexed provenance corpus, filters, freshness and retrieval evaluations, federated through Search Hub. Distinct from regex file-content search. |
| Undo | research `run-level-undo-and-revert-design`; execute `run-level-undo-and-revert` | Human-only, with partial/mixed provenance explicit. Do not infer reversibility from pathname overlap. |
| Known repair work | execute `restore-git-control-tower-api-test-harness-compatibility`; fix `gct-review-summary-tests-missing-on-failed-check`; fix `readiness-git-control-tower` | Reproduce only through a safe validation path; readiness is an outcome, not a substitute for defined obligations. |
| Superseded proposals | Dropped `gct-security-review-tab`, `gct-commit-level-gating`, `gct-commit-level-agent`, `qa-git-control-tower-tests-playbook-schema-20260515`, and `ecosystem-loop-health-snapshot-contract` | Read rationale before any revival. The old merge goal references gating machinery whose historical owner item may no longer represent the current design. |

The full scenario-filtered backlog read returned 30 records, including cross-owner sandbox research/rollout and completed code-quality work. Those dependencies must be reconciled rather than copied into a new plan as unfinished GCT implementation.

## 4. Current code findings and maturity work

These are source observations and design consequences, not a complete security audit or runtime verification.

### Contract and authority

- `PRD.md` still calls for agents to perform version-control workflows and includes agent-facing stage/commit/branch/stash capabilities. Its single-repo direction also lags the implemented repo registry. Rewrite the contract around the accepted human/agent boundary before linking new requirements.
- `api/routes.go` exposes legacy REST mutations; `api/connect_wiring.go` mounts newer Connect services. `api/internal/policygate/interceptor.go` maintains a separate mutation map containing Worktree procedures; unlisted procedures pass through that interceptor.
- `api/internal/policygate/policygate.go` treats unknown callers as non-agents; caller and override headers are signals, not independently verified human identity. Require authenticated authority at the owning operation, complete transport coverage, and a narrow agent capability surface. Do not label the existing coarse gate a sufficient public-webhook boundary.
- `.vrooli/agent-manager/reviewer.json` uses `code.default`, a 500-turn ceiling, and skipped permission prompts. `api/agent_manager_handler.go` requests sandboxed execution. Reuse the launcher only after defining and proving a constrained advisory profile whose setup cannot mutate Git.
- Tests in `api/repo_status_service_test.go`, `staging_service_test.go`, and `diff_service_test.go` invoke real checkout/commit operations. Replace fixture arrangement with safe supplied data and fake seams before automated execution; do not simply remove expected behavioral coverage.

### Shared evidence and durable reviews

- `api/diff_model.go` already supports staged/unstaged/untracked, base, commit and annotated content. Extend this foundation into an explicit change subject and bounded evidence bundle. Define merge-commit parent/base semantics, selected paths/hunks, renames, binary/large files, content digests, omissions and freshness.
- Current changes must be scoped explicitly in a shared workspace. No ambient active-repository switch should retarget an in-flight review. Host PR identity includes host instance, repository ID, PR/MR ID, base/head SHA and reviewed revision.
- `api/review_store.go` is an in-memory map; cleanup deletes entries by age, including entries still running. A mature advisory job needs durable state, idempotent creation, attachment/wait, restart recovery, terminal classification and retention independent of active work.
- `api/review_handler.go` rebuilds summaries after running checks. `fetchTestsDimension` in `review_handler_dimensions.go` reads the latest execution and returns nil when the provider is unavailable. Preserve each requested check's exact result and failure evidence; do not substitute whatever execution is latest. This is also a named backlog defect.
- Reuse Test Genie's receipts, descriptors, comparison semantics and artifact IDs. Keep test outcome, AI finding, draft completeness, operator acceptance and release authorization separate.

### Ownership and deduplication

- Consolidate legacy REST/handwritten wire models into typed domain services and generated clients as the relevant domains mature. `ui/src/lib/api-types-operations.ts` and `api-scenarios.ts` illustrate handwritten contracts alongside newer generated services. Retire the replaced routes/types with their callers.
- `api/connect_wiring.go` explicitly documents a second Git runner adapter. Share low-level safe execution mechanics while retaining narrow read and human-write interfaces; avoid one unrestricted runner for advisory code.
- `api/internal/dbschema/schema.sql` holds audit, repository and precommit domains together. Apply per-domain schema ownership and repository interfaces for new durable review/draft/provider state, migrating existing ownership deliberately under `storage-steer`.
- `approved_changes_model.go` lacks the run outcome, per-file provenance state and originating-surface fields called out in old work. Verify source-to-UI propagation, not only the schema of the producer. Path overlap alone is not exact authorship or proof a change entered a commit.
- Existing `ui/src/features/baselines/` already has shared primitives. Preserve that consolidation. Inspect legacy visual/workflow capture routes against the newer descriptor-based evidence surface before retiring any route: existence alone does not prove duplication, but two competing baseline/artifact authorities must not survive.
- The security posture and coherence docs contain placeholders; PROBLEMS includes historical scaffold statements alongside later shipped work. Update these to evidence-backed current contracts and retain dated findings as history.
- Check pagination, cancellation, bounded output, cache identity, accessibility, mobile layouts, safe diff rendering, and large repositories with meaningful representative fixtures. Recorded tidiness budgets are not a current measured score.

### Repository status and diff scale budget — 2026-11-21

The production `GetRepoStatus` path starts with porcelain-v2 status, classifies
files, computes staged/unstaged numstat, reads untracked-file stats, and returns
all changed paths. `GetDiff` is a one-file view; the existing API read context
is 10 seconds, and diff content reads reject a file above 4 MiB. These are the
current behavior limits, not a general status-response byte limit: status has
no aggregate pagination or truncation contract. A 4 MiB per-file content limit
does not cap annotation or JSON expansion, nor the total status response.

The reproducible owner benchmark is
[`repo_status_scale_benchmark_test.go`](../../api/repo_status_scale_benchmark_test.go).
It builds disposable 256- and 4,096-tracked-file repositories (one quarter
modified, plus one quarter as untracked), runs the actual status assembler and
one-file diff service, and records JSON bytes. Ten one-operation repetitions
are reported; p50 is the median, and p95 uses nearest-rank (the maximum for ten
samples). Run with:

```sh
go -C scenarios/git-control-tower/api test . -run '^$' -bench '^BenchmarkRepoStatusScale' -benchtime=1x -count=10
GCT_BENCH_REAL_REPO=/absolute/path/to/checkout go -C scenarios/git-control-tower/api test . -run '^$' -bench '^BenchmarkRepoStatusRealWorktree' -benchtime=1x -count=10
```

Environment for the recorded run: Linux/amd64, AMD Ryzen 9 7950X, Go 1.26.0,
Git 2.43.0, local NVMe filesystem. Synthetic results after the bounded
untracked-stat worker change:

| Workload | Status p50 / p95 | Status JSON bytes | One-file diff p50 / p95 | Diff JSON bytes |
| --- | ---: | ---: | ---: | ---: |
| 256 tracked, 64 modified, 64 untracked | 7.74 / 8.70 ms | 20,028–20,029 | 1.52 / 1.66 ms | 930–931 |
| 4,096 tracked, 1,024 modified, 1,024 untracked | 88.77 / 99.68 ms | 312,833–312,834 | 2.24 / 2.52 ms | 931–932 |

The actual read-only Vrooli worktree had 7,886 changed entries (5,742 tracked
changes, 2,144 untracked) and a 2,591,403–2,591,404-byte status JSON body. The
baseline sample had six fewer entries and a 2,590,172–2,590,257-byte body, so
the shared worktree was not byte-identical between runs. Its first ten samples
had 4.50 s median and 13.19 s nearest-rank p95, exceeding the
10-second request context in two samples; these overlapped a Test Genie unit
run. After untracked stats were moved to a fixed eight-worker pool, a fresh
ten-sample run measured 3.44 s median and 3.62 s p95. A later serial diagnostic
on the shared worktree showed 10.14 s for the first cold sample, 6.54 s for the
second, and 3.42–3.94 s for the remaining warm samples; the tree had changed
again, so this is not a controlled A/B comparison. The selected changed-file
diff measured 14.25 ms median and 15.03 ms p95 (10,709 JSON bytes). These real
worktree observations are sensitive to cache state and shared host load, not a
portable SLA. The synthetic 4,096-file workload stayed well under the existing
request deadline, with its measured output under the reference-fixture
regression budget of 350,000 bytes.

At the original measurement, synthetic scale met the 10-second context; the
pool-mode real-worktree samples also met it. The serial cold-cache diagnostic
exceeded it, which supports retaining bounded concurrency for untracked reads
but does not prove a same-state speedup. The aggregate-output gap is addressed
by bounded continuation below. Paired synthetic page traversal now measures
reduced repeated status assembly and latency; large actual repositories and
live UI rendering remain open.

#### Bounded status continuation — 2026-11-21

`GetRepoStatusRequest` now accepts `page_size` (1–512) and `page_token`;
`GetRepoStatusResponse.page` reports the global category-entry total, entries
returned, semantic snapshot id, continuation token, completion, and measured
response bytes. Category order is staged, unstaged, untracked, conflicts,
ignored, preserving order within each source list. A path staged and unstaged
is counted in both categories. Summary counts stay global on every page;
path-indexed maps and stats are projected to paths present on that page. The
server shrinks the selected count until both protobuf wire size (including the
Connect envelope) and proto JSON are at most 256 KiB. A single entry above that
limit returns ResourceExhausted. A no-size legacy request preserves the old
complete shape only below the same cap; over-cap legacy requests receive an
actionable ResourceExhausted response.

Continuation tokens bind an HMAC-authenticated snapshot id, canonical repository
path and absolute entry offset. The server materializes the complete response
once into a bounded in-memory LRU (64 MiB / 128 entries; no wall-clock TTL).
Continuation verifies fresh porcelain-v2 bytes, the index digest, all changed
path content digests, author configuration, and worktree identity, then projects
from the stored response. A mismatch returns FailedPrecondition and invalidates
the snapshot; capacity eviction has the same restart behavior. This removes
repeated full status assembly while retaining fresh change detection. Browser
`fetchRepoStatus` collects pages before exposing
a `RepoStatus`, discards a changed partial aggregate and retries from page one
at most twice. CLI `repo status` follows all pages and validates snapshot and
completion; its display uses exact global summary counts. Internal in-process
status consumers remain full-result calls and are not subject to the Connect
cap. The old `handleRepoStatus` function is not registered in `routes.go`, so
there is no active REST status consumer to migrate.

The prior fresh-status-per-page benchmark on the 4,096 tracked / 1,024 modified
/ 1,024 untracked fixture measured
312,833–312,834 B for the prior full JSON response. With preferred page size
512, page encoding/client aggregation produced 4 pages, 238,800 B aggregate
serialized page payload, and 396.49–430.69 ms including a fresh full status
assembly on each page in three runs. It is superseded as the continuation path
but remains useful historical evidence. A controlled paired benchmark now
alternates the prior four-fresh-assemblies path and the materialized-snapshot
path on the same disposable fixture and cache/load conditions. Each one-op
sample traverses all pages; the optimized path performs one full status
assembly, then checks porcelain, index, path hashes and identity on each
continuation. Ten samples (`-benchtime=1x -count=10`) measured:

| Path | Pages | Full assemblies | Total page bytes | Largest page | Traversal ms p50 / p95 |
| --- | ---: | ---: | ---: | ---: | ---: |
| Fresh status per page | 4 | 4 | 251,016 | 68,927 | 418.7 / 427.4 |
| Materialized snapshot + fresh guard | 4 | 1 | 251,597 | 69,121 | 262.45 / 274.5 |

Both largest pages are below the 262,144-byte cap. Cached-path metadata adds
581 aggregate bytes across four pages. Median end-to-end traversal latency fell
37.31% (418.7 to 262.45 ms) on this synthetic run. The benchmark includes
fresh Git status assembly and worktree-identity checks; it excludes HTTP/Connect
transport. It does not establish a real-worktree speedup, and earlier noisy
shared-worktree measurements remain noncausal.

The reusable size matrix runs 10 / 5 / 3 one-operation paired samples at
4,096 / 16,384 / 32,768 tracked files. Each fixture has one quarter modified
and an equal number of untracked paths. The order alternates between the
fresh-status-per-page baseline and materialized snapshot; each traversal
asserts exact category membership, exact global totals, stable snapshot,
duplicate-free paths, equal page count and the 262,144-byte response bound.
Fresh/cached pages and full status assemblies were 4/4 vs 4/1, 16/16 vs 16/1,
and 32/32 vs 32/1. Traversal p50/p95 (ms) was 450.15/508.8 vs 293.0/400.0,
9,531/9,826 vs 2,663/2,789, and 55,543/56,483 vs 7,927/8,174. Aggregate
serialized bytes (fresh/cached) were 245,896/246,505, 983,952/987,012, and
1,968,008/1,974,289; maximum page bytes were 67,391/67,594, 67,393/67,597,
and 67,394/67,596. Cached resident bytes were 818,434, 3,269,894 and
6,538,498, below the 64 MiB / 128-entry cap. The harness is reusable with
`GOWORK=off go -C scenarios/git-control-tower/api test . -run '^$' -bench '^BenchmarkRepoStatusPageMatrix/tracked_<SIZE>$' -benchtime=1x -count=<N>`
using sizes `4096,16384,32768` and counts `10,5,3`. Per-sample receipts:
`evidence/pt2-gct-status-matrix-{4096,16384,32768}.log`.

Browser consumer proof runs production `fetchRepoStatus` in Chrome through Vite
and a same-origin test proxy to the generated RepoService Connect handler. A
disposable 4,096 / 1,024 modified / 1,024 untracked repository is changed after
the first page; the UI discards/restarts, displays exact final membership for
2,049 entries, and reaches visible completion. A second refresh mutates at
each attempt and the UI displays the exhausted-retry error. The proxy observed
five initial requests, four failed continuations, nine successful pages, and
no page over the byte budget. This is browser-backed local API evidence; it is
not a shared production service sample. Reproduce with:

```sh
GOWORK=off GCT_STATUS_BROWSER_E2E=1 go -C scenarios/git-control-tower/api test . -run '^TestRepoStatusBrowserConsumerTraversesRetriesAndShowsError$' -count=1
```

The source pointer remains open until a comparable large real-worktree sample
and clean focused Test Genie receipt are available. PT2's three-window
showcase, sampled closure audit, and PT5 ratchet remain separate requirements.

### PT5 owner measurement adapter

`scripts/problem_owner_metrics.py` validates Git Control Tower's owner row from
the canonical portfolio index against the live `docs/internal/PROBLEMS.md`
size, SHA-256 fingerprint, and modification age. It emits counts and digests,
not problem text, and refuses a stale index or a missing/closed/misowned source
pointer. The current checked row is 1 active source file, 174,957 B, modified
age 0.0 days (the index rounds age to tenths; exact observed age was 0.016
days). This age is the file's modification age; dates inside the problem log
are not parsed as individual issue ages. It makes no closure claim and has not
recorded a PT5 real-window ratchet receipt. Run:

```sh
python3 scenarios/git-control-tower/scripts/problem_owner_metrics.py ~/.vrooli/portfolio/steward/state/problems.json
python3 -m unittest scenarios/git-control-tower/scripts/test_problem_owner_metrics.py -v
```

The owner tests cover exact source agreement, stale bytes/fingerprint/age,
missing closure state, and rejection of an understated fractional age from the
canonical ratchet. Full evidence: `steward-free-problems-20261121/evidence/pt5-gct-owner-measurement.json`.

The attempted read-only target was the shared Vrooli worktree, HEAD
`a75277a48e0eee9cdd7ee8add417314c6c139152`. One status observation had 6,155
tracked category entries (149 staged, 6,006 unstaged, five paths in both),
2,215 untracked paths, 8,370 category entries / 8,365 porcelain records, and
2,754,571 full-status JSON bytes at 4.246 s. Its
porcelain digest stayed `626727a5…` before/after; `git status` refreshed index
stat-cache data and the index digest changed, so it is not a paired unchanged-
index observation. The paged path was refused by the existing guard because
at least one changed symlink resolves outside the repository. No path or
content was included in the receipt and the guard was not relaxed. Further
real status reads stopped after observing Git's index refresh.

The disposable fallback matches the repository's 75,908 tracked-file scale
and changed composition: 149 staged, 6,006 unstaged, five category overlaps,
and 2,215 untracked paths. Five alternating paired page traversals (fixture
setup outside the custom timers) returned 17 pages; fresh status was assembled
17 times and cached status once. Fixture regular-file bytes were 7,710,595.
Traversal p50/p95 was 40,794/44,398 ms versus 4,696/5,150 ms. Aggregate
response bytes were 960,393 / 963,465; maximum page bytes 67,388 / 67,580;
cache residency 3,481,515 B. Every run compared exact expected category membership,
totals, order between paths, snapshot stability, duplicate absence and the
262,144-byte cap. This is a matched synthetic composition, not a live-repo
latency result. Receipt: `evidence/pt2-gct-real-composition-paged-pair-75908-20261121.log`.

The comparable live-worktree page contract remains unverified: the shared
tree is actively changing, normal Git status refreshed index cache data, and
the external-symlink guard correctly prevents continuation. Do not close the
source pointer from this fallback.

The prepared-status serializer/client aggregation run at page size 256 produced
8 pages, 240,366 B and 36.76 ms. This is not end-to-end latency. Neither that
run nor the old single-response measurement is an apples-to-apples comparison
with the paired continuation benchmark. Reproduce the paired result with:

```sh
GOWORK=off go -C scenarios/git-control-tower/api test . -run '^$' -bench '^BenchmarkRepoStatusScale/files_4096/status_page_comparable_pair$' -benchtime=1x -count=10
```

The corresponding owner regressions cover content changes within the same
porcelain category, index and untracked changes, cursor tampering/replay,
invalidation and capacity expiry, concurrent first-page requests, exact
category/path traversal, and protobuf/JSON response-byte limits. Focused owner
tests pass, and the UI API aggregation suite passes 16/16. Test
Genie structure passed L3; focused unit phase `20261121-134031-83740cf9` ended
failed/L0 on UI type-check and three broad API test commands, plus existing
architecture/coverage findings. A concurrently active unit run
(`20261121-125406-9fb4b98c`) failed on existing UI missing `ui/card` and Vite
5/6 type conflicts, broad API failures, and coverage debt; no status-page test
failure was reported. The browser proof above covers the local consumer and
generated API; a large actual-repository workload and a shared production UI
request remain unmeasured, so the owner problem and source pointer remain open.
prepared-status serializer/client aggregation run at page size 256 produced 8
pages, 240,366 B and 36.76 ms. Neither run includes Connect transport or
worktree identity lookup; the earlier single-response measurement is not an
apples-to-apples latency comparison. Dense 4,096-entry tests use long paths and assert each response,
including metadata, stays <=262,144 bytes; pages shrink below requested count.
The same tests assert each category path occurs once and changed worktree data
rejects continuation.

End-to-end multi-page latency against a large actual repository remains
unbenchmarked. Shared-repository sampling and production requests were not performed. The
response cap and continuation correctness are owner-tested, but the broader
performance obligation and source pointer remain open pending end-to-end scale
measurements and PT2 showcase conditions. Earlier noisy read-only worktree
results are not causal evidence.

## 5. Proposed skill and program set

Names are illustrative until contracts are authored. Reuse one evidence bundle across all interfaces.

| Skill | Program composition candidates | Evidence for success |
|---|---|---|
| `git-control-tower` usage | Select existing safe operations; evidence and baseline reading | Correct explicit subject, next action and complete attempt record. |
| Change summary | `change-evidence`, `summarize-changes` | Claims grounded in selected change evidence; omissions and unknown intent visible. |
| Commit-message drafting | `draft-commit-message`, shared `trailer-context` | Editable message and supported links for the exact selected changes; no commit. |
| PR drafting | `draft-pull-request` | Reviewer-oriented problem/behavior/validation narrative tied to base/head, separate from publication. |
| Change review | `review-changes` | Actionable findings with locations, reasoning, evidence and coverage limits; no code modification. |
| Release-note drafting | `draft-release-notes` | Audience-appropriate change communication traced through source/PR/release evidence; no invented compatibility claim. |
| Provenance investigation | `provenance-context` | Resolved references and uncertainty, with source-backed results from the provenance owner. |
| Improve | `setpoint-read`, `learning-read`, bounded corpus evaluation where needed | Comparable before/after evidence and an attributable repair that simplifies the appropriate layer. |

A deterministic one-command operation remains a CLI leaf. Do not create every illustrative program unless composition earns it. Typed inference uses the shared AI infrastructure; delegated review uses a declared, constrained workflow. All programs have bounded inputs/output, classified failures, declared bindings/effects, budgets, fixtures and one interpretable result on each path.

Learning records retain failed/unresolved attempts, actual advice use, comparison context, evidence identities, caller provenance, observed effort and capture coverage. Test attempts remain separate. Improve metrics include scope accuracy, unsupported claims, stale-evidence rate, trailer correctness, review precision/known-defect recall, operator corrections, completion effort, binding condition and external friction. Human acceptance alone is not quality; fewer findings alone is not better review. Baselines remain unearned until representative comparable observations exist.

## 6. Integration Hub and host neutrality

The existing scenario is `integration-hub`, singular. Read its `README.md`, `.vrooli/service.json`, `docs/internal/SEAMS.md`, `api/hub.go`, `api/main.go`, and `api/credential_cli.go`.

Observed: a small programmatic pilot, generated ConnectionService contract, OpenRouter-specific dispatch, credential-authority references, JSON metadata persistence, thin CLI, and a Web Console connection (`scenarios/web-console/api/main.go`). No PRD, requirements tree, testing config or scenario skills were present in the inspected file inventory. There is insufficient evidence to assert how it was originally generated. Its current shape lacks much of the standard scenario maturity contract; an API-only scenario does not require a gratuitous UI.

Important maturation work before public/multi-user use:

- Verified identity: `identity()` accepts a caller-supplied identity header or hashes a bearer value; this code does not itself authenticate the asserted identity.
- Honest connection verification: `checkConnection()` checks credential-authority configuration. That does not prove the OpenRouter account or requested provider capability works. Credential presence, successful remote authentication and effective permissions need distinct standing.
- Real connector interface, OAuth/App installation lifecycle where applicable, token expiry/rotation/revocation, scope changes, installation/repository access changes, provider-instance identity and audit.
- Durable idempotency and failure recovery across credential side effects and metadata persistence, bounded retained state, concurrency, health semantics, HTTP timeouts/shutdown, tests and measured operations.
- Template alignment, PRD/requirements/domain ownership, generated CLI contracts, skills/programs, and discoverable measures.

Recommended ownership:

| Owner | Responsibility |
|---|---|
| Integration Hub | Connections, installations, credential references, consent and provider authentication lifecycle; shared connector lifecycle mechanics. |
| Credential authority | Secret material and secure credential operations. |
| GCT | Repository-host capability contracts, PR/MR/change/review/release semantics, exact revision mapping and presentation. |
| Switchboard | Conversational ingress, sender/thread trust, capability attenuation, budgets and reply routing when treating host comments as a conversational channel. |
| Agent Manager / Program Runtime | Governed agent execution and bounded compositions. |

Do not put every external service's business API into Integration Hub. A host adapter must expose capabilities explicitly: comments, review threads, checks/statuses, releases, diffs, pagination and permissions are not identical across GitHub, GitLab and Bitbucket. Support a finite tested adapter set plus an extension contract, not an unsupported promise that any host works.

Switchboard has actual channel/trust/dispatch code despite README/progress text claiming no implementation. Its maturity needs current validation; neither that prose nor file presence proves a working end-to-end Git-host channel. Reuse the owner boundary without assuming the integration is complete.

Operator direction after this assessment: GitHub, GitLab and other host connectors belong in Integration Hub. The operator will discuss Integration Hub's overall scope separately; do not prepare its maturity plan here. GCT depends on precise connection and host-operation contracts rather than taking over connector implementation.

## 7. Mention-driven assistance

GitHub exposes comment/review webhook events; an app can receive a comment event and inspect an explicit invocation such as `@vrooli summarize` or `@vrooli review`. GitLab and Bitbucket expose analogous comment events, with different payloads and permissions. This is event handling plus command interpretation, not a universal mention primitive.

Recommended behavior: verify delivery, deduplicate, resolve exact host/repo/thread/revision and requester authority, apply a bounded advisory command, produce a grounded result, and publish only through a separately authorized reply operation. Ignore self-generated events, bound spend/rate/output, cancel or mark results stale when head changes, and retain an operation ID for delivery recovery. A public mention does not grant access to internal plans, logs, credentials or unpublished changes.

Never execute PR code, hooks or user-supplied commands to answer a mention. Review content is untrusted data. Start with public-safe summary/review/explanation commands; allow broader internal context only through explicit data-access authority. External replies can be an install-time operator-authorized capability while repository mutation remains human-only; this decision is still open.

Official references:

- https://docs.github.com/en/webhooks/webhook-events-and-payloads
- https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
- https://docs.gitlab.com/user/project/integrations/webhook_events/
- https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/

## 8. Focused scenario repositories and promotion

Distinguish three products: a showcase with source/release links; a buildable source distribution; a bidirectionally synchronized development repository. Recommended starting point for source sharing: a reproducible, one-way source distribution with the monorepo authoritative. Do not imply independent buildability for a showcase-only projection.

A source distribution needs more than copying `scenarios/<name>`:

- Explicit source commit or approved content digest; source-to-export path mapping and export recipe version.
- Recursive shared package/proto/schema/assets/build-tool closure, local Go replacements, workspace package links and generated artifacts.
- Explicit runtime dependency requirements and remote/local modes; build closure and runtime closure are different.
- Allowlisted publishable files, secret/private-data exclusion, licenses/notices, safe symlink handling, and no inherited private Git history.
- Honest standalone build/run instructions, known limitations, reproducible checksums and provenance manifest.
- Clean export validation using prepared data without Git mutation; export failure when required files/dependencies cannot be resolved.
- Drift detection, operator-reviewed update preview, refusal to overwrite downstream edits, contribution routing back to canonical paths, and lifecycle for renamed/retired scenarios.

SDA should resolve dependency graphs and file/package obligations; its current DAG and bundle-manifest surfaces are useful inputs, not evidence that standalone source export is implemented. Deployment Manager owns target/release approval and artifact identity. A source-distribution ramp can own packaging if that becomes a recurring multi-scenario product. GCT owns repository mapping, host presentation, source provenance and operator publication handoff. Decide the ramp's placement through ecosystem-fit before naming a new scenario.

History-preserving extraction and bidirectional sync are substantially larger products, with privacy, attribution, conflict and publication consequences. Neither should be silently included in a promotional mirror. Creating or updating Git history and pushing mirror contents remain human actions under the current constraint.

Useful promotional outputs from this same foundation: scenario-specific README drafts, source links, release highlights, changelog entries, evidence-linked screenshots/demos supplied by their owners, install links and contribution guidance. GCT supplies verifiable change facts; product/marketing owners choose positioning and claims. Generated copy should never imply deployment readiness merely because an export exists.

## 9. Scope boundary for the eventual plan

One coherent GCT plan can cover the accepted product contract, safe authority/test boundary, shared change evidence, durable advisory operations, skills/programs/learning, trailers/provenance/search, host-neutral repository collaboration, mention integration, and selected operator-only controls. It must retire replaced contracts/workarounds and reconcile old backlog states and dependencies.

Integration Hub implementation is externally owned and excluded from this planning discussion. The operator supports investigating source distribution as a new deployment ramp; GCT retains its interface and publication handoff. This separation prevents the GCT plan from acquiring credential authority, general messaging, test execution, or all deployment packaging.

Before authoring: settle mirror mode; human-only merge/undo feature inclusion; host adapter support scope; publication/reply authority; source-export ownership; and the accepted definition of mature. Completion must mean all selected behavioral, authority, durability, evidence, performance and interface obligations pass, with unknown/unavailable evidence represented honestly. R3 or R4 is selected through the maturity owner, not asserted by this report. No full implementation sequence, phase estimates, or execution plan is created here.

## 10. Follow-up investigation: source ramp and UX

These are design proposals based on source and experience-contract inspection, not a live visual audit. No UI or production code was changed.

### Ramp fit and implementation implications

`packages/delivery-ramp-go/doc.go` and `adapters.go` already define shared probing/building/driving/distribution boundaries, artifact references, checksums, and target evidence. `scenarios/deployment-manager/docs/DEPLOYMENT-GUIDE.md` explicitly gives the ramp packaging, evidence production, and publication responsibilities while Deployment Manager owns the release decision. The ramp requests that decision. Reuse this direction.

There are two gaps to resolve rather than copying another ramp verbatim:

1. Deployment Manager's `ui/src/lib/tiers.ts` and `features/deployments/GuidedFlow.tsx` still use local/desktop/mobile/cloud/appliance choices. A source repository is a delivery format, not another numbered runtime tier. Introduce a capability-derived delivery-format choice and keep runtime requirements separate. Existing numeric aliases can remain compatible.
2. The shared ramp evidence contract includes protocol, visual, and release-visual profiles with visual launch/capture states. A source distribution needs a truthful nonvisual validation profile. Reuse artifact identity, bounded execution, persistence and governance, but prove source closure, archive integrity, reproducibility, documentation and promised clean build/run behavior. Do not manufacture video evidence or pretend a skipped visual check proves source-package quality. Real visual journeys remain appropriate if the exported application claims a working UI.

Working name: `scenario-to-repository`, displayed as **Source repository**. This names the user outcome without binding the ramp to GitHub or promising bidirectional mirroring. The name is a proposal, not a created scenario.

The ramp should own export recipes, deterministic assembly, source manifests, verification, destination mapping and publication receipts. Integration Hub owns host connectors. GCT supplies change/provenance evidence and review/presentation affordances. Deployment Manager owns exact-artifact approval. Distribution must preserve the human-only Git policy: preparing an archive/directory is distinct from creating commits, tags or pushes, including mutations through a host API. Publication cannot be reported complete until the human action has verifiable destination evidence.

A reusable source-closure primitive could also help plugin distribution, which asks for a standalone-installable scenario. Verify that seam against actual plugin contracts; do not make every ramp consume a new source packager by default.

### GCT workspace UX

Preserve the diff-first desktop layout and mobile panel navigation in `experience/pages/workspace.json`. Existing `AppPanels.tsx`, `CommitPanel.tsx`, `ScenarioReviewPanel.tsx`, `GitHistory.tsx`, and `SettingsTabIntegrations.tsx` provide the natural insertion points.

Add a persistent **subject bar** above evidence and advisory content:

- Repository and branch or remote PR/MR identity.
- Subject: selected current changes, staged changes, commit, range, or pull request.
- Scope: selected paths/targets and inclusion count, with an expandable exclusions list.
- Snapshot time/revision and a clear changed-since-review indicator.

Selecting a subject only changes the inspected data. Selecting a PR or historical commit never checks out a branch, fetches implicitly, or stages files. Keep the subject stable when the operator opens a different view; switching it explicitly creates/selects the corresponding draft/review context.

Suggested desktop shape: file list on the left, diff in the center, optional inspector on the right with **Summary / Findings / Drafts / Evidence**. On narrow displays these become full-width views; do not shrink three columns into a phone screen. Existing scenario review remains available for broad validation evidence, with the subject bar making the distinction from an exact change review clear.

### Concrete interactions

**Summarize selected changes.** Select files or a target group, choose Summarize, and receive a short explanation of behavior changes plus links to supporting hunks. Expand to rationale, risks and unknowns. Scope selection is independent of Git staging. Untracked, binary and omitted files must remain visible in coverage. A changed subject produces “Files changed since this summary” with an explicit refresh action; it does not silently overwrite text.

**Draft a commit message.** The existing commit panel gains a Draft message action. It first confirms the subject in plain language, then shows editable title/body and suggested work-link chips. Links distinguish verified references from unresolved suggestions. Applying the text to the editor is distinct from the human Commit action. Regeneration shows a proposed revision and preserves manual edits. A draft made from selected unstaged changes must not be silently used for a different staged set.

**Review a change.** Findings carry severity, a concrete failure explanation, file/hunk location, evidence and suggested next investigation. Show coverage separately from finding count: “No findings in 8 reviewed files; 2 files not reviewed” is truthful. Resolution or dismissal records a reason and the revision; neither implies the code was fixed. Reruns preserve previous findings and mark outdated locations. Existing test evidence and AI conclusions remain separate sections.

**Read or draft a PR.** A Pull Requests workspace lists host, repo, base/head, author, updated time and review standing. Selecting one reuses the same diff/inspector. Draft PR opens a text editor with summary, behavior, validation and limitations plus evidence references. Connection absence still permits local drafting from an explicit range. A separate operator-controlled publication action shows the exact destination and content; no automatic branch creation or push is hidden in that action.

**Trace changes.** AI Changes and historical provenance share a timeline of runs, applied changes, linked work and commits, preserving mixed/unknown attribution. Link chips drill into evidence instead of auto-selecting or mutating files. A public view shows only publishable references; private work IDs cannot leak through a generated description.

**Use a host mention.** The host comment is an entry point into the same review operation. Reply with a compact summary, exact reviewed revision, coverage, and a link to the result where access permits. GCT can show these requests in the PR's activity stream with queued/running/partial/completed standing and delivery failures. Do not create an unrelated chatbot product or bury failures in an agent transcript.

**Connection settings.** Keep repository Git transport credentials distinct from host API permissions. Show the Integration Hub connection, bound host/repository, last verification and capabilities such as reading PRs or posting replies. Manage connection links to its owner. A configured SSH credential must not make the UI claim PR API access.

### Source-repository UX

Entry points: **Share source** from a scenario in GCT, or **Source repository** as a delivery format in Deployment Manager. Both open the same ramp-owned distribution record with return links; avoid duplicate wizards and state.

The first-time flow is:

1. **Choose source:** scenario, exact source revision and distribution name. Working changes can produce a labeled preview, but are not silently assigned a commit identity.
2. **Review contents:** an export tree separates app code, required shared code, generated files, docs/assets, exclusions and unresolved dependencies. Explain why each shared component is included. Separate required runtime services from bundled source files.
3. **Preview the public repository:** render README, screenshots/demo links, installation instructions, limitations, contribution routing and canonical-source link. Suggest description/topics from approved product material. Private/internal references are excluded.
4. **Verify:** individual results for closure, build, promised runtime behavior, reproducibility and publication eligibility. Each failed row has a concrete next action. A successful archive alone does not earn “standalone app.”
5. **Review and hand off:** choose a connected destination, inspect visibility/name and exact update diff, obtain Deployment Manager's artifact decision, then perform the human publication step. Offer a prepared archive independently. Record publication only from destination evidence.

Afterward show a durable card: canonical revision, exported revision, destination URL, verification time, last publication and update standing. Suggested states: **Up to date**, **Source changes available**, **Export needs repair**, **Destination changed independently**, **Publication unverified**. For downstream edits, show the divergence and stop rather than overwrite. “Prepare update” reuses the same recipe and previews exactly what would change.

Promotion is helped by an honest, polished destination: clear product introduction, quick start, demo/install links, source provenance, release highlights and a contribution path. License choices and marketing positioning remain owner decisions; neither is inferred from package contents.

### UX implementation quality

GCT has a detailed workspace experience contract but no root `DESIGN.md` in the inspected tree. Establish its design contract while preserving the recently implemented mobile layout. Adopt shared RCL components through their supported asset workflow. New panels need keyboard/focus behavior, deep links, durable drafts, resumable operations, explicit empty/error/partial/stale states, accessible status wording, and safe rendering of source/Markdown. Do not show internal skill/program identifiers in routine product flows.

UI evaluation should include: changing selection after drafting, two concurrent operators, stale PR heads, missing host access, absent provenance, very large diffs, a disconnected learning store, mobile navigation, export closure failure and independently edited destinations. Use fake host adapters and safe supplied repository evidence; no Git mutation to set up or clean up tests.
