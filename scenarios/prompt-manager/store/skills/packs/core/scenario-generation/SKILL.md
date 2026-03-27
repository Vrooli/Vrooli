# Scenario Generation

## Purpose

Reference knowledge for creating and improving Vrooli scenarios. Use this skill when authoring implementation plans for idea-type backlog items so the plan includes the correct operational steps, CLI commands, and tool integrations.

This is **not** an orchestration script — it describes what tools exist and how to use them, so that plans can include the right steps for the specific situation.

## Required Reading

- `prompt-manager skill read ecosystem-manager-tools` — CLI reference for task creation, steering, and queue management

## Scenario Scaffolding

### Listing Templates

```bash
vrooli scenario template list
```

### Selecting a Template

| Scenario Characteristic | Recommended Template |
|-------------------------|---------------------|
| Has a web UI | `react-vite` |
| API-only / backend service | Go or API-focused template |
| CLI tool | CLI-focused template |

When unsure, `react-vite` is a safe default for anything with a user-facing component.

### Generating the Scaffold

```bash
vrooli scenario generate <template> --id <name> --display-name "<title>" --description "<one-line purpose>"
```

Follow the template's post-generation checklist (dependency installs, `go mod tidy`, etc.).

## PRD Workflow (Preserve-First)

Always check for an existing PRD baseline before generating one from scratch. This preserves user-provided or previously refined specifications.

### Decision Flow

```
Does an existing PRD exist (archive/PRD.md, or scenarios/<name>/PRD.md)?
  → Yes → Copy to scenarios/<name>/PRD.md as baseline, then validate/fix
  → No  → Generate from context (fallback path)
```

### Validate and Fix Existing PRD

```bash
prd-control-tower prd validate <name> --json
prd-control-tower prd fix <name> --auto --json
prd-control-tower prd validate <name> --json
```

### Generate PRD (Fallback Only)

Only when no viable baseline exists or fix attempts leave it invalid:

```bash
prd-control-tower prd generate <name> --context-file /tmp/prd_context_<name>.md --publish --json
```

The context file should synthesize information from the backlog item's plan, workshop rounds, research findings, and archive materials.

## Requirements Workflow (Preserve-First)

Same preserve-first pattern as PRD.

### Decision Flow

```
Does an existing requirements baseline exist (archive/requirements/index.json)?
  → Yes → Copy to scenarios/<name>/requirements/, then validate/fix
  → No  → Generate from PRD (fallback path)
```

### Validate and Fix Existing Requirements

```bash
prd-control-tower requirements validate <name> --json
prd-control-tower requirements fix <name> --json
prd-control-tower requirements validate <name> --json
```

### Generate Requirements (Fallback Only)

```bash
prd-control-tower requirements generate <name> --context-file /tmp/requirements_context_<name>.md --json
prd-control-tower requirements validate <name> --json
```

## Archive and Staging Material Incorporation

Backlog items may contain refined materials that should be incorporated into the scenario.

### Material Sources (Priority Order)

1. **`enhance/` staging materials** (preferred) — pre-synthesized by the workshop enhance step:
   - `enhance/prd-context.md` — ready-to-use PRD context
   - `enhance/requirements-context.md` — ready-to-use requirements context
   - `enhance/doc-outlines.md` — documentation structure outlines
2. **`archive/` raw materials** (fallback) — user-provided files when staging doesn't exist:
   - `archive/PRD.md`, `archive/requirements/` — structured baselines
   - Other archive files — reference configs, design docs, prior work

### Incorporation Rules

- Always prefer `enhance/` staging when available — it incorporates answered questions, accepted suggestions, and resolved conflicts
- When updating an existing scenario's PRD/requirements, **merge with backup** — do not blindly overwrite
- Documentation materials → `scenarios/<name>/docs/`
- Reference configs → `scenarios/<name>/.vrooli/` or noted in README

## Ecosystem-Manager Integration

After initializing or updating a scenario, use ecosystem-manager to create an improvement task that drives iterative agent development.

### Creating Improvement Tasks

```bash
# With a steering profile (works for both new and existing scenarios)
ecosystem-manager task improve --steer-profile <profile-id> scenario <name>

# With a single steer mode (improver tasks only)
ecosystem-manager task improve --steer-mode "<mode>" scenario <name>

# With an ordered list of steer modes (improver tasks only)
ecosystem-manager task improve --steer-queue "progress,test,refactor" scenario <name>
```

### Creating Generator Tasks (New Scenarios)

```bash
ecosystem-manager task add --steer-profile <profile-id> scenario <name>
```

### Validate Before Creating

```bash
ecosystem-manager task improve --dry-run --steer-profile balanced scenario <name>
```

### Ensure Queue is Running

```bash
ecosystem-manager queue status
ecosystem-manager queue start  # if paused
```

## Steering Strategy Selection

| Scenario Characteristic | Recommended Profile | Reasoning |
|-------------------------|-------------------|-----------|
| Newly initialized, needs broad first pass | `balanced` or `rapid-mvp` | Covers all aspects of implementation |
| Quality/stability focus | `production-ready` | Extra test and refactor passes |
| Heavy UI/UX component | `ux-excellence` | Dedicated UX iteration modes |
| Test debt or refactoring | `refactor-test-focus` | Prioritizes code quality |
| Single improvement focus | `--steer-mode` | Single-pass (e.g., "test") |
| Unique improvement needs | `--steer-queue` | Tailor mode order to requirements |

**Rules:**
- Prefer built-in templates when one fits
- `--steer-mode` and `--steer-queue` only work on **scenario improver** tasks
- One task per scenario at a time — check with `ecosystem-manager task list --type scenario`

## Validation Tools

After initialization, capture baseline metrics for handoff:

```bash
vrooli scenario status <name>
scenario-completeness-scoring score <name>
scenario-auditor audit <name> --timeout 240
```

Expect low completeness scores and auditor failures for freshly initialized scenarios — this is normal. Capture the results for notes and handoff to the improvement agent.

## Scenario Documentation Initialization

Ensure these files exist with baseline content (the template may have created some already):

- `README.md` — purpose, how to run, link to PRD
- `docs/PROGRESS.md` — initialization entry with date
- `docs/PROBLEMS.md` — open issues and deferred ideas from research
- `docs/RESEARCH.md` — uniqueness check, related scenarios, references

Use `enhance/doc-outlines.md` as a guide when available.

## Anti-Patterns

- **Don't** hand-write PRD.md — always generate or validate via `prd-control-tower`
- **Don't** discard archive/staging materials — they represent refined context
- **Don't** use raw archive when enhance/ staging exists — staging is pre-synthesized
- **Don't** skip validation after PRD/requirements copy or generation
- **Don't** create ecosystem-manager tasks without verifying the queue processor is running
- **Don't** use custom steer queues when a built-in profile fits
