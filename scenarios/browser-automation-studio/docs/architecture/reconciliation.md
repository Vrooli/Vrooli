# Recording Reconciliation Architecture

This document explains the three reconciliation systems that ensure data consistency across the browser-automation-studio stack.

## Overview

Recording and workflow management involves multiple data sources that can drift out of sync:
- Filesystem (JSON workflow files)
- Database (indexed metadata for fast queries)
- Live recording events (raw browser actions)
- AI navigation steps (when AI drives the browser)

Three reconciliation systems work together to maintain consistency:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         RECONCILIATION LAYERS                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐        │
│   │  1. BACKEND     │    │  2. FRONTEND    │    │  3. AI STEP     │        │
│   │  WORKFLOW SYNC  │    │  ACTION MERGE   │    │  RECONCILIATION │        │
│   │                 │    │                 │    │                 │        │
│   │  Filesystem ↔   │    │  Raw actions →  │    │  AI steps +     │        │
│   │  Database       │    │  Clean actions  │    │  Recorded →     │        │
│   │                 │    │                 │    │  Timeline       │        │
│   └─────────────────┘    └─────────────────┘    └─────────────────┘        │
│                                                                             │
│   api/services/         ui/src/domains/        ui/src/domains/             │
│   workflow/sync.go      recording/utils/       recording/types/            │
│                         mergeActions.ts        timeline-unified.ts         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 1. Backend Workflow Sync

**Location:** `api/services/workflow/sync.go`

### Problem It Solves

The filesystem is the source of truth for workflow definitions, but we need a database index for fast queries (listing, searching, filtering). These can drift apart when:
- Files are deleted/moved outside the application
- External tools modify project structures
- Database entries become stale from failed operations

### Design Decisions

**Why filesystem as source of truth?**
- Workflows are JSON files that users can version control with git
- External tools (editors, scripts) can modify workflows
- Files are portable and self-contained

**Why database index?**
- O(1) lookups by workflow ID
- Fast filtering by project, folder, name
- Metadata caching (version, timestamps)

### Algorithm

```
syncProjectWorkflows()
         │
         ▼
┌─────────────────────────────────────┐
│  1. Load DB records into maps       │   O(n) where n = existing DB records
│     dbWorkflows[uuid] → record      │
│     dbAssets[path]    → record      │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  2. Walk filesystem                 │   O(m) where m = files on disk
│     - Max 1000 files (safety limit) │
│     - Max 4 directory levels        │
│     - Skip hidden directories       │
│                                     │
│     For each file:                  │
│       If workflow JSON → upsert DB  │
│       Else → index as asset         │
│       Mark as "seen"                │
└─────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│  3. Garbage collection              │   O(n) scan of DB records
│                                     │
│     For each DB record NOT seen:    │
│       DELETE from database          │
│       Remove from path cache        │
└─────────────────────────────────────┘
```

### Concurrency

Per-project mutex locks prevent concurrent syncs from corrupting state:
```go
lock := s.getProjectLock(project.ID)
lock.Lock()
defer lock.Unlock()
```

### External Format Conversion

When sync encounters non-native workflow formats (e.g., imported from other tools), it converts them IN-PLACE to native format. This is intentional - we normalize everything to a single format for consistency.

---

## 2. Frontend Action Merging

**Location:** `ui/src/domains/recording/utils/mergeActions.ts`

### Problem It Solves

Browser recording captures low-level events that are noisy for users:
- Typing "hello" creates 5 separate `input` events
- Scrolling creates dozens of `scroll` events per second
- Focus events before typing are redundant

Without merging, generated workflows become:
- Verbose and hard to read
- Less reliable (more steps = more failure points)
- Slow to execute

### Design Decisions

**Why client-side merging?**
- Instant preview: users see cleaned actions as they record
- WYSIWYG: "what you see = what you get" in the workflow
- Mirrors backend logic (`api/handlers/record_mode.go`) for consistency

**Why greedy forward merging?**
- Single-pass O(n) algorithm - efficient for real-time updates
- Predictable: always merges forward, never backtracks
- Simple to reason about

### Merge Rules

Each rule addresses a specific UX problem:

| Rule | Problem | Solution |
|------|---------|----------|
| **Focus removal** | Focus events before typing are implicit | Skip focus if followed by input on same element |
| **Input merge** | Typing creates many events | Concatenate consecutive inputs on same selector |
| **Scroll merge** | Scrolling is very granular | Keep only final scroll position |
| **Navigate merge** | Redirects create chains | Keep only final URL |

### Algorithm

