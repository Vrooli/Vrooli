# WebSocket Proxy Improvements

Summary of improvements made to `@vrooli/api-base` WebSocket support based on real-world bug discovery and testing.

## Background

While integrating `@vrooli/api-base` into the `app-issue-tracker` scenario, we discovered a critical bug where WebSocket connections were failing with:

```
websocket: unsupported version: 13 not found in 'Sec-Websocket-Version' header
```

**Root Cause**: The `proxyWebSocketUpgrade` function was incorrectly filtering out WebSocket-specific headers (`Sec-WebSocket-Version`, `Sec-WebSocket-Key`, etc.) because they were listed as "hop-by-hop" headers. While these headers ARE hop-by-hop for regular HTTP requests, they MUST be forwarded during WebSocket upgrade handshakes.

---

## Improvements Made

### 1. Fixed WebSocket Header Filtering (CRITICAL BUG FIX)

**Files Changed**:
- `src/shared/constants.ts`
- `src/server/proxy.ts`

**Changes**:
```typescript
// Before: WebSocket headers were filtered out
export const HOP_BY_HOP_HEADERS = new Set([
  'connection',
  'keep-alive',
  // ...
  'sec-websocket-key',        // ❌ WRONG - Filtered out
  'sec-websocket-version',    // ❌ WRONG - Filtered out
  // ...
])

// After: Separated WebSocket headers
export const HOP_BY_HOP_HEADERS = new Set([
  'connection',
  'keep-alive',
  // ... (no WebSocket headers)
])

export const WEBSOCKET_HEADERS = new Set([
  'sec-websocket-key',
  'sec-websocket-accept',
  'sec-websocket-version',    // ✅ MUST be forwarded
  'sec-websocket-protocol',
  'sec-websocket-extensions',
])
```

**Impact**: WebSocket connections now work through the proxy. Without this fix, ALL WebSocket connections through api-base would fail.

---

### 2. Enhanced Error Handling and Logging

**File**: `src/server/proxy.ts`

**Before**:
```javascript
if (!port) {
  if (verbose) {
    console.error('[ws-proxy] Invalid API_PORT, closing connection')
  }
  clientSocket.destroy()
  return
}
```

**After**:
```javascript
if (!port) {
  const errorMsg = `Invalid API_PORT configuration: ${apiPort}`
  console.error(`[ws-proxy] ${errorMsg}`)
  // Send proper HTTP error response before closing
  clientSocket.write(
    'HTTP/1.1 502 Bad Gateway\r\n' +
    'Content-Type: text/plain\r\n' +
    'Connection: close\r\n\r\n' +
    `${errorMsg}\r\n`
  )
  clientSocket.destroy()
  return
}
```

**New Features**:
1. **Detailed error responses**: Client receives proper HTTP 502 with explanation
2. **Header validation**: Warns if required WebSocket headers are missing
3. **Verbose logging**: Shows exactly which headers are being forwarded
4. **Connection diagnostics**: Logs connection attempts, errors, and closures

**Example Verbose Output**:
```
[ws-proxy] Upgrade request: GET /api/v1/ws -> 127.0.0.1:8080
[ws-proxy] Client headers: host, connection, upgrade, sec-websocket-key, sec-websocket-version
[ws-proxy] Connected to upstream 127.0.0.1:8080
[ws-proxy] Forwarding upgrade request with headers:
[ws-proxy]   Sec-WebSocket-Version: 13
[ws-proxy]   Sec-WebSocket-Key: [present]
[ws-proxy]   Connection: Upgrade
[ws-proxy]   Upgrade: websocket
```

**Benefits**:
- Easier debugging of connection issues
- Clear error messages for misconfiguration
- Validation prevents silent failures
- Helps users understand what's happening

---

### 3. Integration Tests

**New File**: `src/__tests__/integration/websocket-proxy.test.ts`

