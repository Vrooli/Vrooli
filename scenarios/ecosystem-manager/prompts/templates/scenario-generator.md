You are executing a **scenario generation** task for the Ecosystem Manager.

## Boundaries & Focus
- Only touch files under `{{PROJECT_PATH}}/scenarios/{{TARGET}}/` (including `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/`, `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/`, `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/`). Leave everything else untouched, even if you notice issues elsewhere.
- Your job ends once the scenario is fully initialized: research completed, PRD drafted, lifecycle + requirements scaffolding ready. **Do not implement features or deliver a working P0.**

## Generator Deliverables
1. **Research packet** (`docs/RESEARCH.md`): confirm uniqueness, note overlapping scenarios/resources, capture external references.
2. **Scenario skeleton**: scaffold from the official template via the CLI (see below), set folder structure, and pick the tech approach without writing business logic.
3. **`.vrooli/` configuration**: service metadata, resources, testing config, etc. at `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/` so lifecycle commands work immediately.
4. **Operational Targets PRD** (`PRD.md`): see structure below.
5. **Requirements registry** (`requirements/index.json` + optional modules) plus a `requirements/README.md` describing the organization.
6. **Documentation set**: baseline `README.md`, `docs/PROGRESS.md`, `docs/PROBLEMS.md`, `docs/RESEARCH.md`, and `requirements/README.md` per the checklist below.

Only produce these files; do not invent ad-hoc documentation outside `README.md`, `docs/`, or `requirements/`.

## Research & Template Selection
- Run `rg -l '{{TARGET}}' {{PROJECT_PATH}}/scenarios/` (or use scenario catalog) to ensure the capability doesn’t already exist.
- Note which resources/scenarios this effort will depend on or augment.
- Use the web to find more information about the problem scope and implementation approach, if relevant.
- Scaffold via the template CLI:
  1. `vrooli scenario template list` → confirm available templates (currently `react-vite`).
  2. `vrooli scenario template show react-vite` → review required/optional variables, stack details, and hooks.
  3. `vrooli scenario generate react-vite --id {{TARGET}} --display-name "{{TITLE}}" --description "{{SCENARIO_DESCRIPTION}}"` (add `--var KEY=VALUE` for optional placeholders).
  4. Capture the CLI’s post-generation checklist in your notes and follow it (it mirrors the deliverables below).
- The generated folder lives at `{{PROJECT_PATH}}/scenarios/{{TARGET}}/`. All subsequent work happens there.

## PRD Structure (Operational Targets Document)
Capture **what** we want, not detailed implementation instructions. Use this outline:
```
# Overview
  - Purpose (1-2 sentences)
  - Target users / verticals
  - Intended usage (internal tool, SaaS, microservice, etc.)
  - Deployment surfaces (Web, iOS, Desktop, etc.)
  - Value proposition (qualitative impact, not $$ estimates)

# Operational Targets
  ## P0 – Must ship for viability
    - OT-ID: Title — narrative of the desired outcome
      - Success signals / acceptance hints
      - Dependencies (resources/scenarios)
      - Linked technical requirement IDs (from requirements registry)
  ## P1 – Should have soon after launch
  ## P2 – Nice to have / future expansion

# Experience & Interfaces
  - Primary workflows / personas
  - Planned surfaces (UI screens, API endpoints, background jobs)
  - Access patterns (iframe embedding, standalone portal, automation hooks)

# Tech Direction Snapshot
  - Preferred stacks (UI: React/TS/Vite/shadcn/lucide by default, API language, storage choices)
  - Integration strategy (how it talks to other scenarios/resources)
  - Non-goals / out-of-scope notes

# Dependencies & Launch Plan
  - Required resources (Ollama, Postgres, etc.)
  - Scenario dependencies (monitoring, auth, etc.)
  - Open questions / risks
```
Keep it readable—use bullet trees, avoid code snippets or API specs.

### Operational Target Tiers (P0/P1/P2)
- **P0**: Absolutely necessary outcomes. Without them the scenario fails (e.g., “Operators can schedule recycling pickups”).
- **P1**: Important enhancements that unblock scale or multi-user workflows.
- **P2**: Nice-to-have polish or expansion ideas.
Each target must include: narrative, success indicators, dependencies, and the technical requirement ID(s) that will enforce it.

## `.vrooli/` Setup Checklist
- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/service.json`: name, displayName, description, tags, API/UI port ranges, resource dependencies.
- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/endpoints.json` or other metadata files required by the lifecycle system.
- Ensure ports land in the scenario allocation range and never collide with existing services.

## Requirements Registry Seeding
- Create `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/index.json` (and modular children if helpful).
- For every operational target, define at least one requirement with:
  - `id` (match `[A-Z][A-Z0-9]+-[A-Z0-9-]+`), `title`, `criticality`, `prd_ref` pointing back to the PRD section, and empty `validation` arrays.
  - Optional grouping via `children` so the parent covers the full operational target.
- Document the expected test phases (unit/integration/business) even though tests don’t exist yet.
- Include instructions in README/PRD for improvers on how to tag tests with `[REQ:ID]` and run `{{PROJECT_PATH}}/scripts/requirements/report.js --scenario {{TARGET}} --mode sync` once real tests appear.

## Documentation Checklist
- **README.md** (at scenario root):
  - Purpose + operational summary
  - How to run (`vrooli scenario run`, ports) and test commands (even if placeholders)
  - Reference to `PRD.md`, `docs/PROGRESS.md`, `docs/PROBLEMS.md`, and requirements directory
- **docs/PROGRESS.md**: initialize with a table, e.g.
  ```
  | Date | Author | Status Snapshot | Notes |
  |------|--------|-----------------|-------|
  | {{today}} | {{your_name}} | Initialization complete | Scenario scaffold + PRD seeded |
  ```
- **docs/PROBLEMS.md**: create sections for "Open Issues" and "Deferred Ideas"; list bullet items with context and links back to PRD targets.
- **docs/RESEARCH.md**: summarize repo findings, overlapping scenarios/resources, and external links that informed the PRD.
- **requirements/README.md**: describe how the registry is organized (modules, naming pattern, how to tag tests, command to sync).

Future agents will append to PROGRESS/PROBLEMS/RESEARCH—call out in README that these are the canonical outlets for updates.

## Collision Avoidance & Validation
- Stay inside `{{PROJECT_PATH}}/scenarios/{{TARGET}}/`. If you uncover bugs elsewhere, note them in your summary but leave the files untouched.
- After scaffolding, run:
  1. `vrooli scenario status {{TARGET}}` – confirm lifecycle metadata is discoverable and note any missing files.
  2. `scenario-auditor audit {{TARGET}} --timeout 240` – expect gaps, but document them so improvers know where to start.
- Capture these outputs (or summaries) in your final handoff.

## Task Context

**Task ID**: {{TASK_ID}}
**Title**: {{TITLE}}
**Type**: {{TYPE}}
**Operation**: {{OPERATION}}
**Priority**: {{PRIORITY}}
**Category**: {{CATEGORY}}
**Status**: {{STATUS}}
**Current Phase**: {{CURRENT_PHASE}}

### Notes
{{NOTES}}
