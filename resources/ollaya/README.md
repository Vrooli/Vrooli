# Ollaya Resource

Decision models for Vrooli. A **decision model** is non-generative: given a
`state` (text or JSON) and typed questions, it returns one answer per question
from a closed set, with calibrated probabilities, in one forward pass. It never
writes prose, so the answer is always a value code can branch on.

This resource does two things:

1. **Runs the [Ollaya](https://github.com/ollaya-dev/ollaya) daemon** ("Ollama
   for decision models", Apache-2.0) as a managed service on `127.0.0.1:11435`.
   The artifact is the checksum-pinned v0.18.0 release: on NVIDIA hosts
   (compute ≥ 7.5, driver R580+) the base archive is composed with the CUDA 13
   library pack (~1.5 GB); elsewhere the 23 MB CPU archive is staged.
2. **Owns the System One wire client** for every decision upstream: the local
   daemon (native `/api/decide`) and the hosted, credential-gated TypeSafe
   format endpoints — TypeSafe Jev, Jev through Eden AI, and Cloudflare Clef.
   Credentials stay here; ai-gateway only ever sees a role name and a typed
   failure code.

ai-gateway is the only intended caller. Scenarios use
`ai-gateway inference decide`, never this CLI.

## Question types

| Type | Criteria | Answer |
|---|---|---|
| `choice` | 2–255 labels, each with a description | `choice`, `confidence`, `probabilities` per label |
| `score` | 2–10 ordered level descriptions, level 0 first | fractional `score` (Σ i·pᵢ), `confidence`, `probabilities` per level |
| `noul` | optional `true` / `false` descriptions | `noul`: P(statement holds) |

`confidence` is TypeSafe's normalized top probability, (K·p_max − 1)/(K − 1).
Thresholds do not transfer across models; tune per model.

## Lifecycle

```bash
vrooli resource install ollaya         # stage the pinned artifact + CLI
vrooli resource start ollaya
resource-ollaya models pull            # laya router + laya:en + laya:multilingual, by digest
resource-ollaya gateway health
vrooli resource status ollaya
vrooli resource stop ollaya
```

Models live in `${RESOURCE_DATA_DIR}/models` (relocatable, regenerable cache).
Every model in `model-policy.json` is pinned by its registry manifest digest;
`models pull` pulls `name@sha256:<digest>` and fails if the daemon then holds a
different digest. The daemon itself verifies each weight blob's sha256 and
fetches weights from the model author's Hugging Face repository. Before any
missing model is pulled, the CLI reserves the estimated peak on the managed
model-store path through the shared capacity disk-job owner. It samples growth
while the pull runs and completes the job only after the daemon reports the
pinned digests installed. JSON output includes the durable disk-job receipt and
verified model digests. If a pull or verification fails, partial files are
preserved and the admission stays active for capacity-owner recovery because
the daemon API does not expose a cleanup proof.

For missing-model pulls, the CLI holds Ollaya's managed-service lifecycle lock
for the full operation. It verifies the owner record's PID, process incarnation
and ownership token, checks the service process's `OLLAYA_MODELS` and loopback
listener against the admitted path, and rejects an `OLLAYA_URL` that does not
match the managed resource endpoint. Pull and inventory requests then use the
verified private listener directly.

### Disk retention

Storage Manager decides when downloaded models leave disk (free-space bands,
the 20 GiB store budget in `resource.json`, real use); this resource owns the
deletion.

```bash
resource-ollaya models retention --json
resource-ollaya models evict <name> --digest <sha256> --size <bytes> --json
```

Ollaya models are regenerable: every one is digest-pinned in
`model-policy.json`, and `resource-ollaya models pull --model <name>` restores
the exact digest. Default pulls and route targets are required and never
eligible; a model whose installed digest differs from its pin, or that has no
pin, is refused because it could not be restored exactly. `models evict` holds
the managed lifecycle lock and the per-model request fence that the activity
edge holds for every request, re-checks digest, size, residency and the policy
under them, and appends the reacquire command and expected size to
`~/.vrooli/state/resources/ollaya/model-evictions.jsonl` before deleting.
Request times are recorded in `~/.vrooli/state/resources/ollaya/model-usage.json`.

## Roles

`resource-ollaya policy roles --json` is the contract ai-gateway reads. Every
local role carries the served model's `limits` (`context_tokens`, `max_options`,
`score_levels` when the model serves exactly that many, `max_questions` when it
bounds a request below the daemon's cap) and `placement` (`device` `gpu`, `cpu`
or `any`; `vram_bytes`, `ram_bytes`, `keep_alive`), so callers size digests and
batches from data. A router (`laya`) reports what all its targets honour.

