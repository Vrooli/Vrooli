# Scenario Creator's Guide

> **Tutorial: Build your first $10K-50K business application**

📍 **Navigation**: [Main Docs](../README.md) → [Getting Started](../GETTING_STARTED.md) → **Scenario Creation Tutorial**

⚠️ **Prerequisites**: Complete the [Main Getting Started Guide](../GETTING_STARTED.md) first and choose "Path 2: Create New Scenarios"

**This guide assumes you have:**
- ✅ Vrooli installed and setup complete
- ✅ Resources running (`vrooli resource status` shows healthy)  
- ✅ Successfully run at least one existing scenario
- ✅ Understanding of direct execution (no build steps)

> 📚 **[Back to Scenario Documentation](README.md)** | **[Main Getting Started](../GETTING_STARTED.md)**

## 🎯 What You'll Learn

This tutorial teaches you to:
- Create scenarios from templates in 15 minutes
- Configure resource orchestration for business value
- Write integration tests that validate deployment readiness
- Structure scenarios for AI generation compatibility

**Time Required**: 30-45 minutes  
**Outcome**: Your first custom revenue-generating scenario

---

## Step 1: Understanding Scenario Anatomy

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
  "dependencies": {
    "resources": {
      "ollama": {"type": "ai", "enabled": true, "required": true},
      "postgres": {"type": "database", "enabled": true, "required": true},
      "whisper": {"type": "ai", "enabled": true, "required": false}
    }
  },
  "spec": {
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

## Step 2: Create Your First Scenario

### Choose Your Template

Vrooli currently ships a single, production-ready template optimized for React + TypeScript + Vite + shadcn/ui + lucide (UI) with a Go API backend: `scripts/scenarios/templates/react-vite/`. Always start from an existing template rather than inventing a new one. We may add new ones in the future if the need arises.

```bash
# 1. Inspect the available templates
vrooli scenario template list
vrooli scenario template show react-vite

# 2. Generate a new scaffold (fills placeholders automatically)
vrooli scenario generate react-vite \
  --id my-customer-portal \
  --display-name "My Customer Portal" \
  --description "Self-service customer portal with AI chat support"

cd scenarios/my-customer-portal/

# 3. Install UI dependencies (pnpm ships with the repo)
pnpm install --dir ui

# 4. Examine the template structure
ls -la
# .vrooli/         # Lifecycle + health metadata
# api/             # Go API skeleton
# cli/             # CLI installer + tests
# docs/            # PROGRESS.md starter
# requirements/    # Requirement registry seed
# ui/              # React + Vite front-end
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
  "dependencies": {
    "resources": {
      "ollama": {"type": "ai", "enabled": true, "required": true},      // ✏️ Add/remove as needed
      "postgres": {"type": "database", "enabled": true, "required": true},
      "whisper": {"type": "ai", "enabled": true, "required": false},     // ✏️ Optional resources
      "browserless": {"type": "agent", "enabled": true, "required": false}
    }
  },
  "spec": {
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

## Step 3: Test Your Scenario

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

## Step 4: Validate Business Value

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

## Step 5: Prepare for AI Generation

### AI-Friendly Structure
Ensure your scenario is optimized for AI consumption:

```json
// service.json - AI-readable structure
{
  "metadata": {
    "name": "descriptive-kebab-case",           // Clear, descriptive naming
    "description": "Specific, actionable description"  // No ambiguity
  },
  "dependencies": {
    "resources": {
      "ollama": {"type": "ai", "enabled": true, "required": true}    // Exact resource specifications
    },
    "conflicts": ["browserless"]              // Prevent resource conflicts
  },
  "spec": {
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

## Next Steps

### Immediate Actions
1. **Create Your First Scenario**: Follow this guide with a simple business idea
2. **Test Integration**: Ensure all resources work together
3. **Document Business Value**: Write compelling README with revenue justification

### Continue Learning
- [AI Generation Guide](ai-generation-guide.md) - Optimize for AI scenario creation
- [Validation Framework](VALIDATION.md) - Deep dive into testing
- [Deployment Guide](DEPLOYMENT.md) - Production deployment strategies
- [Main Documentation](../README.md) - Complete platform documentation

---

## Success Checklist

Before moving to production, ensure you can:

- [ ] Create a scenario from template in under 15 minutes
- [ ] Run integration tests successfully with `./test.sh`
- [ ] Explain the specific business value proposition
- [ ] Identify all required resources and their roles
- [ ] Structure service.json for AI-generation compatibility

**Congratulations!** You're now ready to create profitable business applications with Vrooli's scenario system.

---

*Ready for advanced topics? Continue with the [AI Generation Guide](ai-generation-guide.md) to learn how to optimize scenarios for AI creation.*
