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

## 🚀 Major Updates

### Direct Execution Architecture (NEW!)
Scenarios now run directly from their source location:
- **Leverages Proven Systems**: Uses existing manage.sh and lib/inject.sh scripts
- **No Conversion Needed**: Runs directly from scenarios/ folder
- **One-Shot Experience**: Customer requirements → running app in minutes
- **Battle-Tested**: Built on proven resource management from scripts/resources/

### Declarative Testing Framework
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
# Run structure checks for a scenario
test-genie execute my-scenario --phases structure --fail-fast

# Run full suite (quick preset)
test-genie execute my-scenario --preset quick --fail-fast
```

### For App Generation
```bash
# Run scenario directly using existing resource infrastructure
vrooli scenario run campaign-content-studio

# Run with verbose output
vrooli scenario run research-assistant --verbose

# Preview what would be done without executing
vrooli scenario run research-assistant --dry-run

# Test a scenario
vrooli scenario test secure-document-processing
```

### For Developers
```bash
# 1. Explore existing scenarios
ls -la scripts/                             # See all available scenarios

# 2. Create a new scenario using the unified React + Vite template
vrooli scenario generate react-vite --id my-new-scenario --display-name "My New Scenario" --description "One sentence summary"
cd scenarios/my-new-scenario/
# Edit service.json and initialization files
# Template supports both manual editing AND AI generation patterns

# 3. Test your scenario structure
./test.sh                                # Run validation tests

