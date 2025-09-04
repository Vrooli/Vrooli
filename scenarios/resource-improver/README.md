# Resource Improver

Continuous improvement system for existing Vrooli resources using v2.0 contract validation and health monitoring patterns.

## Overview

The Resource Improver automatically identifies and implements improvements to existing resources, ensuring they:
- Meet v2.0 contract requirements
- Have robust health checking
- Implement proper lifecycle hooks
- Provide consistent CLI interfaces
- Follow best practices from Qdrant memory

## Architecture

### Components
- **Contract Validator**: Checks v2.0 compliance
- **Health Monitor**: Improves health check implementations
- **CLI Enhancer**: Standardizes command interfaces
- **Documentation Generator**: Updates READMEs
- **Memory Integration**: Learns from past improvements

### Resources
- **Claude Code**: AI-powered improvement generation
- **PostgreSQL**: Improvement history and compliance tracking
- **Redis**: Queue management and session caching
- **Qdrant**: Long-term memory and pattern matching
- **N8n**: Improvement workflow automation

## Key Features

### 1. v2.0 Contract Enforcement
Ensures every resource has:
- Complete lib/ directory with required scripts
- Health check implementation with timeouts
- All lifecycle hooks (setup, develop, test, stop)
- CLI integration with standard commands
- Proper service.json configuration

### 2. Health Monitoring Improvements
- Adds timeout handling to health checks
- Implements retry logic with backoff
- Provides detailed error messages
- Supports multiple health endpoints
- Adds startup grace periods

### 3. Content Management Standardization
- Implements consistent content add/remove/list patterns
- Adds support for workflows, models, configs
- Provides batch operations
- Ensures proper validation

### 4. CLI Enhancement
- Standardizes command structure
- Adds help documentation
- Implements verbose and JSON output
- Provides command completion
- Ensures consistent error handling

## Usage

### Starting the Improver
```bash
# Setup (first time)
vrooli scenario resource-improver setup

# Run improvement agent
vrooli scenario resource-improver develop
```

### Adding Improvement Tasks
```bash
# Add to queue manually
cp queue/templates/improvement.yaml queue/pending/100-improve-ollama.yaml
# Edit with specific requirements

# Or use CLI
resource-improver add --target ollama --type health-check --priority high
```

### Monitoring Progress
```bash
# Check queue status
resource-improver status

# View recent improvements
ls -lt queue/completed/*.yaml | head -10

# Check compliance scores
resource-improver compliance --all
```

## Queue System

The queue manages improvement tasks:

```
queue/
├── pending/        # Improvements waiting
├── in-progress/    # Currently being applied
├── completed/      # Successfully completed
├── failed/         # Failed with logs
└── templates/      # Task templates
```

## Improvement Process

### 1. Resource Analysis
- Check v2.0 contract compliance
- Identify missing components
- Review health check implementation
- Analyze CLI coverage
- Check documentation completeness

### 2. Priority Assessment
- Contract violations (critical)
- Health check issues (high)
- CLI gaps (medium)
- Documentation updates (low)

### 3. Implementation
- Generate improvement code
- Apply changes incrementally
- Test each modification
- Document changes

### 4. Validation
- Run health checks
- Execute test suite
- Verify CLI commands
- Check breaking changes

### 5. Documentation
- Update README
- Document new commands
- Add troubleshooting guides
- Update API references

## Common Improvements

### Health Check Enhancements
```bash
# Before
check_health() {
    curl http://localhost:$PORT/health
}

# After
check_health() {
    local timeout="${1:-5}"
    local retries="${2:-3}"
    
    for i in $(seq 1 $retries); do
        if timeout $timeout curl -sf http://localhost:$PORT/health >/dev/null 2>&1; then
            echo "✅ Health check passed"
            return 0
        fi
        [ $i -lt $retries ] && sleep 1
    done
    
    echo "❌ Health check failed after $retries attempts"
    return 1
}
```

### CLI Standardization
```bash
# Adds standard commands
resource-ollama status       # Show status
resource-ollama health       # Check health
resource-ollama logs         # View logs
resource-ollama content list # List content
resource-ollama help         # Show help
```

### Lifecycle Improvements
```bash
# Adds proper cleanup
stop() {
    echo "Stopping $RESOURCE_NAME..."
    
    # Graceful shutdown
    pkill -TERM -f "$RESOURCE_PROCESS"
    sleep 2
    
    # Force if needed
    pkill -KILL -f "$RESOURCE_PROCESS" 2>/dev/null || true
    
    # Cleanup
    rm -f "$PID_FILE"
    echo "✅ $RESOURCE_NAME stopped"
}
```

## Resource Categories

### Core Infrastructure
**postgres, redis, questdb, minio**
- Focus: Connection management, monitoring

### AI/ML Resources
**ollama, litellm, claude-code, comfyui**
- Focus: Model management, API consistency

### Workflow Automation
**n8n, node-red, huginn, windmill**
- Focus: Workflow persistence, error recovery

### Development Tools
**judge0, browserless, vault**
- Focus: Security, performance

## Success Metrics

Tracks per resource:
- v2.0 compliance score (target: 100%)
- Health check reliability (target: 99.9%)
- CLI command coverage (target: 100%)
- Documentation completeness (target: 90%)
- Test coverage (target: 80%)

## Examples

### Example: Improving Ollama
```yaml
# queue/pending/001-improve-ollama-health.yaml
id: improve-ollama-health-20250103
title: "Add robust health checking to Ollama"
target: ollama
type: health-check
priority: high
requirements:
  - Add timeout handling
  - Implement retry logic
  - Check model availability
  - Add startup grace period
```

### Example: CLI Enhancement
```yaml
# queue/pending/002-enhance-n8n-cli.yaml
id: enhance-n8n-cli-20250103
title: "Standardize n8n CLI commands"
target: n8n
type: cli-enhancement
priority: medium
requirements:
  - Add 'content list' command
  - Implement JSON output
  - Add verbose mode
  - Improve error messages
```

## Development

### Testing Improvements
```bash
# Test specific resource
resource-improver test ollama

# Validate all resources
resource-improver validate --all

# Check compliance
resource-improver compliance --report
```

### Adding New Patterns
1. Identify common improvement need
2. Add to prompts/main-prompt.md
3. Create template in queue/templates/
4. Test on sample resource
5. Document in README

## Troubleshooting

### Improvement Failures
- Check logs in queue/failed/
- Verify resource is running
- Review breaking changes
- Check dependency conflicts

### Compliance Issues
- Run `resource-improver compliance <resource>`
- Review missing components
- Check service.json validity
- Verify file permissions

## Related Documentation

- [v2.0 Contract](/docs/resources/v2.0-contract.md)
- [Migration Plan](/auto/docs/MIGRATION_PLAN.md)
- [Resource Generator](../resource-generator/README.md)
- [Scenario Improver](../scenario-improver/README.md)