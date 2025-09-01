# Vrooli Scenario Testing Guide

This guide explains how to properly test Vrooli scenarios at different stages of development.

## Testing Levels

### 1. Static Analysis (Pre-Generation)
Validates scenario structure without generating an app.

**What it checks:**
- service.json location (must be in `.vrooli/` directory)
- JSON syntax validity (no trailing commas, proper structure)
- Schema compliance with `.vrooli/schemas/service.schema.json`
- File path validation (complex - see below)
- Resource configuration validity

**How to run:**
```bash
# Recommended: Use dedicated validator
./scenarios/tools/validate-scenario.sh <scenario-name>

# Alternative: scenario-to-app.sh includes basic validation before generation
./scenarios/tools/scenario-to-app.sh <scenario-name>
```

### 2. Integration Testing (Post-Generation)
Tests the generated app's functionality.

**What it tests:**
- Resource availability and health
- API endpoints functionality
- Workflow deployment verification
- End-to-end scenarios
- Performance metrics

**How to run:**
```bash
# Using scenario-test-runner (current)
./scenarios/validation/scenario-test-runner.sh --scenario ./<scenario-name>

# Proposed automated approach
./scenarios/tools/run-integration-test.sh <scenario-name>
```

### 3. Lifecycle Testing
Tests setup, develop, test, and stop lifecycle events.

**Status:** The `test` lifecycle event is untested and may not work.

## File Path Validation Complexity

File paths in service.json are relative to the **generated app's root**, not the scenario directory. This creates validation challenges:

### Valid Path Patterns

1. **Scenario-specific files** (exist in scenario):
   - `initialization/automation/n8n/workflow.json`
   - `deployment/startup.sh`

2. **Vrooli framework files** (copied during generation):
   - `scripts/lib/setup.sh`
   - `scripts/resources/populate/populate.sh`
   - `scripts/manage.sh`

### Validation Strategy

Check if path exists in:
1. `<scenario-root>/path` - Direct scenario files
2. `<vrooli-root>/path` - Framework files that get copied
3. Known copy patterns from scenario-to-app.sh

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
- Auto-convert scenario to app (check hash)
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
# Quick validation (no app generation) - RECOMMENDED
./scenarios/tools/validate-scenario.sh agent-metareasoning-manager

# Full integration test (generates app, runs lifecycle, tests)
./scenarios/tools/run-integration-test.sh agent-metareasoning-manager

# Manual testing of generated app
./scenarios/tools/scenario-to-app.sh agent-metareasoning-manager
cd ~/generated-apps/agent-metareasoning-manager
./scripts/manage.sh setup
./scripts/manage.sh develop
./scripts/manage.sh test  # May not work yet
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