# Workflow Test Fixtures

This directory contains workflow definition files for testing automation resource integrations within Vrooli's ecosystem. These workflows serve as "smoke tests" to verify that automation platforms are operational and can integrate with other Vrooli resources.

## 📁 Directory Structure

```
workflows/
├── README.md                    # This file
├── metadata.yaml                # Workflow metadata and test expectations
├── test_helpers.sh              # Helper functions for workflow testing
├── validate_metadata.py         # Metadata validation script
├── node-red/                   # Node-RED flow fixtures
├── huginn/                     # Huginn agent fixtures
├── comfyui/                    # ComfyUI workflow fixtures
└── integration/                # Multi-platform integration workflows
```

## 🎯 Purpose

Workflow fixtures enable comprehensive testing of automation resource integrations:

1. **Platform Operability**: Verify automation platforms can execute workflows
2. **Resource Integration**: Test connectivity between automation and other resources
3. **Workflow Patterns**: Validate common automation patterns and use cases
4. **Error Handling**: Test resilience and fault tolerance
5. **Performance**: Benchmark workflow execution times

## 🔧 Automation Platforms

### N8N Workflows
Visual workflow automation platform for business processes.
- **Port**: 5678
- **Strengths**: Visual editor, extensive integrations, user-friendly
- **Test Focus**: HTTP APIs, webhooks, data transformations

### Node-RED Flows  
Flow-based programming for IoT and automation.
- **Port**: 1880
- **Strengths**: Real-time processing, IoT integration, lightweight
- **Test Focus**: Event-driven flows, sensor data, messaging

### Windmill Workflows
Developer-centric workflow platform with code-first approach.
- **Port**: 5681
- **Strengths**: Script-based, versioning, enterprise features
- **Test Focus**: Complex logic, data processing, API orchestration

### Huginn Agent Networks
Agent-based automation for monitoring and actions.
- **Port**: 4111
- **Strengths**: Event monitoring, conditional logic, web scraping
- **Test Focus**: Content monitoring, automated responses, data collection

### ComfyUI Workflows
Node-based UI for AI image generation workflows.
- **Port**: 8188
- **Strengths**: AI model chaining, image processing, stable diffusion
- **Test Focus**: AI image generation, model coordination, batch processing

## 🔗 Integration Categories

### AI Integration Workflows
Test automation platform integration with AI resources:
- **Ollama**: LLM inference and text generation
- **Whisper**: Speech-to-text transcription
- **Unstructured.io**: Document parsing and extraction

### Storage Integration Workflows
Test file and data storage integrations:
- **MinIO**: Object storage for files and artifacts
- **Vault**: Secure secret and configuration management
- **Qdrant**: Vector database for AI embeddings
- **QuestDB**: Time-series data storage
- **PostgreSQL**: Relational database operations
- **Redis**: Caching and session storage

### Agent Integration Workflows
Test browser automation and web interaction:
- **Agent S2**: Advanced browser automation
- **Browserless**: Headless browser operations
- **SearXNG**: Meta-search engine integration
- **Judge0**: Code execution and validation

## 📊 Workflow Categories

### Basic Integration Tests
Single-resource connectivity validation:
```
Automation Platform → Single Resource → Output
```

### Multi-Resource Pipelines
Complex workflows spanning multiple resources:
```
Input → Resource A → Resource B → Resource C → Output
```

### Error Handling Patterns
Resilience and fault tolerance testing:
```
Input → Try Resource A → On Error: Resource B → Validate → Output
```

### Performance Benchmarks
Load and performance testing scenarios:
```
Multiple Inputs → Parallel Processing → Aggregated Output
```

## 🧪 Test Patterns

### Document Intelligence Pipeline
```
File Upload → Unstructured.io → Ollama Analysis → Qdrant Storage → MinIO Archive
```

### Audio Processing Workflow
```
Audio File → Whisper Transcription → Ollama Summarization → PostgreSQL Storage
```

### Web Automation Chain
```
SearXNG Query → Agent S2 Extraction → Data Processing → Storage
```

### AI Content Creation
```
Text Prompt → Ollama Enhancement → ComfyUI Generation → MinIO Storage
```

### Secure Data Processing
```
Vault Secrets → Encrypted Processing → Validated Output → Audit Log
```

## 🚀 Using Workflow Fixtures

### In BATS Tests
```bash
@test "node-red ollama integration" {
    local workflow="$FIXTURES_DIR/workflows/node-red/node-red-workflow.json"
    
    # Import workflow to Node-RED
    run import_node_red_workflow "$workflow"
    assert_success
    
    # Execute workflow
    run execute_node_red_workflow "$workflow"
    assert_success
    assert_output --partial "Hello from integration test"
}
```

