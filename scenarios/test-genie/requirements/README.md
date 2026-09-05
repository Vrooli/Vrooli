# Requirements Registry

This registry mirrors the operational targets captured in the PRD:

1. `01-internal-orchestrator` → OT-P0-001 (scenario-local test orchestration)
2. `02-suite-generation` → OT-P0-002 (execution-driven remediation)
3. `03-vault-coverage` → OT-P1-003 (remediation evidence UX)

## Usage
- Tag source/tests with `[REQ:ID]` (see each module JSON file for canonical IDs).
- Run the full suite via `test-genie execute test-genie --preset comprehensive` or `cd scenarios/test-genie && make test`.
- Requirement status auto-sync happens automatically after comprehensive test execution.
- Validate the registry with `vrooli scenario requirements validate test-genie --json` before changing requirement status or validation references.
