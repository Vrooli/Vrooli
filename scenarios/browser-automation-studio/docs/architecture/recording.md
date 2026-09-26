# Recording Architecture: Actions vs History

_Last reviewed: 2026-01-31_

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
│  - service.go (orchestrator, routes to unified recording)       │
│  - workflow_generator.go (convert actions → workflows)          │
│  - action_registry.go (action type mapping)                     │
│                                                                  │
│  api/services/recording/                                        │
│  - service.go (unified recording: dedup, cache, persistence)    │
│  - persistence/ (SQLite repository)                             │
└──────────────────────┬──────────────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         ▼             ▼             ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────────┐
│  Session     │ │  Unified     │ │  Playwright      │
│  Manager     │ │  Recording   │ │  Driver Client   │
│              │ │  Service     │ │  (Forward calls) │
└──────────────┘ └──────────────┘ └──────────────────┘
         │             │                      │
         └─────────────┼──────────────────────┘
                       │ WebSocket Broadcasts & Storage
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    STORAGE LAYER                                │
├─────────────────────────────────────────────────────────────────┤
│  - api/services/archive-ingestion/session_profiles.go           │
│  - Database (SQLite via recording/persistence)                  │
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
     │      services/recording/service.go             │
     │  ┌─────────────────────────────────────────┐    │
     │  │ RecordAction()    RecordPageEvent()     │    │
     │  │  • Dedup navigate  • Track page state   │    │
     │  │  • Assign PageID   • Emit page_created  │    │
     │  │  • Hot cache +     • Emit page_navigated│    │
     │  │    DB persist      • Emit page_closed   │    │
     │  └─────────────────────────────────────────┘    │
     └────────────────────────┬────────────────────────┘
                              │
                              ▼
     ┌─────────────────────────────────────────────────┐
     │           HOT CACHE + DATABASE                  │
     │  In-memory cache + SQLite persistence           │
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
| **Storage** | Hot cache + SQLite DB | Hot cache + SQLite DB + SessionProfile |
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

## Journal Identity

Retries use the observed action ID. The repository assigns one durable sequence
inside the insert and rejects conflicting reuse of an ID. Separate observations
to the same URL remain separate; timestamp proximity is not a deduplication key.

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
| **Service** | [CODE: api/services/recording/service.go] | Unified recording + dedup + persistence |
| **Service** | [CODE: api/services/live-capture/service.go] | Session orchestration, routes to unified recording |
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
3. Validate replay context and join complete drag phases. Unknown action types,
   incomplete drags, multiple/ambiguous pages and frame observations fail explicitly.
4. Coalesce full snapshots only within the same page/frame/URL/selector:
   - Input snapshots → final complete value (including empty), preserving submit boundaries
   - Absolute scroll snapshots with matching axes → final coordinates
   - Focus immediately before input on the same target → input supplies focus
   Raw recording payloads remain unchanged.
5. Use compiler parameter builders to create typed WorkflowDefinitionV2 nodes and
   insert typed smart waits. Modifiers, click counts, focus/blur and drag/drop
   retain their action semantics through compilation.
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

## Unified Recording Architecture

The recording system uses a **unified recording service** that handles all action and page event
persistence. Both manual recording (user interactions) and AI navigation actions flow through
the same recording pipeline, ensuring consistency.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         UNIFIED RECORDING ARCHITECTURE                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   MANUAL RECORDING              AI NAVIGATION                               │
│   (user clicks in browser)      (navigator issues commands)                 │
│          │                              │                                    │
│          │ HTTP callback                │ Step callback                      │
│          ▼                              ▼                                    │
│   ┌─────────────────────────────────────────────────────────────────┐       │
│   │         LIVE-CAPTURE SERVICE (api/services/live-capture/)        │       │
│   │                                                                  │       │
│   │   AddTimelineAction(sessionID, action, pageID)                  │       │
│   │     • Routes to unified recording service                       │       │
│   │     • Source determined from payload["source"]                  │       │
│   │       - "ai" → ActionSourceAI                                   │       │
│   │       - default → ActionSourceManual                            │       │
│   │                                                                  │       │
│   └─────────────────────────────────────────────────────────────────┘       │
│                              │                                               │
│                              ▼                                               │
│   ┌─────────────────────────────────────────────────────────────────┐       │
│   │           UNIFIED RECORDING SERVICE                              │       │
│   │           (api/services/recording/service.go)                    │       │
│   │                                                                  │       │
│   │   • RecordAction(sessionID, action, pageID, source)             │       │
│   │   • RecordPageEvent(sessionID, event)                           │       │
│   │   • Single deduplication (500ms navigate threshold)             │       │
│   │   • Durable journal append (timeline_entries)               │       │
│   │   • WebSocket broadcast                                         │       │
│   └─────────────────────────────────────────────────────────────────┘       │
│                              │                                               │
│                              ▼                                               │
│   ┌─────────────────────────────────────────────────────────────────┐       │
│   │           PERSISTENCE LAYER                                      │       │
│   │           (api/services/recording/persistence/)                  │       │
│   │                                                                  │       │
│   │   Database Table: timeline_entries                               │       │
│   │   • id, session_id, page_id, sequence                           │       │
│   │   • type (action | page_event)                                  │       │
│   │   • action_data, page_event_data (JSON)                         │       │
│   │   • timestamp, created_at                                       │       │
│   └─────────────────────────────────────────────────────────────────┘       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Driver/Navigator Callback**: Actions arrive via HTTP POST (manual) or step callback (AI)
2. **Live-Capture Service**: [CODE: api/services/live-capture/service.go] routes to unified recording
3. **Deduplication**: [CODE: api/services/recording/service.go] filters duplicate navigate actions (500ms)
4. **Persistence**: Commits through the repository before notification
5. **Broadcast**: WebSocket listeners notified of new actions

