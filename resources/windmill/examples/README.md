# Windmill Examples

This directory contains comprehensive examples demonstrating all three core Windmill concepts: Scripts, Flows, and Apps.

## 📁 Directory Structure

```
examples/
├── scripts/          # Individual functions in TypeScript, Python, Go, or Bash
├── flows/           # Visual workflows that chain scripts with logic
└── apps/            # Low-code UI applications
```

## 🔧 Scripts

Scripts are individual functions that perform specific tasks. They're the building blocks for larger workflows.

### Available Script Examples

1. **Basic TypeScript Script** (`scripts/basic-typescript-script.ts`)
   - Input validation and type safety
   - Multi-language greeting functionality
   - Structured return values with metadata

2. **Python Data Processing** (`scripts/python-data-processing.py`)
   - CSV data parsing and manipulation
   - Statistical analysis and validation
   - Multiple analysis modes

3. **Webhook Integration** (`scripts/webhook-trigger.json`)
   - Event-driven webhook handler
   - Payload validation
   - TypeScript implementation

4. **API Integration Client** (`scripts/api-integration.json`)
   - Bearer token authentication
   - Retry logic with exponential backoff
   - Multiple HTTP methods

[View Scripts README →](scripts/README.md)

## 🔄 Flows

Flows are visual workflows that orchestrate multiple scripts with control logic, branching, and error handling.

### Available Flow Examples

1. **Data Pipeline Flow** (`flows/data-pipeline-flow.json`)
   - ETL pipeline with extraction, transformation, and loading
   - Error handling and notifications
   - Scheduled execution

2. **Expense Approval Workflow** (`flows/approval-workflow.json`)
   - Multi-level approval chain
   - Conditional routing based on amount
   - Human-in-the-loop pattern

3. **Multi-System Integration** (`flows/integration-workflow.json`)
   - Synchronizes data across multiple systems
   - Conflict detection and resolution
   - Parallel execution with partial failure handling

[View Flows README →](flows/README.md)

## 📱 Apps

Apps are low-code UI applications built with drag-and-drop components that can execute scripts and flows.

### Available App Examples

1. **User Management Dashboard** (`apps/admin-dashboard.json`)
   - CRUD operations for user management
   - Real-time metrics and activity feed
   - Bulk operations and exports

2. **Customer Onboarding Form** (`apps/data-entry-form.json`)
   - Multi-step form with validation
   - File uploads and document processing
   - Progress tracking and autosave

3. **System Monitoring Dashboard** (`apps/monitoring-dashboard.json`)
   - Real-time system metrics
   - Interactive charts and visualizations
   - Alert management and service control

[View Apps README →](apps/README.md)

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