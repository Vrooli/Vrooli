# Browserless Workflow System

## Overview

The Browserless Workflow System transforms browser automation from code-centric JavaScript functions to declarative, user-friendly workflows. Instead of writing complex JavaScript, you can now define automation steps in simple YAML or JSON format.

## Key Features

- **Declarative Syntax**: Define workflows in YAML/JSON without JavaScript knowledge
- **Pre-built Actions**: Library of common browser automation actions
- **Two-Level Debugging**: Global workflow settings + per-step debug flags
- **Automatic Compilation**: Workflows compile to optimized JavaScript functions
- **Template Library**: Ready-to-use templates for common patterns
- **Error Handling**: Built-in retry logic and error recovery
- **Data Extraction**: Extract structured data from web pages
- **Performance Monitoring**: Track execution times and bottlenecks

## Quick Start

### 1. Create a Simple Workflow

Create a file `my-workflow.yaml`:

```yaml
workflow:
  name: "my-first-workflow"
  description: "Navigate to a page and take a screenshot"
  version: "1.0.0"
  debug_level: "steps"  # none, steps, or verbose
  
  parameters:
    url:
      type: string
      required: true
      description: "URL to visit"
  
  steps:
    - name: "go-to-page"
      action: "navigate"
      url: "${params.url}"
      wait_for: "networkidle2"
      
    - name: "capture-page"
      action: "screenshot"
      output: "${context.outputDir}/page.png"
      full_page: true
```

### 2. Create the Workflow

```bash
vrooli resource-browserless workflow create my-workflow.yaml
```

### 3. Run the Workflow

```bash
vrooli resource-browserless workflow run my-first-workflow --params '{"url": "https://example.com"}'
```

## Workflow Structure

### Metadata Section

```yaml
workflow:
  name: "workflow-name"           # Unique identifier
  description: "What it does"     # Human-readable description  
  version: "1.0.0"                # Semantic versioning
  debug_level: "steps"            # Debug verbosity: none|steps|verbose
```

### Parameters Section

Define input parameters with validation:

```yaml
parameters:
  username:
    type: string
    required: true
    description: "Login username"
  
  max_retries:
    type: number
    required: false
    default: 3
    description: "Maximum retry attempts"
  
  credentials:
    type: object
    sensitive: true  # Won't be logged
    properties:
      api_key:
        type: string
```

### Steps Section

Sequential automation steps:

```yaml
steps:
  - name: "step-identifier"
    action: "action-type"
    # Action-specific parameters
    on_error: "continue"  # continue|retry|fail
    condition: "${params.enabled}"  # Optional condition
    debug:
      screenshot: true
      console: true
      network: false
      timing: true
```

## Available Actions

### Navigation & Page Control

#### navigate
Navigate to a URL:
```yaml
action: "navigate"
url: "https://example.com"
wait_for: "networkidle2"  # load|domcontentloaded|networkidle0|networkidle2
timeout: 30000
```

#### wait
Pause execution:
```yaml
action: "wait"
duration: 2000  # milliseconds
```

#### wait_for_element
Wait for element to appear:
```yaml
action: "wait_for_element"
selector: "#my-element"
visible: true
timeout: 10000
```

#### wait_for_redirect
Wait for URL change:
```yaml
action: "wait_for_redirect"
expected_url: "https://example.com/dashboard"
timeout: 15000
```

#### wait_for_network_idle
Wait for network to settle:
```yaml
action: "wait_for_network_idle"
idle_time: 1000
timeout: 10000
```

### Interaction Actions

#### click
Click an element:
```yaml
action: "click"
selector: "button[type='submit']"
wait_for_navigation: true
```

#### fill_form
Fill multiple form fields:
```yaml
action: "fill_form"
selectors:
  username: "#username"
  password: "#password"
  email: "input[name='email']"
values:
  username: "${params.username}"
  password: "${params.password}"
  email: "${params.email}"
```

#### select
Select dropdown option:
```yaml
action: "select"
selector: "#country"
value: "United States"
```

