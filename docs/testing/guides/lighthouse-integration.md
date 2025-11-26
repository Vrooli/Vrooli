# 🚦 Lighthouse Integration Guide

Complete guide to integrating Lighthouse performance and accessibility testing into Vrooli scenarios.

## 📋 Overview

Lighthouse testing provides automated auditing of:
- **Performance**: Load times, responsiveness, metrics (FCP, LCP, CLS, etc.)
- **Accessibility**: WCAG compliance, ARIA usage, color contrast
- **Best Practices**: Security, browser compatibility, modern standards
- **SEO**: Meta tags, structured data, crawlability

**Key Features**:
- ✅ Integrated with phase-based testing system
- ✅ Automatic requirements tracking and evidence collection
- ✅ Per-page thresholds (error/warn levels)
- ✅ Desktop and mobile viewport testing
- ✅ HTML and JSON report generation
- ✅ Configurable Chrome flags and Lighthouse settings

---

## 🚀 Quick Start

### Prerequisites

1. **Node.js 16+** installed
2. **Chrome/Chromium** browser installed
3. **Scenario with UI** component

### Step 1: Install Lighthouse Dependencies

```bash
cd scripts/scenarios/testing/lighthouse
npm install
```

This installs `lighthouse` (v11+) and `chrome-launcher` globally for the testing infrastructure.

### Step 2: Initialize Lighthouse for Your Scenario

```bash
cd scenarios/your-scenario

# Initialize Lighthouse (creates .vrooli/lighthouse.json and directories)
../../scripts/scenarios/testing/lighthouse/config.sh init .
```

This creates:
```
scenarios/your-scenario/
├── .vrooli/
│   └── lighthouse.json     # Page definitions and thresholds
├── test/artifacts/lighthouse/  # Report output directory
└── .gitignore              # Updated to exclude reports
```

### Step 3: Configure Pages to Test

Edit `.vrooli/lighthouse.json`:

```json
{
  "_metadata": {
    "description": "Lighthouse testing configuration for your-scenario"
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
    },
    {
      "id": "dashboard",
      "path": "/dashboard",
      "label": "Main Dashboard",
      "viewport": "desktop",
      "waitForSelector": "[data-testid='dashboard-loaded']",
      "waitForMs": 1000,
      "thresholds": {
        "performance": { "error": 0.80, "warn": 0.90 },
        "accessibility": { "error": 0.90, "warn": 0.95 }
      }
    }
  ],
  "global_options": {
    "chrome_flags": ["--headless", "--no-sandbox", "--disable-gpu"],
    "timeout_ms": 60000
  },
  "reporting": {
    "formats": ["json", "html"],
    "keep_reports": 10,
    "fail_on_error": true,
    "fail_on_warn": false
  }
}
```

### Step 4: Update Performance Phase Script

If your scenario doesn't have `test/phases/test-performance.sh`, copy from the template:

```bash
cp ../../scripts/scenarios/templates/react-vite/test/phases/test-performance.sh test/phases/
```

The template already includes Lighthouse integration. If you have an existing file, ensure it includes:

```bash
#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

# Run Lighthouse audits if config exists
if [ -f "${TESTING_PHASE_SCENARIO_DIR}/.vrooli/lighthouse.json" ]; then
  source "${APP_ROOT}/scripts/scenarios/testing/lighthouse/runner.sh"

  if lighthouse::run_audits; then
    log::success "✅ Lighthouse audits passed"
  else
    testing::phase::add_error "Lighthouse audits failed"
  fi
else
  log::info "No .vrooli/lighthouse.json found; skipping Lighthouse"
fi

testing::phase::end_with_summary "Performance checks completed"
```

### Step 5: Create Performance Requirements

Create `requirements/performance/lighthouse.json`:

