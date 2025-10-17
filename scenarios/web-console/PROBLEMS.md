# Web Console - Problems & Solutions

## 2025-10-04: CDN Dependencies Causing 502 Bad Gateway Errors (FIXED)

**Problem**: UI was completely broken, displaying Cloudflare 502 Bad Gateway error pages instead of the terminal interface. The error message started with "itsagitime.com | 502: Bad gateway" and showed HTML instead of a functional UI.

**User Report**: "I'm seeing an error message in the UI that starts like this: '<!DOCTYPE html> <!--[if It IE 7]> <html class="no-js ie6 oldie"...' First of all, this error isn't very useful. It should be in english! But also, there ideally shouldn't be an error at all!"

**Root Cause**: The UI was loading vendor libraries from external CDNs:
- `https://cdn.jsdelivr.net/npm/xterm@5.3.0/` - Terminal emulator
- `https://unpkg.com/lucide@latest` - Icons library
- `https://cdn.jsdelivr.net/npm/html2canvas@1.4.1/` - Screenshot capture

When these CDN requests failed (due to network restrictions, CDN outages, or Cloudflare blocking), the entire UI broke with cryptic HTML error pages instead of user-friendly error messages.

> **Implementation note:** After migrating to Vite, the frontend entry lives at `ui/src/main.js` and vendor bundles are stored in `ui/public/lib/`.

**Solution**: Moved to offline-first architecture by downloading all vendor dependencies locally:

1. **Created vendor directory**: `ui/public/lib/`
2. **Downloaded dependencies locally** (~1.1MB total):
   - xterm@5.3.0 (JS + CSS)
   - xterm-addon-fit@0.7.0
   - lucide icons (547KB UMD bundle)
   - html2canvas@1.4.1
3. **Updated index.html**: Changed all CDN URLs to local paths
   - Before: `<script src="https://cdn.jsdelivr.net/npm/xterm@5.3.0/lib/xterm.min.js"></script>`
   - After: `<script src="/lib/xterm.min.js"></script>`

**Impact**:
- ✅ Console works reliably in offline/restricted network environments
- ✅ No external dependencies - parent scenarios won't be blocked by CDN access issues
- ✅ Faster initial load (local files vs CDN)
- ✅ Predictable behavior - no surprise failures from external services
- ✅ Better privacy - no external requests tracking users

**Verification**: Screenshot at /tmp/web-console-ui.png shows fully functional UI with all lucide icons rendering, terminal active, no console errors.

**Lessons Learned**:
- External dependencies create single points of failure
- Error messages should be user-friendly, not raw HTML
- Infrastructure tools should be resilient to network issues
- "Lightweight for iframe embedding" means truly self-contained

## 2025-10-04: Terminal Canvas Not Refreshing After Visibility Change - Interactive Apps Fix (FIXED)

**Problem**: Even after previous fixes for terminal refresh, interactive applications (like Claude Code CLI waiting for authentication) would still show a blank terminal when returning from another tab. The terminal would only appear after typing something.

**User's Critical Insight**: "I'm thinking maybe it has something to do with interactive terminal apps specifically? When I use Claude Code, it asks me to open a link to get a sign-in code. When I come back the terminal is blank again."

**Root Cause Analysis**:
The previous fix added basic terminal refresh on visibility change, but it had timing and rendering pipeline issues:

1. **Timing Issue**: The canvas refresh happened too quickly, before the browser fully restored the page's rendering context
2. **Missing Focus**: The terminal wasn't being focused, which is critical for xterm.js's render pipeline
3. **Render Pipeline Not Triggered**: Simply calling `refresh()` doesn't always trigger the internal render cycle if the terminal is in a paused state
4. **Interactive Apps**: When an app like Claude Code is waiting for input, there's no new output to trigger a render - the terminal is just showing a static prompt

**Enhanced Solution**:

1. **Visibility Change Handler with Proper Sequencing** (ui/src/main.js:368-421):
   ```javascript
   // Wait 100ms for browser to fully restore canvas
   setTimeout(() => {
     // 1. Focus terminal FIRST - critical for render pipeline
     activeTab.term.focus()

     // 2. Fit to container
     activeTab.fitAddon?.fit()

     // 3. Force render pipeline by writing empty string
     activeTab.term.write('')

     // 4. Refresh entire display
     activeTab.term.refresh(0, activeTab.term.rows - 1)
   }, 100)
   ```

2. **Enhanced Tab Focus Handler** (ui/src/main.js:699-717):
   - Added `term.write('')` before refresh to trigger render pipeline
   - Ensures content is visible when switching between tabs

