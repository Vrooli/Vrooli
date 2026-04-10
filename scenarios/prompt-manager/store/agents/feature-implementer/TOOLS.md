# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **feature-scope** — Scope boundary enforcement.
- **api-steer** — API patterns.
- **react-stability** — React patterns.
- **test** — Testing strategy.

## Build and Test Commands
- `vrooli scenario test <name>` — Run scenario tests.
- `cd scenarios/<name> && make test` — Run tests via Makefile.
- `gofumpt -w <files>` — Format Go code.

## Usage Rules
- Build in small, testable increments.
- Write tests alongside implementation, not after.
- Follow the design — discuss before deviating.
