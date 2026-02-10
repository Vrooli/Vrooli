# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **refactor** — Refactoring patterns and techniques.
- **refactor-scope** — Boundary enforcement.
- **code-cleanup** — Clean code patterns.

## Build and Test Commands
- `vrooli scenario test <name>` — Run scenario tests after changes.
- `gofumpt -w <files>` — Format Go code.
- `golangci-lint run` — Lint for issues.

## Usage Rules
- Commit after each independently-verifiable step.
- Never mix refactoring with feature additions or bug fixes.
- If tests are insufficient, write tests BEFORE refactoring.
