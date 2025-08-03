# Getting Started with Vrooli Scenarios

> **Your step-by-step guide to creating, testing, and deploying scenarios**

## 🎯 What You'll Learn

By the end of this guide, you'll be able to:
- ✅ Create a complete scenario from scratch
- ✅ Test scenario integration with resources  
- ✅ Understand the business validation process
- ✅ Prepare scenarios for AI generation

**Time Required**: 30-45 minutes

---

## 🚀 Quick Setup

### Prerequisites
```bash
# 1. Ensure you're in the Vrooli root directory
cd /home/matthalloran8/Vrooli

# 2. Verify scenarios directory exists
ls -la scripts/scenarios/core/

# 3. Check available resources
./scripts/resources/index.sh --action discover
```

### Your First Look
```bash
# Explore the scenario ecosystem
cd scripts/scenarios/core/

# Browse available scenarios
ls -la                                    # See all scenarios

# Check categories and organization
cat _index/categories.yaml               # Business categorization

# Look at a simple example
cd multi-resource-pipeline/
cat metadata.yaml                        # Scenario configuration
cat README.md                           # Business documentation
```

---

## 🏗️ Step 1: Understanding Scenario Anatomy

Every scenario has **four core components**:

### 1. **metadata.yaml** - The Configuration Brain
```yaml
scenario:
  id: "my-scenario"
  name: "Customer Service Assistant"
  description: "AI-powered customer service automation"
  version: "1.0.0"
  
complexity: intermediate
  
resources:
  required: ["ollama", "n8n", "postgres"]
  optional: ["whisper", "agent-s2"]
  
business:
  value_proposition: "90% automated issue resolution"
  target_market: ["e-commerce", "saas", "service-businesses"]
  revenue_range: { min: 15000, max: 25000, currency: "USD" }
  
testing:
  timeout: 600
  requires_display: false
```

### 2. **README.md** - The Business Case
- **Executive Summary**: Why this scenario matters
- **Target Market**: Who would pay for this
- **Technical Architecture**: How it works
- **Revenue Model**: Pricing and business value

### 3. **test.sh** - The Integration Validator
```bash
#!/bin/bash
source ../../../framework/helpers/test-helpers.sh

test_scenario() {
    log_info "Testing customer service assistant integration"
    
    # Test resource connectivity
    check_service_health "ollama" "http://localhost:11434"
    check_service_health "postgres" "postgresql://localhost:5432"
    
    # Test core functionality
    test_ai_response
    test_database_integration
    test_workflow_automation
    
    log_success "All integration tests passed"
}

# Run the test
test_scenario
```

