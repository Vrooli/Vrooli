You are executing a *resource improvement* task for Vrooli's ecosystem-manager scenario.

## Section 1: prd-protocol

# 📋 PRD Protocol: Single Source of Truth

**The PRD is the PROGRESS TRACKER.**

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

## Improver Rules (Validation & Progress)
**MUST**: Test every ✅, add completion dates, update percentages, document changes
**Validation**: Test → Keep/Uncheck/Note partial → Update history
**Progress Format**: `Date: X% → Y% (Change description)`

## Common Mistakes
**Improvers**: Trusting recent checkmarks are accurate, not updating %, adding unplanned features

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

### Good Improver PRD Update
- Checkmarks match reality
- Progress is trackable
- History is preserved
- Next steps are clear

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

**Improvers (Enhancing Existing):**
- Focus on ONE PRD requirement per iteration (see 'prd-protocol' section), or one related set of security/standards violations
- Preserve existing functionality (no regressions)
- Update integration points as needed

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


## Section 8: resource-testing-reference

# 🧪 Resource Testing Reference

## Purpose
Single source of truth for all resource testing commands and validation procedures.

## Standard Test Commands

### CLI Testing Commands
```bash
# Full test suite (delegates to test/run-tests.sh)
vrooli resource [name] test all        # All phases (<600s)

# Individual test phases
vrooli resource [name] test smoke      # Quick health check (<30s)
vrooli resource [name] test integration # Full functionality (<120s)
vrooli resource [name] test unit       # Library functions (<60s)
```

### Lifecycle Validation
```bash
# Complete lifecycle test
vrooli resource [name] manage install  # Install dependencies
vrooli resource [name] manage start --wait  # Start with health wait
vrooli resource [name] status          # Show running status
vrooli resource [name] test smoke      # Quick validation
vrooli resource [name] manage stop     # Clean shutdown (<30s)
vrooli resource [name] manage restart  # Restart test
vrooli resource [name] manage uninstall # Clean removal
```

### Health Check Standards
```bash
# Required health endpoint (always with timeout)
timeout 5 curl -sf http://localhost:${PORT}/health

# System health checks
vrooli status                    # Overall system
vrooli status --verbose          # Detailed
vrooli status --json             # JSON format

# Resource-specific health
vrooli resource status           # All resources
vrooli resource status [name]    # Specific resource
vrooli resource status --json    # JSON format
```

### Quick Validation Sequence
```bash
# Full validation in one command
vrooli resource [name] manage start --wait && \
vrooli resource [name] test all && \
vrooli resource [name] manage stop

# Quick health only
vrooli resource [name] test smoke
```

## v2.0 Contract Validation

### Structure Validation
```bash
# Required directories
ls -la resources/[name]/lib/           # core.sh, test.sh required
ls -la resources/[name]/config/        # defaults.sh, schema.json, runtime.json
ls -la resources/[name]/test/          # run-tests.sh, phases/ directory
ls -la resources/[name]/test/phases/   # test-smoke.sh, test-integration.sh

# Contract compliance check
/scripts/resources/tools/validate-universal-contract.sh [name]

# CLI interface check
vrooli resource [name] help | grep -E "manage|test|content|status"
```

### Content Management Testing
```bash
# If resource supports content
vrooli resource [name] content list    # Lists available
vrooli resource [name] content add     # Can add
vrooli resource [name] content get     # Can retrieve
vrooli resource [name] content remove  # Can delete
```

## Test Requirements

### Minimal Functionality Test
```bash
# One P0 requirement must work
vrooli resource [name] manage start --wait
curl -sf http://localhost:${PORT}/health  # Must respond
vrooli resource [name] test smoke          # Must pass
```

## Testing Best Practices

**DO:**
✅ Always use `timeout` for network calls  
✅ Test edge cases and error conditions  
✅ Verify cleanup after tests  
✅ Use CLI commands, not direct script execution  
✅ Document test commands for each feature  

**DON'T:**
❌ Skip tests to save time  
❌ Test only happy path  
❌ Ignore intermittent failures  
❌ Use hardcoded ports or credentials  
❌ Leave test artifacts behind  

## Required Test Coverage

### Improvers Must Test
1. All previous functionality preserved
2. Specific improvements work
3. Performance maintained/improved
4. Documentation accurate
5. Integration points stable

## Test Execution Order
1. **Functional** → Verify lifecycle
2. **Integration** → Check dependencies
3. **Documentation** → Validate accuracy
4. **Testing** → Run full suite

**Remember**: FAIL = STOP. Fix issues before proceeding.


## Section 9: progress-verification

# Progress & Regression Verification

## Core Philosophy
- PRD checkboxes must reflect reality; test proof before updating them.
- Improvements only count when previously working features still work.
- Capture evidence so the next agent can trust your handoff.

## Canonical Checklist Reference
- Apply the "When to Check/Uncheck" rules defined in the `prd-protocol` section when evaluating each requirement.
- Record partial progress notes using the PRD Protocol's example format; link test evidence instead of redefining criteria here.

