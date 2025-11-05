# Vitest Requirement Reporter - Implementation Plan

**Status**: Draft
**Created**: 2025-11-04
**Author**: Claude Code AI
**Target Scenario**: browser-automation-studio (pilot), then all vitest-based scenarios

---

## Table of Contents

1. [Problem Statement](#problem-statement)
2. [Current State Analysis](#current-state-analysis)
3. [Solution Architecture](#solution-architecture)
4. [Implementation Plan](#implementation-plan)
5. [Migration Guide](#migration-guide)
6. [Success Metrics](#success-metrics)
7. [Appendices](#appendices)

---

## Problem Statement

### The Gap

Vrooli's testing infrastructure for browser-automation-studio demonstrates an excellent requirements-driven testing approach:

- ✅ **Modular requirements registry**: `requirements/*.yaml` files link PRD to technical implementation
- ✅ **Phase-based testing**: 6 standardized test phases (structure, dependencies, unit, integration, business, performance)
- ✅ **Auto-sync infrastructure**: `scripts/requirements/report.js` updates requirement status based on test evidence
- ✅ **BAS self-testing**: Workflows test themselves using BAS automation (novel approach)

However, there's a **critical missing link**: **automatic test→requirement correlation**.

### Current Manual Process

**In Go tests**: Requirements are hardcoded in phase scripts:

```bash
# test/phases/test-unit.sh
UNIT_REQUIREMENTS=(
  BAS-WORKFLOW-PERSIST-CRUD
  BAS-EXEC-TELEMETRY-STREAM
  BAS-REPLAY-TIMELINE-PERSISTENCE
  # ... 5 more
)

# All marked passed/failed together - no granularity
for requirement_id in "${UNIT_REQUIREMENTS[@]}"; do
  testing::phase::add_requirement --id "$requirement_id" --status passed
done
```

**In Vitest tests**: Requirements tagged but not parsed:

```typescript
// ui/src/stores/__tests__/projectStore.test.ts
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects successfully', async () => { ... });
});
```

The `[REQ:ID]` tag exists but:
- ❌ Not extracted automatically
- ❌ Not correlated with test pass/fail
- ❌ Not reported to phase helpers
- ❌ Requires manual maintenance in phase scripts

### Impact

1. **Maintenance burden**: Developers must manually update requirement lists when tests change
2. **Sync drift**: Hardcoded lists get out of sync with actual test coverage
3. **No granularity**: Can't tell which specific test validated which requirement
4. **Lost traceability**: `requirement.yaml` says "validated by unit tests" but doesn't specify which ones
5. **Blocked automation**: Can't auto-generate requirement coverage reports

### Vision

**Full automation loop**:
```
Developer tags test [REQ:ID]
    ↓
Test runs with vitest
    ↓
Reporter extracts requirement from tag
    ↓
Reporter correlates with test result (pass/fail/skip)
    ↓
Phase helper receives requirement evidence
    ↓
Phase results include per-requirement status
    ↓
requirements/report.js auto-syncs requirement.yaml status
```

**Zero manual maintenance** required.

---

## Current State Analysis

### What Works Today

#### 1. Requirements Registry Structure

**Location**: `scenarios/browser-automation-studio/requirements/`

```yaml
# requirements/index.yaml
meta:
  scenario: browser-automation-studio
  version: 0.2.0

imports:
  - projects/dialog.yaml
  - workflow-builder/core.yaml
  - execution/telemetry.yaml
  - replay/core.yaml
  - ai/generation.yaml
  - persistence/version-history.yaml

requirements:
  - id: BAS-FUNC-001
    category: foundation
    prd_ref: "Functional Requirements > Must Have > Visual workflow builder"
    title: Persist visual workflows
    status: in_progress  # Auto-updated by report.js
    criticality: P0
    children:
      - BAS-PROJECT-DIALOG-OPEN
      - BAS-WORKFLOW-PERSIST-CRUD
```

**Child requirement** (`requirements/projects/dialog.yaml`):

```yaml
requirements:
  - id: BAS-WORKFLOW-PERSIST-CRUD
    category: projects.persistence
    prd_ref: "Functional Requirements > Must Have > Visual workflow builder"
    title: Projects must persist workflow changes to disk
    status: complete  # Auto-updated!
    criticality: P0
    validation:
      - type: test
        ref: api/services/workflow_service_test.go
        phase: unit
        status: implemented
      - type: test
        ref: ui/src/stores/__tests__/projectStore.test.ts
        phase: unit
        status: implemented  # But not auto-correlated!
```

**Key insight**: The infrastructure expects `validation` entries to link to test files, but there's no automated parsing of those files.

#### 2. Phase-Based Test Execution

**Test runner**: `test/run-tests.sh`

```bash
#!/bin/bash
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

testing::runner::init \
  --scenario-name "browser-automation-studio" \
  --test-dir "$TEST_DIR" \
  --default-manage-runtime true

# Register phases
testing::runner::register_phase --name unit --script "$TEST_DIR/phases/test-unit.sh" --timeout 120
testing::runner::register_phase --name integration --script "$TEST_DIR/phases/test-integration.sh" --timeout 240

testing::runner::execute "$@"
```

**Phase script**: `test/phases/test-unit.sh`

```bash
#!/bin/bash
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

# Run Go + vitest tests
if testing::unit::run_all_tests --go-dir "api" --node-dir "ui"; then
  # PROBLEM: Hardcoded requirement tracking
  for requirement_id in "${UNIT_REQUIREMENTS[@]}"; do
    testing::phase::add_requirement --id "$requirement_id" --status passed
  done
fi

testing::phase::end_with_summary
```

**Output**: `coverage/phase-results/unit.json`

```json
{
  "phase": "unit",
  "scenario": "browser-automation-studio",
  "status": "passed",
  "tests": 42,
  "duration_seconds": 87,
  "requirements": [
    {
      "id": "BAS-WORKFLOW-PERSIST-CRUD",
      "status": "passed",
      "criticality": "P0",
      "evidence": "Unit test suites"  // Generic! No specifics!
    }
  ]
}
```

#### 3. Requirements Auto-Sync

**After tests run**: `test/run-tests.sh` calls reporter

```bash
if node "$APP_ROOT/scripts/requirements/report.js" --scenario "$SCENARIO_NAME" --mode sync; then
  log::info "📋 Requirements registry synced after test run"
fi
```

**Reporter logic** (`scripts/requirements/report.js`):

1. Reads `requirements/*.yaml` files
2. Reads `coverage/phase-results/*.json` files
3. Matches requirement IDs between them
4. Derives requirement status from live evidence:
   - All validations passed → `status: complete`
   - Some passed, some not → `status: in_progress`
   - All failed → `status: in_progress`
5. Auto-updates `requirements/*.yaml` files with new status

**This works!** But only as good as the evidence it receives.

#### 4. Vitest Test Suite

**Current tests**: `ui/src/**/__tests__/*.test.ts`

```typescript
// ui/src/stores/__tests__/projectStore.test.ts
import { describe, it, expect } from 'vitest';
import { useProjectStore } from '../projectStore';

describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects successfully', async () => {
    const mockProjects = [{ id: '1', name: 'Test' }];
    // ... test implementation
    expect(result).toEqual(mockProjects);
  });

  it('creates project successfully', async () => {
    // ... test implementation
  });
});
```

**Vitest config**: `ui/vitest.config.ts`

```typescript
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
    },
    reporters: ['default'],  // Only standard reporter!
  },
});
```

**The problem**: No custom reporter to extract `[REQ:ID]` tags.

### What's Missing

| **Component** | **Status** | **Gap** |
|---------------|-----------|---------|
| Requirements registry | ✅ Complete | None |
| Phase-based testing | ✅ Complete | None |
| Test evidence tracking | ⚠️ Manual | No automatic test→requirement extraction |
| Vitest requirement reporter | ❌ Missing | Need custom reporter |
| Go test requirement parser | ❌ Missing | Future enhancement |
| Phase script automation | ⚠️ Hardcoded | Remove manual requirement lists |

---

## Solution Architecture

### Design Principles

1. **Shared infrastructure**: One reporter package for all scenarios
2. **Minimal boilerplate**: Scenarios add 3 lines to config, that's it
3. **Conventional tagging**: Simple `[REQ:ID]` pattern in test names
4. **Zero runtime overhead**: Reporter only runs during test execution
5. **Backward compatible**: Existing tests work without tags
6. **Framework-agnostic**: Same pattern for vitest, Go, Python, etc.

### Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                    Shared Reporter Package                        │
│  packages/vitest-requirement-reporter/                           │
│    ├── src/reporter.ts     (Main reporter implementation)        │
│    ├── src/types.ts        (TypeScript interfaces)               │
│    └── package.json        (Shared dependency)                   │
└──────────────────────────────────────────────────────────────────┘
                                ↓ imported by
┌──────────────────────────────────────────────────────────────────┐
│                 Scenario Vitest Configuration                     │
│  scenarios/*/ui/vitest.config.ts                                 │
│    import RequirementReporter from '@vrooli/vitest-...'          │
│    reporters: ['default', new RequirementReporter()]             │
└──────────────────────────────────────────────────────────────────┘
                                ↓ runs during
┌──────────────────────────────────────────────────────────────────┐
│                      Test Execution                               │
│  npm run test (or vitest)                                        │
│    → Vitest executes tests                                       │
│    → Reporter extracts [REQ:ID] from test names                  │
│    → Reporter tracks pass/fail per requirement                   │
│    → Outputs: ui/coverage/vitest-requirements.json               │
└──────────────────────────────────────────────────────────────────┘
                                ↓ parsed by
┌──────────────────────────────────────────────────────────────────┐
│                  Shared Unit Test Orchestrator                    │
│  scripts/scenarios/testing/unit/parse-requirements.sh            │
│    testing::unit::parse_vitest_requirements()                    │
│      → Reads vitest-requirements.json                            │
│      → Calls testing::phase::add_requirement for each            │
└──────────────────────────────────────────────────────────────────┘
                                ↓ aggregated by
┌──────────────────────────────────────────────────────────────────┐
│                     Phase Result Output                           │
│  coverage/phase-results/unit.json                                │
│    "requirements": [                                              │
│      {                                                            │
│        "id": "BAS-WORKFLOW-PERSIST-CRUD",                         │
│        "status": "passed",                                        │
│        "evidence": "Vitest (3 tests): projectStore.test.ts:..."  │
│      }                                                            │
│    ]                                                              │
└──────────────────────────────────────────────────────────────────┘
                                ↓ consumed by
┌──────────────────────────────────────────────────────────────────┐
│                  Requirements Auto-Sync                           │
│  scripts/requirements/report.js --mode sync                       │
│    → Reads phase-results/*.json                                  │
│    → Matches with requirements/*.yaml                            │
│    → Auto-updates status fields                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Data Flow

```mermaid
graph TD
    A[Developer writes test with [REQ:ID] tag] --> B[npm run test]
    B --> C[Vitest executes tests]
    C --> D[RequirementReporter extracts tags]
    D --> E[Outputs vitest-requirements.json]
    E --> F[testing::unit::run_all_tests calls parser]
    F --> G[parse_vitest_requirements reads JSON]
    G --> H[testing::phase::add_requirement for each]
    H --> I[testing::phase::end_with_summary]
    I --> J[Writes coverage/phase-results/unit.json]
    J --> K[test/run-tests.sh completes]
    K --> L[report.js --mode sync]
    L --> M[Auto-updates requirements/*.yaml status]
```

### File Structure

```
Vrooli/
├── packages/
│   └── vitest-requirement-reporter/      # NEW: Shared package
│       ├── package.json
│       ├── tsconfig.json
│       ├── src/
│       │   ├── reporter.ts               # Main reporter class
│       │   ├── types.ts                  # TypeScript interfaces
│       │   └── index.ts                  # Package exports
│       ├── dist/                         # Compiled output
│       │   ├── reporter.js
│       │   ├── reporter.d.ts
│       │   └── types.d.ts
│       └── README.md
│
├── scripts/
│   └── scenarios/testing/
│       └── unit/
│           ├── run-all.sh                # EXISTING: Multi-language orchestrator
│           └── parse-requirements.sh     # NEW: Parse test outputs
│
└── scenarios/
    └── browser-automation-studio/
        └── ui/
            ├── vitest.config.ts          # UPDATED: Import reporter
            ├── package.json              # UPDATED: Add dependency
            ├── coverage/
            │   ├── vitest-requirements.json  # AUTO-GENERATED
            │   └── phase-results/            # EXISTING
            └── src/**/__tests__/*.test.ts    # Tag with [REQ:ID]
```

---

## Implementation Plan

### Phase 1: Create Shared Reporter Package

**Estimated Time**: 2-3 hours

#### Step 1.1: Create Package Structure

```bash
mkdir -p packages/vitest-requirement-reporter/src
cd packages/vitest-requirement-reporter
```

#### Step 1.2: Package Configuration

**File**: `packages/vitest-requirement-reporter/package.json`

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
    },
    "./types": {
      "types": "./dist/types.d.ts",
      "default": "./dist/types.js"
    }
  },
  "files": [
    "dist",
    "src",
    "README.md"
  ],
  "scripts": {
    "build": "tsc",
    "dev": "tsc --watch",
    "prepublishOnly": "npm run build"
  },
  "keywords": [
    "vitest",
    "reporter",
    "requirements",
    "testing",
    "coverage"
  ],
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

#### Step 1.3: TypeScript Configuration

**File**: `packages/vitest-requirement-reporter/tsconfig.json`

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

#### Step 1.4: Type Definitions

**File**: `packages/vitest-requirement-reporter/src/types.ts`

```typescript
/**
 * Result for a single requirement tracked by the reporter
 */
export interface RequirementResult {
  /** Requirement ID (e.g., BAS-WORKFLOW-PERSIST-CRUD) */
  id: string;

  /** Worst status across all tests for this requirement */
  status: 'passed' | 'failed' | 'skipped';

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

  /** Number of tests that were skipped */
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
}
```

#### Step 1.5: Reporter Implementation

**File**: `packages/vitest-requirement-reporter/src/reporter.ts`

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
        ids.forEach(id => requirements.add(id.trim()));
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

    const status =
      task.result?.state === 'pass' ? 'passed' :
      task.result?.state === 'skip' ? 'skipped' : 'failed';

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
        // Worst status wins: failed > skipped > passed
        if (status === 'failed' || (status === 'skipped' && existing.status === 'passed')) {
          existing.status = status;
        }

        // Accumulate evidence and metrics
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

    const skipped = report.requirements.filter(r => r.status === 'skipped');
    if (skipped.length > 0) {
      console.log(`   ⏭️  Skipped: ${skipped.map(r => r.id).join(', ')}`);
    }

    console.log(`   Output: ${this.options.outputFile}\n`);
  }
}
```

#### Step 1.6: Package Index

**File**: `packages/vitest-requirement-reporter/src/index.ts`

```typescript
export { default } from './reporter.js';
export type {
  RequirementReporterOptions,
  RequirementResult,
  RequirementReport,
} from './types.js';
```

#### Step 1.7: Documentation

**File**: `packages/vitest-requirement-reporter/README.md`

```markdown
# @vrooli/vitest-requirement-reporter

Custom Vitest reporter for tracking requirement coverage in Vrooli scenarios.

## Installation

```bash
pnpm add @vrooli/vitest-requirement-reporter@workspace:*
```

## Usage

### Basic Setup

**vitest.config.ts**:
```typescript
import { defineConfig } from 'vitest/config';
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  test: {
    reporters: [
      'default',
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
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
describe('projectStore', () => {
  it('fetches projects [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => { ... });
  it('validates names [REQ:BAS-PROJECT-VALIDATION]', async () => { ... });
});

// Multiple requirements
describe('CRUD operations [REQ:BAS-WORKFLOW-PERSIST-CRUD, BAS-PROJECT-API]', () => {
  it('handles all operations', async () => { ... });
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
      "evidence": "ui/src/stores/__tests__/projectStore.test.ts:projectStore > fetches projects; ...",
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
| `pattern` | `RegExp` | `/\[REQ:...\]/gi` | Custom extraction pattern |

## Integration with Phase Testing

This reporter outputs data consumed by `scripts/scenarios/testing/unit/parse-requirements.sh`, which automatically reports requirements to phase helpers.

No manual configuration needed in phase scripts!
```

#### Step 1.8: Build Package

```bash
cd packages/vitest-requirement-reporter
npm install
npm run build
```

**Verify output**:
```bash
ls -la dist/
# Should see:
# reporter.js
# reporter.d.ts
# types.js
# types.d.ts
```

---

### Phase 2: Create Shared Requirement Parser

**Estimated Time**: 1-2 hours

#### Step 2.1: Create Parser Script

**File**: `scripts/scenarios/testing/unit/parse-requirements.sh`

```bash
#!/usr/bin/env bash
# Parse requirement outputs from test frameworks and report to phase helpers
set -euo pipefail

# Parse vitest requirement report and record to phase
testing::unit::parse_vitest_requirements() {
    local report_file="${1:-ui/coverage/vitest-requirements.json}"

    if [ ! -f "$report_file" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        log::warning "jq not available; skipping vitest requirement parsing"
        return 0
    fi

    local req_count=0
    while IFS= read -r req_entry; do
        local req_id=$(echo "$req_entry" | jq -r '.id')
        local req_status=$(echo "$req_entry" | jq -r '.status')
        local req_evidence=$(echo "$req_entry" | jq -r '.evidence')
        local test_count=$(echo "$req_entry" | jq -r '.test_count')

        testing::phase::add_requirement \
            --id "$req_id" \
            --status "$req_status" \
            --evidence "Vitest (${test_count} tests): ${req_evidence}"

        ((req_count++))
    done < <(jq -c '.requirements[]' "$report_file" 2>/dev/null)

    if [ $req_count -gt 0 ]; then
        log::success "📋 Parsed $req_count requirements from vitest"
    fi
}

# Parse Go test requirement comments (future enhancement)
# Expected format:
#   // REQ: BAS-WORKFLOW-PERSIST-CRUD
#   func TestWorkflowPersistence(t *testing.T) { ... }
testing::unit::parse_go_requirements() {
    local go_dir="${1:-api}"

    # TODO: Implement go test -json parser
    # For now, return 0 (no-op)
    return 0
}

export -f testing::unit::parse_vitest_requirements
export -f testing::unit::parse_go_requirements
```

#### Step 2.2: Update Unit Test Orchestrator

**File**: `scripts/scenarios/testing/unit/run-all.sh`

Find the end of `testing::unit::run_all_tests()` function and add:

```bash
testing::unit::run_all_tests() {
    # ... existing implementation ...

    local result=$?

    # NEW: Parse requirement outputs if phase helpers are available
    if declare -F testing::phase::add_requirement >/dev/null 2>&1; then
        source "${APP_ROOT}/scripts/scenarios/testing/unit/parse-requirements.sh"

        # Parse vitest requirements if node tests ran
        if [ "$skip_node" != "true" ] && [ -n "$node_dir" ]; then
            testing::unit::parse_vitest_requirements "${node_dir}/coverage/vitest-requirements.json"
        fi

        # Parse Go requirements if Go tests ran
        if [ "$skip_go" != "true" ] && [ -n "$go_dir" ]; then
            testing::unit::parse_go_requirements "$go_dir"
        fi
    fi

    return $result
}
```

---

### Phase 3: Integrate into BAS

**Estimated Time**: 1 hour

#### Step 3.1: Update pnpm Workspace

**File**: `pnpm-workspace.yaml`

Ensure it includes:

```yaml
packages:
  - 'packages/*'
  - 'scenarios/*/ui'
```

#### Step 3.2: Add Package Dependency

```bash
cd scenarios/browser-automation-studio/ui
pnpm add @vrooli/vitest-requirement-reporter@workspace:*
```

#### Step 3.3: Update Vitest Config

**File**: `scenarios/browser-automation-studio/ui/vitest.config.ts`

```typescript
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      reportsDirectory: './coverage',
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/__tests__/**',
        'src/test/**',
      ],
    },
    reporters: [
      'default',
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
        // scenario auto-detected from cwd
        verbose: true,
      }),
    ],
  },
});
```

#### Step 3.4: Simplify Unit Phase Script

**File**: `scenarios/browser-automation-studio/test/phases/test-unit.sh`

**Before** (44 lines with hardcoded requirements):

```bash
#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# HARDCODED requirement list!
UNIT_REQUIREMENTS=(
  BAS-WORKFLOW-PERSIST-CRUD
  BAS-EXEC-TELEMETRY-STREAM
  BAS-REPLAY-TIMELINE-PERSISTENCE
  BAS-REPLAY-EXPORT-BUNDLE
  BAS-VERSION-AUTOSAVE
)

if testing::unit::run_all_tests \
    --go-dir "api" \
    --node-dir "ui" \
    --skip-python \
    --coverage-warn 40 \
    --coverage-error 30 \
    --verbose; then
  testing::phase::add_test passed
  # Manually mark all requirements
  for requirement_id in "${UNIT_REQUIREMENTS[@]}"; do
    testing::phase::add_requirement --id "$requirement_id" --status passed --evidence "Unit test suites"
  done
else
  testing::phase::add_error "Unit test runner reported failures"
  testing::phase::add_test failed
  for requirement_id in "${UNIT_REQUIREMENTS[@]}"; do
    testing::phase::add_requirement --id "$requirement_id" --status failed --evidence "Unit test suites"
  done
fi

testing::phase::end_with_summary "Unit tests completed"
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

#### Step 3.5: Tag Existing Tests

**File**: `scenarios/browser-automation-studio/ui/src/stores/__tests__/projectStore.test.ts`

**Before**:
```typescript
describe('projectStore', () => {
  it('fetches projects successfully', async () => { ... });
});
```

**After**:
```typescript
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects successfully', async () => { ... });
  it('creates project successfully', async () => { ... });
  it('updates project successfully', async () => { ... });
});
```

Repeat for other test files based on `requirements/` mappings.

---

### Phase 4: Testing & Validation

**Estimated Time**: 1 hour

#### Step 4.1: Unit Test the Reporter

```bash
cd scenarios/browser-automation-studio/ui
npm run test
```

**Expected output**:
```
✓ src/stores/__tests__/projectStore.test.ts (3)
✓ src/components/__tests__/ProjectModal.test.tsx (2)

Test Files  2 passed (2)
     Tests  5 passed (5)

📋 Requirement Coverage Report:
   Scenario: browser-automation-studio
   Requirements: 2 covered
   Tests: 5/5 passed
   ✅ BAS-WORKFLOW-PERSIST-CRUD (3 tests, 187ms)
   ✅ BAS-PROJECT-UI-COMPONENTS (2 tests, 94ms)
   Output: coverage/vitest-requirements.json
```

**Verify output file**:
```bash
cat coverage/vitest-requirements.json
```

#### Step 4.2: Phase Test Integration

```bash
cd scenarios/browser-automation-studio
./test/run-tests.sh unit
```

**Expected**:
1. Unit phase runs vitest tests
2. Reporter creates `ui/coverage/vitest-requirements.json`
3. Parser reads JSON and calls `testing::phase::add_requirement`
4. Phase summary shows individual requirements
5. `coverage/phase-results/unit.json` includes requirements with vitest evidence

**Verify phase results**:
```bash
cat coverage/phase-results/unit.json | jq '.requirements'
```

Expected:
```json
[
  {
    "id": "BAS-WORKFLOW-PERSIST-CRUD",
    "status": "passed",
    "criticality": "P0",
    "evidence": "Vitest (3 tests): ui/src/stores/__tests__/projectStore.test.ts:projectStore > fetches projects; ..."
  }
]
```

#### Step 4.3: Full Test Suite

```bash
./test/run-tests.sh
```

**Verify**:
1. All phases run successfully
2. `coverage/phase-results/unit.json` has detailed vitest requirements
3. After run completes, `node scripts/requirements/report.js --mode sync` runs
4. `requirements/projects/dialog.yaml` status fields are auto-updated

---

### Phase 5: Documentation

**Estimated Time**: 1 hour

#### Step 5.1: Create Tagging Guide

**File**: `docs/testing/REQUIREMENT_TAGGING.md`

```markdown
# Requirement Tagging Guide

## Overview

Vrooli's testing infrastructure automatically tracks which tests validate which requirements using simple `[REQ:ID]` tags in test names.

## Vitest Tests

### Basic Tagging

Tag suite or test names with `[REQ:ID]`:

```typescript
// Suite-level (all tests inherit)
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects', async () => { ... });
  it('creates project', async () => { ... });
});

// Test-level
describe('projectStore', () => {
  it('fetches [REQ:BAS-WORKFLOW-PERSIST-CRUD]', async () => { ... });
  it('validates [REQ:BAS-PROJECT-VALIDATION]', async () => { ... });
});
```

### Multiple Requirements

Comma-separated for tests that validate multiple requirements:

```typescript
describe('CRUD operations [REQ:BAS-WORKFLOW-PERSIST-CRUD, BAS-PROJECT-API]', () => {
  it('handles all operations', async () => { ... });
});
```

### Finding Requirement IDs

1. Check `requirements/index.yaml` for top-level requirements
2. Check domain-specific files like `requirements/projects/dialog.yaml`
3. Search: `grep -r "id: BAS-" requirements/`

## Go Tests (Future)

Use comments to tag Go tests:

```go
// TestWorkflowPersistence validates BAS-WORKFLOW-PERSIST-CRUD
func TestWorkflowPersistence(t *testing.T) {
    // REQ: BAS-WORKFLOW-PERSIST-CRUD
    t.Run("create workflow", func(t *testing.T) { ... })
}
```

## Verification

After tagging, run tests and check output:

```bash
npm run test  # In ui/ directory

# Look for:
# 📋 Requirement Coverage Report:
#    Requirements: X covered
```

Or check generated file:

```bash
cat coverage/vitest-requirements.json
```

## Integration

Tagged requirements automatically flow through the testing pipeline:

1. Test runs with tag
2. Reporter extracts requirement ID
3. Phase helper records evidence
4. Requirements registry auto-syncs status

No manual tracking needed!
```

#### Step 5.2: Update Architecture Docs

**File**: `docs/testing/architecture/PHASED_TESTING.md`

Add section after "Phase 3: Unit":

```markdown
### Automatic Requirement Tracking

The unit phase automatically correlates test results with requirements using the `@vrooli/vitest-requirement-reporter` package.

**How it works**:

1. Developer tags tests with `[REQ:ID]` in test names
2. Vitest reporter extracts tags during test execution
3. Reporter outputs `coverage/vitest-requirements.json`
4. Phase orchestrator parses output and reports to phase helpers
5. Phase results include per-requirement evidence
6. Requirements registry auto-syncs status after test run

**No manual tracking required!**

See [Requirement Tagging Guide](../REQUIREMENT_TAGGING.md) for usage.
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
// vitest.config.ts
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  test: {
    reporters: [
      'default',
      new RequirementReporter(),  // Uses smart defaults
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

**Audit existing tests**:

```bash
# Find all test files
find ui/src -name "*.test.*"

# Check which have requirement tags
grep -r "\[REQ:" ui/src/**/*.test.*
```

**Gradual adoption**:

1. Add reporter to vitest.config.ts
2. Run tests - reporter won't find requirements but won't break
3. Tag tests incrementally over time
4. Each tagged test automatically tracked

**No breaking changes** - untagged tests still run normally.

---

## Success Metrics

### Technical Metrics

| **Metric** | **Before** | **After** | **Improvement** |
|------------|-----------|----------|-----------------|
| Lines in test-unit.sh | 44 | 12 | -73% boilerplate |
| Manual requirement lists | 1 per phase | 0 | 100% eliminated |
| Requirement evidence detail | "Unit test suites" | "Vitest (3 tests): projectStore.test.ts:..." | Specific test paths |
| Sync accuracy | Manual updates | Automatic | 100% reliable |
| Developer effort per test | Tag + update phase script | Tag only | 50% reduction |

### Quality Metrics

- **Traceability**: Every requirement links to specific tests
- **Accuracy**: No drift between tests and requirement lists
- **Granularity**: Per-test requirement tracking instead of all-or-nothing
- **Maintainability**: Changes to tests automatically reflected in reports

### Adoption Metrics

- **Scenarios using reporter**: 0 → 1 (BAS) → all vitest scenarios
- **Tagged tests**: 0 → 15 (BAS initial) → 100% coverage
- **False positives**: 0 (tests must pass for requirement to be marked complete)

---

## Appendices

### Appendix A: Example Output Files

#### vitest-requirements.json

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
      "id": "BAS-ERROR-HANDLING",
      "status": "passed",
      "evidence": "ui/src/utils/__tests__/errorHandler.test.ts:errorHandler > catches API errors",
      "duration_ms": 32,
      "test_count": 1
    },
    {
      "id": "BAS-PROJECT-UI-COMPONENTS",
      "status": "passed",
      "evidence": "ui/src/components/__tests__/ProjectModal.test.tsx:ProjectModal > renders correctly; ui/src/components/__tests__/ProjectModal.test.tsx:ProjectModal > handles submit",
      "duration_ms": 94,
      "test_count": 2
    },
    {
      "id": "BAS-PROJECT-VALIDATION",
      "status": "failed",
      "evidence": "ui/src/stores/__tests__/projectStore.test.ts:projectStore > validates project name; ui/src/stores/__tests__/projectStore.test.ts:projectStore > rejects invalid names",
      "duration_ms": 76,
      "test_count": 2
    },
    {
      "id": "BAS-WORKFLOW-PERSIST-CRUD",
      "status": "passed",
      "evidence": "ui/src/stores/__tests__/projectStore.test.ts:projectStore > fetches projects successfully; ui/src/stores/__tests__/projectStore.test.ts:projectStore > creates project successfully; ui/src/stores/__tests__/projectStore.test.ts:projectStore > updates project successfully",
      "duration_ms": 187,
      "test_count": 3
    }
  ]
}
```

#### phase-results/unit.json

```json
{
  "phase": "unit",
  "scenario": "browser-automation-studio",
  "status": "failed",
  "tests": 42,
  "errors": 1,
  "warnings": 0,
  "skipped": 1,
  "duration_seconds": 87,
  "target": "90s",
  "updated_at": "2025-11-04T18:32:18-05:00",
  "requirements": [
    {
      "id": "BAS-ERROR-HANDLING",
      "status": "passed",
      "criticality": "P1",
      "evidence": "Vitest (1 tests): ui/src/utils/__tests__/errorHandler.test.ts:errorHandler > catches API errors"
    },
    {
      "id": "BAS-EXEC-TELEMETRY-STREAM",
      "status": "passed",
      "criticality": "P0",
      "evidence": "Go: api/browserless/client_test.go:TestTelemetryPersistence; Go: api/browserless/runtime/session_test.go:TestSessionPayloads"
    },
    {
      "id": "BAS-PROJECT-UI-COMPONENTS",
      "status": "passed",
      "criticality": "P0",
      "evidence": "Vitest (2 tests): ui/src/components/__tests__/ProjectModal.test.tsx:ProjectModal > renders correctly; ui/src/components/__tests__/ProjectModal.test.tsx:ProjectModal > handles submit"
    },
    {
      "id": "BAS-PROJECT-VALIDATION",
      "status": "failed",
      "criticality": "P0",
      "evidence": "Vitest (2 tests): ui/src/stores/__tests__/projectStore.test.ts:projectStore > validates project name; ui/src/stores/__tests__/projectStore.test.ts:projectStore > rejects invalid names"
    },
    {
      "id": "BAS-WORKFLOW-PERSIST-CRUD",
      "status": "passed",
      "criticality": "P0",
      "evidence": "Vitest (3 tests): ui/src/stores/__tests__/projectStore.test.ts:projectStore > fetches projects successfully; Go: api/services/workflow_service_test.go:TestWorkflowCRUD"
    }
  ]
}
```

### Appendix B: Future Enhancements

#### Go Test Parser

```bash
# scripts/scenarios/testing/unit/parse-go-requirements.sh
testing::unit::parse_go_requirements() {
    local go_dir="${1:-api}"

    # Run go test with JSON output
    local test_output
    test_output=$(cd "$go_dir" && go test -json ./... 2>&1)

    # Parse for requirement comments and correlate with test results
    # Implementation TBD
}
```

#### Coverage Correlation

Enhance reporter to link requirements to code coverage:

```typescript
// In reporter.ts
private async correlateWithCoverage(): Promise<void> {
  const coverageFile = 'coverage/coverage-final.json';
  if (!existsSync(coverageFile)) return;

  const coverage = JSON.parse(readFileSync(coverageFile, 'utf-8'));

  // For each requirement, calculate coverage of related files
  // based on test paths in evidence
}
```

#### Dashboard Visualization

Build requirement coverage dashboard:

```typescript
// UI component showing:
// - Requirement hierarchy
// - Test coverage per requirement
// - Code coverage per requirement
// - Trend over time
```

### Appendix C: Troubleshooting

#### Reporter not outputting JSON

**Symptom**: No `coverage/vitest-requirements.json` file created

**Causes**:
1. Reporter not registered in vitest.config.ts
2. No tests with `[REQ:ID]` tags ran
3. vitest failed before reporter could finish

**Debug**:
```bash
# Check if reporter is registered
cat ui/vitest.config.ts | grep RequirementReporter

# Check for tagged tests
grep -r "\[REQ:" ui/src/**/*.test.*

# Run with verbose logging
npm run test -- --reporter=verbose
```

#### Requirements not appearing in phase results

**Symptom**: `coverage/phase-results/unit.json` has empty requirements array

**Causes**:
1. Parser not being called by orchestrator
2. vitest-requirements.json in wrong location
3. jq not available

**Debug**:
```bash
# Check if parser exists
ls scripts/scenarios/testing/unit/parse-requirements.sh

# Check if vitest output exists
ls ui/coverage/vitest-requirements.json

# Manually run parser
source scripts/scenarios/testing/unit/parse-requirements.sh
testing::unit::parse_vitest_requirements ui/coverage/vitest-requirements.json
```

#### Requirement status not syncing

**Symptom**: `requirements/*.yaml` files not updating after tests

**Causes**:
1. report.js not being called after test run
2. Requirement IDs mismatch between tests and yaml
3. Phase results not written

**Debug**:
```bash
# Manually run sync
node scripts/requirements/report.js --scenario browser-automation-studio --mode sync

# Check phase results exist
ls coverage/phase-results/*.json

# Verify requirement IDs match
grep "id: BAS-" requirements/**/*.yaml
grep "\[REQ:BAS-" ui/src/**/*.test.*
```

---

### Phase 6: Auto-Sync Validation Entries

**Estimated Time**: 3-4 hours

**Goal**: Enhance `scripts/requirements/report.js` to automatically add, update, and optionally prune validation entries in `requirements/*.yaml` files based on live test evidence.

#### Why Phase 6?

Currently `report.js --mode sync` only updates `status:` fields. This phase extends it to:
1. **Auto-add** missing `validation:` entries when tests declare `[REQ:ID]` tags
2. **Auto-update** validation `status:` and `phase:` fields from live results
3. **Detect orphaned** validations (YAML references non-existent files)
4. **Auto-prune** orphaned validations (optional, requires `--prune-stale` flag)
5. **Report gaps** in test coverage for requirements

**Philosophy**: Requirements YAML files remain the source of truth, but automatically stay synchronized with actual test codebase.

#### Step 6.1: Understand Current Sync Logic

**Current behavior** (`syncRequirementFile()` function):

```javascript
// Lines 1159-1230 in report.js
function syncRequirementFile(filePath, requirements) {
  // Reads YAML file line-by-line
  // Updates validation.status based on liveStatus from phase-results
  // Updates requirement.status based on validation statuses
  // Writes modified lines back to file
}
```

**Key insight**: The function already:
- ✅ Tracks line numbers for status fields (`validation.__meta.statusLine`)
- ✅ Derives validation status from live evidence (`liveStatus: 'passed'` → `status: implemented`)
- ✅ Preserves YAML structure and comments
- ✅ Supports multiple validations per requirement

**What's missing**: Adding/removing validation entries (not just updating existing ones).

#### Step 6.2: Design Validation Sync Algorithm

**Input sources**:
1. **Existing validations** from YAML files (requirement.validations array)
2. **Live evidence** from phase-results JSON (evidence strings with test file paths)
3. **File system** to check if validation refs still exist

**Sync logic**:

```javascript
for each requirement:
  // 1. Parse live evidence to extract test file paths
  const liveTestFiles = extractTestFilesFromEvidence(requirement);

  // 2. Build current validation map
  const existingValidations = new Map();
  requirement.validations?.forEach(v => {
    if (v.type === 'test') {
      existingValidations.set(v.ref, v);
    }
  });

  // 3. Detect orphaned validations
  const orphaned = [];
  existingValidations.forEach((validation, ref) => {
    if (!fileExists(ref)) {
      orphaned.push({ ref, validation });
    }
  });

  // 4. Find missing validations
  const missing = [];
  liveTestFiles.forEach(testFile => {
    if (!existingValidations.has(testFile.ref)) {
      missing.push(testFile);
    }
  });

  // 5. Apply changes
  if (options.pruneStale) {
    removeOrphanedValidations(requirement, orphaned);
  }
  addMissingValidations(requirement, missing);
  updateExistingValidations(requirement, liveTestFiles);
```

#### Step 6.3: Implement Evidence Parser

**Add new function** to `scripts/requirements/report.js`:

```javascript
/**
 * Extract test file references from evidence strings
 * @param {Object} requirement - Requirement with validations and live evidence
 * @param {string} scenarioRoot - Path to scenario directory
 * @returns {Array<{ref: string, phase: string, framework: string}>}
 */
function extractTestFilesFromEvidence(requirement, scenarioRoot) {
  const testFiles = [];

  if (!Array.isArray(requirement.validations)) {
    return testFiles;
  }

  requirement.validations.forEach(validation => {
    const evidence = validation.evidence || '';
    const liveStatus = validation.liveStatus || '';
    const phase = validation.phase || 'unit';

    // Skip if no evidence or non-test validation
    if (!evidence || validation.type !== 'test') {
      return;
    }

    // Parse evidence patterns:
    // Vitest: "Vitest (3 tests): ui/src/stores/__tests__/projectStore.test.ts:..."
    // Go: "Go: api/services/workflow_service_test.go:TestWorkflowCRUD"
    // BAS: "Workflow test/playbooks/ui/projects/new-project-create.json executed"

    const vitestMatch = evidence.match(/Vitest.*?:\s*([^:]+\.test\.(ts|tsx|js|jsx|mjs))/);
    if (vitestMatch) {
      testFiles.push({
        ref: vitestMatch[1],
        phase,
        framework: 'vitest',
        status: liveStatus,
      });
      return;
    }

    const goMatch = evidence.match(/Go:\s*([^:]+_test\.go)/);
    if (goMatch) {
      testFiles.push({
        ref: goMatch[1],
        phase,
        framework: 'go',
        status: liveStatus,
      });
      return;
    }

    const basMatch = evidence.match(/Workflow\s+(test\/playbooks\/[^)]+\.json)/);
    if (basMatch) {
      testFiles.push({
        ref: basMatch[1],
        phase,
        framework: 'bas-workflow',
        status: liveStatus,
      });
      return;
    }
  });

  return testFiles;
}

/**
 * Check if a validation reference file exists on disk
 * @param {string} ref - Relative file path
 * @param {string} scenarioRoot - Scenario directory
 * @returns {boolean}
 */
function validationFileExists(ref, scenarioRoot) {
  if (!ref || !scenarioRoot) {
    return false;
  }

  const absolutePath = path.join(scenarioRoot, ref);
  return fs.existsSync(absolutePath);
}
```

#### Step 6.4: Implement Validation Entry Insertion

**Challenge**: YAML files are parsed line-by-line for status updates, but adding validation entries requires structural insertion.

**Approach**: Use hybrid strategy:
1. Parse YAML to AST for structural changes (add/remove validations)
2. Use line-based updates for status fields (preserve comments/formatting)

**Add dependency** to `package.json` (if not already present):

```bash
cd /home/matthalloran8/Vrooli
npm install js-yaml --save
```

**Add new function**:

```javascript
const yaml = require('js-yaml');

/**
 * Add missing validation entries to requirement YAML
 * @param {string} filePath - Path to requirements YAML file
 * @param {Array} requirements - Requirements with missing validations
 * @param {Object} options - Sync options
 * @returns {Array} - Array of changes made
 */
function addMissingValidations(filePath, requirements, scenarioRoot, options) {
  const changes = [];

  // Read and parse YAML
  const originalContent = fs.readFileSync(filePath, 'utf8');
  const doc = yaml.load(originalContent);

  if (!doc || !Array.isArray(doc.requirements)) {
    return changes;
  }

  // Track modifications
  let modified = false;

  doc.requirements.forEach((requirement, reqIndex) => {
    const matchingReq = requirements.find(r => r.id === requirement.id);
    if (!matchingReq) {
      return;
    }

    // Extract live test files from evidence
    const liveTestFiles = extractTestFilesFromEvidence(matchingReq, scenarioRoot);
    if (liveTestFiles.length === 0) {
      return;
    }

    // Ensure validation array exists
    if (!Array.isArray(requirement.validation)) {
      requirement.validation = [];
    }

    // Build set of existing refs
    const existingRefs = new Set(
      requirement.validation
        .filter(v => v.type === 'test')
        .map(v => v.ref)
    );

    // Add missing validations
    liveTestFiles.forEach(testFile => {
      if (existingRefs.has(testFile.ref)) {
        return; // Already exists
      }

      // Determine initial status based on live status
      const validationStatus =
        testFile.status === 'passed' ? 'implemented' :
        testFile.status === 'failed' ? 'failing' :
        testFile.status === 'skipped' ? 'planned' : 'not_implemented';

      const newValidation = {
        type: 'test',
        ref: testFile.ref,
        phase: testFile.phase,
        status: validationStatus,
        notes: `Auto-added from ${testFile.framework} evidence`,
      };

      requirement.validation.push(newValidation);
      modified = true;

      changes.push({
        type: 'validation_added',
        requirement: requirement.id,
        ref: testFile.ref,
        phase: testFile.phase,
        framework: testFile.framework,
        file: filePath,
      });
    });
  });

  // Write back if modified
  if (modified) {
    const newContent = yaml.dump(doc, {
      indent: 2,
      lineWidth: 100,
      noRefs: true,
      sortKeys: false,
    });

    fs.writeFileSync(filePath, newContent, 'utf8');
  }

  return changes;
}
```

#### Step 6.5: Implement Orphaned Validation Detection

**Add new function**:

```javascript
/**
 * Detect and optionally remove orphaned validation entries
 * @param {string} filePath - Path to requirements YAML file
 * @param {Array} requirements - Requirements to check
 * @param {string} scenarioRoot - Scenario directory
 * @param {Object} options - Sync options (pruneStale flag)
 * @returns {Object} - { orphaned: Array, removed: Array }
 */
function detectOrphanedValidations(filePath, requirements, scenarioRoot, options) {
  const orphaned = [];
  const removed = [];

  // Read and parse YAML
  const originalContent = fs.readFileSync(filePath, 'utf8');
  const doc = yaml.load(originalContent);

  if (!doc || !Array.isArray(doc.requirements)) {
    return { orphaned, removed };
  }

  let modified = false;

  doc.requirements.forEach(requirement => {
    if (!Array.isArray(requirement.validation)) {
      return;
    }

    // Check each test validation
    const validValidations = [];

    requirement.validation.forEach(validation => {
      // Only check test validations (keep automation, manual, etc.)
      if (validation.type !== 'test') {
        validValidations.push(validation);
        return;
      }

      // Check if file exists
      if (validationFileExists(validation.ref, scenarioRoot)) {
        validValidations.push(validation);
      } else {
        // File doesn't exist
        orphaned.push({
          requirement: requirement.id,
          ref: validation.ref,
          phase: validation.phase,
          status: validation.status,
          file: filePath,
        });

        if (options.pruneStale) {
          // Don't add to valid list (removes it)
          modified = true;
          removed.push({
            requirement: requirement.id,
            ref: validation.ref,
            file: filePath,
          });
        } else {
          // Keep it but flag in orphaned list
          validValidations.push(validation);
        }
      }
    });

    // Update validation array if pruning
    if (modified) {
      requirement.validation = validValidations;
    }
  });

  // Write back if modified
  if (modified) {
    const newContent = yaml.dump(doc, {
      indent: 2,
      lineWidth: 100,
      noRefs: true,
      sortKeys: false,
    });

    fs.writeFileSync(filePath, newContent, 'utf8');
  }

  return { orphaned, removed };
}
```

#### Step 6.6: Integrate into Main Sync Flow

**Update `syncRequirementRegistry()` function**:

```javascript
function syncRequirementRegistry(fileRequirementMap, scenarioRoot, options) {
  const updates = [];
  const addedValidations = [];
  const orphanedValidations = [];
  const removedValidations = [];

  for (const [filePath, requirements] of fileRequirementMap.entries()) {
    // Phase 1: Detect and optionally remove orphaned validations
    const orphanResult = detectOrphanedValidations(filePath, requirements, scenarioRoot, options);
    orphanedValidations.push(...orphanResult.orphaned);
    removedValidations.push(...orphanResult.removed);

    // Phase 2: Add missing validations from live evidence
    const added = addMissingValidations(filePath, requirements, scenarioRoot, options);
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

#### Step 6.7: Update CLI Argument Parser

**Add `--prune-stale` flag** to `parseArgs()`:

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

#### Step 6.8: Enhance Sync Report Output

**Update console output** when `--mode sync`:

```javascript
function printSyncSummary(syncResult) {
  console.log('\n📋 Requirements Sync Report:\n');

  // Status updates (existing)
  if (syncResult.statusUpdates.length > 0) {
    console.log('✏️  Status Updates:');
    syncResult.statusUpdates.forEach(update => {
      if (update.type === 'requirement') {
        console.log(`   ${update.requirement}: status → ${update.status}`);
      } else if (update.type === 'validation') {
        console.log(`   ${update.requirement} validation[${update.index}]: status → ${update.status}`);
      }
    });
    console.log();
  }

  // Added validations (new)
  if (syncResult.addedValidations.length > 0) {
    console.log('✅ Added Validations:');
    syncResult.addedValidations.forEach(added => {
      console.log(`   ${added.requirement}`);
      console.log(`     + ${added.ref} (${added.framework}, phase: ${added.phase})`);
    });
    console.log();
  }

  // Orphaned validations (new)
  if (syncResult.orphanedValidations.length > 0) {
    console.log('⚠️  Orphaned Validations (file not found):');
    syncResult.orphanedValidations.forEach(orphan => {
      console.log(`   ${orphan.requirement}`);
      console.log(`     × ${orphan.ref} (phase: ${orphan.phase})`);
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
      console.log(`   ${removed.requirement}`);
      console.log(`     - ${removed.ref}`);
    });
    console.log();
  }

  // Coverage gaps (new)
  const requirementsWithoutTests = findRequirementsWithoutTestCoverage(syncResult);
  if (requirementsWithoutTests.length > 0) {
    console.log('💡 Requirements Without Test Coverage:');
    requirementsWithoutTests.forEach(req => {
      const validationTypes = req.validations?.map(v => v.type).join(', ') || 'none';
      console.log(`   ${req.id} (has: ${validationTypes})`);
    });
    console.log();
  }
}

function findRequirementsWithoutTestCoverage(syncResult) {
  // Implementation: find requirements that have no validation.type === 'test'
  // This would require passing requirements data to printSyncSummary
  // Left as exercise for full implementation
  return [];
}
```

#### Step 6.9: Update Main Function

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

#### Step 6.10: Testing & Validation

**Test Case 1: Auto-add missing validations**

1. Tag a test with `[REQ:BAS-NEW-REQUIREMENT]`
2. Ensure requirement exists in YAML but has no validation entries
3. Run tests: `npm run test`
4. Run sync: `node scripts/requirements/report.js --scenario browser-automation-studio --mode sync`
5. Verify YAML now has validation entry with correct ref, phase, status

**Test Case 2: Detect orphaned validations**

1. Add fake validation to YAML: `ref: ui/src/fake/DoesNotExist.test.ts`
2. Run sync: `node scripts/requirements/report.js --scenario browser-automation-studio --mode sync`
3. Verify console shows orphaned warning
4. Verify YAML unchanged (no --prune-stale flag)

**Test Case 3: Prune orphaned validations**

1. Add fake validation to YAML
2. Run sync with prune: `node scripts/requirements/report.js --scenario browser-automation-studio --mode sync --prune-stale`
3. Verify console shows removal
4. Verify YAML no longer has fake validation entry

**Test Case 4: Update existing validation status**

1. Tag test with existing requirement that has validation entry
2. Make test fail
3. Run tests (should fail)
4. Run sync
5. Verify validation status changed to `failing`
6. Fix test, re-run
7. Run sync again
8. Verify validation status changed to `implemented`

**Test Case 5: Preserve manual validations**

1. Ensure requirement has mix of validation types:
   - `type: test` (auto-managed)
   - `type: automation` (manual)
   - `type: manual` (manual)
2. Run sync
3. Verify only test validations are modified
4. Verify automation/manual entries untouched

#### Step 6.11: Documentation Updates

**Update help text** in `parseArgs()`:

```javascript
function printUsage() {
  console.log(`
Usage: node scripts/requirements/report.js [options]

Options:
  --scenario NAME         Scenario name (required)
  --mode MODE             Operation mode: report, sync, validate, phase-inspect
  --format FORMAT         Output format: json, markdown, trace
  --prune-stale           Remove orphaned validation entries during sync
  --include-pending       Include pending requirements in report
  --output FILE           Write output to file instead of stdout
  --phase NAME            Filter to specific phase (for phase-inspect mode)

Modes:
  report          Generate requirement coverage report (default)
  sync            Auto-update YAML files based on live test evidence
  validate        Check for broken references and missing data
  phase-inspect   Extract requirements for specific test phase

Examples:
  # Generate JSON report
  node scripts/requirements/report.js --scenario browser-automation-studio

  # Sync validations (conservative - only add/update)
  node scripts/requirements/report.js --scenario browser-automation-studio --mode sync

  # Sync and prune orphaned validations
  node scripts/requirements/report.js --scenario browser-automation-studio --mode sync --prune-stale
  `);
}
```

**Update `docs/testing/REQUIREMENT_TAGGING.md`**:

Add section:

```markdown
## Auto-Sync Validation Entries

After Phase 6 implementation, validation entries are automatically managed:

### What Gets Auto-Added

When you tag a test with `[REQ:ID]`, the sync process will:
1. Detect the test file from phase-results evidence
2. Add a validation entry to the requirement YAML if missing
3. Set appropriate status based on test result

**Before** (`requirements/projects/dialog.yaml`):
\`\`\`yaml
- id: BAS-WORKFLOW-PERSIST-CRUD
  validation: []  # Empty!
\`\`\`

**After running sync**:
\`\`\`yaml
- id: BAS-WORKFLOW-PERSIST-CRUD
  validation:
    - type: test
      ref: ui/src/stores/__tests__/projectStore.test.ts
      phase: unit
      status: implemented
      notes: Auto-added from vitest evidence
\`\`\`

### What Gets Auto-Updated

Existing validation entries get status updates:
- Test passes → `status: implemented`
- Test fails → `status: failing`
- Test skipped → `status: planned`

### What Gets Detected as Orphaned

If a YAML file references a test that doesn't exist:
\`\`\`yaml
validation:
  - type: test
    ref: ui/src/components/__tests__/DeletedComponent.test.ts  # ← Doesn't exist!
\`\`\`

Sync will warn:
\`\`\`
⚠️  Orphaned Validations (file not found):
   BAS-UI-COMPONENTS
     × ui/src/components/__tests__/DeletedComponent.test.ts
\`\`\`

Use `--prune-stale` to remove them automatically.

### What's NOT Auto-Managed

Manual validations are preserved:
- `type: automation` - BAS workflows, manual test procedures
- `type: manual` - Human verification steps
- Custom notes and metadata

Only `type: test` entries are auto-managed.
\`\`\`

#### Step 6.12: Integration with Test Runner

**No changes needed!** The test runner already calls:

```bash
node scripts/requirements/report.js --scenario "$SCENARIO_NAME" --mode sync
```

This will now automatically add/update validations in addition to updating statuses.

---

## Checklist

### Phase 1: Package Creation
- [ ] Create `packages/vitest-requirement-reporter/` directory
- [ ] Write `package.json` with dependencies
- [ ] Write `tsconfig.json` for TypeScript compilation
- [ ] Implement `src/types.ts` with interfaces
- [ ] Implement `src/reporter.ts` with main logic
- [ ] Write `src/index.ts` export file
- [ ] Write `README.md` documentation
- [ ] Run `npm install`
- [ ] Run `npm run build` and verify dist/ output
- [ ] Test reporter in isolation with mock vitest data

### Phase 2: Parser Creation
- [ ] Create `scripts/scenarios/testing/unit/parse-requirements.sh`
- [ ] Implement `testing::unit::parse_vitest_requirements()`
- [ ] Add stub `testing::unit::parse_go_requirements()`
- [ ] Update `scripts/scenarios/testing/unit/run-all.sh`
- [ ] Add parser calls after test execution
- [ ] Export functions properly
- [ ] Test parser with sample JSON

### Phase 3: BAS Integration
- [ ] Update `pnpm-workspace.yaml` if needed
- [ ] Add reporter dependency to BAS ui/package.json
- [ ] Run `pnpm install` in BAS ui directory
- [ ] Update `ui/vitest.config.ts` with reporter
- [ ] Simplify `test/phases/test-unit.sh`
- [ ] Tag existing tests with `[REQ:ID]`
- [ ] Remove hardcoded UNIT_REQUIREMENTS array

### Phase 4: Testing
- [ ] Run `npm run test` in ui/ directory
- [ ] Verify console shows requirement coverage report
- [ ] Verify `coverage/vitest-requirements.json` created
- [ ] Run `./test/run-tests.sh unit`
- [ ] Verify parser logs appear
- [ ] Verify `coverage/phase-results/unit.json` has requirements
- [ ] Run full test suite: `./test/run-tests.sh`
- [ ] Verify requirements auto-sync after completion
- [ ] Check `requirements/*.yaml` files for updated status

### Phase 5: Documentation
- [ ] Create `docs/testing/REQUIREMENT_TAGGING.md`
- [ ] Update `docs/testing/architecture/PHASED_TESTING.md`
- [ ] Update `packages/vitest-requirement-reporter/README.md`
- [ ] Add examples to package README
- [ ] Update scenario template if needed

### Phase 6: Auto-Sync Validation Entries
- [ ] Understand current `syncRequirementFile()` logic in report.js
- [ ] Design validation sync algorithm (add/update/prune)
- [ ] Implement `extractTestFilesFromEvidence()` parser
- [ ] Implement `validationFileExists()` file checker
- [ ] Add `js-yaml` dependency to root package.json
- [ ] Implement `addMissingValidations()` function
- [ ] Implement `detectOrphanedValidations()` function
- [ ] Update `syncRequirementRegistry()` to call new functions
- [ ] Add `--prune-stale` flag to `parseArgs()`
- [ ] Implement `printSyncSummary()` for enhanced output
- [ ] Update `main()` to pass scenarioRoot and options
- [ ] Test auto-add missing validations
- [ ] Test detect orphaned validations
- [ ] Test prune orphaned validations with flag
- [ ] Test update existing validation status
- [ ] Test preserve manual validations (automation, manual types)
- [ ] Add help text/usage documentation for --prune-stale
- [ ] Update `docs/testing/REQUIREMENT_TAGGING.md` with auto-sync section
- [ ] Verify integration with test runner (no changes needed)
- [ ] Run full test cycle and verify YAML auto-updates

---

**End of Implementation Plan**
