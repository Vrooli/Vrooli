# N8n Workflow Development and Enhancement

You are tasked with creating or fixing n8n workflows in Vrooli. N8n workflows are the automation backbone that makes scenarios intelligent and autonomous - they orchestrate resources, process data, and implement complex business logic.

## Understanding N8n in Vrooli

**What N8n Workflows Are**: JSON-based visual automation workflows that connect resources, process data, and implement complex orchestration logic. They enable scenarios to be truly autonomous by automating routine tasks and decision-making.

**Why This Matters**: Workflows make scenarios easier to create and more powerful. A well-designed workflow can be reused across multiple scenarios, multiplying its value. Great workflows turn simple scenarios into intelligent applications.

**Important Version Note**: This guide uses n8n's modern Code node (v0.198.0+). The deprecated Function node should NOT be used in new workflows.

## Pre-Implementation Research

**Essential Reading**: Study n8n documentation at `scripts/resources/automation/n8n/README.md` and examine existing workflow patterns in `scripts/scenarios/core/*/initialization/automation/n8n/`. Focus on injection system and API patterns.

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

## Implementation Phases

### Phase 1: Planning & Architecture Design
- Document workflow requirements (purpose, inputs, outputs, resources)
- Map complete flow from trigger to response
- Identify resource integrations and error handling needs

### Phase 2: Core Implementation
- Create workflow JSON with modern Code nodes (not deprecated Function nodes)
- Essential patterns: `const input = $json; return {id: input.id || 'unknown'};`
- Use HTTP Request nodes for external calls, Code nodes only for data transformation
- Always include fallbacks: `$json?.field || 'default'`

### Phase 3: Integration & Testing  
- Connect to Vrooli resources using documented patterns
- Implement robust error handling with try/catch and fallbacks
- Test locally: `curl -X POST http://localhost:5678/webhook/your-workflow`

### Phase 4: Deployment & Validation
- Place workflow in scenario's `initialization/automation/n8n/` folder
- Test injection system and workflow discovery
- Validate all workflow paths with realistic data

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