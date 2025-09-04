# Scenario Improver

A continuous improvement system for existing Vrooli scenarios using PRD-driven validation and cross-scenario impact analysis.

## Overview

The Scenario Improver is a specialized agent that automatically identifies and implements improvements to existing scenarios. It ensures all scenarios:
- Meet their PRD requirements
- Pass validation gates consistently  
- Maintain cross-scenario compatibility
- Follow Vrooli best practices
- Leverage collective knowledge from Qdrant memory

## Architecture

### Components
- **Queue System**: File-based task queue in `queue/` directory
- **Improvement Engine**: PRD-driven improvement logic
- **Validation System**: Multi-gate validation framework
- **Memory Integration**: Qdrant-based learning system
- **Cross-Scenario Analyzer**: Impact assessment tools

### Resources
- **PostgreSQL**: Improvement history and tracking
- **Redis**: Queue state and active sessions
- **N8n**: Improvement workflow orchestration
- **Qdrant**: Long-term memory and pattern matching
- **QuestDB**: Time-series metrics tracking

## Key Features

### 1. PRD Compliance Tracking
- Reads each scenario's PRD.md
- Tracks requirement completion
- Prioritizes missing P0 requirements
- Updates PRD completion percentages

### 2. Validation Gates
Every improvement passes through:
1. Functional validation
2. Integration testing
3. Documentation updates
4. Test suite execution
5. Memory system updates
6. Cross-scenario compatibility

### 3. Queue-Based Processing
- Prioritized task queue
- One improvement at a time
- Automatic failure recovery
- Manual intervention support
- Cooldown periods to prevent conflicts

### 4. Cross-Scenario Impact Analysis
- Dependency mapping
- Breaking change detection
- Resource conflict prevention
- API contract validation

## Usage

### Starting the Improver
```bash
# Setup (first time)
vrooli scenario scenario-improver setup

# Run improvement agent
vrooli scenario scenario-improver develop
```

### Adding Improvement Tasks
```bash
# Add to queue manually
cp queue/templates/improvement.yaml queue/pending/100-improve-system-monitor.yaml
# Edit the file with specific requirements

# Or use CLI
scenario-improver add --target system-monitor --type prd-compliance --priority high
```

### Monitoring Progress
```bash
# Check queue status
scenario-improver status

# View recent completions
ls -lt queue/completed/*.yaml | head -10

# Check failed improvements
scenario-improver failures --details
```

## Queue System

The queue system manages improvement tasks:

```
queue/
├── pending/        # Waiting for processing
├── in-progress/    # Currently being improved (max 1)
├── completed/      # Successfully completed
├── failed/         # Failed with logs
└── templates/      # Task templates
```

See [queue/README.md](queue/README.md) for detailed queue documentation.

## Improvement Process

1. **Selection**: Choose highest priority item from queue
2. **Analysis**: Review scenario against PRD and best practices  
3. **Memory Search**: Find relevant patterns and past solutions
4. **Implementation**: Make focused, targeted improvements
5. **Validation**: Run through all validation gates
6. **Documentation**: Update README, PRD, and inline docs
7. **Memory Update**: Refresh Qdrant embeddings
8. **Completion**: Move to completed queue, or retry if failed

## Integration with Auto System

During the migration from `auto/`:
- Coordinates with auto loops to prevent conflicts
- Shares queue state and metrics
- Gradually takes over improvement responsibilities
- Preserves all auto/ knowledge and patterns

## Success Metrics

Tracks improvement effectiveness:
- PRD completion rate increase
- Validation gate pass rate
- Cross-scenario compatibility score
- Documentation quality score
- Test coverage improvement
- Performance metric gains

## Examples

### Example: Improving PRD Compliance
```yaml
# queue/pending/050-prd-compliance-system-monitor.yaml
id: prd-compliance-system-monitor-20250103
title: "Complete P0 requirements for System Monitor"
type: prd-compliance
target: system-monitor
priority: high
priority_estimates:
  impact: 8
  urgency: high
  success_prob: 0.85
  resource_cost: moderate
requirements:
  - "Implement missing threshold configuration UI"
  - "Add automated alert escalation"
  - "Create performance baseline feature"
```

### Example: Cross-Scenario Optimization
```yaml
# queue/pending/100-optimize-shared-workflows.yaml
id: optimize-shared-workflows-20250103
title: "Optimize shared N8n workflows across scenarios"
type: optimization
target: all-scenarios
priority: medium
cross_scenario:
  affected_scenarios: ["system-monitor", "agent-metareasoning-manager"]
  shared_resources: ["n8n", "ollama"]
```

## Development

### Running Tests
```bash
# Test improvement logic
vrooli scenario scenario-improver test

# Validate queue processing
./test/test-queue-processing.sh
```

### Adding New Improvement Types
1. Create template in `queue/templates/`
2. Update prompt in `prompts/main-prompt.md`
3. Add validation logic for new type
4. Test with sample improvements

## Troubleshooting

### Common Issues

**Queue Processing Stuck**
```bash
# Check for stale locks
ls queue/in-progress/*.yaml
# If stuck > 2 hours, move back to pending
mv queue/in-progress/*.yaml queue/pending/
```

**Memory Search Not Working**
```bash
# Refresh Qdrant embeddings
vrooli resource qdrant embeddings refresh
```

**Validation Gates Failing**
- Check logs in failed queue items
- Review validation criteria
- Ensure resources are running

## Related Documentation

- [Migration Plan](/auto/docs/MIGRATION_PLAN.md)
- [Auto System](/auto/README.md)
- [Queue System](queue/README.md)
- [PRD Methodology](/docs/prd-methodology.md)