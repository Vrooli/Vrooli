# Visited Tracker

Persistent file visit tracking with staleness detection for systematic code analysis

## 🎯 Purpose & Vision

A systematic file visit tracking tool that enables maintenance scenarios to efficiently analyze large codebases over multiple conversations without losing track of progress. Ensures no file is left behind during comprehensive code reviews, testing, and analysis workflows.

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
- Node.js 16+ for web interface
- Basic file system access for tracking

### Installation
```bash
# Setup the scenario
vrooli scenario setup visited-tracker

# Start the services
vrooli scenario start visited-tracker
```

### Basic Usage
```bash
# Create a new campaign
visited-tracker create --name "api-analysis" --pattern "**/*.go" --from-agent "api-manager"

# Get files to visit next (least visited first)
visited-tracker least-visited --campaign "api-analysis" --limit 5

# Record visits after analyzing files
visited-tracker visit --campaign "api-analysis" --files "src/main.go,src/utils.go"

# Check campaign progress
visited-tracker status --campaign "api-analysis"
```

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
http://localhost:17695
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

Access the web interface at: `http://localhost:37695`

Provides a visual dashboard for managing campaigns, viewing file visit statistics, monitoring staleness metrics, and tracking analysis progress across multiple projects.

## 🔗 Integration

### CLI Integration
```bash
# Example: Get least visited files for systematic analysis
visited-tracker least-visited --campaign my-analysis --limit 10

# Example: Record file visits after analysis
visited-tracker visit --campaign my-analysis --files "src/*.go"

# Example: Export campaign data for reporting
visited-tracker export --campaign my-analysis --format json
```

### Programmatic Usage
Other scenarios can integrate with visited-tracker to manage systematic file analysis workflows:
- **api-manager**: Track API files across all scenarios for periodic review
- **test-genie**: Identify files that need test coverage attention
- **security-auditor**: Ensure all files are regularly reviewed for vulnerabilities

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

- **API Documentation**: Available at `http://localhost:17695/docs`
- **Testing Guide**: See `api/TESTING_GUIDE.md`
- **Development Setup**: Follow the standard Vrooli development workflow