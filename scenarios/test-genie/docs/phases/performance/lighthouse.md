# Lighthouse Integration Guide

**Status**: Active
**Last Updated**: 2025-12-04

---

Complete guide to integrating Lighthouse performance and accessibility testing into Vrooli scenarios.

> **Note**: Lighthouse testing is executed through the Go-native test-genie orchestrator during the performance phase using Google's official Lighthouse CLI. Configuration is done via `.vrooli/lighthouse.json` in your scenario.

**Config location and schema**
- Place your config at `.vrooli/lighthouse.json` inside the scenario root.
- Validate against `scenarios/test-genie/schemas/lighthouse.schema.json` (repo-relative path).

## Overview

Lighthouse testing provides automated auditing of:
- **Performance**: Load times, responsiveness, metrics (FCP, LCP, CLS, etc.)
- **Accessibility**: WCAG compliance, ARIA usage, color contrast
- **Best Practices**: Security, browser compatibility, modern standards
- **SEO**: Meta tags, structured data, crawlability

**Key Features**:
- Uses Google's official Lighthouse CLI (most reliable and up-to-date)
- Integrated with phase-based testing system
- Automatic requirements tracking and evidence collection
- Per-page thresholds (error/warn levels)
- Desktop and mobile viewport testing
- JSON report generation to `coverage/lighthouse/`
- Retry support for flaky network conditions
- Configurable Lighthouse settings

---

## Quick Start

### Prerequisites

1. **Node.js** installed (for Lighthouse CLI)
2. **Lighthouse CLI** installed: `npm install -g lighthouse` (or will use npx)
3. **Chrome/Chromium** installed on the system
4. **Scenario with UI** component running

### Step 1: Create Lighthouse Configuration

Create `.vrooli/lighthouse.json` in your scenario directory:

```json
{
  "_metadata": {
    "description": "Lighthouse testing configuration"
  },
  "enabled": true,
  "pages": [
    {
      "id": "home",
      "path": "/",
      "label": "Home Page",
      "viewport": "desktop",
      "thresholds": {
        "performance": { "error": 0.85, "warn": 0.90 },
        "accessibility": { "error": 0.90, "warn": 0.95 }
      },
      "requirements": ["YOUR-PERF-HOME-LOAD"]
    }
  ],
  "global_options": {
    "timeout_ms": 60000,
    "retries": 2
  },
  "reporting": {
    "formats": ["json"],
    "fail_on_error": true
  }
}
```

### Step 2: Run Tests

```bash
# Ensure scenario is running
vrooli scenario start your-scenario

# Run performance phase
test-genie execute your-scenario --phases performance
```

### Step 3: View Results

```bash
# View per-page reports
cat coverage/lighthouse/home.json

# View summary
cat coverage/lighthouse/summary.json

# View phase results (for requirements integration)
cat coverage/phase-results/lighthouse.json
```

---

## Configuration Reference

### Page Configuration

