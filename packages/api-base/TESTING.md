# Testing Guide for api-base

Comprehensive guide for running and understanding tests in `@vrooli/api-base`.

## Test Structure

```
src/__tests__/
├── client/               # Client-side resolution tests
│   ├── resolve.test.ts   # API/WS URL resolution
│   ├── detect.test.ts    # Proxy context detection
│   ├── url.test.ts       # URL building utilities
│   └── config.test.ts    # Runtime configuration
├── server/               # Server-side tests
│   ├── proxy.test.ts     # HTTP & WebSocket proxy
│   ├── health.test.ts    # Health endpoint
│   ├── config.test.ts    # Config endpoint
│   ├── inject.test.ts    # Metadata injection
│   └── template.test.ts  # Server template
└── integration/          # Integration tests
    └── websocket-proxy.test.ts  # End-to-end WebSocket tests
```

---

## Running Tests

### All Tests

```bash
pnpm test
```

### Watch Mode (Development)

```bash
pnpm test:watch
```

### With Coverage

```bash
pnpm test:coverage
```

Coverage reports will be generated in `./coverage/` directory.

### Specific Test Files

```bash
# Run only unit tests
pnpm test src/__tests__/client
pnpm test src/__tests__/server

# Run only integration tests
pnpm test src/__tests__/integration

# Run specific file
pnpm test src/__tests__/integration/websocket-proxy.test.ts
```

---

## Test Types

### 1. Unit Tests

**Purpose**: Test individual functions in isolation using mocks.

**Characteristics**:
- Fast (milliseconds)
- Use mocked dependencies
- Test logic, not network behavior
- ~150 tests total

**Example**:
```typescript
it('resolves WebSocket base URL', () => {
  const result = resolveWsBase({ appendSuffix: true })
  expect(result).toContain('ws://')
})
```

**Limitations**:
- Won't catch integration bugs (like the WebSocket header filtering bug we fixed)
- Mocks may not match real behavior
- Can't test actual network communication

---

### 2. Integration Tests

**Purpose**: Test complete flows with real servers and network connections.

**Characteristics**:
- Slower (30-100ms per test)
- Use real WebSocket servers and clients
- Test actual network behavior
- 8 tests for WebSocket proxy

**Example**:
```typescript
it('establishes WebSocket connection through proxy', async () => {
  const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`
  const ws = new WebSocket(wsUrl)

  await new Promise((resolve) => {
    ws.on('open', resolve)
  })

  expect(ws.readyState).toBe(WebSocket.OPEN)
  ws.close()
})
```

**What They Test**:
1. ✅ Connection establishment through proxy
2. ✅ Header forwarding (validates the fix for header filtering bug)
3. ✅ Bidirectional data flow (send/receive messages)
4. ✅ Multiple concurrent connections
5. ✅ Large message handling (64KB+)
6. ✅ Connection close and cleanup
7. ✅ Query parameters in WebSocket URLs
8. ✅ Sec-WebSocket-Version header preservation (regression test)

**Why They Matter**:
- Would have caught the WebSocket header filtering bug
- Test real network behavior
- Prevent regressions
- Validate fixes actually work

---

## Integration Test Architecture

### Test Setup

```typescript
beforeAll(async () => {
  // 1. Find available ports
  uiPort = await findAvailablePort(30000)
  apiPort = await findAvailablePort(30100)

  // 2. Create API server with WebSocket endpoint
  apiServer = http.createServer(/* ... */)
  wsServer = new WebSocket.Server({ noServer: true })

  // 3. Create UI server with proxy
  const app = createScenarioServer({
    uiPort,
    apiPort,
    /* ... */
  })
  uiServer = app.listen(uiPort)

  // 4. Setup WebSocket upgrade handler
  uiServer.on('upgrade', (req, socket, head) => {
    proxyWebSocketUpgrade(req, socket, head, { apiPort })
  })
})
```

### Test Flow

```
Test → UI Server → Proxy → API Server → WebSocket Server
         ↓
    WebSocket Upgrade
         ↓
   Bidirectional Pipe
         ↓
    Client ← → Server
```

### Cleanup

```typescript
afterAll(async () => {
  // Close all servers in reverse order
  await new Promise((resolve) => {
    wsServer.close(() => {
      apiServer.close(() => {
        uiServer.close(resolve)
      })
    })
  })
})
```

---

## Coverage Requirements

Current thresholds (defined in `vitest.config.ts`):
- **Lines**: 90%
- **Functions**: 90%
- **Branches**: 85%
- **Statements**: 90%

### Viewing Coverage

```bash
pnpm test:coverage
```

Open `coverage/index.html` in browser to see detailed report.

---

## Writing New Tests

### Unit Test Template

```typescript
import { describe, it, expect } from 'vitest'
import { functionToTest } from '../yourModule.js'

describe('functionToTest', () => {
  it('does what it should', () => {
    const result = functionToTest('input')
    expect(result).toBe('expected')
  })

  it('handles edge case', () => {
    const result = functionToTest('')
    expect(result).toBeNull()
  })
})
```

### Integration Test Template

```typescript
import { describe, it, expect, beforeAll, afterAll } from 'vitest'

