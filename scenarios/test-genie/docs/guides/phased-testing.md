# Phased Testing Architecture

This guide explains Vrooli's comprehensive phased testing architecture and how to use Test Genie for scenario validation.

## Overview

Vrooli uses a **descriptor-backed testing architecture** that progressively validates scenarios from basic structure through provider-backed health phases. Test Genie orchestrates planning, applicability, provider readiness, lifecycle, run records, and reporting; each health provider owns its phase metadata and maturity contract in `scenarios/<provider>/.vrooli/test-genie.json`.

## The Phase Architecture

The effective phase registry is built from checked-in provider descriptors plus Test Genie-owned runner bindings. A run follows this order:

1. Load provider descriptors from `scenarios/*/.vrooli/test-genie.json`.
2. Evaluate target applicability from cheap declarative facts such as files, capabilities, UI/API presence, and testing config sections.
3. Select phases from the requested preset or explicit phase list against applicable phases.
4. Check or start selected providers according to descriptor policy.
5. Start target runtime surfaces only when selected phases need them.
6. Apply runnability gates for the current environment.
7. Execute provider validations and record findings, observations, metrics, and phase status.

Applicability answers "should this phase judge this target?" Runnability answers "can this already-applicable phase execute in this environment?" Provider readiness answers "is the selected provider usable for this execution?" These are separate states in JSON previews and run records.

### How provider readiness is established

Readiness asks three questions, none of whose answers depend on the target: is the provider live, does it speak the current contract, and is it the provider its descriptor claims. It answers them with `ScenarioValidationService.DescribeProvider`, whose request carries no target fields at all — the provider replies from facts it resolved at startup, so the probe is O(1) regardless of how large the scenario under test is.

Providers that have not adopted `DescribeProvider` return `Unimplemented`, and only then does readiness fall back to a `ValidateScenario` call with `include_execution=false`. That fallback answers the same three questions but pays a full target analysis to do so: for an inspection-only provider such as `security-health` or `architecture-cartographer`, `include_execution=false` and `true` mean the same thing, so the provider's entire phase workload ran twice per suite — once to prove readiness and once to produce the result. Adopting `DescribeProvider` is what removes that duplicate.

Mount the shared implementation with `assessment.Serve(handler, describer)` (see `packages/maturity-go/assessment/describe.go`). The zero `Describer` is safe: it reports `Unimplemented`, which selects the fallback, so adoption is per-provider rather than a fleet-wide flag day.

### Provider staleness

A provider that answers readiness can still be running code that no longer matches the repository — a long-lived process never re-checks itself, so it can serve findings from a binary built weeks earlier. Readiness therefore also asks whether the running provider is current, using two exact digest comparisons and no git state at all (a commit changes no file content, so a design keyed to `HEAD` would mark every provider stale on every commit):

1. **Rebuilt but not restarted** — the provider reports the freshness digest it read at startup; a different digest on disk means the live process is superseded.
2. **Source changed since the build** — the ordinary freshness comparison, evaluated against the same manifest the provider's own preflight uses.

When a provider is stale, readiness restarts it, which makes its preflight rebuild and re-exec so the phase scores against current source. Findings name what kind of change caused it, because the consequences differ sharply:

| Class | Meaning |
|---|---|
| `own_code` | the provider's own source changed — its verdict is most likely to be wrong |
| `shared_package` | a package it compiles changed — usually incidental |
| `dependency` | `go.mod` / `go.sum` moved |
| `toolchain` | Go version, arch, or cgo changed — hits every provider at once |
| `rebuilt_not_restarted` | the binary moved underneath the running process |

The provider's own tree is evaluated **first**, as its own subset. The underlying comparison stops at the first offending path in alphabetical order, so a change under `packages/` would otherwise mask a simultaneous change to the provider's own code.

Four rails bound the cost, because on an active branch staleness is the normal state rather than the exception — one edit to a widely shared package legitimately stales most of the fleet:

- **Per-run cap** (`DefaultMaxStaleRestarts`, 4). Past it, providers are reported rather than restarted, so a run always finishes. Set `MaxStaleRestarts` negative for report-only mode.
- **Cool-down** (`DefaultRestartCooldown`, 30m), persisted across runs. It suppresses repeat restarts caused by churn **outside** the provider's own tree, so an agent editing a shared package for hours does not trigger a rebuild every run. It never applies to `own_code` or `rebuilt_not_restarted`.
- **No cascade.** Only the provider itself is restarted; its own preflight decides about its dependencies.
- **Fail open everywhere.** No repo root, no manifest, an unreadable manifest, an empty reported digest, or any evaluation error all mean *not stale*. A restart never happens on evidence that could not be read.

