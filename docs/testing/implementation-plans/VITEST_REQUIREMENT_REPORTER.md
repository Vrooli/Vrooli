# Vitest Requirement Reporter - Implementation Plan

**Target**: browser-automation-studio (pilot), then all vitest-based scenarios

## Context

Vrooli's requirements system links PRD goals → technical requirements → test validations. Requirements live in `requirements/*.json` (JSON format), and tests use `[REQ:ID]` tags to declare coverage. However, the tag→requirement correlation is currently **manual**.

**The Problem**: Developers must maintain hardcoded requirement lists in phase scripts:
```bash
# test/phases/test-unit.sh - MANUAL TRACKING
UNIT_REQUIREMENTS=(BAS-WORKFLOW-PERSIST-CRUD BAS-EXEC-TELEMETRY-STREAM ...)
for requirement_id in "${UNIT_REQUIREMENTS[@]}"; do
  testing::phase::add_requirement --id "$requirement_id" --status passed
done
```

**The Solution**: Automatic extraction via custom Vitest reporter:
```typescript
// Test declares coverage
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => { ... });

// Reporter extracts during test run → outputs JSON + stdout
// Phase helpers consume → write phase-results
// report.js --mode sync → auto-updates requirements/*.json
```

**Key Constraints**:
- Must work with existing `_node_collect_requirement_tags()` parser (emit parseable stdout)
- BAS uses `vite.config.ts` with embedded test config (not separate vitest.config.ts)
- Requirements are JSON (native parsing, no YAML library needed)
- Scope: Vitest only (Go/Python are future work)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  packages/vitest-requirement-reporter/                      │
│    ├── src/reporter.ts    (Extract [REQ:ID] from tests)    │
│    ├── src/types.ts       (TypeScript interfaces)          │
│    └── package.json       (Shared workspace package)       │
└─────────────────────────────────────────────────────────────┘
                          ↓ imported by
┌─────────────────────────────────────────────────────────────┐
│  scenarios/*/ui/vite.config.ts                              │
│    reporters: ['default', new RequirementReporter()]       │
└─────────────────────────────────────────────────────────────┘
                          ↓ runs during
┌─────────────────────────────────────────────────────────────┐
│  npm run test → vitest executes                             │
│    → Reporter extracts [REQ:ID] tags                        │
│    → Writes coverage/vitest-requirements.json               │
│    → Emits stdout: "✓ PASS REQ:BAS-WORKFLOW-PERSIST-CRUD"  │
└─────────────────────────────────────────────────────────────┘
                          ↓ parsed by
┌─────────────────────────────────────────────────────────────┐
│  scripts/scenarios/testing/unit/node.sh                     │
│    _node_collect_requirement_tags() reads stdout           │
│    → testing::phase::add_requirement()                     │
└─────────────────────────────────────────────────────────────┘
                          ↓ aggregated by
┌─────────────────────────────────────────────────────────────┐
│  coverage/phase-results/unit.json                           │
│    { "requirements": [{ "id": "...", "evidence": "..." }] } │
└─────────────────────────────────────────────────────────────┘
                          ↓ consumed by
┌─────────────────────────────────────────────────────────────┐
│  scripts/requirements/report.js --mode sync                 │
│    → Reads phase-results/*.json                             │
│    → Updates requirements/*.json status fields              │
│    → (Phase 6) Auto-adds/removes validation entries        │
└─────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Create Shared Reporter Package

**Time Estimate**: 2-3 hours

### Package Structure

```
packages/vitest-requirement-reporter/
├── package.json
├── tsconfig.json
├── src/
│   ├── reporter.ts
│   ├── types.ts
│   └── index.ts
├── dist/ (compiled)
└── README.md
```

### package.json

```json
{
  "name": "@vrooli/vitest-requirement-reporter",
  "version": "1.0.0",
  "description": "Vitest reporter for tracking requirement coverage",
  "type": "module",
  "main": "./dist/reporter.js",
  "types": "./dist/reporter.d.ts",
  "exports": {
    ".": {
      "types": "./dist/reporter.d.ts",
      "default": "./dist/reporter.js"
    }
  },
  "files": ["dist", "src", "README.md"],
  "scripts": {
    "build": "tsc",
    "dev": "tsc --watch",
    "prepublishOnly": "npm run build"
  },
  "peerDependencies": {
    "vitest": ">=1.0.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0",
    "vitest": "^3.2.0"
  }
}
```

### tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "node",
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

### src/types.ts

```typescript
/**
 * Result for a single requirement tracked by the reporter
 */
export interface RequirementResult {
  /** Requirement ID (e.g., BAS-WORKFLOW-PERSIST-CRUD) */
  id: string;

  /** Worst status across all tests for this requirement */
  status: 'passed' | 'failed';

  /** Evidence string showing which tests validated this requirement */
  evidence: string;

  /** Total duration in milliseconds across all tests */
  duration_ms: number;

  /** Number of tests that validated this requirement */
  test_count: number;
}

/**
 * Complete requirement coverage report output
 */
export interface RequirementReport {
  /** ISO timestamp when report was generated */
  generated_at: string;

  /** Scenario name (auto-detected or configured) */
  scenario: string;

  /** Testing phase (always 'unit' for vitest) */
  phase: 'unit';

  /** Test framework identifier */
  test_framework: 'vitest';

  /** Total number of tests executed */
  total_tests: number;

  /** Number of tests that passed */
  passed_tests: number;

  /** Number of tests that failed */
  failed_tests: number;

  /** Number of tests that were skipped (NOT tracked in requirements) */
  skipped_tests: number;

  /** Total test execution time in milliseconds */
  duration_ms: number;

  /** Array of requirement results, sorted by ID */
  requirements: RequirementResult[];
}

/**
 * Configuration options for RequirementReporter
 */
export interface RequirementReporterOptions {
  /** Output file path (default: coverage/vitest-requirements.json) */
  outputFile?: string;

  /** Scenario name (auto-detected from cwd if not provided) */
  scenario?: string;

  /** Enable verbose console output (default: true) */
  verbose?: boolean;

  /** Pattern to extract requirement IDs (default: /\[REQ:([A-Z0-9_-]+(?:,\s*[A-Z0-9_-]+)*)\]/gi) */
  pattern?: RegExp;

