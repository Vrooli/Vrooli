# Phone Agent — Meta-Orchestrator Summary

## Source

Brainstorming session (2026-04-22) about generalizing the existing morning-vision-walk skill into a phone-callable agent. Trigger was the existing `continuous-audio-platform` initiative's audio extraction work plus the unwired Twilio resource — the user wondered whether those two together unlocked a "the phone calls me for my vision walk" experience.

## Vision

A phone- and browser-accessible agent assistant. Bridges a single voice call (browser mic OR Twilio phone number) to per-turn agent-manager runs. Routines (recurring or ad-hoc) are registered against the system; first routine is morning-vision-walk, but the architecture is intentionally routine-shaped (standups, decision readouts, ideation calls, etc. follow the same pattern).

The morning-vision-walk skill stays where it is — it just becomes one registered routine, not the whole product.

## Key Architectural Decisions

### 1. Transport-pluggable voice-session from day one

The voice-session abstraction in audio-tools must be a first-class interface, not browser-WebSocket-with-Twilio-bolted-on-later. Concrete transports that need to compose against it:

- `browser-voice` — ships with the audio-tools greenfield (was already planned as the WebSocket streaming transcription endpoint; we are just declaring it as a transport instead of "the only shape")
- `twilio-voice` — ships separately under the phone-agent initiative (`execute/audio-tools-twilio-media-stream-transport`)
- Future: SIP, WebRTC SDK, etc.

**Why this matters:** retrofitting transport pluggability after the greenfield ships would be expensive. Doing it on day one is cheap if it's part of the contract from the start. This is the reason the two existing items in `continuous-audio-platform` were updated rather than left alone — the greenfield needs to land in the right shape.

### 2. Agent-manager is the runner, not web-console's subprocess spawner

Each user utterance maps to one agent-manager run with rolling conversation history packed in. Reasons:

- Voice turns are conceptually discrete agent invocations — agent-manager's run model is a natural fit
- Sandboxing, profile-based permissions, and append-only event streams come for free
- Web-console's spawner is tuned for an interactive TUI — using it for voice means parsing ANSI, handling DA/DSR probes (see web-console claude-hang memory), etc. Wrong abstraction.
- Agent-manager's event stream is already the right shape for token-level deltas → TTS

**Caveats** (called out in the routine-and-runner research item):
- agent-manager multi-phase runs / sessions are P2 (OT-P2-001), so phone-agent owns the rolling history and re-packs it each turn
- Real-time event streaming (WebSocket) is P1 (OT-P1-007); confirm this is on the critical path
- Token-level assistant deltas may need a follow-up fix — research will determine

### 3. Browser-first product, phone-second

The phone-agent UI must be useful with no Twilio infrastructure at all. `browser-voice` is the default transport; `twilio-voice` is incremental. This means:

- A useful product ships before a phone number is purchased, before TwiML is wired, before resource-twilio grows a voice surface
- Twilio work happens in parallel and lights up the phone channel when ready
- Ops debugging is free: live transcript + waveform UI is the same surface used for non-phone calls

The phone-agent greenfield item (`execute/phone-agent-greenfield-scenario`) deliberately does NOT depend on the Twilio transport — only on audio-tools greenfield + the routine/runner research.

### 4. Duplicate before extract — keep the routine registry local

Per existing project preference (see `feedback_duplicate_before_extract`), the routine registry stays inside phone-agent, not extracted into a shared package. If a future scenario also needs routines, copy first, extract later as a dedicated scenario.

### 5. One active transport per call

