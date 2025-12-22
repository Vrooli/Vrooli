# Browser Automation Studio CLI

Cross-platform CLI for managing workflows, executions, playbooks, and recordings in the Browser Automation Studio scenario.

## Architecture

```
cli/
- main.go                # Entry point
- app.go                 # App wiring & command registration
- status/                # DOMAIN: Health & status checks
- playbooks/             # DOMAIN: Playbook registry management
- workflows/             # DOMAIN: Workflow CRUD & execution
- executions/            # DOMAIN: Execution monitoring & exports
- recordings/            # DOMAIN: Recording import
- internal/              # Shared helpers (API, output, utils)
```

## Install

```bash
cd scenarios/browser-automation-studio/cli
./install.sh
```

## Commands

```bash
browser-automation-studio status
browser-automation-studio playbooks order
browser-automation-studio workflow list
browser-automation-studio workflow execute <workflow-id>
browser-automation-studio execution watch <execution-id>
browser-automation-studio recording import <archive.zip>
```

Adhoc execution with a starting URL:

```bash
browser-automation-studio workflow execute \
  --from-file bas/actions/open-project.json \
  --start-url http://localhost:8080/ \
  --wait
```

Seeded workflows (actions that rely on test-genie seed state) can declare
`metadata.labels.seed_required = "true"`. Use `--seed applied` to satisfy the
seed gate when running workflows directly:

```bash
browser-automation-studio workflow execute \
  --from-file bas/actions/open-demo-project.json \
  --params '{"initial_params": {...}}' \
  --seed applied \
  --wait
```

To auto-apply seeds (through test-genie) and clean up afterward, use:

```bash
browser-automation-studio workflow execute \
  --from-file bas/actions/open-demo-project.json \
  --seed needs-applying \
  --wait
```

For async runs, `--seed needs-applying` schedules cleanup automatically after the execution reaches a terminal state.
