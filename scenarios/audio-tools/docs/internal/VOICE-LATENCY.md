# Voice Latency

This document records the browser voice-runtime contracts referenced by the
audio integration hooks.

These are design constraints, not measured SLO achievements. PRD OT-P0-005
owns the responsive-streaming obligation; [PERFORMANCE.md](PERFORMANCE.md)
defines clocks and denominators, and [TESTING.md](TESTING.md) owns the pending
numeric bands and paced-device qualification. Measure the user-visible path
before choosing prewarming, caching, or buffering changes.

## Microphone readiness SLO

For a warm session with microphone permission already granted, measure from the
user's microphone-button press until the capture/transcription path accepts the
first voiced sample. This press-to-first-accepted-speech interval must be at
most 500 ms at p95. Do not end the clock at button handling, permission
resolution, stream creation, WebSocket connection, or a UI readiness label;
those are diagnostic milestones. A multi-second delay before spoken input is
accepted is a failed interactive-latency observation requiring an owner-backed
trace and repair.

## Repeated-session reliability

After a session stops, microphone reuse must work in the same page and browser
context without a refresh, app restart, or manual recovery. Qualification runs
must complete at least five consecutive warm press/speak/transcribe/stop cycles;
each cycle must accept voiced input, produce the expected transcription, and
finish its final drain. Record every cycle's lifecycle, transport, transcription,
and error milestones. One failed cycle fails the sequence and requires an
owner-backed trace and repair.

## Audio cue contract

Audio cues acknowledge recording, completion, and recoverable failures. They
must not mask transcription state or replace accessible status text.

## Visibility-based mic lifecycle

Pause microphone capture when the document becomes hidden. Re-check permission
and device readiness before capture resumes.

## Pre-create AudioContext on first gesture

Create or resume the audio context from the first user gesture. Browser autoplay
policy can otherwise delay audio feedback and capture analysis.

## Background capability check

Check provider capability away from the latency-sensitive capture path. Surface
the result before recording starts.
Bound refresh work and report stale/unavailable results. Background preparation
must not record audio, download a model, reserve paid capacity or transmit data
without the relevant permission; cached health is not a successful session.

## WebSocket pre-connection

Open the browser-side stream transport only when the selected provider supports
streaming. The provider must still defer resource admission until audio arrives.

The warm capture path also caches the AudioWorklet module load per
`AudioContext`. Rebuilding a Blob URL and awaiting `addModule()` on every
microphone press is part of the button-to-first-PCM interval; a rejected load
is evicted so later turns retain the tested ScriptProcessor fallback.
When AudioWorklet is unavailable or unscheduled, that compatibility fallback
uses a 1024-sample callback buffer. At the canonical 16 kHz wire rate this
bounds the first callback to roughly 64 ms; the downstream 50 ms host wire
cadence, exact sample cursors, and durable journal contract remain unchanged.
On the actual voice gesture, the shared core now starts that cached module load
immediately after creating/resuming the context, overlapping it with
`getUserMedia()` and journal restoration. This does not acquire or retain a
microphone before the gesture, and webdriver fake-media keeps its immediate
ScriptProcessor path.

The standalone Dictation Studio recorder follows the same gesture-scoped warm
path: the microphone-button handler resumes the shared `AudioContext` and
starts loading the canonical capture worklet before awaiting microphone
acquisition, journal restoration, or provider startup. This keeps setup off the
first-PCM critical path while preserving the browser gesture requirement and
the ScriptProcessor fallback.

For the audio-tools host, the shared PCM provider also sends a bounded 20 ms
first wire batch before returning to the normal 100 ms batches. This preserves
exact sample cursors and durable replay while allowing a native decoder to begin
work before the steady-state batching timer fires. The package default remains
the conservative 100 ms first batch for other hosts until they opt into the
host-specific policy.

## Audio ducking deep dive

Duck playback while speech capture is active. Restore the prior playback level
when the session ends or fails.

## Stream injection vs stream acquisition

Injected test audio and microphone acquisition share the same downstream PCM
pipeline. Only the source differs.

## Persistent noise floor cache

Keep the estimated noise floor for the active audio context. Reset it when the
input device or sample rate changes.

## Voice-control behavior contract

The microphone control is the primary status surface. Its accessible label and
visual state must distinguish preparing, ready/listening, receiving partial
speech, stopping/final-draining, completed, and recoverable-error states.

- Warm permission-granted capture must become ready to accept speech within
  500 ms at p95. First visible revisable text must arrive within 1 s at p95.
  The browser diagnostic anchors speech-to-visible-partial timing to the first
  captured PCM sample above the signal threshold, not to an earlier device
  warm-up or leading-silence callback. Capture-start and first-voice timestamps
  remain separate milestones in traces.
- In timeout/auto-stop mode, show the remaining silence timeout as progress on
  the control itself. In always-on/persistent mode there is no single timeout
  countdown; show listening and voice-activity state instead. Do not imply an
  auto-stop promise in persistent mode.
- Normal preparation, listening, partial-text latency, and final-drain status
  belong in the control and its adjacent status text, not transient banners.
  Use a banner only for an actionable persistent failure, privacy/permission
  decision, or recovery action. A slow-but-active request is not a warning
  banner.
- A stop must settle the final tail before the control returns to idle. The
  same page and browser context must then support at least five consecutive
  press/speak/transcribe/stop cycles, with at least one second between cycles,
  without refresh, app restart, or manual recovery.
- If a cycle fails, preserve its lifecycle and transport milestones and expose
  a retry/recovery action. Never silently leave the control disabled or require
  a refresh as the only recovery path.
