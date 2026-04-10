# Injection Strategy Troubleshooting

Common issues and solutions for recording script injection.

## Quick Diagnosis

Run with diagnostics enabled:

```bash
INJECTION_DIAGNOSTICS=true npm start
```

Check the logs for:
- Which strategy was selected
- Whether injection succeeded
- Sanity check results

## Common Issues

### Issue: Timeline shows no events

**Symptoms:**
- Recording starts without errors
- User performs actions (clicks, scrolls, inputs)
- Timeline remains empty

**Likely Cause:** Script not injected or not in MAIN context

**Solutions:**

1. **Force init-script strategy:**
   ```bash
   INJECTION_STRATEGY=init-script npm start
   ```

2. **Check sanity check results:**
   ```typescript
   const initializer = new RecordingContextInitializer({
     runSanityCheck: true,
     onSanityCheckComplete: (result) => {
       if (!result.ready) {
         console.error('Injection issues:', result.issues);
       }
     },
   });
   ```

3. **Verify strategy selection:**
   ```typescript
   const strategyName = initializer.getInjectionStrategyName();
   console.log('Using strategy:', strategyName);
   ```

### Issue: "Script running in ISOLATED context"

**Symptoms:**
- Sanity check reports `inMainContext: false`
- History API events (SPA navigation) not captured
- Other events may work partially

**Cause:** Script was injected via `page.evaluate()` instead of HTML injection

**Solutions:**

1. **Ensure init-script strategy:**
   ```bash
   INJECTION_STRATEGY=init-script npm start
   ```

2. **Check that route-injection is NOT being used:**
   - Route injection is broken with rebrowser-playwright
   - It may inject but in wrong context

3. **Verify the script source:**
   ```typescript
   // The script should set this marker
   window.__vrooli_recording_script_context = 'MAIN';
   ```

### Issue: "Script not loaded"

**Symptoms:**
- Sanity check reports `loaded: false`
- No events captured at all
- Browser console shows no recording script errors

**Cause:** Injection mechanism failed

**Solutions:**

1. **Check navigation type:**
   - Only HTTP(S) pages get injected
   - `about:blank`, `data:`, `file:` URLs may not work

2. **Try CDP injection:**
   ```bash
   INJECTION_STRATEGY=cdp-injection npm start
   ```

3. **Check for route interception issues:**
   - rebrowser-playwright may block route handlers
   - Use init-script strategy instead

### Issue: CDP errors with Firefox/WebKit

**Symptoms:**
- Errors mentioning CDP
- "Protocol error" messages
- Strategy works on Chromium but not Firefox

**Cause:** CDP is Chromium-only

**Solutions:**

1. **Use init-script strategy:**
   ```bash
   INJECTION_STRATEGY=init-script npm start
   ```
   Init-script works on all browsers.

2. **Avoid cdp-injection strategy:**
   ```typescript
   const initializer = new RecordingContextInitializer({
     injectionStrategy: 'init-script',  // Not 'cdp-injection'
   });
   ```

### Issue: Low handler count

**Symptoms:**
- Sanity check reports `handlersCount < 7`
- Some event types missing

**Cause:** Script partially initialized

**Solutions:**

1. **Check for JavaScript errors:**
   - Open browser console
   - Look for errors in recording script

2. **Increase init timeout:**
   ```typescript
   await waitForScriptReady(page, 10000);  // 10 second timeout
   ```

3. **Check for conflicting scripts:**
   - Other scripts may interfere with event handlers
   - Content Security Policy may block inline scripts

### Issue: Strategy auto-detection takes too long

**Symptoms:**
- Long delay on first page load
- Multiple "trying strategy" log messages

**Cause:** Auto-detector tries multiple strategies

**Solutions:**

1. **Explicitly specify strategy:**
   ```bash
   INJECTION_STRATEGY=init-script npm start
   ```

2. **Cache detected strategy:**
   ```typescript
   // Detect once
   const result = await detector.detect(context, options);
   const cachedStrategy = result.strategyName;

   // Reuse for subsequent contexts
   const initializer = new RecordingContextInitializer({
     injectionStrategy: cachedStrategy,
   });
   ```

## Diagnostic Commands

### Check injection stats

```typescript
const stats = initializer.getInjectionStrategyStats();
console.log('Stats:', JSON.stringify(stats, null, 2));
```

### Manual sanity check

```typescript
const result = await initializer.runSanityCheckOnPage(page);
console.log('Sanity check:', JSON.stringify(result, null, 2));
```

### Verify via browser console

In the browser's developer console:

```javascript
// Check if script is loaded
console.log('Loaded:', window.__vrooli_recording_script_loaded);

// Check context
console.log('Context:', window.__vrooli_recording_script_context);

// Check handlers
console.log('Handlers:', window.__vrooli_recording_handlers_count);

// Check version
console.log('Version:', window.__vrooli_recording_script_version);
```

## Log Messages Reference

### Success indicators

```
injection: using injection strategy {"strategy":"init-script"}
recording: sanity check PASSED {"handlersCount":7}
```

### Warning indicators

```
injection: using legacy route-injection  # May not work with rebrowser
recording: sanity check FAILED {"issues":["..."]}
```

### Error indicators

```
injection: init-script strategy already initialized  # Double init
injection: strategy not initialized  # Missing initialize() call
recording: sanity check ERROR  # Exception during check
```

## Getting Help

If these solutions don't resolve the issue:

1. Enable full diagnostics:
   ```bash
   INJECTION_DIAGNOSTICS=true DEBUG=* npm start
   ```

2. Capture the following information:
   - Browser type and version
   - Playwright provider (standard or rebrowser)
   - Full console output
   - Sanity check result JSON

3. Check the architecture documentation:
   - [ARCHITECTURE.md](ARCHITECTURE.md)
   - [CDP-USAGE.md](CDP-USAGE.md)
