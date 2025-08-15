# N8n Workflow Development and Enhancement

You are tasked with creating or fixing n8n workflows in Vrooli. N8n workflows are the automation backbone that makes scenarios intelligent and autonomous - they orchestrate resources, process data, and implement complex business logic.

## Understanding N8n in Vrooli

**What N8n Workflows Are**: JSON-based visual automation workflows that connect resources, process data, and implement complex orchestration logic. They enable scenarios to be truly autonomous by automating routine tasks and decision-making.

**Why This Matters**: Workflows make scenarios easier to create and more powerful. A well-designed workflow can be reused across multiple scenarios, multiplying its value. Great workflows turn simple scenarios into intelligent applications.

**Important Version Note**: This guide uses n8n's modern Code node (v0.198.0+). The deprecated Function node should NOT be used in new workflows.

## Pre-Implementation Research

### 1. Understand N8n Resource Architecture
```bash
# Study n8n resource structure
tree scripts/resources/automation/n8n/

# Read n8n documentation
cat scripts/resources/automation/n8n/README.md
cat scripts/resources/automation/n8n/docs/API.md
```

### 2. Examine Existing Workflows
```bash
# Study workflow patterns in scenarios
find scripts/scenarios/core/ -name "*.json" -path "*/automation/n8n/*" | head -5

# Analyze complex workflow examples
cat scripts/scenarios/core/agent-metareasoning-manager/initialization/automation/n8n/reasoning-chain.json
```

### 3. Understanding N8n Integration
**Key Files to Read**:
- `scripts/resources/automation/n8n/lib/inject.sh` - How workflows get injected
- `scripts/resources/automation/n8n/config/defaults.sh` - N8n configuration
- `scripts/resources/automation/n8n/lib/api.sh` - N8n API patterns

## N8n Workflow Architecture

### Core Workflow Components

**1. Trigger Nodes**: How workflows start
- Webhook triggers for external API calls
- Schedule triggers for time-based automation
- Manual triggers for testing

**2. Logic Nodes**: Processing and decision-making
- Function nodes for custom JavaScript logic
- Switch nodes for conditional branching
- Merge nodes for data combination

**3. Integration Nodes**: Connecting to resources
- HTTP Request nodes for API calls
- Database nodes for PostgreSQL/Redis
- Custom nodes for Vrooli resources

**4. Response Nodes**: Providing output
- Webhook response for API endpoints
- Data transformation for clean output

### Workflow Design Patterns

**Key Patterns**: Multi-Step Chains, Resource Orchestration, Data Pipelines

**Reference Examples**:
```bash
# Study complete working examples:
cat scripts/scenarios/core/agent-metareasoning-manager/initialization/n8n/reasoning-chain.json
cat scripts/resources/automation/n8n/examples/webhook-workflow.json
```

## Implementation Steps

### 1. Workflow Planning and Design
```bash
# Document your workflow requirements
echo "## Workflow Requirements
- Purpose: [What this workflow achieves]
- Inputs: [What data it expects]
- Outputs: [What it returns]
- Resources: [Which Vrooli resources it uses]
- Error Handling: [How it handles failures]
" > workflow-plan.md
```

### 2. Design Workflow Architecture
- Map out the complete flow from trigger to response
- Identify all resource integrations needed
- Plan error handling and edge cases
- Design data transformation steps

### 3. Create Workflow JSON

**Complete Modern Workflow Structure** (Code node, not Function node):
```json
{
  "name": "Your Workflow Name",
  "nodes": [
    {
      "type": "n8n-nodes-base.webhook",
      "name": "Webhook",
      "position": [0, 0],
      "parameters": {
        "path": "your-endpoint",
        "responseMode": "lastNode",
        "responseData": "={{ $json || {'status': 'received'} }}"
      }
    },
    {
      "type": "n8n-nodes-base.code",
      "name": "Process Data",
      "position": [300, 0],
      "parameters": {
        "mode": "runOnceForEachItem",
        "jsCode": "// Modern Code node syntax\nconst data = $json;\n// Always validate and provide fallbacks\nreturn {\n  processed: true,\n  id: data.id || 'unknown',\n  timestamp: new Date().toISOString()\n};"
      }
    }
  ],
  "connections": {
    "Webhook": {
      "main": [
        [
          {
            "node": "Process Data",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
```
**Full examples**: See `scripts/resources/automation/n8n/examples/`

### 4. Implement Core Logic

**Modern Code Node Patterns** (n8n v0.198.0+):