  /** Emit parseable stdout for existing shell infrastructure (default: true) */
  emitStdout?: boolean;
}
```

### src/reporter.ts

```typescript
import type { Reporter, File, Task } from 'vitest';
import { existsSync, mkdirSync, writeFileSync } from 'fs';
import { dirname, relative } from 'path';
import type {
  RequirementReporterOptions,
  RequirementResult,
  RequirementReport,
} from './types.js';

/**
 * Custom Vitest reporter that extracts requirement IDs from test descriptions
 * and correlates them with test results.
 *
 * @example
 * // vitest.config.ts
 * import RequirementReporter from '@vrooli/vitest-requirement-reporter';
 *
 * export default defineConfig({
 *   test: {
 *     reporters: [
 *       'default',
 *       new RequirementReporter({
 *         outputFile: 'coverage/vitest-requirements.json',
 *         emitStdout: true,  // Required for phase helper integration
 *         verbose: true,
 *       }),
 *     ],
 *   },
 * });
 */
export default class RequirementReporter implements Reporter {
  private options: Required<RequirementReporterOptions>;
  private requirementMap: Map<string, RequirementResult> = new Map();
  private stats = { total: 0, passed: 0, failed: 0, skipped: 0 };
  private startTime = 0;

  constructor(options: RequirementReporterOptions = {}) {
    this.options = {
      outputFile: options.outputFile || 'coverage/vitest-requirements.json',
      scenario: options.scenario || this.detectScenario(),
      verbose: options.verbose ?? true,
      pattern: options.pattern || /\[REQ:([A-Z0-9_-]+(?:,\s*[A-Z0-9_-]+)*)\]/gi,
      emitStdout: options.emitStdout ?? true,
    };
  }

  /**
   * Auto-detect scenario name from current working directory
   */
  private detectScenario(): string {
    const cwd = process.cwd();
    const scenariosMatch = cwd.match(/scenarios\/([^/]+)/);
    return scenariosMatch?.[1] || 'unknown';
  }

  /**
   * Validate requirement ID against schema pattern: [A-Z][A-Z0-9]+-[A-Z0-9-]+
   */
  private validateRequirementId(id: string): boolean {
    const pattern = /^[A-Z][A-Z0-9]+-[A-Z0-9-]+$/;
    if (!pattern.test(id)) {
      console.warn(`⚠️  Invalid requirement ID format: ${id} (expected: [A-Z][A-Z0-9]+-[A-Z0-9-]+)`);
      return false;
    }
    return true;
  }

  /**
   * Initialize reporter state at the start of test run
   */
  onInit(): void {
    this.startTime = Date.now();
    this.requirementMap.clear();
    this.stats = { total: 0, passed: 0, failed: 0, skipped: 0 };
  }

  /**
   * Extract requirement IDs from test name hierarchy
   * Walks up from test → suite → parent suite, collecting all [REQ:ID] tags
   */
  private extractRequirements(task: Task): string[] {
    const requirements: Set<string> = new Set();
    let current: Task | undefined = task;

    // Walk up the test hierarchy
    while (current) {
      const matches = current.name.matchAll(this.options.pattern);
      for (const match of matches) {
        // Handle comma-separated requirements: [REQ:ID1, ID2, ID3]
        const ids = match[1].split(/,\s*/);
        ids.forEach(id => {
          const trimmedId = id.trim();
          if (this.validateRequirementId(trimmedId)) {
            requirements.add(trimmedId);
          }
        });
      }
      current = current.suite;
    }

    return Array.from(requirements);
  }

  /**
   * Generate human-readable test path for evidence string
   * Format: relative/path/to/file.test.ts:Suite > Nested Suite > Test Name
   */
  private getTestPath(task: Task, file: File): string {
    const parts: string[] = [];
    let current: Task | undefined = task;

    while (current && (current.type !== 'suite' || current.name)) {
      if (current.name) {
        // Remove [REQ:...] tags from display path for cleaner evidence
        const cleanName = current.name.replace(this.options.pattern, '').trim();
        parts.unshift(cleanName);
      }
      current = current.suite;
    }

    const relativePath = relative(process.cwd(), file.filepath);
    return `${relativePath}:${parts.join(' > ')}`;
  }

  /**
   * Process a single test task and update requirement tracking
   */
  private processTask(task: Task, file: File): void {
    // Recursively process suite children
    if (task.type !== 'test') {
      task.tasks?.forEach(child => this.processTask(child, file));
      return;
    }

    this.stats.total++;

    const requirements = this.extractRequirements(task);
    if (requirements.length === 0) {
      // No requirements tagged, skip tracking
      return;
    }

    // IMPORTANT: Skipped tests are completely ignored (user requirement)
    if (task.result?.state === 'skip') {
      this.stats.skipped++;
      return;
    }

    const status = task.result?.state === 'pass' ? 'passed' : 'failed';
    this.stats[status]++;

    const duration = task.result?.duration || 0;
    const evidence = this.getTestPath(task, file);

    // Update each requirement this test covers
    requirements.forEach(reqId => {
      const existing = this.requirementMap.get(reqId);

      if (!existing) {
        // First test for this requirement
        this.requirementMap.set(reqId, {
          id: reqId,
          status,
          evidence,
          duration_ms: duration,
          test_count: 1,
        });
      } else {
        // Aggregate results for this requirement
        // Worst status wins: failed > passed
        if (status === 'failed') {
          existing.status = status;
        }

        // USER REQUIREMENT: List ALL tests (if 10 tests, show all 10)
        existing.evidence += `; ${evidence}`;
        existing.duration_ms += duration;
        existing.test_count++;
      }
    });
  }

  /**
   * Generate and write final report after all tests complete
   */
  async onFinished(files?: File[]): Promise<void> {
    if (!files) return;

    // Process all test files
    files.forEach(file => {
      file.tasks.forEach(task => this.processTask(task, file));
    });

    const report: RequirementReport = {
      generated_at: new Date().toISOString(),
      scenario: this.options.scenario,
      phase: 'unit',
      test_framework: 'vitest',
      total_tests: this.stats.total,
      passed_tests: this.stats.passed,
      failed_tests: this.stats.failed,
      skipped_tests: this.stats.skipped,
      duration_ms: Date.now() - this.startTime,
      requirements: Array.from(this.requirementMap.values()).sort((a, b) =>
        a.id.localeCompare(b.id)
      ),
    };

    // Ensure output directory exists
    const outputDir = dirname(this.options.outputFile);
    if (!existsSync(outputDir)) {
      mkdirSync(outputDir, { recursive: true });
    }

    // Write JSON report
    writeFileSync(this.options.outputFile, JSON.stringify(report, null, 2));

    // Optional console summary
    if (this.options.verbose) {
      this.printSummary(report);
    }

    // Emit parseable output for existing shell infrastructure
    // CRITICAL: This enables backward compatibility with _node_collect_requirement_tags()
    if (this.options.emitStdout) {
      this.printParseableOutput(report);
    }
  }