```json
{
  "_metadata": {
    "description": "Performance and accessibility requirements validated by Lighthouse",
    "auto_sync_enabled": true
  },
  "requirements": [
    {
      "id": "YOUR-PERF-HOME-LOAD",
      "category": "performance.vrooli",
      "title": "Home page loads with >85% performance score",
      "description": "Home page must achieve Lighthouse performance score of 0.85+ on desktop",
      "status": "in_progress",
      "prd_ref": "OT-P0-001",
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
    },
    {
      "id": "YOUR-A11Y-WCAG-AA",
      "category": "accessibility.vrooli",
      "title": "All pages meet WCAG 2.1 AA standards",
      "description": "Lighthouse accessibility score of 0.90+ across all tested pages",
      "status": "in_progress",
      "prd_ref": "OT-P0-002",
      "validation": [
        {
          "type": "lighthouse",
          "ref": ".vrooli/lighthouse.json",
          "page_id": "home",
          "category": "accessibility",
          "threshold": 0.90,
          "phase": "performance",
          "status": "implemented"
        }
      ]
    }
  ]
}
```

### Step 6: Add Import to Requirements Index

Edit `requirements/index.json` to import the new file:

```json
{
  "imports": [
    "projects/dialog.json",
    "workflow-builder/core.json",
    "performance/lighthouse.json"  // Add this
  ]
}
```

### Step 7: Run Tests

```bash
# Ensure scenario is running
vrooli scenario start your-scenario

# Run performance phase
./test/run-tests.sh --phases performance

# Or run directly
./test/phases/test-performance.sh
```

### Step 8: View Results

```bash
# List generated reports
ls -la test/artifacts/lighthouse/

# Open HTML report in browser
open test/artifacts/lighthouse/home_*.html

# View phase results
cat coverage/phase-results/lighthouse.json

# Check requirement status
node ../../scripts/requirements/report.js --scenario your-scenario --format json
```

---

## 📖 Configuration Reference

### Page Configuration

Each page in the `pages` array can have:

