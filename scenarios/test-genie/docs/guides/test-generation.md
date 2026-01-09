# 🤖 Automated Test Generation with test-genie

## Overview

test-genie is a Vrooli scenario that **automatically generates comprehensive test suites** for scenarios, resources, and system components. It eliminates the manual effort of test creation while ensuring high coverage and quality, allowing AI agents and developers to focus on implementation rather than test authoring.

**Status**: Active - Core orchestration complete, AI generation in development
**Coverage Target**: 95%+ automated coverage
**Integration**: Connects to App Issue Tracker for asynchronous test generation

> **Implementation Status**:
> - ✅ Test execution via CLI and API (fully functional)
> - ✅ Phase orchestration (all 7 phases in Go)
> - ✅ Requirements sync from `[REQ:ID]` tags
> - 🔄 AI-powered test generation (delegation to App Issue Tracker)
> - 🔄 Coverage gap analysis with AI recommendations

## What is test-genie?

test-genie transforms testing from manual labor to intelligent automation by:

- **Delegating test generation** to App Issue Tracker for asynchronous AI-powered generation
- **Analyzing scenario structure** (API, UI, CLI) to determine test requirements
- **Generating comprehensive suites** across unit, integration, performance, and vault testing
- **Creating requirement-tagged tests** that automatically integrate with tracking system
- **Providing fallback templates** when delegation service unavailable

**Key Differentiator**: test-genie doesn't just scaffold boilerplate—it analyzes your PRD, requirements, and code to generate **contextually relevant, requirement-tagged tests**.

## When to Use test-genie

### ✅ **Ideal Use Cases**

1. **New scenario development** - Bootstrap testing infrastructure from PRD and requirements
2. **Coverage gaps** - Generate missing tests for uncovered code paths
3. **Requirement validation** - Auto-generate tests tagged with `[REQ:ID]` for traceability
4. **Vault testing** - Create comprehensive lifecycle tests (setup → deploy → monitor)
5. **Regression suites** - Generate regression tests after major refactors

### ⚠️ **Less Suitable For**

1. **Highly specialized tests** - Custom business logic requiring domain expertise
2. **Visual regression testing** - UI screenshot comparisons (use BAS workflows instead)
3. **Performance tuning** - Baseline establishment fine, but optimization requires manual analysis
4. **Security penetration testing** - Requires security expert review

### 🤝 **Complementary Tools**

test-genie works **with**, not instead of:
- **Manual test authoring** - For complex business logic
- **BAS workflows** - For UI automation (test-genie can generate workflow scaffolds)
- **Requirement tracking** - test-genie generates `[REQ:ID]` tagged tests

## How It Works

### Architecture

```mermaid
graph TD
    subgraph "1. Request"
        USER[Developer/AI Agent]
        CLI[test-genie CLI]
        USER -->|vrooli test generate| CLI
    end

    subgraph "2. Analysis"
        ANALYZE[Analyze Scenario]
        PRD[Read PRD.md]
        REQ[Read requirements/*.json]
        CODE[Analyze codebase]

        CLI --> ANALYZE
        ANALYZE --> PRD
        ANALYZE --> REQ
        ANALYZE --> CODE
    end

    subgraph "3. Delegation"
        TRACKER[App Issue Tracker]
        ISSUE[Create Generation Issue]
        AGENT[Downstream AI Agent]

        ANALYZE --> ISSUE
        ISSUE --> TRACKER
        TRACKER --> AGENT
    end

    subgraph "4. Generation"
        GEN[Generate Tests]
        TAG[Tag with [REQ:ID]]
        SUITE[Test Suite]

        AGENT --> GEN
        GEN --> TAG
        TAG --> SUITE
    end

    subgraph "5. Integration"
        COMMIT[Commit to Scenario]
        PHASE[Run Phase Tests]
        SYNC[Auto-sync Requirements]

        SUITE --> COMMIT
        COMMIT --> PHASE
        PHASE --> SYNC
    end

    subgraph "Fallback"
        FALLBACK[Local Templates]
        ANALYZE -.->|Tracker unavailable| FALLBACK
        FALLBACK -.-> SUITE
    end

    style USER fill:#e1f5ff
    style SUITE fill:#e8f5e9
    style SYNC fill:#fff9c4
    style FALLBACK fill:#ffebee
```

### Step-by-Step Flow

