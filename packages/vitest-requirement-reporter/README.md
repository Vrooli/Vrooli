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
# Native assertion observations

Reports include `native_observations`, one entry per run. The observed profile is
Vitest **2.1.9** only. Other versions remain `unknown`, even when tests pass.
`nativeAssertionOptions(installedVersion)` enables `expect.requireAssertions`
only for that exact supported version; callers must supply the installed runner
version. The reporter reads the actual runner version and per-project resolved
configuration independently, using public reporter APIs.

Native IDs, project/file identity, final state, retry count, retained errors, and
unhandled errors remain separate from assertion status. Skipped tests and
expected-failure inversion never become assertion-positive evidence. Failed
setup or matcher failures remain unknown unless the native assertion-absence
diagnostic is present. Alternate assertion libraries are not observed by this
profile and cannot establish a clean native assertion result.

`checked_clean` means only that the enabled native check passed: bare `expect`,
tautologies, and setup-hook assertions can satisfy it. It does not establish
matcher completion, edge-case coverage, or behavioral adequacy. Vitest 2.1.9
exposes retry final state and retained errors, not a complete attempt history.
Append mode preserves separate native run records instead of collapsing them.

Unit Health supplies `VROOLI_TEST_RUN_ID` and `VROOLI_TEST_QUALITY_OUTPUT` for
its command-scoped handoff. The reporter writes a separate native record at
that path; its embedded run identity must match the collector's expected
identity. Normal requirement output and append history remain independent.

### Multi-project command observations

The public `mergeNativeObservationReports` export joins one command's native
observations without changing runner support or assertion status. It requires
consistent nonempty run IDs, schema, runner version and support profile. It
rejects duplicate `(project, file, test_id)` identities and preserves failures,
unhandled errors and limitations. It does not establish that every expected
project ran. The command runner must compare its expected cohort, retain raw
shards and add missing-project errors. Use `nativeObservations` for a scoped
empty/unknown report when no shard completed; do not reuse a prior run's file.

Vitest versions outside the retained native conformance profile remain
unsupported after merging. A successful test or merge does not establish
behavioral adequacy or upgrade the assertion-support profile.

Failure artifact directories use an at-most80-character readable prefix plus a
16-hex SHA256 suffix of the JSON-encoded pair `[fullTestPath, nativeTaskId]`.
The full path includes file/suite/test identity; the native Vitest task ID also
distinguishes repeated rows or requirement-tagged cases whose readable paths
match. Legacy records without a task ID use null in that pair; identical full
paths without distinct native IDs cannot be distinguished by this fallback.
Both publication and printed README links resolve `artifactsDir` from the actual
working directory, including absolute paths; no scenario-parent rewrite applies.
The maintained10-case regression exercises actual `onFinished` publication:
long names, punctuation, separate files, repeated rows and requirement-tagged
cases retain10 distinct payload directories per run. Absolute/relative paths
from a UI working directory produce stable leaves and20 resolving printed links
across the two runs. This qualifies diagnostic identity and paths, not test
adequacy or assertion-profile support.
