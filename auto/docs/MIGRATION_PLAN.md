# 📋 Migration Plan: From Auto/ to Distributed Scenarios

## Executive Summary

This document outlines the complete migration strategy for transitioning from the `auto/` continuous improvement loops to distributed, specialized scenarios. The migration preserves all embedded intelligence while enabling parallel, specialized operations through five distinct scenarios.

## Current State Analysis

### Auto/ System Intelligence
- **scenario-improvement-loop.md**: 312 lines of scenario improvement knowledge
- **resource-improvement-loop.md**: 648 lines of resource improvement knowledge
- **Embedded Intelligence**: Years of operational learning, patterns, and best practices
- **Success Metrics**: 439+ successful iterations with 100% success rate

### Replacement Scenario Assessment

| Scenario | Purpose | Current Status | Completeness |
|----------|---------|----------------|--------------|
| **scenario-generator-v1** | Create NEW scenarios | ✅ Exists | 60% - Missing PRD compliance, validation gates |
| **scenario-improver** | Improve EXISTING scenarios | ❌ Doesn't exist | 0% - Needs creation |
| **resource-generator** | Create NEW resources | ❌ Doesn't exist | 0% - Needs creation |
| **resource-improver** | Improve EXISTING resources | ❌ Doesn't exist | 0% - Needs creation |
| **resource-experimenter** | Add resources to scenarios | ✅ Exists | 40% - Limited scope, missing standards |

## Critical Gaps Identified

### 1. Missing Core Scenarios
- **No scenario-improver**: Cannot improve existing scenarios systematically
- **No resource-generator**: Cannot create brand new resources
- **No resource-improver**: Cannot improve existing resources systematically

### 2. Incomplete Knowledge Transfer
- **PRD Compliance**: Not enforced in any successor scenario
- **Validation Gates**: Missing from all scenarios
- **Qdrant Memory System**: Only in auto/, needs to be everywhere
- **v2.0 Contract**: Not mentioned in any resource-related scenario
- **Cross-Scenario Impact**: Not considered in current scenarios

### 3. Resource-Experimenter Misconception
- **NOT for creating new resources** - it adds resources to existing scenarios
- Still need a dedicated **resource-generator** for scaffolding new resources

## 🎯 Target Architecture

### Five Specialized Scenarios

```
1. scenario-generator-v1 (Enhancement Required)
   └── Purpose: Create NEW scenarios from scratch
   └── Knowledge Gap: PRD compliance, validation gates, memory system
   └── Queue: backlog/ folder (existing, works well)

2. scenario-improver (Create New)
   └── Purpose: Improve EXISTING scenarios
   └── Knowledge Source: auto/scenario-improvement-loop.md
   └── Queue: improvements/ folder system

3. resource-generator (Create New)
   └── Purpose: Create brand NEW resources
   └── Knowledge Source: New resource sections from auto/resource-improvement
   └── Queue: new-resources/ folder system

4. resource-improver (Create New)
   └── Purpose: Improve EXISTING resources
   └── Knowledge Source: auto/resource-improvement-loop.md
   └── Queue: improvements/ folder system

5. resource-experimenter (Enhancement Required)
   └── Purpose: Add resources to existing scenarios
   └── Knowledge Gap: v2.0 contract, health monitoring standards
   └── Queue: experiments/ folder (needs creation)
```

## 🧠 Universal Knowledge Distribution

### Knowledge That MUST Be in ALL Scenarios

#### 1. Qdrant Memory System
```markdown
## 🧠 Vrooli's Memory System

The Qdrant embeddings system is Vrooli's **long-term memory** - it remembers every solution, 
pattern, failure, and capability across all apps and resources.

### Before Starting Any Work:
**Search Memory**: Use Vrooli's long-term memory to learn from past work:
- `vrooli resource-qdrant search-all "topic keywords"`
- `vrooli resource-qdrant search "specific pattern" resources`
- `vrooli resource-qdrant search "error message" scenarios`

### After Completing Work:
**Feed the Memory**: Update docs and refresh embeddings so future agents learn:
1. Document your changes clearly in README.md or relevant docs
2. Run `vrooli resource qdrant embeddings refresh`
3. Your insights now help all future agents avoid repeating work
```

