# @vrooli/vitest-requirement-reporter

Custom Vitest reporter for tracking requirement coverage in Vrooli scenarios.

## Installation

```bash
pnpm add @vrooli/vitest-requirement-reporter@file:../../../packages/vitest-requirement-reporter
```

## Quick Setup

```typescript
import { defineConfig } from 'vitest/config';
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

export default defineConfig({
  test: {
    reporters: [
      'default',
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
        emitStdout: true,
        verbose: true,
      }),
    ],
  },
});
```

## Tagging Tests

Tag tests with `[REQ:ID]` in describe/it names:

```typescript
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects', async () => { ... });
  it('creates project', async () => { ... });
});
```

## Documentation

For complete documentation including configuration options, concise mode, tag inheritance, troubleshooting, and integration with the phased testing system, see:

**[Test Genie Vitest Requirement Reporter Reference](../../scenarios/test-genie/docs/reference/vitest-requirement-reporter.md)**

## Refreshing Scenario Consumers

```bash
vrooli package refresh vitest-requirement-reporter all --no-restart
```

## Related

- [Requirements Sync Guide](../../scenarios/test-genie/docs/guides/requirements-sync.md)
- [Scenario Unit Testing](../../scenarios/test-genie/docs/guides/scenario-unit-testing.md)
