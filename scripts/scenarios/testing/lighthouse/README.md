# 🚦 Lighthouse Testing Infrastructure

Centralized Lighthouse testing system for Vrooli scenarios. Provides automated performance, accessibility, best practices, and SEO auditing integrated with the phase-based testing framework and requirements tracking system.

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [Usage](#usage)
- [Requirements Integration](#requirements-integration)
- [Troubleshooting](#troubleshooting)
- [Examples](#examples)

---

## 🚀 Quick Start

### 1. Install Dependencies

```bash
cd scripts/scenarios/testing/lighthouse
npm install
```

This installs `lighthouse` and `chrome-launcher` required for running audits.

### 2. Initialize a Scenario

```bash
cd scenarios/browser-automation-studio

# Create .vrooli directory and config template
../../scripts/scenarios/testing/lighthouse/config.sh init .
```

This creates:
- `.vrooli/lighthouse.json` - Configuration file
- `test/artifacts/lighthouse/` - Output directory for reports
- Updates `.gitignore` to exclude HTML/JSON reports

### 3. Configure Pages and Thresholds

Edit `.vrooli/lighthouse.json`:

```json
{
  "enabled": true,
  "pages": [
    {
      "id": "home",
      "path": "/",
      "label": "Dashboard Home",
      "viewport": "desktop",
      "thresholds": {
        "performance": { "error": 0.85, "warn": 0.90 },
        "accessibility": { "error": 0.90, "warn": 0.95 }
      },
      "requirements": ["BAS-PERF-HOME-LOAD", "BAS-A11Y-DASHBOARD"]
    }
  ]
}
```

### 4. Run Tests

```bash
# Run all performance tests (including Lighthouse)
./test/run-tests.sh --phases performance

# Or run performance phase directly
./test/phases/test-performance.sh
```

---

## 🏗️ Architecture

### File Structure

```
scripts/scenarios/testing/lighthouse/
├── README.md              # This file
├── package.json           # Node.js dependencies
├── runner.sh              # Bash orchestration script
├── runner.js              # Node.js Lighthouse executor
└── config.sh              # Default configs and utilities

scenarios/<scenario-name>/
├── .vrooli/
│   ├── config.json        # Page definitions and thresholds
│   └── custom-config.js   # Optional Lighthouse config overrides
├── test/
│   ├── phases/
│   │   └── test-performance.sh  # Sources lighthouse/runner.sh
│   └── artifacts/
│       └── lighthouse/    # Generated reports (gitignored)
└── requirements/
    └── performance/
        └── lighthouse.json  # Performance requirements
```

### Execution Flow

```
test-performance.sh
    ↓
lighthouse::run_audits (runner.sh)
    ↓
node runner.js --config .vrooli/lighthouse.json
    ↓
For each page:
    - Launch Chrome
    - Run Lighthouse audit
    - Check thresholds
    - Save HTML + JSON reports
    - Record requirement evidence
    ↓
Write coverage/phase-results/lighthouse.json
    ↓
requirements/report.js --mode sync
    ↓
Update requirement files with live status
```

---

## ⚙️ Configuration

### Top-Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | boolean | Enable/disable Lighthouse testing |
| `pages` | array | List of pages to audit |
| `global_options` | object | Lighthouse and Chrome settings |
| `reporting` | object | Report output configuration |

### Page Configuration

```json
{
  "id": "workflow-builder",
  "path": "/workflow/new",
  "label": "Workflow Builder",
  "viewport": "desktop",
  "waitForSelector": "[data-testid='canvas-container']",
  "waitForMs": 2000,
  "thresholds": {
    "performance": { "error": 0.75, "warn": 0.85 },
    "accessibility": { "error": 0.90, "warn": 0.95 },
    "best-practices": { "error": 0.80, "warn": 0.90 },
    "seo": { "error": 0.75, "warn": 0.85 }
  },
  "requirements": ["BAS-PERF-WORKFLOW-INTERACTIVE"]
}
```

### Field Reference

| Field | Required | Description |
|-------|----------|-------------|
| `id` | ✅ | Unique page identifier (used in reports) |
| `path` | ✅ | URL path relative to base URL (e.g., "/", "/dashboard") |
| `label` | ❌ | Human-readable label (defaults to path) |
| `viewport` | ❌ | `"desktop"` or `"mobile"` (default: desktop) |
| `waitForSelector` | ❌ | CSS selector to wait for before auditing |
| `waitForMs` | ❌ | Additional wait time in milliseconds |
| `thresholds` | ❌ | Per-category thresholds (see below) |
| `requirements` | ❌ | Array of requirement IDs this page validates |

### Threshold Configuration

Thresholds are specified per Lighthouse category:

```json
{
  "performance": { "error": 0.85, "warn": 0.90 },
  "accessibility": { "error": 0.90, "warn": 0.95 },
  "best-practices": { "error": 0.85, "warn": 0.90 },
  "seo": { "error": 0.80, "warn": 0.90 }
}
```

- **error**: Score below this fails the test (blocks CI/CD)
- **warn**: Score below this generates a warning (doesn't block)
- Scores are 0.0 to 1.0 (displayed as 0-100%)

**Recommended Thresholds:**

| Page Type | Performance | Accessibility | Best Practices | SEO |
|-----------|-------------|---------------|----------------|-----|
| Static Landing | 0.90 / 0.95 | 0.95 / 0.98 | 0.90 / 0.95 | 0.90 / 0.95 |
| Dashboard (SPA) | 0.80 / 0.90 | 0.90 / 0.95 | 0.85 / 0.90 | 0.80 / 0.90 |
| Heavy Interactive | 0.70 / 0.85 | 0.90 / 0.95 | 0.80 / 0.90 | 0.75 / 0.85 |
| Mobile | -0.10 adjustment | Same | Same | Same |

### Global Options

```json
{
  "global_options": {
    "lighthouse": {
      "extends": "lighthouse:default",
      "settings": {
        "onlyCategories": ["performance", "accessibility", "best-practices", "seo"],
        "throttlingMethod": "simulate",
        "formFactor": "desktop"
      }
    },
    "chrome_flags": ["--headless", "--no-sandbox", "--disable-gpu"],
    "timeout_ms": 60000,
    "retries": 2
  }
}
```

### Reporting Options

```json
{
  "reporting": {
    "output_dir": "test/artifacts/lighthouse",
    "formats": ["json", "html"],
    "keep_reports": 10,
    "fail_on_error": true,
    "fail_on_warn": false
  }
}
```

---

## 🎯 Usage

### Command Line

```bash
# Run from scenario directory
cd scenarios/browser-automation-studio

# Run performance phase (includes Lighthouse if configured)
./test/phases/test-performance.sh

# Run all tests with performance phase
./test/run-tests.sh --phases structure unit integration performance

# View reports
ls -la test/artifacts/lighthouse/
open test/artifacts/lighthouse/home_*.html
```

### From Phase Scripts

```bash
#!/bin/bash
source "${APP_ROOT}/scripts/scenarios/testing/lighthouse/runner.sh"

# Check if Lighthouse is configured
if lighthouse::has_config; then
  # Run audits
  if lighthouse::run_audits; then
    log::success "Lighthouse audits passed"
  else
    log::error "Lighthouse audits failed"
  fi
else
  log::info "Lighthouse not configured"
fi
```

### Helper Functions

```bash
# Initialize Lighthouse for a scenario
scripts/scenarios/testing/lighthouse/config.sh init scenarios/my-scenario

# Check dependencies
lighthouse::check_dependencies

# (Optional) Source helpers for advanced usage
source scripts/scenarios/testing/lighthouse/config.sh

# Validate config file
lighthouse::validate_config .vrooli/lighthouse.json

# List configured pages
lighthouse::list_pages .vrooli/lighthouse.json
```

---

## 🔗 Requirements Integration

### Linking Pages to Requirements

In `.vrooli/lighthouse.json`:

```json
{
  "pages": [
    {
      "id": "home",
      "path": "/",
      "label": "Dashboard Home",
      "requirements": ["BAS-PERF-HOME-LOAD", "BAS-A11Y-DASHBOARD"]
    }
  ]
}
```

### Defining Lighthouse Requirements

In `requirements/performance/lighthouse.json`:

```json
{
  "requirements": [
    {
      "id": "BAS-PERF-HOME-LOAD",
      "category": "performance.vrooli",
      "title": "Dashboard loads with >85% performance score",
      "description": "Home page must achieve Lighthouse performance score of 0.85+ on desktop",
      "status": "complete",
      "criticality": "P0",
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
  ]
}
```

### Automatic Evidence Collection

After tests run:

1. **Phase Results**: `coverage/phase-results/lighthouse.json` contains:
   ```json
   {
     "requirements": [
       {
         "id": "BAS-PERF-HOME-LOAD",
         "status": "passed",
         "evidence": "Lighthouse: Dashboard Home - Performance: 0.92, Accessibility: 0.95"
       }
     ]
   }
   ```

2. **Auto-Sync**: `scripts/requirements/report.js --mode sync` updates requirement files

3. **Validation Status**: Requirements show live status from latest test run

### Adding Lighthouse to Existing Requirements

You can add Lighthouse validations to any existing requirement:

```json
{
  "id": "MY-EXISTING-REQUIREMENT",
  "validation": [
    {
      "type": "test",
      "ref": "ui/src/__tests__/Home.test.tsx",
      "phase": "unit",
      "status": "implemented"
    },
    {
      "type": "lighthouse",
      "ref": ".vrooli/lighthouse.json",
      "page_id": "home",
      "category": "performance",
      "phase": "performance",
      "status": "implemented"
    }
  ]
}
```

---

## 🐛 Troubleshooting

### Chrome Launch Failures

**Error**: `Chrome didn't launch properly`

**Solutions**:
```bash
# Check Chrome/Chromium is installed
which google-chrome chromium-browser

# Install if missing (Ubuntu/Debian)
sudo apt-get install chromium-browser

# macOS
brew install chromium

# Add required flags to config
"chrome_flags": ["--headless", "--no-sandbox", "--disable-dev-shm-usage"]
```

### Port Conflicts

**Error**: `Unable to determine UI_PORT`

**Solutions**:
```bash
# Ensure scenario is running
vrooli scenario status browser-automation-studio

# Start if not running
vrooli scenario start browser-automation-studio

# Verify UI port
vrooli scenario port browser-automation-studio UI_PORT
```

### Low Scores

**Issue**: Lighthouse scores lower than expected

**Common Causes**:
1. **Development Build**: Using dev build instead of production
   - Solution: Run `npm run build` before testing
2. **Resource Contention**: CPU/memory constrained
   - Solution: Close other applications, increase timeout
3. **Network Throttling**: Too aggressive throttling
   - Solution: Adjust `throttlingMethod` in config
4. **Unoptimized Assets**: Large images, unminified JS
   - Solution: Optimize assets, enable compression

### Flaky Scores

**Issue**: Scores vary between runs

**Solutions**:
```json
{
  "global_options": {
    "lighthouse": {
      "settings": {
        "throttlingMethod": "simulate",  // More consistent than "devtools"
        "throttling": {
          "cpuSlowdownMultiplier": 4
        }
      }
    },
    "retries": 2  // Retry failed audits
  }
}
```

### Missing Dependencies

**Error**: `Cannot find module 'lighthouse'`

**Solution**:
```bash
cd scripts/scenarios/testing/lighthouse
npm install
```

---

## 📖 Examples

### Example 1: Simple Single-Page Site

```json
{
  "enabled": true,
  "pages": [
    {
      "id": "home",
      "path": "/",
      "label": "Landing Page",
      "thresholds": {
        "performance": { "error": 0.90, "warn": 0.95 },
        "accessibility": { "error": 0.95, "warn": 0.98 },
        "seo": { "error": 0.90, "warn": 0.95 }
      }
    }
  ]
}
```

### Example 2: Multi-Page Dashboard

```json
{
  "enabled": true,
  "pages": [
    {
      "id": "home",
      "path": "/",
      "label": "Dashboard Home",
      "viewport": "desktop",
      "thresholds": {
        "performance": { "error": 0.85, "warn": 0.90 },
        "accessibility": { "error": 0.90, "warn": 0.95 }
      },
      "requirements": ["BAS-PERF-HOME-LOAD"]
    },
    {
      "id": "workflow-builder",
      "path": "/workflow/new",
      "label": "Workflow Builder",
      "viewport": "desktop",
      "waitForSelector": "[data-testid='canvas-container']",
      "waitForMs": 2000,
      "thresholds": {
        "performance": { "error": 0.70, "warn": 0.80 },
        "accessibility": { "error": 0.90, "warn": 0.95 }
      },
      "requirements": ["BAS-PERF-WORKFLOW-INTERACTIVE"]
    },
    {
      "id": "mobile-home",
      "path": "/",
      "label": "Dashboard (Mobile)",
      "viewport": "mobile",
      "thresholds": {
        "performance": { "error": 0.80, "warn": 0.90 },
        "accessibility": { "error": 0.90, "warn": 0.95 }
      }
    }
  ],
  "reporting": {
    "fail_on_error": true,
    "fail_on_warn": false
  }
}
```

### Example 3: Custom Lighthouse Configuration

Create `.vrooli/custom-config.js`:

```javascript
module.exports = {
  extends: 'lighthouse:default',
  settings: {
    onlyAudits: [
      'first-contentful-paint',
      'largest-contentful-paint',
      'cumulative-layout-shift',
      'total-blocking-time',
      'speed-index'
    ],
    skipAudits: ['uses-http2']
  }
};
```

---

## 🔧 Advanced Configuration

### Custom Chrome Flags

```json
{
  "global_options": {
    "chrome_flags": [
      "--headless",
      "--no-sandbox",
      "--disable-gpu",
      "--disable-dev-shm-usage",
      "--disable-extensions",
      "--disable-setuid-sandbox",
      "--user-data-dir=/tmp/chrome-lighthouse"
    ]
  }
}
```

### Network Throttling Presets

```json
{
  "global_options": {
    "lighthouse": {
      "settings": {
        "throttlingMethod": "simulate",
        "throttling": {
          "rttMs": 150,
          "throughputKbps": 1638.4,
          "requestLatencyMs": 150,
          "downloadThroughputKbps": 1638.4,
          "uploadThroughputKbps": 675,
          "cpuSlowdownMultiplier": 4
        }
      }
    }
  }
}
```

### Per-Page Custom Config

```json
{
  "pages": [
    {
      "id": "heavy-page",
      "path": "/complex-dashboard",
      "lighthouse_config": {
        "settings": {
          "maxWaitForLoad": 90000,
          "throttlingMethod": "provided"
        }
      }
    }
  ]
}
```

---

## 📚 Additional Resources

- [Lighthouse Documentation](https://developer.chrome.com/docs/lighthouse/overview/)
- [Lighthouse Configuration](https://github.com/GoogleChrome/lighthouse/blob/main/docs/configuration.md)
- [Performance Budgets](https://web.dev/performance-budgets-101/)
- [Vrooli Testing Guide](../../../docs/testing/guides/scenario-testing.md)
- [Requirements Tracking](../../../docs/testing/guides/requirement-tracking.md)

---

## 🚀 Next Steps

1. **Install dependencies**: `cd scripts/scenarios/testing/lighthouse && npm install`
2. **Initialize a scenario**: Run `scripts/scenarios/testing/lighthouse/config.sh init <scenario>`
3. **Configure pages**: Edit `.vrooli/lighthouse.json`
4. **Define requirements**: Add to `requirements/performance/lighthouse.json`
5. **Run tests**: `./test/run-tests.sh --phases performance`
6. **View reports**: Open HTML files in `test/artifacts/lighthouse/`

For questions or issues, see [CLAUDE.md](/CLAUDE.md) or create an issue in the repository.
