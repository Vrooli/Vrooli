You are executing a **scenario generation** task for the Ecosystem Manager.

## Boundaries & Focus
- Only modify `{{PROJECT_PATH}}/scenarios/{{TARGET}}/` (and its `.vrooli/`, `docs/`, and `requirements/` folders). If you notice bugs elsewhere, log them in your notes rather than editing other directories.
- Your job stops once the scenario is fully initialized: research completed, PRD drafted, configuration + requirements scaffolding in place. **Do not implement a working P0 or ship business logic.**

## Quick Validation Loop (after scaffolding)
1. `vrooli scenario status {{TARGET}}`
   - Confirms scenario is set up correctly. Re-run at the end and capture notable warnings in your summary.
2. `scenario-completeness-scoring score {{TARGET}}`
   - Shows objective quality score (0-100) with breakdown of quality, coverage, quantity, and UI metrics. Expect low scores initially (no code exists yet), but review recommendations to understand what future iterations should prioritize. Include the score and key metrics in your final handoff.
3. `scenario-auditor audit {{TARGET}} --timeout 240`
   - Expect failures (no code exists yet). Capture the summary-focused output (severity counts, top violations, artifact path) for your final handoff.

## Generator Deliverables
1. **Research packet** – `{{PROJECT_PATH}}/scenarios/{{TARGET}}/docs/RESEARCH.md`
   - Uniqueness check within the repo (`rg -l '{{TARGET}}' {{PROJECT_PATH}}/scenarios/`)
   - Related scenarios/resources + external references
2. **Scenario skeleton** – scaffold via the CLI template (see below) and keep the generated structure untouched except for configuration updates.
3. **Configuration & metadata** – `.vrooli/` directory populated so `vrooli scenario status {{TARGET}}` succeeds (service.json, endpoints.json, testing/lighthouse configs as required by the template).
4. **Operational Targets PRD** – `{{PROJECT_PATH}}/scenarios/{{TARGET}}/PRD.md` authored via the `business-health` wizard.
5. **Requirements registry** – `{{PROJECT_PATH}}/scenarios/{{TARGET}}/requirements/index.json` plus numbered operational-target folders (`01-<first-target-name>`, `02-<second-target-name>`, etc.) and a concise `requirements/README.md` explaining the mapping.
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

## PRD Authoring
Do not hand-write `PRD.md` from a blank page. Author `{{PROJECT_PATH}}/scenarios/{{TARGET}}/PRD.md` through the `business-health` wizard — a deterministic interview whose output is conformant by construction (canonical shape: `scenarios/business-health/docs/reference/canonical-prd-template.md`):

1. Draft the product intent yourself. No strict format required, but decide:
- Overview: purpose, target users/verticals, deployment surfaces, value proposition
- P0 operational targets list: Without these targets, the scenario fails (core capability)
- P1 operational targets list: Important enhancements enabling scale, security, multi-user flows, and other professional and mature features
- P2 operational targets list: Nice-to-have polish or expansion ideas. Important for making product unique and enticing for potential customers
- Tech direction snapshot: preferred stacks, storage expectations, integration strategy, non-goals
- Dependencies & launch plan: required resources, scenario dependencies, operational risks, sequencing
- UX & branding: desired look/feel, accessibility bar, brand tone

Make sure you *do*:
- Use concise narratives and focus on measurable outcomes rather than implementation details.

Make sure you *do not:*
- Reference requirements

2. Drive the wizard (interactive TTY, or supply the answers as JSON):
```bash
business-health wizard start {{TARGET}} --interactive
# or non-interactive: business-health wizard answer {{TARGET}} --answers /tmp/wizard_answers_{{TARGET}}.json
business-health wizard preview {{TARGET}}
business-health wizard apply {{TARGET}}
```

Take the wizard's capability-dedup hints seriously — "similar capability exists in scenario X" means compose that scenario rather than rebuilding it.

## Requirements Registry

The wizard's apply step already scaffolded the registry from your operational targets:
- `requirements/index.json` - Module registry linking to PRD targets
- `requirements/README.md` - Documentation with auto-sync guidance and validation commands
- `requirements/<nn>-<tier>/module.json` - Per-target requirement stubs

Flesh out requirement descriptions and validation refs, then validate (deterministic gaps are auto-fixable):
```bash
vrooli scenario requirements validate {{TARGET}} --json
business-health fix preview {{TARGET}}
business-health fix apply {{TARGET}}
```

## `.vrooli/` Setup Checklist
- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/service.json` – service metadata, tags, port ranges. Keep ports in the scenario allocation bands.
- `{{PROJECT_PATH}}/scenarios/{{TARGET}}/.vrooli/endpoints.json` (and any additional files such as testing.json, lighthouse.json) so lifecycle commands know how to monitor/test the scenario.
- Document any required environment variables or secrets in README.md.

## Requirements Registry (Reference)
The requirements registry is scaffolded automatically by `business-health wizard apply` (see above). The structure created is:
- `requirements/index.json` - Module registry with metadata
- `requirements/README.md` - Documentation with validation commands and test tagging guidance
- `requirements/01-<target-name>/module.json` - Per-target requirements

Each requirement has:
- `id` (pattern `REQ-P0-001`, `REQ-P1-001`, etc.)
- `title`, `description`, `prd_ref` linking to operational target
- `criticality` derived from target (P0/P1/P2)
- `validation` phases (unit/integration/business/performance)

The generated README instructs improvers to tag tests with `[REQ:ID]` comments.

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
  2. `scenario-completeness-scoring score {{TARGET}}` – capture the score and key recommendations for improvers.
  3. `scenario-auditor audit {{TARGET}} --timeout 240` – capture the output path or summarize the gaps for improvers.
- Include the command outputs (or summaries) in your final handoff so the next agent knows the exact starting point.

## Final Handoff (Required Format)
1. **Validation Evidence** – Re-run the Quick Validation Loop (status + completeness + scenario-auditor) and list any important information/tips/links/findings. Include the completeness score and top recommendations.
2. **Deliverables & Files** – Enumerate the docs/configs you created or edited (PRD, README, `requirements/index.json`, `docs/PROGRESS.md`, etc.). Mention any template steps you intentionally skipped.
3. **Scenario Status** – Summarize what's complete (research, scaffold, requirements), what remains open, and any known failures or blockers.
4. **Next Iteration Notes** – Capture learnings, open questions, or prioritized recommendations so the next agent can continue seamlessly.

Use this block as your final response structure; ecosystem-manager parses it to gauge readiness and queue the next iteration.

## Task Context

**Task ID**: {{TASK_ID}}
**Title**: {{TITLE}}
**Type**: {{TYPE}}
**Operation**: {{OPERATION}}
**Priority**: {{PRIORITY}}
**Category**: {{CATEGORY}}
**Status**: {{STATUS}}
**Current Phase**: {{CURRENT_PHASE}}

### Origin Handoff

- Source: {{ORIGIN_SOURCE}}
- Backlog item: {{ORIGIN_BACKLOG_ITEM}}
- Item folder: {{ORIGIN_ITEM_FOLDER}}
- Handoff dir: {{ORIGIN_HANDOFF_DIR}}
- Handoff brief: {{ORIGIN_HANDOFF_BRIEF_PATH}}
- Handoff manifest: {{ORIGIN_HANDOFF_MANIFEST_PATH}}
- Handoff source index: {{ORIGIN_HANDOFF_SOURCE_INDEX_PATH}}

If a swarm-manager handoff is present, treat the handoff brief and manifest as the authoritative upstream planning contract. Read them before making product or scope decisions.

### Notes
{{NOTES}}