**Coverage**:
- ✅ WebSocket connection through proxy
- ✅ Header forwarding (validates the fix)
- ✅ Bidirectional data flow
- ✅ Multiple concurrent connections
- ✅ Large message handling (64KB+)
- ✅ Connection close/cleanup
- ✅ Query parameters in WebSocket URLs
- ✅ `Sec-WebSocket-Version` header preservation (regression test)

**Example Test**:
```typescript
it('preserves Sec-WebSocket-Version header', async () => {
  // This test verifies the fix for the bug where WebSocket headers were filtered
  const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`
  const ws = new WebSocket(wsUrl)

  const connected = await new Promise<boolean>((resolve) => {
    ws.on('open', () => {
      resolve(true)
      ws.close()
    })
    ws.on('error', () => resolve(false))
  })

  // If this passes, Sec-WebSocket-Version was properly forwarded
  expect(connected).toBe(true)
})
```

**Why This Matters**:
- Would have caught the header filtering bug
- Prevents regressions
- Tests actual network behavior, not just mocks
- Covers edge cases (concurrent connections, large messages)

---

### 4. Comprehensive Troubleshooting Guide

**File**: `docs/concepts/websocket-support.md`

**New Sections**:

1. **Connection Refused / 404 Error**
   - Missing upgrade handler
   - API server not running
   - Wrong WebSocket endpoint

2. **"unsupported version: 13" Error**
   - The exact bug we fixed
   - How to verify the fix
   - Expected verbose output

3. **Mixed Content Error**
   - ws:// vs wss:// on HTTPS
   - Automatic protocol selection

4. **Connection Works in Localhost, Fails in Production**
   - Hardcoded URLs
   - Missing TLS support
   - Reverse proxy config

5. **Connection Drops Unexpectedly**
   - Proxy timeouts
   - Heartbeat/ping-pong
   - Reconnection logic

6. **Multiple Connections Created**
   - React cleanup issues
   - useEffect dependencies

7. **Messages Not Received**
   - Event handler debugging
   - Server-side logging
   - Proxy forwarding verification

**Debugging Checklist**:
- Browser console errors
- Network tab (101 Switching Protocols)
- Server logs with verbose mode
- URL resolution
- Upgrade handler presence
- API health check
- Port configuration
- Protocol matching
- Header forwarding
- Firewall settings
- Proxy context detection

---

### 5. Complete End-to-End Example

**New File**: `docs/examples/websocket-scenario.md`

**Complete Chat Application** with:
- **Go WebSocket Server**: Hub pattern, broadcast, connection management
- **React Frontend**: useWebSocket hook, reconnection, UI
- **Server Proxy**: Upgrade handler, proper configuration
- **Vrooli Integration**: Service.json, lifecycle management

**Key Features Demonstrated**:
1. **Proper WebSocket Resolution**:
   ```typescript
   const wsUrl = resolveWsBase({
     appendSuffix: true,
     apiSuffix: '/api/v1/ws'
   })
   ```

2. **Correct Upgrade Handler**:
   ```javascript
   server.on('upgrade', (req, socket, head) => {
     if (req.url?.startsWith('/api')) {
       proxyWebSocketUpgrade(req, socket, head, {
         apiPort,
         verbose: true
       })
     }
   })
   ```

3. **Reconnection Logic**:
   ```typescript
   // Exponential backoff with jitter
   const backoff = Math.min(
     reconnectInterval * Math.pow(1.5, attempt - 1),
     30000
   )
   ```

4. **Proper Cleanup**:
   ```typescript
   return () => {
     ws.close()
   }
   ```

**Testing Instructions**:
- Manual testing (multiple browser tabs)
- Browser console testing
- CLI testing with wscat
- Deployment context testing (localhost, tunnel, proxy)

---

## Impact Summary

### Before Improvements

❌ **WebSocket connections failed** with "unsupported version: 13" error
❌ **No clear error messages** when configuration was wrong
❌ **No integration tests** to catch the bug
❌ **Limited troubleshooting guidance** in docs
❌ **No complete working example** for reference

### After Improvements

✅ **WebSocket connections work** through proxy in all contexts
✅ **Clear error messages** with actionable information
✅ **Integration tests** prevent regressions
✅ **Comprehensive troubleshooting** covers all common issues
✅ **Complete working example** demonstrates best practices

---

## Testing the Improvements

### 1. Run Integration Tests

```bash
cd packages/api-base
pnpm install  # Install ws package
pnpm test     # Run all tests including new integration tests
```

### 2. Test with app-issue-tracker

```bash
# Rebuild api-base
cd packages/api-base
pnpm run build

