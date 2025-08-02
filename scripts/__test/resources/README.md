# Vrooli Resource Integration Testing Framework

A comprehensive integration testing framework for Vrooli's resource ecosystem that validates real-world functionality without mocking dependencies.

## 🎯 Overview

This framework provides true integration testing for Vrooli resources by:
- **Resource Discovery**: Automatically discovers available resources and validates health
- **Configuration-Based Testing**: Tests only enabled and healthy resources  
- **Real Integration**: Uses actual services without mocking for authentic validation
- **Multi-Resource Workflows**: Tests complex workflows involving multiple resources
- **Comprehensive Reporting**: Detailed results with actionable recommendations

## 📚 Documentation

**New to the framework?** Start with our developer-friendly guides:

- **[Getting Started Guide](docs/GETTING_STARTED.md)** - 30-second quick start + 5-minute tour
- **[Architecture Overview](docs/ARCHITECTURE_OVERVIEW.md)** - Visual system overview and component relationships  
- **[Common Patterns](docs/COMMON_PATTERNS.md)** - Copy-paste examples for 90% of use cases
- **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)** - Solutions for common problems

## 🚀 Quick Start

### Essential Commands (Copy & Paste)

```bash
# Check if your system is ready
./validate-setup.sh

# Test a single service
./quick-test.sh ollama

# Get beginner-friendly help
./run.sh --help-beginner

# See interactive demo
./demo-capabilities.sh
```

### From Project Root (Recommended)

```bash
# Using npm/pnpm scripts from project root
cd /home/matthalloran8/Vrooli

# Run all resource tests
pnpm test:resources

# Run single-resource tests only
pnpm test:resources:single

# Run business scenarios only
pnpm test:resources:scenarios

# Quick test (30s timeout)
pnpm test:resources:quick

# Debug mode with HTTP logging
pnpm test:resources:debug

# Test specific resource
pnpm test:resources -- --resource qdrant
```

### From Test Directory

```bash
# Run all available tests
./run.sh

# Test specific resource
./run.sh --resource ollama

# Debug mode with verbose output and HTTP logging
./run.sh --debug --resource qdrant

# Use Makefile shortcuts
make test-debug R=qdrant
make test-quick
make clean
```

## 📁 Directory Structure

```
tests/
├── run.sh                 # Main test runner entry point
├── docs/                   # Developer-friendly documentation
│   ├── GETTING_STARTED.md   # 30-second quick start guide
│   ├── ARCHITECTURE_OVERVIEW.md # Visual system overview
│   ├── COMMON_PATTERNS.md   # Copy-paste examples
│   └── TROUBLESHOOTING.md   # Problem-solving guide
├── quick-test.sh           # One-command testing script
├── validate-setup.sh       # System readiness checker
├── demo-capabilities.sh    # Interactive capability showcase
├── framework/              # Core framework components
│   ├── discovery.sh        # Resource discovery and health validation
│   ├── runner.sh           # Test execution engine
│   ├── reporter.sh         # Results reporting and analysis
│   ├── helpers/            # Utility functions
│   │   ├── assertions.sh   # Test assertion functions
│   │   ├── cleanup.sh      # Test cleanup utilities
│   │   └── fixtures.sh     # Test data generation
│   └── config/             # Configuration files
│       └── timeouts.conf   # Resource-specific timeout values
├── fixtures/               # Test data and artifacts
│   ├── audio/              # Sample audio files for transcription
│   ├── images/             # Test images for vision processing
│   ├── documents/          # Sample documents and data
│   └── workflows/          # Workflow definitions for automation
├── single/                 # Single-resource tests
│   ├── ai/                 # AI service tests
│   │   ├── ollama.test.sh  # Ollama LLM testing
│   │   └── whisper.test.sh # Whisper speech-to-text testing
│   ├── automation/         # Automation platform tests
│   ├── agents/             # Agent service tests
│   ├── search/             # Search service tests
│   └── storage/            # Storage service tests
├── multi/                  # Multi-resource integration tests
│   └── ai-pipeline.test.sh # AI workflow integration (Whisper + Ollama)
└── scenarios/              # Real-world use case tests
    ├── document-processing/
    ├── content-generation/
    └── system-monitoring/
```

## 🧪 Test Categories

### Single-Resource Tests
- **Health Validation**: Service connectivity and basic functionality
- **API Testing**: Core endpoints and feature validation  
- **Error Handling**: Edge cases and failure scenarios
- **Performance**: Basic performance characteristics

### Multi-Resource Integration Tests
- **Data Flow**: Information passing between services
- **Workflow Orchestration**: Complex multi-step processes
- **Error Propagation**: How failures cascade through pipelines
- **End-to-End Scenarios**: Complete user workflows

## ⚙️ Configuration

### Resource Selection
Resources are tested if they are:
1. **Discovered**: Found by the resource discovery system
2. **Enabled**: Marked as enabled in `~/.vrooli/resources.local.json`
3. **Healthy**: Responding to health checks and ready for testing

