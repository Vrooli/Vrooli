# Audio Tools model footprint audit — 2026-11-20

## Runtime refresh — 2026-11-24 10:21 UTC

Read-only host snapshot: product API 20,516 KiB RSS, lab API 19,760 KiB,
Whisper activity process 11,372 KiB, Kale Python 16,180 KiB, and the attached
Sherpa process 7,316 KiB. The Sherpa listener returned HTTP 200 at `/healthz`;
the Kale process is listening on its configured port. These process RSS values
are not complete model or peak-memory measurements. Audio Tools processes were
idle or near-idle at sampling time (the Sherpa process briefly read 2.5% CPU).

`nvidia-smi` reported 5,375 MiB allocated and 1% utilization. Its compute
process list attributed 608 MiB and 2,984 MiB to two Ollama runners and 1,380
MiB to the shared reranker; no Audio Tools process appeared in the allocation
list. This is point-in-time attribution, not per-request causality. It does
not support attributing the current large GPU allocations to Audio Tools.

The checked-in compact defaults remain Whisper Small (488 MB), streaming
Sherpa Zipformer (310 MB), and Kokoro (330 MB) with conditional Kale Nano
selection; Audio Tools summarize remains Qwen3 1.7B (1.36 GB). Smaller STT
candidates lack the joined quality and latency evidence needed to replace
these defaults. Voice cloning has no qualified default. No model, resource,
or service was changed for this audit.

## Owner-measured CPU inference — 2026-11-23 19:57 UTC

The lab-only `audio-tools --instance lab validation run-host-portability
--json` path recorded artifact
`audio-tools:host-portability:20261123T195701.047636597Z:amd64` for retained
target `portable-voice-v2`. The digest-pinned, network-disabled D1 CPU runner
matched its Whisper Small fixture in 2.561 s with a measured cgroup-v2 peak of
1,330,667,520 bytes against the declared 2 GiB model budget. The D2 constrained
CPU runner passed its Sherpa Zipformer smoke in 1.870 s with a measured peak of
450,588,672 bytes against the same declared budget. Both per-model memory
receipts were persisted successfully.

The overall host-portability result remains **incomplete**: the required D1
Kyutai accelerator cell is unavailable because its provider is not serviceable.
The D1 check is pinned CLI inference, not the managed Whisper HTTP path; D2 is
a bounded native smoke, not browser timing or a quality evaluation. No
cross-platform, microphone, acoustic-quality, or full portability claim follows.

The first attempt through the default product API performed inference but could
not persist evidence because experiment evidence endpoints are lab-only. The
validation CLI now probes that read-only service before resource-intensive
work and directs product-instance invocations to `--instance lab`.

## Runtime refresh — 2026-11-23 19:43 UTC

Read-only host snapshot: Audio Tools was healthy at API port 19630. Its API
process used 21,936 KiB RSS; Kale Nano's CPU-side Python process used 18,220
KiB RSS; and the attached Sherpa server used 6,576 KiB RSS while idle. The
product-selected streaming engine remains `sherpa-streaming`, and the local
auto TTS route reports Kale healthy. RSS does not include all mapped model
pages, so these values are process observations rather than complete model
memory measurements.

Ollama `/api/ps` listed shared `qwen3.5:4b` and `nomic-embed-text`; Audio
Tools' `qwen3:1.7b` summarize model was not resident. The generic shared model
and the embedding model are not Audio Tools' selected summarize model. GPU
memory was 7,471 / 16,376 MiB at 19% utilization. Compute-process attribution
showed about 5,072 MiB for an Ollama runner, 606 MiB for the embedding runner,
and 1,380 MiB for the shared reranker. This is a point-in-time reading; it
does not establish peak memory or prove which request caused a model to load.

No model was started, stopped, or unloaded. Current Audio Tools selections
remain the compact licensed Sherpa Zipformer streaming route, Qwen3 1.7B for
Audio Tools summarization, and conditional Kale Nano TTS with Kokoro retained
as the catalog default pending human quality review. Voice cloning still has
no qualified supported default. This snapshot does not support attributing
the observed shared GPU allocations to Audio Tools.

## Runtime refresh — 2026-11-23 16:47 UTC

This read-only snapshot reconciles the earlier host audit with the current
resolver and process state. `audio-tools health show` reports Sherpa STT and
Kale TTS available; Kokoro and Whisper are unavailable. The effective STT
provider is available through Sherpa, while the model resolver's batch default
remains Whisper Small. `resource-whisper status --json` reports stopped. The
selected Sherpa listener is still an externally launched process at
`/tmp/sherpa-bundle-refresh-20261121/server/sherpa-onnx-server`, so its presence
does not prove a managed Sherpa install. The catalog TTS default remains
Kokoro; product `auto` may select Kale Nano when healthy. Kale's runtime
process is CPU-side, but its RSS does not measure mapped model pages or its
complete resource footprint.

