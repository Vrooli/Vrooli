# Host Scenario E2E Tests

Comprehensive end-to-end tests for the host scenario pattern using Playwright.

## Overview

These tests validate the complete host scenario pattern (e.g., app-monitor embedding other scenarios). They use a real browser (Chromium via Playwright) to catch issues that HTTP-based tests cannot detect.

## Test Organization

Tests are organized by concern for better maintainability:

```
host-scenario/
├── setup.ts                      # Shared test utilities and server setup
├── host-pages.test.ts           # Host's own pages and assets
├── child-proxy.test.ts          # Child scenario through proxy
├── base-tag-injection.test.ts   # Base tag correctness
├── isolation.test.ts            # No interference between host/child
├── websockets.test.ts           # WebSocket functionality
├── network-monitoring.test.ts   # Network requests and console
└── asset-detection.test.ts      # Asset detection and SPA fallback
```

## What Each Test File Covers

### setup.ts
**Purpose**: Shared test infrastructure

**Provides**:
- `setupTestEnvironment()`: Creates browser, servers, and test context
- `cleanupTestEnvironment()`: Tears down test environment
- `findAvailablePort()`: Finds available ports for test servers
- `setupChildScenario()`: Creates child scenario server
- `setupHostScenario()`: Creates host scenario server
- `makeRequest()`: Helper for HTTP requests

**Usage**: Import utilities in test files:
```typescript
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'
```

### host-pages.test.ts
**Purpose**: Verify host serves its own content correctly

**Tests**:
- ✓ Host page loads without errors
- ✓ Host CSS loads correctly
- ✓ Host JavaScript executes
- ✓ Missing assets return 404 (not HTML)
- ✓ No CORS errors
- ✓ Correct Content-Type headers

**Why Important**: Ensures host's own content works before testing proxy functionality.

### child-proxy.test.ts
**Purpose**: Verify child scenarios load through proxy/iframe

**Tests**:
- ✓ Child iframe loads with correct content
- ✓ Child assets load through proxy
- ✓ Child CSS loads through proxy
- ✓ Child JavaScript executes in iframe
- ✓ Proxy metadata injected in child
- ✓ Child can be accessed directly (not just through proxy)
- ✓ Proxy paths with trailing slashes work

**Why Important**: Core functionality of host scenario pattern.

### base-tag-injection.test.ts
**Purpose**: Verify base tags are correctly injected

**Tests**:
- ✓ Host has `<base href="/">`
- ✓ Child has `<base href="/apps/{id}/proxy/">`
- ✓ Host and child have different base tags
- ✓ Data attributes present for debugging
- ✓ Base tag injected early (before other tags)
- ✓ Trailing slash in href
- ✓ Only one base tag per document
- ✓ Relative URLs resolve correctly

**Why Important**: Base tags are critical for asset resolution in nested paths.

### isolation.test.ts
**Purpose**: Verify host and child don't interfere with each other

**Tests**:
- ✓ Separate document contexts (different titles)
- ✓ Separate global window objects
- ✓ Separate base paths
- ✓ Proxy metadata only in child (not host)
- ✓ Separate CSS files
- ✓ Separate JavaScript files
- ✓ Styles don't leak between host and child
- ✓ DOM queries isolated

**Why Important**: Ensures iframe isolation works correctly.

### websockets.test.ts
**Purpose**: Verify WebSocket functionality for host and child

**Tests**:
- ✓ Host WebSocket connections work
- ✓ Child WebSocket connections work
- ✓ Simultaneous connections without interference
- ✓ Multiple messages on same connection
- ✓ Graceful connection close
- ✓ Large messages (64KB)
- ✓ Multiple concurrent connections
- ✓ Message order preserved

**Why Important**: WebSockets are complex and require special handling through proxy.

### network-monitoring.test.ts
**Purpose**: Comprehensive network and console monitoring

**Tests**:
- ✓ Track all network requests
- ✓ Identify failed requests
- ✓ Collect console messages (host + iframe)
- ✓ Detect JavaScript errors
- ✓ Verify resource types
- ✓ Measure page load performance
- ✓ Detect mixed content warnings
- ✓ Detect CSP violations
- ✓ Verify response headers
- ✓ Track request timings

