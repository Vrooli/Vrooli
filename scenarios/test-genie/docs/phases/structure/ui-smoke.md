# UI Smoke Testing Guide

**Status**: Active
**Last Updated**: 2025-12-02

---

## Overview

UI smoke testing validates that a scenario's UI is accessible, renders correctly, and has no critical JavaScript errors. It runs during the **structure** phase as a fast sanity check before deeper testing.

> **Note**: UI smoke testing is now executed through the Go-native orchestrator. The bash shell library references in the integration section below are deprecated and retained for historical context only.

## Quick Start

### Enable UI Smoke Testing

Add UI smoke configuration to `.vrooli/testing.json`:

```json
{
  "phases": {
    "structure": {
      "uiSmoke": {
        "enabled": true,
        "pages": [
          { "path": "/", "name": "Home" },
          { "path": "/dashboard", "name": "Dashboard" }
        ],
        "timeout": 30000
      }
    }
  }
}
```

### Run Smoke Tests

```bash
# Run structure phase (includes UI smoke)
test-genie execute my-scenario --phases structure

# Or directly
./test/phases/test-structure.sh
```

## Configuration Reference

### Full Configuration

```json
{
  "phases": {
    "structure": {
      "uiSmoke": {
        "enabled": true,
        "pages": [
          {
            "path": "/",
            "name": "Home Page",
            "waitForSelector": "[data-testid='app-loaded']",
            "waitForMs": 2000
          },
          {
            "path": "/dashboard",
            "name": "Dashboard",
            "requireAuth": false,
            "skipScreenshot": false
          }
        ],
        "viewport": {
          "width": 1440,
          "height": 900
        },
        "timeout": 30000,
        "screenshots": {
          "enabled": true,
          "directory": "coverage/screenshots"
        },
        "consoleErrors": {
          "failOnError": true,
          "ignorePatterns": [
            "favicon.ico"
          ]
        }
      }
    }
  }
}
```

### Page Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | string | required | URL path to test |
| `name` | string | path | Display name for reports |
| `waitForSelector` | string | - | CSS selector to wait for |
| `waitForMs` | number | 0 | Additional wait after load |
| `requireAuth` | boolean | false | Skip if auth required |
| `skipScreenshot` | boolean | false | Don't capture screenshot |

### Global Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | false | Enable UI smoke testing |
| `timeout` | number | 30000 | Max time per page (ms) |
| `viewport.width` | number | 1440 | Browser width |
| `viewport.height` | number | 900 | Browser height |

## What Gets Tested

### 1. Page Load

- HTTP response status (expects 200)
- Page renders without crash
- No network timeout

### 2. Console Errors

- JavaScript errors captured
- React/Vue hydration errors detected
- Unhandled promise rejections caught

### 3. Screenshots

- Visual snapshot captured
- Stored for regression comparison
- Available in test artifacts

### 4. DOM Validation

- Expected elements exist (via `waitForSelector`)
- No blank page (body has content)

## Browserless Integration

UI smoke tests use Browserless for headless browser automation:

```bash
# Ensure Browserless is running
vrooli resource status browserless

# Or start it
vrooli resource start browserless
```

### Direct API Usage

```bash
# Navigate and capture screenshot
curl -X POST "http://localhost:3000/screenshot" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:5173/",
    "options": {
      "fullPage": true,
      "type": "png"
    }
  }'
```

## Results Format

### Phase Results

```json
{
  "phase": "structure",
  "status": "passed",
  "uiSmoke": {
    "pages": [
      {
        "path": "/",
        "name": "Home",
        "status": "passed",
        "loadTimeMs": 1250,
        "consoleErrors": [],
        "screenshot": "coverage/screenshots/home.png"
      },
      {
        "path": "/dashboard",
        "name": "Dashboard",
        "status": "passed",
        "loadTimeMs": 890,
        "consoleErrors": [],
        "screenshot": "coverage/screenshots/dashboard.png"
      }
    ]
  }
}
```