Declared defaults remain compact and no model-policy change is supported by
current qualification evidence: Whisper Small is 488 MB / MIT, Sherpa Zipformer
is 310 MB / Apache-2.0, and Kokoro is 330 MB / Apache-2.0. Audio Tools
summarize resolves to Qwen3 1.7B at 1,360 MB / Apache-2.0, and it was not
resident in Ollama `/api/ps`. Voice cloning has no qualified supported default;
the smaller-looking candidates still lack a complete runnable, consented,
licensed, quality-qualified route. Kale Nano remains a conditional candidate,
not a human-reviewed catalog default.

At the same sample, product API, lab API, attached Sherpa, and Kale Python RSS
were 22,684 / 21,900 / 5,364 / 15,708 KiB. The two Ollama runner processes
were 415,272 and 80,652 KiB RSS; RSS is not the full model residency. GPU use
was 7,363 / 16,376 MiB at 7% utilization. Compute-process attribution showed
5,080 MiB and 606 MiB for the Ollama runners and 1,278 MiB for the shared
reranker; about 399 MiB remained unassigned by this process query. Ollama
`/api/ps` listed shared `qwen3.5:4b` (4,272,234,167 bytes VRAM) and
`nomic-embed-text:latest` (323,150,151 bytes); Audio Tools' `qwen3:1.7b` was
not resident. The current large GPU allocations therefore belong to shared
Ollama and reranker activity, not Audio Tools' selected summarizer. Total host
memory was 60 GiB, with 28 GiB available; the root filesystem was 90% used
with 176 GiB free. These are point-in-time host facts, not Audio Tools peak
inference memory or per-request attribution.

No model was started, stopped, or unloaded for this audit. The only identified
resource action is evidence reconciliation: keep the small licensed defaults,
retain Kale's quality gate, and do not attribute shared Ollama/reranker GPU
residency to Audio Tools. The unresolved device/runtime work remains an
externally managed Sherpa listener, unmeasured inference peaks, and the
unassigned GPU remainder.

## TTS automatic-selection policy update (2026-11-21 12:11 UTC)

The source policy has a narrow, declaration-gated exception to ordinary TTS
auto selection. Active providers with a commercially usable, qualified
`default` model remain the standard route. A provider may separately opt into
`conditional_auto` for a non-default commercial candidate; Kale Nano declares
this path and is ordered before Kokoro when its runtime is healthy. Kokoro
remains the qualified catalog default and availability fallback. This runtime
preference does not promote Kale's provider or model, and does not establish
held-out multi-voice quality or lifecycle qualification. The pinned Nano
revision is public, non-gated, and declares Apache-2.0; its CC BY 4.0 training
set and bundled third-party notices remain part of release review. The live
product API process predates this source policy, so source tests establish the
selection rule, not current live-product behavior.

This is a snapshot of the checked-out declarations and receipts, not a claim
about which processes are running now. No shared service was started, stopped,
or reconfigured for this audit. The model registry and resource manifests are
the authority for declared defaults; a persisted product-path receipt is
stronger evidence than a model card or a single resource-level smoke.

## Live host snapshot — 2026-11-20 20:50 UTC

Read-only runtime evidence: `vrooli scenario status audio-tools --json`
reported the scenario healthy at API port 19630 and UI port 23345. The API
process was 19,812 KiB RSS and UI server 39,228 KiB RSS. The separate attached
Sherpa server was 7,868 KiB RSS while idle. `audio-tools stt stream-config`
resolved `sherpa-streaming`; `audio-tools model resolve --capability summarize
--json` resolved `qwen3:1.7b` as the available, Apache-2.0 summarize default.

Ollama `/api/ps` showed `qwen3.5:9b` resident with 5,880,141,577 bytes VRAM
allocation and `nomic-embed-text` with 323,150,151 bytes. Neither is the
Audio Tools summarize default; the 9B role remains shared `chat.default`, and
the embedding model serves shared embedding traffic. `qwen3:1.7b` was not
resident in that snapshot. This distinguishes process residency from Audio
Tools model policy: the API and UI processes are small, while the largest
observed GPU allocation belongs to a different shared workload. No model was
unloaded or service restarted.

## Declared defaults and evidence

