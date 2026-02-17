# Process: Idea

## Purpose

Orchestrate the creation or improvement of a Vrooli scenario by reading a fully-refined idea specification and delegating implementation to the ecosystem-manager's steered agent loops.

## Input Context

**Required reading:**
- `prompt-manager skill read swarm-manager-processing-guidance ecosystem-manager-tools` -- shared processing workflow, decision hierarchy, and CLI reference

## Scope

**In scope:**
- Reading and synthesizing idea context (enhance, clarify, suggest, research artifacts)
- Selecting a steering strategy (profile, steer-mode, or steer-queue)
- Creating ecosystem-manager tasks via CLI
- Verifying queue processor is running
- Documenting decisions in notes.md

**Out of scope:**
- Implementing scenario code directly (delegate to ecosystem-manager)
- Creating or modifying steering profiles (API/UI-only)
- Managing ecosystem-manager internals or process lifecycle

## Output Requirements

### For New Scenario
- Ecosystem-manager task created with appropriate steering
- Queue processor running
- `notes.md` in item folder documenting task ID, steering strategy, and rationale

### For Scenario Improvement
- Ecosystem-manager task created targeting existing scenario
- Queue processor running
- `notes.md` in item folder documenting task ID, steering strategy, and rationale

## Success Criteria

- [ ] All context read and understood (enhance, clarify, suggest, research)
- [ ] Operation type determined (new vs improve)
- [ ] Steering strategy selected with clear rationale
- [ ] Ecosystem-manager task created with correct steering
- [ ] Queue processor confirmed running
- [ ] All accepted suggestions reflected in task specification
- [ ] All answered questions respected in task configuration
- [ ] Completion summary written with task ID and monitoring instructions

## Instructions

You are processing an idea to create or improve a Vrooli scenario. Your role is to **orchestrate** -- read the refined specification, select the right steering strategy, and delegate to ecosystem-manager for iterative implementation.

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
   - Note any user-added context files

2. **Determine operation type**
   - **New scenario**: No existing scenario, use `task add`
   - **Improve existing**: Target scenario exists, use `task improve`

3. **Search existing steers**
   ```bash
   ecosystem-manager steer profiles
   ecosystem-manager steer templates
   ```
   Review available profiles to find one that matches the idea's scope and needs.

4. **Select steering strategy**

   Use this decision process:
   - Review the refined specification for key characteristics (scope, quality needs, UX requirements)
   - Match against the decision table in ecosystem-manager-tools
   - **Prefer an existing profile** when one fits the idea's scope
   - **Fall back to `--steer-mode`** (single mode) or **`--steer-queue`** (ordered list) only when no profile matches — these only work on scenario improver tasks
   - Document your rationale for the selection

5. **Create ecosystem-manager task**

   For a new scenario:
   ```bash
   ecosystem-manager task add --steer-profile <profile-id> scenario <name>
   ```

   For an improvement:
   ```bash
   ecosystem-manager task improve --steer-profile <profile-id> scenario <name>
   ```

   Or with a single steer mode (scenario improver tasks only):
   ```bash
   ecosystem-manager task improve --steer-mode "<mode>" scenario <name>
   ```

   Or with custom steering queue (scenario improver tasks only):
   ```bash
   ecosystem-manager task improve --steer-queue "progress,test,refactor" scenario <name>
   ```

   > **Tip:** Use `--dry-run` to validate task configuration before creating. This runs full validation without persisting:
   > ```bash
   > ecosystem-manager task add --dry-run --steer-profile balanced scenario <name>
   > ```

   > **Common issue:** If you get "task already exists for this scenario," check existing tasks with `ecosystem-manager task list --type scenario` before creating. For all other errors, see the troubleshooting table in `ecosystem-manager-tools`.

6. **Ensure queue is running**
   ```bash
   ecosystem-manager queue status
   ecosystem-manager queue start  # if paused
   ```

7. **Write notes.md** in the item folder:
   ```markdown
   # Processing Notes

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
| Standard new scenario | `balanced` or `rapid-mvp` | Good default coverage |
| Quality/stability focus | `production-ready` | Extra test and refactor passes |
| Heavy UI/UX component | `ux-excellence` | Dedicated UX iteration modes |
| Test debt or refactoring | `refactor-test-focus` | Prioritizes code quality |
| Single improvement focus | `--steer-mode` (scenario improver only) | Single-pass improvement (e.g., "test") |
| Unique improvement needs | Custom `--steer-queue` (scenario improver only) | Tailor mode order to specific requirements |

## Quality Guidelines

**Good orchestration:**
- Thoroughly reads all context before deciding
- Selects steering that matches the specification's needs
- Documents rationale clearly in notes.md
- Verifies task creation and queue status
- Provides monitoring instructions

**Poor orchestration:**
- Skips reading context artifacts
- Picks a profile without considering the specification
- Fails to document steering rationale
- Does not verify queue is running
- Leaves no monitoring instructions

## Anti-Patterns

- **Don't** implement scenario code directly -- delegate to ecosystem-manager
- **Don't** ignore answered questions or accepted suggestions when configuring the task
- **Don't** skip reading the ecosystem-manager-tools skill
- **Don't** use custom steer queues when an existing profile fits
- **Don't** forget to verify the queue processor is running
- **Don't** omit the task ID from notes.md

## Troubleshooting

For CLI errors, steering validation failures, or queue issues, refer to the troubleshooting table in `ecosystem-manager-tools` (`prompt-manager skill read ecosystem-manager-tools`).