If a phone call is active and the browser UI opens, the browser is observer-only (muted mic). No simultaneous mics on the same session — AEC between two live mics is a rabbit hole with no payoff. The pub/sub event stream design is what makes the observer use case cheap (UI just subscribes, doesn't try to participate in audio).

## New Upstream Dependencies (added 2026-04-22)

Phone-agent now also depends on two initiatives from the conversational-surface brainstorming session on 2026-04-22:

- **`agent-inbox-unified-retrieval`** — the dual-track (agent-led + passive) retrieval layer that lets a voice call `just ask and the system finds the right tool/command/widget`. Voice has no tool-picker UI, so auto-mode retrieval is effectively required for phone-agent to be useful to non-technical users.
- **`widget-standard`** — the visual surface for inline scenario UI sections. Phone-agent is voice-first but not voice-only: when a user is on a browser voice call (or reviewing a phone call post-hoc), rendering the relevant widget inline is the bridge from conversational intent to actionable UI.

Phone-agent's existing `continuous-audio-platform` dependency is unchanged. The two new dependencies are *not* blocking for the greenfield scenario itself — phone-agent can ship a usable v1 before auto-retrieval lands, falling back to the current `enable specific tools per chat` UX. But the frictionless `open the app and start talking` vision the initiative was scoped around needs them.

Revisit sequencing during workshop: if auto-retrieval research lands fast, pull it into phone-agent's critical path. If it lags, phone-agent greenfield ships with the current retrieval behavior and upgrades in a follow-up.

## Cross-Initiative Updates Made in This Session

Two items in `continuous-audio-platform` were updated to support phone-agent's needs:

1. **`research/audio-tools-shared-scenario-contract`** — appended research goal #5 covering transport-pluggable voice-session, pub/sub event stream, barge-in signal contract, and resampler scope decisions. Added 4 new deliverables.

2. **`execute/audio-tools-greenfield-scenario`** — added "Voice-Session Architecture (transport-pluggable)" as the lead P0 section. Added incremental TTS input requirement, voice-session subscription API, transport plugin layer, and resamplers as P0.

If workshop on either of those items has already advanced when phone-agent items reach workshop, reconcile against the updated scope.

## Dependency Reasoning

```
continuous-audio-platform (existing)
   │
   ▼
phone-agent (new, this initiative)
   │
   ├── research/phone-agent-routine-and-runner-model  (no upstream — start now)
   ├── execute/resource-twilio-voice-surface          (no upstream — start now, parallel)
   ├── execute/audio-tools-twilio-media-stream-transport  (waits on audio-tools greenfield + twilio voice)
   ├── execute/phone-agent-greenfield-scenario        (waits on routine research + audio-tools greenfield)
   └── execute/phone-agent-morning-vision-walk-routine  (last — needs phone-agent greenfield)
```

Two workstreams can move in parallel before audio-tools greenfield lands: the routine/runner research and the resource-twilio voice surface buildout. Everything else gates on the audio-tools greenfield finishing.

## Explicitly Out of Scope (v1)

These were considered and rejected for the first cut. Workshop should not pull them back in without an explicit reason.

- **Observer-mode participation via typing during a phone call** — adds an audio+text turn-ordering problem; nice to have, defer
- **Simultaneous phone + browser audio on the same call** — AEC hell, no payoff
- **Multi-user ACL on call transcripts** — single-user trust model is fine for v1
- **Generic browser voice agent for non-phone use cases** — that is a different scenario if/when it's needed; do not let phone-agent grow into it

## Operational Targets (for the greenfield item)

Captured in `execute/phone-agent-greenfield-scenario` description; reproduced here for quick reference:

- Call lifecycle: inbound answer, outbound dial, disconnect/reconnect with context resume
- Transport selector UI (browser or phone)
- Per-turn agent-manager runs with rolling history
- Token-level event → TTS streaming
- Per-call state persistence (transcript, history, outcomes)
- Barge-in coordination (VAD → cancel TTS + cancel in-flight run)
- Routine registry + scheduled trigger integration (uses existing schedule surface)
- UI: live transcript, waveform (browser-voice), call controls, routine picker, call history
- Caller-ID allow-list
- Per-call billing/cost emission

## Open Questions Deferred to Workshop

- Where exactly the resampler utilities live — shared audio-tools package vs. per-transport. (Surfaced in research/audio-tools-shared-scenario-contract goal #5; greenfield assumes shared but workshop should confirm.)
- Whether the existing 5am director-swarm `vision-walk-prep` agent becomes a routine pre-hook or stays standalone with the routine just consuming `last-handoff.md` as initial context. (Surfaced in research/phone-agent-routine-and-runner-model.)
- How call billing events join with `continuous-audio-platform`'s billing/cost surface — that surface itself is still being shaped via `research/audio-provider-routing-contract`, so phone-agent should follow rather than predefine.
- Whether agent-manager's run event stream already exposes token-level assistant deltas, or whether a follow-up fix item is needed. (Surfaced in research/phone-agent-routine-and-runner-model goal #2.)
