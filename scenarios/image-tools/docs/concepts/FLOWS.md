# Flows — Image Tools


This document maps the stateful user and system workflows implemented by
Image Tools. It is not a list of aspirational features: a flow marked
**partial** has an explicit remaining proof or integration gate.

## Flow Inventory

| Flow | Owner | Current maturity | Main state / ordering rule |
|---|---|---|---|
| Durable model job | `internal/jobs`, `internal/jobrunner` | Implemented; live IT2 showcase partly qualified | A job is admitted before execution, releases owned claims before terminal persistence, and can be canceled only through cleanup. Capacity denial can requeue work without starting it. |
| Model lifecycle | `internal/models` | Explicit transition table and persisted evidence; full fresh-host provisioning showcase open | Install and runtime smoke evidence are separate; Ready binds artifact and runtime hashes. Changed artifacts/runtime make prior smoke stale. |
| Adapter lifecycle | `internal/adapters` | Install, import and cross-process reader ownership implemented; full lifecycle matrix open | A model job holds an adapter reader through worker reap; install/remove must not replace a live artifact. |
| Local capacity admission and residency | Image Tools job manager + shared platform capacity clients | Focused and live partial evidence; direct GPU contention remains open | CPU/RAM/GPU claims belong to an admitted job; warm workers retain separate residency claims and are reaped before ownership is released. |
| Look compilation and execution | `internal/looks` | Implemented as the single compound workflow format | A Look compiles ordered operation steps from the shared operation vocabulary; there is no separate recipe graph. |
| Quality evaluation and model comparison | `internal/quality` | Evaluator and reference metrics implemented; judge-backed default change remains open | Evaluation binds outputs, references, rubric, budget and provenance; promotion requires evidence, not a metric-only ranking. |
| Model/adapter import | `internal/models`, `internal/adapters` | Inspected import and explicit commercial-rights attestation implemented | Registration and weight acquisition are refused until required metadata and rights attestation are present. Attestation is not legal review. |
| Photographic preparation and review | `internal/sceneprep` | Durable metadata/idempotency substrate implemented; full consumer journey is partial | A stable operation id maps retries to one immutable prepared photograph. Pixels and geometry remain blob references. |

## State Machines

The job and model tables below are the persisted state machines. Adapter
reader leases and model-use leases add cleanup ordering to the job transitions.

### Durable model job lifecycle

A client submits an operation through API, CLI or UI. Image Tools validates
its request, stores any retained input reference and creates a durable job.
The scheduler owns execution; disconnecting a client does not cancel the job.

The transition table in `internal/jobs/lifecycle.go` is authoritative:

```text
queued → admitting → running → succeeded | failed
   │         │          │
   └→ canceled   → canceling → canceled | failed
             └→ queued       (capacity refusal before work starts)
```

The `admitting → queued` transition is a retryable capacity deferral; it must
not execute provider work. A running cancellation moves through `canceling`.
The manager waits for worker reap and cancellation-independent cleanup before
it writes terminal `canceled` state. A cleanup failure is a failure, not a
successful cancellation. Orphaned nonterminal rows recovered after process loss
are failed/interrupted; execution is not guessed or duplicated.

For local model work, admission reserves CPU and RAM plus a GPU claim when the
resolved tier is local GPU. Model and adapter artifact readers are acquired
before execution. The shared capacity client renews cross-process leases. Warm
ONNX workers keep their own RAM/model residency claims between jobs and are
reaped on idle expiry, model switch or cancellation. Per-job claims are
released independently. `require_gpu` is an opt-in execution constraint: a
local CPU/cloud resolution or broker CPU degradation does not count as a
successful GPU job.

Cancellation proof and mixed-job receipts are recorded in the current Image
Tools research checkpoint for IT2. Unit tests prove transition, denial and
cleanup behavior; those do not replace served cross-scenario GPU contention.

### Model lifecycle

The persisted model lifecycle is owned by `internal/models/lifecycle.go`; the
state describes readiness independently from enablement and hardware fit.

| State | Meaning |
|---|---|
| `absent` | No valid installed artifact is available. |
| `installing` | The installer owns acquisition and verification. |
| `installed` | Artifact exists, but runtime/build/smoke readiness is not yet proven. |
| `building` | A derived model artifact is being prepared. |
| `smoking` | The current artifact/runtime pair is being checked. |
| `ready` | Current artifact and runtime fingerprints have a passing applicable smoke verdict. |
| `stale` | Prior readiness evidence no longer matches current artifacts/runtime, or validation is pending. |
| `failed` | The current lifecycle attempt failed with a persisted reason. |

The installer and `models doctor` read the same lifecycle record. A read path
reports readiness but does not trigger downloads, model builds or device probes.
Explicit install/provision performs revalidation. Enablement and host fit remain
separate gates: `ready` alone does not promise that a model is enabled or usable
on every machine.

The one-command `image-tools provision` entrypoint submits a durable
preparation job for runtime, backend tools and models. Its present contract and
receipts are narrower than the IT1 showcase: a fresh-host run proving every
enabled capability installed, smoke-tested and ready, with total time measured,
remains open.

### Adapter lifecycle and model reads