3. **Replay Completion Refresh** (ui/src/main.js:1779-1815):
   - Force terminal refresh after replay completes
   - Ensures reconnected sessions show content immediately
   - Uses same focus → write → refresh sequence

**Why This Works**:

1. **100ms Delay**: Gives browser time to fully restore the page's rendering context and canvas
2. **Terminal Focus**: xterm.js requires focus to activate its render pipeline - without this, the terminal won't update
3. **Empty Write**: `term.write('')` triggers the internal render cycle even when there's no new data
4. **Full Refresh**: `term.refresh(0, rows - 1)` redraws all rows from the buffer to the canvas

**Verification**:
```bash
# Start the scenario
make start

# Test with interactive app (Claude Code):
# 1. Open web console (http://localhost:36233)
# 2. Run `claude` or another interactive CLI
# 3. Wait for authentication prompt with link
# 4. Switch to another tab/app to copy code
# 5. Return to web console
# Expected: Terminal immediately shows the waiting prompt, no blank screen
```

**Impact**:
- ✅ Interactive applications work perfectly (Claude Code, vim, etc)
- ✅ Terminal content visible immediately after 100ms
- ✅ No user interaction required to restore display
- ✅ Works across tab switches, app switches, and screen locks
- ✅ Preserves all session state and terminal buffer
- ✅ Minimal performance impact (single 100ms delay)

**Lessons Learned**:
1. Canvas rendering requires time to restore after page backgrounding - don't rush it
2. Terminal focus is CRITICAL for xterm.js render pipeline - never skip this step
3. Writing empty string forces internal render cycle that refresh() alone doesn't trigger
4. Interactive apps are the hardest test case - they expose timing issues that normal shells hide
5. Proper sequencing matters: focus → fit → write → refresh

## 2025-10-04: Blank Terminal on Tab Switch (FIXED - PREVIOUS ATTEMPT)

**Problem**: When switching away from and back to the web-console tab, the terminal would appear completely blank - no prompt, no previous output, nothing. Only after typing would the terminal come back to life with a new prompt.

**User's Insight**: "The terminal is completely blank - which may be a clue. If it was a new session, I'd see my username and path with the blinking cursor. But I don't see that. So maybe the root cause is more nuanced."

**Root Cause**: The terminal replay logic unconditionally called `term.reset()` on every WebSocket reconnection (including when returning to a tab). This cleared ALL terminal content, including the shell prompt. When the replay buffer was empty or didn't contain the initial prompt (which is often the case since the prompt is written during session startup before the buffer starts recording), the result was a completely blank terminal. Typing would trigger new output (echo + prompt), which would fill the terminal again.

**Why This Wasn't a Session Issue**: The session was still running and the WebSocket was reconnecting properly. The problem was purely visual - the terminal display was being reset without content to replace it.

**Solution**: Introduced `hasEverConnected` flag to distinguish first-time connections from reconnections:
1. **First Connection**: Terminal is reset and replay buffer populates it (normal startup flow)
2. **Reconnection**: Existing terminal state is preserved, replay buffer is written additively on top

This prevents the blank screen while maintaining the ability to sync with actual session state when needed.

**Changes**:
- Added `hasEverConnected: false` flag to tab state (ui/src/main.js:507)
- Modified `handleReplayPayload` to only reset terminal on first connection with non-empty buffer (app.js:1698-1708)
- Set `hasEverConnected = true` after successful replay completion (app.js:1734)

**Verification**:
```bash
# Start the scenario
make start

# Test in browser:
# 1. Open web console (http://localhost:36233)
# 2. Note the shell prompt is visible
# 3. Run a command like `ls` or `pwd`
# 4. Switch to another browser tab
# 5. Wait a few seconds, then return to web-console tab
# Expected: Terminal still shows prompt and previous output, no blank screen
```

**Impact**:
- ✅ Terminal content persists across tab switches
- ✅ No more blank screen when returning to tab
- ✅ Reconnection is seamless and non-disruptive
- ✅ Terminal state matches session state properly

**Lessons Learned**:
1. Terminal display state and session state are separate concerns - clearing one doesn't require clearing the other
2. The initial shell prompt may not be in the replay buffer if written before buffer started recording
3. Preserving client-side visual state on reconnection provides better UX than aggressive resets
4. Sometimes the absence of expected content (blank screen vs missing prompt) is the key debugging clue

## 2025-10-04: Session Reconnection and Persistence Issues

**Problem**: When switching browser tabs and returning to the web console, the terminal often had trouble reconnecting. Sometimes it wouldn't show any terminal output until the user typed something, which would reset the terminal session.

