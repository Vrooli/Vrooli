# Scenario Dependency Analyzer

A meta-intelligence capability for analyzing, visualizing, and optimizing scenario and resource dependencies within Vrooli.

## 🎯 Purpose

This scenario provides Vrooli with **architectural self-awareness** by:

- **Analyzing existing scenarios** to extract resource, declared scenario, and actual import-level inter-scenario dependencies
- **Visualizing dependency graphs** with interactive, technical interface
- **Detecting declared-vs-actual drift** between `service.json` and real proto/Go import evidence
- **Exposing an interface graph seam** for `tech-tree-designer`, `test-genie`, and future planning scenarios
- **Predicting dependencies** for proposed scenarios using AI analysis
- **Optimization recommendations** for resource usage and deployment strategies
- **Enabling surgical deployments** with minimal resource footprint

## 🚀 Quick Start

### Prerequisites
- SQLite storage managed by the scenario lifecycle
- `proto-health` available for protobuf surface facts
- `code-facts` available for import facts
- Ollama resource available (`resource-ollama`)
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

# Show deployment readiness for a scenario
scenario-dependency-analyzer deployment chart-generator

# Export recursive dependency DAG
scenario-dependency-analyzer dag export chart-generator --recursive

# Scan and optionally apply inferred dependencies
scenario-dependency-analyzer scan chart-generator --apply

# Analyze all scenarios (may take several minutes)
scenario-dependency-analyzer analyze all

# Generate declared dependency graph
scenario-dependency-analyzer graph combined --format json

# Generate actual interface graph through the Connect-backed seam
scenario-dependency-analyzer graph actual --json

# Report declared-vs-actual scenario dependency drift
scenario-dependency-analyzer drift scenario-dependency-analyzer --json

# Validate dependency health through the producer contract
scenario-dependency-analyzer health scenario-dependency-analyzer --json

# Search approved dependency governance memory
scenario-dependency-analyzer deps approved search "React graph library" --json

# Validate a scenario against approved dependency governance memory
scenario-dependency-analyzer deps approved validate scenario-dependency-analyzer --json

# Review grouped fleet governance decisions and one dependency's usage
scenario-dependency-analyzer deps approved triage
scenario-dependency-analyzer deps approved findings --severity WARNING --json
scenario-dependency-analyzer deps approved usage npm/react --json
scenario-dependency-analyzer deps approved approve-observed npm/react --from-findings --json
scenario-dependency-analyzer deps approved widen-range npm/react --to-major-line --json

# Preview a governance approval; add --apply to write the registry
scenario-dependency-analyzer deps approved approve npm/react --range "18.2.0" --range-policy major_line --rationale "Approved UI runtime framework." --json

# Preview a governance denial; add --apply to write the registry
scenario-dependency-analyzer deps approved deny npm/left-pad --reason "Use native string padding." --replacement "String.prototype.padStart/padEnd" --json

# Preview a Security Health-derived vulnerability remediation; add --apply to deny-vulnerable to write the registry
scenario-dependency-analyzer deps approved security-gaps --minimum-severity high
scenario-dependency-analyzer deps approved remediate npm/vite --vulnerability GHSA-example --json
scenario-dependency-analyzer deps approved deny-vulnerable npm/vite --vulnerability GHSA-example --json

# Analyze proposed scenario
scenario-dependency-analyzer propose "AI-powered task scheduler with database storage"

# Get help
scenario-dependency-analyzer help
```

#### Web Interface
Access the interactive dependency graph visualization and catalog at the `UI_PORT`
reported by `vrooli scenario status scenario-dependency-analyzer`.

Features:
- Real-time dependency graph visualization
- Interactive node selection and filtering
- Scenario catalog panel with last-scan status
- Detail view showing declared vs detected dependencies with drift badges
- Deployment readiness panel with tier fitness, blockers, and bundle manifest insight
- Governance view for grouped fleet triage, dependency findings, dependency usage groups, reviewed records, dry-run approval/denial, batch proposal/upsert workflows, normalized direct/dev/indirect/security signal classes, and Security Health-derived security-gap/denied-range remediation
- One-click scan and scan+apply actions per scenario
- System statistics and health monitoring
- Export functionality for graphs
- Technical NASA mission control aesthetic

#### API Endpoints
```bash
API_PORT=$(vrooli scenario port scenario-dependency-analyzer API_PORT)

# List scenarios + metadata
curl "http://localhost:${API_PORT}/api/v1/scenarios"

# Get stored detail for a scenario
curl "http://localhost:${API_PORT}/api/v1/scenarios/chart-generator"

# Trigger scan (set apply=true to update service.json automatically)
curl -X POST "http://localhost:${API_PORT}/api/v1/scenarios/chart-generator/scan" \
  -H "Content-Type: application/json" \
  -d '{"apply":false}'