| Capability | Declared default now | Footprint and rights | Quality / runtime evidence | Assessment |
| --- | --- | --- | --- | --- |
| Batch STT | `whisper-small` | 488 MB weights; MIT; eager `ggml-small.bin`; 600 MB default budget | One-host canonical clean clip exact; 75 ms product-path response; 716 MiB observed VRAM. No representative dictation, long-form timestamp-drift, or device-matrix qualification. | Small is applied in the registry and Whisper resource. Strong footprint improvement; keep quality limits explicit. |
| Streaming STT | `sherpa-zipformer-en-2023-06-26` | 310 MB extracted; Apache-2.0 model declaration; CPU-capable; 600 MB shared STT budget | End-to-end server path works, but the canonical clip and JFK sample contain recognition substitutions. It is a low-footprint streaming option, not a dictation-quality result. | Compact default is applied for this engine. Do not generalize its smoke tests into STT quality acceptance. |
| TTS | Catalog default `kokoro-82m-v1.0`; `AUDIO_TTS_BACKEND=auto` conditionally prefers Kale Nano | Kokoro is the 330 MB local default; auto mode selects Kale when its managed runtime is healthy and falls back to Kokoro when unavailable. | Kale product-route output and fallback/recovery are verified, but naturalness and release qualification remain open; Nano's pinned revision declares Apache-2.0 while release provenance and multi-voice quality receipts remain incomplete. | Runtime preference is implemented; Kale is not the catalog default or qualified release. |
| Summarize | `qwen3:1.7b` via `summarize.default` | 1.36 GB installed; Apache-2.0; 4 GB budget. Runtime memory is not measured in the current receipt. | One-host short report kept 8/8 facts and loaded in 2.2 s; 300-word report kept 14/22 and dropped the last paragraph. `qwen3.5:4b` kept 22/22 on that long fixture at 3.39 GB, but its upstream licence is unverified. | Smaller 1.7B default is applied; there is a concrete fact-retention tradeoff. For long fact-dense summaries, the 4B fallback has better evidence, but no commercial default promotion is justified while its licence is unknown. |
| Speaker verification | `sherpa-onnx-speaker-embedding` | About 27 MB declared weights; Apache-2.0; 100 MB budget | Resource reports hybrid centroid/clip scores; no EER or threshold qualification is pinned. Actual loaded memory/startup contribution is not isolated. | Small default is applied, but the quality threshold remains unqualified. This is speaker verification, not voice cloning. |
| Voice cloning | No local qualified default | Reverb TTS advertises cloning, but its weights are CC-BY-NC-4.0 and it is a credentialed remote route. | No Audio Tools naturalness, latency, or cloning receipt. | No model should be promoted as a commercial local cloning default from current evidence. |

## Applied versus proposed

The applied changes visible in the current tree are Whisper Small as the
resource-pinned batch default and registry default, Sherpa Zipformer as the
streaming default, and Qwen 3 1.7B as the shared summarize role primary. These
choices materially reduce the earlier Whisper Medium and Qwen 3.5 9B footprint.
The registry enforces per-capability default weight budgets (600 MB STT,
400 MB TTS, 4,000 MB summarize, 100 MB speaker) and checks that local defaults
refer to eager artifacts.

Kale Nano is a *candidate*, not the model catalog default. The checked-in
evidence records a pinned 14.77 MB int8 graph and hash, a checksum-pinned
Linux/amd64 runtime candidate, 6.14 s managed-resource startup, a 3.05 s
resource-level synthesis request at 770 ms cold / 59 ms warm, and a separate
Audio Tools product-route response (2,048 ms audio, 175 ms warm request, cache
reuse). The model card still says “DRAFT, NOT PUBLISHED”; vendor leaderboard
numbers are not independent quality evidence. A 498 MiB composed runtime tree
also means the graph size is not the installed runtime footprint. Auto mode
prefers healthy Kale and retains Kokoro as fallback, while the catalog default
stays Kokoro until release and quality gates pass. A release-ready
license/provenance review, reviewed multi-voice output quality, and product
path resource/latency receipts qualify Nano.

**Source correction (2026-11-21):** Hugging Face API metadata for the pinned
Nano revision reports public, non-gated Apache-2.0 status (last modified
2026-11-19). The earlier draft-status statement in this dated audit is
superseded for Nano only. Tiny remains draft.

Kale Tiny has no managed runtime in this evidence set. Kroko is a smaller
streaming STT candidate but remains unqualified for quality, latency, timestamp
behavior, and license obligations. Rumble now has an opt-in batch adapter for
an operator-configured vLLM endpoint, but it has no Audio Tools-managed resource
or artifact. None should be silently turned into a default because its
parameter count or download is smaller.

### Rumble batch adapter gate

The checked-in Rumble record remains `candidate`, has no engine id or local
artifact reference, and is not eligible to become the STT default. Audio Tools
now has a shared `BatchBackend`/`BatchProvider` seam alongside the existing
Whisper service adapter. Rumble uses a bounded adapter for an explicitly
configured vLLM OpenAI-compatible audio-chat endpoint; it is wired only when
the model declaration names that protocol. Native streaming adapters still use
a separate WebSocket contract, so Rumble remains batch-only.

