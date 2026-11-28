const e=`# Maturation State

Updated 2026-11-24 UTC. This page is the current state of the Audio Tools
maturation program, edited in place so it always describes today. It holds no
dated evidence: scope, stance, policy and acceptance live in
[MATURATION-PROGRAM.md](MATURATION-PROGRAM.md), dated milestones in
[PROGRESS.md](PROGRESS.md), incident-shaped findings in
[PROBLEMS.md](PROBLEMS.md), and the evidence this page grew from before the
2026-11-22 rewrite in
[history/MATURATION-STATE-HISTORY-2026-11-21-to-22.md](history/MATURATION-STATE-HISTORY-2026-11-21-to-22.md).

## Current

Audio Tools provides speech-to-text (batch and streaming), text-to-speech,
summarization, eight audio transforms, speaker verification, a model registry,
and provider routing across local, BYOK, and LPBS tiers, with CLI and UI
surfaces. The destination is the full program in MATURATION-PROGRAM.md: generated
catalogs, runtime paths, model/resource choices, quality evidence, reliability
gates, and product/lab separation all true and verified.

Where things stand: the declaration-driven substrate (provider, operation, and
model catalogs with conformance gates) is in place and the low-footprint
defaults are live. The remaining distance is qualification evidence (quality,
latency, device matrix), full duplex, and release provenance for native
resources. The acceptance board reads \`acceptance=unknown\`; nothing here is a
readiness claim.

Read model choices from the owners, not from this page:

\`\`\`bash
audio-tools model resolve --capability stt
audio-tools model resolve --capability stt --engine sherpa-streaming --language en
audio-tools stt stream-config
audio-tools stt engines
audio-tools model resolve --capability summarize
resource-ollama policy resolve --role summarize.audio_tools
resource-ollama policy resolve --role summarize.default
audio-tools model resolve --capability tts
audio-tools health show
\`\`\`

An unscoped \`stt\` resolve reports the generic multilingual batch default; pass
\`--engine\` to resolve that streaming engine's declared model default.
\`stt stream-config\` reports the selected streaming engine; \`stt engines\` shows
availability and selection reasons. The
\`summarize\` resolver reports the Audio Tools \`summarize.audio_tools\` role, while
\`summarize.default\` is the generic shared role. The TTS resolver reports its
catalog default and any conditional product route. \`health show\` reports
provider liveness.

Model defaults are efficiency choices made from the declarations and the
shared host policy; they change when those change, and the resolve commands are
the only current answer.

## Needs you

- **GPT-Live key.** The canonical address is \`vrooli/openai-gpt-live\` /
  \`api-key\`; live \`vrooli credentials status --identity
  vrooli/openai-gpt-live --field api-key --json\` reports \`configured=false\`,
  \`provider=encrypted-file\`, and \`provider_state=available\`. The credential
  doctor reports host-bound TPM2 storage writable on this host. Enter it
  with \`vrooli credentials provision --identity vrooli/openai-gpt-live --field
  api-key\` (secure prompt or stdin, never argv). Provisioning does not mount a
  session route; no key exists and no OpenAI call has been made.
- **Authenticated browser-soak admission.** Test Genie run
  \`20261123-082346-ccbab5b7\` stopped before browser execution because its
  request lacked the Agent Manager identity, BAS browser grant, and Test Genie
  lease receipts. The identity broker currently returns
  \`identity_refresh_unavailable\`; this session has no supported Agent
  Manager/BAS integration to launch the owner-bound run that supplies them.
  A fresh Chrome attachment attempt also failed because this desktop session
  has no browser network policy (\`Browser attachment requires a network
  policy\`). Do not mint or forward receipts manually. See the durable result
  and next route in [PROGRESS.md](PROGRESS.md) and [PROBLEMS.md](PROBLEMS.md).
- **Maturity commission launch.** Last recorded: the Research Desk
  authorization to retire the prepared commission and launch the current one
  was still pending. Not re-checked in this rewrite.
- **Sherpa release owner.** Version and checksum pins in
  \`resources/sherpa-onnx/resource.json\` now match two repeatable unsigned
  Linux/amd64 stages (\`1.13.2-vrooli.6\`, tree digest
  \`96f3606d500423e7213e4f7494368ae0527d49fba21ac4074e1172720096f2d5\`).
  The candidate remains acquisition-ineligible and unsupported: signing,
  publication, and managed install qualification still need the release
  owner's publication URL and provenance.
- **Credential-authority leases.** Last recorded: three expired-heartbeat
  install leases attached to a live SDA process, cleanup unverified. Use the
  governed dependency flow; do not mint a service credential locally.

## Surface scoreboard

State vocabulary: \`live\` works on this host today; \`source\` implemented and
tested but not exercised end to end; \`open\` not built or not qualified.

| Surface | Target | State | Next gate |
|---|---|---|---|
| Declarations (providers, operations, models) | Adding a provider or model is data plus at most one adapter, with generated projections and drift gates | source | Model memory now has a dedicated \`MODEL_MEMORY\` evidence kind, strict owner parser, canonical model/runtime/artifact join, and promotion resolver. Live Linux/amd64 CPU receipts exist for Whisper Small and Sherpa Zipformer against their declared 2 GiB RAM budgets; they remain profile-specific and are not referenced as promotion facets. Platform and license receipt owners are still missing, and quality/latency plus the remaining profile-specific memory evidence still block new-default promotion. Four capability defaults and the existing Sherpa engine-specific route retain exact legacy exceptions. Provider health strategy and aggregate membership are declared; generic HTTP providers now declare constrained base-URL fields and readiness/liveness paths in the catalog. Specialized Whisper/Ollama/request-scoped/transcode behavior remains adapter code |
| Batch STT | Small default model, explicit per-request model on the unary route | live | Quality evidence remains open. The Whisper service is not running until requested. Rumble has an opt-in unary route (\`X-Audio-Model-ID\`) through its declaration-selected vLLM audio-chat adapter; no endpoint or managed runtime is configured on this host, and Rumble remains an unqualified candidate |
| Streaming STT | Low-footprint native streaming | live (attached) | Sherpa is selected by the persisted engine pin and attached at \`127.0.0.1:8881\`. A fresh native server + WebSocket test on the bundled 5-word clean-speech fixture returned exact lexical content (WER 0.00); warm inference was 147 ms for 1.955 s audio (RTF 13.29x), with a 3.04 s model load. A new 11-second JFK fixture run through the selected product Connect API emitted 21 visible partials, two final segments, and a terminal completion after the cumulative durable acknowledgment reached sequence 110 and sample 176,000; the final segment also ended at sample 176,000. The ending survived, but the opening and a phrase boundary were misrecognized. This file-fed CLI route check is not paced browser latency or quality qualification; the fixture remains a counterexample. A fresh 30-attempt headless Chrome fake-microphone cohort through Dictation Studio completed 30/30 turns with final drain, at least two rendered revisions, and captured/committed sample equality; first-partial p50/p95/max was 1,098.3/1,161.2/1,195.9 ms, so the adopted 1,000 ms p95 target still fails. The post-onset audio p95 was 1,117 ms and excess wall latency p95 was 55.5 ms; stop-to-final-tail p95 was 165.8 ms. This is synthetic-media latency and terminal cursor evidence only, not transcript-quality or physical-device qualification; see \`coverage/browser-paced-sherpa-30-20261123.json\`. The attached process is unmanaged; after first product inference it remained near 315 MiB RSS, consistent with the declared ~310 MB model, while idle RSS before inference was only 13 MiB. The resource manager still reports unsupported on Linux until signed publication; native word intervals are absent. Connect bidi sessions now reset a 30 s idle timer on audio and persist input_idle_timeout when the stream goes quiet; focused handler race tests pass |
| TTS | Qualified default plus a conditional product auto route | live | Kale Nano is the auto-route candidate when healthy and remains unqualified; the catalog default stays Kokoro until human quality review. Review-only four-voice packets exist at \`~/.local/share/vrooli/audio-tools/qualification/kale-nano-review-20261122/\` and \`.../kale-nano-review-20261123/\`; no human review or paired Kokoro comparison is recorded. A current product TTS request selected \`kale-tts-1-nano\` through the local auto route. Five unique warm synthesis calls measured 61.8–94.8 ms provider latency (median 68.9 ms); a sampled WAV was valid 24 kHz mono, 2.55 s, and non-silent. The Kale process reached about 90 MiB RSS while GPU use remained unchanged. This confirms the small local route works, not naturalness or multi-voice acceptance. The managed CPython bundle now uses the upstream stripped distribution: installed tree is 257 MiB, down from 498 MiB (about 48%), with artifact SHA-256 \`cf38c6dc79ff578c40454b79c12b35356f6c4049de43b013b525d3fb95facb63\`. After reacquire/start, the live product route selected Kale for a cold 2.55-second WAV in 1,073.5 ms and a distinct warm 3.05-second WAV in 113.8 ms. The 20-request, ten-voice integrity sweep passed (median 75.2 ms, max 614.13 ms; no empty, near-silent, or clipped output); its receipt is \`~/.local/share/vrooli/audio-tools/qualification/kale-nano-stripped-runtime-20261123.json\`. RSS peaked near 225 MiB during this run against the manifest's 512 MiB RAM budget. These checks cover packaging, lifecycle, routing, and signal integrity; naturalness remains unmeasured, so no default promotion is claimed. |
| Summarize | Audio Tools uses the app-scoped \`summarize.audio_tools\` role (Qwen3 1.7B); generic \`summarize.default\` remains Qwen3.5 4B for shared consumers | live | Latency and fact-retention qualification; 1.7B evidence is limited to a short report (8/8 facts) and one 300-word report (14/22). The 4B model shares its runner with \`chat.small\` and \`classify.routing\`; resident memory is not attributable to Audio Tools |
| Audio transforms and speaker verification | Eight transforms reachable from CLI and UI; compact speaker embeddings | live / source | Speaker threshold needs live calibration; speaker-verification Python tests need an owner route |
| Corpus and timestamp quality | Human-reviewed held-out references, paired clean/noisy cohorts, named-model quality score, and timed references for drift scoring | open | The lab corpus has 1,140 VCTK 0.92 CC BY 4.0 clips: 360 held-out clean utterances with paired 20/10 dB variants plus 60 speaker-disjoint dev utterances. All 1,140 clips are now bound to the source manifest by verified audio and transcript digests (360 complete three-variant groups, 60 dev singletons); zero clips are marked reviewed. The lab UI is reachable at \`/dictation-studio\` and proxies corpus operations on its isolated instance, while the product API does not expose corpus routes. Group review now requires all source variants and writes human provenance atomically; the next step is for a reviewer to listen to each variant and submit corrections through the lab UI. Its metadata-only manifest is \`~/.local/share/vrooli/audio-tools/qualification/voice-pilot-en-v1-manifest.json\`; the 360 held-out clean clips span six speakers and 3,782 whitespace-counted words. The persisted Sherpa Zipformer clean-cohort run \`296c84f3-5204-4d30-b698-5cb7a6dd85f0\` measured 4.8464% WER over 3,776 scorer-normalized reference words, with zero clip errors and a passing stream-safety gate; no latency/RTF was measured, and it remains non-promotable pending VCTK reference audit, noisy cohorts, interval accounting, recovery/fault profiles, and device evidence. The same 21-minute AMI ES2002a audio completes through the corrected lab CLI/Sherpa path in 1m58.9s (10.7x RTF), with 115 segments and all 1,242 sequences processed; no transcript quality or timestamps were scored. Three distinct >10-minute AMI review packets (ES2002a, IS1001a, ES2008a) are now staged outside Git with source audio, annotation XMLs, aligner-only draft transcript/timings, attribution, SHA-256 sums, and blank review ledgers. ES2008a includes three words without timestamps that require correction. All remain ineligible until an attributable human reviews each full transcript and boundary, resolves disputes, and completes the ledgers. |
| Runtime/device qualification | Browser, OS, device, quality, latency, recovery, and model-bound memory cells with real receipts | open | The fresh governed board read (\`prog_7db272b6-517d-4eda-8219-196f0e0b4e8f\`, 2026-11-24 14:55Z) remains \`unknown\`: 6/18 required rows resolve. Host portability remains failed because the producer has not recorded pinned host/browser provenance and a matching visible product-path receipt; corpus quality remains unavailable because human-reference provenance is unverified. Device qualification is still 0/27 D1-D6 product/device cells. D1 CPU Whisper and D2 constrained Sherpa inference and memory observations are recorded in [PROGRESS.md](PROGRESS.md), but these isolated Linux/amd64 measurements do not qualify managed product/browser or cross-platform execution. The latest 30-attempt paced product cohort has first-visible p95 1,122.1 ms against the 1,000 ms target; it reconciles all captured/committed samples and sent/processed cursors, but uses synthetic media and does not qualify device or transcript quality. The five-cycle authenticated browser session remains unrun because its supported admission receipts are unavailable. |
| Full duplex | Provider-backed route with consent, privacy, cost limits, interruption, session receipts | source | The authenticated WebSocket transport, source-bound revocable consent checks, LPBS reservation/settlement/recovery, and SQLite recovery journal are implemented. Provider catalog entries now own the consent source and active-session revalidation declaration; every full-duplex provider must declare a nonblank source and revalidation. Production bootstrap wires credential-authority lookup for the declared LPBS principal, runtime discovery for Scenario Authenticator and AI Gateway, and an explicitly configured \`AUDIO_GPT_LIVE_MAX_SESSION_BUDGET_MICROS\`; LPBS URL and positive budget are required. Admission fails closed without a verified human, matching receipt, verifier, or active revalidator. Composition remains fail-closed because the provider/model pair is still planned/candidate, so no production route or hosted request is active. The OpenAI key remains unconfigured and is provisioned only through \`vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key\`. |
| Voice cloning | A consented, pinned, quality-measured route | open | Sopro V2 Turbo is declared in the model/provider catalogs as a candidate: upstream lists 120M parameters, Apache-2.0, 4 languages, 5–20 s reference cloning, and CPU/browser paths; pinned snapshot \`c12d0dc7c4a33b3cb6ea3d11ceb70bc60c7c6312\` is 952 MB including Safetensors and alternate GGUF weights ([upstream](https://github.com/samuel-vitorino/sopro), [model card](https://huggingface.co/samuel-vitorino/sopro-v2-turbo/tree/c12d0dc7c4a33b3cb6ea3d11ceb70bc60c7c6312)). Current artifacts require \`sopro>=2.2.0\`. It has no Audio Tools resource, adapter, or runnable path; the implementation boundaries are recorded in [SOPRO-VOICE-CLONING-CONTRACT.md](SOPRO-VOICE-CLONING-CONTRACT.md). Runtime dependency licensing, source-bound authorization, transient reference handling, bounded deletion, held-out quality and resource qualification remain open. Qwen3-TTS 0.6B Base remains a 2.52 GB candidate without a Base adapter; Reverb is ~307 MB int8 but CC BY-NC 4.0; OpenVoice V2 lacks a retrievable pinned MeloTTS/checkpoint bundle. No candidate is default-eligible; add a separate consented cloning operation and prove bounded deletion, refusal/revocation, quality, and resource use before routing. |
| Product/lab boundary | Product excludes corpus, experiment, soak, and validation routes | live, profile-aware | Product remains running at its existing build \`sha256:ffc488f141a31ba74ad339a4bb0e4a09bbe647c1851ffe51a82df63b0b853df7\` on API 19630. A supported lab-scoped restart keeps the API-only instance at API 15373, build \`sha256:b3a4787555c23ec5e50d195421796c2498d316600da6c77fac60627cc001646e\`. After the guarded root install and \`vrooli runtime supervisor install --user\`, the supervisor runs build identity \`9db19196767db25a692f668a488e8d7ba0f000505a43cd0b9035ab97385995ca\`, and the lab status is healthy with no \`UI_PORT\` error. The service safeguard reports the process image current and the systemd unit active/enabled/accepted; 91 instances remain supervised and zero are unverified. Lab health/evidence are available; product health returns HTTP 200 and the lab-only evidence route returns HTTP 404 on product. |
| Debt and reliability | Races, portability, duplication, and complexity reduced with broad verification | in progress | Refreshed full API and CLI race suites/vet, UI type-check/lint and 1,361 tests across 143 files, production build, catalog checks, and Windows/macOS builds pass. UI dependency analyzer validation is clean after deduplicating \`tailwind-merge\` and upgrading Vite 6.4.4 and Vitest 4.1.12. Main JS is 643.35 kB / 194.31 kB gzip, down 17.89 kB / 6.04 kB gzip from the prior 661.24 / 200.35 bundle; Vite still warns above 500 kB. Security Health reports zero blocking errors and no Vite/Vitest advisories, with 284 warning/informational findings remaining (L1). Duplex admission now requires reservation-capable spend holds; LPBS settlement overflow guards have regression coverage. Direct \`gocyclo\` over changed production code in report, soak, STT strategy, and duplex has no results above 15. The light \`tidiness-manager scan audio-tools --type light\` attempt on 2026-11-22 failed with API 500, so it provides no current issue count; direct source complexity and line counts are recorded here instead. The scan does not certify product quality or device/browser qualification. Earlier full Test Genie run \`20261122-072139-31378112\` passed 23 phases, failed 7, and skipped 1. Experience-contract's wrong phase identity was reported as \`knw-1795334159222034566\`; the scenario-qa heartbeat is globally paused. Workflow execution still needs an authenticated BAS account. |

Verified on 2026-11-24 UTC: \`audio-tools validation check-freshness --json\`
reports fresh requirements. \`audio-tools validation device-matrix --json\`
reports 0 of 27 qualified product/device cells. A fresh lab acceptance join
(\`audio-tools:acceptance-join:20261124T001858.065201342Z:a7a500a21ab7d883\`)
passed receipt validation but is incomplete: six source receipts, twelve
missing required rows, and \`acceptance_complete=false\`. Overall acceptance
remains unknown. The previous labelled local simulation resolves explicit
routing, settlement, private
lifecycle, adapter maintainability, and acceptance-integrity. A local
speaker-policy probe uses a temporary unregistered profile, reports degraded
fallback on batch and native streaming paths, verifies both fail closed when
fallback is disabled, and restores the lab configuration; it does not inspect
operator-enrolled profiles. Whisper Small started on demand for the batch check
and was stopped after \`audio-tools stt engine-impact\` reported no other
consumers; GPU usage returned to its 7,463 MiB baseline. Sherpa remains the
selected streaming engine. See the latest dated row in [PROGRESS.md](PROGRESS.md).
These receipts do not qualify paced browser latency, held-out quality, or full
portable-voice acceptance.

## Open problems

Detail and history live in [PROBLEMS.md](PROBLEMS.md); the headings below are
the entries to start from.

- **Quality gold is missing.** "2026-11-20 — Hypothesis timings retained;
  qualification still needs gold". A quality claim for any compact model needs
  held-out references and a measured run.
- **Streaming decoder and finalization.** "2026-11-20 — Sherpa attach, compact
  summarize, and a decoder gap" and "2026-11-20 — Kroko finalization tail and
  sherpa-hosted Kokoro route". Kroko stays on-demand and experimental.
- **Native Sherpa release provenance.** "2026-11-20 — Current host model
  residency is outside the Audio Tools path" and "2026-08-17 — Native
  audio-resource migration remains release-evidence-blocked". The current
  \`1.13.2-vrooli.6\` digest matches two repeatable unsigned stages; signed
  publication and managed install evidence remain open.
- **Shared-host footprint.** "2026-11-20 — Host resource attribution and
  Whisper-small product check" and "2026-11-20 — The machine footprint was an
  accident, not a decision". Shared Ollama and reranker processes dominate GPU
  use and are not Audio Tools' to stop.
- **Latency and long-turn reliability backlog.** "2026-09-16 — Kyutai
  product-path latency is backend-bound after capture", "2026-07-12 — Kyutai
  deterministic long-form replay remains an unresolved flow-control blocker",
  and "2026-09-15 — Repeated microphone sessions can require refresh or
  restart".
- **Portable voice acceptance.** "2026-09-09 — Portable voice contract
  aligned; execution and qualification remain open".
- **Billing journey and browser cycles.** LPBS now has an offline loopback for
  a simulated $10 Audio-only checkout, app-scoped grants, a bundle upgrade,
  fixture invoice amount, payment failure/recovery, and period-end cancellation
  through terminal deletion. This does not prove Stripe-calculated proration
  or settlement, and no Audio Tools SKU is configured in the owner catalog,
  landing page, or Web Console funnel. The five-cycle BAS case still needs an
  authenticated account before it can run.

## Next unit

In order of what unblocks the most:

1. **Qualification evidence.** Human-reviewed timed references and a held-out
   cohort, then paired quality/latency runs for the current compact routes
   (including Kale's valid but not yet quality-qualified product route);
   run the D1 accelerator cell when a serviceable engine exists; execute the
   five-cycle real-browser session protocol.
2. **Full duplex route.** Qualify the provider/model and confirm trustworthy
   usage evidence; then provision the declared LPBS account-credit service
   principal, configure the LPBS URL and explicit maximum budget, and exercise a
   funded, owner-authenticated session with settlement verification.
3. **Native Sherpa release.** Release-owner provenance and publication inputs,
   then signed publication and an install lifecycle for the now-reproducible
   artifact.
4. **Candidate models.** Voice cloning needs an artifact and license
   inventory, a consented speaker cohort, and an isolated pinned service
   before any trial; Tiny Voca needs a managed runtime and adapter before a
   paired benchmark.
5. **Gate health.** Preserve the passing explicit unit phase, then rerun broader
   Test Genie phases once the shared tree and Test Genie runner are stable;
   close the unrelated failed phases it reports.

## Keeping this page current

Edit this page in place: change a row's state, replace a stale sentence, delete
what is no longer true. Do not add dated sections. Put dated evidence (commands,
run IDs, timings, receipts) in [PROGRESS.md](PROGRESS.md), incident findings in
[PROBLEMS.md](PROBLEMS.md), and long write-ups under \`history/\`. Cite a resolve
command instead of copying a model tag, size, or digest.
`;export{e as default};
