# Voice Latency

This document records the browser voice-runtime contracts referenced by the
audio integration hooks.

These are design constraints, not measured SLO achievements. PRD OT-P0-005
owns the responsive-streaming obligation; [PERFORMANCE.md](PERFORMANCE.md)
defines clocks and denominators, and [TESTING.md](TESTING.md) owns the pending
numeric bands and paced-device qualification. Measure the user-visible path
before choosing prewarming, caching, or buffering changes.

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

## Audio ducking deep dive

Duck playback while speech capture is active. Restore the prior playback level
when the session ends or fails.

## Stream injection vs stream acquisition

Injected test audio and microphone acquisition share the same downstream PCM
pipeline. Only the source differs.

## Persistent noise floor cache

Keep the estimated noise floor for the active audio context. Reset it when the
input device or sample rate changes.
