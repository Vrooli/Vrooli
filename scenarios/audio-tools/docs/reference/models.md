# Model and voice registry

<!-- Generated from api/internal/modelcatalog/models.seed.json. Do not edit by hand.
     Regenerate from api/ with: go run ./internal/modelcatalog/cmd/render-models -->

Generated 2026-11-20 from `api/internal/modelcatalog/models.seed.json` (schema `1.0.0`, 35 records).

Model choice is data plus evidence. This is the audio-tools analogue of image-tools' `models/registry.seed.json`: one record per model or voice family, carrying stable provider ids, STT engine ids, licence and commercial eligibility, readiness, the hardware envelope, and benchmark evidence. Consumers ask `api/internal/modelcatalog` instead of hardcoding engine ids.

The registry is a runtime surface, not only documentation: `ModelRegistryService.ListModels` returns these records and `ModelRegistryService.ResolveModel` applies the selection policy for a capability and language. The CLI exposes them as `audio-tools model list` / `audio-tools model resolve`, and the UI renders them on the Models page. Edit the seed, regenerate, and every surface moves together.

## How selection works

Ranking is deterministic:

1. readiness (`available` > `candidate` > `experimental` > `watch`)
2. role signature (`default` > `fast` > `quality` > `edge`)
3. commercial clarity (`yes` > `unknown` > `no`)
4. model id ascending

Only `available` records are default-eligible. A default candidate must also carry the `default` role; a merely available record is an alternative, not a default. A `candidate`, `experimental`, or `watch` record may carry the `default` role only with `allow_default: true`, and availability still ranks first among default candidates. `CommercialDefaultFor` fails closed: it returns only records whose `commercial_use` is explicitly `yes`.

An STT record may additionally declare `engine_defaults` for an engine-specific model choice. `DefaultForEngine` and `model resolve --engine` honor that declaration without changing the generic capability default; these routes use the same evidence gate and commercial filter.

In the tables, `Default` is `yes` for an available promoted default, `opt-in` for a non-available record that carries the default role with `allow_default`, and `—` for a record that is a valid alternative but not a promoted default.

Entry points: `All`, `ByCapability`, `ByProvider`, `ByEngine`, `CandidatesFor`, `CandidatesForEngine`, `DefaultFor`, `DefaultForEngine`, `CommercialDefaultFor`. `DefaultFor(capability, language, platform)` matches language by primary subtag (`mul` matches every language), filters by declared os/arch, and admits a record only when its CPU/GPU/RAM/VRAM envelope fits the supplied platform. Unknown platform facts are not treated as a shortfall; a known shortfall is.

## Speech to text (stt)