| Role | Upstream | Model | Needs |
|---|---|---|---|
| `decide.local` | local daemon | `laya` (router → `laya:en` / `laya:multilingual`) | `models pull` |
| `decide.local.accurate` | local daemon | `winnow:e4b` | `models pull --model winnow:e4b` (8 GB; ~9 GB VRAM free to run on GPU) |
| `decide.jev.edenai` | Eden AI `/v3/alpha/decisions` | `typesafe/jev-latest` | `vrooli/edenai` api-key |
| `decide.jev.typesafe` | TypeSafe `/v1/systemone` | `jev-latest` | `vrooli/typesafe` api-key (waitlisted) |
| `decide.clef.cloudflare` | Workers AI REST | `clef-flash` | `vrooli/cloudflare-workers-ai` api-token + account-id |

### Bake-off candidates

These local roles exist to be measured, never for production routing: ai-gateway
reaches them only through `decide.candidates` with `pin_resource_role`. All are
`default_pull: false`; pull one with `resource-ollaya models pull --model <name>`.
Rights are classified only from a fetched license text (URL, sha256,
`observed_at`); a model whose license evidence is incomplete stays an explicit
`unknown` with the reason, and the gateway refuses it (`policy resolve` still
works, so it can be sized, but no decision is dispatched).

| Role | Model | Rights |
|---|---|---|
| `decide.local.typed` | `laya:typed-decisions` | Apache-2.0, classified |
| `decide.local.coba-400m`, `decide.local.coba-1b` | `coba:400m`, `coba:1b` | Apache-2.0, classified |
| `decide.local.largha` | `largha:150m` | Apache-2.0, classified |
| `decide.local.decima-clean` | `decima:base-clean` | Apache-2.0, trained without non-commercial data, classified |
| `decide.local.pplx` | `pplx-decider:flash` | Apache-2.0, classified; GPU only |
| `decide.local.kev` | `kev:4b` | Apache-2.0, classified; GPU only |
| `decide.local.decima-agent`, `decide.local.decima-small` | `decima:agent`, `decima:small` | unknown: training data includes non-commercial sets |
| `decide.local.nli-modernbert`, `decide.local.nli-deberta` | `nli:modernbert-large`, `nli:deberta-v3-large` | unknown: the author's card says non-commercial training data is included |
| `decide.local.credence` | `credence:e4b-calibrated` | unknown: training-data license manifest unreviewed; GPU only |

Excluded by license: StartLux and `nimble:9b-v3` (CC BY-NC), `d1` and `d1-omni`
(LFM Open License, revenue-capped).

`policy resolve --role <id>` succeeds only when the role can serve: hosted
roles need their credentials in the process environment, local roles need the
daemon reachable with every model (and router target) pulled. Otherwise it
exits non-zero with a `VROOLI_PROVIDER_ERROR {"code":"unconfigured",…}` stderr
line, which ai-gateway records as a skipped candidate.

Provision a hosted key through the credential authority, never a file:

```bash
vrooli credentials provision --identity vrooli/edenai --field api-key
```

## Gateway command

```bash
echo '{"state":"Stop, that is not what I asked for.",
       "questions":{"kind":{"type":"choice","criteria":{"correction":"Corrects the agent","praise":"Approves","other":"Anything else"}},
                    "stop":{"type":"noul","instructions":"The operator wants the agent to stop."}}}' \
  | resource-ollaya gateway decide --role decide.local --input-stdin --json
```

Output is normalized across upstreams: `{upstream, locality, model, answers,
usage, cost_usd, latency_ms, eval_ms, load_ms}`. A decision on truncated state is
refused (`context_overflow`), on the native route as on the hosted ones.

Failure codes: `unconfigured` (missing key, bad key, model not pulled),
`unreachable`, `rate_limited` (with the observed `Retry-After`),
`provider_overloaded`, `context_overflow`, `provider_failed`.

## GPU and capacity

