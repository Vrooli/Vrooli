# Processing Guidance

## Purpose

This document provides shared guidance for all processing operations (process-idea, process-fix, process-execute). Agents should read this file before executing any processing task.

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — canonical reference for folder structure, artifact schemas, reading order, and decision hierarchy.

## Core Principles

### 1. Folder is Source of Truth

The backlog item folder at `{{ITEM_FOLDER}}` contains everything you need:

```
item-folder/
├── spec.json              # Item metadata (kind, title, description, status)
├── clarify/
│   └── questions.json     # Clarifying Q&A (if clarify workflow ran)
├── suggest/
│   └── suggestions.json   # Suggestions with accept/reject (if suggest ran)
├── enhance/
│   └── summary.md         # Refined plan (if enhance ran)
├── handoff/
│   ├── brief.md           # (idea only, generated at process-time) execution context for the Swarm workflow
│   ├── manifest.json      # (idea only) machine-readable downstream contract
│   └── source-index.json  # (idea only) pointers back to source artifacts
├── research/
│   └── summary.md         # Research findings (if deep research ran)
├── archive/
│   └── ...                # Superseded artifacts from previous runs
└── [user files]           # Any additional context added by user
```

**Always read what exists before acting.**

### 2. Respect Decisions

See `swarm-manager-backlog-tools` for the full reading order and decision hierarchy.

### 2.1 Preserve-First Artifact Policy (Ideas)

For idea processing that initializes or updates scenarios:

- If structured spec artifacts exist in backlog (`archive/PRD.md`, `path:archive/requirements/`), treat them as baseline source material.
- Copy/merge those artifacts first, then run the `vrooli scenario requirements validate` / `business-health fix` loops.
- Drive the business-health wizard with answers synthesized from the archive/enhance materials only as fallback when baseline artifacts are missing or irreparably invalid.
- For existing scenarios, default to merge-with-backup, not overwrite.
- Never silently discard existing scenario PRD/requirements content; record conflict decisions in `notes.md`.

### 2.2 Idea Execution Context

For `process-idea` runs, swarm-manager may generate `{{ITEM_FOLDER}}/handoff/` just before execution begins. When that directory exists:

- Treat the rendered plan as the execution contract. Use `handoff/brief.md` as supporting context; it preserves the finalized backlog decisions and provenance for the Swarm workflow.
- Treat `handoff/manifest.json` and `handoff/source-index.json` as machine-readable provenance, execution boundaries, and source pointers.
- Execute the next bounded plan slice through the declared swarm-manager workflow. Do not create a separate ecosystem-manager task or translate the handoff into a second queue.
- Do not handcraft a replacement summary from memory. Preserve the generated context and record the true frontier in the swarm-manager execution handoff.

### 3. Leave Clear Evidence

Every processing operation must leave evidence of what was done:

**For Ideas (scenario creation/improvement):**
- Update or create the scenario
- Write `notes.md` in the item folder with completion summary
- Include the executed plan slice or swarm-manager workflow execution reference
- Include the generated handoff directory when one was used
- List files created/modified
- Note any deviations from the plan

**For Fixes:**
- Apply the fix to the codebase
- Write `notes.md` with what was changed and why
- Reference the specific commits/changes
- Document verification steps taken

**For Execute tasks:**
- Perform the requested action
- Write `summary.md` with what was accomplished
- Include any outputs or artifacts

**For Research items:**
- Read `conclusion.md` produced by the research workshop
- Execute each action in the Actions section (create backlog items, update documents, etc.)
- Write `notes.md` with completion summary listing actions taken and any deviations

### 4. Completion Summary Requirements

Every processing operation must produce a summary in the item folder. This summary should include:

1. **What was done** - Specific actions taken
2. **Files affected** - Created, modified, deleted
3. **Deviations** - Any differences from the plan (and why)
4. **Verification** - How the result was verified
5. **Follow-up** - Any remaining work or considerations