### Database Schema

**File**: [CODE: api/internal/<domain>/storage/sqlite/schemas/]

```sql
-- Recording sessions (aggregate root)
CREATE TABLE recording_sessions (
    id TEXT PRIMARY KEY,
    profile_id TEXT,                              -- Optional link to SessionProfile
    status TEXT NOT NULL DEFAULT 'active',        -- active | closed
    viewport_width INTEGER,
    viewport_height INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Recording actions (persisted interactions)
CREATE TABLE recording_actions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES recording_sessions(id) ON DELETE CASCADE,
    page_id TEXT NOT NULL,
    sequence_num INTEGER NOT NULL,
    action_type TEXT NOT NULL,                    -- click, type, navigate, etc.
    timestamp TIMESTAMP NOT NULL,
    duration_ms INTEGER,
    selector TEXT,                                -- JSON: SelectorSet
    element_meta TEXT,                            -- JSON: ElementMeta
    bounding_box TEXT,                            -- JSON: BoundingBox
    payload TEXT,                                 -- JSON: action-specific data
    url TEXT,
    page_title TEXT,
    confidence REAL DEFAULT 1.0,
    source TEXT DEFAULT 'auto',                   -- auto | manual | ai_suggested
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id, sequence_num)
);
```

### Domain Entities

#### RecordingAction
**File**: [CODE: api/domain/action.go]

```go
type RecordingAction struct {
    ID              uuid.UUID
    SessionID       string
    PageID          uuid.UUID
    SequenceNum     int
    ActionType      string                 // click, type, navigate, etc.
    Timestamp       time.Time
    DurationMs      int
    Selector        *SelectorSet
    ElementMeta     *ElementMeta
    BoundingBox     *BoundingBox
    Payload         map[string]interface{}
    URL             string
    PageTitle       string
    Confidence      float64
    Source          ActionSource           // auto, manual, ai_suggested
    CreatedAt       time.Time
}
```

#### RecordingSession
**File**: [CODE: api/domain/session.go]

```go
type RecordingSession struct {
    ID              string
    ProfileID       string                 // Optional link to SessionProfile
    Status          SessionStatus          // active | closed
    ViewportWidth   int
    ViewportHeight  int
    CreatedAt       time.Time
    ClosedAt        *time.Time
    ActionCount     int                    // Computed on read
}
```

### Repository Interface

[CODE: api/services/recording/persistence/repository.go] owns session lifecycle,
atomic `AppendTimelineEntry`, single-entry reads, complete filtered/offset history
queries, counts and cleanup. Its SQLite adapter hides the engine from the service.
Both production and leased test pools install internal/recording/schema.sql.

### Query Patterns

**File**: [CODE: api/recording/persistence/repository.go]

```go
type ActionQuery struct {
    SessionID     string              // Required: filter by session
    PageID        *uuid.UUID          // Optional: filter by page
    ActionTypes   []string            // Optional: filter by action type
    StartTime     *time.Time          // Optional: actions after this time
    EndTime       *time.Time          // Optional: actions before this time
    Source        *domain.ActionSource // Optional: filter by source
    MinConfidence *float64            // Optional: minimum confidence threshold
    Limit         int                 // Max results (default 100, max 1000)
    Offset        int                 // Pagination offset
}
```

### Commit and Query Authority

The service appends through the repository before returning success or notifying
subscribers. Storage failure propagates through HTTP ingestion and suppresses its
broadcast. Query totals and offsets always come from durable history; there is no
process-local history or sequence cache. A missing repository is an error.

### Relationship to Existing Components

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                               SessionProfile                                 │
│                         [CODE: api/services/archive-ingestion/]             │
│   - History[] (navigation breadcrumbs)                                      │
│   - StorageState (cookies, localStorage)                                    │
│   - BrowserProfile (anti-detection)                                         │
│   - OpenTabs[] (tab restoration)                                            │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │ 1:N (profile has many sessions)
                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           RecordingSession                                   │
