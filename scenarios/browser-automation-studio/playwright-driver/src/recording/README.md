# Recording Module

This module implements **Record Mode** - the ability to capture user actions in a browser and convert them to replayable automation instructions.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           RECORD MODE FLOW                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   Browser Page                    Node.js Driver                             │
│   ────────────                    ──────────────                             │
│                                                                              │
│   ┌─────────────────┐            ┌─────────────────┐                        │
│   │   User Action   │            │   API Request   │                        │
│   │  (click, type)  │            │ POST /record/   │                        │
│   └────────┬────────┘            │     start       │                        │
│            │                     └────────┬────────┘                        │
│            ▼                              │                                  │
│   ┌─────────────────┐                     ▼                                  │
│   │  injector.ts    │◀────────── RecordModeController                       │
│   │  (injected JS)  │            startRecording()                           │
│   │                 │                     │                                  │
│   │  Listens for:   │                     │                                  │
│   │  - clicks       │                     ▼                                  │
│   │  - input        │            page.exposeFunction()                      │
│   │  - scroll       │            __recordAction()                           │
│   │  - navigation   │                     │                                  │
│   └────────┬────────┘                     │                                  │
│            │                              │                                  │
│            │ RawBrowserEvent              │                                  │
│            │                              │                                  │
│            ▼                              ▼                                  │
│   window.__recordAction(event) ────▶ handleRawEvent()                       │
│                                              │                               │
│                                              ▼                               │
│                                    rawBrowserEventToTimelineEntry()         │
│                                    (proto/recording.ts)                     │
│                                              │                               │
│                                              ▼                               │
│                                    TimelineEntry (proto)                    │
│                                              │                               │
│                                              ▼                               │
│                                    onEntry callback                         │
│                                    (sent to API)                            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. RecordModeController (`controller.ts`)

**Responsibility:** Orchestrates the recording lifecycle.

```typescript
// Start recording
await controller.startRecording({
  sessionId: 'session-123',
  onEntry: (entry) => sendToApi(entry),
});

// Stop recording
await controller.stopRecording();
```

**Key behaviors:**
- Injects event listener script into browser via `page.evaluate()`
- Re-injects script after navigation (pages lose injected JS on navigate)
- Manages recording generation counter to prevent race conditions
- Delegates replay to `ReplayPreviewService`

### 2. Injector (`injector.ts`)

**Responsibility:** Browser-side event capture (runs in page context).

This is JavaScript that gets **stringified and injected** into the browser. It:
- Attaches DOM event listeners (click, input, scroll, etc.)
- Generates CSS selectors for target elements
- Scores selectors by uniqueness confidence
- Calls `window.__recordAction()` with raw events

### 3. Proto Recording Utilities (`../proto/recording.ts`)

**Responsibility:** Convert raw events to proto format.

```typescript
// Converts browser event to wire format
const entry = rawBrowserEventToTimelineEntry(rawEvent, {
  sessionId: 'session-123',
  sequenceNum: 42,
});
```

### 4. ReplayPreviewService (`replay-service.ts`)

**Responsibility:** Execute recorded actions for preview/validation.

```typescript
// Preview an action before saving
const result = await controller.replayPreview({
  entry: timelineEntry,
  timeoutMs: 5000,
});
```

### 5. Action Executor (`action-executor.ts`)

**Responsibility:** Registry of action executors.

```typescript
// Register a new action type
registerActionExecutor('custom-action', async (page, params) => {
  // Execute the action
  return { success: true };
});
```

### 6. Handler Adapter (`handler-adapter.ts`)

**Responsibility:** Bridge between recording and handlers.

Allows recorded actions to reuse the same handler implementations used for automation instructions, avoiding code duplication.

## Data Flow