### Console Errors

```json
{
  "consoleErrors": [
    {
      "type": "error",
      "text": "Uncaught TypeError: Cannot read property 'map' of undefined",
      "url": "http://localhost:5173/assets/main-abc123.js",
      "lineNumber": 42
    }
  ]
}
```

## Best Practices

### 1. Test Critical Pages Only

```json
{
  "pages": [
    { "path": "/", "name": "Landing" },
    { "path": "/dashboard", "name": "Main Dashboard" },
    { "path": "/settings", "name": "Settings" }
  ]
}
```

Don't test every route - focus on entry points.

### 2. Use waitForSelector

```json
{
  "path": "/dashboard",
  "waitForSelector": "[data-testid='dashboard-loaded']"
}
```

Ensures page is fully rendered, not just network idle.

### 3. Handle Auth Gracefully

```json
{
  "path": "/admin",
  "requireAuth": true
}
```

Pages requiring auth are skipped in smoke tests (test in integration phase instead).

### 4. Ignore Expected Errors

```json
{
  "consoleErrors": {
    "ignorePatterns": [
      "favicon.ico",
      "analytics",
      "Third-party cookie"
    ]
  }
}
```

Filter out noise from third-party scripts.

### 5. Set Appropriate Timeouts

```json
{
  "timeout": 30000,
  "pages": [
    {
      "path": "/heavy-page",
      "waitForMs": 5000
    }
  ]
}
```

Slow pages need longer timeouts.

## Troubleshooting

### Page Times Out

**Symptom**: Smoke test fails with timeout error

**Causes**:
1. Page never reaches networkidle
2. Infinite loading spinner
3. Scenario not running

**Solutions**:
```json
{
  "path": "/slow-page",
  "waitForMs": 5000
}
```

Or increase global timeout:
```json
{
  "timeout": 60000
}
```

### Console Error False Positives

**Symptom**: Tests fail on expected/harmless errors

**Solution**: Add to ignore patterns:
```json
{
  "consoleErrors": {
    "ignorePatterns": [
      "specific error message",
      "third-party-script.js"
    ]
  }
}
```

### Screenshots Not Captured

**Symptom**: Screenshot directory empty

**Causes**:
1. Browserless not running
2. Permission denied
3. Directory doesn't exist

**Solutions**:
```bash
# Check Browserless
vrooli resource status browserless

# Create directory
mkdir -p coverage/screenshots

# Check permissions
ls -la coverage/
```

### Blank Screenshots

**Symptom**: Screenshots are empty or white

**Causes**:
1. Page hasn't rendered yet
2. CSS loading issues
3. JavaScript errors preventing render

**Solutions**:
```json
{
  "path": "/",
  "waitForSelector": "[data-testid='content']",
  "waitForMs": 2000
}
```

## Integration with Other Phases

### Structure Phase

UI smoke runs as part of structure validation:
- Fast (< 30s total)
- No scenario startup required (uses existing)
- Catches obvious UI breaks

### Integration Phase

For deeper UI testing, use BAS workflows:
- Full user journeys
- Form submissions
- State changes

See [UI Automation with BAS](ui-automation-with-bas.md).

### Performance Phase

For UI performance metrics, use Lighthouse:
- Core Web Vitals
- Accessibility scores
- Best practices

See [Lighthouse Integration](../performance/lighthouse.md).

## See Also

- [Structure Phase](README.md) - Structure phase overview
- [UI Automation with BAS](../playbooks/ui-automation-with-bas.md) - Full UI testing
- [Writing Testable UIs](../../guides/ui-testability.md) - Design for automation
- [Lighthouse Integration](../performance/lighthouse.md) - Performance testing
- [Phases Overview](../README.md) - Phase architecture
- [Troubleshooting](../../guides/troubleshooting.md) - Debug common issues