#### keyboard
Send keyboard input:
```yaml
action: "keyboard"
key: "Enter"  # or any key combination
```

#### scroll
Scroll the page:
```yaml
action: "scroll"
direction: "bottom"  # top|bottom
selector: "#section"  # Optional: scroll to element
```

#### upload_file
Upload a file:
```yaml
action: "upload_file"
selector: "input[type='file']"
file_path: "/path/to/file.pdf"
```

### Data Extraction

#### extract_text
Extract text content:
```yaml
action: "extract_text"
selector: "h1"
output: "${context.outputDir}/title.txt"
```

#### extract_data
Extract structured data:
```yaml
action: "extract_data"
selectors:
  title: ".product-name"
  price: ".price"
  description: ".description"
parent_selector: ".product-item"  # Optional container
output: "${context.outputDir}/products.json"
```

#### extract_table
Extract table data:
```yaml
action: "extract_table"
selector: "table#data"
format: "json"  # json|csv
output: "${context.outputDir}/table-data.json"
```

### Validation & Testing

#### assert_element
Check element exists:
```yaml
action: "assert_element"
selector: ".success-message"
timeout: 5000
```

#### assert_no_element
Check element doesn't exist:
```yaml
action: "assert_no_element"
selector: ".error-message"
```

#### assert_text
Verify text content:
```yaml
action: "assert_text"
selector: "h1"
expected_text: "Welcome"
```

#### assert_url
Verify current URL:
```yaml
action: "assert_url"
expected_url: "https://example.com/dashboard"
```

### Capture & Debug

#### screenshot
Take a screenshot:
```yaml
action: "screenshot"
output: "${context.outputDir}/page.png"
full_page: true
```

#### evaluate
Run custom JavaScript:
```yaml
action: "evaluate"
script: |
  return document.title;
output: "${context.outputDir}/title.txt"
```

#### log
Add log entry:
```yaml
action: "log"
message: "Processing step ${context.step}"
level: "info"  # debug|info|warn|error
```

## Debug Levels

### Global Debug Levels

Set in workflow metadata:

- **none**: No debug output
- **steps**: Screenshot after each step, basic timing
- **verbose**: Full debugging - screenshots, console logs, network activity, detailed timing

### Per-Step Debug Flags

Override global settings per step:

```yaml
debug:
  screenshot: true    # Capture screenshot
  console: true       # Collect console logs
  network: true       # Monitor network requests
  timing: true        # Track execution time
```

## Variables & Context

### Parameter Variables
Access input parameters:
- `${params.username}` - Simple parameter
- `${params.config.api_key}` - Nested object
- `${params.items[0]}` - Array element

### Context Variables
Built-in context values:
- `${context.outputDir}` - Output directory path
- `${context.timestamp}` - Current timestamp
- `${context.step}` - Current step number
- `${context.workflow}` - Workflow name

### Conditional Execution

Use conditions to control step execution:

```yaml
- name: "premium-features"
  action: "click"
  selector: ".premium-button"
  condition: "${params.is_premium}"  # Only runs if true
```

## Error Handling

### Error Strategies

Each step can define error behavior:

```yaml
on_error: "continue"  # Skip and continue
on_error: "retry"     # Retry the step
on_error: "fail"      # Stop workflow
```

### Retry Configuration

```yaml
- name: "flaky-action"
  action: "click"
  selector: ".button"
  on_error: "retry"
  max_retries: 3
  retry_delay: 1000
```

## Workflow Templates

### Available Templates

1. **simple-navigation.yaml** - Basic page navigation and screenshot
2. **login-dashboard.yaml** - Authentication and dashboard access
3. **data-extraction.yaml** - Extract data with pagination
4. **ecommerce-checkout.yaml** - Complete purchase flow
5. **form-submission.yaml** - Submit forms with uploads
6. **monitoring-health-check.yaml** - Monitor site availability
7. **search-and-filter.yaml** - Search with filters and pagination

### Using Templates

