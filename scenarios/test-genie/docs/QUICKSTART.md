# Test Genie Quick Start

Welcome to Test Genie - the comprehensive testing platform for Vrooli scenarios and resources.

## What is Test Genie?

Test Genie is a Go-native testing orchestration platform that:

- **Runs tests** - Execute multi-phase test suites with configurable presets
- **Tracks coverage** - Monitor test health across all scenarios
- **Syncs requirements** - Automatically track `[REQ:ID]` tags from tests
- **Provides APIs** - REST and CLI interfaces for agent automation

## Quick Start

### 1. Start Test Genie

```bash
cd scenarios/test-genie
make start
```

Or via CLI:
```bash
vrooli scenario start test-genie
```

### 2. Run Tests for a Scenario

**Via CLI:**
```bash
test-genie execute my-scenario --preset comprehensive
```

**Via API:**
```bash
API_PORT=$(vrooli scenario port test-genie API_PORT)
curl -X POST "http://localhost:${API_PORT}/api/v1/test-suite/my-scenario/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"preset": "comprehensive"}'
```

**Via Dashboard:**
1. Open `http://localhost:${UI_PORT}`
2. Select a scenario from the Catalog
3. Click "Run Tests"

### 3. View Results

Results are available via:
- **Dashboard** - Visual results with phase breakdowns
- **API** - `GET /api/v1/executions/{id}`
- **CLI** - `test-genie status --executions`

## Test Presets

| Preset | Phases | Time | Use Case |
|--------|--------|------|----------|
| **Quick** | Structure, Standards, Docs, Business, Unit, Proto | Scenario-aware | Fast sanity check |
| **Smoke** | Structure, Standards, Quality, Docs, Business, Integration, Proto | Scenario-aware | Pre-push validation |
| **Architecture Audit** | Structure, Contracts, UI Health, Docs, Standards, Architecture, Proto | Scenario-aware | Surface and architecture review |
| **Comprehensive** | All catalog phases | Scenario-aware | Full coverage |

See [Presets Reference](reference/presets.md) for details.

## Test Phases

Test Genie uses a catalog-driven testing architecture:

```
Structure → Contracts → UI Health → Standards → Architecture → Dependencies → Quality → Docs → Performance → Smoke → Unit → Integration → Playbooks → Business → Coverage → Tidiness → Security → Measures → Proto
```

| Phase | Purpose | Timeout |
|-------|---------|---------|
| **Structure** | Validate files and config | 15s |
| **Contracts** | Validate CLI manifest bindings | 60s |
| **UI Health** | Validate UI manifest bindings | 60s |
| **Standards** | scenario-auditor standards rules | 60s |
| **Architecture** | Structural cohesion audit | 120s |
| **Dependencies** | Check tools and resources | 30s |
| **Quality** | Static quality contracts via quality-health | 120s |
| **Docs** | Markdown, mermaid, link validation | 60s |
| **Performance** | Run benchmarks (optional) | 60s |
| **Smoke** | UI load + iframe-bridge | 90s |
| **Unit** | Run unit tests (Go, Node, Python) | 60s |
| **Integration** | Test API/UI connectivity | 120s |
| **Playbooks** | Execute BAS browser automation workflows | 120s |
| **Business** | Validate workflows and rules | 180s |
| **Coverage** | Parse coverage artifacts | 30s |
| **Tidiness** | File/function quality checks | 120s |
| **Security** | Security posture validation | 180s |
| **Measures** | Measures coverage validation | 180s |
| **Proto** | Protocol Buffer contract validation | 120s |

See [Phased Testing Guide](guides/phased-testing.md) for the complete architecture.

## Documentation Navigation

### Getting Started
- [Glossary](GLOSSARY.md) - Testing terminology and definitions

### Safety (Read First!)
- [Safety Guidelines](safety/GUIDELINES.md) - **CRITICAL** - Prevent data loss in test scripts

