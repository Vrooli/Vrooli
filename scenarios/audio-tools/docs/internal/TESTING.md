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

### Lab/qualification tests

Lab packages (`internal/experiment`, `internal/eval`, `internal/soak`, and the
`handlers/{experiment,eval,soak,validation}` transports) compile only into the
API built with `-tags audio_lab`. The product API dependency graph excludes
them. Their contract — no transitive import from serving code, explicit
composition crossings, and no lab packages in the product build — is documented
and enforced by `api/internal/lab`. Run their tests with:

```bash
cd scenarios/audio-tools && make lab-test
cd api && go test ./internal/lab/... -count=1
```

The evaluation package exposes `ScoreTimestampDrift` for a supplied, versioned
timed reference and provider word intervals. Its result reports exact
lexical-match coverage and start/end error distributions; it intentionally does
not assign zero drift to insertions, deletions, or substitutions. `eval.Replay`
now retains bounded, provider-reported hypothesis intervals in
`ClipResult.HypothesisWordTimestamps` when a segment carries them. The Whisper
VAD segmenter retains timings only for newly committed words after pre-roll
overlap removal, anchored at the absolute segment sample; it marks truncation
above 20,000 words per clip and never persists audio bytes. Native streaming
providers may omit word timings.
The scorer also enforces the same 20,000-word-per-side ceiling before token
normalization or its identical-stream fast path, so callers cannot bypass the
bounded-memory contract with a very large identity comparison.

No checked-in, human-reviewed timed reference currently exists for the
61-minute dictation fixture, so `TimestampDrift` remains unset and this scorer
is not evidence about any model. Do not compare provider timings to themselves,
or infer qualification from synthetic unit fixtures or the presence of a report
field.

`go test ./internal/eval -run '^$' -bench 'BenchmarkScoreTimestampDrift_(61MinuteWordCount|20000WordIdentity)$' -benchmem -benchtime=1x`
exercises the 10,000-word fixture and the full 20,000-word admitted identity
case. Identical lexical streams use a linear-time path; other streams retain
the canonical minimum-edit traceback, with a 512M-cell ceiling (about 128 MiB
of packed traceback, enough for a full 20,000-by-20,000 matrix). On the
recorded Linux/amd64 host the 10k and 20k identity cases took 2.72 ms and
6.18 ms, using 2.34 MB and 4.57 MB respectively. The mismatch worst case
remains quadratic in time and memory. These are synthetic alignment checks,
not speech-model measurements.

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

## Declaration and registry gates

Generated surfaces are gated, not trusted:

```bash
make catalog-check
make catalog-gen
make operations-gen
make models-doc
make endpoints
```

Run these commands from `scenarios/audio-tools`. The catalog gate checks
provider, operation, model, and cross-catalog declarations. The generators
render provider Go/TypeScript surfaces, operation UI specs, model reference
documentation, and endpoint declarations, respectively.

Each generator has a drift test that regenerates in memory and compares
byte-for-byte. A hand-edited generated file fails the suite. The
cross-catalog conformance test (`api/internal/catalogconformance`) also proves
that provider, engine, and model references resolve in both directions, that
the STT engine manifest and the provider catalogue agree, and that the
summarize records mirror `resources/ollama/model-policy.json`.

## Coverage thresholds

The enforced floors live in `api/coverage_policy_test.go`; `.vrooli/testing.json`
configures the harness, not a second numeric list. A floor is a measured
regression baseline: add behavior-focused tests before raising it, and lower it
only with a comment naming what regressed. An unlisted package is unenforced,
not exempt by design.

```bash
cd api && go test -run TestCoverageFloors -v .
```

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

