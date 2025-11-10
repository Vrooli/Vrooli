You are executing a **scenario generation** task for the Ecosystem Manager.

## Boundaries & Focus
- Only touch files under `{{PROJECT_PATH}}/scenarios/{{TARGET}}/` (including `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/`, `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/`, `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/`). Leave everything else untouched, even if you notice issues elsewhere.
- Your job ends once the scenario is fully initialized: research completed, PRD drafted, lifecycle + requirements scaffolding ready. **Do not implement features or deliver a working P0.**

## Generator Deliverables
1. **Research packet**: confirm the scenario is unique, note overlapping capabilities, and capture any reference resources/scenarios.
2. **Scenario skeleton**: copy the appropriate template, set folder structure, choose tech approach (API, UI, CLI, storage) without writing business logic.
3. **`.vrooli/` configuration**: service metadata, ports, resources, lifecycle hooks at `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/` so the platform can run health/status checks immediately.
4. **PRD.md (Operational Targets Document)**: authoritative description of what success looks like (see structure below).
5. **Requirements registry**: seed `requirements/index.json` (and child modules if needed) so each operational target maps to technical requirement placeholders.
6. **Progress log**: create `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/PROGRESS.md` with a starter table to track future work (generators only add the initial entry describing the scaffold).

## Research & Template Selection
- Run `rg -l '{{TARGET}}' {{PROJECT_PATH}}/scenarios/` (or use scenario catalog) to ensure the capability doesn’t already exist.
- Note which resources/scenarios this effort will depend on or augment.
- Use the web to find more information about the problem scope and implementation approach, if relevant
- Copy the official template from `{{PROJECT_PATH}}/scripts/scenarios/templates/react-vite/` into `{{PROJECT_PATH}}/scenarios/{{TARGET}}/` and customize from there—do not create new template variants.

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

## Progress Tracking
- Create `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/PROGRESS.md` with a template such as:
```
| Date | Author | Status Snapshot | Notes |
|------|--------|-----------------|-------|
| {{today}} | {{your_name}} | Initialization complete | Scenario scaffold + PRD seeded |
```
Future agents will append rows; do not track progress inside PRD.md anymore.

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
