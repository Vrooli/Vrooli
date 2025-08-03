# Scenario Generator V1

## 🎯 Overview

The **Scenario Generator V1** is a meta-scenario that uses Vrooli's own platform to generate new scenarios from customer requirements. This revolutionary tool transforms simple text descriptions into complete, deployable SaaS applications worth $10,000-$50,000 each.

### Key Features
- **One-Shot Generation**: Create complete scenarios from a single prompt
- **AI-Powered**: Uses Claude Code to generate production-ready code
- **Self-Validating**: Tests and validates generated scenarios
- **Revenue-Optimized**: Estimates realistic revenue potential for each scenario
- **Resource-Efficient**: Selects only necessary resources for each use case

## 🚀 Quick Start

### Prerequisites
Ensure these resources are installed and running:
```bash
# Check resource status
./scripts/resources/index.sh --action discover

# Required resources:
- windmill (UI interface)
- n8n (workflow orchestration)
- claude-code (AI generation)
- postgres (data storage)
- minio (file storage)
```

### Deployment

1. **Deploy the scenario**:
```bash
# Convert scenario to running application
./scripts/scenario-to-app.sh \
  --scenario scenario-generator-v1 \
  --environment development \
  --auto-start true
```

2. **Access the UI**:
```
http://localhost:5681  # Windmill interface
```

3. **Start generating scenarios**:
- Navigate to the "Generate New Scenario" page
- Enter customer requirements
- Click "Generate Scenario"
- View and manage generated scenarios in the list page

## 📁 Structure

```
scenario-generator-v1/
├── metadata.yaml                 # Scenario configuration
├── workflows/
│   └── n8n-scenario-generator.json  # Generation workflow
├── ui/
│   └── windmill/
│       ├── idea-input.tsx       # Input interface
│       └── scenario-list.tsx    # Scenario management
├── prompts/
│   └── scenario-generation-prompt.md  # Claude Code prompt
├── initialization/
│   ├── database/
│   │   └── schema.sql           # PostgreSQL schema
│   └── configuration/
│       └── config.json          # Runtime config
└── deployment/
    └── startup.sh               # Deployment script
```

## 💡 Usage Examples

### Example 1: Customer Support Bot
**Input**: "I need a customer support chatbot that handles returns, tracks orders, and answers FAQs for my e-commerce store"

**Generated Resources**: ollama, n8n, postgres, redis  
**Revenue Potential**: $15,000-$25,000

### Example 2: Document Processing
**Input**: "Build a system that extracts data from invoices and automatically enters it into QuickBooks"

**Generated Resources**: unstructured-io, n8n, postgres, minio, windmill  
**Revenue Potential**: $20,000-$35,000

### Example 3: Content Generator
**Input**: "Create a social media content generator that creates posts with images based on blog articles"

**Generated Resources**: ollama, comfyui, n8n, minio, postgres  
**Revenue Potential**: $18,000-$30,000

## 🔧 Configuration

### Environment Variables
Set in `initialization/configuration/config.json`:
```json
{
  "claude_code": {
    "api_key": "your-api-key",
    "model": "claude-3-opus",
    "max_tokens": 8000
  },
  "storage": {
    "minio_bucket": "generated-scenarios",
    "postgres_db": "scenario_generator"
  }
}
```

### Resource Configuration
Modify `metadata.yaml` to adjust resources:
```yaml
resources:
  required:
    - windmill
    - n8n
    - claude-code
    - postgres
    - minio
  optional:
    - ollama  # For local validation
    - redis   # For caching
```

## 📊 Database Schema

The PostgreSQL database tracks all generated scenarios:

```sql
CREATE TABLE scenario_generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_input TEXT NOT NULL,
    generated_at TIMESTAMP DEFAULT NOW(),
    scenario_id VARCHAR(100) UNIQUE NOT NULL,
    scenario_name VARCHAR(255) NOT NULL,
    resources_required JSONB,
    status VARCHAR(50) DEFAULT 'generating',
    generation_time_ms INTEGER,
    complexity VARCHAR(20),
    category VARCHAR(50),
    estimated_revenue JSONB,
    storage_path VARCHAR(500),
    metadata JSONB
);
```

## 🔄 Workflow Process

1. **User Input** → Windmill UI receives requirements
2. **Webhook Trigger** → n8n workflow activated
3. **Prompt Enhancement** → Documentation references added
4. **Claude Code Call** → AI generates complete scenario
5. **File Storage** → Scenario saved to MinIO
6. **Database Update** → PostgreSQL records generation
7. **Response** → User receives scenario details

## 🧪 Testing

### Manual Testing
```bash
# Test the generation workflow
curl -X POST http://localhost:5678/webhook/scenario-generator \
  -H "Content-Type: application/json" \
  -d '{
    "customerInput": "Build a customer support chatbot",
    "complexity": "intermediate",
    "category": "customer-service"
  }'
```

### Validation Testing
```bash
# Validate a generated scenario
curl -X POST http://localhost:5678/webhook/validate-scenario \
  -H "Content-Type: application/json" \
  -d '{"scenarioId": "scn-xxxxx"}'
```

## 📈 Metrics & Monitoring

Track key metrics:
- **Total Scenarios Generated**: Count of all generations
- **Success Rate**: Percentage of successful generations
- **Average Generation Time**: Time from request to completion
- **Resource Usage**: Most commonly selected resources
- **Revenue Potential**: Total estimated revenue from scenarios

Access metrics:
```sql
-- Connect to PostgreSQL
psql -h localhost -p 5433 -U postgres -d scenario_generator

-- View generation statistics
SELECT 
    COUNT(*) as total_scenarios,
    AVG(generation_time_ms) as avg_time_ms,
    SUM((estimated_revenue->>'max')::int) as total_revenue_potential
FROM scenario_generations
WHERE status = 'completed';
```

## 🚨 Troubleshooting

### Common Issues

**1. Claude Code Not Responding**
```bash
# Check Claude Code status
claude-code --version

# Verify API configuration
cat ~/.vrooli/claude-code/config.json
```

**2. n8n Workflow Not Triggering**
```bash
# Check n8n status
curl http://localhost:5678/healthz

# View workflow logs
docker logs n8n
```

**3. Storage Issues**
```bash
# Check MinIO connectivity
curl http://localhost:9000/minio/health/live

# Verify bucket exists
mc ls minio/generated-scenarios
```

## 🔮 Future Enhancements (V2 and Beyond)

- **AI Model Selection**: Choose between Claude, GPT-4, or local models
- **Template Library**: Pre-built scenario templates for common use cases
- **Deployment Pipeline**: Direct deployment to Kubernetes
- **Customer Portal**: Self-service scenario generation for clients
- **Version Control**: Git integration for scenario tracking
- **A/B Testing**: Multiple scenario variations for optimization
- **Marketplace**: Share and sell scenarios to other users
- **Analytics Dashboard**: Real-time generation and deployment metrics

## 📚 Additional Resources

- [Vrooli Scenario Documentation](/scripts/scenarios/README.md)
- [Resource Ecosystem Guide](/scripts/resources/README.md)
- [Scenario-to-App Conversion](/scripts/scenario-to-app.sh)
- [Integration Patterns](/docs/architecture/ai-resource-integration-plan.md)

## 🤝 Contributing

To improve the scenario generator:
1. Test with diverse customer requirements
2. Document successful patterns
3. Report issues and edge cases
4. Suggest prompt improvements
5. Add validation rules

## 📝 License

Part of the Vrooli platform - proprietary scenario generation system.

---

**Remember**: Each scenario generated can produce $10,000-$50,000 in revenue. Quality matters!