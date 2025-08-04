# Vrooli Scenario System

> **Dual-Purpose Architecture: Validation Testing + App Generation**

## 🎯 **What Makes This Revolutionary**

Every scenario serves **TWO critical functions**:

1. **Validation Testing**: Integration tests ensuring resources work together correctly
2. **App Generation**: Complete blueprints for deployable customer applications ($10k-50k projects)

This dual-purpose design means when you create a scenario, you're simultaneously:
- Creating tests that validate resource integration
- Building a template for revenue-generating applications
- Enabling one-shot AI generation from customer requirements

📚 **[Read the full improvement guide →](IMPROVED_SCENARIO_STRUCTURE.md)**

## 🎯 What Are Scenarios?

**Scenarios are declarative templates that enable Vrooli to generate complete, deployable SaaS applications from customer requirements.** Each scenario represents the minimal set of files needed to validate, test, and deploy a specific type of business application.

### The Revolutionary Approach

Instead of building every feature into one monolithic platform, Vrooli uses scenarios to:
- **Generate Apps on Demand**: AI creates complete applications from customer specifications  
- **Validate Business Models**: Integration tests prove each scenario works before deployment
- **Deploy Efficiently**: Only required resources are deployed, not the entire Vrooli stack
- **Scale Profitably**: Serve both Upwork clients and self-initiated business ideas

### Business Impact
- **Speed**: Customer requirements → working app in one AI conversation
- **Reliability**: Standardized templates ensure consistent, testable results  
- **Efficiency**: Resource orchestration deploys only what's needed
- **Profitability**: Validated scenarios targeting $10K-$25K project values

---

## 🚀 Major Update: New Declarative Testing Framework

We've completely rewritten our testing infrastructure achieving **95% code reduction**:
- **Before**: 8,622 lines of imperative bash across 13 scenarios
- **After**: 440 lines + declarative YAML configurations
- **Result**: Tests are now AI-generatable, maintainable, and business-focused

### Migration Complete
All scenarios have been migrated to the new framework:
```bash
# Old format: 600-1000 lines of bash per scenario
# New format: 34 lines + YAML configuration

# Example reduction: image-generation-pipeline
# Before: 1,300 lines → After: 34 lines (98% reduction!)
```

## 🚀 Quick Start

### For Validation Testing
```bash
# Run integration tests for a specific scenario (NEW FRAMEWORK)
cd core/multi-modal-ai-assistant
./test.sh  # Now only 34 lines instead of 1000+!

# Run all scenario tests  
for dir in core/*/; do
    echo "Testing $(basename $dir)..."
    (cd "$dir" && ./test.sh)
done

# Test scenarios using a specific resource
./tools/test-by-resource.sh --resource ollama
```

### For App Generation
```bash
# Generate a complete app from a scenario
./tools/scenario-to-app.sh \
  --scenario multi-modal-ai-assistant \
  --output ~/my-customer-app

# Generate with customization
./tools/scenario-to-app.sh \
  --scenario multi-modal-ai-assistant \
  --output ~/my-app \
  --config customer-config.yaml
```

### For Developers
```bash
# 1. Explore existing scenarios
ls -la core/                             # See all available scenarios
cat catalog.yaml                         # Browse registry
cat _index/categories.yaml               # Browse by category

# 2. Create a new scenario using the unified template
cp -r templates/full/ core/my-new-scenario/
cd core/my-new-scenario/
# Edit metadata.yaml, manifest.yaml, initialization files
# Template supports both manual editing AND AI generation patterns

# 3. Test your scenario structure
./test.sh                                # Run validation tests

# 4. Deploy as application
../../tools/scenario-to-app.sh my-new-scenario
```

### For AI Generation
```bash
# Use the unified template for AI generation
cp -r templates/full/ core/ai-generated-scenario/
# → AI agents can use the built-in placeholder patterns:
#   - SCENARIO_ID_PLACEHOLDER, VALUE_PROPOSITION_PLACEHOLDER, etc.
#   - AI guidance comments throughout all files
#   - Both Jinja2 templates AND AI placeholders supported

# Deploy AI-generated scenario
./scripts/scenario-to-app.sh ai-generated-scenario
```

