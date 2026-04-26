# Temporal Flows — Web Console

> Source of truth: the code. Verify claims below against actual implementation.

## Async Operation Inventory

### API (Go)

| Flow | File | Trigger | Depends On | Updates | Completion |
|------|------|---------|------------|---------|------------|
| **PTY readLoop** | `session.go:146` | `SessionManager.Create()` | PTY file descriptor | Broadcasts to client channels, sets `processExited`, closes `exitCh` | PTY read error (process exit) |
| **Session auto-cleanup** | `session.go:261` | `SessionManager.Create()` | `<-sess.Done()` (exitCh closed) | Removes session from manager map | Immediate after Done signal |
| **ExpirationSweeper** | `session_policy.go:157` | `Server` startup via `NewServer` | 30s ticker, session list | Deletes expired sessions, emits events, updates metrics | `Stop()` called in server Cleanup |
| **WebSocket output forwarder** | `terminal_ws.go:101` | WS upgrade in `handleTerminalWS` | PTY output channel + `ctx.Done()` | Writes stdout/exit to WS conn, calls `FlushPending` after each write | Channel closed (process exit), WS write error, or `ctx.Done()` (input loop exited) |
| **WebSocket input loop** | `terminal_ws.go` | WS upgrade (inline, same goroutine) | WS read | Writes to PTY stdin, triggers resize | WS read error → `defer cancel()` signals forwarder |
| **AI provider chain** | `ai_generate.go:253` | `POST /api/v1/ai/generate` | HTTP request context | Per-provider timeout context, health metrics | All providers tried or first success |

### UI (React/TypeScript)

| Flow | File | Trigger | Depends On | Updates | Cleanup |
|------|------|---------|------------|---------|---------|
| **Terminal init** | `TerminalPane.tsx:41` | Component mount | Container DOM ref | Creates Terminal, FitAddon, WebLinksAddon | `term.dispose()`, clears refs |
| **WebSocket lifecycle** | `useTerminalSocket.ts:96` | `sessionId` + `terminal` ready | Terminal instance | WS connection, input listener | Disposes input listener, closes WS |
| **Resize observer** | `TerminalPane.tsx:72` | Terminal + sendResize ready | Container element | fit() + sendResize via rAF throttle | Disconnects observer, cancels rAF |
| **Health check** | `App.tsx:23` | App mount | React Query | Retry with backoff (3 retries, 1s delay) | Managed by React Query |
| **Session creation** | `useSessionManager.ts:39` | User action (launch) | API call | Panes, activePane, createError | Error auto-dismiss timer cleaned on unmount |
| **Error auto-dismiss** | `useSessionManager.ts` | Failed session creation | setTimeout ref | Clears createError after 8s | Timer cleared on unmount via ref |
| **AI generation** | `AiInput.tsx:29` | User prompt submit | API call | command, provider, error, isLoading | Generation ID guards stale responses on unmount |
| **Countdown timer** | `useCountdown.ts` | Session with expiry policy | 1s setInterval | Remaining seconds display | Interval cleared on unmount |
| **Settings load** | `SettingsPage.tsx:131` | Page mount | API call | profiles, error, loading | Cancellation signal on unmount |
| **Integrations health** | `IntegrationsPanel.tsx` + `useCapabilities.ts` | Panel open | react-query 30s poll | Capability states | Polling paused when `open=false` |

## Ordering Assumptions & Stability

### Stable (tested/guaranteed)
- **Terminal init → WS connect**: Terminal `useState` triggers WS effect. Ordering enforced by React's `useEffect` dependency chain.
- **PTY readLoop → auto-cleanup**: Cleanup goroutine waits on `<-sess.Done()`. Channel close is the coordination signal.
- **Sweeper start → stop**: Started in `NewServer`, stopped in server `Cleanup` callback. Lifecycle tied to HTTP server.
- **Output forwarder ↔ input loop**: Coordinated via `context.WithCancel` + `writeMu` mutex for WS writes. Input loop exit → cancel() → forwarder sees ctx.Done(). Forwarder exit → conn.Close() → input loop ReadMessage fails.

### Assumptions to monitor
- **Session limit check → create**: `isSessionLimitReached()` uses RLock, then `Create()` takes write lock. TOCTOU gap exists but is acceptable for a soft limit.
- **Expired session access window**: Up to 30s after expiration, a session can still accept WS connections. The sweeper runs on a 30s interval.

## Race Conditions & Mitigation

| Race | Location | Status | Mitigation |
|------|----------|--------|------------|
| Concurrent WS writes | `terminal_ws.go:89` | **Mitigated** | `writeMu sync.Mutex` serializes all WS writes |
| Client channel coalescing | `broadcast.go` | **By design** | Non-blocking send with coalescing; slow clients receive merged data on catchup via `FlushPending`. Pending buffer capped at the fixed `pendingBufferMax`; on overflow the oldest bytes are truncated and the next snapshot replay restores correct state |
| Event subscriber drop | `events.go:74` | **By design** | Non-blocking fan-out; full channels skip events |
| Stale AI response | `AiInput.tsx` | **Mitigated** | Generation ID ref discards stale setState calls |
| Stale settings load | `SettingsPage.tsx` | **Mitigated** | Cancellation signal prevents setState on unmounted component |
| Error dismiss after unmount | `useSessionManager.ts` | **Mitigated** | Timer ref cleaned up in useEffect cleanup |