### Phase Documentation
- [Phases Overview](phases/README.md) - phase architecture with mermaid diagrams
- [Structure Phase](phases/structure/README.md) - File and CLI validation
- [Standards Phase](phases/standards/README.md) - Standards enforcement via scenario-auditor
- [Dependencies Phase](phases/dependencies/README.md) - Runtime and tool checks
- [Quality Phase](phases/quality/README.md) - Static quality contracts
- [Docs Phase](phases/docs/README.md) - Markdown, mermaid, and link validation
- [Unit Phase](phases/unit/README.md) - Test runners and coverage
- [Integration Phase](phases/integration/README.md) - CLI and API testing
- [Workflow Phase](phases/workflow/README.md) - BAS workflow validation through workflow-health
- [Business Phase](phases/business/README.md) - Requirements validation
- [Performance Phase](phases/performance/README.md) - Build benchmarks and Lighthouse

### Guides (How-To)
- [Phased Testing](guides/phased-testing.md) - Understanding the catalog-driven phase architecture
- [Test Generation](guides/test-generation.md) - AI-powered test creation
- [Requirements Sync](phases/business/requirements-sync.md) - Automatic requirement tracking
- [Scenario Unit Testing](phases/unit/scenario-unit-testing.md) - Go, Node, Python unit tests
- [UI Testability](guides/ui-testability.md) - Design testable UIs
- [Workflow Phase](phases/workflow/README.md) - Vrooli Ascension workflow validation
- [UI Smoke Testing](phases/structure/ui-smoke.md) - Fast UI validation on Browser Automation Studio
- [Lighthouse Integration](phases/performance/lighthouse.md) - Performance and accessibility testing
- [Performance Testing](phases/performance/performance-testing.md) - Build benchmarks and regression detection
- [CLI Manifest Contract](phases/structure/cli-approaches.md) - Manifest-driven scenario CLI adapter patterns
- [Custom Presets](guides/custom-presets.md) - Create tailored test presets for CI/CD
- [Dashboard Guide](guides/dashboard-guide.md) - Using the web dashboard UI
- [Vault Testing](guides/vault-testing.md) - Multi-phase lifecycle validation
- [Sync Execution](guides/sync-execution.md) - Blocking execution for agents
- [Validation Best Practices](guides/validation-best-practices.md) - Quality validation guidelines
- [End-to-End Example](guides/end-to-end-example.md) - Complete PRD to coverage walkthrough
- [Troubleshooting](guides/troubleshooting.md) - Debug and fix common issues

### Reference (Technical Details)
- [API Endpoints](reference/api-endpoints.md) - REST API reference
- [CLI Commands](reference/cli-commands.md) - test-genie CLI reference
- [Presets](reference/presets.md) - Quick/Smoke/Comprehensive definitions
- [Test Runners](phases/unit/test-runners.md) - Language-specific runners
- [Requirement Schema](reference/requirement-schema.md) - JSON schema for requirements
- [Gaming Prevention](reference/gaming-prevention.md) - Test integrity detection
- [Gold Standard Examples](reference/examples.md) - Exemplary implementations

### Concepts (Architecture)
- [Architecture](concepts/architecture.md) - Go orchestrator design
- [Requirement Flow](concepts/requirement-flow.md) - End-to-end requirement validation
- [Testing Strategy](concepts/strategy.md) - Three-layer validation approach
- [Infrastructure](concepts/infrastructure.md) - Testing tools and frameworks

## Common Tasks

### Generate Tests for a New Scenario

```bash
test-genie generate my-scenario --types unit,integration
```

See [Test Generation Guide](guides/test-generation.md).

### Track Requirements from Tests

Add `[REQ:ID]` tags to your tests:

```go
func TestCreateProject(t *testing.T) {
    t.Run("creates project [REQ:MY-PROJECT-CREATE]", func(t *testing.T) {
        // test code
    })
}
```

Run comprehensive tests to auto-sync:
```bash
test-genie execute my-scenario --preset comprehensive
```

See [Requirements Sync Guide](phases/business/requirements-sync.md).

