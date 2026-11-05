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
fetches weights from the model author's Hugging Face repository.

## Roles

`resource-ollaya policy roles --json` is the contract ai-gateway reads.

| Role | Upstream | Model | Needs |
|---|---|---|---|
| `decide.local` | local daemon | `laya` (router → `laya:en` / `laya:multilingual`) | `models pull` |
| `decide.local.accurate` | local daemon | `winnow:e4b` | `models pull --model winnow:e4b` (8 GB; ~9 GB VRAM free to run on GPU) |
| `decide.jev.edenai` | Eden AI `/v3/alpha/decisions` | `typesafe/jev-latest` | `vrooli/edenai` api-key |
| `decide.jev.typesafe` | TypeSafe `/v1/systemone` | `jev-latest` | `vrooli/typesafe` api-key (waitlisted) |
| `decide.clef.cloudflare` | Workers AI REST | `clef-flash` | `vrooli/cloudflare-workers-ai` api-token + account-id |

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
