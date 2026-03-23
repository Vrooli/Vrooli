# Process: Idea

## Purpose

Orchestrate the creation or improvement of a Vrooli scenario by reading a fully-refined idea specification. The primary source of truth is `plan.md` (the implementation plan produced through the workshop loop). For **new scenarios**, initialize the scenario scaffold, PRD, and requirements before delegating to ecosystem-manager. For **existing scenarios**, delegate improvement directly to ecosystem-manager's steered agent loops.

## Input Context

**Required reading:**
- `prompt-manager skill read swarm-manager-processing-guidance ecosystem-manager-tools` -- shared processing workflow, decision hierarchy, and CLI reference

## Scope

**In scope:**
- Reading and synthesizing idea context (plan.md as primary source, workshop rounds, research, archive artifacts)
- Determining operation type (new scenario vs improve existing)
- For new scenarios: scaffolding, PRD generation, requirements registry, archive incorporation
- For improvements: incorporating staging/archive materials into the existing scenario, selecting a steering strategy (profile, steer-mode, or steer-queue)
- Creating ecosystem-manager tasks via CLI
- Verifying queue processor is running
- Documenting decisions in notes.md

**Out of scope:**
- Implementing scenario business logic directly (delegate to ecosystem-manager)
- Creating or modifying steering profiles (API/UI-only)
- Managing ecosystem-manager internals or process lifecycle

## Output Requirements

### For New Scenario
- Scenario scaffolded from appropriate template
- Existing structured artifacts preserved first: `archive/PRD.md` and `archive/requirements/` copied into the scenario when present
- PRD validated (and fixed if needed) via `prd-control-tower`; regenerate only as fallback when no viable baseline exists
- Requirements validated (and fixed if needed) via `prd-control-tower`; regenerate only as fallback when no viable baseline exists
- Archive materials incorporated into scenario docs/configuration
- Validation loop passed (status, completeness, auditor)
- Ecosystem-manager improvement task created with appropriate steering
- Queue processor running
- `notes.md` in item folder documenting initialization summary, task ID, steering strategy, and rationale

### For Scenario Improvement
- Staging/archive materials incorporated into scenario PRD, requirements, and docs using merge-with-backup by default
- Existing scenario artifacts are not overwritten silently; conflicts are documented in `notes.md`
- Ecosystem-manager task created targeting existing scenario
- Queue processor running
- `notes.md` in item folder documenting materials incorporated, task ID, steering strategy, and rationale

## Success Criteria

- [ ] All context read and understood (plan.md, workshop rounds, research, archive)
- [ ] Operation type determined (new vs improve)
- [ ] **If new scenario:**
  - [ ] Template selected and scenario scaffolded
  - [ ] Archive baseline artifacts copied first when available (`archive/PRD.md`, `archive/requirements/`)
  - [ ] PRD validated/fixed after copy; generation used only as fallback
  - [ ] Requirements validated/fixed after copy; generation used only as fallback
  - [ ] Archive materials incorporated into scenario
  - [ ] Validation loop passed
- [ ] Steering strategy selected with clear rationale
- [ ] Ecosystem-manager task created with correct steering
- [ ] Queue processor confirmed running
- [ ] All accepted suggestions reflected in task specification
- [ ] All answered questions respected in task configuration
- [ ] Completion summary written with task ID and monitoring instructions

## Instructions

You are processing an idea to create or improve a Vrooli scenario. Your role is to **orchestrate** -- read the refined specification, initialize the scenario if it's new, select the right steering strategy, and delegate to ecosystem-manager for iterative implementation.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

> **Note:** CLI commands below are summarized for quick reference. For full CLI documentation, flags, and troubleshooting, see `ecosystem-manager-tools` (required reading).

### Processing Steps

1. **Read all context**
   - Start with `enhance/summary.md` (the refined plan)
   - Review `clarify/questions.json` for answered questions
   - Check `suggest/suggestions.json` for accepted suggestions
   - Read `research/summary.md` if available
   - **Check for staging materials in `enhance/`** — `prd-context.md`, `requirements-context.md`, `doc-outlines.md`. These are pre-synthesized by the enhance step and are the preferred source for scenario incorporation.
   - **Fall back to `archive/`** — if enhance staging materials don't exist (enhance step didn't run or pre-dates staging support), inventory `archive/` files directly. These may contain requirements docs, design mockups, reference material, or prior work that must be incorporated.
   - Note any user-added context files in the item root

   > **Staging vs archive:** The enhance step synthesizes raw archive materials into ready-to-use staging artifacts in `enhance/`. Always prefer `enhance/` staging materials when available — they incorporate answered questions, accepted suggestions, and resolved conflicts. Only fall back to raw `archive/` when staging materials are absent.

