# Actions

Actions are the executable layer between prompt-manager's judgment-oriented skills and Vrooli-controlled CLI implementations.

Status: implemented and in adoption. Action storage, API CRUD, validation, governed API execution, CLI list/show/create/update/delete/validate/run, AI indexing, opt-in discovery, graph integration, seed Actions, and UI browse/detail/validate/edit/run surfaces are implemented.

## Why Actions Exist

Prompt-manager already models agents, teams, and skills. Skills are intentionally prose-based: they teach agents how to reason, decide, and apply judgment. That makes them flexible, but it also makes repeatable operational work more expensive and less reproducible when a skill has collapsed to "run this command with these inputs."

Actions provide a stricter form for that end state:

```text
Truth lives in the Plan of Record.
Judgment lives in Skills.
Execution lives in Actions.
Implementation lives in CLIs.
Unbuilt work lives in the Backlog.
Raw learning starts in typed knowledge topics.
```

An Action is a typed, discoverable wrapper over exactly one Vrooli-controlled CLI command. It makes an operation easier for agents to find, validate, and run without embedding operational prose in a skill.

## Relationship to Other Entities

```text
Team
  -> coordinates Agents

Agent
  -> reads Skills for judgment
  -> discovers and runs Actions for execution

Skill
  -> explains how to decide, evaluate, or synthesize
  -> may reference Actions for deterministic substeps

Action
  -> declares inputs, outputs, permissions, examples, and validation
  -> invokes one Vrooli-controlled CLI command

CLI
  -> owns implementation, branching, retries, and operational behavior
```

Actions do not replace skills. They remove deterministic execution from skills when the execution can be expressed as a stable command contract.

## What Counts as an Action

Good Action candidates have all of these properties:

- Stable input and output schema
- Deterministic success and failure states
- Exactly one Vrooli-controlled command target
- No shell pipelines, ad hoc scripts, or embedded branching
- Permission requirements that can be declared before execution
- Validation command, fixture, or health check
- Reusable across agents, teams, or skills

Examples:

| Action | Purpose | Likely command owner |
|--------|---------|----------------------|
| `scenario.ui.screenshot` | Capture a scenario UI screenshot | scenario or project CLI |
| `scenario.test.run` | Run a scenario test suite | project lifecycle CLI |
| `team.decisions.list` | List pending decisions for a team | prompt-manager CLI |
| `scenario.logs.tail` | Read recent scenario logs | project lifecycle CLI |
| `skill.health.audit` | Run a skill health check | prompt-manager or meta-optimization CLI |

## What Does Not Count

These should not be Actions:

- Judgment-heavy workflows, such as deciding which architecture is better
- Multi-step shell recipes with branching in the wrapper
- Raw external tools such as `git`, `docker`, `psql`, `curl`, or `grep`
- One-off local scripts that are not owned by a Vrooli scenario, resource, or project CLI
- Prose summaries, reports, or generated copy that still require LLM synthesis

If an operation needs raw external tooling, create a Vrooli-controlled CLI wrapper first. The Action should wrap that controlled command, not the external tool directly.

## Allowed Command Targets

Actions may wrap commands owned by Vrooli:

- `vrooli ...` project lifecycle commands
- `prompt-manager ...` commands
- Scenario CLIs managed through the Vrooli lifecycle
- Resource CLIs when they are Vrooli-owned or exposed through a Vrooli wrapper

Actions should not wrap commands outside Vrooli's control. This keeps execution observable, portable, testable, and governed by lifecycle conventions.

## Contract Shape

The `action.json` contract includes at least:

```json
{
  "kind": "action",
  "schemaVersion": 1,
  "id": "scenario.ui.screenshot",
  "name": "Take Scenario Screenshot",
  "description": "Capture a screenshot of a running scenario UI.",
  "status": "active",
  "owner": {
    "type": "scenario",
    "id": "prompt-manager"
  },
  "command": {
    "argv": ["vrooli", "scenario", "screenshot", "{{scenario}}", "--viewport", "{{viewport}}"]
  },
  "inputs": {
    "scenario": {
      "type": "string",
      "required": true
    },
    "viewport": {
      "type": "string",
      "enum": ["desktop", "mobile"],
      "default": "desktop"
    }
  },
  "outputs": {
    "imagePath": {
      "type": "file",
      "description": "Path to the generated screenshot."
    }
  },
  "permissions": {
    "filesystemWrite": true,
    "localhostNetwork": true
  },
  "examples": [
    {
      "description": "Capture a desktop screenshot of prompt-manager.",
      "input": {
        "scenario": "prompt-manager",
        "viewport": "desktop"
      }
    }
  ],
  "validation": {
    "argv": ["prompt-manager", "action", "validate", "scenario.ui.screenshot"]
  }
}
```

