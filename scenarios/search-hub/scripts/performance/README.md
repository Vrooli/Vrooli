# Search Hub load measurement

`measure.py` runs the same JSONL workload through the normal `search-hub query`
CLI (or directly through the same Connect `RoutingService.Query` API contract)
and records per-case client wall time, server-reported API latency/status/size, privacy-safe
counts of query-part coverage/degradation when supplied by the response, plus sampled
CPU ticks and resident memory for explicitly named Linux processes. Optional
cgroup groups add service-scope CPU and sampled `memory.current`; PSS groups add
proportional memory and SwapPss for shared model workers. Each report includes
the harness process CPU time so measurement overhead stays visible. It
also snapshots Search Hub's rolling 15-minute Insights counters before and after the
batch, then records cache deltas, and reads metadata-only AI Gateway evidence
for new Search Hub events to report provider-reported model tokens. Reports
retain only case IDs, SHA-256 query hashes, and redacted route metadata.
Queries, result bodies, query-part/provider identifiers, paths and hit text are
not copied into measurement output. Part status/evidence counts are availability
signals only; they do not replace SH1's independent per-part grade.

Each report summarizes `latency_ms` as client wall-time percentiles,
`api_response_latency_ms` as the server's `latencyMs` percentiles when present,
and `response_bytes` as output size. The wall/API difference includes client,
transport and server response handling; do not attribute it to a specific stage.

## Pairing equivalent arms

Use a frozen SH1 workload file and the same `--run-id`, workload hash, target
provider/corpus snapshot, and request options for every arm. Save one report for
each phase (`cold`, `warm`, `concurrent`, `degraded`, `model-free`). Repeat each
arm enough to estimate its distribution. The phase label is descriptive: this
tool never restarts Search Hub, stops providers, toggles models, or injects
faults. Prepare state with the supported scenario/resource owner and cite the
state/fault receipt with `--note`.

The harness does not grade output. Join case IDs to SH1's independent per-part
grading and task-acceptance receipt before calculating cost per accepted result.
Attach Search Hub resolver-cache and embedding/model token evidence separately;
the query response does not expose per-request stage telemetry. Stored router
telemetry has aggregate classifier, resolver, fanout, rerank and provider-leg
latencies, but the ordinary Insights RPC does not return those stage records or
request-level correlation. This harness therefore attributes end-to-end request
time and process/cgroup work, not per-stage work. It also does not currently
provide complete provider/model token and process-level cost attribution. Do not compare arms
when their workload hash, corpus snapshot, query options, or fault schedule
differ.

Process groups match Linux `/proc/<pid>/comm` with a regular expression. Example:

```sh
python3 -I scenarios/search-hub/scripts/performance/measure.py \
  --workload /path/to/frozen-workload.jsonl \
  --output /path/to/evidence/baseline-warm.json \
  --phase warm --run-id sh4-cohort-01 --arm baseline \
  --concurrency 1 \
  --api-base http://localhost:19157 \
  --metrics-api-base http://localhost:19157 \
  --process-group 'search-hub=search-hub-api' \
  --process-group 'providers=(web-search-api|workflow-health-api)' \
  --cgroup-group 'search-hub=^search-hub-api$' \
  --cgroup-group 'providers=^(web-search-api|workflow-health-api)$' \
  --pss-group 'model=^(ollama_llama|llama-server)'
```

If the local CLI cannot refresh or build, use the ordinary API directly with
`--transport connect --api-base http://localhost:<API_PORT>`. This keeps the
same request fields and Query RPC while avoiding the CLI launcher path.

The `/proc` sampler is system-wide but only reports configured process groups.
Use narrow expressions against `/proc/<pid>/comm` (truncated to 15 characters)
so unrelated services are not included. A cgroup group reports v2 `cpu.stat`
usage deltas and sampled `memory.current` for matching processes' scopes. It
deduplicates nested scopes before aggregation and records the exact cgroup paths.
Both kernel counters include descendant cgroups, so nested scopes must not be
summed twice ([Linux cgroup v2 documentation](https://docs.kernel.org/admin-guide/cgroup-v2.html)).
It does not reset `memory.peak`; use the sampled peak as an estimate and do not
use a scope's lifetime high-water mark as a run peak. A PSS group samples
`smaps_rollup` PSS, shared PSS and SwapPss plus process CPU ticks. Use that for
model workers whose daemon/worker processes share a broader cgroup with other
agents. Process PSS/CPU still reflects all callers of a shared worker, so use an
isolated serving window or mark that cost unattributable. Do not add process RSS
and PSS or an inclusive cgroup total together as separate costs.
When `--api-base` or `--metrics-api-base` is supplied, the runner reads the
existing Search Hub `MetricsService.Insights` RPC with a rolling 15-minute
window for cache snapshots;
without one, cache deltas are marked unavailable. It reads AI Gateway's
existing `routing evidence-list --scenario search-hub` contract for token
usage. Those events are not correlated to individual task hashes, so unrelated
concurrent Search Hub queries can contaminate token/cache deltas. Run paired
arms in an isolated serving window and report unavailable or truncated token
evidence rather than inferring zero usage.
Optional snapshot calls use `--evidence-timeout-s` (default 10 seconds); they
run outside the sampled request interval. A timeout keeps that evidence
unavailable and does not invalidate completed query rows.
RSS is summed for matched processes and can double-count shared pages. Its
1,000 ms default can miss short peaks; use identical sampling intervals across
arms and corroborate peak memory with the owning cgroup's memory peak when
available. CPU is cumulative process CPU time over the measurement interval.
Do not treat a CLI exit code as quality acceptance.
