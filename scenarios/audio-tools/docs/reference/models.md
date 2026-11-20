# Model and voice registry

<!-- Generated from api/internal/modelcatalog/models.seed.json. Do not edit by hand.
     Regenerate from api/ with: go run ./internal/modelcatalog/cmd/render-models -->

Generated 2026-11-19 from `api/internal/modelcatalog/models.seed.json` (schema `1.0.0`, 30 records).

Model choice is data plus evidence. This is the audio-tools analogue of image-tools' `models/registry.seed.json`: one record per model or voice family, carrying stable provider ids, STT engine ids, licence and commercial eligibility, readiness, the hardware envelope, and benchmark evidence. Consumers ask `api/internal/modelcatalog` instead of hardcoding engine ids.

The registry is a runtime surface, not only documentation: `ModelRegistryService.ListModels` returns these records and `ModelRegistryService.ResolveModel` applies the selection policy for a capability and language. The CLI exposes them as `audio-tools model list` / `audio-tools model resolve`, and the UI renders them on the Models page. Edit the seed, regenerate, and every surface moves together.

## How selection works

Ranking is deterministic:

1. readiness (`available` > `candidate` > `experimental` > `watch`)
2. role signature (`default` > `fast` > `quality` > `edge`)
3. commercial clarity (`yes` > `unknown` > `no`)
4. model id ascending

Only `available` records are default-eligible. A default candidate must also carry the `default` role; a merely available record is an alternative, not a default. A `candidate`, `experimental`, or `watch` record may carry the `default` role only with `allow_default: true`, and availability still ranks first among default candidates. `CommercialDefaultFor` fails closed: it returns only records whose `commercial_use` is explicitly `yes`.

In the tables, `Default` is `yes` for an available promoted default, `opt-in` for a non-available record that carries the default role with `allow_default`, and `—` for a record that is a valid alternative but not a promoted default.

Entry points: `All`, `ByCapability`, `ByProvider`, `ByEngine`, `CandidatesFor`, `CandidatesForEngine`, `DefaultFor`, `DefaultForEngine`, `CommercialDefaultFor`. `DefaultFor(capability, language, platform)` matches language by primary subtag (`mul` matches every language), filters by declared os/arch, and admits a record only when its CPU/GPU/RAM/VRAM envelope fits the supplied platform. Unknown platform facts are not treated as a shortfall; a known shortfall is.

## Speech to text (stt)

| Model | Provider ids | Engine ids | Runtime | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Default role | Roles |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `whisper-medium` | whisper-stt | whisper-local | whisper.cpp | `whisper/medium-model` | 769M | 1530 MB | en, mul | MIT | yes | available | — | quality |
| `whisper-large-v3` | whisper-stt | whisper-local | whisper.cpp | — | 1550M | 3100 MB | en, mul | MIT | yes | candidate | — | quality |
| `whisper-large-v3-turbo` | whisper-stt | whisper-local | whisper.cpp | `whisper/turbo-model` | 809M | 1620 MB | en, mul | MIT | yes | available | — | quality |
| `whisper-small` | whisper-stt | whisper-local | whisper.cpp | `whisper/small-model` | 244M | 488 MB | en, mul | MIT | yes | available | yes | default |
| `whisper-base` | whisper-stt | whisper-local | whisper.cpp | `whisper/base-model` | 74M | 148 MB | en, mul | MIT | yes | available | — | fast, edge |
| `whisper-tiny` | whisper-stt | whisper-local | whisper.cpp | `whisper/tiny-model` | 39M | 78 MB | en, mul | MIT | yes | available | — | edge |
| `kyutai-stt-1b-en-fr` | kyutai-stt | kyutai | kyutai | `kyutai-stt/moshi-weights` | 1B | 2000 MB | en, fr | CC-BY-4.0 | yes | available | — | fast |
| `kyutai-stt-2.6b-en` | kyutai-stt | kyutai | kyutai | — | 2.6B | 5200 MB | en | CC-BY-4.0 | yes | candidate | — | quality |
| `sherpa-zipformer-en-2023-06-26` | sherpa-stt | sherpa-streaming | sherpa-onnx | `sherpa-onnx/streaming-zipformer-en-2023-06-26` | unknown | 310 MB | en | Apache-2.0 | yes | available | — | fast, edge |
| `sherpa-zipformer-en-kroko-2025-08-06` | sherpa-stt | sherpa-streaming | sherpa-onnx | `sherpa-onnx/streaming-zipformer-en-kroko-2025-08-06` | unknown | 68 MB | en | CC-BY-SA-4.0 (community model) | unknown | available | — | edge |
| `rumble-2-flash` | rumble-stt | — | vllm | — | 0.5B | 1000 MB | en, mul | Apache-2.0 | yes | candidate | opt-in | default, quality |
| `openai-whisper-1` | openai-whisper | — | byok | — | vendor-managed | — | en, mul | Proprietary service terms | yes | available | — | quality |
| `deepgram-nova-3` | deepgram | — | byok | — | vendor-managed | — | en, mul | Proprietary service terms | yes | available | — | fast |