**Root Causes**:
1. **No Automatic Reconnection**: When a WebSocket disconnected (e.g., due to browser tab becoming inactive), the client didn't attempt to reconnect to the still-running backend session
2. **Limited Output Buffer**: Backend only kept last 100 chunks (~100KB) for replay on reconnect, causing output loss on longer sessions
3. **No Keepalive**: WebSocket connections could silently die when tabs were inactive without detection
4. **Session Marked as Closed**: On WebSocket disconnect, sessions were immediately marked as 'closed' even if the backend PTY was still running
5. **No Visibility Detection**: No handling for browser tab visibility changes to trigger reconnection

**Solutions Implemented**:

### Frontend (ui/src/main.js)
1. **Smart Reconnection Logic** (lines 1567-1609):
   - Distinguish between clean closes (codes 1000/1001) and unexpected disconnects
   - Automatically attempt reconnection after 1 second for unexpected disconnects
   - Only mark session as 'closed' for clean shutdowns or explicit closure requests

2. **Heartbeat Mechanism** (lines 1522-1535):
   - Send WebSocket heartbeat every 30 seconds to keep connection alive
   - Clear heartbeat interval on disconnect to prevent leaks
   - Properly clean up on tab destruction

3. **Visibility Change Detection** (lines 368-382):
   - Listen for `visibilitychange` events
   - When tab becomes visible, check all sessions with disconnected sockets
   - Automatically reconnect to backend sessions that are still running

4. **Proper Cleanup** (lines 568-572):
   - Clear heartbeat intervals when destroying tabs
   - Prevent interval leaks and memory issues

### Backend (api/session.go)
1. **Increased Buffer Size** (lines 102-106):
   - Increased from 100 to 500 chunks for better reconnect experience
   - Added 1MB byte limit to prevent excessive memory usage
   - Track both chunk count and byte size

2. **Smart Buffer Management** (lines 302-325):
   - Trim buffer based on both chunk count and byte size
   - Remove oldest chunks first when limits exceeded
   - Properly track buffer size by decoding base64 data

**Verification Steps**:
```bash
# Start the scenario
make start

# Test in browser:
# 1. Open web console
# 2. Start a long-running command (e.g., `tail -f /var/log/syslog`)
# 3. Switch to another browser tab for 30+ seconds
# 4. Return to web console tab
# Expected: Terminal reconnects automatically and shows recent output

# Monitor reconnection events in browser console
# Expected log messages:
# - "ws-close" event when tab becomes inactive
# - "ws-reconnecting" event when attempting reconnect
# - "visibility-reconnect" event when tab becomes visible
# - "output-replay-complete" event with chunk count
```

**Impact**:
- ✅ Sessions persist properly when switching tabs
- ✅ Automatic reconnection on tab visibility change
- ✅ Up to 1MB of recent output preserved for replay
- ✅ Heartbeat prevents silent connection drops
- ✅ No more terminal resets when typing after reconnect

**Lessons Learned**:
1. Browser tab lifecycle events are critical for web-based terminals
2. WebSocket keepalive is essential for long-lived connections
3. Sufficient buffering (500 chunks/1MB) provides good reconnect UX
4. Smart reconnection logic (distinguish clean vs dirty closes) prevents unnecessary session termination
5. Visibility API is key for detecting when users return to inactive tabs

# Web Console - Problems & Solutions

## 2025-10-03: Port Conflict with Ecosystem Manager

**Problem**: Web Console was not accessible on its fixed port (36233). The API was attempting to bind to the wrong port (17364) which was already in use by ecosystem-manager, causing the service to fail silently.

**Root Cause**: Environment variables `API_PORT=17364` and `UI_PORT=36110` were inherited from the ecosystem-manager shell session. The web-console lifecycle system attempted to use these incorrect port values instead of allocating its own ports per the service.json configuration.

**Solution**: Stopped and restarted the web-console scenario via `make stop` and `make start`, which properly allocated ports:
- API Port: 17085 (dynamic allocation from range 15000-19999)
- UI Port: 36233 (fixed as per service.json)

**Prevention**: The lifecycle system should isolate port allocation per scenario. This may require clearing inherited environment variables before starting each scenario, or ensuring the port registry properly scopes ports per scenario.

**Verification**:
- ✅ Health endpoint responding: `curl http://localhost:17085/healthz`
- ✅ UI accessible: `curl http://localhost:36233/`
- ✅ Ports correctly bound (verified via `lsof`)
- ✅ UI rendering correctly (screenshot captured at `/tmp/web-console-ui-20251003.png`)

**Lessons Learned**:
1. Environment variable inheritance can cause port conflicts between scenarios
2. The lifecycle system status showed "RUNNING" even when the API wasn't actually bound to any port
3. Fixed UI ports in service.json are respected, but dynamic API ports can conflict if environment variables are set globally
