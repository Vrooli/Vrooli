# Testing — Audio Tools

Select focused phases under the [repository test protocol](../../../../docs/TESTING.md)
and use the server-owned runner:

```bash
vrooli scenario test audio-tools --phases unit
```

Wait once with the command returned by the runner. Do not poll or start API
binaries directly.

## API tests

Place domain behavior tests beside the owning package under `api/internal/`.
Place transport tests beside `api/handlers/<domain>/`. Use generated proto
types and real SQLite test handles where persistence behavior matters. Keep
handlers thin and test business rules below the transport boundary.

### Shared test infrastructure

Audio fixtures belong under `api/internal/stt/segmenter/testaudio`. Use the
named duration fixtures (`SpeechTonePauseTone3s`, `Silence1s`) or the
sample-rate-aware builders there instead of repeating PCM byte arithmetic in a
test. Provider tests should use the shared fakes in
`api/internal/ai/sttchain/mocks`; builders return fresh instances so call
counts and scripted results cannot leak between tests.

Browser audio tests use the shared `ui/src/audio-integration/test-support`
fixtures for `MediaStream` shapes. Browser APIs, clocks, transport clients and
server endpoints must be injected or mocked at their boundary; tests must not
wait on scheduler timing to establish a transport state. Prefer a readiness
channel or an explicit seam over `time.Sleep`.

Run focused checks during development:

```bash
cd api && GOWORK=off go test ./internal/ai/sttchain
```

## CLI tests

Place command tests under `cli/domains/<domain>/`. Every runtime command must
be declared in `cli/manifest.json` or listed in `exceptions` with a valid
special-case reason. Run:

```bash
cd cli && GOWORK=off go test ./domains/stt
```

## UI tests

Place feature tests beside the component or hook. Use generated clients and
assert user-visible behavior, including failure and accessibility paths. Run:

```bash
cd ui && pnpm test:coverage
```

## Coverage thresholds

Use the thresholds in `.vrooli/testing.json` and the owning package's test
configuration; this guide does not maintain a second set of numeric floors.
The coverage gate is a regression boundary. Add behavior-focused tests before
raising its threshold. Do not weaken the configured threshold to pass a run.

## Streaming checks

After changes to STT selection or stream events, run diagnostics and a smoke
transcription. The stream must produce partial output and a final segment.

```bash
audio-tools diagnostics run
audio-tools stt transcribe-stream --file "<smoke-fixture>"
```

For the complete scenario contract, see `.vrooli/testing.json`,
[`SEAMS.md`](SEAMS.md), and [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md).

## Development pilot evidence contract

This is the target validation design for the first
[development mandate](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md), not
a report of passing tests. Reuse the existing corpus, diagnostics, conformance,
browser capture, experiments, and device evidence before building new instruments.
Keep the dated limitations in `PROBLEMS.md`; do not replace them with green target prose.

| Dimension | Required evidence |
| --- | --- |
| Local | Deterministic adapter/control-flow fixtures plus real-engine streaming, quality, and performance runs. |
| BYOK | Simulated provider protocol, partial revisions, invalid/revoked key, disconnect, rate limit, and honest batch-only behavior. Separately label authorized live-provider smoke. |
| Subscription | LPBS-owned routed fixtures for active/expired entitlement, unavailable inference, and session/usage linkage. Simulated inference is not hosted-service qualification. |
| Credits | Sufficient/insufficient/exhausted balance, concurrent usage, cancellation, settlement retry, and duplicate-delivery cases against the reviewed billing policy. |
| Capture lifecycle | Permission denial, silence, paced speech, stop/final drain, cancellation, reconnect, slow consumer, duplicate events, reload, and recovery. |
| Device/runtime | Declared browser/host, CPU/accelerator, missing-resource, device-change, low-memory, and throttled profiles. Simulation is not native device certification. |
| Consumer portability | Exercise Audio Tools and a separate consumer through the same shared voice contract. |

Use Landing Page Business Suite's fixture owner for subscription/wallet state and
a separate provider boundary for simulated inference. Propagate routed test identity
explicitly. A test process alone does not activate simulation. Verify that fixtures,
test credentials, and seeded balances cannot affect production paths.

