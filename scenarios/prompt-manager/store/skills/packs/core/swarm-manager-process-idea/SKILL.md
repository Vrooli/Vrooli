# Process: Idea

## Purpose

Orchestrate the creation or improvement of a Vrooli scenario by reading a fully-refined idea specification and delegating implementation to the ecosystem-manager's steered agent loops.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` -- shared processing workflow and decision hierarchy.

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

3. **Read ecosystem-manager-tools skill**
   ```
   prompt-manager skill read ecosystem-manager-tools
   ```
   This provides the full CLI reference, steering profiles, and decision table.

4. **Search existing steers**
   ```bash
   ecosystem-manager steer profiles
   ecosystem-manager steer templates
   ```
   Review available profiles to find one that matches the idea's scope and needs.

5. **Select steering strategy**

   Use this decision process:
   - Review the refined specification for key characteristics (scope, quality needs, UX requirements)
   - Match against the decision table in ecosystem-manager-tools
   - **Prefer an existing profile** when one fits the idea's scope
   - **Fall back to `--steer-queue`** with a custom mode order only when no profile matches
   - Document your rationale for the selection

6. **Create ecosystem-manager task**

   For a new scenario:
   ```bash
   ecosystem-manager task add scenario <name> --steer-profile <profile-id>
   ```

   For an improvement:
   ```bash
   ecosystem-manager task improve scenario <name> --steer-profile <profile-id>
   ```

   Or with custom steering:
   ```bash
   ecosystem-manager task add scenario <name> --steer-queue "progress,test,refactor"
   ```

7. **Ensure queue is running**
   ```bash
   ecosystem-manager queue status
   ecosystem-manager queue start  # if paused
   ```

8. **Write notes.md** in the item folder with:
   - Ecosystem-manager task ID
   - Chosen steering (profile name or custom queue)
   - Rationale for steering selection
   - How to monitor progress: `ecosystem-manager task show <id>`
   - Summary of key specification points passed to the task

### Steering Selection Guidelines

When choosing a steering strategy, consider:

| Idea Characteristic | Recommended Profile | Reasoning |
|---------------------|-------------------|-----------|
| Standard new scenario | `balanced` or `rapid-mvp` | Good default coverage |
| Quality/stability focus | `production-ready` | Extra test and refactor passes |
| Heavy UI/UX component | `ux-excellence` | Dedicated UX iteration modes |
| Test debt or refactoring | `refactor-test-focus` | Prioritizes code quality |
| Unique needs | Custom `--steer-queue` | Tailor mode order to specific requirements |

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
