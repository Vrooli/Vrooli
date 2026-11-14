---
name: audio-tools-improve
description: "Close Audio Tools' contract and evidence gaps within an authorized development mandate: measure real voice outcomes, route the highest broken layer to its owner, and retain unknown device, streaming, quality, and billing results."
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [audio, improvement, streaming, reliability, evidence, monetization]
  status: active
  revision: 4
  requires:
    scenarios: [audio-tools, program-runtime, test-genie, business-health]
    commands: [program-runtime library run, audio-tools, business-health, vrooli scenario test, prompt-manager skill read]
  origin:
    kind: authored
---
## Practice focus: Audio Tools improvement

Close the highest broken contract, obligation, evidence, or implementation layer
for the authorized voice target. Preserve unknown results until the relevant
product path has been measured.

When Audio Tools is the target of a Swarm development grant on a mandate plan,
run in goal or sliced mode, keep the Swarm approval, campaign checkpoint and
aggregate grant as the engagement identity. This skill supplies Audio's outcome judgment and owner routes; it does
not approve the target, launch a private branch, or convert a workflow result
into product acceptance.

### Scope and required reading

Read `path:docs/agent-system/SCENARIO_DEVELOPMENT.md`, then
`prompt-manager skill read scenario-work-ladder` and
`prompt-manager skill read improvement-do-and-dont`. The improvement loop
itself belongs to `prompt-manager skill read scenario-improvement-campaign`;
this skill supplies only Audio's sensors, priorities, routes and stop rules.
Read `path:scenarios/audio-tools/PRD.md`,
`path:scenarios/audio-tools/docs/internal/TESTING.md`, and prior findings in
`path:scenarios/audio-tools/docs/internal/PROBLEMS.md`.
Use TESTING.md's full-mandate/completion contract and published-target mapping.
Read the owning architecture, performance or monetization document when selecting
work in that boundary; do not infer scope from the local-only proposal specimen.

The mandate determines write authority. In-scope repairs stay in that engagement;
this skill does not create one backlog item per repair. A missing or broken
Audio-owned sensor, resolver, program, test, or evidence join is itself an
in-scope repair: implement it, validate it, and continue to the next row in the
same mandate. Observation-only work returns findings only when the caller has no
authority to repair the owner. Shared capture, resource, and monetization
changes require those owners in the change boundary. Commercial decisions do
not follow from a sensor.

### Setpoint

| Row | Source of target | Interpretation |
| --- | --- | --- |
| dictation-trust | PRD OT-P0-001 | Every captured interval is processed, retained, or explicitly recoverable |
| provider-parity | PRD OT-P0-002 | Same trust floor for stable engines; Whisper and Kyutai are initial independent candidates, not permanent architecture |
| speaker-policy | PRD OT-P0-003 | Applied/degraded state and required-policy fail-closed behavior |
| provider-evidence | PRD OT-P1-001 | Provider-neutral persisted cells and verdicts |
| mobile-recovery | PRD OT-P1-002 | User-accessible recovery and metadata diagnostics |
| device-qualification | PRD OT-P2-001 | Named native-device receipts; simulation remains separate |
| interactive-latency, corpus-quality, owned-settlement | PRD OT-P0-005, OT-P0-006, OT-P0-007; TESTING.md and `docs/business/MONETIZATION.md` | Numeric bands, corpus recipe and billing policy v1 adopted 2026-09-13; interactive readiness includes press-to-first-accepted-speech |
| explicit-routing, host-portability, private-lifecycle, acceptance-integrity | PRD OT-P0-004, OT-P0-008, OT-P0-009, OT-P0-010 | Required; owner-backed sensors are still absent |
| adapter-maintainability, shared-voice-capabilities | PRD OT-P1-003, OT-P1-004 | P1-003 required; P1-004 excluded as expansion, no regression permitted |

