# Multi-Tab Recording Implementation Plan

**Status:** Draft for Review
**Created:** 2025-12-26
**Scenario:** browser-automation-studio

---

## Executive Summary

This document outlines the complete implementation plan for adding multi-tab (multi-page) recording support to browser-automation-studio. The current system only records actions from a single page; when users click links that open new tabs, those tabs are opened but not recorded, with no indication to the user.

The solution involves changes across all system layers: Playwright-driver protocol, Go API services, WebSocket messaging, and React UI. Implementation is divided into 6 phases, each delivering incremental value while building toward full multi-tab support.

---

## Table of Contents

1. [Current State Analysis](#1-current-state-analysis)
2. [Target Architecture](#2-target-architecture)
3. [Phase 1: Page Tracking Foundation](#3-phase-1-page-tracking-foundation)
4. [Phase 2: Page Lifecycle Events](#4-phase-2-page-lifecycle-events)
5. [Phase 3: UI Tab Management](#5-phase-3-ui-tab-management)
6. [Phase 4: Multi-Page Timeline](#6-phase-4-multi-page-timeline)
7. [Phase 5: Frame Streaming Optimization](#7-phase-5-frame-streaming-optimization)
8. [Phase 6: Multi-Page Playback](#8-phase-6-multi-page-playback)
9. [Data Model Changes](#9-data-model-changes)
10. [API Specification](#10-api-specification)
11. [WebSocket Protocol](#11-websocket-protocol)
12. [Testing Strategy](#12-testing-strategy)
13. [Migration & Backwards Compatibility](#13-migration--backwards-compatibility)
14. [Risk Analysis](#14-risk-analysis)
15. [Edge Cases & Error Handling](#15-edge-cases--error-handling)
16. [Performance Considerations](#16-performance-considerations)
17. [Implementation Checklist](#17-implementation-checklist)

---

## 1. Current State Analysis

### 1.1 What Exists Today

| Component | Current Behavior |
|-----------|------------------|
| **Session Model** | One session = one browser context = one implicit page |
| **Action Recording** | `RecordedAction` has no `pageId` field |
| **Frame Streaming** | Streams from the initial page only |
| **WebSocket Protocol** | No page lifecycle messages |
| **UI** | Single preview canvas, no tab awareness |
| **Playback** | Assumes single page throughout workflow |

### 1.2 Key Files Affected

```
api/
├── automation/driver/types.go          # RecordedAction struct
├── automation/driver/client.go         # Driver communication
├── automation/session/session.go       # Session management
├── automation/session/spec.go          # Session specification
├── handlers/record_mode.go             # HTTP handlers
├── services/live-capture/service.go    # Recording service
├── websocket/hub.go                    # WebSocket broadcasting
└── domain/                             # Domain types (new files)

ui/src/
├── domains/recording/RecordingSession.tsx
├── domains/recording/capture/PlaywrightView.tsx
├── domains/recording/hooks/useRecordMode.ts
├── domains/recording/components/          # New tab components
└── contexts/WebSocketContext.tsx
```

### 1.3 Current Limitations

1. **No page tracking**: System doesn't know when new pages open
2. **No page attribution**: Actions have no page context
3. **Single stream**: Frame streaming bound to initial page
4. **No UI indication**: Users unaware of new tabs
5. **Broken playback**: Multi-tab workflows cannot replay

---

## 2. Target Architecture

### 2.1 Conceptual Model

```
┌─────────────────────────────────────────────────────────────────┐
│                     Recording Session                            │
├─────────────────────────────────────────────────────────────────┤
│  Browser Context (Playwright)                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Page 1     │  │   Page 2     │  │   Page 3     │          │
│  │  (initial)   │  │  (popup)     │  │  (link)      │          │
│  │              │  │              │  │              │          │
│  │  [Actions]   │  │  [Actions]   │  │  [Actions]   │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
├─────────────────────────────────────────────────────────────────┤
│  Unified Timeline                                                │
│  ┌────┬────┬────┬────┬────┬────┬────┬────┬────┐               │
│  │ A1 │ A2 │ P2 │ A3 │ A4 │ P3 │ A5 │ A6 │ C2 │               │
│  │ p1 │ p1 │ ↑  │ p2 │ p2 │ ↑  │ p1 │ p3 │ ↓  │               │
│  └────┴────┴────┴────┴────┴────┴────┴────┴────┘               │
│    A=Action  P=PageOpened  C=PageClosed  p1/p2/p3=PageID        │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Design Principles

1. **Page Attribution**: Every event tagged with its page context
2. **Unified Timeline**: All events (actions + page lifecycle) in chronological order
3. **Active Page Focus**: One page streams full frames; others are tracked but not streamed
4. **Graceful Degradation**: Single-page workflows work exactly as before
5. **Explicit State**: UI always shows what pages exist and which is active

### 2.3 Key Invariants

- A session always has at least one page (the initial page)
- Every action belongs to exactly one page
- Page IDs are stable for the session lifetime (UUID)
- The timeline is strictly ordered by timestamp
- Closing a page doesn't delete its recorded actions

---

## 3. Phase 1: Page Tracking Foundation

**Goal**: Enable the system to track multiple pages and attribute actions to them.

### 3.1 Domain Model Changes

#### 3.1.1 New File: `api/domain/page.go`

```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

// Page represents a browser page/tab within a recording session
type Page struct {
    ID          uuid.UUID  `json:"id"`
    SessionID   uuid.UUID  `json:"sessionId"`
    URL         string     `json:"url"`
    Title       string     `json:"title"`
    OpenerID    *uuid.UUID `json:"openerId,omitempty"`    // Page that opened this one
    CreatedAt   time.Time  `json:"createdAt"`
    ClosedAt    *time.Time `json:"closedAt,omitempty"`
    IsInitial   bool       `json:"isInitial"`             // First page in session
    Status      PageStatus `json:"status"`
}

type PageStatus string

const (
    PageStatusActive  PageStatus = "active"
    PageStatusClosed  PageStatus = "closed"
)

// PageEvent represents a page lifecycle event
type PageEvent struct {
    ID        uuid.UUID     `json:"id"`
    Type      PageEventType `json:"type"`
    PageID    uuid.UUID     `json:"pageId"`
    URL       string        `json:"url,omitempty"`
    Title     string        `json:"title,omitempty"`
    OpenerID  *uuid.UUID    `json:"openerId,omitempty"`
    Timestamp time.Time     `json:"timestamp"`
}

type PageEventType string

const (
    PageEventCreated   PageEventType = "page_created"
    PageEventNavigated PageEventType = "page_navigated"
    PageEventClosed    PageEventType = "page_closed"
)
```

#### 3.1.2 Update: `api/automation/driver/types.go`

```go
// Add PageID to RecordedAction
type RecordedAction struct {
    ID            uuid.UUID              `json:"id"`
    PageID        uuid.UUID              `json:"pageId"`        // NEW: Required
    ActionType    ActionType             `json:"actionType"`
    URL           string                 `json:"url"`
    Title         string                 `json:"title"`         // NEW: Page title at action time
    Selector      *SelectorInfo          `json:"selector,omitempty"`
    Payload       map[string]interface{} `json:"payload,omitempty"`
    FrameID       string                 `json:"frameId,omitempty"`
    Confidence    float64                `json:"confidence"`
    Timestamp     time.Time              `json:"timestamp"`
    SequenceNum   int                    `json:"sequenceNum"`
}
```

### 3.2 Session Manager Changes

#### 3.2.1 Update: `api/automation/session/session.go`

```go
// Add page tracking to Session
type Session struct {
    ID            uuid.UUID
    ContextID     string                    // Playwright context ID
    Driver        *driver.Client
    Pages         map[uuid.UUID]*domain.Page // NEW: All pages in session
    ActivePageID  uuid.UUID                  // NEW: Currently focused page
    InitialPageID uuid.UUID                  // NEW: First page created
    mu            sync.RWMutex
    // ... existing fields
}

// NewSession creates session with initial page
func (m *Manager) Create(ctx context.Context, spec Spec) (*Session, error) {
    // ... existing context creation ...

    // Create initial page record
    initialPageID := uuid.New()
    initialPage := &domain.Page{
        ID:        initialPageID,
        SessionID: sessionID,
        URL:       spec.StartURL,
        CreatedAt: time.Now(),
        IsInitial: true,
        Status:    domain.PageStatusActive,
    }

    session := &Session{
        ID:            sessionID,
        Pages:         map[uuid.UUID]*domain.Page{initialPageID: initialPage},
        ActivePageID:  initialPageID,
        InitialPageID: initialPageID,
        // ...
    }

    // Register page event listener with driver
    session.registerPageListeners(ctx)

    return session, nil
}

// Page management methods
func (s *Session) AddPage(page *domain.Page) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.Pages[page.ID] = page
}

func (s *Session) GetPage(pageID uuid.UUID) (*domain.Page, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    page, ok := s.Pages[pageID]
    return page, ok
}

func (s *Session) GetActivePageID() uuid.UUID {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.ActivePageID
}

func (s *Session) SetActivePage(pageID uuid.UUID) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.Pages[pageID]; !ok {
        return fmt.Errorf("page %s not found in session", pageID)
    }
    s.ActivePageID = pageID
    return nil
}

func (s *Session) ClosePage(pageID uuid.UUID) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    page, ok := s.Pages[pageID]
    if !ok {
        return fmt.Errorf("page %s not found", pageID)
    }
    now := time.Now()
    page.ClosedAt = &now
    page.Status = domain.PageStatusClosed

    // If active page closed, switch to another open page
    if s.ActivePageID == pageID {
        for id, p := range s.Pages {
            if p.Status == domain.PageStatusActive {
                s.ActivePageID = id
                break
            }
        }
    }
    return nil
}

func (s *Session) ListPages() []*domain.Page {
    s.mu.RLock()
    defer s.mu.RUnlock()
    pages := make([]*domain.Page, 0, len(s.Pages))
    for _, p := range s.Pages {
        pages = append(pages, p)
    }
    // Sort by CreatedAt
    sort.Slice(pages, func(i, j int) bool {
        return pages[i].CreatedAt.Before(pages[j].CreatedAt)
    })
    return pages
}
```

### 3.3 Driver Protocol Changes

The Playwright-driver needs to:
1. Report page creation events
2. Report page close events
3. Include `pageId` in all recorded actions
4. Support switching which page receives input forwarding

#### 3.3.1 Driver Request Types

```go
// Request to subscribe to page events
type PageEventSubscription struct {
    SessionID string `json:"sessionId"`
    Callback  string `json:"callbackUrl"`  // POST page events here
}

// Page event from driver
type DriverPageEvent struct {
    SessionID   string `json:"sessionId"`
    DriverPageID string `json:"driverPageId"`  // Playwright's internal ID
    VrooliPageID string `json:"vrooliPageId"`  // Our UUID (set by us, echoed back)
    EventType   string `json:"eventType"`      // "created" | "navigated" | "closed"
    URL         string `json:"url"`
    Title       string `json:"title"`
    OpenerDriverPageID string `json:"openerDriverPageId,omitempty"`
    Timestamp   string `json:"timestamp"`
}

// Request to set active page for input forwarding
type SetActivePageRequest struct {
    SessionID    string `json:"sessionId"`
    DriverPageID string `json:"driverPageId"`
}
```

### 3.4 Handler Changes

#### 3.4.1 New Endpoint: Receive Page Events

```go
// POST /api/v1/recordings/live/{sessionId}/page-event
// Called by Playwright-driver when page lifecycle events occur
func (h *Handler) ReceivePageEvent(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "sessionId")

    var event DriverPageEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    session, err := h.sessions.Get(sessionID)
    if err != nil {
        http.Error(w, "session not found", http.StatusNotFound)
        return
    }

    var pageEvent *domain.PageEvent

    switch event.EventType {
    case "created":
        pageID := uuid.New()
        var openerID *uuid.UUID
        if event.OpenerDriverPageID != "" {
            // Map driver's opener ID to our page ID
            openerID = session.GetPageIDByDriverID(event.OpenerDriverPageID)
        }

        page := &domain.Page{
            ID:        pageID,
            SessionID: session.ID,
            URL:       event.URL,
            Title:     event.Title,
            OpenerID:  openerID,
            CreatedAt: time.Now(),
            IsInitial: false,
            Status:    domain.PageStatusActive,
        }
        session.AddPage(page)
        session.MapDriverPageID(event.DriverPageID, pageID)

        pageEvent = &domain.PageEvent{
            ID:        uuid.New(),
            Type:      domain.PageEventCreated,
            PageID:    pageID,
            URL:       event.URL,
            Title:     event.Title,
            OpenerID:  openerID,
            Timestamp: time.Now(),
        }

    case "navigated":
        pageID := session.GetPageIDByDriverID(event.DriverPageID)
        if pageID != nil {
            page, _ := session.GetPage(*pageID)
            page.URL = event.URL
            page.Title = event.Title

            pageEvent = &domain.PageEvent{
                ID:        uuid.New(),
                Type:      domain.PageEventNavigated,
                PageID:    *pageID,
                URL:       event.URL,
                Title:     event.Title,
                Timestamp: time.Now(),
            }
        }

    case "closed":
        pageID := session.GetPageIDByDriverID(event.DriverPageID)
        if pageID != nil {
            session.ClosePage(*pageID)

            pageEvent = &domain.PageEvent{
                ID:        uuid.New(),
                Type:      domain.PageEventClosed,
                PageID:    *pageID,
                Timestamp: time.Now(),
            }
        }
    }

    if pageEvent != nil {
        // Add to timeline
        h.recordModeService.AddPageEvent(r.Context(), sessionID, pageEvent)

        // Broadcast via WebSocket
        h.wsHub.BroadcastPageEvent(sessionID, pageEvent)
    }

    w.WriteHeader(http.StatusOK)
}
```

### 3.5 Action Attribution

#### 3.5.1 Update: `ReceiveRecordingAction`

```go
func (h *Handler) ReceiveRecordingAction(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "sessionId")

    var action RecordedAction
    if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    session, err := h.sessions.Get(sessionID)
    if err != nil {
        http.Error(w, "session not found", http.StatusNotFound)
        return
    }

    // Map driver's page ID to our page ID
    if action.DriverPageID != "" {
        pageID := session.GetPageIDByDriverID(action.DriverPageID)
        if pageID != nil {
            action.PageID = *pageID
        } else {
            // Unknown page - likely a race condition, use active page
            action.PageID = session.GetActivePageID()
            log.Warn().Str("driverPageId", action.DriverPageID).Msg("action from unknown page, using active page")
        }
    } else {
        // Legacy action without page ID - use active page
        action.PageID = session.GetActivePageID()
    }

    // ... rest of existing logic ...
}
```

### 3.6 Deliverables - Phase 1

- [ ] `api/domain/page.go` - Page and PageEvent types
- [ ] Update `RecordedAction` with `PageID` field
- [ ] Session page tracking (add/get/list/close pages)
- [ ] Driver-to-Vrooli page ID mapping
- [ ] `POST /page-event` endpoint
- [ ] Page event storage in timeline
- [ ] WebSocket broadcast for page events
- [ ] Action page attribution logic
- [ ] Unit tests for session page management

---

## 4. Phase 2: Page Lifecycle Events

**Goal**: Ensure page creation, navigation, and closure are captured and broadcast in real-time.

### 4.1 Timeline Unification

The timeline now contains two types of entries:
1. **Actions**: User interactions (click, type, navigate, etc.)
2. **Page Events**: Page lifecycle (created, navigated, closed)

#### 4.1.1 Unified Timeline Entry

```go
// TimelineEntry is a union type for the recording timeline
type TimelineEntry struct {
    ID        uuid.UUID       `json:"id"`
    Type      TimelineType    `json:"type"`  // "action" | "page_event"
    Timestamp time.Time       `json:"timestamp"`
    PageID    uuid.UUID       `json:"pageId"`

    // One of these will be set based on Type
    Action    *RecordedAction `json:"action,omitempty"`
    PageEvent *PageEvent      `json:"pageEvent,omitempty"`
}

type TimelineType string

const (
    TimelineTypeAction    TimelineType = "action"
    TimelineTypePageEvent TimelineType = "page_event"
)
```

### 4.2 Service Layer Updates

#### 4.2.1 Timeline Service

```go
// services/live-capture/timeline.go

type TimelineService struct {
    entries map[string][]TimelineEntry  // sessionID -> entries
    mu      sync.RWMutex
}

func (s *TimelineService) AddAction(sessionID string, action *RecordedAction) {
    s.mu.Lock()
    defer s.mu.Unlock()

    entry := TimelineEntry{
        ID:        uuid.New(),
        Type:      TimelineTypeAction,
        Timestamp: action.Timestamp,
        PageID:    action.PageID,
        Action:    action,
    }
    s.entries[sessionID] = append(s.entries[sessionID], entry)
}

func (s *TimelineService) AddPageEvent(sessionID string, event *PageEvent) {
    s.mu.Lock()
    defer s.mu.Unlock()

    entry := TimelineEntry{
        ID:        uuid.New(),
        Type:      TimelineTypePageEvent,
        Timestamp: event.Timestamp,
        PageID:    event.PageID,
        PageEvent: event,
    }
    s.entries[sessionID] = append(s.entries[sessionID], entry)
}

func (s *TimelineService) GetTimeline(sessionID string) []TimelineEntry {
    s.mu.RLock()
    defer s.mu.RUnlock()

    entries := s.entries[sessionID]
    // Already sorted by insertion order, but ensure by timestamp
    sort.Slice(entries, func(i, j int) bool {
        return entries[i].Timestamp.Before(entries[j].Timestamp)
    })
    return entries
}

func (s *TimelineService) GetTimelineForPage(sessionID string, pageID uuid.UUID) []TimelineEntry {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var filtered []TimelineEntry
    for _, entry := range s.entries[sessionID] {
        if entry.PageID == pageID {
            filtered = append(filtered, entry)
        }
    }
    return filtered
}
```

### 4.3 WebSocket Page Event Messages

#### 4.3.1 Hub Updates

```go
// websocket/hub.go

func (h *Hub) BroadcastPageEvent(sessionID string, event *domain.PageEvent) {
    message := map[string]interface{}{
        "type":       "recording_page_event",
        "session_id": sessionID,
        "event":      event,
        "timestamp":  time.Now().Format(time.RFC3339Nano),
    }
    h.broadcastToSession(sessionID, message)
}

// Also update BroadcastRecordingAction to include pageId
func (h *Hub) BroadcastRecordingAction(sessionID string, action *RecordedAction) {
    message := map[string]interface{}{
        "type":       "recording_action",
        "session_id": sessionID,
        "page_id":    action.PageID.String(),  // NEW
        "action":     action,
        "timestamp":  time.Now().Format(time.RFC3339Nano),
    }
    h.broadcastToSession(sessionID, message)
}
```

### 4.4 Deliverables - Phase 2

- [ ] `TimelineEntry` union type
- [ ] `TimelineService` with unified storage
- [ ] Page event WebSocket broadcast
- [ ] Action broadcast includes `pageId`
- [ ] `GET /timeline` endpoint returns unified timeline
- [ ] Timeline filtering by page
- [ ] Integration tests for timeline ordering

---

## 5. Phase 3: UI Tab Management

**Goal**: Give users visibility into all open pages and ability to switch between them.

### 5.1 New Components

#### 5.1.1 Tab Bar Component

```tsx
// ui/src/domains/recording/components/TabBar.tsx

import { useMemo } from "react";
import { cn } from "@/lib/utils";
import { X, ExternalLink, Globe } from "lucide-react";

interface Page {
  id: string;
  url: string;
  title: string;
  isInitial: boolean;
  status: "active" | "closed";
  hasActivity?: boolean;  // Recent action on this page
}

interface TabBarProps {
  pages: Page[];
  activePageId: string;
  onSelectPage: (pageId: string) => void;
  onClosePage?: (pageId: string) => void;
  className?: string;
}

export function TabBar({
  pages,
  activePageId,
  onSelectPage,
  onClosePage,
  className
}: TabBarProps) {
  const openPages = useMemo(
    () => pages.filter(p => p.status === "active"),
    [pages]
  );

  const getDisplayTitle = (page: Page) => {
    if (page.title) return page.title;
    try {
      return new URL(page.url).hostname;
    } catch {
      return "New Tab";
    }
  };

  const getFaviconUrl = (page: Page) => {
    try {
      const url = new URL(page.url);
      return `https://www.google.com/s2/favicons?domain=${url.hostname}&sz=32`;
    } catch {
      return null;
    }
  };

  return (
    <div className={cn(
      "flex items-center gap-1 bg-muted/50 p-1 rounded-lg overflow-x-auto",
      className
    )}>
      {openPages.map((page) => (
        <button
          key={page.id}
          onClick={() => onSelectPage(page.id)}
          className={cn(
            "flex items-center gap-2 px-3 py-1.5 rounded-md text-sm transition-colors",
            "hover:bg-background/80 min-w-0 max-w-[200px]",
            page.id === activePageId
              ? "bg-background shadow-sm"
              : "text-muted-foreground",
            page.hasActivity && page.id !== activePageId && "ring-2 ring-primary/50"
          )}
        >
          {/* Favicon */}
          {getFaviconUrl(page) ? (
            <img
              src={getFaviconUrl(page)!}
              alt=""
              className="w-4 h-4 flex-shrink-0"
            />
          ) : (
            <Globe className="w-4 h-4 flex-shrink-0 text-muted-foreground" />
          )}

          {/* Title */}
          <span className="truncate">
            {getDisplayTitle(page)}
          </span>

          {/* External link indicator for non-initial pages */}
          {!page.isInitial && (
            <ExternalLink className="w-3 h-3 flex-shrink-0 text-muted-foreground" />
          )}

          {/* Close button (optional, for user-initiated close during recording) */}
          {onClosePage && !page.isInitial && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onClosePage(page.id);
              }}
              className="p-0.5 hover:bg-destructive/20 rounded"
            >
              <X className="w-3 h-3" />
            </button>
          )}
        </button>
      ))}
    </div>
  );
}
```

#### 5.1.2 Page Activity Indicator

```tsx
// ui/src/domains/recording/components/PageActivityIndicator.tsx

import { motion, AnimatePresence } from "framer-motion";

interface PageActivityIndicatorProps {
  isActive: boolean;
  recentActionCount?: number;
}

export function PageActivityIndicator({
  isActive,
  recentActionCount = 0
}: PageActivityIndicatorProps) {
  return (
    <AnimatePresence>
      {recentActionCount > 0 && !isActive && (
        <motion.span
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          exit={{ scale: 0 }}
          className="absolute -top-1 -right-1 bg-primary text-primary-foreground
                     text-xs rounded-full w-5 h-5 flex items-center justify-center"
        >
          {recentActionCount > 9 ? "9+" : recentActionCount}
        </motion.span>
      )}
    </AnimatePresence>
  );
}
```

### 5.2 State Management

#### 5.2.1 Page State Hook

```tsx
// ui/src/domains/recording/hooks/usePages.ts

import { useState, useEffect, useCallback, useMemo } from "react";
import { useWebSocket } from "@/contexts/WebSocketContext";
import { api } from "@/lib/api";

interface Page {
  id: string;
  url: string;
  title: string;
  openerId?: string;
  isInitial: boolean;
  status: "active" | "closed";
  createdAt: string;
  closedAt?: string;
}

interface PageEvent {
  id: string;
  type: "page_created" | "page_navigated" | "page_closed";
  pageId: string;
  url?: string;
  title?: string;
  openerId?: string;
  timestamp: string;
}

export function usePages(sessionId: string | null) {
  const [pages, setPages] = useState<Map<string, Page>>(new Map());
  const [activePageId, setActivePageId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const { lastMessage, subscribe } = useWebSocket();

  // Initial fetch
  useEffect(() => {
    if (!sessionId) return;

    setIsLoading(true);
    api.get(`/recordings/live/${sessionId}/pages`)
      .then((response) => {
        const pageMap = new Map<string, Page>();
        response.data.pages.forEach((page: Page) => {
          pageMap.set(page.id, page);
        });
        setPages(pageMap);
        setActivePageId(response.data.activePageId);
      })
      .finally(() => setIsLoading(false));
  }, [sessionId]);

  // Subscribe to page events
  useEffect(() => {
    if (!sessionId) return;
    return subscribe("recording", sessionId);
  }, [sessionId, subscribe]);

  // Handle WebSocket messages
  useEffect(() => {
    if (!lastMessage || lastMessage.type !== "recording_page_event") return;
    if (lastMessage.session_id !== sessionId) return;

    const event = lastMessage.event as PageEvent;

    setPages((prev) => {
      const next = new Map(prev);

      switch (event.type) {
        case "page_created":
          next.set(event.pageId, {
            id: event.pageId,
            url: event.url || "",
            title: event.title || "",
            openerId: event.openerId,
            isInitial: false,
            status: "active",
            createdAt: event.timestamp,
          });
          break;

        case "page_navigated":
          const existing = next.get(event.pageId);
          if (existing) {
            next.set(event.pageId, {
              ...existing,
              url: event.url || existing.url,
              title: event.title || existing.title,
            });
          }
          break;

        case "page_closed":
          const closing = next.get(event.pageId);
          if (closing) {
            next.set(event.pageId, {
              ...closing,
              status: "closed",
              closedAt: event.timestamp,
            });
          }
          break;
      }

      return next;
    });
  }, [lastMessage, sessionId]);

  // Switch active page
  const switchToPage = useCallback(async (pageId: string) => {
    if (!sessionId) return;

    try {
      await api.post(`/recordings/live/${sessionId}/pages/${pageId}/activate`);
      setActivePageId(pageId);
    } catch (error) {
      console.error("Failed to switch page:", error);
    }
  }, [sessionId]);

  // Derived values
  const openPages = useMemo(
    () => Array.from(pages.values()).filter(p => p.status === "active"),
    [pages]
  );

  const activePage = useMemo(
    () => activePageId ? pages.get(activePageId) : null,
    [pages, activePageId]
  );

  return {
    pages: Array.from(pages.values()),
    openPages,
    activePageId,
    activePage,
    switchToPage,
    isLoading,
  };
}
```

### 5.3 RecordingSession Integration

```tsx
// Update RecordingSession.tsx to include TabBar

import { TabBar } from "./components/TabBar";
import { usePages } from "./hooks/usePages";

export function RecordingSession() {
  // ... existing state ...

  const {
    openPages,
    activePageId,
    activePage,
    switchToPage
  } = usePages(sessionId);

  return (
    <div className="flex flex-col h-full">
      {/* Tab Bar */}
      {openPages.length > 1 && (
        <TabBar
          pages={openPages}
          activePageId={activePageId || ""}
          onSelectPage={switchToPage}
          className="mb-2"
        />
      )}

      {/* Active Page Info */}
      {activePage && (
        <div className="text-sm text-muted-foreground mb-2 truncate">
          {activePage.url}
        </div>
      )}

      {/* Live Preview - now receives activePageId */}
      <PlaywrightView
        sessionId={sessionId}
        pageId={activePageId}  // NEW: Pass active page
        // ... other props
      />

      {/* Timeline */}
      <RecordingTimeline
        sessionId={sessionId}
        pages={openPages}  // NEW: Pass pages for filtering/display
      />
    </div>
  );
}
```

### 5.4 API Endpoints

```go
// GET /api/v1/recordings/live/{sessionId}/pages
// Returns all pages in the session

// POST /api/v1/recordings/live/{sessionId}/pages/{pageId}/activate
// Switches the active page for frame streaming and input forwarding
```

### 5.5 Deliverables - Phase 3

- [ ] `TabBar` component
- [ ] `PageActivityIndicator` component
- [ ] `usePages` hook
- [ ] `GET /pages` endpoint
- [ ] `POST /pages/{pageId}/activate` endpoint
- [ ] Integration with `RecordingSession`
- [ ] Pass `pageId` to `PlaywrightView`
- [ ] Visual feedback for page switching
- [ ] E2E tests for tab switching

---

## 6. Phase 4: Multi-Page Timeline

**Goal**: Display a unified timeline with actions and page events, color-coded by page.

### 6.1 Timeline Component Updates

#### 6.1.1 Enhanced Timeline Entry

```tsx
// ui/src/domains/recording/components/TimelineEntry.tsx

import { cn } from "@/lib/utils";
import {
  MousePointer2,
  Keyboard,
  Navigation,
  ExternalLink,
  X as CloseIcon,
  ArrowRight
} from "lucide-react";

interface TimelineEntryProps {
  entry: {
    id: string;
    type: "action" | "page_event";
    timestamp: string;
    pageId: string;
    action?: RecordedAction;
    pageEvent?: PageEvent;
  };
  page?: Page;
  isSelected?: boolean;
  onSelect?: () => void;
  pageColorMap: Map<string, string>;  // pageId -> color class
}

export function TimelineEntry({
  entry,
  page,
  isSelected,
  onSelect,
  pageColorMap
}: TimelineEntryProps) {
  const pageColor = pageColorMap.get(entry.pageId) || "bg-gray-500";

  if (entry.type === "page_event") {
    return (
      <div className={cn(
        "flex items-center gap-3 py-2 px-3 rounded-md",
        "bg-muted/30 border border-dashed border-muted-foreground/30"
      )}>
        {/* Page color indicator */}
        <div className={cn("w-2 h-2 rounded-full", pageColor)} />

        {/* Icon */}
        {entry.pageEvent?.type === "page_created" && (
          <ExternalLink className="w-4 h-4 text-green-500" />
        )}
        {entry.pageEvent?.type === "page_closed" && (
          <CloseIcon className="w-4 h-4 text-red-500" />
        )}
        {entry.pageEvent?.type === "page_navigated" && (
          <ArrowRight className="w-4 h-4 text-blue-500" />
        )}

        {/* Description */}
        <span className="text-sm text-muted-foreground">
          {entry.pageEvent?.type === "page_created" && "New tab opened"}
          {entry.pageEvent?.type === "page_closed" && "Tab closed"}
          {entry.pageEvent?.type === "page_navigated" && "Navigated to"}
          {entry.pageEvent?.url && (
            <span className="ml-1 font-mono text-xs">
              {new URL(entry.pageEvent.url).pathname}
            </span>
          )}
        </span>
      </div>
    );
  }

  // Action entry
  const action = entry.action!;

  return (
    <button
      onClick={onSelect}
      className={cn(
        "flex items-center gap-3 py-2 px-3 rounded-md w-full text-left",
        "hover:bg-muted/50 transition-colors",
        isSelected && "bg-primary/10 ring-1 ring-primary"
      )}
    >
      {/* Page color indicator */}
      <div className={cn("w-2 h-2 rounded-full flex-shrink-0", pageColor)} />

      {/* Action icon */}
      {action.actionType === "click" && (
        <MousePointer2 className="w-4 h-4 text-muted-foreground flex-shrink-0" />
      )}
      {action.actionType === "input" && (
        <Keyboard className="w-4 h-4 text-muted-foreground flex-shrink-0" />
      )}
      {action.actionType === "navigate" && (
        <Navigation className="w-4 h-4 text-muted-foreground flex-shrink-0" />
      )}

      {/* Action description */}
      <div className="flex-1 min-w-0">
        <div className="text-sm truncate">
          {action.actionType === "click" && `Click: ${action.selector?.primary || "element"}`}
          {action.actionType === "input" && `Type: "${action.payload?.text || "..."}"`}
          {action.actionType === "navigate" && `Navigate: ${action.url}`}
        </div>

        {/* Page badge (when filtering is off) */}
        {page && (
          <div className="text-xs text-muted-foreground truncate">
            {page.title || new URL(page.url).hostname}
          </div>
        )}
      </div>

      {/* Timestamp */}
      <span className="text-xs text-muted-foreground flex-shrink-0">
        {formatTimestamp(entry.timestamp)}
      </span>
    </button>
  );
}
```

#### 6.1.2 Timeline Container with Filtering

```tsx
// ui/src/domains/recording/components/RecordingTimeline.tsx

import { useState, useMemo } from "react";
import { TimelineEntry } from "./TimelineEntry";
import { Select } from "@/components/ui/select";

// Predefined color palette for pages
const PAGE_COLORS = [
  "bg-blue-500",
  "bg-green-500",
  "bg-purple-500",
  "bg-orange-500",
  "bg-pink-500",
  "bg-cyan-500",
  "bg-yellow-500",
  "bg-red-500",
];

interface RecordingTimelineProps {
  entries: TimelineEntry[];
  pages: Page[];
  onSelectEntry?: (entry: TimelineEntry) => void;
  selectedEntryId?: string;
}

export function RecordingTimeline({
  entries,
  pages,
  onSelectEntry,
  selectedEntryId,
}: RecordingTimelineProps) {
  const [filterPageId, setFilterPageId] = useState<string | "all">("all");

  // Assign colors to pages
  const pageColorMap = useMemo(() => {
    const map = new Map<string, string>();
    pages.forEach((page, index) => {
      map.set(page.id, PAGE_COLORS[index % PAGE_COLORS.length]);
    });
    return map;
  }, [pages]);

  // Filter entries
  const filteredEntries = useMemo(() => {
    if (filterPageId === "all") return entries;
    return entries.filter(e => e.pageId === filterPageId);
  }, [entries, filterPageId]);

  // Group by page for legend
  const pageMap = useMemo(() => {
    const map = new Map<string, Page>();
    pages.forEach(p => map.set(p.id, p));
    return map;
  }, [pages]);

  return (
    <div className="flex flex-col h-full">
      {/* Header with filter */}
      <div className="flex items-center justify-between p-2 border-b">
        <span className="text-sm font-medium">
          Timeline ({filteredEntries.length})
        </span>

        {pages.length > 1 && (
          <Select
            value={filterPageId}
            onValueChange={setFilterPageId}
          >
            <option value="all">All pages</option>
            {pages.map((page) => (
              <option key={page.id} value={page.id}>
                <span className={cn("w-2 h-2 rounded-full mr-2", pageColorMap.get(page.id))} />
                {page.title || new URL(page.url).hostname}
              </option>
            ))}
          </Select>
        )}
      </div>

      {/* Legend (when showing all pages) */}
      {filterPageId === "all" && pages.length > 1 && (
        <div className="flex flex-wrap gap-2 p-2 border-b bg-muted/20">
          {pages.map((page) => (
            <div key={page.id} className="flex items-center gap-1 text-xs">
              <div className={cn("w-2 h-2 rounded-full", pageColorMap.get(page.id))} />
              <span className="truncate max-w-[100px]">
                {page.title || new URL(page.url).hostname}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Entries */}
      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {filteredEntries.map((entry) => (
          <TimelineEntry
            key={entry.id}
            entry={entry}
            page={pageMap.get(entry.pageId)}
            isSelected={entry.id === selectedEntryId}
            onSelect={() => onSelectEntry?.(entry)}
            pageColorMap={pageColorMap}
          />
        ))}

        {filteredEntries.length === 0 && (
          <div className="text-center text-muted-foreground py-8">
            No actions recorded yet
          </div>
        )}
      </div>
    </div>
  );
}
```

### 6.2 Timeline Data Fetching

```tsx
// ui/src/domains/recording/hooks/useTimeline.ts

import { useState, useEffect, useCallback } from "react";
import { useWebSocket } from "@/contexts/WebSocketContext";
import { api } from "@/lib/api";

export function useTimeline(sessionId: string | null) {
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const { lastMessage } = useWebSocket();

  // Initial fetch
  useEffect(() => {
    if (!sessionId) return;

    setIsLoading(true);
    api.get(`/recordings/live/${sessionId}/timeline`)
      .then((response) => {
        setEntries(response.data.entries);
      })
      .finally(() => setIsLoading(false));
  }, [sessionId]);

  // Handle real-time updates
  useEffect(() => {
    if (!lastMessage) return;
    if (lastMessage.session_id !== sessionId) return;

    if (lastMessage.type === "recording_action") {
      const action = lastMessage.action;
      setEntries((prev) => [...prev, {
        id: action.id,
        type: "action",
        timestamp: action.timestamp,
        pageId: lastMessage.page_id,
        action,
      }]);
    }

    if (lastMessage.type === "recording_page_event") {
      const event = lastMessage.event;
      setEntries((prev) => [...prev, {
        id: event.id,
        type: "page_event",
        timestamp: event.timestamp,
        pageId: event.pageId,
        pageEvent: event,
      }]);
    }
  }, [lastMessage, sessionId]);

  return { entries, isLoading };
}
```

### 6.3 Deliverables - Phase 4

- [ ] `TimelineEntry` component with page colors
- [ ] `RecordingTimeline` with filtering
- [ ] Page color assignment logic
- [ ] Timeline legend
- [ ] `useTimeline` hook
- [ ] `GET /timeline` returns unified entries
- [ ] Real-time timeline updates via WebSocket
- [ ] Visual tests for timeline rendering

---

## 7. Phase 5: Frame Streaming Optimization

**Goal**: Efficiently stream frames for the active page while maintaining awareness of other pages.

### 7.1 Active Page Switching

When the user switches tabs in the UI, the system must:
1. Tell the driver to stream frames from the new page
2. Update input forwarding to the new page
3. Update the UI preview

#### 7.1.1 API Endpoint

```go
// POST /api/v1/recordings/live/{sessionId}/pages/{pageId}/activate
func (h *Handler) ActivatePage(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "sessionId")
    pageID := chi.URLParam(r, "pageId")

    pageUUID, err := uuid.Parse(pageID)
    if err != nil {
        http.Error(w, "invalid page ID", http.StatusBadRequest)
        return
    }

    session, err := h.sessions.Get(sessionID)
    if err != nil {
        http.Error(w, "session not found", http.StatusNotFound)
        return
    }

    // Verify page exists and is open
    page, ok := session.GetPage(pageUUID)
    if !ok {
        http.Error(w, "page not found", http.StatusNotFound)
        return
    }
    if page.Status != domain.PageStatusActive {
        http.Error(w, "page is closed", http.StatusBadRequest)
        return
    }

    // Get driver's page ID for this page
    driverPageID := session.GetDriverPageID(pageUUID)
    if driverPageID == "" {
        http.Error(w, "page not registered with driver", http.StatusInternalServerError)
        return
    }

    // Tell driver to switch active page
    err = h.driver.SetActivePage(r.Context(), sessionID, driverPageID)
    if err != nil {
        http.Error(w, "failed to switch page", http.StatusInternalServerError)
        return
    }

    // Update session state
    session.SetActivePage(pageUUID)

    // Broadcast page switch to all clients
    h.wsHub.BroadcastPageSwitch(sessionID, pageUUID)

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "activePageId": pageID,
    })
}
```

### 7.2 Driver Protocol for Page Switching

```go
// automation/driver/client.go

// SetActivePage switches which page receives frame streaming and input
func (c *Client) SetActivePage(ctx context.Context, sessionID, driverPageID string) error {
    req := SetActivePageRequest{
        SessionID:    sessionID,
        DriverPageID: driverPageID,
    }

    resp, err := c.httpClient.Post(
        c.baseURL+"/sessions/"+sessionID+"/active-page",
        "application/json",
        jsonBody(req),
    )
    if err != nil {
        return fmt.Errorf("set active page: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("set active page failed: %s", resp.Status)
    }

    return nil
}
```

### 7.3 PlaywrightView Updates

```tsx
// Update PlaywrightView to handle page switching

interface PlaywrightViewProps {
  sessionId: string;
  pageId?: string;  // NEW: Which page to display
  // ... other props
}

export function PlaywrightView({ sessionId, pageId, ...props }: PlaywrightViewProps) {
  const [isTransitioning, setIsTransitioning] = useState(false);
  const prevPageIdRef = useRef(pageId);

  // Handle page switch transition
  useEffect(() => {
    if (prevPageIdRef.current !== pageId) {
      setIsTransitioning(true);
      // Brief transition effect while waiting for new frames
      const timer = setTimeout(() => setIsTransitioning(false), 500);
      prevPageIdRef.current = pageId;
      return () => clearTimeout(timer);
    }
  }, [pageId]);

  // Frame fetching now uses pageId
  const fetchFrame = useCallback(async () => {
    const params = new URLSearchParams({
      quality: "65",
      ...(pageId && { pageId }),
    });

    const response = await fetch(
      `/api/v1/recordings/live/${sessionId}/frame?${params}`
    );
    // ... rest of frame handling
  }, [sessionId, pageId]);

  return (
    <div className={cn(
      "relative",
      isTransitioning && "opacity-50 transition-opacity"
    )}>
      <canvas ref={canvasRef} />

      {isTransitioning && (
        <div className="absolute inset-0 flex items-center justify-center bg-background/50">
          <Loader2 className="w-6 h-6 animate-spin" />
        </div>
      )}
    </div>
  );
}
```

### 7.4 Input Forwarding for Active Page

When forwarding input, the WebSocket hub must route to the correct page:

```go
// websocket/hub.go

case "recording_input":
    sessionID := msg["session_id"].(string)
    input := msg["input"].(map[string]interface{})

    // Get active page for this session
    session, err := h.sessions.Get(sessionID)
    if err != nil {
        h.sendError(client, "session not found")
        return
    }

    activePageID := session.GetActivePageID()
    driverPageID := session.GetDriverPageID(activePageID)

    // Forward input to the active page
    h.inputForwarder(sessionID, driverPageID, input)
```

### 7.5 Deliverables - Phase 5

- [ ] `POST /pages/{pageId}/activate` endpoint
- [ ] Driver `SetActivePage` method
- [ ] WebSocket page switch broadcast
- [ ] PlaywrightView page transition handling
- [ ] Frame endpoint accepts `pageId` parameter
- [ ] Input forwarding uses active page
- [ ] Frame stream seamlessly switches between pages
- [ ] Performance testing for page switching latency

---

## 8. Phase 6: Multi-Page Playback

**Goal**: Enable playback of recorded multi-page workflows.

### 8.1 Playback Architecture

During playback, the system must:
1. Track expected vs actual pages
2. Wait for pages to open when expected
3. Execute actions on the correct page
4. Handle page closures

### 8.2 Execution Model

```go
// services/execution/multi_page_executor.go

type MultiPageExecutor struct {
    driver      *driver.Client
    sessionID   string
    pageMap     map[uuid.UUID]string  // Recorded pageID -> Runtime driverPageID
    pageChan    chan PageCreatedEvent // Receives page creation from driver
    mu          sync.RWMutex
}

func (e *MultiPageExecutor) Execute(ctx context.Context, timeline []TimelineEntry) error {
    for _, entry := range timeline {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        switch entry.Type {
        case TimelineTypePageEvent:
            if err := e.handlePageEvent(ctx, entry.PageEvent); err != nil {
                return fmt.Errorf("page event %s: %w", entry.ID, err)
            }

        case TimelineTypeAction:
            if err := e.executeAction(ctx, entry.Action); err != nil {
                return fmt.Errorf("action %s: %w", entry.ID, err)
            }
        }
    }

    return nil
}

func (e *MultiPageExecutor) handlePageEvent(ctx context.Context, event *PageEvent) error {
    switch event.Type {
    case PageEventCreated:
        // Wait for the page to be created (driver will notify us)
        select {
        case created := <-e.pageChan:
            e.mu.Lock()
            e.pageMap[event.PageID] = created.DriverPageID
            e.mu.Unlock()

        case <-time.After(10 * time.Second):
            return fmt.Errorf("timeout waiting for page %s to open", event.PageID)

        case <-ctx.Done():
            return ctx.Err()
        }

    case PageEventClosed:
        e.mu.Lock()
        delete(e.pageMap, event.PageID)
        e.mu.Unlock()

    case PageEventNavigated:
        // Navigation is implicit from actions, just log
    }

    return nil
}

func (e *MultiPageExecutor) executeAction(ctx context.Context, action *RecordedAction) error {
    e.mu.RLock()
    driverPageID, ok := e.pageMap[action.PageID]
    e.mu.RUnlock()

    if !ok {
        return fmt.Errorf("page %s not found in runtime", action.PageID)
    }

    // Switch driver focus to this page
    if err := e.driver.SetActivePage(ctx, e.sessionID, driverPageID); err != nil {
        return fmt.Errorf("switch to page: %w", err)
    }

    // Execute the action
    return e.driver.ExecuteAction(ctx, e.sessionID, driverPageID, action)
}
```

### 8.3 Waiting for Page Creation

The executor must wait for pages that don't exist yet:

```go
// When the recorded timeline says a page was created, we must wait
// for the actual runtime browser to create it (via click, etc.)

func (e *MultiPageExecutor) waitForPage(ctx context.Context, expectedPageID uuid.UUID) error {
    timeout := time.After(10 * time.Second)

    for {
        select {
        case created := <-e.pageChan:
            // Check if this is the page we're waiting for
            // (Match by opener relationship or URL pattern)
            if e.matchesExpectedPage(created, expectedPageID) {
                e.mu.Lock()
                e.pageMap[expectedPageID] = created.DriverPageID
                e.mu.Unlock()
                return nil
            }
            // Not the page we wanted, keep waiting

        case <-timeout:
            return fmt.Errorf("timeout waiting for page %s", expectedPageID)

        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### 8.4 Page Matching Strategy

When a new page opens during playback, we need to match it to the expected recorded page:

```go
func (e *MultiPageExecutor) matchesExpectedPage(actual PageCreatedEvent, expectedID uuid.UUID) bool {
    expected := e.getExpectedPage(expectedID)
    if expected == nil {
        return false
    }

    // Strategy 1: Match by opener
    if expected.OpenerID != nil {
        expectedOpenerDriverID := e.pageMap[*expected.OpenerID]
        if actual.OpenerDriverPageID == expectedOpenerDriverID {
            return true
        }
    }

    // Strategy 2: Match by URL pattern (if opener doesn't match)
    if urlPatternMatch(actual.URL, expected.URL) {
        return true
    }

    // Strategy 3: If this is the only pending page, assume it's a match
    if e.onlyOnePendingPage() {
        return true
    }

    return false
}
```

### 8.5 Workflow Storage Format

The saved workflow must include page definitions:

```json
{
  "id": "workflow-uuid",
  "name": "Multi-Tab Checkout Flow",
  "version": 2,
  "pages": [
    {
      "id": "page-1-uuid",
      "isInitial": true,
      "startUrl": "https://shop.example.com"
    },
    {
      "id": "page-2-uuid",
      "isInitial": false,
      "openerId": "page-1-uuid",
      "openTrigger": "link_click"
    }
  ],
  "timeline": [
    {
      "id": "entry-1",
      "type": "action",
      "pageId": "page-1-uuid",
      "timestamp": "2025-01-15T10:00:00Z",
      "action": { ... }
    },
    {
      "id": "entry-2",
      "type": "page_event",
      "pageId": "page-2-uuid",
      "timestamp": "2025-01-15T10:00:01Z",
      "pageEvent": {
        "type": "page_created",
        "url": "https://shop.example.com/cart",
        "openerId": "page-1-uuid"
      }
    },
    ...
  ]
}
```

### 8.6 Deliverables - Phase 6

- [ ] `MultiPageExecutor` implementation
- [ ] Page event handling during playback
- [ ] Page matching strategy
- [ ] Timeout handling for missing pages
- [ ] Workflow format v2 with page definitions
- [ ] Migration for v1 workflows (single implicit page)
- [ ] Execution UI shows multi-page progress
- [ ] Integration tests for multi-page playback
- [ ] E2E test: record multi-tab → save → replay

---

## 9. Data Model Changes

### 9.1 Database Schema Changes

```sql
-- New table: recording_pages
CREATE TABLE recording_pages (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES recording_sessions(id),
    driver_page_id TEXT NOT NULL,
    url TEXT,
    title TEXT,
    opener_id UUID REFERENCES recording_pages(id),
    is_initial BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,

    UNIQUE(session_id, driver_page_id)
);

-- Add page_id to recorded_actions
ALTER TABLE recorded_actions
    ADD COLUMN page_id UUID REFERENCES recording_pages(id);

-- New table: page_events
CREATE TABLE page_events (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES recording_sessions(id),
    page_id UUID NOT NULL REFERENCES recording_pages(id),
    event_type TEXT NOT NULL,  -- 'created', 'navigated', 'closed'
    url TEXT,
    title TEXT,
    opener_id UUID REFERENCES recording_pages(id),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add index for timeline queries
CREATE INDEX idx_page_events_session_timestamp
    ON page_events(session_id, timestamp);
CREATE INDEX idx_recorded_actions_page
    ON recorded_actions(page_id);
```

### 9.2 Proto Changes (if using protobuf)

```protobuf
message Page {
  string id = 1;
  string session_id = 2;
  string url = 3;
  string title = 4;
  optional string opener_id = 5;
  bool is_initial = 6;
  string status = 7;
  google.protobuf.Timestamp created_at = 8;
  optional google.protobuf.Timestamp closed_at = 9;
}

message PageEvent {
  string id = 1;
  PageEventType type = 2;
  string page_id = 3;
  string url = 4;
  string title = 5;
  optional string opener_id = 6;
  google.protobuf.Timestamp timestamp = 7;
}

enum PageEventType {
  PAGE_EVENT_TYPE_UNSPECIFIED = 0;
  PAGE_EVENT_TYPE_CREATED = 1;
  PAGE_EVENT_TYPE_NAVIGATED = 2;
  PAGE_EVENT_TYPE_CLOSED = 3;
}

message TimelineEntry {
  string id = 1;
  TimelineEntryType type = 2;
  string page_id = 3;
  google.protobuf.Timestamp timestamp = 4;

  oneof content {
    RecordedAction action = 5;
    PageEvent page_event = 6;
  }
}
```

---

## 10. API Specification

### 10.1 New Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/recordings/live/{sessionId}/pages` | List all pages in session |
| POST | `/recordings/live/{sessionId}/pages/{pageId}/activate` | Switch active page |
| POST | `/recordings/live/{sessionId}/page-event` | Receive page event from driver |
| GET | `/recordings/live/{sessionId}/timeline` | Get unified timeline |

### 10.2 Endpoint Details

#### GET /recordings/live/{sessionId}/pages

**Response:**
```json
{
  "pages": [
    {
      "id": "uuid",
      "url": "https://example.com",
      "title": "Example",
      "openerId": null,
      "isInitial": true,
      "status": "active",
      "createdAt": "2025-01-15T10:00:00Z"
    },
    {
      "id": "uuid-2",
      "url": "https://example.com/page2",
      "title": "Page 2",
      "openerId": "uuid",
      "isInitial": false,
      "status": "active",
      "createdAt": "2025-01-15T10:00:05Z"
    }
  ],
  "activePageId": "uuid-2"
}
```

#### POST /recordings/live/{sessionId}/pages/{pageId}/activate

**Response:**
```json
{
  "activePageId": "uuid-2"
}
```

#### GET /recordings/live/{sessionId}/timeline

**Query Parameters:**
- `pageId` (optional): Filter by page
- `since` (optional): Only entries after this timestamp
- `limit` (optional): Max entries to return (default 100)

**Response:**
```json
{
  "entries": [
    {
      "id": "entry-uuid",
      "type": "action",
      "pageId": "page-uuid",
      "timestamp": "2025-01-15T10:00:01Z",
      "action": { ... }
    },
    {
      "id": "entry-uuid-2",
      "type": "page_event",
      "pageId": "page-uuid-2",
      "timestamp": "2025-01-15T10:00:02Z",
      "pageEvent": {
        "type": "page_created",
        "url": "https://example.com/new",
        "openerId": "page-uuid"
      }
    }
  ],
  "hasMore": false
}
```

---

## 11. WebSocket Protocol

### 11.1 New Message Types

#### Server → Client: Page Event

```json
{
  "type": "recording_page_event",
  "session_id": "session-uuid",
  "event": {
    "id": "event-uuid",
    "type": "page_created",
    "pageId": "page-uuid",
    "url": "https://example.com/new",
    "title": "New Page",
    "openerId": "opener-page-uuid",
    "timestamp": "2025-01-15T10:00:02Z"
  }
}
```

#### Server → Client: Page Switch Confirmation

```json
{
  "type": "recording_page_switched",
  "session_id": "session-uuid",
  "active_page_id": "page-uuid"
}
```

#### Updated: Recording Action (includes pageId)

```json
{
  "type": "recording_action",
  "session_id": "session-uuid",
  "page_id": "page-uuid",
  "action": { ... }
}
```

---

## 12. Testing Strategy

### 12.1 Unit Tests

| Component | Test Cases |
|-----------|------------|
| Session page management | Add/get/close pages, driver ID mapping |
| Timeline service | Add entries, ordering, filtering |
| Page matching | Opener matching, URL matching, fallback |
| Multi-page executor | Sequential execution, page waits, timeouts |

### 12.2 Integration Tests

| Scenario | Description |
|----------|-------------|
| Page creation flow | Driver reports page → API stores → WebSocket broadcasts → UI updates |
| Page switching | UI requests switch → API updates → Driver switches → Frames change |
| Timeline sync | Actions and page events maintain correct order |
| Playback flow | Execute multi-page workflow end-to-end |

### 12.3 E2E Tests

```typescript
// e2e/multi-tab-recording.spec.ts

test("records actions across multiple tabs", async ({ page }) => {
  // Start recording session
  await page.goto("/record");
  await page.fill("[data-testid=start-url]", "https://example.com");
  await page.click("[data-testid=start-recording]");

  // Wait for live preview
  await expect(page.locator("[data-testid=live-preview]")).toBeVisible();

  // Perform action that opens new tab (via input forwarding)
  await page.click("[data-testid=live-preview]", { position: { x: 100, y: 200 } });

  // Wait for new tab to appear in tab bar
  await expect(page.locator("[data-testid=tab-bar] button")).toHaveCount(2);

  // Switch to new tab
  await page.click("[data-testid=tab-bar] button:nth-child(2)");

  // Verify preview shows new tab content
  await expect(page.locator("[data-testid=active-page-url]")).toContainText("/new-page");

  // Stop recording
  await page.click("[data-testid=stop-recording]");

  // Verify timeline has page event
  await expect(page.locator("[data-testid=timeline-entry][data-type=page_event]")).toBeVisible();
});

test("replays multi-tab workflow correctly", async ({ page }) => {
  // Load saved multi-tab workflow
  await page.goto("/workflows/multi-tab-test");
  await page.click("[data-testid=run-workflow]");

  // Watch execution progress
  await expect(page.locator("[data-testid=execution-status]")).toContainText("Running");

  // Verify it completes successfully
  await expect(page.locator("[data-testid=execution-status]")).toContainText("Completed", {
    timeout: 60000
  });

  // Check execution log shows multiple pages
  await expect(page.locator("[data-testid=execution-log]")).toContainText("New tab opened");
});
```

### 12.4 Test Data

Create fixture workflows for testing:
- Single page workflow (backwards compatibility)
- Two-tab workflow (click opens new tab)
- Three-tab workflow with closes
- OAuth flow simulation (popup that redirects back)

---

## 13. Migration & Backwards Compatibility

### 13.1 Workflow Format Migration

Existing v1 workflows have no explicit page tracking. Migrate by:

1. Add default page definition
2. Assign all actions to that page
3. Mark as v2 format

```go
func MigrateWorkflowV1ToV2(v1 WorkflowV1) WorkflowV2 {
    // Create implicit initial page
    initialPageID := uuid.New()
    initialPage := PageDefinition{
        ID:        initialPageID,
        IsInitial: true,
        StartURL:  v1.StartURL,
    }

    // Convert actions to timeline entries
    var timeline []TimelineEntry
    for _, action := range v1.Actions {
        timeline = append(timeline, TimelineEntry{
            ID:        uuid.New(),
            Type:      TimelineTypeAction,
            PageID:    initialPageID,  // All actions on initial page
            Timestamp: action.Timestamp,
            Action:    &action,
        })
    }

    return WorkflowV2{
        ID:       v1.ID,
        Name:     v1.Name,
        Version:  2,
        Pages:    []PageDefinition{initialPage},
        Timeline: timeline,
    }
}
```

### 13.2 API Compatibility

- Existing endpoints continue to work
- `RecordedAction` without `pageId` → assigned to active page
- Old clients not sending page-aware requests → work on active page

### 13.3 UI Fallback

When only one page exists:
- Tab bar is hidden (same as current UI)
- Timeline shows no page badges
- Experience is identical to current

---

## 14. Risk Analysis

### 14.1 Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Driver doesn't report all page events | Medium | High | Test extensively with different page opening methods |
| Race condition: action before page_created | Medium | Medium | Buffer actions, retry page matching |
| Frame streaming latency during switch | Low | Low | Show loading indicator, preload frames |
| Memory growth with many pages | Low | Medium | Limit max pages, cleanup closed pages |

### 14.2 UX Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Users confused by multi-page timeline | Medium | Medium | Clear visual design, filtering options |
| Unexpected page switches | Medium | Low | Toast notifications for page events |
| Lost actions during page switch | Low | High | Ensure actions queue during transition |

### 14.3 Performance Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Too many WebSocket messages | Low | Medium | Batch updates, debounce broadcasts |
| Slow playback with many pages | Low | Low | Parallel page creation when possible |
| Database query performance | Low | Low | Proper indexes, pagination |

---

## 15. Edge Cases & Error Handling

### 15.1 Edge Cases

| Scenario | Handling |
|----------|----------|
| Page opens and closes before driver reports | Log warning, skip in timeline |
| Multiple pages open simultaneously | Queue page_created events, process in order |
| Page opened by JavaScript (no user action) | Record as page_created with no opener action |
| iframe creates new page context | Treat as page, not iframe |
| Authentication popup (OAuth) | Handle same as any popup |
| Page crashes | Record as page_closed with error flag |
| Session loses driver connection | Attempt reconnect, mark affected pages uncertain |
| User closes browser tab manually | Record page_closed, may trigger playback wait |

### 15.2 Error Handling

```go
// Page not found during action execution
if _, ok := session.GetPage(action.PageID); !ok {
    // Log warning and attempt recovery
    log.Warn().
        Str("actionId", action.ID.String()).
        Str("pageId", action.PageID.String()).
        Msg("action references unknown page, using active page")
    action.PageID = session.GetActivePageID()
}

// Page closed during playback wait
if waitErr != nil && errors.Is(waitErr, ErrPageClosed) {
    // Page was closed before we could interact with it
    // This may be expected (popup that auto-closes)
    if isOptionalPage(expectedPage) {
        log.Info().Msg("optional page closed, continuing")
        continue
    }
    return fmt.Errorf("required page closed unexpectedly: %w", waitErr)
}
```

### 15.3 Graceful Degradation

If page tracking fails:
1. Fall back to single-page behavior
2. Log detailed diagnostics
3. Continue recording actions (may be on wrong page)
4. Alert user about potential issues

---

## 16. Performance Considerations

### 16.1 Frame Streaming

- Only active page streams full-quality frames
- Background pages: no streaming (or low-FPS thumbnails)
- Page switch should complete in <200ms

### 16.2 WebSocket Traffic

- Page events are small (~500 bytes)
- Batch multiple rapid events if needed
- Use binary frames for efficiency where possible

### 16.3 Database

- Index on `(session_id, timestamp)` for timeline queries
- Index on `(session_id, page_id)` for page-filtered queries
- Consider partitioning by session for large deployments

### 16.4 Memory

- Limit concurrent pages to 20 per session
- Clean up closed page metadata after session ends
- Stream frames directly, don't buffer

---

## 17. Implementation Checklist

### Phase 1: Page Tracking Foundation
- [ ] Create `api/domain/page.go` with Page and PageEvent types
- [ ] Add `PageID` field to `RecordedAction` in `driver/types.go`
- [ ] Implement page tracking in Session (add/get/list/close)
- [ ] Add driver-to-Vrooli page ID mapping
- [ ] Create `POST /page-event` handler
- [ ] Update `ReceiveRecordingAction` to assign page IDs
- [ ] Add page event to timeline storage
- [ ] Write unit tests for session page management
- [ ] Integration test: receive page event → store → retrieve

### Phase 2: Page Lifecycle Events
- [ ] Create `TimelineEntry` union type
- [ ] Implement `TimelineService` with unified storage
- [ ] Add `BroadcastPageEvent` to WebSocket hub
- [ ] Update `BroadcastRecordingAction` to include `pageId`
- [ ] Create `GET /timeline` endpoint
- [ ] Add timeline filtering by page
- [ ] Write integration tests for timeline ordering
- [ ] Test WebSocket broadcasts

### Phase 3: UI Tab Management
- [ ] Create `TabBar` component
- [ ] Create `PageActivityIndicator` component
- [ ] Implement `usePages` hook
- [ ] Create `GET /pages` endpoint
- [ ] Create `POST /pages/{pageId}/activate` endpoint
- [ ] Integrate TabBar into RecordingSession
- [ ] Update PlaywrightView to receive `pageId`
- [ ] Add page switch transition effects
- [ ] Write component tests for TabBar
- [ ] E2E test: switch between tabs

### Phase 4: Multi-Page Timeline
- [ ] Create enhanced `TimelineEntry` component with page colors
- [ ] Update `RecordingTimeline` with filtering
- [ ] Implement page color assignment logic
- [ ] Add timeline legend component
- [ ] Create `useTimeline` hook
- [ ] Update `GET /timeline` to return unified entries
- [ ] Handle real-time timeline updates
- [ ] Write visual tests for timeline rendering
- [ ] Test timeline filtering

### Phase 5: Frame Streaming Optimization
- [ ] Implement `POST /pages/{pageId}/activate` endpoint
- [ ] Add `SetActivePage` to driver client
- [ ] Add WebSocket page switch broadcast
- [ ] Update PlaywrightView with transition handling
- [ ] Update frame endpoint to accept `pageId`
- [ ] Route input forwarding to active page
- [ ] Test frame switching performance
- [ ] Performance benchmarks for page switching

### Phase 6: Multi-Page Playback
- [ ] Implement `MultiPageExecutor`
- [ ] Add page event handling during playback
- [ ] Implement page matching strategy
- [ ] Add timeout handling for missing pages
- [ ] Define workflow format v2 with page definitions
- [ ] Write migration for v1 workflows
- [ ] Update execution UI for multi-page progress
- [ ] Write integration tests for multi-page playback
- [ ] E2E test: record → save → replay multi-tab workflow

### Final Validation
- [ ] Full regression test on single-page workflows
- [ ] Load testing with 10+ pages
- [ ] Security review (page isolation)
- [ ] Documentation update
- [ ] Changelog entry

---

## Appendix A: Sequence Diagrams

### A.1 Recording with New Tab

```
User          UI           API          Driver       Browser
 │             │            │             │            │
 │─────────────┼────────────┼─────────────┼────────────┤
 │ Click link  │            │             │            │
 │ (target=    │            │             │            │
 │  _blank)    │            │             │            │
 │             │ forward    │             │            │
 │             │ input ─────┼────────────>│            │
 │             │            │             │───────────>│
 │             │            │             │            │ Open new tab
 │             │            │             │<───────────│ page event
 │             │            │ POST /page- │            │
 │             │            │ event <─────│            │
 │             │            │             │            │
 │             │            │ Create Page │            │
 │             │            │ Store event │            │
 │             │            │             │            │
 │             │ WS: page_  │             │            │
 │             │ event <────│             │            │
 │             │            │             │            │
 │             │ Add tab    │             │            │
 │             │ to TabBar  │             │            │
 │             │            │             │            │
 │             │ Add entry  │             │            │
 │             │ to timeline│             │            │
 │             │            │             │            │
```

### A.2 Switching Active Page

```
User          UI           API          Driver       Browser
 │             │            │             │            │
 │ Click tab   │            │             │            │
 │────────────>│            │             │            │
 │             │ POST       │             │            │
 │             │ /activate ─┼────────────>│            │
 │             │            │             │            │
 │             │            │ Set active  │            │
 │             │            │ page ───────┼───────────>│
 │             │            │             │            │
 │             │            │<────────────┼────────────│ OK
 │             │            │             │            │
 │             │<───────────│ OK          │            │
 │             │            │             │            │
 │             │ Update     │             │            │
 │             │ preview ───┼─────────────┼───────────>│
 │             │            │             │ New frames │
 │             │<───────────┼─────────────┼────────────│
 │             │ Show new   │             │            │
 │             │ page       │             │            │
 │             │            │             │            │
```

---

## Appendix B: Glossary

| Term | Definition |
|------|------------|
| **Page** | A browser tab/window within a recording session |
| **Active Page** | The page currently receiving frame streaming and input |
| **Initial Page** | The first page created when recording starts |
| **Opener** | The page that triggered creation of another page |
| **Driver Page ID** | Playwright's internal identifier for a page |
| **Vrooli Page ID** | Our UUID for a page (stable across session) |
| **Timeline** | Ordered list of actions and page events |
| **Page Event** | Lifecycle event: created, navigated, or closed |

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-12-26 | Claude | Initial draft |

---

**Next Steps**: Review this plan, provide feedback, and approve before implementation begins.
