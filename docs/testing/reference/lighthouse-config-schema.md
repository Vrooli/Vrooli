# Lighthouse Configuration Schema Reference

Complete reference for `.vrooli/lighthouse.json` schema used by Vrooli's Lighthouse testing infrastructure.

## Overview

The Lighthouse configuration file defines which pages to audit, performance/accessibility thresholds, Chrome launch options, and reporting preferences for a scenario's UI testing.

**Schema Version**: 1.0.0
**Location**: `scenarios/<scenario-name>/.vrooli/lighthouse.json`
**Processed By**: `scripts/scenarios/testing/lighthouse/runner.js`

---

## Top-Level Structure

```json
{
  "_metadata": { /* ... */ },
  "enabled": true,
  "pages": [ /* ... */ ],
  "global_options": { /* ... */ },
  "reporting": { /* ... */ }
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `_metadata` | object | No | `{}` | File-level documentation and ownership |
| `enabled` | boolean | No | `true` | Whether Lighthouse testing is active for this scenario |
| `pages` | array | **Yes** | - | List of pages to audit (see [Pages Array](#pages-array)) |
| `global_options` | object | No | See defaults | Chrome and Lighthouse global settings |
| `reporting` | object | No | See defaults | Report output configuration |

---

## _metadata Object

Documentation and tracking information for the config file.

```json
{
  "_metadata": {
    "description": "Lighthouse testing configuration for browser-automation-studio",
    "last_updated": "2025-11-05T00:00:00Z",
    "owner": "browser-automation-studio team",
    "notes": "Thresholds set conservatively during pilot phase"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Human-readable description of this config |
| `last_updated` | ISO 8601 | When config was last modified |
| `owner` | string | Team or person responsible for maintaining thresholds |
| `notes` | string | Additional context about configuration decisions |

---

## Pages Array

Defines which routes/pages to audit with Lighthouse.

### Basic Page Example

```json
{
  "id": "dashboard",
  "path": "/",
  "label": "Dashboard Home",
  "viewport": "desktop",
  "thresholds": {
    "performance": { "error": 0.75, "warn": 0.85 },
    "accessibility": { "error": 0.90, "warn": 0.95 }
  },
  "requirements": ["BAS-PERF-DASHBOARD-LOAD"]
}
```

### Page Object Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | **Yes** | - | Unique page identifier (used in reports, file names) |
| `path` | string | **Yes** | - | URL path relative to scenario UI base (e.g., `/`, `/dashboard`, `/settings`) |
| `label` | string | **Yes** | - | Human-readable page name for reports |
| `viewport` | enum | No | `"desktop"` | `"desktop"` or `"mobile"` - determines screen emulation |
| `thresholds` | object | No | Global defaults | Per-page threshold overrides (see [Thresholds](#thresholds-object)) |
| `requirements` | array | No | `[]` | Requirement IDs linked to this page's performance/accessibility |
| `waitForSelector` | string | No | - | CSS selector to wait for before auditing (e.g., `"[data-testid='loaded']"`) |
| `waitForMs` | number | No | `0` | Additional milliseconds to wait after page load/selector |
| `notes` | string | No | - | Page-specific context for threshold decisions |

### Viewport Values

| Value | Screen Emulation | Use Case |
|-------|-----------------|----------|
| `"desktop"` | 1440x900, no mobile flags | Desktop/laptop experience |
| `"mobile"` | 360x640, mobile emulation | Smartphone experience |

**Note**: Mobile viewport automatically applies:
- `formFactor: "mobile"`
- `screenEmulation: { mobile: true, width: 360, height: 640 }`
- Throttling appropriate for mobile networks

---

## Thresholds Object

Defines pass/warn/fail criteria for Lighthouse categories.

```json
{
  "performance": { "error": 0.75, "warn": 0.85 },
  "accessibility": { "error": 0.90, "warn": 0.95 },
  "best-practices": { "error": 0.85, "warn": 0.90 },
  "seo": { "error": 0.80, "warn": 0.90 }
}
```

### Threshold Levels

Each category has two threshold levels:

| Level | Meaning | Behavior |
|-------|---------|----------|
| `error` | Hard failure threshold | Score below this **fails** the phase (if `fail_on_error: true`) |
| `warn` | Soft warning threshold | Score below this logs **warning** (fails if `fail_on_warn: true`) |

**Score Range**: 0.0 to 1.0 (Lighthouse scores are 0-100 internally, converted to 0-1 for config)

### Supported Categories

| Category | Description | Typical Threshold |
|----------|-------------|-------------------|
| `performance` | FCP, LCP, TTI, CLS, TBT metrics | `error: 0.75-0.85` |
| `accessibility` | WCAG compliance, ARIA, contrast | `error: 0.90` (P0) |
| `best-practices` | Security, HTTPS, console errors | `error: 0.85-0.90` |
| `seo` | Meta tags, structured data, robots | `error: 0.80-0.90` |

**Per-Page vs Global**:
- Pages can override global thresholds: `pages[].thresholds`
- Omitted categories use global defaults from `global_options.vrooli.settings`

### Threshold Best Practices

1. **Start Loose, Tighten Gradually**: Begin with `error: 0.70`, increase as optimizations land
2. **Use `warn` Level**: Set `warn: 0.10` above `error` to get early signals without blocking
3. **Different Thresholds by Page**:
   - Static pages (docs): `performance: 0.90`
   - Interactive SPAs (editors): `performance: 0.65-0.75`
   - Mobile pages: `-0.10` vs desktop due to throttling
4. **Accessibility is Binary**: Use `error: 0.90` minimum (WCAG 2.1 AA)

---

## global_options Object

Chrome launch settings and Lighthouse configuration applied to all pages.

```json
{
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
        "deviceScaleFactor": 1,
        "disabled": false
      }
    }
  },
  "chrome_flags": [
    "--headless",
    "--no-sandbox",
    "--disable-gpu"
  ],
  "timeout_ms": 60000,
  "retries": 2
}
```

### lighthouse Object

Direct pass-through to Lighthouse config. See [Lighthouse Configuration](https://github.com/GoogleChrome/lighthouse/blob/main/docs/configuration.md).

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `extends` | string | `"lighthouse:default"` | Base config to extend |
| `settings` | object | See example | Lighthouse settings object |

**Common settings**:
- `onlyCategories`: Array of categories to run (omit to run all)
- `throttlingMethod`: `"simulate"` (consistent) or `"devtools"` (real network)
- `formFactor`: `"desktop"` or `"mobile"` (overridden by page viewport)
- `screenEmulation`: Screen size and device pixel ratio

### chrome_flags Array

Chrome launch flags passed to `chrome-launcher`.

**Recommended Flags**:
```json
[
  "--headless",              // Headless mode (no UI)
  "--no-sandbox",            // Required for some CI/Docker environments
  "--disable-gpu",           // Disable GPU hardware acceleration
  "--disable-dev-shm-usage", // Overcome limited resource problems
  "--disable-extensions",    // Disable Chrome extensions
  "--disable-software-rasterizer" // Prevent software rendering issues
]
```

**Security Note**: `--no-sandbox` disables Chrome's sandbox for compatibility. Only use in trusted environments.

### timeout_ms

Maximum time (milliseconds) for a single page audit before timeout.

**Guidelines**:
- Simple pages: `30000` (30s)
- Complex SPAs: `60000` (60s)
- Heavy pages (video, large datasets): `90000` (90s)

### retries

Number of retry attempts if an audit fails due to transient issues (network, timeout).

**Default**: `2` (total 3 attempts: initial + 2 retries)

---

## reporting Object

Report generation and artifact management settings.

```json
{
  "output_dir": "test/artifacts/lighthouse",
  "formats": ["json", "html"],
  "keep_reports": 10,
  "fail_on_error": true,
  "fail_on_warn": false
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `output_dir` | string | `"test/artifacts/lighthouse"` | Directory for generated reports (relative to scenario root) |
| `formats` | array | `["json", "html"]` | Report formats to generate |
| `keep_reports` | number | `10` | Number of reports to retain per page (oldest auto-deleted) |
| `fail_on_error` | boolean | `true` | Exit with failure if any page has `error` threshold violations |
| `fail_on_warn` | boolean | `false` | Exit with failure if any page has `warn` threshold violations |

### formats Array

Supported report formats:

| Format | Description | File Extension | Use Case |
|--------|-------------|----------------|----------|
| `"json"` | Full Lighthouse JSON result | `.json` | Programmatic analysis, CI/CD parsing |
| `"html"` | Interactive HTML report | `.html` | Human review, detailed diagnostics |

**File Naming**: `{page_id}_{timestamp}.{ext}`
Example: `dashboard_1699200000000.html`

### Report Rotation

When `keep_reports` is exceeded:
1. Reports sorted by timestamp (oldest first)
2. Oldest reports deleted until count = `keep_reports`
3. Deletion logged to phase output

**Recommendation**: Set `keep_reports: 10` for local dev, `keep_reports: 3` for CI to save disk space.

---

## Complete Example

Full configuration for a complex scenario with multiple pages and viewports:

```json
{
  "_metadata": {
    "description": "Lighthouse testing for multi-page SaaS application",
    "last_updated": "2025-11-05T00:00:00Z",
    "owner": "frontend-team"
  },
  "enabled": true,
  "pages": [
    {
      "id": "landing",
      "path": "/",
      "label": "Marketing Landing Page",
      "viewport": "desktop",
      "thresholds": {
        "performance": { "error": 0.90, "warn": 0.95 },
        "accessibility": { "error": 0.95, "warn": 0.98 },
        "seo": { "error": 0.90, "warn": 0.95 }
      },
      "requirements": ["APP-PERF-LANDING", "APP-SEO-HOMEPAGE"],
      "notes": "Static page - strict thresholds"
    },
    {
      "id": "dashboard",
      "path": "/dashboard",
      "label": "User Dashboard",
      "viewport": "desktop",
      "waitForSelector": "[data-testid='dashboard-loaded']",
      "waitForMs": 1000,
      "thresholds": {
        "performance": { "error": 0.75, "warn": 0.85 },
        "accessibility": { "error": 0.90, "warn": 0.95 }
      },
      "requirements": ["APP-PERF-DASHBOARD-INTERACTIVE"]
    },
    {
      "id": "editor",
      "path": "/editor/new",
      "label": "Rich Text Editor",
      "viewport": "desktop",
      "waitForSelector": ".editor-loaded",
      "waitForMs": 2000,
      "thresholds": {
        "performance": { "error": 0.65, "warn": 0.75 },
        "accessibility": { "error": 0.90, "warn": 0.95 }
      },
      "requirements": ["APP-PERF-EDITOR-TTI"],
      "notes": "Heavy Monaco/TinyMCE - lower threshold acceptable"
    },
    {
      "id": "mobile-landing",
      "path": "/",
      "label": "Landing Page (Mobile)",
      "viewport": "mobile",
      "thresholds": {
        "performance": { "error": 0.80, "warn": 0.90 },
        "accessibility": { "error": 0.95, "warn": 0.98 }
      },
      "requirements": ["APP-PERF-MOBILE-LANDING"]
    }
  ],
  "global_options": {
    "lighthouse": {
      "extends": "lighthouse:default",
      "settings": {
        "onlyCategories": ["performance", "accessibility", "best-practices", "seo"],
        "throttlingMethod": "simulate",
        "throttling": {
          "rttMs": 40,
          "throughputKbps": 10240,
          "cpuSlowdownMultiplier": 1
        }
      }
    },
    "chrome_flags": [
      "--headless",
      "--no-sandbox",
      "--disable-gpu",
      "--disable-dev-shm-usage"
    ],
    "timeout_ms": 90000,
    "retries": 2
  },
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

## Requirements Integration

Lighthouse results automatically sync to requirements system via phase results.

### Linking Pages to Requirements

```json
{
  "pages": [
    {
      "id": "dashboard",
      "requirements": ["BAS-PERF-DASHBOARD-LOAD", "BAS-A11Y-DASHBOARD"]
    }
  ]
}
```

### Requirement Definition Example

```json
{
  "id": "BAS-PERF-DASHBOARD-LOAD",
  "title": "Dashboard loads with acceptable performance",
  "criticality": "P0",
  "validation": [
    {
      "type": "lighthouse",
      "ref": ".vrooli/lighthouse.json",
      "page_id": "dashboard",
      "category": "performance",
      "threshold": 0.75,
      "phase": "performance",
      "status": "implemented"
    }
  ]
}
```

### Evidence Collection

After Lighthouse runs, phase results include:

```json
{
  "phase": "performance",
  "requirements": [
    {
      "id": "BAS-PERF-DASHBOARD-LOAD",
      "status": "passed",
      "evidence": "Lighthouse: Dashboard Home - Performance: 0.87 (threshold: 0.75)",
      "updated_at": "2025-11-05T14:32:15Z"
    }
  ]
}
```

**Auto-Sync**: Run `node scripts/requirements/report.js --scenario <name> --mode sync` to update requirement files.

---

## Validation

Validate your config against the schema:

```bash
# Check config syntax
cat .vrooli/lighthouse.json | jq empty

# Dry-run Lighthouse (doesn't update requirements)
node scripts/scenarios/testing/lighthouse/runner.js \
  --config .vrooli/lighthouse.json \
  --base-url http://localhost:5050 \
  --output-dir /tmp/lighthouse-test \
  --scenario test-scenario \
  --dry-run
```

---

## Troubleshooting

### Common Issues

**Chrome fails to launch**:
- Ensure Chrome/Chromium installed: `which google-chrome chromium`
- Add `--disable-dev-shm-usage` to `chrome_flags`
- Check Docker `/dev/shm` size: `df -h /dev/shm` (should be >64MB)

**Timeout errors**:
- Increase `timeout_ms` (try `90000`)
- Check `waitForSelector` is correct (typo in selector?)
- Ensure scenario is running: `vrooli scenario status <name>`

**Low performance scores**:
- Use production build: `pnpm build` (not `pnpm dev`)
- Disable React DevTools in test environment
- Check network throttling settings (may be too aggressive)

**Inconsistent scores**:
- Use `throttlingMethod: "simulate"` (not `"devtools"`)
- Increase `waitForMs` for dynamic content
- Check for background timers/animations affecting metrics

## See Also

### Documentation
- [Lighthouse Integration Guide](../guides/lighthouse-integration.md) - Complete setup and usage guide
- [Requirement Tracking Guide](../guides/requirement-tracking.md) - Linking audits to requirements
- [Phased Testing Architecture](../guides/scenario-testing.md) - Test phase system overview

### Tools
- [scripts/scenarios/testing/lighthouse/runner.js](/scripts/scenarios/testing/lighthouse/runner.js) - Node.js executor
- [scripts/scenarios/testing/lighthouse/runner.sh](/scripts/scenarios/testing/lighthouse/runner.sh) - Bash orchestration
- [scripts/scenarios/testing/lighthouse/config.sh](/scripts/scenarios/testing/lighthouse/config.sh) - Helper utilities

### Examples
- [browser-automation-studio/.vrooli/](/scenarios/browser-automation-studio/.vrooli/) - 4-page reference configuration
- [app-monitor/.vrooli/](/scenarios/app-monitor/.vrooli/) - Simple 2-page setup
- [app-issue-tracker/.vrooli/](/scenarios/app-issue-tracker/.vrooli/) - Table-focused testing

### External Resources
- [Lighthouse Configuration](https://github.com/GoogleChrome/lighthouse/blob/main/docs/configuration.md) - Official config documentation
- [Lighthouse Scoring Guide](https://github.com/GoogleChrome/lighthouse/blob/main/docs/scoring.md) - Understanding scores