  /**
   * Print parseable format compatible with existing testing::unit::_node_collect_requirement_tags()
   * Format matches pattern expected by node.sh:353-382
   * Emits lines matching: [MARKER] REQ:ID (description)
   */
  private printParseableOutput(report: RequirementReport): void {
    console.log('\n--- Requirement Coverage (Parseable) ---');

    report.requirements.forEach(req => {
      // Use markers that existing parser recognizes (✓ PASS, ✗ FAIL)
      const marker = req.status === 'passed' ? '✓ PASS' : '✗ FAIL';

      // Format: [MARKER] REQ:ID (test count, duration)
      console.log(`${marker} REQ:${req.id} (${req.test_count} tests, ${req.duration_ms}ms)`);
    });

    console.log('--- End Requirement Coverage ---\n');
  }

  /**
   * Print human-readable summary to console
   */
  private printSummary(report: RequirementReport): void {
    console.log('\n📋 Requirement Coverage Report:');
    console.log(`   Scenario: ${report.scenario}`);
    console.log(`   Requirements: ${report.requirements.length} covered`);
    console.log(`   Tests: ${report.passed_tests}/${report.total_tests} passed`);

    const failed = report.requirements.filter(r => r.status === 'failed');
    if (failed.length > 0) {
      console.log(`   ⚠️  Failed: ${failed.map(r => r.id).join(', ')}`);
    }

    console.log(`   Output: ${this.options.outputFile}\n`);
  }
}
```

### src/index.ts

```typescript
export { default } from './reporter.js';
export type {
  RequirementReporterOptions,
  RequirementResult,
  RequirementReport,
} from './types.js';
```

### README.md

```markdown
# @vrooli/vitest-requirement-reporter

Custom Vitest reporter for tracking requirement coverage in Vrooli scenarios.

## Installation

```bash
pnpm add @vrooli/vitest-requirement-reporter@workspace:*
```

## Usage

### Basic Setup

**vite.config.ts** (or vitest.config.ts):
```typescript
import { defineConfig } from 'vitest/config';
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  test: {
    reporters: [
      'default',
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
        emitStdout: true,  // Required for phase helper integration
        verbose: true,
      }),
    ],
  },
});
```

### Tagging Tests

Tag tests with `[REQ:ID]` in describe/it names:

```typescript
// Suite-level (all tests inherit)
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects', async () => { ... });
  it('creates project', async () => { ... });
});

// Test-level
it('validates names [REQ:BAS-PROJECT-VALIDATION]', async () => { ... });

// Multiple requirements
describe('CRUD [REQ:BAS-WORKFLOW-PERSIST-CRUD, BAS-PROJECT-API]', () => {
  it('handles operations', async () => { ... });
});
```

### Output Format

**coverage/vitest-requirements.json**:
```json
{
  "generated_at": "2025-11-04T18:32:15.234Z",
  "scenario": "browser-automation-studio",
  "phase": "unit",
  "test_framework": "vitest",
  "total_tests": 24,
  "passed_tests": 22,
  "failed_tests": 1,
  "skipped_tests": 1,
  "duration_ms": 3421,
  "requirements": [
    {
      "id": "BAS-WORKFLOW-PERSIST-CRUD",
      "status": "passed",
      "evidence": "ui/src/stores/__tests__/projectStore.test.ts:projectStore > fetches; ...",
      "duration_ms": 187,
      "test_count": 3
    }
  ]
}
```

## Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `outputFile` | `string` | `'coverage/vitest-requirements.json'` | Output file path |
| `scenario` | `string` | Auto-detected from cwd | Scenario name |
| `verbose` | `boolean` | `true` | Enable console summary |
| `emitStdout` | `boolean` | `true` | Emit parseable stdout (required for phase integration) |
| `pattern` | `RegExp` | `/\[REQ:...\]/gi` | Custom extraction pattern |

## Integration with Phase Testing

This reporter outputs data consumed by `scripts/scenarios/testing/unit/node.sh`, which automatically reports requirements to phase helpers via `testing::phase::add_requirement()`.

**No manual configuration needed in phase scripts!**
```

### Build Package

```bash
cd packages/vitest-requirement-reporter
npm install
npm run build

# Verify output
ls -la dist/
# Should see: reporter.js, reporter.d.ts, types.js, types.d.ts
```

---

## Phase 3: Integrate into BAS

**Time Estimate**: 1 hour

### Step 3.1: Update pnpm Workspace

Ensure `pnpm-workspace.yaml` includes:
```yaml
packages:
  - 'packages/*'
  - 'scenarios/*/ui'
```

### Step 3.2: Add Package Dependency

```bash
cd scenarios/browser-automation-studio/ui
pnpm add @vrooli/vitest-requirement-reporter@workspace:*
```

### Step 3.3: Update Vite Config

**IMPORTANT**: BAS uses `vite.config.ts` with embedded test configuration.

**File**: `scenarios/browser-automation-studio/ui/vite.config.ts`

Add import at top:
```typescript
import RequirementReporter from '@vrooli/vitest-requirement-reporter';
```

Update `test` section (around line 166):
```typescript
export default defineConfig({
  plugins: [react(), healthEndpointPlugin()],
  resolve: { /* existing */ },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test-utils/setupTests.ts',
    globals: true,
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    exclude: ['tests/**/*', 'node_modules/**'],
    reporters: [
      'default',
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
        emitStdout: true,  // CRITICAL: Required for existing shell parsers
        verbose: true,
      }),
    ],
    coverage: { /* existing */ },
    environmentOptions: { /* existing */ },
  },
  // ... rest unchanged
});
```

### Step 3.4: Tag Existing Tests

**File**: `scenarios/browser-automation-studio/ui/src/stores/__tests__/projectStore.test.ts`

Tests already have some tags - verify and complete:
```typescript
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects successfully', async () => { ... });
  it('creates project successfully [REQ:BAS-PROJECT-CREATE-SUCCESS]', async () => { ... });
  it('handles errors [REQ:BAS-PROJECT-CREATE-VALIDATION]', async () => { ... });
  // ... etc
});
```

Repeat for other test files based on `requirements/` mappings.

### Step 3.5: Simplify Unit Phase Script

**CRITICAL**: Do NOT remove hardcoded requirements until AFTER verifying reporter works!

**Verification Steps**:
1. Run `npm run test` in ui/ directory
2. Verify console shows: `✓ PASS REQ:BAS-WORKFLOW-PERSIST-CRUD`
3. Verify file created: `ui/coverage/vitest-requirements.json`
4. Run full phase test: `./test/run-tests.sh unit`
5. Verify `coverage/phase-results/unit.json` has requirements with vitest evidence

**After verification**, simplify `test/phases/test-unit.sh`:

**Before** (44 lines with hardcoded requirements):
```bash
UNIT_REQUIREMENTS=(
  BAS-WORKFLOW-PERSIST-CRUD
  BAS-EXEC-TELEMETRY-STREAM
  # ... manual list
)