```bash
# List available templates
vrooli resource-browserless workflow list-templates

# Create from template
vrooli resource-browserless workflow create-from-template login-dashboard

# Customize template
cp ~/.browserless/templates/login-dashboard.yaml my-login.yaml
# Edit my-login.yaml
vrooli resource-browserless workflow create my-login.yaml
```

## CLI Commands

### Workflow Management

```bash
# Create a workflow
vrooli resource-browserless workflow create workflow.yaml

# List all workflows
vrooli resource-browserless workflow list

# Describe a workflow
vrooli resource-browserless workflow describe my-workflow

# Validate workflow syntax
vrooli resource-browserless workflow validate workflow.yaml

# Delete a workflow
vrooli resource-browserless workflow delete my-workflow
```

### Workflow Execution

```bash
# Run with default parameters
vrooli resource-browserless workflow run my-workflow

# Run with custom parameters
vrooli resource-browserless workflow run my-workflow \
  --params '{"url": "https://example.com", "username": "test"}'

# Run with parameter file
vrooli resource-browserless workflow run my-workflow \
  --params-file params.json

# Run with debug override
vrooli resource-browserless workflow run my-workflow \
  --debug-level verbose
```

### Results & Debugging

```bash
# View last run results
vrooli resource-browserless workflow results my-workflow

# View specific run results
vrooli resource-browserless workflow results my-workflow --run-id abc123

# Export debug data
vrooli resource-browserless workflow export-debug my-workflow \
  --output debug-report.json
```

### Advanced Operations

```bash
# Compile workflow to JavaScript (for inspection)
vrooli resource-browserless workflow compile workflow.yaml \
  --output compiled.js

# Test workflow without execution
vrooli resource-browserless workflow dry-run my-workflow \
  --params '{"url": "https://example.com"}'

# Schedule workflow (requires cron)
vrooli resource-browserless workflow schedule my-workflow \
  --cron "0 */6 * * *" \
  --params '{"url": "https://example.com"}'
```

## Best Practices

### 1. Use Descriptive Names
```yaml
# Good
- name: "wait-for-login-form"
  action: "wait_for_element"
  
# Bad  
- name: "step-1"
  action: "wait_for_element"
```

### 2. Handle Errors Gracefully
```yaml
- name: "optional-popup"
  action: "click"
  selector: ".close-popup"
  on_error: "continue"  # Don't fail if popup doesn't appear
```

### 3. Use Parameters for Flexibility
```yaml
parameters:
  environment:
    type: string
    default: "production"
    
steps:
  - name: "navigate"
    action: "navigate"
    url: "${params.base_url}/${params.environment}"
```

### 4. Debug Incrementally
Start with `debug_level: "verbose"` during development, then reduce to `"steps"` or `"none"` for production.

### 5. Modularize Complex Workflows
Break large workflows into smaller, reusable components:

```yaml
- name: "login-sequence"
  action: "include"
  workflow: "common-login"
  params:
    username: "${params.username}"
```

## Troubleshooting

### Common Issues

**Selector Not Found**
```yaml
# Use multiple selector strategies
selector: "button[type='submit'], input[type='submit'], .submit-btn"
```

**Timing Issues**
```yaml
# Add explicit waits
- name: "wait-after-click"
  action: "wait"
  duration: 2000
```

**Dynamic Content**
```yaml
# Wait for network to settle
- name: "wait-for-ajax"
  action: "wait_for_network_idle"
  idle_time: 1000
```

### Debug Output

When `debug_level: "verbose"`, outputs include:

- **Screenshots**: `~/.browserless/outputs/<workflow>/screenshots/`
- **Console Logs**: `~/.browserless/outputs/<workflow>/console.json`
- **Network Activity**: `~/.browserless/outputs/<workflow>/network.json`
- **Timing Data**: `~/.browserless/outputs/<workflow>/performance.json`
- **Error Stack Traces**: `~/.browserless/outputs/<workflow>/errors.json`

## Advanced Features

### Custom Actions

Create custom actions by extending the action library:

```bash
# Add to ~/.browserless/actions/my-action.sh
action::impl_my_custom_action() {
    local params="$1"
    # Generate JavaScript code
    cat <<EOF
    // Custom JavaScript implementation
    await page.evaluate(() => {
        // Your code here
    });
EOF
}
```

