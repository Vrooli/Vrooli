# MCP Tool Integration

This module provides integration between the Vrooli execution architecture and the Model Context Protocol (MCP) tool system.

## Architecture

The integration bridges the existing MCP implementation with the three-tier execution architecture:

```
┌─────────────────────────────────────────────────────┐
│                 Tier 3 - Execution                  │
│  ┌─────────────────────────────────────────────┐   │
│  │          ToolOrchestrator                   │   │
│  │  - Tool discovery and execution             │   │
│  │  - Approval management                      │   │
│  │  - Resource tracking                        │   │
│  └──────────────────┬──────────────────────────┘   │
│                     │                               │
│  ┌──────────────────▼──────────────────────────┐   │
│  │       IntegratedToolRegistry                │   │
│  │  - Unified tool access                      │   │
│  │  - Dynamic tool registration                │   │
│  │  - Execution delegation                     │   │
│  └──────────────────┬──────────────────────────┘   │
└─────────────────────┼───────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────┐
│               MCP Infrastructure                     │
│  ┌─────────────────────────────────────────────┐   │
│  │         ToolRegistry (MCP)                  │   │
│  │  - Tool definitions                         │   │
│  │  - Schema management                        │   │
│  └──────────────────┬──────────────────────────┘   │
│                     │                               │
│  ┌──────────────────▼──────────────────────────┐   │
│  │       CompositeToolRunner                   │   │
│  │  ┌────────────┐  ┌────────────────────┐   │   │
│  │  │McpToolRunner│  │OpenAIToolRunner    │   │   │
│  │  └────────────┘  └────────────────────┘   │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │          Tool Implementations               │   │
│  │  - BuiltInTools (CRUD, messaging, etc.)    │   │
│  │  - SwarmTools (state management)           │   │
│  │  - OpenAI tools (web_search, etc.)         │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

## Components

### IntegratedToolRegistry

The central registry that provides:
- Unified access to all tool types (MCP, Swarm, OpenAI)
- Dynamic tool registration for runtime additions
- Execution context management
- Approval tracking and enforcement
- Usage metrics

### Tool Types

1. **Built-in MCP Tools**
   - `define_tool` - Dynamic schema generation
   - `resource_manage` - CRUD operations on Vrooli resources
   - `send_message` - Message sending
   - `run_routine` - Routine execution
   - `spawn_swarm` - Swarm creation

2. **Swarm Tools** (available in swarm context)
   - `update_swarm_shared_state` - Modify swarm state
   - `end_swarm` - Terminate swarm execution

3. **OpenAI Tools**
   - `web_search` - Search the web
   - `file_search` - Search uploaded files

4. **Dynamic Tools**
   - Registered at runtime by routines or swarms
   - Scoped to specific execution contexts

## Usage

### Basic Tool Execution

```typescript
const registry = IntegratedToolRegistry.getInstance(logger);

const context: IntegratedToolContext = {
    stepId: "step_123",
    runId: "run_456",
    user: { id: "user_789", languages: ["en"] },
    logger,
};

const result = await registry.executeTool({
    toolName: "resource_manage",
    parameters: {
        operation: "find",
        resourceType: "Note",
        filters: { isPrivate: true },
    },
}, context);
```

### Dynamic Tool Registration

```typescript
registry.registerDynamicTool({
    name: "custom_analyzer",
    description: "Analyze data with custom logic",
    inputSchema: {
        type: "object",
        properties: {
            data: { type: "array" },
            method: { type: "string" },
        },
        required: ["data"],
    },
}, {
    runId: "run_456",
    scope: "run",
});
```

### Tool Approval

High-risk tools require user approval before execution:

```typescript
// Register approval request
const approvalId = await registry.registerPendingApproval(
    "run_routine",
    { routineId: "routine_123" },
    context,
    600000, // 10 minute timeout
);

// Process approval (typically from user interaction)
registry.processApproval(approvalId, true, "user_789");
```

## Tool Context

The `IntegratedToolContext` provides:
- `stepId` - Current execution step
- `runId` - Parent run (if applicable)
- `swarmId` - Parent swarm (if applicable)
- `conversationId` - Chat conversation (if applicable)
- `user` - User session information
- `logger` - Contextual logger
- `metadata` - Additional context data

## Error Handling

The registry handles various error types:
- Tool not found
- Missing required context (user, conversation, etc.)
- Approval required/denied
- Execution failures with retry support
- Resource quota exceeded

## Security

- All MCP tools require user authentication
- High-risk operations require explicit approval
- Resource usage is tracked and limited
- Tools are scoped to execution contexts

## Integration Points

### From ToolOrchestrator (Tier 3)

```typescript
// Configure orchestrator with context
toolOrchestrator.configureForExecution(
    stepId,
    availableTools,
    resourceManager,
    {
        runId,
        swarmId,
        conversationId,
        user,
    },
);

// Execute tool
const result = await toolOrchestrator.executeTool({
    toolName: "send_message",
    parameters: { ... },
});
```

### From Swarm Execution (Tier 1)

Swarms can execute tools through their execution context, which will route through the integrated registry with proper swarm context.

### From Routine Execution (Tier 2)

Routines define available tools in their configuration, which are automatically registered as dynamic tools scoped to the run.

## Performance Considerations

- Tool definitions are cached after first retrieval
- Dynamic tools are cleaned up after execution completes
- Approval checks are optimized with in-memory cache
- Usage metrics are aggregated asynchronously

## Future Enhancements

1. **Tool Composition** - Combine multiple tools into workflows
2. **Tool Versioning** - Support multiple versions of the same tool
3. **External Tool Providers** - Plugin system for third-party tools
4. **Tool Analytics** - Detailed usage analytics and optimization
5. **Tool Testing** - Automated testing framework for tool implementations