1. **Developer/Agent invokes test generation**:
   ```bash
   test-genie generate my-scenario \
     --types unit,integration,performance \
     --coverage 95
   ```

2. **test-genie analyzes scenario**:
   - Reads `PRD.md` to understand requirements
   - Parses `requirements/*.json` for requirement IDs
   - Scans codebase for API handlers, UI components, CLI commands
   - Identifies coverage gaps from existing tests

3. **Delegates to App Issue Tracker**:
   - Creates detailed generation issue with:
     - Scenario context (tech stack, architecture)
     - Requirements to validate
     - Coverage targets
     - Test types needed
   - Returns issue ID to user
   - Downstream agent picks up issue asynchronously

4. **AI Agent generates tests**:
   - Reads scenario code and requirements
   - Generates tests tagged with `[REQ:ID]`
   - Creates test files in appropriate locations:
     - `api/*_test.go` - Go unit tests
     - `ui/src/**/*.test.ts` - Vitest UI tests
     - `bas/cases/**/*.json` - BAS workflow scaffolds

5. **Tests integrated into scenario**:
   - Generated tests committed to scenario repo
   - Developer/agent reviews and refines
   - Tests run via normal phase execution
   - Requirements auto-sync based on test results

### Fallback Mode

If App Issue Tracker unavailable, test-genie falls back to **deterministic local templates**:

```bash
# Generates basic test scaffolding immediately
test-genie generate my-scenario --types unit
# Output: Created api/main_test.go, ui/src/App.test.tsx from templates
```

Fallback templates are less sophisticated but provide immediate structure.

## CLI Commands

### Generate Tests

```bash
# Generate all test types with high coverage target
test-genie generate my-scenario \
  --types unit,integration,performance,vault \
  --coverage 95

# Generate only unit tests (fastest)
test-genie generate my-scenario --types unit

# Generate with parallel execution
test-genie generate my-scenario --types unit,integration --parallel

# Dry run to see what would be generated
test-genie generate my-scenario --types unit --dry-run
```

**Output**:
```
✓ Analyzed scenario: my-scenario
✓ Found 12 requirements in requirements/*.json
✓ Identified coverage gaps: 8 untested functions
✓ Delegated generation to App Issue Tracker
  Issue ID: #12345
  Estimated completion: 15-30 minutes

Track progress:
  vrooli issue status 12345

View generated tests:
  git status  # After issue completion
```

### Execute Generated Suite

```bash
# Execute entire generated suite
test-genie execute suite-abc123 --type full --environment local

# Execute with live progress monitoring
test-genie execute suite-abc123 --watch

# Execute specific test type
test-genie execute suite-abc123 --type unit
```

### Coverage Analysis

```bash
# Analyze current coverage and identify gaps
test-genie coverage my-scenario --depth deep --threshold 90

# Generate report
test-genie coverage my-scenario --report coverage-report.json

# Show only missing coverage
test-genie coverage my-scenario --show-gaps
```

### Vault Testing

```bash
# Generate comprehensive lifecycle tests
test-genie vault my-scenario \
  --phases setup,develop,test,deploy,monitor \
  --criteria ./success-criteria.yaml
```

**Vault testing** validates scenarios through their entire lifecycle, not just runtime behavior.

## Integration with Requirement Tracking

test-genie **automatically tags generated tests** with `[REQ:ID]` based on requirement analysis.

### How Tagging Works

1. test-genie reads `requirements/*.json` to understand requirements
2. Analyzes which code files relate to which requirements (via `validation.ref` paths)
3. Generates tests in those files with appropriate `[REQ:ID]` tags

**Example generated test**:

```typescript
// Generated by test-genie from requirements/projects/dialog.json
describe('ProjectModal [REQ:BAS-PROJECT-DIALOG-OPEN]', () => {
  it('opens dialog when Create Project button clicked', () => {
    // Auto-generated test implementation
    render(<ProjectModal />);
    const button = screen.getByRole('button', { name: /create project/i });
    fireEvent.click(button);
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('closes dialog on Cancel', () => {
    // Auto-generated test implementation
    render(<ProjectModal open onClose={mockClose} />);
    const cancel = screen.getByRole('button', { name: /cancel/i });
    fireEvent.click(cancel);
    expect(mockClose).toHaveBeenCalled();
  });
});
```

### After Generation

Generated tests immediately integrate with requirement tracking:

1. **First test run**: vitest-requirement-reporter extracts `[REQ:ID]` tags
2. **Phase results**: Go orchestrator records test status per requirement
3. **Auto-sync**: Test Genie syncs requirements after comprehensive test runs
4. **Validation entries**: Added automatically to `requirements/index.json`

**No manual requirement tracking needed** - test-genie → tagging → auto-sync handles it all.

## test-genie Dashboard

Launch the web dashboard to manage test generation visually:

```bash
cd scenarios/test-genie/ui && npm start
# Open http://localhost:3000
```

### Dashboard Features

- **📈 Real-time Metrics**: Active suites, coverage trends, generation queue
- **🧪 Test Suite Management**: View, execute, and monitor generated suites
- **📊 Coverage Visualization**: Interactive gap analysis with drill-down
- **🧠 AI-Driven Recommendations**: Suggested test additions based on code changes
- **🏛️ Vault Builder**: Visual drag-and-drop vault test creation
- **📋 Generation History**: Complete audit trail of all generated tests

## Best Practices

### 1. Start with Requirements

**Before generating tests**, ensure requirements are well-defined:

```bash
# Check current coverage status
test-genie coverage my-scenario --show-gaps

# Or use the dashboard
# Navigate to Catalog → my-scenario to see requirement coverage
```

Well-defined requirements → better test generation.

### 2. Review Generated Tests

Generated tests are **starting points**, not finished products:

```bash
# After generation, review changes
git diff

# Run tests to verify they work
make test

# Refine tests as needed
# - Add edge cases
# - Improve assertions
# - Add setup/teardown
```

### 3. Iterate on Coverage

Use test-genie iteratively:

```bash
# Round 1: Generate basic unit tests
test-genie generate my-scenario --types unit

# Run and review
make test

# Round 2: Fill integration gaps
test-genie coverage my-scenario --show-gaps
test-genie generate my-scenario --types integration

# Round 3: Add performance baselines
test-genie generate my-scenario --types performance
```

### 4. Combine with Manual Testing

test-genie handles **volume**, manual testing handles **complexity**:

- **test-genie**: CRUD operations, basic validations, happy paths
- **Manual**: Complex business logic, edge cases, security scenarios

### 5. Keep Requirements Updated

For best results, keep requirements synchronized with code:

```bash
# After adding feature, update requirements first
vim requirements/new-feature.json

# Then generate tests
test-genie generate my-scenario --types unit,integration
```

Updated requirements → more relevant generated tests.

## Troubleshooting

### Generated Tests Don't Compile/Run

**Cause**: test-genie may generate syntactically correct but semantically broken tests

**Solution**:
1. Check imports are correct for your scenario
2. Verify test utilities exist (`test_helpers.go`, `setupTests.ts`)
3. Adjust test setup/teardown
4. Report patterns to test-genie via feedback

### Tests Generated Without [REQ:ID] Tags

**Cause**: test-genie couldn't map code to requirements

**Solution**:
1. Ensure requirements have `validation.ref` pointing to code files
2. Run generation with `--verbose` to see mapping logic
3. Manually add tags to generated tests
4. Update requirements with better validation references

### Delegation Hangs/Times Out

**Cause**: App Issue Tracker offline or overloaded

**Solution**:
1. Check tracker status: `vrooli scenario status app-issue-tracker`
2. Use fallback mode: `test-genie generate --fallback`
3. Check issue tracker queue: `vrooli issue list --filter test-generation`

### Generated Tests Too Basic

**Cause**: test-genie optimizes for coverage, not sophistication

**Solution**:
1. Use generated tests as starting point
2. Enhance with domain-specific logic
3. Add edge cases manually
4. Consider manual authoring for complex scenarios

## Comparison with Manual Testing

| Aspect | test-genie | Manual Testing |
|--------|------------|----------------|
| **Speed** | Minutes for 100s of tests | Hours/days for comprehensive suite |
| **Coverage** | High volume, basic assertions | Focused, sophisticated assertions |
| **Requirement Tracking** | Automatic `[REQ:ID]` tagging | Manual tagging required |
| **Context Awareness** | PRD + requirements driven | Developer domain knowledge |
| **Maintenance** | Regenerate when code changes | Manual updates required |
| **Best For** | CRUD, validations, happy paths | Business logic, edge cases, security |

**Recommended**: Use both together for comprehensive testing.

## Integration with Other Testing Tools

### With browser-automation-studio

