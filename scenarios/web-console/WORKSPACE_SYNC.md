# Web Console Workspace Synchronization

This document describes the cross-device workspace synchronization feature implemented for the web-console scenario.

## Overview

The web-console now persists tab layout, labels, colors, and session associations in the API server and synchronizes changes in real-time across all connected clients via WebSocket. This enables users to:

- Open web-console on multiple devices and see the same tabs
- Create/rename/close tabs on one device and see changes immediately on all others
- Maintain consistent terminal session state across device switches
- Survive browser refreshes without losing tab layout

## Architecture

### Server-Side (Go API)

#### 1. Workspace Persistence Layer (`workspace.go`)
- **Storage**: JSON file at `data/sessions/workspace.json`
- **Data Model**:
  ```go
  type workspace struct {
    ActiveTabID string
    Version     int64
    Tabs        []tabMeta
  }

  type tabMeta struct {
    ID        string
    Label     string
    ColorID   string
    Order     int
    SessionID *string // nil if no active session
  }
  ```

- **Thread Safety**: All operations protected by `sync.RWMutex`
- **Atomic Updates**: Version number increments on every change

#### 2. REST API Endpoints (`http_handlers.go`)
- `GET /api/v1/workspace` - Fetch current workspace state
- `PUT /api/v1/workspace` - Replace entire workspace state
- `POST /api/v1/workspace/tabs` - Create new tab
- `PATCH /api/v1/workspace/tabs/{id}` - Update tab label/color
- `DELETE /api/v1/workspace/tabs/{id}` - Remove tab

#### 3. WebSocket Broadcast System (`workspace_websocket.go`)
- **Endpoint**: `GET /api/v1/workspace/stream`
- **Protocol**:
  - Server sends initial workspace state on connect
  - Server broadcasts events to all connected clients when workspace changes
  - Client messages are ignored (read-only stream)

#### 4. Event Types
```json
{
  "type": "tab-added",
  "timestamp": "2025-10-01T12:34:56Z",
  "payload": {
    "id": "tab-123",
    "label": "Terminal 1",
    "colorId": "sky",
    "order": 0,
    "sessionId": "session-xyz"
  }
}
```

Event types:
- `workspace-full-update` - Complete state replacement
- `tab-added` - New tab created
- `tab-updated` - Tab label or color changed
- `tab-removed` - Tab deleted
- `active-tab-changed` - Different tab selected
- `session-attached` - Terminal session started in tab
- `session-detached` - Terminal session ended

#### 5. Session Lifecycle Integration (`session_manager.go`)
- When session created: Automatically attach to tab if `tabId` provided in request
- When session closes: Automatically detach from associated tab
- Sessions outlive browser refreshes—workspace reconnects to running sessions

### Client-Side (JavaScript)

#### 1. Initialization (`app.js`)
```javascript
async function initializeWorkspace() {
  // 1. Fetch workspace from server
  const workspace = await fetch('/api/v1/workspace')

  // 2. Recreate tabs from workspace state
  workspace.tabs.forEach(tab => createTerminalTab(tab))

  // 3. Reconnect to active sessions
  tabs.filter(t => t.sessionId).forEach(reconnectSession)

  // 4. Connect workspace WebSocket for live updates
  connectWorkspaceWebSocket()
}
```

#### 2. Real-Time Sync
- **Tab Creation**: Local create → POST to server → WebSocket broadcasts to others
- **Tab Update**: Local edit → PATCH to server → WebSocket broadcasts to others
- **Tab Deletion**: Local delete → DELETE to server → WebSocket broadcasts to others
- **Tab Switch**: Local switch → PUT workspace → WebSocket broadcasts to others

#### 3. Conflict Resolution
- **Optimistic UI**: Local changes applied immediately
- **Server Authority**: WebSocket events override local state
- **Reconnection**: On disconnect, reconnect after 3 seconds and re-fetch workspace

#### 4. Session Management
- `tabId` field added to `createSessionRequest`
- Server automatically links session to tab on creation
- UI sends `tabId` when starting sessions
- Workspace detaches sessions automatically on close

## Data Flow Examples

### Example 1: Creating a Tab on Device A

```
Device A (Browser)
  │
  ├─> createTerminalTab()           // Local tab created
  │
  ├─> POST /api/v1/workspace/tabs   // Sync to server
      {
        "id": "tab-456",
        "label": "Terminal 2",
        "colorId": "emerald",
        "order": 1
      }
      │
      └─> Server persists to disk
          │
          ├─> Broadcasts via WebSocket:
              {
                "type": "tab-added",
                "payload": {...}
              }
              │
              ├─> Device B receives event
              │   └─> handleTabAdded()
              │       └─> createTerminalTab() // Remote tab appears
              │
              └─> Device C receives event
                  └─> handleTabAdded()
                      └─> createTerminalTab() // Remote tab appears
```

### Example 2: Starting a Session on Device B

```
Device B (Browser)
  │
  ├─> startSession(tab, {reason: 'new-tab'})
  │
  ├─> POST /api/v1/sessions
      {
        "tabId": "tab-456",
        "reason": "new-tab"
      }
      │
      └─> Server creates session
          │
          ├─> Attaches session to tab-456
          │
          ├─> Broadcasts via WebSocket:
              {
                "type": "session-attached",
                "payload": {
                  "tabId": "tab-456",
                  "sessionId": "session-abc"
                }
              }
              │
              ├─> Device A receives event
              │   └─> handleSessionAttached()
              │       └─> Tab shows "running" state
              │
              └─> Device C receives event
                  └─> handleSessionAttached()
                      └─> Tab shows "running" state
```

### Example 3: Closing a Tab on Device C

