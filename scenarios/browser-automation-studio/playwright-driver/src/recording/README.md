# Recording Module

This module implements **Record Mode** - the ability to capture user actions in a browser and convert them to replayable automation instructions.

## Architecture Overview

The recording system uses a **proto-first architecture** where all events are converted to proto `TimelineEntry` format as early as possible. This ensures wire format compatibility and eliminates intermediate type definitions.

### Two-Layer Design

```
Context Setup (once per context)        Recording Sessions (per session)
┌─────────────────────────────────┐     ┌─────────────────────────────────┐
│  RecordingContextInitializer    │     │  RecordingPipelineManager       │
│  ├─ html-injector.ts            │     │  ├─ state-machine.ts (state)    │
│  │   └─ Inject script into HTML │     │  ├─ startRecording()/stop()     │
│  └─ event-route.ts              │     │  ├─ handleRawEvent()            │
│      └─ Page event interception │     │  │   └─ rawBrowserEventTo...()  │
└─────────────────────────────────┘     │  └─ verifyPipeline()            │
                                        └──────────────┬──────────────────┘
Browser Page (MAIN context)                            │
┌─────────────────────────────┐     ┌─────────────────────────────────────┐
│ recording-script.js         │     │  buffer.ts (TimelineEntry storage)  │
│ ├─ Listens for activation   │     └─────────────────────────────────────┘
│ ├─ Captures DOM events      │────▶ fetch POST to event route
│ ├─ Wraps History API        │
│ └─ Generates selectors      │
└─────────────────────────────┘
```

## File Guide

### Understanding the System

| File | Purpose |
|------|---------|
| `state-machine.ts` | Core state machine (single source of truth for recording state) |
| `pipeline-manager.ts` | Main orchestrator (start/stop, state transitions, event handling) |
| `context-initializer.ts` | Context-level setup coordinator |
| `html-injector.ts` | HTML injection into document responses (sub-module of context-initializer) |
| `event-route.ts` | Page-level event route setup (sub-module of context-initializer) |
| `decisions.ts` | Named decision functions (inject? process?) for debuggability |
| `init-script-generator.ts` | Generates init script for `context.addInitScript()` |
| `action-executor.ts` | Proto-native action execution for replay |
| `replay-service.ts` | Replay preview for testing recorded entries |
| `selector-service.ts` | Selector validation on pages |
| `../proto/recording.ts` | Proto conversion (RawBrowserEvent → TimelineEntry) |

### Adding a New Action Type

1. `packages/proto/schemas/.../action.proto` - Add to ActionType enum
2. `../proto/action-type-utils.ts` - Add string ↔ enum mappings
3. `../handlers/*.ts` - Implement handler (preferred)
   OR `action-executor.ts` - Add executor (if handler not suitable)

### Modifying Selector Generation

1. `selector-config.ts` - Configuration (scores, patterns)
2. `browser-scripts/recording-script.js` - Browser-side selector logic
3. `selectors.ts` - Documentation only, not executable

### Modifying Action Buffering

- `buffer.ts` - In-memory TimelineEntry storage with FIFO eviction

## Key Components

### RecordingPipelineManager (`pipeline-manager.ts`)

**Responsibility:** Orchestrates the recording lifecycle using a state machine.

```typescript
// Start recording
await pipelineManager.startRecording({
  sessionId: 'session-123',
  onEntry: (entry) => sendToApi(entry),
});

// Stop recording
const result = await pipelineManager.stopRecording();
```

**Key behaviors:**
- Manages recording state via reducer pattern (state-machine.ts)
- Handles raw browser events and converts to TimelineEntry
- Coordinates with RecordingContextInitializer for infrastructure
- Delegates replay to handler-adapter.ts

### RecordingContextInitializer (`context-initializer.ts`)

**Responsibility:** Coordinates one-time context setup for recording infrastructure.

This is a coordinator that composes specialized modules:

- **`html-injector.ts`** - Intercepts document requests and injects recording script into HTML
- **`event-route.ts`** - Sets up page-level routes for event interception

Why this architecture:
- `rebrowser-playwright` doesn't intercept fetch/XHR via `context.route()` (only navigation)
- Recording script must run in MAIN context (not isolated) for History API wrapping
- Page routes don't persist across navigation (need re-registration)

### Recording Script (`browser-scripts/recording-script.js`)

**Responsibility:** Browser-side event capture (runs in page MAIN context).

This JavaScript gets injected into every page. It:
- Attaches DOM event listeners (click, input, scroll, etc.)
- Generates CSS selectors for target elements
- Scores selectors by uniqueness confidence
- Sends events via exposed binding to Node.js

### Handler Adapter (`handler-adapter.ts`)

**Responsibility:** Bridge between recording and handlers.

Allows recorded actions to reuse the same handler implementations used for automation instructions, avoiding code duplication.

## Data Flow

```
┌──────────────┐    ┌───────────────┐    ┌─────────────────┐    ┌──────────────┐
│ Browser DOM  │───▶│ RawBrowserEvent│───▶│ TimelineEntry   │───▶│ API / Buffer │
│   Event      │    │ (recording-   │    │ (proto format)  │    │   Storage    │
└──────────────┘    │  script.js)   │    └─────────────────┘    └──────────────┘
                    └───────────────┘

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

## State Machines

The recording system uses **two separate state machines** that work together:

### Session State Machine (`../session/state-machine.ts`)

**Purpose:** API-level session lifecycle (6 phases)

```
initializing → ready ↔ executing
                 ↕
              recording → closing
```

- `recording` phase = "User is in record mode" (API-level)
- Used by external consumers (UI, API clients)

### Recording Pipeline State Machine (`state-machine.ts`)

**Purpose:** Infrastructure-level event capture (8 phases)

```
uninitialized → initializing → verifying → ready → starting → capturing
                    ↓              ↓          ↑         ↓          ↓
                  error ←──────────┴──────────┴─────────┴──────────┘
```

- `capturing` phase = "Actively capturing events" (infrastructure-level)
- Used internally for infrastructure health tracking

### Why Two State Machines?

The separation is **intentional**:

1. **Resilience**: Recording infrastructure can recover from errors without affecting API state
2. **Granularity**: Infrastructure has 8 internal phases; API only needs 6
3. **Decoupling**: Infrastructure verification doesn't expose implementation details to API consumers

**Synchronization pattern**: Session queries recording state at boundaries:
```typescript
sessionManager.setSessionPhase(sessionId,
  session.pipelineManager?.isRecording() ? 'recording' : 'ready'
);
```

## Generation Tracking

Three counters track different aspects of recording:

| Counter | Purpose | Scope |
|---------|---------|-------|
| `totalGenerations` | "How many recordings this session?" | Session lifetime |
| `RecordingData.generation` | "Which recording is this?" | Per-recording ID |
| `sequenceNum` | "What order is this event?" | Event ordering |

These serve different purposes and cannot be consolidated.

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
