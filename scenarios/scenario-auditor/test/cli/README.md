# CLI Tests

## Overview
CLI integration tests are part of the comprehensive test suite in `test/phases/test-integration.sh`.

## Test Coverage

The CLI is tested for:
- ✅ Port auto-detection (4-tier precedence)
- ✅ Health check command (`scenario-auditor health`)
- ✅ Rules listing (`scenario-auditor rules`)
- ✅ Scan functionality (`scenario-auditor scan <scenario>`)
- ✅ Audit command (`scenario-auditor audit <scenario>`)
- ✅ Environment variable handling
- ✅ Error messaging and timeout handling

## Running CLI Tests

```bash
# Part of integration test suite
cd test && ./phases/test-integration.sh

# Manual CLI testing
scenario-auditor health
scenario-auditor rules
scenario-auditor scan scenario-auditor
```

## CLI Architecture

The CLI is a lightweight Bash wrapper around the REST API:
- **Port Detection**: `vrooli scenario port <name> API_PORT` for auto-discovery
- **API Communication**: curl-based with proper error handling
- **Async Operations**: Job-based scanning with status polling
- **Timeouts**: Configurable via `--timeout` flag (default 120s for scans)

See `cli/scenario-auditor` for implementation details.

## Adding CLI Tests

CLI behavior is validated through:
1. **Integration tests** (`test/phases/test-integration.sh`) - Core CLI commands
2. **Business tests** (`test/phases/test-business.sh`) - P0 requirement validation
3. **Manual testing** - Real-world usage scenarios

To add new CLI tests, update the relevant test phase script.
