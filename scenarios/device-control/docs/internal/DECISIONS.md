# Decisions — Device Control

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| ID | Date | Decision | Revisit Trigger |
|---|---|---|---|
| D-000 | 2026-08-10 | Use the generated `react-vite` scenario documentation contract. | Scenario adopts a different template or doc contract. |
| D-001 | 2026-08-10 | The capability floor is exactly three operations. | A strategy we need cannot satisfy the floor, or every strategy exceeds it. |
| D-002 | 2026-08-10 | Capabilities are probed, never inferred from device kind. | Never — this is the scenario's core honesty property. |
| D-003 | 2026-08-10 | One flow envelope, several step vocabularies. | A vocabulary needs semantics the shared envelope cannot express. |
| D-004 | 2026-08-10 | Target resolution is a ladder, not a strategy choice. | A fourth rung appears, or rung ordering stops holding. |
| D-005 | 2026-08-10 | All inference goes through `ai-gateway`, even though it blocks us. | `ai-gateway` ships a visual-understanding request kind (unblocks; does not reverse). |
| D-006 | 2026-08-10 | Leases are P0, not a later refinement. | Never while any single-session strategy exists. |
| D-007 | 2026-08-10 | The CLI is the contract, because the agent drives it. | Agent mode stops driving the CLI. |
| D-008 | 2026-08-10 | `vrooli-bridge` owns reach; this scenario owns operation. | Bridge changes its fleet model, or a device class fits neither side. |
| D-009 | 2026-08-10 | Recording is a guaranteed capability with a synthesized fallback. | The ramps stop requiring video evidence. |
| D-010 | 2026-08-10 | A capability gap report is a successful response, not an error. | Never — making it an error would hide the scenario's most common honest answer. |
| D-011 | 2026-08-10 | Errors carry identifiers and reasons, never device content. | Never — this is a redaction bypass, not a style preference. |
| D-012 | 2026-08-10 | Every declared capability is exercisable by some flow construct. | Never — an unexercisable capability is an unfalsifiable claim. |

## Decision Details

### D-000 — Generated `react-vite` documentation contract

The scenario scaffold was generated from the template, so docs start with
stubs and maturity metadata in `docs/manifest.json`. Maturity values are
maintained to match actual content in both directions — a doc that
over-claims `active` is as much a defect as one that under-claims `stub`.

### D-001 — The capability floor is exactly three operations

`observe()`, `actuate()`, `describe()`. Everything richer is optional and
declared.

*Why this size:* the floor is the largest set that `ios-mirror` — iPhone
Mirroring driving synthetic HID events with OCR, no accessibility tree at
all — can satisfy. A richer floor would exclude a strategy we need. Shared
capabilities (vision understanding, vision-driven control) are built once
against the floor, which is what makes them work on every strategy rather
than only on the well-instrumented ones.

*Rejected alternative:* require a semantic tree. It would have made flows
simpler and excluded physical iPhone control entirely until Apple Developer
enrollment exists.

### D-002 — Capabilities are probed, never inferred from device kind

A capability that cannot be proven at probe time is `unavailable` with a
named prerequisite and next action.

*Why:* the failure this prevents is a release gate believing a device
validated something it never ran. "Android phones usually have X" is not
evidence about *this* phone right now.

### D-003 — One flow envelope, several step vocabularies

`device.*`, `ai.*`, `flow.*`, `wait.*`, `bas.*` share one envelope, one
timeline, and one evidence record.

*Why not one unified action vocabulary:* device verbs and web-content verbs
are genuinely different domains — tapping a coordinate is not clicking a
selector. But a real automation crosses that boundary constantly (drive
native onboarding, then drive the web content inside the app). Sharing the
envelope makes them composable; merging the vocabularies would make both
worse.

*Consequence:* `browser-automation-studio` remains the authority for web
content on every surface, and this scenario never grows a competing
web-automation engine.

### D-004 — Target resolution is a ladder, not a strategy choice

`semantic` → `visual-anchor` → `vision`, highest supported rung wins, rung
and confidence always recorded.

*Why:* this is the mechanism that makes "shared capabilities work on any
strategy" true rather than aspirational. Vision is the universal fallback;
semantics are the fast, free, deterministic path. The flow author writes
intent once and does not choose — the strategy's declared capabilities
decide. Recording the rung is what lets a reviewer tell a proven result
from an inferred one.

### D-005 — All inference goes through `ai-gateway`, even though it blocks us

`ai-gateway` has no visual-understanding request kind today, so `ai.*` steps
start `unavailable`.

*Why accept the block:* the alternative is the shortcut
`browser-automation-studio` already took — a direct OpenRouter/Anthropic/
Ollama client in `playwright-driver/src/ai/vision-client/` — which is
exactly the coupling `ai-gateway`'s conformance phase exists to flag. Taking
it twice would make the gateway boundary fictional. The gap is declared as a
prerequisite instead.

### D-006 — Leases are P0, not a later refinement

*Why:* several strategies are physically single-session. Without a lease,
two consumers do not collide loudly — they interleave and quietly corrupt
each other's evidence. That is the hardest class of failure to diagnose
after the fact, and retrofitting exclusivity into an executor that assumed
it was free is expensive.

### D-007 — The CLI is the contract, because the agent drives it

CLI completeness is a P0 target rather than a convenience.