```
for each action in chronological order:
    if focus → input on same element: SKIP focus

    if input on element X:
        while next is also input on X:
            concatenate text
            advance index
        emit merged input

    if scroll:
        while next is also scroll:
            update final position
            advance index
        emit merged scroll

    if navigate:
        while next is also navigate:
            update final URL
            advance index
        emit merged navigate
```

### Metadata Preservation

Merged actions retain metadata for UI display:
```typescript
action._merged = {
  mergedCount: 5,           // "Merged 5 keystrokes"
  mergedIds: ['a1', ...],   // Original action IDs
  mergeType: 'input',       // Type of merge applied
};
```

This enables "undo" at the individual keystroke level if needed.

---

## 3. AI Step Reconciliation

**Location:** `ui/src/domains/recording/types/timeline-unified.ts` (function `mergeActionsWithAISteps`)

### Problem It Solves

When AI navigation drives the browser, we have two event streams:
1. **Recorded actions**: Raw browser events captured by the recorder
2. **AI steps**: High-level decisions from the AI (with reasoning, token usage)

Users want to see both together: the action that happened AND why the AI did it.

### Design Decisions

**Why timestamp-based matching?**
- AI step and recorded action happen near-simultaneously
- No shared identifiers between AI and recording systems
- Time proximity is a reliable correlation signal

**Why 5-second window?**
- Typical action latency: 50-500ms
- Network/processing delays: up to 2-3 seconds
- 5 seconds provides margin while avoiding false matches

**Why greedy best-match?**
- Each AI step should match at most one recorded action
- Prevents duplicate attribution
- Simple to understand and debug

### Algorithm

```
unmatchedSteps = copy of aiSteps

for each recorded action:
    actionTime = action.timestamp
    actionType = action.actionType

    bestMatch = null
    bestTimeDiff = Infinity

    for each step in unmatchedSteps:
        if |stepTime - actionTime| < 5000ms AND types match:
            if timeDiff < bestTimeDiff:
                bestMatch = step
                bestTimeDiff = timeDiff

    if bestMatch:
        remove from unmatchedSteps  // consumed
        emit action with AI metadata attached
    else:
        emit action without AI metadata
```

### Type Normalization

AI uses different action type names than the recorder:
```typescript
const mapping = {
  type: 'input',      // AI "type" → recorded "input"
  keypress: 'keyboard',
  done: 'wait',
};
```

---

## How They Connect

```
  Browser Recording          AI Navigation
        │                          │
        ▼                          ▼
┌───────────────────┐    ┌───────────────────┐
│  Raw Actions      │    │  AI Steps         │
│  (noisy, verbose) │    │  (with reasoning) │
└─────────┬─────────┘    └─────────┬─────────┘
          │                        │
          ▼                        │
┌───────────────────┐              │
│ mergeConsecutive  │◄─────────────┘
│ Actions()         │
│ (deduplicate)     │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│ mergeActionsWithAI│
│ Steps()           │
│ (correlate)       │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│   Timeline UI     │
│   (display)       │
└─────────┬─────────┘
          │
          ▼  Save Workflow
┌───────────────────┐
│  Workflow JSON    │
│  (on disk)        │
└─────────┬─────────┘
          │
          ▼  Background Sync
┌───────────────────┐
│ syncProjectWorkflows() │
│ (index in DB)     │
└───────────────────┘
```

---

## Key Files Reference

| System | File | Entry Point |
|--------|------|-------------|
| Backend Sync | `api/services/workflow/sync.go` | `SyncProjectWorkflows()` |
| Action Merge | `ui/src/domains/recording/utils/mergeActions.ts` | `mergeConsecutiveActions()` |
| AI Reconciliation | `ui/src/domains/recording/types/timeline-unified.ts` | `mergeActionsWithAISteps()` |
| Usage | `ui/src/domains/recording/RecordingSession.tsx` | Lines 639-651 |

---

## Performance Characteristics

| System | Time Complexity | Space Complexity | Limits |
|--------|-----------------|------------------|--------|
| Backend Sync | O(n + m) | O(n + m) | 1000 files, depth 4 |
| Action Merge | O(n) single-pass | O(n) output | None |
| AI Reconciliation | O(n * m) | O(m) | m typically < 50 |

Where:
- n = number of recorded actions / DB records
- m = number of files on disk / AI steps

---

## When Each System Runs

| System | Trigger |
|--------|---------|
| Backend Sync | Project creation, explicit API call, after file operations |
| Action Merge | Every render when `actions` dependency changes (memoized) |
| AI Reconciliation | Only during AI navigation sessions |
