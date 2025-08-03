# Vrooli Scenario Templates

## Overview

Scenario templates provide pre-configured application stacks that can be deployed instantly to Vrooli's resource ecosystem. Each template includes database schemas, workflows, AI models, and all necessary configurations for a complete application.

## Quick Start

### 1. Choose a Template

Browse available templates in the `templates/` directory:
- **ecommerce** - Complete e-commerce platform
- **real-estate** - Property listing and management system
- **saas-multitenant** - Multi-tenant SaaS application
- **ai-assistant** - AI-powered assistant with RAG
- **data-pipeline** - Data processing and analytics pipeline

### 2. Deploy a Template

```bash
# Copy template to your scenarios file
cp templates/ecommerce/scenario.json ~/.vrooli/scenarios.json

# Inject all resources
./scripts/resources/index.sh --action inject-all

# Or inject with validation
./scripts/resources/index.sh --action inject-all --validate
```

### 3. Customize a Template

```bash
# Generate customized scenario from template
./tools/generate-from-template.sh \
  --template ecommerce \
  --output my-store.json \
  --vars my-vars.env

# Validate your customized scenario
./tools/validate-template.sh my-store.json
```

## Template Structure

Each template contains:
```
template-name/
├── scenario.json       # Main scenario configuration
├── README.md          # Setup instructions and documentation
└── assets/            # Supporting files
    ├── schema.sql     # Database schemas
    ├── seed-data.sql  # Sample data
    ├── workflows/     # Automation workflows
    ├── models/        # AI model configurations
    └── configs/       # Additional configurations
```

## Available Templates

### 🛍️ E-commerce Platform
Complete online store with inventory management, order processing, and analytics.

**Includes:**
- Product catalog database (PostgreSQL)
- Image storage (MinIO)
- Order processing workflows (n8n)
- Payment integration (Vault secrets)
- Sales analytics (QuestDB)

**Use Cases:**
- Online retail stores
- Marketplace platforms
- B2B commerce portals

### 🏠 Real Estate Platform
Property listing and lead management system with virtual tours and market analytics.

**Includes:**
- Property listings database (PostgreSQL)
- Document/image storage (MinIO)
- Lead capture workflows (Node-RED)
- Property search vectors (Qdrant)
- Market analytics (QuestDB)

**Use Cases:**
- Real estate agencies
- Property rental platforms
- Real estate marketplaces

### 💼 SaaS Multi-tenant
Scalable multi-tenant SaaS application with tenant isolation and billing.

**Includes:**
- Tenant-isolated schemas (PostgreSQL)
- Per-tenant secrets (Vault)
- Onboarding automation (Windmill)
- Usage tracking (QuestDB)
- Billing workflows (n8n)

**Use Cases:**
- B2B SaaS applications
- Enterprise software
- Platform-as-a-Service

### 🤖 AI Assistant
Intelligent assistant with RAG capabilities and multi-modal generation.

**Includes:**
- Language models (Ollama)
- Knowledge base vectors (Qdrant)
- Image generation (ComfyUI)
- Conversation management (PostgreSQL)
- Integration workflows (n8n)

**Use Cases:**
- Customer support bots
- Knowledge assistants
- Creative AI tools

### 📊 Data Pipeline
End-to-end data processing pipeline with real-time analytics.

**Includes:**
- Timeseries storage (QuestDB)
- Data lake (MinIO)
- ETL workflows (Windmill)
- Stream processing (Node-RED)
- Monitoring dashboards (n8n)

**Use Cases:**
- IoT data processing
- Business intelligence
- Real-time analytics

## Customization Guide

### Variables and Environment

Templates support variable substitution using `${VARIABLE_NAME}` syntax:

```json
{
  "secrets": [
    {
      "path": "secret/data/api",
      "data": {
        "key": "${API_KEY}",
        "url": "${API_ENDPOINT}"
      }
    }
  ]
}
```

Create a variables file:
```bash
# my-vars.env
API_KEY=sk-abc123xyz
API_ENDPOINT=https://api.example.com
```

Generate scenario with variables:
```bash
./tools/generate-from-template.sh \
  --template ecommerce \
  --output custom.json \
  --vars my-vars.env
```

### Combining Templates

Merge multiple templates for complex scenarios:

```bash
./examples/combine-templates.sh \
  --base ecommerce \
  --add ai-assistant \
  --output ecommerce-with-ai.json
```