| Model | Provider ids | Engine ids | Engine defaults | Runtime | Runtime model id | Batch adapter | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Pricing | Default role | Roles | Promotion evidence |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `whisper-medium` | whisper-stt | whisper-local | — | whisper.cpp |  |  | `whisper/medium-model` | 769M | 1530 MB | en, mul | MIT | yes | available | — | — | quality | not promoted |
| `whisper-large-v3` | whisper-stt | whisper-local | — | whisper.cpp |  |  | — | 1550M | 3100 MB | en, mul | MIT | yes | candidate | — | — | quality | not promoted |
| `whisper-large-v3-turbo` | whisper-stt | whisper-local | — | whisper.cpp |  |  | `whisper/turbo-model` | 809M | 1620 MB | en, mul | MIT | yes | available | — | — | quality | not promoted |
| `whisper-small` | whisper-stt | whisper-local | — | whisper.cpp |  |  | `whisper/small-model` | 244M | 488 MB | en, mul | MIT | yes | available | — | yes | default | legacy exception |
| `whisper-base` | whisper-stt | whisper-local | — | whisper.cpp |  |  | `whisper/base-model` | 74M | 148 MB | en, mul | MIT | yes | available | — | — | fast, edge | not promoted |
| `whisper-tiny` | whisper-stt | whisper-local | — | whisper.cpp |  |  | `whisper/tiny-model` | 39M | 78 MB | en, mul | MIT | yes | available | — | — | edge | not promoted |
| `kyutai-stt-1b-en-fr` | kyutai-stt | kyutai | — | kyutai |  |  | `kyutai-stt/moshi-weights` | 1B | 2000 MB | en, fr | CC-BY-4.0 | yes | available | — | — | fast | not promoted |
| `kyutai-stt-2.6b-en` | kyutai-stt | kyutai | — | kyutai |  |  | — | 2.6B | 5200 MB | en | CC-BY-4.0 | yes | candidate | — | — | quality | not promoted |
| `sherpa-zipformer-en-2023-06-26` | sherpa-stt | sherpa-streaming | sherpa-streaming | sherpa-onnx | sherpa-onnx/streaming-zipformer-en-2023-06-26 |  | `sherpa-onnx/streaming-zipformer-en-2023-06-26` | unknown | 310 MB | en | Apache-2.0 | yes | available | — | — | fast, edge | legacy exception |
| `sherpa-zipformer-en-20m-2023-02-17` | sherpa-stt | — | — | sherpa-onnx | sherpa-onnx/streaming-zipformer-en-20M-2023-02-17 |  | — | 20M | 44 MB | en | Apache-2.0 | yes | candidate | — | — | fast, edge | not promoted |
| `sherpa-zipformer-en-kroko-2025-08-06` | sherpa-stt | sherpa-streaming | — | sherpa-onnx |  |  | `sherpa-onnx/streaming-zipformer-en-kroko-2025-08-06` | unknown | 68 MB | en | CC-BY-SA-4.0 (community model) | unknown | available | — | — | edge | not promoted |
| `rumble-2-flash` | rumble-stt | — | — | vllm | Reson8/Rumble-2-Flash | vllm-openai-chat-audio | — | 0.5B | 1390 MB | en, mul | Apache-2.0 | yes | candidate | — | — | quality | not promoted |
| `openai-whisper-1` | openai-whisper | — | — | byok |  |  | — | vendor-managed | — | en, mul | Proprietary service terms | yes | available | — | — | quality | not promoted |
| `deepgram-nova-3` | deepgram | — | — | byok |  |  | — | vendor-managed | — | en, mul | Proprietary service terms | yes | available | — | — | fast | not promoted |

## Text to speech (tts)

| Model | Provider ids | Engine ids | Engine defaults | Runtime | Runtime model id | Batch adapter | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Pricing | Default role | Roles | Promotion evidence |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `kokoro-82m-v1.0` | kokoro-tts | — | — | kokoro | kokoro |  | `kokoro/kokoro-model` | 82M | 330 MB | en, es, fr, hi, it, ja, pt, zh | Apache-2.0 | yes | available | — | yes | default | legacy exception |
| `kokoro-82m-v1.0-sherpa-int8` | kokoro-tts | — | — | sherpa-onnx | kokoro |  | `sherpa-onnx/kokoro-multi-lang-v1-0` | 82M | — | en, es, fr, hi, it, ja, pt, zh | Apache-2.0 | yes | candidate | — | — | fast | not promoted |
| `kale-tts-1-nano` | kale-tts | — | — | onnxruntime | kale-tts-1-nano |  | `kale-tts/nano-int8-onnx` | 7.5M | 14.8 MB | en | Apache-2.0 | yes | candidate | — | — | fast, edge | not promoted |
| `kale-tts-1-tiny` | kale-tts | — | — | onnxruntime |  |  | — | 45.7M | 79.9 MB | en | Apache-2.0 | yes | candidate | — | — | fast, edge | not promoted |
| `kale-tts-1-base` | kale-tts | — | — | pytorch |  |  | — | 81.8M | — | en | Apache-2.0 | yes | experimental | — | — | quality | not promoted |
| `qwen3-tts-12hz-0.6b-base` | qwen3-tts | — | — | pytorch | Qwen/Qwen3-TTS-12Hz-0.6B-Base |  | — | 0.6B | 2520 MB | zh, en, ja, ko, de, fr, ru, pt, es, it | Apache-2.0 | yes | candidate | — | — | edge | not promoted |
| `reverb-tts` | reverb-tts | — | — | pytorch |  |  | — | 297M | 307 MB | en, de, es, fr, it, pl, pt | CC-BY-NC-4.0 (weights); Apache-2.0 (code) | no | experimental | — | — | quality | not promoted |
| `sopro-v2-turbo-voice-cloning` | sopro-tts | — | — | pytorch | samuel-vitorino/sopro-v2-turbo |  | — | 120M | 952 MB | en, pt, fr, de | Apache-2.0 | yes | candidate | — | — | edge | not promoted |
| `verba-tts-indic-zero` | verba-tts | — | — | browser |  |  | — | 1.8M | 7.6 MB | en, hi, bn, ta, te, mr, gu, kn, ml, pa, or, as | unknown | unknown | watch | — | — | edge | not promoted |
| `openai-tts-1` | openai-tts | — | — | byok |  |  | — | vendor-managed | — | en, mul | Proprietary service terms | yes | available | — | — | quality | not promoted |
| `elevenlabs-multilingual-v2` | elevenlabs | — | — | byok |  |  | — | vendor-managed | — | mul | Proprietary service terms | yes | available | — | — | quality | not promoted |