```
┌──────────────┐    ┌───────────────┐    ┌─────────────────┐    ┌──────────────┐
│ Browser DOM  │───▶│ RawBrowserEvent│───▶│ TimelineEntry   │───▶│ API / Buffer │
│   Event      │    │ (injector.ts) │    │ (proto format)  │    │   Storage    │
└──────────────┘    └───────────────┘    └─────────────────┘    └──────────────┘

RawBrowserEvent = {
  actionType: 'click',
  selector: 'button.submit',
  timestamp: 1699999999999,
  payload: { ... }
}

TimelineEntry = {
  id: 'uuid',
  sessionId: 'session-123',
  sequenceNum: 1,
  action: { click: { selector: 'button.submit' } },
  capturedAt: Timestamp,
  ...
}
```

## Replay Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           REPLAY PREVIEW FLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   POST /record/replay-preview                                                │
│            │                                                                 │
│            ▼                                                                 │
│   RecordModeController.replayPreview()                                       │
│            │                                                                 │
│            ▼                                                                 │
│   ReplayPreviewService.replayPreview()                                       │
│            │                                                                 │
│            ▼                                                                 │
│   executeTimelineEntry()                                                     │
│            │                                                                 │
│            ├───▶ action-executor.ts (registry lookup)                        │
│            │            │                                                    │
│            │            ▼                                                    │
│            │     handler-adapter.ts (bridges to handlers/)                   │
│            │            │                                                    │
│            │            ▼                                                    │
│            │     handlers/interaction.ts (actual execution)                  │
│            │                                                                 │
│            ▼                                                                 │
│   ActionReplayResult                                                         │
│   { success: boolean, error?: string, ... }                                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Proto-First Architecture

Events are converted to proto format (`TimelineEntry`) as early as possible. This:
- Eliminates intermediate type definitions
- Ensures wire format compatibility with API
- Simplifies serialization

### 2. Recording Generation Counter

```typescript
private recordingGeneration = 0;

async startRecording() {
  this.recordingGeneration++;
  const currentGeneration = this.recordingGeneration;

  // Later, in async callbacks:
  if (this.recordingGeneration !== currentGeneration) {
    return; // Ignore stale callback
  }
}
```

This prevents race conditions when:
- Recording is stopped and restarted quickly
- Async callbacks from old recording arrive after new one starts

### 3. Script Re-injection After Navigation

Browser pages lose all injected JavaScript when they navigate. The controller:
1. Listens for `page.on('load')` events
2. Re-injects the recording script after each navigation
3. Uses exponential backoff retry (100ms, 200ms, 400ms)

### 4. Selector Confidence Scoring

The injector generates multiple selector strategies and scores them:

```javascript
const strategies = [
  { selector: '#unique-id', confidence: 1.0 },
  { selector: '[data-testid="button"]', confidence: 0.9 },
  { selector: 'button.submit', confidence: 0.7 },
  { selector: 'div > div > button', confidence: 0.3 },
];
```

The best selector (highest confidence above threshold) is used.

## Configuration

Recording behavior is controlled by `config.recording`:

| Option | Default | Description |
|--------|---------|-------------|
| `maxBufferSize` | 10000 | Max actions buffered per session |
| `minSelectorConfidence` | 0.3 | Minimum confidence for selector selection |
| `debounce.inputMs` | 500 | Input event debounce (batches keystrokes) |
| `debounce.scrollMs` | 150 | Scroll event debounce |
| `selector.maxCssDepth` | 5 | Max CSS path traversal depth |
| `selector.includeXPath` | true | Include XPath as fallback |

## File Reference

| File | Lines | Purpose |
|------|-------|---------|
| `controller.ts` | ~470 | Recording lifecycle orchestration |
| `injector.ts` | ~790 | Browser-side event capture (stringified JS) |
| `replay-service.ts` | ~200 | Replay preview execution |
| `action-executor.ts` | ~150 | Action executor registry |
| `handler-adapter.ts` | ~100 | Bridge to handlers/ |
| `selector-service.ts` | ~500 | Selector validation |
| `action-types.ts` | ~100 | Action type normalization maps |
| `types.ts` | ~50 | Shared type definitions |
