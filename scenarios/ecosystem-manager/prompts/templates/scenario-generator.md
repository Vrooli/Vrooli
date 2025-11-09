You are executing a *scenario generation* task for Vrooli's ecosystem-manager scenario.

## Section 1: prd-protocol

# 📋 PRD Protocol: Single Source of Truth

**The PRD is the PRIMARY DELIVERABLE.**

# PRD Essentials

## Quick Reference
- **PRD** = Product Requirements Document
- **Priority**: P0 (must have) > P1 (should have) > P2 (nice to have)  
- **Progress**: Track via checkboxes (☐/✅) with completion percentages
- **Purpose**: Define permanent capability being added to Vrooli

## Core Structure
```markdown
## P0 Requirements (Must Have - 5-7 max)
- [ ] **Feature Name**: What it must do
- [ ] **Health Check**: Must respond with status
- [ ] **Lifecycle**: setup/develop/test/stop work

## P1 Requirements (Should Have - 3-4 max)  
## P2 Requirements (Nice to Have - 2-3 max)
```

## PRD Rules
1. **Checkbox accuracy is sacred** - Only check when tested/verified
2. **Progress reflects reality** - Percentages match actual completion  
3. **Requirements are testable** - Each has validation command
4. **PRD drives development** - Implementation follows requirements

## Required Sections
1. **Executive Summary** (5 lines): What/Why/Who/Value/Priority
2. **Requirements Checklist** (P0/P1/P2 with checkboxes)  
3. **Technical Specifications** (Architecture/Dependencies/APIs)
4. **Success Metrics** (Completion/Quality/Performance targets)

## Generator Rules (50% of effort)
**MUST**: 5+ P0 requirements, acceptance criteria, technical approach, revenue justification
**Process**: Research → Requirements → Technical → Metrics → PRD.md
**Quality Gate**: All sections filled, revenue > $10K justified

## Common Mistakes
**Generators**: Vague requirements, missing criteria, no revenue justification, <5 P0s

## PRD Checkbox Rules

### When to Check ✅
- Feature FULLY works as specified
- Tests pass for that feature
- Documentation exists
- No known bugs

### When to Uncheck ☐
- Feature broken/missing
- Tests fail
- Major bugs present
- Spec not met

### Partial Completion
Use notes, not checkmarks:
```markdown
- [ ] User authentication (PARTIAL: login works, logout broken)
```

## PRD Success Metrics

### Good Generator PRD
- Reader understands WHAT to build
- Reader understands WHY to build it
- Reader knows HOW to build it
- Reader can TEST if it works

## Golden Rules
1. PRD is contract - What's in PRD gets built
2. PRD is truth - Checkmarks must be honest  
3. PRD is progress - Track everything
4. PRD is permanent - Never delete history


## Section 2: security-requirements

# Security & Standards Enforcement

For resources:
```bash
resource-auditor audit <resource-name> --timeout 240 > /tmp/audit-<resource-name>.json
jq '{security: .security.outcome, standards: .standards.outcome}' /tmp/audit-<resource-name>.json
```

For scenarios:
```bash
scenario-auditor audit <scenario-name> --timeout 240 > /tmp/audit-<scenario-name>.json
jq '{security: .security.outcome, standards: .standards.outcome}' /tmp/audit-<scenario-name>.json
```

## Section 3: implementation-methodology

# Implementation Methodology

## Core Principles
**Standards**: Follow codebase patterns; maintain consistency; adhere to the lastest standards  
**Validation**: Test continuously; ensure gates pass; fix security/standards violations; document verification

## Implementation Phases

### Phase 1: Setup (10% effort)
- Verify dependencies, tools, permissions
- Define success criteria aligned with PRD requirement (see 'PRD Protocol' section) or violations fixing (see 'Security & Standards Enforcement' section)
- Prepare rollback procedure and test commands

### Phase 2: Core Implementation (60% effort)

**Generators (Creating New):**
- Start from appropriate template/reference
- Implement minimal viable functionality first
- Integrate incrementally: dependencies → service.json → CLI → API

**Quality Standards:**
- Well-organized and maintainable code. No monolithic files
- Clear descriptive naming; helpful error messages
- Graceful error handling with recovery hints
- Follow established CLI/API patterns

### Phase 3: Testing & Validation (20% effort)
**Required Tests:**
- Functional: core features, edge cases, error handling
- Integration: dependencies, APIs, CLI commands, UI (if applicable)
- Performance: response times, resource usage, scaling

