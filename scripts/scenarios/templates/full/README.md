# {{ scenario.name }} - Unified Scenario Template

> **Complete scenario-to-app blueprint with deployment orchestration + AI generation support**

<!-- 
🔄 TEMPLATE CONSOLIDATION COMPLETED:
This template now serves BOTH manual development AND AI generation needs.

DUAL TEMPLATING APPROACH:
- For deployment orchestration: Uses Jinja2 syntax {{ variable.name }}
- For AI generation: Use PLACEHOLDER_NAME patterns (see AI guidance comments)
- AI agents should replace both placeholder types during generation

MIGRATION COMPLETED:
- All scenarios upgraded to this unified structure
- templates/ai-generation/ deprecated and merged into this template
- scenario-to-app.sh deployment script works with all scenarios
-->

## 🆕 **What's New**

This template includes the **improved scenario structure** that enables seamless conversion from scenario validation to deployable applications:

- ✅ **`service.json`** - Unified configuration with deployment orchestration
- ✅ **`initialization/`** - Complete app startup data  
- ✅ **`deployment/`** - Orchestration scripts
- ✅ **One-command deployment** via `scenario-to-app.sh`

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
- **Range**: ${{ business.revenue_potential.min | number_format }} - ${{ business.revenue_potential.max | number_format }} {{ business.revenue_potential.currency }}
- **Market Demand**: {{ business.market_demand }}
- **Pricing Model**: {{ business.revenue_potential.pricing_model }}
<!-- AI: Adjust min/max based on scenario complexity and business value -->

### **Competitive Advantage**
{{ business.competitive_advantage }}
<!-- AI: Replace with COMPETITIVE_ADVANTAGE_PLACEHOLDER -->

### **ROI Metrics**
{% for metric in business.roi_metrics %}
- {{ metric }}
{% endfor %}
<!-- AI: Replace with ROI_METRIC_1_PLACEHOLDER, ROI_METRIC_2_PLACEHOLDER -->

## 🏗️ **Architecture**

### **Required Resources**
{% for resource in resources.required %}
- **{{ resource }}**: [Purpose and integration]
{% endfor %}

### **Optional Resources**  
{% for resource in resources.optional %}
- **{{ resource }}**: [Enhancement capability]
{% endfor %}

### **System Components**
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Frontend UI   │────▶│   Workflows     │────▶│  AI Processing  │
│   (Windmill)    │     │   (n8n/etc)     │     │   (Ollama/etc)  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                │                           │
                                ▼                           ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │   Database      │     │   Storage       │
                        │  (PostgreSQL)   │     │  (MinIO/etc)    │
                        └─────────────────┘     └─────────────────┘
```

## 🚀 **Quick Start**

### **1. Deploy as Application**
```bash
# Convert scenario to running application
./scripts/scenario-to-app.sh {{ scenario.id }}

# Result: Complete application with UI, workflows, database, monitoring
```

### **2. Access Application**
After successful deployment:
- **UI Application**: http://localhost:5681/app/{{ scenario.id }}
- **API Endpoints**: http://localhost:3000/api/{{ scenario.id }}
- **Workflow Webhooks**: http://localhost:5678/webhook/{{ scenario.id }}-webhook

### **3. Monitor Health**
```bash
# Check application status
cd {{ scenario.id }}
./deployment/monitor.sh status

# View logs
./deployment/monitor.sh logs

# Check application status
./deployment/monitor.sh status
```

## 📁 **File Structure**

### **Core Files**
```
{{ scenario.id }}/
├── service.json               # Unified business model, configuration, and deployment
├── README.md                  # This documentation
└── test.sh                    # Integration tests
```

### **Initialization Data**
```
initialization/
├── database/
│   ├── schema.sql             # Database structure
│   └── seed.sql               # Initial data
├── workflows/
│   ├── n8n/                   # n8n workflow definitions
│   ├── windmill/              # Windmill scripts
│   └── triggers.yaml          # Workflow activation
├── configuration/
│   ├── app-config.json        # Runtime settings
│   ├── resource-urls.json     # Service endpoints
│   └── feature-flags.json     # Feature toggles
├── ui/
│   └── windmill-app.json      # Professional UI
└── storage/
    └── minio-config.json      # Object storage setup