The command is intentionally argv-shaped instead of shell-shaped. The Action runtime does not interpret pipes, command separators, conditionals, or environment-specific shell syntax.

## Execution Governance

The API runtime can run active Actions only after validation marks the command runnable. It applies typed input defaults, renders placeholders into argv tokens, executes without a shell, enforces timeout and process-wide concurrency limits, caps stdout/stderr, and writes bounded `runs.jsonl` audit entries. `execution.runEligible: false` keeps an Action discoverable and editable while blocking API runs.

### Validated vs unvalidated Actions

Every validation result carries a `certainty` flag derived from the command's owning scenario:

| Owner scenario state | Certainty | Effect on the Action |
|---|---|---|
| Owning scenario has declared `cli/manifest.json` (and the command appears in it) | `command` or `operation` (driven by the manifest's `governance` block) | Valid + runnable. Governance fields — `effect`, `run_eligible`, `permissions`, `requires_confirmation` — flow into the action validation response. |
| Owning scenario has no `cli/manifest.json` yet, or the command isn't in it | `owner-only` | Valid + runnable + flagged `unvalidated: true`. The action runs, but the validation response carries an info-level note that the owning scenario hasn't declared CLI governance. CLI output appends ", unvalidated"; the UI status pill shows "· unvalidated" with an amber banner. |
| Owning scenario is unknown / command path does not resolve to any scenario | `none` | Validation fails; the action cannot run. |

The `unvalidated` state is a safety net: scenarios are migrating to `cli/manifest.json` incrementally, and Actions that target a not-yet-migrated scenario continue to run. Once the owning scenario adopts a manifest (per `path:templates/scenarios/react-vite/docs/reference/cli-commands.md`), the `unvalidated` flag drops off automatically without any change to the Action itself.

Governance vocabulary (from `cli-manifest/v1`):

- `effect`: `read | write | destructive` — side-effect classification.
- `run_eligible`: whether `prompt-manager action run` may invoke the command. `false` rejects runs without rejecting the Action contract.
- `permissions`: starter vocabulary `filesystem:read | filesystem:write | network:internal | network:external | admin`.
- `requires_confirmation`: defaults to `true` when `effect == destructive`.

These fields belong to the **command** (the CLI surface of the owning scenario), not the Action; Actions reference a command and inherit its governance.

The CLI exposes `prompt-manager action run` as a thin API client for trusted workflows that intentionally select an active runnable Action. The UI Action editor delegates to the same governed API route for dry-run and run requests, blocks runs while local contract edits are unsaved, and renders the run response envelope.

The first shipped seed Actions are action:scenario.status.show, which wraps `vrooli scenario status {{scenario}}`, and action:team.decisions.list, which wraps `prompt-manager team decision-list {{team}} --json`. Both are read-oriented Actions with API/CLI validation and dry-run coverage.

## Graduation from Skills

A skill section can graduate to an Action when:

1. The section maps to one Vrooli-controlled CLI command.
2. The command has stable inputs and outputs.
3. The command handles branching internally.
4. The Action can declare permissions up front.
5. The Action can be validated programmatically.
6. The skill no longer needs agent judgment for that operation.

If a skill still contains judgment, split it:

- Keep the decision rule in the skill.
- Move deterministic execution into one or more Actions.
- Move missing implementation work into a backlog item or `capability-gap`.

## Discovery

The intended discovery path is capability-first rather than entity-first:

```bash
prompt-manager discover "take screenshot of scenario UI" --type all
```

The result set can include both skills and Actions when `--type all` is used. Omitting `--type` remains skill-only for compatibility. Agents should prefer an exact Action match for deterministic execution and use skills when they need judgment, methodology, or synthesis.

## Naming Note

Prompt-manager already uses "capabilities" for agent and skill requirement matching. Actions should not be called capabilities in the data model because that would conflate executable operations with permission/ability declarations. "Action" names the executable wrapper without colliding with the existing capability-matching system.
