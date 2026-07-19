# Scenario Auditor

> Comprehensive standards enforcement system for Vrooli scenarios

The Scenario Auditor is Vrooli's permanent quality gatekeeper, ensuring every scenario maintains consistent, high-quality patterns that compound across the entire ecosystem. It provides unified auditing of API security, configuration compliance, UI testing practices, and development standards.

## ✨ Features

### 🛡️ Comprehensive Standards Enforcement
- **API Standards**: Go best practices, security patterns, documentation requirements
- **Configuration Standards**: service.json schema compliance, lifecycle completeness  
- **UI Standards**: BAS/Playwright testing practices, accessibility, performance
- **Testing Standards**: Phase-based structure, coverage requirements, integration patterns

### 🔧 Rule Engine
- **Modular YAML Rules**: Easy to read, write, and maintain
- **Toggleable Preferences**: Enable/disable rules with persistent storage
- **Category Organization**: Rules grouped by api, cli, test, makefile (structure/config/ui rules migrated to the `structure-health` scenario)
- **AI-Powered Creation**: Generate and edit rules using natural language

### 📊 Quality Dashboard
- **Real-time Monitoring**: Live standards compliance tracking
- **Violation Analysis**: Detailed breakdown by category and severity
- **Quality Scoring**: 0-100 score based on violations and importance
- **Trend Analysis**: Historical compliance data and improvements

### 🤖 AI Integration
- **Automated Fixes**: Claude Code agent spawning for violation fixes
- **Rule Creation**: Create new rules from natural language descriptions via `/api/v1/rules/create`
- **Rule Editing**: AI-assisted modification of existing rules via `/api/v1/rules/ai/edit/{ruleId}`
- **Intelligent Analysis**: Smart categorization and prioritization

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+ (for UI)
- Git
- curl, jq (for testing)

### Installation
```bash
# Full setup
make setup

# Or step by step
make install  # Install dependencies
make build    # Build binaries
```

### Usage
```bash
# Start the auditor
make run

# Access the dashboard
open http://localhost:35001

# Use the CLI
scenario-auditor standards scan current --wait
scenario-auditor rules list --category api
scenario-auditor fixes status
scenario-auditor standards violations --scenario browser-automation-studio
```

## 📋 CLI Commands

The CLI is a lightweight wrapper around the scenario-auditor API. Make sure the scenario is running before using CLI commands:

```bash
# Start the scenario first
vrooli scenario start scenario-auditor

# Then use CLI commands
scenario-auditor standards scan [scenario] [--rule <rule_id>] [--wait] [--timeout <duration>]
scenario-auditor rules list          # List available rules
scenario-auditor status              # Check API health
scenario-auditor version             # Show version
scenario-auditor help                # Show help
```

**Note**: The CLI now uses async job-based scanning. Scans run in the background and the CLI polls for completion. Large scenarios may take 20-30 seconds to scan completely.

`scenario-auditor standards scan --wait` is the main agent/operator flow: it starts a standards job, polls until completion, and returns the final job status. Use `scenario-auditor standards summary <job-id>` when you need the curated summary payload with severity counts, top violations, and recommended steps. Pass `--json` to either command for machine-readable output.

### Summary API & Artifacts

- `GET /api/v1/scenarios/scan/jobs/{id}/summary` and `GET /api/v1/standards/check/jobs/{id}/summary` expose the same actionable payload that the CLI renders (total counts, severity distribution, rule aggregates, top violations, recommended steps). Query parameters:
  - `limit=<n>` – cap the `top_violations` array (default 20, max buffered 200)
  - `min_severity=<critical|high|medium|low|info>` – filter noisy entries when thousands of low-level warnings exist
  - `group_by=rule` – include an aggregated `groups` array keyed by rule_id for quick remediation planning
