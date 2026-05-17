# Progress — Audio Tools

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-16 | claude | done | Full feature completion sweep: persistence (system.sql + internal/store), BYOK AES-GCM crypto, SettingsService end-to-end, async usage reporter, STT admin handlers (stream config/wakeword/speaker), TTS config/status/cache/playback, ffmpeg-backed audio ops (transcode/trim/merge/split/fade/volume/normalize/metadata) with multipart route, SessionService SendText + Subscribe streaming + WS bridge, UI try-its (synthesize+transcode), Configuration edit forms, real Voices probes, CLI settings/usage groups + real diagnose, embed stub-text cleanup. Zero Unimplemented embeds in handlers/. |
| 2026-05-16 | claude | done | Streaming STT decoupling phases A–G landed end-to-end (plan: audio-tools-streaming-stt-decouple-strategy-from-provider). A: replaced `Provider.StreamingCapability() bool` with `ProviderTraits{Batch, Stream, Strategies}` on all 3 providers; added `StrategyKind` constants, `internal/stt/selector.go` (Selector + StreamConfig + typed errors), `internal/stt/strategy/` (Strategy interface + BufferedFallback). B: `internal/stt/segmenter/` with Segmenter.Run, `Chain.StreamCandidates` accessor, deterministic `testaudio` PCM helper, parity baseline test. C: Connect bidi handler rewired through Segmenter; `Deps.Selector` plumbed; `mapChainError` maps the new typed errors; parity test drives the real Connect handler over an HTTP/2 httptest server. D: `voice.Service.HandleStreamWS` (~440 lines) deleted; new `handlers/stt/stream_ws.go::StreamWSHandler` routes browser WS through the shared Segmenter and translates StreamEvents to the `@audio-tools/embed` JSON wire shape; `internal/voice/stream_ws.go` reduced to small WebM/bitrate constants used by transports; `internal/transports/browser` reduced to session-context helpers; `VADSegmentStrategy` with amplitude-based server-side VAD on PCM lands as the default strategy for LocalProvider; PROBLEMS.md regression entry filed for browser WebM input. E: 5 lever fields added to proto `StreamConfig` + `UpdateStreamConfigRequest` (regen run); persisted in `streamCfgDoc` with documented defaults; `validateStreamingLevers` enforces ranges; selector honors `streaming_mode=off`; CLI `voice stream-config` + `voice stream-config-set` commands added. F: Deepgram BYOK adapter implements `TranscribeStreaming` against wss://api.deepgram.com/v1/listen with linear16/16kHz framing; traits flip to `Stream=true, Strategies=[…, passthrough]`. G: `OverlapAgreeStrategy` with token-prefix LocalAgreement (Macháček 2023) and configurable window/commit-runs; LocalProvider whitelists overlap_agree; reachable via `strategy_preference=overlap`. Full selector compatibility matrix landed in `internal/stt/selector.go` and verified by table-driven `selector_test.go`; greenfield grep guards clean; `go build ./... && go test ./... -timeout 300s` green. Two PROBLEMS.md entries closed: strategy/provider fusion + two-transports-no-parity. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
