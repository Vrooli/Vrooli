# Scenario Dependency Analyzer

A meta-intelligence capability for analyzing, visualizing, and optimizing scenario and resource dependencies within Vrooli.

## 🎯 Purpose

This scenario provides Vrooli with **architectural self-awareness** by:

- **Analyzing existing scenarios** to extract resource and inter-scenario dependencies
- **Visualizing dependency graphs** with interactive, technical interface
- **Predicting dependencies** for proposed scenarios using AI analysis
- **Optimization recommendations** for resource usage and deployment strategies
- **Enabling surgical deployments** with minimal resource footprint

## 🚀 Quick Start

### Prerequisites
- PostgreSQL resource running and accessible
- Claude Code resource available (`resource-claude-code`)
- Qdrant resource available (`resource-qdrant`)

### Setup
```bash
# Navigate to the scenario
cd scenarios/scenario-dependency-analyzer

# Set up the scenario (builds API, installs CLI, initializes database)
vrooli scenario run scenario-dependency-analyzer
```

### Usage

#### CLI Commands
```bash
# Check system status
scenario-dependency-analyzer status

# Analyze a specific scenario
scenario-dependency-analyzer analyze chart-generator

# Scan and optionally apply inferred dependencies
scenario-dependency-analyzer scan chart-generator --apply

# Analyze all scenarios (may take several minutes)
scenario-dependency-analyzer analyze all

# Generate dependency graph
scenario-dependency-analyzer graph combined --format json

# Analyze proposed scenario
scenario-dependency-analyzer propose "AI-powered task scheduler with database storage"

# Get help
scenario-dependency-analyzer help
```

#### Web Interface
Access the interactive dependency graph visualization and catalog at:
`http://localhost:20401`

Features:
- Real-time dependency graph visualization
- Interactive node selection and filtering
- Scenario catalog panel with last-scan status
- Detail view showing declared vs detected dependencies with drift badges
- One-click scan and scan+apply actions per scenario
- System statistics and health monitoring
- Export functionality for graphs
- Technical NASA mission control aesthetic

#### API Endpoints
```bash
# List scenarios + metadata
curl http://localhost:20400/api/v1/scenarios

# Get stored detail for a scenario
curl http://localhost:20400/api/v1/scenarios/chart-generator

# Trigger scan (set apply=true to update service.json automatically)
curl -X POST http://localhost:20400/api/v1/scenarios/chart-generator/scan \
  -H "Content-Type: application/json" \
  -d '{"apply":false}'

# Get scenario dependencies
curl http://localhost:20400/api/v1/scenarios/chart-generator/dependencies

# Generate dependency graph
curl http://localhost:20400/api/v1/graph/combined

# Analyze proposed scenario
curl -X POST http://localhost:20400/api/v1/analyze/proposed \
  -H "Content-Type: application/json" \
  -d '{"name":"test","description":"Database-driven AI scenario","requirements":["postgres"]}'
```

## 🔧 Architecture

### Components
- **Go API Server** (`api/`) - Core dependency analysis and graph generation
- **CLI Tool** (`cli/`) - Command-line interface for analysis operations
- **Web UI** (`ui/`) - Interactive dependency graph visualization
- **Database Schema** (`initialization/`) - PostgreSQL schema for metadata storage

### Resource Dependencies
- **postgres** (required) - Stores dependency metadata and analysis results
- **claude-code** (required) - AI-powered analysis of proposed scenarios
- **qdrant** (required) - Semantic similarity matching for scenario patterns
- **redis** (optional) - Performance optimization through result caching

### Key Features

#### Dependency Detection
- **Resource Dependencies**: Extracted from `.vrooli/service.json` files
- **Inter-scenario Dependencies**: Detected via code scanning for CLI calls
- **Shared Workflows**: Identified through initialization file analysis

