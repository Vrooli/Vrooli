# Test Logging Guide

This guide explains how to control logging verbosity during tests to reduce noise and improve debugging experience.

## Overview

The test environment now has enhanced logging controls to reduce noise by default while still allowing detailed debugging when needed.

## Environment Variables

### LOG_LEVEL
Controls the Winston logger level for the application. In tests, defaults to 'error'.

Valid levels (from most to least verbose):
- `debug`
- `info`
- `notice`
- `warning` (or `warn`)
- `error`
- `crit`
- `alert`
- `emerg`

**Usage:**
```bash
# Run tests with info-level logging
LOG_LEVEL=info pnpm test

# Run tests with debug logging (most verbose)
LOG_LEVEL=debug pnpm test

# Run tests with only critical errors
LOG_LEVEL=crit pnpm test
```

### TEST_LOG_LEVEL
Controls the test setup logging (container management, initialization, etc.)

Valid levels:
- `DEBUG` - Show all setup/teardown logs
- `INFO` - Show important setup information
- `WARN` - Show warnings only
- `ERROR` - Show errors only (default)

**Usage:**
```bash
# Debug test infrastructure issues
TEST_LOG_LEVEL=DEBUG pnpm test

# Normal test run (errors only)
pnpm test
```

## What's Suppressed by Default

In test environment, the following are automatically suppressed:
- Queue health check intervals (30-second checks)
- Swarm and Run monitor intervals
- Redis connection warnings during cleanup
- Console output (unless LOG_LEVEL=debug)

## Capturing Logs in Tests

When you need to verify that your code produces specific logs:

```typescript
import { LogCapture } from "../__test/helpers/logCapture.js";

it("should log an error when validation fails", () => {
    const capture = new LogCapture();
    capture.start();
    
    // Your code that should produce logs
    validateUser({ name: "" });
    
    const logs = capture.stop();
    const errorLogs = capture.findLogs({ level: "error", message: /validation failed/i });
    expect(errorLogs).toHaveLength(1);
});
```

## Suppressing Console Output

For tests that produce expected console output:

```typescript
import { suppressConsole } from "../__test/helpers/logCapture.js";

it("should handle error without console noise", async () => {
    await suppressConsole(async () => {
        // Code that produces console errors
        throw new Error("Expected error");
    });
});
```

## Debugging Failed Tests

When tests fail and you need more information:

1. **First, try with info-level logging:**
   ```bash
   LOG_LEVEL=info pnpm test failing-test.test.ts
   ```

2. **If you need test infrastructure logs:**
   ```bash
   TEST_LOG_LEVEL=DEBUG LOG_LEVEL=info pnpm test failing-test.test.ts
   ```

3. **For maximum verbosity:**
   ```bash
   TEST_LOG_LEVEL=DEBUG LOG_LEVEL=debug pnpm test failing-test.test.ts
   ```

## Best Practices

1. **Keep default settings for CI** - Don't set verbose logging in CI unless debugging specific issues
2. **Use LogCapture for log assertions** - Don't rely on console output for test assertions
3. **Clean up intervals** - Always clear intervals in afterEach/afterAll hooks
4. **Use appropriate log levels** - Use error for actual errors, warn for warnings, info for general information

## Common Issues

### "MaxListenersExceededWarning"
- **Cause:** Multiple event listeners added without cleanup
- **Fix:** Ensure proper cleanup in afterEach hooks

### Excessive "Queue Redis connection not ready" logs
- **Cause:** Health check intervals running during tests
- **Fix:** Already suppressed in test environment

### "Error in swarm/run monitor interval"
- **Cause:** Monitor intervals running during tests
- **Fix:** Already suppressed in test environment

## Troubleshooting

If you're still seeing excessive logs:

1. Check if `NODE_ENV=test` is set
2. Verify no code is overriding log levels
3. Look for direct `console.log/error` calls instead of using the logger
4. Ensure intervals are properly cleaned up in tests