Build and test outputs are excluded from freshness inputs in `cliutil` (`BuildOutputSkipNames`/`BuildOutputSkipSuffixes`), shared by the writer, the evaluator, and the preflight checker. Without that, a `go test -coverprofile` run would rewrite `coverage.out`, mark the binary stale, trigger a rebuild, and be invalidated again by the next test run.

See [Phases Overview](../phases/README.md) for the generated effective registry, policy dimensions, and phase definitions.

## Provider Descriptor Contract

Each provider-backed phase is declared by `scenarios/<provider>/.vrooli/test-genie.json`. The descriptor must include provider and phase identity, `source: "validation-provider"`, a positive timeout, validation contract `scenario-validation/v1`, declarative applicability, policy, runnability, `docs.path`, and an embedded `maturity` block. Retired `.vrooli/maturity.json` files are rejected so maturity cannot drift from the phase descriptor.

Operators can inspect the effective descriptor projection without reading code:

```bash
test-genie phases inspect <phase> --json
test-genie phases applicability <scenario> --phase <phase> --json
test-genie phases plan <scenario> --preset comprehensive --json
test-genie provider-contract scan --json
```

The phase inspection and plan surfaces expose provider, descriptor path, docs path, policy, runnability, applicability reasons, freshness requirement, profile membership, phase/runtime class, dimensions, and finding source.

## Comparison contracts and provenance

Every run freezes a descriptor snapshot. Each phase carries a semantic
comparison fingerprint and a `comparison.mode`: `compatible`,
`changed-unreviewed` (the default), `invalidated`, or `superseded`. Display
copy and ordering do not affect this fingerprint; validation policy, provider,
applicability, and evidence semantics do. Therefore a same-key validator edit
cannot silently turn a pass-to-fail result into a regression: it is reported as
an explicit contract change until the provider declares compatibility.

`runs compare` returns behavior, coverage, compatibility, and provenance for
each phase and for the aggregate. Provider outages, skipped phases,
inapplicability, missing artifacts, and legacy snapshots are coverage or
provenance facts with structured diagnostics—not clean behavior. Gate-quality
run starts compute a source tree plus frozen plan identity before coalescing and
recheck it at execution start; a changed source or plan is refused instead of
being attributed to the earlier request.

## Running Tests with Test Genie

### Using Test Presets

Test Genie provides preconfigured presets for common testing scenarios:

```bash
# Quick sanity check (~1 min)
test-genie execute my-scenario --preset quick

# Smoke test before pushing (~4 min)
test-genie execute my-scenario --preset smoke

# Full validation before release (~8 min)
test-genie execute my-scenario --preset comprehensive
```

| Preset | Phases | Use Case |
|--------|--------|----------|
| **Quick** | Structure, Docs, Business, Unit, Proto | Fast feedback during development |
| **Smoke** | Structure, Quality, Docs, Business, Integration, Proto | Pre-push validation |
| **Architecture Audit** | Structure, Contracts, UI Health, Docs, Architecture, Proto | Surface and architecture review |
| **Comprehensive** | All applicable registry phases | Full coverage before release |

See [Presets Reference](../reference/presets.md) for detailed preset definitions.

### Using the REST API

For CI/CD and agent automation, create a server-owned execution and wait once
for its durable result:

```bash
# Get API port
API_PORT=$(vrooli scenario port test-genie API_PORT)

# Execute with comprehensive preset
curl -X POST "http://localhost:${API_PORT}/api/v1/executions" \
  -H "Content-Type: application/json" \
  -d '{
    "scenarioName": "my-scenario",
    "preset": "comprehensive",
    "failFast": true
  }'
```

Use the returned execution ID with the durable run protocol in the
[Server-Owned Execution Guide](sync-execution.md).

### Using the Makefile

From within a scenario directory:

```bash
cd scenarios/my-scenario
make test     # Run all tests via test-genie
make logs     # View test logs
```

## What Each Phase Validates

### Phase 1: Structure Validation

**Purpose**: Ensure scenario has required files and valid configuration.

**Checks**:
- Required files exist (README.md, PRD.md, Makefile)
- `.vrooli/service.json` is valid JSON with required fields
- Test directory structure is correct
- File permissions are appropriate

**Example failures**:
- Missing `.vrooli/service.json`
- Schema validation failure in configuration
- Missing test directory

### Phase 2: Dependencies Check

**Purpose**: Verify all required tools and resources are available.

