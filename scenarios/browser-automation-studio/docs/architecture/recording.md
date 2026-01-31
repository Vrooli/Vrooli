# Recording Architecture: Actions vs History

_Last reviewed: 2026-01-30_

## Overview

The browser-automation-studio has two distinct but integrated recording mechanisms:

1. **Action Recording** - Captures user interactions (clicks, typing, scrolling, etc.)
2. **History Recording** - Captures navigation events (page loads, URL changes, page creation/closure)

Both flow through a **unified Timeline** that interleaves actions and page events chronologically.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              UNIFIED TIMELINE                                │
│  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐         │
│  │ Action  │  │ Page    │  │  Action  │  │ Action  │  │ Page     │   ...   │
│  │ (click) │→ │ (nav)   │→ │  (type)  │→ │ (click) │→ │ (close)  │         │
│  └─────────┘  └─────────┘  └──────────┘  └─────────┘  └──────────┘         │
│       ↑             ↑                                       ↑               │
└───────┼─────────────┼───────────────────────────────────────┼───────────────┘
        │             │                                       │
   ACTION RECORDING   │                              HISTORY RECORDING
        │             │                                       │
┌───────┴───────┐     │                              ┌────────┴────────┐
│ User Actions  │     │                              │  Page Events    │
│ • click       │     │                              │  • page_created │
│ • type        │     │                              │  • page_navigated│
│ • scroll      │     │                              │  • page_closed  │
│ • hover       │     │                              └─────────────────┘
│ • dragDrop    │     │
└───────────────┘     │
                      │
              ┌───────┴───────┐
              │  Navigation   │
              │  Actions      │
              │  (both!)      │
              │  • navigate   │
              │  • reload     │
              │  • goBack     │
              │  • goForward  │
              └───────────────┘
```

## Data Flow: CLI → API → Storage

```
┌─────────────────────────────────────────────────────────────────┐
│                    USER/CLI LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│  cli/                                                           │
│  - recordings/command.go                                        │
│  - executions/command.go                                        │
│  - internal/api/request.go (HTTP client wrapper)                │
└──────────────────────┬──────────────────────────────────────────┘
                       │ HTTP Requests
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    API HANDLER LAYER                            │
├─────────────────────────────────────────────────────────────────┤
│  api/handlers/                                                  │
│  - record_mode.go (main handler, ~2000 lines)                   │
│  - record_mode_types.go (request/response types)                │
│  - record_mode_pages.go (page lifecycle)                        │
│  - recordings.go (import/export)                                │
│  - session_profiles.go (history/tab persistence)                │
└──────────────────────┬──────────────────────────────────────────┘
                       │ Delegates to Service Layer
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    SERVICE LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│  api/services/live-capture/                                     │
│  - service.go (orchestrator)                                    │
│  - timeline.go (timeline management & deduplication)            │
│  - workflow_generator.go (convert actions → workflows)          │
│  - action_registry.go (action type mapping)                     │
└──────────────────────┬──────────────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────────┐
│  Session     │ │  Timeline    │ │  Playwright      │
│  Manager     │ │  Service     │ │  Driver Client   │
│              │ │              │ │  (Forward calls) │
└──────────────┘ └──────────────┘ └──────────────────┘
         │             │                      │
         └─────────────┼──────────────────────┘
                       │ WebSocket Broadcasts & Storage
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    STORAGE LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│  - api/services/archive-ingestion/session_profiles.go           │
│  - Database (PostgreSQL via catalog service)                    │
│  - Session profile storage (cookies, storage state, history)    │
└─────────────────────────────────────────────────────────────────┘
```

## Data Structures

### Action Recording - RecordedAction

**File**: [CODE: api/automation/driver/types.go:149-171]

```
┌─────────────────────────┐
│ RecordedAction          │
├─────────────────────────┤
│ ID: uuid                │
│ SessionID               │
│ SequenceNum: 1,2,3...   │
│ Timestamp               │
│ ActionType: click|type  │
│ Confidence: 0.0-1.0     │
│ Selector: {             │
│   Primary: "css"        │
│   Candidates: [...]     │
│ }                       │
│ ElementMeta: tag,id,etc │
│ BoundingBox: x,y,w,h    │
│ Payload: {action data}  │
│ URL                     │
│ PageID                  │
│ PageTitle               │
└─────────────────────────┘
```

### History Recording - PageEvent + SessionProfile

**File**: [CODE: api/domain/page.go]

```
┌─────────────────────────┐             ┌─────────────────────────┐
│ PageEvent               │             │ SessionProfile          │
├─────────────────────────┤             │ (persistence)           │
│ ID: uuid                │             ├─────────────────────────┤
│ Type: page_created      │             │ ID                      │
│       page_navigated    │             │ Name                    │
│       page_closed       │             │ StorageState (cookies,  │
│ PageID                  │────────────▶│   localStorage)         │
│ URL                     │             │ History: [              │
│ Title                   │             │   {URL, Title, Time}    │
│ Timestamp               │             │ ]                       │
└─────────────────────────┘             │ OpenTabs: [{state}]     │
                                        └─────────────────────────┘
