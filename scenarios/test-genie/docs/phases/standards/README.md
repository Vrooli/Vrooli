# Standards Phase

The `standards` phase runs **scenario-auditor** standards rules to enforce repository conventions (PRD structure, `.vrooli/service.json` lifecycle setup, API proxy patterns, and other scenario hygiene checks).

## How It Runs

Test Genie invokes:

```bash
scenario-auditor audit <scenario> --standards-only --timeout <seconds> --json
```

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip standards
```

Disable per scenario via `.vrooli/testing.json`:

```json
{
  "phases": {
    "standards": { "enabled": false }
  }
}
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json`:

```json
{
  "phases": {
    "standards": { "timeout": "120s" }
  }
}
```

Environment variables:

- `TEST_GENIE_STANDARDS_FAIL_ON` (default: `high`)
- `TEST_GENIE_STANDARDS_LIMIT` (default: `20`)
- `TEST_GENIE_STANDARDS_MIN_SEVERITY` (default: `medium`) — affects which violations are printed in observations