### Timeout Configuration
Resource-specific timeouts in `framework/config/timeouts.conf`:
```bash
# AI Resources
RESOURCE_TIMEOUT_OLLAMA=60
RESOURCE_TIMEOUT_WHISPER=45

# Multi-resource workflows  
TEST_TIMEOUT_AI_PIPELINE=180
```

## 📊 Output Formats

### Text Output (Default)
Human-readable format with:
- Resource discovery summary
- Test execution progress  
- Detailed results and recommendations
- Color-coded status indicators

### JSON Output
Machine-readable format for CI/CD:
```bash
./run.sh --output-format json > test-results.json
```

## 🔧 Writing New Tests

### Single-Resource Test Template
```bash
#!/bin/bash
# Test metadata
TEST_RESOURCE="myresource"
REQUIRED_RESOURCES=("myresource")

# Source framework helpers
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# Test setup
setup_test() {
    require_resource "$TEST_RESOURCE"
}

# Test functions
test_basic_functionality() {
    # Test implementation
    assert_not_empty "$result" "Feature works"
}

# Main execution
main() {
    setup_test
    test_basic_functionality
    print_assertion_summary
}

main "$@"
```

### Multi-Resource Test Template
```bash
#!/bin/bash
# @requires: resource1 resource2
REQUIRED_RESOURCES=("resource1" "resource2")

# Test multi-resource workflow
test_workflow() {
    require_resources "${REQUIRED_RESOURCES[@]}"
    
    # Step 1: Process with resource1
    result1=$(call_resource1_api)
    assert_not_empty "$result1" "Resource1 processing"
    
    # Step 2: Process with resource2  
    result2=$(call_resource2_api "$result1")
    assert_not_empty "$result2" "Resource2 processing"
    
    # Validate workflow
    assert_not_equals "$result1" "$result2" "Data transformation occurred"
}
```

### Standard Exit Codes

The testing framework uses standard exit codes to communicate test results:

| Exit Code | Status | Description | Framework Action |
|-----------|--------|-------------|------------------|
| `0` | **PASS** | Test completed successfully | Count as passed test |
| `1` | **FAIL** | Test failed due to assertion failures or errors | Count as failed test, show error output |
| `77` | **SKIP** | Test skipped (e.g., required resources unavailable) | Count as skipped, don't treat as failure |
| `124` | **TIMEOUT** | Test exceeded time limit | Convert to failure, show timeout message |

#### Using Exit Codes in Tests

```bash
# Proper test termination
main() {
    setup_test
    test_functionality
    
    # Exit based on assertion results
    if [[ ${ASSERTION_FAILURES:-0} -gt 0 ]]; then
        echo "❌ Test failed"
        exit 1
    else
        echo "✅ Test passed"
        exit 0
    fi
}

# Skip test when resources unavailable
if ! check_resource_available; then
    echo "⏭️  Skipping test - resource unavailable"
    exit 77
fi

# Handle timeouts (automatically handled by framework)
timeout 300 run_long_test || exit 124
```

#### Best Practices

- **Always exit explicitly** with appropriate codes
- **Use exit 77** for skipped tests, not exit 1
- **Leverage `require_resource()`** for automatic skip handling
- **Test timeout logic** in long-running tests
- **Provide descriptive messages** for each exit scenario

## 🏥 Health Checks

The framework includes resource-specific health checks:
- **HTTP Services**: Endpoint availability and response validation
- **Docker Containers**: Container health status and port accessibility
- **Custom Protocols**: Resource-specific validation logic

## 🧹 Cleanup and Isolation

Automatic cleanup includes:
- **Test Artifacts**: Temporary files and generated data
- **Resource State**: Test-specific data in services
- **Environment**: Test-specific configuration and containers

## 📈 CI/CD Integration

Example GitHub Actions workflow:
```yaml
- name: Run Integration Tests
  run: |
    ./scripts/resources/tests/run.sh \
      --output-format json \
      --timeout 900 > test-results.json
      
- name: Upload Results
  uses: actions/upload-artifact@v3
  with:
    name: integration-test-results
    path: test-results.json
```

## 🔍 Troubleshooting

For comprehensive troubleshooting, see **[docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)** with solutions for every common issue.

### Quick Fixes

**No healthy resources found**
```bash
# Check what's running
../index.sh --action discover

# Start basic service
../ai/ollama/manage.sh --action start
```

**Tests hanging or timing out**
```bash
# Increase timeout
./run.sh --timeout 600

# Check specific service
curl http://localhost:11434/api/tags
```

**Need help getting started**
```bash
# System validation
./validate-setup.sh --fix

# Beginner-friendly help
./run.sh --help-beginner
```

## 🎉 Success Metrics

A successful test run indicates:
- ✅ **Resource Ecosystem Health**: All enabled resources are operational
- ✅ **Integration Reliability**: Multi-resource workflows function correctly  
- ✅ **Data Flow Integrity**: Information passes correctly between services
- ✅ **Error Resilience**: System handles failures gracefully
- ✅ **Performance Baseline**: Response times within acceptable ranges

This framework ensures Vrooli's AI tiers can reliably orchestrate the resource ecosystem for production workloads.