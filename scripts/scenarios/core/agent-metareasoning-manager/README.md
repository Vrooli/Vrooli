# Agent Metareasoning Manager

A **minimal coordination API** built in Go that enhances AI agent decision-making through structured reasoning patterns and workflows.

## 🎯 **What It Does**

This scenario provides metareasoning capabilities to AI agents by orchestrating reasoning workflows through n8n and Windmill. Instead of complex business logic in code, it acts as a **lightweight coordination layer** that lets workflows handle the actual reasoning.

## ⚡ **Architecture** 

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│                 │    │                  │    │                     │
│   CLI Client    ├────► Go Coordination  ├────► n8n/Windmill       │
│                 │    │ API (8MB binary) │    │ Reasoning Workflows │
│                 │    │                  │    │                     │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                                │                         │
                                │                         │
                                ▼                         ▼
                       ┌──────────────────┐    ┌─────────────────────┐
                       │                  │    │                     │
                       │  Workflow Config │    │   Ollama LLM        │
                       │  (JSON Registry) │    │   Local Inference   │
                       │                  │    │                     │
                       └──────────────────┘    └─────────────────────┘
```

## 🚀 **Key Features**

- **Minimal Footprint**: Single 8MB Go binary (vs 100MB+ Node.js apps)
- **Direct Workflow Integration**: Calls n8n webhooks and Windmill jobs directly
- **No Database Required**: Uses simple JSON configuration files
- **5 Reasoning Patterns**: Pros/cons, SWOT, risk assessment, decision analysis, self-review
- **CLI Interface**: `metareasoning` command for easy interaction
- **Cross-Platform**: Linux, macOS, Windows support

## 📋 **Available Analysis Types**

1. **Pros/Cons Analysis**: Weighted advantages vs disadvantages with scoring
2. **SWOT Analysis**: Strategic strengths, weaknesses, opportunities, threats  
3. **Risk Assessment**: Probability × Impact analysis with mitigation strategies
4. **Decision Analysis**: Multi-factor decision support with recommendations
5. **Self-Review**: Iterative reasoning validation and improvement

## 🛠️ **Quick Start**

### **Setup**
```bash
# Install via Vrooli lifecycle system
./scripts/manage.sh setup --target native-linux

# This automatically:
# - Installs Go runtime if needed
# - Builds the 8MB coordination API binary  
# - Sets up n8n workflows and Windmill apps
# - Installs global 'metareasoning' CLI command
```

### **Usage**
```bash
# Start the system
./scripts/manage.sh develop

# Use the CLI
metareasoning health                                    # Check system status
metareasoning api                                       # Show API info
metareasoning analyze pros-cons "Remote work policy"   # Run analysis
metareasoning workflow list                             # Show available workflows
```

### **API Direct Usage**
```bash
# Health check
curl http://localhost:8093/health

# List workflows
curl http://localhost:8093/workflows

# Run analysis
curl -X POST http://localhost:8093/analyze/pros-cons \
  -H "Content-Type: application/json" \
  -d '{"input": "Should we migrate to microservices?", "context": "Legacy monolith"}'
```

## 📁 **Project Structure**

```
agent-metareasoning-manager/
├── api/                           # Go coordination API
│   ├── main.go                    # Complete API server (150 lines)
│   ├── workflows.json             # Workflow registry configuration
│   └── go.mod                     # Dependencies (2 packages)
├── cli/                           # Command-line interface
│   └── metareasoning-cli.sh       # Bash CLI with auto-detection
├── initialization/
│   ├── automation/n8n/            # 5 reasoning workflows
│   ├── automation/windmill/       # UI dashboards and apps
│   └── configuration/             # Prompt libraries and templates
└── deployment/
    └── startup.sh                 # Go-only startup script
```

## 🔄 **How It Works**

1. **CLI/API Request**: User requests analysis via CLI or direct API
2. **Workflow Dispatch**: Go API identifies appropriate n8n/Windmill workflow
3. **LLM Processing**: Workflow calls Ollama for actual reasoning
4. **Result Processing**: Structured results returned via API
5. **Response**: Formatted output delivered to user

## 🎯 **Cross-Application Benefits**

### **Easy Extension**
- **Add New Patterns**: Just add new n8n workflow + registry entry
- **Share Workflows**: Other apps can use same reasoning patterns
- **No Code Changes**: All logic in workflows, not API code

### **Resource Efficiency**
- **Single Binary**: No complex runtime dependencies
- **Fast Startup**: <0.5s vs 5-8s for Node.js equivalents
- **Low Memory**: 8MB binary vs 100MB+ typical alternatives

## 🧪 **Testing**

```bash
# Build test
./scripts/manage.sh test

# Manual testing
metareasoning analyze decision "Should we adopt GraphQL?"
metareasoning analyze swot "Our SaaS product" "competitive market"
```

## 🔧 **Configuration**

The API uses environment variables for configuration:

```bash
export PORT=8093                                    # API port
export N8N_BASE_URL=http://localhost:5678          # n8n instance
export WINDMILL_BASE_URL=http://localhost:8000     # Windmill instance
export WINDMILL_WORKSPACE=demo                     # Windmill workspace
```

CLI configuration is stored in `~/.metareasoning/config.json`:

```json
{
  "api_base": "http://localhost:8093",
  "default_format": "table",
  "api_token": "",
  "created_at": "2024-01-01T00:00:00Z"
}
```

## ⚡ **Why Go Instead of Node.js?**

| **Aspect** | **Go** | **Node.js** |
|------------|--------|-------------|
| **Binary Size** | 8MB single file | 100MB+ with node_modules |
| **Startup Time** | <0.5 seconds | 5-8 seconds |
| **Memory Usage** | ~10MB | ~50-100MB |
| **Dependencies** | 2 packages | 15+ packages |
| **Deployment** | Copy single binary | Complex runtime setup |
| **Cross-platform** | Single build → all platforms | Platform-specific considerations |

## 🎯 **Perfect for Scenarios**

This demonstrates the Vrooli principle: **scenarios should orchestrate, not implement**. By keeping business logic in workflows and using a minimal coordination layer, we get:

- ✅ **90% less code** to maintain
- ✅ **Better performance** and resource usage  
- ✅ **Easier extension** by other applications
- ✅ **Simpler deployment** and scaling
- ✅ **Cross-application sharing** of reasoning patterns

## 📚 **Related Documentation**

- [n8n Workflow Development](../../resources/automation/n8n/)
- [Windmill App Creation](../../resources/automation/windmill/)  
- [Go Runtime Setup](../../lib/system/go.sh)
- [Scenario Architecture Guide](../../../docs/scenarios/)

---

**The Agent Metareasoning Manager proves that powerful AI coordination can be achieved with minimal, efficient code by leveraging the right architectural patterns.**