### 4. **ui/** - The User Interface (Optional)
- **deploy-ui.sh**: Windmill UI deployment script
- **config.json**: UI configuration  
- **scripts/**: TypeScript backend services

---

## 🎯 Step 2: Create Your First Scenario

### Choose Your Template

| Template | Best For | Complexity |
|----------|----------|------------|
| [**basic/**](../templates/basic/) | Resource integration testing | ⭐ Simple |
| [**business/**](../templates/business/) | Customer applications | ⭐⭐ Moderate |
| [**ai-generation/**](../templates/ai-generation/) | AI-optimized scenarios | ⭐⭐ Moderate |

**For this tutorial, we'll use the business template:**

```bash
# 1. Copy the business template
cp -r templates/business/ my-customer-portal/
cd my-customer-portal/

# 2. Examine the template structure
ls -la
# metadata.yaml    # Template configuration
# README.md        # Template documentation
# test.sh          # Template test script
# ui/              # UI components (if applicable)
```

### Customize Your Scenario

#### 1. **Edit metadata.yaml**
```yaml
scenario:
  id: "customer-portal"                    # ✏️ Change this
  name: "Enterprise Customer Portal"       # ✏️ Change this
  description: "Self-service customer portal with AI chat support"  # ✏️ Change this
  version: "1.0.0"

complexity: intermediate                   # basic, intermediate, or advanced

resources:
  required: ["ollama", "n8n", "postgres"] # ✏️ Add/remove resources as needed
  optional: ["whisper", "browserless"]    # ✏️ Add/remove optional resources

business:
  value_proposition: "50% reduction in support tickets through self-service"  # ✏️ Change this
  target_market: ["b2b-saas", "e-commerce", "service-businesses"]           # ✏️ Change this
  revenue_range: { min: 20000, max: 35000, currency: "USD" }                # ✏️ Adjust pricing

testing:
  timeout: 900                            # ✏️ Adjust based on complexity
  requires_display: false                 # Set to true if UI testing needed
```

#### 2. **Update README.md**
Edit the business case to match your scenario:
- Update the executive summary
- Define your target market  
- Explain the technical approach
- Justify the revenue model

#### 3. **Implement test.sh**
```bash
#!/bin/bash
source ../../../framework/helpers/test-helpers.sh

test_customer_portal() {
    log_info "Testing customer portal integration"
    
    # Test required resources
    check_service_health "ollama" "http://localhost:11434"
    check_service_health "postgres" "postgresql://localhost:5432/vrooli"
    
    # Test AI chat functionality
    test_ollama_response "Hello, I need help with my account" || return 1
    
    # Test database operations
    test_postgres_connection || return 1
    
    # Test n8n workflow
    test_n8n_workflow_execution || return 1
    
    log_success "Customer portal integration validated"
}

# Run the test
test_customer_portal
```

---

## 🧪 Step 3: Test Your Scenario

### Run Integration Tests
```bash
# Basic test run
./test.sh

# Extended timeout for complex scenarios
TEST_TIMEOUT=1800 ./test.sh

# Debug mode (skip cleanup)
TEST_CLEANUP=false ./test.sh

# Custom resource URLs
OLLAMA_BASE_URL=http://remote:11434 ./test.sh
```

### Common Test Patterns
```bash
# Test AI functionality
test_ollama_response() {
    local prompt="$1"
    curl -s -X POST http://localhost:11434/api/generate \
        -d "{\"model\": \"llama3.1:8b\", \"prompt\": \"$prompt\"}" \
        | jq -r '.response' | grep -q "."
}

# Test database connectivity
test_postgres_connection() {
    psql -h localhost -U postgres -d vrooli -c "SELECT 1;" > /dev/null 2>&1
}

# Test n8n workflow
test_n8n_workflow_execution() {
    curl -s -X POST http://localhost:5678/webhook/test \
        -H "Content-Type: application/json" \
        -d '{"test": "data"}' | grep -q "success"
}
```

### Debugging Failed Tests
```bash
# Check resource health
./scripts/resources/index.sh --action status --resources ollama

# View resource logs
./scripts/resources/ai/ollama/manage.sh --action logs

# Test individual components
curl http://localhost:11434/api/tags          # Test Ollama
psql -h localhost -U postgres -l              # Test PostgreSQL
```

---

## 📊 Step 4: Validate Business Value

### Revenue Model Validation
Ask yourself:
- **Target Market**: Who specifically would pay for this?
- **Pain Point**: What problem does this solve worth $15K-$25K?
- **Competitive Advantage**: Why choose this over alternatives?
- **Scalability**: Can this serve multiple customers with minimal changes?

### Market Research Integration
```bash
# Future capability: Market validation
vrooli market-research --scenario customer-portal --platform upwork
# → Analyzes similar projects, pricing, demand metrics
```

### Success Criteria Checklist
- ✅ **Technical**: All integration tests pass
- ✅ **Business**: Clear value proposition for target market
- ✅ **Financial**: Realistic revenue range based on market research
- ✅ **Scalability**: Minimal customization needed for different customers
- ✅ **AI-Ready**: Structured for reliable AI generation

---

## 🎯 Step 5: Prepare for AI Generation

### AI-Friendly Structure
Ensure your scenario is optimized for AI consumption:

```yaml
# metadata.yaml - AI-readable structure
scenario:
  id: "descriptive-kebab-case"           # Clear, descriptive naming
  description: "Specific, actionable description"  # No ambiguity
  
resources:
  required: ["specific", "resources"]    # Exact resource names
  conflicts: ["mutually-exclusive"]     # Prevent resource conflicts
  
business:
  value_proposition: "Quantified benefit with metrics"  # Measurable outcomes
  target_market: ["specific-industries"]               # Targeted markets
```

### Documentation Standards
- **Clear Requirements**: Unambiguous technical specifications
- **Business Context**: Specific use cases and target markets  
- **Integration Patterns**: Reusable resource interaction patterns
- **Success Metrics**: Measurable validation criteria

### AI Generation Readiness Checklist
- ✅ **Atomic**: Self-contained with minimal external dependencies
- ✅ **Testable**: Comprehensive integration test coverage
- ✅ **Documented**: Clear, structured documentation
- ✅ **Business-Focused**: Clear value proposition and target market
- ✅ **Resource-Efficient**: Minimal required resources for maximum value

---

## 🚀 Next Steps

### Immediate Actions
1. **Create Your First Scenario**: Follow this guide with a simple business idea
2. **Test Integration**: Ensure all resources work together
3. **Document Business Value**: Write compelling README with revenue justification

### Advanced Learning
- 📖 [AI Generation Guide](ai-generation-guide.md): Optimize scenarios for AI creation
- 🏗️ [Architecture Deep Dive](architecture.md): Understand the underlying system
- 🔧 [Resource Integration](resource-integration.md): Advanced resource orchestration
- 💼 [Business Framework](business-framework.md): Revenue modeling and market validation

### Community & Support
- 🔍 [Troubleshooting Guide](troubleshooting.md): Common issues and solutions
- 💡 [Example Scenarios](examples/): Detailed walkthroughs and tutorials
- 📚 [Complete Documentation](../): Full documentation library

---

## 🎯 Success Checklist

You're ready to move forward when you can:

- [ ] **Create** a scenario from template in under 15 minutes
- [ ] **Test** scenario integration successfully  
- [ ] **Explain** the business value and target market clearly
- [ ] **Identify** required resources and their roles
- [ ] **Structure** scenarios for AI-generation readiness

**Congratulations!** You now understand the Vrooli scenario system and are ready to create profitable, AI-generated SaaS applications.

---

*Ready for advanced topics? Continue with the [AI Generation Guide](ai-generation-guide.md) to learn how to optimize scenarios for AI creation.*