The pinned upstream repository contains about 1.39 GB of files (1.02 GB main
safetensors plus a 357 MB CTC head). This is download footprint only. Peak RAM,
VRAM, host startup, inference latency, CPU support, and complete managed runtime
size are unknown; the seed's 2 GB VRAM figure is an unverified lower-bound
estimate, not a safe admission budget. The documented upstream paths are
GPU-oriented vLLM and Transformers, neither of which is installed or managed by
Audio Tools for Rumble.

The current endpoint-backed path requires no idle Audio Tools model process or
weights download. It has bounded request/response sizes, cancellation, and
read-only health and served-model checks. Before creating a managed resource or
calling Rumble qualified, require: checksum-pinned model and runtime artifacts;
a measured resource envelope and CPU/GPU support declaration; Audio Tools
product-route smoke and latency receipts; and representative
multilingual/long-form WER, timestamp, and language receipts. Keep license
quality/qualification separate from the upstream Apache-2.0 assertion. No
weights were downloaded or loaded for this audit.

## What to measure before another promotion

- TTS: compare Kokoro and candidate voices on the same licensed, held-out text
  set and named voices; retain intelligibility/naturalness judgments, first
  audio and total latency, cold and warm resource startup, peak RSS/VRAM, and
  complete runtime disk footprint. Test the Audio Tools route, not only the
  model server.
- STT: use the approved held-out corpus and long-form cohort; record WER,
  dropped spans, timestamp drift, product-path latency, cold/warm residency,
  language coverage, and CPU/GPU behavior. Keep clean one-clip smoke results
  separate from qualification.
- Summarize: use the persisted experiment harness on short and fact-dense
  reports with fact retention as an explicit outcome; measure cold load, warm
  latency, and resident memory for each concrete tag. Resolve upstream rights
  before any commercial-default choice.
- Speaker / cloning: qualify speaker EER and threshold on the authorized corpus;
  treat cloning as a separate capability with its own consent, rights, quality,
  and resource evidence. Do not infer cloning qualification from embeddings.

## Source records

- `api/internal/modelcatalog/models.seed.json` and generated `docs/reference/models.md`
- `api/internal/modelcatalog/footprint.go` and `footprint_test.go`
- `resources/whisper/resource.json` and `resources/kokoro/resource.json`
- `resources/ollama/model-policy.json`
- `docs/internal/PROBLEMS.md`, entries dated 2026-11-20 (“machine footprint” and
  “Kale Nano prototype”) and the 2026-11-19 Whisper receipts

## Runtime refresh (2026-11-20 22:09 UTC)

Read-only status confirmed Audio Tools is healthy with two processes; its API
process used about 23.7 MiB RSS and had been running for over 15 hours. The
live CLI still resolves Whisper Small for batch STT, Sherpa streaming with
`passthrough`, Qwen3 1.7B for summarize, Kokoro as the model-catalog TTS
default, and Sherpa speaker embedding for speaker verification. The deployed
process predates the current source edits, so its provider-routing output
(`kokoro down`, BYOK up, Vrooli down) does not validate the new Kale auto
selection. Kale's managed resource is installed but stopped; its last managed
run ended at 21:31 UTC. The sherpa resource is disabled/unsupported for managed
Linux lifecycle, while the separately attached `/healthz` endpoint on port
8881 returned HTTP 200. This confirms the attach-only health path, not that
Kale or Audio Tools has been restarted onto current source. No shared service
was restarted or model unloaded.

## Runtime refresh (2026-11-20 21:20 UTC)

The live `audio-tools` CLI still resolves streaming STT to `sherpa-streaming`
with `passthrough`, and summarize to `qwen3:1.7b`. The shared API and UI
processes measured 22,272 KiB and 35,896 KiB RSS, respectively; they were
started at 06:26 UTC, before the current source edits. The host GPU reported
9,390 MiB used of 16,376 MiB at 1% utilization. Ollama `/api/ps` attributed
5.88 GB of VRAM to `qwen3.5:9b` and 323 MB to `nomic-embed-text:latest`.
The 9B model remains resident for the separate shared chat role; no Audio Tools
operation was routed to it, and no shared model was stopped or unloaded.

## Managed TTS product route (2026-11-20 21:31 UTC)

The isolated Audio Tools product Connect route was exercised against the
managed Kale Nano resource. A 2,048 ms product response was 98,424 bytes; the
warm request took 175 ms and repeat requests hit cache. A 1.25x speed request
returned a valid finite-size WAV, including corrected RIFF/data chunk sizes.
This closes model-backed route integration, but it does not assess naturalness,
release status, or promotion readiness. Kale remains a candidate; Kokoro stays
the catalog default while `auto` can conditionally prefer healthy Kale.