### Extending Templates

1. **Copy template as base:**
   ```bash
   cp -r templates/ecommerce templates/my-custom-store
   ```

2. **Modify scenario.json:**
   - Add new resources
   - Update configurations
   - Include custom workflows

3. **Add custom assets:**
   - Place SQL files in `assets/`
   - Add workflows in `assets/workflows/`
   - Include configuration files

4. **Validate changes:**
   ```bash
   ./tools/validate-template.sh templates/my-custom-store/scenario.json
   ```

## Best Practices

### 1. Resource Dependencies
Always define dependencies in the correct order:
```json
{
  "dependencies": ["postgres", "minio"],
  "resources": {
    "n8n": {
      "description": "Depends on postgres and minio being ready"
    }
  }
}
```

### 2. Secrets Management
Never hardcode sensitive data:
```json
// ❌ Bad
"password": "admin123"

// ✅ Good
"password": "${DB_PASSWORD}"
```

### 3. Idempotency
Design templates to be re-runnable:
- Use `CREATE TABLE IF NOT EXISTS`
- Check for existing resources
- Include rollback configurations

### 4. Documentation
Always include:
- Clear setup instructions
- Required environment variables
- Common customizations
- Troubleshooting guide

## Testing Templates

### Validate Structure
```bash
./tools/validate-template.sh templates/ecommerce/scenario.json
```

### Dry Run
```bash
./scripts/resources/index.sh --action inject-all --dry-run
```

### Test in Isolation
```bash
# Create test environment
./tools/test-scenario.sh \
  --template ecommerce \
  --isolated \
  --cleanup-after
```

## Contributing Templates

1. **Create template directory:**
   ```bash
   mkdir -p templates/my-template/assets
   ```

2. **Add scenario.json:**
   - Follow existing template structure
   - Include comprehensive resource configuration
   - Add clear descriptions

3. **Include assets:**
   - Database schemas
   - Sample data
   - Workflow definitions
   - Configuration files

4. **Document thoroughly:**
   - Create README.md
   - Include architecture diagram
   - Provide setup instructions
   - Add troubleshooting section

5. **Test extensively:**
   - Validate JSON structure
   - Test injection process
   - Verify rollback works
   - Check all dependencies

## Troubleshooting

### Common Issues

**Template not found:**
```bash
# Check template exists
ls templates/

# Verify path in command
./tools/generate-from-template.sh --template <name>
```

**Variable substitution fails:**
```bash
# Check variable file format
cat my-vars.env

# Ensure variables are exported
source my-vars.env
```

**Resource injection fails:**
```bash
# Check resource is running
./scripts/resources/index.sh --action status

# Validate configuration
./scripts/resources/index.sh --action validate
```

**Dependencies not met:**
```bash
# Check dependency order in scenario.json
jq '.dependencies' ~/.vrooli/scenarios.json

# Start required resources
./scripts/resources/index.sh --action start
```

### Debug Mode

Enable verbose output for troubleshooting:
```bash
DEBUG=1 ./tools/generate-from-template.sh --template ecommerce
```

### Getting Help

1. Check template README: `templates/<name>/README.md`
2. Review logs: `~/.vrooli/logs/injection.log`
3. Validate scenario: `./tools/validate-template.sh`
4. Test in isolation: `./tools/test-scenario.sh`

## Advanced Usage

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
- name: Deploy Scenario
  run: |
    ./tools/generate-from-template.sh \
      --template ${{ env.TEMPLATE }} \
      --vars production.env \
      --output scenario.json
    
    ./scripts/resources/index.sh --action inject-all
```

### Dynamic Scenarios

Generate scenarios programmatically:
```javascript
const scenario = generateScenario({
  template: 'saas-multitenant',
  tenants: ['acme', 'globex'],
  features: ['billing', 'analytics']
});

fs.writeFileSync('scenario.json', JSON.stringify(scenario));
```

### Multi-Environment

Maintain separate scenarios per environment:
```bash
scenarios/
├── development.json
├── staging.json
└── production.json
```

## Resources

- [Injection Engine Documentation](../injection/README.md)
- [Resource API Reference](../README.md)
- [Schema Validation](../injection/config/schema.json)
- [Example Scenarios](examples/)

## Support

For issues or questions:
1. Check the troubleshooting guide above
2. Review template-specific README
3. Consult resource documentation
4. Open an issue with detailed logs