**Test Commands:**
- **Resources**: See 'Resource Testing Reference' section
- **Scenarios**: See 'Scenario Testing Reference' section

### Phase 4: Documentation & Finalization (10% effort)
- Update README: features, usage, troubleshooting
- Update relevant docs: e.g. PROBLEMS.md with problems you discovered
- Update PRD per 'prd-protocol' section guidelines

Quality and maintainability over speed. Small steps win. Test everything.


## Section 4: failure-recovery

# Failure Recovery

## Purpose
Capture failures fast, choose the right response, and record what the next agent must know.

## Severity Snapshot
- **Level 1 – Trivial**: Cosmetic issues, non-blocking warnings, minor test flakes.
- **Level 2 – Minor**: Single feature broken, <20% performance dip, recoverable errors.
- **Level 3 – Major**: Core feature down, multiple failing tests, API contract break, data risk.
- **Level 4 – Critical**: Service will not start, data corruption, exposed vulnerability.
- **Level 5 – Catastrophic**: Active outage, data loss in progress, security breach.

## Response Decision Tree
```
Start
└─ Assess severity level (1-5)
    ├─ Levels 1-2 (Trivial/Minor)
    │     └─ Action: Keep iterating → apply targeted fix → rerun tests
    │            Documentation: Note quick fix in final summary (no formal failure log)
    ├─ Level 3 (Major)
    │     └─ Action: Stop feature work → stabilize scenario → verify regression removed
    │            Documentation: Add Failure Log (template below) + reference impacted PRD items
    └─ Levels 4-5 (Critical/Catastrophic)
          └─ Action: Stop immediately → follow Task Completion Protocol → do NOT attempt git rollback
                 Documentation: Complete Failure Log + highlight blockers + call out needed escalation
```

## Failure Log Template
Use when severity ≥3 or the issue remains unresolved at handoff.

```
### Failure Summary
- **Component**: [resource/scenario name]
- **Severity Level**: [1-5]
- **What Happened**: [Clear description]
- **Root Cause**: [If identified]

### What Was Attempted
1. [First fix attempt] — Result: [Failed/Partial/Success]
2. [Second fix attempt] — Result: [Failed/Partial/Success]

### Current State
- **Status**: [Working/Degraded/Broken]
- **Key Issues**: [Main problems]
- **Workaround**: [If any temporary fix was applied]

### Lessons Learned
- [What this failure teaches us]
- [How to prevent similar issues]
```

Refer back to 'Task Completion Protocol' for wrap-up steps once the log is captured.

## Key Principles
- Document everything — future agents rely on your trail.
- Edit forward — no git rollbacks; if it’s broken, stabilize or stop.
- Know when to halt — unresolved severity ≥3 means step back and report.
- Capture lessons — every failure should improve the ecosystem.


## Section 5: collision-avoidance

# Collision Avoidance

## Purpose
You work on one assigned resource or scenario. Avoid conflicts by staying within your boundaries and working around limitations gracefully.

## Core Principle: Stay in Your Lane
**Work only within your assigned resource/scenario directory.** Do not modify:
- Files in other resources/scenarios
- Core Vrooli CLI code
- Shared infrastructure files
- Other agents' assigned tasks

## Handling Limitations
When you encounter blockers:

**❌ DO NOT try to fix external issues:**
- Critical bugs in main Vrooli CLI
- Semantic knowledge system not working
- Other resources/scenarios broken

**✅ DO work around limitations:**
- Document the limitation in your response
- Find alternative approaches within your scope
- Focus on what you can control

## Managing Dependencies
**You MAY start/install existing resources/scenarios if:**
- They are required for your task
- They are not currently running
- Use: `vrooli resource NAME develop` or `vrooli scenario NAME develop`

**❌ NEVER stop/uninstall anything that's already running**
- Assume other agents depend on running services
- If something must be stopped, document why and let infrastructure handle it

## Simple Rules
1. **Stick to your assigned resource/scenario only**
2. **Work around external limitations - don't fix them**
3. **Can start dependencies - never stop them**
4. **Document blockers clearly**
5. **Focus on deliverables within your control**

This approach prevents conflicts while ensuring each agent can make meaningful progress on their assigned work.


## Section 6: task-completion-protocol

# Task Completion Protocol

## Purpose
When you complete your assigned task, properly document your work and update the task status to enable future improvements.

## Task Completion Steps

