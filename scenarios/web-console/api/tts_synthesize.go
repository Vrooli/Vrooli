package main

import inttts "web-console/internal/tts"

// TTSSynthesizer is the testability seam for TTS synthesis.
// DOC: docs/internal/SEAMS.md#tts-synthesizer-seam
type TTSSynthesizer = inttts.Synthesizer

// SynthesizeRequest holds parameters for a TTS synthesis call.
type SynthesizeRequest = inttts.SynthesizeRequest

// KokoroSynthesizer proxies synthesis requests to a Kokoro-FastAPI instance.
type KokoroSynthesizer = inttts.KokoroSynthesizer

const maxSynthesizeInputLength = 5000

// HTTP handler for /api/v1/tts/synthesize moved to handlers/tts. The
// validation, default-application, and cache-on-write logic now live in
// internal/tts.Service.
