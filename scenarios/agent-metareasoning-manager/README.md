# Agent Metareasoning Manager

> **A lightweight Go API that orchestrates AI reasoning workflows, providing structured decision-making patterns for Vrooli agents**

## 📝 **Overview**

This scenario enhances Vrooli's AI agents with sophisticated metareasoning capabilities—the ability to think about thinking. It provides a database-driven Go API that orchestrates complex reasoning workflows through n8n and Windmill, enabling agents to:

- Analyze decisions with multiple reasoning frameworks
- Self-review and improve their own reasoning
- Track performance metrics and learn from past decisions
- Share reasoning patterns across the entire Vrooli ecosystem

## ⚡ **Architecture** 

```mermaid
flowchart TB
    subgraph "Client Layer"
        CLI[CLI Client<br/>metareasoning]
        API_CLIENT[API Clients<br/>HTTP/REST]
    end
    
    subgraph "Coordination Layer"
        GO_API[Go Coordination API<br/>Database-Driven<br/>Port: API_PORT]
    end
    
    subgraph "Workflow Orchestration"
        N8N[n8n Workflows<br/>5 Reasoning Patterns]
        WINDMILL[Windmill Apps<br/>UI & Orchestration]
    end
    
    subgraph "AI Processing"
        OLLAMA[Ollama LLM<br/>Local Inference<br/>llama3.2, mistral, codellama]
    end
    
    subgraph "Data Storage"
        POSTGRES[(PostgreSQL<br/>Workflows, History<br/>Metrics, Versions)]
        QDRANT[(Qdrant<br/>Vector Embeddings<br/>Semantic Search)]
    end
    
    CLI -->|Request| GO_API
    API_CLIENT -->|Request| GO_API
    GO_API -->|Webhook| N8N
    GO_API -->|Job| WINDMILL
    GO_API <-->|CRUD| POSTGRES
    N8N -->|Inference| OLLAMA
    WINDMILL -->|Inference| OLLAMA
    N8N -->|Store| POSTGRES
    WINDMILL -->|Store| POSTGRES
    N8N -->|Embeddings| QDRANT
    WINDMILL -->|Embeddings| QDRANT
    
    style GO_API fill:#e1bee7,stroke:#333,stroke-width:2px
    style OLLAMA fill:#b3e5fc,stroke:#333,stroke-width:2px
    style N8N fill:#fff9c4,stroke:#333,stroke-width:2px
    style WINDMILL fill:#fff9c4,stroke:#333,stroke-width:2px
```

## 🚀 **Key Features**

- **Database-Driven**: PostgreSQL stores all workflows, no static JSON files
- **Full CRUD API**: Complete workflow management with versioning support
- **Dynamic Workflows**: Add, update, and manage workflows at runtime
- **5+ Built-in Patterns**: Pros/cons, SWOT, risk assessment, decision analysis, self-review
- **Execution Tracking**: Every execution logged with performance metrics
- **Thin CLI**: Minimal 174-line wrapper (down from 690 lines)
- **Multiple Interfaces**: REST API, CLI tool, and Windmill UI dashboards
- **Auto-Metrics**: Database triggers automatically update statistics
- **Authentication**: Token-based API security

## 📋 **Available Analysis Types**

| Pattern | Description | Use Case |
|---------|-------------|----------|
| **Pros/Cons** | Weighted advantages vs disadvantages | Feature decisions, trade-offs |
| **SWOT** | Strengths, Weaknesses, Opportunities, Threats | Strategic planning, market analysis |
| **Risk Assessment** | Probability × Impact with mitigations | Project planning, change management |
| **Decision Analysis** | Multi-factor scoring with recommendations | Complex choices, vendor selection |
| **Self-Review** | Iterative reasoning validation | Critical decisions, quality assurance |

## 🛠️ **Quick Start**

### **Prerequisites**
```bash
# Ensure required Ollama models are available
ollama pull llama3.2          # Primary reasoning model
ollama pull mistral           # Alternative model for comparison
ollama pull codellama         # Code-specific reasoning
ollama pull nomic-embed-text  # Embedding model for semantic search

# Verify models are downloaded
ollama list | grep -E "llama3.2|mistral|codellama|nomic-embed-text"
```

### **Setup**
```bash
# Navigate to the scenario directory
cd /path/to/vrooli/scenarios/agent-metareasoning-manager

# Run setup lifecycle phase (with target environment specification)
../../manage.sh setup --target native-linux

# This automatically executes the steps defined in .vrooli/service.json:
# - Executes generic setup (network, system, git)
# - Adds app data to resources  
# - Installs CLI command globally as 'metareasoning'
# - Builds the 8MB Go coordination API binary
# - Displays service access URLs with dynamic ports
```

