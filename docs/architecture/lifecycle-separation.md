# Lifecycle Phase Separation

## Overview
The lifecycle system has been radically simplified:
1. **98% code reduction**: From 6,327 lines to 135 lines
2. **Clear separation**: Generic infrastructure vs application-specific logic
3. **Scenario-first**: Base Vrooli is now just a development environment for scenarios

## Architecture

### Generic Setup (`scripts/lib/lifecycle/phases/setup.sh`)
Handles ONLY universal tasks needed by ANY application:
- System preparation (permissions, clock sync)
- Core dependencies (git, jq, bash essentials)
- Docker setup and verification
- Network diagnostics and firewall
- Development tools (bats, shellcheck for dev/CI environments)

### Application-Specific Setup
Each application defines its own setup steps in `.vrooli/service.json`:
```json
{
  "lifecycle": {
    "setup": {
      "steps": [
        {
          "name": "run-generic-setup",
          "run": "bash scripts/lib/lifecycle/phases/setup.sh",
          "description": "Execute generic setup"
        },
        {
          "name": "app-specific-step",
          "run": "...",
          "description": "Application-specific setup"
        }
      ]
    }
  }
}
```

## Benefits

1. **Clear Separation**: Generic vs app-specific logic is explicit
2. **Flexibility**: Each app controls exactly what it needs
3. **Simplicity**: No complex detection logic or conditionals
4. **Maintainability**: Changes to generic setup don't affect apps
5. **Modularity**: Apps can skip generic setup if not needed

## Migration Guide

For existing scenarios:
1. Ensure first setup step calls generic setup if needed
2. Add app-specific steps after generic setup
3. Resources should be installed via app-specific steps, not generic setup

## Architecture Philosophy

### Base Vrooli Repository
- **Purpose**: Development environment for creating and testing scenarios
- **NOT**: A main application (that will be a scenario)
- **Lifecycle phases**: Only run minimal generic tasks
- **service.json**: Extremely simple, just calls generic phase handlers

### Scenarios
- **Purpose**: Self-contained applications that ARE the Vrooli platform
- **Examples**: 
  - Dashboard scenario (will replace current Vrooli UI)
  - Agent metareasoning manager
  - Analytics dashboard
  - And 20+ other scenarios
- **service.json**: Contains all app-specific logic

### Generic Phase Scripts
- `scripts/lib/setup.sh`: Docker, dependencies, network (the only remaining phase script)
- Other phases: Handled entirely by service.json steps

## Key Insight
The main Vrooli codebase (packages/server, packages/ui, etc.) will eventually be refactored into scenarios. Until then, it exists alongside the scenario system, but the lifecycle management is already scenario-first.