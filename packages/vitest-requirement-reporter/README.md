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
| `conciseMode` | `boolean` | `false` | Enable concise output with failure artifacts |
| `artifactsDir` | `string` | `'coverage/unit'` | Directory for failure artifacts |
| `autoClear` | `boolean` | `true` (when conciseMode enabled) | Auto-clear artifacts before run |

## Concise Mode (NEW!)

Enable concise output mode for minimal console spam and structured failure artifacts.

**IMPORTANT:** When using `conciseMode: true`, **remove `'default'` from the reporters array**. Our reporter provides all necessary output in a clean format.

```typescript
// ❌ WRONG - Includes 'default' reporter (causes verbose spam)
reporters: [
  'default',
  new RequirementReporter({ conciseMode: true, ... })
]

// ✅ CORRECT - Only custom reporter
reporters: [
  new RequirementReporter({
    outputFile: 'coverage/vitest-requirements.json',
    emitStdout: true,
    verbose: true,
    conciseMode: true,        // Enable concise output
    artifactsDir: 'coverage/unit',
    autoClear: true,
  })
]
```

**Result:** 18x less output (351 lines → 19 lines in real tests)

### Features

**Minimal Console Output:**
```
[INFO]    Running 3 projects: components-palette, components-builder, utils

✅ components-builder: 12/12 passed (2.3s)
✅ utils: 8/8 passed (1.1s)
❌ components-palette: 5/6 passed, 1 failed (0.3s)
   ↳ NodePalette > renders categories and exposes every node definition
   ↳ Read coverage/unit/components-palette/NodePalette-renders-categories/README.md

Summary: 25/26 passed (1 failed) in 3.7s
```

**Structured Failure Artifacts:**
```
coverage/unit/
├── components-palette/
│   └── NodePalette-renders-categories/
│       ├── README.md          # Context-aware failure explanation
│       ├── error.txt          # Full error message
│       ├── stack-trace.txt    # Complete stack trace
│       ├── html-snapshot.html # DOM state (if applicable)
│       └── test-context.json  # Test metadata and timing
```

**Context-Aware README.md:**
- Error summary with first line highlighted
- Requirements tested (with ❌ indicators)
- Likely causes (pattern-matched from error)
- Recent changes from git status
- Deleted files that may be related
- Debug artifact inventory

### Pattern Matching

The reporter analyzes failures and suggests likely causes:

| Error Pattern | Suggested Cause |
|---------------|----------------|
| "Unable to find element" | Element not rendered or query selector incorrect + checks deleted files |
| "Expected X to be Y" | Assertion mismatch - check expected vs actual values |
| "Timeout" / "Timed out" | Async operation not completing - check waitFor conditions |
| "Cannot read property" / "undefined" | Null/undefined access - check component state initialization |

### Auto-Clear

When `autoClear: true`, the artifacts directory is cleared on each test run, ensuring:
- Fresh debug output without accumulation
- No stale artifacts from previous runs
- Disk space management

The artifacts are preserved until the next test run, giving you time to review them.

## Why This Exists

Vrooli's phased testing system automatically tracks which requirements are covered by tests. This reporter extracts `[REQ:ID]` tags from Vitest test names and correlates them with requirement registries, enabling:

- **Automatic validation status updates** - No manual tracking required
- **Live coverage reports** - See which requirements pass/fail in real-time
- **Traceability** - Connect PRD requirements → implementation → tests → results
- **CI/CD gates** - Fail builds when critical requirements lack coverage

Without this reporter, Vitest tests wouldn't integrate with Vrooli's requirement tracking system.

## Integration with Phase Testing

This reporter generates `coverage/vitest-requirements.json` which is consumed by `scripts/scenarios/testing/unit/node.sh`. The phase script automatically calls `testing::phase::add_requirement()` for each tagged test, creating a complete audit trail from requirements → tests → results.

**Flow:**
```
Vitest runs → Reporter extracts [REQ:ID] tags → Writes vitest-requirements.json
→ Phase script reads JSON → Calls testing::phase::add_requirement()
→ Writes coverage/phase-results/unit.json → report.js syncs requirement files
```

**No manual configuration needed in phase scripts!** The integration is automatic once you add the reporter to `vite.config.ts`.

## Tag Inheritance

Tags in `describe` blocks apply to **all child tests**:

```typescript
// ✅ All 3 tests inherit the tag
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects', () => { ... });  // Inherits REQ tag
  it('creates project', () => { ... });   // Inherits REQ tag
  it('updates project', () => { ... });   // Inherits REQ tag
});

// ✅ Nested suites inherit too
describe('CRUD operations [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  describe('create', () => {
    it('validates input', () => { ... });    // Inherits tag
    it('persists to DB', () => { ... });     // Inherits tag
  });
});

// ⚠️ Test-level tags override suite tags
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('validates names [REQ:BAS-PROJECT-VALIDATION]', () => { ... });  // Has BOTH tags
});
```