**Why Important**: Catches subtle issues like slow requests, console warnings, or security problems.

### asset-detection.test.ts
**Purpose**: Verify asset detection and SPA fallback

**Tests**:
- ✓ 404 for missing .js/.css/.json/.png files (not HTML)
- ✓ 404 for Vite HMR paths
- ✓ 404 for /assets/ and /src/ paths
- ✓ Handle query strings and hash fragments
- ✓ Existing assets load correctly
- ✓ Browser doesn't receive HTML for JS requests
- ✓ No "Unexpected token <" errors (HTML-as-JS bug)
- ✓ Common file extensions detected
- ✓ Asset prefixes detected
- ✓ Case-insensitive extensions
- ✓ Child assets use correct detection

**Why Important**: The #1 source of bugs in host scenarios is SPA fallback returning HTML for asset requests.

## Running Tests

### Run all host scenario tests
```bash
cd packages/api-base
pnpm test e2e/host-scenario
```

### Run specific test file
```bash
pnpm test e2e/host-scenario/host-pages.test.ts
pnpm test e2e/host-scenario/websockets.test.ts
```

### Run in watch mode
```bash
pnpm test:watch e2e/host-scenario
```

### Run with browser visible (debugging)
Edit test file and change:
```typescript
browser = await chromium.launch({
  headless: false, // Set to false to see browser
})
```

## Test Performance

Each test file runs independently with its own setup/teardown:
- **Setup time**: ~2-5 seconds (launch browser, start servers)
- **Per test**: ~1-3 seconds
- **Total per file**: ~30-60 seconds

Running all tests in parallel:
```bash
pnpm test e2e/host-scenario
# Expected: ~2-3 minutes for all tests
```

## Debugging Tests

### View browser during test
Set `headless: false` in test file

### Add delays for inspection
```typescript
await ctx.page.waitForTimeout(10000) // Wait 10 seconds
```

### Check console messages
All tests capture console output. Failed tests will show:
- Network requests
- Console errors
- Failed requests

### Screenshot on failure
Add to test:
```typescript
await ctx.page.screenshot({ path: 'test-failure.png' })
```

## Common Issues

### Port conflicts
If tests fail with "EADDRINUSE", another process is using test ports (45000-45400).

**Solution**:
```bash
# Find and kill processes
lsof -i :45000-45400
kill -9 <PID>
```

### Timeout errors
Tests timeout after 30 seconds by default.

**Solution**: Increase timeout in test:
```typescript
it('slow test', async () => {
  // test code
}, 60000) // 60 second timeout
```

### Browser not found
Playwright needs browser binaries installed.

**Solution**:
```bash
pnpm exec playwright install chromium
```

## Best Practices

### Writing new tests
1. Use `setupTestEnvironment()` in `beforeAll()`
2. Use `cleanupTestEnvironment()` in `afterAll()`
3. Set reasonable timeouts (30s default, 60s for complex tests)
4. Clean up listeners (`ctx.page.removeAllListeners()`)
5. Use data-testid attributes for reliable selectors

### Test organization
- Keep test files focused (single concern)
- Share setup code in `setup.ts`
- Use descriptive test names
- Group related tests with `describe()`

### Debugging
- Start with browser visible (`headless: false`)
- Add strategic `waitForTimeout()` calls
- Check console output first
- Use network monitoring to find failed requests

## Migration from Old Tests

The old monolithic `host-scenario-browser.test.ts` (743 lines) has been split into focused files.

**Mapping**:
- Lines 450-530 → `host-pages.test.ts`
- Lines 532-625 → `child-proxy.test.ts`
- Lines 484-624 → `base-tag-injection.test.ts`
- Lines 698-783 → `isolation.test.ts`
- (WebSockets were added as new comprehensive tests)
- Lines 626-741 → `network-monitoring.test.ts`
- (Asset detection was added as new comprehensive tests)

**Benefits**:
- ✓ Easier to maintain (focused files)
- ✓ Faster test runs (parallel execution)
- ✓ Better organization (clear separation of concerns)
- ✓ More comprehensive (added missing scenarios)

## Contributing

When adding new tests:
1. Determine which file the test belongs in
2. If it doesn't fit any existing file, create a new focused file
3. Update this README with the new test file
4. Ensure all tests pass before committing
