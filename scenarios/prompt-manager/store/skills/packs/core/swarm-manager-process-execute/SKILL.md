# Process: Execute

## Purpose

Carry out a general execution task that doesn't fit the idea or fix categories. Execute tasks include operations, maintenance, refactoring, documentation, and other actionable work.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-processing-guidance` — shared processing workflow and decision hierarchy.

## Output Requirements

- Complete the requested action
- Write `summary.md` in item folder with:
  - What was accomplished
  - Actions taken
  - Outputs/artifacts produced
  - Verification of completion

## Success Criteria

- [ ] Requested action completed as specified
- [ ] Outputs match expectations
- [ ] No unintended side effects
- [ ] Completion summary documents results
- [ ] Any artifacts properly stored/referenced

## Instructions

You are executing a task for the Swarm Manager. Your goal is to complete the requested action exactly as specified and document what was done.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Understand the task**
   - Read `spec.json` carefully
   - Review `research/summary.md` if available
   - Check any reference files provided
   - Identify success criteria

2. **Plan the execution**
   - Break down into specific steps
   - Identify dependencies or prerequisites
   - Note any risks or considerations

3. **Execute systematically**
   - Follow the plan step by step
   - Document progress as you go
   - Handle errors gracefully

4. **Verify completion**
   - Check all requirements met
   - Validate outputs
   - Confirm no side effects

5. **Document results**
   - Write comprehensive summary
   - Include evidence of completion
   - Note any follow-up items

### Common Execute Task Types

#### Operations Tasks
Examples: Deploy, migrate, configure, backup
```markdown
## Execution Summary: Database Migration

### Task
Migrate user table to add new email_verified column

### Actions Taken
1. Created migration file: `migrations/20240115_add_email_verified.sql`
2. Applied migration to dev environment
3. Verified column exists with correct default
4. Applied migration to staging

### Verification
- [x] Column exists in dev: `SELECT column_name FROM information_schema.columns WHERE table_name='users' AND column_name='email_verified'` → 1 row
- [x] Default value correct: false
- [x] Existing rows unaffected
- [x] Staging verified

### Artifacts
- Migration file: `migrations/20240115_add_email_verified.sql`
- Rollback script: `migrations/rollback_20240115.sql`
```

#### Maintenance Tasks
Examples: Update dependencies, clean up, optimize
```markdown
## Execution Summary: Dependency Update

### Task
Update all Go dependencies to latest compatible versions

### Actions Taken
1. Ran `go get -u ./...` in each scenario
2. Ran `go mod tidy` to clean up
3. Verified builds pass
4. Ran tests

### Changes
| Scenario | Package | Old | New |
|----------|---------|-----|-----|
| prompt-manager | chi | 5.0.0 | 5.1.0 |
| swarm-manager | pgx | 5.4.0 | 5.5.0 |

### Verification
- [x] All scenarios build
- [x] All tests pass
- [x] No deprecation warnings
```

#### Documentation Tasks
Examples: Write docs, update README, create guides
```markdown
## Execution Summary: API Documentation

### Task
Document the backlog API endpoints

### Actions Taken
1. Reviewed all endpoints in handler.go
2. Created OpenAPI spec at `api/openapi.yaml`
3. Updated README with API overview
4. Added example requests

### Outputs
- OpenAPI spec: `scenarios/swarm-manager/api/openapi.yaml`
- README section: API Documentation (lines 45-120)

### Verification
- [x] All endpoints documented
- [x] Examples are accurate
- [x] OpenAPI spec validates
```

#### Refactoring Tasks
Examples: Reorganize code, improve structure
```markdown
## Execution Summary: Handler Refactoring

### Task
Split large handler.go into domain-specific files

### Actions Taken
1. Extracted item handling to `item_handler.go`
2. Extracted file operations to `file_handler.go`
3. Extracted research operations to `research_handler.go`
4. Updated imports and references

### Changes
| File | Action | Lines |
|------|--------|-------|
| handler.go | Modified | 1200 → 400 |
| item_handler.go | Created | 350 |
| file_handler.go | Created | 250 |
| research_handler.go | Created | 200 |

### Verification
- [x] Build passes
- [x] All tests pass
- [x] No functionality changed
- [x] Code review passed
```

### Execution Guidelines

- **Follow instructions exactly** - Don't add unrequested work
- **Document as you go** - Don't rely on memory
- **Verify at each step** - Catch errors early
- **Leave clean state** - Don't leave partial work

## Quality Guidelines

**Good execution:**
- Task completed exactly as specified
- Clear documentation of what was done
- Evidence of verification
- Clean, no side effects

**Poor execution:**
- Scope creep or deviations
- Undocumented changes
- No verification
- Left partial state

## Anti-Patterns

- **Don't** expand scope beyond what's requested
- **Don't** skip verification steps
- **Don't** leave work undocumented
- **Don't** proceed without understanding the task
- **Don't** ignore errors - handle or escalate them

## Edge Cases

### If Task is Blocked
- Document what's blocking
- Document what was accomplished (if partial)
- Leave system in clean state
- Note what's needed to unblock

### If Task is Ambiguous
- Check research folder for clarification
- Make reasonable assumptions
- Document assumptions in summary
- Flag for review if critical

### If Task has Side Effects
- Document expected side effects
- Verify side effects occurred correctly
- Note in summary for awareness
