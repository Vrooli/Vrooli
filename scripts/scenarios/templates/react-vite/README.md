# {{ scenario.name }} - React + Vite Scenario Template (Go API + CLI)

> **Enterprise-grade scenario template with Go API server, CLI tool, and complete deployment orchestration**

<!-- 
🔄 TEMPLATE ENHANCED WITH API & CLI PATTERNS:
This template now includes the successful patterns from agent-metareasoning-manager:
- Go API server for coordination
- Bash CLI tool for command-line access
- Database-driven architecture
- Complete lifecycle management

DUAL TEMPLATING APPROACH:
- For deployment orchestration: Uses Jinja2 syntax {{ variable.name }}
- For AI generation: Use PLACEHOLDER_NAME patterns (see AI guidance comments)
- AI agents should replace both placeholder types during generation
-->

## 🆕 **What's New in This Template**

This template includes the **modern scenario architecture** based on agent-metareasoning-manager pattern:

- ✅ **Go API Server** - RESTful API with database integration
- ✅ **CLI Tool** - Command-line interface for all operations
- ✅ **`service.json`** - Unified configuration with lifecycle management
- ✅ **PostgreSQL Integration** - Database-driven architecture
- ✅ **Complete Testing** - API, CLI, and integration tests
- ✅ **Seeded Requirements Registry** - `requirements/index.json` demonstrates how operational targets link to validations
- ✅ **One-command deployment** via scenario lifecycle phases

## 🎯 **Business Overview**

### **Value Proposition**
{{ business.value_proposition }}
<!-- AI: Replace with VALUE_PROPOSITION_PLACEHOLDER - include specific metrics/outcomes -->

### **Target Markets**
{% for market in business.target_markets %}
- {{ market }}
{% endfor %}
<!-- AI: Replace with PRIMARY_MARKET_PLACEHOLDER, SECONDARY_MARKET_PLACEHOLDER -->

### **Pain Points Addressed**
{% for pain_point in business.pain_points %}
- {{ pain_point }}
{% endfor %}
<!-- AI: Replace with PAIN_POINT_1_PLACEHOLDER, PAIN_POINT_2_PLACEHOLDER -->

### **Revenue Potential**
- **Range**: ${{ business.revenue_potential.min | number_format }} - ${{ business.revenue_potential.max | number_format }}
- **Market Demand**: {{ business.market_demand }}
- **Pricing Model**: {{ business.revenue_potential.pricing_model }}
<!-- AI: Adjust min/max based on scenario complexity and business value -->

## 🏗️ **Architecture**

### **System Components**
```
┌─────────────────┐     ┌─────────────────┐
│      CLI        │────▶│   Go API Server │
│  (CLI_NAME)     │     │   (Port: 8090+) │
└─────────────────┘     └─────────────────┘
                                │
                ┌───────────────┼───────────────┐
                ▼               ▼               ▼
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Frontend UI   │     │   Workflows     │     │  AI Processing  │
│   (Windmill)    │     │   (n8n/etc)     │     │   (Ollama/etc)  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                │                           │
                                ▼                           ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │   Database      │     │   Storage       │
                        │  (PostgreSQL)   │     │  (MinIO/etc)    │
                        └─────────────────┘     └─────────────────┘
```

### **Required Resources**
- **PostgreSQL**: Primary database for all application data
- **n8n**: Workflow automation and orchestration
- **Windmill**: UI applications and dashboards
<!-- AI: Add additional required resources based on scenario needs -->

### **Optional Resources**
- **Ollama**: Local AI model inference
- **Qdrant**: Vector database for semantic search
- **MinIO**: Object storage for files
<!-- AI: Add optional resources that enhance functionality -->

## 🚀 **Quick Start**

### **1. Setup and Build**
```bash
# Navigate to scenario directory
cd {{ scenario.id }}

# Run setup lifecycle (builds API, installs CLI)
../../manage.sh setup --target native-linux

# This automatically:
# - Builds Go API server
# - Installs CLI globally
# - Initializes database
# - Imports workflows
```