# Restart scenario
cd ../../scenarios/app-issue-tracker
make stop
make start

# Open browser and verify WebSocket connection works
# Check console - should see "Connected" status
```

### 3. Enable Verbose Logging

Edit `scenarios/app-issue-tracker/ui/server.js`:
```javascript
server.on('upgrade', (req, socket, head) => {
  if (req.url?.startsWith('/api')) {
    proxyWebSocketUpgrade(req, socket, head, {
      apiPort: process.env.API_PORT,
      verbose: true  // ← Enable this
    })
  }
})
```

Check logs:
```bash
vrooli scenario logs app-issue-tracker --step start-ui
```

Should see:
```
[ws-proxy] Upgrade request: GET /api/v1/ws -> 127.0.0.1:19751
[ws-proxy] Client headers: host, connection, upgrade, sec-websocket-key, sec-websocket-version
[ws-proxy] Connected to upstream 127.0.0.1:19751
[ws-proxy] Forwarding upgrade request with headers:
[ws-proxy]   Sec-WebSocket-Version: 13
[ws-proxy]   Sec-WebSocket-Key: [present]
[ws-proxy]   Connection: Upgrade
[ws-proxy]   Upgrade: websocket
```

---

## Lessons Learned

### 1. Unit Tests Aren't Enough

We had 156 unit tests, but none caught the WebSocket header filtering bug because:
- Unit tests use mocks, not real network connections
- WebSocket handshake is complex, requires actual socket communication
- Integration bugs need integration tests

**Solution**: Added real WebSocket integration tests with actual servers and connections.

### 2. Verbose Logging is Critical

When the bug occurred, there was no way to see what was happening inside the proxy. Users saw:
- Browser: "Connection failed"
- API logs: "unsupported version"
- No way to see that headers were missing

**Solution**: Added detailed verbose logging that shows exactly which headers are being forwarded.

### 3. Documentation Needs Real Examples

The docs showed how to use individual functions, but not how to put it all together. Users needed:
- Complete working code
- All pieces connected
- Common pitfalls explained

**Solution**: Created complete end-to-end example with explanations of why each piece is needed.

### 4. Error Messages Should Be Actionable

Before: "Invalid API_PORT" - What does that mean? What should I do?

After:
- "Invalid API_PORT configuration: undefined"
- Troubleshooting guide explains how to check environment variables
- Shows where to add API_PORT in service.json
- Provides example of correct configuration

---

## Future Improvements

Based on this experience, here are potential future improvements:

1. **Type-Level Separation**:
   - Create separate types for HTTP proxy and WebSocket proxy
   - Prevent mixing of HTTP-only options with WebSocket calls

2. **Health Checks for WebSocket**:
   - Add WebSocket connection testing to health endpoint
   - Verify upgrade handler is working

3. **Proxy Analytics**:
   - Track connection counts
   - Monitor message throughput
   - Detect connection failures

4. **Better Testing Utilities**:
   - Provide test helpers for WebSocket scenarios
   - Mock WebSocket server for unit testing
   - Connection simulation utilities

5. **Performance Monitoring**:
   - Measure proxy latency
   - Track message queue depths
   - Alert on anomalies

---

## Conclusion

This improvement cycle demonstrates the value of:
- Real-world testing (found the bug)
- Comprehensive error handling (easier debugging)
- Integration testing (prevents regressions)
- Detailed documentation (helps users avoid issues)
- Complete examples (reference implementation)

The WebSocket support in api-base is now **production-ready** with proper error handling, testing, and documentation.
