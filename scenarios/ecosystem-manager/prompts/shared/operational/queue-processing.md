# Queue Processing

## Purpose
Queue processing ensures organized, collision-free work distribution across multiple agents. The queue system provides visibility, prevents duplicate work, and enables manual intervention.

## Queue System Architecture

### Directory Structure
```
[scenario/resource]/queue/
├── pending/              # Work items waiting to be processed
│   ├── 001-critical-*.yaml    # Priority 001-099: Critical
│   ├── 100-high-*.yaml        # Priority 100-199: High
│   ├── 200-medium-*.yaml      # Priority 200-299: Medium
│   └── 300-low-*.yaml         # Priority 300-399: Low
├── in-progress/          # Currently being worked on (max 1 per agent)
│   └── [item].yaml      # Moved here with agent ID and timestamp
├── completed/            # Successfully completed work
│   └── [item]-[timestamp].yaml
├── failed/               # Failed attempts with logs
│   └── [item]-[timestamp].yaml
└── templates/            # Templates for different work types
    ├── improvement.yaml
    ├── bug-fix.yaml
    └── new-feature.yaml
```

### Queue Item Schema
```yaml
# Metadata in header comment
# Priority: 100 (high)
# Created: 2025-01-07T10:00:00Z
# Created By: human|ai-agent|system-monitor

id: unique-identifier-timestamp
title: "Clear, actionable title"
description: |
  Detailed description of what needs to be done
  Can be multiple lines
  
type: improvement|bug-fix|new-feature|refactor|optimization
target: [scenario-name|resource-name]
priority: critical|high|medium|low

# Intelligent prioritization
priority_estimates:
  impact: 8              # 1-10: Business/technical value
  urgency: high          # critical|high|medium|low
  success_prob: 0.85     # 0-1: Likelihood of success
  resource_cost: moderate # minimal|moderate|heavy
  
requirements:
  - "Specific requirement 1"
  - "Specific requirement 2"
  - "Must maintain backward compatibility"
  
validation_criteria:
  - "How to verify success"
  - "What tests should pass"
  - "Expected output or behavior"
  
dependencies:
  - "Must complete X first"
  - "Requires resource Y running"
  
# Context from memory system
memory_context: |
  Related past work:
  - Previous similar fix in resource-X
  - Pattern documented in docs/patterns/Y
  Search queries to run:
  - vrooli resource-qdrant search "relevant pattern"
  
metadata:
  created_by: human|ai|system
  created_at: "2025-01-07T10:00:00Z"
  estimated_hours: 2.5
  actual_hours: null  # Filled on completion
  cooldown_until: "2025-01-07T14:00:00Z"  # Prevents collision
  attempt_count: 0
  last_attempt: null
  assigned_to: null    # Agent ID when in-progress
  completed_at: null   # Timestamp when done
  
# Filled by processing agent
result:
  status: null  # success|failed|partial
  summary: null
  prd_progress: null  # Requirements completed
  issues_encountered: null
  follow_up_items: null
```

## Queue Selection Process

### Selection Algorithm
```python
def select_queue_item():
    # 1. Check cooldowns
    available_items = filter_cooldown_expired(pending_items)
    
    # 2. Calculate priority scores
    for item in available_items:
        item.score = calculate_priority_score(item)
    
    # 3. Sort by score
    sorted_items = sort_by_score_desc(available_items)
    
    # 4. Check dependencies
    for item in sorted_items:
        if dependencies_met(item):
            return item
    
    return None  # No suitable items

def calculate_priority_score(item):
    # Weighted scoring
    impact_weight = 0.4
    urgency_weight = 0.3
    success_weight = 0.2
    cost_weight = 0.1
    
    # Normalize urgency to number
    urgency_score = {
        'critical': 10,
        'high': 7,
        'medium': 5,
        'low': 3
    }[item.urgency]
    
    # Inverse cost (lower cost = higher score)
    cost_score = {
        'minimal': 10,
        'moderate': 5,
        'heavy': 2
    }[item.resource_cost]
    
    score = (
        item.impact * impact_weight +
        urgency_score * urgency_weight +
        item.success_prob * 10 * success_weight +
        cost_score * cost_weight
    )
    
    # Boost for critical priority
    if item.priority == 'critical':
        score *= 1.5
        
    return score
```

### Collision Avoidance
```bash
# Cooldown calculation based on worker count
MAX_CONCURRENT_WORKERS=${MAX_CONCURRENT_WORKERS:-3}
ITERATION_TIME_HOURS=4

# Set cooldown to prevent collision
cooldown_hours=$((ITERATION_TIME_HOURS * MAX_CONCURRENT_WORKERS))
cooldown_until=$(date -d "+${cooldown_hours} hours" --iso-8601)

# Check if item is available
is_available() {
    local item=$1
    local cooldown=$(yq '.metadata.cooldown_until' $item)
    local now=$(date --iso-8601)
    
    if [[ "$now" > "$cooldown" ]]; then
        return 0  # Available
    else
        return 1  # Still cooling down
    fi
}
```

## Queue Operations