### **2. Start Development Environment**
```bash
# Start all services
../../manage.sh develop --target native-linux

# Services will be available at:
# - API Server: http://localhost:${API_PORT}
# - API Docs: http://localhost:${API_PORT}/docs
# - Windmill UI: http://localhost:5681
# - n8n Workflows: http://localhost:5678
```

### **3. Use the CLI**
```bash
# After setup, CLI is available globally
CLI_NAME_PLACEHOLDER health                          # Check system health
CLI_NAME_PLACEHOLDER list resources                  # List all resources
CLI_NAME_PLACEHOLDER get resources <id>              # Get specific resource
CLI_NAME_PLACEHOLDER create resources name "Test"    # Create resource
CLI_NAME_PLACEHOLDER execute workflow-1 "Input data" # Execute workflow
```

### **4. Access API Directly**
```bash
# Health check
curl http://localhost:${API_PORT}/health

# List resources (with authentication)
curl -H "Authorization: Bearer API_TOKEN_PLACEHOLDER" \
     http://localhost:${API_PORT}/api/v1/resources

# Create resource
curl -X POST \
     -H "Authorization: Bearer API_TOKEN_PLACEHOLDER" \
     -H "Content-Type: application/json" \
     -d '{"name": "Test", "description": "Example"}' \
     http://localhost:${API_PORT}/api/v1/resources
```

## 📁 **File Structure**

### **Core Files**
```
{{ scenario.id }}/
├── .vrooli/
│   └── service.json           # Unified configuration and lifecycle
├── api/                       # Go API server
│   ├── cmd/server/main.go     # API entry point
│   ├── go.mod                 # Go dependencies
│   └── go.sum                 # Dependency checksums
├── cli/                       # Command-line interface
│   ├── cli.sh                 # CLI implementation
│   ├── install.sh             # CLI installer
│   └── cli-tests.bats         # CLI tests
├── README.md                  # This documentation
├── scenario-test.yaml         # Scenario validation tests
└── test.sh                    # Integration tests
```

### **Initialization Data**
```
initialization/
├── automation/
│   ├── n8n/                   # n8n workflow definitions
│   │   └── main-workflow.json # Primary workflow
│   └── windmill/              # Windmill apps
│       └── windmill-app.json  # UI application
├── configuration/
│   ├── app-config.json        # Runtime settings
│   ├── resource-urls.json     # Service endpoints
│   └── feature-flags.json     # Feature toggles
└── storage/
    ├── postgres/              # PostgreSQL database
    │   ├── schema.sql         # Database structure
    │   └── seed.sql           # Initial data
    ├── qdrant/                # Vector database (optional)
    │   └── collections.json   # Collection definitions
    └── minio/                 # Object storage (optional)
        └── buckets.json       # Bucket configuration
```

### **Deployment Scripts**
```
deployment/
├── startup.sh                 # Application initialization
└── monitor.sh                 # Health monitoring
```

## 🔧 **API & CLI Development**

### **API Server**
The Go API server provides RESTful endpoints for all scenario operations:

```go
// api/cmd/server/main.go
// Key endpoints:
// GET    /health              - Health check
// GET    /docs                - API documentation
// GET    /api/v1/resources    - List resources
// POST   /api/v1/resources    - Create resource
// GET    /api/v1/resources/:id - Get resource
// PUT    /api/v1/resources/:id - Update resource
// DELETE /api/v1/resources/:id - Delete resource
// POST   /api/v1/execute      - Execute workflow
```

### **CLI Tool**
The CLI provides command-line access to all API functionality:

```bash
# Basic commands
CLI_NAME_PLACEHOLDER health              # Check system health
CLI_NAME_PLACEHOLDER list resources      # List all resources
CLI_NAME_PLACEHOLDER get resources <id>  # Get specific resource
CLI_NAME_PLACEHOLDER create resources name "Example" description "Test"
CLI_NAME_PLACEHOLDER execute workflow-1 "Process this data"

# Configuration
CLI_NAME_PLACEHOLDER configure api_base http://localhost:8090
CLI_NAME_PLACEHOLDER configure api_token your-token-here
CLI_NAME_PLACEHOLDER configure output_format json
```

### **Authentication**
The API uses Bearer token authentication:
```bash
curl -H "Authorization: Bearer API_TOKEN_PLACEHOLDER" \
     http://localhost:${API_PORT}/api/v1/resources
```

## 🔧 **Customization Guide**

### **Business Configuration**
Edit `.vrooli/service.json` metadata section:
```json
"metadata": {
  "businessModel": {
    "valueProposition": "Your unique value proposition",
    "targetMarket": "Your primary market",
    "revenuePotential": {
      "initial": "$15000",
      "recurring": "$5000",
      "totalEstimate": "$30000"
    }
  }
}
```

### **API Customization**
Edit `api/cmd/server/main.go`:
- Add new endpoints for your business logic
- Customize database queries
- Implement workflow triggers
- Add validation and business rules

### **CLI Customization**
Edit `cli/cli.sh`:
- Add scenario-specific commands
- Customize output formatting
- Add shortcuts and aliases
- Implement batch operations

### **Database Schema**
Edit `initialization/storage/postgres/schema.sql`:
- Add business-specific tables
- Configure indexes and constraints
- Set up views and functions
- Define relationships

### **Workflow Logic**
Edit `initialization/automation/n8n/main-workflow.json`:
- Add business logic nodes
- Configure API integrations
- Set up data processing steps
- Define triggers and schedules

## 🧪 **Testing & Validation**

### **Lifecycle Testing**
```bash
# Run test lifecycle phase
../../manage.sh test --target native-linux

# This executes:
# - Go compilation test
# - API health checks
# - API endpoint tests
# - CLI command tests
# - Integration tests
```

### **Manual Testing**
```bash
# Test API endpoints
curl http://localhost:${API_PORT}/health
curl -H "Authorization: Bearer API_TOKEN_PLACEHOLDER" \
     http://localhost:${API_PORT}/api/v1/resources

# Test CLI commands
CLI_NAME_PLACEHOLDER health
CLI_NAME_PLACEHOLDER list resources
CLI_NAME_PLACEHOLDER create resources name "Test"

# Run integration tests
./test.sh
```

### **Requirement Coverage**
- Document PRD-level requirements in `docs/requirements.json` or a modular `requirements/` registry. Each requirement should list the phase scripts, unit tests, or workflow automations that validate it.
- Tag unit tests with `REQ:<ID>` (for example, `t.Run("persists workflows [REQ:SAMPLE-FUNC-001]", ...)`) so the shared Go/Node runners automatically record pass/fail status.
- Execute YAML-defined automations with `testing::phase::run_workflow_yaml --file automation/workflows/<name>.yaml` inside phase scripts to keep requirement evidence aligned.
- Generate coverage reports:
  ```bash
  vrooli scenario requirements {{ scenario.id }} --format markdown
  vrooli scenario requirements {{ scenario.id }} --fail-on-critical-gap --format json --output coverage/requirements.json
  ```
- Export `TESTING_REQUIREMENTS_ENFORCE=1` when running phases to treat missing requirement updates as failures.

### **Expected Results**
- ✅ All resources healthy
- ✅ API server running
- ✅ CLI commands working
- ✅ Database initialized
- ✅ Workflows deployed and active
- ✅ UI accessible
- ✅ End-to-end functionality working

## 📊 **Performance Expectations**

### **Response Times**
- **API Calls**: < 100ms (p50), < 500ms (p95)
- **Workflow Execution**: < 30s typical
- **UI Load Time**: < 2 seconds
- **CLI Commands**: < 1 second

