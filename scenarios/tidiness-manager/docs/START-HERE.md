# Start Here

Tidiness Manager owns maintainability and cleanup guidance for scenarios. It produces tidiness findings for long files, complexity, duplication, technical-debt markers, coupling, issue queues, and cleanup campaigns.

Quality Health owns lint, type checking, static-quality configuration, and protective comments. Do not add static-quality policy back into Tidiness Manager.

## Initialization Protocol

1. Read `PRD.md`, `README.md`, and `docs/concepts/ARCHITECTURE.md`.
2. Confirm the current boundary with `docs/internal/SEAMS.md`.
3. Use lifecycle commands for scenario management:

```bash
cd scenarios/tidiness-manager
make start
make status
make stop
```

4. Validate focused behavior with Test Genie:

```bash
test-genie execute tidiness-manager --phases tidiness --json
test-genie execute tidiness-manager --phases quality --json
```

## Architecture Rules

- Route HTTP behavior through the Go API handlers and coordinators.
- Route agent workflows through the CLI commands documented in `docs/reference/cli-commands.md`.
- Keep scanner path handling root-bound and timeout-aware.
- Keep static-quality contracts in Quality Health. Tidiness Manager may run tests and linters for its own health, but it does not own policy for other scenarios.

## Replacing The Example Domain

This scenario is no longer a generated example. Do not reintroduce the generated notes domain or placeholder product language. New work should extend the existing domains in `docs/concepts/DOMAINS.md` or record a new domain decision in `docs/internal/DECISIONS.md`.
