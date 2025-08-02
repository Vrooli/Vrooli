# [Scenario Name] - [Brief Description]

## 🎯 Overview

Brief description of what this scenario demonstrates and its business value.

## 📋 Prerequisites

- Required resources: `resource1`, `resource2`
- Optional resources: `resource3` (for enhanced functionality)
- System requirements: Any special requirements

## 🚀 Quick Start

```bash
# Run the test scenario
./test.sh

# Run with custom configuration
TEST_TIMEOUT=1200 ./test.sh

# Skip cleanup for debugging
TEST_CLEANUP=false ./test.sh
```

## 💼 Business Value

### Use Cases
- Primary use case this scenario addresses
- Secondary use cases

### Target Market
- Industries or business segments
- Specific roles who would benefit

### Revenue Potential
- Project range: $X,000 - $Y,000
- Recurring revenue opportunities

## 🔧 Technical Details

### Architecture
Brief description of how the components work together.

### Resources Used
- **Resource1**: How it's used in this scenario
- **Resource2**: Its role and importance
- **Resource3** (optional): Additional capabilities it provides

### Data Flow
1. Step 1: Initial data input
2. Step 2: Processing stage
3. Step 3: Output generation

## 🔌 Resource Integration (if applicable)

<!-- Only include this section if your scenario includes resource-specific artifacts -->

### Directory Structure
```
scenario-name/
├── metadata.yaml
├── test.sh
├── README.md
├── ui/                    # Optional: UI components
└── resources/             # Optional: Resource-specific artifacts
    ├── n8n/              # n8n workflow exports
    ├── windmill/         # Windmill scripts
    ├── postgres/         # Database schemas/seeds
    ├── api/              # API collections
    └── config/           # Resource-specific configs
```

### n8n Workflows
<!-- Include if scenario uses n8n -->
- **File**: `resources/n8n/workflow-name.json`
- **Import**: Navigate to n8n UI → Import → Select file
- **Configuration**: Update webhook URL to `${N8N_BASE_URL}/webhook/scenario-name`
- **Required Credentials**: List any credentials that need to be configured

### Windmill Scripts
<!-- Include if scenario uses Windmill -->
- **Directory**: `resources/windmill/`
- **Deployment**: `wmill sync push --workspace scenario-name`
- **Dependencies**: Listed in `resources/windmill/requirements.txt`
- **Entry Point**: `resources/windmill/main.py` or `main.ts`

### Database Setup
<!-- Include if scenario uses PostgreSQL -->
- **Schema**: `resources/postgres/schema.sql`
- **Seed Data**: `resources/postgres/seed.sql`
- **Indexes**: `resources/postgres/indexes.sql`
- **Apply Schema**: 
  ```bash
  psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -f resources/postgres/schema.sql
  psql -h $POSTGRES_HOST -U $POSTGRES_USER -d $POSTGRES_DB -f resources/postgres/seed.sql
  ```

### API Collections
<!-- Include if scenario has API endpoints to test -->
- **Postman Collection**: `resources/api/postman-collection.json`
- **Environment Variables**: `resources/api/environment.json`
- **OpenAPI Spec**: `resources/api/openapi.yaml`
- **Usage**: Import collection and environment into Postman/Insomnia

### Configuration Files
<!-- Include if scenario needs special configuration -->
- **Environment Template**: `resources/config/.env.template`
- **Docker Compose Override**: `resources/config/docker-compose.override.yml`
- **Service Configuration**: `resources/config/service-config.yaml`

## 🛠️ Setup Instructions (if applicable)

<!-- Include detailed setup steps if the scenario requires manual configuration -->

### Initial Setup
1. **Configure Resources**: Ensure all required resources are running
2. **Import Artifacts**: Load any workflows, scripts, or schemas
3. **Set Environment Variables**: Copy and configure `.env.template` if provided
4. **Initialize Data**: Run database migrations and seed data

### Resource-Specific Setup
<!-- Add sections for each resource that needs configuration -->

#### n8n Setup
```bash
# Import workflow
curl -X POST ${N8N_BASE_URL}/api/v1/workflows \
  -H "X-N8N-API-KEY: $N8N_API_KEY" \
  -H "Content-Type: application/json" \
  -d @resources/n8n/workflow.json
```

#### Database Setup
```bash
# Create database and apply schema
./resources/postgres/setup.sh
```

## 🧪 Test Coverage

This scenario validates:
- ✅ Capability 1
- ✅ Capability 2
- ✅ Integration between services
- ✅ Error handling and recovery

## 📊 Success Metrics

- Metric 1: Target value
- Metric 2: Expected range
- Performance: Latency/throughput expectations

## 🎨 UI Components (if applicable)

If this scenario includes a UI:
- Component description
- Deployment instructions
- User workflows

## 🚧 Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| Service not accessible | Check service health with `check_service_health` |
| Timeout errors | Increase TEST_TIMEOUT value |

## 📚 Related Documentation

- [Link to relevant docs]
- [API documentation]
- [Best practices guide]

## 🏷️ Tags

`tag1`, `tag2`, `tag3`