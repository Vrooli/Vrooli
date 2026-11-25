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

`compare.py` is the post-run attribution step. It requires baseline and
candidate reports plus separate sanitized grade receipts issued by SH1. A
receipt has this shape (one row per frozen `case_id`; no expected IDs, query
text, or result content):

```json
{
  "schema_version": 1,
  "issuer": "SH1",
  "receipt_id": "sh1-receipt-reference",
  "evidence_uri": "<SH1-restricted-evaluator-evidence-URI>",
  "evidence_sha256": "<source-evaluation-artifact-digest>",
  "run_tag": "sh1-holdout-v2-candidate-build-x",
  "evaluation_run_ids": ["<stored-eval-run-id>"],
  "tier": "federated",
  "build_id": "sha256:<graded-build-identity>",
  "acceptance_policy_id": "sh1-frozen-part-task-policy",
  "cohort_id": "sh1-holdout-v2-20261123",
  "task_set_sha256": "<frozen-task-set-digest>",
  "cases": {
    "case-001": {
      "query_sha256": "<exact-frozen-query-digest>",
      "parts": {"accepted": 2, "total": 3, "verdicts": [true, true, false]},
      "tasks": {"accepted": 0, "total": 1, "verdicts": [false]}
    }
  }
}
```

Each arm has its own receipt because acceptance can change. The comparator
checks workload digest, phase, concurrency, transport, sampling interval, case
IDs, and per-case query hashes before computing group CPU per accepted part
and task. Each grade row's query digest must match the measured request, each
receipt's build ID must match that measured arm, and run tags must be distinct.
The default grade tier is `federated` (choose `provider_direct` only for a
provider-direct measurement). It also requires matching frozen corpus and
fault-schedule IDs, and matching independently graded cohort/task-set/tier,
case-query-set identity, acceptance-policy ID, and per-case denominators. Each
`verdicts` list follows the frozen case part/task order. The comparator rejects
denominator drift and reports per-part/per-task regression counts without
including per-case verdicts in its output; unchanged aggregate acceptance
cannot hide a lost previously accepted part. SH1 retains the verdicts and
supplies only the per-case counts and identity hashes needed for this join. Each
comparison is a content-addressed receipt with source artifact hashes, state
receipt IDs, build IDs, the selected resource-group definition, and a digest of
case/query hashes. It reports peak memory and keeps cache and Gateway token evidence in
separate fields. Select exactly one resource group and source; that group must
cover the declared serving boundary. Do not add
process, PSS, and inclusive cgroup totals. Candidate part and task acceptance
rates must both be non-regressing for a resource improvement to be eligible.
It also reports selected-group point-estimate checks against SH4's 20%
CPU-per-accepted-part and 15% peak-memory floors. A missing owner-scope receipt
marks them ineligible. Even with a receipt reference these are not a final pass:
the whole-serving boundary and receipt semantics remain subject to review, and repeated paired runs,
uncertainty, idle bounds, provider/model attribution, and an independently
reviewed quality gate still apply.

The current public SH1 scorecard is aggregate; SH4 must not expand those totals
into case-level verdicts. SH1 must supply the sanitized per-case receipt above
from its restricted grading store, retaining a verified run tag/build/tier and
the frozen query hashes. The `issuer` field and file hash aid traceability but
are not a cryptographic signature; receipt provenance remains an SH1/manager
review responsibility.

## SH1 producer handoff

The normative field allowlist is
[`sh1-grade-export-v1.schema.json`](sh1-grade-export-v1.schema.json). It defines
the minimum SH1-to-SH4 export, not a new grading rule. SH1 remains the only
producer of verdicts and must derive the export from its restricted evaluator
results:

1. Bind one export to the frozen `cohort_id` and `task_set_sha256`, one tier,
   one verified serving `build_id`, and the exact SH1 `run_tag` plus stored
   evaluator run ID(s). Use `federated` for ordinary Search Hub Query.
