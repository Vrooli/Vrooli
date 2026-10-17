# ACE-Step API

The managed service listens on the upstream resource port and exposes:

- `GET /health` — HTTP200 only when both models are loaded and initialization has
  no error; HTTP503 while initializing or degraded. Both responses retain
  readiness metadata, selected variant, planner, device, and
  initialization failure details.
- `GET /v1/variants` — installed/default DiT and planner identities.
- `POST /v1/compose` — one generated WAV. The request accepts `caption`,
  `lyrics`, `duration`, `bpm`, `keyscale`, `seed`, optional inference settings,
  and an explicit `rewrite_caption` opt-in.
- `POST /v1/capacity/degrade?to=full|offload-dit` — diagnostic rung operation.

The resource produces one 48 kHz stereo WAV per request. Batch orchestration,
provenance, and pool state are owned by `music-tools`.

## Generated output identity

A successful `/v1/compose` WAV response includes `X-ACE-Step-Model-ID`,
`X-ACE-Step-License-Lane` and `X-ACE-Step-Profile-Rung`. Model and rung identify
what was loaded, captured when initialization succeeded. Changing the desired
capacity rung does not relabel an already loaded model. `/v1/capacity/degrade`
reports `restart_required` when loaded and requested rungs differ; callers must
refuse that operation or arrange an authorized managed lifecycle change. The
resource does not restart itself or another GPU tenant in this request.

Capacity acknowledgement refuses with 503 while model initialization owns the
load lock. When already loaded, `applied_rung` is the actual loaded rung and
`restart_required` reports a different requested posture. Music supplies
an explicit `admitted_rungs` list on composition; the resource checks actual
loaded identity after initialization and before importing/running inference.
The list accepts one or two known values (`full`, `offload-dit`); an empty list,
unknown label, or malformed value is invalid. A full 7 GiB Music admission
permits either loaded rung; a reduced 6 GiB admission permits only `offload-dit`.
This compatibility is specific to the reviewed two-rung profile and changes no
budget or floor. The actual response header and take/job provenance retain the
loaded rung, even when a larger budget admitted it.

Existing callers may still supply strict `expected_rung`. If both restrictions
are provided, both must match; `admitted_rungs` never weakens the strict option.
Omitting both preserves the existing direct-call behavior. Music records its
budget label as `capacity_admitted_rung`. Successful job `model_id` and
`applied_rung` summarize output only when that field is uniform across all
takes. If either varies, `output_identity: mixed` marks the batch and the
varying summary field is omitted; each take retains its actual model and rung.
Historical job metadata is retained without rewriting it.
An incompatible loaded rung never starts new inference. This is admission
consistency, not GPU tenant remediation.


## Seed and sampling contract

`seed` is a strict unsigned 32-bit integer (0 through 4294967295). An omitted
seed draws one random uint32 when the request is validated; explicit zero stays
zero. The resource disables upstream random seed selection and checks the
resolved output seed before serving audio. This makes the seed an actual output
witness; historical outputs produced by older wrappers have unknown effective
seeds and must not be relabeled.

`expected_variant` optionally requires `acestep-v15-turbo` or `acestep-v15-sft`.
A mismatch with the loaded model refuses inference with HTTP503. This field
checks identity; it does not switch models. Invalid request fields return
HTTP422. Omit optional inference settings to use resource defaults:

| Loaded variant | Steps | Effective guidance | Shift |
| --- | --- | --- | --- |
| turbo | 8 (maximum 20) | 1, including when a caller supplies another CFG value | 3 |
| SFT | 50 (maximum 100) | 7 unless supplied | 1 |

`shift` accepts finite values from 1 through 5. The service retains LM semantic
code generation (`thinking=True`) while caption and metadata rewriting remain
opt-in. These are sampling choices, not a guarantee of musical quality.

Successful audio responses additionally require actual resolved sampling
witnesses: `X-ACE-Step-Seed`, `X-ACE-Step-Planner`,
`X-ACE-Step-Inference-Steps`, `X-ACE-Step-Guidance-Scale`, and
`X-ACE-Step-Shift`. Planner is `acestep-5Hz-lm-0.6B` or
`acestep-5Hz-lm-1.7B`. When upstream returns semantic audio-code tokens, the
paired `X-ACE-Step-LM-Code-SHA256` and `X-ACE-Step-LM-Code-Count` identify that
bounded text without exposing it. Missing paired headers mean no code witness
is available, not that LM generation was disabled. Missing or inconsistent
resolved seed/sampling data refuses the response with HTTP503.

Health exposes `configured_variant`, `configured_planner`, and
`configured_profile` separately from `loaded_variant`, `loaded_planner`, and
`loaded_profile`. Loaded values remain tied to successful initialization.


The single-item pinned PyTorch LM route ignores upstream's seeds argument.
The resource therefore resets the actual PyTorch CPU/CUDA RNG to the effective
uint32 seed immediately before generation. One process-local generation lock
protects that reset and the shared model generation; a concurrent compose
request receives HTTP503 `generation is busy` without reseeding or queuing.
Errors release the lock. Health and capacity observations remain available.
This controls planner sampling RNG; it does not promise bit-identical outputs
across accelerator kernels, versions or platforms. A returned per-audio seed
alone does not prove equal semantic conditioning: compare the actual code
hashes when interpreting a controlled experiment. Historical1.1.4 results lack
this request-boundary planner RNG reset and remain preserved as such.