if testing::unit::run_all_tests ...; then
  testing::phase::add_test passed
  for requirement_id in "${UNIT_REQUIREMENTS[@]}"; do
    testing::phase::add_requirement --id "$requirement_id" --status passed --evidence "Unit test suites"
  done
else
  # ... manual failure tracking
fi
```

**After** (12 lines, fully automatic):
```bash
#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run all unit tests - requirement parsing happens automatically!
if testing::unit::run_all_tests \
    --go-dir "api" \
    --node-dir "ui" \
    --skip-python \
    --coverage-warn 40 \
    --coverage-error 30 \
    --verbose; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Unit test runner reported failures"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Unit tests completed"
```

---

## Phase 4: Testing & Validation

**Time Estimate**: 1-2 hours

### Step 4.1: Unit Test the Reporter

```bash
cd scenarios/browser-automation-studio/ui
npm run test
```

**Expected output**:
```
✓ src/stores/__tests__/projectStore.test.ts (8)
✓ src/components/__tests__/ProjectModal.test.tsx (2)

Test Files  2 passed (2)
     Tests  10 passed (10)

📋 Requirement Coverage Report:
   Scenario: browser-automation-studio
   Requirements: 3 covered
   Tests: 10/10 passed
   Output: coverage/vitest-requirements.json

--- Requirement Coverage (Parseable) ---
✓ PASS REQ:BAS-PROJECT-CREATE-SUCCESS (1 tests, 45ms)
✓ PASS REQ:BAS-PROJECT-CREATE-VALIDATION (1 tests, 32ms)
✓ PASS REQ:BAS-WORKFLOW-PERSIST-CRUD (8 tests, 187ms)
--- End Requirement Coverage ---
```

**Verify**:
```bash
# JSON was created
[ -f ui/coverage/vitest-requirements.json ] && echo "✅ JSON created" || echo "❌ FAIL: No JSON"

# Stdout was emitted (check test output for parseable section)
npm run test 2>&1 | grep -q "✓ PASS REQ:" && echo "✅ Stdout emitted" || echo "❌ FAIL: No stdout"

# Inspect JSON structure
cat ui/coverage/vitest-requirements.json | jq '.requirements[] | {id, status, test_count}'
```

### Step 4.2: Phase Test Integration

```bash
cd scenarios/browser-automation-studio
./test/run-tests.sh unit
```

**Expected**:
1. Unit phase runs vitest tests
2. Reporter creates `ui/coverage/vitest-requirements.json`
3. Parser reads JSON and stdout, calls `testing::phase::add_requirement`
4. Phase summary shows individual requirements
5. `coverage/phase-results/unit.json` includes requirements with vitest evidence

**Verify phase results**:
```bash
cat coverage/phase-results/unit.json | jq '.requirements[] | select(.evidence | contains("Vitest"))'
```

Expected output:
```json
{
  "id": "BAS-WORKFLOW-PERSIST-CRUD",
  "status": "passed",
  "criticality": "P0",
  "evidence": "Vitest (3 tests): ui/src/stores/__tests__/projectStore.test.ts:projectStore > fetches; Go: api/..."
}
```

### Step 4.3: Full Test Suite

```bash
./test/run-tests.sh
```

**Verify**:
1. All phases run successfully
2. `coverage/phase-results/unit.json` has detailed vitest requirements
3. After run completes, `report.js --mode sync` runs
4. `requirements/*.json` files status fields are auto-updated

---

## Phase 5: Documentation

**Time Estimate**: 1 hour

### Step 5.1: Create Tagging Guide

**File**: `docs/testing/REQUIREMENT_TAGGING.md`

```markdown
# Requirement Tagging Guide

## Overview

Vrooli's testing infrastructure automatically tracks which tests validate which requirements using `[REQ:ID]` tags in test names.

## Vitest Tests

### Basic Tagging

```typescript
// Suite-level (all tests inherit)
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects', async () => { ... });
  it('creates project', async () => { ... });
});

// Test-level
it('validates [REQ:BAS-PROJECT-VALIDATION]', async () => { ... });

// Multiple requirements
describe('CRUD [REQ:BAS-WORKFLOW-PERSIST-CRUD, BAS-PROJECT-API]', () => {
  it('handles operations', async () => { ... });
});
```

### Finding Requirement IDs

1. Check `requirements/index.json` for top-level requirements
2. Check domain-specific files like `requirements/projects/dialog.json`
3. Search: `grep -r '"id": "BAS-' requirements/`

### Verification

```bash
npm run test  # In ui/ directory

# Look for parseable output:
# ✓ PASS REQ:BAS-WORKFLOW-PERSIST-CRUD (3 tests, 187ms)

# Or check generated file:
cat coverage/vitest-requirements.json
```

## Integration Flow

Tagged requirements automatically flow through the testing pipeline:

1. Test runs with tag
2. Reporter extracts requirement ID
3. Phase helper records evidence
4. Requirements registry auto-syncs status

**No manual tracking needed!**
```

### Step 5.2: Update Architecture Docs

**File**: `docs/testing/architecture/PHASED_TESTING.md`

Add section after "Phase 3: Unit":

```markdown
### Automatic Requirement Tracking

The unit phase automatically correlates test results with requirements using `@vrooli/vitest-requirement-reporter`.

**How it works**:

1. Developer tags tests with `[REQ:ID]` in test names
2. Vitest reporter extracts tags during test execution
3. Reporter outputs `coverage/vitest-requirements.json` + parseable stdout
4. Phase orchestrator parses output and reports to phase helpers
5. Phase results include per-requirement evidence
6. Requirements registry auto-syncs status after test run

**No manual tracking required!**

See [Requirement Tagging Guide](../REQUIREMENT_TAGGING.md) for usage.
```

---

## Phase 6: Auto-Sync Validation Entries (Vitest Only)

**Time Estimate**: 1-2 hours (simplified by JSON format)

### Goal

Enhance `scripts/requirements/report.js` to automatically add, update, and optionally prune validation entries in `requirements/*.json` files based on live vitest evidence.

> **🎯 SCOPE: Vitest UI Tests Only**
>
> Phase 6 auto-syncs ONLY:
> - `type: test` validations
> - Where `ref` matches `ui/src/**/*.test.{ts,tsx}`
>
> **PRESERVED** (never auto-modified):
> - `type: automation` (BAS workflows)
> - `type: manual` validations
> - Go tests (`api/**/*_test.go`)
> - Python tests
> - Any validation.metadata or custom fields

### Why Phase 6?

Currently `report.js --mode sync` only updates `status` fields. This phase extends it to:

1. **Auto-add** missing vitest validation entries when tests declare `[REQ:ID]` tags
2. **Auto-update** vitest validation status fields from live test results
3. **Detect orphaned** vitest validations (JSON references non-existent ui/ test files)
4. **Auto-prune** orphaned vitest validations (optional, requires `--prune-stale` flag)

### Step 6.1: Understand Current Sync Logic

**Current behavior** (`syncRequirementFile()` function - lines 1011-1097 in report.js):

```javascript
function syncRequirementFile(filePath, requirements) {
  // 1. Read JSON file with JSON.parse()
  const parsed = JSON.parse(fs.readFileSync(filePath, 'utf8'));

  // 2. Update validation.status based on liveStatus from phase-results
  // 3. Update requirement.status based on validation statuses
  // 4. Update _metadata.last_synced_at

  // 5. Write back with JSON.stringify(parsed, null, 2)
  fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n');
}
```

**Key insight**: Already parses JSON, updates statuses, and writes back. We just need to add logic to insert/remove validation entries.

### Step 6.2: Extract Test Files from Phase Results

**CRITICAL**: Read from `coverage/phase-results/unit.json` (live evidence), NOT from `validation.evidence` (stale until sync runs).

**Add new function** to `scripts/requirements/report.js`:

```javascript
/**
 * Extract vitest test file references from live phase-results evidence
 * SCOPE: Only processes vitest test files (ui/src/**/*.test.{ts,tsx})
 * @param {string} scenarioRoot - Path to scenario directory
 * @returns {Map<string, Array<{ref, phase, status}>>} - Map of requirement ID to test files
 */
