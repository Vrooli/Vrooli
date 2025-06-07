# Tool Integration in ExecutionArchitecture

## Overview

The ExecutionArchitecture factory now properly initializes and wires the IntegratedToolRegistry into the three-tier execution system. This ensures that all tiers have access to a unified tool infrastructure that bridges MCP tools, OpenAI tools, and dynamic runtime tools.

## Key Components

### 1. IntegratedToolRegistry
The central registry that manages all tool types:
- **MCP Built-in Tools**: defineTool, resourceManage, sendMessage, etc.
- **Swarm Tools**: updateSwarmSharedState, endSwarm
- **OpenAI Tools**: web_search, file_search
- **Dynamic Tools**: Runtime-registered tools specific to runs or swarms

### 2. ConversationStateStore
Provides persistent storage for conversation state and tool configurations:
- Backed by Prisma for database persistence
- L1/L2 caching for performance
- Integrates with SwarmTools for swarm-specific operations

### 3. ToolOrchestrator (Tier 3)
Executes tools within the context of step execution:
- Configured per execution with available tools
- Handles resource tracking and quotas
- Manages retry policies and error handling
- Emits telemetry for monitoring

## Architecture Flow

```
ExecutionArchitecture.create()
    ├── initializeSharedServices()
    │   ├── EventBus (Redis/In-memory)
    │   ├── State Stores (Swarm & Run)
    │   ├── ConversationStateStore (Prisma + Cache)
    │   └── IntegratedToolRegistry (Singleton)
    │
    ├── initializeTier3()
    │   └── UnifiedExecutor
    │       └── ToolOrchestrator (uses IntegratedToolRegistry)
    │
    ├── initializeTier2()
    │   └── TierTwoRunStateMachine
    │
    └── initializeTier1()
        └── TierOneSwarmStateMachine
```

## Usage Example

```typescript
// Create architecture with tool integration
const architecture = await ExecutionArchitecture.create({
    useRedis: true,
    telemetryEnabled: true,
});

// Access the tool registry
const toolRegistry = architecture.getToolRegistry();

// Register a dynamic tool
toolRegistry.registerDynamicTool({
    name: "custom_tool",
    description: "My custom tool",
    inputSchema: { /* ... */ },
}, {
    runId: "run_123",
    scope: "run",
});

// Execute a step that uses tools
const tier3 = architecture.getTier3();
const result = await tier3.execute({
    // Execution request with tool resources
});
```

## Tool Lifecycle

1. **Initialization**: IntegratedToolRegistry is created as a singleton during shared services initialization
2. **Configuration**: ToolOrchestrator is configured per execution with available tools
3. **Execution**: Tools are executed through the registry with proper context
4. **Tracking**: Usage metrics and telemetry are collected
5. **Cleanup**: Dynamic tools are unregistered when runs/swarms complete

## Benefits

- **Unified Interface**: Single registry for all tool types
- **Resource Management**: Integrated quota and cost tracking
- **Approval System**: High-risk operations require user approval
- **Telemetry**: Comprehensive monitoring and metrics
- **Flexibility**: Dynamic tool registration for runtime extensibility

## Configuration

The ExecutionArchitecture accepts options to customize the tool infrastructure:

```typescript
interface ExecutionArchitectureOptions {
    useRedis?: boolean;        // Use Redis for state (production)
    telemetryEnabled?: boolean; // Enable tool usage telemetry
    logger?: Logger;           // Custom logger instance
    config?: {
        tier3?: {
            // UnifiedExecutor config including tool limits
        }
    };
}
```

## Future Enhancements

1. **Tool Discovery**: Automatic discovery of available tools from external sources
2. **Tool Marketplace**: Registry of community-contributed tools
3. **Tool Composition**: Combining multiple tools into higher-level operations
4. **Tool Versioning**: Support for multiple versions of the same tool
5. **Tool Analytics**: Advanced analytics on tool usage patterns