```

### Unified Timeline Entry

**File**: [CODE: api/domain/timeline.go]

```
┌─────────────────────────────────────────────────────────────┐
│                      TimelineEntry                           │
├─────────────────────────────────────────────────────────────┤
│ ID: uuid                                                     │
│ Type: "action" | "page_event"                                │
│ Timestamp: time.Time                                         │
│ PageID: uuid                                                 │
├─────────────────────────────────────────────────────────────┤
│ Action: *RecordedActionEntry    (if Type == "action")        │
│   └─ ID, ActionType, URL, SequenceNum, Timestamp,            │
│      Selector, Payload, Confidence, PageTitle                │
├─────────────────────────────────────────────────────────────┤
│ PageEvent: *PageEvent           (if Type == "page_event")    │
│   └─ ID, Type, PageID, URL, Title, OpenerID, Timestamp       │
└─────────────────────────────────────────────────────────────┘
```

## Recording Flow in Detail

```
                    USER INTERACTS WITH BROWSER
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
     ┌─────────────────┐             ┌─────────────────┐
     │ Mouse/Keyboard  │             │ Navigation      │
     │ Events          │             │ Events          │
     └────────┬────────┘             └────────┬────────┘
              │                               │
              ▼                               ▼
     ┌─────────────────┐             ┌─────────────────┐
     │ Playwright      │             │ Page Lifecycle  │
     │ Driver Capture  │             │ Monitoring      │
     └────────┬────────┘             └────────┬────────┘
              │                               │
              │  POST /action                 │  WebSocket broadcast
              ▼                               ▼
     ┌─────────────────────────────────────────────────┐
     │            record_mode.go Handler               │
     │  ┌─────────────────┐   ┌─────────────────────┐  │
     │  │ReceiveRecording │   │ Page Event Handlers │  │
     │  │Action()         │   │ (create/nav/close)  │  │
     │  └────────┬────────┘   └──────────┬──────────┘  │
     └───────────┼────────────────────────┼────────────┘
                 │                        │
                 ▼                        ▼
     ┌─────────────────────────────────────────────────┐
     │         live-capture/timeline.go                │
     │  ┌─────────────────────────────────────────┐    │
     │  │ AddAction()        AddPageEvent()       │    │
     │  │  • Dedup navigate  • Track page state   │    │
     │  │  • Assign PageID   • Emit page_created  │    │
     │  │  • Store in        • Emit page_navigated│    │
     │  │    timeline        • Emit page_closed   │    │
     │  └─────────────────────────────────────────┘    │
     └────────────────────────┬────────────────────────┘
                              │
                              ▼
     ┌─────────────────────────────────────────────────┐
     │               UNIFIED TIMELINE                  │
     │  map[sessionID][]TimelineEntry                  │
     └────────────────────────┬────────────────────────┘
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
     ┌─────────────────┐             ┌─────────────────┐
     │ WebSocket       │             │ Workflow        │
     │ Broadcast to UI │             │ Generation      │
     │ (real-time)     │             │ (on demand)     │
     └─────────────────┘             └─────────────────┘