- `GET /api/v1/standards/violations/summary?scenario=<name>` returns the cached summary for the latest standards scan of a scenario (or the fleet-wide `all` bucket when omitted). It accepts the same filtering parameters so dashboards like app-monitor can present prioritized results without rerunning a scan.
- `GET /api/v1/scenarios/scan/jobs/{id}/artifact` (and `/standards/.../artifact`) streams the archived JSON blob persisted under `logs/scenario-auditor/<type>/<scenario>/…`. Artifacts stay on disk for 14 days and the CLI’s `--download-artifacts <dir>` flag simply mirrors these endpoints locally.
- Consumers that still need the full payload can fetch the scan artifact endpoint directly after the job completes; everyone else should rely on the lightweight summary for faster feedback loops.

### Port Detection

The CLI automatically detects the correct API port using this precedence:

1. `SCENARIO_AUDITOR_API_PORT` environment variable (scenario-specific)
2. Auto-detection from running `scenario-auditor-api` process (most reliable)
3. `API_PORT` environment variable (only if in valid range 15000-19999)
4. Default port `18507`

This ensures the CLI works correctly even when multiple scenarios are running with different port assignments.

### Examples
```bash
# Scan current scenario
scenario-auditor standards scan scenario-auditor --wait

# Target a specific rule and wait for completion (recommended for fixes)
scenario-auditor standards scan swarm-manager --rule health_check --wait --timeout 10m

# Run a full scan only if you need the complete violation list
scenario-auditor standards scan swarm-manager

# List available rules
scenario-auditor rules list

# Check API health
scenario-auditor status

# Show version
scenario-auditor version

# List a scenario's violations after the scan job completes
scenario-auditor standards violations --scenario swarm-manager

```

### Targeted Rule Validation

The CLI supports targeted validations so agents (or humans) can re-check a single rule without triggering the entire standards catalog:

- Use `--rule <rule_id>` (repeatable) or `--rules <id1,id2>` to restrict the scan.
- Combine with `--wait` to stream job status and `--timeout <seconds>` (default 600) to bound execution time.
- Prefer targeted scans when closing issues; fall back to a full scan only when you need the global violation list.

Example:

```bash
# Re-run a targeted scan and surface the resulting job details
scenario-auditor scan agent-dashboard --rule service_json_ports --wait --timeout 600
```

## 🏗️ Architecture

### Core Components
- **Rule Engine**: Executes configurable quality rules against scenario files
- **Standards Scanner**: Validates scenarios against established best practices
- **AI Integration**: Generates new rules and fixes violations automatically
- **Preferences Manager**: Maintains user-configurable rule toggles
- **Dashboard Interface**: Provides comprehensive standards management UI

### Rule Categories
1. **API Standards** (`api/rules/api/`): Go best practices, security patterns
2. **CLI Standards** (`api/rules/cli/`): CLI conventions and output patterns
3. **Testing Standards** (`api/rules/test/`): Phase-based testing structure
4. **Makefile Standards** (`api/rules/makefile/`): Makefile conventions and targets

> **Note**: The structure, config, and UI rule packs (e.g. `required_layout`, `env_validation`, `service_ports`, `helmet`, `focus_visible`, `spatial_nav`) have been migrated to the **`structure-health`** scenario, which now backs test-genie's Structure phase. Scenario Auditor retains only the cross-cutting policy/content standards rules: api, cli, test, makefile.

### Technology Stack
- **Backend**: Go with Gorilla Mux, PostgreSQL
- **Frontend**: React + TypeScript + Tailwind CSS
- **CLI**: Go with structured output
- **Rules**: YAML-based definitions
- **AI**: Claude Code agent integration

## 📊 Standards Categories

### service.json Validation
- ✅ Schema compliance with project-level schema
- ✅ Lifecycle completeness (setup, develop, test, stop)
- ✅ Binary naming conventions (`<scenario>-api`, `<scenario>`)
- ✅ Health check configuration (API + UI endpoints)
- ✅ Required step ordering and presence
- ✅ Develop lifecycle includes start-api/start-ui conventions
- ✅ Ports configuration enforces API_PORT 15000-19999 and UI_PORT 35000-39999 ranges
- ✅ Test lifecycle uses the Go orchestrator via test-genie (`vrooli scenario test <scenario>`)
- ✅ Setup steps cover scenario-specific build-api plus required UI bundle work
- ✅ Setup conditions ensure binaries and CLI checks match the scenario name
- ✅ Lifecycle.health config enforces /health endpoints and http checks