### With Metadata Helpers
```bash
source "$FIXTURES_DIR/workflows/test_helpers.sh"

@test "all basic integration workflows" {
    # Get all basic integration workflows
    while IFS= read -r workflow_path; do
        local platform=$(get_workflow_platform "$workflow_path")
        run execute_workflow "$platform" "$FIXTURES_DIR/workflows/$workflow_path"
        assert_success
    done < <(get_workflows_by_tag "basic-integration")
}
```

### Multi-Platform Testing
```bash
@test "cross-platform ollama integration" {
    for platform in node-red; do
        local workflow=$(get_workflow_for_integration "$platform" "ollama")
        if [[ -n "$workflow" ]]; then
            run test_platform_integration "$platform" "$workflow"
            assert_success
        fi
    done
}
```

## 📋 Workflow Properties

Each workflow fixture includes metadata describing:

- **Platform**: Target automation platform (node-red, huginn, comfyui, etc.)
- **Integration**: Resources being tested
- **Complexity**: Basic, intermediate, or advanced
- **Expected Duration**: Typical execution time
- **Resource Requirements**: CPU, memory, network needs
- **Test Expectations**: Success criteria and expected outputs
- **Error Scenarios**: Expected failure conditions

## 🔍 Validation

### Workflow Validation
```bash
# Validate workflow syntax and metadata
./validate_metadata.py

# Test workflow execution
source test_helpers.sh
validate_all_workflows
```

### Integration Testing
```bash
# Test specific integration
test_workflow_integration "node-red" "ollama"

# Test all integrations for a platform
test_platform_integrations "node-red"

# Test cross-platform compatibility
test_cross_platform_workflows
```

## 📚 Workflow Documentation

Each workflow includes:
- **Purpose**: What the workflow tests
- **Prerequisites**: Required resources and configuration
- **Expected Behavior**: Success criteria and outputs
- **Error Conditions**: Known failure scenarios
- **Troubleshooting**: Common issues and solutions

## 🛠️ Adding New Workflows

### 1. Choose Platform Directory
Place workflow in appropriate platform subdirectory:
```
workflows/node-red/node-red-new-integration.json
```

### 2. Follow Naming Convention
Use descriptive names indicating platform and integration:
```
{platform}-{primary-resource}-{test-type}.json
```

### 3. Update Metadata
Add entry to `workflows.yaml`:
```yaml
- path: node-red/node-red-new-integration.json
  platform: node-red
  integration: [resource1, resource2]
  complexity: basic
  expectedDuration: 30
  tags: [integration-test, smoke-test]
```

### 4. Document Workflow
Include comments in workflow definition explaining:
- Purpose and test scenario
- Expected inputs and outputs
- Resource requirements
- Success criteria

## 🔧 Helper Scripts

### test_helpers.sh
Bash functions for workflow testing:
- `execute_workflow()` - Run workflow on specified platform
- `import_workflow()` - Import workflow definition
- `get_workflow_status()` - Check execution status
- `validate_workflow_output()` - Verify expected results

### validate_metadata.py
Python script for metadata validation:
- Verify workflow files exist and are valid
- Check platform compatibility
- Validate expected properties match actual workflow content
- Generate validation reports

## 🚦 Best Practices

1. **Minimal Viable Workflows**: Keep test workflows simple and focused
2. **Clear Documentation**: Document purpose and expected behavior
3. **Resource Cleanup**: Ensure workflows clean up test data
4. **Error Handling**: Include proper error handling in workflows
5. **Timeout Management**: Set appropriate timeouts for operations
6. **Security**: Use test credentials, never production secrets
7. **Idempotency**: Workflows should be safe to run multiple times

## 🔄 Maintenance

### Regular Tasks
- Validate workflow metadata monthly
- Update workflows when platforms change
- Test workflows after resource updates
- Clean up orphaned test data

### Validation Commands
```bash
# Validate all workflow metadata
python validate_metadata.py

# Test workflow execution
bash test_all_workflows.sh

# Check platform compatibility
bash check_platform_versions.sh
```

## 📈 Coverage Goals

### Current Coverage
- **N8N**: Basic Ollama integration ✅
- **Node-RED**: Basic Ollama integration ✅
- **Other Platforms**: Not covered ❌

### Target Coverage
- All 5 automation platforms with basic integrations
- Core AI resource integrations (Ollama, Whisper, Unstructured.io)
- Essential storage integrations (MinIO, Vault, PostgreSQL)
- Multi-resource pipeline examples
- Error handling and resilience patterns

## 🤝 Contributing

When adding workflow fixtures:
1. Test workflows thoroughly before committing
2. Include comprehensive metadata
3. Document expected behavior and requirements
4. Add appropriate error handling
5. Update test suites and validation scripts

For questions about workflow testing, see the main [Test Framework Documentation](../framework/README.md).
