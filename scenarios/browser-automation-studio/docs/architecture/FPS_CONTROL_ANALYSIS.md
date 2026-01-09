# FPS Control Architecture Analysis

## Executive Summary

The FPS (frames per second) control for the live recording preview has **fragmented control levers** and **inconsistent defaults** scattered across multiple components. This document analyzes the current state and proposes a consolidated architecture.

**Root Cause of "6 FPS" Issue:** The session creation handler hardcodes `streamFPS := 6` which never gets used because recording start uses a completely different FPS path that defaults to 30. However, the session FPS is what gets stored and reported back, creating confusion.

---

## Current Architecture Problems

### 1. Multiple FPS Defaults (Domain Compression Issue)

| Location | Default | When Used |
|----------|---------|-----------|
| `api/handlers/record_mode.go:82` | **6** | Session creation |
| `api/services/live-capture/service.go:253` | **30** | Recording start |
| `api/config/config.go:644` | **30** | Config system (unused!) |
| `docs/ENVIRONMENT.md:92` | **6** | Documentation |
| `playwright-driver/src/routes/record-mode/types.ts:29` | **6** | Driver doc comment |
| `playwright-driver/src/frame-streaming/types.ts:40` | **30** | Driver frame-streaming doc |
| `ui/src/domains/recording/capture/StreamSettings.tsx:59` | **20** | UI custom settings |

**Impact:** Nobody knows what the actual FPS default is because it differs based on where you look.

### 2. Dual FPS Control Points (Control Surface Issue)

There are **two independent FPS parameters** at different lifecycle points:

```
┌─────────────────────────────┐
│ CreateRecordingSession      │
│   └─ StreamFPS (default: 6) │  ← Stored in session but NOT used for streaming
└─────────────────────────────┘
              ↓
┌─────────────────────────────┐
│ StartLiveRecording          │
│   └─ FrameFPS (default: 30) │  ← Actually controls frame streaming rate
└─────────────────────────────┘
```

**Impact:** Session creation FPS is ignored. Only recording start FPS matters. But the session stores the wrong value (6) which gets reported back to UI.

### 3. Naming Inconsistency (Domain Compression Issue)

The same concept is called different names:

| Component | Field Name |
|-----------|------------|
| API Handler (session) | `StreamFPS` |
| API Handler (recording) | `FrameFPS` |
| API Config | `DefaultStreamFPS` |
| Driver | `frame_fps`, `targetFps` |
| UI | `fps` |

**Impact:** Developers don't realize they're dealing with the same concept.

### 4. CDP vs Polling Strategy Disparity (Architecture Issue)

```
Polling Strategy:
  - Uses FpsController for adaptive rate control
  - Respects targetFps parameter
  - updateTargetFps() method works

CDP Screencast Strategy:
  - Chrome controls frame rate (compositor timing)
  - targetFps parameter is IGNORED
  - updateTargetFps() method does NOT exist
  - everyNthFrame is always 1
```

**Impact:** FPS settings only work for polling fallback, not the primary CDP screencast path.

### 5. Config System Not Wired Up (FIXED)

~~The config system has `BAS_RECORDING_DEFAULT_STREAM_FPS=30` but it's never used.~~

**Status: RESOLVED** - Config system is now wired up:
- `handlers/record_mode.go` uses `config.Load().Recording.DefaultStreamFPS`
- `services/live-capture/service.go` uses config for both CreateSession and StartRecording
- All hardcoded defaults removed in favor of config values

---

## Frame Streaming Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           UI Layer                                        │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐         │
│  │ useFrameStream  │──▶│ WebSocketContext│──▶│ Canvas Render   │         │
│  │   (hook)        │   │ subscribeToBinary│   │                 │         │
│  └─────────────────┘   └─────────────────┘   └─────────────────┘         │
└──────────────────────────────────────────────────────────────────────────┘
                                    ▲
                                    │ WebSocket binary frames
                                    │
┌──────────────────────────────────────────────────────────────────────────┐
│                           Go API Layer                                    │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐         │
│  │ record_mode.go  │──▶│ live-capture    │──▶│ WebSocket Hub   │         │
│  │   (handlers)    │   │ service.go      │   │ BroadcastBinary │         │
│  └─────────────────┘   └─────────────────┘   └─────────────────┘         │
└──────────────────────────────────────────────────────────────────────────┘
                                    ▲
                                    │ HTTP/WebSocket to driver
                                    │
┌──────────────────────────────────────────────────────────────────────────┐
│                       Playwright Driver Layer                             │
│  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐         │
│  │ recording-      │──▶│ frame-streaming │──▶│ CDP Screencast  │         │
│  │ lifecycle.ts    │   │ manager.ts      │   │ OR Polling      │         │
│  └─────────────────┘   └─────────────────┘   └─────────────────┘         │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## CDP Screencast Behavior