# Get scenario dependencies
curl "http://localhost:${API_PORT}/api/v1/scenarios/chart-generator/dependencies"

# Generate declared dependency graph
curl "http://localhost:${API_PORT}/api/v1/graph/combined"

# Describe actual interface graph through Connect RPC
# /vrooli.scenario_dependency_analyzer.v1.graph.InterfaceGraphService/DescribeInterfaceGraph

# Analyze proposed scenario
curl -X POST "http://localhost:${API_PORT}/api/v1/analyze/proposed" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","description":"Database-driven AI scenario","requirements":["postgres"]}'

# Export recursive DAG
curl "http://localhost:${API_PORT}/api/v1/scenarios/chart-generator/dag/export?recursive=true"
```

**Dependency Deployment Features**:
- **📊 Recursive DAG Export**: Full dependency tree with metadata gaps via `/dag/export` endpoint and `dag export` CLI command
- **🔍 Metadata Gap Analysis**: Automatic detection of missing deployment metadata across dependency trees
- **📚 Comprehensive Documentation**: See `docs/api.md` and `docs/integration.md` for full API reference and integration patterns

## What You Get

- Actual interface graph assembly from `proto-health` proto facts and `code-facts` import facts.
- Declared-vs-actual drift reporting for scenario dependencies.
- Dependency-health producer contract for Test Genie and future agents, including Code Facts-backed surface inventory, dependency readiness checks, runtime dependency status, approved dependency governance, pnpm release-age policy validation, security-health dependency-index availability, and graph drift.
- Approved dependency governance records exposed through a generated Connect contract, CLI, and Governance UI. These records are review memory, not an exhaustive allowlist; agents may suggest better unrecorded dependencies with purpose, version/range, range policy, alternatives, and security/license notes.
- REST, Connect, CLI, and UI surfaces for operators and downstream scenarios.

## Documentation Map

- Start with `docs/START-HERE.md` for orientation.
- Use `docs/reference/api-endpoints.md` and `docs/reference/cli-commands.md` for stable interface lookup.
- Use `docs/concepts/ARCHITECTURE.md`, `docs/concepts/DOMAINS.md`, and `docs/concepts/INTEGRATIONS.md` for implementation context.
- Use `docs/internal/SEAMS.md` and `docs/internal/TESTING.md` before changing boundaries or tests.

## Customize Safely

- Keep source extraction in upstream fact scenarios; SDA interprets fleet facts.
- Add generated proto/Connect surfaces before introducing new programmatic API contracts.
- Update `.vrooli/service.json`, docs, and requirements together when dependencies or operational targets change.

## 🔧 Architecture

### Components
- **Go API Server** (`api/`) - Core dependency analysis and graph generation
- **CLI Tool** (`cli/`) - Command-line interface for analysis operations
- **Web UI** (`ui/`) - Interactive dependency graph visualization
- **SQLite Storage** (`api/internal/<domain>/schema.sql`) - Domain-owned schemas for history-bearing metadata
- **Interface Graph Domain** (`api/internal/interfacegraph/`) - On-demand graph assembly from proto-health and code-facts facts

### Resource Dependencies
- **sqlite** (required) - Stores dependency metadata, recommendations, and analysis history
- **proto-health** (required scenario) - Provides batch proto surface and cross-scenario import evidence
- **code-facts** (required scenario) - Provides batch language-level import facts
- **qdrant** (required) - Semantic similarity matching for scenario patterns
- **redis** (optional) - Performance optimization through result caching

### Key Features

#### Dependency Detection
- **Resource Dependencies**: Extracted from `.vrooli/service.json` files
- **Declared Scenario Dependencies**: Extracted from `.vrooli/service.json` files
- **Actual Scenario Dependencies**: Computed from `proto_import` and `go_import` evidence supplied by upstream fact scenarios
- **Shared Workflows**: Identified through initialization file analysis

#### Drift Detection
- **Undeclared but used**: import-level evidence exists but the dependency is not declared; reported as `WARNING`
- **Declared without import evidence**: declaration exists but no proto/Go import evidence was found; reported as `INFO` because runtime URL and CLI shell-out facts are deferred to AST analyzers

#### AI-Powered Analysis
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
API_PORT=$(vrooli scenario port scenario-dependency-analyzer API_PORT)
curl "http://localhost:${API_PORT}/health"
```

## 🤝 Integration with Other Scenarios

This scenario is designed to be consumed by:
- **deployment-optimizer** - For surgical deployment configurations
- **capability-planner** - For strategic scenario development planning
- **swarm-manager** - For dependency prediction in generated scenarios
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
- Uses SQLite for reliable local metadata storage
- Leverages Qdrant for semantic similarity matching
- Provides REST, Connect, CLI, and visual interfaces
- Designed for horizontal scaling and distributed analysis

---

**This scenario represents a fundamental capability that makes every other scenario in Vrooli more intelligent and deployable.**
