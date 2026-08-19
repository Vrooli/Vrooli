# Audio Tools experience contract

This directory is the authored UX contract for the Audio Tools operator
surface. It describes what the UI communicates; it does not replace the
browser workflow cases or the server-owned qualification evidence.

Dictation Studio is the primary voice-input surface. Its contract deliberately
distinguishes preparation, active capture, transcription, bounded recovery,
captured audio, and terminal failure. A passing workflow or a visible button
is not treated as proof that the audio was durably processed.

The contract is intentionally schema version 1.0.0 while the UI continues to
use the existing product selectors and diagnostic attributes. Claims about
long-form reliability remain earned by the audio-tools qualification artifacts
under `coverage/`.
