# Lighthouse Integration Guide

**Status**: Active
**Last Updated**: 2025-12-02

---

Complete guide to integrating Lighthouse performance and accessibility testing into Vrooli scenarios.

> **Note**: Lighthouse testing is executed through the Go-native test-genie orchestrator during the performance phase. Configuration is done via `.vrooli/lighthouse.json` in your scenario.

## Overview

Lighthouse testing provides automated auditing of:
- **Performance**: Load times, responsiveness, metrics (FCP, LCP, CLS, etc.)
- **Accessibility**: WCAG compliance, ARIA usage, color contrast
- **Best Practices**: Security, browser compatibility, modern standards
- **SEO**: Meta tags, structured data, crawlability

**Key Features**:
- Integrated with phase-based testing system
- Automatic requirements tracking and evidence collection
- Per-page thresholds (error/warn levels)
- Desktop and mobile viewport testing
- HTML and JSON report generation
- Configurable Chrome flags and Lighthouse settings

---

## Quick Start

### Prerequisites

1. **Node.js 16+** installed
2. **Chrome/Chromium** browser installed
3. **Scenario with UI** component

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
    "chrome_flags": ["--headless", "--no-sandbox"],
    "timeout_ms": 60000
  },
  "reporting": {
    "formats": ["json", "html"],
    "fail_on_error": true
  }
}
```

### Step 4: Run Tests

```bash
# Ensure scenario is running
vrooli scenario start your-scenario

# Run performance phase
test-genie execute your-scenario --phases performance
```

### Step 5: View Results

```bash
# Open HTML report
open test/artifacts/lighthouse/home_*.html

# View phase results
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
  "waitForSelector": "CSS selector",
  "waitForMs": 2000,
  "thresholds": { ... },
  "requirements": ["REQ-ID"]
}
```

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
- **warn**: Score below this generates warning
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
        "throttlingMethod": "simulate"
      }
    },
    "chrome_flags": ["--headless", "--no-sandbox"],
    "timeout_ms": 60000,
    "retries": 2
  }
}
```

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

```json
{
  "waitForSelector": "[data-testid='content-loaded']",
  "waitForMs": 2000
}
```

---

## Troubleshooting

### Chrome Not Found

```bash
# Ubuntu/Debian
sudo apt-get install chromium-browser

# macOS
brew install chromium
```

### Scenario Not Running

```bash
vrooli scenario status your-scenario
vrooli scenario start your-scenario
```

### Low Scores

1. Open HTML report and review suggestions
2. Check if using dev build instead of production
3. Verify assets are optimized

### Flaky Scores

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

---

## See Also

- [Performance Phase](README.md) - Performance phase overview
- [Performance Testing](performance-testing.md) - Build benchmarks and audits
- [Phases Overview](../README.md) - 7-phase architecture
- [UI Smoke Testing](../structure/ui-smoke.md) - Fast UI validation
- [Requirements Sync](../business/requirements-sync.md) - Evidence collection
- [Troubleshooting](../../guides/troubleshooting.md) - Debug common issues