function extractVitestFilesFromPhaseResults(scenarioRoot) {
  const phaseResultPath = path.join(scenarioRoot, 'coverage/phase-results/unit.json');
  if (!fs.existsSync(phaseResultPath)) {
    return new Map();
  }

  const phaseResults = JSON.parse(fs.readFileSync(phaseResultPath, 'utf8'));
  const vitestFiles = new Map();

  if (!Array.isArray(phaseResults.requirements)) {
    return vitestFiles;
  }

  phaseResults.requirements.forEach(reqResult => {
    const evidence = reqResult.evidence || '';

    // Parse vitest evidence pattern only
    // Expected format: "Vitest (3 tests): ui/src/stores/__tests__/projectStore.test.ts:..."
    const vitestMatch = evidence.match(/Vitest.*?:\s*([^:;]+\.test\.(ts|tsx))/);

    if (vitestMatch && vitestMatch[1]) {
      const testRef = vitestMatch[1].trim();

      // Only track ui/ vitest files (filter out any other patterns)
      if (testRef.startsWith('ui/src/') && testRef.match(/\.test\.(ts|tsx)$/)) {
        if (!vitestFiles.has(reqResult.id)) {
          vitestFiles.set(reqResult.id, []);
        }

        vitestFiles.get(reqResult.id).push({
          ref: testRef,
          phase: 'unit',
          status: reqResult.status === 'passed' ? 'implemented' : 'failing',
        });
      }
    }
  });

  return vitestFiles;
}
```

### Step 6.3: Add Missing Validations

**Add new function** to `scripts/requirements/report.js`:

```javascript
/**
 * Add missing vitest validation entries to requirement JSON
 * @param {string} filePath - Path to requirements JSON file
 * @param {Array} requirements - Requirements to check
 * @param {Map} vitestFiles - Map of requirement ID to test files from live evidence
 * @param {string} scenarioRoot - Scenario directory path
 * @returns {Array} - Array of changes made
 */
function addMissingValidations(filePath, requirements, vitestFiles, scenarioRoot) {
  const changes = [];

  // 1. Read and parse JSON
  const parsed = JSON.parse(fs.readFileSync(filePath, 'utf8'));

  if (!parsed || !Array.isArray(parsed.requirements)) {
    return changes;
  }

  let modified = false;

  parsed.requirements.forEach((requirement) => {
    const liveTestFiles = vitestFiles.get(requirement.id) || [];

    if (liveTestFiles.length === 0) {
      return; // No vitest evidence for this requirement
    }

    // 2. Ensure validation array exists
    if (!Array.isArray(requirement.validation)) {
      requirement.validation = [];
    }

    // 3. Build set of existing vitest refs
    const existingRefs = new Set(
      requirement.validation
        .filter(v => v.type === 'test' && v.ref && v.ref.startsWith('ui/src/'))
        .map(v => v.ref)
    );

    // 4. Add missing validations
    liveTestFiles.forEach(testFile => {
      if (existingRefs.has(testFile.ref)) {
        return; // Already exists
      }

      const newValidation = {
        type: 'test',
        ref: testFile.ref,
        phase: testFile.phase,
        status: testFile.status,
        notes: 'Auto-added from vitest evidence',
      };

      requirement.validation.push(newValidation);
      modified = true;

      changes.push({
        type: 'add_validation',
        requirement: requirement.id,
        validation: testFile.ref,
        status: testFile.status,
      });
    });
  });

  // 5. Write back if modified
  if (modified) {
    if (!parsed._metadata) {
      parsed._metadata = {};
    }
    parsed._metadata.last_synced_at = new Date().toISOString();

    fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
  }

  return changes;
}
```

### Step 6.4: Detect Orphaned Validations

**Add new function**:

```javascript
/**
 * Detect and optionally remove orphaned vitest validation entries
 * SCOPE: Only checks type:test validations pointing to ui/src/**/*.test.{ts,tsx}
 * @param {string} filePath - Path to requirements JSON file
 * @param {string} scenarioRoot - Scenario directory
 * @param {Object} options - Sync options (pruneStale flag)
 * @returns {Object} - { orphaned: Array, removed: Array }
 */