*Why:* agent mode spawns an `agent-manager` run driven by a `prompt-manager`
skill, and that skill controls devices through this scenario's CLI. Any
capability reachable only from the API or UI is invisible to the agent, so
CLI parity is a functional requirement, not ergonomics.

### D-008 — `vrooli-bridge` owns reach; this scenario owns operation

A phone is modelled in bridge as an *attached device*: a fleet member that
does not run the bridge agent and is reachable only through a host node.

*Why the split:* bridge's value is being the single answer to "what do I
control and may I." Putting a second device registry here would split that.
But if bridge also held ADB, WebDriverAgent, and mirroring logic it would
become a mobile toolkit, and every new device platform would bloat the trust
plane. Reach and operation are different rates of change.

### D-009 — Recording is a guaranteed capability with a synthesized fallback

Native when the strategy declares it; synthesized from the `observe()` frame
stream otherwise. The evidence always records which path produced the video
and its effective frame rate.

*Why not leave recording optional:* the delivery ramps
(`scenario-to-ios`, `scenario-to-android`) require video evidence for their
conformance journeys. If recording were merely an optional declared
capability, a strategy could legitimately not have it and the ramp would
lose the evidence it exists to produce — on exactly the device kinds
(physical iPhone via mirroring) where the evidence is most valuable.

*Why the fallback is sound:* `observe()` is in the floor, so a frame stream
is available on every strategy by construction. Encoding frames into video
is a local, deterministic operation. This is the same "build the shared
capability once against the floor" pattern as vision — applied to evidence
production instead of control.

*What it must never do:* label a synthesized capture as native, or omit the
effective frame rate. A 2 fps reconstruction and a 60 fps native capture are
both legitimate evidence, but they support different claims, and a reviewer
must be able to tell them apart without opening the file.

### D-010 — A capability gap report is a successful response, not an error

Asking "can this flow run on this device?" and getting back "no, and here is
exactly which capabilities are missing" is a **successful** call carrying a
structured gap report. Only *dispatching* a flow despite a known gap is an
error.

*Why:* "this device cannot do X" is the single most common honest answer this
scenario gives. Modelling it as an error would make the normal case
indistinguishable from a broken request, and would force every caller to
parse error strings to recover structured data the API already holds.

*Rejected alternative:* raise on any unsatisfiable request. It reads as
stricter and is actually weaker — callers end up catching broadly and losing
the per-capability detail that makes the gap report useful.

### D-011 — Errors carry identifiers and reasons, never device content

No error message, log line, or CLI output may contain frame bytes, capture
paths, text read from a device screen, clipboard values, device logs, or
credentials.

*Why:* errors do not pass through `EvidenceSink`, so they never receive
redaction verification. That makes the error channel the one path by which
device content could leave unchecked — a redaction bypass rather than a
formatting preference.

*Rejected alternative:* echo what the resolver searched for and what it saw.
This is the *natural* implementation and the reason the rule needs writing
down: `could not find target "password" in frame; visible text was "Enter
code 481920"` is a genuinely helpful debugging message and a credential leak
in the same sentence. Naming the target identifier and the per-rung outcome
is strictly more useful for debugging and carries nothing off the device.

### D-012 — Every declared capability is exercisable by some flow construct

No capability may be declarable unless something in the flow vocabulary
behaves differently because of it, and no flow construct may require a
capability the registry does not list.

"Exercisable" rather than "has a verb", because there are three legitimate
mechanisms and only the first is a step requirement:

| Mechanism | Example |
|---|---|
| A step **requires** it | `device.install` requires `app-lifecycle` |
| A resolution **rung** requires it | the `semantic` rung requires `semantic-tree` |
| A step's execution **path** selects on it | `device.record` uses `native-recording` when declared, synthesis otherwise |

The first draft of this decision said "reachable by at least one step verb",
and verifying it immediately flagged `semantic-tree` and `native-recording` as
violations — both of which are perfectly legitimate. The rule was too narrow,
not the registry. Recorded here because the narrow version is the obvious one
to reach for again.

*Context:* assembling [`../reference/capabilities.md`](../reference/capabilities.md)
surfaced five capabilities — `network-control`, `orientation`, `clipboard`,
`file-transfer`, `device-logs` — that a strategy could declare and probe but
that no step could ever use. `device.network`, `device.orientation`,
`device.clipboard`, `device.push` / `device.pull`, and `device.logs` were added
to close the gap.

*Why the rule rather than just the fix:* an unreachable capability is an
**unfalsifiable claim**. It can be declared, it can be probed, it can appear
`available` in a report — and nothing can ever exercise it to prove the
declaration was true. That is the same class of confident-but-empty statement
`D-002` refuses when it insists capabilities are probed rather than inferred,
arriving by a different route. The registry is only trustworthy if every row
in it is exercisable.

*Rejected alternative:* drop the five capabilities as aspirational. Cheaper,
and wrong for two of them — `file-transfer` and `device-logs` are load-bearing
for the delivery ramps, which need to pull artifacts and diagnostics off a
device after a conformance journey.

*Consequence:* adding a capability now requires adding the construct that
exercises it in the same change. The reverse also holds — a proposed construct
whose capability is not in the registry is a signal that the registry is
incomplete, which is how `webview-attach` was found.

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`SEAMS.md`](SEAMS.md) — the seams these decisions imply
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
