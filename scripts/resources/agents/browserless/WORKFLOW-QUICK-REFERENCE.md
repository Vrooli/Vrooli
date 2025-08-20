# Browserless Workflow Quick Reference

## Essential Commands

```bash
# Create workflow
vrooli resource-browserless workflow create my-workflow.yaml

# Run workflow
vrooli resource-browserless workflow run my-workflow --params '{"url": "https://example.com"}'

# List workflows
vrooli resource-browserless workflow list

# View results
vrooli resource-browserless workflow results my-workflow
```

## Minimal Workflow Template

```yaml
workflow:
  name: "my-workflow"
  description: "What this workflow does"
  version: "1.0.0"
  debug_level: "steps"  # none|steps|verbose
  
  parameters:
    url:
      type: string
      required: true
      
  steps:
    - name: "step-name"
      action: "navigate"
      url: "${params.url}"
```

## Common Actions Quick Reference

| Action | Purpose | Key Parameters |
|--------|---------|----------------|
| **navigate** | Go to URL | `url`, `wait_for`, `timeout` |
| **click** | Click element | `selector`, `wait_for_navigation` |
| **fill_form** | Fill form fields | `selectors`, `values` |
| **wait** | Pause execution | `duration` |
| **wait_for_element** | Wait for element | `selector`, `visible`, `timeout` |
| **screenshot** | Capture screenshot | `output`, `full_page` |
| **extract_data** | Extract structured data | `selectors`, `output` |
| **assert_element** | Verify element exists | `selector`, `timeout` |
| **select** | Choose dropdown option | `selector`, `value` |
| **keyboard** | Send keyboard input | `key` |

## Variable Reference

```yaml
# Input parameters
${params.username}              # Simple parameter
${params.config.api_key}        # Nested object
${params.items[0]}              # Array element

# Context variables
${context.outputDir}            # Output directory
${context.timestamp}            # Current timestamp
${context.step}                 # Step number
${context.workflow}             # Workflow name
```

## Debug Flags

```yaml
# Global (in workflow metadata)
debug_level: "verbose"

# Per-step override
debug:
  screenshot: true    # Capture screenshot
  console: true       # Console logs
  network: true       # Network activity
  timing: true        # Execution timing
```

## Error Handling

```yaml
on_error: "continue"  # Skip and continue
on_error: "retry"     # Retry the step
on_error: "fail"      # Stop workflow

# With retry config
on_error: "retry"
max_retries: 3
retry_delay: 1000
```

## Common Patterns

### Login Flow
```yaml
- action: "navigate"
  url: "${params.login_url}"
  
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

### Data Extraction
```yaml
- action: "navigate"
  url: "${params.data_url}"
  
- action: "wait_for_element"
  selector: ".data-table"
  
- action: "extract_data"
  selectors:
    title: ".item-title"
    price: ".item-price"
  parent_selector: ".data-row"
  output: "${context.outputDir}/data.json"
```

### Form Submission
```yaml
- action: "fill_form"
  selectors:
    name: "#name"
    email: "#email"
    message: "#message"
  values:
    name: "${params.name}"
    email: "${params.email}"
    message: "${params.message}"
    
- action: "click"
  selector: "button[type='submit']"
  
- action: "wait_for_element"
  selector: ".success-message"
```

### Pagination
```yaml
- action: "extract_data"
  selectors:
    title: ".result-title"
  output: "${context.outputDir}/page-1.json"
  
- action: "click"
  selector: ".next-page"
  on_error: "continue"
  
- action: "wait_for_network_idle"
  idle_time: 1000
  
- action: "extract_data"
  selectors:
    title: ".result-title"
  output: "${context.outputDir}/page-2.json"
```

## Selector Strategies

```yaml
# Multiple fallback selectors
selector: "button[type='submit'], input[type='submit'], .submit-btn"

# Common selector patterns
"#id"                         # ID selector
".class"                      # Class selector
"button:contains('Text')"     # Text content
"[data-testid='element']"     # Data attribute
"form input[name='field']"    # Nested selector
```

## Output Files

Default output locations:
- Screenshots: `~/.browserless/outputs/<workflow>/screenshots/`
- Data: `~/.browserless/outputs/<workflow>/data/`
- Logs: `~/.browserless/logs/<workflow>/`
- Debug: `~/.browserless/outputs/<workflow>/debug/`

## Tips & Tricks

1. **Start with verbose debugging**: Use `debug_level: "verbose"` during development
2. **Use explicit waits**: Add `wait` actions between steps if timing is critical
3. **Handle optional elements**: Use `on_error: "continue"` for elements that might not exist
4. **Test selectors**: Use browser DevTools to verify selectors work
5. **Parameterize everything**: Use parameters instead of hardcoding values
6. **Check network idle**: Use `wait_for_network_idle` after AJAX operations
7. **Capture state**: Take screenshots before and after critical actions
8. **Use conditions**: Add `condition` to skip steps based on parameters

## Available Templates

- `simple-navigation` - Basic page visit and screenshot
- `login-dashboard` - Authentication workflow
- `data-extraction` - Extract data with pagination
- `ecommerce-checkout` - Complete purchase flow
- `form-submission` - Submit forms with uploads
- `monitoring-health-check` - Monitor site availability
- `search-and-filter` - Search with filters

## Need Help?

- Full documentation: `README-WORKFLOWS.md`
- Examples: `~/.browserless/templates/`
- View workflow details: `vrooli resource-browserless workflow describe <name>`
- Validate syntax: `vrooli resource-browserless workflow validate <file>`