```
Device C (Browser)
  │
  ├─> closeTab("tab-456")
  │   │
  │   ├─> stopSession()            // End terminal session
  │   ├─> destroyTerminalTab()     // Remove from DOM
  │   └─> DELETE /api/v1/workspace/tabs/tab-456
      │
      └─> Server removes tab
          │
          ├─> Broadcasts via WebSocket:
              {
                "type": "tab-removed",
                "payload": {"id": "tab-456"}
              }
              │
              ├─> Device A receives event
              │   └─> handleTabRemoved()
              │       └─> destroyTerminalTab()
              │
              └─> Device B receives event
                  └─> handleTabRemoved()
                      └─> destroyTerminalTab()
```

## Testing

### Unit Tests (`workspace_test.go`)
- ✅ Create and load workspace
- ✅ Add/update/remove tabs
- ✅ Attach/detach sessions
- ✅ Event broadcasting to subscribers
- ✅ Persistence across restarts

### Integration Testing
To test cross-device sync manually:

1. **Start the API**:
   ```bash
   cd scenarios/web-console
   make run
   ```

2. **Open Device A**:
   - Navigate to `http://localhost:<port>/`
   - Create 2-3 tabs
   - Rename one tab to "Device A Test"
   - Start a terminal session

3. **Open Device B** (different browser/incognito):
   - Navigate to same URL
   - Verify all tabs from Device A appear
   - Create a new tab
   - Switch to Device A and verify new tab appeared

4. **Test Session Persistence**:
   - On Device A, start a command (e.g., `ls -la`)
   - Refresh Device A's browser
   - Verify terminal reconnects and shows previous output

5. **Test Real-Time Updates**:
   - On Device A, rename a tab
   - On Device B, verify rename appears immediately
   - On Device B, close a tab
   - On Device A, verify tab disappears immediately

## Migration Notes

### Backward Compatibility
- Workspace file created on first run if it doesn't exist
- If workspace is empty, creates initial tab automatically
- Existing sessions not linked to tabs continue working normally

### Performance
- Workspace updates write to disk synchronously
- For high-frequency updates, consider batching or debouncing
- WebSocket broadcasts are fire-and-forget (slow clients drop events)

## Future Enhancements

### Potential Improvements
1. **User-scoped workspaces**: Add authentication and per-user workspace isolation
2. **Tab reordering**: Drag-and-drop tab reordering synced across devices
3. **Workspace templates**: Save/restore named workspace configurations
4. **Tab groups**: Organize tabs into collapsible groups
5. **Session history**: Persist terminal transcripts and replay on reconnect
6. **Collaborative features**: Show which user is viewing which tab
7. **Workspace versioning**: Track workspace history for undo/redo

### Known Limitations
1. **No conflict resolution**: Last write wins (acceptable for single user)
2. **No offline support**: Requires constant server connection
3. **No tab ordering sync**: Tab order may drift between clients
4. **No session transcript persistence**: Terminal output lost on session restart

## API Reference

### Workspace Object
```typescript
interface Workspace {
  activeTabId: string
  version: number
  tabs: TabMeta[]
}

interface TabMeta {
  id: string
  label: string
  colorId: string
  order: number
  sessionId?: string
}
```

### REST Endpoints

#### GET /api/v1/workspace
Returns current workspace state.

**Response**: `200 OK`
```json
{
  "activeTabId": "tab-123",
  "version": 42,
  "tabs": [
    {
      "id": "tab-123",
      "label": "Terminal 1",
      "colorId": "sky",
      "order": 0,
      "sessionId": "session-abc"
    }
  ]
}
```

#### PUT /api/v1/workspace
Replace entire workspace state.

**Request Body**:
```json
{
  "activeTabId": "tab-456",
  "tabs": [...]
}
```

**Response**: `200 OK`
```json
{"status": "updated"}
```

#### POST /api/v1/workspace/tabs
Create a new tab.

**Request Body**:
```json
{
  "id": "tab-789",
  "label": "New Tab",
  "colorId": "emerald",
  "order": 2
}
```

**Response**: `201 Created`

#### PATCH /api/v1/workspace/tabs/{id}
Update tab label and color.

**Request Body**:
```json
{
  "label": "Updated Label",
  "colorId": "violet"
}
```

**Response**: `200 OK`

#### DELETE /api/v1/workspace/tabs/{id}
Remove a tab.

**Response**: `200 OK`
```json
{"status": "deleted"}
```

### WebSocket Events

#### Connection: ws://host/api/v1/workspace/stream

**Initial State** (sent on connect):
```json
{
  "activeTabId": "tab-123",
  "version": 42,
  "tabs": [...]
}
```

**Update Events**:
```json
{
  "type": "tab-added",
  "timestamp": "2025-10-01T12:34:56Z",
  "payload": {
    "id": "tab-456",
    "label": "Terminal 2",
    "colorId": "amber",
    "order": 1
  }
}
```

## Troubleshooting

### Tabs not syncing
- Check WebSocket connection in browser DevTools (Network → WS)
- Verify server logs for WebSocket upgrade errors
- Ensure proxy (if any) supports WebSocket upgrades

### Workspace file corruption
- Delete `data/sessions/workspace.json` to reset
- Server will create fresh workspace on next request

### High memory usage
- Limit number of WebSocket subscribers (current: unlimited)
- Consider adding subscriber throttling for slow clients

## Conclusion

The workspace synchronization feature transforms web-console from a single-device terminal into a seamless multi-device experience. Users can now:

- Start a session on their laptop
- Continue on their tablet during a meeting
- Check status on their phone
- Return to their desktop and see all tabs exactly as they left them

All without any manual configuration or state transfer—it just works.