**Best Practice**: Use suite-level tags for shared requirements, test-level tags for specific requirements.

## Propagating Package Updates

After modifying this reporter inside the monorepo, run the refresh helper so dependent scenarios reinstall and rebuild with the new code:

```bash
./scripts/scenarios/tools/refresh-shared-package.sh vitest-requirement-reporter <scenario|all> [--no-restart]
```

The helper rebuilds the package, filters to the scenarios that include `@vrooli/vitest-requirement-reporter`, runs `vrooli scenario setup`, and restarts just the ones that were already running (unless you pass `--no-restart`).

## Troubleshooting

### Requirements Not Showing in Phase Results

**Symptom:** Tagged tests run but requirements don't appear in `coverage/phase-results/unit.json`

**Possible Causes:**

1. **Reporter not configured**
   ```typescript
   // ❌ Missing reporter
   export default defineConfig({
     test: {
       reporters: ['default']  // No RequirementReporter!
     }
   });

   // ✅ Correct
   import RequirementReporter from '@vrooli/vitest-requirement-reporter';
   export default defineConfig({
     test: {
       reporters: [
         'default',
         new RequirementReporter({ ... })
       ]
     }
   });
   ```

2. **emitStdout disabled**
   ```typescript
   // ❌ Phase integration won't work
   new RequirementReporter({
     emitStdout: false  // WRONG
   })

   // ✅ Required for phase integration
   new RequirementReporter({
     emitStdout: true  // CRITICAL
   })
   ```

3. **Tag pattern doesn't match**
   ```typescript
   // Test uses:
   it('test [REQ:BAS-WORKFLOW-CRUD]', () => { ... })

   // But requirement registry has:
   { "id": "BAS-WORKFLOW-PERSIST-CRUD", ... }  // Different ID!

   // ✅ Must match exactly
   it('test [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => { ... })
   ```

4. **Requirement not in registry**
   - Check `requirements/*.json`
   - Ensure requirement ID exists before tagging tests
   - Run `node scripts/requirements/validate.js --scenario <name>`

### Tags Not Being Detected

**Symptom:** Some tests tagged but not showing in output

**Check:**
1. Tag format: `[REQ:ID]` with capital letters
2. ID pattern: `[A-Z][A-Z0-9]+-[A-Z0-9-]+`
3. Tests actually executed (not skipped)

```typescript
// ❌ Skipped tests are ignored
it.skip('test [REQ:BAS-WORKFLOW-CRUD]', () => { ... })  // Won't track

// ✅ Only executed tests tracked
it('test [REQ:BAS-WORKFLOW-CRUD]', () => { ... })
```

### Output File Not Generated

**Symptom:** `ui/coverage/vitest-requirements.json` doesn't exist

**Causes:**
1. Coverage directory doesn't exist - create it: `mkdir -p ui/coverage`
2. File permissions issue - check write permissions
3. Tests failed before reporter ran - fix test failures first

### Multiple Requirements Not Detected

**Symptom:** Only first requirement in comma-separated list detected

**Check format:**
```typescript
// ✅ Correct - comma with optional space
describe('CRUD [REQ:BAS-WORKFLOW-CRUD, BAS-PROJECT-API]', () => { ... })

// ✅ Also correct
describe('CRUD [REQ:BAS-WORKFLOW-CRUD,BAS-PROJECT-API]', () => { ... })

// ❌ Wrong - semicolon separator
describe('CRUD [REQ:BAS-WORKFLOW-CRUD; BAS-PROJECT-API]', () => { ... })
```

## See Also

### Documentation
- [Requirement Tracking Guide](/docs/testing/guides/requirement-tracking.md) - Complete system overview
- [Phased Testing Architecture](/docs/testing/architecture/PHASED_TESTING.md) - How requirements integrate with phases
- [Requirement Schema Reference](/docs/testing/reference/requirement-schema.md) - Registry structure

### Tools
- [scripts/requirements/report.js](/scripts/requirements/report.js) - Reporting and sync tool
- [scripts/requirements/validate.js](/scripts/requirements/validate.js) - Schema validation
- [scripts/scenarios/testing/unit/node.sh](/scripts/scenarios/testing/unit/node.sh) - Phase integration

### Examples
- [browser-automation-studio tests](/scenarios/browser-automation-studio/ui/src/) - Reference implementation
- [projectStore.test.ts](/scenarios/browser-automation-studio/ui/src/stores/__tests__/projectStore.test.ts) - Suite-level tagging example
