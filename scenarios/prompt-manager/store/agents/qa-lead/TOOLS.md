# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **screaming-architecture-audit** — Architecture alignment assessment.
- **progress** — Priority ordering for audit findings.
- **documentation-health** — Documentation quality baseline.

## Vrooli Commands
- `vrooli scenario test <name>` — Run scenario tests to assess pass rate.
- `cd scenarios/<name> && make test` — Run tests via Makefile.

## Usage Rules
- Always provide evidence (file paths, line numbers) for findings.
- Rate severity: Critical, High, Medium, Low.
- Distinguish between bugs and quality improvement opportunities.
