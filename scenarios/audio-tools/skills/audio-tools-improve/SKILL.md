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
  revision: 2
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

### Scope and required reading

Read `path:docs/agent-system/SCENARIO_DEVELOPMENT.md`, then
`prompt-manager skill read scenario-work-ladder` and
`prompt-manager skill read improvement-do-and-dont`.
Read `path:scenarios/audio-tools/PRD.md`,
`path:scenarios/audio-tools/docs/internal/TESTING.md`, and prior findings in
`path:scenarios/audio-tools/docs/internal/PROBLEMS.md`.
Use TESTING.md's full-mandate/completion contract and published-target mapping.
Read the owning architecture, performance or monetization document when selecting
work in that boundary; do not infer scope from the local-only proposal specimen.

The mandate determines write authority. In-scope repairs stay in that engagement;
this skill does not create one backlog item per repair. Observation-only work
returns findings. Shared capture, resource, and monetization changes require those
owners in the change boundary. Commercial decisions do not follow from a sensor.

### Setpoint

| Row | Source of target | Interpretation |
| --- | --- | --- |
| dictation-trust | PRD OT-P0-001 | Every captured interval is processed, retained, or explicitly recoverable |
| provider-parity | PRD OT-P0-002 | Same trust floor for stable engines; Whisper and Kyutai are initial independent candidates, not permanent architecture |
| speaker-policy | PRD OT-P0-003 | Applied/degraded state and required-policy fail-closed behavior |
| provider-evidence | PRD OT-P1-001 | Provider-neutral persisted cells and verdicts |
| mobile-recovery | PRD OT-P1-002 | User-accessible recovery and metadata diagnostics |
| device-qualification | PRD OT-P2-001 | Named native-device receipts; simulation remains separate |
| interactive-latency, corpus-quality, owned-settlement | PRD OT-P0-005, OT-P0-006, OT-P0-007; TESTING.md and MONETIZATION.md | Published scope; numeric bands, corpus and billing policy still require adoption |
| Explicit routing, host compatibility, privacy, full receipt coverage | PRD OT-P0-004, OT-P0-008, OT-P0-009, OT-P0-010 | v2 lists the required outcomes; owner-backed sensors are still absent |
| Adapter maintainability and other voice operations | PRD OT-P1-003, OT-P1-004 | Governed shared contracts and operation-specific evidence; identify explicitly which expansion is in the mandate |

The v2 board names all 15 PRD targets and declares `portable-voice-v1`. It preserves
the nine original row IDs and adds six; target summaries are not numerical bands.
Build the required inventory from the mandate and registry and verify program
coverage as that contract evolves. W2 work must still add owner-backed joins.
Do not equate target enumeration, engine count or replay metrics with acceptance.
Preserve pending decisions and unmeasured outcomes separately; never fill either
with a zero or a passing band.
The 2026-09-08 setup observation is nine unknown outcome rows, with live inventory
and selected replay metrics readable. See PROBLEMS.md for run references. Re-read
the board each cycle; this dated observation is not a cached acceptance decision.

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
engagement. Update the existing owner docs when the contract changes. Use the
shared Memory work-record path; avoid a duplicate campaign ledger.

### Stop rules

Stop a repair when its scoped acceptance and regression evidence pass. Continue
to other authorized gaps under a development mandate. Declare the whole target
complete only when every required cohort has current accepted evidence; all
readable rows green is insufficient. Request a decision when the next action
needs new authority, paid spend, private data, or a product promise. A pending
commercial decision does not block unrelated authorized local improvements.

### Troubleshooting & Edge Cases

For operation-level failures, read `prompt-manager skill read audio-tools`.
If a sensor is absent, route instrumentation work to its owner; do not hand-grade
an unknown outcome. If a program fixture fails while direct CLI reads succeed,
inspect the governed response and transport classification before changing the
product. Use Test Genie's server-owned wait once for a running validation; do not
poll, rerun, or weaken it merely because it is slow.