#### 2. PRD-Driven Development
```markdown
## 📄 PRD-Driven Development

EVERY scenario/resource MUST have a PRD.md that serves as the single source of truth:
- Template: `scripts/[scenarios|resources]/templates/*/PRD.md`
- Track progress by checking off completed requirements
- Priority: P0 (must have) → P1 (should have) → P2 (nice to have)
- PRD defines the permanent capability being added to Vrooli
```

#### 3. Validation Gates
```markdown
## ✅ Validation Gates

All changes must pass these gates before completion:
1. **Functional**: Component starts and responds to health checks
2. **Integration**: Works with declared dependencies
3. **Documentation**: README and PRD updated
4. **Testing**: Integration tests pass (if present)
5. **Memory**: Changes documented and embeddings refreshed

If any gate fails twice, stop and document the blocker.
```

## 📁 Standardized Queue Architecture

### Folder Structure
Each scenario maintains its own queue:
```
scenarios/[scenario-name]/queue/
├── pending/           # Work items waiting
│   ├── 001-critical-fix.yaml
│   ├── 100-medium-improvement.yaml
│   └── 200-low-priority-enhancement.yaml
├── in-progress/       # Currently being worked on (max 1)
├── completed/         # Successfully finished
├── failed/           # Failed attempts with logs
├── templates/        # Templates for different task types
│   ├── improvement.yaml
│   ├── new-feature.yaml
│   └── fix.yaml
└── README.md         # Queue management documentation
```

### Queue Item Schema
```yaml
# Priority and requester in header comment
id: unique-identifier
title: "Clear, actionable title"
description: |
  Detailed description of what needs to be done
  
type: improvement|fix|new-feature|refactor
target: resource-name|scenario-name
priority: critical|high|medium|low

# Selection intelligence from auto/
priority_estimates:
  impact: 8              # 1-10 scale
  urgency: high         # critical|high|medium|low
  success_prob: 0.8     # 0-1 probability
  resource_cost: moderate # minimal|moderate|heavy

requirements:
  - "Specific requirement 1"
  - "Specific requirement 2"

validation_criteria:
  - "How to verify success"
  - "What tests should pass"

# Context from Qdrant memory
memory_context: |
  Related past work:
  - Previous similar fix in resource-X
  - Pattern documented in docs/patterns/Y

metadata:
  created_by: human|ai|system-monitor|auto-migration
  created_at: "2025-01-15T10:00:00Z"
  cooldown_until: "2025-01-16T10:00:00Z"  # Collision avoidance
  attempt_count: 0
  last_attempt: null
```

## 📝 Prompt Organization Strategy

### Scenario Prompt Structure
```
scenarios/[scenario-name]/prompts/
├── main-prompt.md           # Core task-specific logic
├── includes/                # Reusable components
│   ├── memory-system.md     # Qdrant usage
│   ├── validation-gates.md  # Testing requirements
│   └── prd-compliance.md    # PRD tracking
└── queue-handlers/          # Queue-specific prompts
    ├── process-queue.md     # How to select from queue
    └── update-queue.md      # How to update queue items
```

### Shared Knowledge Library
```
scripts/shared/prompts/       # Shared across ALL scenarios
├── vrooli-context.md        # System understanding
├── memory-system.md         # Qdrant universal usage
├── prd-methodology.md       # PRD-driven development
├── validation-gates.md      # Universal testing approach
├── resource-standards.md    # v2.0 contract, health checks
├── scenario-standards.md    # Cross-scenario impact
└── selection-logic.md       # Priority scoring, cooldowns
```

## 🔄 Knowledge Extraction Mapping

### From auto/scenario-improvement-loop.md

| Line Range | Knowledge Area | Target Scenarios | Priority |
|------------|----------------|------------------|----------|
| 29-34 | PRD compliance methodology | ALL scenarios | Critical |
| 89-100 | Validation gate system | ALL scenarios | Critical |
| 23-24 | Cross-scenario impact | scenario-generator, scenario-improver | High |
| 42-43 | Shared workflow adoption | scenario-improver, resource-experimenter | High |
| 53-72 | Selection algorithms | ALL scenarios (queue population) | Medium |
| 61-65 | Iteration priority rubric | scenario-improver | Medium |

### From auto/resource-improvement-loop.md

| Line Range | Knowledge Area | Target Scenarios | Priority |
|------------|----------------|------------------|----------|
| 5,11,27-28 | Qdrant memory system | ALL scenarios | Critical |
| 31-41 | v2.0 contract compliance | resource-generator, resource-improver | Critical |
| 88,644-647 | Content management patterns | resource-improver, resource-generator | High |
| 84-86 | Health monitoring patterns | resource-improver, resource-generator | High |
| 195-630 | New resource catalog | resource-generator | Medium |
| 66-72 | Selection priority logic | resource-improver queue | Medium |

## 🚀 Implementation Phases

### Phase 1: Create Missing Scenarios (Week 1)

#### 1. scenario-improver
```bash
# Bootstrap from app-debugger
cp -r scenarios/app-debugger scenarios/scenario-improver

# Key modifications:
# - Generalize from "debugging" to "improving"
# - Add queue/ folder structure
# - Import improvement logic from auto/scenario-improvement-loop.md
# - Add PRD compliance checking
```

#### 2. resource-generator
```bash
# Create from scratch
mkdir -p scenarios/resource-generator/{api,cli,prompts,queue,ui}

# Core features:
# - Resource scaffolding templates
# - Port allocation logic
# - Docker configuration
# - CLI setup automation
# - Extract resource catalog from auto/
```

#### 3. resource-improver
```bash
# Bootstrap from resource-experimenter
cp -r scenarios/resource-experimenter scenarios/resource-improver

# Major modifications:
# - Change focus from "adding to scenarios" to "improving resources"
# - Import all improvement logic from auto/resource-improvement-loop.md
# - Add v2.0 contract compliance
# - Add health monitoring patterns
```

### Phase 2: Enhance Existing Scenarios (Week 1-2)

#### scenario-generator-v1 Enhancements
- Add PRD compliance checking to all prompts
- Include validation gate system
- Add Qdrant memory system usage
- Include cross-scenario impact assessment
- Add shared workflow preference logic

#### resource-experimenter Enhancements
- Add v2.0 contract knowledge
- Include health monitoring requirements
- Add content management patterns
- Include CLI interface standards
- Add memory system usage

### Phase 3: Knowledge Distribution (Week 2)

#### 1. Create Shared Prompt Library
```bash
mkdir -p scripts/shared/prompts

# Extract and create:
# - memory-system.md (from auto/resource-improvement)
# - prd-methodology.md (from auto/scenario-improvement)
# - validation-gates.md (from both auto/ prompts)
# - vrooli-context.md (from docs/context.md)
# - resource-standards.md (v2.0 contract)
# - scenario-standards.md (cross-scenario patterns)
```

#### 2. Update All Scenario Prompts
Add to the top of each main prompt:
```markdown
{{INCLUDE: scripts/shared/prompts/memory-system.md}}
{{INCLUDE: scripts/shared/prompts/prd-methodology.md}}
{{INCLUDE: scripts/shared/prompts/validation-gates.md}}
```

#### 3. Migrate Selection Algorithms
```bash
# Copy and adapt selection tools
cp -r auto/tools/selection scripts/shared/
# Modify for queue-based operation instead of direct selection
```

### Phase 4: Parallel Testing (Week 3)

#### Testing Strategy
1. Run auto/ loops alongside new scenarios
2. Compare:
   - Success rates
   - Quality of improvements
   - Knowledge preservation
   - Processing speed
3. Ensure no regression in capabilities
4. Validate queue processing works correctly

#### Success Criteria
- [ ] All auto/ knowledge preserved in new scenarios
- [ ] Queue processing rate ≥ auto/ iteration rate  
- [ ] Parallel execution improves throughput 2-3x
- [ ] No regression in improvement quality
- [ ] Manual queue intervention works smoothly

### Phase 5: Migration Execution (Week 4)

#### Migration Steps
1. **Freeze auto/ loops** - Stop new iterations
2. **Export current state** - Save summaries and recent work
3. **Populate initial queues** - Convert pending work to queue items
4. **Start new scenarios** - Begin parallel operation
5. **Monitor for 48 hours** - Ensure stable operation
6. **Deprecate auto/** - Mark as legacy, preserve for reference

## ⚠️ Critical Preservation Requirements

### Must Not Lose
1. **PRD-driven development methodology** - Core to Vrooli's quality
2. **Validation gate system** - Ensures reliability
3. **Cross-scenario impact assessment** - Prevents breaking changes
4. **Resource health checking patterns** - Maintains stability
5. **v2.0 contract compliance logic** - Ensures consistency
6. **Qdrant memory integration** - Preserves institutional knowledge
7. **Selection algorithms with cooldown** - Prevents collision
8. **Priority scoring formulas** - Optimizes work selection

### Knowledge at Risk
- Selection rubrics and priority logic (deeply embedded in prompts)
- Workaround patterns for known issues (scattered throughout)
- Iteration best practices (implicit in prompt structure)
- Error recovery patterns (embedded in validation logic)

## 📊 Success Metrics

### Quantitative Metrics
- Queue processing rate ≥ 50 improvements/day
- Parallel execution across 3+ scenarios simultaneously
- Success rate maintained at >80%
- Zero knowledge regression from auto/
- 2-3x improvement in throughput

### Qualitative Metrics
- Easier manual intervention via queue files
- Better specialization of improvements
- Clearer separation of concerns
- Improved debugging and monitoring
- Enhanced collaboration potential

## 🚨 Risk Mitigation

### Risk 1: Knowledge Loss
**Mitigation**: 
- Document all extracted knowledge in shared prompts
- Create comprehensive mapping of auto/ content
- Test each scenario against known good improvements

### Risk 2: Queue Coordination Issues
**Mitigation**:
- Standardize queue format across all scenarios
- Implement cooldown logic in queue handlers
- Use file locks to prevent concurrent access

### Risk 3: Performance Regression
**Mitigation**:
- Run parallel testing for 1 week minimum
- Monitor all success metrics continuously
- Keep auto/ available as fallback

## 🎬 Conclusion

This migration represents an evolution from monolithic loops to specialized, parallel scenarios. The key is preserving ALL embedded intelligence while gaining:
- Parallel execution capabilities
- Better specialization
- Easier manual intervention
- Clearer separation of concerns

The migration is not just moving code - it's preserving years of operational learning and making it more accessible and maintainable for the future.

## 📅 Timeline

- **Week 1**: Create missing scenarios
- **Week 2**: Enhance existing scenarios, distribute knowledge
- **Week 3**: Parallel testing and validation
- **Week 4**: Execute migration, deprecate auto/

Total estimated time: 4 weeks to complete migration with full validation.

---

*Last Updated: 2025-01-03*
*Status: PHASE 4 SAFE TESTING COMPLETE - Ready for parallel execution*

## 📊 Implementation Status

### ✅ Phase 1: Create Missing Scenarios (COMPLETE)
- ✅ Created scenario-improver from app-debugger template
- ✅ Created resource-generator from scratch  
- ✅ Created resource-improver from resource-experimenter template
- All scenarios have queue systems, prompts, and documentation

### ✅ Phase 2: Knowledge Enhancement (COMPLETE)
- ✅ Created shared prompts library at `/scripts/shared/prompts/`:
  - memory-system.md - Qdrant memory usage
  - prd-methodology.md - PRD-driven development
  - validation-gates.md - 5-gate validation system
  - cross-scenario-impact.md - Impact analysis
  - v2-resource-standards.md - v2.0 contract requirements
- ✅ Enhanced scenario-generator-v1 with new prompt including all shared knowledge
- ✅ Enhanced resource-experimenter with v2.0 compliance requirements
- ✅ Created documentation for prompt library usage

### ✅ Phase 3: Knowledge Distribution (COMPLETE)
- ✅ Updated all scenario prompts to include shared knowledge
- ✅ Created queue-based selection system at `/scripts/shared/selection/`:
  - queue-select.sh - Priority-based item selection
  - queue-manage.sh - Complete queue management utilities
  - README.md - Comprehensive documentation
- ✅ Migrated selection algorithms from auto/ to queue-based operation
- ✅ All scenarios now use shared prompt library includes

### ✅ Phase 4: Safe Testing and Validation (COMPLETE)
- ✅ Validated all scenario service.json configurations are valid
- ✅ Tested queue directory structures - all properly initialized
- ✅ Tested queue template files exist in all scenarios
- ✅ **Live tested queue selection algorithm** - successfully selected test item with priority score 11.92
- ✅ **Live tested queue management system** - successfully completed workflow from pending → in-progress → completed
- ✅ Validated all prompt {{INCLUDE}} directives reference existing files
- ✅ Confirmed all required dependencies available (bc, yq, jq, date)
- ✅ Verified shared prompt library properly structured with memory-system, prd-methodology, validation-gates, cross-scenario-impact
- ✅ All scenarios ready for safe parallel execution (no agent triggers)

### ⏳ Phase 5: Migration Execution (PENDING)
- [ ] Freeze auto/ loops
- [ ] Export current state
- [ ] Populate initial queues
- [ ] Start new scenarios
- [ ] Monitor for 48 hours
- [ ] Deprecate auto/