### UI Testing Best Practices
- ✅ Unique element IDs for reliable testing
- ✅ data-testid attributes for interactive elements
- ✅ Semantic HTML for accessibility
- ✅ Loading state indicators
- ✅ Error handling display
- ✅ Consistent naming conventions

### Iframe Bridge Integration
- Add `@vrooli/iframe-bridge` to the UI package dependencies using a relative `file:` path.
- In the UI entry file, import `initIframeBridgeChild` from the shared package, guard with `window.parent !== window`, and initialize it with an `appId` so orchestration can identify the UI.
- Scenarios that previously vendored `iframeBridgeChild.ts` should remove the copy and rely on the shared package instead.

### Package Governance Enforcement
- Shared-package adoption policy is enforced through the external `PACKAGE_GOVERNANCE_SCENARIO_ADOPTION` rule provided by `scenario-stack-governor`.
- That rule delegates to `vrooli package audit --all`, so scenario-auditor stays aligned with the canonical package-governance engine instead of re-implementing package policy locally.
- This covers manifest-backed package adoption, forbidden workspace-star usage in real scenarios, and unauthorized shared-package propagation hacks.
- Canonical documentation lives in [docs/package-governance.md](/home/matthalloran8/Vrooli/docs/package-governance.md:1) and [scenario-stack-governor/README.md](/home/matthalloran8/Vrooli/scenarios/scenario-stack-governor/README.md:21).

### Phase-Based Testing
- ✅ test/phases/ directory structure
- ✅ Unit, integration, business, dependencies tests
- ✅ Performance and structure validation
- ✅ Test artifacts organization
- ✅ Executable test scripts

### API Security & Standards
- ✅ Secure tunnel proxy enforces proxyToApi usage for tunnel deployments
- ✅ Go module configuration
- ✅ Health check endpoints
- ✅ CORS configuration
- ✅ Error handling patterns
- ✅ Structured logging
- ✅ Graceful shutdown
- ✅ Environment variable usage

## 🧪 Testing

### Running Tests
```bash
# Full test suite
make test

# Specific test phases
make test-unit          # Unit tests
make test-integration   # Integration tests
make test-structure     # Directory structure
make test-deps          # Dependencies check
make test-business      # Business logic
make test-performance   # Performance metrics
make test-rules-engine  # Rule engine validation (ADDED 2025-10-05)
make test-ui-practices  # UI practices enforcement (ADDED 2025-10-05)
```

### Test Structure
```
test/
├── phases/                # Phase-based tests
│   ├── test-unit.sh       # Unit tests
│   ├── test-integration.sh # Integration tests
│   ├── test-structure.sh   # Structure validation
│   ├── test-dependencies.sh # Dependency checks
│   ├── test-business.sh    # Business logic
│   ├── test-performance.sh # Performance tests
│   ├── test-rules-engine.sh # Rule engine functionality (ADDED 2025-10-05)
│   └── test-ui-practices.sh # UI best practices validation (ADDED 2025-10-05)
└── artifacts/             # Test outputs and logs
```

**Note**: The test suite now includes comprehensive rule engine and UI practices validation that matches the service.json lifecycle configuration.

## 🎯 Development

### Project Structure
```
scenario-auditor/
├── .vrooli/
│   └── service.json       # Lifecycle configuration
├── api/                   # Go API server
│   ├── main.go           # Server entry point
│   ├── handlers_*.go     # API handlers
│   ├── rules/        # Interpreted rule sources (organized by category)
│   │   ├── api/          # API standards
│   │   ├── cli/          # CLI standards
│   │   ├── test/         # Testing requirements
│   │   └── makefile/     # Makefile standards
│   │   # NOTE: structure/, config/, ui/ rule packs migrated to the structure-health scenario
│   └── storage/          # Data persistence
├── cli/                  # Command-line tool
│   ├── main.go          # CLI entry point
│   └── install.sh       # Installation script
├── ui/                 # React dashboard
│   ├── src/pages/      # Page components
│   ├── src/components/ # Reusable components
│   └── src/services/   # API client
├── test/               # Phase-based testing
└── docs/               # Documentation
```

