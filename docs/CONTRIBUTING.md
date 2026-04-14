# Contributing to Vrooli

Vrooli is a local, self-improving software foundry. Contributions should strengthen the current platform, expand reusable capability, or improve the conditions under which future scenarios and resources can be built safely.

Start with:

- [README.md](README.md)
- [QUICKSTART.md](QUICKSTART.md)
- [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md)
- [reference/cli-commands.md](reference/cli-commands.md)
- [repo-contract.md](repo-contract.md)
- [package-governance.md](package-governance.md)

## Core Expectations

- prefer edits to existing files over creating parallel structures
- use the current `vrooli` CLI and scenario Makefiles, not direct execution or shell-era shortcuts
- keep docs, manifests, validation rules, and code aligned in the same change when you touch shared platform contracts
- treat scenarios as long-lived business and platform capabilities, not disposable demos
- treat resources as shared local building blocks, not one-off integrations

## Getting Started

Clone the repository, then use the canonical setup flow:

```bash
make setup
vrooli help
vrooli develop
```

If you are working on a specific scenario, prefer its lifecycle Makefile:

```bash
cd scenarios/<name>
make start
make test
make logs
make stop
```

Do not start scenarios with direct binary or script execution.

## Contribution Areas

Common high-value contribution categories:

- project-level CLI and control-plane improvements under `cmd/` and `internal/`
- scenario improvements that expand reusable business capability
- resource improvements that make the local platform more capable or more reliable
- package governance, repo-contract, and validation work that reduces platform drift
- documentation updates that improve current-state accuracy

## Development Workflow

Use a normal branch-based workflow:

```bash
git checkout -b feature/your-change
```

Before opening a PR:

```bash
vrooli help
```

Run the validations that match your change:

```bash
vrooli scenario test <name>
go test ./cmd/vrooli/... ./internal/...
make validate-repo-contract
make validate-package-governance
```

Do not treat every command above as mandatory for every change. Run the checks that cover the surface you touched, then call out anything you could not run.

## Working On Scenarios

- use `vrooli scenario ...` for inspection and orchestration when needed
- prefer scenario-local `make start|test|logs|stop` for day-to-day lifecycle work
- do not add new direct-execution guidance to docs or scripts
- keep scenario docs inside the scenario when behavior is scenario-specific

## Working On Resources

- use `vrooli resource ...` as the operator surface
- keep resource behavior manifest-driven
- do not default to cloning old shell-first resource layouts
- if you change resource lifecycle semantics, update the docs and validation surface in the same PR

## Shared Contracts

When touching shared platform structure:

- repo-aware layout rules belong in `.vrooli/repo-contract.json`
- package sharing rules belong in package manifests under `packages/<name>/.vrooli/package.json`
- do not introduce new repo-root heuristics or canonical path assembly when a shared helper already exists
- validate repo-contract and package-governance changes in the same change set

## Documentation

Project-level docs should distinguish clearly between:

- canonical current-state guidance
- specialized reference material
- historical or proposal documents

Do not let older historical docs override the canonical layer under `docs/README.md`, `docs/QUICKSTART.md`, `docs/concepts/`, and `docs/reference/`.

## Pull Requests

A good PR does four things:

- explains the problem clearly
- explains the chosen approach briefly
- names the validations run
- calls out follow-up work or remaining risk honestly

Use conventional, readable titles such as:

- `docs: refresh contributor workflow`
- `fix(cli): normalize scenario path resolution`
- `feat(resource): add governed blueprint restore flow`

## Review Standard

Expect review to focus on:

- correctness
- drift against current architecture and contracts
- operational safety
- test and validation coverage
- documentation accuracy

If a change updates platform truth, update the docs in the same PR.
