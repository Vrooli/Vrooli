# Playwright Provider Capabilities Matrix

This document explains the behavioral differences between playwright implementations
and the architectural decisions made to work around them.

## Quick Reference

| Feature | rebrowser-playwright | playwright | Pure CDP |
|---------|----------------------|------------|----------|
| `page.evaluate()` context | ISOLATED | MAIN | Configurable |
| `exposeBinding()` context | ISOLATED | MAIN | N/A |
| `addInitScript()` context | MAIN | MAIN | N/A |
| History API wrapping | Only via HTML injection | Works in evaluate | Full control |
| Anti-detection | Built-in | Manual | Manual |
| Service workers | Full (headless=new) | Full | Full |
| CDP access | Full | Full | Native |
| Multi-browser | Chromium only | All browsers | Chromium only |

## Context Isolation Explained

### What is "ISOLATED" vs "MAIN" context?

When Playwright executes JavaScript in the browser, it can run in different
JavaScript "worlds" or contexts:

**MAIN Context**
- The JavaScript world where the page's own scripts run
- Can access and modify page globals (window, document)
- Can wrap native APIs (History.pushState, fetch, etc.)
- Event listeners work as expected
- **Detectable**: Page scripts can find injected code

**ISOLATED Context**
- A separate JavaScript world created by Playwright
- Cannot access MAIN world variables directly
- Cannot wrap native APIs (changes don't affect page)
- Protects automation from page interference
- **Stealth**: Page scripts cannot see automation code

### Why rebrowser-playwright Uses Isolated Context

rebrowser-playwright patches Playwright to run `page.evaluate()` in an isolated
context to prevent bot detection. Websites detect automation by:

1. **Checking for injected functions**: `window.__selenium_evaluate`, `cdc_` prefixes
2. **Detecting wrapped APIs**: Modified History.pushState, overridden fetch
3. **Finding Playwright globals**: `__playwright`, binding functions

By running automation code in an isolated context, rebrowser-playwright:
- Keeps the MAIN context pristine (no detectable modifications)
- Prevents page scripts from finding automation code
- Passes common bot detection checks

### The Problem for Recording

The recording system needs to:

1. **Capture History API navigation** (pushState, replaceState)
   - Requires wrapping these methods in MAIN context
   - If wrapped in ISOLATED, page's own calls aren't intercepted

2. **Capture user events** (clicks, types, scrolls)
   - Event listeners work in either context
   - But communication back to Node.js varies by context

3. **Send events to Node.js** (the recorded actions)
   - `exposeBinding()` only works from ISOLATED context in rebrowser
   - Need alternative communication for MAIN context scripts

## Current Architecture: Workarounds for Isolated Context

### 1. HTML Injection via Route Interception

Instead of using `page.evaluate()` or `page.addInitScript()`, we inject the
recording script directly into the HTML:

```typescript
// In context-initializer.ts
await context.route('**/*', async (route) => {
  if (request.resourceType() === 'document') {
    const response = await route.fetch();
    let body = await response.text();

    // Inject script at start of <head>
    body = body.replace('<head>', `<head><script>${initScript}</script>`);

    await route.fulfill({ response, body });
  }
});
```

**Why this works:**
- Script becomes part of the page's HTML
- Browser parses and executes it in MAIN context
- Can successfully wrap History API
- Runs before any page scripts (when in `<head>`)

### 2. Fetch-Based Event Communication

Since `exposeBinding()` doesn't work from MAIN context, we use fetch to a
special URL that's intercepted by Playwright's route handler:

```javascript
// In recording-script.js (runs in MAIN context)
function sendEvent(eventData) {
  fetch('/__vrooli_recording_event__', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(eventData),
    keepalive: true,
  });
}
```

```typescript
// In context-initializer.ts (Node.js)
await context.route('**/__vrooli_recording_event__', async (route) => {
  const postData = request.postData();
  const event = JSON.parse(postData);
  this.eventHandler(event);
  await route.fulfill({ status: 200 });
});
```

**Why this works:**
- `fetch()` works from any context
- Route interception catches the request before it leaves the browser
- Response is fulfilled immediately (no network traffic)
- `keepalive: true` ensures delivery even during page navigation

### 3. SessionStorage for Activation State

Message passing between contexts is unreliable. We use sessionStorage
(shared between contexts) for activation signaling:

```javascript
// Activation script (runs via page.evaluate in ISOLATED)
sessionStorage.setItem('__vrooli_recording_activation__', JSON.stringify({
  active: true,
  sessionId: 'xxx',
  timestamp: Date.now()
}));

// Recording script (runs in MAIN context)
// Polls or checks sessionStorage for activation state
```

## Capability Details

### `page.evaluate()` Context

| Provider | Context | Implication |
|----------|---------|-------------|
| rebrowser-playwright | ISOLATED | Can't wrap History API, can't access page globals |
| playwright | MAIN | Full access to page, but detectable |

**Code pattern:**
```typescript
if (provider.capabilities.evaluateIsolated) {
  // Use HTML injection for MAIN context execution
  await context.route('**/*', injectScript);
} else {
  // Can use evaluate directly
  await page.evaluate(script);
}
```

### `exposeBinding()` Context

| Provider | Works From | Implication |
|----------|------------|-------------|
| rebrowser-playwright | ISOLATED only | Can't call from HTML-injected scripts |
| playwright | MAIN and ISOLATED | Direct event communication works |

**Code pattern:**
```typescript
if (provider.capabilities.exposeBindingIsolated) {
  // Use fetch + route interception
  await context.route('/__event__', handleEvent);
} else {
  // Use exposeBinding directly
  await context.exposeBinding('__recordAction', handler);
}
```

### Anti-Detection

| Provider | Detection Status |
|----------|-----------------|
| rebrowser-playwright | Passes most bot detection |
| playwright | Easily detected (navigator.webdriver=true) |

**When to use each:**
- **rebrowser-playwright**: Production recording, live websites
- **playwright**: Testing, debugging, controlled environments

## Alternative Approaches

### Pure CDP (Chrome DevTools Protocol)

Browser-use migrated from Playwright to pure CDP for better control:

**Pros:**
- Full control over execution context
- No abstraction overhead
- Direct access to all browser features
- Can specify exactly which world to execute in

**Cons:**
- Chromium-only (no Firefox, WebKit)
- Low-level, verbose API
- No Playwright convenience methods
- Must implement many helpers from scratch

**Use case:** If context isolation continues to cause issues, consider
implementing recording with pure CDP via `context.newCDPSession()`.

### browser-use

A Python library for AI browser automation that moved to pure CDP:

**Relevant insight:** They encountered similar context issues with Playwright
and chose to bypass the abstraction entirely.

**Our situation:** We're invested in the Playwright ecosystem and the
route interception workaround is working. CDP remains a fallback option.

## Decision Rationale

### Why rebrowser-playwright + Workarounds?

1. **Anti-detection is essential**: Recording on production websites
   requires passing bot detection.

2. **Workarounds are working**: HTML injection and fetch-based
   communication successfully record user actions.

3. **Playwright ecosystem benefits**: Selectors, waiting, error handling,
   multi-browser (future), and community support.

4. **Migration cost of CDP**: Would require rewriting significant portions
   of the recording system.

### When to Reconsider

- If route interception causes issues (service worker conflicts)
- If bot detection evolves to detect our workarounds
- If we need Firefox/WebKit support with anti-detection
- If pure CDP libraries mature with better abstractions

## Critical: Route Handler Ordering

### route.fallback() vs route.continue()

When using multiple route handlers, understanding the difference between these
methods is critical:

**route.continue()** - Sends the request to the network
- Bypasses ALL remaining route handlers
- The request goes directly to the server
- Use when you want to let the request proceed normally

**route.fallback()** - Passes to the next matching route handler
- Allows other route handlers to process the request
- Chain of responsibility pattern
- Use when you want another handler to take over

### Route Handler Order

Routes are checked in **reverse registration order** (last registered first):

```typescript
// Registration order:
await context.route('**/__event__', handleEvent);  // First registered
await context.route('**/*', catchAll);             // Second registered

// Check order when request arrives:
// 1. catchAll('**/*') is checked first (registered last)
// 2. handleEvent('**/__event__') checked only if catchAll uses fallback()
```

### The Bug Pattern (AVOID)

```typescript
// This pattern breaks event handling:
await context.route('**/__event__', handleEvent);
await context.route('**/*', async (route) => {
  if (url.includes('__event__')) {
    await route.continue();  // BUG: Sends to network, skips event handler!
    return;
  }
  // ... handle other requests
});
```

### The Correct Pattern

```typescript
// Use fallback() to pass to the event handler:
await context.route('**/__event__', handleEvent);
await context.route('**/*', async (route) => {
  if (url.includes('__event__')) {
    await route.fallback();  // CORRECT: Passes to event handler
    return;
  }
  // ... handle other requests
});
```

## Debugging Context Issues

### Verify Script Context

```javascript
// In recording-script.js
console.log('[Recording] Context check:', {
  // This will be undefined in ISOLATED context
  hasDocumentAccess: typeof document !== 'undefined',
  // This tests if we can see page variables
  pageGlobalVisible: typeof window.somePageGlobal !== 'undefined',
});
```

### Check if History Wrapping Works

```javascript
// Test in browser console after recording starts
history.pushState({}, '', '/test-path');
// Check if navigation event was captured

// If not captured, the script is running in ISOLATED context
// and History wrapping isn't affecting the page
```

### Verify Event Communication

```javascript
// Check if events are being sent
// In browser console:
fetch('/__vrooli_recording_event__', {
  method: 'POST',
  body: JSON.stringify({ test: true }),
}).then(r => console.log('Response:', r.status));
// Should return 200 if route interception is set up
```

## Automated Diagnostics

The recording system includes built-in diagnostics to catch issues early:

### Quick Sanity Check

Enable automatic sanity check on first page load:

```typescript
const initializer = createRecordingContextInitializer({
  runSanityCheck: true,
  onSanityCheckComplete: (result) => {
    if (!result.ready) {
      console.error('Recording not ready:', result.issues);
    }
  },
});
```

### Manual Diagnostics

Run diagnostics at any time:

```typescript
import { runRecordingDiagnostics, RecordingDiagnosticLevel } from './recording/diagnostics';

const result = await runRecordingDiagnostics(page, context, {
  level: RecordingDiagnosticLevel.FULL,
  logResults: true,
});

if (!result.ready) {
  for (const issue of result.issues) {
    console.error(`[${issue.severity}] ${issue.code}: ${issue.message}`);
    if (issue.suggestion) {
      console.log(`  → ${issue.suggestion}`);
    }
  }
}
```

### Script Verification

Verify script injection on a specific page:

```typescript
import { verifyScriptInjection, waitForScriptReady } from './recording/verification';

// Wait for script with timeout
const verification = await waitForScriptReady(page, 5000);

console.log('Script status:', {
  loaded: verification.loaded,
  ready: verification.ready,
  inMainContext: verification.inMainContext,
  handlersCount: verification.handlersCount,
});
```

## Common Issues and Solutions

### Events Not Captured At All

1. Check `verifyScriptInjection(page)` - if `loaded: false`, HTML injection failed
2. Verify page was navigated via HTTP(S), not `page.setContent()` or `data:` URLs
3. Check browser console for JavaScript errors
4. Ensure `setEventHandler()` was called before user interaction

### Navigation Events Missing (pushState/replaceState)

1. Check `inMainContext` in verification - if `false`, the script is in ISOLATED context
2. This is the classic context isolation issue with rebrowser-playwright
3. Verify script is injected via HTML, not `page.evaluate()`
4. Check route interception is working and modifying HTML

### Duplicate Events

1. Check `handlersCount` - should be 7-12, not doubled
2. May indicate script injected multiple times without cleanup
3. Verify the cleanup function runs on re-injection

### Delayed Events

1. Check debounce settings in `selector-config.ts`
2. Monitor network tab for slow responses to `/__vrooli_recording_event__`

## Environment Configuration

Switch providers via environment variable:

```bash
# Use rebrowser-playwright (default)
PLAYWRIGHT_PROVIDER=rebrowser-playwright

# Use standard playwright (for debugging)
PLAYWRIGHT_PROVIDER=playwright
```

Note: Currently only rebrowser-playwright is installed. To use standard playwright:
```bash
pnpm add playwright
```

## Related Files

- `src/playwright/types.ts` - Provider interface and capability types
- `src/playwright/provider.ts` - Active provider configuration
- `src/recording/io/context-initializer.ts` - HTML injection and route setup
- `src/recording/testing/diagnostics.ts` - Automated diagnostic system
- `src/recording/validation/verification.ts` - Script injection verification
- `src/recording/capture/browser-scripts/recording-script.js` - Browser-side capture
- `src/recording/orchestration/pipeline-manager.ts` - Recording orchestration
