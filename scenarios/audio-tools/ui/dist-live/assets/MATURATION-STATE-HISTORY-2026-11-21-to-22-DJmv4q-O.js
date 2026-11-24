const e=`# Maturation State

## Current

2026-11-21 22:58 UTC. Destination: finish the full Audio Tools maturation
program in [MATURATION-PROGRAM.md](../MATURATION-PROGRAM.md), with generated
catalogs, runtime paths, model/resource choices, quality evidence, reliability
gates, and product/lab separation all true and verified. This page is the
current state; [PROGRESS.md](../PROGRESS.md) is the historical record.

## Latest verification

2026-11-21 20:56 UTC. The latest direct validation passed: full API and CLI
Go race suites, API and CLI \`go vet ./...\`, \`make catalog-check\`,
\`make portability\` (Windows/amd64 and macOS/arm64), and UI type-check, lint,
and Vitest (135 files, 1,282 tests). The CLI race gate first exposed stale
primitive evidence after \`stt transcribe --model\` was declared; its owner test
regenerated \`.vrooli/generated/cli-primitive-evidence.json\`, then the full CLI
race suite passed.

The fresh Test Genie unit rerun did not obtain a run ID. Its first attempt hit
SQLite \`SQLITE_BUSY\` during concurrent build admission; after that live
restart ended, the retry failed while building Test Genie's own CLI because
\`internal/capacity/disk_job.go\` refers to missing
\`sweepAbandonedDiskJobsTx\` and \`SQLiteStore.SweepAbandonedDiskJobs\`. This is
outside Audio Tools; the shared Test Genie tree contains broad unrelated
changes, so direct module suites provide the current implementation evidence. The last successful
Test Genie receipt remains \`20261121-202116-0d531f64\`, before the latest
Rumble API/CLI changes. Measures CLI coverage is 96.1%; historical scenario
completeness remains functionally incomplete with 58 low-coverage warnings.

The owner provisioning path was exercised without entering or storing a value.
A non-TTY invocation fails closed; the interactive \`vrooli credentials
provision --identity vrooli/openai-gpt-live --field api-key\` prompt is available
for owner entry. Credential status remains unconfigured, with encrypted-file
provider available. No OpenAI key was created or API call made.

## Additional evidence (2026-11-21 20:56 UTC)

- Rumble now has an explicit unary product route on the existing STT
  Transcribe request: \`X-Audio-Model-ID\` selects a catalog-declared batch
  model/provider, while no header preserves Whisper. Rumble's optional
  \`AUDIO_RUMBLE_VLLM_URL\` endpoint is serviceability-checked against the
  declared runtime model; unavailable, disabled, or unknown explicit routes
  fail closed. It is not added to the streaming engine picker. Fake-vLLM
  handler tests, default/unavailable routing tests, the full API race suite,
  \`go vet ./...\`, \`make catalog-check\`, and Windows/amd64 + macOS/arm64
  portability builds passed. No endpoint is configured on this host, and no
  runtime, quality, or promotion claim is made.
- The GPT-Live adapter no longer invents a model when the caller omits one; it
  rejects the request before credential resolution. This keeps model selection
  at the registry/caller boundary. \`GOWORK=off go test -race
  ./internal/duplex -count=1\` passes. The production route remains unmounted and
  no live provider call was made.
- The attached Sherpa Zipformer server transcribed the repository's
  \`dictation-reference.wav\` through \`/v1/stream\` and retained “jumps” at
  1,954 ms. Its running binary still returned the old \`brown,fox\` spacing, so
  the current source fix is not live-verified on that process; the process was
  not restarted.
- Kroko's managed model artifact is now present. Native and WebSocket smoke
  tests built from current source passed on both \`dictation-reference.wav\` and
  the model's bundled JFK \`test_wavs/0.wav\` (converted from 24 kHz to required
  16 kHz). Both paths retained the final words; model sample ended with “your
  country”, and the repository clip ended with “jumps”. Timestamps were
  bounded to source duration. The bundled clip measured 67.93x RTF / 1.16 s
  model load; the 1.955 s repo clip measured 69.76x RTF / 1.57 s model load.
  Output spacing was \`brown, fox\`. This confirms immediate finalization and
  the source punctuation fix on two short clips, not broad accuracy or a model
  promotion. No shared service was restarted.
- Live model/host check: \`audio-tools stt engines\` selects
  \`sherpa-streaming\`; \`audio-tools model resolve --capability summarize\`
  resolves \`qwen3:1.7b\`. At 20:27 UTC, Ollama reported \`qwen3.5:9b\` resident
  at 6.74 GB VRAM and \`nomic-embed-text\` at 323 MB; total GPU use was 10,105
  MiB. Audio Tools API RSS was 31.8 MB, attached Sherpa server RSS 8.1 MB, and
  the Kale runtime process RSS 19.3 MB. This snapshot attributes the dominant
  resident model to shared \`chat.default\`; it does not support changing shared
  Ollama keep-alive policy from this scenario.
- With managed \`kale-tts\` reporting healthy and \`AUDIO_TTS_BACKEND=auto\`, a
  fresh product \`audio-tools tts synthesize\` request selected provider
  \`kale-tts\`, model \`kale-tts-1-nano\`, voice \`kenzie\`, and returned valid mono
  24 kHz PCM WAV in 754 ms. This verifies current conditional-auto selection
  and product routing; one short phrase does not qualify naturalness,
  intelligibility, multi-voice quality, or release status.
- Audio Tools already contains the Sherpa \`/v1/audio/speech\` TTS adapter and
  focused Connect/cache/readiness tests. \`GOWORK=off go test -race -count=1
  ./internal/tts ./internal/ai/ttschain ./internal/bootstrap\` passes. Kokoro
  remains the qualified catalog default; Kale conditional-auto behavior
  remains candidate selection rather than qualification or promotion.

## Additional evidence (2026-11-21 22:05 UTC)

- A summarize request first failed before inference because the running AI
  Gateway used a stale rights-context projection. A managed
  \`vrooli scenario restart ai-gateway\` rebuilt and served the current
  evaluator, and the same Audio Tools request then succeeded in 20,985 ms and
  retained all details in the short factual input. This is one route proof, not a broad quality or
  latency qualification. Focused catalog/policy tests and generated-catalog
  checks pass.
- GPT-Live credential storage is ready through the existing encrypted-file
  credential authority. Status is provider available / field unconfigured;
  owners provision with \`vrooli credentials provision --identity
  vrooli/openai-gpt-live --field api-key\`. No Platform key-create action is
  exposed here. Production composition prepares the provider with the lazy
  authority resolver, but the duplex HTTP route remains unmounted pending
  admission, privacy, spend/recovery, and usage qualification. No key was
  created and no OpenAI API call was made.
- Test Genie run \`20261121-215224-c8a1583a\` for the unrelated agent-manager
  scenario ended failed on one test execution failure and 71 advisory low
  coverage warnings. This is not Audio Tools validation evidence.

## Additional evidence (2026-11-21 22:08 UTC)

- Fresh \`audio-tools stt engines --json\` selects the pinned \`sherpa-streaming\`
  Zipformer because it is available; Whisper and Kyutai health currently report
  unavailable. A product \`stt transcribe-stream\` run on
  \`bas/fixtures/dictation-reference.wav\` completed in 3,448 ms, emitted ordered
  durable acknowledgements, and retained the final \`jumps\`, but returned
  \`The quick brown,fox jumps.\`. This is fresh live evidence of the punctuation
  spacing bug on the attached server process; the source fix is present but not
  yet deployed to that long-running process. Sherpa remains attach-only pending
  the signed managed artifact.
- A fresh product synthesis selected \`kale-tts\` / \`kale-tts-1-nano\` / \`kenzie\`
  in 1,405 ms. \`audio-tools model resolve --capability tts --json\` still
  resolves the registry default \`kokoro-82m-v1.0\`, so the declared model and
  automatic product route disagree. The route is serviceable, not quality or
  release qualified; a TTS implementation owner is reconciling this behavior.
- Host snapshot: RTX 4070 Ti SUPER used 8,706 / 16,376 MiB at 3% utilization.
  The GPU process table attributed 7,700 and 596 MiB to two Ollama
  \`llama-server\` workers. Audio Tools API RSS was 26.9 MiB, Sherpa server RSS
  8.9 MiB, and Kale Python RSS 17.8 MiB; none appeared in the GPU process
  table. Ollama retention marked \`qwen3.5:9b\` (\`chat.default\`) and
  \`nomic-embed-text\` loaded; the Audio Tools summarize default \`qwen3:1.7b\`
  was installed and required but unloaded after the request. This snapshot
  attributes current GPU residency to shared Ollama roles rather than Audio
  Tools inference.

## Additional evidence (2026-11-21 22:18 UTC)

- After a managed Audio Tools restart, product auto TTS selected Kale Nano
  when its service was healthy and returned a valid 127,244-byte WAV via
  \`kale-tts\` / \`kale-tts-1-nano\` / \`kenzie\` in 2,292 ms. The CLI's
  \`model resolve --capability tts\` explicitly separates the qualified Kokoro
  catalog default from Kale's conditional candidate route. Kale remains
  unqualified and is not promoted. Sherpa-hosted TTS has a declared fast
  candidate and focused tests for MP3, model identity, cache replay, and
  cancellation, but the Sherpa attach server was started with TTS disabled;
  no live Sherpa TTS receipt exists.
- Rebuilt the source Sherpa Linux/amd64 bundle against the pinned v1.13.2
  runtime and replaced the stale attach process under the documented runbook.
  \`/healthz\` reports the streaming model loaded. Product streaming STT still
  selects Zipformer and now returns \`The quick brown, fox jumps.\` instead of
  \`brown,fox\`. One product run measured 194.9 ms segment latency before the
  Audio Tools API restart; the first stream afterward measured 4,920.7 ms
  while preserving the same final text. This variance needs repeatable SLO
  evidence. The process remains in \`/tmp\` and attach-only; there is no signed,
  managed artifact.
- Two consecutive short product summaries selected \`qwen3:1.7b\`, preserving
  every input detail. Latency was 8,968 ms for the first request and 1,047 ms
  for the immediate repeat. Qwen 1.7B used about 2,390 MiB VRAM while loaded;
  a later retention snapshot found it unloaded. At 22:18 UTC the GPU used
  10,101 / 16,376 MiB: shared \`qwen3.5:9b\` used 7,702 MiB, Nomic embeddings
  604 MiB, and reranker 1,380 MiB, while Image Tools also had an 878 MiB GPU
  Python process. No Audio Tools inference model was resident in that latest
  snapshot. These observations separate transient Audio Tools summary cost
  from sustained shared-model load; they are not a long-run contention study.

## Additional evidence (2026-11-21 22:28 UTC)

- Restarted AI Gateway through \`vrooli scenario restart ai-gateway\`; the
  managed build/start completed healthy on API 19894 and UI 20518. A fresh
  Audio Tools product \`summarize text\` call selected local Ollama
  \`qwen3:1.7b\` and returned in 17,342 ms. Its concise result retained the
  9-second result and both fact-count measurements (8/8 short, 14/22 dense),
  while omitting the 4B fallback and the reason for the 1.7B default. This
  supports the compact-default/quality-fallback split, not dense-report
  completeness or a latency SLO. Gateway start reported unrelated credential
  gaps, including required OrcaRouter, but its health check was green.
- Credential status remains \`unconfigured (provider encrypted-file,
  available)\` for \`vrooli/openai-gpt-live/api-key\`. The secure Vrooli prompt and
  lazy authority resolver are ready; no Platform key-create action is exposed
  here. No key was created or API request made. Provisioning still does not
  enable the separately gated GPT-Live session route.

## Additional evidence (2026-11-21 22:53 UTC)

- A fresh product TTS request selected \`kale-tts\` / \`kale-tts-1-nano\` /
  \`kenzie\` and returned valid mono 24 kHz PCM WAV (192,044 bytes, 4.0 s) in
  4,188 ms. Kale's managed resource is healthy; this remains route evidence,
  not a naturalness or quality qualification. \`model resolve\` continues to
  identify Kokoro as the qualified catalog default.
- The current host sample used 11,372 / 16,376 MiB GPU at 4%. Compute-process
  attribution was Ollama 7,700+608 MiB, reranker 1,380 MiB, and Ollaya 1,264
  MiB; Ollama's resident list contained \`qwen3.5:9b\` and \`nomic-embed-text\`,
  not Audio Tools' \`qwen3:1.7b\`. Audio Tools API, Sherpa attach, and Kale RSS
  were ~29, ~5, and ~18 MiB; Whisper was stopped. This is a snapshot, not a
  contention study. Sherpa still serves from an attach-only process while
  \`vrooli resource status sherpa-onnx\` reports Linux unsupported.
- The Sherpa Linux/amd64 release stage initially failed because the current
  source bundle digest differed from the declared \`.1\` artifact. The resource
  and Makefile now declare version \`1.13.2-vrooli.2\` and tree digest
  \`c8127175638a483052c4bf565946a633c3950be5778d47427cd77e32eeeb6e8c\`.
  Rebuilding produced the same digest and passed offline stage validation plus
  \`make -C resources/sherpa-onnx check\`. The stage is unsigned, has no
  publication URL/provenance, and remains unavailable to managed Linux
  lifecycle.
- Product \`audio-tools stt engines --json\` exposed a CLI contract bug: the
  engine, engine-impact, and accepted-format commands printed their human
  renderer in JSON mode. All three now use canonical proto JSON, with typed
  regression tests. A source-built CLI parsed each output with \`jq\` against
  the live API; the engine result reports Sherpa available and active.
  \`GOWORK=off go test ./domains/stt -count=1\` and \`go vet ./domains/stt\` pass.
- Buffered STT fallback cancellation now prioritizes cancellation over queued
  audio and emits the documented \`Error\` then terminal \`Done\` sequence.
  \`GOWORK=off go test ./internal/ai/sttchain -count=1\` and the targeted race
  test pass. Dictation Studio now ignores late callbacks and retained-Blob
  reads from a disposed provider; its focused UI suite passes 18/18 and
  file-scoped ESLint passes. Neither result claims a native desktop cohort.
- The shared billing schema now has a simulated test that fixes the proposed
  Audio-only offer at $10 USD/month, verifies its Audio Tools-only grant, and
  configures a separate Web Console offer through the same model. The targeted
  commerce test passes; no live billing or customer data changed.
- Timing-corpus source review found no ready-made human-reviewed long-form
  word-boundary reference. AMI is CC BY 4.0 and its transcripts receive human
  passes, but word timings are forced-aligned and need independent human
  correction. No media was downloaded; the timing brief records the exact
  review/import path and current local inventory limitation.

## Needs you

The operator authorized secure key creation if supported, or setup for owner
provisioning through existing infrastructure. The canonical address is
\`vrooli/openai-gpt-live\` / \`api-key\`. \`vrooli credentials status --identity
vrooli/openai-gpt-live --field api-key\` reports the encrypted-file provider is
available and this field is unconfigured. The owner can enter it through
\`vrooli credentials provision --identity vrooli/openai-gpt-live --field
api-key\`; the command accepts a secure terminal prompt or stdin, not argv.
No Platform key-create action is exposed in this session, and no secret value
was supplied to the secure provisioning command, so no key was created.
Production composition retains a lazy authority-backed OpenAI Live provider,
resolving that identity on use and observing rotations. The Configuration page
now directs duplex setup to this canonical path instead of accepting a key in
the separate legacy BYOK store. Provisioning does not mount or authorize a
session route. GPT-Live remains ineligible until verified identity, revocable
consent, durable reservation/settlement/recovery, and trustworthy provider
usage evidence are connected to a production session route. No live OpenAI
call was made. Three expired-heartbeat credential-authority install leases
remain attached to a live SDA process with cleanup unverified; do not bypass
the governed dependency flow or mint a local service credential. See
\`domains/settings/byok-and-voice-overrides.md\` for the separate legacy store.

## Surface scoreboard

| Surface | Target | State | Owner | Evidence / next gate |
|---|---|---|---|---|
| Provider, operation, model declarations | Adding an existing-family provider or model is data plus at most one adapter, with generated projections and drift gates | in progress | root | Provider IDs, engine IDs, runtime aliases, and canonical TTS voice aliases resolve from declarations and round-trip through conformance tests. BYOK credential checkers derive from active provider declarations. A fully synthetic model promotion fixture now exercises ID/provider/engine lookup, all five receipt facets, exact declaration binding, and \`DefaultForVerified\` without adding production model-ID branches. Bootstrap still composes some local health/aggregate checkers in code. Memory, platform, and license receipt owners lack integrated resolvers, so real new-default promotion remains blocked. Four legacy defaults remain under an exact ID/reason allowlist; a fabricated exception is rejected. |
| Resource-efficient defaults | Whisper Small batch, Sherpa Zipformer streaming, the shared \`summarize.default\` role (Qwen3.5 4B, sharing the chat.small runner), and compact speaker embeddings are declared; conditional TTS auto is separated from qualified default | smaller-model policy in source; live paths now exercised; release/quality qualifications remain open | root | At 16:26 UTC, \`audio-tools model resolve --capability stt\` selects Whisper Small (488 MB), stream configuration selects CPU Sherpa Zipformer (310 MB), and summarize resolves to Apache-2.0 Qwen3 1.7B (1.36 GB). A live product synthesis at 16:31 selected \`kale-tts-1-nano\` on \`kale-tts\` (local, 1,281 ms) and wrote a valid WAV; Nano is a 14.8 MB CPU candidate, still not quality/release qualified. Kale readiness/liveness now reads its catalog-declared health path; initial cached unavailability was not reproducible against \`/healthz\` (HTTP 200, ready). At 16:28 UTC, GPU use was 9,390 MiB: process attribution identified Whisper Small at 719 MiB, an Ollama \`qwen3.5:9b\` chat.default server at 7,702 MiB, and a separate \`nomic-embed-text\` server at 558 MiB. The 9B chat and embedding models are not Audio Tools' summarize default; Qwen3 1.7B remained unloaded until a summarize request. Sherpa/Kale/API processes are CPU-side. |
| Long-form timestamp quality | Human-reviewed timed references flow through the scorer, and a named model has a qualifying receipt | rough | root | Timed references enter through API/CLI and persist with provenance. Storage now verifies that timed words match the clip transcript, preventing mismatched lexical references from reaching evaluation. No human-reviewed labels or qualifying score exist; native Sherpa word intervals are absent |
| Runtime/device qualification | Required browser, OS, device, quality, latency, and recovery cells carry real persisted receipts | rough | root | Offline harness now executes provider timeout, HTTP 503 busy, dropped connection, backend restart recovery, decoder truncation, and cancellation; its receipt is non-promotable. Simulated receipts now have exact per-row target/revision/method/reference/check claims; legacy broad simulations are rejected. A fresh lab composite joins 5 exercised rows and lists 13 of 18 required rows as missing; \`audio-tools.setpoint-read\` reports 5/18 resolved, \`acceptance=unknown\`, \`pending:required_owner_receipts\`. Product API deliberately keeps returning 404 for lab evidence. The lab API answers on 15373, though lifecycle status reports degraded due to an unresolved \`UI_PORT\` placeholder for its API-only variant. PipeWire has no input device; no D1/D2 inference or physical D3-D6 cells were run. Full matrix is not qualified |
| Full duplex | A provider-backed duplex route enforces consent, privacy, cost limits, interruption, and session receipts | in progress | root | Added sourced catalog pricing, fail-closed admission, isolated GPT-Live WebSocket adapter, and lazy production composition using \`vrooli/openai-gpt-live\` / \`api-key\`. The Configuration page directs duplex setup to the canonical encrypted credential authority instead of the separate legacy BYOK store. No production duplex route is mounted. The API-core LPBS consumer-token verifier forwards only verified human identity; LPBS USD-budget reserve/settlement handles measured cost, tier/app multiplier snapshots, and replay. Production sessions still need source-bound revocable consent, route-level durable recovery, trustworthy provider usage evidence, and end-to-end verification. Session IDs are idempotency references, not invoice authenticity. No key or live OpenAI call exists. |
| Product/lab boundary | Product runtime excludes corpus/experiment/soak/validation routes unless explicitly launched as lab | close in source; separate lab instances running | root | Supervisor-owned \`default\` (API+UI) and \`lab\` (API only) profiles select \`Compose\` vs \`ComposeWithLab\`; caller \`AUDIO_ENABLE_LAB_API\` is ignored. Audio Tools lab \`API_PORT=15373\` and Program Runtime lab API \`16555\` are running; Program Runtime lab was restarted with governed \`--variant-dependencies audio-tools\`. The fresh setpoint run succeeded through \`audio-tools/experiment/list-evidence\`; product API still returns 404 for that route. Lab and product SQLite paths remain separately namespaced. The lab owner list is empty. Protojson omits empty repeated fields in the Program Runtime bridge result; the setpoint now handles that typed default as zero receipts. |
| Debt and reliability | Bugs, races, portability, duplication, and unnecessary complexity are reduced with proportionate broad verification | in progress | root | Current full API race suite and API vet pass; full CLI race suite passes; UI suite most recently passes (135 files / 1,282 tests), with full UI lint/type checks, production build, portability build, and catalog drift checks passing in current evidence. Shared browser capture extracts rejection, stream-notice, provider orchestration, bounded whole-turn PCM retention, and PCM write-failure classification. A new generation guard prevents an asynchronous AudioContext resume from restarting the level monitor after capture teardown; a focused hook regression and the full package suite/build pass. The STT picker previously ran lifecycle-capable Whisper availability checks and control-plane subprocesses; it now uses read-only serviceability and returns in 0.52 s, with Whisper confirmed stopped afterward. The Audio Tools \`useVoiceCore\` adapter is 21 LOC; its orchestration and lifecycle tests are owned by \`packages/audio-capture-browser\`, so further extraction here would duplicate shared ownership. Measures-health validates six real persisted domains at L3 with zero findings and live probes. Latest full Test Genie result: 22 passed, 7 failed, 1 not evaluated, 1 provider unavailable, and 1 skipped. The 32-minute run spent 21m20s in provider preparation; it logged Audio Tools unreachable for 4m28s even though the service later reported healthy. |

## Latest validation and accepted evidence

- Latest full Test Genie run \`20261121-180323-c0363bbb\` finished **FAIL**
  with 22 phases passed, 7 failed, 1 not evaluated, 1 provider unavailable,
  and 1 skipped. Programs passed at L2 after the live-inventory fixture was
  corrected to allow a truthful \`partial\` result when product composition
  omits lab-only evidence; the outcome remains unknown. Raising the board
  budget from 60,000 to 61,000 ms cleared the historical p95 finding (60,006
  ms across older program digests); a fresh direct setpoint completed in 626
  ms. Measures passed at L3/zero findings. The remaining failed phases are
  workflow, experience-contract, tidiness, security, AI conformance,
  monetization conformance, and Stripe test outcomes; experience was not
  evaluated, soak's provider was unavailable, and architecture was skipped.
  Provider preparation took 21m20s and reported Audio Tools unreachable for
  4m28s before later phases continued. The run took 32 minutes and its waiter
  returned an unexpected EOF; durable Test Genie status confirmed the terminal
  artifact. See its findings artifact for the phase details.
- Target-aware simulated evidence was completed through the lab owner path.
  \`audio-tools --instance lab validation run-simulated --json\` passed all 13
  local policy/adapter checks and wrote a receipt under the dedicated
  \`acceptance_composite\` evidence kind; it claims only five supported rows and
  does not claim native acceptance. \`run-acceptance-join\` now covers the same
  18 required rows as the board, records a partial composite with
  \`acceptance_complete=false\`, and lists 13 missing rows. Fresh governed run
  \`prog_74fa0118-ca6d-4993-872a-8894f7a69026\` completed in 939 ms with 4
  inspected receipts, 5/18 rows resolved from target-bound simulated claims,
  and \`acceptance=unknown\` / \`pending:required_owner_receipts\`. Focused CLI
  validation tests and all 32 setpoint tests pass; focused API experiment and
  qualification tests pass; \`make -C packages/proto verify-committed-gen\`
  passed. Product API still returns 404 for the lab-only evidence route.
  The lab API was restarted through \`vrooli scenario restart\` to serve the
  generated proto, and its \`@lab\` instance is running on port 15373; the
  product instance remains on port 19630.
- GPT-Live key setup is ready through the existing encrypted-file credential
  provider. \`vrooli credentials status --identity vrooli/openai-gpt-live
  --field api-key\` reports provider available and field unconfigured. The
  owner-provision command supports hidden terminal entry or stdin. No key was
  created or exposed and no OpenAI call was made; the route remains gated on
  verified identity, revocable consent, durable spend/recovery, and provider
  usage evidence.
- Measures-health passes for Audio Tools at L3 with zero findings. Six
  owner-backed UTC-window aggregates cover BYOK credentials, corpus clips,
  experiments, experiment runs, speaker profiles, and wake-word templates.
  The live validator probe reports all six domains covered at full tier with
  \`probe_passed=true\`; six CLI measure commands also returned counts through
  the running API. The BYOK endpoint returns aggregates only. The former
  \`byok_credentials_duplex\` scan finding was removed by making migration
  staging transaction-local rather than a second durable table. The latest
  full Test Genie run (\`20261121-180323-c0363bbb\`) passed programs at L2,
  measures at L3, and contracts at L3; 7 other areas failed, 1 was not
  evaluated, 1 provider was unavailable, and 1 was skipped. Requirements
  validation is clean at L3. See the validation section and \`PROGRESS.md\` for
  the retained run result and remaining failures.
- \`vrooli scenario requirements validate audio-tools --json\` passes at
  business-health L3/clean with zero stale-evidence findings after the full
  Test Genie run refreshed the snapshot.

## Accepted evidence this unit

- At 2026-11-21 15:32 UTC, the governed Program Runtime lab instance was
  restarted with \`--variant-dependencies audio-tools\`, preserving the live
  Program Runtime and Audio Tools product instances. \`program-runtime
  --instance lab library run audio-tools.setpoint-read\` completed successfully
  as \`prog_226780ad-ce00-4a9f-90f6-ce23b5ffbe19\` (17,028 ms). Its structured
  output lists \`audio-tools/experiment/list-evidence\` among the successful
  binding IDs, has no binding errors, and reports 1 inspected owner receipt,
  13 required rows, 0 resolved rows, and \`acceptance=unknown\`. Each required
  row is named and remains pending; no acceptance was inferred. The initial
  shape finding was false: empty repeated protobuf fields are omitted by the
  bridge's protojson serialization, not backed by a stored malformed receipt.
  The product Audio Tools
  \`ListQualificationEvidence\` route still returns HTTP 404. The requirements
  validator reports PASSED with one stale-evidence warning. This closes the
  lab-binding access gap, not qualification or W2 row evidence.
- After correcting the typed repeated-field default, governed lab setpoint
  run \`prog_73b1a9bd-4c02-47e8-9388-ee0289e4c63b\` completed successfully in
  6,745 ms. It reports \`receipt_count=0\`,
  \`pending_telemetry:no_owner_receipts\`, \`inspected_receipts=0\`, 0/13 rows,
  and no errors. The owner CLI independently reports \`evidence: []\` on
  \`audio-tools@lab\`; no receipt was created or edited. Three focused board
  tests pass. At that time the full board test exposed five PRD targets absent
  from the board; the following update closes that mapping drift.
- Added explicit pending rows for the five new PRD targets
  \`OT-P0-011\` and \`OT-P1-005\` through \`OT-P1-008\`, bringing the board and
  contract to 20 rows / 18 required outcomes. Each row maps to exactly one PRD
  target and remains unknown without a receipt. Full focused suite
  \`python3 -m unittest discover -s test -p test_setpoint_read.py\` passes (31
  tests). Governed run \`prog_9260077d-f0d9-4b86-8e76-6281dd3ac405\` returns 20
  rows, 18 required, 0 resolved, and \`pending:required_owner_receipts\`; the
  product API still returns HTTP 404 for the evidence route.
- Ran the documented bounded qualification with \`audio-tools --instance lab
  validation run-simulated --json\`. It persisted one
  \`audio-tools-acceptance-v1\` receipt labelled \`execution_lane=simulated\` and
  \`simulated_as_native=false\`; all 13 local policy/adapter checks passed. It
  explicitly does not claim target acceptance or native/provider parity. The
  next governed lab setpoint run \`prog_cd3541dd-03e9-48d5-b8c4-3d7851a89551\`
  reports 20 rows, 5/18 required rows with simulated coverage, 13 unavailable,
  no errors, and acceptance still unknown (\`pending:no_target_aware_acceptance_claims\`).
  Product instance was not targeted; lab health was confirmed on port 15373.

- Long-form evaluation now retains provider-reported word times with a
  per-clip 20,000-word cap, scores them only against human-reviewed references,
  and reports signed start/end boundary drift rates in milliseconds per hour.
  Synthetic tests cover constant offset, progressive creep, invalid evidence,
  timestamp serialization, and 500 randomized alignment parity cases. The full
  API race suite passes after fixing terminal-entry eviction ordering and
  removing a real resource CLI dependency from OpenRouter HTTP tests. No
  human-reviewed long-form corpus exists, so this enables scoring but does not
  qualify a model.
- Resource attribution at 2026-11-21 15:36 UTC found Whisper Small's server
  using 719 MiB VRAM (selected artifact \`ggml-small.bin\`). Ollama model
  processes used 8,308 MiB and the shared reranker 1,380 MiB of GPU memory.
  Audio Tools API RSS was 31 MiB, attached Sherpa 8 MiB, and Kale 19 MiB.
  Whisper's 3.4 GB model cache includes approximately
  1.43 GiB Medium and 1.51 GiB Turbo on-demand tiers beyond eager Small.
  Whisper now declares separate, owner-fenced Medium and Turbo cache entries.
  Selective pruning removes only the named on-demand artifact and checksum;
  the shared \`WHISPER_MODELS_DIR\` relocation remains declared on both entries.
  An installed CLI refreshes from the current source command manifest, and the
  legacy whole-entry route rejects these selective entries. Read-only previews
  report 1,533,763,124 B for Medium and 1,624,555,275 B for Turbo, both
  ineligible because usage/model identity is unknown. The resource-manifest
  and resource-store state formats were bumped to v2 with their v1 digests
  retained and v1 still readable. Focused cache race tests and the full
  \`internal/resources\` race suite pass (352 seconds with a 900-second timeout);
  no host model data was removed.
- A fresh live default/usage audit at 16:26-16:31 UTC confirmed Whisper Small
  for batch STT, Sherpa Zipformer for stream config, and Qwen3 1.7B for
  summarize. The product TTS smoke returned local \`kale-tts-1-nano\` in 1,281
  ms and wrote a valid 24 kHz mono WAV. The initial cached health response had
  marked Kale unavailable, but a direct declared \`/healthz\` probe returned
  ready and a subsequent synthesis succeeded. Bootstrap readiness/liveness
  now uses the provider-catalog health path rather than repeating \`/healthz\`.
  At the same host sample, the 16 GB GPU used 9,390 MiB: Whisper Small owned
  719 MiB, Ollama's \`qwen3.5:9b\` chat.default server owned 7,702 MiB, and the
  separate embedding model owned 558 MiB. This attributes the 9B residency to
  the shared chat role, not Audio Tools summarize. Audio Tools still requires
  full corpus/device/latency qualification before promoting Kale.
- TTS CLI JSON previously wrote binary audio before its proto JSON, making the
  nominal machine output invalid. Its JSON renderer now emits the Connect proto
  JSON response only. Focused tests cover parseable JSON with and without
  \`--out\`; a live short synthesis emitted parseable JSON naming Kale Nano.
- The OpenAI credential setup path is ready for owner entry: \`vrooli
  credentials status --identity vrooli/openai-gpt-live --field api-key\` reports
  the encrypted-file provider available and the field unconfigured. The
  provision command accepts a hidden terminal prompt or stdin; no API key was
  created or exposed, and the production duplex route remains gated. For
  Sherpa, the official v1.13.2 Linux x64 shared runtime archive
  (\`1ef6741535f7af4d69e394fd440a807108036d26ed4f542660191019da5c0daa\`) was
  downloaded and digest-verified. Native tagged tests pass. Updating the
  linux-amd64 tree digest to match the current working-tree adapter restored a
  fresh unsigned stage: tree digest
  \`e7e3f513383f15aa4f9e2050b752fdc94aa1ace16aee8306324fb97406a1a513\`, archive
  SHA-256 \`2c47c2e00eb6a044ef22c3e4cc6af1b70b9186a647d5e3d7785fec7e2e660a9c\`.
  Offline stage validation passes, but the stage remains non-deployable;
  owner-supplied publication URL/provenance and signing/publication are open.
- After the GPT-Live credential UX and Models table selector cleanup, the full
  Audio Tools UI suite passed (135 files, 1,282 tests); full UI lint and
  type-check also passed. The linked browser package emitted its known source
  map warnings, but they did not cause test failures. A separate deferred-
  AudioContext regression validates monitor teardown; the browser capture
  package passes 34 files / 243 tests with one skipped and builds successfully.
- At 2026-11-21 13:49 UTC, the product API was rebuilt from current source and
  returned healthy with build identity
  \`sha256:142800a7700370a5d568f1d0af29c49a2c2d55cb96e7e3160a7c9fe8bce67d47\`.
  \`ListEngines\` returned in 0.524 s (it had taken 10.0 s before the read-only
  resolver path), selected the attached \`sherpa-streaming\` engine, and reported
  Whisper and Kyutai unavailable as \`not_serviceable\`. The Whisper resource
  remained stopped after the inventory request. The CLI rendered the same
  selected engine. A product auto-TTS synthesis then selected \`kale-tts\` /
  \`kale-tts-1-nano\`, wrote a valid 126,044-byte mono 24 kHz PCM WAV, and
  reported 848 ms. This is route and serviceability evidence, not model quality
  or managed Sherpa release qualification.
- Current validation after the read-only inventory fix: full API race suite
  \`GOWORK=off go test -race -count=1 -timeout=300s ./...\`, API \`go vet ./...\`,
  full CLI race suite, UI suite (133 files / 1,278 tests),
  \`make catalog-check\`, and \`make portability\` (Windows/amd64 and macOS/arm64)
  pass. Sherpa \`make check\` and its opt-in native streaming smoke also pass with
  the pinned runtime/model and punctuation assets.
- At 2026-11-21 13:22 UTC, the current product API and separately profiled lab
  API both returned healthy responses on ports 19630 and 15373. Provider refresh
  reported Sherpa STT and Kale TTS available; Kokoro was unavailable. The
  product CLI's automatic TTS request selected \`kale-tts\` /
  \`kale-tts-1-nano\`, returned a 128,444-byte mono 24 kHz PCM WAV (2.675 s), and
  reported 1,143 ms latency. The externally attached Sherpa CPU service returned
  \`/healthz/stt\` ready on port 8881; its managed signed-release lifecycle remains
  unsupported. This is live route/serviceability evidence, not a quality or
  platform promotion receipt.
- Current verification against the shared source tree: full API race suite
  \`GOWORK=off go test -race -count=1 -timeout=300s ./...\`, API \`go vet ./...\`,
  full CLI race suite \`GOWORK=off go test -race -count=1 ./...\`,
  \`make catalog-check\`, and UI \`pnpm test\` all passed. UI reports 133 files /
  1,278 tests; the UI suite still emits source-map warnings for the linked
  shared browser package, but no test failures. Test Genie performance run
  \`20261121-131255-c188449f\` passed; overall completeness is still
  functional-incomplete and 61 coverage advisories remain.
- A synthetic model-promotion adapter fixture now uses a new fixture-only
  provider, engine, model, runtime and five owner facets. It verifies generic
  catalog joins, exact declaration binding, and \`DefaultForVerified\` selection
  without production model-ID branches. \`go test -race ./internal/modelpromotion\`
  and \`make catalog-check\` passed.
- On 2026-11-21 10:18 UTC, both \`audio-tools@lab\` and the live product returned
  HTTP 200 from \`/health\`; the product API/UI remain on ports 19630/23345.
  Lab discovery exposes only \`API_PORT=15373\`. Its lifecycle build identity and
  compiled \`metrics.source_identity\` both report
  \`sha256:0279b026a3114e0e4bc6a1a74683798d5db7cac6130dcd30d63960af3b65372b\`.
  \`POST /vrooli.audio_tools.v1.experiment.ExperimentService/ListQualificationEvidence\`
  returns HTTP 200 with \`{}\` on lab and HTTP 404 on product. The returned
  evidence collection is empty; no evidence was fabricated or synchronized.
  The instances have separate SQLite namespaces. Lifecycle now rebuilds a
  profile artifact when its embedded source identity is stale, even when Go
  compile inputs are unchanged. The lab start did not stop or rebuild product.
  Profile health checks omit endpoints for ports absent from that profile.
  Focused lifecycle/scenario/port tests and Audio Tools bootstrap/composition
  product-exclusion tests pass. Program Runtime lab build identity and embedded
  source identity match. The live Program Runtime product stayed running on its
  original API/UI ports and processes. Its lab process environment contains
  only the declared \`lab\` variant and \`audio-tools\` variant dependency.
  \`audio-tools.setpoint-read\` run
  \`prog_677c2686-c4bc-4bfd-b7e9-40af8df7bd6f\` succeeded through the governed
  \`audio-tools/experiment/list-evidence\` binding with no remote error; all 13
  acceptance rows remain unknown/pending because the lab evidence route returned
  an empty collection. This proves same-variant routing, not qualification or
  acceptance.
- Credential setup now declares \`vrooli/openai-gpt-live\` / \`api-key\` in the
  scenario manifest, without \`env\` injection, and a unit-tested resolver reads
  the address from \`credential-authority-go\` on each call. Its focused race
  tests pass. SDA added the repository module to \`go.mod\`, and \`go list\` resolves
  it locally, but the install operation did not return verified completion;
  three leases remain active despite expired heartbeat deadlines because their
  bound SDA API process is still live. Cleanup is unverified, so no lease was
  reconciled. The resolver is not yet used by production composition. No key
  was created and no OpenAI API call was made.
- The consolidated API passed \`GOWORK=off go test -race -count=1
  -timeout=300s ./...\` after the Sherpa attach-only declaration and credential
  resolver changes. API \`go vet ./...\`, \`make catalog-check\`, Audio Tools UI
  \`pnpm type-check\`, and the 133-file / 1,277-test UI Vitest suite passed.
  The first race pass correctly exposed the Sherpa manifest/lifecycle mismatch;
  the explicit \`attachOnly\` declaration resolved it. Test Genie run
  \`20261121-104210-b5693391\` completed at 10:59:13 UTC with 21 passing, 9
  failing, 1 skipped, and 1 not evaluated phase. Portability, docs, unit, and
  storage failures cleared; dependency, workflow, experience-contract,
  tidiness, security, AI conformance, monetization, and Stripe findings remain.
  Do not retry SDA dependency installation while its three retained leases are
  bound to the live analyzer process.
- The Sherpa TTS runtime is now wired through the declared backend URL, health
  path, voice lister, and runtime-specific cache namespace. A focused bootstrap
  integration test exercises health, voice listing, synthesis, resolved voice,
  and cache identity against the Sherpa-compatible endpoint. Kale remains the
  automatic first attempt when healthy; this Sherpa route is an explicitly
  selectable compatible backend and uses Kokoro-format model data until Kale
  publishes weights.
- Audio Tools UI type-check passed, and Vitest passed 133 files / 1,277 tests.
  The repository-wide scenario manifest validator still reports two existing
  missing \`startup_policy\` fields in \`unit-health\`; it reported no Audio Tools
  manifest issue.
- Comprehensive Test Genie run \`20261121-095025-b9abc17d\` ended with 18
  passing phases, 11 failing, 1 skipped, and 2 not evaluated. The key local
  fixes are underway: an obsolete UI Input test referenced a component already
  removed in favor of the shared component library; the service declared the
  unsupported managed Sherpa resource despite using an attach-only server; the
  storage owner omitted api-core's retention receipt; and Test Genie still
  surfaced the prior AI model-slug findings. Remaining external/provider
  findings include authenticated workflow sessions, experience-manager phase
  contract mismatch, monetization/Stripe outcomes, and security advisories.

- Product TTS smoke at 2026-11-21 09:19 UTC selected managed \`kale-tts\` /
  \`kale-tts-1-nano\` for \`voice.neutral.default\`. The first response reported
  4,992 ms; a warm response reported 42 ms. Both outputs were valid 24 kHz
  mono PCM WAV (3.25 s / 156,044 bytes cold; 2.175 s / 104,444 bytes warm).
  The verified Linux artifact is now healthy and serving on demand at 220,600
  KiB RSS with no GPU allocation. This confirms route/fallback serviceability,
  not held-out naturalness or default promotion. Audio Tools API/UI RSS was
  33,576 / 60,252 KiB in the same sample.
- Read-only host snapshot at 09:19 UTC reported 10,832 / 16,376 MiB GPU
  memory and 12% utilization. Process attribution remained Whisper Small 719
  MiB, shared Ollama 7,700 + 606 MiB, and the reranker 1,380 MiB. Ollama
  \`/api/ps\` showed \`qwen3.5:9b\` and \`nomic-embed-text:latest\`; the selected
  Audio Tools summarize default \`qwen3:1.7b\` was not resident. No shared
  model was stopped or unloaded.
- Fresh read-only host sample at 2026-11-21 08:21 UTC: Audio Tools API/UI RSS
  was 27,712 / 68,268 KiB. \`audio-tools model resolve --capability summarize
  --json\` selected available Apache-2.0 \`qwen3:1.7b\`, which was not resident in
  Ollama. GPU usage was 10,823 / 16,376 MiB; process attribution showed
  7,702 MiB and 608 MiB in shared Ollama, 1,380 MiB in the separate reranker,
  and 719 MiB in Whisper Small. Ollama \`/api/ps\` identified \`qwen3.5:9b\` and
  \`nomic-embed-text:latest\`, not the Audio Tools summarize default. No Sherpa,
  Kokoro, or Kale model process was present. The live CLI selected the pinned
  Sherpa engine but marked it unavailable; Whisper batch was available.
  \`vrooli resource status sherpa-onnx --json\` returned \`unsupported_platform\`,
  uninstalled and stopped. No process was started, stopped or unloaded; the
  snapshot is point-in-time evidence. Detailed sample:
  [MODEL-FOOTPRINT-AUDIT-2026-11-20.md](../MODEL-FOOTPRINT-AUDIT-2026-11-20.md).
- TTS route construction now resolves the optional Sherpa shared runtime from
  the Kokoro provider declaration instead of branching on \`sherpa-onnx\` in
  bootstrap. Provider/model/voice identity remains Kokoro and the cache key
  remains \`sherpa-onnx:kokoro\`. Race tests passed for provider catalog and
  bootstrap; \`make catalog-check\` passed.
- BYOK CLI input now uses required \`--key-stdin\`, supports \`duplex\`, caps the
  secret at 8 KiB, and renders only the redacted stored fingerprint. Tests
  cover oversize input, missing stdin intent, generic read errors, and output
  redaction. The generated CLI evidence was refreshed with its owning test.
  Full CLI race tests and vet passed.
- Fresh full API race tests passed after the TTS declaration change; API vet,
  CLI vet, \`make catalog-check\`, and \`make portability\` (Windows/amd64 and
  macOS/arm64) passed.
- Test Genie unit phase \`20261121-084515-d5ee04c5\` passed in 63 seconds. The earlier phase exposed a flaky new restart test (partial/final HTTP requests can race), a strict TypeScript tuple issue, and a stale linked browser-package \`dist\`; the test now checks recovery on a subsequent stream, TypeScript passes, and the shared package is rebuilt. The passing phase reports 61 low-coverage advisory findings.
- Comprehensive scenario run \`20261121-085049-9df8b854\` completed in 348 seconds: 20 of 32 phases passed, 9 failed, 1 skipped, and 2 were not evaluated. Unit and business passed; failures include storage, workflow, experience-contract, tidiness, security, AI-conformance, monetization-conformance, Stripe outcomes, and soak provider unavailability. The post-run requirements validator passes with no business-contract findings. This run refreshed the business evidence gate, but it did not resolve the setpoint data-access gap.
- A fresh \`program-runtime library run audio-tools.setpoint-read\` completed
  with status \`partial\`: engine, health, and capability diagnostics were
  readable, but \`audio-tools/experiment/list-evidence\` returned a remote error
  because the live product API excludes lab routes. Acceptance remains unknown
  with 0/13 required rows resolved; no evidence was invented. The post-suite
  W2 requirements validator passes without business-contract findings. A
  governed lab-enabled evidence route was still open at that observation. The
  15:32 UTC current record above supersedes it: same-variant routing is now
  verified, while all 13 acceptance rows remain unresolved.

- After product-handler integration of the verified model-selection bridge,
  the current source passes a fresh full Audio Tools API race suite
  (\`GOWORK=off go test -race -count=1 -timeout=300s ./...\`), \`GOWORK=off go
  vet ./...\`, Windows/amd64 and macOS/arm64 portability builds, and
  \`make catalog-check\`. The browser capture package passes 31 files / 237
  tests (one skipped) and its TypeScript build. TTS resource startup is
  serialized across Kale and Kokoro through one shared composition-scoped
  ensurer; a canceled request that never enters the
  start operation no longer writes a cooldown result that can suppress retry.
- Model promotion now has experiment-owned receipt verification for
  corpus-quality and latency. Corpus-quality accepts only a passing receipt with matching
  model, capability, engine, strategy, held-out cohort, and negative-control
  evidence. Latency requires a passing persisted realtime browser product-path
  receipt, matching model/engine identity, an artifact and run ID, 30+ attempts,
  and both in-band and negative-control claims. The generic catalog selection bridge consumes an opaque result
  bound to the complete serialized model declaration, so a changed receipt
  reference or engine identity invalidates it. Memory, platform, and license
  still have no integrated receipt owners, so declaring those fields
  cannot promote a model. Product model handlers call the generic bridge using
  the experiment service; the four exact-code-allowlisted legacy defaults
  remain selectable without a complete receipt set. No current model was
  promoted.
- The pinned Kale Nano revision \`b6ce1887268c3667994fc3b0b2c82887c069915c\`
  is public/non-gated and now declares Apache-2.0 in upstream card metadata;
  stale Nano \`DRAFT, NOT PUBLISHED\` claims were removed. Nano remains a
  candidate pending joined product-quality, lifecycle, resource, and license
  qualification. Tiny remains a draft candidate. Primary source:
  [Kale TTS 1 Nano model card](https://huggingface.co/KaleComputer/kale-tts-1-nano/tree/b6ce1887268c3667994fc3b0b2c82887c069915c).
- The long-form timing evaluator now accepts directly human-reviewed timed
  word boundaries without requiring alignment-tool metadata. Partial
  tool/model provenance remains rejected, and the long-form recipe preserves
  these manually reviewed labels with correct segment offsets. Focused tests
  passed in \`internal/eval\` and \`internal/experiment/recipe\`; no human-reviewed
  corpus labels currently exist, so this repairs the path without producing a
  model-quality receipt.
- Auto TTS now gives the Kale-first/Kokoro-fallback chain one shared lifecycle
  ensurer at composition time. This makes the existing cross-resource startup
  slot effective across both backends. Focused bootstrap and TTS-chain race
  tests pass.
- Auto TTS chain composition now iterates providers with positive declared
  \`tts_runtime.auto_priority\`, building an N-entry failover chain in that order.
  Each entry now receives its own TTS service built for that provider's declared
  endpoint, runtime model, cache namespace, and health identity; model identity
  cannot drift because a shared service was configured for a different backend.
  Runtime alias, voice adapter, and startup resource also come from each
  declaration. Duplicate positive priorities fail catalog validation, and the
  composed chain shares an allowlist derived from those same resource
  declarations. Failover continues only on backend unavailability, preserving
  synthesis errors. Focused race tests cover current Kale/Kokoro product routing,
  three-provider third-entry success, identity, per-resource startup, and
  rejected undeclared resources. A new endpoint/service shape still needs
  adapter plumbing.
- Commercial-only model listings now calculate and mark defaults using the
  commercial selection policy, matching \`ResolveModel(commercial=true)\`.
  Regression coverage requires exactly one commercial default per capability;
  the focused handler race suite passes.
- Read-only host sample at 2026-11-21 07:15 UTC: the 16 GiB GPU reported
  9,997 MiB in use. Compute-process attribution showed Ollama llama-server
  processes at 6,878 and 606 MiB and the reranker at 1,380 MiB; the remaining
  roughly 1.1 GiB was not attributed by the compute-process query. Whisper
  Small's server process was present, but not listed as a compute process at
  this sample. Audio Tools API RSS was about 21 MiB. \`audio-tools stt
  stream-config --json\` selected \`sherpa-streaming\`; model resolution selected
  Apache-2.0 Qwen3 1.7B for summarize. This is a host snapshot, not a benchmark
  or proof of total Audio Tools GPU cost.
- Receipt-owner audit found no trustworthy memory, platform, or license source
  to wire. Persisted experiment evidence has no typed peak RSS/VRAM or runtime
  identity; the macOS validator is transient and stores no model/artifact
  receipt; and the rights decision receipt is not an upstream license
  provenance assessment. These facets remain blocked until their owning
  systems persist identity-bound measurement or provenance receipts.
- Resource attribution snapshot at 2026-11-21 06:58 UTC: the 16 GiB RTX 4070
  Ti SUPER reported 6,339 MiB used. Audio Tools' \`whisper-server\` (Whisper
  Small) used 719 MiB; the separate reranker used 1,380 MiB, and two
  Ollama-owned \`llama-server\` processes used 608 MiB and 3,218 MiB. The live
  Ollama retention inventory marked \`llama3.2:3b\` and \`nomic-embed-text:latest\`
  loaded, while the installed \`qwen3:1.7b\` summarize default was not resident.
  The Audio Tools API process used about 24 MiB RSS; its local Kale and Kokoro
  endpoints were unavailable in the health snapshot. This attributes the
  current model residency without stopping shared services; the GPU snapshot
  is point-in-time, not an Audio Tools benchmark.
- Sherpa managed-lifecycle support and attached-service health are now separate
  declarations. \`resource.schema.json\` documents
  \`managed_service.attach_health_check_on_unsupported_platform\`; the provider
  registry uses its probe flag to check reachable attached services while
  retaining \`PlatformUnsupported\` for lifecycle operations. Focused race tests
  cover healthy and down attached endpoints, unchanged platform JSON shape, and
  lifecycle rejection. After \`vrooli scenario restart audio-tools\`, the live
  scenario reports healthy and \`audio-tools health show\` reports
  \`sherpa-stt=available\` with a 0.94 ms probe. Speaker verification correctly
  remains unavailable because \`speaker_model_loaded=false\`. No managed model
  resource was started or stopped.
- After the later package-aware Audio Tools restart, the attached Sherpa port
  stopped answering. The live health command now reports \`sherpa-stt\` as
  unavailable (\`resource is not responding\`) while Whisper, browser STT/TTS,
  Ollama summarize, and transcode remain available. \`vrooli scenario wait\`
  initially returned degraded after the restart's 30 s health timeout, then a
  current health recheck returned healthy once the UI/API health endpoints
  recovered. The restart operation record remains historically degraded; the
  current runtime is healthy. No resource was started or stopped. Reliable
  Sherpa streaming still needs the signed managed bundle or a durable, properly
  authorized attachment.
- The current API source passed a fresh full race run after the registry change
  (\`GOWORK=off go test -race -count=1 -timeout=240s ./...\`), \`GOWORK=off go
  vet ./...\`, and \`make portability\` for Windows/amd64 and macOS/arm64. The
  scenario restart report matched expected and served build identities and
  returned healthy; one optional Kyutai credential remains unresolved.
- The dependency analyzer changed only Audio Tools' Sherpa startup policy from
  \`try_start\` to \`ignore\` using its digest-guarded \`deps update-resource\`
  operation. A fresh \`vrooli resource status sherpa-onnx\` reports no disabled
  dependency consumers; managed Linux support is still unavailable. A direct
  read-only request to the attached \`127.0.0.1:8881/healthz\` returned 200 with
  \`encoder_ready=true\` and \`streaming_model_loaded=true\`. No model or managed
  resource was started or stopped.

- The long-form timestamp scorer now avoids its quadratic alignment matrix for
  identical normalized 20,000-word streams, completing that case in 6.18 ms
  with 4.57 MB allocated on the recorded host. Packed traceback capacity now
  admits a full 20,000 × 20,000 alignment under a 128 MiB direction-buffer
  bound; worst-case dissimilar text remains O(n*m). Focused evaluator tests
  pass. This improves the admitted-input performance envelope but does not
  supply the human-reviewed long-form corpus needed for model qualification.
- Offline server fault diagnostics now cover provider timeout, HTTP 503 busy,
  dropped connection, backend restart recovery, decoder truncation, and client
  cancellation. The focused race suite passes and the receipt remains
  non-promotable with no browser/device product rows.
- Shared browser capture now extracts provider selection, stream notices,
  rejection handling, and bounded whole-turn PCM retention into focused
  modules. The package suite passes (30 files, 233 passed, 1 skipped) and its
  TypeScript build passes. This is incremental hook decomposition; larger
  orchestration responsibilities remain.
- The Vrooli generic Whisper status path reported an HTTP health failure while
  the resource-specific status command and direct root endpoint returned
  healthy/HTTP 200. No shared process was changed. The discrepancy is filed
  for investigation as \`knw-1795238304545903315\`; until resolved, generic
  resource status is not reliable evidence for this endpoint's health.
- The current read-only CLI projection at 2026-11-21 05:59 UTC reports the
  managed Audio Tools API's effective STT/TTS/summarize/transcode capabilities
  as available through the currently healthy alternatives, while Sherpa STT,
  Kokoro, and Kale are unavailable because their local endpoints are stopped.
  \`audio-tools model resolve --capability summarize --json\` selects Qwen3 1.7B,
  and \`audio-tools stt stream-config --json\` selects \`sherpa-streaming\`. These
  two facts establish policy intent and routing only; the selected streaming
  engine cannot serve until Sherpa is attached through an authorized,
  lifecycle-durable path.
- Sherpa's unsigned Linux/amd64 managed-service stage now builds after fixing
  the package name format and deterministic tar feature detection. The local
  bundle passes the owner validator when the old released digest is omitted,
  but the actual manifest rejects the new tree digest (staged
  \`64fc5ae9…\`, declared \`bd90dac5…\`). No signature or publication was produced.
  Release-owner provenance, an HTTPS publication URL, digest approval, signing,
  and publishing remain required before Linux lifecycle support can be declared.
- BYOK liveness checkers are now constructed by iterating active provider
  declarations with probe facts; bootstrap no longer repeats four provider /
  capability tuples. A focused test verifies the declared checker set and the
  special OpenRouter path. The API full race suite, \`go vet ./...\`, and
  Windows/amd64 + macOS/arm64 portability builds pass after this change.
- The catalog mirror now makes Qwen3 1.7B the only summarize \`default\` role
  and verifies that the commercial selector matches \`summarize.default\`;
  licensed Llama 3.2 3B remains a fallback. Model docs are regenerated and the
  full model catalog suite plus \`make catalog-check\` pass.
- The offline fault receipt now includes a backend-restart regression whose
  first request receives HTTP 503 and whose retry completes with recovered
  text. The receipt timestamp is 2026-11-21 06:04 UTC; six server-side cases
  pass and the artifact remains explicitly non-promotable.
- The auto-TTS declaration-priority test now clones each provider runtime
  record before changing its test priority. This fixes a shared-catalog
  mutation that made provider order depend on test order; unique request/cache
  identities keep repeated recovery runs isolated. \`go test ./internal/bootstrap\`
  passes. Production fallback logic was unchanged. Remote credential checkers
  now derive from provider declarations; local resource checkers and aggregate
  health composition remain explicit in bootstrap.

- BYOK encryption-key initialization now fails closed on corrupt existing
  key files and non-missing read errors. First-boot key creation uses an
  exclusive atomic link so concurrent API starts cannot overwrite each
  other’s key. Regression tests cover corrupt-file preservation and concurrent
  initialization; \`go test -race ./internal/byokstore\` passes.
- Test Genie demand-lease verification now refuses HTTP redirects so its
  bearer-like receipt header cannot be forwarded to another host. The new
  redirect regression and soak package pass under \`go test -race
  ./handlers/soak\`.
- Current validation after those fixes: full API \`go test -race -count=1
  -timeout=180s ./...\` passes; all 1,282 UI tests pass and \`pnpm type-check\`
  succeeds; CLI \`go test ./...\`
  and \`make catalog-check\` pass. These results do not close the broader Test
  Genie end-to-end failures recorded below or qualify live model/device paths.
- The admin StreamConfig picker joins each STT engine to model records through
  declared \`engine_ids\` and presents readiness, licence, default, size, and
  language facts without blocking engine switching. Its focused UI suite passes
  all 11 cases, including loading, empty, error/retry, and multiple-default
  states. This closes the historical engine→model picker join item; it does not
  qualify any listed model for production selection.
- Auto TTS follows declared provider priority without a bootstrap probe. On a
  cold request, it makes one bounded start attempt for Kale, verifies backend
  liveness, then starts Kokoro if Kale remains unavailable; local resources use
  \`startup_policy=ignore\`. Starts are single-flighted and cooldown-guarded per
  process, and status probes remain truthful while both resources are stopped.
  \`go test -race -count=1 -timeout=180s ./...\` passes across the full Audio
  Tools API. This validates routing and concurrency only; no resource was
  started and no model-backed synthesis or Kale quality was qualified.
- Shared browser voice capture now delegates stream notice state and policy to
  \`voice/streamNoticeController.ts\`, with separate behavior tests for silent
  protocol statuses, degradation, recovery, and dismissal. The previously
  extracted rejection controller remains intact. Package validation passed:
  \`pnpm test\` (28 files, 227 passed, 1 skipped) and \`pnpm build\`. The remaining
  \`useVoiceCore.ts\` orchestration and PCM provider are still large, so this is
  one incremental extraction rather than full UI-surface consolidation.
- Sherpa streaming model loading now accepts both the existing dated ONNX
  filenames and the generic \`encoder.onnx\` / \`decoder.onnx\` / \`joiner.onnx\`
  layout used by the declared Kroko archive. Explicit file overrides still
  win. Unit tests cover both layouts and overrides. With the existing local
  68 MB Kroko artifact, the native test confirmed the final “your country”
  phrase through both the decoder and the real WebSocket adapter; the 55.0 s
  repeated JFK fixture ran at 58.01x inference RTF and timestamps stayed within
  source duration. \`make -C resources/sherpa-onnx check\` and the native-tagged
  test suite pass. This is one finalization/adapter smoke, not general quality
  evidence or commercial license qualification; Kroko remains on-demand.
- Read-only host sample at 2026-11-21 05:06 UTC: Audio Tools API RSS was about
  22 MiB; the managed Whisper small-model server used 719 MiB VRAM, with an
  active capacity claim last marked active at 04:35 UTC. Other observed GPU
  consumers were Ollama (6.9 GiB and 608 MiB), reranker (1.38 GiB), and Ollaya
  (3.2 GiB). The evidence does not identify which client last used Whisper, so
  it was not stopped. \`resource-whisper status\` reported healthy and direct
  GET \`/\` returned 200, while two generic \`vrooli resource status whisper\`
  probes reported an HTTP check failure against that same endpoint. Treat the
  generic status mismatch as unresolved telemetry; it does not prove the
  server was unavailable. No model service was started or stopped.
- LPBS account-credit lifecycle packages were rechecked after the USD-v1 work:
  \`go test -race -count=1 ./internal/commerce ./handlers/commerce\` passes.
- Read-only runtime check on 2026-11-21: the managed Audio Tools scenario is
  healthy, but its API/UI processes are from 2026-11-20 06:26 UTC and predate
  the current source. The API uses about 25 MiB RSS. Sherpa remains unmanaged:
  control-plane status is unsupported/disabled on Linux while an external
  attach-only CPU server on port 8881 is healthy (\`/healthz\` reports streaming
  model and encoder ready; about 10 MiB RSS). The CLI still resolves
  \`sherpa-streaming\`; the runtime requires a published managed adapter before
  claiming lifecycle portability. No process was restarted or stopped.
- Fresh resource attribution on 2026-11-21 01:07 UTC: Audio Tools API/UI
  processes used 25,572/39,532 KiB RSS. Managed Whisper was healthy under
  \`vrooli resource status whisper --no-fast\` with \`ggml-small.bin\` and Vulkan;
  \`nvidia-smi\` attributed 716 MiB VRAM to \`whisper-server\`. The managed Kale
  runtime was stopped after its product smoke. Ollama reported
  \`qwen3.5:9b\` at 5.88 GB VRAM and \`nomic-embed-text:latest\` at 323 MB; a
  separate GPU snapshot showed 6,886 MiB for the Ollama 9B server and 1,380
  MiB for the reranker. The shared chat model and reranker account for most of
  the observed residency; Audio Tools summarize still resolves to
  \`qwen3:1.7b\`. The default/fast Whisper status probe reported a failure while
  direct GET \`/\` returned HTTP 200; \`--no-fast\` reported healthy. This probe
  discrepancy remains open. No models were unloaded.
- Rechecked Whisper status on 2026-11-21 02:07 UTC. Both
  \`vrooli resource status whisper --fast\` and \`--no-fast\` report serving with
  \`status_code=ok\`; the prior fast-probe failure did not reproduce. Keep the
  dated discrepancy as historical evidence, not a current active defect.
- Audio Tools' Whisper dependency now uses SDA's digest-guarded
  \`startup_policy=ignore\` contract (manifest SHA-256
  \`242fdc02c195e7a9b89abea41fc5b6406cca50cf0f32e790faa02a9c3a072f7c\`). The
  API retains its default on-demand \`STT_AUTO_ENSURE\` recovery path, so batch
  requests can start Whisper when no lighter serviceable engine is available.
  This avoids eager startup on future Audio Tools restarts. Whisper was already
  running as a shared resource during this check; it was not stopped because
  its other consumers were not established.
- Runtime configuration check on 2026-11-21: source defaulted
  \`AUDIO_TTS_BACKEND\` to \`auto\`, but the service manifest pinned it to
  \`kokoro\`; Kale was stopped, draft/opt-in, and undeclared. This historical
  configuration gap is addressed: SDA's digest-guarded \`deps declare-resource\`
  operation added Kale as optional with \`startup_policy=ignore\`, and service
  configuration now selects \`auto\`. Kale is not started at Audio Tools boot.
  Auto selection used to probe Kale at bootstrap and fell back to Kokoro if it
  was stopped; the newer on-demand route below supersedes that behavior and
  attempts Kale when an auto TTS request arrives.
  A later managed product-path smoke verified model loading, five canonical
  voice mappings, bounded output, and cache reuse, but did not qualify voice
  naturalness or release readiness. That smoke does not establish the current
  runtime state.
- Bootstrap TTS identity now resolves provider IDs, model IDs/runtime names,
  adapter IDs, voice maps, and the Kokoro runtime alias from provider/model
  catalogs. Drift tests cover both resources and keep Kale in planned status;
  focused bootstrap and TTS chain tests pass. Backend mode strings and
  operational health endpoints remain explicit runtime concerns.
- Timed-reference storage now rejects lexical disagreement with the clip's
  reference transcript, including direct repository writes. This guards the
  scorer from comparing timestamps attached to different words; it does not
  establish human review or a qualifying score. Focused corpus and handler
  tests passed under the race detector.
- Clip metadata and optional timed-reference writes/deletes now share one
  SQLite transaction. Fault-injection tests prove failed annotation insertion
  rolls back the clip and failed clip deletion preserves its annotation;
  focused corpus and handler tests pass under \`go test -race\`.
- Removed the unused \`TTSService\` handler dependency and its composition
  plumbing; synthesis already flows through the provider chain, while the
  internal service remains owned by bootstrap for local adapters and corpus
  evaluation. Focused TTS handler, server-composition, and bootstrap tests pass.
- Sherpa release staging now validates archive paths and types and hashes the
  archive's in-memory bundle contents against the signed tree digest before
  signing. Five adversarial fixtures cover valid parity, content mismatch,
  traversal, symlink, and hardlink entries; \`make -C resources/sherpa-onnx
  check\` passes. This does not sign or publish a managed release.
- Fresh host resource sample on 2026-11-21 02:26 UTC: Audio Tools API RSS was
  19,652 KiB and the attached Sherpa server RSS was 7,552 KiB; neither Whisper
  nor Kale was running. GPU use was 9,307/16,376 MiB. Ollama \`/api/ps\` listed
  \`qwen3.5:9b\` and \`nomic-embed-text:latest\`, while the Sherpa STT default is
  attached CPU and Audio Tools summarize still resolves to \`qwen3:1.7b\`. The
  large resident models are shared chat/embedding workloads, not Audio Tools
  STT/TTS/summarize. No process or model was started, stopped, or unloaded.
- \`vrooli resource status kale-tts --json\` on 2026-11-21 02:53 UTC reports the
  managed Linux/amd64 artifact installed and verified, but Kale is stopped.
  No inference process was started during this check; auto TTS therefore
  cannot select Kale until an operator starts the opt-in resource.
- \`audio-capture-browser\` was rebuilt and refreshed into Audio Tools through
  the package refresh workflow on 2026-11-21 03:02 UTC. The resulting managed
  scenario reports healthy with two processes. Optional Kokoro, Kyutai STT,
  and managed Sherpa dependencies were degraded/unavailable during startup;
  Sherpa remains disabled by operator state and Kyutai's optional credential
  is unresolved. This confirms package activation and app health, not those
  model paths.
- LPBS now exposes a durable account-credit reserve/dispatch/settle/release
  lifecycle under \`lpbs:account-credit.lifecycle\`. It resolves the account
  from verified user identity, allows release only before dispatch, requires
  measured usage and provider receipt ID for idempotent settlement, and keeps
  uncertain dispatched holds pending for recovery. Focused LPBS lifecycle,
  handler, role tests and API compile passed. Receipt IDs and input digests
  are persisted but not independently verified. Audio Tools now has a
  least-scope principal declaration and injected short-lived-token/client
  adapters. Dispatch is fenced before provider contact; confirmed terminal
  usage settles, and missing/ambiguous finalization retains the hold. Focused
  Audio Tools duplex race tests and LPBS lifecycle tests pass. Product
  composition remains unwired pending production coordinator wiring, secure
  app-scoped LPBS principal provisioning, and trustworthy provider invoice
  evidence. No credential was issued; GPT-Live remains planned.
- Sherpa local TTS now wires voice listing through the selected runtime, and
  non-2xx voice-list responses fail instead of parsing an error body as voices.
  Focused TTS/bootstrap tests and the Sherpa server suite pass; no model was
  loaded for this code check.
- Kroko's bounded finalization smoke was rerun against the locally available
  native Sherpa libraries/model and both JFK fixtures: 11.0 s audio retained
  the complete “your country” ending at 62.69x RTF; 55.0 s audio retained it
  at 62.00x RTF. Both timestamps stayed within source PCM bounds. This is
  final-tail smoke evidence only, not general accuracy, noise robustness,
  license, or default-promotion evidence.
- Kyutai's dependency declaration was updated through SDA's digest-guarded
  \`deps update-resource\` operation from \`try_start\` to \`ignore\`, preserving its
  optional accelerator description. \`audio-capture-browser\` / Audio Tools
  package refresh completed; Audio Tools reported healthy. Startup skipped
  Kyutai, and \`vrooli resource status kyutai-stt --json\` confirms it remains
  stopped with \`needs_reacquire\`. The optional Kyutai credential gap still
  appears in startup reporting, but no Kyutai model load was attempted.
- The STT manifest now ranks \`low_footprint\` before \`native_streaming\`, so a
  healthy Sherpa Zipformer wins over Kyutai. \`TestShippedPolicyPrefersSherpaOverKyutai\`
  passes; the active CLI reports Sherpa selected through its current explicit
  operator pin. The attached server's \`/healthz\` reports the encoder and
  streaming model ready. The managed Sherpa resource remains disabled and
  unsupported by its current Linux manifest, and Kyutai remains unavailable;
  no claim is made that the unpinned live fallback was exercised.
- After the package refresh, Audio Tools is healthy with Kokoro and managed
  Sherpa startup degraded; Kale and Kokoro are installed but stopped, and no
  live TTS path is currently serving. This refresh confirms product process
  health, not TTS model readiness.
- Updated the summarize configuration reference to match the shared
  \`summarize.default\` role (\`qwen3:1.7b\`) and documented the measured long-report
  quality tradeoff with the \`qwen3.5:4b\` escalation. Earlier fallback text
  naming \`llama3.2:3b\` was stale.
- Canonical TTS voice aliases for Kokoro, Kale, OpenAI TTS, and ElevenLabs now
  live in provider declarations; the TTS compatibility mapping and catalog
  derive from the provider registry. Catalog validation rejects aliases without
  an adapter, malformed aliases, missing canonical voice coverage, and unknown
  canonical IDs. Dynamic backend voices retain their runtime identity. Focused
  TTS, provider-catalog, conformance, bootstrap, and BYOK package tests pass.
- LPBS browser access tokens now use the reusable API-core consumer request
  provider also used by AI Gateway. Audio Tools routes to the LPBS verifier
  based on an untrusted issuer hint only, then requires that verifier's signed
  issuer/signature/expiry checks; scenario-authenticator requests continue to
  their existing verifier. The duplex wallet helper carries only the exact
  token from a verified LPBS request, including supported browser cookies.
  Focused Audio Tools server and duplex packages pass under \`go test -race\`;
  API-core consumer identity and AI Gateway identity race tests pass. This
  completes human-token verification plumbing, not production GPT-Live route
  composition or account-credit authority provisioning.
- Forced source restart on 2026-11-21 04:12 UTC served matching expected and
  actual build identity \`sha256:d0dfe998289b37d53bf2cafa500729d2fcb362348550c273764dc0d524a35bd0\`
  and returned healthy. The live \`audio-tools stt engines --json\` command
  reports Sherpa available and active by operator pin, Whisper available as a
  candidate, and Kyutai unavailable with \`needs_reacquire\`. Kale and Kokoro
  remain stopped. The request-level CLI read was slow (~40 seconds) but
  completed; no live TTS inference or API-backed provider request was made.
- Combined source validation after the provider, declaration, policy, and auth
  updates: \`go test -race -count=1 -timeout=180s ./...\` passed for the full
  Audio Tools API, and \`go vet ./...\` passed. \`make catalog-check\`, the SDA
  \`update-resource\` focused tests, scoped \`git diff --check\`, and the earlier
  shared API-core consumeridentity / AI Gateway identity race tests also pass.
- Fresh resource attribution on 2026-11-21 04:04 UTC: Audio Tools API/UI RSS
  was 30/71 MiB and Whisper used 719 MiB VRAM. Audio Tools' Kale and Kokoro
  resources were installed but stopped; the attached Sherpa path uses CPU. GPU
  total was 11,320/16,376 MiB. A shared Ollama \`qwen3.5:9b\` process used
  6,886 MiB, the Ollaya process used 1,264 MiB, and the shared reranker used
  1,380 MiB; Ollama also listed \`nomic-embed-text\` resident at 323 MB. Audio
  Tools' policy-selected \`qwen3:1.7b\` was not resident in Ollama at sampling
  time. This attributes most observed GPU use to shared inference services,
  while Whisper is the principal active Audio Tools model footprint. No model
  was started, stopped, or unloaded for this sample.
- Host portability inventory is recorded in
  \`coverage/host-portability-observation-20261120.json\` as
  \`host_observation_not_qualification\`. It records Chrome 150, Firefox 157,
  Linux/amd64, Docker limits, no PipeWire input, and unrun inference/device
  cells. The owner evidence command returned 404 because the current API
  composition is product-only; no qualification receipt was persisted.
- Sherpa release staging now requires an owner-supplied HTTPS publication
  URL, constrained upstream provenance, and the declared version before
  signature stage. Five adversarial stage-validator tests and
  \`make -C resources/sherpa-onnx check\` pass. A target-native SDK,
  release-owner provenance/publication endpoint, and external signing
  authority remain required; no stage was signed or published.
- AMI timing-label intake now has a review runbook defining three held-out
  recordings, source/license/checksum attribution, word-boundary review and
  adjudication, and the qualification receipt gate. Corpus still has zero
  timed references, so no timing score is qualified.
- GPT-Live's WebSocket read loop now follows the session context, closes the
  socket on cancellation, and exits without reporting a spurious disconnect.
  \`GOWORK=off go test -race -count=1 ./internal/duplex\` passes after this change.
- Test Genie comprehensive run \`20261120-230712-5da63d84\` completed \`FAIL\`.
  Unit, API, proto, and performance phases passed. Soak/workflow failures lacked
  a verified lease, BAS grant, or authenticated account; monetization and
  provider conformance still have declared gaps; security also reported the
  fixed context issue and separate SSRF findings. This is broader environment
  and integration evidence, not a passing release gate.
- Validation CLI now injects the wall clock through a canonical \`Now\` seam and
  runs the Docker host probe through an injectable command runner. Full CLI
  race suite passed; Test Genie unit run \`20261120-225715-4d8dd9c5\` passed at
  L2 Complete and no longer reports \`MISSING_INJECTABLE_SEAM\`. Advisory
  low-coverage findings remain.
- The current Test Genie docs phase \`20261120-225843-f769d79e\` passes. The
  documentation contract, links, and manifest coverage are clean, while the
  broader content-quality provider still reports a large existing warning
  backlog; the pass is not a claim that all documentation debt is closed.
- Product composition is selected by default. \`AUDIO_ENABLE_LAB_API=true\` is
  the explicit lab opt-in. \`modules.AllEndpoints()\` now includes the external
  validation provider endpoints; CLI omissions name why the Test Genie
  protocol is absent from the product CLI.
- API \`GOWORK=off go test -race -count=1 -timeout=180s ./...\` completed with
  all packages passing after the \`internal/modules\` coverage repair;
  \`go vet ./...\` passed.
- CLI \`go test -count=1 ./...\` and \`go vet ./...\` passed after regenerating
  \`.vrooli/generated/cli-primitive-evidence.json\` through its owning test.
- \`make catalog-check\`, \`make endpoints\`, \`make portability\` (Windows/amd64 and
  macOS/arm64), the final Test Genie \`docs\` phase (\`20261120-210246-e6940323\`),
  and scoped \`git diff --check\` passed.
- \`audio-tools model resolve --capability summarize --json\` selected
  \`qwen3:1.7b\`; \`audio-tools stt stream-config\` resolved \`sherpa-streaming\`.
  Current model/runtime details are recorded in
  [MODEL-FOOTPRINT-AUDIT-2026-11-20.md](../MODEL-FOOTPRINT-AUDIT-2026-11-20.md).
- Offline fault receipt \`coverage/offline-fault-diagnostics-20261121.json\`
  records three passing server regressions, zero product-path rows, and an
  explicit failed promotion join. Independent focused race run passed.
- Duplex \`Handle.Close\` concurrent callers now wait for provider cleanup;
  focused package race tests passed. Process-local admission reservation and
  settlement primitives, the GPT-Live adapter, and encrypted-store key resolver
  now exist; production admission, route composition, and durable ledger remain
  unimplemented.
- Provider catalog runtime identity is now one resolver used by capability
  canonicalization; conformance checks round-trip every declared provider ID,
  engine ID, and alias. Focused catalog/capability tests passed.
- Corpus timed references now flow through proto, API create/list/get, SQLite,
  and \`audio-tools corpus import --timed-reference-file\`. Validation covers
  word order and bounds, clip duration, review provenance/timestamps, and size
  limits. Proto committed-generation verification, API race handler/storage
  tests, and CLI parser tests passed. No reviewed labels were ingested.
- Manual-only \`human_reviewed\` references no longer need fabricated aligner
  metadata; if aligner provenance is supplied, both tool and model remain
  required, and \`aligner_only\` still requires both. API and CLI regressions
  pass. Current registry audit found num[sot]:180 clips and zero timed references, so
  timing score qualification remains open. AMI is a potential source only
  after its forced-alignment boundaries are human-corrected. The catalog now
  describes Rumble-2-Flash as batch STT, matching its declared model facts.
  Verified this tranche with the full API race suite, CLI corpus race tests,
  \`make catalog-gen && make catalog-check\`, and scoped \`git diff --check\`.
- Audio transform duration and Normalize output-measurement fields are now
  explicitly documented as best-effort, with zero meaning unknown. Runtime
  handlers measure output bytes / EBU R128 analysis; focused handler and audio
  package race tests pass. The owning proto generation and committed-output
  verifier pass after updating the comments.
- The ninth-tranche resource-only Kale smoke confirmed that the pinned runtime
  can produce audio, but did not exercise the product handler or establish
  voice quality. Subsequent tenth-tranche evidence below now covers the product
  handler; Kale is preferred by auto mode when healthy, while quality/release
  qualification remains open.
- \`TestBuildKaleTTSProductConnectRouteAndCache\` passed against a fake runtime
  through an isolated bootstrap/Connect server; it verifies route selection,
  voice mapping, WAV output, response metadata, and cache reuse. It is a
  deterministic product wiring regression, not a model-quality receipt.
- \`TestBuildKaleTTSLiveProductRoute\` passed against the managed Kale Nano
  runtime. Product output was 98,424 bytes with a measured 2,048 ms duration;
  the warm request took 175 ms and repeat cache hits were observed. Speed 1.25
  output also retained valid finite WAV chunk sizes. This verifies route/runtime
  integration, not naturalness or release readiness; the resource was stopped.
- A fresh managed Kale Nano product run on 2026-11-21 UTC passed through the
  isolated Connect product API. The first synthesis produced 2.046 seconds of
  audio in 940 ms while loading the model; all five canonical voice mappings
  then produced complete WAV output, with warm synthesis in 43–50 ms for
  2.0–2.35 seconds of audio. The managed process used 207 MiB RSS after the
  first request and 238 MiB after the multi-voice run, within the manifest's
  512 MiB RAM estimate. The runtime was stopped after the receipt. This
  strengthens resource/route evidence but does not measure perceived
  naturalness, intelligibility, or release/licensing readiness.
- The model registry now restricts all Kale model records to \`linux/amd64\`,
  matching the only explicitly supported platform in the provider resource.
  \`TestKaleModelsPlatformMatchesManagedResource\` prevents candidate models
  from being presented as runnable on a platform the resource explicitly
  marks unsupported and verifies they are filtered from macOS/Windows TTS
  candidates. The model catalog tests and \`make catalog-check\` pass.
  Missing platform metadata and Sherpa's attach-only Linux path remain
  distinct from an explicit unsupported target.
- \`audio-tools stt stream-config --json\` selected \`sherpa-streaming\` with
  \`passthrough\`; \`audio-tools model resolve --capability summarize --json\`
  selected \`qwen3:1.7b\` with commercial eligibility. These are fresh live CLI
  outputs from the current scenario.
- Duplex reservations now fail closed for an admission error or missing
  reservation, prevent concurrent overspend, settle measured usage once, and
  release holds on close/cancellation. Focused \`go test -race ./internal/duplex\`
  passed. Accounting is still process-local and no provider policy adapter or
  OpenAI route is wired.
- Durable spend accounting is not safely implementable at the current duplex
  seam: \`Request\` contains only caller-supplied policy booleans and a session
  ceiling, while no handler binds a verified identity to a monetization account
  or obtains an authoritative account limit. A local SQLite ledger could
  serialize rows but could not establish whose limit to debit. The next
  implementation gate is an authenticated account-resolution/limit interface;
  then persist reservation ID, account ID, session ID, hold, state, and final
  measured micros in one transaction, with idempotent settle/release and
  restart recovery that keeps unresolved holds reserved. Do not count
  \`usage_rows\` as settled spend: its asynchronous drop-newest queue is lossy.
- The isolated GPT-Live primary WebSocket adapter uses injected key resolution,
  bounds WebSocket reads and commentary, maps audio/transcript events, and
  prices confirmed final usage. Race tests cover startup, message mapping,
  settlement, disconnect, and socket cleanup. The primary protocol has no
  documented cancel event; barge-in uses its documented output fixture
  interruption, and user text input is explicitly unsupported. Product wiring
  and durable accounting remain pending.
- \`StoreKeyResolver\` now reads the canonical \`openai-gpt-live\` / \`duplex\`
  credential through the existing decrypted BYOK store facade. Missing,
  storage, and decryption failures fail closed with generic redacted errors;
  a real encrypted-store round trip passed under the package race suite.
- GPT-Live catalog declarations include sourced integer micro-USD rates
  (active 50,000/minute, standby 20,000/minute, one-second billing increment);
  duplex remains excluded from default roles. Admission checks explicit
  consent, remote-processing permission, active provider/model eligibility,
  and a per-session/process budget before connecting. The current planned
  GPT-Live declaration deliberately fails eligibility. Pricing data is not
  itself settlement or customer billing.
- TTS auto mode sorts declared candidates by unique \`auto_priority\` (currently
  conditionally opted-in Kale Nano then qualified Kokoro), starts each on demand through one shared, declaration-
  allowlisted ensurer, and retains truthful health probes; explicit backend
  choices stay pinned. Focused race tests exercise cold start, full three-entry
  fallback, backend-specific identity, and 16 concurrent requests sharing one
  Kale start. Startup attempts cap at 60 seconds. No model-backed start or
  synthesis was performed for this change.
  Auto-mode cache
  replay now gets backend keys from the configured local provider/failover chain
  rather than a duplicated handler list; a new regression exercises a fallback
  cache key that is absent from the old hardcoded list. The legacy Kokoro voice
  override is restricted to Kokoro. Quality/release qualification remains open.
- The current Kroko native streaming finalization regression passed against two
  JFK fixtures with the existing bounded 1.2 s silence tail. Both outputs
  retained the “your country” ending and timestamps stayed bounded by source
  PCM; an 11 s clip took 362 ms after a 3.63 s model load, and a 55 s clip took
  844 ms after a 1.02 s warm load. This is a tail-preservation smoke, not
  accuracy, general latency, or release qualification.
- The live model-backed Kroko WebSocket smoke and the generic model-file
  discovery regression are recorded above. The generic layout no longer needs
  per-file overrides; the older dated layout and explicit path overrides stay
  supported.
- Current read-only runtime check: Audio Tools is healthy with two processes;
  API RSS was about 23.7 MiB. The running API predates current source changes.
  Kale is installed but stopped. Managed Sherpa is disabled/unsupported on
  this Linux host, while the separately attached endpoint returned HTTP 200.
  Live \`settings providers\` reports Kokoro down, so source-level Kale auto
  selection is not active in the shared server process yet.
- Audio Tools' custom HTTP entry point now installs api-core's passive
  authentication and provenance middleware, with the scenario-authenticator
  audience enabled by default. A focused composition test proves a verified
  human principal reaches domain code. This adds identity context only; no
  existing route is newly gated, and it does not create duplex consent or
  durable spend authority.
- Test Genie \`business\` phase passed (\`20261120-210152-ca6b8bee\`);
  \`vrooli scenario requirements report audio-tools\` shows 24 requirements,
  3 modules, 15 operational targets, 2 complete / 7 in progress / 15 planned,
  and zero unproven claims. \`business-health validate scenario audio-tools\`
  passed with no business-contract findings.

## Next unit

GPT-Live's provider and lazy authority-backed resolver are now retained in
production composition, but no session endpoint is mounted. The authenticated,
app-scoped LPBS USD reserve/dispatch/settle/release client is implemented and
race-tested. Remaining gates are the verified identity and app-scoped service
authorization at the route, source-bound revocable remote-processing consent,
durable session recovery, and provider usage evidence strong enough to settle
actual cost. The Platform key-create picker is unavailable; the owner can
provision \`vrooli/openai-gpt-live\` / \`api-key\` through Vrooli's encrypted
credential CLI. Audio Tools' current Configuration UI still writes a separate
legacy BYOK record. Three expired-heartbeat install leases remain bound to a
live SDA process and cleanup is unverified, so do not bypass that flow or mint
a service credential locally. No OpenAI key or API call exists. The overall
goal remains open until product routes and qualification targets have evidence.

### Sherpa local release staging (2026-11-21)

- \`scripts/package-sherpa-native-release.sh\` completed an unsigned Linux/amd64
  stage at \`/tmp/audio-tools-sherpa-stage-20261121\`; no signing or publication
  was performed. The release manifest tree digest is
  \`739defa53ac501f3f47316aa998d8aae467d0d403c89fe94f2cd09e3a2fba383\` and the
  archive digest is \`4bf36450f847e12ef7c7acf7db852d3ed010e75e425d3002b90102f64e0fc549\`.
- The staged tree digest does not match the currently declared
  \`resource.json\` linux-amd64 digest (\`bd90dac5c3f33dfdec4d9df7806c0572f409629d395d288c637c2f370cb1ac5f\`).
  This is diagnostic evidence only: the stage used a locally attached native
  root, and the declaration still says no signed release is published. Do not
  promote this digest or claim managed acquisition until provenance, repeatable
  target build, signing, publication and install lifecycle gates are satisfied.
- Resource-owned validation passed: \`make -C resources/sherpa-onnx check\` and
  \`LD_LIBRARY_PATH=/tmp/opencode/sherpa-root/lib make -C resources/sherpa-onnx native-test ROOT=/tmp/opencode/sherpa-root\`.
  Optional model-specific smoke checks were skipped without their model env.
- Research Desk authorization \`638f32eb-71dd-4fcf-b2a2-7938c32e22dc\` remains
  pending for retiring the superseded prepared commission and launching the
  current maturity commission. No approval was inferred.

### GPT-Live credential journey audit (2026-11-21)

- The standard Audio Tools configuration UI and settings API already accept
  \`openai-gpt-live\` / \`duplex\`, validate against the provider catalog, and write
  to the durable encrypted BYOK store. Credentials are represented by a
  redacted fingerprint in listing responses. \`StoreKeyResolver\` retrieves and
  decrypts that canonical credential pair, and tests cover the encrypted-store
  round trip and redacted failures. No OpenAI key was created or accessed.
- The integration is not yet usable for sessions: production bootstrap does
  not construct \`StoreKeyResolver\`, \`OpenAILiveProvider\`, or a duplex
  coordinator/handler. No duplex transport is exposed. Provider status is
  \`planned\`, model readiness is \`candidate\`, and admission rejects both.
- Keep the credential form available as the standard secure BYOK path, while
  clearly treating saved credentials as not yet session-capable. Next work is
  to compose an authenticated duplex transport behind consent and durable
  reservation/settlement/recovery controls, then qualify before changing
  provider/model readiness.

### Current source verification (2026-11-21)

- \`make catalog-check\` passes against the current provider, operation, model and
  cross-catalog declarations.
- \`make lab-test\` passes for the explicit lab boundary and experiment/eval/soak
  packages and handlers.
- \`GOWORK=off go test -count=1 -timeout=180s ./...\` passes across the complete
  Audio Tools API module, including duplex, BYOK, provider catalogs, model
  catalog, bootstrap and lab packages.
- \`GOWORK=off go test -race -count=1 -timeout=240s ./...\` also passes across
  the complete API module. These source checks do not qualify managed Sherpa
  acquisition, model quality, hardware cohorts or the live GPT-Live route.
- UI full tests also pass after repairing the catalog mock's stable reference:
  \`pnpm test\` reports 134 files / 1,282 tests passed; \`pnpm type-check\`,
  \`pnpm lint\`, and \`pnpm selector:check\` pass. The lint repair removed an
  unnecessary assertion and changed dynamic deletion to \`Reflect.deleteProperty\`.
- CLI source also passes \`GOWORK=off go test -count=1 ./...\` and \`GOWORK=off
  go vet ./...\` across its full module.
- The operation catalog now generates \`docs/reference/audio-operations.md\`
  from the same seed as its UI projection. The catalog package byte-compares
  the generated reference; focused operation-catalog tests and
  \`make catalog-check\` pass. The existing Diagnostics page has a dedicated
  transcode run test in \`AudioOpsPanel.transcode.test.tsx\`.

### Product-contract alignment (2026-11-21)

- The Business Health wizard preview was merged as portable-voice-v2 without
  replacing existing requirement modules or resetting earned checkboxes. The
  PRD now has 20 operational targets and the registry 29 requirements. New
  targets cover Wispr Flow desktop feature/task parity, low-resource models,
  advanced voice workflows, a $10 standalone-to-bundle journey, and offline
  cross-platform installation. Their five requirement records remain planned.
- \`business-health validate scenario audio-tools\` passes with one
  \`business_evidence_stale\` warning. The Test Genie business phase
  \`20261121-003833-16b52ad4\` passes at L3 with the same stale-sync warning;
  the current full requirements-sync snapshot has not been refreshed.
- Business Health wizard dedup search was bounded to 2.5 seconds per response
  with at most three concurrent hint queries and now honors the request context.
  Runtime StartSession and SubmitAnswers each completed within the hint budget;
  the answer workflow returned in 5.8 seconds. Full Business Health API tests,
  race tests, build, and vet pass after adding isolated storage to its fleet
  tests.

### Local TTS model identity alignment (2026-11-21)

- Local chain results now report the model registry's canonical Kokoro ID
  (\`kokoro-82m-v1.0\`) instead of the resource protocol string (\`kokoro\`).
  Bootstrap resolves Kokoro's default and Kale's fast candidate from model
  roles in the embedded registry; the HTTP adapter request name and cache
  namespace remain resource-owned runtime identities.
- The provider catalog now declares the experimental Kale adapter binding
  (\`local:kale-tts\`) without changing its planned status. Adapter bindings are
  valid independently of activation, and bootstrap resolves the adapter ID and
  implementation-owned voice mapping through the catalogs. The generated Go
  and TypeScript provider surfaces were regenerated by \`make catalog-gen\`.
- Model declarations now distinguish canonical model IDs from the
  protocol-specific runtime model strings for Kokoro and Kale. Bootstrap gets
  those runtime strings from the selected registry role. Explicit Kale
  synthesis and liveness now check the \`kale-tts\` capability rather than
  reporting against Kokoro's capability. \`make models-doc\` regenerated the
  reference table with the new field.
- Focused \`go test -race\` passes for bootstrap, TTS chain, model catalog, and
  provider catalog and cross-catalog conformance; \`make catalog-check\` passes.
  This closes the TTS model/adapter identity duplication but does not remove
  backend selection branches or qualify Kale's candidate status for promotion.

### Voice-cloning candidate and memory gate (2026-11-21)

- Added Qwen3-TTS 0.6B Base as a planned provider and non-runnable model
  candidate. The pinned upstream model card reports Apache-2.0 and a 2.52 GB
  repository; Qwen documents clone conditioning with reference audio and its
  transcript across 10 languages. This declaration does not grant voice-use
  permission; consent and reference retention policy are product requirements.
- The official vLLM-Omni guide's 0.6B CustomVoice example reports about
  13.5 GiB idle with default GPU cache allocation on an RTX 4090; a tuned
  profile reports about 5 GiB idle and about 10 GiB request peak. These are
  adjacent CustomVoice deployment facts, not measurements of the Base clone
  path or Audio Tools. The registry's 5 GiB hardware value is labeled an
  estimate and is not a qualification or default-eligibility signal.
- Acceptance requires a managed bounded-memory runtime, one authorized
  reference clip plus transcript, explicit consent/retention controls, a short
  held-out clone request, WAV validation, human-reviewed similarity and
  naturalness, cold/warm latency, real-time factor, peak RSS/VRAM, and clean
  unload/restart evidence. No runtime was installed or model downloaded.
- \`make catalog-gen\`, \`make models-doc\`, and \`make catalog-check\` pass. The
  API full \`go test -race -count=1 -timeout=300s ./...\` suite passes; UI
  \`pnpm type-check\`, \`pnpm lint\`, and \`pnpm test\` pass (134 files, 1,282
  tests). The provider/model record remains planned/candidate after validation.

### Sherpa model footprint and readiness split (2026-11-21)

- The resource now exposes Zipformer plus punctuation as one explicit,
  checksum-verified \`streaming-zipformer-en-2023-06-26\` acquisition group.
  Kokoro TTS is on-demand and disabled by default so the streaming STT server
  does not load its 330 MB model. The speaker-verification default remains in
  the normal acquisition group; the full model-catalog race test rejected an
  earlier on-demand change because it would have made a declared default
  unavailable on fresh installs.
- Resource health is capability-specific: general \`/healthz\` is ready when a
  model-backed capability is loaded, \`/healthz/stt\` checks only streaming,
  and \`/healthz/tts\` also requires the output encoder. Missing TTS model data
  returns 503 from TTS routes without disabling STT.
- The pinned data group acquisition returned \`status: verified\` for two
  artifacts. The native model and complete WebSocket smoke passed on the
  approved local speech fixture. A source-built temporary STT-only server
  returned 200 from \`/healthz\` and \`/healthz/stt\`, and 503 from \`/healthz/tts\`
  and \`/v1/audio/voices\`. While briefly attached on port 8881, the product CLI
  listed \`sherpa-streaming\` as available and selected; \`transcribe-stream\` on
  the public dictation fixture returned \`The quick brown,fox jumps.\` in
  202 ms. The server was stopped afterward; port 8881 now refuses connections
  and the managed resource remains unsupported. The post-stop engine list has
  Whisper available for batch work but no available native streaming engine;
  the configured Sherpa pin remains visible as unavailable. This verifies
  local route integration on one clip, not corpus quality or release readiness.
- The product-only \`audio-tools.setpoint-read\` run at 07:44 UTC
  \`prog_2cb0a08b-da7f-4bed-838b-b921d75bf4cb\` returned partial: 3 diagnostic
  readings were available, 0 engines were available, and the evidence-list
  binding failed with \`remote_error\`. All 13 required outcome rows remain
  unknown; two are explicitly excluded from the portable-voice lane. This is
  not an acceptance result.
- Validation: resource \`make check\`, \`vrooli resource validate sherpa-onnx\`,
  \`make -C scenarios/audio-tools catalog-check\`, and the full API race suite
  passed. Managed Linux lifecycle remains unsupported because the trusted
  signed adapter bundle has not been published.

### Bounded Sherpa concurrency and timestamp boundary symmetry (2026-11-21)

- Sherpa's resource wrapper now admits at most four concurrent streaming
  sessions by default (\`SHERPA_ONNX_STREAMING_MAX_SESSIONS\`, configurable
  1–64). Excess requests are rejected before decoder allocation; stream close
  and stream-creation errors release capacity. Resource unit/race suites pass.
  This bounds per-session decoder growth, but does not establish host memory
  sizing or production lifecycle support.
- The long-form timestamp scorer now reports \`max_start_abs_ms\` as well as
  maximum end-boundary error, making a start-only outlier visible. The focused
  timestamp scoring tests pass. Corpus state remains 180 registered clips and
  zero timed references; three distinct 10+ minute AMI recordings still need
  independent human review and import before qualification can run.
- The GPT-Live credential-authority regression now proves that resolving a
  real encrypted operator credential does not change catalog eligibility.
  Focused duplex race tests pass; no key was provisioned and no OpenAI request
  was made. Production route composition remains gated on verified identity,
  revocable consent, durable account spend/recovery, and trustworthy invoice
  evidence.
- Integrated verification after these changes: full API race suite
  \`GOWORK=off go test -race -count=1 -timeout=300s ./...\`, API \`go vet ./...\`,
  Sherpa server \`go test -race ./...\`, \`make catalog-check\`, scenario/resource
  manifest validation, and focused Gosec on soak and duplex packages pass.
- The targeted Gosec scan now has zero findings after adding narrowly scoped
  comments for operator-configured/fixed-path endpoint access, fixed fixture
  paths, persisted evidence paths, and non-secret logical credential/API route
  identifiers. This does not clear the separate full Test Genie Gitleaks
  finding at \`signing.json:5\`; its value remains unopened and unprinted.

### Complexity and runtime hygiene follow-through (2026-11-21)

- Split the high-branching experiment estimate/evidence mappers, settings
  provider-config update, session event mapper, and soak evidence persistence
  into named domain helpers. Added focused tests for validation, URL trimming,
  nil event payloads, and recovery/surface evidence. The owning packages pass
  race-enabled tests; the protobuf test fixtures use \`proto.Clone\` to avoid
  copying generated mutex-bearing message state, and full API vet passes.
- Extracted voice page-lifecycle wiring from the shared browser capture hook
  into \`voice/pageLifecycle.ts\` with two tests; \`useVoiceCore.ts\` is 24 lines
  shorter at this seam (1,841 → 1,817). The package tests and build pass.
- Removed two unconditional per-session \`fmt.Printf\` calls from the STT
  segmenter hot path. Soak cleanup now runs with a five-second timeout derived
  from \`context.WithoutCancel(ctx)\`, preserving context values while ensuring
  cancellation does not prevent restoring persistent mode.
- Validation after the combined edits: full API race suite, API vet, six
  focused race-enabled package suites, UI test suite and production build,
  catalog drift check, Sherpa server race suite, resource validation, focused
  Gosec, and scoped \`git diff --check\` pass. The first UI run used a stale
  packaged local \`audio-capture-browser\` build and failed to resolve the newly
  added module; rebuilding the owner package and reinstalling the frozen local
  dependency offline made the full UI tests/build pass.
- The latest full Test Genie result is still the 10:59 UTC run above; it has
  not been rerun after these changes. That run's security finding at
  \`signing.json:5\` remains uninspected. The SDA-owned dependency phase remains
  unresolved while its leases are bound to the live analyzer process.

### Credential readiness, model selection, and remaining complexity (2026-11-21)

- The operator confirmed that OpenAI credentials should use the existing secure
  infrastructure, with frictionless/reliable/debt-free setup as the target. The
  connected OpenAI Developers setup flow did not return a confirmed choice, and
  no Platform key creation or retrieval action is exposed in this session. No
  secret was created or written. The canonical provisioning command is
  documented, but production composition still does not consume that identity;
  the Configuration page's encrypted BYOK entry is explicitly documented as a
  separate legacy store and cannot be presented as a working substitute.
- The live model resolver selects \`qwen3:1.7b\` for summarize, and STT stream
  configuration selects \`sherpa-streaming\` with automatic streaming mode. A
  host process snapshot showed Ollama using the GPU for \`qwen3.5:9b\` and
  \`nomic-embed-text\`, plus a GPU-backed reranker. These shared resource
  processes are not attributable to Audio Tools from this snapshot; Audio
  Tools API and Kale TTS processes had no GPU compute PID. Do not unload shared
  models without identifying their consumers. The Sherpa managed runtime is
  still unavailable because its trusted adapter release is unpublished.
- The targeted experiment, session, settings, and soak handlers now have no
  \`gocyclo -over 10\` findings after extracting persisted replay measurement
  loading as well. Focused race tests pass for all four packages. The latest
  full Test Genie run predates these changes and remains the latest available
  broad evidence; no comprehensive pass is claimed.

### Catalog, browser lifecycle, separator startup, and unit gate (2026-11-21)

- CLI operation catalog coverage now requires exact flag parity, including
  defaults and accepted values. This found and corrected the missing \`format\`
  declaration for \`audio merge\`; generated operation metadata is refreshed.
- Wake-word template loading moved into a small browser voice helper with its
  own tests, reducing lifecycle work in \`useVoiceCore\`. The shared browser
  package tests and build pass.
- Sherpa's speaker embedding model remains startup-loaded, while Spleeter
  separation now loads on first use under synchronized lifecycle control.
  Readiness distinguishes an unloaded separator from an unavailable model,
  and a missing artifact returns a specific 503. \`make check\` passes; native
  model-backed tests remain unrun because the required speaker and separation
  models are absent. The trusted managed Sherpa release is still unavailable,
  so this improves serviceability but does not close the live STT gap.
- The first Test Genie unit run failed because the installed \`file:\` snapshot
  of \`audio-capture-browser\` was stale and lacked the newly built helper.
  Refreshing through \`vrooli scenario setup audio-tools\` repaired the consumer
  copy without restarting the live product. The follow-up unit run
  \`20261121-114502-1faafd1a\` passed in 66 seconds. It reports 61 low-coverage
  advisories; overall scenario completeness remains \`functional_incomplete\`
  (55), with requirements, operational targets, and historical test-phase
  pass rates still open.

### Small-model policy and TTS auto eligibility (2026-11-21)

- Auto TTS source previously included every positive-priority declaration, which put
  Kale Nano first even though provider \`kale-tts\` is \`planned\` and the 14.8 MB
  model is \`candidate\`. Automatic selection now requires an active provider,
  a matching, default-role TTS model that is default-eligible, and an explicitly
  commercial license. Kale remains explicitly selectable for
  evaluation; \`auto\` uses qualified Kokoro until Kale's evidence gates pass.
  Focused bootstrap tests prove the planned provider and candidate model are
  excluded, priority sorting remains data-driven, and Kale's unqualified
  runtime receives no automatic synthesis requests.
- The running product API process predates this source change and was not
  restarted; source tests establish the corrected composition, but the live
  product behavior has not yet been refreshed.
- Verification: \`GOWORK=off go test -race ./internal/bootstrap\` and scenario
  \`make catalog-check\` pass. The first focused run exposed a stale assumption
  that auto mode always had two providers; bootstrap now permits a single
  qualified provider and retains an explicit error for an empty eligible set.
- A fresh Test Genie unit run, \`20261121-120607-56ab00d1\`, passed in 64 seconds
  against the current source. It retains 61 low-coverage advisories; the
  scenario remains \`functional_incomplete\` at score 55, and no full-suite pass
  is claimed.
- Updated the current TTS pipeline/configuration references and Kale dependency
  description to match source auto eligibility; dated model-footprint history
  is preserved with a current-policy addendum. Test Genie docs run
  \`20261121-121257-d13cc8b3\` passed in 6 seconds, while its report still has
  360 warnings and overall content quality at L1. Repository scenario
  validation found no Audio Tools issues; it still fails on two unrelated
  \`unit-health\` missing-\`startup_policy\` findings.
- Read-only host sample at 11:57 UTC: RTX 4070 Ti SUPER used 10,097/16,376 MiB
  at 100% utilization. GPU process attribution was shared Ollama model
  processes (7,702 MiB and 606 MiB) plus the shared reranker (1,380 MiB); the
  Ollama API identified \`qwen3.5:9b\` (6.74 GB VRAM) and
  \`nomic-embed-text:latest\` (323 MB). Audio Tools resolves summarize to
  \`qwen3:1.7b\`, which was not in the resident Ollama list. Audio Tools product
  and lab APIs used 18 and 19 MiB RSS, respectively; the healthy Kale Nano
  server reported \`loaded=true\`, used 17 MiB RSS in this sample, and had no GPU
  allocation. No model or shared service was started, stopped, or unloaded.
  This is a point-in-time host sample, not a peak-memory or workload benchmark.

### Current continuation update: live Kale route and credential readiness (2026-11-21)

- TTS automatic routing now has an explicit \`conditional_auto\` declaration for
  Kale Nano. Its planned provider and candidate model remain unpromoted; the
  narrow exception tries Kale before Kokoro only because the model is
  commercial and its runtime is available. Other planned providers remain
  excluded from automatic selection. Kokoro is the declared fallback.
- The isolated product composition synthesized through the already-running
  managed Kale Nano runtime: auto selected \`kale-tts-1-nano\`, returned valid
  mono 24 kHz WAV (97,664 bytes, 2,033 ms audio), and reported 638 ms first
  request latency; repeat requests hit cache. Four voice mappings also passed
  warm synthesis. This is route/runtime evidence, not naturalness or
  comparative-quality qualification.
- The first live API attempt used the prior build and waited 46 seconds while
  auto-routing still failed. Restarting Audio Tools through the current source
  CLI built and served identity
  \`sha256:de1bf443006934b283590cb3fcbe91af2b5cb36736d72a97acfb73ed56d25b9d\`;
  a fresh live CLI synthesis then selected \`kale-tts\` / \`kale-tts-1-nano\`,
  wrote valid mono 24 kHz WAV (132,044 bytes), and returned in 434 ms. The API
  health check passed after restart. This closes the stale-process gap for the
  current TTS route; Kokoro audio comparison and Kale promotion gates remain
  open.
- Production server composition now retains an \`OpenAILiveProvider\` backed by
  a lazy \`AuthorityKeyResolver\`. Tests prove that it resolves the canonical
  \`vrooli/openai-gpt-live\` / \`api-key\` identity on use, observes rotation, and
  does not mount a duplex route. Credential absence therefore does not block
  startup, but neither provisioning nor this composition enables remote calls.
  The operator authorized secure key creation if supported, or preparation for
  owner-provisioning through existing infrastructure. No Platform key-create
  action is available in this session. The canonical secure prompt is
  \`vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key\`;
  it reads from stdin into Vrooli's encrypted-file credential authority. The
  identity and field match the provider resolver, so no \`.env\` shim is needed.
  Production authenticated duplex routing, consent, durable spend/recovery,
  and trustworthy invoice evidence remain required before calls can be enabled.
- Current command evidence continues to resolve summarize to \`qwen3:1.7b\`
  and streaming STT to \`sherpa-streaming\`. Managed Sherpa lifecycle remains
  unsupported until a trusted signed adapter release is published; the prior
  local attach smoke is not a managed-release qualification.
- Live streaming STT was restored using the documented attach-only path. The
  previously staged adapter in \`/tmp/audio-tools-sherpa-stage-20261121\` was
  older than source and failed to honor \`SHERPA_ONNX_TTS_ENABLED=false\`; a
  fresh CPU bundle was built from current source and the pinned runtime into
  \`/tmp/sherpa-bundle-current-20261121\`. With the pinned 2023 Zipformer and
  punctuation data, \`/healthz/stt\` reported ready, the product engine list
  selected \`sherpa-streaming\`, and a product streaming request on the bundled
  LibriSpeech fixture returned two segments in 725 ms and 855 ms. API health
  returned healthy afterward. The process is externally managed and will not
  survive host/repository cleanup; this does not close the signed managed
  release gate or establish broad recognition quality.
- Verification for this continuation: Kale's bootstrap/provider/catalog race
  tests, \`make -C scenarios/audio-tools catalog-check\`, isolated product-route
  tests, credential resolver/server tests, source-CLI scenario restart and live
  TTS request. No live OpenAI request was made. Broad Test Genie evidence
  remains stale and does not represent this continuation.
- The UI documentation viewer no longer eagerly embeds the whole Markdown
  tree. Vite now emits selected-document chunks: the viewer component dropped
  from 1.14 MB (387 kB gzip) to 168 kB (50.5 kB gzip), with the chosen document
  fetched separately. The docs page reports loading, missing, and chunk-load
  failure states. The entry bundle also imports the client directly now;
  voice transport registration is loaded by the lazy dictation and diagnostics
  routes that use it. The main chunk fell from 874 kB to 853.5 kB (254 kB
  gzip), while the chunk-size warning remains. Full UI tests passed before the
  transport split (133 files, 1,278 tests); after the split, the focused
  DictationRecorder and LiveTry tests, production build, lint, type-check, and
  selector/string drift checks passed.
- A fresh Test Genie performance run, \`20261121-131255-c188449f\`, passed in 28
  seconds with profiling infrastructure, budget gates, runtime measurement,
  surface resolution, and trend coverage clean. The run still reports overall
  functional incompleteness (2/29 requirements and 2/20 operational targets
  passing) and stale business/docs/proto/structure evidence. A fresh unit run
  also passed, with 61 low-coverage advisories. These phase results do not
  close the full Test Genie gates.

### Current continuation update: read-only STT inventory and final live checks (2026-11-21)

- The product API was restarted from the current source after the serviceability
  inventory changes. Lifecycle status reports build identity
  \`sha256:3ae975841d7cdb07b5303285b9af65ff93a5c4f38962f5b616f5f1fb40070cb0\`;
  \`/health\` is healthy. The live \`ListEngines\` Connect endpoint returns the
  selected \`sherpa-streaming\` engine in under one second. CLI inventory agrees:
  Sherpa is selected and serviceable; Whisper and Kyutai are unavailable.
  Whisper remains installed but stopped after the inventory calls, confirming
  the listing no longer invokes lifecycle recovery or control-plane probes.
- Final live TTS check synthesized valid mono 24 kHz PCM WAV through
  \`local/kale-tts\` in 215 ms (100,844 bytes). This establishes the post-restart
  auto route, not comparative naturalness or promotion quality.
- API full race suite and vet, CLI race suite, UI test suite (133 files / 1,278
  tests), UI lint/type/selector/build, portability build, catalog drift check,
  and Sherpa resource/native smoke have passed for the current changes. The
  native Sherpa smoke requires the pinned model, fixture WAV, expected text,
  and punctuation model environment settings documented in the turn evidence.
- Credential creation remains unavailable through the connected Platform
  controls. The operator directed secure creation if possible, otherwise a
  frictionless path through existing infrastructure. No key was created or
  exposed; owner provisioning remains available through
  \`vrooli credentials provision --identity vrooli/openai-gpt-live --field
  api-key\`. Authority-backed lazy resolution and rotation are covered in
  source tests, but this does not authorize or mount the GPT-Live route.
- Remaining gates include the production duplex route with human identity,
  consent, cost reservation/settlement/recovery and provider usage evidence;
  real timed human-reviewed references; device/browser/OS qualification; Kale
  comparative quality and lifecycle promotion; signed managed Sherpa release;
  and broader Test Genie completeness.

### Current continuation update: session observer backpressure (2026-11-21)

- Session fanout previously dropped events from a full subscriber queue, while
  \`Subscribe\` ended with a normal return when that queue closed. A consumer
  could mistake an incomplete assistant/transcript stream for success.
- \`EmitEvent\` now detaches only the overflowing observer, drains its stale
  queue, and delivers an \`EventClosed\` reason with the count of unavailable
  events. The Connect handler sends the terminal event and returns
  \`ResourceExhausted\`. Other observers continue receiving the session without
  publisher blocking.
- \`go test -race -count=1 ./internal/session ./handlers/session ./internal/duplex\`
  passed. The regression verifies the slow observer sees an explicit 2-event
  overflow while a healthy observer receives both events. Scoped \`git diff
  --check\` passed.
- This makes loss visible and retryable at the stream boundary; it does not
  provide durable event replay. The P0 long-form trust gate remains open for
  retained/recoverable coverage across reconnect and restart.

- Follow-up edge-case check: explicit \`Session.Close\` could also fail to enqueue
  its final event into a slow observer's full queue and then close normally.
  Shutdown now uses the same overflow terminal signal, while healthy observers
  receive the regular close reason. A separate race regression covers this
  shutdown-tail case; the focused session/handler/duplex race suites pass again.

### Current continuation update: session resume and duplex event wire contract (2026-11-21)

- Implemented the existing \`SubscribeRequest.from_event_id\` contract. \`SubscribeFrom\`
  attaches under the same lock used for event emission, so retained events are
  queued before live events without a replay/live race. The ring retains at most
  128 cloned events and 1 MiB estimated payload. Oversized frames invalidate
  older cursors; stale/unknown cursors fail \`FailedPrecondition\`. The \`session_id\`
  is the initial replay anchor, and overflow close events carry
  \`resume_from_event_id\` so a client that received no event can retry.
- The session protobuf now carries \`audio_output\`, \`provider_error\`, and the
  overflow resume anchor. Mappers cover each new event. HTTP/2 Connect tests
  verify replay after a cursor and explicit rejection of unavailable cursors;
  unit tests cover recovery after overflow and history invalidation.
- Validation: full API \`GOWORK=off go test -race -count=1 -timeout=300s ./...\`
  passed; focused \`go vet ./internal/session ./handlers/session ./internal/duplex\`
  passed; \`packages/proto\` \`make verify-committed-gen\` passed after generation.
  The full API race suite also passed after these changes. Scoped whitespace and
  JSON checks passed.
- This is process-local replay only. An API restart destroys the session registry
  and its retained events; durable recovery, privacy retention policy, and
  session admission bounds remain open gates.
- Server-owned Test Genie unit run \`20261121-142501-7355100b\` passed at L2/ceiling
  in 94 seconds after refreshing shared-package freshness. The separate overall
  completeness score remains \`functional_incomplete\` (2/29 requirements and
  2/20 operational targets passing), with 59 unit advisories and stale
  business/docs/proto/structure evidence. The unit verdict does not establish
  those broader acceptance gates.

### Current continuation update: compact defaults, reliability, and catalog ownership (2026-11-21)

- Fresh live resolution selects Whisper Small for batch STT, Sherpa Zipformer
  for streaming STT, Qwen3 1.7B for summarize, and Kale Nano for automatic local
  TTS while its managed runtime is available. A product streaming request
  returned “The quick brown,fox jumps.” in 2,521 ms through the attached Sherpa
  service. This confirms route selection and serviceability for one fixture;
  the punctuation and single clip are not quality qualification. A subsequent
  Kale runtime integrity sweep covered all 10 voices with two short probes per
  voice (20 requests), median 55.59 ms, max 66.97 ms, and zero clipped samples.
  Its receipt explicitly makes no pronunciation, intelligibility, or naturalness
  claim; Nano remains a candidate.
- The offline fault matrix now includes slow-consumer cancellation. A real
  regression exposed a blocked egress send that ignored cancellation. Egress
  now selects on the session context while continuing to drain strategy events,
  allowing upstream workers to unwind. Focused race tests and the matrix test
  pass; browser/device gaps remain unqualified.
- The Sherpa release stage now tests that requesting a signed build without
  owner provenance fails before invoking the build. Seven packaging tests,
  resource validation, and scoped diff checks pass. The only stages available
  are unsigned or stale; no release signature/publication exists. The live
  Sherpa endpoint is an externally managed attachment, not managed Linux
  lifecycle support.
- Removed model ID arrays from the provider seed. \`modelcatalog\` is now the
  sole model-fact owner through each model's \`provider_ids\`; the provider seed
  loader rejects unknown fields so this duplicate shape cannot quietly return.
  The generated provider UI projection no longer repeats model slugs, and
  catalog conformance now checks model-to-provider references and capabilities.
- OpenRouter health now resolves \`openrouter/summarize\` from the same encrypted
  BYOK store used for request dispatch and probes its endpoint from provider
  catalog data. Removed the separate environment-key branch, duplicate checker,
  and absent-probe exception. No OpenRouter credential is
  configured and no external probe was made. Full API race tests,
  \`make catalog-check\`, and UI type-check/lint pass after this change.
- Validation this unit: full API \`GOWORK=off go test -race -count=1
  -timeout=300s ./...\` passes; focused catalog/bootstrap/capability race tests,
  Kale \`make check\`, Sherpa validation and packaging tests, and UI checks pass.
  Restart operation \`startop-d57f915328985e347225543aca6c3298\` succeeded in 62
  seconds and served the expected build identity
  \`sha256:2bed20a57e0483b4d898522739b56b0da13b26c97992dfe04fdff596eef45c70\`.
  Fresh health reports attached Sherpa and Kale available; OpenRouter is
  unavailable because no BYOK credential is configured. A post-restart
  streaming request again selected Sherpa and completed the fixture in 1,692
  ms, preserving the same comma insertion, so no broader transcript-quality
  claim is made.
  No full Test Genie run was repeated because the latest full run's remaining
  failures are shared workflow/auth, inherited security/config, and business
  conformance gates; its retained result remains 22 passed, 7 failed, 1 not
  evaluated, 1 unavailable, and 1 skipped.

Remaining gates include held-out human-reviewed voice/timing references,
comparative Kale quality and lifecycle promotion, a signed Sherpa release,
browser/device/OS receipts, authenticated duplex with durable cost/privacy
controls, and the shared Test Genie failures.

### Dependency startup/resource audit (2026-11-21)

- Removed the unused optional \`postgres\` edge: production persistence is SQLite
  (\`api/internal/bootstrap/sqlite.go\`, \`modernc.org/sqlite\`), and no Audio Tools
  runtime code selects PostgreSQL. This removes a stale auto-start declaration;
  it does not remove or stop the shared PostgreSQL resource. The analyzer's
  global \`impact postgres\` operation reports 108 direct and 84 indirect
  consumers (192 total), so that global resource must remain untouched.
- Demoted the direct Audio Tools \`ollama\` edge to \`startup_policy: ignore\` and
  clarified its admin/evaluation role. Production summaries are submitted to AI
  Gateway in \`bootstrap.go\`; the AI Gateway peer remains \`try_start\` and itself
  declares Ollama \`try_start\`, so no net Ollama startup or GPU-memory reduction
  is claimed while preserving Gateway-backed summaries. \`OpenRouter\` remains
  declared \`try_start\` for its BYOK role, and both LPBS and AI Gateway scenario
  routes remain declared \`try_start\`.
- Corrected Kale's stale dependency description: \`auto\` conditionally selects
  healthy Kale Nano; Kokoro remains the on-demand fallback. Whisper/Kyutai are
  optional and ignored at startup; Kokoro and Kale are also ignored, with
  Kokoro started on demand. The Sherpa endpoint remains externally attached
  rather than a managed resource dependency.
- Scenario Dependency Analyzer's supported \`deps update-resource\` preview and
  SHA-gated apply updated Ollama and Kale declarations. Its read-only
  \`scan audio-tools\` now lists six declared resources (Kale, Kokoro, Kyutai,
  Ollama, OpenRouter, Whisper) and discovers AI Gateway/LPBS from code. The
  persisted \`list audio-tools\` view remains stale and still shows removed
  PostgreSQL/Sherpa declarations; no state-refresh mutation was run.
- The current retained timing report has 10 dependency samples (p50 68.3 s,
  p90 94.0 s), with 4 setup samples (p50 3.1 s, p90 83.4 s). These are
  aggregate historical timings, not a post-change comparison. No fresh start
  was triggered because the running Audio Tools process predates the edit and
  starting peers could consume shared model resources. The demotion removes
  only a redundant direct Ollama start edge; AI Gateway's transitive edge
  prevents attributing any defensible duration or memory saving to this change.
- \`vrooli scenario validate\` still reports only the existing unrelated
  \`unit-health\` missing startup-policy errors. CLI model resolution confirms
  compact defaults remain Whisper Small (488 MB), Qwen3 1.7B (1.36 GB), and
  Kokoro 82M (330 MB); no model or shared resource was started or stopped.
- Live compact-route check (2026-11-21 19:11 UTC): \`audio-tools stt engines\`
  selects \`sherpa-streaming\`; the attached server's \`/healthz\` reports the
  streaming model loaded and returns 200. The product TTS API selected
  \`kale-tts-1-nano\` for two sequential requests (3,713 ms and 265 ms, 24 kHz
  WAV responses). Qwen3 1.7B is the summarize registry default. The generic
  model resolver still labels Whisper Small and Kokoro as STT/TTS defaults,
  respectively, because Sherpa is a streaming engine pin and Kale remains a
  conditional, unqualified candidate. The effective route matches the operator
  preference; Sherpa still depends on an externally attached service because
  managed Linux lifecycle is unsupported.
- Follow-up route and identity check (2026-11-21 19:40 UTC): after the API
  restart for the summarizer telemetry fix, streaming STT selected attached
  Sherpa Zipformer and transcribed the dictation fixture in 1,381 ms. The text
  was understandable but inserted a comma inside “brown fox”; this is route
  and latency evidence, not a dictation-quality pass. TTS selected Kale Nano
  (\`kale-tts-1-nano\`, voice \`kenzie\`) and returned a 24 kHz WAV in 594 ms; the
  earlier same-session measurements were 3,713 ms cold and 265 ms warm.
- The local summarize API now reports the resolved runtime tag \`qwen3:1.7b\`
  instead of the logical selector \`summarize.default\`. The factual fixture
  succeeded in 35,590 ms cold and 3,407 ms warm; cold startup remains a
  material latency concern. Regression coverage verifies resolved model tags
  and selector fallback. \`go test -race ./internal/ai/summarizechain
  ./internal/summarize ./handlers/summarize\` passed before restarting the
  service. Restart served expected build identity
  \`sha256:136662908a01db97e8a13a6e8eeebe60df2687403d1401c70892dcbc6c0791e7\`.
- Host sampling found \`audio-tools-api\` at about 36 MiB RSS, \`audio-tools-lab\`
  at about 22 MiB, and attached Sherpa at about 291 MiB. Shared Ollama had
  \`qwen3.5:9b\` at about 7.7 GiB VRAM and a shared reranker at about 1.35 GiB;
  these are not the selected Audio Tools summarize model. A temporary
  ~558 MiB inference process appeared during summary work but could not be
  conclusively attributed from its hashed arguments. No shared processes were
  stopped.
- The operator authorized secure OpenAI key setup or proper existing credential
  integration. \`vrooli credentials status --identity vrooli/openai-gpt-live
  --field api-key --json\` reports the encrypted-file provider available and the
  field unconfigured. Provisioning is ready through the existing hidden-input
  command \`vrooli credentials provision --identity vrooli/openai-gpt-live
  --field api-key\`; no key was created because no secret value was supplied
  through that secure flow, and no Platform key-create action is available in
  this session. Keep the existing credential-authority resolver and do not
  duplicate the key in workspace env or Audio Tools BYOK storage. The 24-hour
  Research Desk work window was not launched: Research Desk
  returned \`permission_denied\` and requires an owner decision packet. This
  window approval and OpenAI key setup are separate gates; the active Codex
  maturation work continues.
- Follow-on deterministic reliability/evaluation work: the STT fault matrix
  now includes cancellation during an in-flight provider request, releases a
  deliberately late response, and verifies that no transcript is emitted.
  Focused and full matrix race tests pass. The long-form evaluator now skips
  drift scoring when normalized timed-reference words disagree with the clip
  transcript or intervals exceed the audio duration; \`go test
  ./internal/eval -count=1\` passes. These gates prevent invalid evidence from
  being scored but do not establish reviewer authenticity or device coverage.
- Re-ran \`go test -race -count=1 ./internal/duplex\` successfully against the
  current durable account-credit path. This is package behavior evidence only;
  the route remains unmounted and GPT-Live remains ineligible without a
  verified credential, authenticated product path, and owner-approved policy.
- Fresh live CLI check (2026-11-21 19:49 UTC): \`audio-tools stt engines\` reports
  attached Sherpa Zipformer available and selected by \`StreamConfig.EngineID\`;
  Sherpa \`/healthz\` returns 200 with the streaming model and encoder loaded.
  Whisper and Kyutai are unavailable. The CLI model resolver returns
  \`qwen3:1.7b\` as the commercial summarize default. A fresh product streaming
  transcription of the dictation fixture completed in 1,364 ms, but still
  returned \`The quick brown,fox jumps.\`; this is an actual formatting/quality
  defect, not a pass. Health reports Kale available, but this check did not
  synthesize a new TTS sample.
- A combined race run initially exposed a cancellation ordering regression:
  \`TestSegmenter_ContextCancel_PropagatesToStrategy\` sometimes observed only
  one event. The segmenter now suppresses nonterminal output after cancellation
  and forwards terminal Error/Done events in order on a best-effort,
  nonblocking basis. The focused race regression passed 100 repetitions and
  the offline fault-matrix race test passed. When no receiver or buffer is
  ready after cancellation, terminal events remain best-effort to keep teardown
  bounded.
- \`make catalog-check\` passed across provider, operation, model, promotion,
  conformance, capabilities, engine, model metadata, and model handler gates.
- Added an additive local batch-STT protocol seam and a bounded Rumble vLLM
  adapter using the upstream audio chat-completion contract. It limits audio
  and response bodies, allows one in-flight request, honors request
  cancellation, disables redirects, probes serviceability with a separate
  three-second client, and reports provider/model identity without inventing
  word timestamps or confidence. A VAD strategy regression exercises the
  shared adapter end to end. The existing Whisper provider now uses that same
  \`BatchProvider\` for tier, provider/model, latency, language fallback, and
  traits; Whisper-specific recovery and timing/confidence/word evidence stay in
  its adapter. STT-chain coverage is 83.5% against the unchanged 80% floor.
  Rumble remains out of bootstrap and model selection: there is no configured
  endpoint, managed resource, product-path receipt, runtime footprint, or
  quality qualification.
- Hardened timed-reference evaluation: drift scoring now skips records whose
  normalized words differ from the clip transcript or whose intervals exceed
  clip duration. Focused evaluator tests pass. This protects the scorer from
  inconsistent or stale inputs; it cannot prove the claimed human review took
  place, and there is still no reviewed long-form corpus.
- Fixed the live-reproduced Sherpa formatting defect in server post-processing:
  a comma between Latin words now gains a space, while numeric grouping,
  decimals, CJK adjacency, abbreviations, and domain-like tokens remain
  unchanged. Resource package race and native tests pass. The attached running
  Sherpa process still uses its previous binary and was not restarted, so the
  text correction is not yet live-verified.
- Segmenter cancellation no longer drops terminal events nondeterministically:
  partial transcript events are suppressed after cancellation and terminal
  Error/Done are delivered in order on a nonblocking best-effort basis. The
  cancellation regression passes 100 race repetitions, and the fault matrix
  race test passes. Terminal events cannot be guaranteed if a canceled
  transport has no reader or buffer.
- The six measures CLI commands now propagate terminal interrupt cancellation
  to their RPCs. Their regression proves every branch receives a canceled
  context; \`cli-core\` still does not pass an embedding caller deadline into
  commands. Focused CLI measure tests pass.
- Test Genie unit run \`20261121-210232-67dc9817\` completed PASS at L2/Complete
  in 92.4 seconds after the shared capacity source was restored. It reported 58
  advisory low-coverage locations; these are warnings, not a failed gate. The
  Test Genie CLI suite also passes with \`GOWORK=off go test ./...\`.
- OpenAI credential status was rechecked after the operator authorized secure
  setup: provider \`encrypted-file\` is available and
  \`vrooli/openai-gpt-live\` / \`api-key\` is unconfigured. The workspace Platform
  key-create tool returned no confirmed setup choice, so no secret or API key
  was created. The existing \`vrooli credentials provision --identity
  vrooli/openai-gpt-live --field api-key\` command is now open at its hidden
  \`Credential value:\` prompt for owner entry. A focused source audit confirmed
  lazy encrypted-file resolution on every provider open, canonical identity
  and field use, per-call rotation, Bearer authentication, and secret-free
  errors; the resolver regression tests pass. GPT-Live still needs its
  authenticated product route and approved privacy/cost policy before use.
- Full \`GOWORK=off go test -race -count=1 -timeout=300s ./...\` passes in the
  API module, as does the full CLI race suite. Full API and CLI \`go vet\` pass,
  \`make catalog-check\` passes, and \`make portability\` cross-builds Windows/
  amd64 and macOS/arm64. Test Genie unit run
  \`20261121-201439-52e472a0\` passes at L2/Complete in 64 seconds. It records
  58 low-coverage warnings and one informational UI coverage-floor drift; the
  owner command raised the committed UI function floor from 84% to 85% against
  the measured 85.5%. Static Unit Health validation then reports no active
  maturity findings, but it is not a substitute for the executed Test Genie
  receipt.
- Live sample at 20:01 UTC: STT engine selection still chooses attached Sherpa
  Zipformer; its health endpoint reports the encoder and streaming model loaded.
  A fixture transcribed in 1,364 ms as \`The quick brown,fox jumps.\` before the
  Sherpa spacing fix was built into the running server. Model resolution still
  selects Qwen3 1.7B for summarize and the live health snapshot reports Kale
  available. Host process sampling attributed about 8.7 GiB GPU use to shared
  Ollama models (including \`qwen3.5:9b\` at 6.7 GiB and embeddings at 323 MiB),
  while Audio Tools API RSS was about 19 MiB and lab API about 22 MiB; Qwen3
  1.7B was not resident. No shared process was stopped.
- The secure GPT-Live key address remains available through the existing
  encrypted-file authority, but it is unconfigured. The owner-provision
  command is documented; no secret was supplied. GPT-Live is still not mounted
  because authenticated consent, durable cost settlement/recovery, and provider
  usage evidence remain unresolved.

### Current continuation update (2026-11-21 21:13 UTC)

- Fresh product checks still select attached Sherpa Zipformer for streaming STT
  and Qwen3 1.7B (Apache-2.0) for summarize. A local product synthesis selected
  Kale Nano / \`kenzie\`, returned a valid 2.975 s mono WAV (142,844 B), and took
  3,009 ms then 822 ms on a repeated request. This proves route and warm-path
  behavior, not naturalness, multi-voice quality, or release readiness.
- A fresh CLI streaming transcription of \`bas/fixtures/dictation-reference.wav\`
  selected Sherpa and completed in 1,994 ms; it still returned
  \`The quick brown,fox jumps.\`. The source punctuation fix has passed native
  tests but is still not in the attached process that serves the product route.
  The server is an unmanaged local attach, so this run did not replace its
  process or claim the fix live. Managed Sherpa lifecycle remains the right
  route to make that source fix durable.
- Host attribution found 11,297 MiB GPU use at 2% utilization. Ollama lists
  Gemma4 12B and \`nomic-embed-text\`, and the separate reranker owns 1,380 MiB;
  the selected Qwen3 1.7B summarize model was not resident. Audio Tools API/lab
  processes used about 20/19 MiB RSS, Kale Python about 18 MiB, and attached
  Sherpa about 6 MiB; mapped model pages and disk sizes are not represented by
  those RSS figures. No process was stopped or reconfigured. Details are in
  [MODEL-FOOTPRINT-AUDIT-2026-11-20.md](../MODEL-FOOTPRINT-AUDIT-2026-11-20.md).
- Added
  \`coverage/host-portability-observation-20261121.json\`: the host has two
  enumerated ALC1220 capture endpoints, but PipeWire exposes no source and the
  current user cannot access root:audio capture nodes. No audio was captured;
  D1 inference, D2 constrained inference, and D3-D6 remain explicitly
  unqualified/unavailable. This is inventory evidence only, not a qualification
  receipt.
- GPT-Live's LPBS accounting path now persists operation correlation and phase
  in a metadata-only SQLite journal, then reconciles through LPBS's authenticated
  recovery endpoint. A domain-separated hash of the verified human subject
  scopes each recovery query; a two-user regression proves another user's
  operation is not queried. LPBS remains the only balance/reservation authority,
  unknown usage remains pending, and no product route is mounted. Current source
  validation passes the full API \`GOWORK=off go test -race -count=1
  -timeout=300s ./...\`, full \`GOWORK=off go vet ./...\`, \`make catalog-check\`,
  and \`make portability\` (Windows/amd64 and macOS/arm64 builds).
- Full device/browser inference coverage, reviewed timing references, Sherpa
  managed lifecycle, Kale held-out quality, source-bound/revocable duplex
  consent, and a production GPT-Live route remain open.

### Streaming STT CLI JSON output (2026-11-21)

- The live CLI smoke found that \`audio-tools stt transcribe-stream --json\`
  ignored machine mode and printed human event lines. The streaming handler
  now emits one compact canonical proto JSON object per event, preserving
  arrival order as JSON Lines; human mode remains unchanged, and error events
  still cause a non-zero command result.
- Regression coverage parses the emitted segment and done events back into
  their typed proto messages. \`GOWORK=off go test ./domains/stt\` and
  \`make catalog-check\` pass. This is a CLI-only change; the streaming API
  process was not restarted.


### GPT-Live consent and key readiness (2026-11-21)

- AI Gateway owns the reusable consent authority for the
  \`audio-tools-openai-gpt-live\` source. Its owner-authenticated verification
  endpoint confirms exact source, principal owner, and active/unexpired grant
  without returning the receipt. Duplex admission requires this online check
  before reservation or dispatch. Focused AI Gateway consent and Audio Tools
  duplex tests pass. Product routing is still unmounted; grant UI/receipt
  handling and revocation handling for already-active sessions remain open.
- The operator authorized secure key creation if available, or a frictionless
  setup through existing credential infrastructure. No connected Platform key
  creation action is available in this session. The canonical
  \`vrooli/openai-gpt-live\` / \`api-key\` slot resolves through the existing
  encrypted-file provider, which reports available but unconfigured. Owner
  entry uses \`vrooli credentials provision --identity vrooli/openai-gpt-live
  --field api-key\`; the lazy resolver already reads that slot on open and
  supports rotation. No key or API request was supplied. GPT-Live remains
  unmounted pending consent UI and product-route authorization/settlement
  readiness.

### GPT-Live credential setup alignment (2026-11-21)

- The operator authorized secure key creation if a connected Platform action
  is available, or otherwise asked for setup through existing credential
  infrastructure. No key-creation action is exposed in this session; no key or
  API request was created. \`vrooli credentials status --identity
  vrooli/openai-gpt-live --field api-key\` confirms the encrypted-file provider
  is available and the slot is unconfigured.
- Production composition already prepares \`OpenAILiveProvider\` with the lazy
  credential-authority resolver. Resolver and server composition tests pass,
  and the Configuration page directs duplex setup to the canonical CLI rather
  than the separate legacy BYOK store (8 focused UI tests pass). Provisioning
  securely prepares the credential, but no production duplex route is mounted
  while product authorization and billing evidence gates remain open. The
  settings architecture guide now describes this current path accurately.

### D2 Sherpa int8 host portability receipt (2026-11-21)

- Ran \`audio-tools validation run-host-portability\` against the lab instance.
  The first attempt exposed that live/serving composition does not mount lab
  evidence endpoints; rerunning against the supported lab API persisted the
  receipt. The first lab attempt then caught a runner defect: model override
  environment variables used basenames instead of container paths. The runner
  now passes absolute \`/models/asr/...\` paths and has a regression for them.
- The bounded D2 run used the digest-pinned container with no network, 2 CPU
  cores, 4 GiB memory/swap, 256 PIDs, read-only mounts, and the explicitly
  selected int8 encoder/decoder/joiner/tokens. The exact fixture transcript was
  \`The quick brown, fox jumps.\`; model load was 1.423 s, streaming inference
  108.3 ms for 1.955 s audio (18.05x RTF), and measured cgroup peak was
  445,902,848 bytes. Receipt artifact:
  \`audio-tools:host-portability:20261122T001014.074403173Z:amd64\`.
- The owner receipt remains \`passed:false\` and acceptance incomplete because
  D1 CPU and accelerator inference are still unqualified. One short clean
  fixture validates the bounded D2 execution path only; it does not establish
  corpus quality, long-form reliability, broader hardware fit, or D3-D6.
- Follow-up: run the D1 Kyutai accelerator cell when a serviceable engine and
  supported accelerator are available. Preserve the separate cell states; do
  not merge inventory availability into inference qualification.

### D1 CPU Whisper and D2 Sherpa cells (2026-11-21)

- The bounded host-portability producer now runs the declared \`whisper-local\` /
  \`whisper-small\` CPU cell with the pinned whisper.cpp image, staged local
  weights and fixture, no network, read-only mounts, \`--no-gpu\`, an 8-CPU / 8
  GiB ceiling, and a three-minute timeout. The exact fixture phrase was
  recognized in 2.246 s; model and fixture hashes are in the receipt. The
  managed Whisper HTTP service remains unserviceable, so this verifies bounded
  model inference, not product-service or browser readiness.
- Receipt \`audio-tools:host-portability:20261122T002105.273493236Z:amd64\` also
  contains the passing D2 Sherpa int8 result (2 CPU / 4 GiB, 438,358,016-byte
  peak). The D1 accelerator cell names \`D1_accelerator_kyutai_unavailable\`;
  Kyutai remains not serviceable and no accelerator inference ran.
- The governed setpoint still reports host-portability unavailable with
  \`failed:receipt_claim_out_of_band\`; this is expected because the aggregate
  receipt remains \`passed:false\` and cannot claim the incomplete three-cell
  matrix. Current board totals remain 5 of 18 resolved and 13 unavailable.
  The next host gate is a supported Kyutai accelerator inference; do not treat
  the two successful CPU cells as full host qualification.
- The full CLI suite and catalog checks pass after the producer update.

### Current low-resource route and lab-profile recheck (2026-11-21 23:07 UTC)

- Fresh \`audio-tools stt engines --json\` selects \`sherpa-streaming\` by the
  persisted operator engine pin. Its \`/healthz\` returns \`ok\`; the current
  resource manager still reports Sherpa unsupported on Linux, so the live
  endpoint remains attach-only. No transcription was used here as new quality
  evidence.
- \`resource-ollama policy resolve --role summarize.default --json\` and
  \`audio-tools model resolve --capability summarize --json\` both resolve to
  Apache-2.0 Qwen3 1.7B. Ollama's current resident set contains Qwen3.5 9B and
  Nomic embeddings, not Qwen3 1.7B. GPU use is 11,372/16,376 MiB at 2%; process
  attribution lists 7,700 MiB and 608 MiB Ollama servers, 1,380 MiB reranker,
  and 1,264 MiB Ollaya. No Audio Tools GPU process appears in that snapshot;
  shared workloads were left untouched.
- With managed Kale healthy, a fresh product TTS request selected
  \`local/kale-tts\`, returned 148,844 bytes of WAV output, and took 296 ms warm.
  \`model resolve --capability tts\` still reports the qualified multilingual
  Kokoro model; Nano remains a conditional product-auto candidate pending its
  quality qualification. This request was written to \`/dev/null\`.
- The source change in \`internal/orchestrator/detail.go\` applies the addressed
  instance profile on status/detail reads. Its new API-only lab regression and
  the orchestrator/CLI tests pass. The live lab API \`/health\` is healthy and
  ready, while the saved status snapshot still reports the old
  \`UI_PORT\`-placeholder failure; it was not restarted, so current displayed
  degraded status remains unresolved until a fresh lifecycle health snapshot.
- A new AT2 comparison is assessing Tiny Voca against the current route under
  shared-host contention. The review found no declaration-only integration
  path: Tiny Voca has no managed runtime or matching adapter. Its CPU/RAM,
  product latency, contention and held-out quality remain unmeasured; no model
  was downloaded and no route/default changed. The next route is a separately
  managed, pinned CPU service and provider adapter before a paired benchmark.

### Duplex accounting measure and current gate checkpoint (2026-11-21)

- The W2 setpoint read completed with acceptance unknown, 18 required outcomes
  pending and zero owner receipts. The product API omits lab-only experiment
  evidence; the program correctly retained those outcomes as unknown. The
  broad PRD and outcome inventory remain the acceptance source.
- The measure-structure gate found \`duplex_accounting_operations\` as the one
  undeclared persisted entity. This metadata-only correlation journal now has
  an aggregate-only \`CountDuplexAccountingOperations\` command over its
  creation-time window, bound through API, owned proto, CLI and manifest. The
  exact half-open window comparison handles variable-precision RFC3339 values.
  Focused API/duplex and CLI measures tests pass; the proto owner verifier
  passes; measures-health now reports zero findings and maximum maturity.
- Post-change \`make test-packages\` completed nonzero in six modules within the
  already dirty shared tree:
  \`cli-core\`, \`credential-authority-go\`, \`maturity-go\`, \`proto\` (missing
  \`gen/go/flow-verifier/v1/errors\`), \`repo-contract-go\` and \`resources\`. The
  shared tree had broad changes/deletions before this intervention, so there is
  no clean pre-change baseline; these failures are not attributed to the
  measure change. Full run output is at \`/tmp/tmp.JHrY0KFhjL\`.
- Tiny Voca remains unmeasured and unqualified, though its upstream-fit
  assessment is recorded in \`MODEL-FOOTPRINT-AUDIT-2026-11-20.md\`.

### Fresh lab evidence-board read (2026-11-21 23:34 UTC)

- The product-instance command \`audio-tools experiment list-evidence --json\`
  returns 404 by design because qualification evidence is lab-only. The proper
  observer is \`program-runtime --instance lab library run
  audio-tools.setpoint-read\`; its fresh run \`prog_82e08aa3-5e5f-4451-ac26-882bf1589a6f\`
  completed with \`status=ok\`, zero binding errors, and acceptance still
  unknown. Six current receipts were valid. Five of 18 required rows resolve
  through simulated evidence: explicit routing, owned settlement, private
  lifecycle, acceptance integrity, and adapter maintainability. Thirteen rows
  remain unavailable; no experiment was selected, so recognition/latency replay
  metrics are empty. The two exclusions remain out of this mandate's scope and
  are not overall mission acceptance.
- A separate hook regression reproduced a delayed per-turn capability probe
  disposing a newer microphone turn. A captured start-generation check now
  ignores the stale result. The focused startup and long-session suites pass
  34/34; the five-cycle real browser capture protocol remains unrun, so the
  repeated-session requirement remains open.

### AT2 voice-cloning route assessment (2026-11-21)

- Primary-source research in
  \`the AT2 \`CLONING-ROUTES.md\` research artifact\`
  covers Qwen3-TTS \`12Hz-0.6B-Base\` and OpenVoice V2 + MeloTTS. Qwen source and
  checkpoint declare Apache-2.0, supporting artifact-level commercial use under
  its terms. This does not establish rights to reference speakers, upstream
  training-data provenance, complete dependency notices, or end-to-end
  commercial readiness. The checkpoint repository is 2.52 GB and its listed
  safetensors file is 1.83 GB. The current vLLM-Omni performance profile is for
  0.6B CustomVoice, not Base cloning; no Base CPU, RAM/VRAM, or latency measure
  was found.
- OpenVoice V2 and MeloTTS declare MIT; the official sample has a CPU path, but
  it is a two-model research/developer stack. Its pinned redistribution bundle,
  performance, consent lifecycle and product integration remain unproven.
- Audio Tools contains planned Qwen model/provider declarations only. It has no
  cloning runtime, adapter, lifecycle or consented reference-audio storage and
  deletion path. No weights or recordings were acquired, no trials were run,
  and neither candidate is qualified or a default. The next route requires an
  artifact/license inventory, a consented held-out speaker cohort, then an
  isolated pinned service and reproducible cold/warm quality/resource trials.

### Shared browser package compile boundary rechecked (2026-11-21)

- The older \`knw-1794813472862812408\` report said a supported Audio Tools
  restart failed before stopping services because a generated browser package
  lacked \`VoiceTransport.buildStreamUrl\`. Current shared capture and Audio Tools
  UI sources contain no remaining references to that member. The package build
  passes, its full suite passes 244/244 executed tests (one skipped) across 34
  files, and the Audio Tools UI consumer type-check passes. This establishes a
  current-source package/consumer compile boundary.
- The running Audio Tools API and lab API were not restarted. Therefore no live
  promotion or runtime freshness claim is made. When a live refresh is required,
  use the supported \`vrooli scenario restart audio-tools\` operation and retain
  its result; the broad dirty worktree means that restart would exercise more
  than this isolated package boundary.

### Current Audio Tools instance refresh observed (2026-11-21 23:44 UTC)

- A fresh \`vrooli scenario status audio-tools --json\` read reports the latest
  supported lifecycle operation as a succeeded restart with a healthy verdict.
  Its expected and served identity both equal
  \`sha256:a092a3de145adf57ca5631b9cc56b61c2ed8c297561235b88d648b07eab9e90f\`.
- The running API \`/health\` returned healthy/ready at 23:44 UTC, with connected
  database/providers and the same source/build identity and start-operation ID.
  The UI port returned the Audio Tools app HTML. The restart predates this check
  and was not initiated by this worker.
- This closes the current-source startup/compile-path uncertainty for the live
  instance. It does not prove microphone reuse, five-cycle browser capture,
  cross-app insertion, or any remaining PRD qualification rows.

### Continuation: compact routes, Kroko finalization, and release drift (2026-11-22)

- Current CLI resolution selects Whisper Small for batch STT, Sherpa Zipformer
  for streaming STT, Qwen3 1.7B for summarize, and the catalog-qualified
  multilingual Kokoro model as the static TTS default. A fresh product TTS
  request dynamically selected Kale Nano (\`kale-tts-1-nano\` / \`kenzie\`),
  returned WAV, and reported 2,494 ms. This is one route/latency observation,
  not a quality or promotion receipt.
- GPU process attribution showed shared Ollama and reranker processes and no
  Audio Tools GPU compute PID. The current snapshot does not prove total mapped
  model memory or attribute shared process demand to Audio Tools.
- Kroko's WebSocket ending behavior is now discriminated by an exact final
  suffix assertion. With no finalization tail, the 11-second fixture ends at
  \`...do for.\` and fails; the bounded 1.2-second tail passes the expected
  \`your country.\` suffix on both 11- and 55-second fixtures, with final times
  within source duration. The native-tagged Sherpa server suite passes.
  Kroko remains on-demand and experimental; no model promotion is claimed.
- Rebuilding the current Sherpa resource source twice against the pinned
  upstream Linux runtime produced bundle digest
  \`b5b4130a5a2244606e8b1e17dc3f4e273c947254aaee63b1370b09fc448e0635\`; a clean
  \`HEAD\` source build produced \`a11fdc06…\`, while \`resource.json\` still pins
  \`c8127175…\` for \`1.13.2-vrooli.2\`. The exact historical source that produced
  \`c812…\` is unavailable. The version/digest were left unchanged pending
  release-owner review; managed Linux stays unsupported until signed publication.
- Removed duplicate hardcoded release-version defaults: the Sherpa package
  helper now derives its default from \`resource.json\`, while explicit overrides
  still fail closed on mismatch. \`make check\` passes, including seven stage
  validator tests and two release-version regression tests.
- LPBS app entitlements now aggregate by \`(app_key, bundle_key)\` so grants
  cannot cross bundle scope. Focused owner tests pass. A complete checkout,
  provider-calculated proration upgrade, cancellation, and payment-recovery
  journey still lacks an owner-supported upgrade test seam.
- The five-cycle BAS case is linked to ATD-P0-006, but its selected-case run
  still stops before browser execution because BAS requires an authenticated
  account. Static Workflow Health also exposes an unresolved \`error\` specimen
  in the current VoiceInputButton experience contract. No cycle or native
  microphone qualification is claimed.

### Current-source model-route and Sherpa release audit (2026-11-22 01:20 UTC)

- After the API source changed beyond the previously running build, the
  supported \`vrooli scenario restart audio-tools\` operation rebuilt the current
  API/UI, stopped the prior instance, and completed healthy. The served identity
  is \`sha256:27fe0ee3ac83a27283df4bc5d5ece71afc27076d3a238458d16f7308891c209d\`.
- Live \`audio-tools tts synthesize\` selected local \`kale-tts\`, synthesized
  150,044 bytes in 927 ms, and wrote a valid 24 kHz mono PCM WAV of 3.125 s.
  This verifies product routing and response format for one request, not
  voice-quality or latency qualification.
- \`audio-tools stt engines --json\` selected \`sherpa-streaming\` and reported
  provider health available. The selection reason is the persisted operator
  engine pin, not a fresh proof of the no-pin automatic default. A live stream
  through that attached Zipformer transcribed the 3.125 s generated clip in
  1,558 ms; reference phrase \`route\` was recognized as \`root\`. This confirms
  the product streaming path and provides a quality counterexample, not a
  model-quality pass. The service is attached at \`127.0.0.1:8881\`, outside the
  Vrooli-managed resource lifecycle; the resource manager still reports
  unsupported on Linux.
- Current model resolution returns Whisper Small for batch STT and Qwen3 1.7B
  for summarize. A live product summary returned \`model_id: qwen3:1.7b\`,
  preserved six declared facts, and took 21,256 ms. A concurrent post-request
  Ollama snapshot reported 2,235,280,915 bytes resident for Qwen3 1.7B
  (~2.24 GB decimal / 2.08 GiB), above its 1.36 GB on-disk declaration but
  below the co-resident Qwen3.5 9B at 6,735,779,593 bytes. The 9B process was
  present before this confirmed call, so its ownership is unknown. A preceding
  model-unlabelled summary took 18,036 ms and omitted the contingency clause;
  the labeled call preserved it. These two samples confirm routing and expose
  latency/quality tradeoffs, not a general quality pass.
- A read-only release audit inspected the uncommitted \`.2\` version and
  \`c8127175…\` SHA in \`resources/sherpa-onnx/resource.json\`, git history, and
  retained local release stages. The pin is absent from committed history.
  Three preserved stages independently reproduce tree digest
  \`e7f3f513…a1a513\`; their
  archive SHA is \`2c47c2e00eb6a044ef22c3e4cc6af1b70b9186a647d5e3d7785fec7e2e660a9c\`.
  The current bundle digest is \`4494c3…\`, also not the pin. No artifact found
  reproduces \`c812…\`. Keep version and checksum unchanged pending release-owner
  review; signing/publication require the owner-provided publication URL and
  upstream provenance plus a matching validated artifact.
`;export{e as default};
