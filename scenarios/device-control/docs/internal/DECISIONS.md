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

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-10 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history

## Durable decisions taken during initialization

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
