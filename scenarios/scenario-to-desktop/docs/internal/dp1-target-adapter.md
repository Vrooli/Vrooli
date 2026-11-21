# DP1 desktop startup target adapter

`api/desktopoptimize` is the scenario-owned second target adapter for LM6. It
defines target `desktop-startup`, request schema
`desktop-startup.optimize-request.v1`, receipt schema
`desktop-startup.optimize-receipt.v1`, `MinimumPlan`, the staged
`Measure`/`Promote`/`Cleanup` APIs, and the aggregate
`Execute(context.Context, Dependencies, Request) Receipt` wrapper. It has no
Program Runtime dependency. The Program Runtime side owns orchestration and
maps its target contract to these APIs.

## Minimum matrix

DP1 is a paired control/candidate startup matrix on one fixed cohort:
`scenario`, predecessor artifact digest, host fingerprint, display, deployment
mode, and profiler mode must match. The denominator is five pairs in each of
`cold` and `warm` for each of these scenario-to-desktop measured phases:

| Phase | Meaning |
|---|---|
| `process_to_splash_first_paint` | Process launch to first splash paint |
| `splash_to_app_ready` | Splash paint to app ready |
| `runtime_spawn_to_ready` | Bundled runtime spawn to ready |

Each pair includes a common `pair_id`, separate control/candidate launch IDs,
nonempty evidence references, and nonnegative durations. The matrix carries
the baseline cohort identity and exact candidate artifact digest. Missing or
unavailable phases fail closed; they do not shrink the denominator. `MinimumPlan`
provides the evaluator the exact target/classes/phases/count and its pairing
rule. The evaluator is the scenario-to-desktop owner that performs actual
launch measurements and emits target-owned evidence.

`docs/internal/performance-baseline.json` remains a historical baseline
candidate, not a live sensor or approved budget. Its five cold/five warm samples
are not DP1 paired candidate measurements. A future evaluator must create the
paired matrix above, preserving artifact/host/display/mode/profile identity.
Source tests use synthetic matrices only to verify contract and refusal logic;
they are not performance evidence.

## Admission, mutation, and recovery

Before measurement, requests require current preflight LM5 rights and disk
proofs bound to the pre-acquisition request digest. Before promotion and
predecessor removal, they require fresh proofs bound to the post-acquisition
candidate cohort; disk admission must cover the requested bytes. The aggregate
`Execute` wrapper uses its one rights/disk proof set for both checkpoints.
The staged adapter refuses missing, expired, mismatched, or insufficient
proofs before invoking each owner. It delegates measurement,
promotion, predecessor removal, and rollback through injected interfaces.
Promotion occurs only after a complete matrix and a p95 no-regression check on
all phases plus a median first-paint improvement in at least one cohort. It
then removes the predecessor through its owner; cleanup failure invokes owner
rollback and records both results.

The append-only JSONL `FileJournal` flushes measurement, promotion, and
predecessor-removal intents and stage receipts; `Find(run_id)` returns the most
recent receipt. A retry replays an incomplete owner call with the same stable
run/stage operation ID, which the owner must use to return its original receipt
instead of repeating the effect. A different immutable target/acquisition
request under the same run ID is refused. The journal digest excludes the two
stage proof sets and their binding digests, which are validated afresh before
their respective effects. The source interface intentionally does not
implement artifact storage or invent a host-side optimization operation.

## Parent engine wire mapping

Required request fields: `schema`, `run_id`, `target`, `subject`, `baseline_ref`,
`candidate`, `acquisition`, `minimum_pairs`, and `max_regression_fraction`;
staged calls also require `preflight_rights`/`preflight_disk` before measurement
and `rights`/`disk` before promotion and cleanup. The immutable request digest
binds the full target/acquisition identity and settings, excluding stage proof
sets and proof-binding digests.
Receipt fields include terminal `disposition`, `reason`, timestamps, request
digest, subject and candidate, six class/phase summaries with both evidence
reference lists, promotion receipt, predecessor-removal receipt, and optional
rollback receipt. See the Go JSON tags as canonical field spelling.

Program Runtime's staged adapter integration, field mapping, proof-binding gap,
and supported transaction shapes are specified in
[`lm6-program-runtime-handoff.md`](lm6-program-runtime-handoff.md). The current
`Measure`, `Promote`, and `Cleanup` operations expose the staged owner boundary;
`Execute` remains a wrapper. Owner callbacks receive stable per-stage operation
IDs, and the request's acquisition proof binds the exact acquired candidate
digest before measurement or promotion. A Program Runtime adapter must pass its
canonical `proof_binding_digest` and gate proofs unchanged.

