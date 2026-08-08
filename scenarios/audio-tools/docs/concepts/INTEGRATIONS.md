# Integrations — Audio Tools

> Plan: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`

## Resource dependencies (`.vrooli/service.json`)

| Resource | Required | Used by | Degraded behavior |
|---|---|---|---|
| `whisper` | false | Local STT (`internal/voice.Service.Transcribe`) | Local STT tier reports unavailable; chain falls to BYOK/Vrooli or errors. |
| `kokoro` | false | Local TTS (`internal/tts.Service.Synthesize`) | Local TTS tier reports unavailable; chain falls to BYOK or errors. |
| `ollama` | false | Local summarize (`internal/tts.Summarizer`) | Local summarize tier reports unavailable; chain falls to BYOK or errors. |
| `postgres` | false | Optional usage backend (SQLite is default) | Usage rows store in SQLite. |

Every resource is `required: false`. audio-tools starts cleanly with zero local resources — BYOK-only operation is supported.

## Scenario dependencies (`.vrooli/service.json`)

| Scenario | Required | Used by | Notes |
|---|---|---|---|
| `landing-page-business-suite` | false | Vrooli tier in all three provider chains | Disabled by `AUDIO_AI_ENABLE_VROOLI=false` until `execute/lpbs-audio-gateway-endpoints` ships. |

## Outbound third-party services (BYOK)

| Adapter | Capability | Endpoint | Credential header |
|---|---|---|---|
| `openai-whisper` | STT | `https://api.openai.com/v1/audio/transcriptions` | `Authorization: Bearer <X-Audio-BYOK-Key>` |
| `deepgram` | STT | `https://api.deepgram.com/v1/listen` | `Authorization: Token <X-Audio-BYOK-Key>` |
| `openai-tts` | TTS | `https://api.openai.com/v1/audio/speech` | `Authorization: Bearer <X-Audio-BYOK-Key>` |
| `elevenlabs` | TTS | `https://api.elevenlabs.io/v1/text-to-speech/<voice-id>` | `xi-api-key: <X-Audio-BYOK-Key>` |
| `openrouter` | Summarize | `https://openrouter.ai/api/v1/chat/completions` | `Authorization: Bearer <X-Audio-BYOK-Key>` |

Per-request credentials travel in metadata headers `X-Audio-BYOK-{Provider,Key}`, `X-Audio-LPBS-Token`, `X-Audio-User-Identity`. Adapters never log unredacted keys. Recorded fixtures via go-vcr are the default test path; an `--integration` build tag runs against real sandbox keys.

## Consumer scenarios (inbound)

| Consumer | Mechanism | Status |
|---|---|---|
| `web-console` | Connect-RPC via `scenarios/web-console/api/integrations/audiotools/` adapter; UI via the shared browser capture package and web-console's transport adapter | Shared browser integration migration. |
| `swarm-manager` | Future (covered by its own execute item) | Not started. |
| `agent-manager` | Future | Not started. |
| `phone-agent` | Future (twilio-voice transport in audio-tools) | Not started. |

## Discovery + lifecycle

Consumers discover the audio-tools URL via `api-core/discovery.ResolveScenarioURLDefault`. Captured URLs are short-lived: on any transport failure, the integration adapter calls `Client.HandleTransportFailure()` which invalidates the cached URL so the next call re-resolves. See `interoperability-steer §12`.

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, notes reference | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None yet. | not-applicable | SQLite is embedded by default. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario is standalone. | Add when this scenario calls or composes another scenario. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
