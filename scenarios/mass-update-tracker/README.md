# Mass Update Tracker

Campaign-based mass file operation tracking that maintains state across Claude Code conversations and agent iterations.

## Core Value

This scenario adds **persistent progress tracking** to Vrooli, enabling:
- Large-scale refactoring that survives conversation restarts
- Multi-agent coordination on file operations  
- Historical tracking of what files have been processed
- Resumable campaigns after context loss

## Features

✅ **Campaign Management**: Create campaigns with glob patterns to track files
✅ **Progress Persistence**: State survives across restarts and conversations
✅ **REST API**: Full programmatic access for agents
✅ **CLI Tool**: Command-line interface for campaign operations
✅ **Web Dashboard**: Visual campaign management and progress monitoring

## Quick Start

```bash
# Run the scenario
vrooli scenario run mass-update-tracker

# Create a campaign
mass-update-tracker create "refactor-typescript" "src/**/*.ts" --description "Convert JS to TS"

# List campaigns
mass-update-tracker list

# Check files in campaign
mass-update-tracker files <campaign-id>

# Update file status
mass-update-tracker update-file <campaign-id> src/main.ts completed
```

## API Access

- **Health**: http://localhost:20251/health
- **Campaigns**: http://localhost:20251/api/v1/campaigns
- **Dashboard**: http://localhost:3251

## Architecture

- **Go API**: High-performance REST API with PostgreSQL storage
- **CLI**: Bash-based CLI wrapper for all operations
- **UI**: Modern dashboard with real-time progress tracking
- **Database**: PostgreSQL with automatic schema initialization

## Use Cases

- **Code Migrations**: Track progress converting codebases between frameworks
- **Mass Refactoring**: Update patterns across hundreds of files systematically
- **Documentation Updates**: Track which docs have been updated
- **Test Coverage**: Monitor which files have tests added
- **Claude Code Integration**: Maintain state across multiple conversations

## Integration

Other scenarios can leverage mass-update-tracker via:
- REST API for programmatic access
- CLI for script integration
- PostgreSQL for direct data access

This becomes a foundational capability that amplifies all future mass operation scenarios.