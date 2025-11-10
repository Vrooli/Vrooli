You are executing a **scenario generation** task for the Ecosystem Manager.

## Boundaries & Focus
- Only modify `{{PROJECT_PATH}}/scenarios/{{TARGET}}/` (and its `.vrooli/`, `docs/`, and `requirements/` folders). If you notice bugs elsewhere, log them in your notes rather than editing other directories.
- Your job stops once the scenario is fully initialized: research completed, PRD drafted, configuration + requirements scaffolding in place. **Do not implement a working P0 or ship business logic.**

## Quick Validation Loop (after scaffolding)
1. `vrooli scenario status {{TARGET}}`
   - Confirms scenario is set up correctly. Re-run at the end and capture notable warnings in your summary.
2. `scenario-auditor audit {{TARGET}} --timeout 240`
   - Expect failures (no code exists yet). Record the output path or summarize the key gaps for improvers.

## Generator Deliverables
1. **Research packet** – `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/RESEARCH.md`
   - Uniqueness check within the repo (`rg -l '{{TARGET}}' {{PROJECT_PATH}}/scenarios/`)
   - Related scenarios/resources + external references
2. **Scenario skeleton** – scaffold via the CLI template (see below) and keep the generated structure untouched except for configuration updates.
3. **Configuration & metadata** – `.vrooli/` directory populated so `vrooli scenario status {{TARGET}}` succeeds (service.json, endpoints.json, testing/lighthouse configs as required by the template).
4. **Operational Targets PRD** – `{{PROJECT_PATH}}/scenarios/{{TARGET}}/PRD.md` following the structure below.
5. **Requirements registry** – `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/index.json` (+ modules if helpful) plus `requirements/README.md` describing the organization.
6. **Documentation set** – README.md, docs/PROGRESS.md, docs/PROBLEMS.md, docs/RESEARCH.md initialized with baseline entries and instructions for future agents.

## Research & Template Selection
1. Explore existing scenarios/resources to ensure the capability is unique.
2. Use public/web research to understand the domain, typical features, and technical constraints.
3. Scaffold via the official template:
   - `vrooli scenario template list`
   - `vrooli scenario template show react-vite`
   - `vrooli scenario generate react-vite --id {{TARGET}} --display-name "{{TITLE}}" --description "<one sentence purpose>"`
   - Add `--var KEY=VALUE` for optional template variables (e.g., category, database name).
4. Follow the template’s post-generation checklist (dependency installs, go mod tidy, etc.) and note anything you skip.
5. All files now live at `{{PROJECT_PATH}}/scenarios/{{TARGET}}/`; run the rest of the steps from that directory.

## PRD Structure (Operational Targets Document)
Capture **what outcomes** we expect, not implementation details. Recommended outline:
```
# Overview
  - Purpose, target users/verticals, intended usage, deployment surfaces
  - Value proposition (qualitative impact, not revenue numbers)

# Operational Targets
  ## P0 – Must ship for viability
    - OT-ID: Title — narrative description of the outcome
      - Success signals / acceptance hints
      - Dependencies (resources/scenarios)
      - Linked technical requirement IDs
  ## P1 – Should have post-launch
  ## P2 – Future / expansion ideas

# Experience & Interfaces
  - Primary workflows / personas
  - Planned surfaces (UI screens, API endpoints, background jobs)
  - Access patterns (iframe, standalone portal, automation hooks)

# Tech Direction Snapshot
  - Preferred stacks (UI defaults to React/TS/Vite/shadcn/lucide; note API/storage choices)
  - Integration strategy with other scenarios/resources
  - Non-goals / out-of-scope statements

# Dependencies & Launch Plan
  - Required resources + scenario dependencies
  - Operational risks / open questions
```
Each operational target must name its technical requirement IDs so improvers know what to implement.

### Operational Target Tiers
- **P0**: Without this outcome the scenario fails (core capability).
- **P1**: Important enhancements enabling scale, security, multi-user flows, and other professional and mature features
- **P2**: Nice-to-have polish or expansion ideas. Important for making product unique and enticing for potential customers
Use concise narratives and focus on measurable outcomes rather than implementation details.

## `.vrooli/` Setup Checklist
- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/service.json` – service metadata, tags, port ranges. Keep ports in the scenario allocation bands.
- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/endpoints.json` (and any additional files such as testing.json, lighthouse.json) so lifecycle commands know how to monitor/test the scenario.
- Document any required environment variables or secrets in README.md.

## Requirements Registry Seeding
- Create `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/index.json` (break into modules if helpful).
- For every operational target, add requirement entries with:
  - `id` (pattern `[A-Z][A-Z0-9]+-[A-Z0-9-]+`), title, `prd_ref`, `criticality`, optional children/dependencies, empty `validation` array.
  - Notes on which phases (unit/integration/business/performance) should validate the requirement once implemented.
- In README/PRD, instruct improvers to tag tests with `[REQ:ID]` and run the scenario’s test phases; the lifecycle automatically syncs requirement coverage when tests execute (no manual `report.js` calls).

## Documentation Checklist
- **README.md**
  - Purpose summary, how to run (`vrooli scenario run {{TARGET}}`), how to test once code exists, link to PRD + docs
- **docs/PROGRESS.md**
  ```
  | Date | Author | Status Snapshot | Notes |
  |------|--------|-----------------|-------|
  | YYYY-MM-DD | Generator Agent | Initialization complete | Scenario scaffold + PRD seeded |
  ```
- **docs/PROBLEMS.md** – sections for “Open Issues” and “Deferred Ideas” with bullets describing known risks.
- **docs/RESEARCH.md** – uniqueness check, overlapping scenarios/resources, external references.
- **requirements/README.md** – explain module layout, naming pattern, how to tag tests, and which commands run the phases.

Future agents append to these files; call that out in README.md so progress remains centralized.

## Collision Avoidance & Validation
- Stay inside `{{PROJECT_PATH}}/scenarios/{{TARGET}}/`. Mention any repo-wide issues in your summary.
- At the end, run:
  1. `vrooli scenario status {{TARGET}}` – confirm lifecycle metadata loads without schema errors.
  2. `scenario-auditor audit {{TARGET}} --timeout 240` – capture the output path or summarize the gaps for improvers.
- Include the command outputs (or summaries) in your final handoff so the next agent knows the exact starting point.

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
