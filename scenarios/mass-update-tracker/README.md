# 📊 Mass Update Tracker

## What It Is
A campaign-based tracking system for mass file operations that maintains persistent state across Claude Code conversations and agent iterations. Enables systematic coordination of large-scale refactoring, feature additions, and batch operations.

## Why It's Revolutionary
- **Persistent Context**: Never lose track of progress when conversations restart or context limits are reached
- **Loop-Friendly**: Perfect for iterative workflows where agents work on batches of files
- **Cross-Conversation Continuity**: Pick up exactly where you left off, even with different agents
- **Scalable Tracking**: Handle campaigns with thousands of files efficiently
- **API Integration**: Other scenarios and agents can programmatically manage campaigns

## Key Features
- **Campaign Management**: Create named campaigns with glob pattern file selection
- **Status Tracking**: Track each file as pending/in-progress/completed/failed/skipped
- **Progress Persistence**: All state survives system restarts and conversation boundaries
- **REST API**: Full programmatic access for automation and integration
- **CLI Interface**: Command-line tools for script and agent integration
- **Web Dashboard**: Visual campaign management and progress monitoring

## UX Style
**Technical Professional** - Clean, information-dense dashboard inspired by GitHub Actions and GitLab CI interfaces. Dark theme with clear progress indicators and status visualization optimized for developer workflows.

## Scenario Dependencies
- **Database**: PostgreSQL for campaign and file tracking persistence
- **Core Resources**: File system access for glob pattern resolution

## Use Cases
- **Large Refactoring**: Track TypeScript migration across hundreds of files
- **Code Standardization**: Apply linting fixes or style guide changes systematically  
- **Feature Rollout**: Add new capabilities across multiple components
- **Documentation Updates**: Generate or update docs across entire codebases
- **Testing Initiatives**: Add tests to previously untested files
- **Security Updates**: Apply security patterns across all relevant files

## Quick Start

### Create a Campaign
```bash
# Track TypeScript migration
mass-update-tracker create "migrate-to-ts" "src/**/*.js" "tests/**/*.js" \
  --description "Convert JavaScript files to TypeScript" \
  --scenario "typescript-migration" \
  --working-dir "/path/to/project"
```

### Track Progress
```bash
# See pending files
mass-update-tracker files abc-123 --pending

# Update file status  
mass-update-tracker update-file abc-123 src/main.js in_progress
mass-update-tracker update-file abc-123 src/main.js completed

# Check overall progress
mass-update-tracker status --campaign-id abc-123
```

### API Integration
```bash
# Get pending files for automated processing
curl "http://localhost:20251/api/v1/campaigns/abc-123/files?status=pending"

# Update file status programmatically
curl -X PATCH "http://localhost:20251/api/v1/campaigns/abc-123/files/by-path/src%2Fmain.js/status" \
  -H "Content-Type: application/json" \
  -d '{"status": "completed"}'
```

## Claude Code Integration Pattern

This scenario is specifically designed for Claude Code workflows:

```markdown
1. Agent creates campaign: `mass-update-tracker create "fix-imports" "src/**/*.ts"`
2. Agent queries pending files: `mass-update-tracker files abc-123 --pending --json`  
3. Agent processes files in batches, updating status after each
4. Conversation restarts due to context limits
5. New agent checks progress: `mass-update-tracker status --campaign-id abc-123`
6. Agent resumes with remaining files seamlessly
```

## Integration Points
- **Other Scenarios**: Any scenario needing mass file operations can leverage campaigns
- **Monitoring**: Track progress of complex multi-step operations
- **Automation**: Integrate with CI/CD for staged rollouts
- **Analytics**: Historical data enables optimization of batch operation strategies

## Revenue Potential  
- **Enterprise Developer Teams**: $15K-25K per deployment
- **Code Migration Services**: $5K-15K per project
- **DevOps Automation**: $10K-20K for CI/CD integration
- **SaaS Platform Integration**: $500-2K/month for API access

## Future Enhancements
- Git integration for tracking uncommitted changes
- Campaign templates and preset patterns  
- Webhook notifications and external tool integration
- Multi-repository campaign support
- AI-powered operation scheduling based on file dependencies
- Collaborative campaigns for team coordination

## Technical Notes
- **Lightweight API**: Go-based for fast response times and low resource usage
- **Efficient Storage**: PostgreSQL with optimized indexes for large file sets
- **Glob Patterns**: Full Go filepath.Glob support for flexible file selection
- **Atomic Updates**: Database transactions ensure data consistency
- **Auto Status Updates**: Campaign status automatically updates based on file completion
- **Performance**: Handles 10K+ files per campaign efficiently