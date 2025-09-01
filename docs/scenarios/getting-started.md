# Getting Started with Vrooli Scenarios

> **Complete guide to creating, testing, and deploying business scenarios**

**Prerequisites**: Complete the [Quick Start Guide](../GETTING_STARTED.md) first (15 minutes)

## 🎯 What You'll Learn

By the end of this guide, you'll be able to:
- ✅ Create a complete scenario from scratch
- ✅ Test scenario integration with resources  
- ✅ Understand the business validation process
- ✅ Prepare scenarios for AI generation

**Time Required**: 30-45 minutes
**Audience**: Scenario creators building business applications

---

## 🚀 Quick Setup

### Prerequisites
```bash
# 1. Ensure you're in the Vrooli root directory
cd ${VROOLI_ROOT}

# 2. Verify scenarios directory exists
ls -la scenarios/

# 3. Check available resources
./resources/index.sh --action discover
```

### Your First Look
```bash
# Explore the scenario ecosystem
cd scenarios/

# Browse available scenarios
ls -la                                    # See all scenarios

# Check available scenarios
ls -la                                    # List all scenarios

# Look at a simple example
cd research-assistant/
cat service.json                        # Scenario configuration
cat README.md                           # Business documentation
```

---

## 🏗️ Step 1: Understanding Scenario Anatomy

Every scenario has **four core components**:

### 1. **service.json** - The Configuration Brain
```json
{
  "metadata": {
    "name": "customer-service-assistant",
    "displayName": "Customer Service Assistant",
    "description": "AI-powered customer service automation",
    "version": "1.0.0",
    "complexity": "intermediate"
  },
  "spec": {
    "dependencies": {
      "resources": [
        {"name": "ollama", "type": "ai", "optional": false},
        {"name": "n8n", "type": "automation", "optional": false},
        {"name": "postgres", "type": "database", "optional": false},
        {"name": "whisper", "type": "ai", "optional": true}
      ]
    },
    "business": {
      "valueProposition": "90% automated issue resolution",
      "targetMarkets": ["e-commerce", "saas", "service-businesses"],
      "revenueRange": {"min": 15000, "max": 25000, "currency": "USD"}
    },
    "testing": {
      "timeout": 600,
      "requiresDisplay": false
    }
  }
}
```

### 2. **README.md** - Optional Documentation
- **Only needed for complex scenarios** with special setup requirements
- **Business case information** is stored in service.json metadata section
- **Most scenarios don't need README** - service.json contains all required information

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
| [**full/**](../templates/full/) | Customer applications | ⭐⭐ Moderate |

**For this tutorial, we'll use the full template:**

```bash
# 1. Copy the full template
cp -r templates/full/ scenarios/my-customer-portal/
cd scenarios/my-customer-portal/

# 2. Examine the template structure
ls -la
# service.json     # Scenario configuration
# README.md        # Business documentation
# test.sh          # Integration test script
# deployment/      # Deployment scripts
# initialization/  # Startup data and workflows
```

### Customize Your Scenario

#### 1. **Edit service.json**
```json
{
  "metadata": {
    "name": "customer-portal",                    // ✏️ Change this
    "displayName": "Enterprise Customer Portal",  // ✏️ Change this
    "description": "Self-service customer portal with AI chat support",  // ✏️ Change this
    "version": "1.0.0",
    "complexity": "intermediate"                  // basic, intermediate, or advanced
  },
  "spec": {
    "dependencies": {
      "resources": [
        {"name": "ollama", "type": "ai", "optional": false},      // ✏️ Add/remove as needed
        {"name": "n8n", "type": "automation", "optional": false},
        {"name": "postgres", "type": "database", "optional": false},
        {"name": "whisper", "type": "ai", "optional": true},       // ✏️ Optional resources
        {"name": "browserless", "type": "agent", "optional": true}
      ]
    },
    "business": {
      "valueProposition": "50% reduction in support tickets through self-service",  // ✏️ Change this
      "targetMarkets": ["b2b-saas", "e-commerce", "service-businesses"],          // ✏️ Change this
      "revenueRange": {"min": 20000, "max": 35000, "currency": "USD"}             // ✏️ Adjust pricing
    },
    "testing": {
      "timeout": 900,                           // ✏️ Adjust based on complexity
      "requiresDisplay": false                  // Set to true if UI testing needed
    }
  }
}
```

#### 2. **README.md (Optional)**
Only create a README if your scenario has complex setup requirements not covered in service.json:
- Advanced troubleshooting steps
- Custom development workflows  
- Special deployment considerations
- For most scenarios, service.json contains all needed information

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
./resources/index.sh --action status --resources ollama

# View resource logs
resource-ollama logs

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

```json
// service.json - AI-readable structure
{
  "metadata": {
    "name": "descriptive-kebab-case",           // Clear, descriptive naming
    "description": "Specific, actionable description"  // No ambiguity
  },
  "spec": {
    "dependencies": {
      "resources": [
        {"name": "ollama", "type": "ai", "optional": false}    // Exact resource specifications
      ],
      "conflicts": ["browserless"]              // Prevent resource conflicts
    },
    "business": {
      "valueProposition": "Quantified benefit with metrics",  // Measurable outcomes
      "targetMarkets": ["specific-industries"]               // Targeted markets
    }
  }
}
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