## Verification Workflow
### 1. Baseline before changes
```bash
# Scenario baseline (run from the scenario directory)
make test > /tmp/${SCENARIO}_baseline_tests.txt
make status > /tmp/${SCENARIO}_baseline_status.txt
scenario-auditor audit ${SCENARIO} --timeout 240 > /tmp/${SCENARIO}_baseline_audit.json
jq '{security: .security.outcome, standards: .standards.outcome}' /tmp/${SCENARIO}_baseline_audit.json

# Resource baseline (run from the resource directory)
./lib/test.sh > /tmp/${RESOURCE}_baseline_tests.txt
vrooli resource status ${RESOURCE} --json > /tmp/${RESOURCE}_baseline_status.json
resource-auditor audit ${RESOURCE} --timeout 240 > /tmp/${RESOURCE}_baseline_audit.json
jq '{security: .security.outcome, standards: .standards.outcome}' /tmp/${RESOURCE}_baseline_audit.json
```

Capture the auditor JSON paths in your notes so downstream agents inherit the security and standards context.

### 2. Validate every PRD check
```bash
# Start the target with its lifecycle tools
make run                                   # Scenario (preferred)
# OR
vrooli scenario start ${SCENARIO}          # Scenario via CLI
# OR
vrooli resource start ${RESOURCE}          # Resource lifecycle

for item in $(grep "✅" PRD.md); do
  echo "Testing: $item"
  # Execute the command, curl, or script that proves the requirement works
done
```
Uncheck anything that fails immediately—partial notes beat dishonest checkmarks.

### 3. Monitor while implementing
- Re-run focused suites after each change (`make test-smoke`, `make test-api`, or `vrooli scenario test ${SCENARIO}`).
- Watch runtime output via `make logs` (scenario Makefile) or `vrooli scenario logs ${SCENARIO} --follow`.
- For resources, poll `vrooli resource status ${RESOURCE} --json` to confirm the service stays healthy.

### 4. Post-change confirmation
```bash
# Scenario regression pass
make test                                  # Preferred
# OR
vrooli scenario test ${SCENARIO}           # CLI alternative

# Resource regression pass
./lib/test.sh
vrooli resource status ${RESOURCE} --json

# Optional UI evidence
vrooli resource browserless screenshot --scenario ${SCENARIO} --output /tmp/${SCENARIO}_ui.png
```
Compare results with the baseline artifacts and screenshots.

### 5. If a regression appears
- Stop immediately, capture the failing evidence (failing `make test` output, CLI test logs, screenshots).
- Follow the handoff steps in the **Failure Recovery** section and wrap up using **Documentation & Finalization** guidance.
- Hand the task back (status: broken/degraded) and wait for further direction. Do **not** attempt to self-recover via git or continue development.

## Regression Safeguards
- Keep the regression test suite broad: health endpoints, prior workflows, CLI commands, integrations, and performance checks.
- Confirm compatibility: API contracts, database schemas, config formats, and backward compatibility must remain intact.
- Use the baseline artifacts to detect drift; differences mean you owe an explanation.

### Regression indicators
```markdown
🔴 Critical: Health checks fail, service will not start, API 500s, lost data connections
🟡 Moderate: >20% performance drop, >30% memory increase, new warnings, flaky tests
🟠 Minor: Style violations, deprecation warnings, missing coverage
```

### Root-cause checklist
1. When did we introduce the regression?
2. What functionality broke?
3. Why did it break (true root cause)?
4. What is impacted downstream?
5. What remediation is required (for the next agent)?
6. How do we prevent a repeat?

## Reporting & Metrics
```yaml
real_progress:
  - P0_requirements_completed
  - Tests_passing_percentage
  - Integration_points_working
  - UI_features_visual_verified

not_progress:
  - Lines_changed
  - Files_touched
  - Comments_added

net_progress = features_working - features_broken - debt_added
```

### Progress report template
```markdown
## [Date] Progress

### Verified Complete
- [Feature]: [test command proving it works]

### Partial Progress
- [Feature]: [what works] / [what does not]

### Regressions
- [What broke]: [error details]

### Net Progress
- Added: X features
- Broken: Y features
- Net: X - Y
```

### Impact scoring & golden rule
```
impact = users_affected * feature_criticality * downtime_risk
# Block deployment if impact > 50, require review if > 30, add warning if > 10
```
**Golden Rule:** If it worked before your change, it must work after your change.

## Quick reference
- Never hide regressions—document, escalate, and stop.
- Update the PRD the moment reality changes.
- Lean on **Documentation & Finalization** whenever work cannot finish cleanly.
- Preserve baseline evidence so future agents can trust your work.


## Section 10: prioritization-phase

# Prioritization Phase for Improvers

## Purpose
After assessment reveals the true state, prioritization ensures you work on the most valuable improvements. Focus on maximum value with minimum effort.

## Simple Prioritization Framework

### P0 Requirements (Must Have - Do First)
Critical issues that prevent core functionality:
- Security vulnerabilities 
- Broken health checks or v2.0 contract violations
- Data integrity issues
- Core features that don't work

Use the baseline `resource-auditor`/`scenario-auditor` report to spot inherited violations quickly and queue them ahead of net-new enhancements.