function detectOrphanedValidations(filePath, scenarioRoot, options) {
  const orphaned = [];
  const removed = [];

  // 1. Read and parse JSON
  const parsed = JSON.parse(fs.readFileSync(filePath, 'utf8'));

  if (!parsed || !Array.isArray(parsed.requirements)) {
    return { orphaned, removed };
  }

  let modified = false;

  parsed.requirements.forEach(requirement => {
    if (!Array.isArray(requirement.validation)) {
      return;
    }

    const validValidations = [];

    requirement.validation.forEach(validation => {
      // PRESERVE all non-test validations (automation, manual, etc.)
      if (validation.type !== 'test') {
        validValidations.push(validation);
        return;
      }

      // PRESERVE Go/Python tests (not vitest scope)
      const ref = validation.ref || '';
      if (!ref.startsWith('ui/src/') || !ref.match(/\.test\.(ts|tsx)$/)) {
        validValidations.push(validation);
        return;
      }

      // Check if vitest test file exists
      const exists = fs.existsSync(path.join(scenarioRoot, ref));

      if (exists) {
        validValidations.push(validation);
      } else {
        // Orphaned vitest validation found!
        orphaned.push({
          requirement: requirement.id,
          ref,
          phase: validation.phase,
          file: filePath,
        });

        if (options.pruneStale) {
          removed.push({
            requirement: requirement.id,
            ref,
            file: filePath,
          });
          modified = true;
          // Don't push to validValidations (removes it)
        } else {
          // Keep but mark as orphaned
          validValidations.push(validation);
        }
      }
    });

    // Replace validation array if pruning
    if (modified && options.pruneStale) {
      requirement.validation = validValidations;
    }
  });

  // Write back if modified
  if (modified) {
    if (!parsed._metadata) {
      parsed._metadata = {};
    }
    parsed._metadata.last_synced_at = new Date().toISOString();

    fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
  }

  return { orphaned, removed };
}
```

### Step 6.5: Integrate into Main Sync Flow

**Update `syncRequirementRegistry()` function**:

```javascript
function syncRequirementRegistry(fileRequirementMap, scenarioRoot, options) {
  const updates = [];
  const addedValidations = [];
  const orphanedValidations = [];
  const removedValidations = [];

  // Extract live vitest evidence once (from phase-results)
  const vitestFiles = extractVitestFilesFromPhaseResults(scenarioRoot);

  for (const [filePath, requirements] of fileRequirementMap.entries()) {
    // Phase 1: Detect and optionally remove orphaned validations
    const orphanResult = detectOrphanedValidations(filePath, scenarioRoot, options);
    orphanedValidations.push(...orphanResult.orphaned);
    removedValidations.push(...orphanResult.removed);

    // Phase 2: Add missing validations from live evidence
    const added = addMissingValidations(filePath, requirements, vitestFiles, scenarioRoot);
    addedValidations.push(...added);

    // Phase 3: Update status fields (existing logic)
    updates.push(...syncRequirementFile(filePath, requirements));
  }

  return {
    statusUpdates: updates,
    addedValidations,
    orphanedValidations,
    removedValidations,
  };
}
```

### Step 6.6: Update CLI Argument Parser

**Add `--prune-stale` flag**:

```javascript
function parseArgs(argv) {
  const options = {
    scenario: '',
    format: 'json',
    includePending: false,
    output: '',
    mode: 'report',
    phase: '',
    pruneStale: false,  // NEW
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      // ... existing cases ...

      case '--prune-stale':
        options.pruneStale = true;
        break;

      default:
        break;
    }
  }

  return options;
}
```

### Step 6.7: Enhance Sync Report Output

**Add new function**:

```javascript
function printSyncSummary(syncResult) {
  console.log('\n📋 Requirements Sync Report:\n');

  // Status updates (existing)
  if (syncResult.statusUpdates.length > 0) {
    console.log('✏️  Status Updates:');
    syncResult.statusUpdates.forEach(update => {
      if (update.type === 'requirement') {
        console.log(`   ${update.requirement}: status → ${update.status}`);
      }
    });
    console.log();
  }

  // Added validations (new)
  if (syncResult.addedValidations.length > 0) {
    console.log('✅ Added Validations:');
    syncResult.addedValidations.forEach(added => {
      console.log(`   ${added.requirement}: + ${added.validation}`);
    });
    console.log();
  }

  // Orphaned validations (new)
  if (syncResult.orphanedValidations.length > 0) {
    console.log('⚠️  Orphaned Validations (file not found):');
    syncResult.orphanedValidations.forEach(orphan => {
      console.log(`   ${orphan.requirement}: × ${orphan.ref}`);
    });
    console.log();

    if (!syncResult.removedValidations.length) {
      console.log('   💡 Use --prune-stale to remove orphaned validations\n');
    }
  }

  // Removed validations (new)
  if (syncResult.removedValidations.length > 0) {
    console.log('🗑️  Removed Orphaned Validations:');
    syncResult.removedValidations.forEach(removed => {
      console.log(`   ${removed.requirement}: - ${removed.ref}`);
    });
    console.log();
  }
}
```

### Step 6.8: Update Main Function

**Modify `main()` to pass options through**:

```javascript
function main() {
  const options = parseArgs(process.argv.slice(2));
  const scenarioRoot = resolveScenarioRoot(process.cwd(), options.scenario);

  // ... existing requirement collection logic ...

  if (options.mode === 'sync') {
    const syncResult = syncRequirementRegistry(
      fileRequirementMap,
      scenarioRoot,  // Pass scenario root
      options        // Pass options (includes pruneStale)
    );

    printSyncSummary(syncResult);

    console.log('✅ Requirements synced successfully\n');
    return;
  }

  // ... rest of modes ...
}
```

### Step 6.9: Testing

**Test Case 1: Auto-add missing validations**

```bash
# 1. Tag a test with new requirement
# 2. Ensure requirement exists in JSON but has no vitest validation
# 3. Run tests: npm run test
# 4. Run sync: node scripts/requirements/report.js --scenario browser-automation-studio --mode sync
# 5. Verify JSON now has validation entry with correct ref, phase, status
```

**Test Case 2: Detect orphaned validations**

```bash
# 1. Manually add fake validation to JSON: "ref": "ui/src/fake/DoesNotExist.test.ts"
# 2. Run sync without prune
# 3. Verify console shows orphaned warning
# 4. Verify JSON unchanged
```

**Test Case 3: Prune orphaned validations**

```bash
# 1. Add fake validation
# 2. Run sync with prune: --mode sync --prune-stale
# 3. Verify console shows removal
# 4. Verify JSON no longer has fake validation
```

**Test Case 4: Preserve non-vitest validations**

```bash
# 1. Ensure requirement has mix: type:test (Go), type:automation, type:manual
# 2. Run sync --prune-stale
# 3. Verify only ui/ vitest validations are modified
# 4. Verify Go tests, automation, manual entries untouched
```

### Step 6.10: Update Help Text

```javascript
function printUsage() {
  console.log(`
Usage: node scripts/requirements/report.js [options]

Options:
  --scenario NAME         Scenario name (required)
  --mode MODE             Operation mode: report, sync, validate, phase-inspect
  --format FORMAT         Output format: json, markdown, trace
  --prune-stale           Remove orphaned vitest validation entries during sync
  --include-pending       Include pending requirements in report
  --output FILE           Write output to file instead of stdout
  --phase NAME            Filter to specific phase (for phase-inspect mode)

Modes:
  report          Generate requirement coverage report (default)
  sync            Auto-update JSON files based on live test evidence
  validate        Check for broken references and missing data
  phase-inspect   Extract requirements for specific test phase

Examples:
  # Generate JSON report
  node scripts/requirements/report.js --scenario browser-automation-studio

  # Sync validations (conservative - only add/update)
  node scripts/requirements/report.js --scenario browser-automation-studio --mode sync

  # Sync and prune orphaned vitest validations
  node scripts/requirements/report.js --scenario browser-automation-studio --mode sync --prune-stale
  `);
}
```

### Step 6.11: Update Documentation

**File**: `docs/testing/REQUIREMENT_TAGGING.md`

Add section:

```markdown
## Auto-Sync Validation Entries