## Text to speech (tts)

| Model | Provider ids | Engine ids | Runtime | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Default role | Roles |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `kokoro-82m-v1.0` | kokoro-tts | — | kokoro | `kokoro/kokoro-model` | 82M | 330 MB | en, es, fr, hi, it, ja, pt, zh | Apache-2.0 | yes | available | yes | default |
| `kale-tts-1-nano` | kale-tts | — | onnxruntime | — | 7.5M | 14.8 MB | en | Apache-2.0 | yes | candidate | — | fast, edge |
| `kale-tts-1-tiny` | kale-tts | — | onnxruntime | — | 45.7M | 79.9 MB | en | Apache-2.0 | yes | candidate | — | fast, edge |
| `kale-tts-1-base` | kale-tts | — | pytorch | — | 81.8M | — | en | Apache-2.0 | yes | experimental | — | quality |
| `reverb-tts` | reverb-tts | — | pytorch | — | 297M | 307 MB | en, de, es, fr, it, pl, pt | CC-BY-NC-4.0 (weights); Apache-2.0 (code) | no | experimental | — | quality |
| `verba-tts-indic-zero` | verba-tts | — | browser | — | 1.8M | 7.6 MB | en, hi, bn, ta, te, mr, gu, kn, ml, pa, or, as | unknown | unknown | watch | — | edge |
| `openai-tts-1` | openai-tts | — | byok | — | vendor-managed | — | en, mul | Proprietary service terms | yes | available | — | quality |
| `elevenlabs-multilingual-v2` | elevenlabs | — | byok | — | vendor-managed | — | mul | Proprietary service terms | yes | available | — | quality |

## Summarization (summarize)

| Model | Provider ids | Engine ids | Runtime | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Default role | Roles |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `qwen3.5:9b` | ollama | — | ollama | — | 9B | 6590 MB | mul | Unverified upstream (Qwen3.5) | unknown | available | — | quality |
| `qwen3:1.7b` | ollama | — | ollama | — | 1.7B | 1360 MB | mul | Apache-2.0 | yes | available | yes | default, fast |
| `qwen3.5:4b` | ollama | — | ollama | — | 4B | 3390 MB | mul | Unverified upstream (Qwen3.5) | unknown | available | — | quality |
| `llama3.2:3b` | ollama | — | ollama | — | 3B | 2020 MB | mul | Llama 3.2 Community License | yes | available | yes | default, quality |
| `llama3.2:1b` | ollama | — | ollama | — | 1B | 1320 MB | mul | Llama 3.2 Community License | yes | available | — | fast, edge |
| `gemma3:4b` | ollama | — | ollama | — | 4B | 3300 MB | mul | Gemma Terms of Use | yes | candidate | — | quality |
| `phi4-mini:3.8b` | ollama | — | ollama | — | 3.8B | 2500 MB | mul | MIT | yes | candidate | — | fast |
| `openrouter-summarize` | openrouter | — | gateway | — | provider-selected | — | mul | Proprietary provider terms | yes | available | — | quality |

## Speaker (speaker)

| Model | Provider ids | Engine ids | Runtime | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Default role | Roles |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `sherpa-onnx-speaker-embedding` | speaker-verification | — | sherpa-onnx | `sherpa-onnx/speaker-eres2net-large-zh-cn-16k` | unknown | 27 MB | mul | Apache-2.0 | yes | available | yes | default |

## Declared defaults

Computed with an unconstrained platform (`DefaultFor(cap, "", Platform{})`), so GPU-required records are excluded. The commercial column applies the fail-closed filter.

| Capability | Default | Commercial default |
|---|---|---|
| stt | `whisper-small` | `whisper-small` |
| tts | `kokoro-82m-v1.0` | `kokoro-82m-v1.0` |
| summarize | `qwen3:1.7b` | `qwen3:1.7b` |
| speaker | `sherpa-onnx-speaker-embedding` | `sherpa-onnx-speaker-embedding` |

## Resource footprint

Default selection is also a resource decision. Each capability's promoted default must fit its declared budget (`DefaultFootprintBudgetMB`); a seed edit that promotes a larger model fails the budget gate instead of silently reserving more memory. Budgets: stt 600 MB, tts 400 MB, summarize 4000 MB, speaker 100 MB.

| Capability | Default | Runtime | Declared size | Budget | Artifact |
|---|---|---|---|---|---|
| stt | `whisper-small` | whisper.cpp | 488 MB | 600 MB | `whisper/small-model` |
| tts | `kokoro-82m-v1.0` | kokoro | 330 MB | 400 MB | `kokoro/kokoro-model` |
| summarize | `qwen3:1.7b` | ollama | 1360 MB | 4000 MB | — |
| speaker | `sherpa-onnx-speaker-embedding` | sherpa-onnx | 27 MB | 100 MB | `sherpa-onnx/speaker-eres2net-large-zh-cn-16k` |

