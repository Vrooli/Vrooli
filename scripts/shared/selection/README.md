# Queue Selection System

Shared selection and queue management tools for distributed Vrooli scenarios.

## Overview

These tools provide queue-based task selection and management, replacing the direct selection mechanisms from the `auto/` system. They enable:
- Priority-based selection with scoring
- Cooldown periods to prevent conflicts
- Queue state management
- Event tracking for auditing

## Tools

### queue-select.sh
Selects the highest priority item from a pending queue.

**Features:**
- Priority scoring based on impact, urgency, success probability, and resource cost
- Cooldown checking to prevent conflicts
- Automatic move to in-progress
- Event logging

**Usage:**
```bash
./queue-select.sh <queue-dir> [scenario-type]

# Example
./queue-select.sh scenarios/scenario-improver/queue improvement
```

### queue-manage.sh
Comprehensive queue management utilities.

**Commands:**
- `init` - Initialize queue directories
- `status` - Show queue status
- `complete` - Mark current item as completed
- `fail` - Mark current item as failed
- `retry` - Retry a failed item
- `clear` - Clear stuck in-progress items
- `add` - Add new item from template

**Usage:**
```bash
./queue-manage.sh <command> [queue-dir] [item]

# Examples
./queue-manage.sh status queue
./queue-manage.sh complete queue
./queue-manage.sh fail queue "API timeout"
./queue-manage.sh retry queue "001-fix-health.yaml"
```

## Queue Structure

```
queue/
├── pending/          # Items waiting to be processed
├── in-progress/      # Currently being worked on (max 1)
├── completed/        # Successfully completed items
├── failed/           # Failed items with error logs
├── templates/        # Templates for new items
└── events.ndjson     # Event log for tracking
```

## Priority Scoring

Items are scored using the formula:
```
score = (impact * 2 + urgency * 1.5) * success_probability / resource_cost
```

Where:
- **impact**: 1-10 scale of improvement value
- **urgency**: critical(10), high(7), medium(5), low(2)
- **success_probability**: 0-1 probability of success
- **resource_cost**: minimal(1), moderate(2), heavy(4)

## Queue Item Schema

```yaml
id: unique-identifier
title: "Clear, actionable title"
type: improvement|fix|optimization
priority: critical|high|medium|low

priority_estimates:
  impact: 8
  urgency: high
  success_prob: 0.85
  resource_cost: moderate

metadata:
  created_at: "2025-01-03T10:00:00Z"
  cooldown_until: "2025-01-03T11:00:00Z"
  attempt_count: 0
  failure_reasons: []
```

## Integration with Scenarios

Each scenario should:
1. Initialize its queue during setup
2. Use `queue-select.sh` to pick work items
3. Use `queue-manage.sh` to update status
4. Check cooldowns to prevent conflicts
5. Log events for auditing

### Example Integration

```bash
#!/bin/bash
# In scenario improvement loop

# Select next item
SELECTED=$(./scripts/shared/selection/queue-select.sh queue)

# Process the item
process_improvement "$SELECTED"

# Mark as complete or failed
if [[ $? -eq 0 ]]; then
    ./scripts/shared/selection/queue-manage.sh complete queue
else
    ./scripts/shared/selection/queue-manage.sh fail queue "$ERROR_MSG"
fi
```

## Event Tracking

All queue operations are logged to `events.ndjson`:

```json
{"type":"queue_selection","ts":"2025-01-03T10:00:00Z","item":"001-fix.yaml","score":15.5}
{"type":"queue_complete","ts":"2025-01-03T10:30:00Z","item":"001-fix.yaml","detail":"success"}
```

This enables:
- Audit trails
- Performance metrics
- Debugging failed items
- Success rate tracking

## Migration from Auto System

### Old (Direct Selection)
```bash
# auto/tools/selection/scenario-select.sh
./scenario-select.sh app-monitor
```

### New (Queue-Based)
```bash
# scripts/shared/selection/queue-select.sh
./queue-select.sh scenarios/scenario-improver/queue
```

### Key Differences
1. **Queue-based**: Items in files, not direct selection
2. **One at a time**: Only one in-progress item
3. **State tracking**: Clear pending/progress/complete states
4. **Manual intervention**: Easy to add/modify queue items
5. **Cooldowns**: Prevent conflicts between agents

## Best Practices

1. **Always check in-progress** before selecting new items
2. **Set cooldowns** to prevent rapid retries
3. **Document failures** with clear reasons
4. **Archive old completed items** periodically
5. **Monitor queue depth** to prevent backlog

## Dependencies

Required:
- `bash` 4.0+
- `bc` for calculations
- `date` with ISO format support

Optional but recommended:
- `yq` for YAML processing
- `jq` for JSON handling
- `flock` for file locking

## Troubleshooting

### Queue Stuck
```bash
# Check and clear stuck items
./queue-manage.sh clear queue
```

### All Items in Cooldown
```bash
# Check cooldown times
for f in queue/pending/*.yaml; do
    echo "$(basename $f): $(yq .metadata.cooldown_until $f)"
done
```

### Selection Not Working
```bash
# Debug selection process
bash -x ./queue-select.sh queue
```

## Future Enhancements

- [ ] Web UI for queue visualization
- [ ] Metrics dashboard
- [ ] Automatic retry with backoff
- [ ] Queue depth alerts
- [ ] Priority adjustment based on age