```json
{
  "id": "unique-page-id",           // Required: Used in reports and filenames
  "path": "/relative/path",         // Required: URL path to audit
  "label": "Human Label",           // Optional: Display name
  "viewport": "desktop",            // Optional: "desktop" or "mobile"
  "waitForSelector": "CSS selector", // Optional: Wait for element before auditing
  "waitForMs": 2000,                // Optional: Additional wait time
  "thresholds": { ... },            // Optional: Per-page thresholds
  "requirements": ["REQ-ID"]        // Optional: Link to requirement IDs
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

- **error**: Score below this fails the test (exit code 1)
- **warn**: Score below this generates warning (doesn't fail unless `fail_on_warn: true`)
- Scores are 0.0-1.0 (displayed as 0-100%)
- Missing categories are not checked

**Recommended Values by Page Type:**

| Page Type | Performance | Accessibility | Best Practices | SEO |
|-----------|-------------|---------------|----------------|-----|
| Static Content | 0.90 / 0.95 | 0.95 / 0.98 | 0.90 / 0.95 | 0.90 / 0.95 |
| SPA Dashboard | 0.80 / 0.90 | 0.90 / 0.95 | 0.85 / 0.90 | 0.80 / 0.90 |
| Heavy Interactive | 0.70 / 0.85 | 0.90 / 0.95 | 0.80 / 0.90 | 0.75 / 0.85 |
| Mobile Views | -0.05 to -0.10 | Same | Same | Same |

### Global Options

```json
{
  "global_options": {
    "lighthouse": {
      "extends": "lighthouse:default",
      "settings": {
        "onlyCategories": ["performance", "accessibility", "best-practices", "seo"],
        "throttlingMethod": "simulate",
        "formFactor": "desktop",
        "screenEmulation": {
          "mobile": false,
          "width": 1440,
          "height": 900,
          "deviceScaleFactor": 1
        }
      }
    },
    "chrome_flags": [
      "--headless",
      "--no-sandbox",
      "--disable-gpu",
      "--disable-dev-shm-usage"
    ],
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

## 🔗 Requirements Integration

### Workflow

1. **Define requirement** in `requirements/performance/lighthouse.json`
2. **Link page to requirement** via `"requirements": ["REQ-ID"]` in config
3. **Run tests** - Lighthouse executes and records evidence
4. **Auto-sync** - Requirements file updated with pass/fail status

### Validation Type: lighthouse

In requirements files, use type `"lighthouse"`:

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
      "status": "implemented",
      "notes": "Desktop viewport, threshold: 85%"
    }
  ]
}
```

### Evidence Format

After tests run, `coverage/phase-results/lighthouse.json` contains:

```json
{
  "requirements": [
    {
      "id": "YOUR-PERF-HOME-LOAD",
      "status": "passed",
      "evidence": "Lighthouse: Home Page - Performance: 0.92, Accessibility: 0.95",
      "page_id": "home",
      "updated_at": "2025-11-05T14:32:15Z"
    }
  ]
}
```

The `scripts/requirements/report.js --mode sync` reads this and updates requirement files.

---

## 🎯 Best Practices

### 1. Start with Loose Thresholds

Begin with lower thresholds and tighten over time:

```json
{
  "performance": { "error": 0.60, "warn": 0.75 }  // Initial
  // Later tighten to:
  "performance": { "error": 0.75, "warn": 0.85 }  // Improved
}
```

### 2. Use Warn vs Error Strategically

- **error**: Critical issues that block deployment (P0 requirements)
- **warn**: Nice-to-have improvements (P1/P2 requirements)

```json
{
  "reporting": {
    "fail_on_error": true,   // Block CI/CD on error
    "fail_on_warn": false    // Log warnings but don't block
  }
}
```

### 3. Test Key User Journeys

Choose 3-5 critical pages that represent typical user flows:

- Landing/Home page
- Main dashboard/feature page
- Settings/configuration page
- Mobile variants of above

### 4. Separate Desktop and Mobile

Test same routes with different viewports:

```json
{
  "pages": [
    {
      "id": "home-desktop",
      "path": "/",
      "viewport": "desktop",
      "thresholds": { "performance": { "error": 0.85, "warn": 0.90 } }
    },
    {
      "id": "home-mobile",
      "path": "/",
      "viewport": "mobile",
      "thresholds": { "performance": { "error": 0.75, "warn": 0.85 } }
    }
  ]
}
```

### 5. Use Production Builds

Always test against production builds, not development:

```bash
# Build for production first
cd ui && npm run build

# Then run Lighthouse tests
cd .. && ./test/phases/test-performance.sh
```

### 6. Wait for Dynamic Content

For SPAs that load data asynchronously:

```json
{
  "waitForSelector": "[data-testid='content-loaded']",
  "waitForMs": 2000
}
```

### 7. Consistent Testing Environment

Use `throttlingMethod: "simulate"` for consistent results:

```json
{
  "lighthouse": {
    "settings": {
      "throttlingMethod": "simulate"  // More consistent than "devtools"
    }
  }
}
```

---

## 🐛 Troubleshooting

### Chrome Not Found

**Error**: `No Chrome installations found`

**Solution**:
```bash
# Ubuntu/Debian
sudo apt-get install chromium-browser

# macOS
brew install chromium

# Or specify Chrome path in config
"chrome_flags": ["--chrome-path=/usr/bin/google-chrome"]
```

### Scenario Not Running

**Error**: `Unable to determine UI_PORT`

**Solution**:
```bash
# Check scenario status
vrooli scenario status your-scenario

# Start if not running
vrooli scenario start your-scenario

# Verify it's healthy
vrooli scenario port your-scenario UI_PORT
```

### Low Scores

**Issue**: Scores lower than expected

**Debug Steps**:
1. Open HTML report and review suggestions
2. Check if using dev build instead of production
3. Verify assets are optimized (images compressed, JS minified)
4. Review Network tab in Lighthouse report
5. Check for console errors

**Common Fixes**:
```bash
# Use production build
npm run build