### **Usage**
```bash
# Start the development environment
../../manage.sh develop --target native-linux

# The 'develop' phase starts:
# - Go API server on dynamic $API_PORT
# - Connects to n8n on ${RESOURCE_PORTS[n8n]} 
# - Connects to Windmill on ${RESOURCE_PORTS[windmill]}

# Use the CLI (after setup)
agent-metareasoning-manager health                                    # Check system status
agent-metareasoning-manager api                                       # Show API info
agent-metareasoning-manager analyze pros-cons "Remote work policy"   # Run analysis
agent-metareasoning-manager list                                      # Show available workflows
agent-metareasoning-manager version                                   # Show CLI version
```

### **API Direct Usage**
```bash
# Get the actual ports from the running system
API_PORT=$(../../resources/port_registry.sh --action get --service agent-metareasoning-manager)
N8N_PORT=$(../../resources/port_registry.sh --action get --service n8n)
WINDMILL_PORT=$(../../resources/port_registry.sh --action get --service windmill)

# Health check
curl http://localhost:$API_PORT/health

# List workflows
curl http://localhost:$API_PORT/workflows

# Run analysis (requires authentication)
curl -X POST http://localhost:$API_PORT/analyze/pros-cons \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer agent_metareasoning_manager_cli_default_2024" \
  -d '{"input": "Should we migrate to microservices?", "context": "Legacy monolith"}'
```

### **Available Lifecycle Commands**
All lifecycle phases are defined in `.vrooli/service.json`:
```bash
../../manage.sh setup    # Initialize scenario and build components
../../manage.sh develop  # Start development environment  
../../manage.sh test     # Run scenario tests
../../manage.sh stop     # Stop all scenario services

# Each command accepts options:
../../manage.sh <phase> --target <environment>  # native-linux, docker, k8s, etc.
../../manage.sh <phase> --help                  # Show phase-specific options
```

## 📁 **Project Structure**

```
agent-metareasoning-manager/
├── api/                           # Go coordination API
│   ├── main.go                    # Database-driven API server
│   └── go.mod                     # Dependencies (uuid, mux, pq)
├── cli/                           # Command-line interface
│   └── agent-metareasoning-manager-cli.sh  # Thin API wrapper (90 lines)
├── initialization/
│   ├── automation/n8n/            # 5 reasoning workflows
│   ├── automation/windmill/       # UI dashboards and apps
│   ├── configuration/             # Prompt libraries and templates
│   └── storage/postgres/          # Database schema and seed data
├── deployment/
│   └── startup.sh                 # Go-only startup script
└── IMPLEMENTATION_PLAN.md         # Detailed upgrade plan and progress
```

## 🔄 **How It Works**

1. **Request**: User sends analysis request via CLI or REST API
2. **Database Lookup**: Go API queries PostgreSQL for workflow configuration
3. **Orchestration**: Request forwarded to n8n webhook or Windmill job
4. **AI Processing**: Workflow invokes Ollama LLM for reasoning
5. **Storage**: Results saved to PostgreSQL with automatic metrics updates
6. **Response**: Structured JSON results with execution tracking
7. **Metrics**: Database triggers update workflow statistics automatically

## 🎯 **Extension Points**

- **Add Reasoning Patterns**: Use API `POST /workflows` to add new patterns dynamically
- **Version Control**: Updates create new versions, preserving history
- **Custom Models**: Configure different Ollama models per workflow
- **UI Dashboards**: Build Windmill apps that consume the API
- **Semantic Search**: Qdrant ready for embedding integration (Phase 3)
- **Import/Export**: Coming in Phase 3 for workflow portability

## 🧪 **Testing**

### **Quick Testing with Helper Script**
```bash
# Use the test helper for rapid validation
./test-helper.sh health    # Check all services are running
./test-helper.sh api       # Test API endpoints
./test-helper.sh cli       # Test CLI functionality  
./test-helper.sh workflows # List deployed n8n workflows
./test-helper.sh quick     # Run quick smoke tests
./test-helper.sh full      # Run complete test suite
```

### **Standard Lifecycle Testing**
```bash
# Run the test lifecycle phase (defined in .vrooli/service.json)
../../manage.sh test --target native-linux

# This executes:
# - Go compilation test
# - API health endpoint test
# - Workflows endpoint test  
# - Analysis endpoint test
# - CLI test suite (if bats file exists)
```