Measure these boundaries separately: button action, permission completion, capture
readiness, session readiness, first voiced sample, first partial displayed, final
commit, and stop completion. Derive click-to-ready, speech-to-visible-partial,
final-tail latency, and committed-audio-position lag. A timestamp since the latest
capture callback cannot establish that committed audio is caught up.

Retain distributions, sample counts, failures/timeouts, cold/warm cohorts, corpus,
source/build and engine identity, and host profile. Do not silently exclude failed
attempts from performance or reliability denominators.

Use fast deterministic checks for iteration, paced browser runs for visible behavior,
and real-duration/device runs for claims requiring them. A virtual-time soak is not
an hour of real capture. Assert minimum input duration, sample/event counts, early
EOF, final-tail retention, and requested duration for each relevant branch.

Include negative controls: truncated audio, failed provider children, stale evidence,
duplicate settlement, and a success-shaped wrapper with an unavailable required row.
The instrument must reject those claims before its green result supports acceptance.
Numeric SLOs, corpus floors, platform commitments, and paid-test budgets remain
review decisions. The concrete candidates below make that review actionable;
they do not describe achieved results.

### Pilot decision sheet and safe first slice

The [machine-readable local pilot proposal](local-dictation-proposal.json) can
be inspected through Swarm's non-launching `transitions preview-development`
command. It is a review input, not an accepted target or a backlog item. Token
and wall-time limits deliberately remain zero until proposed and approved by
the operator; zero is a missing-budget finding, not unlimited authority. The
first slice targets evidence instrumentation and local partial/final integrity,
not the complete commercial/device vision.

Preparation check on 2026-09-08: the usage/improve skill set validated; the
read-only `audio-tools.setpoint-read` program completed
(`prog_f5b694d3-69da-4635-b437-afddf48fc404`). It returned two available engines,
one native-streaming engine, and nine unknown outcome rows. All 15 evidence-board
unit tests passed. This proves discovery and conservative collection behavior,
not browser latency, recognition quality, billing correctness, or fresh-agent
continuation. No development agent was launched for this check.

Keep pending targets explicit until the operator approves them in the product
contract. Do not encode the candidate values below as passing requirements.

