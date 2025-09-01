# Vrooli Scenario Testing Guide

This guide explains how to properly test Vrooli scenarios at different stages of development.

## Testing Levels

### 1. Static Analysis
Validates scenario structure before execution.

**What it checks:**
- service.json location (must be in `.vrooli/` directory)
- JSON syntax validity (no trailing commas, proper structure)
- Schema compliance with `.vrooli/schemas/service.schema.json`
- File path validation
- Resource configuration validity

**How to run:**
```bash
# Recommended: Use dedicated validator
./scenarios/tools/validate-scenario.sh <scenario-name>

# Or check with CLI dry-run
vrooli scenario run <scenario-name> --dry-run
```

### 2. Integration Testing
Tests the scenario's functionality during direct execution.

**What it tests:**
- Resource availability and health
- API endpoints functionality
- Workflow deployment verification
- End-to-end scenarios
- Performance metrics

**How to run:**
```bash
# Run scenario tests directly
vrooli scenario test <scenario-name>

# Or from scenario directory
cd scenarios/<scenario-name>
../../scripts/manage.sh test
```

### 3. Lifecycle Testing
Tests setup, develop, test, and stop lifecycle events.

**Status:** The `test` lifecycle event is untested and may not work.

## File Path Validation

File paths in service.json are relative to the **scenario's root** directory.

### Valid Path Patterns

1. **Scenario-specific files** (exist in scenario):
   - `initialization/automation/n8n/workflow.json`
   - `deployment/startup.sh`

2. **Vrooli framework files** (accessed via relative paths):
   - `../../scripts/lib/setup.sh`
   - `../../scripts/resources/populate/populate.sh`
   - `../../scripts/manage.sh`

### Validation Strategy

Check if path exists in:
1. `<scenario-root>/path` - Direct scenario files
2. `<vrooli-root>/path` - Framework files accessed via relative paths

## scenario-test.yaml Structure

```yaml
name: scenario-name
description: Test suite description
version: 1.0.0

prerequisites:
  resources:
    - ollama
    - n8n
    - postgres
  environment:
    SERVICE_PORT: ${SERVICE_PORT:-dynamic}
    N8N_BASE_URL: http://localhost:${RESOURCE_PORTS[n8n]}

tests:
  - name: resource_availability
    description: Verify required resources
    steps:
      - check: api_health
        command: curl -f http://localhost:${SERVICE_PORT}/health
        expected_status: 200
        
  - name: workflow_deployment
    description: Verify n8n workflows
    steps:
      - check: workflow_exists
        command: curl -X GET http://localhost:${RESOURCE_PORTS[n8n]}/rest/workflows
        contains: workflow-name

validation:
  timeout: 300
  retry_count: 3
  success_threshold: 0.95
```

## Proposed Improvements

### 1. Enhanced Static Validator
Create `scripts/scenarios/tools/validate-scenario.sh`:
- Full JSON schema validation using ajv or Python jsonschema
- Comprehensive file path checking
- Resource dependency validation
- Integration with --dry-run

### 2. Rename & Enhance Integration Testing
- Rename `scenario-test.yaml` → `integration-test.yaml`
- Run scenario directly (no conversion needed)
- Run setup, develop, test lifecycle
- Better error reporting

### 3. Test Discovery
Make it easier to find and run tests:
```bash
# List all available tests for a scenario
vrooli test list <scenario-name>

# Run specific test type
vrooli test static <scenario-name>
vrooli test integration <scenario-name>
vrooli test lifecycle <scenario-name>
```

## Quick Testing Commands

```bash
# Quick validation - RECOMMENDED
./scenarios/tools/validate-scenario.sh agent-metareasoning-manager

# Run scenario directly
vrooli scenario run agent-metareasoning-manager

# Run scenario tests
vrooli scenario test agent-metareasoning-manager

# Manual testing from scenario directory
cd scenarios/agent-metareasoning-manager
../../scripts/manage.sh setup
../../scripts/manage.sh develop
../../scripts/manage.sh test
```

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| "service.json not found" | Place in `.vrooli/service.json` |
| "Invalid JSON" | Remove trailing commas, validate with `jq` |
| "Path not found" | Check if path exists in scenario OR vrooli root |
| "Resource not available" | Ensure resource is enabled in service.json |
| "Test lifecycle fails" | This is expected - not implemented yet |

## Testing Checklist

- [ ] service.json in `.vrooli/` directory
- [ ] JSON validates with `jq empty`
- [ ] Schema compliance with service.schema.json
- [ ] All file paths resolve correctly
- [ ] Resource dependencies declared
- [ ] scenario-test.yaml (or integration-test.yaml) present
- [ ] Custom test scripts if needed
- [ ] Documentation updated