# Scenario Development and Enhancement

You are tasked with creating or fixing scenarios in Vrooli. Scenarios are the crown jewel of Vrooli's architecture - they serve dual purposes as both validation tests and complete application blueprints worth $10K-50K each.

## Understanding Scenarios in Vrooli

**What Scenarios Are**: Complete, deployable application blueprints that demonstrate specific business capabilities. Each scenario validates resource integration while simultaneously serving as a template for revenue-generating customer applications.

**Why This Matters**: Scenarios are where all of Vrooli's power comes together. Each scenario you build becomes a permanent capability that can be:
- Deployed as a complete SaaS application for customers
- Used as validation tests for resource integration
- Referenced by AI agents for building similar applications
- Composed with other scenarios for complex solutions

## Pre-Implementation Research

### 1. Study Scenario Architecture
```bash
# Understand scenario structure
tree scripts/scenarios/core/prompt-manager/

# Read scenario documentation
cat scripts/scenarios/README.md
cat scripts/scenarios/docs/CONCEPTS.md
```

### 2. Examine Existing Scenarios
```bash
# Study successful scenario patterns
ls scripts/scenarios/core/

# Analyze scenario service.json files
find scripts/scenarios/core/ -name "service.json" | head -3 | xargs cat
```

### 3. Understand Testing Framework
**Key Files to Read**:
- `scripts/scenarios/docs/TESTING_GUIDE.md` - Testing patterns
- `scripts/scenarios/framework/scenario-test-runner.sh` - Test execution
- `scripts/scenarios/tools/validate-scenario.sh` - Validation tools

## Required Scenario Structure

```
scripts/scenarios/core/{scenario-name}/
├── service.json         # Configuration
├── README.md           # Documentation  
├── scenario-test.yaml  # Tests
├── api/                # Backend service
├── cli/                # CLI tools
├── ui/                 # Frontend (optional)
├── initialization/     # Resources & data
│   ├── automation/     # n8n workflows
│   └── storage/        # Database schemas
└── deployment/         # Startup scripts
```

**Complete structure example**: 
```bash
tree -L 2 scripts/scenarios/core/prompt-manager/
```

## Implementation Steps

### 1. Create service.json Configuration

**Minimal Template**:
```json
{
  "service": {
    "name": "your-scenario",
    "type": "business-application"
  },
  "resources": {
    "postgres": { "enabled": true }
  }
}
```

**Complete Examples**:
```bash
# Study real implementations:
cat scripts/scenarios/core/prompt-manager/service.json
cat scripts/scenarios/core/agent-metareasoning-manager/service.json
```

### 2. Design Resource Integration

Determine which resources your scenario needs:

```bash
# Review available resources
./scripts/resources/index.sh --action discover

# Study resource integration patterns
cat scripts/resources/docs/integration-cookbook.md
```

**Common Resource Combinations**:
- **Data Processing**: PostgreSQL + Ollama + Qdrant
- **Automation**: N8n + Node-RED + Windmill  
- **AI Applications**: Ollama + Whisper + ComfyUI
- **Web Automation**: Agent-S2 + Browserless + SearXNG

### 3. Create Database Schema

Design your database schema in `initialization/storage/postgres/schema.sql`:

```sql
-- Your Scenario Database Schema
CREATE TABLE IF NOT EXISTS your_entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_your_entities_name ON your_entities(name);
CREATE INDEX idx_your_entities_created_at ON your_entities(created_at DESC);
```

### 4. Implement Application Logic

**API Service Pattern** (api/main.go):
```go
package main

import (
    "database/sql"
    "net/http"
    _ "github.com/lib/pq"
)

func main() {
    db, _ := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    // Setup routes, start server on PORT
}
```
**Full examples**: See any `api/main.go` in `/scripts/scenarios/core/*/`

### 5. Create Automation Workflows

Design n8n workflows for your scenario's automation needs:

```bash
# Study workflow patterns
cat scripts/scenarios/core/agent-metareasoning-manager/initialization/automation/n8n/reasoning-chain.json

# Create your workflow
mkdir -p initialization/automation/n8n/
# Design workflow JSON based on your requirements
```

### 6. Implement Testing

**Test Configuration** (scenario-test.yaml):
```yaml
name: your-scenario
tests:
  - name: Health Check
    type: http
    endpoint: http://localhost:8080/health
    expected: { status: 200 }
```
**Complete test examples**: `/scripts/scenarios/core/*/scenario-test.yaml`

### 7. Create Startup and Validation Scripts