```

## Key Distinction: Action vs History Recording

| Aspect | Action Recording | History Recording |
|--------|------------------|-------------------|
| **What** | User interactions | Page lifecycle |
| **Examples** | click, type, scroll, hover | page_created, page_navigated, page_closed |
| **Source** | Playwright event capture | Page lifecycle hooks |
| **Storage** | Timeline (ephemeral) | Timeline + SessionProfile (persistent) |
| **Purpose** | Replay interactions | Restore tabs/cookies, track navigation |
| **Deduplication** | Navigate actions within 500ms | None needed |

## Navigation Events: The Overlap

Navigation creates BOTH an action AND a page event:

```
User clicks link or API handler called
              │
              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Navigation Event                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────────┐    ┌─────────────────────────────┐ │
│  │ RecordedAction      │    │ PageEvent                   │ │
│  │ ActionType:         │    │ Type: page_navigated        │ │
│  │   navigate |        │    │ PageID: uuid                │ │
│  │   reload |          │    │ URL: new location           │ │
│  │   goBack |          │    │ Title: page title           │ │
│  │   goForward         │    │ Timestamp: now              │ │
│  └─────────────────────┘    └─────────────────────────────┘ │
│            │                           │                     │
│            ▼                           ▼                     │
│      Timeline Entry              WebSocket Broadcast         │
│      (for replay)                (for UI update)             │
└─────────────────────────────────────────────────────────────┘
```

## Page Lifecycle (Multi-Tab Support)

**File**: [CODE: api/automation/session/pages.go]

```
┌────────────┐     user opens tab      ┌──────────────┐
│ Page Created─────────────────────────▶Page Navigated│
└────────────┘     (page_created)      └──────┬───────┘
                                              │
                                     user navigates
                                     (page_navigated)
                                              │
                   ┌──────────────────────────┼──────────────────────┐
                   ▼                          ▼                      ▼
            (repeat for each nav)      (user closes tab)    (page_closed)
                                              │
                                    ┌─────────▼──────────┐
                                    │ Page Status=closed │
                                    └────────────────────┘
```

### PageTracker Implementation

```go
type PageTracker struct {
    pages          map[uuid.UUID]*domain.Page     // All pages by Vrooli ID
    activePageID   uuid.UUID                      // Currently active page
    driverToVrooli map[string]uuid.UUID          // Driver ID → Vrooli ID
    vrooliToDriver map[uuid.UUID]string          // Vrooli ID → Driver ID
}
```

## Deduplication Logic

**Problem**: When user navigates via API handler AND browser JavaScript, two navigate events fire.

**Solution**: Deduplicate within 500ms window (in [CODE: api/services/live-capture/timeline.go])

```go
const duplicateNavigateThreshold = 500 * time.Millisecond