2. **Determine operation type**
   - **Archived revival guardrail**: if `spec.json` indicates archive metadata (`sourceScenarioName`, `archiveReason`, `preservedFiles`), treat this as a scenario revival workflow. Use `sourceScenarioName` as the target scenario ID unless user explicitly overrides.
   - **Target naming guardrail**: do not assume backlog folder `<name>` is the scenario ID. Use `TARGET_SCENARIO_ID` when provided; fallback to `<name>` only when no override metadata exists.
   - **Readiness gate**: if `clarify/questions.json` exists and any `importance: critical` question has an empty/null answer, stop and write `notes.md` explaining the blocking questions. Do not create an ecosystem-manager task until critical gaps are resolved.
   - **New scenario**: No existing scenario at `scenarios/<name>/` → proceed to **step 3** (Initialize Scenario)
   - **Improve existing**: Target scenario already exists → skip to **step 5** (Search Existing Steers)

   Verify with:
   ```bash
   vrooli scenario status <name>
   ```
   A failure or "not found" confirms this is a new scenario.

---

### New Scenario Path (steps 3–4)

3. **Initialize the scenario**

   This step scaffolds a new scenario from a template and generates its PRD and requirements, incorporating all context from the backlog item. Follow these sub-steps in order:

   #### 3a. Select template

   Review available templates and pick the best match for the idea's tech stack:
   ```bash
   vrooli scenario template list
   ```

   Use the enhanced specification to guide selection:
   | Spec Characteristic | Recommended Template |
   |---------------------|---------------------|
   | Has a web UI | `react-vite` |
   | API-only / backend service | Go or API-focused template |
   | CLI tool | CLI-focused template |

   If unsure, `react-vite` is a safe default for scenarios with any user-facing component.

   #### 3b. Scaffold the scenario

   ```bash
   vrooli scenario generate <template> --id <name> --display-name "{{ITEM_TITLE}}" --description "<one-line purpose from enhanced spec>"
   ```

   Follow the template's post-generation checklist (dependency installs, `go mod tidy`, etc.).

   #### 3c. Preserve-first PRD transfer (required)

   Before generating anything, check for structured baseline artifacts in `archive/`:

   ```
   Does archive/PRD.md exist?
     → Yes → Copy it to scenarios/<name>/PRD.md as the baseline
     → No  → Build PRD context for generation (fallback path)
   ```

   If baseline PRD exists, validate/fix in-place first:
   ```bash
   prd-control-tower prd validate <name> --json
   prd-control-tower prd fix <name> --auto --json
   prd-control-tower prd validate <name> --json
   ```

   Only if baseline is missing or still invalid after fix attempts, run generation:
   ```bash
   prd-control-tower prd generate <name> --context-file /tmp/prd_context_<name>.md --publish --json
   ```

   #### 3d. Preserve-first requirements transfer (required)

   Before generating requirements, check for a structured requirements baseline:

   ```
   Does archive/requirements/index.json exist?
     → Yes → Copy archive/requirements/ into scenarios/<name>/requirements/ as baseline
     → No  → Generate requirements from PRD (fallback path)
   ```

   If baseline requirements exist, validate/fix in-place first:
   ```bash
   prd-control-tower requirements validate <name> --json
   prd-control-tower requirements fix <name> --json
   prd-control-tower requirements validate <name> --json
   ```

   Only if baseline is missing or still invalid after fix attempts, run generation:
   ```bash
   prd-control-tower requirements generate <name> --context-file /tmp/requirements_context_<name>.md --json
   prd-control-tower requirements validate <name> --json
   ```

   > **Decision hierarchy reminder:** Answered questions are definitive. Accepted suggestions must be included. Rejected suggestions must NOT appear. `enhance/summary.md` supersedes `spec.json`. Research findings are advisory.

   #### 3f. Incorporate remaining materials

   Check for materials not fully captured in the PRD/requirements generation:

   ```
   Does enhance/doc-outlines.md exist?
     → Yes → Use its outlines to seed scenario documentation
     → No  → Review archive/ for documentation-relevant materials
   ```

   For any remaining archive files not covered by staging artifacts:
   - Documentation → `scenarios/<name>/docs/`
   - Reference configs → `scenarios/<name>/.vrooli/` or noted in README
   - Other materials → Referenced in `docs/RESEARCH.md` with a note on how they informed the PRD

   #### 3g. Initialize documentation

   Ensure these files exist with baseline content (the template may have created some already). Use `enhance/doc-outlines.md` as a guide if available:
   - `README.md` — purpose, how to run, link to PRD
   - `docs/PROGRESS.md` — initialization entry with date
   - `docs/PROBLEMS.md` — open issues and deferred ideas from research
   - `docs/RESEARCH.md` — uniqueness check, related scenarios, references (incorporate `research/summary.md` findings)

   #### 3h. Run validation loop

   ```bash
   vrooli scenario status <name>
   scenario-completeness-scoring score <name>
   scenario-auditor audit <name> --timeout 240
   ```

   Expect low completeness scores and auditor failures (no implementation yet). Capture the results for your notes and handoff.

