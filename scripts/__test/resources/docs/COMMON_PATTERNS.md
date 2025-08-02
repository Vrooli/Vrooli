# 📋 Common Testing Patterns - Copy & Paste Examples

**Real-world examples for frequent testing scenarios. Just copy, paste, and modify as needed.**

## 🎯 Quick Reference Commands

### **Basic Testing**
```bash
# Test everything that's running
./run.sh

# Test just AI services
./run.sh --resource ollama --resource whisper

# Test with detailed output
./run.sh --verbose

# Test and stop on first failure
./run.sh --fail-fast

# Generate JSON for CI/CD
./run.sh --output-format json > test-results.json
```

### **Service-Specific Testing**
```bash
# Individual AI services
./run.sh --resource ollama
./run.sh --resource whisper  
./run.sh --resource comfyui

# Individual automation services
./run.sh --resource n8n
./run.sh --resource node-red
./run.sh --resource windmill

# Individual storage services
./run.sh --resource minio
./run.sh --resource postgres
./run.sh --resource redis

# Individual agent services
./run.sh --resource agent-s2
./run.sh --resource browserless
```

### **Scenario Testing**
```bash
# Test business scenarios only
./run.sh --scenarios-only

# Test AI-focused scenarios
./run.sh --scenarios "category=ai-assistance"

# Test specific complexity levels
./run.sh --scenarios "complexity=beginner"
./run.sh --scenarios "complexity=intermediate"
./run.sh --scenarios "complexity=advanced"

# Test high-value scenarios
./run.sh --scenarios "revenue_potential=high"

# List all available scenarios
./run.sh --list-scenarios
```

---

## 🔧 Development & Debugging Patterns

### **Debugging Failed Tests**
```bash
# Run with maximum debugging
./run.sh --resource your-service --debug --verbose

# Check if service is actually running
curl http://localhost:YOUR_PORT/health

# Check resource discovery
../index.sh --action discover

# Test with longer timeout
./run.sh --resource slow-service --timeout 900
```

### **Creating New Tests**
```bash
# 1. Copy an existing test as template
cp single/ai/ollama.test.sh single/ai/your-service.test.sh

# 2. Edit the key variables
# TEST_RESOURCE="your-service"
# BASE_URL="http://localhost:YOUR_PORT"

# 3. Test your new test
./run.sh --resource your-service --verbose

# 4. Add to the appropriate category directory:
# single/ai/          - AI and ML services
# single/automation/  - Workflow automation
# single/agents/      - Interactive agents  
# single/storage/     - Data storage
# single/search/      - Search services
```

### **CI/CD Integration**
```bash
# Basic CI/CD test command
./run.sh --output-format json --fail-fast --timeout 600 > results.json

# Check exit code in scripts
if ./run.sh --single-only --fail-fast; then
    echo "✅ All tests passed"
else
    echo "❌ Tests failed"
    exit 1
fi

# Parse JSON results
jq '.summary.passed_tests' results.json
jq '.summary.failed_tests' results.json
jq '.failed_test_names[]' results.json
```

---

## 🎭 Business Scenario Patterns

### **Complete Workflow Testing**
```bash
# Multi-modal AI assistant (voice + visual + automation)
./run.sh --scenarios "multi-modal-ai-assistant"

# Document processing pipeline
./run.sh --scenarios "document-intelligence-pipeline"

# Research assistant workflow
./run.sh --scenarios "research-assistant"

# Business process automation
./run.sh --scenarios "business-process-automation"
```

### **Integration Testing**
```bash
# Test specific resource combinations
./run.sh --scenarios "services=ollama,minio"
./run.sh --scenarios "services=n8n,postgres"
./run.sh --scenarios "services=whisper,ollama,comfyui"

# Test by market value
./run.sh --scenarios "market_demand=high"
./run.sh --scenarios "business_value=intelligent-automation"
```

---

## 💾 File Management Patterns

### **Test Data Organization**
```bash
# Audio test files
fixtures/audio/
├── speech_sample.ogg      # General speech
├── music_classical.mp3    # Music samples
├── silent_5sec.wav        # Silence testing
└── speech_test_short.mp3  # Quick tests

# Document test files  
fixtures/documents/
├── samples/census_income_report.pdf    # Real-world PDF
├── structured/customers.csv            # Structured data
├── office/pdf/                         # Office documents
└── international/chinese_technical.txt # Multi-language

# Image test files
fixtures/images/
├── real-world/nature/nature-landscape.jpg  # Real photos
├── synthetic/colors/large-red.png          # Generated images
├── ocr/images/1_simple_text.png            # OCR testing
└── dimensions/small/small-green.jpg        # Size testing
```

### **Workflow Configuration**
```bash
# ComfyUI workflows
fixtures/workflows/comfyui/
├── comfyui-text-to-image.json      # Basic image generation
└── comfyui-ollama-guided.json     # AI-guided generation

# n8n workflows
fixtures/workflows/n8n/
├── n8n-workflow.json              # General automation
└── n8n-whisper-transcription.json # Audio processing

# Node-RED flows
fixtures/workflows/node-red/
├── node-red-workflow.json         # Event-driven flows
└── node-red-minio-storage.json    # Storage integration
```

---

## 🔍 Resource Discovery Patterns

### **Health Checking**
```bash
# Check what's currently running
../index.sh --action discover

# Check specific resource status
../ai/ollama/manage.sh --action status
../automation/n8n/manage.sh --action status
../storage/minio/manage.sh --action status

# Check enabled resources in config
cat ~/.vrooli/resources.local.json | jq '.services'

# Health check with timeout
timeout 30s ../index.sh --action discover
```