The operator adopted the candidate values below on 2026-09-13; the approved
bands and scope are in the [outcome evidence inventory](#outcome-evidence-inventory).
Adoption makes them targets, not passing results. Every row still needs evidence.

### Current access inventory (2026-09-09)

This inventory describes what the current workspace can inspect or exercise; it
is not a qualification receipt and does not authorize a launch.

| Required input | Current access | Qualification consequence |
| --- | --- | --- |
| Audio source and held-out corpus | Existing checked-in synthetic/test fixtures and the declared corpus recipe; no newly supplied licensed or consented speech inventory | Transport and fixture behavior can be tested; recognition floors remain unknown |
| Local engines | Existing Audio Tools discovery and deterministic provider fakes; real-engine host/profile receipt is not attached to this packet | Engine availability is observable; stable-provider and duration claims remain unearned |
| Native devices | No nominated physical D1–D6 device, browser version, operator or access receipt | Device rows remain unavailable |
| BYOK provider | Protocol fixtures and configuration boundaries are present; no live key is supplied | Simulated route refusal can be tested; live delivery remains unavailable |
| Owned hosted provider | LPBS-owned routed fixture seam is the intended owner path; no production account or paid allowance is supplied | Entitlement/inference/settlement fixtures must remain separate from hosted qualification |
| Acceptance resolvers | The read-only board enforces `audio-tools-acceptance-v2` target claims and provider-cell/device identity. A current simulated producer receipt covers five policy/adapter rows; remaining required owner receipts are unavailable or incomplete. | Full-target acceptance cannot be submitted as complete |
| Agent authority | Controlled owner stubs and fallback capability fixtures are available; no human-issued development grant is attached | Q2 continuation can be exercised; Q3 real-agent evidence remains pending |

The next access decisions belong to the operator and the named owners: corpus
permission, device nominations, provider credentials, commercial policy,
resolver registration and positive aggregate limits. Missing access stays an
explicit unknown and is never filled by a local simulation.

| Decision | Proposed review starting point | Evidence needed before adoption |
| --- | --- | --- |
| Interactive latency | Warm press-to-first-accepted-speech p95 ≤ 500 ms; speech-to-visible-partial p95 ≤ 1 s; stop-to-final-tail p95 ≤ 1 s. Report cold readiness separately. | Paced browser traces with permissions already granted, at least 30 attempts per declared route/profile, including failures; readiness ends at the first voiced sample accepted by the capture/transcription path, not button handling or a readiness banner; separately measure first permission flow. |
| Transcript quality | English pilot: micro-averaged WER ≤ 10% for clean speech, ≤ 20% for the noisy cohort; report every accent/vocabulary subgroup separately. These candidate floors are not measured baselines. | Human-reviewed references, corpus contract below, and real-engine runs; product review must accept the language scope and per-cohort floors. |
| Reliability and soak | Zero silently lost or duplicated committed intervals in the test cohort; every interrupted interval processed, retained, or explicitly recoverable. Repeated microphone use must pass at least five consecutive warm press/speak/transcribe/stop cycles in one page and browser context, with no refresh, app restart, or manual recovery. Three real 60-minute runs per claimed local engine/profile, with final-tail assertions. | Existing OT-P0-001 still governs all intervals. For repeated-session evidence, retain every cycle's start/stop, first-accepted-speech, transcription, final-drain, error, and recovery result; one failed cycle fails the sequence. This sample establishes tested behavior, not a fleet reliability percentage. Recognition mistakes and transport loss are separate metrics. |
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

For the active portable-voice mandate, “external access unavailable” must be
interpreted against the grant, not against the full product vision. The grant
authorizes an open-licensed `voice-pilot-en-v1` corpus, the local D1 host, a
constrained D2 cell, and simulated BYOK/owned route and settlement fixtures.
Agents must build the corpus/qualification harness and emit honest owner-backed
receipts for those lanes. Private recordings, physical D3-D6 devices, live
provider keys, and paid-provider allowance are not required for this mandate and
must remain excluded rather than stopping local implementation. They can block
only a later live qualification grant.

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

The operator adopted the decision sheet on 2026-09-13; see the inventory below.
Still open, and to be proposed by the mandate's agent for review rather than
fixed silently: bounded cold-start/admission timeouts, maximum supported
session/retention limits, concurrency per profile, and the billing
sub-parameters listed in MONETIZATION.md. Cold model download/provisioning and
human permission time need separate measurements; neither is silently folded
into a warm-start percentile. An agent cannot invent a completed numerical or
commercial acceptance contract beyond what the inventory states.

### Outcome evidence inventory

Inventory revision `portable-voice-v1`, bands approved by the operator on
2026-09-13. Row IDs are the `audio-tools.setpoint-read` row IDs. A row is in
band only when it reads from a current owner-backed receipt that matches this
revision, method, cohort, build, device, policy, and corpus identity.

#### Owner-backed board receipt join

The Audio Tools experiment evidence store is the owner of the transcript-free
board receipts. Each qualification artifact may carry an `acceptance` object in
`machine_json` with this shape:

```json
{
  "target_revision": "portable-voice-v1",
  "method": "audio-tools-acceptance-v2",
  "cohort": "voice-pilot-en-v1",
  "build": "audio-tools-api:<immutable-build>",
  "device_profile": "D1",
  "policy_revision": "policy-v1",
  "corpus_revision": "voice-pilot-en-v1",
  "provider_cell": {
    "engine_id": "simulated-policy",
    "model_id": "simulation-v1",
    "strategy": "simulation",
    "device_profile": "D1"
  },
  "acceptance": {
    "dictation-trust": {"in_band": true, "negative_controls_passed": true}
  }
}
```

The program joins only receipts with a non-empty artifact reference, exact
target revision, non-empty cohort/build, `in_band: true`, and
`negative_controls_passed: true`. A required row is green only when every
required row has a matching receipt in one current identity tuple and no matching
receipt for that row failed. Out-of-band claims are surfaced as failed rather
than pending. Missing, malformed, stale, mixed-context, excluded,
or simulated-as-native receipts remain visible as unknown/failed evidence.
`shared-voice-capabilities` and `device-qualification` are explicitly excluded
for this mandate and can never be promoted by this join. This keeps the board a
read-only resolver over owner evidence; it does not manufacture receipts or
change a band.

After all required owner producers have fresh passing rows, run
`audio-tools validation run-acceptance-join`. This owner-side command is the
only composite producer: it validates the target/method/cohort/policy/corpus
identity and 24-hour freshness of each source, persists every source artifact
and execution lane in `details`, and emits one composite acceptance identity
for the strict board join. Composite evidence is not native evidence:
simulated rows remain labelled `simulated`, native rows remain labelled
`native`, and `simulated_as_native` is always false. The command fails with the
missing row names when a prerequisite is absent.

**Grant scope.** Required evidence is local and simulated: real local engines on
this Linux host as device profile D1, plus D2 as a 2-vCPU/4-GiB constrained
container, and deterministic capability/route simulations for D1–D6. BYOK and
owned/credit routes are proven with LPBS routed fixtures and independent fake
providers. There are no live keys, paid providers, purchases, real money, or
physical D3–D6 devices in this grant. Label simulations explicitly; they cannot
qualify native device or live-provider claims. Additional native D3–D6 and live
BYOK/owned delivery remain unqualified and outside this grant. The recognition
corpus may be built from openly licensed public corpora (for example
LibriSpeech, CC BY 4.0, or Mozilla Common Voice, CC0), with each license
recorded; no private recordings.

| Row | Outcome | Band | Validity |
| --- | --- | --- | --- |
| `dictation-trust` | OT-P0-001 | Zero silently lost or duplicated committed intervals across the deterministic transport lane and the interrupted-capture scenario. Three real 60-minute sessions per qualified local engine on D1, each with its known ending phrase fully committed. Every interrupted interval is processed, retained or explicitly recoverable. | Declared fault schedule; failed attempts retained |
| `provider-parity` | OT-P0-002 | Each engine offered as stable independently meets the `dictation-trust`, `speaker-policy`, `interactive-latency` and `corpus-quality` bands on D1. An engine that does not is visibly not offered as stable. | Per-engine receipts; model revision pinned |
| `speaker-policy` | OT-P0-003 | Requested verification or extraction exposes an applied or degraded state per segment on batch and native-streaming paths. Required policy fails closed. The verify error rate is operator-visible. | Negative controls: resource down, verify error |
| `explicit-routing` | OT-P0-004 | The route-refusal and route-fallback scenarios pass for local, simulated BYOK and owned fixture routes. There is no automatic switch to a billable route without a prior user policy, and the actual route is shown. | Simulated routes labelled simulated |
| `interactive-latency` | OT-P0-005 | Warm click-to-ready p95 ≤ 500 ms. Speech-to-visible-partial p95 ≤ 1 s. Stop-to-final-tail p95 ≤ 1 s. Cold readiness is reported separately. Batch-only routes are identified as batch-only. | ≥ 30 paced attempts per route/profile cohort, failures included, in Audio Tools and Web Console on D1 and D2 |
| `corpus-quality` | OT-P0-006 | Micro-averaged WER ≤ 10% clean and ≤ 20% noisy (20 dB and 10 dB SNR) on held-out `voice-pilot-en-v1`, which meets the corpus recipe below. Every subgroup is reported; a missing subgroup is unqualified. | Licenses recorded; dev and held-out disjoint; normalization revision pinned |
| `owned-settlement` | OT-P0-007 | Every "required proof" cell of the adopted billing policy v1 passes on LPBS routed fixtures with an independent fake provider. Local and BYOK incur zero Vrooli debit. | Simulation only; open sub-parameters flagged for review |
| `host-portability` | OT-P0-008 | D1 (CPU-only and accelerator cells) and D2 each select a compatible path or return an explicit unavailable state. Every claimed route and device has a matching receipt, and D3–D6 are listed as not claimed. | Pinned OS, browser, CPU, memory, accelerator, engine and model |
| `private-lifecycle` | OT-P0-009 | Each item has a test with a negative control: authorized access, bounded recoverable retention and deletion, metadata-only diagnostics, and explicit consent before a destination change. | Current build |
| `acceptance-integrity` | OT-P0-010 | The instrument-integrity scenario passes: early EOF, a failed child, a stale receipt, an omitted target and simulated-as-native evidence are all rejected. Every required row reads from an owner-backed receipt. | Program fixtures plus a live read |
| `desktop-dictation-parity` | OT-P0-011 | On each supported desktop cohort, Audio Tools, Web Console, and Swarm Manager match the reviewed 2026-11-20 Wispr Flow feature/task matrix and achieve at least equal task completion, including accessible insertion, onboarding, permissions, offline use, and recovery. | Versioned competitor/task matrix; matched scripts, corpus, profile, and denominators; a separate product-path receipt per consumer; failed tasks retained |
| `provider-evidence` | OT-P1-001 | Dictation Studio persists provider-neutral metrics and promotion verdicts per provider, engine, strategy, policy, fault and replay cell. | Persisted experiment reports |
| `mobile-recovery` | OT-P1-002 | In a mobile-viewport browser cohort, a user inspects coverage, recovers retained audio and exports metadata-only diagnostics without developer tools. | Emulated device labelled emulated; native mobile excluded |
| `adapter-maintainability` | OT-P1-003 | Capability and host adapters are versioned and covered by shared conformance tests. No consumer duplicates capture, recovery, routing or billing rules. | Conformance suite result |
| `shared-voice-capabilities` | OT-P1-004 | Excluded as expansion. Existing TTS and summarization behavior must not regress. | Regression tests only |
| `efficient-model-selection` | OT-P1-005 | Local route resolution uses declared language, rights, held-out quality, host capacity, cost, and fallback; Sherpa streaming Zipformer, eligible Kale Nano, Qwen3:1.7b, and consented cloning candidates retain their declared readiness. | Catalog and declaration checks; per-capability model-resolution receipts on D1/D2; matching quality/capacity evidence; candidate or experimental status is not qualification |
| `advanced-voice-workflows` | OT-P1-006 | Each offered workflow—synthesis, voice customization, consented cloning, conversion, speaker separation, wake word, and duplex—has a reusable contract and measured quality, latency, and recovery. | Separate workflow/route receipts and negative controls; cloning consent and privacy evidence; duplex consent and durable spend/accounting evidence; unsupported or unmeasured workflows remain unavailable |
| `standalone-offer-upgrade` | OT-P1-007 | The Audio Tools entry journey presents the approved $10/month standalone offer with correct entitlement, payment recovery, cancellation, and clear upgrade to the full bundle. | LPBS-owned routed test account and checkout/entitlement/upgrade receipts; production balances and purchases excluded; each lifecycle failure retained |
| `offline-install-lifecycle` | OT-P1-008 | On every supported desktop install profile, offline model provisioning and failed install, update, upgrade, or model provisioning recover cleanly. | Installer/release-owner receipts with pinned platform and signed artifact identity; network-disabled provisioning plus interrupted/rollback/retry scenarios; unsupported platforms remain explicit |
| `device-qualification` | OT-P2-001 | Excluded. | — |

Evidence audit: no band edited, no sensor edited to move a row, every required row
read from the program as often as its protocol requires, and excluded rows
reported as excluded.

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

The bounded labelled policy/adapter simulation is executable with
`audio-tools validation run-simulated --json`. It exercises explicit local,
BYOK, and consented-owned routing; bounded reservation/finalization with
idempotent retry; zero-debit local/BYOK controls; metadata-only lifecycle
deletion; adapter identity; and acceptance negative controls. It records one
`audio-tools-acceptance-v2` receipt labelled `execution_lane: simulated` and
`simulated_as_native: false`. That receipt is valid only for the simulated
policy/adapter rows. It must not be used as native, provider-parity,
dictation-trust, mobile-recovery, host-portability, corpus-quality, or
wall-clock latency proof; those product gates still require their own
owner-backed receipts when the corresponding paths are available.

Use `program-runtime --instance lab library run audio-tools.setpoint-read` to
read lab-owned qualification receipts. The live Audio Tools instance does not
mount the lab-only experiment service. `provider_cell.device_profile` must
exactly match top-level `device_profile`; the board rejects envelopes that
omit or contradict this execution identity.

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

Each clip's transcript must have a separate human audit record before it can
support a quality receipt. `corpus import --reference-review-file` accepts one
JSON object containing `reviewer`, `protocol`, and RFC3339 `reviewed_at`; the
server binds it to the exact reference text with `reference_sha256`. The
reviewer must compare the transcript with the recording using the named
protocol and correct every discrepancy. Automated alignment, dataset-provided
labels, and an unattended import do not count as review. Existing rows and
imports without this record remain unreviewed and fail the quality cohort gate.
The record is an accountable provenance assertion; software does not observe
or independently certify the human listening session. Never backfill review
records onto existing references without performing and recording that audit.

The server-owned quality experiment budget permits up to two hours for one
explicit provider cell. Paired held-out noise cohorts can require more wall time
than the 60-minute recognition-duration rung; this budget does not authorize
multi-cell matrices or change any acceptance band.
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
`audio-tools validation device-matrix` (or `--json`) enumerates the declared
cells from `coverage/device-qualification-matrix.json`. The empty starting
receipt register is `coverage/device-qualification-receipts.json`. To evaluate
observations, pass `--receipt-path coverage/device-qualification-receipts.json`. The receipt
file uses schema version 1 and a `receipts` array keyed by exact `cell_id`.
The checked-in declaration uses `qualification_boundary: product_path_receipts`;
an `inventory_only` declaration is never allowed to accept a passing receipt.
Each receipt declares `outcome` (`passed`, `unavailable`, or `failed`). A pass
requires observation time, exact OS version and architecture, CPU and memory,
accelerator identity, browser name and version, named device/host, declared
provider and model plus their versions, a passing product path, and artifact
reference plus SHA-256. Constrained profiles must report the declared resource
limits exactly. Unknown/duplicate cells, unknown receipt fields, mismatched
platform/provider/model, and incomplete passing receipts are rejected. An
unavailable result requires a concrete reason and remains
unqualified; a missing receipt remains `not_run`. `matrix_complete` is true only
when every declared cell has a valid passing receipt. Receipts are accountable
operator evidence; the command checks structure and identity consistency but
cannot independently observe a human/device session. Kale Nano remains
conditional on its runtime being available.

| Profile | Native cohort to nominate | Required distinction |
| --- | --- | --- |
| D1 | Linux amd64, Chromium, CPU-only and accelerator as separate cells; accelerator may be explicitly unavailable when no device is assigned | First local implementation cohort; Whisper and Kyutai qualify independently |
| D2 | Linux amd64 Chromium, 2-vCPU/4-GiB constrained test environment, CPU-only | Real resource limits recorded; browser CPU throttling alone is not this profile |
| D3 | macOS arm64, Safari and Chromium | Do not infer local-engine artifact support from browser compatibility |
| D4 | Windows amd64, Chromium | Qualify native capture and compatible local engine artifacts independently |
| D5 | iOS Safari and installed PWA on a named physical device | Foreground/background transitions, audio interruption, permission, reload and retained-audio recovery |
| D6 | Android Chrome on a named physical device | Device switch, Bluetooth interruption, memory pressure and foreground recovery |

The setpoint host-portability join has a fixed required-cell inventory for this
grant: D1 Linux/amd64 Chromium CPU, D1 Linux/amd64 Chromium accelerator, and D2
Linux/amd64 Chromium at 2 CPU cores and 4 GiB. Each receipt must enumerate all
three cells and pin OS/browser versions, host identity, engine/model revision,
and artifact references. Each cell also needs a passing product-path artifact
with the same artifact reference as its hardware/browser observation.
`available` requires a passed inference and a named compatible path;
`unavailable` is a valid portability outcome only with a concrete reason and a
visible fallback in the product path. Missing cells, versions, provenance, or
fallback proof keep `host-portability` failed/unknown. D3–D6 remain excluded
from this matrix until their named native devices and operators provide receipts.

All profiles exercise simulated route control flow where the browser is available.
Local inference is claimed only where that engine has a supported artifact and a
passing native receipt. Unsupported local inference must yield a visible reason
and explicit alternative; it must not silently spend credits. Live BYOK/owned
qualification is a separate grant. D3–D6 remain unqualified until devices and
their operators are nominated. BAS emulation can unblock control-flow work, not
substitute for these native receipts.

### Measurement program and acceptance boundaries

`program-runtime --instance lab library run audio-tools.setpoint-read` gathers
inventory and lab-owned owner receipts. The product Program Runtime instance
resolves the live Audio Tools API, which intentionally does not mount the
experiment evidence service.
Add `--input experiment_id=<id>` to inspect a selected persisted report. Selection
is explicit: the newest successful experiment may be stale or from a different
engine, corpus, lane or build. The program exposes bounded replay WER/RTF and
finalization metrics with their limitations. It never converts replay timing to
speech-to-visible-partial latency and never starts inference or a paid operation.
Qualification evidence is a repeated response field and is consumed through
Program Runtime's raw response projection; response metadata is not treated as
the evidence list. The resolver scans up to 512 persisted rows, using a
sentinel above that bound rather than silently truncating history. This
distinction is covered by the board regression.

The v3 program enumerates all 20 targets with `target_revision=portable-voice-v2`
and resolves required rows through the owner-backed receipt join. It retains the
original row IDs and adds targets from the current contract revision; its target strings summarize scope, not
approved numeric bands. The all-published-targets regression catches inventory
drift. Missing receipts remain unknown, while the two excluded rows are explicit.
The join rejects mismatched target,
method, corpus, build, device or policy revisions; missing attempts; early EOF;
failed children; missing entitlement/settlement proof; and simulated evidence
presented as native or live. Receipt ownership remains the producer's
responsibility. The current v3 program consumes only resulting target-aware
receipts; it does not create them or embed billing rules/provider qualification
policy in Python.

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