### Use Sync Execution for CI/Agents

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/test-suite/my-scenario/execute-sync" \
  -H "Content-Type: application/json" \
  -d '{"preset": "smoke", "failFast": true}'
```

Returns complete results in a single blocking request.

See [Sync Execution Guide](guides/sync-execution.md) and [Cheatsheet](reference/sync-execution-cheatsheet.md).

## Troubleshooting

### Tests Fail with "scenario not found"

Ensure the scenario exists:
```bash
vrooli scenario list | grep my-scenario
```

### Phase Times Out

Increase timeout in `.vrooli/testing.json`:
```json
{
  "phases": {
    "unit": { "timeout": 120 }
  }
}
```

### Requirements Not Syncing

Ensure tests have `[REQ:ID]` tags and run with comprehensive preset.

**For more help:** See the [Troubleshooting Guide](guides/troubleshooting.md) for comprehensive debugging help.

See [PROBLEMS.md](PROBLEMS.md) for known issues.

## Quick Decision Tree

Use this guide to find the right documentation for your task:

```
START: What are you testing?

+-- New to Testing?
|   +-- Read: QUICKSTART (this doc) + Safety Guidelines
|
+-- Complete Scenario/App?
|   +-- First time --> Scenario Unit Testing Guide
|   +-- Adding tests --> Scenario Unit Testing Guide
|   +-- Complex multi-component --> Phased Testing Guide
|
+-- Vrooli Resource?
|   +-- Resource functions/CLI --> Resource Unit Testing Guide
|   +-- Resource integration --> Testing Strategy
|   +-- Cross-resource workflows --> Integration Testing
|
+-- Specific Code Type?
|   +-- Go API handlers --> Scenario Unit Testing Guide
|   +-- Node.js/React UI --> Scenario Unit Testing Guide
|   +-- Python scripts --> Scenario Unit Testing Guide
|   +-- Shell scripts --> CLI Testing Guide + Safety Guidelines
|   +-- Database migrations --> Testing Strategy + Integration
|
+-- UI Testing?
|   +-- Browser automation --> Workflow Phase
|   +-- Smoke tests --> UI Smoke Testing
|   +-- Performance --> Lighthouse Integration
|   +-- Design for testing --> UI Testability Guide
|
+-- Performance/Scale Issues?
|   +-- Tests too slow --> Troubleshooting Guide
|   +-- Memory/CPU usage --> Performance Testing
|   +-- Load testing --> Integration Testing + Performance
|
+-- CI/CD Integration?
|   +-- Agent automation --> Sync Execution Guide
|   +-- Test automation --> Test Runners Reference
|   +-- Build pipeline issues --> Troubleshooting Guide
|
+-- Safety Concerns?
|   +-- Tests deleting files --> Safety Guidelines (URGENT)
|   +-- Script safety --> Safety Guidelines + Linter
|
+-- Debugging/Issues?
|   +-- Tests failing --> Troubleshooting Guide
|   +-- Coverage too low --> Examples + Unit Testing
|   +-- Flaky tests --> Troubleshooting Guide
|   +-- Can't find right docs --> This decision tree!
|
+-- Learning/Examples?
    +-- See working examples --> Gold Standard Examples
    +-- Understand architecture --> Architecture Concepts
    +-- Deep dive into tools --> Shell Libraries + Test Runners
```

## Critical Warnings

1. **NEVER** use unguarded `rm` commands in test scripts
2. **ALWAYS** validate variables before file operations
3. **SET** critical variables before any early-exit conditions in shell scripts
4. **PREFER** Go tests over bash scripts for new development
5. **RUN** the safety linter before committing any shell test scripts

See [Safety Guidelines](safety/GUIDELINES.md) for complete safety rules.

## Next Steps

1. **New to testing?** Start with [Phased Testing Guide](guides/phased-testing.md)
2. **Writing unit tests?** See [Scenario Unit Testing](phases/unit/scenario-unit-testing.md)
3. **Building CI pipelines?** Check [Sync Execution Guide](guides/sync-execution.md)
4. **Designing testable UIs?** Read [UI Testability](guides/ui-testability.md)