Example completion summary:

```markdown
# Completion Summary

## Actions Taken
- Executed plan slice: `Foundation — make the generated scenario's onboarding and requirements truthful`
- Used the declared swarm-manager phased-plan workflow
- Preserved `handoff/` as provenance and supporting execution context

## Specification Highlights
- API with 3 endpoints, UI with React components
- Depends on postgres and redis resources
- Key suggestion accepted: use Redis for caching (s3)
- Admin panel deferred to v2 (scope reduction noted in enhance/summary.md)

## Verification
- [x] Workflow execution started with the rendered plan
- [x] Execution handoff records the completed frontier and evidence

## Follow-up
- Continue the next bounded slice through the declared swarm-manager workflow.
```

### 5. Error Handling

If processing cannot complete:

1. **Document what blocked you** - Be specific
2. **Document what was accomplished** - Partial progress matters
3. **Leave the codebase in a clean state** - Don't leave broken code
4. **Update item status** - Mark as blocked with reason

Do NOT:
- Silently fail
- Leave partial changes uncommitted
- Pretend completion when blocked

## Verification Steps

Before marking processing complete:

### For Scenario Creation/Improvement
- [ ] Rendered plan is the execution contract and its next bounded slice is explicit
- [ ] If `handoff/` exists, the execution prompt includes it as supporting context and provenance
- [ ] The declared swarm-manager workflow is started for the bounded slice
- [ ] Execution handoff records the true frontier, evidence, and follow-up
- [ ] service.json is valid (if creating/modifying scenario directly)
- [ ] No security vulnerabilities introduced
- [ ] No hardcoded secrets

### For Fixes
- [ ] Original bug no longer reproduces
- [ ] Existing tests pass
- [ ] New tests cover the fix
- [ ] No regressions in related functionality

### For Execute Tasks
- [ ] Requested action completed
- [ ] Output matches expectations
- [ ] Artifacts produced (if any) are correct

## Common Anti-Patterns

### Don't Ignore Context
- **Wrong**: Jump straight to implementation
- **Right**: Read all folder contents first, understand decisions made

### Don't Deviate Silently
- **Wrong**: Implement differently than planned without explanation
- **Right**: Document deviations and reasoning in completion summary

### Don't Over-Engineer
- **Wrong**: Add "nice to have" features not in scope
- **Right**: Implement exactly what's specified, note potential improvements

### Don't Under-Document
- **Wrong**: Complete the work with no summary
- **Right**: Always write completion summary with verification evidence

### Don't Leave Partial State
- **Wrong**: Stop mid-implementation with broken code
- **Right**: Reach a stable checkpoint or roll back

### Don't Forget Ecosystem Fit
- **Wrong**: Build in isolation
- **Right**: Consider integration with existing scenarios and resources

## Integration Patterns

When creating scenarios, follow Vrooli patterns:

### API Structure
```
scenario/
├── api/
│   ├── main.go
│   ├── internal/
│   │   └── [domain]/
│   │       ├── handler.go
│   │       ├── service.go
│   │       └── types.go
│   └── go.mod
```

### UI Structure
```
scenario/
├── ui/
│   ├── src/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
```

### Service Configuration
```json
{
  "name": "scenario-name",
  "version": "1.0.0",
  "services": {
    "api": {
      "type": "go",
      "port": 8080
    },
    "ui": {
      "type": "vite",
      "port": 3000
    }
  }
}
```

## Using Existing Resources

When your scenario needs common capabilities:

| Need | Use Resource | How |
|------|--------------|-----|
| Database | postgres | Connect via `POSTGRES_URL` env var |
| Caching | redis | Connect via `REDIS_URL` env var |
| Vector search | qdrant | Connect via `QDRANT_URL` env var |
| LLM inference | ollama | Connect via `OLLAMA_URL` env var |
| Code execution | judge0 | Connect via `JUDGE0_URL` env var |

Never reinvent what resources provide.