### **Manual Testing Examples**
```bash
# After 'develop' is running, test CLI commands
agent-metareasoning-manager analyze decision "Should we adopt GraphQL?"
agent-metareasoning-manager analyze swot "Our SaaS product" "competitive market"
agent-metareasoning-manager analyze risk-assessment "Cloud migration" "Limited budget"

# Test API directly (remember authentication!)
curl -X POST http://localhost:$API_PORT/analyze/self-review \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer agent_metareasoning_manager_cli_default_2024" \
  -d '{"decision": "Implement microservices", "iterations": 3}'
```

### **Comprehensive Scenario Testing**
```bash
# Run the full scenario test suite
vrooli scenario test agent-metareasoning-manager

# Or use the scenario test YAML directly
../../scenarios/tests/runner.sh agent-metareasoning-manager/scenario-test.yaml
```

## 🔧 **Configuration**

The API uses environment variables for configuration (automatically set by service.json):

```bash
# These are set dynamically during 'develop' phase:
export PORT=$API_PORT                                          # Dynamic API port
export N8N_BASE_URL=http://localhost:${RESOURCE_PORTS[n8n]}       # n8n instance
export WINDMILL_BASE_URL=http://localhost:${RESOURCE_PORTS[windmill]} # Windmill instance
export WINDMILL_WORKSPACE=demo                                    # Windmill workspace
export EMBEDDING_MODEL=nomic-embed-text                           # Ollama embedding model for semantic search

# To get actual runtime ports:
source ../../resources/port_registry.sh
echo "n8n port: ${RESOURCE_PORTS[n8n]}"
echo "Windmill port: ${RESOURCE_PORTS[windmill]}"
```

CLI configuration is stored in `~/.metareasoning/config.json`:

```json
{
  "api_base": "http://localhost:${API_PORT}",
  "default_format": "table",
  "api_token": "agent_metareasoning_manager_cli_default_2024",
  "created_at": "2024-01-01T00:00:00Z"
}
```

**Note:** The CLI should auto-detect the service port from the running API, or you can manually update the config file after deployment.

## 🔍 **Troubleshooting & Testing Tips**

### **Port Discovery**
```bash
# Find what ports are actually in use
../../resources/port_registry.sh --action list

# Check if services are running on expected ports
lsof -i :${RESOURCE_PORTS[n8n]}       # Should show n8n
lsof -i :${RESOURCE_PORTS[windmill]}  # Should show Windmill
lsof -i :$API_PORT                 # Should show Go API
```

### **Service Health Checks**
```bash
# Check each service independently
curl -f http://localhost:${RESOURCE_PORTS[n8n]}/healthz       # n8n health
curl -f http://localhost:${RESOURCE_PORTS[windmill]}/api/version # Windmill
curl -f http://localhost:$API_PORT/health                  # Go API
```

### **Common Issues**

| **Issue** | **Solution** |
|-----------|-------------|
| **Port conflicts** | Check `../../resources/port_registry.sh --action check` for conflicts |
| **Go build fails** | Ensure Go 1.21+ is installed: `go version` |
| **CLI not found** | Re-run setup: `../../manage.sh setup --target native-linux` |
| **Workflows missing** | Check n8n UI at `http://localhost:${RESOURCE_PORTS[n8n]}` |
| **Authentication errors** | Use token: `agent_metareasoning_manager_cli_default_2024` in API calls |

### **Testing Checklist**
- [ ] All resources started (`ollama`, `n8n`, `windmill`, `postgres`, `qdrant`)
- [ ] Go API compiled and running on `$API_PORT`
- [ ] CLI installed globally as `metareasoning`
- [ ] n8n workflows imported (5 workflows visible in n8n UI)
- [ ] Windmill apps deployed (2 apps visible in Windmill)
- [ ] PostgreSQL tables created (check with `psql`)
- [ ] Qdrant collections initialized (check via API)

## 📚 **Related Documentation**

- [Port Registry System](../../resources/port_registry.sh) - Central port definitions
- [Scenario Lifecycle Management](../../manage.sh) - How lifecycle commands work
- [n8n Workflow Development](../../resources/automation/n8n/)
- [Windmill App Creation](../../resources/automation/windmill/)  
- [Go Runtime Setup](../../lib/runtimes/go.sh)
- [Scenario Architecture Guide](../../../docs/scenarios/)
- [Direct Scenario Execution](../../../docs/scenarios/DEPLOYMENT.md)

---

**The Agent Metareasoning Manager proves that powerful AI coordination can be achieved with minimal, efficient code by leveraging the right architectural patterns.**