### Adding New Rules
1. Create YAML rule definition in appropriate category folder
2. Include name, description, condition, and fix information
3. Test rule against sample scenarios
4. Document rule purpose and usage

Runtime signature: Go-based rules must expose an exported `Check` function with the shape `func (r *RuleStruct) Check(content string, filepath string, scenario string) ([]Violation, error)`. The `scenario` argument contains the scenario name during repository scans (and is empty for playground/unit tests), allowing rules to tailor messaging or behaviour per scenario while remaining deterministic.

### Rule Definition Format
```yaml
- id: config-service-json-exists
  name: "Service JSON File Exists"
  description: "Ensures that .vrooli/service.json file exists"
  category: config
  severity: critical
  enabled: true
  condition:
    type: file_exists
    target: ".vrooli/service.json"
  message: "Missing required .vrooli/service.json file"
  fix:
    type: file_create
    action: create
    target: ".vrooli/service.json"
    template: "service-json-template"
    description: "Create service.json from template"
  tags:
    - service-json
    - configuration
    - required
```

### Embedded Rule Tests
- Use `<test-case ...>` blocks inside Go rule files to describe sample inputs.
- Provide `path="relative/file.ext"` when a rule depends on filename or location (e.g., `.vrooli/service.json`, `api/main.go`). The test runner passes this path to the rule exactly as it would appear during a real scan.
- Optional attributes:
  - `scenario="name"` supplies the third argument to `Check`, letting rules verify scenario-specific logic.
  - `id`, `should-fail`, and `path`/`file`/`filepath`, plus the existing `<input>`, `<expected-violations>`, and `<expected-message>` sections remain supported.

## 🔗 Integration

### With Scenario Development
- Automatic validation during scenario creation
- Pre-commit hooks for standards checking
- CI/CD integration for quality gates
- Real-time feedback during development

### With Vrooli Ecosystem
- Cross-scenario learning and pattern recognition
- Standards evolution based on ecosystem feedback
- Integration with maintenance workflows
- Quality metrics for scenario assessment

## 📈 Quality Metrics

### Scoring System
- **100 points**: Perfect compliance
- **Critical violations**: -20 points each
- **High violations**: -10 points each
- **Medium violations**: -5 points each
- **Low violations**: -2 points each
- **Minimum score**: 0 points

### Categories Tracked
- Total violations by severity
- Auto-fixable violation count
- Files scanned and rules executed
- Compliance trends over time
- Category-specific scores

## 🤝 Contributing

### Adding Rules
1. Identify standards gap or new requirement
2. Write rule definition following existing patterns
3. Test rule against multiple scenarios
4. Submit PR with rule and documentation

### Improving Engine
1. Follow Go best practices
2. Maintain backward compatibility
3. Add comprehensive tests
4. Update documentation

## 📚 Resources

- [Product Requirements Document](PRD.md)
- [UI Testing Best Practices (BAS/Playwright)](../browser-automation-studio/README.md)
- [Phase-Based Testing Guide](test/README.md)
- [Rule Creation Documentation](docs/RULE_CREATION.md)

## 🔧 Troubleshooting

### Common Issues
- **API not starting**: Check port conflicts, run `make health`
- **Rules not loading**: Verify YAML syntax, check rule files
- **UI build failures**: Ensure Node.js deps installed
- **Test failures**: Check prerequisites, review logs in test/artifacts/

### Getting Help
- Check logs: `make logs`
- Run health check: `make health`
- View test artifacts: `ls test/artifacts/`
- CLI help: `scenario-auditor help`

---

**The Scenario Auditor becomes Vrooli's permanent quality gatekeeper - ensuring every scenario maintains the highest standards and contributes to the ecosystem's continuous improvement.**
