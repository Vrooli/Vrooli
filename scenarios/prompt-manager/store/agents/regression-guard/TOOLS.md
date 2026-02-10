# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **test** — Testing strategy.
- **e2e-testing** — End-to-end validation.
- **unit-testing-architecture-steer** — Unit test design.

## Testing Commands
- `vrooli scenario test <name>` — Run full scenario test suite.
- `cd scenarios/<name> && make test` — Run tests via Makefile.

## Usage Rules
- Always establish a baseline before refactoring begins.
- Run tests after EVERY refactoring step, not just at the end.
- Flag subtle behavioral changes, not just test failures.