## Summarization (summarize)

| Model | Provider ids | Engine ids | Engine defaults | Runtime | Runtime model id | Batch adapter | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Pricing | Default role | Roles | Promotion evidence |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `qwen3.5:9b` | ollama | — | — | ollama |  |  | — | 9B | 6590 MB | mul | Apache-2.0 | yes | available | — | — | quality | not promoted |
| `qwen3:1.7b` | ollama | — | — | ollama |  |  | — | 1.7B | 1360 MB | mul | Apache-2.0 | yes | available | — | yes | default, fast | legacy exception |
| `qwen3.5:4b` | ollama | — | — | ollama |  |  | — | 4B | 3390 MB | mul | Apache-2.0 | yes | available | — | — | quality | not promoted |
| `llama3.2:3b` | ollama | — | — | ollama |  |  | — | 3B | 2020 MB | mul | Llama 3.2 Community License | yes | available | — | — | quality | not promoted |
| `llama3.2:1b` | ollama | — | — | ollama |  |  | — | 1B | 1320 MB | mul | Llama 3.2 Community License | yes | available | — | — | fast, edge | not promoted |
| `gemma3:4b` | ollama | — | — | ollama |  |  | — | 4B | 3300 MB | mul | Gemma Terms of Use | yes | candidate | — | — | quality | not promoted |
| `phi4-mini:3.8b` | ollama | — | — | ollama |  |  | — | 3.8B | 2500 MB | mul | MIT | yes | candidate | — | — | fast | not promoted |
| `openrouter-summarize` | openrouter | — | — | gateway |  |  | — | provider-selected | — | mul | Proprietary provider terms | yes | available | — | — | quality | not promoted |

## Speaker (speaker)

| Model | Provider ids | Engine ids | Engine defaults | Runtime | Runtime model id | Batch adapter | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Pricing | Default role | Roles | Promotion evidence |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `sherpa-onnx-speaker-embedding` | speaker-verification | — | — | sherpa-onnx |  |  | `sherpa-onnx/speaker-eres2net-large-zh-cn-16k` | unknown | 27 MB | mul | Apache-2.0 | yes | available | — | yes | default | legacy exception |

## Full-duplex voice (duplex)