## Runtime refresh (2026-11-21 01:07 UTC)

Read-only process sampling found the shared Audio Tools API at 25,572 KiB RSS
and UI at 39,532 KiB RSS. The Audio Tools API process still predates the
current source edits. The attached Sherpa `/healthz` endpoint on port 8881
returned HTTP 200, and the Kale resource was stopped after the multi-voice
product smoke. The CPU-only Kale resource has no accelerator capacity block,
so `vrooli resource start kale-tts --no-reclaim` was refused; an ordinary
managed start succeeded and was stopped after the smoke. The managed Whisper resource was healthy and serving under
`vrooli resource status whisper --no-fast`, using the Vulkan build with
`ggml-small.bin`; `nvidia-smi` attributed 716 MiB of VRAM to its
`whisper-server` process.

The same GPU snapshot attributed 6,886 MiB to an Ollama `llama-server`, 602
MiB to a second Ollama server process, and 1,380 MiB to `reranker`. Ollama's
`/api/ps` identified `qwen3.5:9b` at 5.88 GB VRAM and
`nomic-embed-text:latest` at 323 MB. Audio Tools' summarize resolver still
selects `qwen3:1.7b`; the observed 9B model is the separate shared chat role,
not the Audio Tools summarize selection. No models were stopped or unloaded.

The default/fast Whisper status command reported unhealthy with an HTTP check
failure even though a direct GET to its declared `/` health target returned
200; `--no-fast` then reported healthy/serving. This is a status-path
discrepancy to investigate, not a Whisper outage. No shared service was
restarted.


## Runtime snapshot (2026-11-21, read-only)

At the audit sample, `nvidia-smi` reported 9,267 / 16,376 MiB used (2% GPU
utilization). Compute-process attribution showed two shared Ollama
`llama-server` processes at 6,878 and 608 MiB and the separate reranker at
1,380 MiB. The Audio Tools API process was 22,848 KiB RSS. No Whisper, Sherpa,
Kokoro, or Kale model-serving process appeared in the process or GPU compute
inventory, so this sample attributes no resident model footprint to Audio
Tools; it does not establish that all Audio Tools paths are always idle. The
remaining roughly 401 MiB GPU allocation was not attributed by that query.

No declaration change is justified by this snapshot. Current product defaults
already use the smaller declared choices, and there is no model-quality or
lifecycle receipt that justifies promoting a smaller candidate. The next
footprint decision should be a same-host, product-path comparison of Kokoro
and pinned Kale Nano capturing cold/warm latency, peak RSS/VRAM, complete
runtime disk size, and held-out multi-voice naturalness; retain Kokoro as the
catalog default until those checks pass. The only measured GPU-heavy processes
here belong to shared Ollama and reranker workloads, so reducing Audio Tools
model weights cannot reclaim their current residency.

## Runtime refresh (2026-11-21 08:21 UTC)

Read-only host sampling found Audio Tools API/UI RSS of 27,712 / 68,268 KiB.
`audio-tools model resolve --capability summarize --json` selected the
available Apache-2.0 `qwen3:1.7b` as the summarize default. It was not resident
in Ollama's `/api/ps`; that shared runtime instead had `qwen3.5:9b` and
`nomic-embed-text:latest` loaded. GPU usage was 10,823 / 16,376 MiB at 2%
utilization. Compute attribution showed 7,702 MiB and 608 MiB in two shared
Ollama `llama-server` processes, 1,380 MiB in the separate reranker, and 719
MiB in Whisper Small. The Ollama API identified the 9B model as
`qwen3.5:9b`; its approximately 7.7 GiB process allocation is not the
Audio Tools summarize model. No model-serving process for Sherpa, Kokoro, or
Kale appeared in the sample.

The live `audio-tools stt engines --json` output keeps the operator-pinned
`sherpa-streaming` active but unavailable; Whisper Local is available as a
batch engine. `vrooli resource status sherpa-onnx --json` reports
`unsupported_platform`, uninstalled, and stopped. Thus the compact policy is
set, but the chosen streaming path cannot currently serve through a managed
resource. No process or model was started, stopped, or unloaded. This is a
point-in-time attribution, not a benchmark; it shows the present large GPU
resident is shared Ollama chat work, not Audio Tools summarize traffic.

## Runtime refresh (2026-11-21 21:12 UTC)

