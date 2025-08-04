# Scenario Testing Framework

## Overview
A declarative testing framework that replaces 1000+ lines of imperative bash scripts with ~35 lines of code + YAML configuration. Achieved **95% code reduction** across all business scenarios.

## Quick Start

### Running Tests
```bash
# Test a specific scenario
cd scripts/scenarios/core/[scenario-name]
./test.sh

# Test with verbose output
./test.sh --verbose

# Test with specific timeout
./test.sh --timeout 60
```

### Migrating Old Tests
```bash
# Migrate a single scenario
./scripts/scenarios/framework/migrate-scenario.sh --scenario [name]

# Migrate all scenarios
./scripts/scenarios/framework/migrate-scenario.sh --all
```

## Architecture

```
framework/
├── scenario-test-runner.sh    # Main orchestrator
├── handlers/                   # Test type implementations
│   ├── http.sh                # HTTP API testing
│   ├── chain.sh               # Multi-step workflows
│   ├── database.sh            # Database operations
│   └── custom.sh              # Custom business logic
├── validators/                 # Validation utilities
│   ├── structure.sh           # File/directory validation
│   └── resources.sh           # Service health checks
├── clients/                    # Resource integrations
│   ├── common.sh              # Shared utilities
│   └── ollama.sh              # Resource-specific clients
└── migrate-scenario.sh        # Migration automation

```

## Test Configuration (scenario-test.yaml)

### Basic Structure
```yaml
version: 1.0
scenario: [scenario-name]

# File/directory requirements
structure:
  required_files:
    - metadata.yaml
    - manifest.yaml
    - README.md
  required_dirs:
    - initialization
    - deployment

# Resource dependencies
resources:
  required: [ollama, whisper]
  optional: [browserless]
  health_timeout: 30

# Test definitions
tests:
  - name: "Test Name"
    type: [http|chain|database|custom]
    # ... test-specific config
```

### Test Types

#### HTTP Tests
```yaml
- name: "API Endpoint Test"
  type: http
  service: ollama
  endpoint: /api/tags
  method: GET
  expect:
    status: 200
    body:
      type: json
      contains: "models"
```

#### Chain Tests (Multi-Step Workflows)
```yaml
- name: "Processing Pipeline"
  type: chain
  steps:
    - id: step1
      service: whisper
      action: transcribe
      input:
        file: audio.mp3
      output: transcript
    
    - id: step2
      service: ollama
      action: generate
      input:
        prompt: "Summarize: ${transcript}"
      output: summary
```

#### Database Tests
```yaml
- name: "Database Operations"
  type: database
  operations:
    - action: insert
      table: users
      data:
        id: "test-123"
        name: "Test User"
    - action: select
      table: users
      where:
        id: "test-123"
      expect:
        count: 1
```

#### Custom Tests
```yaml
- name: "Business Logic"
  type: custom
  script: custom-tests.sh
  function: test_business_workflow
```

## Custom Business Logic (custom-tests.sh)

For complex business logic that doesn't fit declarative patterns:

```bash
#!/bin/bash
# Custom Business Logic Tests

test_scenario_workflow() {
    print_custom_info "Testing business workflow"
    
    # Test implementations
    if test_core_services; then
        print_custom_success "Core services operational"
    fi
    
    # Return success/failure
    return 0
}

export -f test_scenario_workflow
```

## Resource Discovery

Resources are automatically discovered from:
1. `.vrooli/resources.local.json` (primary)
2. Environment variables (fallback)
3. Default ports (last resort)

Example resource configuration:
```json
{
  "ollama": {
    "enabled": true,
    "urls": ["http://localhost:11434"]
  },
  "whisper": {
    "enabled": true,
    "urls": ["http://localhost:9000"]
  }
}
```

## Migration Results

| Scenario | Old Lines | New Lines | Reduction |
|----------|-----------|-----------|-----------|
| multi-modal-ai-assistant | 1,073 | 33 | 97% |
| document-intelligence-pipeline | 623 | 33 | 95% |
| image-generation-pipeline | 1,300 | 34 | 98% |
| business-process-automation | 664 | 34 | 95% |
| **Total (13 scenarios)** | **8,622** | **440** | **95%** |

## Benefits

- **95% code reduction** - From 8,622 to 440 lines
- **AI-friendly** - Can be generated in one shot
- **Declarative** - Describe what, not how
- **Business-focused** - Tests validate business operations
- **Maintainable** - Clear separation of concerns
- **Extensible** - Easy to add new test types

## Common Patterns

### Testing Service Availability
```yaml
- name: "Service Health"
  type: http
  service: [service-name]
  endpoint: /health
  method: GET
  expect:
    status: 200
```

### Testing Multi-Service Integration
```yaml
- name: "Integration Test"
  type: chain
  steps:
    - id: service1_check
      service: service1
      action: health_check
    - id: service2_check
      service: service2
      action: health_check
```

### Optional Service Testing
```yaml
- name: "Optional Service"
  type: http
  service: optional-service
  endpoint: /status
  required: false  # Won't fail if service unavailable
```

## Troubleshooting

### Test Failures
- Check resource availability: `curl http://localhost:[port]/health`
- Verify file structure: `ls -la initialization/`
- Review logs: Tests output detailed error messages

### Resource Discovery Issues
- Verify `.vrooli/resources.local.json` exists
- Check environment variables are set
- Ensure services are running on expected ports

### Migration Issues
- Backup original test.sh: `cp test.sh test.sh.backup`
- Manually adjust scenario-test.yaml if needed
- Update custom-tests.sh with business logic

## Future Enhancements

- [ ] Visual test runner UI
- [ ] Parallel test execution
- [ ] Test result caching
- [ ] Performance benchmarking
- [ ] AI test generation
- [ ] Cloud resource testing

## Contributing

To add new test types:
1. Create handler in `framework/handlers/`
2. Add case in `scenario-test-runner.sh`
3. Document usage here
4. Add example to scenarios

## Support

For issues or questions:
- Check existing scenarios for examples
- Review framework source code
- Create issue in repository