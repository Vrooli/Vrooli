# N8n Adapter for Browserless

## Overview

The N8n adapter provides browser automation for N8n workflows using the new **YAML-based flow control system**. Instead of generating JavaScript strings manually, workflows are defined declaratively in YAML and compiled to state machines.

## Architecture Change

### Before (500+ lines of bash)
```bash
# Manual JavaScript generation
function_code="${function_code//%WORKFLOW_ID%/$workflow_id}"
curl -X POST ... --data "$function_code"
```

### After (30 lines of YAML)
```yaml
workflow:
  name: "n8n-execute-workflow"
  steps:
    - name: "navigate"
      action: "navigate"
      url: "${params.n8n_url}/workflow/${params.workflow_id}"
    - name: "branch"
      action: "jump_if"
      condition: "needsLogin"
      success_target: "auth_flow"
      failure_target: "execute_flow"
```

## Features

- **Automatic Login Handling**: Detects authentication redirects and handles login
- **Atomic Operations**: Simple, debuggable step-by-step execution
- **Visual Debugging**: Screenshots saved at each step for troubleshooting
- **Error Recovery**: Automatic retry and error handling
- **Session Management**: Persistent browser contexts for complex workflows

## Workflow Definitions

### Execute Workflow (`execute-workflow.yaml`)
- Navigates to workflow
- Detects login redirects
- Handles authentication
- Executes workflow
- Waits for completion
- Extracts results

### List Workflows (`list-workflows.yaml`)
- Lists all workflows
- Handles authentication
- Filters active/inactive
- Returns JSON array

### Export Workflow (`export-workflow.yaml`)
- Exports workflow as JSON
- Downloads via UI
- Fallback methods

## Usage

### Execute N8n Workflow
```bash
# Basic execution
./api.sh execute-workflow <workflow-id>

# With parameters
./api.sh execute-workflow <workflow-id> http://n8n.local 60000 '{"key": "value"}'

# With environment credentials
export N8N_EMAIL="user@example.com"
export N8N_PASSWORD="password"
./api.sh execute-workflow <workflow-id>
```

### List Workflows
```bash
# List all workflows
./api.sh list-workflows

# List from specific instance
./api.sh list-workflows http://n8n.local

# Active workflows only
./api.sh list-workflows http://n8n.local false
```

### Export Workflow
```bash
# Export to default location
./api.sh export-workflow <workflow-id>

# Export to specific file
./api.sh export-workflow <workflow-id> http://n8n.local /path/to/output.json
```

## Flow Control Features

### Conditional Branching
```yaml
- name: "check-auth"
  action: "jump_if"
  condition: "document.querySelector('.login-form')"
  success_target: "login_flow"
  failure_target: "main_flow"
```

### Multi-way Routing
```yaml
- name: "route"
  action: "switch"
  expression: "pageState"
  cases:
    "login": "handle_login"
    "dashboard": "execute_workflow"
    "error": "handle_error"
```

### Wait for Multiple Conditions
```yaml
- name: "wait-completion"
  action: "wait_for_one_of"
  selectors:
    success: ".execution-success"
    error: ".execution-error"
  on_match:
    success: "extract_results"
    error: "handle_error"
```

### Sub-workflows
```yaml
sub_workflows:
  n8n_login:
    steps:
      - name: "fill-credentials"
        action: "fill_form"
        selectors:
          email: "#email"
          password: "#password"
```

## Benefits

1. **94% Less Code**: 30 lines of YAML vs 500+ lines of bash
2. **Maintainable**: Declarative YAML instead of string manipulation
3. **Reliable**: Compiled state machines vs procedural code
4. **Testable**: Individual steps can be tested
5. **Extensible**: Easy to add new workflows

## Technical Details

### Compilation Process
1. YAML workflow definitions are parsed
2. Flow control graph is built
3. State machine JavaScript is generated
4. Execution via browserless API

### State Machine Execution
```javascript
let currentStep = 'entry_point';
while (currentStep) {
    const step = flowGraph.steps[currentStep];
    const result = await executeStep(step);
    currentStep = determineNextStep(result);
}
```

## Migration from Old Adapter

The new adapter maintains **backward compatibility** with the same function signatures:
- `browserless::execute_n8n_workflow()`
- `n8n::list_workflows()`
- `n8n::export_workflow()`

Simply source the new `workflows.sh` and existing code continues to work, but now uses the flow control system internally.

## Extending

To add new N8n operations:

1. Create YAML workflow in `workflows/` directory
2. Define flow control and steps
3. Add function wrapper in `workflows.sh`
4. Register in `api.sh`

Example:
```yaml
workflow:
  name: "n8n-new-operation"
    enabled: true
  steps:
    - name: "step1"
      action: "navigate"
      # ...
```

## Troubleshooting

### Authentication Issues
- Ensure `N8N_EMAIL` and `N8N_PASSWORD` environment variables are set
- Check login form selectors match your N8n version

### Compilation Errors
- Verify YAML syntax is correct
- Check flow control compiler is sourced
- Review logs for parse errors

### Execution Failures
- Check browserless service is running
- Verify N8n URL is accessible
- Review execution screenshots in output directory