| Model | Provider ids | Engine ids | Engine defaults | Runtime | Runtime model id | Batch adapter | Artifact | Params | Size | Languages | Licence | Commercial | Readiness | Pricing | Default role | Roles | Promotion evidence |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `gpt-live-1.1` | openai-gpt-live | — | — | byok |  |  | — | provider-hosted | — | mul | Proprietary | yes | candidate | $0.05 active / $0.02 standby USD/min; 1s increments; [source](https://developers.openai.com/api/docs/pricing) | — | quality | not promoted |

## Declared defaults

Computed with an unconstrained platform (`DefaultFor(cap, "", Platform{})`), so GPU-required records are excluded. The commercial column applies the fail-closed filter.

| Capability | Default | Commercial default |
|---|---|---|
| stt | `whisper-small` | `whisper-small` |
| tts | `kokoro-82m-v1.0` | `kokoro-82m-v1.0` |
| summarize | `qwen3:1.7b` | `qwen3:1.7b` |
| speaker | `sherpa-onnx-speaker-embedding` | `sherpa-onnx-speaker-embedding` |
| duplex | _none_ | _none_ |

### Promotion evidence status

A seed status or reference is not proof: catalog-check joins each facet to its owner receipt and checks owner, model, capability, and outcome. A `Legacy exception` is code-allowlisted for a named operational route and states its evidence gap; it never claims qualification. Owner integration covers corpus quality, interactive latency, and strict model-memory receipts; memory still blocks until a real producer records a matching receipt. Platform and license remain blocking until their owners expose resolvable receipts.

- `whisper-small`: Existing operational default retained pending a joined corpus-quality, latency, memory, platform, and license receipt.
- `kokoro-82m-v1.0`: Existing operational default retained pending a joined corpus-quality, latency, memory, platform, and license receipt.
- `qwen3:1.7b`: Audio Tools compact summarize target retained pending owner-qualified receipts; evidence is one short report at 8/8 facts and one 300-word report at 14/22 facts, so this is not a general fact-faithfulness qualification.
- `sherpa-onnx-speaker-embedding`: Existing operational default retained pending a joined corpus-quality, latency, memory, platform, and license receipt; current resource score has no EER or threshold qualification.
- `sherpa-zipformer-en-2023-06-26` (engine default: sherpa-streaming): Existing operator-selected Sherpa streaming route retained pending joined corpus-quality, latency, memory, platform, and license receipts.

## Resource footprint

Default selection is also a resource decision. Each capability's promoted default must fit its declared budget (`DefaultFootprintBudgetMB`); a seed edit that promotes a larger model fails the budget gate instead of silently reserving more memory. Budgets: stt 600 MB, tts 400 MB, summarize 4000 MB, speaker 100 MB, duplex 0 MB.

| Capability | Default | Runtime | Declared size | Budget | Artifact |
|---|---|---|---|---|---|
| stt | `whisper-small` | whisper.cpp | 488 MB | 600 MB | `whisper/small-model` |
| tts | `kokoro-82m-v1.0` | kokoro | 330 MB | 400 MB | `kokoro/kokoro-model` |
| summarize | `qwen3:1.7b` | ollama | 1360 MB | 4000 MB | — |
| speaker | `sherpa-onnx-speaker-embedding` | sherpa-onnx | 27 MB | 100 MB | `sherpa-onnx/speaker-eres2net-large-zh-cn-16k` |
| duplex | _none_ | — | — | 0 MB | — |

Promoted defaults (the weight disk a fresh install of the referenced resources needs): **2205 MB**. Every available local record declared in this registry totals **21279 MB**; that is a catalogue total, not what an install pulls: a resource installs its eager artifact and leaves on-demand tiers unpulled. Remote (`byok`/`gateway`) and `browser` records carry no local weight footprint.

## Cautions

Records that are not both available and commercially clear. A caution is not a selection ban by itself: the readiness and commercial filters already enforce the hard rules.

- `whisper-large-v3` — readiness `candidate`, commercial `yes`. ~3.1 GB ggml artifact. The 6 GB VRAM figure is an operational comfort estimate, not a measured floor; CPU inference works but is slow.
- `kyutai-stt-2.6b-en` — readiness `candidate`, commercial `yes`. Candidate quality tier. The 16 GB VRAM figure is an estimate for the 2.6B checkpoint plus decoder overhead and is not measured on this host.
- `sherpa-zipformer-en-20m-2023-02-17` — readiness `candidate`, commercial `yes`. Registry-only candidate. The existing sherpa-onnx online Zipformer runtime is protocol-compatible with this model's encoder/decoder/joiner layout, but the managed resource pins and loads the 310 MB 2023-06-26 model and has no selectable model-dir route. No artifact is declared or acquired for this candidate, and no model default changes. Add a separately pinned resource artifact and runtime selection only after checksum, corpus quality, latency, memory, and platform qualification.
- `sherpa-zipformer-en-kroko-2025-08-06` — readiness `available`, commercial `unknown`. The Kroko community model is CC-BY-SA-4.0: commercial use is permitted but derivative distributions carry share-alike and attribution obligations. Kroko sells a separate commercial/OEM licence. Treat a commercial deployment as requiring a licence review until the operator confirms one.
- `rumble-2-flash` — readiness `candidate`, commercial `yes`. Batch ASR, not native streaming. The public model repository is 1.39 GB (1.02 GB main safetensors plus a 357 MB CTC head). Exact runtime RAM/VRAM requirements are unmeasured; the 2 GB VRAM value is only a registry lower-bound estimate, not a qualification. The upstream model card documents vLLM and Transformers GPU-oriented paths. This is not a default: the opt-in endpoint adapter exists; explicit unary selection remains unavailable unless the configured endpoint serves this model. No managed resource exists and quality/runtime qualification remains open.
- `kokoro-82m-v1.0-sherpa-int8` — readiness `candidate`, commercial `yes`. Explicit Sherpa-hosted quantized Kokoro route. Candidate status reflects incomplete runtime lifecycle and quality qualification; it is not the default model.
- `kale-tts-1-nano` — readiness `candidate`, commercial `yes`. Conditional product auto route: Kale Nano is tried before Kokoro when the Kale runtime is healthy; this does not promote Kale or change model resolve's Kokoro default. The pinned public, non-gated Hugging Face revision b6ce1887268c3667994fc3b0b2c82887c069915c declares Apache-2.0 in card metadata (last modified 2026-11-19); the training dataset is CC BY 4.0 and third-party frontend notices remain applicable. The int8 CPU ONNX artifact is 14.77 MB with upstream SHA256 4d5269974de4fba7a66f947608a9a5381b4e39c1a8a6073df6001ab7e7f76288; the complete installed model bundle is about 20 MB including voice assets. A checksum-pinned Linux/amd64 CPU runtime and product route exist, but joined multi-voice quality, lifecycle reliability, latency, memory, platform, and license receipts are incomplete. Keep candidate.
- `kale-tts-1-tiny` — readiness `candidate`, commercial `yes`. Public weights are available at upstream revision 69011009e3380f1cc8fe990cbb55b799bde8455a, but its model card still says DRAFT, NOT PUBLISHED. The int8 CPU ONNX artifact is about 79.9 MB. The graph requires Kale's own ONNX Runtime frontend/server; it is not compatible with sherpa's current Kokoro adapter. Keep candidate until the scenario pins the full runtime bundle and qualifies output quality, latency, licensing notices, and resource use.
- `kale-tts-1-base` — readiness `experimental`, commercial `yes`. Experimental: the PyTorch path is a scratch/reference implementation and artifact size is not published. Not a default.
- `qwen3-tts-12hz-0.6b-base` — readiness `candidate`, commercial `yes`. Qwen documents cloning from reference audio plus transcript and 10 languages. The pinned model repository totals 2.52 GB. No Audio Tools adapter/resource exists; the model is not selectable or a default. Voice-use authorization, ephemeral reference handling, bounded memory, output quality, and lifecycle remain unimplemented.
- `reverb-tts` — readiness `experimental`, commercial `no`. Weights are CC BY-NC 4.0: non-commercial use only, with attribution. Commercial use requires a negotiated licence from Software Mansion. Runtime code is Apache-2.0.
- `sopro-v2-turbo-voice-cloning` — readiness `candidate`, commercial `yes`. Compact voice-cloning candidate only; not selectable or default-eligible. Audio Tools must implement source-bound permission and revocation, transient reference-audio handling with bounded deletion, an isolated adapter/runtime, dependency-license review, and held-out quality/resource qualification before routing.
- `verba-tts-indic-zero` — readiness `watch`, commercial `unknown`. No confirmed licence or public release was found. Treat every use as unlicensed until the owner publishes terms.
- `gemma3:4b` — readiness `candidate`, commercial `yes`. 3.3 GB is the quantized pull size. Do not pull without operator approval.
- `phi4-mini:3.8b` — readiness `candidate`, commercial `yes`. 2.5 GB is the quantized pull size. Lower-latency fallback role candidate.
- `gpt-live-1.1` — readiness `candidate`, commercial `yes`. Hosted full-duplex multilingual voice model. Catalogued for source, readiness, and published rate visibility; Audio Tools integration and qualification are incomplete, so this model is not runnable or qualified.

## Provider and engine vocabulary

Provider ids are the stable health/lifecycle ids owned by `internal/providercatalog` (exposed through `internal/capabilities`). Engine ids are STT stream-selection ids owned by `internal/sttengine`; non-STT records, BYOK/gateway/browser providers, and batch adapters with no engine leave `engine_ids` empty. An engine id that no provider owns is drift: the cross-catalog conformance test rejects it.

Provider ids: `whisper-stt`, `kyutai-stt`, `sherpa-stt`, `kokoro-tts`, `ollama`, `openai-whisper`, `deepgram`, `openai-tts`, `elevenlabs`, `openrouter`, `browser-stt`, `browser-tts`, `rumble-stt`, `kale-tts`, `qwen3-tts`, `sopro-tts`, `reverb-tts`, `verba-tts`, `speaker-verification`, `openai-gpt-live`.

Engine ids: `whisper-local`, `kyutai`, `sherpa-streaming`.