The current CLI selects attached `sherpa-streaming`, and the model resolver
selects available Apache-2.0 `qwen3:1.7b` for summarize. A product
`audio-tools tts synthesize` request selected `kale-tts` /
`kale-tts-1-nano` / `kenzie`, returned a valid 2.975 s mono WAV (142,844 B),
and reported 3,009 ms on the first request and 822 ms on the repeated request.
This confirms current auto routing and warm-path latency, not voice quality or
release qualification.

At the same read-only sample, the RTX 4070 Ti SUPER reported 11,297 MiB used
of 16,376 MiB at 2% utilization. Ollama `/api/ps` showed `gemma4:12b` at
8,390,747,094 B and `nomic-embed-text` at 323,150,151 B; compute attribution
showed 8,896 MiB and 606 MiB across two Ollama `llama-server` processes,
1,380 MiB in the separate reranker, and about 415 MiB not accounted for by
those process rows. The API did not report `qwen3:1.7b` as resident. Audio
Tools API and lab API were about 20.4 / 19.4 MiB RSS (19.0 / 17.8 MiB PSS),
Kale's Python process about 17.8 MiB RSS, and attached Sherpa about 6.2 MiB
RSS; mapped model pages and full disk/runtime sizes are not represented by
those process RSS values. The 8.39 GB model and reranker are not Audio Tools'
selected summarizer or STT models, so changing Audio Tools defaults would not
reclaim their current VRAM. No shared process was stopped or reconfigured.

## Tiny Voca comparison (2026-11-21)

Tiny Voca was evaluated against the current product-auto TTS route, Kale Nano.
This is an upstream/runtime fit assessment, not a paired benchmark: Tiny Voca
was not downloaded or executed, and no model route, global role, or shared GPU
state was changed.

| Measure | Tiny Voca | Current route: Kale Nano | Comparison status |
| --- | --- | --- | --- |
| Weights | Upstream lists a 21.4 MB FP32 ONNX graph (and a separate 18.2 MB PyTorch checkpoint); 4,537,291 parameters. | Pinned int8 ONNX graph is 14.77 MB, SHA256 `4d5269974de4fba7a66f947608a9a5381b4e39c1a8a6073df6001ab7e7f76288`. | Artifact-only comparison: Tiny's ONNX graph is 6.63 MB larger. Neither number is full installed/runtime footprint. |
| Runtime and coverage | CPU-capable ONNX Runtime path documented upstream; English, fixed F2/M2 voices, no streaming or cloning. Package uses `dizione` phonemizer; graph limit is 512 tokens / 32 seconds per call. | Managed Linux/amd64 CPU ONNX runtime and Audio Tools product route exist; English candidate voice `kenzie` was exercised. | Tiny has no managed resource/provider or Audio Tools product route. Kale's current serving adapter is model-specific and cannot load Tiny's graph. |
| Latency | Maintainer reports about 70x realtime on four CPU threads. Not measured here. | Product CLI at 2026-11-21 21:12 UTC: 3.009 s first request and 822 ms repeated request for a 2.975 s, 142,844-byte WAV. | Not comparable: Tiny figure is an upstream throughput claim without our text, hardware, cold/warm protocol, or route overhead. |
| CPU/RAM/VRAM | No local measurement. CPU inference is documented; memory/peak RSS unknown. | At a separate point-in-time sample, Kale Python process was 17.8 MiB RSS; mapped model pages and peak/call attribution were not measured. GPU attribution showed no Audio Tools model process; shared GPU use was 11,297/16,376 MiB, dominated by shared Ollama chat (8.39 GB model) plus reranker. | No peak-RAM/VRAM or contention comparison. Tiny's CPU path could avoid GPU demand, but that is not a measured resource result. Existing shared GPU consumers make a low-resource CPU candidate relevant; Audio Tools defaults did not account for the observed large allocation. |
| Rights/evidence | Upstream GitHub and HF declare Apache-2.0 for code/weights; phonemizer dictionary notices and dataset/source attribution remain to be reviewed. No formal paper or independent listening evidence found; upstream describes only a small sanity check. | Model declaration records Apache-2.0 weights and CC BY 4.0 training dataset; product routing works, quality/release qualification remains incomplete. | Neither route earns a quality qualification from these receipts. Tiny needs complete transitive/dataset notice review and held-out/blinded evidence. |

