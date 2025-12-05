# Record Mode Implementation Plan

> **Status**: Draft
> **Created**: 2024-12-04
> **Authors**: Human + Claude collaboration
> **Target**: browser-automation-studio scenario

---

## Executive Summary

This document describes the implementation plan for "Record Mode" - a feature that allows users to create browser automation workflows by simply performing actions in a browser, rather than manually building workflows node-by-node.

**The core insight**: The friction of creating automation workflows should approach the friction of performing the task manually. When these are equal, users will naturally gravitate toward browser-automation-studio for all repetitive browser tasks.

---

## Execution Plan & Priorities

### Context & Constraints

This plan was developed with the following constraints in mind:

1. **Target audience**: External users who need to automate back-office tasks
2. **Primary goal**: Minimize churn before subscription - users who download the app should succeed quickly
3. **Timeline**: Urgent - needs to ship in ~2 weeks
4. **Quality bar**: Must be polished enough that first impressions don't cause abandonment

### Why This Execution Order?

We considered two approaches:

**Option A: Vertical Slice First** - Build minimal end-to-end quickly, iterate
**Option B: De-risk Selectors First** - Validate the hardest problem before investing

**We chose a hybrid approach** because:

1. **Selectors are the make-or-break risk**. If selectors fail >30% of the time, the entire feature is compromised. A 2-3 day spike validates feasibility before committing weeks of work.

2. **The selector editor is more important than perfect selectors**. Perfect selectors are impossible (sites vary too much), but users CAN recover if editing is easy. A user who fixes a selector and succeeds feels empowered; a user who hits a wall churns.

3. **80% of workflows need only Click, Type, Navigate**. Login flows, form submissions, and basic navigation cover most use cases. Shipping these well beats shipping everything poorly.

### Execution Phases

```
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 0: Selector Spike (2-3 days)                              │
│ • Build selector generator in isolation                         │
│ • Test on 10 diverse sites (login forms, dashboards, SPAs)      │
│ • Measure: what % survive a page refresh?                       │
│ • Decision gate: proceed if >70% reliability                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 1: Minimal Vertical Slice (5-7 days)                      │
│ • Click + Type + Navigate only (covers 80% of workflows)        │
│ • Basic timeline (display actions, delete unwanted)             │
│ • Simple workflow generation                                    │
│ • Goal: record 5-step login flow → execute successfully         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ PHASE 2: Polish for External Users (4-5 days)                   │
│ • Selector editor with live validation                          │
│ • Warnings for unstable selectors before save                   │
│ • Action editing (change type, modify parameters)               │
│ • Error states with clear messages and recovery paths           │
│ • Visual polish for professional appearance                     │
└─────────────────────────────────────────────────────────────────┘
```

**Total: ~2 weeks to shippable**

### What's Included vs. Deferred

| Included (Ship) | Deferred (Fast Follow) |
|-----------------|------------------------|
| Click, Type, Navigate | Scroll, Hover, Select, Drag |
| Basic timeline | Action merging/deduplication |
| Selector editor | Replay preview before save |
| Delete actions | Reorder actions |
| Single-page flows | iframe support |
| Manual workflow naming | AI-suggested names |

### The Key Insight

> **The selector editor is more important than perfect selectors.**
>
> - Perfect selectors are impossible (sites vary too much)
> - But users CAN recover if editing is easy
> - A user who fixes a selector and succeeds feels empowered
> - A user who hits a wall and can't proceed churns
>
> Phase 2's selector editor is non-negotiable for external users.

---

## Table of Contents

