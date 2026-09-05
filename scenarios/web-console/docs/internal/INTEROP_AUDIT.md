# Interop Audit — web-console

This document captures the interoperability contract between web-console
and the scenarios it integrates with. The architectural rule, enforced
by the boundary tests below, is:

> **A scenario UI only talks to its own scenario API. All inter-scenario
> calls are server-to-server, expressed as Connect-RPC, and addressed via
> `api-core/discovery`.**

## Dependencies

### audio-tools (`required: true`)

| Hop | Wire | Code path |
|---|---|---|
| Browser → web-console | Connect-Web (same-origin) | `handlers/audio_admin/` + `handlers/audio_runtime/` |
| web-console → audio-tools | Connect-RPC (server↔server) | `internal/audioports/remote_*.go` via `integrations/audiotools.Client` |
| Browser ↔ audio-tools (WS) | Same-origin WS, proxied | `voice_stream_proxy.go` (registered at `/api/v1/voice/stream`) |

#### Resolved RPCs (web-console handler → audio-tools RPC)

- `AudioAdminService.GetStreamConfig` → `STTAdminService.GetStreamConfig`
- `AudioAdminService.UpdateStreamConfig` → `STTAdminService.UpdateStreamConfig`
- `AudioAdminService.GetWakeWordConfig` → `STTAdminService.GetWakeWordConfig`
- `AudioAdminService.UpdateWakeWordTemplate` → `STTAdminService.UpdateWakeWordTemplate`
- `AudioAdminService.DeleteWakeWordTemplate` → `STTAdminService.DeleteWakeWordTemplate`
- `AudioAdminService.GetSpeakerConfig` → `STTAdminService.GetSpeakerConfig`
- `AudioAdminService.UpdateSpeakerConfig` → `STTAdminService.UpdateSpeakerConfig`
- `AudioAdminService.GetSpeakerStatus` → `STTAdminService.GetSpeakerStatus`
- `AudioAdminService.ListSpeakerProfiles` → `STTAdminService.ListSpeakerProfiles`
- `AudioAdminService.EnrollSpeakerProfile` → `STTAdminService.EnrollSpeakerProfile`
- `AudioAdminService.ClearSpeakerProfileBinding` → `STTAdminService.ClearSpeakerProfileBinding`
- `AudioAdminService.UnbindSpeakerProfile` → `STTAdminService.UnbindSpeakerProfile`
- `AudioAdminService.DeleteSpeakerProfile` → `STTAdminService.DeleteSpeakerProfile`
- `AudioAdminService.GetTTSConfig` → `TTSService.GetConfig`
- `AudioAdminService.UpdateTTSConfig` → `TTSService.UpdateConfig`
- `AudioAdminService.GetSummarizeConfig` → `SummarizeService.GetSummarizeConfig`
- `AudioAdminService.UpdateSummarizeConfig` → `SummarizeService.UpdateSummarizeConfig`
- `AudioRuntimeService.Transcribe` → `STTService.Transcribe`
- `AudioRuntimeService.Synthesize` → `TTSService.Synthesize`
- `AudioRuntimeService.ListVoices` → `TTSService.ListVoices`
- `AudioRuntimeService.GetTTSCache` → `TTSService.GetCache`
- `AudioRuntimeService.RecordPlaybackEvent` → `TTSService.RecordPlaybackEvent`
- `AudioRuntimeService.Summarize` → `SummarizeService.Summarize`

#### Error envelope

The single conversion point is
`web-console/internal/audioports/contracts.go` (proto↔domain) plus the
handler `mapErr` (`*connect.Error` mapping). audio-tools error codes
flow through `audiotools.NormalizeError` once and surface to the UI as
typed Connect errors.

#### Discovery + retry

- `audiotoolsint.Client` resolves the audio-tools URL via
  `api-core/discovery.ResolveScenarioURLDefault("audio-tools")` (or
  `AUDIO_TOOLS_URL` if set).
- On `connect.CodeUnavailable` / `DeadlineExceeded` the client clears
  its `resolved` flag, forcing a fresh resolve on the next call. Bounded
  retry budget lives in `audiotoolsint.Policy{MaxRetries: 3}`.

#### Sentinel: no CORS escape hatch

The architectural rule depends on CORS *never* being added to
audio-tools — the browser must never originate a cross-origin request
against another scenario. audio-tools is verified to continue returning
`405 Method Not Allowed` to foreign-origin `OPTIONS` requests
end-to-end; if anyone ever adds CORS to audio-tools, this audit and the
sentinel test it references should be re-evaluated.

## Boundary tests

- `scenarios/web-console/ui/src/__tests__/audio-boundary.test.ts`
  asserts no UI file imports `@vrooli/proto-types/audio-tools/*` or creates
  a private voice-stream provider instead of using the shared capture package.
- `scenarios/web-console/api/handlers/audio_admin/connect_handler_test.go`
  pins the Connect handler error envelope (Unavailable → CodeUnavailable,
  empty mask → CodeInvalidArgument, missing profile_id → CodeInvalidArgument).

## Open follow-ups

- Convert the voice WS to a Connect server-stream
  (`AudioRuntimeService.StreamSttSession`) once the upstream native
  partials in `audio-tools` PRD OT-P1-014 land. Today's WS reverse
  proxy preserves the wire shape and keeps the boundary clean.
- Add a cross-scenario e2e test that restarts audio-tools mid-call to
  validate the discovery re-resolve under load.