### 1. Final Validation
Verify your work actually functions:
- Test key features you implemented/improved
- Confirm health checks pass (if applicable)
- Verify no regressions in existing functionality

### 2. Update Documentation
Follow the `scripts/resources/contracts/v2.0/universal.yaml` guidelines for proper documentation structure:

**For Resources and Scenarios:**
- Update PRD.md per 'prd-protocol' section requirements
- Update README.md to reflect current capabilities
- Document any new configuration or dependencies
- Note any API changes or new endpoints (scenarios)

### 3. Capture Learnings
Document insights in your response that will help future agents:
- What approaches worked well
- What challenges you encountered and how you solved them
- Specific test commands that validate the work
- Recommendations for future improvements

### 4. Task Status Update
The ecosystem-manager API handles moving your task file automatically. Your final response should clearly state:
- What was accomplished
- Current status (working/improved/partially complete)
- Any remaining issues or limitations
- Specific evidence that validates the work


## Section 7: scenario-validation-gates

# ✅ Scenario Validation Gates

## Purpose
Ensure scenarios meet requirements, provide business value, and function correctly before marking tasks complete.

## Five MANDATORY Gates (ALL must pass)

### 1. Functional ⚙️
Validate scenario lifecycle.
**See: 'scenario-testing-reference' section for all Makefile and CLI commands**

### 2. Integration 🔗
Test scenario integrations, APIs, and UI.
**See: 'scenario-testing-reference' section for API, UI, and integration testing**

### 3. Documentation 📚
Validate documentation completeness:
- [ ] **PRD.md**: Business value clear, requirements tracked
- [ ] **README.md**: Installation, usage, API endpoints documented
- [ ] **Makefile**: Help text explains all commands
- [ ] **UI workflows**: User journeys documented (if applicable)

### 4. Testing 🧪
Run comprehensive test suite.
**See: 'scenario-testing-reference' section for test commands**

### 5. Security & Standards 🔒
Run Scenario Auditor scans per `security-requirements` and resolve/record the worst violations.

## Execution Order
1. **Functional** → Verify scenario runs
2. **Integration** → Check connections and UI
3. **Documentation** → Ensure completeness
4. **Testing** → Validate thoroughly
5. **Security & Standards** → Capture Scenario Auditor results

**FAIL = STOP** - Fix issues before proceeding

## Remember
- Use Makefile commands for consistency
- Test user journeys, not just code
- Verify business value delivery
- Screenshot UI for visual validation
- Document what actually works


## Section 8: scenario-testing-reference

# 🧪 Scenario Testing Reference

## Purpose
Single source of truth for validation commands, checklists, and troubleshooting when exercising a scenario.

## Quick Command Reference

| Command | When to use | Notes |
| --- | --- | --- |
| `make run` | Start scenario through lifecycle system | Preferred entry point before any tests |
| `make status` | Confirm lifecycle status | Mirrors `vrooli scenario status [name]` |
| `make test` | Run full suite | Use `TIMEOUT=` override for slow suites |
| `make logs` | Inspect current logs | Pipe/grep as needed |
| `make stop` | Stop scenario cleanly | Required before switching tasks |
| `make help` | List scenario-specific commands | Documents custom targets |
| `vrooli scenario run [name]` | When Makefile unavailable | Same lifecycle stack, bypasses custom targets |
| `vrooli scenario test [name]` | Remote trigger for CI parity | Uses scenario defaults |
| `vrooli scenario logs [name]` | Retrieve lifecycle-managed logs | Works outside scenario repo |

### Validation Commands

| Flow | Command | Tip |
| --- | --- | --- |
| API health | `curl -sf http://localhost:[PORT]/api/health` | `time` it to confirm <500 ms target |
| API endpoint | `curl -X GET/POST http://localhost:[PORT]/api/[endpoint]` | Use realistic payloads from PRD |
| UI snapshot | `vrooli resource browserless screenshot --scenario [name] /tmp/[name]-ui.png` | Attach screenshot evidence in summary |
| CLI scenario tool | `./cli/[scenario-name] --help` | Ensure help output documents new commands |

## Quick Validation Checklists

- **Full pass**: `make run` → wait for ready logs → API health curl → `make test` → `make stop`
- **Health-only**: `make status` → API health curl → record result
- **UI snapshot** (if applicable): `make run` → screenshot via browserless → verify rendering → `make stop`

