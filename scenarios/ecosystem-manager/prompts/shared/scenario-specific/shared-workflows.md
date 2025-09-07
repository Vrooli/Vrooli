# Shared Workflows

## Purpose
Shared workflows are reusable components that multiple scenarios can leverage. Instead of each scenario implementing its own logic, shared workflows provide consistent, tested functionality that reduces complexity and improves reliability.

## Philosophy

### Why Shared Workflows Matter
- **Reduce duplication** - Write once, use everywhere
- **Improve reliability** - Tested components used broadly
- **Simplify scenarios** - Focus on unique value, not plumbing
- **Enable updates** - Fix once, benefit all users
- **Accelerate development** - Build on proven components

### When to Use Shared Workflows
✅ **Use shared when:**
- Functionality is needed by 2+ scenarios
- Logic is generic and parameterizable
- Behavior should be consistent across scenarios
- Updates should propagate to all users
- Complexity warrants centralization

❌ **Keep scenario-specific when:**
- Logic is unique to one scenario
- Tight coupling with scenario internals
- Performance requires optimization
- Security requires isolation
- Customization would break genericity

## Shared Workflow Locations

### N8n Workflows
```
initialization/n8n/
├── ollama.json                    # LLM interaction
├── claude-code-executor.json      # Code generation
├── document-converter-validator.json  # Document processing
├── embedding-generator.json       # Vector embeddings
├── rate-limiter.json              # API rate limiting
├── cache-manager.json             # Caching logic
├── web-research-aggregator.json   # Web scraping
├── workflow-executor.json         # Meta-workflow execution
└── [other shared workflows]
```

### Node-RED Flows
```
initialization/node-red/
├── data-transform.json            # Data transformation
├── event-router.json              # Event routing
├── monitoring-dashboard.json      # Monitoring
└── [other shared flows]
```

### Huginn Agents
```
initialization/huginn/
├── web-monitor.json               # Web monitoring
├── rss-aggregator.json            # RSS feeds
├── email-processor.json           # Email handling
└── [other shared agents]
```

## Using Shared Workflows

### In Scenario Initialization
```javascript
// In scenario's initialization/setup.js
async function setupScenario() {
    // Import shared workflows
    const sharedWorkflows = [
        'ollama.json',           // For LLM capabilities
        'cache-manager.json',     // For caching
        'rate-limiter.json'       // For API limits
    ];
    
    for (const workflow of sharedWorkflows) {
        await importSharedWorkflow(workflow);
    }
    
    // Import scenario-specific workflows
    await importScenarioWorkflows();
}
```

### Calling Shared Workflows
```javascript
// Instead of direct API calls
// BAD: Direct implementation
const response = await fetch('http://ollama:11434/api/generate', {
    method: 'POST',
    body: JSON.stringify({ prompt: userPrompt })
});

// GOOD: Use shared workflow
const response = await executeWorkflow('ollama', {
    prompt: userPrompt,
    model: 'llama2',
    temperature: 0.7
});
```

### Parameterizing Shared Workflows
```json
// Shared workflow with parameters
{
  "name": "ollama",
  "nodes": [
    {
      "parameters": {
        "prompt": "={{ $json.prompt }}",
        "model": "={{ $json.model || 'llama2' }}",
        "temperature": "={{ $json.temperature || 0.7 }}"
      }
    }
  ]
}
```

## Creating Shared Workflows

### Criteria for Sharing
Before creating a shared workflow, ensure:
- [ ] Used by at least 2 scenarios (or will be)
- [ ] Generic enough to be parameterized
- [ ] Clear input/output contract
- [ ] No scenario-specific hardcoding
- [ ] Documented usage examples
- [ ] Test coverage exists

### Shared Workflow Template
```json
{
  "name": "shared-workflow-name",
  "description": "Clear description of what this does",
  "version": "1.0.0",
  "parameters": {
    "required": ["param1", "param2"],
    "optional": {
      "param3": "default_value"
    }
  },
  "input": {
    "schema": {
      "type": "object",
      "properties": {
        "param1": {"type": "string"},
        "param2": {"type": "number"}
      }
    }
  },
  "output": {
    "schema": {
      "type": "object",
      "properties": {
        "result": {"type": "string"},
        "status": {"type": "string"}
      }
    }
  },
  "nodes": [
    // Workflow implementation
  ]
}
```