#### AI-Powered Analysis
- **Claude Code Integration**: Intelligent analysis of proposed scenario descriptions
- **Qdrant Semantic Search**: Find similar existing scenarios and patterns
- **Heuristic Fallbacks**: Keyword-based predictions when AI resources unavailable

#### Graph Visualization
- **Interactive D3.js graphs** with zoom, pan, and node selection
- **Multiple graph types**: Resources, scenarios, or combined dependencies
- **Real-time filtering** and highlighting of connections
- **Export capabilities** for further analysis

## 🎨 Visual Design

The UI follows a **technical NASA mission control aesthetic**:
- Dark theme with green terminal-style text
- Matrix-style animated background
- Technical typography and layout
- Real-time system status indicators
- Professional dashboard design

## 💡 Use Cases

### For Deployment Optimization
```bash
# Analyze dependencies for minimal deployment
scenario-dependency-analyzer analyze your-scenario --output json

# Generate optimization recommendations
scenario-dependency-analyzer optimize your-scenario

# Apply safe dependency reductions automatically
scenario-dependency-analyzer optimize your-scenario --apply --type resource
```

### For Capability Planning
```bash
# Predict dependencies for proposed scenarios
scenario-dependency-analyzer propose "Your new scenario description"

# Find similar patterns in existing scenarios
scenario-dependency-analyzer analyze all | jq '.similar_patterns'
```

### For System Architecture
```bash
# Generate comprehensive dependency graph
scenario-dependency-analyzer graph combined --format html > deps.html

# Export for external analysis
scenario-dependency-analyzer graph resources --output-file resources.json
```

## 📊 Data Storage

### Database Tables
- `scenario_dependencies` - Individual dependency relationships
- `dependency_graphs` - Computed graph structures with metadata
- `optimization_recommendations` - AI-generated improvement suggestions
- `analysis_runs` - History of analysis operations
- `scenario_metadata` - Cached scenario information

### Dependency Types
- **resource** - Infrastructure dependencies (postgres, redis, etc.)
- **scenario** - Inter-scenario relationships and calls
- **shared_workflow** - Common workflows and patterns
- **cli_tool** - Command-line tool dependencies

## 🔄 Recursive Intelligence

This scenario embodies Vrooli's recursive improvement philosophy:

1. **Self-Analysis**: The scenario analyzes its own dependencies
2. **System-Wide Intelligence**: Every scenario becomes more deployable
3. **Compound Learning**: Each analysis improves future predictions
4. **Optimization Multiplication**: Recommendations apply across all scenarios

## 🧪 Testing

```bash
# Run scenario tests
vrooli scenario test scenario-dependency-analyzer

# Test specific components
./cli/scenario-dependency-analyzer analyze scenario-dependency-analyzer
curl http://localhost:20400/health
```

## 🤝 Integration with Other Scenarios

This scenario is designed to be consumed by:
- **deployment-optimizer** - For surgical deployment configurations
- **capability-planner** - For strategic scenario development planning
- **ecosystem-manager** - For dependency prediction in generated scenarios
- **resource-cost-analyzer** - For economic optimization analysis

## 📈 Value Proposition

- **40-70% reduction** in deployment resource usage through optimization
- **Accelerated development** through dependency insight and gap analysis
- **Architectural intelligence** that compounds with every new scenario
- **Strategic planning** capabilities for capability roadmaps

## 🛠️ Development

### Local Development
```bash
# Build API
cd api && go mod download && go build .

# Install CLI locally
cd cli && ./install.sh

# Start services
vrooli scenario run scenario-dependency-analyzer
```

### Architecture Notes
- Uses PostgreSQL for reliable dependency metadata storage
- Integrates with Claude Code for intelligent scenario analysis
- Leverages Qdrant for semantic similarity matching
- Provides both programmatic (API/CLI) and visual (Web) interfaces
- Designed for horizontal scaling and distributed analysis

---

**This scenario represents a fundamental capability that makes every other scenario in Vrooli more intelligent and deployable.**