Seed adapters are read-only catalog entries; imported/custom adapters are
add-only. An explicit rights attestation is required before acquisition.
Installation uses staged files and integrity verification. A cross-process
reader lease protects each adapter from removal/replacement while a model job
uses it. The job holds its model and adapter readers until worker reap, and
cleanup is cancellation-independent. A lease release failure is surfaced rather
than presented as successful removal or cancellation.

### Look compilation and execution

Looks is the only compound-workflow format. Built-ins are declared in
`internal/looks/seed.go`; custom entries are stored in the Look schema. The
compiler validates a Look against the shared operation vocabulary and emits an
ordered list of normal operation steps. There is no `recipes` package, schema,
CLI or second pipeline representation. Model-backed steps still use their
normal model selection, admission, job and cancellation paths.

Looks now carries a scenario qualifier and typed ports, and the store validates
the chain (unknown op, non-executable op, lane mismatch, bad port, image-after-text)
before it can be saved or compiled. The chain is still linear rather than a DAG,
and cross-scenario steps are declared but rejected for execution. The target for
workflow-level flows (typed ports, scenario-qualified steps, per-owner execution,
cross-scenario composition) is [`COMPOSITION.md`](COMPOSITION.md). Until that
lands, a Look cannot express a non-image step or validate that one step's output
kind matches the next step's input.

### Quality evaluation and promotion

`image-tools quality evaluate` scores an output-set manifest using the
operation-specific reference profile and any configured judge. The manifest
binds candidate/reference files, hashes, profile/rubric version and regression
budgets. A comparison can describe model output, but changing a default requires
a rights-clean model, adequate reference coverage, calibrated budgets and
quality/judge evidence. One synthetic case is not a general operation upgrade.

**The judge.** `judge_model` in the manifest names the judge and binds it: a bare
name (`qwen3-vl:4b`) is a local Ollama model reached through the AI Gateway's
local-only route; `openrouter:google/gemini-3.5-flash` is a hosted judge reached
through a remote-only route that exists only because the manifest named it.
The route evidence must report exactly that provider and model, so a silent
fallback to another model cannot score a case. `judge_model` may be an ordered,
comma-separated chain (a fallback must be in `qualifiedJudges`; today only the hosted judge is): the next
judge is asked only when the previous one is refused or unavailable (hosted
credits exhausted, breaker open, timeout, silent reroute), never because a reply
was poor, and each case's judge result records which judge scored it. Every fallback must be a judge that passed the agreement bar (`qualifiedJudges` in `quality/judge.go`); when none is reachable the case is recorded as `judge_unavailable` with the refusal reasons and cannot pass, rather than being scored by a weaker model. The judge sees a labelled contact
sheet (INPUT, REFERENCE, CANDIDATE) and writes a rationale that lists the source's
distinguishing features before it scores, which is what makes it catch a
different-looking animal.

On the 15 human-graded cases of `guinea-pig-v1` (hat, park, anime, generate) the
hosted `google/gemini-3.5-flash` judge agrees with the human accept/reject call on
11 to 12 of 12 cases (12 of 12 through the gateway) and with 0.79-0.94 of the pairwise rankings across runs (the misses are ties among accepted outputs and
swaps between two low tiers; it never accepted an output the human rejected). The
local `gemma4:12b` judge (accept/reject 10/12, pairwise 0.65, and it missed 2 of 4 human rejects) is not qualified: it accepted the different-animal park case and the SD 1.5 anime. A report scored by an unqualified judge says so in its limits.
The agreement record lives with the corpus, under
`~/.vrooli/research/topics/image-tools-foundation/corpora/guinea-pig-v1/`.

**Perceptual distance.** `reference.lpips` (lower is better) is LPIPS with an
AlexNet backbone, scored at native size in 1024-pixel tiles by the sidecar. It is
computed only when a budget names it, needs equal-sized candidate and reference,
and needs the `lpips-alexnet` support asset installed
(`image-tools models install lpips-alexnet`). PSNR and SSIM reward blur; LPIPS does
not, so it is the metric to use when ranking upscalers.

### Import safety

Model and adapter import separates inspection from registration and acquisition.
The inspected source is resolved to explicit artifact metadata and an operator
must attest commercial rights before a row or weight job is created. Missing or
negative rights attestation fails closed. This protects against accidental
registration of unclear models; it is not an independent legal opinion about
upstream provenance.

## Production Shape

API, CLI and workspace UI submit the same operation contract. The API owns
durable state; clients may disconnect while a job continues.

## Maturity Ladder

Implemented local invariants are distinguished from served showcase receipts.
Each workstream checkpoint tracks what is proven, partial or open.

## Deferred / Unmodeled Flows

Fresh-host all-capability provisioning and served GPU contention remain open.
No flow in this document claims that every operation or platform is qualified.

## Flow Details

The flow inventory links each workflow to its owner and its controlling rule.

## Tests and evidence boundaries

The code's transition and cleanup tests are the behavior gate for local
invariants. `vrooli scenario test image-tools --phases unit` validates the
scenario unit surface; its result is run-specific and does not certify every
operation or platform. The current mission checkpoints record which live
showcases exist and what remains open.

Image Tools does not claim a checked Quint/TLA model for these workflows.
Where formal model checking is required, record the model and replay evidence
beside the flow instead of describing a source transition table as a formal
proof.

## Cross-References

- [`DATA.md`](DATA.md) — persistent state and retention
- [`DOMAINS.md`](DOMAINS.md) — implementation owners
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external service contracts
