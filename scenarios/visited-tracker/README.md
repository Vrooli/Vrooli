# Visited Tracker

Persistent file visit tracking with staleness detection that maintains state across Claude Code conversations, AI models, and agent iterations. Essential infrastructure for systematic code analysis with intelligent prioritization.

## Core Value

This scenario adds **persistent visit tracking and staleness detection** to Vrooli, enabling:
- Comprehensive code coverage guarantees for security audits and bug hunts
- Intelligent prioritization based on visit frequency and modification patterns  
- Cross-model memory - any AI can see what others have analyzed
- Prevention of "code rot" by surfacing frequently modified but rarely reviewed files

## Features

✅ **Visit Tracking**: Count and timestamp every file visit with context
✅ **Staleness Detection**: Calculate risk scores based on modifications vs visits
✅ **Structure Sync**: Handle file additions, deletions, and moves automatically
✅ **Smart Prioritization**: Surface least visited and most stale files
✅ **Coverage Reports**: Track analysis progress with percentages and heatmaps
✅ **Import/Export**: Share and backup visited maps across systems

## Quick Start

```bash
# Run the scenario
vrooli scenario run visited-tracker

# Record file visits
visited-tracker visit "src/**/*.ts" --context security

# Get least visited files
visited-tracker least-visited --limit 10

# Get most stale files (high risk)
visited-tracker most-stale --threshold 5.0

# Check coverage
visited-tracker coverage --group-by directory

# Sync file structure
visited-tracker sync --patterns "**/*.js" --remove-deleted
```

## API Access

- **Health**: http://localhost:20252/health
- **Visit Recording**: http://localhost:20252/api/v1/visit
- **Prioritization**: http://localhost:20252/api/v1/prioritize/*
- **Dashboard**: http://localhost:3252

## Staleness Formula

```
staleness_score = (modifications_since_last_visit × time_since_last_visit) / (visit_count + 1)
```

High staleness = frequently modified but rarely visited = **HIGH BUG RISK**

## Architecture

- **Go API**: High-performance REST API with staleness calculation
- **CLI**: Comprehensive tool for all tracking operations
- **UI**: Heatmap dashboard with coverage visualization
- **Database**: PostgreSQL with optimized indexes for prioritization

## Use Cases

### Security Auditing
```bash
# Start security audit campaign
visited-tracker sync --patterns "**/*.{js,ts,go}"
visited-tracker least-visited --context security --limit 50

# After reviewing files
visited-tracker visit file1.js file2.ts --context security --agent "claude"
```

### Bug Hunting
```bash
# Find high-risk files (stale + critical)
visited-tracker most-stale --patterns "src/core/**" --limit 20

# Track debugging progress
visited-tracker visit src/auth/login.js --context bug --agent "gpt-4"
```

### Performance Analysis
```bash
# Identify unproﬁled hotspots
visited-tracker least-visited --context performance --patterns "**/*service*.go"

# Check coverage
visited-tracker coverage --context performance
```

## Integration

Other scenarios leverage visited-tracker via:
- **REST API** for programmatic tracking
- **CLI** for script integration
- **Events** for real-time staleness alerts

### Example Integration
```javascript
// Security scanner using visited-tracker
const unvisited = await fetch('http://localhost:20252/api/v1/prioritize/least-visited', {
  method: 'GET',
  body: JSON.stringify({
    context: 'security',
    limit: 100
  })
});

// Scan unvisited files first
for (const file of unvisited.files) {
  await scanFile(file.absolute_path);
  await recordVisit(file.file_path, 'security');
}
```

## Coverage Visualization

The dashboard provides:
- **Heatmap View**: Visual representation of visit frequency
- **Staleness Alerts**: Red zones for high-risk files
- **Coverage Badges**: Percentage tracked per directory
- **Timeline**: Visit history and modification patterns

## Why This Matters

Without visited-tracker:
- Files get forgotten during reviews
- Bugs hide in rarely-visited code
- Multiple agents duplicate work
- No way to prove comprehensive coverage

With visited-tracker:
- **100% coverage guarantee** possible
- **Risk-based prioritization** surfaces problems early
- **Cross-agent coordination** prevents duplicate work
- **Audit trail** proves compliance and thoroughness

This becomes a **foundational capability** that every analysis scenario builds upon, creating Vrooli's persistent memory for code intelligence.