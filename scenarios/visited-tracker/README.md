# Visited Tracker

Persistent file visit tracking with staleness detection for systematic code analysis

## 🎯 Purpose & Vision

A systematic file visit tracking tool that enables agent loops to maintain perfect memory across conversations. Designed specifically for ecosystem-manager phases (Progress, UX, Refactor, Test) to ensure comprehensive coverage without redundant work. Stores phase-specific metadata for seamless handoff between analysis modes, ensuring no file is left behind during systematic multi-file workflows.

## 🏗️ Architecture

### Campaign Management
Organizes file tracking into campaigns, each targeting specific directories and file patterns for focused analysis projects.

### Visit Tracking System
Records every file access with timestamps, building a comprehensive view of which files have been analyzed and how recently.

### Staleness Detection
Calculates staleness scores based on file modification time, visit frequency, and time since last visit to prioritize neglected files.

## 🎨 UI Style

**Clean Developer Tool** - A minimal, data-focused interface optimized for developer workflows
- Light theme with blue accents for clear data visualization
- Tabular views for file lists and visit statistics with sortable columns
- Progress indicators showing campaign completion percentages
- Minimalist design prioritizing information density over aesthetics
- Responsive layout that works well in split-screen development setups

## 🔧 Key Features

### Campaign-Based Organization
Create focused tracking campaigns for different analysis projects with custom file patterns and directory targets.

### Intelligent Prioritization
Get recommended files to visit next based on staleness scoring, ensuring systematic coverage of entire codebases.

### Comprehensive Visit Tracking
Record and visualize file visit patterns with detailed statistics on coverage and frequency.

## 🚀 Getting Started

### Prerequisites
- Go 1.19+ for API server
- Basic file system access for tracking

### Installation
```bash
# Setup the scenario
vrooli scenario setup visited-tracker

# Start the services
vrooli scenario start visited-tracker
```

### Agent Integration (Recommended)

**Auto-creation shorthand** - no manual campaign management needed:

```bash
# UX Phase: Track UI work with automatic campaign creation
visited-tracker least-visited \
  --location scenarios/prd-control-tower/ui \
  --pattern "**/*.{ts,tsx}" \
  --tag ux \
  --limit 5

# Returns: 5 least-visited UI files with staleness scores
# Campaign auto-created if none exists for (location="scenarios/prd-control-tower/ui", tag="ux")
```

**Recording work with notes:**
```bash
# After analyzing orientation.tsx
visited-tracker visit scenarios/prd-control-tower/ui/src/pages/orientation.tsx \
  --tag ux \
  --note "Improved Getting Started section and hero CTA. Need a Nudge component still needs work."

# After completing file
visited-tracker exclude scenarios/prd-control-tower/ui/src/pages/orientation.tsx \
  --tag ux \
  --reason "All major UX improvements complete"
```

**Campaign handoff:**
```bash
# Store handoff context for next phase
visited-tracker campaign note \
  --location scenarios/prd-control-tower/ui \
  --tag ux \
  --note "UX review complete. Focus areas for refactor: complex state management in DraftEditor, performance issues in tree rendering."
```

### Agent-Friendly Response Example

```json
{
  "files": [
    {
      "path": "scenarios/prd-control-tower/ui/src/pages/orientation.tsx",
      "staleness_score": 8.3,
      "visit_count": 0,
      "last_visited": null,
      "last_modified": "2025-11-24T10:30:00Z",
      "notes": []
    },
    {
      "path": "scenarios/prd-control-tower/ui/src/components/catalog/ScenarioDetail.tsx",
      "staleness_score": 7.1,
      "visit_count": 1,
      "last_visited": "2025-11-20T14:00:00Z",
      "last_modified": "2025-11-23T09:15:00Z",
      "notes": ["Improved navigation but nested content still needs work"]
    }
  ],
  "campaign": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "location": "scenarios/prd-control-tower/ui",
    "tag": "ux",
    "total_files": 47,
    "visited_files": 12,
    "coverage_percent": 25.5
  }
}
```

## 📝 Phase Usage Patterns

### UX Phase - Systematic UI Improvement

```bash
# Get next files to review
visited-tracker least-visited \
  --location scenarios/X/ui \
  --pattern "**/*.tsx" \
  --tag ux \
  --limit 3

# After improving each file
visited-tracker visit src/pages/dashboard.tsx \
  --tag ux \
  --note "Improved heading hierarchy, added ARIA labels"

# Mark exceptional files
visited-tracker exclude src/pages/home.tsx \
  --tag ux \
  --reason "Already exceptional - no improvements needed"
```

### Refactor Phase - Code Quality Improvements

```bash
# Target both API and UI code
visited-tracker least-visited \
  --location scenarios/X \
  --pattern "{api,ui}/**/*.{go,ts}" \
  --tag refactor \
  --limit 5

# Track refactoring work
visited-tracker visit api/handlers.go \
  --tag refactor \
  --note "Extracted validation logic to separate package. Still need error handling cleanup."
```

