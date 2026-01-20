# CDP (Chrome DevTools Protocol) Usage Guide

This document explains when and how to use CDP in the Browser Automation Studio recording system.

## Executive Summary

**CDP is NOT the primary injection mechanism.** The recommended `init-script` strategy uses Playwright's `context.addInitScript()` API, which is simpler and works across providers.

CDP is used for:
- Verification (checking if injection worked)
- Fallback injection (when init-script fails)
- Frame streaming (screencast)
- Diagnostics

## Current CDP Usage Inventory

| CDP Domain | Usage | Files | Required? |
|-----------|-------|-------|-----------|
| `Runtime.evaluate` | Verify script in MAIN context | `verification.ts`, `diagnostics.ts` | Yes (for verification) |
| `Page.addScriptToEvaluateOnNewDocument` | CDP injection strategy | `cdp-injection.ts` | Only if using cdp-injection |
| `Page.startScreencast` | Frame streaming | `cdp-screencast.ts` | For streaming only |
| `Page.captureScreenshot` | Polling fallback | `polling.ts` | For polling mode |
| `ServiceWorker.*` | SW management | `controller.ts` | For SW scenarios |
| `Input.dispatch*` | Test simulations | `self-test.ts`, `diagnostics.ts` | Testing only |

## When to Use CDP

### Appropriate Use Cases

1. **Verification in MAIN Context**

   With rebrowser-playwright, `page.evaluate()` runs in ISOLATED context. To verify the recording script (which runs in MAIN), we need CDP:

   ```typescript
   // verification.ts
   const client = await page.context().newCDPSession(page);
   const { result } = await client.send('Runtime.evaluate', {
     expression: `(function() {
       return JSON.stringify({
         loaded: window.__vrooli_recording_script_loaded === true,
         inMainContext: window.__vrooli_recording_script_context === 'MAIN',
       });
     })()`,
   });
   ```

2. **Fallback Injection**

   When `context.addInitScript()` doesn't work, CDP provides direct script injection:

   ```typescript
   // cdp-injection.ts
   const session = await page.context().newCDPSession(page);
   await session.send('Page.addScriptToEvaluateOnNewDocument', {
     source: initScript,
     runImmediately: true,
   });
   ```

3. **Frame Streaming**

   For real-time video capture:

   ```typescript
   // cdp-screencast.ts
   await client.send('Page.startScreencast', {
     format: 'jpeg',
     quality: 80,
     everyNthFrame: 1,
   });
   ```

4. **Debugging and Diagnostics**

   For detailed debugging of injection issues:

   ```typescript
   // diagnostics.ts
   await client.send('Input.dispatchMouseEvent', {
     type: 'mousePressed',
     x: 100,
     y: 100,
     button: 'left',
   });
   ```

### When to Avoid CDP

1. **Script Injection (Primary)**
   - Use `context.addInitScript()` instead
   - Works across all providers
   - Simpler, less error-prone

2. **Firefox/WebKit Support**
   - CDP is Chromium-only
   - These browsers require Playwright APIs

3. **Bot Detection Concerns**
   - CDP usage may be detectable
   - rebrowser-playwright tries to hide it, but not guaranteed

4. **Simple Operations**
   - Use Playwright APIs when possible
   - `page.click()`, `page.fill()`, etc. are safer

## CDP vs Playwright APIs

| Operation | CDP Approach | Playwright Approach | Recommendation |
|-----------|--------------|---------------------|----------------|
| Script injection | `Page.addScriptToEvaluateOnNewDocument` | `context.addInitScript()` | **Playwright** |
| Click element | `Input.dispatchMouseEvent` | `page.click()` | **Playwright** |
| Evaluate in page | `Runtime.evaluate` | `page.evaluate()` | Playwright (unless MAIN context needed) |
| Screenshot | `Page.captureScreenshot` | `page.screenshot()` | Playwright |
| Frame streaming | `Page.startScreencast` | N/A | **CDP** (no alternative) |
| Verify MAIN context | `Runtime.evaluate` | N/A | **CDP** (required) |

## Best Practices

### Session Management

Always clean up CDP sessions:

```typescript
const session = await page.context().newCDPSession(page);
try {
  // Use session...
} finally {
  await session.detach().catch(() => {});
}
```

### Error Handling

CDP errors should be caught and handled gracefully:

```typescript
try {
  await session.send('Runtime.evaluate', { expression: script });
} catch (error) {
  // Log and fall back to alternative approach
  logger.warn('CDP evaluation failed, falling back to Playwright API');
  // ...
}
```

### Performance Considerations

- Create sessions only when needed
- Reuse sessions when making multiple calls
- Detach sessions when done

## Troubleshooting

### "Target closed" Errors

This usually means the page navigated or closed during a CDP operation:

```typescript
try {
  await session.send('Runtime.evaluate', { expression });
} catch (error) {
  if (error.message.includes('Target closed')) {
    // Page navigated, handle gracefully
    return null;
  }
  throw error;
}
```

### "Protocol error" Errors

Check that the CDP command is supported by the browser version:

```typescript
// Some CDP commands require specific browser versions
const browserVersion = await browser.version();
logger.debug('Browser version:', browserVersion);
```

### Firefox/WebKit Errors

CDP is not supported:

```typescript
if (!supportsProvider('chromium')) {
  logger.warn('CDP not available on this browser');
  return fallbackApproach();
}
```

## Security Considerations

CDP provides full access to the browser. In the recording system:

1. **Don't expose CDP to untrusted code**
2. **Don't log sensitive data** from CDP evaluations
3. **Be careful with `Runtime.evaluate`** - it can execute arbitrary code

## References

- [Chrome DevTools Protocol Documentation](https://chromedevtools.github.io/devtools-protocol/)
- [Playwright CDP Documentation](https://playwright.dev/docs/api/class-cdpsession)
- [rebrowser-playwright Source](https://github.com/nicke1234/rebrowser-playwright)
