# Why Workflows Are Discouraged

## Core Principle
Direct API integration is faster, more reliable, and easier to maintain than workflow abstractions.

## Why Avoid Workflow Platforms

### Performance Issues
- **Extra Layer**: n8n/node-red/windmill add unnecessary abstraction
- **Network Overhead**: Additional HTTP hops between components  
- **Processing Delays**: Workflow engines introduce latency
- **Resource Usage**: Extra memory and CPU for workflow runtime

### Reliability Problems
- **Single Point of Failure**: Workflow engine goes down = everything breaks
- **Complex Debugging**: Harder to trace issues through workflow layers
- **Version Conflicts**: Workflow engine updates can break existing flows
- **State Management**: Workflow state can become corrupted

### Maintenance Burden
- **Double Maintenance**: Code + workflow configuration to maintain
- **Skill Requirements**: Team needs to learn workflow platform specifics
- **Platform Lock-in**: Hard to migrate away from specific workflow engine
- **Update Complexity**: Changes require both code and workflow updates

## Better Alternative: Direct CLI or API Calls

### Scenario-to-Scenario Communication
```javascript
// GOOD: Direct API call to another scenario, if you know its port/url (NEVER hard-code). This ensures you can specify a specific API verison number, for gradual migration if scenario API is updated
const response = await fetch('http://localhost:${EXAMPLE_SERVICE_API_PORT}$/api/v1/process', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(data)
});

// GOOD: Direct CLI call to another scenario, if port/url is not known
// Example using Node.js child_process to invoke a CLI command
import { exec } from 'child_process';

exec('node process-data.js --input data.json', (error, stdout, stderr) => {
  if (error) {
    console.error(`Error executing CLI: ${error.message}`);
    return;
  }
  if (stderr) {
    console.error(`CLI stderr: ${stderr}`);
    return;
  }
  console.log(`CLI output: ${stdout}`);
});
```

### Resource Integration
```javascript
// GOOD: Resource CLI
// Example: Using the resource-example CLI directly
import { exec } from 'child_process';

exec('vrooli resource resource-example run --input data.json', (error, stdout, stderr) => {
  if (error) {
    console.error(`Error executing resource-example: ${error.message}`);
    return;
  }
  if (stderr) {
    console.error(`resource-example stderr: ${stderr}`);
    return;
  }
  console.log(`resource-example output: ${stdout}`);
});

```

### Shared Logic
Instead of shared workflows, use:
- **API Endpoints**: Reusable endpoints other scenarios can call
- **CLI Commands**: Alternative to access other scenarios, and primary way to access resources

## When Workflows ARE Appropriate

Workflow platforms should ONLY be used when the scenario is specifically for workflow management:

- **workflow-builder**: Visual workflow creation tool
- **automation-studio**: End-user workflow automation
- **process-designer**: Business process management
- **n8n-manager**: Managing n8n instances themselves

## Migration Strategy

### From Workflows to APIs
1. Identify workflow functionality
2. Create equivalent API endpoint  
3. Test API endpoint works correctly
4. Replace workflow calls with API calls
5. Remove workflow configuration
6. Update documentation

### Shared Functionality
Instead of shared workflows, create:
- Utility functions in shared libraries
- Microservice endpoints other scenarios can call
- Database procedures for data operations
- Standardized API contracts

## Key Benefits of Direct Integration

### Performance
- No workflow engine overhead
- Direct communication paths
- Fewer network hops
- Better resource utilization

### Reliability  
- Fewer components to fail
- Simpler error handling
- Direct debugging
- Clear dependency chains

### Maintainability
- Single codebase to maintain
- Standard programming practices
- Version control friendly
- Team-friendly skillset

Remember: Workflow engines solve workflow problems. If you don't have workflow problems, don't create them by using workflow engines.