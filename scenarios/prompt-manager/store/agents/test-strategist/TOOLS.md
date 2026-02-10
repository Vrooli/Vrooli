# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **e2e-testing** — End-to-end testing methodology.
- **unit-testing-architecture-steer** — Unit test design patterns.
- **test** — General testing strategy.

## Testing Commands
- `vrooli scenario test <name>` — Run scenario test suite.
- `cd scenarios/<name> && make test` — Run tests via Makefile.
- `go test ./... -cover` — Go test coverage.

## Usage Rules
- Distinguish behavior coverage from line coverage.
- Prioritize gaps by risk, not by ease of testing.
- Recommend specific test outlines, not vague suggestions.
