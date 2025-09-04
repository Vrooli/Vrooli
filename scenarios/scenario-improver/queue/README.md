# Scenario Improvement Queue System

This directory contains the queue-based task management system for scenario improvements.

## Directory Structure

```
queue/
├── pending/           # Improvements waiting to be processed
├── in-progress/       # Currently being worked on (max 1 at a time)
├── completed/         # Successfully completed improvements
├── failed/           # Failed attempts with detailed logs
├── templates/        # Templates for creating new queue items
└── README.md        # This file
```

## How It Works

### 1. Adding Items to Queue
- Create YAML files in `pending/` using templates from `templates/`
- Use priority prefix in filename: `001-critical-fix.yaml`, `100-medium-improvement.yaml`
- Lower numbers = higher priority

### 2. Processing Items
- Agent selects highest priority item from `pending/`
- Moves item to `in-progress/` (only 1 allowed at a time)
- Implements the improvement
- Moves to `completed/` or `failed/` based on outcome

### 3. Priority System
Items are scored using:
```
priority = (impact * 2 + urgency * 1.5) * success_prob / resource_cost
```

### 4. Cooldown Mechanism
- Each item has a `cooldown_until` timestamp
- Prevents multiple agents from colliding
- Default cooldown: 1 hour after creation

## File Naming Convention

```
[priority]-[type]-[target]-[id].yaml
```

Examples:
- `001-critical-fix-system-monitor-20250103.yaml`
- `050-prd-compliance-scenario-generator-v1.yaml`
- `100-optimization-agent-metareasoning.yaml`

## Queue Item Schema

See `templates/improvement.yaml` for the full schema. Key fields:
- `id`: Unique identifier
- `type`: improvement|fix|optimization|prd-compliance|documentation
- `priority`: critical|high|medium|low
- `priority_estimates`: Scoring factors for selection
- `validation_criteria`: How to verify success
- `cross_scenario`: Impact analysis

## Manual Intervention

### Adding a High-Priority Fix
```yaml
# Save as: pending/001-critical-fix-[scenario]-[date].yaml
id: critical-fix-20250103
title: "Fix breaking change in API endpoint"
priority: critical
priority_estimates:
  impact: 10
  urgency: critical
  success_prob: 0.9
  resource_cost: minimal
```

### Requeuing Failed Items
1. Review item in `failed/` directory
2. Update approach based on failure reason
3. Reset attempt_count if needed
4. Move back to `pending/` with new priority

## Integration with Auto System

During migration period:
- Auto system can read this queue
- File locks prevent concurrent access
- Shared metrics tracking

## Monitoring

Check queue status:
```bash
# Count pending items
ls pending/*.yaml 2>/dev/null | wc -l

# View in-progress work
ls in-progress/*.yaml 2>/dev/null

# Recent completions
ls -lt completed/*.yaml | head -5

# Failed items needing attention
ls failed/*.yaml 2>/dev/null
```

## Best Practices

1. **One item at a time** - Never have multiple items in `in-progress/`
2. **Document failures** - Always add detailed failure_reasons
3. **Update cooldowns** - Respect cooldown periods to prevent conflicts
4. **Clean old items** - Archive completed items older than 30 days
5. **Priority discipline** - Reserve "critical" for truly urgent items