describe('My Integration Test', () => {
  let server: Server

  beforeAll(async () => {
    // Setup real servers
    server = /* ... */
  })

  afterAll(async () => {
    // Cleanup
    await server.close()
  })

  it('tests real behavior', async () => {
    // Make real network requests
    const response = await fetch(/* ... */)
    expect(response.ok).toBe(true)
  })
})
```

---

## Debugging Tests

### 1. Run Single Test

```bash
# Run one test by name
pnpm test -t "establishes WebSocket connection"
```

### 2. Enable Verbose Output

In test file:
```typescript
proxyWebSocketUpgrade(req, socket, head, {
  apiPort,
  verbose: true  // See all proxy logs
})
```

### 3. Add Console Logs

```typescript
it('my test', async () => {
  console.log('Testing with port:', uiPort)
  const result = await doSomething()
  console.log('Result:', result)
  expect(result).toBe('expected')
})
```

### 4. Increase Timeout

```typescript
it('slow test', async () => {
  // ... test code
}, 30000) // 30 second timeout
```

---

## Common Test Failures

### "Port already in use"

**Cause**: Previous test run didn't clean up properly.

**Solution**:
```bash
# Kill any processes using the ports
lsof -ti:30000,30100 | xargs kill -9

# Or use different port range
```

### "Connection timeout"

**Cause**: Server not starting, or too slow.

**Solution**:
- Check if ports are available
- Increase timeout in test
- Enable verbose logging to see what's happening

### "WebSocket connection failed"

**Cause**: Missing upgrade handler or wrong configuration.

**Solution**:
- Verify upgrade handler is attached
- Check logs with verbose mode
- Ensure API server is running

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: pnpm/action-setup@v2
      - uses: actions/setup-node@v3
        with:
          node-version: 20
          cache: 'pnpm'

      - run: pnpm install
      - run: pnpm test:coverage

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage/coverage-final.json
```

---

## Test Dependencies

Required packages (already in package.json):
```json
{
  "devDependencies": {
    "@types/ws": "^8.5.10",      // TypeScript types for ws
    "ws": "^8.16.0",              // WebSocket client/server
    "vitest": "^1.0.4",           // Test runner
    "happy-dom": "^20.0.10",      // DOM environment for tests
    "@vitest/coverage-v8": "^1.0.4"  // Coverage reporter
  }
}
```

### Installing Dependencies

```bash
# Install all dependencies
pnpm install

# Verify ws package is installed
pnpm list ws
```

---

## Performance Benchmarks

Typical test times on modern hardware:

| Test Type | Count | Time | Per Test |
|-----------|-------|------|----------|
| Unit Tests | 150+ | ~1s | ~6ms |
| Integration Tests | 8 | ~100ms | ~12ms |
| **Total** | **158+** | **~1.1s** | **~7ms** |

---

## Continuous Improvement

### When to Add Tests

1. **Bug Fixes**: Always add a test that would have caught the bug
2. **New Features**: Test happy path + edge cases
3. **Refactoring**: Ensure behavior doesn't change
4. **Integration Points**: Test cross-module interactions

### Test Quality Checklist

- [ ] Tests are independent (can run in any order)
- [ ] Tests clean up after themselves
- [ ] Test names clearly describe what they test
- [ ] Both happy path and error cases covered
- [ ] Edge cases considered (empty input, null, undefined, large data)
- [ ] Tests are fast (< 100ms for unit, < 1s for integration)
- [ ] No hardcoded timeouts (use proper async/await)
- [ ] No flaky tests (random failures)

---

## Troubleshooting

### Tests Pass Locally but Fail in CI

**Possible causes**:
- Different Node.js version
- Different pnpm version
- Port conflicts
- Timing issues (CI is slower)

**Solutions**:
- Match Node.js versions
- Add retries for flaky tests
- Use dynamic port allocation
- Increase timeouts in CI

### Integration Tests Are Slow

**Solutions**:
- Run in parallel: `pnpm test --threads`
- Use smaller timeouts where possible
- Consider mocking for non-critical paths

### Coverage Dropping

**Solutions**:
- Check coverage report: `pnpm test:coverage`
- Add tests for uncovered lines
- Review if coverage thresholds are appropriate

---

## Example: Adding a New Integration Test

Let's say you want to test WebSocket reconnection:

```typescript
it('reconnects after connection drop', async () => {
  const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`
  let reconnected = false

  // First connection
  const ws1 = new WebSocket(wsUrl)
  await new Promise((resolve) => ws1.on('open', resolve))

  // Force close from server side
  // (In real test, you'd trigger this from the server)
  ws1.close()
  await new Promise((resolve) => ws1.on('close', resolve))

  // Second connection (reconnect)
  const ws2 = new WebSocket(wsUrl)
  await new Promise((resolve) => {
    ws2.on('open', () => {
      reconnected = true
      resolve()
    })
  })

  expect(reconnected).toBe(true)
  ws2.close()
})
```

---

## Summary

- ✅ **Unit tests** cover individual functions (fast, isolated)
- ✅ **Integration tests** cover complete flows (real network, prevents regressions)
- ✅ **Coverage requirements** ensure quality (90% lines/functions)
- ✅ **Easy to run** with `pnpm test`
- ✅ **Well documented** with this guide

The integration tests we added would have caught the WebSocket header filtering bug that was discovered during app-issue-tracker integration. This demonstrates their value and importance.
