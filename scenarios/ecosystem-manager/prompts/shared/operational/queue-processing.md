# Queue Processing

## Purpose
The queue system provides organized work distribution and prevents duplicate work. Understanding the queue structure and task schema helps agents work effectively within the system.

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
  - vrooli resource qdrant search "relevant pattern"
  
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

## Task Processing

### When Tasks Fail
1. **First failure**: Move back to pending with error note
2. **Second failure**: Break into smaller tasks
3. **Third failure**: Mark as blocked, needs human help

### Queue Status Check
```bash
# Simple status check
ecosystem-manager queue                # Show current status
ls queue/pending/*.yaml | wc -l       # Count pending tasks
ls queue/in-progress/*.yaml            # Show active tasks
```

### Queue Maintenance
```bash  
# Clean up old completed tasks (weekly)
find queue/completed -name "*.yaml" -mtime +7 -delete

# Move stale in-progress tasks back to pending (daily)
find queue/in-progress -name "*.yaml" -mtime +1 -exec mv {} queue/pending/ \;
```

## Key Queue Principles

- **One task at a time** - Focus on single item
- **Simple commands** - Use CLI instead of complex bash
- **Clear task descriptions** - Avoid vague requirements  
- **Memory search first** - Always search before starting
- **Document failures** - Learn from what doesn't work

The queue keeps work organized and prevents conflicts between agents.