**Code Node Processing Modes**:
```javascript
// Mode: "runOnceForEachItem" - processes each item individually
const input = $json; // Access current item's data
if (!input.required_field) {
  return { error: true, message: 'Missing required field', original: input };
}
return {
  id: input.id || 'unknown', // Always use fallbacks
  processed_at: new Date().toISOString(),
  data: input.raw_data?.toUpperCase() || 'N/A'
};

// Mode: "runOnceForAllItems" (default) - batch processing
const items = $input.all(); // Access all items
const processedItems = [];
for (const item of items) {
  if (item.json.status === 'active') {
    processedItems.push({
      json: { id: item.json.id || 'unknown', processed: true, timestamp: new Date().toISOString() }
    });
  }
}
return processedItems;
```

**Expression Fallbacks** (prevent failures from missing data):
```javascript
// In n8n expressions, ALWAYS use fallbacks:
"{{ $json["user"]["email"] || 'default@example.com' }}"
"{{ $json.status || 'pending' }}"
"{{ $node["HTTP Request"].json["data"] || [] }}"
"{{ $json?.nested?.property || 'fallback' }}"
```

**HTTP Request Node Configuration**:
```json
{
  "type": "n8n-nodes-base.httpRequest",
  "requestMethod": "POST",
  "url": "http://localhost:{{$json['port'] || 8080}}/api/endpoint",
  "jsonBody": "={{ JSON.stringify($json || {}) }}",
  "options": {
    "timeout": 30000
  }
}
```
Note: HTTP requests must be made via HTTP Request nodes, not in Code nodes

### 5. Resource Integration
```bash
# Study how to connect to Vrooli resources
# PostgreSQL integration
grep -A 10 -B 5 "postgres" scripts/scenarios/core/*/initialization/automation/n8n/*.json

# Redis integration
grep -A 10 -B 5 "redis" scripts/scenarios/core/*/initialization/automation/n8n/*.json

# Ollama integration
grep -A 10 -B 5 "ollama" scripts/scenarios/core/*/initialization/automation/n8n/*.json
```

### 6. Error Handling Implementation

**Robust Error Handling Pattern**:
```javascript
// Individual item mode: "runOnceForEachItem"
try {
  const requiredField = $json.required_field || null; // Always use fallbacks
  if (!requiredField) {
    return { success: false, error: 'Missing required field', received: $json, timestamp: new Date().toISOString() };
  }
  
  const processed = $json.data?.toUpperCase() || String($json.data || ''); // Safe processing
  return { success: true, data: processed, id: $json.id || 'generated-' + Date.now() };
} catch (error) {
  return { success: false, error: error.message || 'Unknown error', input: $json || {} };
}

// Batch mode: "runOnceForAllItems" 
const items = $input.all();
return items.map(item => {
  try {
    const data = item.json || {};
    return { json: { success: true, id: data.id || 'unknown', processed: data.value || 0 } };
  } catch (error) {
    return { json: { success: false, error: error.message || 'Processing failed', original: item.json || {} } };
  }
});
```

**Key Pattern**: Always use fallbacks (`||`) and optional chaining (`?.`) to prevent crashes from undefined values.

## Testing and Validation

### 1. Local Testing
```bash
# Start n8n resource
./scripts/resources/automation/n8n/manage.sh --action start

# Test workflow endpoints
curl -X POST http://localhost:5678/webhook/your-workflow \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

### 2. Integration Testing
```bash
# Test with actual scenario
# Place workflow in scenario's initialization/n8n/ folder
mkdir -p test-scenario/initialization/n8n/
cp your-workflow.json test-scenario/initialization/n8n/

# Test injection system
./scripts/resources/automation/n8n/lib/inject.sh test-scenario/initialization/n8n/
```

### 3. End-to-End Validation
- Test all workflow paths (success and error cases)
- Verify resource integrations work correctly
- Validate input/output data formats
- Test with realistic data volumes

## Deployment Integration

### 1. Scenario Integration
Place your workflow file in the appropriate scenario:
```bash
# For new scenarios
mkdir -p /path/to/scenario/initialization/n8n/
cp your-workflow.json /path/to/scenario/initialization/n8n/

