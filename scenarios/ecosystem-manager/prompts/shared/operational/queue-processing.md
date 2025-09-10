# Queue Processing

## Purpose
The queue system provides organized work distribution and prevents duplicate work. Understanding the queue structure and task schema helps agents work effectively within the system.

## Queue System Architecture

### Directory Structure
```
ecosystem-manager/queue/
├── pending/              # Tasks waiting to be processed
│   └── [type]-[operation]-[name]-[timestamp].yaml
├── in-progress/          # Currently being executed
│   └── [task-id].yaml
├── review/               # Tasks under review (optional status)
│   └── [task-id].yaml
├── completed/            # Successfully completed tasks
│   └── [task-id].yaml
├── failed/               # Failed tasks with error details
│   └── [task-id].yaml
└── templates/            # Templates for task creation
    ├── resource-generation.yaml
    ├── resource-improvement.yaml
    ├── scenario-generation.yaml
    ├── scenario-improvement.yaml
    └── unified-task-schema.yaml
```

### Actual Task Schema (TaskItem)
```yaml
# Task identification
id: resource-generator-nextcloud-20250110-010900  # Format: [type]-[operation]-[name]-[timestamp]
title: "Human-readable task description"

# Core classification
type: resource            # resource | scenario
operation: generator      # generator | improver
category: productivity    # ai, storage, communication, security, etc.

# Priority and scheduling
priority: medium          # low | medium | high | critical
effort_estimate: 8h       # 1h | 2h | 4h | 8h | 16h+
urgency: ""              # Optional urgency indicator
impact_score: 0          # 0-10 business impact

# Task configuration
requirements: {}          # Map of specific requirements
dependencies: []          # List of task IDs that must complete first
blocks: []               # List of task IDs this blocks
related_scenarios: []    # Affected scenarios
related_resources: []    # Affected resources
assigned_resources: {}   # Resources needed for execution

# Progress tracking
status: pending          # pending | in-progress | review | completed | failed
progress_percentage: 0   # 0-100
current_phase: ""        # initialization | research | implementation | testing | documentation
started_at: ""           # ISO timestamp when started
completed_at: ""         # ISO timestamp when completed
estimated_completion: "" # ISO timestamp estimate

# Quality gates
validation_criteria: []  # List of success criteria

# Metadata
created_by: ""           # user | system | automation
created_at: "2025-01-10T01:09:00Z"
updated_at: "2025-01-10T01:09:00Z"
tags: []                 # Searchable tags
notes: ""                # Detailed notes/instructions

# Execution results (populated after completion)
results: {}              # Map containing execution output, errors, metrics
```

## Task Processing

### When Tasks Fail
1. **First failure**: Move back to pending with error note
2. **Second failure**: Break into smaller tasks
3. **Third failure**: Mark as blocked, needs human help

### Queue Status Check
```bash
# Check queue status via API
curl -s http://localhost:5020/api/queue/status | jq

# Count tasks by status
ls queue/pending/*.yaml 2>/dev/null | wc -l    # Pending tasks
ls queue/in-progress/*.yaml 2>/dev/null | wc -l # Active tasks
ls queue/completed/*.yaml 2>/dev/null | wc -l   # Completed tasks

# View specific task details
curl -s http://localhost:5020/api/tasks/[task-id] | jq
```

### Task Management
```bash
# Tasks are managed through the API, not direct file manipulation
# The ecosystem-manager handles moving tasks between statuses

# View tasks by status
curl -s "http://localhost:5020/api/tasks?status=pending" | jq
curl -s "http://localhost:5020/api/tasks?status=in-progress" | jq

# Filter tasks by type and operation
curl -s "http://localhost:5020/api/tasks?type=resource&operation=generator" | jq
```

## Key Queue Principles

- **One task at a time** - Focus on single item
- **Simple commands** - Use CLI instead of complex bash
- **Clear task descriptions** - Avoid vague requirements  
- **Memory search first** - Always search before starting
- **Document failures** - Learn from what doesn't work

The queue keeps work organized and prevents conflicts between agents.