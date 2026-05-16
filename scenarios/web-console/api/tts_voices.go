package main

import inttts "web-console/internal/tts"

// TTSVoiceLister is the testability seam for voice listing.
// DOC: docs/internal/SEAMS.md#tts-voice-lister-seam
type TTSVoiceLister = inttts.VoiceLister

// TTSVoice represents an available TTS voice.
type TTSVoice = inttts.Voice

// KokoroVoiceLister fetches available voices from a Kokoro-FastAPI instance.
type KokoroVoiceLister = inttts.KokoroVoiceLister

// HTTP handler for /api/v1/tts/voices moved to handlers/tts. The voice-list
// fetch/validation now lives in internal/tts.Service.