4. **Transition to improvement**

   The scenario now exists but has no business logic. Proceed to **step 5** to create an ecosystem-manager improvement task that will implement the P0 operational targets.

---

### Improvement Path (steps 5–10)

5. **Incorporate materials into existing scenario (merge + backup default)**

   Even for existing scenarios, the backlog item may contain refined materials that should update the scenario's PRD, requirements, or documentation before the improvement task runs. This supports the "idea as staging area" workflow — users create ideas to refine plans and then process them to update an existing scenario.

   ```
   Do enhance/ staging materials exist (prd-context.md, requirements-context.md, doc-outlines.md)?
     → Yes → Use staging materials to update the scenario (preferred)
     → No  → Do archive/ files exist?
              → Yes → Incorporate archive materials directly
              → No  → Skip (no materials to incorporate, proceed to steering)
   ```

   When materials exist:
   - Create backups before editing scenario PRD/requirements.
   - **PRD update**:
     - If archive provides `PRD.md`, merge it into current `scenarios/<name>/PRD.md` (do not blind overwrite).
     - Validate/fix after merge:
       ```bash
       prd-control-tower prd validate <name> --json
       prd-control-tower prd fix <name> --auto --json
       prd-control-tower prd validate <name> --json
       ```
     - Only use `prd generate` as fallback when no viable baseline exists.
   - **Requirements update**:
     - If archive provides `requirements/`, merge into current `scenarios/<name>/requirements/` (do not blind overwrite).
     - Validate/fix after merge:
       ```bash
       prd-control-tower requirements validate <name> --json
       prd-control-tower requirements fix <name> --json
       prd-control-tower requirements validate <name> --json
       ```
     - Only use `requirements generate` as fallback when no viable baseline exists.
   - **Documentation update**: If `enhance/doc-outlines.md` or archive contains documentation, update the scenario's docs accordingly.
   - If merge conflicts or ambiguous mappings appear, stop and document the conflict decisions in `notes.md`.

   > **Why this matters:** Without this step, creating an idea for an existing scenario, refining it through clarify/suggest/enhance, and processing it would only create an improvement task — but the scenario's PRD and requirements would remain stale. This step ensures the scenario's spec stays in sync with the refined plan.

6. **Search existing steers**
   ```bash
   ecosystem-manager steer profiles
   ecosystem-manager steer templates
   ```
   Review available profiles to find one that matches the idea's scope and needs.

7. **Select steering strategy**

   Use this decision process:
   - Review the refined specification for key characteristics (scope, quality needs, UX requirements)
   - Match against the decision table in ecosystem-manager-tools
   - **Prefer an existing profile** when one fits the idea's scope
   - **Fall back to `--steer-mode`** (single mode) or **`--steer-queue`** (ordered list) only when no profile matches
   - Document your rationale for the selection

8. **Create ecosystem-manager task**

   For a newly initialized scenario (came from step 4):
   ```bash
   ecosystem-manager task improve --steer-profile <profile-id> scenario <name>
   ```

   For an existing scenario improvement:
   ```bash
   ecosystem-manager task improve --steer-profile <profile-id> scenario <name>
   ```

   Or with a single steer mode:
   ```bash
   ecosystem-manager task improve --steer-mode "<mode>" scenario <name>
   ```

   Or with custom steering queue:
   ```bash
   ecosystem-manager task improve --steer-queue "progress,test,refactor" scenario <name>
   ```

   > **Tip:** Use `--dry-run` to validate task configuration before creating:
   > ```bash
   > ecosystem-manager task improve --dry-run --steer-profile balanced scenario <name>
   > ```

   > **Common issue:** If you get "task already exists for this scenario," check existing tasks with `ecosystem-manager task list --type scenario` before creating. For all other errors, see the troubleshooting table in `ecosystem-manager-tools`.

9. **Ensure queue is running**
   ```bash
   ecosystem-manager queue status
   ecosystem-manager queue start  # if paused
   ```

