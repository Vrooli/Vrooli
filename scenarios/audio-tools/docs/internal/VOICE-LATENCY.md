# Voice Latency

This document records the browser voice-runtime contracts referenced by the
audio integration hooks.

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