After Phase 6 implementation, vitest validation entries are automatically managed.

### What Gets Auto-Added

When you tag a test with `[REQ:ID]`, the sync process will:
1. Detect the test file from live phase-results evidence
2. Add a validation entry to the requirement JSON if missing
3. Set appropriate status based on test result

**Before**:
```json
{
  "requirements": [{
    "id": "BAS-WORKFLOW-PERSIST-CRUD",
    "validation": []
  }]
}
```

**After running sync**:
```json
{
  "requirements": [{
    "id": "BAS-WORKFLOW-PERSIST-CRUD",
    "validation": [{
      "type": "test",
      "ref": "ui/src/stores/__tests__/projectStore.test.ts",
      "phase": "unit",
      "status": "implemented",
      "notes": "Auto-added from vitest evidence"
    }]
  }]
}
```

### What Gets Auto-Updated

Existing vitest validation entries get status updates:
- Test passes → `status: implemented`
- Test fails → `status: failing`

### What Gets Detected as Orphaned

If JSON references a test file that doesn't exist, sync warns:
```
⚠️  Orphaned Validations (file not found):
   BAS-UI-COMPONENTS: × ui/src/components/__tests__/DeletedComponent.test.ts
```

Use `--prune-stale` to remove them automatically.

### What's NOT Auto-Managed

**PRESERVED** (never auto-modified):
- `type: automation` - BAS workflows, manual test procedures
- `type: manual` - Human verification steps
- Go tests (`api/**/*_test.go`)
- Custom notes and metadata

Only vitest `type: test` entries (ui/src/**/*.test.{ts,tsx}) are auto-managed.
```

---

## Migration Guide

### For New Scenarios

**Step 1**: Add dependency
```bash
cd scenarios/my-scenario/ui
pnpm add @vrooli/vitest-requirement-reporter@workspace:*
```

**Step 2**: Update vitest config
```typescript
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  test: {
    reporters: [
      'default',
      new RequirementReporter({ emitStdout: true }),
    ],
  },
});
```

**Step 3**: Tag tests
```typescript
describe('MyFeature [REQ:MY-SCENARIO-FEATURE-001]', () => {
  it('works correctly', () => { ... });
});
```

**Done!** No changes to phase scripts needed.

### For Existing Scenarios

**Gradual adoption**:

1. Add reporter to vitest config
2. Run tests - reporter won't find requirements but won't break
3. Tag tests incrementally over time
4. Each tagged test automatically tracked

**No breaking changes** - untagged tests still run normally.

---

## Success Metrics

| **Metric** | **Before** | **After** | **Improvement** |
|------------|-----------|----------|-----------------|
| Lines in test-unit.sh | 44 | 12 | -73% boilerplate |
| Manual requirement lists | 1 per phase | 0 | 100% eliminated |
| Requirement evidence | "Unit test suites" | "Vitest (3 tests): projectStore.test.ts:..." | Specific paths |
| Sync accuracy | Manual updates | Automatic | 100% reliable |
| Developer effort | Tag + update script | Tag only | 50% reduction |

---

## Appendices

### Appendix A: Example Output Files

**vitest-requirements.json**:
```json
{
  "generated_at": "2025-11-04T18:32:15.234Z",
  "scenario": "browser-automation-studio",
  "phase": "unit",
  "test_framework": "vitest",
  "total_tests": 24,
  "passed_tests": 22,
  "failed_tests": 1,
  "skipped_tests": 1,
  "duration_ms": 3421,
  "requirements": [
    {
      "id": "BAS-WORKFLOW-PERSIST-CRUD",
      "status": "passed",
      "evidence": "ui/src/stores/__tests__/projectStore.test.ts:projectStore > fetches; ui/src/stores/__tests__/projectStore.test.ts:projectStore > creates; ui/src/stores/__tests__/projectStore.test.ts:projectStore > updates",
      "duration_ms": 187,
      "test_count": 3
    }
  ]
}
```

**phase-results/unit.json**:
```json
{
  "phase": "unit",
  "scenario": "browser-automation-studio",
  "status": "passed",
  "requirements": [
    {
      "id": "BAS-WORKFLOW-PERSIST-CRUD",
      "status": "passed",
      "criticality": "P0",
      "evidence": "Vitest (3 tests): ui/src/stores/__tests__/projectStore.test.ts:projectStore > fetches; Go: api/services/workflow_service_test.go:TestWorkflowCRUD"
    }
  ]
}
```

### Appendix B: Alternative vitest.config.ts Approach

If you prefer to separate test config from vite.config.ts:

**vitest.config.ts**:
```typescript
import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config';
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      reporters: [
        'default',
        new RequirementReporter({ emitStdout: true }),
      ],
    },
  })
);
```

**package.json**:
```json
{
  "scripts": {
    "test": "vitest --run --config vitest.config.ts"
  }
}
```

### Appendix C: Troubleshooting

**Reporter not outputting JSON**

Symptom: No `coverage/vitest-requirements.json` file created

Causes:
1. Reporter not registered in vitest config
2. No tests with `[REQ:ID]` tags ran
3. vitest failed before reporter could finish

Debug:
```bash
# Check if reporter is registered
cat ui/vite.config.ts | grep RequirementReporter