Promoted defaults (the weight disk a fresh install of the referenced resources needs): **2205 MB**. Every available local record declared in this registry totals **21279 MB**; that is a catalogue total, not what an install pulls: a resource installs its eager artifact and leaves on-demand tiers unpulled. Remote (`byok`/`gateway`) and `browser` records carry no local weight footprint.

## Cautions

Records that are not both available and commercially clear. A caution is not a selection ban by itself: the readiness and commercial filters already enforce the hard rules.

- `whisper-large-v3` — readiness `candidate`, commercial `yes`. ~3.1 GB ggml artifact. The 6 GB VRAM figure is an operational comfort estimate, not a measured floor; CPU inference works but is slow.
- `kyutai-stt-2.6b-en` — readiness `candidate`, commercial `yes`. Candidate quality tier. The 16 GB VRAM figure is an estimate for the 2.6B checkpoint plus decoder overhead and is not measured on this host.
- `sherpa-zipformer-en-kroko-2025-08-06` — readiness `available`, commercial `unknown`. The Kroko community model is CC-BY-SA-4.0: commercial use is permitted but derivative distributions carry share-alike and attribution obligations. Kroko sells a separate commercial/OEM licence. Treat a commercial deployment as requiring a licence review until the operator confirms one.
- `rumble-2-flash` — readiness `candidate`, commercial `yes`. Batch ASR, not native streaming. Candidate default for local batch/multilingual work; allow_default is explicit so promotion is a seed decision backed by a receipt. engine_ids is empty until a real rumble engine/adapter exists in sttengine and the provider catalog. The vLLM adapter is GPU-oriented in practice, so CPU viability is not claimed even though the weights fit in CPU RAM. Artifact size ~1 GB is an estimate from 0.5B parameters.
- `kale-tts-1-nano` — readiness `candidate`, commercial `yes`. Candidate fast English CPU/on-device TTS. The seed marks it candidate until the private-draft release is approved for the scenario; 14.8 MB is the int8 CPU artifact. Planned host: the sherpa-onnx Kokoro-format ONNX adapter (SHERPA_ONNX_MODEL_DIR); no artifact is declared until Kale publishes weights and a checksum.
- `kale-tts-1-tiny` — readiness `candidate`, commercial `yes`. Candidate expressive fast English TTS; 79.9 MB is the int8 CPU artifact. Same approval gate as Nano. Planned host: the sherpa-onnx Kokoro-format ONNX adapter (SHERPA_ONNX_MODEL_DIR); no artifact is declared until Kale publishes weights and a checksum.
- `kale-tts-1-base` — readiness `experimental`, commercial `yes`. Experimental: the PyTorch path is a scratch/reference implementation and artifact size is not published. Not a default.
- `reverb-tts` — readiness `experimental`, commercial `no`. Weights are CC BY-NC 4.0: non-commercial use only, with attribution. Commercial use requires a negotiated licence from Software Mansion. Runtime code is Apache-2.0.
- `verba-tts-indic-zero` — readiness `watch`, commercial `unknown`. No confirmed licence or public release was found. Treat every use as unlicensed until the owner publishes terms.
- `qwen3.5:9b` — readiness `available`, commercial `unknown`. Quality tier. resource-ollama's summarize.default role resolves to qwen3:1.7b; the 4B tag is the first quality fallback. The upstream Qwen3.5 model card was not verified in-repo, so commercial use is unknown and this tag can never be a commercial default.
- `qwen3.5:4b` — readiness `available`, commercial `unknown`. Quality escalation and first summarize.default fallback. The upstream model card was not available in-repo when this record was written, so commercial use is unknown until an operator verifies it. Unknown never becomes a commercial default.
- `gemma3:4b` — readiness `candidate`, commercial `yes`. 3.3 GB is the quantized pull size. Do not pull without operator approval.
- `phi4-mini:3.8b` — readiness `candidate`, commercial `yes`. 2.5 GB is the quantized pull size. Lower-latency fallback role candidate.

## Provider and engine vocabulary

Provider ids are the stable health/lifecycle ids owned by `internal/providercatalog` (exposed through `internal/capabilities`). Engine ids are STT stream-selection ids owned by `internal/sttengine`; non-STT records, BYOK/gateway/browser providers, and batch adapters with no engine leave `engine_ids` empty. An engine id that no provider owns is drift: the cross-catalog conformance test rejects it.

Provider ids: `whisper-stt`, `kyutai-stt`, `sherpa-stt`, `kokoro-tts`, `ollama`, `openai-whisper`, `deepgram`, `openai-tts`, `elevenlabs`, `openrouter`, `browser-stt`, `browser-tts`, `rumble-stt`, `kale-tts`, `reverb-tts`, `verba-tts`, `speaker-verification`.

Engine ids: `whisper-local`, `kyutai`, `sherpa-streaming`.
