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

## CLI
```bash
scenario-stack-governor status
scenario-stack-governor rules list
scenario-stack-governor rules get REACT_VITE_UI_INSTALLS_DEPENDENCIES
scenario-stack-governor rules disable REACT_VITE_UI_INSTALLS_DEPENDENCIES
scenario-stack-governor scenarios list
scenario-stack-governor run --scenario scenario-auditor
scenario-stack-governor fix --scenario scenario-auditor --dry-run
```

The Go CLI follows the standard `cli-core` contract:
- `status` and `configure` come from the shared scenario scaffold.
- `rules` manages rule inventory and enablement.
- `scenarios` discovers target scenarios.
- `run` executes enabled governance rules.
- `fix` previews or applies automated fixes for selected scenarios.

## Key endpoints
- `GET /health`
- `GET /api/v1/rules`
- `GET /api/v1/config`
- `PUT /api/v1/config`
- `POST /api/v1/run`

## Rules
- `REACT_VITE_UI_INSTALLS_DEPENDENCIES`: verifies React/Vite scenario UIs install dependencies in the scenario-local UI directory.

## Package governance
- `PACKAGE_GOVERNANCE_SCENARIO_ADOPTION`: delegates to Structure Health's project target and turns scenario-scoped package-boundary findings into stack-governor findings.
- This keeps package policy centralized in the Structure Health catalog while still exposing enforcement through stack-governor's rule surface.

## Docs
- `scenarios/scenario-stack-governor/PRD.md`
- `scenarios/scenario-stack-governor/docs/PROGRESS.md`
- `scenarios/scenario-stack-governor/docs/PROBLEMS.md`
- `scenarios/scenario-stack-governor/docs/RESEARCH.md`
