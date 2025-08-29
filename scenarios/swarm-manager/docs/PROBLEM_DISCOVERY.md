# Problem Discovery System

## Overview

The Swarm Manager implements a sophisticated dual-file problem discovery system that automatically identifies issues across the Vrooli ecosystem and generates tasks to resolve them.

## Implementation Status

### ✅ Completed Features

1. **V2.0 Contract Update**
   - Added PROBLEMS.md to recommended documentation files
   - Positioned as "Active issues and unresolved problems (for task generation)"
   - Included in core_decision_making file category for semantic search

2. **Dual-File Scanning**
   - **PROBLEMS.md**: Active, unresolved issues requiring immediate attention
   - **TROUBLESHOOTING.md**: Known issues with solutions, patterns for prevention

3. **Enhanced CLI Commands**
   ```bash
   swarm-manager scan-problems      # Manual scan of all problem sources
   swarm-manager problems list      # View discovered problems
   swarm-manager problems show <id> # Details for specific problem
   swarm-manager problems resolve   # Mark problem as resolved
   ```

4. **Automated Scanning**
   - N8N workflow scans every 5 minutes
   - Processes both PROBLEMS.md and TROUBLESHOOTING.md
   - Different parsers for each file type

5. **Intelligent Task Generation**
   - Critical/high severity problems auto-generate tasks
   - Priority calculation from problem metadata
   - Routing to appropriate scenarios for resolution

## File Formats

### PROBLEMS.md Format
```markdown
<!-- EMBED:ACTIVEPROBLEM:START -->
### Problem Title
**Status:** [active|investigating|resolved]
**Severity:** [critical|high|medium|low]
**Frequency:** [constant|frequent|occasional|rare]
**Impact:** [system_down|degraded_performance|user_impact|cosmetic]

#### Priority Estimates
```yaml
impact: 8
urgency: "high"
success_prob: 0.8
resource_cost: "moderate"
```
<!-- EMBED:ACTIVEPROBLEM:END -->
```

### TROUBLESHOOTING.md Patterns
The system scans for:
- "Known issue:", "Workaround:", "TODO:", "FIXME:"
- Configuration problems and solutions
- Performance degradation patterns
- Integration failure indicators

## Architecture

### Problem Flow
1. **Discovery**: Files scanned every 5 minutes via N8N
2. **Parsing**: 
   - PROBLEMS.md → problem-analyzer.md prompt
   - TROUBLESHOOTING.md → troubleshooting-parser.md prompt
3. **Analysis**: Claude extracts structured data
4. **Task Creation**: High/critical issues generate tasks
5. **Routing**: Tasks assigned to appropriate scenarios
6. **Execution**: Scenarios attempt resolution
7. **Learning**: Success/failure patterns stored for future

### Integration Points
- **N8N Workflow**: problem-scanner.json orchestrates scanning
- **API Endpoints**: /api/problems/* for CRUD operations
- **CLI Interface**: Full problem management capabilities
- **UI Dashboard**: Visual problem tracking (planned)

## Configuration

In `config/settings.yaml`:
```yaml
problem_scanning:
  enabled: true
  scan_interval: 300  # 5 minutes
  auto_create_tasks: true
  scan_paths:
    - /*/PROBLEMS.md
    - /*/TROUBLESHOOTING.md
  severity_thresholds:
    auto_task_creation: high
    alert_notification: critical
```

## Benefits

1. **Proactive Resolution**: Problems discovered and addressed automatically
2. **Pattern Recognition**: Learning from TROUBLESHOOTING.md prevents recurrence
3. **Unified View**: Single dashboard for all system issues
4. **Semantic Understanding**: Structured format enables AI comprehension
5. **Preventive Action**: Patterns in troubleshooting guide prevention tasks

## Usage Examples

### Manual Problem Scan
```bash
$ swarm-manager scan-problems
Scanning for problems...
Found 19 active problems in PROBLEMS.md
Found 14 potential issues in TROUBLESHOOTING.md
✓ Task generation initiated
```

### View Problems
```bash
$ swarm-manager problems list critical
CRITICAL | Claude Code Rate Limiting | frequent | active
CRITICAL | OAuth Token Refresh Failing | constant | investigating
```

### Resolve Problem
```bash
$ swarm-manager problems resolve prob-123 "Fixed by increasing connection pool"
✓ Problem marked as resolved
```

## Future Enhancements

1. **UI Integration**: Visual problem dashboard in Trello-like interface
2. **Pattern Learning**: ML-based pattern recognition from resolved issues
3. **Auto-Resolution**: Attempting known fixes automatically
4. **Metrics Tracking**: Success rates per problem type
5. **Alert System**: Notifications for critical issues

## Migration Guide

For existing resources/scenarios:

1. Create PROBLEMS.md using template from PROBLEMS_TEMPLATE.md
2. Move active issues from TROUBLESHOOTING.md to PROBLEMS.md
3. Keep resolved issues in TROUBLESHOOTING.md with solutions
4. Add embedded markers for semantic extraction
5. Test with `swarm-manager scan-problems`

## See Also

- [PROBLEMS_TEMPLATE.md](/home/matthalloran8/Vrooli/PROBLEMS_TEMPLATE.md)
- [V2.0 Resource Contract](/scripts/resources/contracts/v2.0/universal.yaml)
- [Swarm Manager README](../README.md)