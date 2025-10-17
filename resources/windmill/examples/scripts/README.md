# Windmill Script Examples

Scripts are individual functions written in TypeScript, Python, Go, or Bash that perform specific tasks. They are the fundamental building blocks of Windmill automation.

## Available Examples

### 1. Basic TypeScript Script (`basic-typescript-script.ts`)

A simple greeting function demonstrating TypeScript capabilities in Windmill.

**Features:**
- Input validation with TypeScript interfaces
- Multi-language support (English, Spanish, French, German)
- Structured output with metadata
- Error handling for missing inputs

**Usage:**
```typescript
// Input
{
  "name": "Alice",
  "title": "Dr.",
  "language": "en"
}

// Output
{
  "greeting": "Hello Dr. Alice!",
  "timestamp": "2025-01-26T12:00:00Z",
  "metadata": {
    "language": "en",
    "title": "Dr.",
    "characterCount": 15,
    "wordCount": 3
  }
}
```

### 2. Python Data Processing (`python-data-processing.py`)

Advanced data processing script for CSV analysis with multiple analysis modes.

**Features:**
- CSV parsing and validation
- Three analysis types: summary, detailed, statistics
- Data filtering capabilities
- Automatic numeric column detection
- Statistical calculations (mean, median, std dev)

**Analysis Types:**
- **Summary**: Basic overview with sample data and fill rates
- **Detailed**: Column-by-column analysis with unique values
- **Statistics**: Comprehensive stats for numeric columns

**Example Usage:**
```python
# Input
{
  "csv_data": "name,age,salary\nJohn,30,50000\nJane,25,60000",
  "analysis_type": "statistics",
  "filter_column": "age",
  "filter_value": "30"
}
```

### 3. Webhook Integration (`webhook-trigger.json`)

Event-driven webhook handler for processing external events.

**Supported Events:**
- `user_signup`: New user registration
- `order_placed`: E-commerce order
- `payment_processed`: Payment confirmation

**Features:**
- Payload validation
- Event routing
- Async processing
- Error handling
- Structured responses

**Webhook URL:**
```
http://localhost:5681/api/w/{workspace}/jobs/run/f/webhooks/data_processor
```

### 4. API Integration Client (`api-integration.json`)

Enterprise-grade API client with advanced features.

**Features:**
- Bearer token authentication
- Exponential backoff retry logic
- Timeout handling
- All HTTP methods (GET, POST, PUT, DELETE)
- Response time tracking
- Error categorization

**Configuration:**
- Requires `api_key` resource (secret)
- Requires `base_url` resource (string)
- Optional: `retry_attempts`, `timeout_seconds` variables

## Creating Scripts in Windmill

### Step-by-Step Guide

1. **Access Windmill**: Navigate to http://localhost:5681
2. **Create Workspace**: Select or create a workspace
3. **New Script**: Click "Scripts" → "New Script"
4. **Choose Language**: Select TypeScript, Python, Go, or Bash
5. **Set Path**: Use convention `f/{category}/{script_name}`
6. **Write Code**: Paste example code or write your own
7. **Define Schema**: Windmill auto-detects or manually define
8. **Test**: Use the test panel with sample inputs
9. **Deploy**: Save to make available for execution

### Best Practices

#### Code Organization
- Use clear, descriptive function names
- Define input/output types (TypeScript)
- Add comments for complex logic
- Handle errors gracefully

#### Resource Management
- Store secrets in Windmill resources
- Use environment variables for configuration
- Never hardcode sensitive data

#### Performance
- Optimize for quick execution
- Use appropriate timeouts
- Consider memory usage for large datasets

#### Testing
- Test with various input scenarios
- Include edge cases
- Verify error handling

## Script Execution Methods

### 1. Manual Execution (Web UI)
- Navigate to script in Windmill
- Click "Run" button
- Enter inputs in form
- View results and logs

### 2. API Execution
```bash
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"args": {"name": "World"}}' \
  http://localhost:5681/api/w/{workspace}/jobs/run/{script_path}
```

### 3. Scheduled Execution
- Set up cron schedule in script settings
- Configure timezone and frequency
- Monitor execution history

### 4. Webhook Triggers
- Enable webhook endpoint for script
- Send POST requests to trigger
- Process external events

### 5. From Flows
- Use script as step in flow
- Map inputs from previous steps
- Handle outputs in subsequent steps

### 6. From Apps
- Bind script to UI components
- Execute on user interactions
- Display results in app

## Language-Specific Tips

### TypeScript
- Use interfaces for type safety
- Leverage async/await for promises
- Import from approved npm packages
- Export main function

### Python
- Use type hints for clarity
- Handle imports at top of file
- Return JSON-serializable data
- Use `def main()` as entry point

### Go
- Use `func Main()` as entry point
- Handle errors explicitly
- Return single value or struct
- Keep dependencies minimal

### Bash
- Use strict mode (`set -euo pipefail`)
- Quote variables properly
- Return output via stdout
- Exit codes for success/failure

## Debugging Scripts

### View Logs
- Check execution logs in Windmill UI
- Use console.log() or print() statements
- Review error stack traces

### Test Panel
- Use test panel for quick iterations
- Save test configurations
- Compare outputs across runs

### Local Development
- Export script for local testing
- Use Windmill CLI for deployment
- Set up IDE integration

## Next Steps

1. **Try Examples**: Import and run the provided examples
2. **Modify Code**: Experiment with changes
3. **Create New Scripts**: Build your own automation
4. **Combine in Flows**: Use scripts in workflows
5. **Build Apps**: Create UIs for scripts

For more advanced patterns, see the [Flows examples](../flows/) and [Apps examples](../apps/).