2. For every frozen case, copy only its case ID and SHA-256 query digest. The
   digest must match both the evaluator suite and measured harness request.
3. Emit one boolean per frozen part in frozen `part_ids` order, plus one
   complete-task boolean. Include `accepted` and `total` counts, which SH4
   recomputes from the booleans. Bind decisions to a stable
   `acceptance_policy_id`; SH4 never selects or changes acceptance thresholds.
4. Bind the export to its restricted SH1 evaluation artifact with
   `evidence_uri` and `evidence_sha256`. SH1 or the manager must verify this
   provenance; the JSON `issuer` value is not authentication.
5. Exclude query text, expected identities, source labels, result bodies and
   per-hit data. Unknown fields are rejected so private grading data cannot
   accidentally flow through this interface.

SH4 integration checks confirm exact case coverage and query hashes, measured
build/tier/run-tag linkage, shared frozen denominator and acceptance policy,
exact per-item verdict counts, and per-part/task non-regression. The comparator
only emits aggregate acceptance, regression counts and source references. SH1
has not yet supplied this per-case export; the existing public scorecard alone
is insufficient and cannot be substituted or expanded by SH4.

Example, run once per matched phase after SH1 supplies the sanitized receipts:

```sh
python3 -I scenarios/search-hub/scripts/performance/compare.py \
  --baseline evidence/baseline-warm.json \
  --candidate evidence/candidate-warm.json \
  --baseline-grades evidence/sh1-baseline-grades.json \
  --candidate-grades evidence/sh1-candidate-grades.json \
  --resource-source cgroup_groups --resource-group api \
  --output evidence/warm-comparison.json
```

## Diagnostic component resource groups

`compare.py` can report named component groups alongside the primary serving
total. Define the groups in each `measure.py` run with repeated
`--cgroup-group`, `--process-group`, or `--pss-group` declarations, then name
stable roles in the pair command, for example:

```sh
--resource-source cgroup_groups --resource-group serving_total \
--component-group search_hub_api=search_hub \
--component-group model_worker=model_worker
```

The manifest plan can pin the role map with `"component_groups":
{"search_hub_api":"search_hub", "model_worker":"model_worker"}`. It
recomputes per-role CPU per accepted part/task and memory values from the
hash-checked source reports, then reports separate paired intervals per phase.
Roles and group definitions must remain identical across all repeats. Group
overlap is unknown, so outputs are never added; only the separate primary
serving-total group is used as the SH4 total-cost floor input. Component rows
are diagnostic process/cgroup attribution, not proof that a provider/model is
included or disjoint. They do not replace the signed provider/model scope
receipt or owner semantic review.

Every measurement report, including idle, must declare the externally verified
build, corpus snapshot, fault-schedule receipt (use `none` for the normal path),
owner-issued prepared-state receipt, and manager-issued host admission receipt, using `measure.py`'s
`--build-id`, `--corpus-snapshot-id`, `--fault-schedule-id`, and
`--state-receipt-id`, and `--admission-receipt-id` options. Use the same frozen
corpus and fault schedule for both arms; state and build receipts are retained
per arm. The runner records IDs only; the run manifest stores artifact hashes
and does not authenticate an ID's issuer or contents.

Pass `--serving-scope-receipt-id` when an owner-issued receipt defines the
complete measured serving boundary. Its referenced artifact must be listed in
the manifest's `evidence_receipts`. The receipt should identify the Search Hub
component, each attributable provider/model component, selected meter/resource
group, included/excluded components, shared-worker allocation method, and
unmeasured or remote costs. The harness checks that the same receipt ID is
declared for baseline/candidate and loaded/idle reports, and checks the listed
artifact digest. It authenticates the receipt signature only against a pinned
key; it does not establish that the signer owns every named component or that
the selected group actually covers it. Missing receipt references make
CPU/memory floor signals ineligible; even a verified signature is necessary
but not sufficient to establish complete provider/model attribution.