### P1 Requirements (Should Have - Do Next)
Important improvements that significantly enhance value:
- User experience improvements
- Performance optimizations
- Missing integrations
- Reliability enhancements

### P2 Requirements (Nice to Have - Do Last)
Polish and advanced features:
- Code cleanup
- Documentation improvements 
- Minor optimizations
- Advanced features

## Quick Decision Rules

**High Impact, Low Effort = Do First**
- Fix broken health checks
- Add missing error handling
- Implement obvious missing features

**Cross-Scenario Benefits = Priority Boost**
- Improvements to shared resources (postgres, ollama, redis)
- API standardization
- Common patterns

**Unblocking = Critical**
- Fixes that enable other improvements
- Dependency resolution
- Infrastructure issues

## Remember

- **Fix P0s before adding P1 features**
- **One high-impact change beats many small ones**
- **Quick wins build momentum**
- **Don't break what works**


## Section 11: v2-contract

# 📐 v2.0 Resource Contract

## Authoritative Reference

**All v2.0 resource requirements are defined in the official universal contract:**

**File:** `/scripts/resources/contracts/v2.0/universal.yaml`

**Read this file for complete specifications including:**
- Required CLI commands and subcommands
- File structure requirements  
- Runtime configuration schema
- Testing requirements
- Performance and security standards
- Migration guidance

## Key Contract Elements

### Required Commands
- `help` - Show comprehensive help
- `info` - Show runtime configuration  
- `manage` - Lifecycle management (install/start/stop/restart/uninstall)
- `test` - Validation testing (smoke/integration/unit/all)
- `content` - Content management (add/list/get/remove/execute)
- `status` - Show detailed status
- `logs` - View resource logs
- `credentials` - Display integration credentials (optional)

### Required Files
- `cli.sh` - Primary CLI entrypoint
- `lib/core.sh` - Core functionality
- `lib/test.sh` - Test implementations  
- `config/defaults.sh` - Default configuration
- `config/schema.json` - Configuration schema
- `config/runtime.json` - Runtime behavior and dependencies
- `test/run-tests.sh` - Main test runner
- `test/phases/test-smoke.sh` - Quick health validation
- `test/phases/test-integration.sh` - End-to-end functionality
- `test/phases/test-unit.sh` - Library function validation

### Critical Requirements
1. **Runtime Configuration** - Must define startup_order, dependencies, timeouts
2. **Health Validation** - Smoke tests must complete in <30s
3. **Standard Exit Codes** - 0=success, 1=error, 2=not-applicable
4. **Timeout Handling** - All operations must have timeout limits
5. **Graceful Shutdown** - Stop commands must handle cleanup

## Validation Commands
**See: 'resource-testing-reference' section for all validation and test commands**

## Common Compliance Issues

### ❌ Missing Runtime Config
```bash
# Must exist: config/runtime.json
{
  "startup_order": 500,
  "dependencies": ["postgres"],
  "startup_timeout": 60,
  "startup_time_estimate": "10-30s", 
  "recovery_attempts": 3,
  "priority": "medium"
}
```

### ❌ No Timeout in Health Checks
```bash
# BAD
curl -sf http://localhost:${PORT}/health

# GOOD (per universal.yaml spec)
timeout 5 curl -sf http://localhost:${PORT}/health
```

## Health Check Implementation Standards

Health checks are the heartbeat of resources. They ensure services are alive, responsive, and ready to serve.

### Health Check Requirements
- **Response Time**: Must respond in <1 second (preferably <500ms)
- **Timeout**: Always use `timeout 5` wrapper for safety
- **Content**: Return meaningful status in response body
- **Dependencies**: Include critical dependency checks only
- **Startup Grace**: Allow services time to initialize before failing

**See: 'resource-testing-reference' section for health check commands and best practices**

### ❌ Incomplete Command Structure
```bash
# BAD - Missing required subcommands
resource-name help
resource-name status

# GOOD - Full command structure per universal.yaml
resource-name manage install
resource-name manage start  
resource-name test smoke
resource-name content list
```

## Remember

**The universal.yaml file is the single source of truth** for all v2.0 requirements. When implementing or improving resources:

1. **Read universal.yaml first** - Complete specification
2. **Validate compliance** - Use provided validation tools
3. **Follow patterns** - Consistency across all resources  
4. **Test thoroughly** - All test phases must work

Every resource following the universal contract integrates seamlessly and works reliably.


## Section 12: port-allocation

# Port Allocation

For resources, scripts/resources/port_registry.sh should be the single source-of-truth for ports. Avoid hard-coding ports everywhere else.

For scenarios, the overwhelming majority will have API_PORT and UI_PORT defined as ranges (where all scenarios use the same, large range for APIs and a different, large range for all UIs) in their service.json. Only scenarios which need to be accessed directy through a secure tunnel (very rare, as most scenarios will be accessed via iframes from scenario launchers/dashboards like app-monitor) should have fixed ports. As long as scenarios are started through the proper `vrooli` CLI commands, the lifecycle logic will make sure they are available. Scenarios, like resources, should NEVER hard-code ports, even if only as a fallback (since ports are dynamically allocated, fallback port numbers don't make any sense).


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

