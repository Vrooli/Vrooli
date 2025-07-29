# Vrooli Resource Integration Testing Framework

A comprehensive integration testing framework for Vrooli's resource ecosystem that validates real-world functionality without mocking dependencies.

## 🎯 Overview

This framework provides true integration testing for Vrooli resources by:
- **Resource Discovery**: Automatically discovers available resources and validates health
- **Configuration-Based Testing**: Tests only enabled and healthy resources  
- **Real Integration**: Uses actual services without mocking for authentic validation
- **Multi-Resource Workflows**: Tests complex workflows involving multiple resources
- **Comprehensive Reporting**: Detailed results with actionable recommendations

## 🚀 Quick Start

```bash
# Run all available tests
./run.sh

# Test specific resource
./run.sh --resource ollama

# Verbose output with detailed logging
./run.sh --verbose

# JSON output for CI/CD integration
./run.sh --output-format json

# Test only single resources (no multi-resource tests)
./run.sh --single-only

# Test only multi-resource integration workflows
./run.sh --multi-only
```

## 📁 Directory Structure

```
tests/
├── run.sh                 # Main test runner entry point
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

### Common Issues

**No healthy resources found**
- Verify resources are running: `./scripts/resources/index.sh --action discover`
- Check configuration: `cat ~/.vrooli/resources.local.json`

**Tests hanging or timing out**
- Increase timeout: `./run.sh --timeout 600`
- Check resource logs: `./scripts/resources/<category>/<resource>/manage.sh --action logs`

**Skipped tests**
- Install missing dependencies
- Enable resources in configuration
- Verify resource health and connectivity

## 🎉 Success Metrics

A successful test run indicates:
- ✅ **Resource Ecosystem Health**: All enabled resources are operational
- ✅ **Integration Reliability**: Multi-resource workflows function correctly  
- ✅ **Data Flow Integrity**: Information passes correctly between services
- ✅ **Error Resilience**: System handles failures gracefully
- ✅ **Performance Baseline**: Response times within acceptable ranges

This framework ensures Vrooli's AI tiers can reliably orchestrate the resource ecosystem for production workloads.