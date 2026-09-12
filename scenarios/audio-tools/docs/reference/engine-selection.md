# STT engine selection

When `StreamConfig.EngineID` is set, that operator pin wins. When it is empty,
`sttengine.Registry.Resolve` evaluates the manifest engines through the live
availability seam and applies the declared hard filters followed by
`selectionPolicy.rankedPreferences`; manifest order is the deterministic
tie-break. `ResolveFacts` accepts bounded platform, install, health, and
accelerator observations for portability fixtures. The result includes an
ordered candidate list, a verdict, and a reason naming the signal used.

The shipped policy ranks `accelerated` candidates first. When no candidate is
accelerated, manifest order is the deterministic tie-break; native streaming
is a capability used by strategy negotiation, not a universal ranking override.

The policy declares a 30-second probe TTL for integrations that collect host
facts. Audio-tools uses one shared control-plane client to locate and invoke
`vrooli`; backend recovery, capacity reporting, and provider lifecycle do not
keep independent binary-discovery seams.

To add an engine, add one manifest entry with its resource, capabilities, and
strategies, add the provider adapter, and add the backing resource declaration.
To re-rank engines, edit `selectionPolicy.rankedPreferences` or the manifest
order. Resolver code must not branch on an engine identifier.

If no candidate is serviceable, the resolver returns the typed
`no_serviceable_engine` error and the remediation is to start a configured STT
resource. It never silently selects an unavailable engine.

## Why Whisper and Kyutai both remain

These engines are complementary capability tiers, not duplicate providers.
The authoritative details are in `api/internal/sttengine/manifest.json` and
the resource manifests.

| Property | Whisper (`whisper-local`) | Kyutai (`kyutai`) |
|---|---|---|
| Model and languages | `ggml-medium`; multilingual | `stt-1b-en_fr`; English and French |
| Transport | Batch HTTP `/asr` | Native streaming `/v1/stream` |
| Batch transcription | Yes | No batch endpoint |
| Native streaming | No | Yes |
| Confidence signals | `no_speech_prob`, `avg_logprob` | None declared |
| Strategies | `vad_segment`, `overlap_agree`, `buffered_fallback` | `passthrough` |
| Platform support | Linux supported; Windows build-verified | Linux only |

Whisper cannot be removed without removing batch transcription, the
multilingual tier, Windows speech-to-text support, and the confidence signals
used by `api/internal/stt/egress/confidence_stage.go` to suppress hallucinated
silence transcripts. The measured CPU readiness probe is expensive: a one-
second digital-silence clip produced `you`; the confidence stage is the guard
for that failure mode. Kyutai declares no confidence signals and cannot
replace that filter.

The two-tier decision also applies to health: cheap GET liveness checks are
used by consumer surfaces, while Whisper's real `/asr` inference remains an
explicit full/readiness check. `sherpa-onnx` may subsume Kokoro and speaker
verification in a future consolidation, but it does not subsume Whisper's
batch, multilingual, Windows, or confidence-signal capabilities.