Primary upstream records: [Tiny Voca GitHub](https://github.com/n4ze3m/tiny-voca),
[Hugging Face model card/files](https://huggingface.co/n4ze3m/tiny-voca),
[upstream license](https://github.com/n4ze3m/tiny-voca/blob/main/LICENSE), and
[model implementation notes](https://github.com/n4ze3m/tiny-voca/blob/main/docs/model.md).
The model card reports 4.5M parameters, 24 kHz mono output and approximately
70x realtime on four CPU threads; these remain author claims. It documents
English-only output, two voices, pronunciation limitations and no streaming.

**Integration decision:** do not add a declaration-only model entry yet. The
Audio Tools catalog's `runtime`/`artifact` declaration describes model identity,
but actual local synthesis requires a provider runtime declaration and a
matching adapter. Current managed runtimes are Kale's model-specific ONNX
service and Kokoro/Sherpa's Kokoro-specific service; neither provides a generic
Tiny Voca inference contract, and Kale's provider is already marked planned.
Reusing either endpoint would misstate the model served. A valid next route is a
separately managed, checksum-pinned Tiny Voca CPU service (including the graph,
phonemizer assets/notices, bounded sentence splitting, health/provenance and
request limits), then a declaration-driven provider and product-route adapter
following the existing local TTS composition. Only after that route exists can
the same corpus/texts and host be used for 30+ paced cold/warm attempts per
profile, retaining failures and measuring p50/p95, peak RSS/VRAM, startup,
throughput, full artifact/runtime disk size and shared-host contention. Hold
quality claims until the approved disjoint held-out cohort has blinded review.

## Runtime refresh (2026-11-22 America/New_York; sampled 2026-11-23 02:00 UTC)

Read-only verification used `vrooli scenario status audio-tools --json`, the
Audio Tools model and engine resolvers, `vrooli resource status <slug> --json`,
`ps`/`/proc/<pid>/status`, `nvidia-smi`, and the read-only Ollama
`/api/ps` endpoint. Audio Tools was healthy at API 19630/UI 23345, with build
identity `sha256:90478607e21b1b7514f80268382e6c60bb2f732f20d1a0dcb6ead2be936f35f6`.
The live processes were PID 2634523 (API: 30,860 KiB RSS, 37,160 KiB HWM) and
PID 2634745 (UI: 68,256 KiB RSS, 78,328 KiB HWM). This sample is a point-in-time
resident-set observation, not peak inference memory; RSS includes shared mapped
pages and is not additive physical memory.

Live resolution remained: batch STT `whisper-small` (MIT, 488 MB declared),
catalog TTS `kokoro-82m-v1.0` (Apache-2.0, 330 MB declared), Audio Tools
summarize `qwen3:1.7b` (Apache-2.0, 1,360 MB declared, role
`summarize.audio_tools`), and speaker `sherpa-onnx-speaker-embedding`
(Apache-2.0, ~27 MB declared). Streaming remains operator-pinned to
`sherpa-streaming` with `passthrough`; the engine resolver reports it selected
and available. These weights/rights are registry declarations, not measured
resident sizes.

Resource status at sampling: Whisper and Kokoro were installed but stopped;
Ollama was healthy and serving; Kale TTS was healthy and running. Kyutai STT
was stopped and reported `needs_reacquire` because its staged artifact no
longer matched the manifest and had no staging witness. No reinstall was
attempted because Kyutai is not the selected engine. The sherpa-onnx resource
reports unsupported on Linux, while the selected stream engine's separately
running server was PID 2557133 from
`/tmp/sherpa-bundle-refresh-20261121/server/sherpa-onnx-server` (7,660 KiB
RSS). That temporary-path process is not evidence of a managed sherpa resource
or durable resource attribution.

Kale's running Python/uvicorn process (PID 3742059) measured 17,536 KiB RSS.
Audio Tools `provider list --json` showed Kokoro stopped and did not include a
Kale provider, although `kale-tts` is declared in the scenario resource list
and the managed Kale resource is healthy. This confirms a live Kale resource
but does not attribute it to a particular Audio Tools request or prove a
current product-route model selection. No provider/resource was stopped.

The RTX 4070 Ti SUPER reported 15,242 / 16,376 MiB GPU memory used at 1%
utilization. Compute process rows attributed 5,080 MiB, 7,766 MiB, and 606 MiB
to three Ollama `llama-server` processes and 1,380 MiB to the shared reranker,
for 14,832 MiB; approximately 410 MiB remained unexplained by those rows.
Ollama `/api/ps` identified `qwen3.5:4b` at 4,272,234,167 bytes VRAM,
`nomic-embed-text:latest` at 323,150,151 bytes, and `gemma4:12b` at
7,459,496,261 bytes VRAM (9,299,537,425 bytes model size). Audio Tools'
selected `qwen3:1.7b` did not appear as resident. The observed Ollama models are
shared roles, not the Audio Tools summarize selection. No Whisper, Kokoro,
speaker model, or Audio Tools-selected summarizer appeared in the GPU compute
rows. Kale's measured process is CPU-side, but its process RSS alone does not
measure model-mapped memory or full resource disk footprint.

No default change is justified by this sample: the smaller commercial defaults
remain declared and the selected summarize model is not resident. The one
confirmed install-state discrepancy is the stopped Kyutai artifact's missing
staging witness; it is safely left untouched because it is not selected and
reacquisition could mutate/download the resource. Remaining attribution gaps:
the unexplained ~410 MiB GPU allocation; whether the temporary Sherpa server is
serving Audio Tools traffic; whether the live Kale instance is serving an
Audio Tools request; and per-request peak RSS/VRAM, mapped model memory, and
complete installed disk footprint.


## Runtime refresh — 2026-11-23 18:28 UTC

The live API at build `sha256:9e3367c8bb7c11abcade17dcf4aba7327c4a318e723d790cc61acb5e564f0977` remains healthy. Current health shows Sherpa STT, Kale TTS, and local Ollama summarize available; Whisper and Kyutai are unavailable. Registry resolution selects the 310 MB Apache-2.0 Sherpa Zipformer for `sherpa-streaming` and the 1.36 GB Apache-2.0 Qwen3 1.7B model for Audio Tools summarize. The TTS catalog still identifies Kokoro as the qualified default; the current healthy product `auto` route selected Kale Nano (`kenzie`) and returned a provider latency of 525 ms. This confirms the user's compact-model route preferences are active in product routing while the unqualified catalog-default guard remains intact.

The Ollama process snapshot contains shared `qwen3.5:4b` (4,272,234,167 bytes VRAM) and `nomic-embed-text:latest` (323,150,151 bytes); Audio Tools' Qwen3 1.7B summarize model was not resident. Device-level GPU use was 6,329 / 16,376 MiB at 5% utilization. The separate compute-process query attributed 5,080 and 606 MiB to Ollama runners, 550 MiB to an Image Tools probe, and 1,380 MiB to the shared reranker; their 7,616 MiB sum exceeds the device-level use, so the discrepancy is unresolved and these rows are not treated as additive accounting. The shared Ollama and reranker services were not changed. These are point-in-time observations, not peak model memory or session contention qualification.

Direct loopback Connect `ListEngines` calls took 114.79 ms, then 3.97 and 3.69 ms on immediate repeats, followed by 80.14 ms after a 30+ second idle interval. These are server-request wall times from one process and do not establish browser readiness, model cold start, or an SLO percentile. No provider or shared model was started, stopped, or unloaded for this snapshot.

## Runtime refresh — 2026-11-23 18:10 UTC

This point-in-time snapshot follows one local TTS smoke and five distinct warm
requests. `audio-tools tts synthesize` selected `kale-tts` /
`kale-tts-1-nano`; first-request provider latency was 646.6 ms and the five
warmed requests measured 61.8, 68.9, 75.1, 65.4, and 94.8 ms (median 68.9
ms). A WAV sample was 24 kHz mono, 2.55 s, and non-silent (RMS 2,810 PCM
units). The Kale service process used about 90 MiB RSS after the requests and
about 0.1% CPU at the sample. No GPU use change was observed. This proves a
working compact local route and signal integrity for one voice/text sample,
not human naturalness or broad quality.

GPU use was 7,465 / 16,376 MiB at 2–5% utilization. Compute rows attributed
5,080 MiB and 606 MiB to shared Ollama runners and 1,380 MiB to the shared
reranker; about 399 MiB was not assigned by that process query. Ollama
`/api/ps` listed `qwen3.5:4b` (4,272,234,167 bytes VRAM) and
`nomic-embed-text:latest` (323,150,151 bytes); Audio Tools' app-scoped
`qwen3:1.7b` did not appear. The model resolver still selects Qwen3 1.7B for
Audio Tools summarize. Whisper and standalone Kokoro were stopped; Sherpa was
selected through an attached server, not the unsupported managed resource.
Audio Tools did not own the observed large GPU allocations. The resident
`qwen3.5:4b` serves shared roles, so changing or unloading it would affect
other consumers and is not justified by this Audio Tools audit.

Host memory was 60 GiB total, 19 GiB available, and 46 GiB swap used; root
disk was 89% used with 196 GiB available. Before the stream, the attached Sherpa endpoint reported
`streaming_model_loaded:true` while its process was idle at 13,376 KiB RSS
(11,214 KiB PSS). During a fresh 11-second product Connect stream, 10 ms
sampling across 537 observations measured a maximum 314,808 KiB RSS (301,148
KiB anonymous, 13,660 KiB file-backed); after completion it remained at
314,904 KiB RSS. The terminal completed and the cumulative processed sample
reached 176,000. This is consistent with the declared ~310 MB Zipformer
becoming resident on first inference. It is a point-in-time first-inference
measurement, not a 60-minute growth profile or cross-device resource receipt.
No resources were started, stopped, or unloaded for this snapshot.