| Decision | Proposed review starting point | Evidence needed before adoption |
| --- | --- | --- |
| Interactive latency | Warm click-to-ready p95 ≤ 500 ms; speech-to-visible-partial p95 ≤ 1 s; stop-to-final-tail p95 ≤ 1 s. Report cold readiness separately. | Paced browser traces with permissions already granted, at least 30 attempts per declared route/profile, including failures; separately measure first permission flow. |
| Transcript quality | English pilot: micro-averaged WER ≤ 10% for clean speech, ≤ 20% for the noisy cohort; report every accent/vocabulary subgroup separately. These candidate floors are not measured baselines. | Human-reviewed references, corpus contract below, and real-engine runs; product review must accept the language scope and per-cohort floors. |
| Reliability and soak | Zero silently lost or duplicated committed intervals in the test cohort; every interrupted interval processed, retained, or explicitly recoverable. Three real 60-minute runs per claimed local engine/profile, with final-tail assertions. | Existing OT-P0-001 still governs all intervals. This sample establishes tested behavior, not a fleet reliability percentage. Recognition mistakes and transport loss are separate metrics. |
| Device promise | Qualify profiles D1–D6 below in stages; D1 is the first implementation cohort, not an existing certificate. | Native host receipts for supported claims; emulation only supports simulated claims. |
| Route fallback | Keep the selected local/BYOK/owned route explicit; propose no automatic switch to a billable route without prior user policy. | Provider-unavailable and revoked-key tests prove the configured fallback and visible reason. |
| Settlement | Review the single [proposed billing policy](../business/MONETIZATION.md#proposed-voice-billing-policy-v1); do not create a private Audio Tools ledger. | LPBS routed wallet/session fixtures and duplicate/concurrent settlement assertions against the adopted policy revision. |
| Test spending | No paid-provider or subscription purchase in the initial slice; request a capped allowance for later live tests. | Aggregate accounting across retries and child runs; simulated balances cannot stand in for paid qualification. |

The safe first runtime slice measures and repairs local session discovery, then
proves paced partial rendering and final drain through the shared consumer contract.
Use local deterministic fixtures first; retain observed performance even before a
numeric floor is approved. Add LPBS-routed entitlement/wallet state and a fake provider
boundary next. This proves routing/control flow, not implemented hosted inference.
Only then request live-provider budgets and qualify the reviewed device/quality
matrix. Missing product decisions do not block these authorized instruments, but
they do block claims that the complete voice target has been accepted.

### Published targets and pending decisions

The user-requested `portable-voice-v1` PRD regeneration on 2026-09-09 publishes
the full product scope. It does not turn proposed numbers or commercial terms
into approved bands. The former V1–V4 labels are historical proposal keys;
use the stable OT/ATD links below for current work.

| Concern | Published target | Required obligations |
| --- | --- | --- |
| No-loss sessions and existing engine/policy trust | OT-P0-001 through OT-P0-003 | Existing ATD-P0-001 through ATD-P0-007; preserve their evidence and identifiers |
| Explicit local/BYOK/owned routing (former V1) | OT-P0-004 | ATD-P0-008 |
| Genuine partials, final drain and readiness (former V2) | OT-P0-005 | ATD-P0-009 and ATD-P0-010 |
| Recognition quality | OT-P0-006 | ATD-P0-011 |
| Owned delivery and settlement (former V3) | OT-P0-007 | ATD-P0-012 and ATD-P0-013 |
| Compatible hosts and supported-device evidence | OT-P0-008 | ATD-P0-014 and ATD-P0-015 |
| Private bounded lifecycle | OT-P0-009 | ATD-P0-016 |
| Isolated simulations and honest completion | OT-P0-010 | ATD-P0-017 through ATD-P0-019 |
| Evidence UX, mobile recovery, maintainable adapters (former V4), other voice operations | OT-P1-001 through OT-P1-004 | ATD-P1-001 through ATD-P1-004 |
| Additional native-device expansion | OT-P2-001 | ATD-P2-001; baseline advertised support is already P0 |

The decision sheet still needs explicit adoption of numeric warm/cold latency
bands, bounded cold-start/admission timeouts, language/subgroup floors, realized
corpus, nominated native profiles, billing policy and aggregate execution/spend
allowances. Cold model download/provisioning and human permission time need
separate measurements; neither is silently folded into a warm-start percentile.
Document maximum supported session/retention limits and concurrency per profile.
An agent may improve measured behavior within its grant before these decisions,
but cannot invent a completed numerical or commercial acceptance contract.

### Full mandate and completion contract

The full development objective is to bring the approved portable voice targets
into band, using `audio-tools-improve` to select successive repairs from evidence.
Swarm retains one approved development item, its target/artifact revision,
change/effect boundary, allowance, interruption history and acceptance decision.
Do not submit every ordinary repair as a separate approval or call completion
when only the Linux local slice passes.

The approval packet must identify:

1. Required OT/ATD outcomes and the device/route/language matrix. For the first
   STT engagement include all P0 outcomes plus the evidence/recovery and
   maintainable-adapter work needed to achieve them. Preserve existing TTS and
   summarization behavior. If OT-P1-004 expansion or P2 additional devices are
   deferred, name that explicitly; do not describe STT completion as every voice
   capability being qualified.
2. Approved SLO, corpus, measurement-method and commercial-policy revisions.
   Separate deterministic simulated BYOK/owned correctness from authorized live
   delivery and native hardware. A simulation-only grant cannot earn live claims.
3. Exact permitted shared owners: Audio Tools, shared capture and proto contracts,
   named consumer adapters, relevant resource/control-plane code, and shared
   hosted-delivery/monetization boundaries where needed. Do not copy an excluded
   owner's implementation into Audio Tools to evade the grant.
4. Positive agent token/time allowances, delegated accounting, permitted inference
   and financial effects, cancellation/reconciliation policy and escalation
   triggers. These docs supply none of those grants.
5. Owner-backed evidence resolvers and explicit acceptance policy, including
   behavior on stale/unknown evidence, budget exhaustion and interrupted runs.

The historical `local-dictation-proposal.json` is deliberately narrower and
currently has zero budgets, prose effects/evidence sources and shared-owner
exclusions. It is not a launch-ready full mandate under Swarm's typed review
contract. Author a reviewed full-scope packet only after resolver IDs, authority
and budgets exist; do not invent IDs to make preview pass.

Execution sequence: qualify the measuring instruments; measure and repair local
readiness/streaming through shared consumers; qualify all simulated route and
billing controls; complete owned delivery through its owner; run permitted live
and native cohorts; refactor against those behavioral regressions and rerun the
affected matrix. Refactoring can occur earlier when a broken seam blocks reliable
tests. Reuse previous valid receipts; this order does not require repeating work.

Swarm/Agent Manager own launch dispatch, native-goal versus governed fallback,
continuation, interruption recovery and budgets. Their qualification is a
precondition for autonomous execution, not another Audio Tools feature. Tech Tree
Designer remains deferred. Canonical documentation edits explicitly requested by
the operator do not require its draft-bundle workflow.

Whole-mandate completion requires every required cell to be accepted with matching
current provenance. Unknown or unavailable required cells block completion,
including targets not emitted by the current measurement program. An agent run
ending, a plan marked complete, a passing structural validator or a fresh health
response is not that decision. New targets, broader authority or additional spend
require an amendment; successive repairs inside the same grant do not.

### Acceptance scenarios to implement

These are Gherkin-style validation designs, not executed evidence. Bind each to
the owning tagged test or real qualification receipt as implementation lands.

- **Route refusal:** Given a local-only policy and an unavailable local engine,
  when recording is requested, then the app shows an actionable unavailable state
  and neither remote inference nor a Vrooli debit occurs.
- **Visible streaming:** Given paced speech and a streaming-capable route, when
  revisions arrive before stop, then the rendered interim text updates, committed
  identities remain stable, and stop drains the known final phrase inside the
  approved band. A final-only adapter cannot pass this case.
- **Interrupted capture:** Given acknowledged and unacknowledged intervals, when
  transport, device or process interruption occurs, then recovery replays only
  needed ranges and every interval has explicit coverage without duplicate commit.
- **Owned service:** Given routed subscription/credit fixtures and separate fake
  inference, when partial delivery, exhaustion, cancel and settlement retry occur,
  then the visible result and the shared ledger match the adopted policy exactly.
- **Isolation:** Given absent, expired or wrong-account fixture identity, when a
  test attempts hosted admission, then it cannot reach production funds or keys
  and cannot receive a success-shaped simulated entitlement.
- **Instrument integrity:** Given an early EOF, failed STT child, stale receipt or
  omitted required target, when the wrapper reports success, then acceptance
  rejects it even if a different child ran for an hour.
- **Consumer portability:** Given the same route and fault recipe, when Audio Tools
  and another claimed adopter capture the turn, then both use the shared contract
  and expose equivalent coverage, interim/final semantics and recovery outcomes.

### Corpus commitment for the pilot

Use the existing corpus and experiment services; do not build a second runner or
store speech in skill/program outputs. The proposed release corpus is
`voice-pilot-en-v1`. Create its realized manifest only from licensed or consented
recordings and human-reviewed reference text. No suitable dataset or permission
is assumed by this design.

| Lane | Composition and purpose | What it cannot establish |
| --- | --- | --- |
| Deterministic transport | Existing `api/internal/stt/segmenter/testaudio` duration fixtures and provider fakes; add silence, truncated PCM, duplicate/out-of-order events, rate limit and cancellation controls | Recognition quality, realistic accent coverage, elapsed one-hour reliability |
| Recognition | Candidate minimum: 60 distinct utterances, ≥ 3,000 reference words, ≥ 6 consenting speakers; include US, UK and at least one additional English accent group, each with ≥ 2 speakers and ≥ 500 words. Include numbers, punctuation requests and technical vocabulary | Other languages or populations absent from the cohort; repeated clips are not new speakers or words |
| Noise | Pair each clean recording with versioned 20 dB and 10 dB SNR augmentations; retain clean/noisy identities and subgroup metrics separately | Natural noise or microphone diversity not represented by augmentation |
| Interactive browser | Replay paced audio through the actual shared capture seam, rendering partials in Audio Tools and Web Console; ≥ 30 attempts per route/profile and cold/warm cohort | Native microphone/device claims when browser media is injected |
| Long form | Seeded composition into three 60-minute realtime sessions per claimed local engine/profile, plus distinct ending phrases and declared fault schedules | An accelerated replay or last-N-seconds latency approximation is not a realtime soak |

Pin manifest version, source/consent reference, content hashes, sample rate,
duration, reference-normalization revision, speaker pseudonym, cohort tags, split,
and realized corpus IDs. Keep development and held-out qualification recordings
disjoint. Record every missing subgroup as an unqualified scope, not an excluded
denominator. Follow the existing report normalization policy; WER can exceed 1
and must not be clamped. Inspect insertions/deletions/substitutions alongside WER.
Test tail loss with known ending phrases and interval coverage, not WER alone.

Changing references, normalization, corpus composition, or metric computation
creates a new version and invalidates cross-version acceptance comparisons.
Retain the old evidence; do not silently rewrite it. Store sensitive recordings
only under the corpus owner's storage/retention policy. General work records use
hashes and metadata, not speech or transcript payloads.

### Device qualification profiles

These are proposed coverage commitments, not a list of supported deployments.
Pin actual OS/browser versions, architecture, CPU, memory, accelerator, engine and
model revision in each receipt; `latest browser` is not a reproducible identity.

| Profile | Native cohort to nominate | Required distinction |
| --- | --- | --- |
| D1 | Linux amd64, Chromium, CPU-only and an available accelerator as separate cells | First local implementation cohort; Whisper and Kyutai independently qualified |
| D2 | Linux amd64 Chromium, 2-vCPU/4-GiB constrained test environment, CPU-only | Real resource limits recorded; browser CPU throttling alone is not this profile |
| D3 | macOS arm64, Safari and Chromium | Do not infer local-engine artifact support from browser compatibility |
| D4 | Windows amd64, Chromium | Qualify native capture and compatible local engine artifacts independently |
| D5 | iOS Safari and installed PWA on a named physical device | Foreground/background transitions, audio interruption, permission, reload and retained-audio recovery |
| D6 | Android Chrome on a named physical device | Device switch, Bluetooth interruption, memory pressure and foreground recovery |

All profiles exercise simulated route control flow where the browser is available.
Local inference is claimed only where that engine has a supported artifact and a
passing native receipt. Unsupported local inference must yield a visible reason
and explicit alternative; it must not silently spend credits. Live BYOK/owned
qualification is a separate grant. D3–D6 remain unqualified until devices and
their operators are nominated. BAS emulation can unblock control-flow work, not
substitute for these native receipts.

### Measurement program and acceptance boundaries

`program-runtime library run audio-tools.setpoint-read` gathers inventory.
Add `--input experiment_id=<id>` to inspect a selected persisted report. Selection
is explicit: the newest successful experiment may be stale or from a different
engine, corpus, lane or build. The program exposes bounded replay WER/RTF and
finalization metrics with their limitations. It never converts replay timing to
speech-to-visible-partial latency and never starts inference or a paid operation.

The v2 program enumerates all 15 targets with `target_revision=portable-voice-v1`.
It retains the nine original row IDs and adds six; its target strings summarize
scope, not approved numeric bands. The all-published-targets regression catches
inventory drift. Every outcome still has a null reading and null in-band value.
W2 implementation must add owner-backed receipt joins; until then whole-product
acceptance is unknown by construction. Future acceptance must reject mismatched target,
method, corpus, build, device or policy revisions; missing attempts; early EOF;
failed children; missing entitlement/settlement proof; and simulated evidence
presented as native or live. Fix receipt ownership before adding that join to
the program. Do not embed billing rules or provider qualification policy in Python.

Run local branch regressions with `python3 -m unittest discover -s
scenarios/audio-tools/test -p 'test_setpoint_read.py'` from the repository root.
Run governed fixtures and role declarations with
`vrooli scenario test audio-tools --phases programs,skill-set`.
Those checks validate the instruments, not streaming or commercial readiness.

## Cross-references

- [Flows](../concepts/FLOWS.md) — capture, transcription, and recovery behavior.
- [API](../reference/api-endpoints.md) — transport and generated-client contract.
- [Problems](PROBLEMS.md) — dated failures and evidence limitations.
- [Seams](SEAMS.md) — injectable boundaries and the registry checked by tests.