The daemon loads models on demand and unloads them after `OLLAYA_KEEP_ALIVE`
(30 m). The resource holds a VRAM claim through the shared capacity companion
(`capacity-sync`) and degrades to CPU (`capacity degrade --to cpu`, which
restarts with `OLLAYA_DEVICE=cpu`) when the broker asks. Laya runs well on the
CPU; Winnow does not.

### Fit before load

A model's declared `placement` is checked against live capacity before it is
asked to load, both when its role resolves (`policy resolve`, side-effect free)
and again at dispatch (`gateway decide`). A model that cannot fit is refused
with a typed `insufficient_vram` or `insufficient_memory`, and ai-gateway
continues down its ladder without paying a load:

- a resident model already holds its memory, so nothing is asked of the broker;
- otherwise the shared broker (`vrooli capacity admit`, about 0.1 s) is asked
  for the declared VRAM (accelerator daemon) or RAM (CPU daemon). The broker
  excludes Ollaya's own claim, so at dispatch the ask is the footprint plus
  every model already resident in the daemon, and idle residents are unloaded
  largest first until it fits; a role resolution only asks whether the model
  would fit if Ollaya released everything, and never unloads;
- a model that is not resident is loaded inside a per-daemon load slot (a
  one-slot host semaphore, `OLLAYA_LOAD_LOCK_DIR`, default under
  `$VROOLI_RUNTIME_DIR/resources/ollaya/load`) that spans admission and the
  decision call. The broker answers from memory that is free now, and a load in
  flight has not taken it yet, so without the slot two cold callers are both
  told the same memory is free. The second waits, then finds the model resident
  or asks the broker with the first load visible; a resident model takes no
  slot and a role resolution never takes one. A wait that outlasts the
  caller's timeout is a retryable `provider_overloaded`;
- `vrooli capacity admit` never counts another owner's reclaimable claim (it
  runs with `NoReclaim`), so a yes means no one has to give memory back and a
  no does not mean an idle owner could not. Making room from an idle owner is
  the broker's reclaim path, not something Ollaya triggers;
- the daemon's pinned device (`OLLAYA_DEVICE`) is read from its owner process,
  not from a status audit, which costs seconds;
- a `placement.keep_alive` (for example `2m` on the large bake-off models) makes
  the daemon release a model's memory soon after its last request instead of
  holding it for the daemon-wide default;
- a daemon-side load failure that says it ran out of memory is classified the
  same way, so a footprint that was declared too low still ends in a typed skip.

Declared footprints are measured on the reference host (see the bake-off notes
in the Phase 2 work folder), not guessed: an understated `vram_bytes` makes the
broker admit a load that then fails.

When the CUDA backend is selected the daemon is pinned with
`OLLAYA_DEVICE=cuda`. A Winnow load that cannot allocate its VRAM now fails with
`MODEL_LOAD_FAILED` / `insufficient_vram` instead of silently falling back to a
4–55 s CPU load. `model-policy.json` declares Winnow's `placement.device: gpu`
and its 9 GiB footprint; before a local GPU-only model is dispatched,
`gateway decide` checks the daemon's observed device and asks the shared broker
(`vrooli capacity admit`) whether the declared footprint fits. A refusal is a
typed `insufficient_vram` so ai-gateway falls back to `decide.fast` (Laya)
without paying the CPU load.

The companion never releases a claim just because a CPU-parked daemon holds no
VRAM: the degraded claim records the operating mode, so it is heartbeated until
the broker's upshift pass returns the resource to the accelerator.

The required `activity-edge` companion owns the public loopback port `11435`.
The daemon listens on the private upstream port `21435`. All local decision
clients use the unchanged exported `OLLAYA_URL`; the edge forwards request and
response bodies unchanged. It reports active while `POST /api/decide` requests
are in flight and idle after the shared debounce. Health, model inventory and
resident-model reads do not count as decisions. Hosted upstream calls bypass
this local edge and do not mark the local GPU busy.

Activity reporting is fail-open: a capacity outage does not block decisions.
The edge is a required serving component, so a healthy private daemon alone
does not prove public availability. Its parent is the managed daemon; normal
resource lifecycle owns both. Port overrides use
`OLLAYA_ACTIVITY_EDGE_LISTEN` and `OLLAYA_ACTIVITY_EDGE_UPSTREAM` together with
the corresponding resource endpoint configuration.