0. [Execution Plan & Priorities](#execution-plan--priorities) ← **START HERE**
1. [Vision & Problem Statement](#1-vision--problem-statement)
2. [Architecture Discovery](#2-architecture-discovery)
3. [Technical Design](#3-technical-design)
4. [Implementation Phases](#4-implementation-phases) (detailed technical reference)
5. [Selector Strategy](#5-selector-strategy)
6. [UI/UX Design](#6-uiux-design)
7. [Risk Assessment](#7-risk-assessment)
8. [Future Evolution](#8-future-evolution)
9. [File Reference](#9-file-reference)

---

## 1. Vision & Problem Statement

### 1.1 The Current Pain

browser-automation-studio currently offers two ways to create workflows:

1. **Drag-and-drop builder**: Powerful but requires upfront planning. Users must understand node types, configure parameters, and think abstractly about their workflow before seeing results.

2. **JSON editing**: Even more technical. Only suitable for power users or programmatic generation.

**The result**: High friction means users only automate highly repetitive tasks where the time investment pays off. Casual automation opportunities are missed.

### 1.2 The Vision

**Record Mode** inverts the workflow creation model:

```
Traditional:  Plan → Configure → Test → Iterate → Done
Record Mode:  Do → Review → Save → Done
```

Instead of "plan then execute," it's "execute then extract." This matches how humans actually discover automation opportunities - you notice you've done something repeatedly *after* doing it.

### 1.3 Two Evolutionary Stages

**Stage 1: Dedicated Record Mode Page** (This plan)
- User explicitly enters "Record Mode"
- Actions are captured in a timeline sidebar
- User selects actions to convert to workflow
- Clear mental model: "I'm recording right now"

**Stage 2: Browser-First Experience** (Future)
- browser-automation-studio acts like a browser with passive recording
- Home page has URL bar, actions are always captured
- User can retroactively select any session segment as a workflow
- Zero cognitive overhead - just use it as your browser

This plan focuses on Stage 1, but designs infrastructure to support Stage 2.

### 1.4 Success Criteria

1. **Time to first workflow**: < 2 minutes for a simple 5-step flow
2. **Selector reliability**: > 90% of captured selectors work on page refresh
3. **Action accuracy**: > 95% of user intentions correctly captured
4. **Perceived latency**: < 200ms from action to timeline update

---

## 2. Architecture Discovery

### 2.1 Critical Finding: The System Was Designed For This

Investigation of the codebase revealed that Record Mode was part of the original vision:

1. **`CursorTrail` field exists** in `StepOutcome` contract but is never populated by live execution
2. **Chrome extension recording import** already exists and works
3. **Recording adapter** converts external recordings to the same format as live executions
4. **WebSocket infrastructure** is production-ready for streaming events

**The gap is narrow**: Only the in-session event capture layer is missing. Everything downstream (normalization, persistence, streaming, visualization) already exists.

### 2.2 Existing Infrastructure (Reusable)

| Component | Location | Reuse Level |
|-----------|----------|-------------|
| Recording import handler | `api/handlers/recordings.go` | 100% |
| Recording adapter | `api/services/recording/adapter.go` | 90% |
| Recording types | `api/services/recording/types.go` | 100% |
| WebSocket hub | `api/websocket/` | 100% |
| Event sink | `api/automation/events/` | 95% |
| Session management | `playwright-driver/src/session/` | 100% |
| Telemetry collectors | `playwright-driver/src/telemetry/` | Pattern |
| StepOutcome contract | `api/automation/contracts/contracts.go` | 100% |
| DBRecorder | `api/automation/recorder/` | 100% |

### 2.3 What Needs To Be Built

```
                    ┌─────────────────────────────────────────────────────────────┐
                    │                    EXISTING (reusable)                       │
                    └─────────────────────────────────────────────────────────────┘
                                               ▲
                                               │
┌───────────────────────────────────────────────────────────────────────────────────┐
│                              NEEDS TO BE BUILT                                    │
│                                                                                   │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────┐  │
│  │ Event Capture   │───▶│ Action          │───▶│ Workflow Generation         │  │
│  │ (JS Injection)  │    │ Normalization   │    │ (Actions → Nodes)           │  │
│  └─────────────────┘    └─────────────────┘    └─────────────────────────────┘  │
│                                                                                   │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────────────┐  │
│  │ Record Mode UI  │    │ Timeline        │    │ Selector Generation         │  │
│  │ (Toggle/Status) │    │ Component       │    │ (Element → CSS/XPath)       │  │
│  └─────────────────┘    └─────────────────┘    └─────────────────────────────┘  │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Technical Design

### 3.1 System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                  UI Layer                                        │
│                                                                                  │
│  ┌──────────────┐  ┌──────────────────────┐  ┌─────────────────────────────┐   │
│  │ Record Mode  │  │ Live Browser View    │  │ Action Timeline Sidebar     │   │
│  │ Toggle       │  │ (Screenshot Stream)  │  │ (Real-time action list)     │   │
│  └──────────────┘  └──────────────────────┘  └─────────────────────────────┘   │
│         │                    ▲                           ▲                       │
│         │                    │                           │                       │
│         ▼                    │ Screenshots               │ Events                │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                        WebSocket Connection                               │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        │ HTTP/WS
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                  API Layer                                       │
│                                                                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────────────────┐    │
│  │ Record Mode      │  │ Session          │  │ Workflow Generation        │    │
│  │ Handler          │  │ Handler          │  │ Handler                    │    │
│  │ POST /record/*   │  │ (existing)       │  │ POST /record/generate      │    │
│  └──────────────────┘  └──────────────────┘  └────────────────────────────┘    │
│         │                                              ▲                         │
│         ▼                                              │                         │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                        Recording Service                                  │   │
│  │  • Start/Stop recording sessions                                          │   │
│  │  • Manage recording state                                                 │   │
│  │  • Convert actions to workflow nodes                                      │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│                    │                                                             │
│                    ▼                                                             │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                        Event Sink (existing)                              │   │
│  │  • Broadcasts recording.action events                                     │   │
│  │  • Persists to DB via DBRecorder                                          │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        │ HTTP
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            Playwright Driver                                     │
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                     Record Mode Controller                                │   │
│  │  • Injects JavaScript event listeners into page                           │   │
│  │  • Receives events via page.exposeFunction()                              │   │
│  │  • Normalizes raw events to semantic actions                              │   │
│  │  • Streams actions to API via callback                                    │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│         │                                                                        │
│         ▼                                                                        │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                     Injected JavaScript                                   │   │
│  │  • Listens for: click, input, keydown, scroll, focus, blur               │   │
│  │  • Captures: element metadata, coordinates, timing                        │   │
│  │  • Generates: robust selectors (ID, data-*, CSS, XPath)                  │   │
│  │  • Calls: window.__recordAction(event) exposed by Playwright              │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                     Session State (existing)                              │   │
│  │  + recordingEnabled: boolean                                              │   │
│  │  + recordingCallback: (action) => void                                    │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Data Flow: Recording an Action

```
User clicks element
        │
        ▼
┌─────────────────────────────────────────┐
│ Injected JS: document.addEventListener  │
│ • Captures click event                  │
│ • Extracts element metadata             │
│ • Generates selector candidates         │
│ • Calls window.__recordAction()         │
└─────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────┐
│ Playwright: page.exposeFunction()       │
│ • Receives action from page context     │
│ • Validates action shape                │
│ • Enriches with screenshot/DOM          │
└─────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────┐
│ Record Mode Controller                  │
│ • Normalizes to RecordedAction          │
│ • Assigns timestamp, sequence number    │
│ • Invokes callback to stream to API     │
└─────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────┐
│ API: Recording Handler                  │
│ • Receives streamed action              │
│ • Persists via DBRecorder               │
│ • Emits WebSocket event                 │
└─────────────────────────────────────────┘
        │
        ├──────────────────┐
        ▼                  ▼
┌───────────────┐  ┌───────────────┐
│ PostgreSQL    │  │ WebSocket Hub │
│ (persistence) │  │ (broadcast)   │
└───────────────┘  └───────────────┘
                           │
                           ▼
                   ┌───────────────┐
                   │ UI: Timeline  │
                   │ (live update) │
                   └───────────────┘
```

### 3.3 New Event Types

Add to `api/automation/contracts/events.go`:

```go
const (
    // Existing events...

    // Record Mode events
    EventKindRecordingStarted   EventKind = "recording.started"
    EventKindRecordingAction    EventKind = "recording.action"
    EventKindRecordingStopped   EventKind = "recording.stopped"
    EventKindRecordingError     EventKind = "recording.error"
)
```

### 3.4 RecordedAction Contract

New type for `api/automation/contracts/recording.go`:

```go
// RecordedAction represents a single user action captured during recording.
// This is the primary data structure flowing through the recording pipeline.
type RecordedAction struct {
    // Identity
    ID            string    `json:"id"`                       // Unique action ID (UUID)
    SessionID     string    `json:"session_id"`               // Recording session ID
    SequenceNum   int       `json:"sequence_num"`             // Order in recording

    // Timing
    Timestamp     time.Time `json:"timestamp"`                // When action occurred
    DurationMs    int       `json:"duration_ms,omitempty"`    // For actions with duration (typing)

    // Action classification
    ActionType    string    `json:"action_type"`              // click, type, scroll, navigate, etc.
    Confidence    float64   `json:"confidence"`               // 0-1, how confident we are in classification

    // Target element
    Selector      *SelectorSet `json:"selector,omitempty"`    // Multiple selector strategies
    ElementMeta   *ElementMeta `json:"element_meta,omitempty"` // Tag, text, attributes
    BoundingBox   *BoundingBox `json:"bounding_box,omitempty"`

    // Action-specific data
    Payload       map[string]any `json:"payload,omitempty"`   // Type-specific params
    // Examples:
    // - click: { button: "left", modifiers: ["ctrl"] }
    // - type: { text: "hello", delay: 50 }
    // - scroll: { deltaX: 0, deltaY: 500 }
    // - navigate: { url: "https://..." }

    // Context
    URL           string    `json:"url"`                      // Page URL at action time
    FrameID       string    `json:"frame_id,omitempty"`       // If in iframe

    // Visual artifacts
    Screenshot    *Screenshot `json:"screenshot,omitempty"`   // State after action
    CursorPos     *Point      `json:"cursor_pos,omitempty"`   // Cursor position
}

// SelectorSet contains multiple selector strategies for resilience.
type SelectorSet struct {
    Primary     string            `json:"primary"`              // Best selector
    Candidates  []SelectorCandidate `json:"candidates"`         // Ranked alternatives
}

// SelectorCandidate is a single selector with metadata.
type SelectorCandidate struct {
    Type       string  `json:"type"`       // id, data-testid, css, xpath, text, aria
    Value      string  `json:"value"`      // The actual selector
    Confidence float64 `json:"confidence"` // 0-1, likelihood of uniqueness/stability
    Specificity int    `json:"specificity"` // Higher = more specific
}

// ElementMeta captures information about the target element.
type ElementMeta struct {
    TagName     string            `json:"tag_name"`
    ID          string            `json:"id,omitempty"`
    ClassName   string            `json:"class_name,omitempty"`
    InnerText   string            `json:"inner_text,omitempty"`   // Truncated
    Attributes  map[string]string `json:"attributes,omitempty"`   // Relevant attrs
    IsVisible   bool              `json:"is_visible"`
    IsEnabled   bool              `json:"is_enabled"`
}
```

### 3.5 API Endpoints

New endpoints for record mode:

```
POST   /api/v1/sessions/{id}/record/start
  → Starts recording on existing session
  ← { recording_id: string }

POST   /api/v1/sessions/{id}/record/stop
  → Stops recording
  ← { recording_id: string, action_count: int }

GET    /api/v1/recordings/{id}
  → Get recording metadata
  ← { id, session_id, started_at, stopped_at, action_count }

GET    /api/v1/recordings/{id}/actions
  → Get all recorded actions
  ← { actions: RecordedAction[] }

POST   /api/v1/recordings/{id}/generate-workflow
  → Convert recording to workflow
  Body: {
    name: string,
    project_id: string,
    action_range?: { start: int, end: int },  // Optional subset
    options?: { ... }
  }
  ← { workflow_id: string }

DELETE /api/v1/recordings/{id}
  → Delete recording and actions
  ← { deleted: true }
```

---

## 4. Implementation Phases

> **Note**: This section contains detailed technical specifications for each component. For the actual execution order and timeline, see [Execution Plan & Priorities](#execution-plan--priorities) at the top of this document.
>
> **Mapping to Execution Phases:**
> - **Phase 0 (Selector Spike)**: Uses code from Phase 2 below (Selector Generation)
> - **Phase 1 (Minimal Vertical Slice)**: Combines Phase 1 (Event Capture), Phase 3 (Timeline UI - basic), Phase 4 (Workflow Generation)
> - **Phase 2 (Polish)**: Phase 3 (Timeline UI - editing), Phase 5 (Integration & Polish)

### Technical Phase 1: Event Capture Infrastructure (Foundation)

**Goal**: Capture user interactions in the browser and stream to API.

**Duration estimate**: Medium complexity

**Files to create/modify**:

```
playwright-driver/src/
├── recording/
│   ├── controller.ts        # NEW: Record mode orchestration
│   ├── injector.ts          # NEW: JavaScript injection
│   ├── normalizer.ts        # NEW: Raw events → RecordedAction
│   └── selectors.ts         # NEW: Selector generation
├── session/
│   └── manager.ts           # MODIFY: Add recording state to SessionState
└── handlers/
    └── record-mode.ts       # NEW: HTTP handlers for record control

api/
├── automation/contracts/
│   ├── recording.go         # NEW: RecordedAction, SelectorSet types
│   └── events.go            # MODIFY: Add recording event kinds
├── handlers/
│   └── record_mode.go       # NEW: HTTP handlers
└── services/recording/
    ├── live_recorder.go     # NEW: Manages live recording sessions
    └── action_store.go      # NEW: Persists recorded actions
```

**Key implementation details**:

1. **JavaScript Injection** (`injector.ts`):
```typescript
export function getRecordingScript(): string {
  return `
    (function() {
      // Prevent double-injection
      if (window.__recordingActive) return;
      window.__recordingActive = true;

      const captureAction = (type, event, extra = {}) => {
        const target = event.target;
        const selectors = generateSelectors(target);
        const meta = extractElementMeta(target);

        window.__recordAction({
          actionType: type,
          timestamp: Date.now(),
          selector: selectors,
          elementMeta: meta,
          boundingBox: target.getBoundingClientRect(),
          cursorPos: { x: event.clientX, y: event.clientY },
          url: window.location.href,
          frameId: window.frameElement?.id || null,
          payload: extra
        });
      };

      // Click handling
      document.addEventListener('click', (e) => {
        captureAction('click', e, {
          button: e.button === 0 ? 'left' : e.button === 2 ? 'right' : 'middle',
          modifiers: getModifiers(e)
        });
      }, true);

      // Input handling (debounced)
      let inputBuffer = '';
      let inputTarget = null;
      let inputTimeout = null;

      document.addEventListener('input', (e) => {
        if (inputTarget !== e.target) {
          flushInput();
          inputTarget = e.target;
        }
        inputBuffer = e.target.value;
        clearTimeout(inputTimeout);
        inputTimeout = setTimeout(flushInput, 500);
      }, true);

      function flushInput() {
        if (inputBuffer && inputTarget) {
          captureAction('type', { target: inputTarget, clientX: 0, clientY: 0 }, {
            text: inputBuffer
          });
        }
        inputBuffer = '';
        inputTarget = null;
      }

      // Scroll handling (debounced)
      let scrollTimeout = null;
      document.addEventListener('scroll', (e) => {
        clearTimeout(scrollTimeout);
        scrollTimeout = setTimeout(() => {
          captureAction('scroll', e, {
            scrollX: window.scrollX,
            scrollY: window.scrollY
          });
        }, 150);
      }, true);

      // Navigation is captured differently (via page events)

      // ... more event handlers ...

      function generateSelectors(element) { /* See Section 5 */ }
      function extractElementMeta(element) { /* ... */ }
      function getModifiers(e) {
        return [e.ctrlKey && 'ctrl', e.shiftKey && 'shift',
                e.altKey && 'alt', e.metaKey && 'meta'].filter(Boolean);
      }
    })();
  `;
}
```

2. **Controller** (`controller.ts`):
```typescript
export class RecordModeController {
  private session: SessionState;
  private isRecording: boolean = false;
  private actionCallback: (action: RecordedAction) => void;

  async startRecording(callback: (action: RecordedAction) => void): Promise<void> {
    this.actionCallback = callback;
    this.isRecording = true;

    // Expose callback function to page context
    await this.session.page.exposeFunction('__recordAction', (rawAction: any) => {
      const normalized = this.normalizeAction(rawAction);
      this.actionCallback(normalized);
    });

    // Inject recording script
    await this.session.page.evaluate(getRecordingScript());

    // Re-inject on navigation
    this.session.page.on('load', async () => {
      if (this.isRecording) {
        await this.session.page.evaluate(getRecordingScript());
      }
    });
  }

  async stopRecording(): Promise<void> {
    this.isRecording = false;
    // Clean up injected script
    await this.session.page.evaluate(() => {
      window.__recordingActive = false;
    });
  }
}
```

**Deliverables**:
- [ ] JavaScript injection that captures click, type, scroll events
- [ ] `page.exposeFunction()` callback pipeline
- [ ] Action normalization with timestamps and sequence numbers
- [ ] HTTP endpoints for start/stop recording
- [ ] WebSocket event broadcasting for recorded actions
- [ ] Persistence of recorded actions to database

**Test criteria**:
- Record a click → action appears in database
- Record typing → debounced, single action captured
- Record scroll → debounced, scroll position captured
- Navigate to new page → recording continues, script re-injected

---

### Technical Phase 2: Selector Generation (Critical Path)

**Goal**: Generate robust, reliable selectors that survive page changes.

**Duration estimate**: High complexity (this is the hardest part)

**Files to create/modify**:

```
playwright-driver/src/recording/
└── selectors.ts             # NEW: Multi-strategy selector generation
```

**Selector strategies** (in priority order):

1. **Unique ID**: `#submit-button`
2. **data-testid**: `[data-testid="login-form"]`
3. **data-* attributes**: `[data-action="submit"]`
4. **ARIA labels**: `[aria-label="Close dialog"]`
5. **Role + name**: `button:has-text("Submit")`
6. **Stable CSS path**: `.login-form > button.primary`
7. **XPath fallback**: `//button[contains(text(), "Submit")]`

**Implementation**:

```typescript
// selectors.ts
export function generateSelectors(element: Element): SelectorSet {
  const candidates: SelectorCandidate[] = [];

  // Strategy 1: Unique ID
  if (element.id && isUniqueSelector(`#${CSS.escape(element.id)}`)) {
    candidates.push({
      type: 'id',
      value: `#${CSS.escape(element.id)}`,
      confidence: 0.95,
      specificity: 100
    });
  }

  // Strategy 2: data-testid (highest confidence for test stability)
  const testId = element.getAttribute('data-testid') ||
                 element.getAttribute('data-test-id') ||
                 element.getAttribute('data-cy');
  if (testId && isUniqueSelector(`[data-testid="${testId}"]`)) {
    candidates.push({
      type: 'data-testid',
      value: `[data-testid="${CSS.escape(testId)}"]`,
      confidence: 0.98,
      specificity: 95
    });
  }

  // Strategy 3: ARIA
  const ariaLabel = element.getAttribute('aria-label');
  if (ariaLabel) {
    const selector = `[aria-label="${CSS.escape(ariaLabel)}"]`;
    if (isUniqueSelector(selector)) {
      candidates.push({
        type: 'aria',
        value: selector,
        confidence: 0.85,
        specificity: 80
      });
    }
  }

  // Strategy 4: Role + text content
  const role = element.getAttribute('role') || inferRole(element);
  const text = element.textContent?.trim().slice(0, 50);
  if (role && text) {
    const selector = `${role}:has-text("${escapeText(text)}")`;
    // Use Playwright's built-in text matching
    candidates.push({
      type: 'role-text',
      value: selector,
      confidence: 0.75,
      specificity: 70
    });
  }

  // Strategy 5: CSS path (stable classes only)
  const cssPath = generateStableCSSPath(element);
  if (cssPath && isUniqueSelector(cssPath)) {
    candidates.push({
      type: 'css',
      value: cssPath,
      confidence: assessCSSStability(cssPath),
      specificity: 60
    });
  }

  // Strategy 6: XPath (last resort)
  const xpath = generateXPath(element);
  candidates.push({
    type: 'xpath',
    value: xpath,
    confidence: 0.5,
    specificity: 30
  });

  // Sort by confidence, select best as primary
  candidates.sort((a, b) => b.confidence - a.confidence);

  return {
    primary: candidates[0]?.value || xpath,
    candidates
  };
}

function isUniqueSelector(selector: string): boolean {
  try {
    return document.querySelectorAll(selector).length === 1;
  } catch {
    return false;
  }
}

function generateStableCSSPath(element: Element): string | null {
  // Only use "stable" classes (not generated hashes like `css-1a2b3c`)
  const unstablePatterns = [
    /^css-[a-z0-9]+$/i,      // CSS-in-JS
    /^sc-[a-z]+$/i,          // styled-components
    /^_[a-z0-9]+$/i,         // CSS modules
    /^[a-z]+-[0-9]+$/i,      // Generic hash patterns
  ];

  const path: string[] = [];
  let current: Element | null = element;

  while (current && current !== document.body) {
    const tag = current.tagName.toLowerCase();
    const classes = Array.from(current.classList)
      .filter(c => !unstablePatterns.some(p => p.test(c)))
      .slice(0, 2); // Max 2 classes

    if (classes.length > 0) {
      path.unshift(`${tag}.${classes.join('.')}`);
    } else {
      path.unshift(tag);
    }

    // Stop if we have a unique path
    const selector = path.join(' > ');
    if (isUniqueSelector(selector)) {
      return selector;
    }

    current = current.parentElement;
  }

  return path.length > 0 ? path.join(' > ') : null;
}
```

**Selector validation** (run before saving workflow):

```typescript
async function validateSelector(page: Page, selector: string): Promise<boolean> {
  try {
    const count = await page.locator(selector).count();
    return count === 1;
  } catch {
    return false;
  }
}

async function validateSelectorSet(page: Page, set: SelectorSet): Promise<SelectorCandidate | null> {
  // Try primary first
  if (await validateSelector(page, set.primary)) {
    return set.candidates.find(c => c.value === set.primary) || null;
  }

  // Fall back through candidates
  for (const candidate of set.candidates) {
    if (await validateSelector(page, candidate.value)) {
      return candidate;
    }
  }

  return null;
}
```

**Deliverables**:
- [ ] Multi-strategy selector generator
- [ ] Stability scoring for CSS classes
- [ ] Uniqueness validation
- [ ] Selector fallback chain
- [ ] Validation before workflow save

**Test criteria**:
- Generate selector for button with ID → uses ID
- Generate selector for element with data-testid → uses data-testid
- Generate selector for styled-component → avoids hash classes
- Validate selector on page refresh → same element found

---

### Technical Phase 3: Action Timeline UI

**Goal**: Real-time display of recorded actions with editing capabilities.

**Duration estimate**: Medium complexity

**Files to create**:

```
ui/src/features/record-mode/
├── RecordModePage.tsx       # NEW: Main record mode page
├── RecordModeControls.tsx   # NEW: Start/stop/settings
├── ActionTimeline.tsx       # NEW: List of recorded actions
├── ActionItem.tsx           # NEW: Single action display
├── ActionEditor.tsx         # NEW: Edit action before saving
├── WorkflowGenerator.tsx    # NEW: Convert to workflow dialog
└── hooks/
    ├── useRecordMode.ts     # NEW: Recording state management
    └── useRecordingActions.ts # NEW: WebSocket subscription
```

**Component hierarchy**:

```
RecordModePage
├── RecordModeControls
│   ├── StartButton / StopButton
│   ├── ClearButton
│   └── SettingsDropdown
├── SplitPane
│   ├── BrowserView (screenshot stream)
│   └── ActionTimeline
│       ├── ActionItem (click: button.submit)
│       ├── ActionItem (type: "hello world")
│       ├── ActionItem (scroll: 500px)
│       └── ...
└── WorkflowGenerator (modal)
    ├── ActionRangeSelector
    ├── WorkflowNameInput
    ├── ProjectSelector
    └── GenerateButton
```

**ActionTimeline component**:

```tsx
// ActionTimeline.tsx
export function ActionTimeline({ recordingId }: { recordingId: string }) {
  const { actions, isLoading } = useRecordingActions(recordingId);
  const [selectedRange, setSelectedRange] = useState<[number, number] | null>(null);

  return (
    <div className="action-timeline">
      <div className="timeline-header">
        <h3>Recorded Actions ({actions.length})</h3>
        {selectedRange && (
          <Button onClick={() => setSelectedRange(null)}>Clear Selection</Button>
        )}
      </div>

      <div className="timeline-list">
        {actions.map((action, index) => (
          <ActionItem
            key={action.id}
            action={action}
            index={index}
            isSelected={selectedRange && index >= selectedRange[0] && index <= selectedRange[1]}
            onSelect={() => handleSelect(index)}
          />
        ))}
      </div>

      {selectedRange && (
        <div className="timeline-footer">
          <Button variant="primary" onClick={() => openWorkflowGenerator()}>
            Create Workflow from {selectedRange[1] - selectedRange[0] + 1} actions
          </Button>
        </div>
      )}
    </div>
  );
}
```

**ActionItem display**:

```tsx
// ActionItem.tsx
const ACTION_ICONS: Record<string, string> = {
  click: '🖱️',
  type: '⌨️',
  scroll: '📜',
  navigate: '🔗',
  select: '📋',
  hover: '👆',
};

export function ActionItem({ action, index, isSelected, onSelect }: Props) {
  const [isEditing, setIsEditing] = useState(false);

  const getActionSummary = (action: RecordedAction): string => {
    switch (action.actionType) {
      case 'click':
        return `Click ${action.elementMeta?.tagName || 'element'}`;
      case 'type':
        const text = action.payload?.text || '';
        return `Type "${text.slice(0, 30)}${text.length > 30 ? '...' : ''}"`;
      case 'scroll':
        return `Scroll to (${action.payload?.scrollX}, ${action.payload?.scrollY})`;
      case 'navigate':
        return `Navigate to ${new URL(action.url).hostname}`;
      default:
        return action.actionType;
    }
  };

  return (
    <div
      className={`action-item ${isSelected ? 'selected' : ''}`}
      onClick={onSelect}
    >
      <span className="action-icon">{ACTION_ICONS[action.actionType] || '⚡'}</span>
      <span className="action-index">#{index + 1}</span>
      <span className="action-summary">{getActionSummary(action)}</span>
      <span className="action-selector" title={action.selector?.primary}>
        {action.selector?.primary?.slice(0, 30)}
      </span>
      <button className="edit-btn" onClick={() => setIsEditing(true)}>Edit</button>
    </div>
  );
}
```

**Deliverables**:
- [ ] Record Mode page with browser view + timeline
- [ ] Real-time action streaming via WebSocket
- [ ] Action range selection
- [ ] Action editing (change type, modify selector)
- [ ] "Create Workflow" button with dialog

**Test criteria**:
- Actions appear in timeline within 200ms of occurring
- Can select range of actions
- Can edit action selector before saving
- Can delete unwanted actions

---

### Technical Phase 4: Workflow Generation

**Goal**: Convert recorded actions to executable workflow nodes.

**Duration estimate**: Medium complexity

**Files to create/modify**:

```
api/services/recording/
├── workflow_generator.go    # NEW: Actions → workflow conversion
└── node_mapper.go           # NEW: Map action types to node types
```

**Action to Node mapping**:

| Action Type | Workflow Node | Parameter Mapping |
|-------------|---------------|-------------------|
| `click` | `click` | selector, button, modifiers |
| `type` | `type` | selector, text, delay |
| `scroll` | `scroll` | selector (or viewport), x, y |
| `navigate` | `navigate` | url |
| `select` | `select` | selector, value/text/index |
| `hover` | `hover` | selector, duration |
| `focus` | `focus` | selector |
| `blur` | `blur` | selector |

**Workflow generator**:

```go
// workflow_generator.go
type WorkflowGenerator struct {
    nodeMapper *NodeMapper
}

func (g *WorkflowGenerator) Generate(
    ctx context.Context,
    actions []RecordedAction,
    opts GenerateOptions,
) (*workflow.Workflow, error) {
    nodes := make([]workflow.Node, 0, len(actions))
    edges := make([]workflow.Edge, 0, len(actions)-1)

    var prevNodeID string

    for i, action := range actions {
        node, err := g.nodeMapper.MapActionToNode(action, i)
        if err != nil {
            return nil, fmt.Errorf("failed to map action %d: %w", i, err)
        }

        nodes = append(nodes, node)

        // Chain nodes sequentially
        if prevNodeID != "" {
            edges = append(edges, workflow.Edge{
                Source: prevNodeID,
                Target: node.ID,
            })
        }

        prevNodeID = node.ID
    }

    return &workflow.Workflow{
        ID:          uuid.New(),
        Name:        opts.Name,
        ProjectID:   opts.ProjectID,
        Nodes:       nodes,
        Edges:       edges,
        CreatedAt:   time.Now().UTC(),
        Origin:      "recording",
        RecordingID: opts.RecordingID,
    }, nil
}
```

**Node mapper**:

```go
// node_mapper.go
func (m *NodeMapper) MapActionToNode(action RecordedAction, index int) (workflow.Node, error) {
    nodeID := fmt.Sprintf("node_%d", index+1)

    var nodeType string
    params := make(map[string]any)

    switch action.ActionType {
    case "click":
        nodeType = "click"
        params["selector"] = action.Selector.Primary
        if btn := action.Payload["button"]; btn != nil {
            params["button"] = btn
        }
        if mods := action.Payload["modifiers"]; mods != nil {
            params["modifiers"] = mods
        }

    case "type":
        nodeType = "type"
        params["selector"] = action.Selector.Primary
        params["text"] = action.Payload["text"]
        // Optionally include typing delay
        if delay := action.Payload["delay"]; delay != nil {
            params["delay"] = delay
        }

    case "scroll":
        nodeType = "scroll"
        if action.Selector != nil {
            params["selector"] = action.Selector.Primary
        }
        params["y"] = action.Payload["scrollY"]

    case "navigate":
        nodeType = "navigate"
        params["url"] = action.URL

    // ... other action types

    default:
        return workflow.Node{}, fmt.Errorf("unknown action type: %s", action.ActionType)
    }

    return workflow.Node{
        ID:       nodeID,
        Type:     nodeType,
        Label:    generateNodeLabel(action),
        Position: calculateNodePosition(index),
        Data:     params,
    }, nil
}

func generateNodeLabel(action RecordedAction) string {
    switch action.ActionType {
    case "click":
        if action.ElementMeta != nil && action.ElementMeta.InnerText != "" {
            text := action.ElementMeta.InnerText
            if len(text) > 20 {
                text = text[:20] + "..."
            }
            return fmt.Sprintf("Click: %s", text)
        }
        return "Click element"
    case "type":
        return fmt.Sprintf("Type: %q", truncate(action.Payload["text"].(string), 20))
    // ...
    }
}
```

**Deliverables**:
- [ ] Action-to-node mapping for all common action types
- [ ] Sequential workflow generation
- [ ] Node positioning for visual layout
- [ ] Label generation from action context
- [ ] Recording origin tracking

**Test criteria**:
- 5 recorded actions → workflow with 5 nodes + 4 edges
- Click action → click node with correct selector
- Type action → type node with correct text
- Generated workflow executes successfully

---

### Technical Phase 5: Integration & Polish

**Goal**: End-to-end flow works smoothly with good UX.

**Duration estimate**: Medium complexity

**Tasks**:

1. **Selector validation before save**:
   - Before generating workflow, re-validate all selectors
   - Warn user about potentially unstable selectors
   - Allow manual selector editing

2. **Action merging/deduplication**:
   - Merge consecutive type actions into single node
   - Remove accidental double-clicks
   - Collapse scroll sequences

3. **Replay preview**:
   - Before saving, offer to replay recorded actions
   - Show success/failure for each step
   - Allow retry with edited selectors

4. **Error handling**:
   - Graceful recovery from injection failures
   - Clear error messages for unsupported sites
   - Fallback for sites that block injection

5. **Settings**:
   - Configurable debounce timings
   - Include/exclude action types
   - Auto-screenshot frequency

**Deliverables**:
- [ ] Pre-save selector validation with warnings
- [ ] Action merging/cleanup logic
- [ ] Replay preview before save
- [ ] Comprehensive error handling
- [ ] User-configurable settings

---

## 5. Selector Strategy

### 5.1 Why Selectors Are The Hardest Problem

Selectors are the Achilles' heel of browser automation recording. A selector that works perfectly during recording can break for many reasons:

1. **Dynamic IDs**: React, Vue, Angular often generate unique IDs per render
2. **CSS-in-JS**: Class names like `css-1a2b3c` change on rebuild
3. **A/B testing**: Page structure varies between users
4. **Responsive design**: Different elements at different viewport sizes
5. **Authentication state**: Different DOM when logged in vs. out
6. **Content changes**: Text-based selectors break when content updates

### 5.2 Our Selector Priority

| Priority | Selector Type | Example | Why |
|----------|---------------|---------|-----|
| 1 | data-testid | `[data-testid="login-btn"]` | Explicitly stable, test-friendly |
| 2 | ID (if stable) | `#submit-button` | Unique, but beware dynamic IDs |
| 3 | ARIA attributes | `[aria-label="Close"]` | Semantic, accessibility-focused |
| 4 | Role + text | `button:has-text("Submit")` | Human-readable, semantic |
| 5 | Stable classes | `.btn-primary.large` | Common patterns, filter out hashes |
| 6 | Attribute combinations | `input[type="email"][name="email"]` | Multiple signals |
| 7 | CSS path | `form > div:nth-child(2) > button` | Structural, but fragile |
| 8 | XPath | `//button[text()="Submit"]` | Last resort, powerful but verbose |

### 5.3 Stability Scoring

Each selector gets a confidence score (0-1):

```typescript
function assessSelectorStability(selector: string, element: Element): number {
  let score = 0.5; // Base score

  // Boost for test-friendly attributes
  if (selector.includes('data-testid')) score += 0.4;
  if (selector.includes('data-test')) score += 0.35;
  if (selector.includes('aria-label')) score += 0.25;

  // Boost for semantic selectors
  if (selector.match(/^(button|input|a|form)/)) score += 0.1;

  // Penalize for instability indicators
  if (selector.match(/css-[a-z0-9]+/i)) score -= 0.3;      // CSS-in-JS
  if (selector.match(/:[0-9]+/)) score -= 0.2;             // Position-based
  if (selector.match(/nth-child\(\d+\)/)) score -= 0.15;   // Order-dependent
  if (selector.length > 100) score -= 0.1;                 // Overly specific

  // Boost for uniqueness verification
  if (document.querySelectorAll(selector).length === 1) score += 0.1;

  return Math.max(0, Math.min(1, score));
}
```

### 5.4 User Feedback for Unstable Selectors

When selector confidence is low (< 0.6), show warning:

```tsx
<ActionItem>
  <WarningIcon />
  <span>This selector may be unstable. Consider:</span>
  <ul>
    <li>Adding data-testid to the target element</li>
    <li>Using a more specific selector</li>
    <li>Testing the workflow multiple times</li>
  </ul>
  <Button onClick={openSelectorEditor}>Edit Selector</Button>
</ActionItem>
```

---

## 6. UI/UX Design

### 6.1 Record Mode Page Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ ← Back to Dashboard          Record Mode                    ⚙️ Settings    │
├─────────────────────────────────────────────────────────────────────────────┤
│ URL: [https://example.com________________] [Go]     [● Recording] [Stop]   │
├────────────────────────────────────┬────────────────────────────────────────┤
│                                    │  Actions (12)                    [📋] │
│                                    │ ─────────────────────────────────────  │
│                                    │  1. 🔗 Navigate to example.com         │
│                                    │  2. 🖱️ Click: "Sign In"                │
│      Live Browser Preview          │  3. ⌨️ Type: "user@email.com"          │
│      (Screenshot stream)           │  4. ⌨️ Type: "••••••••"                │
│                                    │  5. 🖱️ Click: "Submit"                 │
│                                    │  6. ⏳ Wait: page load                  │
│                                    │  7. 🖱️ Click: "Dashboard"              │
│                                    │  ...                                    │
│                                    │ ─────────────────────────────────────  │
│                                    │  [Select Range] [Create Workflow]      │
└────────────────────────────────────┴────────────────────────────────────────┘
```

### 6.2 User Flow

```
1. User clicks "Record Mode" from dashboard
                    │
                    ▼
2. Empty state: "Enter a URL to start recording"
   [URL input field] [Start Recording]
                    │
                    ▼
3. Recording active: Browser view + empty timeline
   Visual indicator: pulsing red dot
                    │
                    ▼
4. User performs actions in browser view
   Timeline updates in real-time
                    │
                    ▼
5. User clicks "Stop Recording"
   Timeline shows all captured actions
                    │
                    ▼
6. User reviews actions:
   - Delete unwanted actions
   - Edit selectors if needed
   - Select subset if desired
                    │
                    ▼
7. User clicks "Create Workflow"
   Dialog: Name, Project, options
                    │
                    ▼
8. Workflow generated, opens in builder
   User can further edit if needed
```

### 6.3 Visual Feedback During Recording

| State | Visual Indicator |
|-------|------------------|
| Recording active | Pulsing red dot + "Recording" badge |
| Action captured | Brief green highlight on timeline |
| Selector unstable | Yellow warning icon on action |
| Error occurred | Red error badge + tooltip |
| Paused | Gray "Paused" badge |

### 6.4 Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl/Cmd + R` | Start/stop recording |
| `Ctrl/Cmd + S` | Save as workflow |
| `Escape` | Stop recording |
| `Delete` | Remove selected action |
| `Ctrl/Cmd + Z` | Undo last action removal |

---

## 7. Risk Assessment

### 7.1 Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Selectors break on refresh | High | High | Multi-strategy generation, validation, user editing |
| JS injection blocked by CSP | Medium | Medium | Fallback to CDP-based capture |
| Action classification wrong | Medium | Medium | User editing, confidence scoring, replay validation |
| Performance impact on page | Low | Medium | Throttle screenshot capture, debounce events |
| iframe support complexity | Medium | Low | Phase 2, explicit frame switching |

### 7.2 UX Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Latency feels laggy | Medium | High | Optimize screenshot streaming, async processing |
| Timeline overwhelmed | Medium | Medium | Auto-merge similar actions, collapse sequences |
| Selector editing too technical | High | Medium | Guided editor with suggestions |
| Unclear what was captured | Low | Medium | Visual highlighting in browser view |

### 7.3 Scope Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Feature creep | High | Medium | Strict phase boundaries, explicit non-goals |
| Phase 2 designed in Phase 1 | Medium | Low | Modular architecture, but don't over-engineer |

---

## 8. Future Evolution

### 8.1 Stage 2: Browser-First Experience

Once Stage 1 is stable, evolve toward the browser-first vision:

1. **Always-on recording**: Actions captured passively, no explicit "start"
2. **Session history**: Browse history of all recorded sessions
3. **Retroactive selection**: Pick any time range from any session
4. **Smarter suggestions**: "You've done this 3 times, create workflow?"

### 8.2 Browser Extension

Parallel track for capturing actions outside browser-automation-studio:

1. **Chrome extension**: Capture actions in real browser
2. **Export to browser-automation-studio**: Send recording for processing
3. **Sync**: Extension syncs recordings to server
4. **Inline workflow creation**: Create workflow without opening browser-automation-studio

### 8.3 AI-Assisted Recording

Leverage AI for smarter recording:

1. **Intent detection**: "It looks like you're logging in - should I parameterize username/password?"
2. **Selector suggestions**: "This selector might break - here's a more stable alternative"
3. **Action cleanup**: "I merged 5 scroll events and removed 2 accidental clicks"
4. **Workflow optimization**: "I added waits where the page was loading"

---

## 9. File Reference

### 9.1 Existing Files to Modify

```
api/
├── automation/contracts/
│   └── events.go                    # Add recording event kinds
├── services/recording/
│   ├── types.go                     # Reference for type patterns
│   └── adapter.go                   # Reference for normalization
└── handlers/
    └── recordings.go                # Reference for import flow

playwright-driver/src/
├── session/
│   └── manager.ts                   # Add recording state to SessionState
├── telemetry/
│   └── collector.ts                 # Pattern for event collection
└── types/
    └── index.ts                     # Add recording types

ui/src/
├── contexts/
│   └── WebSocketContext.tsx         # Add recording event handling
└── hooks/
    └── useExecutionWebSocket.ts     # Pattern for WS subscriptions
```

### 9.2 New Files to Create

```
api/
├── automation/contracts/
│   └── recording.go                 # RecordedAction, SelectorSet types
├── handlers/
│   └── record_mode.go               # HTTP handlers for record control
└── services/recording/
    ├── live_recorder.go             # Manages live recording sessions
    ├── action_store.go              # Persists recorded actions
    ├── workflow_generator.go        # Actions → workflow conversion
    └── node_mapper.go               # Map action types to node types

playwright-driver/src/
├── recording/
│   ├── controller.ts                # Record mode orchestration
│   ├── injector.ts                  # JavaScript injection
│   ├── normalizer.ts                # Raw events → RecordedAction
│   └── selectors.ts                 # Selector generation
└── handlers/
    └── record-mode.ts               # HTTP handlers for record control

ui/src/features/record-mode/
├── RecordModePage.tsx               # Main record mode page
├── RecordModeControls.tsx           # Start/stop/settings
├── ActionTimeline.tsx               # List of recorded actions
├── ActionItem.tsx                   # Single action display
├── ActionEditor.tsx                 # Edit action before saving
├── SelectorEditor.tsx               # Edit selector with guidance
├── WorkflowGenerator.tsx            # Convert to workflow dialog
└── hooks/
    ├── useRecordMode.ts             # Recording state management
    └── useRecordingActions.ts       # WebSocket subscription
```

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **Action** | A single user interaction (click, type, scroll) captured during recording |
| **Recording Session** | A period of active recording, from start to stop |
| **Selector** | A string that uniquely identifies a DOM element (CSS selector, XPath, etc.) |
| **SelectorSet** | Multiple selector strategies for the same element, ranked by confidence |
| **Action Normalization** | Converting raw browser events to semantic RecordedAction objects |
| **Workflow Generation** | Converting a sequence of RecordedActions into a workflow with nodes and edges |

---

## Appendix B: Related Documentation

- [SYSTEM_ARCHITECTURE.md](./SYSTEM_ARCHITECTURE.md) - Overall system architecture
- [NODE_INDEX.md](./NODE_INDEX.md) - All supported workflow node types
- [api/automation/contracts/contracts.go](../api/automation/contracts/contracts.go) - Core type definitions
- [api/services/recording/](../api/services/recording/) - Existing recording import code

---

## Appendix C: Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2024-12-04 | Start with Stage 1 (dedicated page) | Lower risk, validates core concepts before browser-first |
| 2024-12-04 | Use page.exposeFunction() for event callback | Cleaner than CDP, works cross-frame |
| 2024-12-04 | Multi-strategy selectors | Single selector too fragile, need fallbacks |
| 2024-12-04 | Debounce input/scroll events | Avoid timeline spam, better UX |
| 2024-12-04 | Re-inject on navigation | Script doesn't persist across page loads |
| 2024-12-04 | Hybrid execution approach (spike + vertical slice) | De-risk selectors early, but get something working fast |
| 2024-12-04 | Phase 0 selector spike before main development | If selectors fail >30%, entire feature is compromised - fail fast |
| 2024-12-04 | Selector editor is non-negotiable for MVP | Users can recover from bad selectors if editing is easy; can't recover from no editing |
| 2024-12-04 | Ship Click/Type/Navigate only initially | Covers 80% of use cases; better to ship these well than everything poorly |
| 2024-12-04 | Defer scroll/hover/select/drag actions | Fast follow after core is proven; reduces initial scope |
| 2024-12-04 | Defer action merging/deduplication | Nice-to-have polish; users can manually delete unwanted actions |
| 2024-12-04 | Defer replay preview before save | Valuable but not critical for MVP; selector editor handles validation need |
