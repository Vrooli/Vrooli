# Injection Strategies

This document provides a quick start guide for selecting and using injection strategies in the Browser Automation Studio recording system.

## Problem

With rebrowser-playwright, the standard `context.route()` approach for injecting recording scripts doesn't work reliably due to anti-detection patches. We need alternative injection mechanisms.

## Solution: Strategy Selection

| Strategy | Use Case | Provider Support |
|----------|----------|------------------|
| `init-script` | **RECOMMENDED** for rebrowser-playwright | All providers |
| `cdp-injection` | Fallback with full CDP control | Chromium only |
| `route-injection` | Legacy, standard Playwright only | **BROKEN** with rebrowser |

## Quick Start

### Default (Recommended)

By default, the system auto-selects `init-script` for rebrowser-playwright:

```typescript
import { RecordingContextInitializer } from './recording/io/context-initializer';

const initializer = new RecordingContextInitializer({
  bindingName: '__vrooli_recordAction',
  runSanityCheck: true,
});

await initializer.initialize(context);
```

### Force a Specific Strategy

#### Via Environment Variable

```bash
# RECOMMENDED: init-script (uses context.addInitScript)
INJECTION_STRATEGY=init-script npm start

# Fallback: CDP injection (Chromium only)
INJECTION_STRATEGY=cdp-injection npm start

# Legacy: Route injection (standard Playwright only)
INJECTION_STRATEGY=route-injection npm start
```

#### Via Code

```typescript
const initializer = new RecordingContextInitializer({
  injectionStrategy: 'init-script',  // or 'cdp-injection', 'route-injection', 'auto'
});
```

## Environment Variables

| Variable | Values | Description |
|----------|--------|-------------|
| `INJECTION_STRATEGY` | `auto`, `init-script`, `cdp-injection`, `route-injection` | Force specific strategy |
| `INJECTION_DIAGNOSTICS` | `true`, `false` | Enable verbose injection logging |

## Strategy Details

### init-script (Recommended)

Uses `context.addInitScript()` to register a script that runs on every new document.

**How it works:**
1. Script registered at context creation
2. Runs in MAIN execution context (not ISOLATED)
3. Executes before any page JavaScript
4. Properly wraps History API
5. Persists across navigations

**When to use:**
- rebrowser-playwright environments
- When route interception is unreliable
- Default for all recording features

### cdp-injection (Fallback)

Uses Chrome DevTools Protocol `Page.addScriptToEvaluateOnNewDocument`.

**How it works:**
1. Creates CDP session per page
2. Registers script via CDP command
3. Script runs in MAIN context
4. Full control and debugging visibility

**When to use:**
- Debugging injection issues
- When init-script doesn't work
- Chromium-only environments

**Limitations:**
- Chromium-only (no Firefox/WebKit)
- CDP usage may be detectable

### route-injection (Legacy)

Uses `context.route()` to intercept HTML and inject scripts.

**How it works:**
1. Registers route handler for all requests
2. Intercepts document requests
3. Modifies HTML to include script
4. Returns modified HTML

**When to use:**
- Standard Playwright (not rebrowser)
- When other strategies aren't available

**Limitations:**
- **BROKEN with rebrowser-playwright**
- Route interception bypassed by anti-detection

## Verification

Check that injection is working:

```typescript
// Get strategy being used
const strategyName = initializer.getInjectionStrategyName();
console.log('Using strategy:', strategyName);

// Get injection stats
const stats = initializer.getInjectionStrategyStats();
console.log('Injections:', stats.successful, '/', stats.attempted);

// Run sanity check
const result = await initializer.runSanityCheckOnPage(page);
if (!result.ready) {
  console.error('Issues:', result.issues);
}
```

## Troubleshooting

### Timeline doesn't show events

1. Check strategy selection:
   ```bash
   INJECTION_DIAGNOSTICS=true npm start
   ```

2. Force init-script:
   ```bash
   INJECTION_STRATEGY=init-script npm start
   ```

3. Check sanity check result in logs

### Script not in MAIN context

This usually means the wrong strategy is being used. Ensure `init-script` or `cdp-injection` is selected:

```bash
INJECTION_STRATEGY=init-script npm start
```

### CDP-related errors

If using cdp-injection and seeing errors:
- Verify browser is Chromium-based
- Check that CDP session can be created
- Fall back to init-script strategy

## See Also

- [ARCHITECTURE.md](ARCHITECTURE.md) - Detailed architecture documentation
- [CDP-USAGE.md](CDP-USAGE.md) - CDP usage guidelines
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues and solutions
