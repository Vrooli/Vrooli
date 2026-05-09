# Testing

Project-level testing guidance is intentionally thin. The canonical rule is:

- use `vrooli scenario test <name>` for scenario suites
- use the relevant Go or package-level test commands for the platform code you changed
- prefer current CLI surfaces and maintained fixtures over shell-era ad hoc test flows

## Start Here

- [reference/cli-commands.md](reference/cli-commands.md)
- [scenarios/VALIDATION.md](scenarios/VALIDATION.md)
- [../scenarios/test-genie/docs/QUICKSTART.md](../scenarios/test-genie/docs/QUICKSTART.md)

## Common Commands

```bash
vrooli scenario test <name>
go test ./cmd/vrooli/... ./internal/...
make hygiene
make validate-package-governance
```

Use the smallest validation surface that honestly covers your change.