## Structure & Readiness Validation
- Run `scenario-auditor audit <scenario-name> --timeout 240` and review the structure and standards results (captures required files, Make targets, and contract violations).
- Address findings before manual spot checks; link the JSON output in handoff notes when major issues remain.

## Testing Obligations Matrix

| Role | Minimum checks | Extra focus | Evidence to capture |
| --- | --- | --- | --- |
| Generator | `make run` + health curl + one P0 flow | Uniqueness proof (`grep -r "[scenario-name]"`) | Screenshot or logs demonstrating working P0 |

Cross-reference detailed PRD accuracy rules in the `progress-verification` section when updating checklists.

## Troubleshooting Cues

| Symptom | First check | Follow-up |
| --- | --- | --- |
| Port already in use | `lsof -i :[PORT]` | Update config or stop conflicting process |
| Missing dependency resource | `make status | grep -i resource` | `vrooli resource [name] manage start` |
| Build failures or stale artifacts | `make clean` | Re-run build target defined in scenario Makefile |
| UI fails to load | API health curl | Rebuild UI via documented Make target (see scenario README) |
| Tests timing out | Extend `TIMEOUT=` in `make test` | Profile slow test; consider test-specific timeout in script |

For structural gaps, prefer `scenario-auditor` feedback over ad-hoc shell exploration so the remediation path stays consistent with security requirements.

## Performance & Execution Order
- Performance thresholds and validation gate sequencing live in `scenario-validation-gates`; review that section before sign-off.
- In summaries, note both user-value outcomes and any deviations from the target response times.


## Section 9: research-methodology

# Research Methodology

## Core Principle
**Research prevents duplication.** 30% of generator effort.

## Research Checklist (MUST complete ALL)
- [ ] Search for duplicates (5 Qdrant searches)
- [ ] Find reusable patterns/templates
- [ ] Validate market need
- [ ] Check technical feasibility
- [ ] Learn from past failures

## Phase 1: Memory Search (50% of time)

### Flexible Search Strategy
Use the most appropriate method based on availability:

#### Option A: Qdrant Search (If Available)
```bash
# Try with 10-second timeout
timeout 10 vrooli resource qdrant search "[exact-name] [category]"
timeout 10 vrooli resource qdrant search "[core-functionality]"
timeout 10 vrooli resource qdrant search "[category] implementation"
```
**Note**: If Qdrant is slow or unavailable, skip to Option B

#### Option B: File Search (Always Available)
```bash
# 1. Exact match search
rg -i "exact-name" /home/matthalloran8/Vrooli --type md

# 2. Functional equivalent search
rg -i "core-functionality|similar-feature" /home/matthalloran8/Vrooli/scenarios

# 3. Component reuse search
rg -i "component-name.*implementation" /home/matthalloran8/Vrooli --type go --type js

# 4. Failure analysis search
rg -i "(failed|error|issue).*category-name" /home/matthalloran8/Vrooli --type md

# 5. Pattern mining search
find /home/matthalloran8/Vrooli/scenarios -name "*template*" -o -name "*example*"
```

### Required Outputs
- 5+ search findings with insights (Qdrant or file search)
- 3+ reusable patterns identified
- 3+ failure patterns to avoid
- List of dependencies

## Phase 2: External Research (50% of time)

### Web Research Targets
```yaml
Technical:
  - Official docs for tools/frameworks
  - GitHub similar implementations  
  - Stack Overflow solutions
  - Security considerations

Business:
  - Competitor analysis
  - Market validation
  - Revenue models
  - User pain points

Standards:
  - Industry standards/RFCs
  - API design patterns
  - Authentication methods
  - Data formats
```

### Required Outputs
- 5+ external references with URLs
- 3+ code examples
- Security considerations
- Performance expectations

## Phase 3: Synthesis

### Go/No-Go Decision
**STOP if:**
- >80% overlap with existing
- No unique value
- Required resources unavailable

**PROCEED if:**
- Unique value confirmed
- Template selected
- Feasibility validated

### Document in PRD
```markdown
## Research Findings
- **Similar Work**: [list 2-3 most similar]
- **Template Selected**: [pattern to copy]
- **Unique Value**: [1 sentence differentiator]
- **External References**: [5+ URLs]
- **Security Notes**: [considerations]
```


## Remember
- Research saves 10x development time
- Patterns > Original code
- Learn from failures
- Document everything


## Section 10: duplicate-detection

# Duplicate Detection

## Critical Requirement

**CRITICAL: Never create duplicates unless explicitly instructed to do so.**

