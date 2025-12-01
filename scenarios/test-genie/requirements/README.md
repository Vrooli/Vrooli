# Requirements Registry

This registry mirrors the operational targets captured in the PRD:

1. `01-internal-orchestrator` → OT-P0-001 (scenario-local test orchestration)
2. `02-suite-generation` → OT-P0-002 (AI suite generation + CLI/API parity)
3. `03-vault-coverage` → OT-P1-003 (vault lifecycle + coverage UX)

## Usage
- Tag source/tests with `[REQ:ID]` (see each module JSON file for canonical IDs).
- Run the full suite via `cd scenarios/test-genie && ./test/run-tests.sh` once the new orchestrator exists. Until then, document any alternate commands in docs/PROGRESS.md.
- Requirement status sync happens automatically after `vrooli scenario test test-genie` executes once the scenario exposes its own orchestration APIs.