```json
{
  "id": "unique-page-id",
  "path": "/relative/path",
  "label": "Human Label",
  "viewport": "desktop",
  "waitForMs": 2000,
  "thresholds": { ... },
  "requirements": ["REQ-ID"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier for the page |
| `path` | string | Yes | URL path relative to base URL |
| `label` | string | No | Human-readable name |
| `viewport` | string | No | `"desktop"` or `"mobile"` |
| `waitForMs` | int | No | Milliseconds to wait before audit (adds to timeout) |
| `thresholds` | object | Yes | Category thresholds |
| `requirements` | array | No | Requirement IDs for evidence tracking |

### Threshold Configuration

```json
{
  "thresholds": {
    "performance": { "error": 0.85, "warn": 0.90 },
    "accessibility": { "error": 0.90, "warn": 0.95 },
    "best-practices": { "error": 0.85, "warn": 0.90 },
    "seo": { "error": 0.80, "warn": 0.90 }
  }
}
```

- **error**: Score below this fails the test
- **warn**: Score below this generates warning (test still passes)
- Scores are 0.0-1.0 (displayed as 0-100%)

**Recommended Values by Page Type:**

| Page Type | Performance | Accessibility |
|-----------|-------------|---------------|
| Static Content | 0.90 / 0.95 | 0.95 / 0.98 |
| SPA Dashboard | 0.80 / 0.90 | 0.90 / 0.95 |
| Heavy Interactive | 0.70 / 0.85 | 0.90 / 0.95 |

### Global Options

```json
{
  "global_options": {
    "lighthouse": {
      "settings": {
        "onlyCategories": ["performance", "accessibility"],
        "throttlingMethod": "simulate",
        "formFactor": "desktop"
      }
    },
    "timeout_ms": 90000,
    "retries": 2
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `lighthouse.settings.onlyCategories` | array | All 4 | Categories to audit |
| `lighthouse.settings.throttlingMethod` | string | - | `"simulate"` or `"devtools"` |
| `lighthouse.settings.formFactor` | string | - | `"desktop"` or `"mobile"` (overrides page viewport) |
| `timeout_ms` | int | 90000 | Max audit time in milliseconds |
| `retries` | int | 0 | Retry count for transient failures |

### Reporting Options

```json
{
  "reporting": {
    "formats": ["json"],
    "fail_on_error": true
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `formats` | array | [] | Report formats: `"json"` |
| `fail_on_error` | bool | true | Whether threshold violations fail the phase |

---

## Requirements Integration

### Workflow

1. **Define requirement** in `requirements/performance/lighthouse.json`
2. **Link page to requirement** via `"requirements": ["REQ-ID"]` in config
3. **Run tests** - Lighthouse executes and records evidence
4. **Auto-sync** - Requirements file updated with pass/fail status

### Validation Type: lighthouse

```json
{
  "validation": [
    {
      "type": "lighthouse",
      "ref": ".vrooli/lighthouse.json",
      "page_id": "home",
      "category": "performance",
      "threshold": 0.85,
      "phase": "performance",
      "status": "implemented"
    }
  ]
}
```

---

## Best Practices

### 1. Start with Loose Thresholds

```json
{
  "performance": { "error": 0.60, "warn": 0.75 }
}
```

Tighten over time as you optimize.

### 2. Test Key User Journeys

Choose 3-5 critical pages:
- Landing/Home page
- Main dashboard/feature page
- Settings/configuration page

### 3. Separate Desktop and Mobile

```json
{
  "pages": [
    {
      "id": "home-desktop",
      "path": "/",
      "viewport": "desktop",
      "thresholds": { "performance": { "error": 0.85 } }
    },
    {
      "id": "home-mobile",
      "path": "/",
      "viewport": "mobile",
      "thresholds": { "performance": { "error": 0.75 } }
    }
  ]
}
```

### 4. Use Production Builds

```bash
cd ui && npm run build
cd .. && test-genie execute my-scenario --phases performance
```

### 5. Wait for Dynamic Content

Use `waitForMs` to add a delay before the Lighthouse audit begins:

```json
{
  "waitForMs": 2000
}
```

---

## Troubleshooting

### Lighthouse CLI Not Found

Ensure Lighthouse is installed:

```bash
# Install globally
npm install -g lighthouse

# Or verify npx can find it
npx lighthouse --version
```

### Chrome Not Found

Lighthouse requires Chrome or Chromium. Install one of:

```bash
# Ubuntu/Debian
sudo apt install chromium-browser

# Or set CHROME_PATH to your Chrome installation
export CHROME_PATH=/path/to/chrome
```

### Scenario Not Running

```bash
vrooli scenario status your-scenario
vrooli scenario start your-scenario
```

### Low Scores

1. Review the JSON report at `coverage/lighthouse/<page-id>.json`
2. Check if using dev build instead of production
3. Verify assets are optimized

### Flaky Scores

Enable retries to handle transient network issues:

```json
{
  "global_options": {
    "lighthouse": {
      "settings": {
        "throttlingMethod": "simulate"
      }
    },
    "retries": 2
  }
}
```

### No UI URL Configured

If you see "no UI URL configured", the scenario's UI isn't running or the port wasn't detected. Ensure:
1. The scenario has a UI component
2. The UI is running (`make start` or `vrooli scenario start`)
3. The port is correctly configured in `.vrooli/service.json`

---

## Output Files

When `reporting.formats` includes `"json"`, the following files are generated:

| File | Description |
|------|-------------|
| `coverage/lighthouse/<page-id>.json` | Full Lighthouse report per page |
| `coverage/lighthouse/summary.json` | Aggregated results across all pages |
| `coverage/phase-results/lighthouse.json` | Requirements integration data |

---

## See Also

- [Performance Phase](README.md) - Performance phase overview
- [Phases Overview](../README.md) - 8-phase architecture
- [UI Smoke Testing](../smoke/README.md) - Fast UI validation
- [Requirements Sync](../business/README.md) - Evidence collection
