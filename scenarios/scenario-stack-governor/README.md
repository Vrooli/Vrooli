# Scenario Stack Governor

Tech-stack governance for Vrooli scenarios: a rule catalog with toggles, explanations, and on-demand execution.

## What it does
- Stores enabled/disabled rules in `scenarios/scenario-stack-governor/config/rules.json`.
- Provides an API to list rules, update config, and run enabled rules.
- Provides a UI that explains each rule and shows run results.

## Run
```bash
cd scenarios/scenario-stack-governor
make start
```

## Key endpoints
- `GET /health`
- `GET /api/v1/rules`
- `GET /api/v1/config`
- `PUT /api/v1/config`
- `POST /api/v1/run`

## First rule (MVP)
- `GO_CLI_WORKSPACE_INDEPENDENCE`: verifies Go-based scenario CLIs build with `GOWORK=off`, and flags missing `go.mod` wiring for common non-transitive `replace` pitfalls.

## Docs
- `scenarios/scenario-stack-governor/PRD.md`
- `scenarios/scenario-stack-governor/docs/PROGRESS.md`
- `scenarios/scenario-stack-governor/docs/PROBLEMS.md`
- `scenarios/scenario-stack-governor/docs/RESEARCH.md`
