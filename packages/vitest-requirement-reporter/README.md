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