```

### **Deployment Scripts**
```
deployment/
├── startup.sh                 # Application initialization
└── monitor.sh                # Health monitoring
```

## 🔧 **Customization Guide**

### **Business Configuration** 
Edit `service.json` metadata section:
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

### **Resource Selection**
Edit `service.json` resources section:
```json
"resources": {
  "ai": {
    "ollama": {
      "enabled": true,
      "required": true
    }
  },
  "automation": {
    "n8n": {
      "enabled": true,
      "required": true
    }
  },
  "storage": {
    "postgres": {
      "enabled": true,
      "required": true
    }
  }
}
```

### **UI Customization**
Edit `initialization/ui/windmill-app.json`:
- Update branding and colors
- Add/remove UI components
- Configure user workflows

### **Database Schema**
Edit `initialization/database/schema.sql`:
- Add business-specific tables
- Configure indexes and constraints
- Set up views and functions

### **Workflow Logic**
Edit `initialization/workflows/n8n/main-workflow.json`:
- Add business logic nodes
- Configure API integrations
- Set up data processing steps

## 🧪 **Testing & Validation**

### **Structure Validation**
```bash
# Validate scenario structure and content
./test.sh

# Check application health
./deployment/monitor.sh status
```

### **Integration Testing**
```bash
# Run full integration test
./test.sh

# Expected results:
# ✅ All resources healthy
# ✅ Database initialized
# ✅ Workflows deployed and active
# ✅ UI accessible
# ✅ End-to-end functionality working
```

## 📊 **Performance Expectations**

### **Response Times**
- **API Calls**: {{ performance.latency.p50 }} (p50), {{ performance.latency.p95 }} (p95)
- **Workflow Execution**: {{ testing.timeout_seconds }}s max timeout
- **UI Load Time**: < 2 seconds

### **Throughput**
- **Concurrent Users**: {{ performance.throughput.concurrent_users }}
- **Requests/Second**: {{ performance.throughput.requests_per_second }}

### **Resource Usage**
- **Memory**: {{ performance.resource_usage.memory }}
- **CPU**: {{ performance.resource_usage.cpu }} cores

## 🔒 **Security & Compliance**

### **Built-in Security**
- Database access controls
- API rate limiting
- Input validation
- Audit logging

### **Production Checklist**
- [ ] Change default passwords
- [ ] Configure SSL certificates  
- [ ] Set up backup procedures
- [ ] Enable monitoring alerts
- [ ] Review access permissions

## 💰 **Business Impact**

### **Revenue Model**
This scenario template targets projects in the **${{ business.revenue_potential.min | number_format }}-${{ business.revenue_potential.max | number_format }}** range with **{{ business.market_demand }}** market demand.

### **Success Criteria**
{% for criterion in success_criteria %}
- {{ criterion }}
{% endfor %}

### **ROI Metrics**
- **Implementation Time**: Hours instead of weeks
- **Resource Efficiency**: Deploy only required services
- **Professional Quality**: Enterprise-ready features included
- **Scalability**: Ready for production deployment

## 🛟 **Support & Resources**

### **Documentation**
- **[Improved Structure Guide](../IMPROVED_SCENARIO_STRUCTURE.md)**: Complete overview of new architecture
- **[Scenarios README](../README.md)**: Main scenarios documentation
- **[Resource Guide](../../README.md)**: Available resources and integration

### **Troubleshooting**
```bash
# Common issues and fixes
./deployment/startup.sh --help         # Deployment options  
./deployment/monitor.sh --help         # Monitoring options
./test.sh --help                       # Testing options
```

### **Getting Help**
- Review existing scenarios for examples
- Check resource health with discovery commands
- Use validation scripts to identify issues
- Monitor logs for detailed error information

## 🎯 **Next Steps**

### **For Development**
1. Copy this template: `cp -r templates/full/ your-scenario/`
2. Customize business configuration in `service.json`
3. Adapt initialization data for your use case
4. Test with `./test.sh`
5. Deploy with `../../../scenario-to-app.sh your-scenario`

### **For Production**
1. Review security configuration
2. Set up monitoring and alerts
3. Configure backup procedures  
4. Plan scaling strategy
5. Train users on the application

### **For AI Generation**
This template is optimized for AI agents to generate complete scenarios from customer requirements. The structure provides clear guidance for:
- Business model definition
- Resource selection
- Technical implementation
- Deployment orchestration

---

**🎉 This improved template transforms scenarios from validation tools into complete application blueprints, enabling rapid deployment of profitable SaaS applications!**