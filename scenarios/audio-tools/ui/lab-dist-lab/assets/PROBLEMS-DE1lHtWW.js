const e=`# Problems — Audio Tools

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## 2026-11-24 — Constrained summarize resolution reported a capacity limit as a license gap

The D2 Linux profile reports 4 GiB RAM, while the catalog declares a 6 GiB
minimum for the selected Qwen3 1.7B summarize model. Resolution correctly
refused that profile, but the API labeled the result “no commercial-eligible
model,” even though the model is Apache-2.0 and commercially allowed. The model
catalog now distinguishes host-excluded defaults from absent commercial models
and includes the model's unmet RAM floor in the error. Focused catalog and model
API tests pass. The D2 CPU quota remains informational because no evidence-backed
minimum CPU requirement is declared. This source change has not been restarted
into the shared lab process; its earlier setpoint receipt still reflects the old
message.

## 2026-11-24 — Fixed-input Sherpa latency and JFK recognition remain model-bound

The same 11-second JFK WAV (SHA-256
\`59dfb9a4acb36fe2a2affc14bacbee2920ff435cb13cc314a08c13f66ba7860e\`) was
decoded five times through the native Sherpa adapter at 100 ms PCM cadence.
Each run emitted its first partial after 1,200 ms of captured audio and about
1,217–1,221 ms wall time. This repeatable result agrees with the browser
cohort's roughly 1,074 ms post-onset audio accumulation plus small delivery and
render overhead; capture readiness and transport are not the limiting path.

All five final transcripts were the same: “What's your country can do for you
as, but you can do for your country.” They did not match the checked JFK
opening (“Ask not what your country can do for you”). The focused smoke
therefore failed its exact-reference assertion on all five runs; that is
negative quality evidence, not a runtime crash. The test used the current
310 MB Zipformer with sherpa-onnx v1.13.2, one synthetic public-speech fixture,
and no held-out corpus. Do not reduce decoder context, treat latency as solved,
or promote a different model from this evidence. The smaller 20M and Kroko
models were slower to first text in the earlier single-run comparison, and
Kroko's commercial terms remain unresolved. Current repeat command was the
native \`TestNativeStreamingModelSmoke\` with \`-count=5\`, the fixture above, and
\`SHERPA_ONNX_STREAMING_PACE_MS=100\`.

## 2026-11-24 — Thread count and int8 do not close Sherpa first-text latency

On the same 11-second JFK fixture and native Sherpa v1.13.2 path, 1, 2, and 4
threads each emitted first text after 1,200 ms of captured audio. Wall times
were 1,223.1, 1,218.8, and 1,216.5 ms, with the same final transcript and the
same failed JFK opening assertion. An int8 encoder/joiner reduced measured
model-feed time from 584 ms to 450 ms at two threads, but first text still
arrived after 1,200 ms and the opening assertion still failed. This supports
the existing diagnosis that first-text delay is dominated by model context
accumulation, not thread scheduling or post-arrival compute. It is one native
smoke per configuration, not a product p95 or quality result. Do not change
the default from this probe; the next useful experiment needs a streaming
configuration or candidate model that passes both opening and ending checks,
then a 30-attempt product browser cohort.

## 2026-11-23 — Paced local browser first-partial p95 remains above target

A fresh 30-attempt Chrome fake-microphone cohort exercised Dictation Studio,
the shared browser capture package, the Audio Tools Connect stream, and local
Sherpa Zipformer. All 30 turns completed with at least two rendered transcript
revisions; every turn sent its completion marker, processed all sent sequences,
and reconciled captured samples to committed samples. No browser page errors
occurred. Click-to-capture-ready p95 was 20.6 ms, visible update-gap p95 was
352.1 ms, and stop-to-final-tail p95 was 165.8 ms.

Speech-to-visible-first-partial p50/p95/max was 1,098.3/1,161.2/1,195.9 ms,
against the adopted 1,000 ms p95 and 2,000 ms maximum. Available post-onset
audio p95 was 1,117.0 ms; excess wall time after that audio span was 55.5 ms
p95 (111.6 ms maximum). This supports speech/context accumulation as the
dominant latency in this cohort and gives no basis for reducing the 50 ms wire
batch. This is synthetic-media evidence only: no independent transcript score,
physical microphone, other host/browser, cold-start, or interruption/resource
pressure claim is made. Receipt:
[\`browser-paced-sherpa-30-20261123.json\`](../../coverage/browser-paced-sherpa-30-20261123.json).

## 2026-11-24 — Smaller Zipformer candidates do not yet close first-text latency

An opt-in native Sherpa comparison fed one 11-second JFK fixture in 100 ms
PCM packets. The selected 310 MB Apache-2.0 Zipformer produced first text after
1,200 ms of captured audio (1,217 ms wall). The 20M Apache-2.0 model reduced
feed-call wall time from 580 ms to 238 ms and loaded faster, but its first text
arrived after 2,100 ms of captured audio (2,107 ms wall). The 68 MB Kroko
candidate matched the checked reference opening and final suffix, with first
text after 1,500 ms (1,534 ms wall); its CC-BY-SA commercial/OEM terms remain
unconfirmed. Each model has one sample. The test adapter is not the product
browser route and measures neither peak memory nor held-out quality. No model
change is justified; retain the 310 MB default pending the existing 30-run
product acceptance and corpus/license evidence. Receipt:
[\`native-sherpa-model-first-text-20261124.json\`](../../coverage/native-sherpa-model-first-text-20261124.json).

## 2026-11-24 — Moonshine Tiny browser smoke does not show a faster first interim

An isolated headless Chrome harness loaded the official Moonshine Tiny
Streaming WASM package and fed the same 11-second JFK fake-microphone fixture.
The model load completed in 441 ms; first interim text arrived at 2,081 ms
from page start, or about 1,567 ms after the microphone entered its listening
state. The callback contained only “And so, my,” before later final callbacks
completed the opening phrase. This harness did not measure acoustic onset,
React-visible product timing, held-out quality, resource use, or cache-cold
installation cost, so its numbers are not directly comparable to the Sherpa
30-attempt product cohort. It provides no evidence that Moonshine Tiny would
close the current first-visible p95 gap. No package or model was added to the
scenario, and no default or provider route changed. A paced product-path cohort
and scored corpus would be required before revisiting this candidate.

## Program view (2026-11-19)

The dated entries below are the incident register. The *program* view of debt,
sequencing, and exit gates lives in
[MATURATION-PROGRAM.md](MATURATION-PROGRAM.md). As of 2026-11-19 the
highest-leverage open classes are:

- **Declaration drift (fixed in the 2026-11-19 tranche, kept as the motivating
  example)** — \`sttengine\` declared two engines while bootstrap constructed
  three; provider ids were spread across ~20 hand-synced files. The provider
  catalog and conformance gate now make that shape fail the build.
- **Unshipped capability (fixed in the 2026-11-19 tranche)** — seven of the
  eight audio transforms existed in proto/handlers but had no CLI or UI
  journey; all eight are now declared in \`operationcatalog\` and reachable from
  both surfaces.
- **Model provenance (registry delivered; qualification partial)** — the
  registry now declares licence, readiness, hardware, per-language facts, and
  the resource artifact that supplies local weights, and serves them through
  \`ModelRegistryService\`, the CLI, and the UI. Constrained defaults are
  Whisper small and the shared \`summarize.default\` role; a one-host product-path
  smoke served the canonical clip with \`whisper-small\` in 75 ms and 716 MiB
  measured VRAM. This is not broad dictation evidence. No timed long-form
  reference exists for timestamp-drift scoring, and no device/browser/OS
  matrix exists.
- **Unenforced lab code** — \`experiment\`, \`eval\`, \`soak\`, and \`validation\` had
  no coverage floor; \`stt/pipeline\` was enforced at 20%. Floors now exist for
  all of them, and the chain/handler floors are back at their original values;
  raising them further is test work, not a policy change.
- **Long-tail reliability** — the 2026-09 long-turn recovery, Kyutai admission,
  and latency entries below remain the evidence backlog, not closed claims.

Do not treat a green unit suite or a source-presence check as evidence that any
of these is closed. Close them with the gates named in the maturation program.

### 2026-11-23 — Sherpa browser first-interim latency remains above target

Three paced headless-Chrome fake-microphone runs through the product Dictation
Studio route selected \`sherpa-onnx/streaming-zipformer-en-2023-06-26\` and
measured speech-to-rendered-interim latency from the browser diagnostic at
1,094–1,109 ms. The adopted p95 target is at most 1,000 ms and the maximum is
2,000 ms, so this small diagnostic sample is out of band and is not a
qualification cohort. Button-to-capture readiness was 72–167 ms; the final
transcript kept the known ending in all three runs; stop-to-captured wall time
was 153–207 ms; all 224–226 sent chunks were processed. Fixture:
\`/tmp/opencode/jfk.wav\`, SHA-256
\`59dfb9a4acb36fe2a2affc14bacbee2920ff435cb13cc314a08c13f66ba7860e\`.

The first trace also showed \`stopToFinalTailMs\` was null because Dictation
Studio did not mark the user's stop action. The recorder now records that
marker before calling \`provider.stop()\`, with an ordering regression test.
The running UI bundle predates the stop marker and the new first-partial
context telemetry. That telemetry records the exact first-signal sample,
captured cursor at the first visible partial, and input-audio span available
then. Its excess-latency measure estimates lag beyond captured audio, but is
not a server-only decode timer. Collect it with the processed cursor on a fresh
product trace before tuning Sherpa; physical-microphone, full cohort, and
device/browser/OS qualification remain open.

**Update — 2026-11-23:** The shared capture package was rebuilt, its local UI
\`file:\` dependency refreshed, and the managed \`dist-live\` UI bundle rebuilt in
place. A fresh one-attempt synthetic-media product trace now records the new
fields (receipt:
\`coverage/browser-first-partial-context-20261123.json\`). First visible text
took 1,096.7 ms after the signal proxy, with 1,094.9 ms of post-onset audio
already captured and 1.7 ms excess wall latency. Capture readiness was 13.1 ms;
stop-to-final-tail was 87.3 ms; sequence 250 was sent and processed, and
captured/committed samples both ended at 200,363. The browser reported no page
errors. This points to the selected model's initial context accumulation,
not browser or transport backlog, as the useful latency investigation. No
batch-size change was made; the measured excess is far below the current 50 ms
wire cadence. This remains a single synthetic-media diagnostic, not a p95,
quality, physical-microphone, or device/browser/OS qualification.

### 2026-11-22 — Shared Ollama residency does not match idle policy

The managed Ollama policy declares only \`embedding.default\` and \`chat.small\`
resident and sets \`idle_unload_seconds\` to 600. The live \`/api/ps\` snapshot at
2026-11-22 19:02 local still showed \`gemma4:12b\` loaded, with an \`expires_at\`
of 12:11 local (already elapsed); the callers inventory's latest attributed
request for that model was 2026-11-21 22:36 UTC. \`gemma4:12b\` remains required
as a primary for shared \`code.local\`, \`image.quality.judge\`, and \`write.default\`
roles, so it was not evicted. \`qwen3.5:4b\` is intentionally resident as
\`chat.small\` primary and had 1,054 tracked requests, 864 attributed to
AI Gateway; Audio Tools' \`qwen3:1.7b\` role was not loaded. The activity edge
was alive. The observed expiry mismatch has no confirmed cause yet (unattributed
traffic or a missed policy reassertion remain possible); investigate at the
Ollama/resource owner before changing global residency or claiming memory
reduction. No model was unloaded and no shared service was restarted.

### 2026-11-22 — Connect streaming could wait forever on a quiet client

The bidi handler's synchronous \`Receive()\` has no per-read cancellation or
deadline, so a client that sent \`StreamStart\` and then stopped transmitting
could leave the handler and its Segmenter alive indefinitely. A goroutine-local
timeout would only abandon a blocked receive and leak its worker. The handler
now watches an injectable idle timer (30 s by default), reset by each audio
chunk. Expiry cancels the Segmenter, persists \`input_idle_timeout\`, and returns
without joining the receive worker; returning tears down the Connect transport
and unblocks \`Receive()\`. Quiet timeout persistence and sustained activity are
covered by handler tests; the targeted handler race suite and full
\`TestTranscribeStream_\` subset pass. This bounds idle streams; it does not add
an absolute maximum session duration.

### 2026-11-22 — Hour-scale qualification stopped at client and ledger limits

The long-running STT qualification used the ordinary 120-second HTTP client
timeout, reused a fixed durable session identity across retries, and streamed
a tone that never produced committed segments. The first two conditions caused
early termination; after they were fixed, the retained session audio reached
the server's 64 MiB ledger limit after about 35 minutes. The qualification
client now relies on its bounded run context, assigns a cryptographically
random session identity to each stream, and sends \`End\`/closes a session at 16
MiB before opening the next. Focused STT/validation race tests pass. The 60-
minute lab run then completed, with 360 TTS probes and 35,994 STT chunks; the
freshness check passes. This qualifies synthetic transport continuity only,
not recognition accuracy, timed drift, provider parity, or device support.

### 2026-11-22 — CLI streaming STT ignored the invocation API override

The Connect streaming client captured \`APIRootBase()\` when the Audio Tools CLI
was constructed, before global flags were parsed. The unary clients use
\`cli-core\`'s dynamic request rebasing; the custom HTTP/2 streaming transport
sent its captured absolute URL unchanged. This meant a command specifying
\`--api-base http://127.0.0.1:15373 --instance lab\` could still post to the
product API at 19630. The streaming transport now rebases requests rooted at
its construction URL against the invocation's selected API at \`Do\` time, and
a fake-transport test parses both flags and asserts the selected host and route.
\`go test ./domains/stt -race -count=1\` passes. A source-built CLI stream of
the bundled short synthetic fixture then completed against lab port 15373:
the lab log recorded a 21.75 s \`TranscribeStream\`, and its terminal event was
\`completed\` on Sherpa Zipformer with one segment. The latest product log entry
remains the earlier 07:05:34 request; this verification did not reach product.

One previous diagnostic invocation with those explicit lab flags is strongly
correlated with a product \`TranscribeStream\` log at 07:05:34 (34.330 s); the
CLI output files were written at the same time and reported substantial
processing followed by \`input_failed\`. This is strong but indirect evidence
that the request reached the product API. The stream handler persists session
ledger state, so a product-side ledger/spool may have been written. No
transcript text was displayed or used as evidence, and no local artifact was
inspected or deleted. The request itself necessarily processed its source
audio. The possible product-side session artifact was left untouched. Treat
the CLI routing test as required before future live streaming probes. The
corrected source-built CLI has since run the same AMI file against the lab API:
it completed with terminal reason \`completed\`, 115 segments, and all 1,242
audio sequences acknowledged as processed. The lab request took 1m58.9s for
1,272.64 seconds of audio (about 10.7x real-time); this is a single-host
throughput observation, not a quality or timestamp qualification. Product log
has no newer stream request. This supports that the earlier failure was
caused by the captured-base routing defect, while leaving the possible prior
product ledger/spool untouched.

### 2026-11-20 — Current host model residency is outside the Audio Tools path

A read-only host spot check after Audio Tools moved to lighter defaults
showed Whisper stopped, sherpa-onnx uninstalled/unsupported on this
Linux host because its signed adapter bundle is unpublished, and Kale installed
but stopped. At the time of this check the sherpa endpoint was live on the
attach-only port and the Audio Tools engine list selected it; the control plane
does not own that process or its lifecycle. This is a working local attach, not
a managed-resource qualification.

Ollama's \`/api/ps\` showed \`qwen3.5:9b\` resident at 5.88 GB plus
\`nomic-embed-text:latest\` at 323 MB, while \`qwen3:1.7b\` was not loaded. The
resource policy assigns \`qwen3:1.7b\` to \`summarize.default\` and \`qwen3.5:9b\` to
\`chat.default\`; the remaining large allocation is therefore shared chat
workload, not Audio Tools summarization. Other active GPU processes included
the reranker and Ollaya. No model was unloaded because those shared consumers
were not identified as idle or Audio Tools-owned. Recheck residency and
ownership before making a shared-resource change.

**2026-11-23 update:** the earlier 498 MiB Kale runtime bundle was replaced by
the pinned upstream CPython \`install_only_stripped\` distribution. The managed
resource was reacquired and restarted through its CLI; artifact
\`cf38c6dc79ff578c40454b79c12b35356f6c4049de43b013b525d3fb95facb63\` is now
healthy and occupies 257 MiB on disk. A current Audio Tools product route
selected Kale Nano after restart, and the documented 20-request sweep passed
all ten voices. This closes the old packaging and route-execution gaps, not the
human naturalness or held-out speech-quality gate; the static catalog default
remains Kokoro.

Follow-up snapshot, 2026-11-20 20:22 UTC: the Audio Tools API was healthy at
28,704 KiB RSS. The attached sherpa \`/healthz\` endpoint remained healthy and
\`audio-tools stt engines --json\` selected \`sherpa-streaming\`; its source-built
CPU process showed 8,288 KiB RSS while idle. The managed sherpa resource still
reports unsupported on Linux because its signed adapter release is unpublished.
GPU use was 9,288 MiB / 16,376 MiB. Ollama still reported the 5.88 GB
\`qwen3.5:9b\` and 323 MB embedding model, but not \`qwen3:1.7b\`; policy maps the
9B model to shared \`chat.default\` and the 1.7B model to Audio Tools
\`summarize.default\`. The Kale resource was stopped after its local synthesis
smoke, and the running Audio Tools API was not configured to route TTS through
Kale. No shared models or product processes were unloaded or restarted.

The 2026-09-16 50 ms PCM wire-batch experiment is rejected. Focused package
tests showed duplicate replay frames after reconnect and a simulated-hour wire
rate of 20/s against the 15/s bound. The production batch remains 1,600
samples (about 100 ms); startup p95 must be repaired at the decoder/admission
owner rather than by increasing transport event pressure.

### 2026-11-20 — Host resource attribution and Whisper-small product check

An idle host sample showed the Audio Tools API at ~24 MiB RSS with no active
Audio Tools model process. GPU use was attributed to the shared Ollama daemon
and reranker, not Audio Tools; \`vrooli capacity reconcile\` measured Ollama at
7.30 GiB against a 5.78 GiB claim. The process inventory did not identify
which scenario last requested the resident \`qwen3.5:9b\`, so it was left
untouched. This shared-resource overclaim is not an Audio Tools default change.

The managed Whisper resource started with its declared \`small\` artifact. The
product CLI transcribed the canonical five-word clip exactly in 75 ms and
reported model \`whisper-small\`; the capacity ledger measured a 716 MiB VRAM
peak under its 1 GiB claim. The resource was stopped after the check and the
owner-scoped claim was absent. This confirms the current light default on one
host and one clip only. The long-form fixture has no timed transcript, and the
product response does not expose word intervals, so timestamp drift remains
unmeasured at that time. The later timing work below supersedes the response-
surface statement, but not the missing reference or qualification. A first start command timed out during claim renewal despite the
resource reaching healthy; a stop command likewise timed out during claim
release, though the final status was stopped and the claim was subsequently
absent. These lifecycle timeouts are recorded as an observation, not as a
change to host-control policy.

### 2026-11-20 — Hypothesis timings retained; qualification still needs gold

The stable unary \`TranscribeResponse\` now exposes provider-reported
\`word_timestamps\` when egress accepts the transcript and source coordinates are
trustworthy. Filtered transcripts and speaker-policy transformed audio do not
expose intervals. Eval replay also keeps provider-reported hypothesis timings
in each clip report, capped at 20,000 words with a truncation flag; it does not
persist audio. Native sherpa streaming segments do not currently carry word
intervals, and no human-reviewed long-form reference exists, so
\`TimestampDrift\` remains unset and no model gets a timestamp qualification.

The host portability producer previously treated \`ListEngines().Available\` as
D1 inference evidence and used an unrelated Docker resource-control probe for
D2. It now records inventory separately, marks D1 CPU, D1 accelerator, and D2
inference as \`not_run\`, and persists an incomplete receipt. The acceptance board
and owner join require all three cells to say \`available\` and \`inference=passed\`.
No such inference receipt exists on this host.

A live attach-only sherpa stream against the bundled five-word clean-speech
fixture returned partials and a final segment, but omitted “jumps” (“The quick
brown fox” versus “The quick brown fox jumps”; 20% WER on this synthetic
diagnostic clip). This is a route smoke, not a promotion or dictation claim. It
also exposed that native streaming segment latency was unset; the provider now
measures elapsed fractional milliseconds from request start through segment
receipt, with a regression test. The running API binary was not restarted for
that source change.

### 2026-11-20 — Sherpa attach, compact summarize, and a decoder gap

Audio Tools moved to the low-footprint path: sherpa streaming for STT, Kale
nano for TTS when release-ready, and the shared Ollama \`summarize.default\` role
for summaries. The STT and summarize paths are wired. Kale Nano/Tiny weights are now publicly
accessible with checksums; Hugging Face API metadata for Nano's pinned
revision reports public, non-gated Apache-2.0 status (checked 2026-11-21),
superseding earlier draft-status notes. Tiny remains draft; the current sherpa
Kokoro adapter cannot run these graphs. Nano now
has a separate pinned Linux/amd64 CPU resource candidate and an explicit API
route. Its managed service has since started healthy and listed all ten pinned
voices, but synthesis and product-path quality remain unverified, so Kokoro
stays the TTS default.

**Summarize** moved from \`qwen3.5:4b\` to \`qwen3:1.7b\` (Apache-2.0, 1.36 GB).
Measured on this host: the 1.7B tag preserved 8/8 numbers and dates on a short
report and 14/22 on a 300-word report, where it dropped the final paragraph;
the 4B tag preserved 22/22. The 4B tag is the first fallback and quality
escalation. The ai-gateway provider catalog was regenerated, and the ollama
resource gained the required \`qwen3:1.7b\` acceleration rung (3 GiB) — without
it, the scenario correctly refused to start.

**Sherpa streaming** is now attached and selected. The source-built adapter
runs on \`127.0.0.1:8881\` with the 2023 English Zipformer (310 MB, CPU) and the
punctuation model. Two integration defects were fixed: Audio Tools pointed at
port \`8880\` while the resource exposes \`8881\`, and the capability checkers
probed \`/ready\`, which exists only when the speaker runtime is loaded; the
sherpa STT probe now uses \`/healthz\`. The live preference order is
\`low_footprint\` → \`native_streaming\` → \`accelerated\`, so Sherpa wins over the
2.3 GB Kyutai tier when both are serviceable.

Attach-only is explicit: when the control plane reports the managed artifact
unsupported/uninstalled, a passing provider health probe still makes the engine
\`available\` and the selection reason says \`attached (provider health confirmed,
lifecycle ownership is external)\`. A dead attached provider is rejected. The
resource manifest still claims no platform support until a signed release
artifact exists; see \`resources/sherpa-onnx/docs/LOCAL-ATTACH.md\`.

**Kroko 2025** (68 MB, CC-BY-SA-4.0) initially lost final words when the native
stream was finished immediately after speech. A native model experiment
reproduced “for.” without padding and retained “your country.” after a 1.2 s
silent tail. \`Finish()\` now adds a bounded tail without advancing client
timestamps; the opt-in smoke test checks the final phrase and that timestamps
stay within source PCM. Kroko remains on-demand pending corpus accuracy,
latency, timestamp, and license receipts; this single JFK smoke does not
support default promotion.

**Local TTS integration** now allows \`AUDIO_TTS_BACKEND=sherpa-onnx\` to route
the existing Kokoro-compatible provider through sherpa's shared speech
contract. The default remains the managed Kokoro resource. Request-path
liveness uses sherpa \`/healthz\` (model + encoder loaded) rather than an extra
synthesis on every short freshness window; full readiness sweeps still
exercise synthesis. Hugging Face API metadata at Nano's pinned revision reports
public, non-gated Apache-2.0 status (checked 2026-11-21); Tiny remains draft.
The isolated managed runtime contract
and explicit \`AUDIO_TTS_BACKEND=kale-onnx\` API route are experimental; Kale is
not promoted in the provider catalog until its installable resource, pinned
dependency closure, and native runtime receipts are complete. This route is
separate from sherpa's Kokoro adapter.

### 2026-11-20 — Kroko finalization tail and sherpa-hosted Kokoro route

Resolved immediate-end truncation in the native sherpa online stream. Direct
\`TestNativeStreamingModelSmoke\` against the local Kroko 2025 model and JFK PCM
ended at “for.” before the fix; appending 1.2 s of silence produced “your
country.” \`Finish()\` now feeds a bounded silent tail before \`InputFinished\` and
deliberately leaves \`totalSamples\` unchanged. The opt-in native smoke test can
assert \`SHERPA_ONNX_STREAMING_EXPECT_TEXT\` and bounds the last event timestamp
by client PCM length. This narrows the finding to a finalization issue; it does
not qualify broad accuracy or promote Kroko.

**2026-11-21 follow-up:** The resource adapter also assumed dated Zipformer
filenames, while the declared 68 MB Kroko archive uses \`encoder.onnx\`,
\`decoder.onnx\`, and \`joiner.onnx\`. The native factory now discovers either
layout and still honors explicit file overrides. A model-backed WebSocket smoke
using the existing local Kroko artifact and a 55-second JFK fixture passed; the
final “your country” phrase survived and its timestamp stayed inside source
duration. \`TestResolveStreamingModelFile\` covers legacy/generic layouts and
override precedence. Kroko remains on-demand because this is a narrow smoke and
its CC-BY-SA-4.0 terms do not qualify it as a commercial default.

The WebSocket smoke now supports \`SHERPA_ONNX_STREAMING_EXPECT_FINAL_SUFFIX\`,
which asserts actual final words rather than only finding the expected phrase
somewhere in the transcript. This distinction exposed that the older
\`SHERPA_ONNX_STREAMING_EXPECT_TEXT\` substring assertion could pass on a
truncated result. On the 11-second JFK clip, a 1.2-second finish tail passes
with final suffix “your country.”; a controlled zero-tail run fails with
“for.”. \`envInt\` intentionally rejects zero-valued tuning settings, so finish
tail parsing now uses a nonnegative-specific helper to make the control real.
The 55-second fixture also passes with the default tail. This verifies the
finalization behavior on two local fixtures, not general model quality or
promotion eligibility.

**2026-11-22 control run:** With the cached Kroko model, punctuation model,
pinned Sherpa 1.13.2 runtime, and 3.846-second fixture, \`make test-native\`
passes the exact final-suffix assertions under the default 1.2-second tail.
Setting \`SHERPA_ONNX_STREAMING_FINISH_TAIL_MS=0\` makes the same assertions fail
with the known truncated ending. This confirms the regression gate detects the
defect class; it does not establish broader recognition quality.

**2026-11-23 recheck:** The exact available 11.000-second JFK PCM fixture at
\`/tmp/opencode/jfk.wav\` (SHA-256
\`59dfb9a4acb36fe2a2affc14bacbee2920ff435cb13cc314a08c13f66ba7860e\`) was
replayed in 3,200-byte WebSocket chunks against isolated native servers using
the cached Kroko model and current pinned runtime. Both the default 1.2-second
tail and an explicit zero-tail control ended with “your country.”, and both
reported a final timestamp of 11,000 ms within the 11,000 ms source duration.
The historical truncation did not reproduce. The earlier control record has no
fixture digest, so byte identity with the original input is unknown; compare
historical model/runtime and capture configuration before treating this as a
regression. No default promotion follows from these runs.

Audio Tools can explicitly route local Kokoro synthesis to the sherpa speech
endpoint with \`AUDIO_TTS_BACKEND=sherpa-onnx\`. The API integration test proves
bootstrap selection, canonical voice forwarding, audio response handling, and
the event cache write boundary against a contract server. \`/healthz\` now
includes TTS model and encoder readiness, allowing cheap request-path
liveness; full capability readiness still exercises synthesis. Kokoro remains
the default.

Two gateway-integrity defects surfaced while verifying the new default:

- \`qwen3:1.7b\` had no \`runtime_identity\` in resource-ollama's policy, so the
  gateway refused the role with "artifact identity is missing or differs from
  the observed digest". Added the observed \`/api/tags\` digest
  (\`8f68893c…1730e7\`) and regenerated the gateway catalog.
- The gateway invokes \`resource-ollama\` as a subprocess. During the first
  verification the source-built CLI was stale, so its auto-rebuild exceeded the
  gateway's command timeout and the role resolution failed. After one explicit
  refresh, the gateway path works with no artifact-mode override. Warm
  summarize through the gateway is 347 ms; a cold model load is ~8.5 s.

**TTS**: the sherpa adapter already serves **Kokoro int8** (183 MB extracted,
114 MB model) through \`/v1/audio/speech\`; the standalone Kokoro resource uses
313 MB. Kale Nano and Tiny have public Apache-2.0 ONNX artifacts and upstream
SHA256SUMS. At the time of this 2026-11-20 observation, both model cards showed
\`DRAFT, NOT PUBLISHED\`; current Hugging Face metadata updates Nano to public,
non-gated Apache-2.0, while Tiny remains draft.
The graphs use a separate ONNX Runtime frontend and do not load in sherpa's
current Kokoro adapter. A local Nano prototype at pinned revision
\`b6ce1887268c3667994fc3b0b2c82887c069915c\` verified the 14.77 MB int8 graph
against its upstream SHA-256 and produced 3.82 s of 24 kHz speech in 648 ms
including load, on CPU with one ONNX thread (Node RSS 238,428,160 bytes). A
managed candidate now pins CPython, 22 CPU wheels, model/voice/frontend assets,
and licensing notices. Its 498 MiB composed runtime tree digest reproduces
offline; the resource requires 768 MiB disk. Audio Tools can route to it with
\`AUDIO_TTS_BACKEND=kale-onnx\`, and separates persistent audio cache entries by
model. The managed lifecycle started healthy in 6.14 s and stopped cleanly. One
model-backed request to the resource generated a 3.05 s mono 24 kHz WAV in 770
ms cold and 59 ms warm, using Nano revision
\`b6ce1887268c3667994fc3b0b2c82887c069915c\` and graph SHA256
\`4d5269974de4fba7a66f947608a9a5381b4e39c1a8a6073df6001ab7e7f76288\`. This is
resource-level synthesis evidence, not an Audio Tools product-path receipt:
the running Audio Tools API was composed without \`AUDIO_TTS_BACKEND=kale-onnx\`
and was left untouched. The default remains Kokoro. One short utterance and one
voice do not qualify intelligibility or naturalness. The pinned Nano revision
is public, non-gated, and declares Apache-2.0 in Hugging Face API metadata
checked 2026-11-21; Tiny remains draft. Provider promotion and default change remain
gated pending release readiness, reviewed output quality, and product-path
receipts.

12 GB of abandoned Kyutai download was also removed (cache 14 GB → 2.3 GB).

### 2026-11-20 — The machine footprint was an accident, not a decision

Measured on this host: the whisper service was running \`ggml-medium\` and holding
1,864 MiB of VRAM while the registry claimed \`large-v3-turbo\` as the default,
and the shared \`summarize.default\` role resolved to \`qwen3.5:9b\` (~5.9 GB
resident). Two defaults, roughly 8 GB of accelerator memory, and neither was
chosen with the rest of the machine in mind.

Fixed:

- \`resources/whisper\` now ships \`ggml-small.bin\` (488 MB, sha256
  \`1be3a9b2…a987b\`) as the eager artifact and runtime default. Turbo, medium,
  base, and tiny are on-demand groups. The capacity claim dropped from 2 GiB to
  1 GiB.
- The model registry declares \`whisper-small\` as the STT default, turbo/medium
  as quality tiers, and base/tiny as fast/edge tiers, each with its own artifact.
- \`whisper recommend-model\` returns \`small\` by default; \`--quality\` is the
  explicit accuracy opt-in. CPU hosts no longer auto-pick medium.
- resource-ollama's \`summarize.default\` role moved to \`qwen3.5:4b\` (3.39 GB installed),
  which was already the role's qualification candidate. \`qwen3.5:9b\` remains a
  quality tier and can never be a commercial default because its licence is
  unverified.
- Each capability has a declared default footprint budget (STT 600 MB, TTS
  400 MB, summarize 4 GB, speaker 100 MB). A seed edit that promotes a larger
  default fails \`TestCapabilityDefaultsFitResourceBudget\`;
  \`TestLocalDefaultArtifactsAreEager\` fails if a default names an on-demand
  artifact.

Receipt (one Linux/Vulkan host, canonical \`the quick brown fox jumps\` clip):
all four Whisper tiers transcribe the clip exactly; warm wall time 0.16 s
(tiny), 0.18 s (base), 0.31 s (small), 0.30 s (turbo) for 1.95 s of audio. A
live \`audio-tools stt transcribe\` against the restarted service returned the
exact text in 84 ms, and the running server held 716 MiB of VRAM instead of
1,864 MiB. This is a smoke receipt, not a dictation-quality or timestamp-drift
claim.

The operational consequence is explicit: restarting the whisper resource now
loads small instead of medium. The larger tiers remain one acquisition command
away, and selecting one does not require a code change.

Also fixed in the same tranche:

- **Sub-millisecond health latency was invisible.** \`latency_ms\` used integer
  milliseconds, so a healthy localhost check (100-800 µs) rounded to 0 and
  looked identical to "never measured". The handler now reports fractional
  milliseconds and \`audio-tools health show\` renders a LATENCY column.
- **The Models page had two nested "Registry" panels.** \`ModelsTable\` rendered
  its own panel inside the page's panel, producing duplicate headings and
  ambiguous aria labels. The table now lives inside the page panel, and a
  Resource footprint panel shows each capability's default and weight size.
- **Lab code had no measured boundary.** Experiment/eval/soak/validation now
  have coverage floors, \`make lab-test\`, and a parser-based import gate proving
  the serving handlers and chains cannot reach them. Only \`bootstrap\` and
  \`server\` cross the boundary, as enumerated composition exceptions.

### 2026-11-19 — Local Kokoro voice overrides were stored under the wrong key

The Voices page wrote overrides keyed by the provider catalog's adapter id,
\`local:kokoro\`. The local TTS provider looked up \`local:kokoro-local\`. The
override was persisted, rendered as configured, and **silently never applied** —
a correctness bug that no transport test could see because both sides were
internally consistent.

Fix: one \`tts.AdapterLocalKokoro = "local:kokoro"\` constant owns the key, the
provider resolves through the central \`LocalKokoroMapping\`, and the pre-catalog
key is still honoured so stored overrides survive the rename. The same pass
moved the OpenAI and ElevenLabs canonical-voice switches into
\`internal/tts/voice_catalog.go\`, wired \`VerifyCatalog\` into \`bootstrap\` (it was
only called from tests), and made the settings edge reject an unknown canonical
voice or adapter id. \`TestTTSAdapterMappingsMatchProviderCatalog\` fails when the
code-side adapter ids and the provider catalog's \`adapter_id\` drift.

Also fixed in the same tranche:

- **Typed-nil panic**: an optional \`*store\` repository assigned to an interface
  is non-nil as an interface value; \`stt.NewConnectHandler\` called \`Get\` on it
  and panicked during construction. \`isNilDependency\` now treats typed nils as
  absent. \`TestCompose_MountsEveryDeclaredEndpoint\` found it by composing the
  server with absent optional services.
- **Silent empty transforms**: ffmpeg can exit 0 and write nothing (for example,
  an \`acrossfade\` longer than the source). \`runFfmpeg\` now treats empty output as
  \`ErrFfmpegExec\`, merge rejects the impossible crossfade before spawning
  ffmpeg, and trim/split reject windows past the measured end.
- **Silent truncated streams**: a decoder failure closed the Frames channel
  exactly like EOF, so the turn looked complete. The adapter now emits
  \`ErrStreamDecode\`, the wire code is \`input_decode_failed\`, and the handler
  fails the replay ledger instead of marking incomplete audio complete.
- **Provider identity**: runtime aliases and BYOK probe facts moved into the
  provider catalog; the hand switches in \`capabilities\` are gone. Active remote
  providers must declare a probe or an explicit absence reason.
- **Timeout hygiene**: readiness probes use a 30 s doer instead of the 120 s
  generation doer, and BYOK probes are bounded at 5 s.
- **Health telemetry**: provider liveness is timed at the checker seam, so the
  health response's \`latency_ms\` is a measurement instead of a permanent zero.

### 2026-11-19 — Whisper turbo became the shipped runtime model, with a first host receipt

The registry declared \`whisper-large-v3-turbo\` as the promoted default while
\`resources/whisper\` still pinned \`ggml-medium.bin\`. That is the same
declaration/runtime drift class as the old \`sherpa-streaming\` gap, just in the
model layer. Fixed by shipping turbo as the eager, checksum-verified artifact
(sha256 \`1fc70f774d38eb169993ac391eea357ef47c88757ef72ee5943879b7e8e2bc69\`,
1,624,555,275 bytes), moving medium to the on-demand \`medium-model\` rollback
group, and updating the capacity profile to the turbo digest.

New gates make the class impossible to reintroduce:

- \`TestWhisperRuntimeArtifactMatchesRegistryDefault\` fails when the registry
  default and the artifact the resource actually starts disagree, and when the
  runtime default artifact is on-demand.
- \`TestAvailableLocalModelsNameShippedResourceArtifacts\` fails when an
  \`available\` local model names no artifact, or names one the owning resource
  does not declare.

Receipt (this host: RTX 4070 Ti SUPER, Vulkan whisper.cpp 1.9.2, 12 threads):

- Canonical \`quality_speech.wav\` reference \`the quick brown fox jumps\`: WER
  0.00 for both turbo and medium. This is a five-word smoke reference, not a
  corpus.
- 61-minute \`dictation-reference-long.wav\`: turbo 30.3 s wall (RTF 120.8x),
  medium 52.8 s (RTF 69.3x); word-sequence agreement 0.9993 over 8,996 words.
- Steady-state short-clip p50 71.7 ms (turbo) vs 70.7 ms (medium).
- Vulkan GPU residency 1,817 MiB (turbo) vs 1,861 MiB (medium), so the
  resource's 2 GiB capacity claim still holds.

Not claimed: long-form timestamp stability (unscored; turbo is known to be
weaker than large-v3 there), multi-language WER, a device/browser/OS matrix, or
CPU-only throughput. The receipt is one host, one corpus shape, and agreement is
not ground-truth WER.

### 2026-09-17 — Long-turn recovery exceeded the browser terminal wait

The fresh managed Kyutai real-duration cohort completed 29 long turns with
normal provider closure, bounded retention, and visible partials. Its final
turn entered the intended bounded Whisper recovery after Kyutai's empty native
terminal; Whisper completed in 16.3 s, just beyond the product soak driver's
15 s \`captured\` wait. The browser therefore reported the turn as unmeasured
even though recovery was still progressing. Falsifiable intervention: give
realtime terminal observation a bounded 60 s recovery window while retaining
the separate provider/recovery diagnostics; do not classify a known in-flight
recovery as a transport loss. Re-run the long-tail product path and retain any
actual recovery failure.

The 60-second observation window was verified by a fresh 120-second tail check:
the turn reached a terminal state, but its quality remained out of band
(WER 0.7419) and first-partial latency was 1,048 ms. The managed 0.30-second
delayed-stream override therefore does not qualify as a replacement for the
checkpoint's 0.5-second setting: the short smoke was green, but the long-form
quality and latency evidence was not materially better than the accepted
baseline. Revert the override and keep the terminal-wait repair independently.

### 2026-09-16 — Long cohort startup p95 still has cold spikes

The required receipt \`browser-soak-1789596734079721301\` passed capture,
terminal, retention, provider identity, interim visibility, and committed-lag
assertions for all 30 turns. First-partial p95 remained 1,286 ms, with a small
number of 2.7 s startup spikes; long-turn fallback WER also varied widely.
Falsifiable startup intervention: halve the shared PCM wire batch from 1,600
to 800 samples so the decoder receives the first voiced batch about 50 ms
earlier. Keep the 100 ms batch as the comparison and reject this change if
focused timing or backpressure tests regress.

### 2026-09-16 — Managed Kyutai segmentation must be explicit

The isolated 120-second probe validated the terminal timing improvement but
reported 96.7% WER because the managed Kyutai process still emitted no durable
segments; the latest rolling partial was only a short tail. The source server
defines a 48-frame continuous-speech force-commit policy, but the resource
manifest did not explicitly pin that setting, leaving artifact/default drift
unobservable. Falsifiable intervention: set
\`KYUTAI_STT_MAX_SEGMENT_FRAMES=48\` in the managed resource environment and
verify a long probe reports nonzero committed segments and quality before
repeating the 30-turn acceptance cohort.

The explicit managed setting was present in the built artifact, but the next
120-second probe still produced zero API segments and nearly complete WER
failure. Promoting the latest rolling partial is therefore not a valid
long-form finalization strategy. The next intervention is to preserve the
existing bounded Whisper recovery for an empty native terminal, but publish a
\`recovering\` diagnostic state as soon as that fallback begins. This keeps
processed coverage observable and prevents the voice button/harness from
appearing stuck in \`recording\` while complete long-form text is recovered.

### 2026-09-16 — Queued reconnects can strand Kyutai admission tickets

The required 30-turn native browser cohort \`browser-soak-1789591585701729803\`
entered repeated \`provider_done\` reconnects after a foreign stream occupied the
local decoder. The resource log then showed an admission-timeout traceback
while sending the timeout response to a client that had already disconnected.
Inspection found that \`FIFOAdmission.acquire\` notified a queued client before
entering its cancellation/timeout guard. A disconnect from that notification
could therefore leave the future in the FIFO, consuming one of the eight
bounded admission slots while the browser retried. The timeout response also
treated a peer disconnect as an unexpected server error, obscuring the actual
admission state.

Falsifiable intervention: make queued notification part of the admission
future's cancellation-safe region, and ensure the stream releases an acquired
ticket exactly once on every post-admission failure. Add deterministic tests
for notification failure and run a concurrent foreign-stream/reconnect probe;
the long cohort must then complete its terminal/processed assertions without
relaxing the 15-second harness timeout or any acceptance band.

### 2026-09-16 — Long Kyutai passthrough turns fall into slow batch recovery

The repaired admission path passed focused tests and a foreign-stream
contention probe, but the fresh 30-turn cohort still failed 16 processed-ready
assertions. The API logs show the common shape: a graceful 120-second Kyutai
stream with hundreds of partials, all processed coverage acknowledged, zero
segments, and an empty provider \`Done\` text. The browser correctly interpreted
an empty terminal as incomplete native delivery and started bounded Whisper
recovery; the 15-second harness timeout expired while that 5–15 second batch
request was still running. This is a terminal-contract defect, not an
admission-capacity result.

Falsifiable intervention: retain the latest normalized passthrough partial in
the API stream handler and use it as the terminal text when the provider emits
an empty \`Done\` with no segments. This should keep complete native Kyutai turns
on the streaming path, preserve the existing empty-audio recovery behavior,
and make long-turn terminal delivery occur before the harness timeout. Add a
handler regression test and rerun the required real-duration cohort.

### 2026-09-16 — Preview-only VAD bypass is pending qualification

Short, still-growing VAD preview windows were observed to return empty from
Whisper before the next preview produced text. The preview request now bypasses
the backend \`vad_filter\`; committed/final requests still use the configured
operator policy. Strategy and validation tests pass, but deployment and a
fresh paced browser cohort are required before attributing any latency change.

**Owner:** Audio Tools STT preview strategy.

### 2026-09-16 — Initial preview padding does not remove periodic latency spikes

The VAD preview lane was tested with bounded trailing-silence padding for its
first sub-second request, while final transcription remained unchanged. A
fresh 30-turn browser cohort preserved capture, WER, interim visibility,
retention, and provider invariants but still measured p95 first-partial latency
of 1491 ms. The intervention is therefore not an acceptance fix and should not
be reintroduced without a different empty-preview diagnosis.

**Owner:** Audio Tools STT preview strategy.

### 2026-09-16 — Kyutai product-path latency is backend-bound after capture

Fresh D1 browser runs prove capture is not the startup bottleneck: the browser
records its first accepted frame in roughly 27–48 ms and sends it within roughly
125 ms. The API/resource path is correctly seated on \`kyutai/stt-1b-en_fr\` when
the operator speaker filter is off, and native streaming emits visible partial
text, but paced realtime first-partial latency remains 1.28–1.57 s p95 across
three-turn cohorts, outside the approved 1 s band. A direct paced API probe with
the same canonical fixture is roughly 0.73–0.83 s, so the remaining product
variance is in browser/BAS pacing, resource admission/decoding, or the model's
documented 0.5 s delayed-stream window—not microphone permission or
AudioWorklet startup.

The same runs exposed an independent policy interaction: with the persisted
speaker filter enabled, the fixture is rejected by the enrolled profiles,
Kyutai interim text is visible, and the browser correctly uses bounded Whisper
recovery. That is valid speaker-policy behavior but cannot be reused as a
Kyutai quality or provider-parity receipt. Keep speaker-off Kyutai runs labeled
as controlled diagnostic cells until an approved matching speaker profile is
available.

**Refs:** \`packages/audio-capture-browser/src/streamDiagnostic.ts\`,
\`api/internal/soak/driver.go\`, \`resources/kyutai-stt/docs/OPERATIONS.md\`.

**Owner:** \`execute/audio-tools-portable-voice-development\`.

### 2026-09-16 — Realtime qualification playback must be synchronized per turn

The D1 browser soak uses a real host-visible PipeWire source. BAS previously
started its looping WAV player during session construction, before navigation
and the recording click. This allowed short fixtures to begin mid-utterance and
made repeated turns observe different loop phases. A first repair added a
bounded start delay; the stronger repair adds an owner-checked playback restart
between turns while retaining the same browser/page session. The soak now
requests this restart for every healthy realtime turn.

Fresh synchronized evidence \`browser-soak-1789553156986274094\` passed provider
identity, signal, interim visibility, terminal delivery, retention, and exact
accounting. It still failed the approved latency p95 (1.17 s) and Kyutai WER
(0.30–0.40), so no acceptance receipt was promoted. The delay and restart are
qualification synchronization aids, not a relaxation of the D1 protocol.

**Refs:** \`scenarios/browser-automation-studio/playwright-driver/src/session/audio/device-evidence.ts\`,
\`scenarios/browser-automation-studio/playwright-driver/src/routes/session-audio.ts\`,
\`api/internal/soak/driver.go\`.

**Owner:** \`execute/audio-tools-portable-voice-development\` with BAS driver support.

### 2026-09-16 — Reduced host-device loopback buffering; latency band still misses

The BAS host-device qualification loopback used a fixed 100 ms PipeWire
latency. The setting was reduced to 20 ms in the authorized BAS device-evidence
owner and the driver was rebuilt and restarted. A fresh synchronized three-turn
Kyutai cell (\`browser-soak-1789553515207349738\`) passed provider identity,
capture, interim visibility, accounting, retention, memory, and wire-rate
invariants. With the persisted speaker filter temporarily disabled for this
controlled provider measurement, first-partial latency improved to 1,039 ms
p95 (turns 740–996 ms), but remains just outside the approved 1,000 ms band;
WER remained 0.20–0.30. The speaker filter was restored immediately afterward.

The preceding run with the 100 ms loopback buffer was invalid for Kyutai
latency because the configured speaker policy rejected the fixture and the
browser correctly recovered through Whisper. The 20 ms change is retained as
an infrastructure improvement, not as acceptance evidence.

**Refs:** \`scenarios/browser-automation-studio/playwright-driver/src/session/audio/device-evidence.ts\`,
\`browser-soak-1789553515207349738\`.

**Owner:** \`execute/audio-tools-portable-voice-development\` with BAS driver support.

### 2026-09-16 — Quality receipt producer omitted the evaluated safety gate

The first fresh 180-clip \`voice-pilot-en-v1\` Kyutai report measured aggregate
WER at 5.98%, but one clean clip produced a 27-word contiguous dropped span.
The evaluation report correctly failed its safety gate; the quality-evidence
producer only checked subgroup WER and emitted a passing target-aware
\`corpus-quality\` receipt anyway. The producer now requires the report safety
gate and has a regression proving that an in-band WER cannot override a safety
failure. A replacement held-out run is in progress to emit the corrected
negative receipt. Corpus quality remains unqualified until the clip-level
safety failure is repaired and a fresh report passes.

**Refs:** \`api/internal/experiment/report/quality_evidence.go\`,
\`api/internal/experiment/report/quality_evidence_test.go\`.

**Owner:** \`execute/audio-tools-portable-voice-development\`.

### 2026-09-16 — Reconnect recovery can expose a stale sample cursor

Fresh owner-backed realtime browser runs reached the reconnect/replay path after
capturing more than one million samples. The persistent journal downgraded to
bounded memory, then raised \`capture_write_failed:sequence_mismatch\`: the next
frame sequence matched the journal cursor but its sample offset differed. This
left the turn without a processed cursor or completed product result.

The browser path now records bounded metadata-only cursor details and has a
constrained reduced-durability repair for this exact next-sequence boundary.
Unit coverage passes, but a fresh owner-backed run after the repair is still
required before this problem can be closed or any dictation-trust claim can be
promoted.

### 2026-09-15 — Repeated microphone sessions can require refresh or restart

Operator observation: a first speech-to-text session may work, while a later
session in the same page can remain broken until the page is refreshed or the
app is restarted. The self-improvement contract now requires at least five
consecutive warm press/speak/transcribe/stop cycles in one page and browser
context, with no refresh, restart, or manual recovery. Reproduce and instrument
the lifecycle and transport reset path; keep this behavior unqualified until
fresh owner-backed evidence passes every cycle.

**Related lifecycle race fixed (2026-11-21):** a delayed per-turn capability
probe could report an unhealthy backend after that turn had stopped and a new
turn had begun. Its callback unconditionally shut down the shared provider and
overwrote current capability state. A deterministic hook regression reproduced
the stale result disposing the newer turn (provider \`dispose\` was called twice
while the newer turn was preparing). The callback now checks its captured start
generation before applying results. \`pnpm vitest run
src/useVoiceCore.startup.test.ts src/longSession.test.ts\` passed 34/34 in
\`packages/audio-capture-browser\`. This closes that specific race only; the
required five-cycle browser capture run has not been performed, so the
repeated-session behavior remains open and unqualified.

### 2026-09-15 — Microphone press-to-speech readiness is suspected above target

Operator observation: after pressing the microphone control, the path can take
several seconds before it accepts spoken input. The portable-voice interactive
latency target already requires warm readiness p95 ≤500 ms; this entry clarifies
that readiness ends at the first accepted voiced sample, not at button handling,
permission resolution, stream creation, or a readiness banner. Reproduce this
in the authorized browser cohorts, instrument each milestone, and repair the
owner path. Keep the row unknown until current paced evidence proves the band.

### 2026-09-09 — Portable voice contract aligned; execution and qualification remain open

The operator requested that Audio Tools documentation express the complete
portable voice intention. Business Health regenerated PRD \`portable-voice-v1\`
after a diff preview. It now has 15 targets; the requirements registry retains
all ten original ATD identities and test references and adds 14 planned claims.
Local/BYOK/owned routing, interactive partial/final latency, recognition quality,
subscription/credits, host qualification, privacy and evidence integrity are no
longer merely prose extensions outside the charter.

The owner validator reported unearned completion for ATD-P0-004, ATD-P0-006,
ATD-P1-001 and ATD-P1-002 because no requirements-sync snapshot exists. Their
statuses were returned to \`in_progress\`, retaining historical observations.
This is correction of an unsupported claim, not proof of a new code regression.
New validation entries are planned test designs, not executed manual attestations.

W0 scope alignment is documented; numeric SLOs, cold/admission bounds, supported
session/concurrency limits, realized corpus/native profiles, billing delivery
semantics and execution/spend allowance still require the decisions in TESTING.md.
W1 obligations are now explicit. The all-published-targets regression caught
the v1 board omitting the newly published scope. A bounded v2 metadata update
now lists all 15 targets, preserves the original row IDs and still leaves every
outcome unknown. W2 remains open: owner-backed receipt joins are not implemented.
Legacy duration/freshness artifacts need
per-branch early-EOF, input-duration and final-tail controls before qualification.

W3 product work remains, including actual owned inference and shared settlement,
route/privacy policy enforcement and paced/native consumer qualification. The
local-only proposal remains a historical non-launchable subset with zero budgets
and prose effects/evidence sources. The full mandate must be authored using real
typed owner resolvers and approved shared-owner boundaries after Swarm launch,
continuation and budget qualification. Tech Tree Designer remains deferred.

Documentation also replaces template security/deployment/market placeholders,
repairs obsolete OT meanings in DOMAINS/FLOWS, and distinguishes non-blocking local
usage history from durable shared billing. No runtime source, paid provider call,
goal launch, production setting, commitment of funds or device certificate is
included in this documentation change. Prior dated findings remain below.

Validation for this revision: Test Genie run \`20260909-120526-95ee9517\` passed
the scoped \`business\`, \`programs\` and \`skill-set\` phases (15 seconds). All 15
local board regressions pass after the target-visibility test first rejected
the incomplete v1 inventory. Both canonical skills pass quick validation.
Business validation retains a stale/missing sync-snapshot warning; no statuses
were promoted and no full suite was requested. The docs-root structure check
passes with its README-layout warning (the declared entrypoint is ../README.md).

Experience validation is not clean: it reports the evicted VoiceInputButton
3.0.0 specimen reference, pinned chrome outside a landscape viewport, and a
40px menu target below the 44px floor. Audio imports component major 4, while
the retained local experience spec still carries older state/dismissal claims.
Reconcile the actual component version and recovery intent with its owner;
do not repoint or remove claims merely to clear the validator. Those findings
were filed to scenario-qa (\`knw-1788955668422940874\` for the component contract;
\`knw-1788955668705488603\` for the UI floors). Structure Health and Experience Manager CLI rebuilds
also fail dependency checks and report stale binary/source fingerprints;
their assessments have that explicit provenance limitation (report
\`knw-1788955668141098902\`). A final local manifest consistency check found zero
missing files, required headings or required links across 40 registered docs;
all ten original requirement identities and 21 test/automation references were
verified preserved. This check is not runtime experience reconciliation. No raw dependency
repair, UI change or artifact eviction was performed here.

### 2026-09-08 — Evidence-board fixture follows the shared program interface

The current setpoint program uses \`program.inputs()\` and \`program.classify()\`.
Its old fixture injected only \`inputs\`, causing all 15 tests to fail before
measurement. The fixture now injects the declared interface and loads the shared
Program Runtime classifier instead of duplicating transport classification.
All 15 assertions pass. Malformed inputs remain explicit domain negative
controls; the fixture does not certify the kernel input gate, live bindings,
session lifecycle or voice-product acceptance. The local pilot proposal now
includes proposed goal guidance, without adopting budgets or launching work.

### 2026-08-18 — Swarm voice route fix is deployed; product qualification remains open

**Finding:** The bounded control \`browser-soak-1787070923294255803\` reached
non-silent capture but lost the WebSocket upgrade at the Swarm UI boundary,
so it fell back to batch transcription and is non-credit.

**Resolution:** The UI now targets \`/ws/voice/stream\`, and Swarm's embedded
proxy rewrites that route to \`/api/v1/voice/stream\`. Focused URL tests and the
production build pass; managed health and a direct routed handshake returned
\`stream_connected\`.

**Remaining gate:** No replacement long run was started. Swarm still lacks a
qualifying product-path artifact with provider identity, committed intervals,
and rendered interim evidence.

**Owner:** audio-tools/swarm-manager integration.

### 2026-08-18 — Rebuilt Swarm Quick Capture still fails before recording (startup root cause fixed)

**Symptom:** After rebuilding the served Swarm Manager UI and restoring all
declared dependencies, the bounded accelerated control
\`browser-soak-1787068170138937224\` timed out waiting for
\`[data-testid="captures-quick-input-mic"][data-state="recording"]\`. No
voice-stream request or qualification diagnostic was observed.

**Root cause identified:** The shared \`useVoiceCore\` hook awaited provider
startup without handling rejection. A failed microphone acquisition therefore
left the control in \`preparing\` without a terminal diagnostic or visible error.

**Workaround:** Keep Swarm Manager product-path qualification non-credit and
use the already-qualified Audio Tools 60-minute shapes for current evidence
until one bounded control proves the served integration path.

**Real fix:** \`useVoiceCore\` now catches provider-start rejection, returns to
idle with an actionable error, and disposes the provider. The focused startup
regression passes. The original product-path artifact remains non-credit and
the integration still needs one bounded post-fix control.

**Owner:** audio-tools/swarm-manager integration.

### 2026-08-18 — Swarm streaming admission and WebSocket route needed hardening

**Symptom:** The post-startup Quick Capture control
\`browser-soak-1787069020597299475\` captured non-silent audio but exposed no
provider identity, stream diagnostic, server ledger, or interim text. It fell
back to batch \`AudioRuntimeService/Transcribe\`. A second control,
\`browser-soak-1787069315005755975\`, recorded WER 0.0 but showed the same
missing streaming evidence, so neither is qualification evidence.

**Root causes:** Swarm's WebSocket builder received an API base already ending
in \`/api/v1\` and appended the versioned route again. Separately, the shared
hook could enter its first one-shot turn before its asynchronous capability
probe resolved, leaving \`streamingAvailable\` false and selecting the weaker
HTTP provider.

**Fix:** Normalize the Swarm WebSocket base to the server root before appending
\`/api/v1/voice/stream\`, and await unresolved capability admission for all voice
modes. The URL regression suite passes 9/9. A fresh bounded runtime control is
still required after the final managed bundle is confirmed; no long soak has
been run for this fix.

**Owner:** audio-tools/swarm-manager integration.

### 2026-08-18 — Fault qualification still needs realtime product-path execution

**Symptom:** The realtime product-path portion of the eleven-fault trust-floor rubric is not yet complete.

**Root cause:** The browser soak now emits durable per-profile fault evidence, and the current coverage directory contains four-passing-assertion accelerated artifacts for all eleven required profiles on the virtual provider cell. The realtime product-path executions have not all been run against the current provider cells.

**Workaround:** Keep the promotion verdict red until the realtime product-path evidence rows exist for the required supported cells; do not infer that coverage from the accelerated virtual matrix, injector source, or unit tests.

**Real fix:** Run the server-owned realtime matrix and inspect the persisted evidence rows for every supported cell.

**Owner:** audio-tools qualification workflow.

### 2026-08-18 — Swarm Manager Quick Capture does not yet expose qualification telemetry

**Symptom:** The governed soak driver can open Swarm Manager Quick Capture and
observe non-silent host capture, but the host surface does not currently expose
provider identity, committed interval coverage, or rendered interim telemetry
for a qualifying realtime turn. A 30-second turn also remained in the shared
voice control's processing state beyond the 60-second terminal budget.

**Root cause:** Swarm Manager consumes the shared voice core through its own
API adapter and composer host, but the current host configuration/path does not
produce the same diagnostic and terminal evidence contract as Audio Tools
Dictation Studio.

**Workaround:** Treat the Swarm smoke artifacts as diagnostic only. Do not
promote Swarm Manager from the passing Audio Tools 60-minute evidence or from a
short capture/control result.

**Real fix:** Make the Swarm host surface expose the shared bounded diagnostic,
provider identity, committed coverage, and interim rendering contract, then run
one bounded product-path qualification against it.

**Owner:** audio-tools/swarm-manager integration.

### 2026-08-18 — Long-turn fallback ignored the durable journal tail (resolved)

**Symptom:** After the browser's bounded complete-turn retry copy filled during
a healthy long streaming session, a later backend failure could not use the
still-unacknowledged PCM already present in the durable journal.

**Root cause:** The fallback predicate and WAV builder consulted only the
bounded whole-turn memory copy. The journal was correctly compacting processed
coverage, but its remaining recovery tail was not selected after the memory
copy overflowed.

**Resolution:** Short turns retain the complete bounded copy for retry UX. Once
that copy overflows, fallback builds its request from the journal's
unacknowledged chunks. Committed terminal finals also delete the journal
records. Focused long-session tests cover both recovery and cleanup.

**Remaining boundary:** The journal quota is finite by design. If a backend is
unavailable longer than that quota, the provider fails visibly with a
durability reason rather than claiming silent success.

**Owner:** audio-capture-browser.


## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [\`../guides/troubleshooting.md\`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

\`\`\`markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
\`\`\`

## Entries
### 2026-08-18 — BAS host-device verification raced client navigation (resolved)

**Symptom:** A host-device qualification could fail during session startup with
\`page.evaluate: Execution context was destroyed, most likely because of a navigation\`.

**Root cause:** BAS navigates the target page during startup, and device
enumeration ran immediately after \`domcontentloaded\` without allowing that
client-side navigation to settle.

**Resolution:** Device verification retries only navigation-context failures,
with a bounded three-attempt limit; unrelated failures still surface
immediately. The regression test and driver build pass. An interrupted
continuous qualification that encountered the old failure remains non-credit.

**Owner:** browser-automation-studio.

### 2026-09-14 — Acceptance board join implemented; qualification receipts remain external

**Symptom:** The live \`audio-tools.setpoint-read\` board sees persisted generic
qualification records but has no target-aware owner-backed acceptance claims,
so the 13 in-grant required rows remain unknown.

**Root cause:** The experiment store previously exposed only generic provider
qualification artifacts; it had no target-aware join for the adopted
\`portable-voice-v1\` inventory. The required receipts themselves have not yet
been produced by the D1/D2 product-path, corpus, route/settlement, lifecycle,
and conformance qualifications.

**Resolution:** The v3 read-only board now resolves rows only from explicit
\`audio-tools-acceptance-v1\` receipt claims carrying the exact target revision,
cohort, build, device, policy, and corpus identities, with passing negative
controls. It rejects stale, malformed, mixed, failed, or simulated-as-native
claims and reports the two excluded rows explicitly. Focused regression tests
cover complete joins and stale/mixed-build rejection.

**Remaining blocker:** A qualifying owner operation must produce current
target-aware receipts. This requires the authorized D1/D2 qualification executions and
their approved corpus/route/settlement evidence; the current workspace has no
such receipts, live provider credentials, private corpus, physical native
devices, or paid allowance. Until those owner artifacts exist, acceptance must
remain unknown.

**Owner:** Audio Tools qualification workflow and the named shared route,
corpus, and settlement owners.

**Refs:** \`.vrooli/program-runtime/setpoint-read.py\`, \`docs/internal/TESTING.md\`,
\`test/test_setpoint_read.py\`.

### 2026-09-14 — Target-aware receipt producer now guards the soak owner

The server-owned soak path now builds the adopted acceptance envelope when a
complete real-time 60-minute conformance run proves the narrow trust/evidence
claims. Accelerated, short, failed, incomplete, or identity-less runs remain
diagnostic-only. The envelope builder and negative controls are covered by
\`api/internal/acceptance\` tests, and the soak owner package passes.

This is producer plumbing, not qualification: the live board still reports 97
generic receipts, zero target-aware claims, and 13 required rows unknown. The
remaining rows require their own authorized D1/D2 local or simulated producers
and current evidence; no receipt is manufactured from the existing generic
history.

**Refs:** \`api/internal/acceptance/acceptance.go\`,
\`api/handlers/soak/module.go\`, Plan Manager execution
\`239232c2-26ef-4030-9b43-72b9f4d15af0\`.

### 2026-09-14 — Legacy experiment reports lacked provider identity

The local experiment path was exercised after lifecycle restart and initially
returned a successful clean-only report whose strategy row had empty
\`engine_id\` and \`model_id\`. That made the measured WER unusable for any
provider-neutral acceptance producer. Legacy strategy specs now resolve to the
declared \`whisper-local\` engine and stamp the actual provider model (or the
explicit \`whisper-unknown\` value) into every report row.

Focused eval, report, acceptance and soak tests pass. A fresh live regression
experiment \`5358c131-7605-4480-92ce-bdbb14344a54\` now reports
\`whisper-local\`/\`whisper-medium\`/\`vad_segment\` with WER 0 for its one clean
clip. This remains diagnostic: it is not a 60-utterance licensed held-out
corpus, has no noisy cohorts, and does not produce an acceptance receipt.

**Refs:** \`api/internal/eval/runner.go\`, \`api/internal/eval/runner_test.go\`.

Archived source history is retained in Swarm Manager.

### 2026-09-16 — Kyutai 100 ms streaming resampling had a phase-reset defect

**Symptom:** The held-out English clip decoded coherently with 1-second
replay chunks but produced Vietnamese-like repeated text at the browser's
100 ms cadence (WER 1.0). The live provider was healthy and receiving audio,
so this was a correctness failure rather than a startup-only failure.

**Root cause:** The Kyutai service independently applied
\`torch.interpolate(..., align_corners=False)\` to every inbound 100 ms chunk.
That reset the interpolation origin at each WebSocket boundary. The fix keeps
one input/output cursor per stream and reproduces PyTorch's half-sample mapping
across chunk boundaries, including the final clamped endpoint.

**Evidence:** After a supported \`vrooli resource install kyutai-stt
--reacquire\` and integrity-checked restart, experiment
\`1409ccb6-fb8c-4d29-b4d6-fe3689ec624d\` ran the same held-out clip at 100 ms:
WER 0.1591, zero dropped-span words, and the safety gate passed. This is a
targeted regression result, not a cohort or acceptance receipt.

**Refs:** \`resources/kyutai-stt/server/server.py\`,
\`resources/kyutai-stt/resource.json\`.

### 2026-09-16 — Kyutai server tests were importing the wrong module

The resource Python tests inserted their own \`test/\` directory into
\`sys.path\`, so \`import server\` could resolve an unrelated installed module
instead of \`resources/kyutai-stt/server/server.py\`. The test seam now points
explicitly at the resource server directory. The actual server state-machine,
backpressure, and streaming-resampler suites pass 21/21; this closes a false
green/false red measurement gap without changing runtime behavior.

The resource quality entrypoint referenced the same incorrect \`tests/\` path.
It now invokes the real \`test/\` suites, so \`make test-python\` is a working
owner command rather than a dead measurement path.

### 2026-09-14 — Corpus provenance was not representable

The corpus service stored only transcript, tags, source, and audio shape. It
could not prove a corpus revision, license/provenance reference, speaker
subgroup, held-out split, clean/noisy condition, or content identity. A local
clean-only corpus could therefore be mistaken for a quality cohort by a future
producer.

The corpus proto, SQLite schema/migration, service, RPC mapper, and import CLI
now carry those fields; the service computes a SHA-256 content digest. A
deterministic cohort assessor rejects missing or mismatched provenance,
non-held-out clips, missing speakers/licenses/hashes, duplicate audio, fewer
than 60 clips or 3,000 reference words, fewer than six speakers, and missing
clean/20 dB/10 dB noisy cohorts.

The live corpus was re-read after lifecycle restart and remains legacy
diagnostic data (eight clean/scripted clips without the new metadata). The
governed board remains \`0/13\` required rows resolved in
\`prog_f53863b7-d823-4c04-8eab-cae3f4995879\`; no quality acceptance claim was
manufactured.

**Refs:** \`packages/proto/schemas/audio-tools/v1/corpus/corpus.proto\`,
\`api/internal/corpus/quality.go\`, \`api/internal/corpus/service.go\`.

### 2026-09-15 — Quality cohort gate is now an owner receipt producer

The corpus assessor initially had no consumer. Completed server-owned
experiments now emit one \`corpus_quality\` qualification artifact per strategy
cell. The artifact contains a target-aware acceptance envelope, provider cell,
cohort counts, subgroup WER measurements, and explicit failure reasons. The
manager persists failed artifacts as well as passing artifacts. The experiment
enum/mappers now preserve \`corpus_quality\` instead of serializing it as
\`UNSPECIFIED\`.

Live regression experiment
\`57282cfd-b9cc-4592-b552-8c7bbf3fbd45\` produced receipt
\`6257686f-13e2-4ec7-9c95-a20acbbab823\`: \`passed=false\`, eight legacy clips,
167 reference words, zero speakers, no clean/noisy cohorts, and missing
provenance. The fresh governed board
\`prog_34c73fee-38a5-4cf6-9beb-35d4e5f70e59\` now explicitly reports
\`corpus-quality\` as \`failed:receipt_claim_out_of_band\`, with one inspected
target-aware receipt cohort and 0/13 required rows resolved.

**Refs:** \`api/internal/experiment/report/quality_evidence.go\`,
\`api/internal/experiment/runner.go\`,
\`packages/proto/schemas/audio-tools/v1/experiment/experiment.proto\`.

### 2026-09-15 — Latency claim lacked an attempt-count gate

The target-aware soak producer could claim only the generic trust/provider
rows, and the conformance run did not expose an explicit minimum-attempt gate
for the adopted latency cohort. \`ClaimsForRealtimeRun\` now adds
\`interactive-latency\` only for a qualified wall-clock run lasting at least 60
minutes with \`Turns >= 30\`; accelerated, short, and single-turn runs remain
ineligible.

Focused acceptance, soak, and conformance tests pass. No live receipt was
promoted by this code-only gate; the next eligible product-path run must still
exercise the D1/D2 route/profile cohorts and provide the target-aware envelope.

**Refs:** \`api/internal/acceptance/acceptance.go\`,
\`api/internal/acceptance/acceptance_test.go\`.

**Refs:** \`playwright-driver/src/session/audio/device-evidence.ts\`,
\`playwright-driver/tests/unit/session/audio-device-evidence.test.ts\`.

### 2026-06-29 — Eval auto-tune sweep/apply RPC deferred

**Symptom:** Dictation Studio experiment reports now recommend a winning
strategy, show corpus adequacy warnings, explain row trade-offs, and expose
per-clip diffs. They do not yet offer a backend-owned sweep/apply operation
that evaluates bounded candidate stream configs and explicitly applies the
recommended config from the experiment surface.

**Root cause:** The validated mutation path for streaming STT config is
\`STTAdminService.UpdateStreamConfig\`. The experiment lab intentionally owns
measurement and report semantics only; adding apply semantics requires
threading the existing STT admin writer or a shared config-apply seam rather
than creating a second JSON config writer.

**Workaround:** Use the report recommendation and row warnings to pick a
bounded lever change, then apply it through \`audio-tools stt
stream-config-set\` or the STT admin UI. Re-run \`audio-tools experiment start
--realtime-repeats N\` and compare reports to evaluate the changed config.

**Real fix:** Add a preview-first bounded sweep RPC that constructs named
candidate arms (\`balanced\`, \`lowest_latency\`, \`lowest_cost\`,
\`highest_stability\`), scores them with the same backend report semantics, and
calls the same validation/apply path as \`STTAdminService.UpdateStreamConfig\`
only when apply is explicit.

**Owner:** unassigned.

**Refs:** \`packages/proto/schemas/audio-tools/v1/eval/eval.proto\`,
\`handlers/eval/\`, \`handlers/stt/stream_config.go\`,
\`docs/reference/eval-harness.md\`.

### 2026-05-16 — Streaming providers declared but not implemented

**Symptom:** STT \`TranscribeStream\` and TTS \`SynthesizeStream\` RPCs work on the wire but emit zero partials, defeating the latency benefit of streaming. Every session falls back to the buffered unary path (\`DoneEvent.FellBackToUnary=true\`).

**Root cause:** Phases D and E of the audio-tools-web-console-restoration plan are deferred. \`LocalProvider\`, \`BYOKProvider\`, and \`VrooliProvider\` all declare \`StreamingCapability()=false\`; \`chain.Stream\` therefore takes the buffered fallback in every case.

**Workaround:** Functional: every consumer sees the final transcript / final audio. No functional gap, only latency.

**Real fix:** Plan Phase D: wrap \`internal/voice\` segmenter behind \`LocalProvider.TranscribeStreaming\`; emit live partials + segments. Plan Phase E: implement Deepgram WS and OpenAI Realtime STT adapters in \`internal/byok\`.

**Owner:** unassigned.

**Refs:** \`internal/ai/sttchain/chain.go::Stream\`, \`internal/ai/ttschain/chain.go::Stream\`, \`~/.vrooli/plans/audio-tools-web-console-restoration-live-mic-diagnostics-chain-routed-streaming.md\` Phases D + E.

### 2026-05-16 — WS handler not yet chain-routed

**Symptom:** \`GET /api/v1/voice/stream\` still calls \`internal/voice/Service.HandleStreamWS\` directly; chain.Stream is only reachable via the Connect bidi RPC.

**Root cause:** Plan Phase F's second half (WS handler rewire) deferred.

**Workaround:** Browser voice input still works via the legacy direct path; only BYOK / Vrooli streaming tiers are bypassed there.

**Real fix:** Rewire \`handlers/stt/stream_ws.go\` to upgrade the WS, marshal incoming binary frames into \`sttchain.AudioChunk\`, and forward \`<-chan sttchain.StreamEvent\` back over the wire shape expected by the shared browser capture client.

**Owner:** unassigned.

**Refs:** \`internal/voice/stream_ws.go\`, \`handlers/stt/stream_ws.go\`, plan Phase F.

### 2026-08-31 — WS provider routing delivered

The browser WebSocket path now upgrades directly into the shared
\`sttchain\`/segmenter pipeline. It applies the encrypted-store default BYOK
credential when handshake headers cannot carry one, preserves explicit
request credentials, and emits provider identity metadata. The historical
entry above is retained as the original defect record; its workaround is no
longer current.

**Evidence:** \`handlers/stt/stream_ws.go\`, focused STT handler tests, and the
live baseline transcription recorded in
\`../../../.vrooli/plan-artifacts/speech-stack-ground-truth/\`.

### 2026-05-27 — ScriptProcessorNode → AudioWorklet migration (deferred)

**Symptom:** Two PCM capture sites use the deprecated \`ScriptProcessorNode\`: the wake-word \`createPassiveCapturePipeline\` (\`hooks/voice/audioUtils.ts\`) and the new streaming \`createScriptProcessorPcmCapture\` (\`hooks/voice/pcmCapture.ts\`). ScriptProcessor runs on the main thread and is deprecated in favor of \`AudioWorkletNode\`.

**Root cause:** ScriptProcessor is universally supported and kept deliberately for broad browser coverage; AudioWorklet requires loading a separate module script and more setup. Both sites carry \`eslint-disable @typescript-eslint/no-deprecated\` with this rationale.

**Real fix:** Introduce a single AudioWorklet processor that emits Float32 frames and back both capture sites with it. The \`pcmCapture.PcmCaptureFactory\` seam already isolates this — swap the production factory only; \`VoiceStreamProvider\` and its tests are unaffected.

**Owner:** unassigned.

**Refs:** \`hooks/voice/pcmCapture.ts\`, \`hooks/voice/audioUtils.ts\`.

### 2026-05-27 — Live ML-resource validation pending operator hardware (deferred)

**Symptom:** The two new ML resources (\`resources/kyutai-stt/\`, \`resources/speaker-verification/\`) are built, self-consistent, and contract-matched to their audio-tools clients, but have NOT been started end-to-end: that needs a GPU (Kyutai) + first-run multi-GB model downloads (Kyutai weights; SpeechBrain ECAPA, CPU-only).

**Status:** Deferred to the operator. Everything audio-tools-side is validated with fakes (Go build/test/lint, CLI, UI tsc/eslint/vitest, scenario restart). To validate live: build + start each resource (\`vrooli resource start kyutai-stt\` / \`vrooli resource start speaker-verification\`), confirm \`/health\` + \`/ready\` flip ready after the model download, then drive a real mic session (Kyutai partials/segments; speaker \`filter\` mode rejecting a second voice / background music). Kyutai enforces a single concurrent streaming session per model instance (asyncio lock) — revisit with a model pool if concurrency is needed.

**Owner:** unassigned (operator hardware).

### 2026-05-27 — Target-speaker extraction (IMPLEMENTED; empirical tuning pending)

**Symptom (original):** \`SpeakerConfig.ExtractionEnabled\` + \`pipeline.ExtractTargetSpeaker\` existed, but the \`speaker-verification\` resource returned HTTP 501 for \`/v1/extract\`; nothing isolated the enrolled speaker's audio.

**Resolution (2026-05-27):** Implemented as a pre-recognition **ingress** stage, not an egress method (egress can only drop text; isolation must substitute audio). \`resources/speaker-verification\` \`/v1/extract\` now runs SepFormer source separation + ECAPA target-selection and returns the isolated 16 kHz mono s16le PCM. The Go side adds \`ingress.TargetExtractor\`/\`ingress.ExtractionEnhancer\` (\`internal/stt/ingress/extraction.go\`), the handler-layer adapter \`speakerExtraction\` + \`currentSpeakerExtraction\` (built only when \`extraction_enabled\` + a bound profile), wired in \`Segmenter.buildIngress\` (config-gated like denoise — no manifest flag) and reachable via \`audio-tools stt speaker-config --extraction-enabled\`. The orphaned \`pipeline.ExtractTargetSpeaker\` (egress-shaped, zero callers) was deleted. Tested with fakes: \`internal/stt/ingress/extraction_test.go\`, \`internal/stt/segmenter/ingress_build_test.go\`, \`handlers/stt/speaker_extraction_test.go\`; resource non-model paths in \`test/integration-test.sh\`.

**Pending (environment-gated):** the separation-model spike — which SepFormer checkpoint + match threshold work best on real two-speaker audio, and CPU-vs-GPU latency — needs a GPU + model download + a live two-speaker A/B. Default OFF; degrades to passthrough if the resource/model is unavailable. See \`resources/speaker-verification/docs/extraction.md\`.

**Interactive tuning (2026-09-16):** the initial implementation buffered ~3 s
before the first extraction call, which made the browser voice button appear
unresponsive whenever an operator had extraction enabled. The default ingress
window is now ~800 ms (25,600 bytes at canonical 16 kHz mono s16le); callers that
prioritize separation quality may still provide a larger \`WindowBytes\`. This is
a latency/quality tuning change, not acceptance evidence; a fresh product-path
cohort is still required.

**Owner:** unassigned.

### 2026-05-16 — WS endpoint tag still \`ops_probe\`

**Symptom:** \`GET /api/v1/voice/stream\` is tagged \`RESTReason: ops_probe\`; semantically it is a \`TransportReason: websocket_transport\`. Cosmetic only.

**Root cause:** The \`websocket_transport\` template constant in \`internal/module\` does not exist yet (tracked in R-PROTO upstream).

**Workaround:** Live with the mistag; documented in \`api-endpoints.md\` "Transport exceptions" with an explicit note.

**Real fix:** Retag when the template constant lands.

**Owner:** unassigned.

**Refs:** \`docs/reference/api-endpoints.md\` "Transport exceptions" table.

### 2026-06-29 — Dictation Studio BAS cannot prove physical microphone capture

**Symptom:** The deterministic BAS flow can validate Dictation Studio's scripted
bench UI, but it cannot prove the browser is receiving real microphone audio or
that a physical recording can be saved and replayed through Eval.

**Root cause:** The shared BAS toolchain does not yet provide mic-permission
grants, deterministic media-stream fixtures, or fake audio-device routing for
the physical capture path.

**Workaround:** Run manual operator validation: select a built-in script, record
with a real mic, confirm the audio meter moves, stop into transcribing,
correct/save the clip, confirm it appears in Corpus, then run Eval.

**Real fix:** Add BAS media-stream fixtures/fake audio routing, then promote a
Dictation Studio real record -> save -> eval workflow from observer coverage to
requirement-bound playbook coverage.

**Owner:** unassigned.

**Refs:** \`scenarios/audio-tools/bas/flows/audio-tools-dictation-studio-scripted-bench.json\`,
\`scenarios/audio-tools/docs/reference/eval-harness.md\`.

### 2026-07-11 - BAS selector manifest qualification and streaming proxy blocker

Persisted BAS workflows now compile selectors from Audio Tools ui/src/consts/selectors.manifest.json through their project root. Execution 3c21f0f4-d75a-4e7a-9704-519aecdbd6d5 completed all browser controls with the WAV fake microphone, but it is not successful PCM-v2 evidence: Audio Tools runtime logs show the normal UI proxy forwarding /api/v1/voice/stream as a plain GET without WebSocket Upgrade, producing HTTP 400. Filed scenario-qa code-defect knw-1783819641843118934. ATD-P0-006 automation therefore remains planned; deterministic P0-005 faults are now authorized by the server-owned routed-isolation lease rather than a boot-only switch.

### 2026-07-11 - P0 dictation requirements are in-progress pending live qualification

**Symptom:** The P0 registry previously showed  even where focused unit and deterministic-browser evidence had passed, while its test-only validations could eventually allow an over-broad completion status.\\n\\n**Root cause:** The original requirement entries had no separate validation for their real-resource and manual qualification gates.\\n\\n**Workaround:** The registry now marks verified test and BAS validation as implemented, leaves each P0 requirement , and records a planned manual gate for the outstanding full provider profile, live admission/recovery, cross-engine speaker policy, consumer adapters, and device evidence.\\n\\n**Real fix:** Complete and record those gates through the qualification harness and ; only then may the full sync promote the requirements.\\n\\n**Owner:** audio-tools provider-parity plan.\\n\\n**Refs:** requirements/01-must-ship/module.json; provider experiments e0a2596e-91e9-4e4f-8b44-b2bec7b79e3d, ddf20231-47c4-4da6-803e-1e5e33ae3fb0, 4e5505e6-3e61-405f-88ab-b9a83cc2538a.

### 2026-07-11 - Correction: P0 dictation qualification gates

**Correction:** The preceding entry's wording was damaged by shell interpolation. Treat this entry as authoritative.\\n\\n**State:** Each P0 dictation requirement is in_progress. Focused unit and deterministic-browser validations are implemented. A planned manual validation remains for each real-resource and device gate, so a future full sync cannot mark a P0 claim complete from unit evidence alone.\\n\\n**Completion path:** Run and record the full provider duration, interruption, recovery, fault, browser, consumer-adapter, cross-engine speaker-policy, and device qualification profiles. Then record manual evidence with the scenario requirements manual-log command and let the normal full-suite sync derive completion.\\n\\n**Refs:** requirements/01-must-ship/module.json; experiments e0a2596e-91e9-4e4f-8b44-b2bec7b79e3d, ddf20231-47c4-4da6-803e-1e5e33ae3fb0, 4e5505e6-3e61-405f-88ab-b9a83cc2538a.

### 2026-07-23 - Eval runner is now transport-free

Evaluation replay/session construction moved from handlers/eval into internal/eval with explicit corpus and speaker adapter ports. Bootstrap invokes the internal runner directly; remaining bootstrap handler imports are transport composition and speaker adapter construction pending experiment-engine relocation. Evidence: GOWORK=off go test ./... passes on 2026-07-23.

### 2026-07-23 - Bootstrap transport and experiment boundaries were separated

The experiment report workflow now lives in internal/experiment/report and HTTP module composition now lives in internal/transport. api/internal/bootstrap/bootstrap.go is 312 lines and has no handler imports; bootstrap's package files have no handler imports. Full API suite passed with GOWORK=off go test ./... on 2026-07-23. Remaining architecture findings still require a fresh evidence-manifest-producing architecture run.

### 2026-11-15 - 2026-11-16 — Cleanup history: centralize provider-tier labels

The five CLI domains settings, STT, summarize, TTS, and usage held byte-identical providerTierLabel implementations. Their callers now use one Audio Tools CLI helper at cli/domains/internal/providerlabels; all prior local implementations were deleted. The separate diagnostics formatter remains because its unknown-value label differs. Focused tests for the shared helper and all five affected domains pass. Production source is 46 lines smaller across the CLI domain boundary; whole-scenario reduction and Test Genie maturity are not claimed.

### 2026-11-15 - 2026-11-16 - Remove unconnected HTTP and remote reporting stubs

Approach: dead-path deletion. Removed the internal/httpx production writer because repository-wide source search found no production caller; its references were its own tests and coverage-floor entry. Removed the LPBS RemoteReporter because it was only self-tested, described itself as future/flag-off until the LPBS gateway shipped, and had no production constructor, Start, or Submit caller. Updated the SEAMS index. Kept the local UsageService recorder and the still-connected LPBS client/provider interfaces. Also aligned one existing summarize test call with the current API, supplying empty optional receipt/policy values; focused summarize tests pass.
## Architecture Drift

Use this section for deferred findings from \`architecture-lenses\`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
\`ARCHITECTURE.md\`, \`SEAMS.md\`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|

### 2026-05-17 — Browser-WS \`/api/v1/voice/stream\` lacks a transport test

**Symptom:** No automated test exercises the WS handshake, message echo, abrupt close, or context-cancel propagation paths in \`api/handlers/stt/stream_ws.go\`. Regressions surface only via the UI smoke run.

**Root cause:** Deferred from the 2026-05-17 post-extraction cleanup. A meaningful WS test requires a \`httptest\`/\`gorilla/websocket\` rig wired to the same session pub/sub fan-out the production transport uses; the fan-out seam exists (\`internal/session/session.go\`) but the handler-level fake is not yet hoisted into \`handlers/stt/mocks/\`.

**Workaround:** Manual UI smoke after restart confirms handshake + echo. Connect-RPC bidi \`TranscribeStream\` is covered end-to-end and shares the segmenter + strategy path, so most regressions surface there.

### 2026-05-17 — Summarize default awaits newer-model local benchmark

**Symptom:** The summarize catalog includes newer research-backed candidates
(\`gemma3:4b\`, \`gemma3n:e2b\`, \`phi4-mini:3.8b\`), but the default remains
\`llama3.2:3b\`.

**Root cause:** The target machine has Ollama 0.11.7 and installed
\`llama3.2:3b\`, \`llama3.2:1b\`, \`qwen2.5:3b\`, \`mistral:latest\`, \`qwen3:*\`,
and \`deepseek-r1:8b\`, but not Gemma 3, Gemma 3n, or Phi-4 Mini. The system
must not pull large model artifacts without explicit operator approval.

**Workaround:** Use the web-console model picker. Missing recommended models
show \`ollama pull <model>\` commands for manual installation.

**Real fix:** After an operator installs candidates, benchmark them against
representative assistant-output fixtures. Change the default only if a
non-reasoning candidate beats \`llama3.2:3b\` on latency and preserves key facts
without prompt leakage or visible reasoning.

**2026-11-19 registry note:** \`resources/ollama/model-policy.json\` now names
\`qwen3.5:9b\` as the \`summarize.default\` primary (with \`qwen3.5:4b\` and
\`llama3.2:3b\` as fallbacks), which is newer than the 2026-05-17 benchmark. The
model registry reflects that reality: \`DefaultFor(summarize)\` is the runtime
primary \`qwen3.5:9b\`, while \`CommercialDefaultFor(summarize)\` is the
evidence-backed, licence-verified \`llama3.2:3b\`. This entry is not closed: there
is still no local receipt that \`qwen3.5:9b\` is non-reasoning, fast enough, or
licence-clear. A drift test fails if the registry stops mirroring the policy,
so this tension cannot be forgotten silently.

**Owner:** unassigned.

**Refs:** \`api/internal/summarize/model_policy.go\`,
\`api/internal/summarize/model_policy_test.go\`,
\`packages/proto/schemas/audio-tools/v1/summarize/summarize.proto\`.

**Real fix:** Hoist a \`handlers/stt/mocks/ws.go\` helper and add \`stream_ws_test.go\` with the four scenarios from plan §G2 (\`audio-tools-post-extraction-cleanup-professionalization.md\`).

**Owner:** unassigned.

**Refs:** plan §G2; \`api/handlers/stt/stream_ws.go\`.


### 2026-05-17 — No cross-domain integration test

**Symptom:** Transcribe → summarize → synthesize is exercised only by per-handler unit tests; a regression in the cross-domain seam (chain wiring, normalizer ownership) would not fail a single test.

**Root cause:** Deferred from the 2026-05-17 post-extraction cleanup. The work needs deterministic stubs for all three external provider tiers in \`internal/testutil/\` and a new \`api/tests/integration/\` package.

**Workaround:** \`vrooli scenario test audio-tools\` exercises the per-handler happy paths against real chains with local providers when present; the cross-domain happy path is the diagnostics UI tab path.

**Real fix:** Add \`api/tests/integration/cross_domain_test.go\` per plan §G3 once the WS deferral above is also being worked.

**Owner:** unassigned.

**Refs:** plan §G3; \`api/internal/testutil/\`.

### 2026-08-16 — Historical contract-discovery limitation

The audio-tools PRD named long-form dictation trust, provider-parity stable
engines, and explicit speaker-policy safety as P0 targets. The attempted goal
comparison could not finish: \`swarm-manager goals list --json\` reported the
scenario stopped, and startup remained in a dependency-start operation. No
W1–W3 gate was claimed. This records that investigation's unavailable evidence,
not a current requirement to create a Swarm goal or read every unrelated goal.
See [Work ladder](#work-ladder) for the current interpretation.


### 2026-05-17 — Pre-existing standards drift not in cleanup scope

**Symptom:** After the 2026-05-17 cleanup, \`scenario-auditor standards scan audio-tools\` reports 6 violations (down from 29 at start of the cleanup):

- (critical) \`Makefile\` "required_layout" — the rule message ("Add the required resource at Makefile") is opaque; reference scenarios like \`agent-inbox\` pass with a similar Makefile shape, suggesting the rule may be probing for a specific resource file rather than a target.
- (high) \`requirements/index.json\` requirement \`FOUND-001\` \`prd_ref\` points to a PRD section that does not exist in \`PRD.md\`.
- (medium) \`requirements/01-foundation/module.json\` lacks operational-target linkage for \`FOUND-001\`.
- (medium) \`ui/src/components/shell/SettingsDrawer.tsx\` registers a \`keydown\` listener inline rather than via a dedicated \`ui/src/hooks/\` hook.
- (medium) \`emitShortcutIntent\` is not called from any keyboard hook in \`ui/src/hooks/\`.
- (medium) \`api/internal/stt/segmenter/testaudio/\` (fixtures-only package) has no \`*_test.go\`.

**Root cause:** All six predate the post-extraction cleanup plan. Fixing them requires substrate-level work (PRD-schema edits, a shared keyboard hook + \`emitShortcutIntent\` wiring, a standards-rule waiver for fixture-only packages, and reverse-engineering the \`required_layout\` resource check) that is out of scope per the plan §4 ("Cross-scenario refactors" and "No new product features").

**Workaround:** The cleanup pass reduced the violation count from 29 → 6. The remaining ones do not block \`go test\`, \`pnpm test\`, or runtime smoke; they only fail the \`standards\` phase of \`vrooli scenario test audio-tools\`.

**Real fix:** A separate cleanup plan should pick these up alongside the next round of standards-rule updates so the changes ride a single deploy.

**Owner:** unassigned.

**Refs:** plan §I; \`scenario-auditor standards scan audio-tools\` (last run 2026-05-17, job \`standards-f069d6b4-…\`).

### 2026-05-17 — \`audio\` domain scope is intentionally smaller than the proto surface

**Symptom:** Proto \`audio-tools/v1/audio\` declares Transcode + ops envelopes for trim/merge/split/fade/volume/normalize/metadata, but the only end-to-end (handler + CLI + UI) flow is Transcode.

**Root cause:** Post-extraction the \`audio\` domain scope was explicitly limited to "Transcode + ffprobe-backed primitives" so the cleanup did not balloon. The primitives exist in \`api/internal/audio/\` (trim/merge/split/fade/volume/normalize) and are exercised by \`ops_test.go\`, but no CLI verb or UI panel exposes them.

**Workaround:** Callers that need those ops can drive them via the multipart REST exception once handlers are added; for now use ffmpeg out-of-band.

**Real fix:** When a downstream scenario demands a second audio op end-to-end, lift the scope decision per [\`DECISIONS.md\`](DECISIONS.md) (2026-05-17 entry) and implement handler + CLI + UI for the new op set.

**Owner:** unassigned.

**Refs:** \`docs/domains/audio/\`; plan §D.

## Test Gaps

The "audio-tools-test-architecture-lift" plan (2026-05-17) closed the
0%-coverage chains (sttchain ≥85%, ttschain ≥85%, summarizechain ≥85%)
and the legacy audio-ops gap. The remaining shortfalls below are
intentional deferrals governed by separate plans.

### \`internal/stt/pipeline\` — stale \`voice.Service\` coverage item (closed)

The legacy \`voice.Service\` migration description above is stale. The package
now exposes \`WhisperTranscriber\`, which owns local Whisper batch transcription
and recovery; stream, wake-word, and speaker configuration have separate
owners. A source search found no \`voice.Service\` or \`voice.NewService\` callers
under \`api\`, and the WebSocket handler/provider path uses the strategy and
\`sttchain\` boundaries. The obsolete wrapper inventory and migration dependency
are therefore not current work.

**Measured evidence:** \`go test ./internal/stt/pipeline
-coverprofile=/tmp/audio-tools-stt-pipeline.cover\` passed on 2026-11-22 with
72.9% statement coverage (the old ~24% figure is no longer representative).
Targeted behavior and cancellation tests now cover \`SetLogger\`,
\`AutoEnsureEnabledFromEnv\`, \`SetWhisperConcurrency\`, \`WhisperAvailable\`, and
\`EnsureWhisperAvailable\` at 100% each. A \`-race\` run of the package passed.

**Remaining test gaps:** Coverage is not uniformly high. \`speaker_client.go\`
still has multiple untested live operations (\`Ready\`, \`Info\`, profile/clip CRUD,
and extraction); \`session_speaker.SmoothedScore\`, speaker-config persistence,
and some configuration/wake-word error branches also remain below full
coverage. These are explicit follow-up test gaps, not superseded
\`voice.Service\` wrappers.

**Status:** Closed as stale. No production behavior was removed.

### \`internal/byok/{deepgram,openai_whisper}.go\` streaming methods

**Symptom:** \`TranscribeStreaming\` on the Deepgram + OpenAI Whisper
adapters is ~5% covered. The non-streaming \`Transcribe\` paths and
non-streaming adapters (openai_tts, elevenlabs, openrouter) sit ≥80%.

**Root cause:** Streaming requires a real WS upstream. Asserting against
a Deepgram-shaped WS handshake means standing up a fake vendor WS
server per test — substantial work for a single adapter branch.

**Workaround:** The streaming dispatch routing IS covered through the
\`BYOKProvider.TranscribeStreaming\` shape tests; only the adapter-side
WS wire format is uncovered.

**Real fix:** Add \`vendor-ws\` test fixtures alongside the streaming
plan that ships native streaming to production. Until then, regressions
are caught by the smoke-test phase of \`vrooli scenario test\`.

**Owner:** unassigned.

**Refs:** plan [audio-tools-test-architecture-lift §3 Symptom 1];
streaming-decoupling plan in [\`PRD.md\`](../../PRD.md).


### Handler \`time.Now()\` and \`log.Printf\` bypasses (Phase 2 / Phase 6 deferred)

**Symptom:** Five handler files still call \`time.Now()\` directly
(\`handlers/{summarize,session,tts,usage,stt}/...\`) and
\`handlers/stt/stream_ws.go\` still uses \`log.Printf\`.

**Root cause:** The substitution wiring (Clock on Deps, Logger on Deps)
is straightforward but touches every module constructor + handler test;
the test-architecture-lift plan deferred this to a focused follow-up so
the chain-coverage work could land independently.

**Workaround:** Tests assert presence/monotonicity of timestamps via
the protobuf shape rather than exact bytes; WS error tests rely on the
status-code branch rather than captured log lines.

**Real fix:** Follow-up plan "handler clock + logger seam wiring" —
mechanical refactor across the listed files.

**Owner:** unassigned.

**Refs:** plan [audio-tools-test-architecture-lift §3 Symptoms 3, 4]
(Phase 2 + Phase 6 carry-over).


### BYOK + LPBS client \`*http.Client\` direct construction (Phase 1 deferred)

**Symptom:** Eight files (\`internal/byok/{openrouter,openai_tts,openai_whisper,deepgram,elevenlabs}.go\` and
\`integrations/lpbs/clients/{stt,tts,summarize}_client.go\`) construct
their own \`*http.Client\` instead of accepting \`httpc.Doer\`.

**Root cause:** The \`httpc.Doer\` seam is declared and \`mocks.FakeDoer\`
ships ready, but threading the field through every constructor +
registry + \`main.go\` is invasive enough that the test-architecture-lift
plan split it out.

**Workaround:** Existing adapter tests use \`httptest.NewServer\` for
wire-format coverage; payload drift is caught indirectly via the
provider-level chain tests.

**Real fix:** Follow-up plan "BYOK adapter Doer wiring" — replaces
\`HTTPClient *http.Client\` with \`Doer httpc.Doer\` and adds FakeDoer
payload-shape tests alongside the existing httptest wire-format tests.

**Owner:** unassigned.

**Refs:** plan [audio-tools-test-architecture-lift §3 Symptom 2]
(Phase 1 carry-over).


### Consumer-side \`store.*Repository\` seams (Phase 5 deferred)

**Symptom:** Admin handlers in \`handlers/{settings,usage,tts,stt}/\`
import \`internal/store\` concrete sqlite types directly; handler tests
must spin SQLite to drive success paths and cannot inject error
branches.

**Root cause:** Per-handler \`Repository\` interface introduction +
\`var _\` checks + fake substitution is a multi-package refactor.

**Workaround:** Existing sqlite-backed handler tests already cover the
happy paths; SQL semantics are pinned in
\`internal/store/*_sqlite_test.go\`.

**Real fix:** Follow-up plan "consumer-side repository seams" —
introduces narrow \`Repository\` interfaces in each handler package and
co-located \`mocks/repository.go\` fakes.

**Owner:** unassigned.

**Refs:** plan [audio-tools-test-architecture-lift §3 Symptom 7]
(Phase 5 carry-over).


### Kyutai/Passthrough segments are not speaker-gated

**Symptom:** Speaker verification only protects the Whisper VAD path. The
Kyutai streaming engine (and any Passthrough strategy) emits segments with no
\`seg.Audio\`, so the egress \`SpeakerStage\` returns them unchanged — a non-enrolled
voice transcribed via Kyutai is NOT rejected.

**Why unsolved:** Identity gating needs the segment PCM, which the streaming
passthrough path never carries. A real fix means either teeing the session PCM
to a side-channel verifier or having the engine surface per-segment audio.

**Workaround:** Use the Whisper engine when speaker isolation must be enforced;
the admin status surface and CLI \`speaker-status\` both print this caveat.

**Owner:** unassigned. **Refs:** plan
\`speaker-verification-quality-overhaul-tiers-1-2\` §Non-goals.

### Speaker threshold needs live calibration

**Symptom:** The canonical default \`speaker.threshold\` is \`0.5\` across all
layers, but the right cutoff depends on the ECAPA embedding distribution for the
actual enrolled voices + room. The plan ships VAD trim + multi-clip centroid +
session smoothing (which recover most of the lost separation) but the final
number is environment-specific.

**Resolution path:** After Phase 1 (VAD trim) is live, enroll 3–5 real clips and
run a genuine-vs-impostor mic test; set \`speaker.threshold\` to sit between the
two bands and record it in \`docs/reference/configuration.md\`. This cannot be
unit-tested (needs real voices + the model).

**Owner:** unassigned (operator calibration step).

### Retained — speaker-verification Python test entrypoint needs an owner route (2026-11-21)

**Disposition:** the old startup failure is obsolete as a current command. Its
named \`resources/speaker-verification/test/integration-test.sh\` no longer
exists, nor do the cited root-level \`scripts/resources/common.sh\` and
\`scripts/lib/service/repository.sh\`; no other checked-in caller references the
removed integration script. Do not file this as a current runtime failure.

**Remaining gap:** the resource still has in-image Python suites at
\`test/test_vad.py\` and \`test/test_profiles.py\`, but its \`Makefile\` \`test\` and
\`check\` targets run only the Go CLI tests. The Python test modules document
manual \`python -m unittest\` invocations and skip model-dependent cases when
Torch is unavailable, but no current resource-owned command joins those suites
to the normal quality gate.

**Owner:** speaker-verification resource owner. **Next step:** add or select a
supported resource-owned test operation that invokes the Python suites in the
declared runtime and reports skipped model-dependent cases honestly; then
verify that operation through the resource's normal lifecycle. Do not revive the
removed generic shell path as an Audio Tools workaround.

+### 2026-07-08 — Continuous-speech client-side auto-stop follow-up

**Symptom:**
With the backpressure wedge closed, mid-recording text is no longer lost — but a
*different* symptom surfaced: during continuous speech the **microphone abruptly
stops** ~10 s in (the mic-state is now honestly synced to transcription, so a
premature turn-end is visible instead of masked by the frozen decode loop).
Root cause is CLIENT-side, not this relay: **kyutai/passthrough emits NO
\`vad-state\`** (grep \`server.py\`/\`provider_kyutai.go\` — none), so on the browser
the auto-stop SSOT (\`decideAutoStop\` §2) falls back to the **client RMS VAD as
the sole turn-ending authority**. Under whisper-VADSegment the server VAD was
authoritative and vetoed the client VAD's known "stops while I'm still talking"
false positives; under kyutai that veto is gone. A client-VAD silence verdict
then ends the whole turn whenever the analyser reads silence — a **muted mic
track** (sleep/wake, default-device change, another app seizing the mic; the
\`ended\` event does NOT fire for muting — see web-console \`streamHealth.ts\`), a
**suspended AudioContext**, or an adaptive \`speechThreshold\` risen above the
user's real speech RMS. Relay evidence: sessions close \`graceful=true
tailFinalDelivered=true\` in a regular ~10–12 s cadence (client-initiated stops,
not backend kills). The frequent kyutai "reaping wedged prior streaming session"
warnings are a *separate*, benign-but-noisy lock-release-latency issue (the relay
dials kyutai + takes \`MODEL.lock\` the instant the browser WS connects — including
the mount-time pre-connect — so overlapping/rapid sessions reap already-delivered
holders; not the cause of the mic stop). Web-console side owns the fix; see its
\`PROBLEMS.md §8c\`. First step done: full client instrumentation + honest
track-mute/ended/recorder-error handling + a guard that suppresses a client-VAD
auto-stop when the audio source is not delivering samples.

### 2026-07-12 - Kyutai deterministic long-form replay remains an unresolved flow-control blocker

**Symptom:** The persisted 15-minute provider profile completed the paced
Kyutai pass (11,337 frames, 154 segments, graceful end) but its preceding
deterministic quality pass closed after 4,330 frames and 57 segments with
\`1011 keepalive ping timeout\`. The incomplete quality transcript had WER 0.663
and 638 deleted reference words; the matching Whisper row had WER 0.042.

**Confirmed contributing defect:** Fast replay generates partials faster than
the Go \`KyutaiProvider\` downstream consumer can drain them. The adapter used to
synchronously write every partial into its 16-slot \`events\` channel, which could
stop \`ReadMessage\`. \`provider_kyutai.go\` now uses a nonblocking bounded send for
the explicitly droppable partial class; segments, session states, errors, and
done remain ordered durable delivery. The 60 MiB
\`TestKyutaiProvider_PartialFloodDoesNotBlockWebSocketReader\` oracle failed before
the change and passes after it.

**Current status (superseded by the 2026-07-12 addendum below):** The full
deterministic native-streaming evaluation still closed at the same 50-second
boundary after this repair (1,865 frames / 24 segments), so partial pressure
was not the sole cause. The server reported \`disconnect\`, not \`reaped\`, and the
paced pass proved the same model/audio path could complete. That isolated the
native transport's unbounded fast writer as the remaining suspect.

**Interim evidence correction:** A \`realtime\` evaluation cell now derives
WER/safety from its first paced repeat and no longer attaches an incompatible
unpaced deterministic failure to a row labeled \`realtime\`. An explicitly
deterministic Kyutai cell remains a required, currently failing qualification
until a real bounded transport-flow-control fix exists; this is not waived for
promotion.

**Validation:** Focused \`go test ./internal/ai/sttchain ./handlers/stt\` passes.
The paced 15-minute rerun is experiment \`71a8ea70-2a4a-4a9e-9e5a-f46dbd2191a4\`.

**Refs:** \`api/internal/ai/sttchain/provider_kyutai.go\`;
\`api/internal/ai/sttchain/provider_kyutai_test.go\`; experiments
\`ddf20231-47c4-4da6-803e-1e5e33ae3fb0\`,
\`71a8ea70-2a4a-4a9e-9e5a-f46dbd2191a4\`; record \`rec-6d413e62862befd6\`.

### 2026-07-12 — Deterministic experiment runtime is bounded; Kyutai decode throughput remains below budget

**Observed:** After bounded \`processed_batches\` credits were deployed, the
former 1011 disconnect no longer occurred, but a 15-minute deterministic run
(\`02f07e9e-aff1-4714-b1b8-2b6b7d3583ca\`) was still active after 40 minutes and
was cancelled as invalid evidence. A controlled 30-second, 100ms-chunk Kyutai
probe then exceeded its 2m30s runtime budget. The same audio with 1-second
chunks completed in about 30 seconds before an abnormal normal-close; adding a
cooperative \`asyncio.sleep(0)\` every four model frames removed that early close,
but the probe still exceeded the 2m30s budget. The resource is healthy and
loaded on the RTX 4070 Ti SUPER, so this is not a CPU fallback.

**Fixed:** \`experiment.Manager\` now receives the server-computed estimate and
enforces a runtime budget of estimate + 25% headroom (minimum two minutes;
30 minutes for unestimated jobs). Deadline expiry is a persisted **failed**
experiment with \`experiment exceeded runtime budget …\`, not a permanently
running worker. The evaluation harness and native passthrough forwarding now
honor context cancellation, so the deadline actually releases the worker.
Kyutai yields cooperatively during multi-frame decode so WebSocket sender and
control tasks are not starved by one large PCM batch.

**Remaining blocker:** Deterministic Kyutai quality evidence is still not
earned. The 100ms path is too slow, while the larger-chunk path needs a clean
\`done\` plus a non-error transcript before it can be used for quality evidence.
Do not promote Kyutai or launch the corrected 60-minute profile until a
deterministic 15-minute run completes inside its budget with valid transcript
and safety results.

**Validation:** \`go test ./internal/eval ./internal/stt/strategy
./internal/experiment ./handlers/experiment\` passes; Kyutai resource suites
pass (13 Python tests and \`make test\`). Live timeout experiment
\`3b2c3483-3a9c-483a-bf6c-0667b838e744\` failed at exactly 2m30s with the
persisted runtime-budget diagnostic. The scheduling probe
\`cfd5bafd-83ca-4bb5-a06a-0c337d3f0cfa\` ran until the same enforced deadline
without the prior 30-second close-before-\`done\`.

### 2026-07-12 — Whisper unary batch is not a valid long-form promotion cell

**Symptom:** persisted full-real-time experiment
\`1733d1a1-4997-4227-af2a-6719a976f699\` reported a 24-word dropped span for
the Whisper row. Before 2026-07-12, promotion aggregation retained only WER
and duration, so that observed safety failure was omitted from the provider
verdict.

**Investigation:** the report's Whisper row is
\`whisper-local/batch/realtime\`, not a segmented streaming strategy. Its one
\`batchSession\` call buffered the entire 3,602,659 ms synthetic clip before
calling the unary Whisper provider. The report confirms a real loss signal:
the repeated product sentence occurs 25 times in reference and 24 times in
the hypothesis, and the hypothesis also ends with an unwanted trailing phrase.
The repeated corpus material can make global word alignment choose a longer
equivalent deletion path, but it does not explain away the missing occurrence
or make the unary result valid no-loss evidence.

**Resolution shipped:** promotion evidence is now keyed by engine, strategy,
and policy profile. Reports with no strategy identity are retained as legacy
history but excluded from promotion aggregation. Observed safety state is
preserved in \`ReplayMeasurement\`, so the batch cell explicitly fails the
dropped-span gate instead of contaminating or being hidden by another Whisper
strategy. The UI contract renders the cell identity.

**Remaining gate:** qualify a real-time long-form \`whisper-local/vad_segment\`
or \`whisper-local/overlap_agree\` provider cell (and the corresponding Kyutai
cell) through the full duration/fault/browser/device matrix. Do not use a
whole-clip unary batch baseline as stable-engine promotion evidence.

**Refs:** \`api/handlers/eval/evalrun.go::batchSession\`,
\`api/internal/stt/trustfloor/rubric.go\`,
\`api/handlers/experiment/promotion_evidence.go\`.

### 2026-07-12 — Dedicated qualification artifacts were previously not durable promotion input

**Symptom:** Deterministic BAS and focused recovery tests could prove parts of the
dictation contract, but promotion verdicts consumed only replay WER, duration,
and safety fields. Browser, recovery, fault, interval-accounting, and device
claims therefore had no persisted, inspectable connection to a provider cell.

**Resolution shipped:** \`ExperimentService\` now owns a qualification-evidence
ledger. Each record stores engine, exact model id, strategy, policy profile,
required artifact reference, pass/fail result, timestamp, and machine identity.
Trust-floor aggregation consumes only records from the exact
engine/model/strategy/policy cell on the same host/OS/architecture as the
report; it never credits another model, strategy, or machine. Failed records
remain visible as explicit promotion reasons. The operator surface is
\`audio-tools experiment record-evidence\` and \`audio-tools experiment
list-evidence\`.

**Remaining gate:** Existing browser and recovery results have not been
backfilled, because their stored artifacts do not assert one exact provider
strategy/policy cell. Run each real qualification profile with that identity and
record its durable artifact reference. Physical-device evidence remains absent.

**Refs:** \`api/internal/experiment/\`; \`api/handlers/experiment/promotion_evidence.go\`;
\`cli/domains/experiment/\`; \`packages/proto/schemas/audio-tools/v1/experiment/experiment.proto\`.

### 2026-08-03 — Resolved: browser package lifecycle uses package-local test tool

**Symptom:** \`vrooli package test audio-capture-browser\` reached the package test
lifecycle but could not execute because \`packages/audio-capture-browser\` had no
local \`vitest\` binary, despite declaring Vitest as a development dependency.

**Root cause:** The package had no standalone pnpm boundary or package-local
dependency installation. The available Vitest binary from another workspace
package could run the tests, but that did not prove the package's normal
lifecycle environment.

**Workaround:** None remains. The borrowed executable was removed from the
validation path.

**Resolution:** Added the package-local pnpm workspace and lockfile, installed
Vitest, React, React DOM, jsdom, Testing Library, TypeScript, and Node types
through SDA, removed the absolute-path Vitest aliases, and reran the normal
control-plane lifecycle. The result is 7 test files and 94 tests passing, including
all 15 long-session recovery tests; the package-local
\`node_modules/.bin/vitest\` is the executable used by the run.

**Owner:** package governance / workspace maintenance.

**Refs:** \`packages/audio-capture-browser/package.json\`,
\`internal/cli/packagecli/\`, \`internal/packagegov/\`.

### 2026-08-09 — Reliability contract is repaired; strategy qualification remains provider-scoped

**Resolution update:** Kyutai now emits durable processed-coverage acknowledgements
only after its backend \`processed_batches\` cursor confirms the exact audio range;
the resource orders the final processed cursor before terminal \`done\`. The
accelerated browser lane passes 60 simulated minutes, bounded retention, exact
interval accounting, zero duplicate committed segments, zero silent terminals,
and a 10 messages/second wire rate. The full audio-tools Go suite and the
browser package suite pass.

**Qualification boundary:** Kyutai declares streaming-only capability and
therefore correctly refuses the batch-oriented \`OverlapAgree\` and \`VADSegment\`
provider cells; their explicit 60-minute Kyutai runs produced zero commits and
failed safety rather than being counted as success. The valid batch-provider
cells \`whisper-local/overlap_agree/realtime\` and
\`whisper-local/vad_segment/realtime\` completed under durable experiment ids
\`21a7ca20-afb8-4e3b-8568-7503588dae2c\` and
\`e8df3a39-835f-4b6e-9f09-6265af481ab8\`. Both realized 3,602,659 ms, but both
failed the configured safety threshold: Overlap committed 244 segments with a
103-word maximum dropped span and VAD committed 222 segments with a 15-word
maximum dropped span, versus the four-word threshold. VAD had lower measured
WER (0.041 versus 0.149) and lower p50/p95 finalization latency (431 ms versus
984 ms), but neither is a promotion-quality long-form result.

**Browser evidence:** Routed Workflow Health runs pass four Dictation Studio
cases for Kyutai passthrough, Overlap, and VAD selections. These are short
product-path/fault-recovery checks, not 60-minute swarm-manager qualification;
the swarm-manager workflow remains blocked by unrelated static workflow errors
and unavailable routed storage isolation. No physical-device evidence exists.

### 2026-08-09 — Long-form repair in qualification: bound transcript overlap and retain failed audio

**Finding:** The first valid Whisper long-form cells completed for the full
duration but failed the four-word dropped-span safety gate. Investigation found
that transcript deduplication searched the entire committed transcript; repeated
phrases later in a long dictation could be mistaken for physical pre-roll and
silently removed. The VAD and Overlap strategies also advanced their audio
cursor after provider errors, empty results, or a no-op stall fallback.

**Resolution:** \`DeduplicateOverlapBounded\` now limits overlap to the configured
physical pre-roll/prompt window. VAD and Overlap retain the pending audio after
provider errors and empty transcripts, and advance the cursor only after a
non-empty forward commit. Focused regressions cover repeated phrases, physical
overlap, VAD retry, Overlap retry, and forced/stall commit behavior. The focused
and full audio-tools Go suites pass; the browser workflow schema/action labels
validate cleanly. Fresh two-cell Whisper realtime qualification is running as
\`002334e2-cf24-4b4b-990f-82f7ff6cc983\`; until its terminal report is inspected,
the repair is implementation evidence rather than a promotion claim.

### 2026-08-09 — Overlap qualification budget exposed redundant post-force calls

**Observation:** The explicit 15-minute production-default Overlap run
\`9913fdc1-78c9-4eed-b933-9bd8d1fbc556\` failed without a report after exceeding
the 18m45s runtime budget. The server trace showed a successful 10.1-second
force commit, then a clean VAD boundary produced a first LocalAgreement
hypothesis that could not commit until a later hypothesis. The next force
window therefore paid for an additional Whisper call over audio that had
already been safely cursor-advanced.

**Root cause:** \`lastAdvanced\` correctly identified that the force commit had
moved the cursor, but the normal two-hypothesis LocalAgreement path still ran
for the first clean post-force VAD boundary. That added a call per safety
window and pushed real-time consumption behind the source.

**Resolution:** Production VAD mode now accepts the first non-empty result at
the first clean boundary after a successful force commit, advances the cursor
through that boundary, and retains the stricter two-hypothesis behavior for
stopwatch mode and timestamp-alignment tests. A focused regression,
\`TestOverlapAgree_VAD_CommitsFirstPostForceBoundary\`, prevents the redundant
uncommittable partial from returning. The focused strategy suite passes; a
fresh 15-minute qualification is queued as
\`4798add5-9aa9-4558-9bbf-1327d53b3fbc\`.

**Qualification result:** The post-force repair completed inside the runtime
budget with WER 0.1282 and max dropped span 5. Tightening the window to 8s
made the result worse (WER 0.1453, max dropped span 9). Retaining a 300ms
physical tail improved WER to 0.1244 but did not clear the five-word safety
failure in \`b83d7890-9f9f-4c51-b387-8c2d4e2a3b90\`. Overlap therefore remains
diagnostic-only; the shipped VAD strategy is the production qualification
candidate.

### 2026-08-10 — VAD long-form boundary context is qualified

**Finding:** The first fresh 60-minute VAD run with 300ms pre-roll completed
the duration but failed safety with a nine-word contiguous deletion. The edit
trace localized the loss to a boundary phrase, not ingress or egress loss.

**Resolution:** Production VAD pre-roll now uses the existing supported maximum
of 800ms, and the bounded text-overlap calculation follows that physical
window. The 800ms 15-minute run \`8defc72d-6b88-46c4-a0de-0e8655866152\` passed
with max dropped span 0. The final 60-minute run
\`e3d3b4ff-5c1f-4fd7-8cef-b92552c1f73c\` realized 3,602,659ms, passed safety
with max dropped span 1 against threshold 4, measured WER 0.022994, RTF
0.031923, 226 commits, and 488.8ms p95 finalization latency.

### 2026-08-17 — Native audio-resource migration remains release-evidence-blocked

**Symptom:** The portable-native resource phases cannot be fully promoted.
\`resources/sherpa-onnx\` now exists and implements the TTS, streaming STT,
speaker, and separation contracts, while the old Kokoro/speaker/Kyutai
resources remain present until signed adapter artifacts and target smoke
evidence exist. Whisper still lacks macOS target delivery and complete
cross-platform smoke evidence. The verified Linux Whisper CPU path is not
blocked on the consumer quality smoke: its provider path, clean-speech WER,
latency, and no-speech suppression pass.

**Root cause:** The repository lacked a native audio stack when the plan
started. The Whisper slice now has a verified v1.9.2 server archive, a
verified medium GGML model, a live Linux CPU contract adapter, and shared
egress suppression for native silence hallucinations. The sherpa resource has
the corresponding resource-local adapters and a qualified Linux build, but
the release boundary and target evidence are not complete. A native CPU path
is not evidence of CUDA parity or macOS/Windows readiness.

**Workaround:** Keep only the still-unmigrated Docker resources and their
truthful platform claims. Do not mark the Whisper Linux result as a
cross-platform qualification or hand-stage any artifact.

**Real fix:** Publish signed target-native sherpa adapter bundles, add the
remaining Whisper platform artifacts and smoke evidence, run the
cross-platform target matrix and same-corpus comparisons, then delete the
remaining replaced Docker resources only after those gates pass.

The missing Whisper macOS build step is now explicit and fail-closed in
\`resources/whisper/Makefile\`; it can only produce a candidate on a Darwin/arm64
host from a source tree pinned to whisper.cpp v1.9.2. That narrows the release
work without changing the unsupported platform claim. Signing, checksum
publication, managed-service smoke, and acquisition wiring remain unresolved.

The local release-authority prerequisite is now resolved: the managed RSA-3072
authority is initialized in the native secure store and its public trust anchor
matches. This changes signing from unavailable to actionable; it does not
substitute for target-native builds, external artifact publication, or target
smoke evidence.

The signer/verification mismatch for \`layout=dir\` managed-service artifacts is
also resolved. \`SignStage\` now computes directory tree digests rather than
trying to read a service bundle as one regular file; the focused production
verification regression passes.

The scenario's monetization declaration gap is resolved for the current
implementation boundary: \`.vrooli/monetization.json\` declares the hosted
\`ai_credits\` Class-A meter and its real LPBS seams, with BYOK explicitly exempt
from the hosted charge. The LPBS gateway remains disabled pending its endpoint
implementation, so no hosted revenue readiness is claimed.

**Owner:** unassigned.

**Refs:** \`resources/whisper/docs/whisper-cpp-managed-service-assessment.md\`,
\`docs/resources/deployment-contract.md\`, \`.vrooli/schemas/acquisition.schema.json\`,
the active plan phases 9–13.

### 2026-08-18 — Realtime qualification was still using Chromium fake media (resolved)

**Symptom:** The realtime soak driver supplied a WAV through BAS's
\`fake_media.microphone_wav\`, so even a successful realtime run could only
prove Chromium's synthetic capture path rather than an operating-system audio
device.

**Root cause:** BAS's PipeWire qualification implementation existed, but was
selected only through a process-wide environment variable. The soak driver had
no per-session way to request it.

**Fix/evidence:** Added \`audio_device_evidence\` to the BAS session contract and
made the realtime soak request it explicitly while retaining the WAV as the
corpus played into the user-owned PipeWire sink. After a lifecycle restart, a
bounded start/inspect/close smoke returned the browser label
\`Vrooli_Qualification_Microphone\`, sample rate 48 kHz, mono; BAS reported zero
sessions after close and \`wpctl status\` contained no qualification nodes.
BAS TypeScript and focused soak-driver checks passed. The soak handler now
persists the existing cell-keyed \`browser_product_path\` and \`device\`
qualification records, including failed runs. A short live realtime smoke
returned both records through \`audio-tools experiment list-evidence\`, with the
device record passed and explicitly scoped to the OS/browser capture path. The
actual 15/60-minute swarm-manager qualification runs remain open.

**Owner:** audio-tools / browser-automation-studio.

**Refs:** \`scenarios/audio-tools/api/internal/soak/driver.go\`,
\`scenarios/browser-automation-studio/playwright-driver/src/routes/session-start.ts\`,
\`scenarios/browser-automation-studio/playwright-driver/src/session/manager.ts\`,
\`scenarios/browser-automation-studio/playwright-driver/src/session/audio/device-evidence.ts\`.

### 2026-08-18 — Deterministic backend-restart recovery was terminalized too early (resolved)

**Symptom:** The existing reconnect workflow ended in \`Recording failed.\`
after the one-shot \`backend_restart\` fault instead of recovering the turn.

**Root cause:** The server fault seam marked the shared session ledger terminal
before the browser could reconnect and replay the durably received chunk. The
browser also forwarded the recoverable fault through its terminal \`onError\`
callback, allowing the host UI to fail the recording before reconnect.

**Fix/evidence:** Recoverable backend restart now leaves the ledger open for
resume, while the browser records the event as status and lets \`onclose\` run
the bounded replay path. \`pnpm vitest run src/longSession.test.ts\` passed 19/19
and the focused STT transport-fault handler tests passed. The full scenario
workflow was not rerun in this focused pass.

**Owner:** audio-tools.

### 2026-08-18 — Retained recovery did not identify its actual batch provider (resolved)

**Symptom:** A Kyutai streaming turn that entered retained-audio recovery gave
no explicit browser status that the turn had moved to bounded batch
transcription. The short product-path diagnostic then showed the batch request
falling through to Whisper, whose activity-edge companion was unavailable.

**Root cause:** The shared transport contract returned only transcript text for
retained recovery, so the browser could not record the unary response's
provider/model identity or announce the recovery boundary separately from the
original stream failure.

**Fix/evidence:** Recovery now emits a bounded-batch status before the request,
accepts optional provider/model metadata, records it in the metadata-only
diagnostic, and reports completion with the actual identity when available.
The audio-tools adapter maps the Connect transcription response into that
metadata-preserving seam. The focused recovery regression passed; package
build and UI TypeScript checks passed. The short end-to-end diagnostic remains
failure evidence for the unhealthy Whisper companion, not quality evidence.

**Owner:** audio-tools.

**Refs:** \`packages/audio-capture-browser/src/transport.ts\`,
\`packages/audio-capture-browser/src/pcmVoiceStreamProvider.ts\`,
\`scenarios/audio-tools/ui/src/audio-integration/api/voice.ts\`,
\`/tmp/audio-tools-soak-diagnostic.json\`.

The attempted server-owned realistic 60-minute Kyutai run on 2026-08-18
exited nonzero without persisting its requested evidence file. It is retained
as an invalid attempt only; no duration or quality credit is assigned. A
replacement continuous-shape Swarm Manager run is currently server-owned under
\`/tmp/audio-tools-swarm-continuous-qualification.log\`, with its requested
artifact at
\`coverage/browser-soak-swarm-continuous-authoritative.json\`; no credit is
assigned until that artifact is complete and qualified.

### 2026-08-18 — Native sherpa publication is reproducible but not promoted

The sherpa resource now owns a target-native \`release-stage\` path that bundles
the cgo server with its matching shared runtime tree, computes the same
deterministic tree digest used by managed-service acquisition, emits a
deterministic tar/gzip archive and \`entry_path\`, and signs the stage through
the managed release authority. The path rejects foreign hosts and has build
recipes for Linux amd64/arm64, macOS arm64, and Windows amd64. This removes
local packaging ambiguity, but it does not create external publication or
target smoke evidence by itself. Keep acquisition and deployment targets
explicitly unsupported until each signed artifact is published and exercised
on its target operating system.

### 2026-08-18 — Correction: device qualification is automated

Earlier entries described a planned manual microphone gate. That wording is
stale for the current long-form plan: the qualification lane creates a
user-owned PipeWire device, feeds the canonical corpus through the operating
system capture path, verifies browser enumeration, and records a durable
\`device\` evidence row. It proves the operating-system path, not a particular
microphone's analog front end, so no human microphone confirmation is required
for this automated gate. Audio Tools now has passing 60-minute device cells
\`browser-soak-1787058809596672412.json\` and
\`browser-soak-1787062594020835127.json\`; the current Swarm Manager product-path
qualification remains open.

### 2026-08-18 — Qualification evidence must follow the conformance verdict

**Symptom:** \`browser-soak-1787053929357290181.json\` failed only its committed
text-lag assertion, but the server had persisted \`browser_product_path\` and
\`device\` rows with \`passed=true\` because the HTTP request itself completed.

**Fix/evidence:** The soak handler now requires
\`conformance.Run.Evaluate().Qualified\` (and no run error) before any browser,
device, or fault evidence can pass. \`handlers/soak\` focused tests cover both a
failed assertion and a run error. Corrective failed rows were appended for the
invalid artifact, so the promotion aggregator retains the failure history and
cannot credit that run. Future runs use the corrected source identity
\`e407507ad7aa401280994117847642c57893ef11f7bd6694e5a69bc380d15190\`.

### 2026-09-08 — Voice readiness investigation: startup latency, commercial routing, and evidence quality

**Scope:** Investigation and recommendations requested by the user. No runtime
configuration, provider selection, credentials, or production implementation
was changed. Source inspection used the current shared worktree; live probes
used the running API at port 19630. Its health response reported build time
\`2026-09-06T02:14:57.167314901Z\` and source identity
\`699800654ab1a55f5a195d02bbb064557f8b75f573a92d7946d4e3626038fe1e\`.
These observations do not establish exact identity between all current source
files and the running build.

**Prior art:** Search Hub recall found the earlier speech-stack validation,
streaming, and recovery work. The program-library search provider was degraded.
The August 31 completion report under
\`~/.vrooli/plan-artifacts/speech-stack-ground-truth/verification/COMPLETION.md\`
already distinguishes working local/browser paths from missing live BYOK and
platform proof. Its latency report measured approximately 5.4–6.2 seconds of
CPU Whisper inference for a two-second clip. The August 18 entries above
record successful 60-minute Audio Tools device cells as well as earlier
failures; those successes must not be erased by citing only the older failed
attempts in \`docs/reference/eval-harness.md\`.

**Hypotheses and observations:**

| Hypothesis | Probe | Result and limit |
|---|---|---|
| Local streaming is absent or always buffers until stop | Transcribe the bundled dictation WAV through the running Connect streaming endpoint | Rejected as a universal explanation: Kyutai returned two partials and a final. This is unpaced file input, not browser or microphone latency evidence. |
| Startup discovery adds substantial delay | Time ListEngines twice, then compare timed streaming invocations | ListEngines took 10,421 ms, then 3 ms immediately afterward. One warmed stream reached its first partial at 936 ms and completed at 1,148 ms; a later stream reached its first partial at 13,929 ms and completed at 14,149 ms. Cache-sensitive startup is a strong cause candidate. A browser trace is still needed to attribute the user's entire delay. |
| Existing green tests do not prove the commercial streaming path or user-perceived speed | Inspect production wiring, fixture ownership, evidence calculations, and focused tests | Confirmed by the gaps below. Passing focused tests coexist with missing production hosted routing and insufficient performance assertions. |

**Findings, in priority order:**

1. **Engine discovery performs expensive work on session startup.**
   \`api/internal/sttengine/live.go\` probes control-plane resource status and
   acceleration sequentially, with a 10-second budget per probe and a
   30-second cache. Both \`handlers/stt/stream_ws.go\` and the Connect/segmenter
   path can await that resolver. Independent parallel diagnostic probes took
   5.4–5.9 seconds for acceleration and 10.8–12.9 seconds for status on this
   host. These are observations under current load, not model inference times.
   Move slow inventory refresh out of stream admission, coalesce refreshes,
   and use bounded provider liveness plus explicit snapshot freshness during
   admission. Preserve platform and resource policy checks.

2. **Hosted audio routing is unfinished.**
   \`api/internal/bootstrap/providers.go\` sets \`EnableVrooli\` but supplies no
   Vrooli provider to the STT/TTS/summarize chains.
   \`api/integrations/lpbs/clients/stt_client.go\` returns an unimplemented error;
   its embedded availability implementation always returns false.
   \`api/internal/ai/sttchain/provider_vrooli.go\` declares batch-only traits.
   \`api/integrations/lpbs/remote_reporter.go\` is also a stub. Its success-shaped
   return does not prove delivery or billing. Complete an authority-owned
   streaming session with reserve/settle/cancel semantics before claiming
   subscription or credit-funded voice readiness. LPBS must own cost-bearing
   enforcement and idempotent settlement.

3. **Subscription fixtures exist, but are not an audio commercial E2E lane.**
   LPBS \`api/fixture_handlers.go\` implements seed/token/balance/zero endpoints;
   \`cli/domains/devfixtures/register.go\` exposes the commands. Fixtures write
   real subscription and wallet tables through request-context database routing.
   They are not an automatic fake response selected merely because a test runs.
   The seed currently includes \`ai_credits\` and a workflow-execution limit.
   Reuse this authority with leased test storage and propagate test context
   through every hop. Add a provider-shaped streaming fake behind real audio
   adapters, with real fixture identity and wallet checks. The existing
   \`internal/testutil/vendorws\` and Deepgram adapter tests are useful substrate.
   \`soak.Options\` and \`conformance.Cell\` do not encode account/billing scenarios
   or expected credit effects.

4. **Some validation claims are stronger than their measurements.**
   \`requirements/evidence/expensive-validation.json\` claims an hour of
   continuous dictation but records 6,400 bytes at 16 kHz PCM16: 0.2 seconds
   of audio, two chunks, and two events. In
   \`cli/domains/validation/handlers.go\`, clean early STT EOF returns success;
   the sibling TTS worker can consume the remaining hour, after which the
   overall runner writes success without a minimum STT duration/coverage gate.
   Current \`validation check-freshness\` correctly fails on stale/missing
   qualification, but freshness alone would not fix this evidence defect.
   Replace this qualification claim with measured audio duration, delivered
   intervals, transcript quality, and a terminal outcome for the full run.

   The browser measurement corrections now cover the source-side gaps: the
   realtime first-partial check enforces absolute p95/maximum bands, and the
   \`capture_callback_processing_budget\` assertion records actual synchronous
   callback processing against source audio duration and fails closed when
   evidence is absent. Committed lag uses captured-versus-committed sample
   positions, and visible update gaps are recorded after rendered revisions.
   \`startToCaptureReadyMs\` now begins at the shared hook's start action and ends
   when both the capture graph and stream-ready handshake exist, exposing
   permission/setup/admission delay in realtime evidence. Direct provider users
   fall back to provider-start timing and must not claim button-to-ready. These
   are instrument and assertion fixes; native-device/browser cohorts still
   need execution through the authorized admission workflow.

   LPBS \`api/monetization_journey.go\` returns literal success observations for
   several operations, including offline access and outbox exactly-once drain;
   those branches do not execute the behavior they claim. Such observations
   must not earn behavioral credit in the proposed audio billing matrix.

5. **Portability and adoption need explicit qualification boundaries.**
   The selectable engine manifest currently contains Whisper and Kyutai.
   Bootstrap constructs sherpa-streaming, but it is absent from that manifest.
   Kyutai's Vrooli resource has a qualified Linux amd64 target and explicitly
   unsupported macOS/Windows/arm64 targets. Sherpa's resource still declares
   release-publication gates. Upstream sherpa supports multiple operating
   systems, but upstream support does not prove Vrooli artifact delivery or
   microphone behavior. Qualify exact model/artifact/OS/architecture/browser
   combinations and retain unsupported/not-measured outcomes.

   Shared browser capture, AudioWorklet with compatibility fallback, the
   segmenter, engine capabilities, and thin host adapters already exist.
   Retain these boundaries. \`useVoiceCore.ts\` remains 2,104 lines and combines
   capture lifecycle, VAD, wake words, recovery, transcription, and UI state.
   Extract cohesive controllers with behavior tests. Preserve the existing
   shared-import guards. Extend the product-path host adapter contract beyond
   the soak driver's current audio-tools/swarm-manager choices to Web Console
   and other adopting apps. Consolidate historical corrections into an explicit
   current status: the present docs still mix obsolete deferrals with repairs.

**Recommended validation design:** Extend the existing experiment and browser
soak infrastructure. Use one Test Genie-owned workflow to return durable,
bounded results for these lanes; do not create another independent pass ledger.

| Lane | What it proves | Required observations |
|---|---|---|
| Fast deterministic | Transport, recovery, routing, billing rules | Real adapters with scripted provider events; local/BYOK cause zero hosted charges; subscription and credit-only accounts; insufficient balance; reserve/settle/cancel; duplicate reconnect; partial delivery followed by failure |
| Short paced product path | User-visible streaming and speed | Real mic button and composer in each host; click-to-capture, speech-to-first-render, update gaps, committed-audio backlog, stop-to-final; warm, expired-discovery-cache, and cold-model cases |
| Model/corpus evaluation | Accuracy and compute cost | Independent varied speech references, silence/noise/accents/names/numbers, short and long speech, WER and real-time factor; exact provider/model identity |
| Duration/fault/device qualification | Sustained delivery and platform behavior | Existing duration ladder, bounded retained audio, disconnect/sleep/device-loss faults, OS device lane, artifact fingerprints; target hardware evidence for platform claims |

Add absolute latency limits and percentile sample requirements beside stability
checks. Set initial product targets explicitly, then validate feasibility per
supported provider/device: for example, warm capture acknowledgement within
200 ms, first readable text within 1 second of speech, and stop-to-final within
1 second. These are proposed targets, not measured guarantees. Keep cold model
readiness and permission prompts separately classified. Simulations can prove
policy and protocol behavior; they cannot certify vendor latency, acoustic
quality, OS permissions, or target-native artifacts.

**Validation performed:** Current-source focused Go tests passed for Audio
Tools \`internal/byok\`, \`internal/sttengine\`, \`internal/conformance\`,
\`internal/ai/sttchain\`, and \`integrations/lpbs/clients\`. LPBS \`TestFixture*\`
passed, including lease-database isolation. Selected monetization package
billing/journey/outbox tests passed. Live local streaming succeeded; timing
limits are recorded above. No new browser, long-duration, paid-provider, or
cross-platform certification run was performed. No fixes are claimed.

**Next repair order:** Correct the evidence assertions and add a bounded
consumer latency regression; remove discovery delays from admission; complete
the hosted session and fixture billing lane; then qualify providers and target
artifacts while extracting the large lifecycle/controller responsibilities.

**Owner:** audio-tools with LPBS, browser-automation-studio, shared browser
capture, and control-plane resource discovery owners.

### 2026-09-08 — Scenario skills and measurement board established; outcome qualification remains open

**Scope:** Documentation-first pilot setup, not a voice-runtime repair or launch
certificate. Added canonical usage/improve roles, the service skill declaration,
and \`audio-tools.setpoint-read\` with its contract and regression tests. Reused
engine/health/experiment owner APIs. No audio inference or paid call was initiated.

**Measured:** Governed read \`prog_62431410-3e86-4187-a422-ea2646809295\`
read two available engines, one native-streaming engine, four available
capabilities, and the explicitly selected August 10 experiment report. Its replay
metrics remained diagnostic; all nine required/pending outcome rows remained
unknown. Method: \`audio-tools-evidence-board-v1\`. This is not a fresh quality run.

**Validation:** Run \`20260908-193304-952b2cd1\` passed the \`programs\` and
\`skill-set\` phases individually at L2. Program execution is enabled by the
provider descriptor; both declared fixtures were exercised. Local board tests
cover input admission, missing targets, failed reads/children, stale and mismatched
reports, missing denominators, nonfinite values, bounds, private payload omission,
and failing cell verdicts. Three in-memory fault injections were detected by
behavioral assertions: unknown→accepted, missing WER→zero, and failed child→ok.
The PRD-to-board target check is structural traceability, not voice evidence.
Both skill frontmatters and the program JSON Schema validated. A focused check
resolved 105 file references and the touched docs' required headings.

**Skill validation:** Usage is a scoped CLI/program selection interface; improve
is judgment inside the existing mandate, not a second scheduler. The operation
paths resolve in the CLI manifest; role reads resolve through Prompt Manager.
Manual divergence probes tested (1) a green inventory with unknown outcome rows,
(2) local repair while billing is undecided, and (3) a read-only caller observing
a shared-owner defect. The tempting alternate executions—declaring acceptance,
blocking all local work, or mutating the owner—violate explicit decision rules.
No divergent compliant disposition was found in these cases. This is an author
review, not independent-agent behavioral qualification.

**Prose retirement map:** README's absent-skill warning now points to the real
roles. Repeated inventory aggregation lives in the governed program. TESTING.md
owns the corpus/device/acceptance review package; MONETIZATION.md alone owns the
proposed billing policy. PERFORMANCE.md no longer claims no observations exist
or treats a template build-time allowance as a voice budget. The documentation
manifest opens the product README and checks its actual headings. No existing
failure history or approved requirement was removed.

**Validation limitations:** The first run \`20260908-192759-10f87576\`
returned aggregate PASS with the advisory programs phase failed on
\`programs.never_exercised\`. A prior governed execution cleared that finding;
later runs passed both phases. Reported the confusing aggregate presentation as
Scenario QA \`knw-1788895738048372481\`; do not interpret aggregate PASS alone.
Structure Health's CLI rebuild failed on module consistency and fell back to a
stale binary, so its reported docs pass is not current-source validation. Report:
\`knw-1788896021764196204\`. No raw dependency repair was attempted.

**Remaining work:** Adopt the reviewed proposal through the product owner before
changing the published PRD/requirements; nominate physical devices and authorized
recordings; build and pin the proposed \`voice-pilot-en-v1\` corpus; resolve the
shared billing policy decisions; produce current target-aware receipts through
the now-implemented authoritative current-receipt join for the unknown outcome
rows. The board is a working diagnostic instrument, not
a completed autonomous product-acceptance gate. Runtime streaming, soak,
cross-device, BYOK and hosted monetization qualification remain untested by this
setup pass. These limitations do not block authorized local diagnosis and repair.

## Cross-references

### 2026-09-14 — Qualification producer is an in-scope mandate obligation

The evidence board is intentionally read-only, but the active portable-voice
mandate grants the local D1, constrained D2, open-licensed corpus, and simulated
BYOK/owned lanes. Therefore the absence of a target-aware qualification
producer or current receipt is a W3 implementation gap, not an external
blocker. The campaign must build or repair the owner operation, run it, and
join its receipts before returning an assessment. Private recordings, physical
D3-D6 devices, live keys, and paid-provider access remain excluded decisions.
This entry supersedes older wording in this register that describes all
qualification receipts as external work; those entries remain historical.

- [\`PROGRESS.md\`](PROGRESS.md) — lifecycle log (forward-looking)
- [\`SEAMS.md\`](SEAMS.md) — boundary registry (load-bearing for tests)
- [\`TESTING.md\`](TESTING.md) — test patterns
- [\`../guides/troubleshooting.md\`](../guides/troubleshooting.md) — generic-template issues

## Work ladder history

Interpret these dated observations under the current \`scenario-work-ladder\`
skill and [development grant model](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md).
A missing named Swarm goal is not itself a contract defect: explicit user intent,
an authorized plan, and product requirements can supply the target. An unavailable
search and a successful search with no match are different evidence states.
Neither permits a claim that unread requirements or failed gates have passed.

- **2026-08-18, W0 observation:** the named-mention search returned no matching
  Swarm goal. The user-supplied reliability objective and plan supplied the active
  contract, aligned with the PRD's P0 trust and speaker-policy targets. The earlier
  conclusion that a named goal was mandatory is superseded. PipeWire qualification
  concerns the operating-system capture path, not a physical microphone's analog
  front end.
- **2026-08-31, W3 observation:** the search found only archived
  \`hosted-cloud-tier-foundation\` and \`portal-front-door\`; neither contradicted the
  speech-stack work. That pass reported passing focused gates and a terminally
  failed broad suite on repository conformance/infrastructure checks. These are
  historical results, not current certification. Choose subsequent validation by
  actual impact under the [test protocol](../../../../docs/TESTING.md), and retain
  the failed suite when reporting evidence.
- **2026-09-11, W2 observation:** governed read
  \`prog_3a9a98e3-0d66-4d45-a654-625fefd14cc8\` (method
  \`audio-tools-evidence-board-v2\`, target \`portable-voice-v1\`, 21 s) returned all
  15 outcome rows \`pending_telemetry\` and 3 diagnostics readable (2 available
  engines, 1 native-streaming engine, 4 capabilities); acceptance \`unknown\`. The
  board reads; owner-backed receipt joins remain unimplemented. This is an
  inventory read, not voice evidence.

### 2026-09-13 — Findings carried over from archived Swarm items

**Symptom:** Nine Swarm backlog items held diagnosed Audio Tools defects outside
this register. They were archived (status \`dropped\`) because the single mandate
item \`execute/audio-tools-portable-voice-development\` now owns every Audio Tools
outcome. The claims below were last verified between 2026-07-06 and 2026-09-05.
Re-check each against HEAD before acting; some may already be fixed.

**Root cause (per item, as diagnosed then):**
- **Dictation loss at the session deadline** (OT-P0-005; was
  \`fix/stt-stream-session-deadline-loss-proofing\`). \`api/handlers/stt/stream_ws.go:84\`
  set an absolute 5-minute \`context.WithTimeout\`. On expiry the final flush in
  \`vad_segment.go:336-342\` ran on the dead context, so buffered audio was dropped
  and the final text was empty. \`backend_recovery.go:119\` excluded deadline errors
  from recovery. The raw error string reached the UI (\`stream_ws.go:181\`). The
  web-console client discarded retained chunks instead of using
  \`attemptHttpFallback\`. The operator lost minutes of dictation on 2026-07-05.
  Intended fix: a sliding idle deadline, a fresh-context final flush, honest
  error mapping, and client fallback.
- **No incremental commit under continuous speech** (OT-P0-005; was
  \`execute/vad-segment-max-cut-incremental-commit\`). VADSegment committed a segment
  only after 1200 ms of silence (\`vad_segment.go:308\`), with no max-window cut.
  Operator decision 2026-07-06: stay on VADSegment; no work on OverlapAgree.
  Intended fix: a \`VADMaxSegmentMs\` forced cut, following OverlapAgree's
  \`MaxWindowMs\` pattern. Validate its WER impact in Dictation Studio.
- **Speaker verification inert** (OT-P0-003; was
  \`fix/speaker-verification-vad-device-mismatch\`). Every \`/v1/verify\` call returned
  HTTP 400 from 2026-05-28 (13,006 calls). \`resources/speaker-verification/docker/vad.py:254\`
  moved Silero VAD to CUDA but \`trim()\` passed a CPU tensor. The verify error
  meant \`Applied=false\`, which \`session_speaker.go:55\` treated as "never reject",
  so the filter passed everything. New enrollments failed on the same path.
  SepFormer extraction worked. Intended fix: correct the tensor device, rebuild
  the container, and add a verify error-rate signal to \`stt speaker-status\`.
- **Quality gates bypassed on the native-streaming path** (OT-P0-002/003; was
  \`execute/kyutai-path-speaker-gate\`). The egress \`SpeakerStage\` short-circuited on
  segments without PCM (\`internal/stt/egress/speaker_stage.go\`), so kyutai skipped
  the speaker gate, the hallucination filter and confidence gating. Degraded-resource
  behaviour (\`fallback_without_verification:false\`, \`mode:filter\`) was ambiguous.
  Related plan: \`speaker-verification-quality-overhaul-tiers-1-2\`.
- **Speaker calibration, rejection UX and enrollment reuse** (OT-P0-003; was
  \`execute/speaker-verification-polish-enrollment-reuse\`). The runtime threshold was
  0.4 against a server default of 0.5 (hybrid aggregation), and was never
  calibrated. Dictation Studio clips are already 16 kHz mono s16le PCM. The
  \`speaker enroll --file\` CLI needs only a WAV header wrap.
- **No engine dimension in Dictation Studio** (OT-P1-001; was
  \`execute/dictation-studio-engine-dimension\`). \`EvalStrategy\` (\`eval.proto:24-35\`)
  had no engine selector, and the provider was hard-wired at \`bootstrap.go:213\`.
  Kyutai serves one stream at a time (\`capabilities.yaml:95\`), and its capacity
  degrade is a full unload.
- **Broad test-suite failures** (was \`chore/audio-tools-suite-green-up\`). The
  2026-07-05 full run failed on api 18, unit 62 and security 382 findings, mostly
  pre-template conformance debt but including real security findings.
- **Soak metrics not adopted** (was \`fix/test-genie-audio-tools-metrics-adoption-regression\`).
  On 2026-09-05 Test Genie health reported \`audio-tools/soak\` \`metricsAdopted=false\`
  (\`metrics_missing\`, \`concurrency_declaration_missing\`).
- **Superseded plan still reading as open** (was \`chore/audio-plan-records-hygiene\`).
  Plan \`audio-tools-stt-experiment-lab-remediation-extensibility\` is superseded:
  its safety-winner and speaker-wiring phases shipped in commit \`e2384cfb83\`.

**Workaround:** None recorded beyond each item's notes.

**Real fix:** The mandate's improvement loop, prioritized by the setpoint board.
Delete a bullet when its row reads in band or its defect is verified fixed.

**Owner:** \`execute/audio-tools-portable-voice-development\`.

### 2026-09-16 — Browser speech clock and paced-session qualification findings

The browser stream diagnostic measured first partial from the first PCM callback,
which included device warm-up/leading silence and did not match the adopted
speech-to-visible-partial protocol. It now records the first non-trivial PCM
sample separately and anchors the latency clock there. The approved p95 gate is
also enforced by the owner soak harness; a 30-turn D1 Kyutai cohort measured
first/last values of 1291/939 ms and is therefore not acceptance evidence for
the <=1000 ms p95 target.

A 50 ms wire-batch experiment changed one-turn first partial from 1189 ms to
1175 ms, so it was rejected as non-material and the production default remains
100 ms. A 30-turn run using a sparse 20 s fixture observed signal on only 13/30
turns; a continuous speech-loop fixture produced signal and interim text on
30/30 turns. The sparse-fixture result is retained as a harness/source finding,
not a product success or quality receipt.

**Refs:** \`packages/audio-capture-browser/src/streamDiagnostic.ts\`,
\`api/internal/soak/driver.go\`, \`api/internal/soak/driver_test.go\`.

**Owner:** \`execute/audio-tools-portable-voice-development\`.

### 2026-09-14 — Evidence envelope shape hardened

**Symptom:** A malformed \`ListQualificationEvidence\` response could be
interpreted as an empty owner receipt list by the board.

**Root cause:** The program projected a missing repeated \`evidence\` field to
\`[]\`, making transport/schema damage indistinguishable from measured empty
telemetry.

**Resolution:** The board now preserves malformed envelopes as shape failures;
only an explicit empty repeated field is reported as empty telemetry. A focused
regression covers the missing-field negative control.

**Remaining:** The live owner store still has no target-aware
\`audio-tools-acceptance-v1\` receipts, so the 13 required rows remain unknown.

**Owner:** \`execute/audio-tools-portable-voice-development\`.
### 2026-09-16 — First-batch admission overlap removes only a small startup bubble

**Finding:** A fresh D1 Chromium one-turn Kyutai run after the managed resource
reacquire measured speech-to-visible-partial at 1,292 ms with the API adapter
waiting for Kyutai \`ready\` before forwarding the first PCM batch. The adapter
was changed to forward exactly that first bounded batch after the validated
\`start\` frame, while holding later batches until \`ready\`; the focused provider
suite passes and the rebuilt API was verified by its new build identity.

The matched browser rerun measured 1,261 ms (server owner log: first audio
frame 220 ms; first partial 1,425 ms from stream start). This is a 31 ms
improvement, but remains outside the approved speech-to-visible target of
1,000 ms. The remaining delay is in Kyutai's first-token decode path, not the
browser WebSocket handshake. The row remains unresolved until a current
30-attempt paced cohort and the required quality/reliability protocols pass.
### 2026-09-16 — Current 30-turn browser cohort confirms reliability but not latency

Fresh owner-backed run \`browser-soak-1789547720901983992\` used the current
Audio Tools API build \`sha256:59df5d61fbb0718479eac9ce2e7536f87ea29a087cc1e1868d2b666d35fa35d2\`,
Kyutai \`kyutai/stt-1b-en_fr\`, D1 Chromium, realistic realtime pacing, and 30
looped speech turns. All 30 turns observed non-silent signal, continuous
interim text, exact interval coverage, bounded browser/server retention, unique
committed segments, and terminal outcomes. The enforced latency assertion
failed honestly: speech-to-visible first-partial p95 was 1,290 ms (first 1,068
ms, last 1,074 ms), above the approved 1,000 ms target. The latency row remains
out of band; the reliability observations are not a substitute for the
required quality, repetition, or 60-minute protocols.
### 2026-09-16 — Product-path latency decomposition and canonical PCM fast path

Fresh owner-backed browser cohort \`browser-soak-1789551294582746873\` exposed
metadata-only milestones: capture start → first capture 27ms, first voiced
sample 48ms, first frame sent 125ms, but first visible partial 1330ms. Repeated
same-framing API probes were 726–737ms, establishing that browser capture was
not the bottleneck and that the product segmenter/stream path added roughly
540ms.

The segmenter was routing already-declared \`pcm_s16le\` Kyutai frames through an
identity decoder relay even though the provider's \`requires.pcm16kMono\`
requirement was already satisfied. The canonical \`passthrough\` path now skips
that redundant relay; non-canonical formats and VAD/overlap strategies retain
normalization. Focused segmenter/sttchain tests pass. The first post-deploy
qualification attempt after this change was invalid for the Kyutai cell because
the browser was seated on \`whisper-local\`; the engine-resolution state must be
re-established before measuring the fast path. No latency or quality acceptance
claim is made from that run.
### 2026-09-16 — First real-browser voice start serialized AudioWorklet loading

**Symptom:** The first real-browser voice turn could spend the full
\`AudioWorklet.addModule()\` load after microphone acquisition and journal
restoration, extending the button-to-first-PCM interval even though the stream
transport was already preconnected.

**Fix:** \`useVoiceCore\` now preloads the shared capture module from the explicit
voice gesture. \`createCanonicalPcmCapture\` reuses the same per-context promise,
so module loading overlaps device setup without holding an idle microphone or
creating an audio graph before user intent. The webdriver fake-media path keeps
its immediate ScriptProcessor fallback.

**Evidence:** \`packages/audio-capture-browser/src/pcmCapture.test.ts\` covers the
preload seam; the browser package build and focused capture/startup suite pass.
### 2026-09-16 — Whisper browser soak routed through batch VAD but first text still waited for the silence boundary

The corrected-port D1 browser cohort proved that capture, WebSocket delivery,
Whisper selection, and VAD segmentation were healthy, but its first visible
partial p95 was 2.934 seconds. The live log showed the first Whisper request
arriving only at the 3-second persisted VAD silence threshold; the existing
preview lane was too late to meet the interactive target on short browser
turns. The production defaults now preview every 500 ms over a bounded 3-second
tail, preserving single-flight/load and retention safeguards. A follow-up
probe confirmed the setting is live, but the first effective server batch is
still about 3 seconds in the BAS host-device path, so the latency target is
not yet met. The failed 30-turn receipt is retained as the before measurement;
the next intervention must measure browser-to-server batch cadence before
another acceptance cohort is attempted.

### 2026-09-16 — Preview-only VAD-filter bypass did not bring browser latency in band

Fresh owner-backed cohort \`browser-soak-1789569280493024710\` ran after the
preview-only VAD-filter change on API build
\`sha256:912d3e233f8dba7937cf683da862fd8b68d61c3e0d8736d512dc89894353fe96\`.
All capture, transport, retention, interim-text, terminal, and quality checks
passed across 30 realtime turns. The enforced latency assertion still failed:
speech-to-visible first-partial p95 was 1,500 ms (target <=1,000 ms; first
929 ms, last 899 ms). The intervention is rejected for the latency target;
the next investigation must explain the periodic product-path scheduling
spikes rather than treating preview filtering as the bottleneck.

### 2026-09-16 — Startup-only extraction window did not bring browser latency in band

Fresh owner-backed cohort \`browser-soak-1789571306035045389\` on the startup
short-window build measured speech-to-visible first-partial p95 at 2,151 ms
(target <=1,000 ms). Capture, transport, retention, terminal, and interim
visibility assertions passed, but the quality assertion had two outlier turns.
Runtime logs showed that the 400 ms extracted window was not previewed until
the 500 ms byte threshold was crossed by the following extraction batch. The
intervention is rejected; the next controlled change is a 250 ms preview
cadence.

### 2026-09-16 — 250 ms preview cadence has no valid qualification yet

The API build for the 250 ms preview-cadence intervention passed focused STT
tests and reported source identity
\`53d3cd278c397b19febd6ecfa40e2d1777be75689706cd995f4b63e8c32b683e\`. The first
owner attempt was blocked by BAS's full 10-session pool; after lifecycle
recovery, cohort \`browser-soak-1789571977798893373\` lost its page before the
processed marker, so latency and quality were not measured. A subsequent
attempt hung before issuing an API request while BAS had no active recording.
This intervention remains unqualified and must not be treated as a pass or
failure until a complete owner-backed cohort is obtained.

### 2026-09-16 — 500 ms initial extraction reduced but did not close latency gap

Complete cohort \`browser-soak-1789572623397995018\` passed capture, transport,
retention, provider identity, interim visibility, terminal, and WER checks over
30 turns. First-visible-partial p95 improved to 1,199 ms (target <=1,000 ms;
first 539 ms, last 447 ms), but periodic Whisper responses of about 400 ms and
occasional empty first previews kept the latency assertion out of band. The
next controlled change is a host-qualified Whisper thread-count increase from
12 to 16.

### 2026-09-16 — Native Whisper 16-thread tuning worsened browser latency

Owner-backed cohort \`browser-soak-1789573069364112812\` completed all 30 turns
with Whisper running at \`-t 16\`. Functional, retention, provider, interim, and
WER checks passed, but first-visible-partial p95 was 1,268 ms, worse than the
12-thread baseline of 1,199 ms. The resource is restored to \`-t 12\`; this
experiment is rejected.

### 2026-09-16 — Whisper 16-thread tuning and preview timestamp omission were rejected

The live resource is restored to its measured 12-thread baseline after
\`browser-soak-1789573069364112812\` worsened first-visible p95 to 1,268 ms.
The follow-up preview-only timestamp omission was also rejected by complete
cohort \`browser-soak-1789573650103329679\`: p95 was 1,202 ms versus the
1,199 ms baseline. All requests retain the existing timestamp contract.

### 2026-09-16 — 250 ms cadence still has an empty-first-preview outlier

Clean diagnostic cohort \`browser-soak-1789572403274586838\` passed product-path
capture, transport, retention, provider identity, interim visibility, terminal,
and WER checks across three turns. First-visible-partial p95 was 1,287 ms:
two turns were 853/914 ms, while an empty first preview produced the outlier.
This is diagnostic evidence only. The next controlled change increases the
initial extraction window to 500 ms to reduce that empty-preview path.

The complete owner-backed timestamp experiment
\`browser-soak-1789573650103329679\` measured first-visible p95 1,202 ms,
effectively unchanged from the 1,199 ms baseline. The experiment is rejected
and the existing preview timestamp contract is restored.

### 2026-09-16 — Provider identity was emitted after the processed acknowledgement

The Kyutai product-path diagnostic selected the correct engine and reached a
terminal server status with \`provider=kyutai\` and
\`model=kyutai/stt-1b-en_fr\`, but the browser diagnostic sampled identity before
that status because Audio Tools queued \`provider_identity\` after the processed
acknowledgement. This made the provider-parity assertion observe blank/old
identity. The wire order is being repaired so identity precedes the
acknowledgement; no parity claim is promoted from the affected run.

The browser recovery path also overwrote the primary Kyutai identity with the
Whisper provider used for its bounded final-text fallback. The diagnostic
recorder must preserve the first streaming identity; fallback execution is a
separate recovery event and cannot relabel the qualified provider cell.

### 2026-09-16 — Kyutai identity repair passed; steady-state latency is in band

After the browser recovery fix and server identity-ordering fix were deployed,
\`browser-soak-1789574964544802940\` observed
\`provider=kyutai\`, \`model=kyutai/stt-1b-en_fr\` for all three turns. The short
diagnostic still measured a cold-start first-partial p95 of 1,831 ms, so it is
not qualification evidence. The complete 30-turn cohort
\`browser-soak-1789574998894105269\` passed provider identity, interim visibility,
WER, retention, memory, terminal, and wire-rate assertions; first-partial p95
was 718 ms (first 802 ms, last 706 ms), inside the <=1,000 ms target. This
separates a first-provider-activation bubble from the steady-state streaming
path. The remaining qualification gap is the required target-aware 60-minute
evidence, including committed-text lag.

The follow-up instrumentation repair is scoped to the measurement seam: when a
route emits no \`segment-final\` event, its terminal final supplies the
stop-to-final-tail observation; when segment commits exist, terminal delivery
does not overwrite their committed-lag measurement. The shared package
regression covers this non-segmenting case before the next qualification run.

### 2026-09-16 — Long-form soak expired at the server timeout boundary

The target-aware 30-turn realtime cohort
\`browser-soak-1789575330195205848\` ran for approximately 60 minutes but its
single BAS execution request ended at the Audio Tools server's 65-minute
context deadline. The resulting artifact contains only \`not_measured\`
assertions and is rejected; it is not audio failure evidence. The producer
timeout now allows ten minutes of post-capture headroom (70 minutes for this
cohort), while accelerated runs retain their existing budget. Focused timeout
regressions cover the revised contract. A fresh cohort is required.
## 2026-09-16 — Processed readiness remained hidden behind stale diagnostic state

The fresh realtime cohort \`browser-soak-1789579561160178070\` captured and sent
1,936,720 samples through Kyutai, observed a 720 ms first partial and 117 ms
terminal-tail lag, and received processed acknowledgement through sequence
1217. Its browser step nevertheless timed out waiting for
\`dictation-turn-processed-ready\`, leaving all 16 assertions \`not_measured\`.

The exported diagnostic identified the cause: \`DictationRecorder\` changed its
local React state to \`recording\`, but \`PcmVoiceStreamProvider\` never changed
the shared diagnostic state from \`preparing\`. The readiness selector correctly
excluded that stale state, so this was a product/harness observability defect,
not evidence of slow backend transcription. The provider now marks the turn
\`recording\` after capture setup, and a regression test covers the transition.

## 2026-09-16 — Long-form cohort exposed terminal-lag and quality-source gaps

Fresh target-aware cohort \`browser-soak-1789582387294769051\` completed all 30
realtime turns in 3,879,005 ms. Thirteen assertions passed, while the approved
latency p95 was 1,063 ms (target <=1,000 ms), committed-text lag was
\`not_measured\`, and WER failed on every turn. The run is rejected for
qualification and no threshold is changed.

The evidence shows two measurement-seam defects: Kyutai's terminal diagnostic
reported a 117 ms committed tail, but periodic snapshots carried no committed
lag and the harness discarded the terminal fallback; and quality was scored
from the UI's latest final field even though the Audio-owned server ledger
contains the complete committed segments for the turn. The soak driver now
uses terminal committed lag only when periodic lag is absent, and grades from
the owner ledger when committed segments exist. Regression tests cover both
paths. A fresh cohort is required after rebuild; the 1,063 ms latency result
remains an open product performance failure.
## 2026-09-16 — Long turns queued every processed cursor ahead of terminal delivery

The fresh 30-turn realtime cohort \`browser-soak-1789586966795218198\` captured
full processed coverage and low-level first-partial/committed-lag telemetry, but
the browser timed out waiting for \`dictation-turn-processed-ready\`. The final
diagnostic still showed \`doneSent=true\`, \`processedSequence=1218\`, and
\`final_pending\`; the API log recorded a graceful completed stream. A 120-second
turn therefore produced roughly 1,200 processed acknowledgement frames, all
queued as durable egress messages ahead of terminal delivery. This is a real
delivery/backpressure defect, not a threshold issue.

The intervention coalesces processed acknowledgements to their newest monotonic
cursor, promotes that cursor immediately before a committed/terminal message,
and keeps provider identity plus terminal messages lossless. The new writer
regression test exercises 1,200 acknowledgements and asserts a two-message
coverage/terminal queue. The long-form qualification must be rerun after the
owner deploys this source hash; the failed cohort remains rejected evidence.
## 2026-09-16 — Kyutai startup warmup was too short for first-partial latency

After the egress-queue fix, a fresh 120-second realtime turn completed, but its
first visible partial was 3.34 s. A warm 4-second follow-up still measured
1.92 s, while capture-to-wire was ~200 ms and committed-text lag was ~110 ms.
The resource was CUDA-backed and healthy, so this isolated the delay to lazy
decoder/kernel initialization rather than browser capture or WebSocket
admission.

The resource default warmup increased from 8 to 32 silent model-rate frames,
and the managed-service environment was updated to pass the same value. After
restarting the owner, the identical 4-second probe measured first-partial p95
834 ms and passed all other realtime assertions. This remains a falsifiable
short-probe result; the required 60-minute cohort is the next acceptance
evidence.
## 2026-09-16 — Long-form cohort hit Kyutai admission/reconnect failure after foreign stream

The fresh owner-backed cohort \`browser-soak-1789591585701729803\` was rejected.
After an unrelated 30-second stream, the cohort's logical session repeatedly
reconnected with zero-duration provider closes. The resource then timed out
queued admissions while the browser retained audio; the final diagnostic showed
\`processedSequence=304\` versus \`capturedSequence=1431\`, \`backend_degraded\`,
\`reconnecting\`, and bounded \`http_fallback\` recovery. The harness correctly
marked 16 assertions failed or unmeasured; this receipt is not acceptance
evidence.

The resource log also records an admission-timeout client-disconnect traceback
while attempting to send the timeout frame. This points to a cleanup/admission
boundary defect or owner/resource routing contention, not a latency-band issue.
The 32-frame warmup short probe remains valid, but the 30-turn cohort must be
rerun after the admission/reconnect path is repaired and independently tested.

## 2026-09-16 — Kyutai delayed-stream setting is the remaining warm start candidate

Two fresh five-turn owner-backed cohorts (\`browser-soak-1789600867470099420\`
and \`browser-soak-1789600941182579767\`) passed transport, terminal, retention,
repeated-session, interim-visibility, and WER checks. Their first visible
partials were 705–847 ms. Browser preconnect and first-audio timing are already
instrumented; the live Kyutai resource still applies its model-configured
\`audio_delay_seconds=0.5\`, so the remaining wait is in the delayed-stream
decoder rather than the voice button or WebSocket handshake.

Falsifiable intervention: make the delay an explicitly bounded resource
setting and probe 250 ms. Accept the change only if speech-to-visible-partial
improves materially while the clean-corpus WER, terminal tail, committed lag,
and recovery checks remain in band. A quality or recovery regression rejects
the setting and leaves the current 500 ms value in force.

Result: rejected. The owner-backed five-turn probe
\`browser-soak-1789601834523620125\` ran with the verified 250 ms setting and
passed its transport, recovery, interim-text, lag, and WER checks, but first
partials were still 706–811 ms. This is not a material improvement over the
known 500 ms cohorts (705–847 ms), so the managed override was removed. The
remaining delay is in first-token scheduling/decoder execution, not delayed-tail
draining.

The proposed one-shot 50 ms initial-batch follow-up was not retained. After
the package deployment, the owner-backed harness observed a zero-partial
read-error turn and stalled waiting for the cohort; no resource stream was
reached, so this is not latency evidence. The production 100 ms batching path
is restored. The batching experiment remains unqualified until an isolated
browser/runtime deployment can prove the resource connection and full-turn
wire-rate receipt.

The handshake-overlap intervention was rejected. Unit coverage proved the
intended ordering, but live receipt \`browser-soak-1789602882634615702\`
produced repeated zero-duration provider closes, lost interim visibility, and
WER 20–50%; the resource did not receive a stable stream. The original
first-audio-before-dial sequencing is restored. This demonstrates that the
resource connection lifecycle, not just the network handshake, is part of the
latency contract and must be optimized with an admission-aware design.

The acceptance producer also had an evidence-contract defect: one failed WER
assertion suppressed otherwise independent transport and provider-evidence
claims from a completed realtime run. The testing inventory explicitly
separates recognition mistakes from interval loss and recovery. The producer
must therefore gate each claim on its own passed assertions, retaining quality
as a separate negative result.

## 2026-09-16 — Browser qualification persistence used the wrong verdict scope

The acceptance producer now emits transport/provider claims independently of
WER, but the browser-product-path persistence layer still used the complete
conformance verdict for its top-level \`Passed\` field. That would leave the
claim envelope looking correct while the board resolver rejected the evidence
row. The persistence layer now scopes \`Passed\` to the target-aware claims in
\`machine_json\`; transport invariants and negative controls remain mandatory.
Focused handler, acceptance, and conformance tests pass. No existing failed
receipt is retroactively promoted; a fresh owner-backed long-form run is still
required.

## 2026-09-16 — Fresh Kyutai owner run still pays a cold first-stream decoder path

Fresh owner-backed run \`browser-soak-1789603669908940143\` passed capture,
transport, interim visibility, terminal delivery, committed-text lag, provider
identity, and WER. It nevertheless measured first-visible p95 at 2,322 ms;
the API trace shows the first stream at 2,521 ms while the next four streams
settled near 1,003–1,008 ms. First audio arrived at 191 ms, so capture and
WebSocket admission are not the dominant delay. The resource was healthy and
already using its 32-frame startup warmup, which means readiness is still
declaring before the first real-session decoder path is fully paid.

The next falsifiable intervention raises only the managed startup warmup to 64
model frames. Accept it only if a fresh multi-turn owner probe removes the cold
first-stream outlier without changing WER, terminal-tail, recovery, or memory
behavior. Revert it if startup cost grows materially without reducing the
first-visible partial.

Result: accepted for continued qualification. Managed warmup is now 64 frames;
fresh owner run \`browser-soak-1789603834481708139\` passed all five turns with
first-partial p95 799 ms (first 799 ms, last 695 ms), while committed lag,
interim visibility, provider identity, terminal behavior, and WER remained in
band. A 30-turn cohort and real-duration trust runs are still required.

The required 30-turn cohort \`browser-soak-1789603896951446874\` then completed
all 30 turns with first-partial p95 972 ms (first 790 ms, last 684 ms),
committed lag, interim visibility, transport, provider identity, and memory
passing. Turn 22 alone had WER 0.5, so the complete run remains rejected for
quality. This confirms the latency improvement is durable across the adopted
sample count while recognition quality remains an independent obligation.

The first real-duration retry was rejected before capture because BAS returned
\`RESOURCE_LIMIT\` with 10 stale sessions. Its zero-duration artifact is not
evidence. The BAS cleanup endpoint reported those sessions as non-idle and its
direct close route required execution/lease metadata not exposed by the health
surface, so the owned BAS scenario was restarted to clear the bounded test
pool. A clean retry is now running with one active session.

## 2026-09-16 — Long-form receipt producer did not enforce the required three-run trust cohort

The testing contract requires three real 60-minute sessions per claimed local
engine/profile. The target-aware browser producer could emit \`dictation-trust\`
and \`provider-evidence\` from the first qualifying long-form run, which was too
weak for that protocol even though the run itself was valid. The persistence
repair will count only prior current target-aware browser receipts for the same
provider cell and identity tuple, emit no acceptance claims for runs one and
two, and emit claims on run three only when the independent transport assertions
pass. Failed or short runs remain visible history and never count toward the
cohort.

The producer now enforces this cohort in code: only distinct passed target-aware
receipts for the same provider/context tuple count, and the third qualifying
run is the first one allowed to carry the trust/provider claims. Focused soak,
acceptance, and conformance tests pass. The live API must be rebuilt before
subsequent runs can use the repair.

## 2026-09-16 — Preconnected voice-input sockets expired before the user spoke

The browser recorder preconnected its WebSocket after microphone permission, but
the server's inactivity deadline also applied before the first audio frame. A
socket left warm for more than the server timeout closed with a provider-failure
diagnostic; the client retained the dead socket until its longer 30-second
preconnect timer, so the next button press paid a fresh handshake and provider
startup. This was a direct source of the perceived voice-button delay, separate
from Kyutai's measured first-partial latency.

The shared PCM provider now sends a four-second JSON keepalive only during the
preconnect window. It sends no microphone data and stops the timer as soon as
capture starts or the warm window expires. The long-session suite has a
regression test for the keepalive and remains green at 30/30 tests. The server
already ignores unknown text messages while refreshing its read deadline, so no
provider is started by the keepalive.

## 2026-09-17 — Trust-cohort receipt gate could not bootstrap from partial claims

The fresh 30-turn owner run \`browser-soak-1789616823554336388\` completed the
required long-form transport cohort: 30/30 turns had exact interval accounting,
bounded retention, provider identity, continuous interim text, and terminal
delivery; first-partial p95 was 738 ms across 90 observations. Its repeated
fixture still failed WER, so the run was not a quality receipt. The persisted
browser receipt correctly carried only \`interactive-latency\` because the trust
repetition gate was not yet satisfied.

Audit of the producer found a bootstrap defect: the three-run gate counted only
prior receipts that already carried \`dictation-trust\` and \`provider-evidence\`.
Runs one and two were intentionally withheld, so they could never contribute to
run three. The producer now counts distinct passed target-aware browser
receipts whose owned long-form artifact independently passes all transport
assertions, while still requiring the configured three-run cohort before
emitting trust/provider claims. A focused regression covers both the passing
artifact and a transport assertion failure. The live API must be rebuilt before
the next qualifying run uses this repair.

## 2026-09-17 — Device companion receipts contradicted browser acceptance joins

After the trust gate was repaired, the fresh browser receipt carried all three
expected claims, but the setpoint board still rejected the tuple. The device
companion row reused the browser acceptance envelope while its own \`Passed\` flag
was false, so the resolver treated the same target tuple as both passing and
failing. Device evidence is not an acceptance producer.

The persistence path now strips target-aware acceptance fields from device
metadata, and the board resolver ignores device-kind rows when building
acceptance candidates while retaining simulated and corpus producers. The board
now resolves dictation trust, provider evidence, interactive latency, corpus
quality, and the existing policy/adapter rows; provider parity, speaker policy,
mobile recovery, and host portability remain genuinely unmeasured.

## 2026-09-17 — Native Kyutai first-visible-partial remains above the interactive target

Fresh browser-product measurements after the transport warm-path repair show
capture startup and first signal in roughly 20–40 ms, with the first PCM sent
within roughly 55 ms after enabling the host-specific fast first wire batch.
The native Kyutai route still produced its first visible partial at p95=1196 ms
across six realtime turns, above the approved <=1000 ms target. The same route
is about 523 ms from a burst-fed CLI stream, confirming that the remaining gap
is realtime decoder/model behavior rather than microphone permission or UI
rendering. The 20 ms first batch reduced the browser result from the prior
1292 ms p95, but does not close the target. Do not mark interactive latency
complete or claim the model delay is fixed until a fresh realtime cohort is in
band; investigate a lower-latency native engine/model or a quality-preserving
speculative path with paired WER evidence.

The next fresh owner probes did not reproduce the earlier 1-second band: six
realtime turns measured p95=1196 ms and ten turns measured p95=1286 ms. Both
probes still passed capture, retention, terminal delivery, committed-text lag,
and interim visibility, but failed the first-partial assertion (and the small
fixture also failed its independent WER assertion). These failed runs remain
diagnostic only; the latency receipt is not promoted. The speaker-policy
validation completed independently with its fail-closed control passing.

An isolated managed-resource experiment raised \`KYUTAI_STT_WARMUP_FRAMES\` from
64 to 96. The owner remained healthy after startup, but a fresh five-turn
probe measured first-partial p95=1194 ms and did not improve the 64-frame
baseline. The change was reverted; 64 remains the adopted setting. The first
probe briefly reported CPU mode during restart, then returned to healthy CUDA
mode after the resource completed initialization.

## 2026-09-17 — Reconnect replay ordering and warm-button status were product-path blockers

The deterministic browser recovery workflow initially reconnected with a newly
captured sequence (for example sequence 12) before replaying retained sequence 0.
The server correctly rejected that gap and then retained a terminal
receive_failed state. The browser transport now holds post-disconnect sends
behind journal replay; the shared capture package test suite passes 192/192
non-skipped tests.

The Dictation Studio button also reset an already-open permission-warmed socket's
visible status from Streaming transcription connected. back to
Connecting to speech stream… during start(). The recorder now preserves the
connected state when the warm socket is already open. The authored
audio-tools workflow passed after rebuild, including the normal product smoke
path and the injected backend-restart recovery path. The remaining
Experience Manager warning about the VoiceInputButton 3.0.0 story specimen is
catalog/readiness metadata, not a workflow failure.

## 2026-09-17 — Lower Kyutai delayed-stream override rejected

The documented KYUTAI_STT_AUDIO_DELAY_SECONDS seam was tested at 0.25
seconds against a fresh five-turn realtime browser cohort. The owner remained
healthy on CUDA and the independent fixture WER assertion passed, but
speech-to-visible-partial p95 worsened to 1342 ms across 10 observations
(target <=1000 ms). The override was reverted and the adopted managed-owner
setting remains the checkpoint default of 0.5 seconds with 64 warmup frames.
The failed receipt remains retained as diagnostic negative evidence and is not
promoted.

## 2026-09-17 — Kyutai text-path warmup and host wire cadence validated

The audio-tools host now uses a 50 ms steady-state PCM wire cadence after its
20 ms first batch; the shared package remains at 100 ms so its recovery-rate
and journal tests retain their existing guarantees. Focused transport coverage
remains green (192 passed, 1 skipped).

The Kyutai owner previously warmed only silent decoder frames. It now evaluates
eight bounded deterministic low-amplitude non-silent frames before \`/ready\`,
then resets the acoustic and language-model streaming state. This exercises the
text-producing CUDA path without allowing warmup text into a user session.

After restart, a fresh one-turn product probe measured first-visible-partial
latency at 705 ms, and a subsequent five-turn cohort measured p95=706 ms with
interim text visible on every turn. The same cohort's supplied reference text
was intentionally incorrect and produced WER=1.0; that receipt is diagnostic
only and must not promote corpus-quality or provider-parity claims. A correct
held-out reference cohort is still required before the latency improvement can
participate in the shared acceptance join.

The canonical four-second product cohort then used the fixture's actual
reference, \`a quick brown fox jumps.\`. Latency stayed in band at p95=790 ms
across 10 realtime observations and interim text was visible on every turn,
but Kyutai WER remained 0.20–0.30 across five turns against the <=0.10 clean
quality floor. This is retained as valid negative provider-quality evidence;
Kyutai must not be promoted to provider parity until product-path settlement or
model quality is repaired and a fresh paired latency/quality cohort passes.

## 2026-09-17 — Full-duration Kyutai product cohort remains quality-negative

The fresh 60-turn, 60-second-per-turn browser cohort
\`browser-soak-1789645783740086219\` completed for 3,808,618 ms. Capture,
settlement, retention, provider identity, interim visibility, committed lag,
and first-partial stability all passed; first partial was 778 ms at p95, with
the first cold turn at 1,996 ms. Recognition quality did not pass: 60/60 turns
failed the adopted WER <=0.10 floor, measuring 0.1290 on most turns (0.1333 and
0.1355 on three turns), dominated by deletions. The receipt is retained as
negative product-quality evidence and cannot produce provider-parity. The
remaining repair is the Kyutai long-form/product settlement quality path, not
the browser transport or streaming-latency path.

## 2026-09-17 — Fresh browser sessions started the looping fixture at arbitrary offsets

The remaining one-turn variance was not a provider timing regression. The BAS
fake-media source is continuous across page readiness, so a new browser session
could begin in the middle of the looping reference fixture. That changed the
first decoded words and made otherwise identical product probes appear to have
unstable WER and latency. The existing between-turn rewind did not cover the
first turn.

The realtime soak owner now calls its leased \`/audio/restart\` endpoint after
navigation and surface readiness, before the first turn, and continues to rewind
between turns. The unit fixtures model that owner endpoint. After rebuilding,
three fresh Kyutai product runs (\`browser-soak-1789638954401080201\`,
\`browser-soak-1789638972157226608\`, and \`browser-soak-1789638982625879332\`)
passed provider identity, capture/settlement, interim visibility, and WER at
0.10. First-visible-partial latency was 716, 716, and 727 ms; committed-text
lag was 95, 97, and 111 ms. These are fresh one-turn diagnostics and do not
replace the required long-form trust cohort or held-out corpus-quality receipt.

The canonical provider identity is \`kyutai/stt-1b-en_fr\`; supplying the shorter
\`stt-1b-en_fr\` label correctly fails the identity assertion. A Whisper browser
attempt (\`browser-soak-1789639039469447936\`) was retained as a failed negative
control because the batch-only candidate cannot enter the realtime product
path. Provider parity remains pending until the owner emits a target-aware
receipt covering every engine actually offered as stable.

The acceptance producer was also missing that composite decision. It now keeps
provider parity unknown unless one full-duration, 30-attempt product receipt
passes transport, latency, interim visibility, and product quality, while the
same provider cell has independent passing native speaker-policy and held-out
corpus receipts. The producer regression covers both the complete prerequisite
set and a failed corpus prerequisite. This is a measurement repair, not a
threshold change; the next fresh long-form receipt will determine whether the
Kyutai cell can enter the parity join.

The first fresh full-duration product attempt (\`browser-soak-1789641692715615741\`)
passed all transport, provider identity, retention, interim, and latency
assertions (p95 first partial 985 ms across 60 observations), but its repeated
120-second fixture cohort measured WER 0.1148 on most turns. It is retained as
negative product-quality evidence and cannot qualify parity. The browser
quality assertion now uses the adopted <=10% clean floor instead of the older
0.25 diagnostic ceiling; the next long-form attempt uses 60-second turns to
separate fixture-cycle accumulation from the model quality gate.

## 2026-09-17 — Host-device playback and realtime acceptance repaired

Fresh host-device evidence isolated the remaining product-path failures to the
qualification harness boundary. BAS playback previously began during session creation,
so the recorder either missed the fixture prefix or accumulated a leading-silence prefix
while \`getUserMedia\` initialized. Continuous playback also extended through the UI stop
handoff. The owner seam now supports deferred playback, explicit turn-boundary restart,
and explicit playback stop; the audio-tools driver arms the source immediately before
the voice gesture and stops it at the feed boundary. Realtime reference expansion uses
the declared playback window, while transport rate uses owner-reported capture timestamps.

The fresh 30-turn host-device cohort \`browser-soak-1789655221034534437\` passed every
assertion: 30/30 signal and settlement turns, interim text visible on 30/30 checkpoints,
WER <=0.10 on every turn, wire rate 16,037–16,051 samples/s, first-partial p95 983 ms,
stable committed lag, bounded retention, and exact provider identity. The fresh 60-second
product cohort \`browser-soak-1789655439380650035\` also passed with WER 0.0452 (154/155
words), 1,202 processed intervals, and 16,003 samples/s. Earlier negative receipts remain
retained as historical evidence and are not overwritten.

## 2026-09-17 — Long trust cohort exposed overlapping reconnect ownership

The first fresh 30-turn, 120-second-per-turn trust attempt
\`browser-soak-1789655589625550196\` is retained as a failed receipt. Turns 1–23
closed normally, but a later backend interruption allowed a stale browser
WebSocket close callback to schedule a second reconnect while the replacement
socket was still taking ownership. Kyutai consequently observed short split
streams (8 frames) while the browser reported \`sentSequence\` progress but no
\`processedSequence\`; the UI remained in \`recording\` and the processed-ready
wait timed out. This is a real lifecycle failure, not evidence to be hidden by
loosening a timeout or excluding the turn.

The browser capture provider now claims a socket before its asynchronous open
callback, ignores messages and closes from sockets that are no longer the
claimed transport, and clears the open socket before scheduling recovery. A
regression test covers a late close after reconnect ownership moves. The full
capture package suite passes (195 tests, 1 skipped). A new live smoke and the
required long trust cohort are still required before this repair can produce
target-aware acceptance evidence.

## 2026-09-17 — Repaired transport survived 30 turns, but long-form quality and cold-start latency remain negative

The fresh 30-turn, 120-second-per-turn cohort
\`browser-soak-1789659085062259477\` completed all 30 turns and passed interval
accounting, capture/settlement, bounded retention, duplicate protection,
provider identity, wire rate, committed lag, and continuous interim visibility.
It is not an acceptance pass: \`first_partial_latency_stable\` measured p95
1,024 ms against the approved 1,000 ms ceiling (cold first partial 1,110 ms),
and \`word_error_rate_stable\` failed on turns 1–3 (WER 0.6885, 0.9115, and
1.2918) plus transient degradation on turns 22–24 (0.1115, 0.1049, 0.0721).
The remaining turns were mostly 0.01–0.03 WER, so the failure is retained as
evidence of cold-start/model stability rather than relabeled as transport
success. No target threshold was loosened; a fresh quality/latency repair and
cohort are still required before the acceptance join can resolve.

## 2026-09-17 — Warm provider rerun clears cold latency, leaves a localized Kyutai quality dip

After a passing five-turn warm-up (\`browser-soak-1789662853577179789\`, first
partial p95 963 ms), the required 30-turn cohort
\`browser-soak-1789662896043329433\` completed all turns with transport,
settlement, retention, provider identity, latency, interim visibility, and
processed coverage intact. The latency assertion passed. The WER assertion did
not: turns 1–19 and 23–30 stayed at 0.0098–0.0328, while turns 20–22 rose to
0.0918, 0.0951, and 0.1016, with 30–31 committed segments and exact server
processed coverage. The receipt remains negative because the adopted per-turn
WER floor is strict; warm-up is not being substituted for a product fix.

This separates the remaining issue from the prior cold-start result: warming
removes the 1,024 ms p95 latency failure, but Kyutai still has a localized
long-form quality-stability dip under repeated product sessions. Provider
parity and the acceptance join remain unresolved until a fresh cohort meets
the quality floor without exclusions.

## Resolved 2026-11-21 — Transcode WAV responses retained streaming size sentinels

Bug inbox entry \`knw-1793084571623423549\` reported a WAV file whose RIFF and
\`data\` lengths remained \`0xffffffff\` after Audio Tools wrote the response to a
regular file. A one-second synthetic 48 kHz stereo WAV reproduced the bad
geometry through \`audio-tools audio transcode\`: the file was 192,078 bytes, but
Python's WAV reader inferred 1,073,741,823 frames and a duration over six hours.
The report's expectation that the CLI default to 16 kHz mono is stale: current
CLI help defines sample rate and channels as optional, with \`0\` meaning
passthrough.

Resolution: both WAV transcode paths now finalize the RIFF and \`data\` chunk
lengths after ffmpeg captures its complete stdout. Other container formats keep
their existing path. This is applied before duration probing and before the
response returns to any caller.

Verification: the focused \`GOWORK=off go test ./internal/audio ./handlers/audio\`
passes, including a real-ffmpeg fixture that checks exact finite RIFF and data
sizes. The pre-fix live CLI reproduction used a synthetic clip only. The
Test Genie unit phase was canceled while its preflight waited on the shared
proto package lock; no run ID was issued. The bug-inbox record remains open for
its designated investigator to read back the owner fix. This evidence does not
change the CLI's documented passthrough default.

## Resolved 2026-11-20 — Audio response metadata is partly unimplemented

\`TranscodeResponse.duration_seconds\`, \`TrimResponse.duration_seconds\`, and
\`MergeResponse.duration_seconds\` are always zero, and
\`NormalizeResponse.measured_lufs\` echoes the requested target. None of the
proto comments say "0 = unknown", so a client can read the value as a
measurement.

**Cause:** the handlers return transformed bytes without a post-transform
probe, and Normalize does not parse \`loudnorm\`/\`ebur128\` output.

**Resolution:** transcode, trim, and merge now probe their encoded output once
through the shared bounded ffprobe seam; they report zero when ffprobe is
unavailable or cannot determine duration. Normalize uses the second-pass EBU
R128 output measurement instead of echoing the target. The proto now documents
zero as unknown for duration and measured loudness fields. Split chunks report
their measured output duration when available.

**Evidence:** \`go test -race ./handlers/audio ./internal/audio\` covers measured
durations, missing-probe zero behavior, output-based duration, and measured
LUFS distinct from the requested target. \`api/handlers/audio/duration_test.go\`,
\`api/handlers/audio/normalize_test.go\`, and \`api/internal/audio/loudness_test.go\`
are the focused regressions. Re-run the owning proto verifier after the comment
change.

## Work ladder

- Rung: W3 — W0, W1, and W2 pass; product qualification and broad operational targets remain open.
- W0 evidence: The operator-approved maturity mission requires desktop dictation parity, footprint-first declared model selection, advanced voice workflows, a $10 standalone-to-bundle journey, and cross-platform installation. The former PRD did not state those outcomes as operational targets. Business Health's wizard generated \`portable-voice-v2\` targets OT-P0-011 and OT-P1-005 through OT-P1-008. Existing earned PRD checkboxes and requirement evidence were preserved. The new desktop requirement uses ATD-P0-020 because ATD-P0-011 already covers OT-P0-006; requirement suffixes are independent of OT IDs.
- W1 gate: \`business-health validate scenario audio-tools\` and \`vrooli scenario requirements validate audio-tools\` pass without findings after \`test-genie requirements sync --dir scenarios/audio-tools --scenario audio-tools --command 'Test Genie unit run 20261122-010826-c409c4d7'\` refreshed evidence.
- W2 gate: \`business-health matrix show audio-tools --format summary\` shows all 29 requirements across 20 operational targets. \`vrooli scenario requirements report audio-tools\` shows 2 complete, 7 in progress, 20 planned, and zero unproven claims. Test Genie business run \`20261121-003833-16b52ad4\` passed at L3; latest W1/W2 validation has no stale-evidence warning.
- W3 evidence: Test Genie unit run \`20261122-010826-c409c4d7\` passed at L2 in 70 seconds (61 observations). The scorecard retains 58 low-coverage advisories, and the completeness projection remains 2/29 requirements and 2/20 targets complete. This turn also passed race tests for \`internal/stt/session\` and \`internal/soak\`, CLI validation tests, and full Test Genie unit execution. Metadata-only session snapshots now omit audio/transcript payloads; an explicit trusted conformance snapshot preserves internal soak grading. Portability receipts require D1 CPU Whisper, D1 Kyutai accelerator identity/model, and D2 Sherpa identity. Speaker-policy orchestration tests found and fixed nil-response panic and restore-before-evidence defects. A supported \`vrooli scenario restart audio-tools\` rebuilt from source and returned healthy build identity \`sha256:27fe0ee3ac83a27283df4bc5d5ece71afc27076d3a238458d16f7308891c209d\`. Live product TTS synthesized valid 24 kHz mono WAV through Kale Nano in 927 ms. A structured product summary confirmed \`qwen3:1.7b\`, preserved six facts, and took 21.256 s; a post-request Ollama snapshot measured its resident size at 2.24 GB. The attached Sherpa Zipformer streamed the 3.125 s clip in 1,558 ms but recognized \`route\` as \`root\`; this smoke proves route execution only and is not quality qualification. Sherpa remains attach-only and unmanaged.
- Remaining: No current comprehensive Audio Tools certification; desktop feature parity/usability, cloning rights and held-out qualification, TTS quality/release qualification, long-form timing references, cross-platform distribution, production duplex transport and durable spend controls, and standalone purchase/upgrade evidence remain open. Sherpa managed acquisition still lacks a reproducible matching signed release artifact: retained local stages do not reproduce the uncommitted \`c8127175…\` pin. Research Desk commission authorization remains pending.
- Measured: 2026-11-22.

## Resolved 2026-11-21 — Package leak checks reported the shared SQL opener

Bug-inbox entry \`knw-1794811974878872342\` reported \`goleak\` failures in
\`internal/ai/sttchain\` and \`internal/stt/segmenter\` for
\`database/sql.(*DB).connectionOpener\`. The opener belongs to
\`api-core/database\`'s package-initialized \`foreignPinPool\`, not a test-local DB
in either package. The owner tests now treat that known dependency goroutine as
process-lifetime (\`IgnoreCurrent\` for the sttchain baseline and an explicit
connection-opener exclusion for segmenter) while retaining their package leak
checks.

Verification: \`GOWORK=off go test ./internal/ai/sttchain ./internal/stt/segmenter
-count=1\` passes. Audio Tools Test Genie run \`20261121-090845-3ffbe5ee\` also
reported its \`unit\` phase passed after the owner changes. The scenario-qa source
record remains open for its paused designated investigator to read back; no
source closure or team trigger was made by PT2.

## Resolved 2026-11-21 — Shared PCM long-session recovery assertions

Bug-inbox entry \`knw-1794811952676210667\` reported seven failures in the
shared browser package's long-session recovery tests: absent recovery callbacks,
terminal cleanup, \`buffered_recovery\`, and sent-cursor updates. The current
package tests now cover these paths directly, including retained-audio recovery
after socket close/empty final, journal cleanup after a committed terminal,
recovery-status publication, sent cursors after wire admission, and buffered
frames before socket open.

Verification: \`pnpm exec vitest run src/longSession.test.ts --reporter=dot\`
passes 31/31; the current package build passes, and its complete test suite
passes 237/238 (one skipped). The package has other-owner uncommitted changes;
PT2 did not modify it. The scenario-qa source record remains open for its paused
designated investigator to read back. No source closure or package restart was
performed by PT2.

## Resolved 2026-11-21 — Shared voice boundary guard rejected its local transport adapter

Test Genie run \`20261121-131712-4e84d4c7\` failed the UI coverage command with
one failing test: \`sharedImportGuard.test.ts\` classified
\`ui/src/audio-integration/registerVoiceTransport.ts\` as an unmarked fork. The
file is an intentional scenario-local adapter: it delegates to the shared
browser package and binds transport operations to Audio Tools RPC calls and
retry policy. The adapter now carries the \`HOST DIFFERENCE:\` marker required by
\`ui/src/audio-integration/README.md\`; runtime behavior is unchanged.

Verification: the focused shared-boundary suite passes 6/6; the related voice
API/configuration/capture/recorder selections pass 115/115; Audio Tools Test
Genie unit run \`20261121-132137-4910bff2\` passes at L2. The preceding failed
run's finding report was 1,132,461 B (one execution error plus 61 coverage
warnings); the passing report is 1,083,581 B (zero execution errors, 61
warnings). A direct \`pnpm run test:coverage\` reproduction before the marker
change failed only \`sharedImportGuard.test.ts\` (1/133 files, 1/1,278 tests).

Retained queue: current Test Genie still reports 61 LOW_COVERAGE advisories:
25 API/Go paths, 26 CLI paths, and 10 UI paths. Route those to the Audio Tools
API, CLI, and UI owners respectively; next step is to test the lowest-covered
behavior at each owner rather than lower policy floors. A comparison with the
11:40Z failed run shows 11 file advisories and one workspace metric no longer
reported, but those changes predate this disposition in a concurrently dirty
tree and have no established cause; they are not counted as PT2 resolutions.
No source or bug-inbox records were closed.


## Owner-verified Test Genie behavior batch — 2026-11-21

Fresh unit run \`20261121-132137-4910bff2\` reported 61 LOW_COVERAGE advisories (25 API/Go, 26 CLI, 10 UI); this batch makes no progress claim for the earlier unexplained 72→61 change. The four clean owner paths selected were API \`handlers/provider_lifecycle/module.go\`, CLI \`cli/domains/soak/handlers.go\`, UI \`ui/src/features/models/components/ModelsTable.tsx\`, and UI \`ui/src/features/admin/components/CopyableCommand.tsx\`.

The API module test mounts the generated Connect service and reads the local-provider registry over HTTP. The CLI now treats a run with zero assertions as unqualified; failed and unmeasured outcomes continue to fail. The ModelsTable test verifies accessible single-row expansion. CopyableCommand previously announced “Copied” when clipboard access was missing or \`writeText\` rejected; it now announces success only after the write resolves, with success/absent/rejected cases tested. Focused owner checks pass: API \`GOWORK=off go test ./handlers/provider_lifecycle -run TestModuleMountsConnectServiceForLocalProviderRead -count=1\`; CLI \`GOWORK=off go test ./domains/soak -count=1\`; UI \`pnpm exec vitest run src/features/models/components/ModelsTable.test.tsx src/features/admin/components/CopyableCommand.test.tsx --reporter=dot\` (4/4).

Final Test Genie unit \`20261121-135425-b5b8ea4f\` passes L2 in 66 s. Findings artifact \`artifact_2f6a70f05f4da5d4e92045fac39e65e2\`, 1,079,668 B, SHA-256 \`c2f7d60ffe0a68a661ca38abe741c131860f0ae7e78af329bc72b42ec72532c0\`. The after inventory is 58 LOW_COVERAGE warnings (25 API/Go, 26 CLI, 7 UI) plus one informational \`COVERAGE_COMMITMENT_BEHIND\`. All three targeted file warnings (\`provider_lifecycle/module.go\`, \`ModelsTable.tsx\`, and \`CopyableCommand.tsx\`) no longer appear. The API count is net unchanged because \`internal/stt/speaker/adapter.go\` appears in the after report; the UI aggregate changed from warning to informational. Those replacement/category changes are not attributed to this batch. The soak handler remains open at 18.5% coverage.

Sample audit of the three absent file findings matched each exact path to its added behavior test and passing owner suite. No canonical source/QA pointers, PROBLEMS issues, or bug-inbox records were closed. PROBLEMS measured 192,118→194252 B before this evidence-note rewrite; top-level issue bullets stayed 65, and no age-only closure occurred. The note grows the file further; final bytes are recorded in the PT2 checkpoint. No actual issue count or backlog-age improvement is claimed.

## Resolved 2026-11-21 — Persisted entity measures and migration staging

\`measures-health validate scenario audio-tools --probe\` passes at L3 with zero
findings. Six real persisted domains have typed, full-tier count measures backed
by owner repositories/stores: BYOK credential capabilities, corpus clips,
experiment runs, experiments, speaker profiles, and wakeword templates. The
measures-go registry and six CLI commands share the same aggregate semantics;
counts use UTC \`[from,to)\` windows and expose no credential secret, embedding,
phrase, identity, or audio payload. No new persistence was added.

The former \`byok_credentials_duplex\` warning came from SQLite's temporary
rebuild table, not an independently persisted entity. The migration now uses a
transaction-local TEMP table, copies encrypted rows back into canonical
\`byok_credentials\`, then drops staging. \`go test ./internal/database\` covers
row preservation, idempotence, and rollback; measures-health no longer reports
the staging table.

Follow-up: \`duplex_accounting_operations\` is a real retained correlation
journal, so it now has an aggregate-only \`CountDuplexAccountingOperations\`
measure using the same UTC half-open creation window as the other persisted
entity measures. The API, proto, CLI and command manifest are bound end to end;
the count exposes no operation, account, provider or receipt details and does
not read LPBS balances. Focused API/duplex and CLI measure tests, proto owner
verification and \`measures-health validate scenario audio-tools\` pass with no
findings. The repository package baseline remains red in six other dirty
modules; see the 2026-11-21 checkpoint in \`PROGRESS.md\`.

## Resolved 2026-11-21 — Buffered STT fallback cancellation terminal event

Cancellation during buffered provider fallback previously emitted an error and
closed without the promised terminal \`Done\`. Cancellation now takes priority
over queued audio and emits \`Error\` followed by terminal \`Done\` with
\`terminal_reason=canceled\`, so downstream stream consumers can close the turn
consistently. The regression reproduces the old \`[error]\` event sequence and
the focused stream, package, and race checks pass. This is a local stream
contract fix; it does not qualify provider parity or long-session recovery.

## Resolved 2026-11-23 — Duplex consent verifier redirects

Gateway consent verification used the supplied HTTP client's default redirect
policy. A 307/308 response could replay the request body, which contains an
opaque consent receipt, to a different host. Both the fixed-URL and
service-discovery verifier constructors now copy the supplied client and
disable redirect following; the caller's shared client is unchanged. Fake
servers verify fail-closed behavior and that neither constructor contacts a
redirect target. \`GOWORK=off go test -race -count=1 ./internal/duplex\` passes.
This protects the consent authority boundary; it does not qualify or enable
GPT-Live.

## Resolved 2026-11-23 — Managed Ollama residency policy reconciliation

Managed-service start and restart previously resumed the private activity edge
without reconciling resident-model TTLs against current Ollama policy. Models
could therefore remain resident beyond their configured policy until another
request arrived. Readiness now reconciles loaded models after health and
placement checks and before resume. The reconciler uses each loaded model's
capability to select the appropriate load-only endpoint and never loads models
that are absent. If the edge is busy, reconciliation skips instead of draining
in-flight work or deadlocking a request that is waking the backend. Focused
race tests pass for the edge and runtime-profile code; the Ollama CLI tests
pass. This is lifecycle policy behavior, not a measured memory reduction on a
live host. The broader \`internal/resources\` package test remains unverified
because the dirty shared tree contains unrelated undefined health-check names
in \`managed_update_admission_test.go\`.

## Resolved 2026-11-23 — Acceptance producers omitted provider-cell device identity

The setpoint board requires the receipt's \`provider_cell.device_profile\` to
match its top-level \`device_profile\`, as well as matching persisted engine,
model, and strategy. The simulated and composite producers omitted the device
field, so their otherwise valid target-aware claims were invisible. The Python
board test helper had concealed this by injecting the missing field into every
synthetic receipt. The simulation, composite join, speaker-policy, and
host-portability envelopes now persist the provider-cell device identity; the
test harness no longer changes input receipts. The focused producer tests and
all 37 board tests pass. A fresh lab simulation receipt now resolves exactly
the five rows it exercised. All other required rows remain unqualified, and
overall acceptance remains unknown.

## Resolved 2026-11-23 — Future processed cursor could erase replay audio

\`TurnJournal.acknowledgeProcessed\` trusted every non-negative server cursor and
removed all retained chunks at or below it. A future cursor could therefore
delete captured audio that had never been assigned to or accepted by the
server. The journal now rejects negative and future cursors before persistence
or compaction. Regressions prove that an invalid cursor preserves the retained
chunk and that a valid inclusive acknowledgement still compacts it. The shared
capture package passes all 257 tests (one skipped) and its TypeScript build.
The check protects replay accounting; paced browser SLO qualification remains
open.

## Resolved 2026-11-23 — Retained recovery could silently change STT destination

Shared browser recovery sent retained audio through the ordinary tiered STT
endpoint without carrying the streaming route. A local stream failure could
therefore be retried through BYOK or an owned provider. Recovery now requires a
known local tier and exact model ID, carries that ID through Audio Tools,
Web Console, Swarm Manager, and Portal unary adapters, and fails before egress
when the route is remote, unknown, or unsupported. Explicit model requests
bypass default credential lookup/attachment and the server's explicit local
route remains fail-closed if that model is unavailable. A refusal leaves the
journal intact. Focused shared, consumer, and server regressions pass; browser
mic/composer qualification and the wider feature matrix remain open.

## Resolved 2026-11-23 — Moderate summary prompt rewarded dropping key facts

The moderate prompt told the model to keep only the single most important
conclusion and compress to 35% of source length. A live Audio Tools request
using the compact Qwen3 1.7B role consequently omitted the pilot size and named
owners from a short operational brief. A same-input quality-role control
(Qwen3.5 4B) invented an invalid date, showing that a larger fallback alone is
not a correctness guarantee. The prompt now explicitly protects names and
responsibilities, dates, quantities, approvals, deadlines, risks, and
contingencies, forbids invention, and removes repetition first. A direct
resource-role probe preserved the controlled brief's facts within the then-
current 90-token floor, with a complete stop; a 120-token follow-up produced
the same output. The moderate short-input token floor is now 120 to provide
generation headroom.

**2026-11-23 refresh:** A separate 397-word synthetic report exposed a longer-
input cap problem. With the fact-preserving prompt and a 210-token ceiling,
Qwen3 1.7B returned \`done_reason=length\` mid-sentence. At the implemented
341-token ceiling it stopped naturally after 259 tokens and retained the
report's named owners, key dates and amounts, grant, staffing, refrigeration
range, training date, and inspection contingency. A same-input, same-cap
control with the previous single-conclusion prompt returned 144 tokens and
omitted the named owners, generator deadline, and grant details. The moderate
output budget now scales to 55% of estimated input tokens, with a 120-token
floor and 2,048-token ceiling; light summaries use 65% of estimated input
tokens with the same ceiling. This was a direct local
model probe against a synthetic source, not an Audio Tools product request or
held-out quality receipt. Product rebuild/restart, the original longer caller
sample, quality-tier behavior, and human review remain unverified.

**2026-11-23 refresh:** The already-running Sherpa attachment still serves
streaming STT, but read-only TTS probes returned HTTP 503 from
\`/healthz/tts\` (\`tts_model_loaded:false\`) and \`/v1/audio/voices\` (TTS model
unavailable). The managed resource status remains \`unsupported_platform\` on
Linux and \`Installed=no\`, \`Running=no\`. Do not claim a current Sherpa/Kokoro
model-backed TTS route from contract tests; acquire the signed managed runtime
or an authorized running attachment before attempting a live product receipt.

## 2026-11-23 — Browser soak admission requires an Agent Manager run

Test Genie run \`20261123-082346-ccbab5b7\` failed before starting the browser
provider. Its \`SOAK_CONFIGURATION_MISSING\` finding names three required
request-scoped receipts: one Agent Manager identity token, one BAS browser
grant receipt, and one Test Genie lease receipt. The persisted phase result
and findings report are artifacts \`artifact_81eab706700791dd09da6e09a30f2902\`
and \`artifact_8cbd6f7782fe5afa7e0678c25ed3c8d6\`; no browser action or
conformance artifact exists. A fresh \`agent-manager run identity --json\`
returned \`identity_refresh_unavailable\`. The browser-agent-access skill
requires an Agent Manager-owned run that injects the identity and grant; these
receipts must not be fabricated or passed manually. This direct session has no
Agent Manager/BAS integration to create that run context. Browser qualification
remains unmeasured; use the supported Agent Manager run path when available.

## 2026-11-23 — Soak CLI owner coverage and input parsing

The soak CLI owner now has request-path tests for owner receipts, API
preflight, flags, evidence export, and measured outcomes. Package coverage rose
from 17.3% to 89.1%. These tests found that \`positiveInt\` rejected valid input
with surrounding whitespace; the parser now trims before conversion. The
unused clock injection was removed from the registration boundary. The full
CLI race suite and \`go vet ./...\` pass. This closes the CLI test gap, not the
separate missing Agent Manager/BAS/Test Genie receipts needed for browser
qualification.

## 2026-11-23 — UI tests read a stale shared-package install while Audio Tools is running

After \`packages/audio-capture-browser\` built \`dist/voice/audioCueSession.js\`,
\`pnpm test\` in the Audio Tools UI still resolved the local dependency copy under
\`ui/node_modules/.pnpm\` that did not contain that module. UI type-check and lint
pass. A temporary Vitest alias to the current package source passed all 136 UI
files and 1,296 tests. The owner command
\`vrooli package refresh audio-capture-browser --no-restart\` correctly returned
\`running_setup_deferred\` because Audio Tools was live, so it did not install the
new package snapshot into the consumer. No scenario was restarted and no live
UI files were changed. Refresh the consumer during its next governed setup or
restart, then rerun the ordinary UI test command.

## 2026-11-23 — Sherpa readiness is backed by an unmanaged temporary listener

Audio Tools currently selects Sherpa for native streaming and its health probe
reports \`sherpa-stt\` available. The Vrooli \`sherpa-onnx\` resource manager
reports Linux unsupported and the resource is not installed or running. The
only listener on \`127.0.0.1:8881\` is PID 2557133,
\`/tmp/sherpa-bundle-refresh-20261121/server/sherpa-onnx-server\`, in a Codex
scope. Its \`/healthz\` reports the streaming model loaded. Product API replay
against a short fixture succeeds through the Sherpa provider, confirming the
live route works while that listener is present; no request-level PID tracing
was added, so the connection between that process and the product response is
strongly indicated by the sole listener and health result, not directly
instrumented. This is not a supported, reproducible installation or browser
qualification. Do not promote the unsigned temporary artifact by editing the
resource manifest; the release-signed adapter publication gate remains open.

## 2026-11-24 — Unfinished voice sessions now fail closed when recovery state is missing

A retry could reuse an unfinished session identity while resetting the journal
cursor to zero if restore failed or returned no durable snapshot. That could
conflict with the server's advanced sequence ledger. The browser provider now
preserves unfinished identities across preconnect and socket expiry, and refuses
to capture/send until recovery state is available; the user receives an
actionable error and the microphone/socket are released. Fresh identities keep
the bounded in-memory fallback. Focused journal acceptance tests pass (19/19),
including missing/rejected restore, restore gating, and socket reuse. This is
source-level evidence; the shared lab runtime was not restarted.
`;export{e as default};