# Compress images
# Minify JavaScript
# Enable gzip compression
# Lazy load images
# Remove unused CSS
```

### Flaky Scores

**Issue**: Scores vary significantly between runs

**Solutions**:
```json
{
  "global_options": {
    "lighthouse": {
      "settings": {
        "throttlingMethod": "simulate",
        "throttling": {
          "cpuSlowdownMultiplier": 4
        }
      }
    },
    "retries": 2
  }
}
```

### Permission Denied

**Error**: `EACCES: permission denied`

**Solution**:
```bash
# Add required Chrome flags
"chrome_flags": ["--headless", "--no-sandbox", "--disable-setuid-sandbox"]
```

---

## 📊 Interpreting Results

### HTML Reports

Open `test/artifacts/lighthouse/page-id_timestamp.html` for:
- Overall scores by category
- Performance metrics (FCP, LCP, TBT, CLS, SI)
- Accessibility issues with remediation steps
- Best practice violations
- SEO recommendations
- Network waterfall
- JavaScript execution timeline

### JSON Reports

Programmatically parse `test/artifacts/lighthouse/page-id_timestamp.json`:

```javascript
const report = require('./test/artifacts/lighthouse/home_1234567890.json');

console.log('Performance:', report.categories.performance.score);
console.log('Accessibility:', report.categories.accessibility.score);
console.log('FCP:', report.audits['first-contentful-paint'].displayValue);
console.log('LCP:', report.audits['largest-contentful-paint'].displayValue);
```

### Phase Results

Check `coverage/phase-results/lighthouse.json` for:
- Summary of all pages tested
- Pass/fail status per page
- Requirement evidence for auto-sync

---

## 🚀 Advanced Usage

### Custom Lighthouse Configuration

Create `.vrooli/custom-config.js`:

```javascript
module.exports = {
  extends: 'lighthouse:default',
  settings: {
    onlyAudits: [
      'first-contentful-paint',
      'largest-contentful-paint',
      'cumulative-layout-shift',
      'total-blocking-time'
    ],
    // Custom performance budgets
    budgets: [{
      resourceSizes: [
        { resourceType: 'script', budget: 300 },
        { resourceType: 'image', budget: 400 }
      ]
    }]
  }
};
```

### Programmatic Execution

```javascript
const { lighthouse } = require('./scripts/scenarios/testing/lighthouse/runner');

async function runCustomAudit() {
  const config = require('./scenarios/my-scenario/.vrooli/lighthouse.json');
  const results = await lighthouse.runForConfig(config, 'http://localhost:3000');
  console.log(results);
}
```

### CI/CD Integration

```yaml
# .github/workflows/lighthouse.yml
name: Lighthouse CI
on: [push, pull_request]

jobs:
  lighthouse:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'

      - name: Install dependencies
        run: |
          cd scripts/scenarios/testing/lighthouse
          npm install

      - name: Start scenario
        run: vrooli scenario start browser-automation-studio

      - name: Run Lighthouse tests
        run: |
          cd scenarios/browser-automation-studio
          ./test/run-tests.sh --phases performance

      - name: Upload reports
        uses: actions/upload-artifact@v3
        with:
          name: lighthouse-reports
          path: scenarios/browser-automation-studio/test/artifacts/lighthouse/
```

---

## 📚 Resources

- [Lighthouse Documentation](https://developer.chrome.com/docs/lighthouse/overview/)
- [Web Vitals](https://web.dev/vitals/)
- [Accessibility Guidelines (WCAG)](https://www.w3.org/WAI/WCAG21/quickref/)
- [Performance Budgets](https://web.dev/performance-budgets-101/)
- [Lighthouse Scoring](https://developer.chrome.com/docs/lighthouse/performance/performance-scoring/)

---

## 🆘 Getting Help

- **Documentation**: `/scripts/scenarios/testing/lighthouse/README.md`
- **Examples**: See `browser-automation-studio` (once Phase 2 complete)
- **Issues**: Check `CLAUDE.md` for common patterns
- **Testing Guide**: `/docs/testing/guides/scenario-testing.md`

---

**Next**: Proceed to Phase 2 - Implement Lighthouse config for browser-automation-studio pilot scenario.