Scope receipts must match
[`provider-model-scope-receipt-v1.schema.json`](provider-model-scope-receipt-v1.schema.json)
and carry an RSA PKCS#1 v1.5/SHA-256 signature over canonical JSON with the
`signature_base64` field omitted. The signer key must be pinned by key ID and
issuer in the source-controlled
[`provider-model-scope-trust-v1.json`](provider-model-scope-trust-v1.json).
That policy currently contains no keys, so all scope receipts fail closed until
an owner key is reviewed and added through source control. Key admission
requires an owner acknowledgement reference, a manager review reference,
matching key fingerprint, purpose-limited key entry, admission-receipt ID and
SHA-256, and two distinct reviewer IDs. Each manifest using the key must list
the exact key-admission receipt artifact in `evidence_receipts`; the manifest
checks its digest and fields against the pinned key entry. To revoke a key,
mark it disabled in a reviewed trust-policy change; do not remove its historical
admission evidence. Signature verification proves which pinned key signed the
coverage assertion; the owner still must provide auditable evidence that the
selected resource group includes every declared component and accounts for
shared or remote costs.

The exact receipt shapes are
[`provider-model-scope-key-admission-v1.schema.json`](provider-model-scope-key-admission-v1.schema.json)
and [`provider-model-scope-trust-v1.schema.json`](provider-model-scope-trust-v1.schema.json).
The admission artifact records the owner and manager references and reviewer
IDs. The trust policy pins its content digest and the fingerprint of the
canonical `(issuer, key_id, modulus, exponent)` tuple. Neither the private key
nor a secret is stored in the repository or run artifact.

Repeat this for `cold`, `warm`, `concurrent`, and `degraded` using equal
workload/corpus/options and the same fault schedule. Obtain cold/degraded state
through the supported owner lifecycle and cite its receipt with `--note`; this
harness and comparator do not change service state. Keep the `model-free`
contrast separately labeled and never merge it into normal-path results.

## Whole-run manifest and idle gate

`manifest.py` validates a reviewed plan after the per-phase comparisons exist.
It requires at least three pair receipts each for cold, warm, concurrent and
degraded, explicit concurrency for every phase, stable workload/case-query
digest and grade denominator, per-phase repeated fault schedule, one selected
resource group, and matching build/admission/corpus identity. It re-hashes each
phase receipt's four report/grade source files and each listed admission,
corpus, state, and fault receipt artifact. It rejects overlapping evidence
windows across idle and loaded runs. A degraded phase
must cite a non-`none` fault receipt. An optional model-free phase is reported
separately. It also takes at least three paired idle reports and checks the
For idle CPU rate and peak-memory, the manifest computes two-sided 95%
Student-t intervals from the baseline replicates and paired
candidate-minus-baseline intervals and requires each full interval to fit
within the corresponding baseline confidence half-width. This prevents an
unchanged candidate mean with much higher run-to-run variation from passing.
The paired runs must remain sequential and share duration, sampling interval,
and selected group definition. This is an audit check, not a performance pass:
repeat count, host isolation, attribution quality and the approved admission
receipt still require reviewer assessment.

Each `phase_receipts.<phase>` array must contain at least three entries. Each
entry names one comparison receipt plus all four source artifacts so the
manifest can re-hash them. Paths are relative to the plan file unless absolute.
The excerpt below shows the core fields and one receipt entry; a complete plan
must add the remaining two or more entries for each required phase:

```json
{
  "manifest_id": "sh4-cohort-v3-window-01",
  "admission_receipt_id": "manager-admission-reference",
  "corpus_snapshot_id": "approved-corpus-snapshot",
  "resource_source": "cgroup_groups",
  "resource_group": "api",
  "minimum_repeats": 3,
  "concurrency_by_phase": {"cold": 1, "warm": 1, "concurrent": 8, "degraded": 1},
  "evidence_receipts": [
    {"receipt_id": "manager-admission-reference", "path": "receipts/admission.json", "sha256": "<file-digest>"},
    {"receipt_id": "approved-corpus-snapshot", "path": "receipts/corpus.json", "sha256": "<file-digest>"},
    {"receipt_id": "owner-state-reference", "path": "receipts/state.json", "sha256": "<file-digest>"},
    {"receipt_id": "approved-degraded-fault-reference", "path": "receipts/fault.json", "sha256": "<file-digest>"}
  ],
  "phase_receipts": {
    "cold": [{
      "receipt": "receipts/cold-01.json",
      "inputs": {
        "baseline_report": "reports/cold-base-01.json",
        "candidate_report": "reports/cold-candidate-01.json",
        "baseline_grade_receipt": "grades/base-01.json",
        "candidate_grade_receipt": "grades/candidate-01.json"
      }
    }]
  },
  "idle_reports": {
    "baseline": ["idle/base-01.json", "idle/base-02.json", "idle/base-03.json"],
    "candidate": ["idle/candidate-01.json", "idle/candidate-02.json", "idle/candidate-03.json"]
  }
}
```

List every referenced state/fault/admission/corpus receipt ID exactly once in
`evidence_receipts`; the entries above are illustrative and real manifests
must include every per-arm state ID across all repeats. The listed file hashes
are checked before the manifest is written.

After admission and evidence exist, create the manifest with:

```sh
python3 -I scenarios/search-hub/scripts/performance/manifest.py \
  --plan evidence/sh4-run-plan.json \
  --output evidence/sh4-run-manifest.json
```

This source contract has not been run. It does not create admission, corpus,
state, fault, grade, or measurement evidence.

The manifest emits `phase_budget_rollup` as a descriptive, per-phase summary.
For each of cold, warm, concurrent, degraded, and optional model-free, it
calculates candidate/baseline ratios from matched repeats for CPU per accepted
part/task, peak memory, client p95, and API-reported p95. Ratios are never
pooled across phases. A metric is marked `unavailable` if any repeat lacks a
positive finite arm value; missing latency is not zero. CPU-per-part and peak
memory confidence floors are eligible only when every repeat's independent
part/task acceptance gate is true and the owner-scope receipt artifact is
listed with a matching digest. The reported 95% Student-t interval is over
within-repeat ratios; `conditional_threshold_met` reports only that the interval
is below its numeric threshold after quality/reference gates. It remains
pending semantic provider/model scope review and SH4 signoff. It does not
replace review of provider/model attribution, SH1 receipt
provenance, host isolation, or approved matched-run evidence.

Each phase retains token and cache telemetry per pair receipt, with explicit
scope descriptions. Gateway tokens are fresh Search Hub events between two
snapshot windows; they are not request/run correlated. Resolver-cache deltas
come from rolling 15-minute Insights counters and can include unrelated Search
Hub queries. The present sources therefore remain `not_run_attributable`, even
when technically available, and are not ratio-aggregated. They are never added,
converted to CPU, or pooled between phases. Missing, truncated, or contaminated
inputs remain explicit rather than becoming zero. A future producer can enable
comparison only after both arms export complete token/cache evidence with
`available: true`, `attribution_status: "run_correlated"`, the exact pair
`run_id`, and frozen `case_query_set_sha256`. The rollup then reports input and
output tokens per independently accepted part as paired ratios, and cache hit
rate as a paired candidate-minus-baseline difference; each has its own interval
and stays inside its phase. The current runner does not emit this correlation,
so these fields are unavailable for its stock Insights/Gateway snapshots.
`idle_check` retains each receipt's CPU ms/second and peak memory normalized to
bytes, and compares candidate idle means against baseline intervals.
The rollup includes no claim that these inputs exist until actual reports and
SH1 receipts pass the manifest gates.

### Owner-signed run-correlated telemetry

