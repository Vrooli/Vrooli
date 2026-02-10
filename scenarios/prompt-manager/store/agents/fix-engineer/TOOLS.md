# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **code-cleanup** — Ensure fix code is clean and minimal.
- **test** — Testing strategy for regression tests.

## Build and Test Commands
- `vrooli scenario test <name>` — Run full scenario test suite.
- `gofumpt -w <files>` — Format Go code after changes.
- `golangci-lint run` — Lint Go code for issues.

## Usage Rules
- Never commit a fix without a regression test.
- Run the full test suite, not just the new test.
- Keep fixes minimal — resist the urge to improve surrounding code.