Chrome's `Page.startScreencast` API:
- **No FPS parameter** - Chrome decides when to send frames
- Sends frames when compositor has changes (up to ~60 FPS)
- `everyNthFrame: 1` means "send every frame Chrome produces"
- Frame rate depends on page activity and system load

This means **FPS control doesn't work for CDP screencast** - it's Chrome's decision.

---

## Proposed Consolidation

### 1. Single Source of Truth for Defaults

```go
// config/config.go - THE ONLY PLACE defaults are defined
type RecordingConfig struct {
    DefaultQuality  int  // 1-100, default 55
    DefaultFPS      int  // 1-60, default 30 (target for polling, ignored by CDP)
    DefaultScale    string // "css" or "device", default "css"
}
```

### 2. Unified Control Lever

Remove `StreamFPS` from session creation. Only one FPS parameter:

```go
// StartRecordingRequest - the ONLY place FPS is specified
type StartRecordingRequest struct {
    TargetFPS int `json:"target_fps,omitempty"` // Target FPS (1-60), actual may vary
}
```

### 3. Consistent Naming

Rename all FPS-related fields to `target_fps` / `targetFps` with clear documentation that:
- For **polling strategy**: This is the actual target frame rate
- For **CDP screencast**: This is a hint but Chrome controls actual rate

### 4. Document CDP Reality

Add prominent comments/documentation explaining:
```
FPS Control Behavior:
- CDP Screencast (Chromium default): Chrome compositor controls frame rate.
  The target_fps is stored for UI display but doesn't control actual rate.
- Polling Fallback (Firefox/WebKit): target_fps controls the capture interval.
  Adaptive FPS may reduce rate if captures are slow.
```

### 5. Wire Up Config System

```go
// handlers/record_mode.go - use config, not hardcoded
func (h *Handler) CreateRecordingSession(...) {
    cfg := config.Load()
    streamFPS := cfg.Recording.DefaultFPS  // Not hardcoded 6!
    if req.TargetFPS != nil && *req.TargetFPS >= 1 && *req.TargetFPS <= 60 {
        streamFPS = *req.TargetFPS
    }
}
```

---

## Implementation Steps

### Phase 1: Consolidate Defaults (Breaking Change: None) - COMPLETED

1. ~~Delete all hardcoded FPS defaults except `config.go`~~ ✅
2. ~~Update `record_mode.go:82` to use `cfg.Recording.DefaultStreamFPS`~~ ✅
3. ~~Update `service.go:253` to use config~~ ✅
4. ~~Update driver defaults to match (30)~~ ✅

**Completed Changes:**
- `api/handlers/record_mode.go`: Now uses `config.Load().Recording.DefaultStreamFPS`
- `api/services/live-capture/service.go`: Both `CreateSession` and `StartRecording` use config
- `playwright-driver/src/routes/record-mode/recording-lifecycle.ts`: Changed fallback from 6→30
- `playwright-driver/src/routes/record-mode/types.ts`: Updated doc comments
- `playwright-driver/src/types/session.ts`: Updated doc comments
- `api/handlers/record_mode_types.go`: Updated doc comments
- `docs/ENVIRONMENT.md`: Fixed documentation to show correct default (30)

### Phase 2: Unify Control Lever (Breaking Change: Minor)

1. Deprecate `stream_fps` in `CreateRecordingSessionRequest`
2. Keep only `frame_fps` in `StartRecordingRequest` (renamed to `target_fps`)
3. Update UI to only send FPS at recording start

### Phase 3: Improve Transparency (Breaking Change: None)

1. Add `strategy` field to status response ("cdp_screencast" | "polling")
2. Document that FPS control only works for polling
3. Add "FPS control limited" indicator in UI when using CDP

---

## Files Requiring Changes

| File | Change |
|------|--------|
| `api/handlers/record_mode.go:82` | Use config instead of hardcoded 6 |
| `api/handlers/record_mode_types.go:26` | Update doc comment |
| `api/services/live-capture/service.go:253` | Use config |
| `playwright-driver/src/routes/record-mode/types.ts:29` | Update doc |
| `playwright-driver/src/frame-streaming/types.ts:40` | Match config default |
| `playwright-driver/src/types/session.ts:307` | Update doc |
| `docs/ENVIRONMENT.md:92` | Update default value |
| `docs/CONTROL_SURFACE.md` | Document actual behavior |

---

## Success Metrics

1. **Single Default**: Only one place defines FPS default (config.go)
2. **One Control Point**: FPS only set at recording start, not session creation
3. **Consistent Naming**: All code uses `target_fps` / `targetFps`
4. **Accurate Reporting**: UI shows actual FPS achieved, not requested target
5. **Clear Documentation**: Users understand CDP vs polling behavior