`run_telemetry.py` now defines a source-only producer adapter for complete
Gateway token events and resolver-cache hit/miss counts exported with exact
case/query identities. It rejects incomplete, truncated, mismatched, duplicate
or non-frozen coverage and emits an unsigned payload. An authorized owner
must verify the export artifacts and sign that payload; SH4 never handles the
private key. Both source exports need stable receipt IDs, evidence URIs and
SHA-256 hashes. The resulting signed receipt binds the measured run ID, arm,
phase and frozen query-set digest. This export format is not yet implemented
by Gateway or cache services.

When owner receipts and a reviewed purpose-limited key are available, pass
both `--baseline-telemetry` and `--candidate-telemetry` to `compare.py`. The
manifest must list those exact receipts plus their source-export and key
admission artifacts under `evidence_receipts`; it verifies each signature,
hash and report/grade/run linkage before marking telemetry run-attributed.
Input and output tokens per SH1-accepted part remain distinct; resolver cache
is a separate hit-rate delta. They are reported per phase and never pooled or
converted to CPU. `run-telemetry-trust-v1.json` is intentionally empty until
owner acknowledgement, manager review, two reviewer IDs, and the exact key
fingerprint/admission receipt digest are supplied through reviewed source
control. Synthetic tests do not establish an actual signed comparison. See
the SH4 run evidence `run-correlated-telemetry-v1.md` for the export fields,
signing sequence, missing authority, checks and rollback.

After `manifest.py --output <bundle>` succeeds, replay the reviewed plan and
all referenced artifacts with:

```sh
python3 -I scenarios/search-hub/scripts/performance/manifest.py \
  --plan evidence/showcase-plan.json \
  --verify-bundle evidence/showcase-manifest.json
```

Replay regenerates the evidence bundle from current inputs and trust policy,
then compares bytes with the saved output. Changed report, grade, idle sample,
pair receipt, authority artifact, trust configuration or bundle content fails
verification. This establishes reproducibility against the current inputs,
not new authentication or semantic proof. Details and rollback are in the
SH4 run's `evidence/manifest-replay-v1.md`.

The generated bundle also contains an artifact provenance index: plan digest,
each pair receipt and its role-labeled report/grade/optional telemetry inputs,
plus the existing idle and authority receipt hashes. It records basenames and
digests, while the reviewed local plan retains full paths. This makes the
evidence chain easier to navigate without changing trust eligibility. The SH4
run's `evidence/artifact-provenance-index-v1.md` documents the fields, proof
and rollback.

Before reading the referenced artifacts, manifest preflight requires a closed
plan field set, exact phase/concurrency keys, paired report/grade inputs, both
telemetry inputs or neither, unique digest-listed authority receipts, and at
least three idle reports per arm. An optional `model-free` phase remains
separate. Unknown fields fail closed. The SH4 run's
`evidence/showcase-plan-preflight-v1.md` records the precise contract and
synthetic rejection cases.

Paired CPU, memory, latency, idle and eligible token intervals use the
two-sided 95% Student-t critical value. The checked table covers df 2–30;
larger repeat counts use a documented Cornish-Fisher expansion rather than a
normal-quantile fallback. This changes interval arithmetic only; it does not
relax the evidence gates. The SH4 run's
`evidence/student-t-interval-large-samples-v1.md` records the reference checks
and rollback.

The CPU target is separately gated per independently accepted part and per
completed accepted task: each candidate/baseline ratio must be at most 0.80.
The task denominator comes only from SH1's full-task grade receipts. The
20%-lower point check also reports both denominators; missing accepted tasks
cannot be inferred from part acceptance. See the SH4 run's
`evidence/full-task-cost-floor-v1.md`.

Client-observed p95 latency has a separate candidate/baseline interval floor
of 0.70, and requires every repeat's independent part and full-task quality
gate. API-reported p95 remains diagnostic and separate; missing p95 prevents a
pass. Latency does not require the provider/model cost-scope receipt, while
CPU and memory still do. Method, tests and rollback:
SH4's `evidence/client-p95-quality-gate-v1.md`.