### **Throughput**
- **Concurrent Users**: 10-100
- **Requests/Second**: 50-500
- **Database Connections**: 5-20 pool size

### **Resource Usage**
- **API Server**: ~50MB RAM, minimal CPU
- **Database**: ~100MB initial size
- **Workflows**: Depends on complexity

## 🔒 **Security & Compliance**

### **Built-in Security**
- Bearer token authentication
- Database access controls
- API rate limiting
- Input validation
- SQL injection prevention
- Audit logging

### **Production Checklist**
- [ ] Change default API tokens
- [ ] Configure SSL certificates
- [ ] Set up database backups
- [ ] Enable monitoring alerts
- [ ] Review access permissions
- [ ] Configure firewall rules

## 💰 **Business Impact**

### **Revenue Model**
This scenario template targets projects in the **$10K-$50K** range with proven market demand.

### **Success Criteria**
- Implementation in hours instead of weeks
- Professional quality from day one
- Ready for production deployment
- Scalable architecture

### **ROI Metrics**
- **Development Speed**: 10x faster than traditional development
- **Resource Efficiency**: Deploy only required services
- **Professional Quality**: Enterprise-ready features included
- **Maintenance**: Self-documenting with clear structure

## 🛟 **Support & Resources**

### **Documentation**
- **[Agent Metareasoning Manager](../../agent-metareasoning-manager/)**: Reference implementation
- **[Scenarios README](../README.md)**: Main scenarios documentation
- **[Resource Guide](../../../resources/README.md)**: Available resources

### **Troubleshooting**
```bash
# Check service health
../../manage.sh test --target native-linux

# View logs
docker logs <container-name>

# Verify ports
lsof -i :${API_PORT}

# Database connection
psql -h localhost -p 5433 -U postgres
```

### **Common Issues**
| Issue | Solution |
|-------|----------|
| API won't start | Check port conflicts, verify Go build |
| CLI not found | Re-run setup phase: `../../manage.sh setup` |
| Database errors | Check PostgreSQL is running, verify schema |
| Workflow failures | Check n8n UI for error details |

## 🎯 **Next Steps**

### **For Development**
1. Copy this template: `cp -r scripts/scenarios/templates/react-vite/ scenarios/your-scenario/`
2. Update `.vrooli/service.json` with your scenario details
3. Customize API endpoints in `api/cmd/server/main.go`
4. Update CLI commands in `cli/cli.sh`
5. Adapt database schema and seed data
6. Build and test: `../../manage.sh setup && ../../manage.sh develop`
7. Run tests: `../../manage.sh test`
8. Deploy: `../../manage.sh deploy`

### **For Production**
1. Review and update security configuration
2. Set up monitoring and alerts
3. Configure backup procedures
4. Plan scaling strategy
5. Document API for external consumers
6. Train users on CLI and UI

### **For AI Generation**
This template is optimized for AI agents to generate complete scenarios. **Key placeholders to replace:**

- `SCENARIO_NAME_PLACEHOLDER` - The scenario's display name
- `SCENARIO_ID_PLACEHOLDER` - The scenario's ID (lowercase, hyphenated)
- `CLI_NAME_PLACEHOLDER` - The CLI command name
- `API_PORT_PLACEHOLDER` - The API server port (8090-8999 range)
- `API_TOKEN_PLACEHOLDER` - Default API authentication token
- `API_MODULE_NAME_PLACEHOLDER` - Go module name for API
- `VALUE_PROPOSITION_PLACEHOLDER` - Business value proposition
- `PRIMARY_MARKET_PLACEHOLDER` - Primary target market
- `PAIN_POINT_*_PLACEHOLDER` - Pain points addressed
- All other `*_PLACEHOLDER` values throughout the template

---

**🎉 This enhanced template provides a complete foundation for building professional scenarios with API servers, CLI tools, and full deployment orchestration - following the proven patterns from agent-metareasoning-manager!**