# 4. Run directly using resource infrastructure
vrooli scenario run my-new-scenario
# Starts all required resources and runs the scenario
```

## 🔄 **Template Consolidation (COMPLETED)**

**✅ All scenarios now use the unified template structure:**

- **Before**: Conflicting templates scattered across different locations
- **After**: Clean organization in `templates/scenarios/`
- **Migration**: All scenarios automatically upgraded to the React + Vite archetype  
- **Current**: `templates/scenarios/react-vite/` (React UI + Go API) — the only supported scenario template

**Benefits**:
- 🎯 **Single source of truth** for all scenario creation
- 🚀 **Full deployment capability** - every scenario runs directly
- 🤖 **AI-friendly** - retains all optimization patterns for AI consumption
- 🔧 **Deployment ready** - scenarios execute without conversion

## 📁 Directory Structure

```
scenarios/
├── core/                     # All working scenarios
│   ├── secure-document-processing/     # Document processing with compliance
│   ├── campaign-content-studio/   # Content creation
│   ├── research-assistant/             # Knowledge management
│   └── ... (8 more scenarios)
├── ../templates/scenarios/        # Scenario templates (copy from here!)
│   └── react-vite/                # React + Vite + Go API archetype
├── tools/                    # Management tools (legacy bash retired; use test-genie)
├── injection/               # Resource injection system
│   ├── engine.sh           # Injection orchestrator
│   ├── schema-validator.sh # Configuration validation
│   └── docs/               # Injection documentation
├── docs/                    # Detailed documentation
└── README.md               # This file
```

---

## 🧭 Navigation Dashboard

| **Getting Started** | **Technical Deep Dive** | **Resource Injection** |
|---|---|---|
| 📖 [Getting Started Guide](docs/getting-started.md) | 🏗️ [Core Concepts](docs/CONCEPTS.md) | 📋 [Injection System](injection/README.md) |
| 🤖 [AI Generation Guide](docs/ai-generation-guide.md) | 🧪 [Validation Framework](docs/VALIDATION.md) | 🔧 [API Reference](injection/docs/api-reference.md) |
| 📋 [Available Templates](../templates/scenarios/) | 🚀 [Deployment Guide](docs/DEPLOYMENT.md) | 🛠️ [Adapter Development](injection/docs/adapter-development.md) |

| **Quick Reference** | **Examples & Support** |
|---|---|
| 📁 [All Scenarios](scenarios/) | 💡 [Example Walkthroughs](docs/examples/) |
| 🎯 [Template Guide](templates/README.md) | 🔍 [Troubleshooting](injection/docs/troubleshooting.md) |
| 📚 [Full Documentation](docs/) | 🆘 [Injection Support](injection/docs/) |

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
- **Total Scenarios**: 9 validated business applications
- **UI-Enabled**: 6 scenarios with professional UI interfaces
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
| **⚙️ Automation** | Node-RED, Huginn | 8/11 |
| **🤖 Agents** | Agent-S2, Browserless, Claude-Code | 5/11 |
| **💾 Storage** | PostgreSQL, MinIO, Qdrant, Redis, Vault | 7/11 |
| **🔍 Search** | SearXNG | 2/11 |

---

## 🏗️ Architecture Philosophy

### **Service-Driven Structure**
The service.json configuration enables seamless conversion from validation tools to deployable applications:

```
scenario-name/
├── service.json               # Complete configuration (metadata, resources, deployment)
├── initialization/            # App startup data
│   ├── database/              # Schema and seed data
│   ├── workflows/             # automation triggers
│   ├── configuration/         # Runtime settings
│   ├── ui/                    # Scenario UI applications
│   └── storage/               # MinIO, Qdrant setup
├── deployment/                # Orchestration scripts
│   ├── startup.sh             # App initialization  
│   └── monitor.sh             # Health monitoring
├── test.sh                    # Integration testing (optional)
└── README.md                  # Documentation (optional, only for complex scenarios)
```

### **Capability Emergence Through Orchestration**
Scenarios don't contain business logic—they orchestrate external resources to create emergent capabilities:

- **AI Resources**: Local models (Ollama), speech processing (Whisper), document analysis (Unstructured-IO)
- **Automation Platforms**: Visual workflow orchestration and real-time processing (Node-RED, Huginn)
- **Agent Services**: Screen automation (Agent-S2), web automation (Browserless)
- **Storage Solutions**: Databases (PostgreSQL), object storage (MinIO), vector search (Qdrant)

### **Three-Tier Integration**
Scenarios integrate with Vrooli's AI architecture:
- **Tier 1**: Coordination Intelligence (scenario selection and planning)
- **Tier 2**: Process Intelligence (resource orchestration)  
- **Tier 3**: Execution Intelligence (direct resource interaction)

### **Direct Scenario Execution**
Scenarios run directly from their source location using the Vrooli framework:

**Key Features:**
- **Leverages Existing Infrastructure**: Uses proven manage.sh and lib/inject.sh scripts
- **No Container Generation**: Orchestrates local resources instead of creating Docker configs
- **Battle-Tested**: Built on existing resource management that already works
- **One-Shot Experience**: Customer requirements → running app in minutes

**How It Works:**
1. **Validation Phase**: Validates scenario structure and service.json
2. **Resource Analysis**: Extracts required resources from configuration
3. **Resource Startup**: Uses existing manage.sh scripts to start each resource
4. **Data Injection**: Uses existing lib/inject.sh scripts to initialize data
5. **Application Startup**: Runs scenario-specific startup scripts
6. **Ready State**: Provides access URLs and keeps application running

**Runtime Architecture:**
```
Scenario Running State:
├── Required Resources (started via manage.sh)
│   ├── postgres (localhost:5432)
│   ├── ollama (http://localhost:11434)
│   └── ... (other resources as needed)
├── Data Injection (via lib/inject.sh)
│   ├── Database schemas and seeds
│   ├── Automation workflow definitions
│   └── Configuration files
└── Application Services
    ├── Custom startup scripts
    ├── Health monitoring
    └── Access point URLs
```

---

## 🎯 For AI Developers

### **Optimal AI Generation Patterns**
Scenarios are designed for reliable AI generation:

```json
// service.json - AI-friendly structure
{
  "metadata": {
    "name": "customer-service-assistant",
    "displayName": "Customer Service AI Assistant",
    "complexity": "intermediate"
  },
  "spec": {
    "dependencies": {
      "resources": [
        {"name": "ollama", "type": "ai", "optional": false},
        {"name": "postgres", "type": "database", "optional": false}
      ]
    },
    "business": {
      "valueProposition": "Automated customer service with 90% issue resolution",
      "targetMarket": ["e-commerce", "saas", "service-businesses"],
      "revenueRange": {"min": 15000, "max": 25000, "currency": "USD"}
    }
  }
}
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
| [**react-vite/**](templates/react-vite/) | Complete app blueprint | ⭐⭐ Moderate | Full deployment orchestration | ✅ Optimized |

**🎯 Recommended**: Always use `templates/react-vite/` for new scenarios. This is the only supported archetype and includes the complete deployment orchestration layer for direct execution with service.json.

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
| 🚀 "How do I deploy scenarios?" | → [Deployment Hub](../docs/deployment/README.md) |

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