## Initialization & Teardown

### Server startup sequence
1. Database connect (`main.go:145`)
2. `NewServer()` — creates SessionManager, EventLogger, Metrics, AI chain, config stores
3. ExpirationSweeper created and started (`main.go` — `sweeper.Start()`)
4. Routes registered
5. `server.Run()` — starts HTTP listener

### Server shutdown sequence
1. HTTP server graceful shutdown (via `server.Run` signal handling)
2. Cleanup callback: `sweeper.Stop()` → `db.Close()`

### UI component lifecycle
1. `App` mounts → health check via React Query
2. `Workspace` renders → `useSessionManager` initializes pane/ref state
3. User launches session → API call → pane added → `TerminalPane` mounts
4. `TerminalPane` mounts → xterm.js init → `useTerminalSocket` connects WS
5. On unmount: WS closed, input listener disposed, ResizeObserver disconnected, rAF cancelled

## Polling, Retry & Scheduling

| Mechanism | Interval | Bounded? | Failure Handling |
|-----------|----------|----------|-----------------|
| ExpirationSweeper | 30s | Yes (stop via channel) | Continues on individual session delete errors |
| Health check retry | 1s backoff, 3 attempts | Yes | Shows error UI after exhaustion |
| Countdown timer | 1s | Yes (cleared on unmount) | Shows 0 when expired |
| ResizeObserver | Per-frame (rAF throttled) | Yes (cancelled on unmount) | No-op on error |

## Configuration Points

| Parameter | Default | Location |
|-----------|---------|----------|
| Sweep interval | 30s | `session_policy.go:129` |
| AI provider timeout | Per-config (`timeout_sec`) | `ai_generate.go:273` |
| HTTP client timeout | 30s | `ai_generate.go:90,149` |
| Offline buffer max | Config-driven | `config.go` |
| Error auto-dismiss | 8s | `consts/config.ts:69` |
| Health retry count | 3 | `consts/config.ts:63` |
| Health retry delay | 1000ms | `consts/config.ts:66` |
| Client channel buffer | Config-driven | `config.go` |

### Voice — Persistent Mode & Segment Finals

| Flow | File | Trigger | Depends On | Updates | Completion |
|------|------|---------|------------|---------|------------|
| **Segment boundary detection** | `vad.ts` | VAD silence ≥ segmentSilenceMs | Audio level monitor (rAF loop) | Emits "segment-boundary" action | Speech resumes or stop timeout |
| **Segment-final transcription** | `voice_stream_ws.go` | Client sends `segment-boundary` message | Audio buffer snapshot, Whisper server | Sends `segment-final` to client | Whisper returns transcription |
| **Adaptive silence threshold** | `useVoiceInput.ts` | Partial transcript contains command prefix | VoiceStreamProvider partials | Temporarily reduces VAD segmentSilenceMs | Segment final resolves |
| **Command suggestion lifecycle** | `VoiceCommandSuggestion.tsx` | Segment-final parsed as command | `parseCommand()` result | Shows suggestion UI above toolbar | User confirms, dismisses, or 5s auto-dismiss |

### Speaker Verification Gate

| Flow | File | Trigger | Depends On | Updates | Completion |
|------|------|---------|------------|---------|------------|
| **Segment speaker verification** | `voice_stream_ws.go` | Segment-boundary goroutine start | Segment audio snapshot, speaker-verification resource | Sends `segment-accepted` or `segment-rejected` to client; gates `segment-final` emission | Resource verify response or timeout (30s) |
| **Final speaker verification** | `voice_stream_ws.go` | Recording done, before full retranscribe | Full audio buffer, speaker-verification resource | Allows or suppresses final transcription | Resource verify response or timeout (30s) |
| **HTTP transcribe verification** | `voice_transcribe.go` | `POST /api/v1/voice/transcribe` | Audio payload, speaker-verification resource | Returns empty text on rejection | Resource verify response or timeout |

**Pipeline ordering** (when speaker verification is enabled in `filter` mode):

```
segment-boundary received
  → snapshot segment audio
  → evaluateSpeakerVerification(ctx, segmentAudio)
    → if !allowed: send segment-rejected, skip transcription
    → if  allowed: transcribeBytes → send segment-accepted + segment-final

recording done
  → wait for in-flight segment-finals (segmentFinalWg)
  → evaluateSpeakerVerification(ctx, fullAudio)
    → if !allowed: send final with empty text
    → if  allowed: transcribeBytes → send final with text
```

**Fallback behavior**: When the speaker-verification resource is unreachable or errors, `FallbackWithoutVerification` controls whether the transcript is allowed through (`true`) or suppressed (`false`, default).

### Two-Tier Transcription Timing

In persistent voice mode, transcription operates at two quality tiers:

1. **Tier 1 — Streaming partials** (every ~500ms): Fast, rough transcription of audio deltas for real-time UI feedback. Same mechanism as one-shot mode.

2. **Tier 2 — Segment finals** (on VAD silence ≥ segmentSilenceMs): High-quality retranscription of the complete speech segment with ffmpeg WAV transcoding. Replaces rough partials with accurate text. Runs in a separate goroutine so it doesn't block Tier 1 partials for the next segment.

**Adaptive silence threshold**: When the command prefix is detected in a partial, segmentSilenceMs is temporarily reduced from its configured value (default 1500ms) to ~700ms so commands resolve faster. The original threshold is restored when the segment final resolves.