test-genie can generate **BAS workflow scaffolds**:

```bash
# Generate workflow JSON templates
test-genie generate my-scenario --types automation

# Output: bas/cases/ui/crud-workflow.json (template)
```

Then:
1. Import scaffold into BAS UI
2. Enhance with actual UI interactions
3. Export back to JSON
4. Tests execute during the business logic phase via test-genie orchestration

### With Vitest

Generated Vitest tests use `vitest-requirement-reporter` automatically:

```typescript
// test-genie ensures config includes reporter
// vite.config.ts
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  test: {
    reporters: ['default', new RequirementReporter({ ... })],
  },
});
```

### With Go Tests

Generated Go tests follow standard conventions:

```go
// test-genie generates with [REQ:ID] tags
func TestProjectCRUD(t *testing.T) {
    t.Run("creates project [REQ:MY-PROJECT-CREATE]", func(t *testing.T) {
        // Generated test body
    })
}
```

Go test parser extracts tags automatically during phase execution.

## Coverage-Driven Development Workflow

Use test-genie iteratively to systematically improve coverage:

```mermaid
graph TB
    A[Check Coverage<br/>test-genie coverage --show-gaps] --> B{Coverage Gaps?}
    B -->|Yes| C[Generate Tests<br/>test-genie generate]
    B -->|No| Z[Done!]
    C --> D[Review Generated Tests<br/>Verify assertions]
    D --> E[Add [REQ:ID] Tags]
    E --> F[Run Tests<br/>make test]
    F --> G{Tests Pass?}
    G -->|No| H[Fix Tests<br/>Adjust generated code]
    H --> F
    G -->|Yes| I[Commit Tests]
    I --> A

    style A fill:#e1f5ff
    style C fill:#fff3e0
    style D fill:#fff9c4
    style F fill:#e8f5e9
    style I fill:#c8e6c9
```

### Systematic Gap-Filling Example

```bash
# Step 1: Identify gaps
$ test-genie coverage my-scenario --format json | jq '.uncovered[]'
"api/handlers/projects.go:45-62"
"api/services/auth.go:89-102"

# Step 2: Generate tests for specific files
test-genie generate my-scenario \
  --types unit \
  --files "api/handlers/projects.go,api/services/auth.go"

# Step 3: Review and tag generated tests
# Add [REQ:ID] tags as needed

# Step 4: Verify coverage improved
make test
test-genie coverage my-scenario
```

## Generated Test Review Checklist

Before committing generated tests, verify:

- [ ] **Assertions match expected behavior** (not just "truthy")
- [ ] **Edge cases covered** (empty, null, boundary values)
- [ ] **Error scenarios tested** (network failures, validation errors)
- [ ] **Mocks realistic** (don't mock away important logic)
- [ ] **Test names descriptive** (explain what's being tested)
- [ ] **[REQ:ID] tags added** (for requirement tracking)
- [ ] **No hallucinated imports** (check all imports exist)
- [ ] **Setup/teardown proper** (resources cleaned up)

## Quick Start Example

```bash
# 1. Check current coverage status
test-genie coverage my-scenario --show-gaps

# 2. Generate tests for gaps
test-genie generate my-scenario --types unit,integration --coverage 90

# 3. Wait for generation (or check status)
# Issue ID: #12345
vrooli issue status 12345

# 4. Review generated tests
git status
git diff

# 5. Run tests (requirements auto-sync on comprehensive preset)
test-genie execute my-scenario --preset comprehensive

# 6. Check updated coverage
test-genie coverage my-scenario
```

**Result**: Comprehensive test suite with automatic requirement tracking, generated in minutes.

## See Also

### Related Guides
- [Requirements Sync](requirements-sync.md) - How requirement tracking works
- [Phased Testing](phased-testing.md) - Complete testing workflow
- [Scenario Unit Testing](scenario-unit-testing.md) - Manual test authoring guide
- [UI Testability](ui-testability.md) - BAS workflow testing
- [Troubleshooting](troubleshooting.md) - Debug common issues

### Reference
- [API Endpoints](../reference/api-endpoints.md) - REST API reference
- [CLI Commands](../reference/cli-commands.md) - CLI reference
- [Presets](../reference/presets.md) - Test preset definitions
- [Glossary](../GLOSSARY.md) - Testing terminology

### Concepts
- [Architecture](../concepts/architecture.md) - Go orchestrator design
