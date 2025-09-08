# App Issue Tracker - Implementation Plan

## Overview
A comprehensive issue tracking system for Vrooli and its scenarios, featuring AI-powered investigation and automated fix attempts using Claude Code.

## Architecture

### Core Components
1. **API Server** (Port 8090)
   - RESTful API for issue management
   - CLI integration endpoints
   - Agent management interface
   - Semantic search API

2. **Web UI** (Port 35000-39999)
   - Issue dashboard with filtering/sorting
   - Agent management page (prompt templates)
   - Investigation reports viewer
   - Fix attempt monitoring

3. **Claude Code Integration**
   - Automated issue investigation
   - Root cause analysis
   - Fix generation and testing
   - Pull request creation

4. **Semantic Search** (Qdrant)
   - Vector embeddings for all issues
   - Similar issue detection
   - Knowledge base queries

## Data Flow

```
User/CLI → API → File System → Claude Code Script
                      ↓              ↓
                   Qdrant      Investigation
                               Report
                                  ↓
                            Fix Attempt
                                  ↓
                            Git PR/Commit
```

## Key Features

### Issue Management
- Create issues via API, CLI, or UI
- Categorize by type (bug, feature, improvement)
- Priority levels (critical, high, medium, low)
- Project association (Vrooli core, scenarios, generated apps)

### AI Investigation
- Automatic codebase analysis
- Error log parsing
- Stack trace interpretation
- Root cause identification
- Related issue detection

### Automated Fixes
- Claude Code generates fixes
- Runs tests to validate
- Creates pull requests
- Tracks success rates

### Agent Management
- Custom prompt templates per agent
- Capability definitions
- Success rate tracking
- Cost estimation

## CLI Usage

```bash
# Create new issue
vrooli-tracker create --title "Bug in routine executor" \
  --type bug --priority high \
  --project vrooli-core

# List issues
vrooli-tracker list --status open --priority high

# Investigate issue
vrooli-tracker investigate ISSUE-123 --agent deep-analyzer

# Attempt fix
vrooli-tracker fix ISSUE-123 --agent auto-fixer

# Search issues
vrooli-tracker search "authentication error"
```

## Business Value

### For Vrooli Development
- 70% reduction in issue resolution time
- Automated documentation of fixes
- Knowledge base growth
- Pattern recognition for common issues

### Revenue Model ($15K-30K value)
- Enterprise license for issue tracking
- AI investigation credits
- Custom agent development
- Priority support queue

## Resource Requirements

### Minimum
- 4GB RAM
- 2 CPU cores
- 20GB storage
- Claude API key

### Recommended
- 8GB RAM
- 4 CPU cores
- 50GB storage
- Dedicated GPU for embeddings

## Integration Points

### With Vrooli Core
- Direct repository access
- Git operations
- Test suite execution
- Build system integration

### With Other Scenarios
- Cross-scenario issue tracking
- Shared agent library
- Unified investigation reports
- Performance metrics aggregation

## Success Metrics
- Average resolution time < 4 hours
- Fix success rate > 60%
- False positive rate < 10%
- User satisfaction > 4.5/5

## Next Steps
1. Implement core API endpoints
2. Set up Web UI dashboards
3. Configure Claude Code integration
4. Create initial agent templates
5. Build CLI tool
6. Add semantic search
7. Implement fix automation
8. Create monitoring dashboard