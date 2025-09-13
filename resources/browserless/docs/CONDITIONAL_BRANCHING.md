# Conditional Branching in Browserless Workflows

## Overview

Browserless workflows support sophisticated conditional branching that enables intelligent automation based on page state, content, and user interactions. This allows workflows to adapt dynamically to different scenarios such as login detection, error handling, and content-based decisions.

## Condition Types

### 1. URL-Based Conditions

Check if the current URL matches a pattern:

```yaml
- name: "check-login-page"
  action: "condition"
  condition_type: "url"
  url_pattern: "/login"  # Can use wildcards: "/admin/*" or regex: "/user/\d+"
  then_steps:
    - action: "log"
      message: "On login page"
  else_steps:
    - action: "log"
      message: "Not on login page"
```

### 2. Element Visibility Conditions

Check if an element is visible on the page:

```yaml
- name: "check-modal-visible"
  action: "condition"
  condition_type: "element_visible"
  selector: ".modal, .dialog"
  then_steps:
    - action: "click"
      selector: ".modal-close"
```

### 3. Element Text Conditions

Check if an element contains specific text:

```yaml
- name: "check-error-message"
  action: "condition"
  condition_type: "element_text"
  selector: ".alert"
  text: "Invalid credentials"
  match_type: "contains"  # Options: exact, contains, regex
  then_steps:
    - action: "log"
      message: "Login error detected"
```

### 4. Input State Conditions

Check if a checkbox or radio button is checked:

```yaml
- name: "check-terms-accepted"
  action: "condition"
  condition_type: "input_checked"
  selector: "#terms-checkbox"
  then_steps:
    - action: "click"
      selector: "#submit-button"
  else_steps:
    - action: "click"
      selector: "#terms-checkbox"
```

### 5. Error Detection Conditions

Check for error messages on the page:

```yaml
- name: "check-for-errors"
  action: "condition"
  condition_type: "has_errors"
  error_patterns: "error,Error,failed,Failed,invalid"  # Comma-separated patterns
  then_steps:
    - action: "screenshot"
      path: "error-screenshot.png"
    - action: "log"
      message: "Errors detected on page"
```

### 6. JavaScript Conditions

Evaluate custom JavaScript expressions:

```yaml
- name: "check-custom-condition"
  action: "condition"
  condition_type: "javascript"
  condition: "document.querySelectorAll('.item').length > 10"
  then_steps:
    - action: "log"
      message: "More than 10 items found"
```

## Branching Patterns

### Pattern 1: Login Detection and Handling

Automatically detect and handle login pages when accessing protected content:

```yaml
steps:
  - name: "navigate-to-protected"
    action: "navigate"
    url: "https://app.example.com/dashboard"
    
  - name: "check-login-redirect"
    action: "condition"
    condition_type: "url"
    url_pattern: "/login"
    then_steps:
      # We were redirected to login
      - action: "fill"
        selector: "#username"
        text: "${params.username}"
      - action: "fill"
        selector: "#password"
        text: "${params.password}"
      - action: "click"
        selector: "button[type='submit']"
      - action: "wait_for_url_change"
        pattern: "/dashboard"
    else_steps:
      # Already logged in
      - action: "log"
        message: "Already authenticated"
```

### Pattern 2: Error Recovery

Handle errors gracefully with retry logic:

```yaml
steps:
  - name: "submit-form"
    action: "click"
    selector: "#submit"
    label: "submit_start"
    
  - name: "check-success"
    action: "condition"
    condition_type: "has_errors"
    then_steps:
      # Error occurred
      - action: "wait"
        duration: 2000
      - action: "jump"
        label: "submit_start"  # Retry
    else_steps:
      # Success
      - action: "log"
        message: "Form submitted successfully"
```

### Pattern 3: Multi-Path Navigation

Navigate different paths based on user state:

```yaml
steps:
  - name: "check-user-type"
    action: "evaluate"
    script: |
      const roleElement = document.querySelector('.user-role');
      return roleElement ? roleElement.textContent : 'guest';
    variable: "user_role"
    
  - name: "route-by-role"
    action: "condition"
    condition: "{{user_role}} === 'admin'"
    condition_type: "javascript"
    then_steps:
      - action: "navigate"
        url: "/admin/dashboard"
    else_steps:
      - action: "condition"
        condition: "{{user_role}} === 'user'"
        condition_type: "javascript"
        then_steps:
          - action: "navigate"
            url: "/user/dashboard"
        else_steps:
          - action: "navigate"
            url: "/public/home"
```

### Pattern 4: Content Verification

Verify content before proceeding:

```yaml
steps:
  - name: "check-data-loaded"
    action: "condition"
    condition_type: "element_visible"
    selector: ".data-table tbody tr"
    then_steps:
      - action: "evaluate"
        script: |
          const rows = document.querySelectorAll('.data-table tbody tr');
          return Array.from(rows).map(row => ({
            id: row.cells[0].textContent,
            name: row.cells[1].textContent
          }));
        output: "extracted-data.json"
    else_steps:
      - action: "wait"
        duration: 3000
      - action: "condition"
        condition_type: "element_visible"
        selector: ".no-data-message"
        then_steps:
          - action: "log"
            message: "No data available"
        else_steps:
          - action: "log"
            message: "Data loading timeout"
```

## Advanced Features

### Nested Conditions

Conditions can be nested for complex logic:

```yaml
- name: "complex-check"
  action: "condition"
  condition_type: "url"
  url_pattern: "/checkout"
  then_steps:
    - action: "condition"
      condition_type: "element_visible"
      selector: ".cart-empty"
      then_steps:
        - action: "navigate"
          url: "/shop"
      else_steps:
        - action: "click"
          selector: "#proceed-to-payment"
```

### Inline Steps vs. Labels

Conditions support both inline steps and label jumps:

```yaml
# Using inline steps (recommended for short sequences)
- action: "condition"
  condition_type: "element_visible"
  selector: ".modal"
  then_steps:
    - action: "click"
      selector: ".modal-close"

# Using labels (better for complex flows)
- action: "condition"
  condition_type: "has_errors"
  then: "error_handler"  # Jump to label
  else: "success_handler"
```

### Sub-Workflow Integration

Combine conditions with sub-workflows for modular logic:

```yaml
sub_workflows:
  handle_login:
    steps:
      - action: "fill"
        selector: "#username"
        text: "${params.username}"
      # ... more login steps

steps:
  - action: "condition"
    condition_type: "url"
    url_pattern: "/login"
    then_steps:
      - action: "call_sub_workflow"
        sub_workflow: "handle_login"
```

## Best Practices

1. **Use Specific Condition Types**: Prefer `condition_type: "url"` over JavaScript for URL checks
2. **Handle Both Branches**: Always provide meaningful `else_steps` or `else` actions
3. **Add Logging**: Include log actions in branches for debugging
4. **Capture State**: Take screenshots when errors are detected
5. **Set Timeouts**: Use wait actions before condition checks on dynamic pages
6. **Test Edge Cases**: Consider network delays, missing elements, and unexpected states

## Examples

Complete example workflows demonstrating conditional branching:

- [`conditional-login.yaml`](../examples/advanced/conditional-login.yaml) - Automatic login detection and handling
- [`error-recovery.yaml`](../examples/advanced/error-recovery.yaml) - Comprehensive error detection and recovery
- [`data-extraction.yaml`](../examples/advanced/data-extraction.yaml) - Conditional data extraction based on page structure

## Debugging Conditions

Enable debug mode to see condition evaluation:

```yaml
workflow:
  debug_level: "verbose"  # Shows condition evaluation details
  
steps:
  - name: "debug-condition"
    action: "condition"
    condition_type: "element_visible"
    selector: "#target"
    debug:
      screenshot: true  # Capture screenshot before evaluation
      console: true     # Log browser console
```

## Performance Considerations

- **Element checks are fast**: Element visibility/text checks are nearly instant
- **URL checks are instant**: URL pattern matching has no delay
- **JavaScript evaluation varies**: Complex scripts may take longer
- **Use appropriate timeouts**: Add wait steps before checking dynamic content
- **Batch related checks**: Group similar conditions to minimize overhead