### Documentation Requirements
```markdown
# Shared Workflow: [Name]

## Purpose
[What this workflow does]

## Usage
\`\`\`javascript
const result = await executeWorkflow('[name]', {
    param1: 'value1',
    param2: 123
});
\`\`\`

## Parameters
- **param1** (required): Description
- **param2** (required): Description
- **param3** (optional, default: X): Description

## Output
\`\`\`json
{
  "result": "...",
  "status": "success|error"
}
\`\`\`

## Examples
[2-3 concrete examples]

## Used By
- scenario-1
- scenario-2
```

## Common Shared Workflows

### 1. LLM Interaction (ollama.json)
```javascript
// Purpose: Consistent LLM interaction
await executeWorkflow('ollama', {
    prompt: 'Generate a summary',
    model: 'llama2',
    max_tokens: 500
});
```

### 2. Document Processing
```javascript
// Purpose: Convert and validate documents
await executeWorkflow('document-converter-validator', {
    file: '/path/to/document.pdf',
    output_format: 'markdown',
    validate: true
});
```

### 3. Web Research
```javascript
// Purpose: Aggregate web research
await executeWorkflow('web-research-aggregator', {
    query: 'latest AI trends',
    sources: ['google', 'bing', 'duckduckgo'],
    max_results: 10
});
```

### 4. Caching
```javascript
// Purpose: Intelligent caching
await executeWorkflow('cache-manager', {
    action: 'get|set|invalidate',
    key: 'user:123:profile',
    value: profileData,
    ttl: 3600
});
```

### 5. Rate Limiting
```javascript
// Purpose: API rate limiting
await executeWorkflow('rate-limiter', {
    api: 'openai',
    request_count: 1,
    wait_if_limited: true
});
```

## Workflow Versioning

### Version Management
```json
{
  "name": "shared-workflow",
  "version": "2.0.0",  // Semantic versioning
  "compatibility": {
    "min_version": "1.0.0",  // Minimum compatible
    "breaking_changes": ["2.0.0"]  // Breaking versions
  }
}
```

### Handling Updates
```javascript
// Check version compatibility
async function useSharedWorkflow(name, params) {
    const workflow = await getWorkflow(name);
    const requiredVersion = scenario.requirements[name];
    
    if (!isCompatible(workflow.version, requiredVersion)) {
        console.warn(`Workflow ${name} version mismatch`);
        // Use compatibility mode or fail
    }
    
    return executeWorkflow(name, params);
}
```

## Best Practices

### DO's
✅ **Keep generic** - No scenario-specific logic
✅ **Document thoroughly** - Clear usage examples
✅ **Version properly** - Semantic versioning
✅ **Test extensively** - Shared = critical
✅ **Handle errors gracefully** - Return consistent errors
✅ **Monitor usage** - Track which scenarios use what

### DON'Ts
❌ **Don't hardcode values** - Use parameters
❌ **Don't break compatibility** - Version breaking changes
❌ **Don't create premature abstractions** - Wait for 2+ users
❌ **Don't bypass shared workflows** - Consistency matters
❌ **Don't modify in scenarios** - Changes go to source

## Migration to Shared Workflows

### Identifying Candidates
```bash
# Find duplicate implementations
grep -r "ollama:11434" scenarios/
# Multiple hits = candidate for sharing

# Find similar patterns
grep -r "fetch.*api/generate" scenarios/
# Similar patterns = extraction opportunity
```

### Extraction Process
1. Identify common pattern across scenarios
2. Extract to shared workflow
3. Parameterize scenario-specific values
4. Test with all affected scenarios
5. Update scenarios to use shared version
6. Document in shared workflow library

## Monitoring and Maintenance

### Usage Tracking
```javascript
// Track workflow usage
async function executeWorkflow(name, params) {
    // Log usage
    await logWorkflowUsage({
        workflow: name,
        scenario: getCurrentScenario(),
        timestamp: Date.now(),
        success: true
    });
    
    // Execute
    return await runWorkflow(name, params);
}
```

### Health Monitoring
```javascript
// Monitor shared workflow health
async function monitorSharedWorkflows() {
    const workflows = await listSharedWorkflows();
    
    for (const workflow of workflows) {
        const health = await checkWorkflowHealth(workflow);
        if (!health.healthy) {
            alert(`Shared workflow ${workflow} unhealthy: ${health.error}`);
        }
    }
}
```

## Remember

**Shared workflows are force multipliers** - One improvement helps many

**Quality over quantity** - Better to have few excellent shared workflows

**Documentation is critical** - Others need to understand usage

**Compatibility matters** - Breaking changes affect many scenarios

**Monitor and maintain** - Shared = critical infrastructure

Shared workflows embody the DRY principle at the scenario level. They're the building blocks that make complex scenarios simple and reliable.