**Checks**:
- Language runtimes (Go, Node.js, Python) at required versions
- Package managers (pnpm, go mod) functional
- Required CLI tools present
- Resource dependencies declared in service.json

**Example failures**:
- Missing runtime or package manager reported by SDA readiness
- Required resource or scenario dependency unavailable
- Declared-vs-actual dependency graph drift
- Missing pnpm release-age policy or blocked dependency governance finding

### Phase 3: Quality

**Purpose**: Run static analysis and type checking to catch errors before runtime.

**Checks**:
- Static quality contracts are delegated to `quality-health audit run <scenario> --json`.
- TypeScript/JavaScript: `tsc --noEmit`, `eslint`
- Python: `ruff`/`flake8`, optional `mypy`

Warnings are informational; error findings fail. See [Quality Phase](../phases/quality/README.md).

### Phase 4: Docs Validation

**Purpose**: Ensure documentation stays portable and healthy.

**Checks**:
- Markdown structure (unclosed fences)
- Mermaid diagram headers + bracket balance
- Link integrity (local files must exist; external URLs HTTP-checked)
- Absolute filesystem paths are blocked (for example, `<absolute-path>` or `C:\\repo\\...`)

See [Docs Phase](../phases/docs/README.md) for configuration options.

### Phase 5: Unit Tests

**Purpose**: Run unit tests for all languages present in the scenario.

**Runners**:
| Language | Detection | Command |
|----------|-----------|---------|
| Go | `go.mod` in `api/` | `go test ./...` |
| Node.js | `package.json` in `ui/` | `pnpm test` |
| Python | `pytest.ini` or `test_*.py` | `pytest` |

**Coverage requirements**:
- Warning: < 80%
- Error: < 70%

See [Scenario Unit Testing](../phases/unit/scenario-unit-testing.md) for writing effective unit tests.

### Phase 6: Integration Tests

**Purpose**: Validate component interactions with running scenario.

**Requires**: Scenario must be running

**Checks**:
- API health endpoints responding
- UI accessible
- CLI commands functional
- Cross-component communication working

**Example tests**:
```bash
# API health check
curl -f http://localhost:${API_PORT}/health

# UI accessibility
curl -f http://localhost:${UI_PORT}

# CLI functionality
my-scenario-cli --version
```

### Phase 7: Workflow Tests

**Purpose**: Validate end-to-end UI workflows through workflow-health.

**Requires**: Scenario + BAS running

**Validates**:
- Declarative browser workflows in `bas/cases/` (via `bas/registry.json`)
- Isolation + seeds (temporary DB/Redis + `coverage/runtime/seed-state.json`)
- Contract correctness with BAS execution + timeline responses

See [Workflow Phase](../phases/workflow/README.md) for current behavior.

### Phase 8: Business Logic Tests

**Purpose**: Validate end-to-end workflows and business requirements.

**Requires**: Scenario must be running

**Validates**:
- Complete user journeys (multi-step workflows)
- Business rules and domain logic
- Data integrity across operations
- Error recovery workflows

**Example workflow test**:
```bash
# Create → Update → Verify workflow
project_id=$(curl -s -X POST "$API_URL/projects" -d '{"name":"Test"}' | jq -r '.id')
curl -s -X PUT "$API_URL/projects/$project_id" -d '{"name":"Updated"}'
result=$(curl -s "$API_URL/projects/$project_id" | jq -r '.name')
[ "$result" = "Updated" ] && echo "PASS" || echo "FAIL"
```

For UI workflows, use [BAS playbooks](ui-testability.md) instead of curl-based testing.

### Phase 9: Performance Tests

**Purpose**: Establish performance baselines and detect regressions.

**Requires**: Scenario must be running

**Measures**:
- API response times (p50, p95, p99)
- Build duration (Go, UI)
- Resource usage
- Throughput under load

**Example checks**:
```bash
# Build time budget
time go build -o /dev/null ./... # Should complete < 30s

# API response time
time curl -s "$API_URL/health" # Should respond < 100ms
```

## Test Directory Structure

Scenarios should have this test structure:

```
scenario/
├── .vrooli/
│   ├── service.json      # Scenario configuration (lifecycle.test.steps invokes test-genie)
│   └── testing.json      # Test-genie configuration (optional)
├── test/                  # Test artifacts directory
│   └── playbooks/        # BAS workflow tests (optional)
├── api/
│   └── *_test.go         # Go unit tests
└── ui/
    └── *.test.ts         # Vitest/Jest tests
```

> **Note**: Testing is orchestrated via `.vrooli/service.json` `lifecycle.test.steps` which invokes `test-genie execute`. The legacy `coverage/run-tests.sh` + `coverage/phases/*` pattern is deprecated.