## 🔄 **Template Consolidation (COMPLETED)**

**✅ All scenarios now use the unified template structure:**

- **Before**: Conflicting templates scattered across different locations
- **After**: Clean organization in `templates/` directory
- **Migration**: All 13 scenarios automatically upgraded to full structure  
- **Current**: `templates/full/` (comprehensive) + `templates/basic/` (simple testing)

**Benefits**:
- 🎯 **Single source of truth** for all scenario creation
- 🚀 **Full deployment capability** - every scenario can become an app
- 🤖 **AI-friendly** - retains all optimization patterns for AI consumption
- 🔧 **Deployment ready** - `scenario-to-app.sh` works with all scenarios

## 📁 Directory Structure

```
scenarios/
├── core/                     # All working scenarios (15+ scenarios)
│   ├── multi-modal-ai-assistant/      # Voice+AI+Visual assistant ($10k-25k)
│   ├── document-intelligence-pipeline/  # Document processing ($15k-30k)
│   ├── ai-content-assistant-example/   # Content creation ($8k-20k)
│   ├── business-process-automation/    # Workflow automation ($12k-35k)
│   ├── research-assistant/             # Knowledge management ($10k-25k)
│   └── ... (10+ more scenarios)
├── templates/                # Templates for creating new scenarios
│   ├── SCENARIO_TEMPLATE/    # Primary template (use this)
│   ├── basic/               # Simple resource integration
│   ├── business/            # Customer-facing applications  
│   └── enterprise/          # Enterprise features
├── tools/                    # Management and conversion tools
│   ├── analyze-resource-usage.sh      # Resource analysis
│   ├── generate-test-suggestions.sh   # Test generation
│   └── (more tools coming)
├── docs/                     # Detailed documentation
├── _index/                   # Legacy categorization (still useful)
├── catalog.yaml              # Scenario registry (to be created)
└── README.md                # This file
```

---

## 🧭 Navigation Dashboard