10. **Write notes.md** in the item folder:

   For **new scenarios** (came through initialization):
   ```markdown
   # Processing Notes

   ## Initialization
   - **Template**: <template used>
   - **PRD**: `scenarios/<name>/PRD.md`
   - **Requirements**: `scenarios/<name>/requirements/index.json`
   - **Archive materials incorporated**: <list of archive files and how they were used>
   - **Validation**: status ✓ | completeness score: <N>/100 | auditor: <summary>

   ## Task
   - **ID**: <ecosystem-manager task ID>
   - **Monitor**: `ecosystem-manager task show <id>`

   ## Steering
   - **Strategy**: <profile name or custom queue>
   - **Rationale**: <why this steering was chosen>

   ## Specification Summary
   - <key points from enhance/summary.md>
   - <accepted suggestions reflected>
   - <answered questions respected>
   ```

   For **improvements** (skipped initialization):
   ```markdown
   # Processing Notes

   ## Materials Incorporated
   - **Source**: <enhance/ staging materials | archive/ fallback | none>
   - **PRD updated**: <yes/no — what changed>
   - **Requirements updated**: <yes/no — what changed>
   - **Docs updated**: <yes/no — what changed>

   ## Task
   - **ID**: <ecosystem-manager task ID>
   - **Monitor**: `ecosystem-manager task show <id>`

   ## Steering
   - **Strategy**: <profile name or custom queue>
   - **Rationale**: <why this steering was chosen>

   ## Specification Summary
   - <key points from enhance/summary.md>
   - <accepted suggestions reflected>
   - <answered questions respected>
   ```

### Steering Selection Guidelines

When choosing a steering strategy, consider:

| Idea Characteristic | Recommended Profile | Reasoning |
|---------------------|-------------------|-----------|
| Newly initialized scenario (from step 4) | `balanced` or `rapid-mvp` | Needs broad first-pass implementation |
| Quality/stability focus | `production-ready` | Extra test and refactor passes |
| Heavy UI/UX component | `ux-excellence` | Dedicated UX iteration modes |
| Test debt or refactoring | `refactor-test-focus` | Prioritizes code quality |
| Single improvement focus | `--steer-mode` | Single-pass improvement (e.g., "test") |
| Unique improvement needs | Custom `--steer-queue` | Tailor mode order to specific requirements |

## Quality Guidelines

**Good orchestration:**
- Thoroughly reads all context before deciding (including archive)
- For new scenarios: synthesizes a comprehensive PRD brief from all backlog artifacts
- For new scenarios: incorporates archive materials into the scenario structure
- Selects steering that matches the specification's needs
- Documents rationale clearly in notes.md
- Verifies task creation and queue status
- Provides monitoring instructions

**Poor orchestration:**
- Skips reading context artifacts or ignores archive materials
- For new scenarios: writes a thin PRD brief that drops backlog context
- For new scenarios: skips validation loop after initialization
- Picks a profile without considering the specification
- Fails to document steering rationale
- Does not verify queue is running
- Leaves no monitoring instructions

## Anti-Patterns

- **Don't** implement scenario business logic directly -- delegate to ecosystem-manager
- **Don't** ignore answered questions or accepted suggestions when configuring the task
- **Don't** skip reading the ecosystem-manager-tools skill
- **Don't** use custom steer queues when an existing profile fits
- **Don't** forget to verify the queue processor is running
- **Don't** omit the task ID from notes.md
- **Don't** skip scenario initialization for new scenarios -- there is no scenario to improve without it
- **Don't** process archived-revival ideas under the `*-archived` folder name when `sourceScenarioName` is present -- target the source scenario name unless explicitly overridden
- **Don't** queue ecosystem-manager work while critical clarify questions remain unanswered
- **Don't** discard staging or archive materials -- they represent refined/user-provided context that must be incorporated
- **Don't** use raw archive materials when enhance/ staging materials exist -- staging is pre-synthesized and preferred
- **Don't** hand-write PRD.md -- always generate via `prd-control-tower`

## Troubleshooting

For CLI errors, steering validation failures, or queue issues, refer to the troubleshooting table in `ecosystem-manager-tools` (`prompt-manager skill read ecosystem-manager-tools`).

### Scenario Initialization Issues

| Problem | Solution |
|---------|----------|
| `vrooli scenario template list` shows no templates | Check that the CLI is installed: `vrooli --version` |
| `prd-control-tower` command not found | Ensure ecosystem-manager resources are available |
| PRD generation fails | Verify the context file exists and is valid markdown |
| Completeness score is 0 | Expected for a freshly initialized scenario with no implementation |
| Auditor reports many failures | Expected -- capture the summary for handoff to the improver agent |