### Workflow Composition

Combine workflows:

```yaml
- name: "run-login"
  action: "workflow"
  workflow_name: "standard-login"
  params:
    username: "${params.username}"
```

### Parallel Execution

Run steps in parallel (experimental):

```yaml
- name: "parallel-checks"
  action: "parallel"
  steps:
    - action: "navigate"
      url: "https://api.example.com/health"
    - action: "navigate"  
      url: "https://app.example.com/status"
```

## Examples

### Example 1: Login and Download Report

```yaml
workflow:
  name: "download-report"
  description: "Login and download daily report"
  version: "1.0.0"
  
  parameters:
    username:
      type: string
      required: true
    password:
      type: string
      required: true
      sensitive: true
  
  steps:
    - name: "login"
      action: "navigate"
      url: "https://app.example.com/login"
      
    - name: "enter-credentials"
      action: "fill_form"
      selectors:
        username: "#username"
        password: "#password"
      values:
        username: "${params.username}"
        password: "${params.password}"
    
    - name: "submit"
      action: "click"
      selector: "button[type='submit']"
      wait_for_navigation: true
      
    - name: "go-to-reports"
      action: "navigate"
      url: "https://app.example.com/reports"
      
    - name: "download-report"
      action: "click"
      selector: ".download-daily-report"
      
    - name: "wait-for-download"
      action: "wait"
      duration: 5000
```

### Example 2: Monitor Prices

```yaml
workflow:
  name: "price-monitor"
  description: "Track product prices"
  version: "1.0.0"
  
  parameters:
    products:
      type: array
      required: true
      items:
        type: object
        properties:
          name:
            type: string
          url:
            type: string
  
  steps:
    - name: "check-product-1"
      action: "navigate"
      url: "${params.products[0].url}"
      
    - name: "extract-price-1"
      action: "extract_data"
      selectors:
        name: ".product-title"
        price: ".price-now"
        availability: ".stock-status"
      output: "${context.outputDir}/product-1.json"
    
    - name: "check-product-2"
      action: "navigate"
      url: "${params.products[1].url}"
      condition: "${params.products.length > 1}"
      
    - name: "extract-price-2"
      action: "extract_data"
      selectors:
        name: ".product-title"
        price: ".price-now"
      output: "${context.outputDir}/product-2.json"
      condition: "${params.products.length > 1}"
```

## Migration from Functions

### Before (JavaScript Function)

```javascript
async function login(page, username, password) {
    await page.goto('https://example.com/login');
    await page.type('#username', username);
    await page.type('#password', password);
    await page.click('button[type="submit"]');
    await page.waitForNavigation();
}
```

### After (Workflow)

```yaml
workflow:
  name: "login"
  steps:
    - action: "navigate"
      url: "https://example.com/login"
    - action: "fill_form"
      selectors:
        username: "#username"
        password: "#password"
      values:
        username: "${params.username}"
        password: "${params.password}"
    - action: "click"
      selector: "button[type='submit']"
      wait_for_navigation: true
```

## Support & Resources

- **Examples**: `~/.browserless/templates/`
- **Logs**: `~/.browserless/logs/`
- **Debug Output**: `~/.browserless/outputs/`
- **Custom Actions**: `~/.browserless/actions/`

## Performance Tips

1. **Minimize Screenshots**: Only capture when necessary
2. **Use Appropriate Timeouts**: Don't wait longer than needed
3. **Batch Operations**: Group related actions together
4. **Cache Selectors**: Reuse common selectors via parameters
5. **Network Idle Strategy**: Choose appropriate wait strategy

## Security Considerations

1. **Sensitive Parameters**: Mark passwords/tokens as `sensitive: true`
2. **Output Sanitization**: Avoid logging sensitive data
3. **File Uploads**: Validate file paths and permissions
4. **JavaScript Evaluation**: Be cautious with dynamic scripts
5. **Network Access**: Restrict to known domains when possible