// Skip duplicate navigate actions to the same URL
if action.ActionType == "navigate" && action.URL != "" {
    for i := len(entries) - 1; i >= len(entries)-5; i-- {  // Check last 5
        existing := entries[i]
        if existing.Action.ActionType == "navigate" &&
           existing.Action.URL == action.URL &&
           ts.Sub(existing.Timestamp) < duplicateNavigateThreshold {
            return  // Duplicate! Skip it
        }
    }
}
```

## Session Profile Persistence

**File**: [CODE: api/services/archive-ingestion/session_profiles.go]

```go
type SessionProfile struct {
    ID              string
    Name            string
    CreatedAt       time.Time
    UpdatedAt       time.Time
    LastUsedAt      time.Time
    StorageState    json.RawMessage       // Cookies, localStorage, etc
    BrowserProfile  *BrowserProfile       // Fingerprint, behavior, proxy
    History         []HistoryEntry        // Navigation history (newest first)
    HistorySettings *HistorySettings      // When to capture history
    OpenTabs        []TabState            // Tabs to restore
}
```

### When History is Captured

| Event | Handler | Purpose |
|-------|---------|---------|
| Tab Restoration | `CreateRecordingSession` | Restore previously open tabs |
| Session Close | `CloseRecordingSession` | Save final state |
| Navigation (initial) | `CreateRecordingSession` | Capture starting URL |
| Stop Recording | `StopLiveRecording` | Persist timeline |
| Manual Persist | `PersistRecordingSession` | On-demand save |

## Supported Action Types

**File**: [CODE: api/automation/actions/types.go]

| Category | Action Types |
|----------|-------------|
| **Navigation** | `navigate` |
| **Mouse** | `click`, `hover`, `dragDrop`, `scroll` |
| **Keyboard** | `type`, `keyboard`, `shortcut` |
| **Focus** | `focus`, `blur` |
| **Forms** | `select`, `uploadFile` |
| **Waiting** | `wait`, `assert`, `screenshot` |
| **Data** | `extract`, `evaluate`, `setVariable` |
| **Page Mgmt** | `tabSwitch`, `frameSwitch` |
| **Control** | `conditional`, `loop`, `subflow` |
| **Storage** | `setCookie`, `getCookie`, `clearCookie`, `setStorage`, `getStorage`, `clearStorage` |
| **Network** | `networkMock` |
| **Custom** | `custom` |

## Key Files Summary

| Layer | File | Purpose |
|-------|------|---------|
| **Handler** | [CODE: api/handlers/record_mode.go] | Routes actions & page events |
| **Handler Types** | [CODE: api/handlers/record_mode_types.go] | Request/response types |
| **Service** | [CODE: api/services/live-capture/timeline.go] | Unified timeline + dedup |
| **Service** | [CODE: api/services/live-capture/workflow_generator.go] | Action → workflow conversion |
| **Types** | [CODE: api/automation/driver/types.go:149-171] | RecordedAction struct |
| **Domain** | [CODE: api/domain/timeline.go] | TimelineEntry, PageEvent types |
| **Domain** | [CODE: api/domain/page.go] | Page, PageStatus types |
| **Session** | [CODE: api/automation/session/pages.go] | Multi-page tracking |
| **Persist** | [CODE: api/services/archive-ingestion/session_profiles.go] | History persistence |

## Workflow Generation from Recording

**Handler**: `GenerateWorkflowFromRecording` in [CODE: api/handlers/record_mode.go]

```
1. User calls POST /api/v1/recordings/live/{sessionId}/generate-workflow
2. Fetch recorded actions from timeline
3. Apply action merging:
   - Consecutive type actions → merge text
   - Consecutive scrolls → use final position
   - Remove focus events before type on same element
4. Insert smart wait nodes between actions
5. Build flow definition (nodes + edges for visual workflow)
6. Create workflow via catalog service
7. Return workflow ID to user
```

## Telemetry Pipeline

```
driver.RecordedAction
         │
         ▼
RecordedActionToTelemetry()
         │
         ▼
ActionTelemetry (unified intermediate)
         │
         ▼
TelemetryToTimelineEntry()
         │
         ▼
bastimeline.TimelineEntry (proto)
         │
         ▼
protojson.Marshal() with snake_case names
         │
         ▼
WebSocket broadcast to UI
```

## Architectural Patterns

1. **Unified Timeline**: Actions AND page events interleaved chronologically
2. **Deduplication**: Navigate actions deduplicated within 500ms window
3. **Multi-Page**: Bidirectional mapping between driver page IDs and Vrooli UUIDs
4. **Dual Format Broadcast**: WebSocket sends both legacy RecordedAction AND proto TimelineEntry
5. **Smart Merging**: Consecutive type/scroll actions merged during workflow generation
6. **Persistence**: Session profiles store cookies, history, browser profile, open tabs
7. **Mode-Aware Session**: Same Session wrapper used for recording OR execution

## Related Documentation

- [DOC: docs/architecture/execution.md] - Workflow execution architecture
- [DOC: docs/architecture/ai-navigation.md] - AI navigation architecture
- [DOC: docs/SYSTEM_ARCHITECTURE.md] - Complete system overview
- [DOC: docs/plans/RECORD_MODE_IMPLEMENTATION_PLAN.md] - Original implementation plan
- [DOC: docs/plans/multi-tab-recording-implementation.md] - Multi-tab support