│                     [CODE: api/services/recording/]                         │
│   - ID, ProfileID (optional link)                                           │
│   - Status (active/closed)                                                  │
│   - ViewportWidth/Height                                                    │
│   - CreatedAt/ClosedAt                                                      │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │ 1:N (session has many actions)
                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           RecordingAction                                    │
│                     [CODE: api/services/recording/]                         │
│   - ID, SessionID, PageID, SequenceNum                                      │
│   - ActionType (click, type, navigate, etc.)                                │
│   - Selector, ElementMeta, Payload                                          │
│   - Confidence, Source (manual | ai)                                        │
└──────────────────────────────────┬──────────────────────────────────────────┘
                                   │ Timeline view
                                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            TimelineEntry                                     │
│                         [CODE: api/domain/timeline.go]                      │
│   - Unified view of Action + PageEvent                                      │
│   - Used for UI display and workflow generation                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Key Files for Persistence

| Layer | File | Purpose |
|-------|------|---------|
| **Domain** | [CODE: api/domain/action.go] | RecordingAction, ActionSource, SelectorSet |
| **Domain** | [CODE: api/domain/session.go] | RecordingSession, SessionStatus |
| **Service** | [CODE: api/services/recording/service.go] | Unified recording: identity, durable append and queries |
| **Persist** | [CODE: api/services/recording/persistence/repository.go] | Repository interface |
| **Persist** | [CODE: api/services/recording/persistence/sqlite.go] | SQLite implementation |
| **Live-Capture** | [CODE: api/services/live-capture/service.go] | Session management, routes to unified recording |
| **Wire** | [CODE: api/internal/wire/wire.go] | Dependency injection |

## Commit Acknowledgment and Observability

`RecordAction` and `RecordPageEvent` are the single recording entrypoints. The
unused alternate ActionRecorder/Unified APIs and cache-first receipts were
removed. HTTP ingestion rejects unknown sessions and returns a storage error
before any timeline broadcast. Delivery counts describe WebSocket delivery;
they do not change the durable commit. Failed browser-operation recording is
reported explicitly after the browser effect, which cannot be rolled back.

Tests in services/recording/service_test.go use production SQLite schema, injected
write failure, more than 1,000 events, reopen, concurrent connections and retries.
handlers/record_mode_integration_test.go exercises the actual HTTP ingestion
boundary. Full driver retry/reconnect and callback gap reporting remain separate
qualification work.

## Manual Recording WebSocket Flow

The following diagram shows the complete WebSocket message flow during a manual recording session:

```
BROWSER CLIENT                    API                      PLAYWRIGHT-DRIVER
     |                             |                              |
     | subscribe_recording         |                              |
     | --------------------------> |                              |
     |                             |                              |
     |                             |  StartRecording(callbacks)   |
     |                             |  ------------------------->  |
     |                             |                              |
     |                             |                              |
     |  ================================================================
     |  RECORDING ACTIVE
     |  ================================================================
     |                             |                              |
     |                             |    <---- User clicks in      |
     |                             |          browser             |
     |                             |                              |
     |                             |    POST /action (callback)   |
     |                             |  <------------------------   |
     |                             |                              |
     |  recording_action           |                              |
     |  <------------------------- |                              |
     |  { type: "click", ... }     |                              |
     |                             |                              |
     |                             |    WS /frames (callback)    |
     |                             |  <------------------------   |
     |                             |                              |
     |  source envelope + JPEG        |                              |
     |  <------------------------- |                              |
     |                             |                              |
     |  recording_input            |                              |
     |  { type: "pointerMove" }    |                              |
     |  -------------------------> |                              |
     |                             |  POST /record/input          |
     |                             |  ------------------------->  |
     |                             |                              |
     |  ================================================================
     |  USER STOPS RECORDING
     |  ================================================================
     |                             |                              |
     |                             |  GenerateWorkflow()          |
     |                             |  ------------->              |
     |                             |                              |
     |  workflow_created           |                              |
     |  { workflowId: "..." }      |                              |
     |  <------------------------- |                              |
     |                             |                              |
```

### WebSocket Message Types

**Client -> Server:**

| Message Type | Purpose |
|--------------|---------|
| `subscribe_recording` | Join a recording session; `frames: false` selects events only (timeline), omission or `true` also receives binary frames (viewer) |
| `unsubscribe_recording` | Leave a recording session |
| `recording_input` | Forward mouse/keyboard input to driver |
| `subscribe_driver_status` | Subscribe to driver health status |

**Server -> Client:**

| Message Type | Purpose |
|--------------|---------|
| `recording_action` | Action captured (click, type, etc.) |
| `binary frame` | Four-byte big-endian header length, version 1 JSON source header, JPEG bytes |
| `page_event` | Page lifecycle (created, navigated, closed) |
| `page_switch` | Active page changed |
| `perf_stats` | Performance data (debug mode) |

