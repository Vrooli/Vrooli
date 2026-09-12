# Integrations — Audio Tools

## Purpose Of This Document

This is the dependency inventory for the current source and its degradation
boundaries. Desired portable routing lives in
[ARCHITECTURE.md](ARCHITECTURE.md#development-target-portable-streaming-voice);
commercial acceptance lives in [MONETIZATION.md](../business/MONETIZATION.md).
A declared dependency or adapter is not evidence of live availability.

## Dependency Inventory

| Dependency | Role | Source of truth |
| --- | --- | --- |
| Embedded SQLite and optional usage storage | Domain persistence | [DATA.md](DATA.md), API bootstrap and repositories |
| Vrooli control plane | Lifecycle, resource discovery, and host operations | [Service manifest](../../.vrooli/service.json) |
| Local resources and external providers | Capability-specific inference | Provider bootstrap, registries, and resource adapters below |
| Landing Page Business Suite (LPBS) | Intended owned-service integration; shared subscription/wallet fixture owner | LPBS clients and pilot evidence contract |
| Shared browser capture and consumer adapters | Capture, streaming, and host integration | [Shared package](../../../../packages/audio-capture-browser), consuming scenario adapters |

## Vrooli Resources

The service manifest declares these resources optional. This permits degraded
startup; it does not mean every capability works with zero resources and no key.

| Resource | Declared use | Unavailable behavior |
| --- | --- | --- |
| `whisper` | Local batch STT | Report unavailable or use a supported route allowed by policy. |
| `kyutai-stt` | Local streaming STT | Do not advertise a ready streaming engine from declaration alone. |
| `sherpa-onnx` | Native TTS, streaming STT, speaker operations, and separation | Qualify each operation and host separately; adapter presence is not platform support. |
| `kokoro` | Local TTS tier | Inspect the resolved adapter; a manifest entry does not establish a separate active engine. |
| `ollama` | Local summarization | Return unavailable when no permitted alternate provider exists. |
| `openrouter` | Cloud summarization via configured model role and credentials | Surface missing credentials, model configuration, or provider availability. |
| `postgres` | Optional usage history backend | Inspect selected storage configuration; SQLite remains the default. |

For engine mappings, inspect
[`BuildChains`](../../api/internal/bootstrap/providers.go) and
[engine metadata](../../api/internal/sttengine/manifest.json).
Manifest, adapter, readiness, and real qualification are distinct facts.

## Scenario Dependencies

LPBS is declared optional. Local and BYOK providers can serve supported requests
without it. The inspected bootstrap wires local/BYOK providers but supplies no
Vrooli provider to the three chains. LPBS STT, TTS, and summarization clients
currently return an unimplemented error; their shared availability implementation
returns false. Enabling `AUDIO_AI_ENABLE_VROOLI` alone cannot deliver hosted inference.

Sources: [provider bootstrap](../../api/internal/bootstrap/providers.go),
[LPBS clients](../../api/integrations/lpbs/clients), and
[environment defaults](../../api/internal/bootstrap/env.go).
Keep gateway/entitlement/metering ownership with the shared service owners; do not
create a consumer-private billing path to fill the missing implementation.

## Third-Party Services

The BYOK registry includes STT (OpenAI Whisper, Deepgram), TTS (OpenAI,
ElevenLabs), and summarization (OpenRouter) adapters. Inspect
[the BYOK source](../../api/internal/byok) for current protocol and credential
behavior rather than maintaining another endpoint/header table here.

Streaming support is provider-specific. Batch transcription is not native
streaming, even when its final text is returned over a streaming transport.
Never record secret values in diagnostics or evidence. See
[configuration](../reference/configuration.md) for operator-facing settings.

## Consumer Integration

Web Console and Swarm Manager have Audio Tools integration source. Neither
adapter presence nor an Audio Tools-only test qualifies the consumer's complete
voice path. Retain per-consumer capture, rendered-partial, final-tail, provider,
and recovery evidence.

- [Web Console adapter](../../../web-console/api/integrations/audiotools/)
- [Swarm Manager adapter](../../../swarm-manager/api/integrations/audiotools/)
- [Shared capture package](../../../../packages/audio-capture-browser/)
- [Qualification limitations](../internal/PROBLEMS.md)

Additional consumers must use the shared contract and their own host adapters;
do not infer their adoption from a proposed dependency list.

## Failure Modes

| Failure | Required interpretation | Evidence owner |
| --- | --- | --- |
| Resource absent or engine not qualified | Capability unavailable or explicitly degraded, not healthy from process liveness. | Engine/provider diagnostics |
| BYOK key rejected or protocol disconnected | Preserve the selected route and actionable reason; evaluate its permitted recovery. | Provider boundary tests |
| LPBS unavailable or hosted client unimplemented | Hosted inference unavailable, not automatic successful subscription fallback. | Integration clients and routing tests |
| Simulated wallet/provider succeeds | Fixture/control-flow evidence only. | LPBS fixtures and fake-provider assertions |
| Consumer stream fails after Audio Tools smoke passes | Consumer qualification remains open. | Paced consumer and device runs |

The [pilot test contract](../internal/TESTING.md#development-pilot-evidence-contract)
owns the local/BYOK/subscription/credits matrix, fixture isolation, negative
controls, and real-versus-simulated evidence rules.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md) — system boundaries
- [DATA.md](DATA.md) — storage ownership
- [Configuration](../reference/configuration.md) — environment and manifest settings
- [Monetization](../business/MONETIZATION.md) — commercial target and missing decisions
- [Deployment](../operations/DEPLOYMENT.md) — deployment readiness