**Startup Script Pattern** (deployment/startup.sh):
```bash
#!/bin/bash
set -euo pipefail

# Initialize database
psql -d $DB_NAME < initialization/storage/postgres/schema.sql

# Start services
cd api && go run main.go &
```
**Reference**: Any `deployment/startup.sh` in existing scenarios

## Validation and Testing

### 1. Scenario Structure Validation
```bash
# Validate scenario structure
./scripts/scenarios/tools/validate-scenario.sh your-scenario

# Check service.json schema
jq . scripts/scenarios/core/your-scenario/service.json
```

### 2. Resource Integration Testing
```bash
# Test resource injection
./scripts/resources/injection/engine.sh --scenario your-scenario --action inject

# Validate resource connectivity
./scripts/scenarios/framework/scenario-test-runner.sh your-scenario
```

### 3. End-to-End Testing
```bash
# Full scenario test
cd scripts/scenarios/core/your-scenario/
./test.sh

# Integration with scenario framework
./scripts/scenarios/framework/run-all-scenarios.sh --filter your-scenario
```

## Business Application Design

### 1. Value Proposition
Each scenario should target a clear business need:
- **SaaS Applications**: $10K-25K deployment value
- **Automation Solutions**: Process optimization value
- **AI-Powered Tools**: Productivity enhancement value
- **Data Processing**: Analytics and insights value

### 2. Customer-Ready Features
- Professional UI/UX design
- Comprehensive API documentation
- CLI tools for power users
- Deployment automation
- Monitoring and observability

### 3. Scalability Considerations
- Database optimization
- Resource efficiency
- Horizontal scaling support
- Performance monitoring

## Advanced Patterns

### 1. Multi-Resource Orchestration
```bash
# Study complex scenarios
cat scripts/scenarios/core/agent-metareasoning-manager/service.json
```

Design scenarios that leverage multiple resources synergistically:
- AI models + databases + automation
- Real-time processing + storage + notifications
- Web automation + data processing + reporting

### 2. Stateful Applications
Implement scenarios that maintain complex state:
- User sessions and preferences
- Workflow state management
- Background job processing
- Event-driven architectures

### 3. API-First Design
Create scenarios with robust API foundations:
- RESTful endpoint design
- Comprehensive error handling
- Rate limiting and security
- API documentation and testing

## Success Criteria

### ✅ Scenario Implementation
- [ ] Complete service.json configuration
- [ ] All required directory structure created
- [ ] API service implemented and tested
- [ ] Database schema designed and tested
- [ ] Resource integrations working

### ✅ Testing and Validation
- [ ] Scenario-test.yaml complete and passing
- [ ] Integration tests successful
- [ ] Resource injection working
- [ ] End-to-end functionality validated
- [ ] Performance requirements met

### ✅ Business Readiness
- [ ] Professional documentation
- [ ] Deployment automation working
- [ ] CLI tools functional
- [ ] UI (if applicable) polished
- [ ] Customer deployment ready

### ✅ Integration
- [ ] Scenario discoverable in catalog
- [ ] Compatible with scenario framework
- [ ] Resource dependencies documented
- [ ] Deployment patterns established

## Common Pitfalls to Avoid

1. **Incomplete Resource Integration**: Ensure all declared resources are properly used
2. **Poor Error Handling**: Implement comprehensive error handling throughout
3. **Missing Documentation**: Scenarios are useless without clear documentation
4. **Inadequate Testing**: All code paths must be tested
5. **Performance Issues**: Test with realistic data volumes
6. **Security Gaps**: Validate input, secure API endpoints, manage secrets properly

## File Locations Reference

- **Scenario Templates**: `scripts/scenarios/templates/`
- **Testing Framework**: `scripts/scenarios/framework/`
- **Validation Tools**: `scripts/scenarios/tools/`
- **Resource Integration**: `scripts/resources/injection/`
- **Example Scenarios**: `scripts/scenarios/core/`

## Deployment and Distribution

### 1. Scenario Catalog Integration
```bash
# Add to scenario catalog
jq '.scenarios["your-scenario"] = {
  "name": "Your Scenario",
  "description": "Brief description",
  "category": "productivity",
  "value": "15000",
  "resources": ["postgres", "ollama"]
}' scripts/scenarios/catalog.json > tmp.json && mv tmp.json scripts/scenarios/catalog.json
```

### 2. Customer Deployment Ready
- Professional README with setup instructions
- One-command deployment process
- Clear value proposition documentation
- Support and troubleshooting guides

### 3. Revenue Optimization
- Target high-value business problems
- Ensure professional quality standards
- Create compelling demonstrations
- Document ROI and business benefits

Remember: Every scenario you build becomes a permanent asset that generates value forever. Build with excellence, document thoroughly, and design for reusability.