## Complete Recording Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│ USER INTERACTIONS IN BROWSER                                        │
│ - Click, type, drag, navigate, etc.                                │
└──────────────────────┬──────────────────────────────────────────────┘
                       | (Native Playwright capture)
                       v
┌─────────────────────────────────────────────────────────────────────┐
│ PLAYWRIGHT-DRIVER                                                   │
│ - Observes actions via browser events                              │
│ - Collects metadata: selectors, element info                       │
│ - POSTs action JSON to callback URL                                │
│ - Captures frames on interval or demand                            │
│ - POSTs frames via WebSocket or HTTP                               │
│ - Emits page lifecycle events                                      │
└──────────────────────┬──────────────────────────────────────────────┘
                       | (HTTP POST / WebSocket)
                       v
┌─────────────────────────────────────────────────────────────────────┐
│ API HANDLER LAYER                                                   │
│ - ReceiveRecordingAction: Parse and store action                   │
│ - GetRecordingFrame: Source-validated preview                     │
│ - HandleDriverFrameStream: WebSocket binary streaming              │
│ - ForwardRecordingInput: Forward input to driver                   │
└──────────────────────┬──────────────────────────────────────────────┘
                       | (Delegates to service)
                       v
┌─────────────────────────────────────────────────────────────────────┐
│ SERVICE LAYER                                                       │
│ - Route to unified recording service                               │
│ - Deduplicate navigate actions (500ms window)                      │
│ - Commit to recording journal                         │
│ - Track pages with PageTracker                                     │
└──────────────────────┬──────────────────────────────────────────────┘
                       | (Broadcasts via WebSocket Hub)
                       v
┌─────────────────────────────────────────────────────────────────────┐
│ WEBSOCKET HUB                                                       │
│ - Broadcast action to subscribed clients                           │
│ - Broadcast source-bearing binary frames                              │
│ - Broadcast page events                                            │
│ - Forward input from clients to driver                             │
└──────────────────────┬──────────────────────────────────────────────┘
                       | (WebSocket messages)
                       v
┌─────────────────────────────────────────────────────────────────────┐
│ BROWSER CLIENT (UI)                                                 │
│ - Display recording timeline                                       │
│ - Show live video stream                                           │
│ - Accept user input (manual interaction)                           │
│ - Allow user to generate workflow from recording                   │
└─────────────────────────────────────────────────────────────────────┘

When user requests workflow generation:
  |
  v
recordModeService.GenerateWorkflow()
  |-- Get recorded actions
  |-- Convert to action nodes
  |-- Insert smart waits
  |-- Build flow definition
  v
catalogService.CreateWorkflow()
  |-- Persist workflow
  |-- Return ID
  v
Workflow available for execution
```

## Related Documentation

- [DOC: docs/architecture/driver-interface.md] - Driver interface and navigator architecture
- [DOC: docs/architecture/execution.md] - Workflow execution architecture
- [DOC: docs/architecture/ai-navigation.md] - AI navigation architecture
- [DOC: docs/SYSTEM_ARCHITECTURE.md] - Complete system overview
- [DOC: docs/plans/README.md#historical-source-files] - Original implementation plan
- [DOC: docs/plans/README.md#historical-source-files] - Multi-tab support
- [DOC: docs/internal/SEAMS.md#recording-bounded-context] - Recording integration seams


### Frame source boundary (2026-09-23)

Both driver streaming strategies send `[uint32 BE header length][JSON][JPEG]`.
The mandatory version 1 header carries `source` (`session_id`, `execution_id`,
`lease_id`, driver `page_id`) and `captured_at`. The optional `timing` object
carries performance telemetry independently. The API accepts a header of at
most 16 KiB, validates the current session lease and active driver page, and
publishes a new header with `version`, `session_id`, canonical `page_id`, and
`captured_at`. Driver credentials never reach viewers. The viewer validates
identity before image decoding. Anonymous and timestamp-only frames are rejected.

HTTP previews translate the requested canonical page to its driver identity.
The driver snapshots source before capture and keys its cache by that source;
the API revalidates the returned source before returning image data or an ETag
hit. The HTTP response includes canonical `page_id` without the producer source.
An omitted page ID follows the active page for miniature previews. The unused
JSON frame-push endpoint and `recording_frame` broadcast facade have been removed.
Same-page navigation epochs and cross-transport ordering remain separate work.


Recording subscriber intent is explicit: the timeline requests `frames: false`
on its event socket. The canvas retains its separate frame subscription and
fallback lifetime. Hub gates binary delivery and frame-subscriber presence on
that choice under the subscription lock. Page/timeline/performance events are
preserved. Resubscription replaces the choice; unsubscribe clears it.
