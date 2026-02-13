# Job-to-Scenario Pipeline

## 🚀 Overview

An autonomous business development engine that converts job board opportunities into revenue-generating scenarios. This system harvests market demand from job boards (starting with Upwork), evaluates feasibility using existing Vrooli capabilities, generates new scenarios for approved opportunities, and produces ready-to-submit proposals.

## 💡 Key Features

- **Multi-source Job Import**: Upwork RSS feeds, screenshot OCR, manual entry
- **Intelligent Research**: Automated feasibility assessment using AI
- **State Management**: File-based tracking through pipeline states
- **Scenario Generation**: Automatic creation of new capabilities as needed
- **Proposal Generation**: AI-powered proposal writing with pricing
- **Trello-like Dashboard**: Professional UI for pipeline visualization

## 🏗️ Architecture

- **No N8N Workflows**: Direct API/CLI integration for maximum speed
- **File-based Storage**: YAML files organized by state for transparency
- **PostgreSQL**: Optional structured data storage and analytics
- **Direct Resource Access**: Fast integration with Ollama, Browserless, etc.

## 📦 Components

### API (Go)
- RESTful endpoints for job management
- Direct resource integration
- Async research and building processes

### CLI
- `job-to-scenario` command for all operations
- Import, research, approve, and generate proposals

### UI
- Trello-style kanban board
- Drag-and-drop screenshot upload
- Real-time pipeline visualization

### Data Management
- File-based state tracking in `data/` directory
- States: pending → researching → evaluated → approved → building → completed

## 🚀 Quick Start

### Setup
```bash
# From scenario directory
cd scenarios/job-to-scenario-pipeline

# Install CLI
./cli/install.sh

# Build API
cd api && go build -o job-to-scenario-pipeline-api .
```

### Run Services
```bash
# Start API
cd api && ./job-to-scenario-pipeline-api

# Start UI (in another terminal)
cd ui && npm install && npm start

# Dashboard will be at http://localhost:35500
```

### Basic Usage

#### Import Jobs
```bash
# Manual import
job-to-scenario import manual --text "Build a React dashboard for analytics"

# Screenshot import
job-to-scenario import screenshot --file job-screenshot.png

# Upwork feeds are imported automatically via Huginn
```

#### Process Pipeline
```bash
# Research pending jobs
job-to-scenario research --batch-size 5

# List evaluated jobs
job-to-scenario list --state evaluated

# Approve a recommended job
job-to-scenario approve JOB-20250106-123456-001

# Generate proposal for completed job
job-to-scenario proposal JOB-20250106-123456-001
```

#### File Management
```bash
# Check pipeline status
./data/manage.sh status

# View specific job
./data/manage.sh view JOB-20250106-123456-001

# Search for jobs
./data/manage.sh search "React"

# Archive old jobs
./data/manage.sh archive 30
```

## 📊 Pipeline States

1. **Pending**: Newly imported jobs awaiting research
2. **Researching**: AI analyzing feasibility
3. **Evaluated**: Research complete with recommendation
4. **Approved**: Approved for scenario building
5. **Building**: Scenarios being generated
6. **Completed**: Ready for proposal generation

## 🔬 Research Evaluations

- **RECOMMENDED**: Good fit, feasible with our capabilities
- **NOT_RECOMMENDED**: Too complex or out of scope
- **ALREADY_DONE**: We have existing scenarios that solve this
- **NO_ACTION**: Not relevant or insufficient information

## 🎯 Value Proposition

### Business Impact
- Converts job boards into continuous revenue pipeline
- $10K-$50K typical value per delivered scenario
- 10-20 hours/week saved on manual evaluation

### Intelligence Amplification
- Each job processed improves pattern recognition
- Successful proposals train better proposals
- Market demand drives capability expansion

## 🔄 Recursive Learning

The system creates a feedback loop:
1. Jobs reveal market needs
2. Scenarios are built to meet needs
3. Proposals win contracts
4. Revenue funds more capabilities
5. More capabilities enable bigger contracts

## 🛠️ Configuration

### Environment Variables
```bash
API_PORT=15500      # API server port
UI_PORT=35500       # Dashboard UI port
```

### Huginn Agent
The Upwork scraper in `initialization/automation/huginn/upwork-scraper.json` needs to be configured in your Huginn instance with appropriate RSS feed URLs for your target job categories.

## 📝 File Structure

```
job-to-scenario-pipeline/
├── .vrooli/
│   └── service.json          # Service configuration
├── api/
│   ├── main.go              # Go API server
│   └── go.mod               # Go dependencies
├── cli/
│   ├── job-to-scenario      # CLI tool
│   └── install.sh           # CLI installer
├── data/
│   ├── pending/             # New jobs
│   ├── researching/         # Jobs being analyzed
│   ├── evaluated/           # Jobs with research complete
│   ├── approved/            # Jobs approved for building
│   ├── building/            # Jobs being built
│   ├── completed/           # Finished jobs
│   ├── archive/             # Old jobs
│   ├── templates/           # Job templates
│   └── manage.sh            # File management script
├── initialization/
│   └── storage/
│       └── postgres/       # Database schema
├── ui/
│   ├── index.html          # Dashboard UI
│   ├── styles.css          # Dashboard styles
│   ├── app.js              # Dashboard logic
│   └── server.js           # Express server
├── PRD.md                  # Product requirements
└── README.md               # This file
```

## 🚨 Troubleshooting

### API Not Running
```bash
# Check if port is in use
lsof -i :15500

# Run API manually
cd api && go run main.go
```

### Jobs Not Importing
- Check API health: `curl http://localhost:15500/health`
- Verify Huginn agent is active
- Check file permissions in `data/` directory

### Research Not Working
- Ensure Ollama is running: `resource-ollama status`
- Check resource availability
- Review API logs for errors

## 🔮 Future Enhancements

- Multi-platform support (Fiverr, Freelancer.com)
- ML-based pricing optimization
- Automated client communication
- Success rate analytics dashboard
- Portfolio generation from completed work

---

**Remember**: Every job processed makes Vrooli smarter, every scenario built expands capabilities, and every successful proposal generates revenue that funds further growth. This is recursive improvement in action!
