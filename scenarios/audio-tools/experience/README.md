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

The `portable-voice-v1` PRD adds explicit route/cost consent, genuine partials,
final drain, private recovery and evidence-scoped device support. These authored
priorities are desired behavior, not claims that the UI or hosted route already
implements them. Existing active/draft status is not a passing UX certificate.

The voice control and its surrounding consumer surface must distinguish idle,
permission/preparation, ready/listening, partial text, committed text, draining,
recovery and refusal. Show the actual local/BYOK/owned destination and whether
streaming is genuinely incremental. Never use an animation, credit count or
entitlement fixture to imply successful transcription or settled delivery.

Qualify the same experience in Audio Tools and each claimed adopter using the
shared capture contract. A component-library specimen proves its narrow visual
claim, not microphone capture, provider delivery, final-tail latency or billing.