For an operator-facing inventory of missing or hash-mismatched local inputs,
use `manifest.py --plan <plan> --preflight-report <new-report.json>`. It lists
artifact roles and basename/digest status plus pending trust/scope blockers;
it does not validate grade semantics or declare the showcase ready. Only
`--output` performs full manifest validation, and `--verify-bundle` replays it.
The report is write-once and cannot overwrite the plan or a referenced file.
Evidence and rollback: SH4's `evidence/preflight-readiness-report-v1.md`.

For a complete reviewed plan, `manifest.py --plan <plan> --output <new-bundle.json> --audit-report <new-summary.json>` validates all receipts, stages the bundle, replays it, and writes both outputs only after the checks succeed. Its summary displays per-phase quality/cost/telemetry gate states while always disabling performance claims pending approved review. This does not make unavailable owner evidence available. See the SH4 run's `evidence/integrated-showcase-audit-v1.md` for the report contract and rollback.

The bundle and operator summary include an integrated gate matrix combining
per-phase part/task CPU, memory, client p95, quality, idle bounds, provider/model
scope, and token/cache reporting availability. Threshold calculations are
marked conditional and never enable a performance claim. Missing telemetry is
separate from numerical floor blockers but prevents complete reporting. See
the SH4 run's `evidence/integrated-showcase-gate-matrix-v1.md`.

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

For a future owner implementation that can export request-correlated routing
spans, `stage_telemetry.py` defines a strict five-stage contract (classifier,
resolver, fanout, rerank, provider). The producer validates complete exact
run/arm/phase and case/query-hash coverage, with explicit executed versus
not-applicable stage state; missing stages are never represented as zero. It
returns an unsigned canonical payload for the trace owner to sign. The
`compare.py --baseline-stage-trace <receipt> --candidate-stage-trace <receipt>`
path verifies both signatures and emits per-pair distributions and source
hashes. The manifest accepts paired stage input roles, re-verifies receipts
under current trust, checks key-admission/source evidence hashes, reproduces
pair summaries, and rolls up stage total-duration ratios by phase across at
least three pairs. `stage-telemetry-trust-v1.json` remains deliberately empty;
absent receipts stay unavailable. These are descriptive latency diagnostics,
not a stage latency floor. Optional stage resources now carry per-case CPU
milliseconds and peak memory bytes with a method ID and hashed scope definition.
`compare.py` requires that method and scope to match between arms and divides
stage CPU by independent SH1 accepted-part/task denominators; the manifest
requires method/scope parity across repeats, a scope receipt ID/URI/SHA-256
entry in the reviewed plan's `evidence_receipts`, and reports separate
per-stage paired intervals after at least three complete repeats. Stage memory
peaks are not summed, and stage values never feed the primary serving-total
floor. The operator audit summary exposes observed scope links and keeps
incomplete stage metrics unavailable. Legacy latency-only receipts remain
valid. The owner exporter, semantic method review and trusted owner key remain
pending.
See SH4 `evidence/stage-span-contract-v1.md`,
`evidence/stage-span-pair-integration-v1.md`, and
`evidence/stage-resource-attribution-v1.md` for contracts and rollback.

Process groups match Linux `/proc/<pid>/comm` with a regular expression. Example:

```sh
python3 -I scenarios/search-hub/scripts/performance/measure.py \
  --workload /path/to/frozen-workload.jsonl \
  --output /path/to/evidence/baseline-warm.json \
  --phase warm --run-id sh4-cohort-01 --arm baseline \
  --build-id sha256:<verified-build> \
  --corpus-snapshot-id <approved-snapshot> \
  --fault-schedule-id none --state-receipt-id <owner-state-receipt> \
  --admission-receipt-id <manager-admission-receipt> \
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