# Update scenario service.json if needed
```

### 2. Injection System Testing
```bash
# Test workflow gets injected properly
./scripts/scenarios/tools/validate-scenario.sh your-scenario
```

### 3. Workflow Discovery
Ensure workflows are discoverable:
```bash
# Test n8n can import and execute your workflow
curl http://localhost:5678/rest/workflows
```

## Best Practices

### 1. Workflow Design
- **Single Responsibility**: Each workflow should have one clear purpose
- **Idempotent Operations**: Workflows should be safe to run multiple times
- **Clear Naming**: Use descriptive names for workflows and nodes
- **Always Use Fallbacks**: Every expression should have a fallback value
- **Use Code Node**: Never use deprecated Function nodes
- **Documentation**: Include descriptions for complex logic

### 2. Performance Optimization
- **Minimize API Calls**: Batch requests where possible
- **Efficient Data Processing**: Transform data efficiently in function nodes
- **Resource Management**: Clean up resources after use
- **Timeout Handling**: Set appropriate timeouts for external calls

### 3. Security Considerations
- **Input Validation**: Always validate incoming data
- **Credential Management**: Use n8n credential system for secrets
- **Rate Limiting**: Implement reasonable rate limits
- **Error Information**: Don't expose sensitive data in error messages

### 4. Maintainability
- **Version Control**: Track workflow changes
- **Testing**: Include test cases for all paths
- **Monitoring**: Add logging for debugging
- **Documentation**: Document complex business logic

## Common Integration Patterns

**Important**: Code nodes handle data transformation only. Use dedicated nodes for external operations:
- **HTTP Requests**: Use HTTP Request node with fallback expressions
- **Database Queries**: Use PostgreSQL/Redis nodes with error handling  
- **Data Processing**: Use Code node for transformation with validation

**Code Node Pattern with Fallbacks**:
```javascript
// Both modes - always use fallbacks for safety
const items = $input.all(); // Batch mode, or use $json for individual mode
return items.map(item => ({
  json: {
    id: item.json?.id || 'unknown',        // Safe object access
    name: item.json?.name || 'Unnamed',    // String fallback
    value: item.json?.value || 0,          // Number fallback
    processed: true
  }
}));
```

**Expression Fallbacks in Node Parameters**:
```javascript
"{{ $json['user']['name'] || 'Guest' }}"              // Object property
"{{ $node['Previous'].json['data'] || [] }}"          // Node reference
"{{ $json?.deeply?.nested?.value || 'fallback' }}"    // Optional chaining
"{{ parseInt($json.count) || 0 }}"                    // Type conversion
```

## Success Criteria

### ✅ Workflow Implementation
- [ ] Complete workflow JSON with all nodes defined
- [ ] All node connections properly configured
- [ ] Function nodes contain robust logic
- [ ] Error handling implemented throughout

### ✅ Integration Testing
- [ ] Workflow imports successfully to n8n
- [ ] All resource connections work
- [ ] Input/output data formats validated
- [ ] Error cases handled gracefully

### ✅ Performance Validation
- [ ] Workflow executes within reasonable time
- [ ] No memory leaks or resource issues
- [ ] Handles expected data volumes
- [ ] Timeouts configured appropriately

### ✅ Documentation
- [ ] Workflow purpose clearly documented
- [ ] Input/output specifications defined
- [ ] Integration patterns documented
- [ ] Troubleshooting guide provided

## Troubleshooting Guide

### Common Issues and Solutions

**1. Workflow Import Failures**
```bash
# Check JSON syntax
jq . your-workflow.json

# Validate against n8n schema
# Check for required fields: name, nodes, connections
```

**2. Node Execution Errors**
- Check Code node syntax (use `$input.all()` for batch, `$json` for individual)
- Ensure all expressions have fallback values (`|| 'default'`)
- Validate HTTP endpoints are accessible
- Verify resource ports are correct
- Test data transformations with sample data

**3. Resource Connection Issues**
```bash
# Verify resource is running
./scripts/resources/index.sh --action status

# Check resource ports
./scripts/resources/port_registry.sh --action list

# Test resource APIs directly
curl http://localhost:{port}/health
```

**4. Performance Issues**
- Profile workflow execution times
- Identify bottleneck nodes
- Optimize data transformations
- Implement caching where appropriate

## File Locations Reference

- **N8n Resource**: `scripts/resources/automation/n8n/`
- **Workflow Examples**: `scripts/scenarios/core/*/initialization/automation/n8n/`
- **N8n API Docs**: `scripts/resources/automation/n8n/docs/API.md`
- **Injection System**: `scripts/resources/automation/n8n/lib/inject.sh`

## Legacy Function Nodes

**Note**: This guide uses n8n's modern Code node (v0.198.0+). If migrating from deprecated Function nodes, replace `items` with `$input.all()` and `functionCode` with `jsCode`. See n8n documentation for full migration details.

## Advanced Patterns

### 1. Multi-Resource Orchestration
Create workflows that coordinate multiple resources:
- Database + AI model + file storage
- Multiple API services with fallbacks
- Complex data processing pipelines

### 2. Stateful Workflows
Implement workflows that maintain state:
- Multi-step user interactions
- Long-running processes
- Resumable operations

### 3. Real-time Processing
Design workflows for real-time scenarios:
- Webhook-based triggers
- Stream processing patterns
- Event-driven architectures

Remember: Every workflow you create makes scenarios more intelligent and capable. Build workflows that other scenarios can leverage and extend.