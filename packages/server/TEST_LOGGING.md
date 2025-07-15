# Test Logging Configuration

## Overview
The test setup now includes configurable log levels to reduce noise in test output.

## Usage

### Environment Variable
Control test setup/teardown logging with the `TEST_LOG_LEVEL` environment variable:

```bash
# Show only errors (default)
TEST_LOG_LEVEL=ERROR pnpm test

# Show errors and warnings
TEST_LOG_LEVEL=WARN pnpm test

# Show errors, warnings, and info
TEST_LOG_LEVEL=INFO pnpm test

# Show all logging (original behavior)
TEST_LOG_LEVEL=DEBUG pnpm test
```

### Log Levels
- `ERROR` (0): Only critical errors and test failures
- `WARN` (1): Warnings + errors (service unavailable warnings)
- `INFO` (2): General information + warnings + errors
- `DEBUG` (3): Full verbose output (original behavior)

### Default Behavior
- **Default**: `ERROR` level - shows only critical issues
- **Previous behavior**: To get the old verbose output, use `TEST_LOG_LEVEL=DEBUG`

## What This Fixes
- **Before**: Each test file showed ~20 lines of setup/teardown logs
- **After**: Only errors are shown by default, making actual test failures more visible
- **Benefits**: Easier to identify real test failures, cleaner CI output, faster test debugging

## Examples

```bash
# Quiet test run (recommended for CI)
TEST_LOG_LEVEL=ERROR pnpm test

# Debug specific test issues
TEST_LOG_LEVEL=DEBUG pnpm test src/specific/test.file.ts

# Show service warnings but not debug info
TEST_LOG_LEVEL=WARN pnpm test
```