# Check for tagged tests
grep -r "\[REQ:" ui/src/**/*.test.*

# Run with verbose logging
npm run test -- --reporter=verbose
```

**Requirements not appearing in phase results**

Symptom: `coverage/phase-results/unit.json` has empty requirements array

Causes:
1. Parser not being called by orchestrator
2. vitest-requirements.json in wrong location
3. Stdout not emitted (emitStdout: false)

Debug:
```bash
# Check if JSON output exists
ls ui/coverage/vitest-requirements.json

# Check stdout emission in test output
npm run test 2>&1 | grep "✓ PASS REQ:"

# Manually verify parser can read JSON
node -e "console.log(JSON.parse(require('fs').readFileSync('ui/coverage/vitest-requirements.json', 'utf8')))"
```

**Requirement status not syncing**

Symptom: `requirements/*.json` files not updating after tests

Causes:
1. report.js not being called after test run
2. Requirement IDs mismatch between tests and json
3. Phase results not written

Debug:
```bash
# Manually run sync
node scripts/requirements/report.js --scenario browser-automation-studio --mode sync

# Check phase results exist
ls coverage/phase-results/*.json

# Verify requirement IDs match
grep '"id": "BAS-' requirements/**/*.json
grep "\[REQ:BAS-" ui/src/**/*.test.*
```

### Appendix D: Smoke Test Script

```bash
#!/bin/bash
# scripts/requirements/test-vitest-reporter.sh
# Smoke test for vitest reporter integration

cd scenarios/browser-automation-studio/ui || exit 1

echo "Running vitest tests..."
npm run test 2>&1 | tee /tmp/vitest-output.log

# Check stdout emission
if ! grep -q "✓ PASS REQ:" /tmp/vitest-output.log; then
  echo "❌ FAIL: Reporter not emitting parseable stdout"
  exit 1
fi

# Check JSON creation
if [ ! -f coverage/vitest-requirements.json ]; then
  echo "❌ FAIL: vitest-requirements.json not created"
  exit 1
fi

# Verify JSON structure
if ! jq -e '.requirements | length > 0' coverage/vitest-requirements.json >/dev/null; then
  echo "❌ FAIL: JSON has no requirements"
  exit 1
fi

echo "✅ Reporter smoke test passed"
```

---

## Checklist

### Phase 1: Package Creation (2-3 hours)
- [ ] Create `packages/vitest-requirement-reporter/` directory
- [ ] Write `package.json` with peer dependencies
- [ ] Write `tsconfig.json` for TypeScript compilation
- [ ] Implement `src/types.ts`
- [ ] Implement `src/reporter.ts` with ID validation
- [ ] Add `printParseableOutput()` method for stdout emission
- [ ] Handle skipped tests (ignore, don't track)
- [ ] List ALL tests in evidence (if 10 tests, show all 10)
- [ ] Write `src/index.ts` export file
- [ ] Write `README.md` with usage examples
- [ ] Run `npm install`
- [ ] Run `npm run build` and verify dist/ output

### Phase 3: BAS Integration (1 hour)
- [ ] Verify workspace package config includes packages/
- [ ] Add reporter dependency to BAS ui/package.json
- [ ] Run `pnpm install` in BAS ui directory
- [ ] Update `ui/vite.config.ts` with reporter
- [ ] Set `emitStdout: true` in reporter options
- [ ] Tag existing tests with `[REQ:ID]`
- [ ] Verify reporter works BEFORE simplifying phase script
- [ ] Simplify `test/phases/test-unit.sh` (remove hardcoded requirements)

### Phase 4: Testing (1-2 hours)
- [ ] Run `npm run test` in ui/ directory
- [ ] Verify console shows parseable requirement output
- [ ] Verify console shows human-readable summary
- [ ] Verify `coverage/vitest-requirements.json` created
- [ ] Run `./test/run-tests.sh unit`
- [ ] Verify `coverage/phase-results/unit.json` has vitest evidence
- [ ] Check evidence shows specific test paths
- [ ] Run full test suite: `./test/run-tests.sh`
- [ ] Verify requirements auto-sync after completion
- [ ] Check `requirements/*.json` files for updated status
- [ ] Verify skipped tests are ignored
- [ ] Verify multiple tests per requirement all listed

### Phase 5: Documentation (1 hour)
- [ ] Create `docs/testing/REQUIREMENT_TAGGING.md`
- [ ] Update `docs/testing/architecture/PHASED_TESTING.md`
- [ ] Update package README with examples
- [ ] Add troubleshooting section to docs

### Phase 6: Auto-Sync Validation Entries (1-2 hours)
- [ ] Implement `extractVitestFilesFromPhaseResults()` (reads from phase-results)
- [ ] Add filter: only `type: test` validations
- [ ] Add filter: only refs matching `ui/src/**/*.test.{ts,tsx}`
- [ ] Implement `addMissingValidations()` using `JSON.parse()`
- [ ] Implement `detectOrphanedValidations()` with scope filters
- [ ] Ensure automation validations preserved
- [ ] Ensure manual validations preserved
- [ ] Ensure Go tests preserved
- [ ] Update `syncRequirementRegistry()` to call new functions
- [ ] Add `--prune-stale` flag to `parseArgs()`
- [ ] Implement `printSyncSummary()` for enhanced output
- [ ] Update `main()` to pass scenarioRoot and options
- [ ] Test auto-add missing vitest validations
- [ ] Test detect orphaned vitest validations
- [ ] Test prune orphaned validations with flag
- [ ] Test preserve non-vitest validations
- [ ] Add help text for --prune-stale
- [ ] Update tagging guide with auto-sync section
- [ ] Verify integration with test runner

---

**End of Implementation Plan**