| **Getting Started** | **Technical Deep Dive** | **Business Focus** |
|---|---|---|
| 📖 [Getting Started Guide](docs/getting-started.md) | 🏗️ [Architecture Overview](docs/architecture.md) | 💼 [Business Framework](docs/business-framework.md) |
| 🎯 [Template Selection Guide](docs/template-guide.md) | 🔧 [Resource Integration](docs/resource-integration.md) | 📊 [Revenue Modeling](docs/business-framework.md#revenue-modeling) |
| 🤖 [AI Generation Guide](docs/ai-generation-guide.md) | 🧪 [Testing Framework](docs/testing-framework.md) | 🚀 [Deployment Guide](docs/deployment-guide.md) |

| **Quick Reference** | **Examples & Tutorials** | **Support** |
|---|---|---|
| 📋 [Available Templates](templates/) | 💡 [Example Walkthroughs](docs/examples/) | 🔍 [Troubleshooting](docs/troubleshooting.md) |
| 📁 [Scenario Categories](_index/categories.yaml) | 🛠️ [Integration Examples](docs/examples/) | 🆘 [Common Issues](docs/troubleshooting.md) |
| 🔍 [Discovery Guide](_index/discovery.md) | 🎨 [UI Development](docs/examples/) | 📚 [Full Documentation](docs/) |

---

## 🎭 Use Cases

### 1. **AI-Generated SaaS Development** 🤖
**Primary Use Case**: AI generates complete scenarios from customer requirements
- **Input**: Customer specifications ("I need a document processing system")
- **Output**: Complete scenario with metadata, tests, UI, and deployment artifacts
- **Value**: One-shot generation of profitable applications

### 2. **Business Validation** 💼
**Testing Market Viability**: Prove scenarios work before customer delivery
- **Integration Tests**: Automated validation of all components
- **Resource Orchestration**: Verify efficient resource usage
- **Performance Validation**: Confirm scalability and reliability

### 3. **Customer Delivery** 🚀
**Production Deployment**: Convert validated scenarios to customer applications
- **Resource Optimization**: Deploy only required components
- **Customization**: Adapt scenarios to specific customer needs  
- **Professional Deployment**: Enterprise-ready applications with monitoring and support

---

## 📊 Current Ecosystem

### **Scenario Statistics**
- **Total Scenarios**: 11 validated business applications
- **UI-Enabled**: 6 scenarios with professional Windmill interfaces
- **Resource Coverage**: 15+ integrated resources (AI, automation, storage, agents)
- **Business Value**: $10K-$25K average project potential

### **Categories**
| Category | Count | Examples | Business Focus |
|----------|-------|----------|----------------|
| **AI Assistance** | 4 | Multi-modal assistant, Research assistant | Personal productivity, Enterprise automation |
| **Content Creation** | 3 | Image generation, Podcast transcription | Creative agencies, Marketing teams |
| **Data Analysis** | 2 | Analytics dashboard, Document intelligence | Business intelligence, Compliance |
| **Business Automation** | 2 | Process automation, Resume screening | HR departments, Operations teams |

### **Resource Integration**
| Category | Resources | Scenarios Using |
|----------|-----------|-----------------|
| **🧠 AI** | Ollama, Whisper, ComfyUI, Unstructured-IO | 9/11 |
| **⚙️ Automation** | n8n, Windmill, Node-RED, Huginn | 8/11 |
| **🤖 Agents** | Agent-S2, Browserless, Claude-Code | 5/11 |
| **💾 Storage** | PostgreSQL, MinIO, Qdrant, Redis, Vault | 7/11 |
| **🔍 Search** | SearXNG | 2/11 |

---

## 🏗️ Architecture Philosophy

### **Improved Scenario-to-App Structure**
The enhanced scenario structure enables seamless conversion from validation tools to deployable applications:

```
scenario-name/
├── metadata.yaml              # Business/scenario metadata (existing)
├── manifest.yaml              # 🆕 Deployment orchestration
├── initialization/            # 🆕 App startup data
│   ├── database/              # Schema and seed data
│   ├── workflows/             # n8n, Windmill, triggers
│   ├── configuration/         # Runtime settings
│   ├── ui/                    # Windmill applications
│   └── storage/               # MinIO, Qdrant setup
└── deployment/                # 🆕 Orchestration scripts
    ├── startup.sh             # App initialization  
    ├── validate.sh            # Pre/post validation
    └── monitor.sh             # Health monitoring
```

### **Capability Emergence Through Orchestration**
Scenarios don't contain business logic—they orchestrate external resources to create emergent capabilities:

- **AI Resources**: Local models (Ollama), speech processing (Whisper), document analysis (Unstructured-IO)
- **Automation Platforms**: Visual workflows (n8n), real-time processing (Node-RED), code execution (Windmill)
- **Agent Services**: Screen automation (Agent-S2), web automation (Browserless)
- **Storage Solutions**: Databases (PostgreSQL), object storage (MinIO), vector search (Qdrant)

### **Three-Tier Integration**
Scenarios integrate with Vrooli's AI architecture:
- **Tier 1**: Coordination Intelligence (scenario selection and planning)
- **Tier 2**: Process Intelligence (resource orchestration)  
- **Tier 3**: Execution Intelligence (direct resource interaction)

### **Scenario-to-App Conversion**
The new `./scripts/scenario-to-app.sh` script converts any scenario into a running application:
1. **Validation Phase**: Structure and resource health checks
2. **Configuration Phase**: Generate efficient resource configuration
3. **Deployment Phase**: Database setup, workflow deployment, UI activation
4. **Monitoring Phase**: Health monitoring and integration testing

---

## 🎯 For AI Developers

### **Optimal AI Generation Patterns**
Scenarios are designed for reliable AI generation:

```yaml
# metadata.yaml - AI-friendly structure
scenario:
  id: customer-service-assistant
  complexity: intermediate
  
resources:
  required: ["ollama", "n8n", "postgres"]
  optional: ["whisper", "agent-s2"]
  
business:
  value_proposition: "Automated customer service with 90% issue resolution"
  target_market: ["e-commerce", "saas", "service-businesses"]
  revenue_range: { min: 15000, max: 25000, currency: "USD" }
```

### **AI Generation Guidelines**
- **Atomic Components**: Each scenario is self-contained and testable
- **Resource Precision**: Exact resource requirements enable efficient deployment
- **Business Focus**: Clear value propositions and target markets
- **Validation Built-in**: Integration tests prove functionality

📖 **Full AI Generation Guide**: [docs/ai-generation-guide.md](docs/ai-generation-guide.md)

---

## 🔧 Template Selection

### **Choose Your Template**

| Template | Use Case | Complexity | Features | AI-Generation Ready |
|----------|----------|------------|----------|-------------------|
| [**SCENARIO_TEMPLATE/**](templates/SCENARIO_TEMPLATE/) | Complete app blueprint | ⭐⭐ Moderate | Full deployment orchestration | ✅ Optimized |
| [**basic/**](templates/basic/) | Resource integration testing | ⭐ Simple | Basic structure only | ✅ Yes |
| [**business/**](templates/business/) | Customer-facing applications | ⭐⭐ Moderate | Business features | ✅ Yes |
| [**enterprise/**](templates/enterprise/) | Full enterprise features | ⭐⭐⭐ Advanced | Enterprise capabilities | 🔄 In Progress |

**🎯 Recommended**: Use `templates/SCENARIO_TEMPLATE/` for all new scenarios - it includes the complete deployment orchestration layer for seamless scenario-to-app conversion.

📋 **Detailed Template Guide**: [docs/template-guide.md](docs/template-guide.md)

---

## 📈 Success Metrics

### **Scenario Quality Indicators**
- ✅ **Integration Tests Pass**: All resources work together seamlessly
- ✅ **Resource Efficiency**: Minimal resource usage for maximum capability
- ✅ **Business Viability**: Clear value proposition and revenue potential
- ✅ **AI-Generation Ready**: Structured for reliable AI consumption
- ✅ **Deployment Ready**: Converts to production app with minimal effort

### **Business Impact Tracking**
- **Time to Market**: Customer requirements → deployed app
- **Resource Efficiency**: Cost per deployed application
- **Success Rate**: Percentage of scenarios that become profitable projects
- **Customer Satisfaction**: Application quality and performance metrics

---

## 🆘 Getting Help

### **Quick Solutions**
| Problem | Solution |
|---------|----------|
| 🚫 "I don't know where to start" | → [Getting Started Guide](docs/getting-started.md) |
| 🤖 "How do I make scenarios AI-friendly?" | → [AI Generation Guide](docs/ai-generation-guide.md) |
| 🔌 "Resource integration isn't working" | → [Resource Integration Guide](docs/resource-integration.md) |
| 🧪 "Tests are failing" | → [Testing Framework](docs/testing-framework.md) |
| 🚀 "How do I deploy scenarios?" | → [Deployment Guide](docs/deployment-guide.md) |

### **Advanced Support**
- 📚 **Complete Documentation**: [docs/](docs/) directory
- 🔍 **Troubleshooting Database**: [docs/troubleshooting.md](docs/troubleshooting.md)
- 💡 **Example Tutorials**: [docs/examples/](docs/examples/)

---

## 🚀 The Vision

**Vrooli scenarios represent the future of software development**: AI agents that can reliably generate profitable, deployable applications from simple customer requirements. By standardizing the patterns and validating the integrations, we've created a foundation for scalable, AI-driven software delivery.

**Every scenario is a proof point** that Vrooli can generate valuable business applications. **Every successful deployment** validates our approach to AI-powered software development.

---

*Ready to build the future of AI-generated SaaS? Start with the [Getting Started Guide](docs/getting-started.md) or explore [existing scenarios](_index/categories.yaml) for inspiration.*