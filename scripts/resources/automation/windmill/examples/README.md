# Windmill Examples

This directory contains example scripts and configurations to help you get started with Windmill workflow automation.

## Available Examples

### 1. Basic TypeScript Script (`basic-typescript-script.ts`)

A simple TypeScript function demonstrating:
- Input validation and type safety
- Multi-language greeting functionality
- Structured return values with metadata
- Error handling

**How to use:**
1. Create a new script in Windmill with path `f/examples/hello`
2. Copy the TypeScript code from the file
3. Test with input: `{"name": "Alice", "title": "Dr.", "language": "en"}`

### 2. Python Data Processing (`python-data-processing.py`)

A comprehensive data processing script showing:
- CSV data parsing and manipulation
- Statistical analysis and data validation
- Different analysis types (summary, detailed, statistics)
- Filtering capabilities

**How to use:**
1. Create a new script in Windmill with path `f/examples/data_processor`
2. Copy the Python code from the file
3. Test with CSV data and analysis parameters

**Example input:**
```json
{
  "csv_data": "name,age,city,salary\nJohn,30,New York,50000\nJane,25,San Francisco,60000\nBob,35,Chicago,55000",
  "analysis_type": "statistics",
  "filter_column": "city",
  "filter_value": "New York"
}
```

### 3. Webhook Integration (`webhook-trigger.json`)

A complete webhook configuration example featuring:
- Event-driven processing for different event types
- Payload validation and error handling
- TypeScript implementation with proper typing
- Security considerations

**How to use:**
1. Create a new script with path `f/webhooks/data_processor`
2. Copy the TypeScript code from the JSON file
3. Test by sending POST requests to the webhook URL
4. Use the provided curl examples

**Webhook URL format:**
```
http://localhost:5681/api/w/{workspace}/jobs/run/f/webhooks/data_processor
```

### 4. API Integration Client (`api-integration.json`)

A robust API client implementation demonstrating:
- Authentication handling with Bearer tokens
- Retry logic with exponential backoff
- Comprehensive error handling
- Rate limiting considerations
- Multiple HTTP methods (GET, POST, PUT, DELETE)

**How to use:**
1. Create resources in Windmill:
   - `api_key` (secret): Your API authentication key
   - `base_url` (string): Base URL of the target API
2. Create a script with path `f/integrations/api_client`
3. Copy the TypeScript code from the JSON file
4. Test with different operations and endpoints

**Example input:**
```json
{
  "operation": "get",
  "endpoint": "/users/123"
}
```

## Setting Up Examples

### Prerequisites

1. **Windmill Installation**: Ensure Windmill is installed and running
   ```bash
   ./manage.sh --action install
   ./manage.sh --action status
   ```

2. **Workspace Creation**: Create a workspace in the Windmill web interface
   - Navigate to http://localhost:5681
   - Login with your admin credentials
   - Create a new workspace

3. **Resource Configuration**: Set up any required resources and variables

### Import Process

For each example:

1. **Navigate to Scripts**: Go to your workspace in Windmill
2. **Create New Script**: Click "New Script" or "+"
3. **Set Script Path**: Use the suggested path from each example
4. **Copy Code**: Paste the code from the example file
5. **Configure Schema**: Windmill will auto-detect input schema, or you can define it manually
6. **Test Execution**: Use the "Test" button with sample inputs
7. **Deploy**: Save and deploy the script

### Testing Examples

#### TypeScript Script
```bash
# Via API (replace {workspace} with your workspace ID)
curl -X POST \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "World", "language": "en"}' \
  http://localhost:5681/api/w/{workspace}/jobs/run/f/examples/hello
```

#### Python Data Processing
```bash
# Test with sample CSV data
curl -X POST \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"csv_data": "name,age\nAlice,30\nBob,25", "analysis_type": "summary"}' \
  http://localhost:5681/api/w/{workspace}/jobs/run/f/examples/data_processor
```

#### Webhook Processing
```bash
# Send webhook event
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"event_type": "user_signup", "data": {"user_id": "123", "email": "test@example.com", "timestamp": "2025-01-26T12:00:00Z"}}' \
  http://localhost:5681/api/w/{workspace}/jobs/run/f/webhooks/data_processor
```

## Customization Tips

### Modifying Examples

1. **Add Dependencies**: Use Windmill's package management for additional libraries
2. **Environment Variables**: Use resources and variables for configuration
3. **Error Handling**: Enhance error handling for your specific use cases
4. **Logging**: Add console.log statements for debugging
5. **Validation**: Add input validation specific to your requirements

### Best Practices

1. **Resource Management**: Store secrets and configuration in Windmill resources
2. **Error Handling**: Always handle errors gracefully and return meaningful messages
3. **Documentation**: Document your scripts with clear descriptions and examples
4. **Testing**: Test scripts thoroughly with various inputs before production use
5. **Monitoring**: Use Windmill's job monitoring to track execution and performance

## Integration Patterns

### Event-Driven Workflows

Use webhooks to trigger workflows from external systems:
- User registration events
- Payment processing notifications
- System monitoring alerts
- Data pipeline triggers

### Scheduled Processing

Create scheduled workflows for:
- Data synchronization
- Report generation
- Cleanup tasks
- Health checks

### API Orchestration

Build workflows that:
- Aggregate data from multiple APIs
- Transform data between systems
- Implement business logic across services
- Handle complex authentication flows

## Advanced Features

### Flow Composition

Combine scripts into workflows:
1. Create individual scripts for specific tasks
2. Use Windmill's flow editor to chain scripts
3. Add conditional logic and error handling
4. Implement parallel execution where appropriate

### Resource Sharing

Share resources across scripts:
- Database connections
- API credentials
- Configuration settings
- Shared utilities

### Version Control

Manage script versions:
- Use Windmill's built-in versioning
- Export/import scripts for backup
- Implement proper change management
- Test changes in development workspace first

## Troubleshooting

### Common Issues

1. **Import Errors**: Ensure all dependencies are available in Windmill
2. **Permission Errors**: Check resource access permissions
3. **Timeout Issues**: Adjust timeout settings for long-running scripts
4. **Memory Errors**: Monitor resource usage and adjust worker limits

### Debugging Tips

1. **Use Console Logs**: Add logging throughout your scripts
2. **Test Incrementally**: Test scripts with simple inputs first
3. **Check Job Logs**: Use Windmill's job interface to view detailed logs
4. **Validate Inputs**: Ensure input data matches expected schema

### Getting Help

- **Windmill Documentation**: https://docs.windmill.dev
- **Community Discord**: https://discord.gg/V7PM2YHsPB
- **GitHub Issues**: https://github.com/windmill-labs/windmill/issues
- **Example Repository**: Check the Windmill examples repository for more samples

## Contributing

To contribute additional examples:

1. Follow the existing file structure and naming conventions
2. Include comprehensive documentation and usage instructions
3. Provide realistic test data and expected outputs
4. Add error handling and input validation
5. Include security considerations where relevant

Each example should be self-contained and demonstrate specific Windmill features or integration patterns.