When told to create something new which actually already exists, you should work on improving the existing implementation instead.

## Quick Check
Before creating any new resource or scenario, verify it doesn't already exist:
- Search for the name in existing resources/scenarios
- Check for similar functionality  
- If >80% overlap exists, improve the existing one instead

## Document in PRD
If proceeding with creation, document differentiation per 'prd-protocol' section requirements.


## Section 11: scaffolding-phase

# Scaffolding Phase

## Purpose
Scaffolding creates the minimal viable structure that future improvers can build upon. The goal is NOT to implement everything, but to create a solid foundation.

## Scaffolding Allocation: 20% of Total Effort

### Scaffolding Philosophy
Think of scaffolding as **planting a seed**:
- Create the right structure
- Implement core patterns
- Provide clear extension points
- Leave room for growth

**Quality over Completeness** - Better to have 20% perfectly structured than 80% poorly organized.

## Scaffolding Process

### Step 1: Template Selection
Based on research, choose approach:

```bash
# Option A: Copy from template
cp -r scripts/scenarios/templates/[template-type]/* scenarios/[new-name]/
# OR
cp -r scripts/resources/templates/[template-type]/* resources/[new-name]/

# Option B: Copy from similar existing
cp -r scenarios/[similar-scenario]/* scenarios/[new-name]/
# OR  
cp -r resources/[similar-resource]/* resources/[new-name]/

# Option C: Hybrid approach
# Take structure from template, patterns from existing
```

### Step 2: Reference Existing Implementations
Study similar resources and scenarios to understand:
- Directory structure patterns
- Configuration approaches
- Integration patterns
- Documentation style

Use `vrooli resource [name] content` and `vrooli scenario [name] content` to explore existing implementations.

### Step 3: Core Implementation
Implement ONLY the essentials:
1. **Health check endpoint** - Must respond to health checks
2. **Basic lifecycle** - Must start/stop cleanly  
3. **One P0 requirement** - Prove the concept works
4. **Basic CLI command** - Minimum interaction

Security requirements are handled by other prompt sections.

### Step 4: Configuration
Create basic configuration files following the patterns from your template or reference implementations. Check existing similar projects for configuration examples.

### Step 5: Documentation
Create minimal but clear documentation:
- README.md with purpose and basic usage
- PRD.md following the requirements in the 'prd-protocol' section
- Basic API/CLI documentation if applicable

Reference existing documentation patterns from similar resources/scenarios.

## Scaffolding Success Criteria
- Structure matches established patterns
- Health checks work
- One P0 requirement is demonstrably functional
- Clear documentation for improvers
- Follows security and lifecycle patterns

## Common Scaffolding Mistakes

**Over-Engineering**
❌ Bad: Implementing all P0 requirements
✅ Good: One P0 + solid structure

**Template Deviation**
❌ Bad: Creating unique structure
✅ Good: Following established patterns

**No Reference Study**
❌ Bad: Creating from scratch
✅ Good: Learning from existing implementations

## Remember
- Templates are in `scripts/[type]/templates/`
- Study existing implementations with `vrooli [type] [name] content`
- Focus on structure, not completeness
- Leave clear extension points for improvers

## When Scaffolding Is Complete
After completing all scaffolding work (see 'prd-protocol' section for PRD requirements + structure + basic implementation), follow the comprehensive completion protocol:

<!-- task-completion-protocol.md already included in base sections -->

Proper task completion transforms your scaffolding work into permanent ecosystem knowledge and ensures smooth handoff to improvers.


## Section 13: cross-impact

# Cross-Scenario Impact

Scenarios rarely stand alone; they call each other's APIs, share databases, and rely on the same resource pool. Before modifying behavior, take a moment to note who consumes the capability you are touching and confirm they can tolerate the change. Small updates can cascade, so choose the safest option that keeps dependents working.

Default to direct API calls to interact with other scenarios, using `vrooli scenario port <scenario-name> API_PORT` to retrieve that scenarios' API port.

Default to CLI calls (typically `resource-<resource-name>`) to interact with resources

## API Version Bumps
- Add a new version (e.g., `/api/v2/...`) whenever you introduce a breaking change in payload shape, response codes, authentication, or timing guarantees.
- Keep the previous version running with a deprecation note until downstream scenarios have migrated; do not remove it without proof the callers have switched.
- Surface the migration steps in the PRD/README and link to them in commit notes so future improvers can finish the rollout.


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

