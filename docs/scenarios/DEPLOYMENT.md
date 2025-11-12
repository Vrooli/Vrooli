# Direct Scenario Deployment Guide

> 📚 **[Back to Scenario Documentation](README.md)**

## 🚀 Running Scenarios Directly

Scenarios run directly from their source location without conversion:

```bash
# Run a scenario
cd scenarios/research-assistant
../../scripts/manage.sh develop

# Or use the CLI
vrooli scenario run research-assistant
```

## How Direct Execution Works

1. **No Conversion Needed**: Scenarios run directly from `scenarios/` folder
2. **Process Isolation**: Each scenario gets its own PM_HOME and PM_LOG_DIR
3. **Resource Sharing**: Scenarios use Vrooli's scripts and libraries
4. **Instant Updates**: Changes take effect immediately

## Running Scenarios

### Using the CLI

```bash
# List available scenarios
vrooli scenario list

# Run a scenario
vrooli scenario run make-it-vegan

# Test a scenario
vrooli scenario test make-it-vegan
```

### Direct Execution

```bash
# Navigate to scenario
cd scenarios/make-it-vegan

# Run the scenario
vrooli scenario run <scenario-name>

# Test the scenario
vrooli scenario test <scenario-name>
```

## Process Management

Each scenario runs with isolated process management:

- **Process Home**: `~/.vrooli/processes/scenarios/<scenario-name>/`
- **Log Directory**: `~/.vrooli/logs/scenarios/<scenario-name>/`
- **Port Allocation**: Handled by Vrooli's port registry

## Resource Sharing

Scenarios share Vrooli's local resources:
- Ollama for LLM inference
- N8n for workflow automation
- PostgreSQL for data persistence
- Redis for caching
- Qdrant for vector search

## Environment Variables

When running in scenario mode, these variables are automatically set:
- `SCENARIO_NAME`: Name of the current scenario
- `SCENARIO_MODE`: Set to `true`
- `SCENARIO_PATH`: Full path to scenario directory
- `PM_HOME`: Scenario-specific process directory
- `PM_LOG_DIR`: Scenario-specific log directory

## 🏗️ service.json Structure

The service.json file contains everything needed for deployment:

```json
{
  "service": {
    "name": "research-assistant",
    "displayName": "AI Research Assistant",
    "description": "Enterprise-grade AI research platform"
  },
  "dependencies": {
    "resources": {
      "postgres": {
        "enabled": true,
        "required": true,
        "initialization": {
          "data": [
            {
              "type": "schema", 
              "file": "initialization/storage/postgres/schema.sql"
            }
          ]
        }
      },
      "ollama": {
        "enabled": true,
        "required": true,
        "models": ["qwen2.5:32b", "nomic-embed-text"]
      },
      "windmill": {
        "enabled": true,
        "required": true,
        "initialization": {
          "apps": [
            {
              "path": "f/research-assistant/dashboard",
              "file": "initialization/automation/windmill/dashboard-app.json"
            }
          ]
        }
      }
    }
  },
  "deployment": {
    "urls": {
      "application": "${service.vrooli.url}",
      "windmill": "${service.windmill.url}"
    }
  }
}
```

## Deployment Strategies

### Local Development

```bash
# Quick start for development
vrooli scenario run <name>

# With specific options
vrooli scenario run <name> --verbose --debug
```

### Production Deployment

```bash
# Package scenarios for deployment
./scripts/deployment/package-scenario-deployment.sh \
    "production-suite" \
    ~/deployments/production \
    research-assistant make-it-vegan invoice-generator
```

### CI/CD Integration

```yaml
# Example GitHub Actions workflow
jobs:
  test-scenarios:
    steps:
      - name: Test Scenarios
        run: |
          for scenario in scenarios/*/; do
            name=$(basename "$scenario")
            echo "Testing $name"
            (cd "$scenario" && vrooli scenario test "$(basename "$scenario")")
          done
```

## Troubleshooting

### Scenario Won't Start

```bash
# Check for service.json
ls scenarios/<name>/.vrooli/service.json

# Verify scenario mode is detected
cd scenarios/<name>
vrooli scenario run <scenario-name> --dry-run
```

### Port Conflicts

```bash
# Check allocated ports
cat ~/.vrooli/port-registry.json

# Reset port allocations
rm ~/.vrooli/port-registry.json
```

### Process Issues

```bash
# Check scenario processes
ps aux | grep "scenarios/<name>"

# Clean up scenario processes
pkill -f "scenarios/<name>"
```

## Best Practices

1. **Always test locally first**: `vrooli scenario test <name>`
2. **Use process manager**: Let Vrooli handle process lifecycle
3. **Monitor logs**: Check `~/.vrooli/logs/scenarios/<name>/`
4. **Clean up when done**: `vrooli stop` to stop all scenarios

## Performance Benefits

Direct execution provides significant improvements:
- **2-5 seconds faster** startup time
- **Zero duplication** of files
- **Instant changes** without regeneration
- **Simpler debugging** with direct source access

## Advanced Configuration

### Custom Process Settings

Edit `.vrooli/service.json` in your scenario:

```json
{
  "service": {
    "name": "my-scenario",
    "description": "Custom scenario configuration"
  },
  "lifecycle": {
    "develop": {
      "steps": ["setup", "start-api", "start-ui"]
    }
  }
}
```

### Resource Requirements

Specify required resources in service.json:

```json
{
  "dependencies": {
    "resources": {
      "ollama": {
        "enabled": true,
        "required": true
      },
      "postgresql": {
        "enabled": true,
        "required": true
      },
      "redis": {
        "enabled": false,
        "required": false
      }
    }
  }
}
```

## Security Considerations

- Scenarios run with user permissions
- Process isolation prevents cross-scenario interference
- Resource access controlled by Vrooli's security model
- No elevated privileges required

## Support

For issues or questions:
- Check logs: `~/.vrooli/logs/scenarios/<name>/`
- Run diagnostics: `vrooli doctor`
- Report issues: https://github.com/anthropics/vrooli/issues