### **Service Management**
```bash
# Start specific resources
../index.sh --action install --resources "ollama,n8n"

# Start by category
../index.sh --action install --resources "ai-only"
../index.sh --action install --resources "automation-only"
../index.sh --action install --resources "essential"

# Check resource logs
../ai/ollama/manage.sh --action logs
../automation/n8n/manage.sh --action logs
```

---

## 🚀 Performance Testing Patterns

### **Benchmark Testing**
```bash
# Basic performance test
./run.sh --resource ollama --verbose | grep "took:"

# Business scenario performance
./run.sh --scenarios "multi-modal-ai-assistant" --verbose

# Timeout testing for slow services
./run.sh --resource slow-service --timeout 1800  # 30 minutes
```

### **Load Testing Simulation**
```bash
# Run tests multiple times
for i in {1..5}; do
    echo "Test run $i"
    ./run.sh --resource ollama --output-format json > "run_${i}.json"
done

# Parallel resource testing
./run.sh --resource ollama &
./run.sh --resource n8n &
./run.sh --resource minio &
wait  # Wait for all background jobs
```

---

## 🎯 Interface Compliance Patterns

### **Category-Specific Validation**
```bash
# Test interface compliance only
./run.sh --interface-only

# Skip interface compliance for development
./run.sh --skip-interface

# Test specific category compliance
# (Modify framework files for this - see templates)
```

### **Capability Testing**
```bash
# These are internal framework patterns used by the test system:

# AI capability validation
# - model_management: Can list and load models
# - inference: Can generate responses
# - health_monitoring: Detailed health status

# Storage capability validation  
# - data_persistence: Read/write operations
# - statistics_reporting: Usage stats
# - backup_restore: Data safety

# Automation capability validation
# - web_interface: UI accessibility
# - workflow_management: CRUD operations
# - trigger_test: Event handling
```

---

## 📊 Reporting Patterns

### **Human-Readable Reports**
```bash
# Standard text output
./run.sh > test-report.txt

# Verbose detailed report
./run.sh --verbose > detailed-report.txt

# Business readiness assessment
./run.sh --business-report > business-assessment.txt
```

### **Machine-Readable Reports**
```bash
# JSON for analysis
./run.sh --output-format json > results.json

# Extract key metrics
jq '.summary' results.json
jq '.failed_test_names[]' results.json
jq '.passed_tests' results.json

# Business scenario results
jq '.scenarios[] | select(.status=="passed")' results.json
jq '.scenarios[] | .revenue_potential' results.json
```

---

## 🎨 Custom Test Patterns

### **Simple Health Check Test Template**
```bash
#!/bin/bash
# Template for basic health check test

set -euo pipefail

# Test metadata
TEST_RESOURCE="your-service"
TEST_TIMEOUT="${TEST_TIMEOUT:-60}"

# Setup
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
require_resource "$TEST_RESOURCE"

# Test implementation
test_service_health() {
    echo "🏥 Testing service health..."
    
    local response
    response=$(curl -s --max-time 10 "http://localhost:YOUR_PORT/health")
    
    assert_http_success "$response" "Service responds"
    assert_not_empty "$response" "Response not empty"
    
    echo "✓ Health check passed"
}

# Main execution
main() {
    echo "🧪 Testing $TEST_RESOURCE"
    test_service_health
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
```

### **Multi-Service Integration Test Template**
```bash
#!/bin/bash
# Template for testing multiple services together

set -euo pipefail

# Required services
REQUIRED_RESOURCES=("service1" "service2" "service3")
TEST_TIMEOUT="${TEST_TIMEOUT:-300}"

# Setup
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
require_resources "${REQUIRED_RESOURCES[@]}"

# Test implementation
test_service_integration() {
    echo "🔗 Testing service integration..."
    
    # Test service 1
    local response1
    response1=$(curl -s "http://localhost:PORT1/api/data")
    assert_json_valid "$response1" "Service 1 JSON"
    
    # Use service 1 output in service 2
    local input_data
    input_data=$(echo "$response1" | jq -r '.data')
    
    local response2
    response2=$(curl -s -X POST "http://localhost:PORT2/process" \
        -H "Content-Type: application/json" \
        -d "{\"input\": \"$input_data\"}")
    assert_json_valid "$response2" "Service 2 JSON"
    
    # Verify integration
    assert_json_field "$response2" ".result" "Integration result"
    
    echo "✓ Integration test passed"
}

main() {
    echo "🧪 Testing Multi-Service Integration"
    test_service_integration
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        exit 1
    fi
}

main "$@"
```

---

## 💡 Pro Tips & Best Practices

### **Writing Effective Tests**
1. **Copy existing patterns** - Use `ollama.test.sh` as your gold standard
2. **Test real functionality** - Don't just ping endpoints, test actual features
3. **Include error cases** - Test what happens when things go wrong
4. **Use helpful messages** - Make assertion messages descriptive
5. **Clean up resources** - Always register cleanup handlers

### **Debugging Test Issues**
1. **Start simple** - Test one service first before complex scenarios
2. **Use verbose mode** - Always debug with `--verbose --debug`
3. **Check service health** - Run `../index.sh --action discover` first
4. **Verify manually** - Test API endpoints with curl manually first
5. **Check timeouts** - Some services need longer than default 300s

### **CI/CD Integration**
1. **Use JSON output** - Machine readable for automation
2. **Set appropriate timeouts** - CI environments can be slower
3. **Fail fast** - Stop on first failure to save time
4. **Cache results** - Save test artifacts for debugging
5. **Monitor trends** - Track test performance over time

---

**🎉 Ready to Test!** These patterns cover 90% of common testing scenarios. Mix and match as needed for your specific use case!