### Adding Items to Queue
```bash
# Manual addition
cp queue/templates/improvement.yaml queue/pending/100-add-auth.yaml
# Edit with specific requirements

# Programmatic addition
create_queue_item() {
    local priority=$1
    local title=$2
    local type=$3
    local target=$4
    
    local id="$(date +%s)-$(uuidgen | cut -c1-8)"
    local filename="queue/pending/${priority}-${id}.yaml"
    
    cat > $filename <<EOF
# Priority: $priority
# Created: $(date --iso-8601)
# Created By: $USER

id: $id
title: "$title"
type: $type
target: $target
# ... rest of template
EOF
}
```

### Processing Queue Items
```bash
# 1. Select item
ITEM=$(select_from_queue)

# 2. Move to in-progress
mv queue/pending/$ITEM queue/in-progress/
yq -i '.metadata.assigned_to = "'$AGENT_ID'"' queue/in-progress/$ITEM
yq -i '.metadata.attempt_count += 1' queue/in-progress/$ITEM

# 3. Process the work
process_item queue/in-progress/$ITEM

# 4. Move to completed or failed
if [ $? -eq 0 ]; then
    mv queue/in-progress/$ITEM queue/completed/${ITEM}-$(date +%s)
    yq -i '.result.status = "success"' queue/completed/${ITEM}-*
else
    mv queue/in-progress/$ITEM queue/failed/${ITEM}-$(date +%s)
    yq -i '.result.status = "failed"' queue/failed/${ITEM}-*
fi
```

### Queue Status Monitoring
```bash
# Check queue status
queue_status() {
    echo "Queue Status:"
    echo "  Pending: $(ls queue/pending/*.yaml 2>/dev/null | wc -l)"
    echo "  In Progress: $(ls queue/in-progress/*.yaml 2>/dev/null | wc -l)"
    echo "  Completed Today: $(find queue/completed -mtime -1 -name "*.yaml" | wc -l)"
    echo "  Failed Today: $(find queue/failed -mtime -1 -name "*.yaml" | wc -l)"
    
    # Show current work
    if ls queue/in-progress/*.yaml >/dev/null 2>&1; then
        echo -e "\nCurrently Processing:"
        for item in queue/in-progress/*.yaml; do
            echo "  - $(yq '.title' $item) [$(yq '.metadata.assigned_to' $item)]"
        done
    fi
}
```

## Queue Management Best Practices

### Priority Guidelines
```markdown
## Priority Assignment

Critical (001-099):
- Security vulnerabilities
- Data loss risks
- System down issues
- Revenue-blocking bugs

High (100-199):
- P0 requirements
- Major functionality gaps
- Performance issues >2x target
- User-facing errors

Medium (200-299):
- P1 requirements
- Minor bugs
- UI improvements
- Documentation updates

Low (300-399):
- P2 requirements
- Cosmetic issues
- Nice-to-have features
- Code cleanup
```

### Failure Handling
```markdown
## When Queue Items Fail

1. First Failure:
   - Document error in result.issues_encountered
   - Increase cooldown period
   - Move back to pending with notes
   - Adjust priority if needed

2. Second Failure:
   - Analyze failure pattern
   - Break into smaller items if too complex
   - Add more specific requirements
   - Consider dependencies

3. Third Failure:
   - Mark as blocked
   - Create investigation task
   - Notify human intervention needed
   - Document in PRD as known issue
```

### Queue Maintenance
```bash
# Clean old completed items (keep 30 days)
find queue/completed -name "*.yaml" -mtime +30 -delete

# Archive failed items (after 7 days)
mkdir -p queue/archive/failed
find queue/failed -name "*.yaml" -mtime +7 -exec mv {} queue/archive/failed/ \;

# Requeue stale in-progress items (>24 hours)
find queue/in-progress -name "*.yaml" -mtime +1 -exec mv {} queue/pending/ \;
```

## Integration with Other Systems

### Memory System Integration
```yaml
# Every queue item should include memory context
memory_context: |
  Before starting, search:
  - vrooli resource-qdrant search "[topic] implementation"
  - vrooli resource-qdrant search "[topic] failed"
  
  Related knowledge:
  - [Previous similar work]
  - [Known patterns]
  - [Things to avoid]
```

### PRD Integration
```yaml
# Link queue items to PRD requirements
requirements:
  - "PRD Section 3.2: User Authentication"
  - "Implement P0 requirement #5"
  
validation_criteria:
  - "PRD test case 3.2.1 passes"
  - "Checkbox can be marked in PRD"
```

## Queue Anti-Patterns

### Queue Stuffing
❌ **Bad**: Adding 50 items at once
✅ **Good**: Adding items as discovered, prioritized

### Zombie Items
❌ **Bad**: Items stuck in-progress forever
✅ **Good**: Timeout and requeue after 24 hours

### Vague Items
❌ **Bad**: "Fix the bug in the system"
✅ **Good**: "Fix auth endpoint returning 404 for valid tokens"

### No Context
❌ **Bad**: Just the task description
✅ **Good**: Include memory searches and previous attempts

## Remember for Queue Processing

**One at a time** - Never process multiple items simultaneously

**Respect cooldowns** - Prevents agent collisions

**Document everything** - Future agents need context

**Fail gracefully** - Move to failed, don't delete

**Learn from failures** - Update memory system

The queue is your organized workload. Use it to maintain steady progress, avoid conflicts, and enable collaboration between multiple agents and humans.