### Test Phase - Coverage Expansion

```bash
# Find least-tested files
visited-tracker least-visited \
  --location scenarios/X/api \
  --pattern "**/*_test.go" \
  --tag test \
  --limit 10

# Record test additions
visited-tracker visit api/handlers_test.go \
  --tag test \
  --note "Added edge case tests for validation logic. Coverage now 87%."
```

### Progress Phase - Feature Implementation

```bash
# Track implementation files
visited-tracker least-visited \
  --location scenarios/X \
  --pattern "**/*.{go,ts,tsx}" \
  --tag progress \
  --limit 8

# Record progress
visited-tracker visit api/new_feature.go \
  --tag progress \
  --note "Implemented core logic. Need to add error handling and tests."
```

## 🤖 Prompt Integration Guide

### For Phase Instructions

**Example UX Phase Prompt Addition:**
```
MEMORY MANAGEMENT:
Use visited-tracker to maintain work history across conversations:

1. At start of each iteration:
   visited-tracker least-visited --location scenarios/{SCENARIO}/ui --pattern "**/*.tsx" --tag ux --limit 5

2. After analyzing each file:
   visited-tracker visit <file-path> --tag ux --note "<brief summary of work done and what remains>"

3. Mark files complete when exceptional:
   visited-tracker exclude <file-path> --tag ux --reason "All improvements complete"

4. Before ending session:
   visited-tracker campaign note --location scenarios/{SCENARIO}/ui --tag ux --note "<handoff context for next session>"
```

### Response Interpretation

Agents should prioritize files with:
- **High staleness_score (>7.0)**: Neglected files needing attention
- **Low visit_count (0-2)**: Files not yet analyzed
- **Recent last_modified**: Files changed since last visit
- **Notes mentioning incomplete work**: Files with known remaining tasks

## 🧪 Testing

Run the comprehensive test suite:
```bash
# Full test suite
vrooli scenario test visited-tracker

# Quick validation
vrooli scenario test visited-tracker quick

# Specific test phases
vrooli scenario test visited-tracker unit
vrooli scenario test visited-tracker integration
```

## 📡 API Reference

### Base URL
```
http://localhost:17694
```

### Key Endpoints
- `GET /health` - Health check
- `POST /api/v1/campaigns` - Create new campaign
- `GET /api/v1/campaigns` - List all campaigns
- `GET /api/v1/campaigns/{id}` - Get campaign details
- `POST /api/v1/campaigns/{id}/visit` - Record file visits
- `GET /api/v1/campaigns/{id}/least-visited` - Get least visited files
- `GET /api/v1/campaigns/{id}/most-stale` - Get most stale files

## 🎨 Web Interface

Access the web interface via the dynamically assigned port:

```bash
open "http://localhost:$(vrooli scenario port visited-tracker UI_PORT)"
```

Provides a visual dashboard for managing campaigns, viewing file visit statistics, monitoring staleness metrics, and tracking analysis progress across multiple projects.

## 🔗 Integration

### CLI Integration for Agents

```bash
# Zero-friction pattern - auto-creation with location + tag
visited-tracker least-visited \
  --location scenarios/X/ui \
  --pattern "**/*.tsx" \
  --tag ux \
  --limit 5

# Record work with contextual notes
visited-tracker visit src/pages/dashboard.tsx \
  --tag ux \
  --note "Improved accessibility and visual hierarchy"

# Phase handoff with campaign notes
visited-tracker campaign note \
  --location scenarios/X/ui \
  --tag ux \
  --note "UI improvements complete. Refactor phase: focus on state management complexity"

# Reset campaign for fresh analysis pass
visited-tracker campaign reset \
  --location scenarios/X/ui \
  --tag ux
```

### Programmatic Usage

Other scenarios and ecosystem-manager phases integrate with visited-tracker for systematic file analysis:
- **UX Phase**: Track UI file improvements with notes on component-level work
- **Refactor Phase**: Monitor code quality improvements across API and UI
- **Test Phase**: Ensure comprehensive test coverage across codebase
- **Progress Phase**: Track feature implementation files and completion status

## 📊 Monitoring

- **Health Check**: `GET /health`
- **Metrics**: Available through the web interface with campaign progress and file statistics
- **Logs**: `vrooli scenario logs visited-tracker`

## 🔧 Configuration

Configuration is managed through `.vrooli/service.json`:
- Port allocation: API (15000-19999), UI (35000-39999)
- Resource dependencies: Local file system, optional PostgreSQL
- Service lifecycle: Development and production modes

## 🤝 Contributing

This scenario follows Vrooli's standard development patterns:
- Go backend with comprehensive test coverage (79.4%)
- Modern web interface with responsive design
- CLI tools for automation and integration
- Phased testing architecture for quality assurance

## 📚 Documentation

- **API Documentation**: Available at `http://localhost:17694/docs`
- **Testing Guide**: See `api/TESTING_GUIDE.md`
- **Development Setup**: Follow the standard Vrooli development workflow
