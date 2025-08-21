# 🚀 Browserless Flow Control Demo

## How It Works

The browserless flow control system transforms YAML workflows into JavaScript state machines that can handle complex scenarios with branching, loops, and error recovery.

## Key Concepts

### 1. **State Machine Execution**
Instead of linear step-by-step execution, workflows become state machines where:
- Each step is a **state**
- Steps can **jump** to any other step
- Execution follows a **flow graph** with conditional paths

### 2. **Flow Control Actions**

#### `jump_to` - Unconditional Jump
```yaml
- name: "skip-to-end"
  action: "jump_to"
  target: "final_step"
```

#### `jump_if` - Conditional Branching
```yaml
- name: "check-login"
  action: "jump_if"
  condition: "document.querySelector('.login-form') !== null"
  success_target: "handle_login"
  failure_target: "continue_workflow"
```

#### `switch` - Multi-way Branching
```yaml
- name: "route-by-page"
  action: "switch"
  expression: "window.location.pathname"
  cases:
    "/login": "login_flow"
    "/dashboard": "dashboard_flow"
    "/error": "error_handler"
  default: "unknown_page"
```

#### `wait_for_one_of` - Multiple Conditions
```yaml
- name: "wait-for-result"
  action: "wait_for_one_of"
  selectors:
    success: ".success-message"
    error: ".error-message"
    timeout: ".timeout-warning"
  on_match:
    success: "handle_success"
    error: "handle_error"
    timeout: "handle_timeout"
  max_wait: 10000
```

### 3. **Labels and Jump Targets**

Steps can have labels that serve as jump targets:
```yaml
- name: "process-data"
  label: "data_processor"  # Can be jumped to
  action: "extract_data"
  # ...
  jump_to: "next_phase"    # Jump after completion
```

### 4. **Sub-workflows**

Reusable workflow fragments:
```yaml
sub_workflows:
  handle_auth:
    steps:
      - name: "fill-login"
        action: "fill_form"
        # ...
      - name: "submit"
        action: "click"
        # ...
```

## Real-World Example: N8n Workflow with Authentication

```yaml
workflow:
  name: "n8n-smart-execution"
  flow_control:
    enabled: true
    
  steps:
    # Try to navigate to workflow
    - name: "go-to-n8n"
      action: "navigate"
      url: "http://n8n.local/workflow/123"
      
    # Detect if we were redirected to login
    - name: "check-page"
      action: "evaluate"
      script: |
        return {
          isLogin: !!document.querySelector('.login-form'),
          isWorkflow: !!document.querySelector('.workflow-canvas')
        }
      
    # Branch based on page type
    - name: "route"
      action: "switch"
      expression: "window.__lastEvalResult.isLogin ? 'login' : 'workflow'"
      cases:
        "login": "auth_flow"
        "workflow": "execute_workflow"
      
    # Authentication flow
    - name: "login"
      label: "auth_flow"
      action: "fill_form"
      selectors:
        email: "#email"
        password: "#password"
      values:
        email: "${env.N8N_EMAIL}"
        password: "${env.N8N_PASSWORD}"
      
    - name: "submit-login"
      action: "click"
      selector: "button[type='submit']"
      wait_for_navigation: true
      jump_to: "execute_workflow"  # Return to main flow
      
    # Main workflow execution
    - name: "run-workflow"
      label: "execute_workflow"
      action: "click"
      selector: ".execute-workflow-button"
      
    - name: "wait-completion"
      action: "wait_for_one_of"
      selectors:
        success: ".execution-success"
        error: ".execution-error"
      on_match:
        success: "extract_results"
        error: "handle_error"
```

## How Compilation Works

1. **Parse YAML** → Extract workflow structure
2. **Build Flow Graph** → Map labels, jumps, and branches
3. **Generate State Machine** → JavaScript with execution loop
4. **Inject into Browserless** → Ready for execution

## Execution Flow

```javascript
// Generated state machine (simplified)
let currentStep = 'entry_point';
const visitedSteps = new Set();

while (currentStep) {
    const stepDef = flowGraph.steps[currentStep];
    
    // Execute step action
    const result = await executeStep(stepDef);
    
    // Determine next step based on result
    if (result.type === 'flow_control') {
        currentStep = result.target;  // Jump to target
    } else {
        currentStep = getNextStep(currentStep);  // Default flow
    }
}
```

## Benefits

1. **Handle Complex Scenarios** - Login redirects, multi-page workflows, error recovery
2. **Reusable Components** - Sub-workflows for common patterns
3. **Intelligent Routing** - Conditional paths based on page state
4. **Loop Support** - Retry failed operations, pagination
5. **Non-linear Execution** - Jump to any step based on conditions

## Testing the System

```bash
# Compile a workflow with flow control
./cli.sh workflow compile examples/simple-demo.yaml

# Inject as N8n workflow
./cli.sh n8n inject examples/n8n-workflow-enhanced.yaml

# List injected workflows
./cli.sh n8n list

# Execute workflow
./cli.sh n8n execute workflow-name
```

The system transforms simple YAML definitions into sophisticated browser automation that can handle real-world complexity!