Bands, grant scope and excluded rows live in TESTING.md's
[outcome evidence inventory](../../docs/internal/TESTING.md#outcome-evidence-inventory);
that table is the finish line. The board's target strings summarize scope; the
inventory holds the bands. W2 work must still add owner-backed joins that compute
`in_band` against those bands. Do not equate target enumeration, engine count or
replay metrics with acceptance. Keep unmeasured outcomes unknown; never fill one
with a zero or a passing band. Re-read the board each cycle.

For interactive latency, warm readiness means the user-visible microphone
contract: after the user presses the microphone control with permission already
granted, the capture/transcription path must accept the first voiced sample
within 500 ms at p95. Measure through first accepted speech, not merely button
handling, permission resolution, WebSocket creation, or a readiness banner. A
multi-second delay before speech is accepted is an interactive-latency defect;
reproduce it, instrument the milestones, repair the owner path, and remeasure
the authorized browser cohorts.

Treat microphone reuse as a first-class reliability contract. After one session
is stopped, the next session must be able to start, accept voiced input, produce
the expected transcription, and stop cleanly without a page refresh, app restart,
or manual recovery. Verify at least five consecutive warm press/speak/transcribe/
stop cycles in the same page and browser context, with at least one second between
stops and starts. A failure on any cycle is a failed run: preserve the cycle
number, lifecycle and transport milestones, visible error state, and whether a
refresh or restart was required; then trace and repair the owner path before
claiming repeated-session acceptance.

Treat the microphone control as the primary user-facing sensor for this
contract. In timeout/auto-stop mode it must expose the remaining silence
timeout as control progress; in always-on/persistent mode it must expose
listening and voice-activity state without pretending that a one-shot timeout
exists. Preparation, active listening, partial-text delay, and final draining
must be represented by the control and adjacent status text. Do not use
transient banners for normal progress or latency warnings; reserve banners for
actionable persistent failures, permission/privacy decisions, and recovery.
Exercise both the first cycle and repeated cycles, preserve every failure, and
repair the owner path when the control remains disabled or requires refresh.

### Sensors

Run `audio-tools.setpoint-read` through `program-runtime library run` first.
Use a specific `experiment_id` to examine persisted measurements; omit it for
inventory only. The contract owns input, envelope, and error vocabulary.
For authoritative obligation/evidence state, read `business-health matrix show
audio-tools --format summary`. Run `measures-health validate scenario audio-tools`
for instrumentation structure, not voice quality. Read scoped binding condition
only when the board reports transport trouble. Do not make unrelated fleet
telemetry a prerequisite for a local repair.

### Golden corpora and curation

Use TESTING.md's corpus lanes, cohort recipe, device profiles, and negative
controls. Inspect `audio-tools corpus list` before importing authorized clips.
Keep consent/license, reference revision, content hashes, and realized IDs with
the experiment. Add a regression for an observed failure without replacing the
qualification denominator. Retire superseded cohorts by explicit version and
reason; retain historical receipts. Synthetic tones qualify transport arithmetic,
not speech recognition. Never import private recordings as shared fixtures.

### Routes and implementation order

| Highest broken layer / observation | Actuator and completion evidence |
| --- | --- |
| W0: target or commercial choice unresolved | `prd-authoring` for authorized charter amendments; the broader scope is already published. Resolve the specific pending SLO/cohort/policy decision without restarting charter generation or inventing prices/device support |
| W1: selected contract has no obligation | Owning domain/experience contract; add the falsifiable obligation and its planned validation before implementation |
| W2: missing, stale, or misleading evidence | `requirements-traceability-steer` for links; `measures-adoption` and the owning test method for missing instruments. Prove negative controls before treating its result as acceptance |
| W3: discovery delay, missing partials, lost tail, recovery failure | `scientific-debugging`; measure boundaries, change the owner, run focused regressions and the affected paced product path |
| W3: simulated entitlement works but hosted provider is absent | Shared AI delivery and LPBS owners; join delivery to entitlement/settlement evidence without claiming the fake qualifies the hosted service |
| Repeated agent rediscovery / manual aggregation | Fix this usage skill or the board; read Program Runtime's construction guides before program changes |
| Program accumulates retry, pricing, or session-state logic | Move the invariant to its scenario owner, then remove the workaround from program and skill |

Start with local discovery and paced shared-consumer streaming. Add route and
settlement simulations next. Qualify live providers and native devices only under
their reviewed scope and spend allowance. This sequence is not a requirement to
restart completed work or to wait for billing decisions before repairing local STT.

For the active portable-voice mandate, the authorized qualification lane is
deliberately local and simulated: build or curate the approved open-licensed
`voice-pilot-en-v1` corpus, qualify the local D1 host, exercise the constrained D2
cell, and use the LPBS/fake-provider route and settlement fixtures. Do not treat
the absence of private recordings, nominated physical D3-D6 devices, live BYOK or
owned-provider keys, or paid allowance as a blocker for this mandate; those inputs
are outside its grant. Produce owner-backed receipts from the authorized local
and simulated lanes, label simulation honestly, and continue repairing the next
in-scope row. Only stop for an external prerequisite when the required action is
outside this grant and no authorized local or simulated lane can satisfy it.

### Anti-gaming

Apply `improvement-do-and-dont` D1 (weakened test), D2 (deleted ledger), and D3
(suppression), and its §2 skeptic test. Keep failed attempts, skipped cells, early EOF, missing receipts,
and stale cohorts visible. Reject a successful wrapper whose required child
failed. Never broaden a latency band to fit the latest run, declare a fake native,
or turn an unmeasured row green. A transcript-free diagnostic is not a privacy
waiver for the underlying corpus.

### Evidence

Retain before/after readings, target revision, build/provider/corpus/device
identity, run references, negative controls, and remaining unknowns in the active
engagement. Update the existing owner docs when the contract changes. Under the
mandate, keep one journal at
`~/.vrooli/plan-artifacts/audio-tools-portable-voice-mandate/journal.md` for
readings, hypotheses and rejected hypotheses; checkpoints go through Plan
Manager, and completed work through the shared Memory work-record path. Keep no
other ledger.

### Stop rules

**Stop a repair** when its scoped acceptance and regression evidence pass, then
choose the next gap from the board. **Stop the mandate** with `complete` only when
every required row in the inventory reads in band from current owner-backed
evidence and the evidence audit passes; all readable rows green is insufficient.
Return `blocked` only for a decision, credential or approval you lack, and name
it: new authority, paid spend, live keys, private data, a native device, or a
product promise. A row that reads unavailable is journaled, not estimated, and
is not by itself blocked. For this mandate, private data, live keys, paid spend,
and physical D3-D6 devices are not required inputs and must not be named as
blockers; use the approved public corpus, local D1/D2 cells, and simulated route
fixtures. Missing instrumentation, an absent resolver, an unimplemented join, a
failing owner binding, an empty sensor, or an unbuilt local qualification harness
is work, not a blocker, when its owner is inside the grant. A pending commercial
sub-parameter does not block unrelated authorized local improvements.

### Troubleshooting & Edge Cases

For operation-level failures, read `prompt-manager skill read audio-tools`.
If a sensor is absent, route instrumentation work to its owner and implement it
when that owner is in the mandate boundary; do not hand-grade an unknown outcome
or hand the row back as blocked. If a program fixture fails while direct CLI reads succeed,
inspect the governed response and transport classification before changing the
product. Use Test Genie's server-owned wait once for a running validation; do not
poll, rerun, or weaken it merely because it is slow.

### Shared browser admission

The long-form soak requests a fresh BAS session, so BAS sessions belonging to
other workflows are not blockers: BAS provides isolated execution contexts and
the soak preflight must admit the run whenever the driver is ready and has spare
capacity. Do not wait for, release, or force-close a foreign session merely to
obtain a zero-session count. A true capacity exhaustion or driver-unready result
is a retryable operational condition; diagnose it and continue after capacity is
available. A stale orphan may be repaired only through BAS's owner-controlled
recovery path, with liveness and ownership evidence recorded first.