## Configuration with `.vrooli/testing.json`

Customize test-genie behavior per scenario:

```json
{
  "phases": {
    "unit": {
      "timeout": 120,
      "coverageWarn": 85,
      "coverageError": 75
    },
    "performance": {
      "enabled": false
    }
  },
  "requirements": {
    "sync": true,
    "syncOnSuccess": true
  },
  "presets": {
    "default": "smoke"
  }
}
```

See [Phases Overview](../phases/README.md) for all configuration options.

## Requirements Tracking

Tag tests with `[REQ:ID]` to automatically track requirement coverage:

```go
func TestCreateProject(t *testing.T) {
    t.Run("creates project with valid data [REQ:MY-PROJECT-CREATE]", func(t *testing.T) {
        // Test implementation
    })
}
```

```typescript
describe('projectStore [REQ:MY-PROJECT-CRUD]', () => {
    it('creates project', () => { /* ... */ });
    it('updates project', () => { /* ... */ });
});
```

After running comprehensive tests, requirements are automatically synced:

```bash
test-genie execute my-scenario --preset comprehensive
# Requirements synced to requirements/index.json
```

See [Requirements Sync Guide](../phases/business/requirements-sync.md) for complete documentation.

## Dynamic Port Discovery

Never hardcode ports. Use dynamic discovery:

```bash
# Get ports from vrooli CLI
API_PORT=$(vrooli scenario port "$scenario_name" API_PORT)
UI_PORT=$(vrooli scenario port "$scenario_name" UI_PORT)

# Build URLs
API_URL="http://localhost:$API_PORT"
UI_URL="http://localhost:$UI_PORT"
```

## Coverage Standards

| Component | Minimum | Target |
|-----------|---------|--------|
| Unit Tests | 70% | 80%+ |
| Integration | All endpoints | All endpoints |
| Business Logic | Core workflows | All PRD requirements |
| Performance | Baselines set | No regressions |

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| "service.json not found" | Create `.vrooli/service.json` |
| "Invalid JSON" | Validate: `jq empty .vrooli/service.json` |
| "Port discovery fails" | Ensure scenario is running |
| "Phase skipped" | Check if scenario needs to be running |
| "Coverage too low" | Add more unit tests |
| "Timeout exceeded" | Increase in `.vrooli/testing.json` |

## Testing Checklist

Before considering a scenario test-ready:

- [ ] `.vrooli/service.json` properly configured with `lifecycle.test.steps` invoking `test-genie execute`
- [ ] Test directory exists for artifacts (playbooks, fixtures, logs)
- [ ] Unit tests with coverage > 70%
- [ ] `[REQ:ID]` tags on tests matching PRD requirements
- [ ] Integration tests for all API endpoints
- [ ] Business logic tests for core workflows
- [ ] Performance baselines established
- [ ] Dynamic port discovery (no hardcoded ports)
- [ ] `make test` works from scenario directory

## See Also

### Phase Documentation
- [Phases Overview](../phases/README.md) - Complete phase reference with mermaid diagrams
- [Structure Phase](../phases/structure/README.md) - File and CLI validation
- [Unit Phase](../phases/unit/README.md) - Test runners and coverage
- [Integration Phase](../phases/integration/README.md) - CLI and API testing
- [Workflow Phase](../phases/workflow/README.md) - BAS browser automation
- [Business Phase](../phases/business/README.md) - Requirements validation
- [Performance Phase](../phases/performance/README.md) - Build benchmarks and Lighthouse

### Related Guides
- [Requirements Sync](../phases/business/requirements-sync.md) - Automatic requirement tracking
- [Scenario Unit Testing](../phases/unit/scenario-unit-testing.md) - Writing effective unit tests
- [Performance Testing](../phases/performance/performance-testing.md) - Build benchmarks and Lighthouse audits
- [Custom Presets](custom-presets.md) - Create tailored presets for CI/CD
- [Dashboard Guide](dashboard-guide.md) - Using the web UI
- [UI Testability](ui-testability.md) - Design testable UIs
- [Sync Execution](sync-execution.md) - API usage for CI/CD
- [Troubleshooting](troubleshooting.md) - Debug common issues

### Reference
- [Presets](../reference/presets.md) - Preset configurations
- [API Endpoints](../reference/api-endpoints.md) - REST API reference
- [Test Runners](../phases/unit/test-runners.md) - Language-specific runners

### Concepts
- [Architecture](../concepts/architecture.md) - Go orchestrator design
