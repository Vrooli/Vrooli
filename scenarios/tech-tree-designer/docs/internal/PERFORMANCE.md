# Performance — Tech Tree Designer

## Purpose Of This Document

Define qualification for OT-P0-008. Repository size must not make routine design depend on whole-fleet scans, eager file copies or rendering every node.

## Budgets

Numeric SLOs are **pending qualification**, not unlimited budgets or demonstrated guarantees. Before claiming performance acceptance, approve a reproducible cohort manifest with node/edge counts and degree distribution, artifact counts/bytes, revision depth, concurrency, hardware, memory/CPU limits, storage type, source latency and cache state.

| Workload | Required measurement | Acceptance decision |
|---|---|---|
| Search, neighborhood, coverage and first useful view | p50/p95/p99 latency, returned/visited nodes, bytes, cache age | Set query/depth/page/response limits and latency SLO per cohort |
| Graph interaction and artifact review | input latency, main-thread stalls, layout time, heap, viewport size | Set responsive UI and memory budgets; include compact devices |
| Proposal create/revise/diff | latency, bytes read/written, retained unique bytes | Bound selected scope and revision growth |
| Apply and recovery | completion/recovery latency, retries, owner calls | Bound per-owner work and expose progress |
| Refresh and maintenance | CPU time, peak RSS, disk IO, queue depth, cancellation lag | Bound concurrency and foreground interference |

Cohorts must include a representative current repository, a deliberately larger synthetic graph, high-degree/skewed graphs, large artifacts, many revisions, concurrent users/agents, cold/warm caches, slow/unavailable sources and a constrained host. The hundreds-of-thousands-of-scenarios vision is a stress horizon, not a current support claim.

## Current Measurements

### Proposed qualification profile v1

**Proposal, not an approved SLO or spending grant.** Review this profile with
`DECISIONS.md`'s development packet. Store the selected cohort seed/digest,
hardware and software revision in each future owner receipt. Reject comparisons
whose cohort, limits or cache state differ. Treat failed trials as failures, not
samples to discard. Numeric choices below are initial review candidates, not
measurements inferred from this host.

| Cohort | Proposed reproducible dimensions |
| --- | --- |
| repository | Snapshot current repository through SDA; record actual node/edge/artifact counts and hash rather than hardcode a moving count |
| large | Seeded 10,000 nodes / 100,000 edges; selected artifact data 64 MiB; 100 retained revisions |
| skew | Same size, one node with 5,000 incident edges; exercise explicit truncation and bounded pagination |
| constrained | 2 CPU, 4 GiB host memory, local SSD; service limit 1 GiB RSS; 390×844 and 1366×768 browser viewports |
| standard | 4 CPU, 8 GiB memory, SSD; four concurrent readers, one foreground proposal mutation and one background worker |
| degraded | Repeat cold/warm with 250 ms source delay, source unavailable, cancellation and queue saturation |

Proposed starting limits: 100 returned nodes/page (maximum 500), 1,000 returned
edges/page, traversal budget 10,000 visited nodes, depth maximum 4 and response
maximum 1 MiB. Limit one proposal to 200 selected artifacts / 64 MiB total /
2 MiB per text artifact. Reject unsafe/unsupported/oversized selections with a
typed limit result; never silently omit selected entries. Admission is per
proposal, not permission to copy the whole repository. Limit queue depth to 16;
return overload explicitly. Pinning overrides garbage collection, not capacity
admission; report storage pressure rather than evict referenced content.

| Workload | Proposed floor for repository/large/skew cohorts |
| --- | --- |
| Bounded warm query | p95 ≤250 ms, p99 ≤750 ms |
| Cold first useful view | p95 ≤2 s; source delay reported separately |
| Keyboard/selection response | p95 ≤100 ms; no single foreground task >200 ms |
| Bounded proposal create/diff | p95 ≤2 s at maximum selected bytes |
| Foreground cancellation acknowledgement | ≤1 s; owner effects may still require reconciliation |
| API memory / browser heap | ≤1 GiB RSS / ≤256 MiB heap on constrained cohort |

For each latency cell collect at least 100 trials in each of three independent
process runs, with separate cold/warm cells and retained failed samples. Require
all three runs to meet their thresholds. Publish distributions and sample counts,
not a percentile from one request. Apply/recovery uses deterministic fault-point
coverage plus observed timings; a multi-owner end-to-end latency promise awaits
the actual owner contracts. The 100,000+ scenario horizon remains stress-only.

Approval must confirm or amend these numbers before they become acceptance
bands. Do not weaken a proposed or approved threshold merely to fit a baseline;
report the tradeoff and required product or scope decision.

No reproducible product-scale baseline is established by this documentation update. Build duration and health responses do not demonstrate graph or proposal performance. Record measurement time, source revision, fixture identity, hardware and evidence location when measurements exist.

## Known Constraints

Use scoped owner queries, stable pagination, incremental refresh and bounded caches with explicit freshness. Keep graph layout asynchronous and avoid blocking interaction on full topology. Store references/deltas and lazily load artifact contents; avoid full repository copies per proposal. Cap artifact sizes, traversal, concurrent work and retained data with actionable limit responses. Unsupported scope must never appear as complete coverage.

Deletion/garbage collection must respect review and receipt pins. Background work requires cancellation and bounded queues; aborting a client must not silently abandon an effectful apply operation.

## Regression Procedure

1. Establish and approve cohort manifests and thresholds before claiming acceptance.
2. Run repeated cold/warm trials; include negative and saturation cases.
3. Capture latency distributions, CPU/RSS/heap/IO, bytes visited/retained and foreground responsiveness.
4. Compare like-for-like baselines; classify noisy or incomparable results as inconclusive.
5. Link evidence to requirements and record regressions without relaxing targets solely to pass.

## Cross-References

- [Testing](TESTING.md)
- [Observability](../operations/OBSERVABILITY.md)
- [Data](../concepts/DATA.md)
