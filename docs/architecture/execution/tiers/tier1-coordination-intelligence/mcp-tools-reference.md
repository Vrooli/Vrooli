# 🔧 MCP Tools Reference: Actual Coordination Capabilities

> **TL;DR**: This is the comprehensive reference for all MCP (Model Context Protocol) tools actually available to Tier 1 coordination agents. These tools enable natural language coordination by providing structured ways to interact with the Vrooli system.

---

## 🎯 Tool Philosophy

MCP tools in Vrooli are designed to feel **natural** to AI agents while providing structured interfaces to the underlying system. They follow a consistent pattern of accepting structured parameters and returning standardized responses.

```typescript
// Example of actual tool usage
await update_swarm_shared_state({
    subTasks: {
        set: [
            { id: "T1", description: "Collect financial data", status: "todo" },
            { id: "T2", description: "Generate insights", status: "todo", depends_on: ["T1"] }
        ]
    }
});
```

---

## 📋 Core System Tools

### **🔧 define_tool**

**Purpose**: Get detailed parameters and schemas for other tools based on a variant

**Usage Pattern**: Agents use this to understand the exact parameters required for complex operations like resource management

```typescript
interface DefineToolParams {
    toolName: string;     // Tool to get definition for (e.g., "resource_manage")
    variant: string;      // Resource type variant (e.g., "Note", "Routine")
    op: string;          // Operation type (e.g., "find", "add", "update", "delete")
}

// Example: Get schema for creating a Note
await define_tool({
    toolName: "resource_manage",
    variant: "Note", 
    op: "add"
});
```

### **💬 send_message**

**Purpose**: Send messages to new or existing chat conversations

**Usage Pattern**: Agents use this to communicate with users, other agents, or specific chats

```typescript
interface SendMessageParams {
    recipient: 
        | { kind: "chat"; chatId: string }
        | { kind: "bot"; botId: string }          // Not yet implemented
        | { kind: "user"; userId: string }        // Not yet implemented  
        | { kind: "topic"; topic: string };      // Not yet implemented
    content: string | Record<string, unknown>;
    metadata?: {
        messageConfig?: MessageConfigObject;
        mcpLlmModel?: string;
        mcpLlmTaskContexts?: TaskContextInfo[];
    };
}

// Example: Send message to a chat
await send_message({
    recipient: { kind: "chat", chatId: "chat_123" },
    content: "Analysis complete. Here are the key findings...",
    metadata: {
        mcpLlmModel: "gpt-4",
        messageConfig: {
            __version: "1.0",
            resources: ["report_v1"],
            role: "assistant"
        }
    }
});
```

### **📊 resource_manage**

**Purpose**: CRUD operations for any resource type in the system

**Usage Pattern**: Universal tool for finding, creating, updating, and deleting resources

```typescript
interface ResourceManageParams {
    op: "find" | "add" | "update" | "delete";
    resource_type: string;                    // Resource type/variant
    // For find operations:
    filters?: Record<string, unknown>;
    // For add operations:
    attributes?: Record<string, unknown>;
    // For update/delete operations:
    id?: string;
}

// Example: Find existing routines
await resource_manage({
    op: "find",
    resource_type: "Routine",
    filters: {
        name: "market analysis",
        isPrivate: false
    }
});

// Example: Create a new Note
await resource_manage({
    op: "add", 
    resource_type: "Note",
    attributes: {
        name: "Market Analysis Notes",
        content: "Key insights from today's analysis..."
    }
});
```

### **⚙️ run_routine**

**Purpose**: Execute routines either inline (synchronous) or as jobs (asynchronous)

**Usage Pattern**: Run existing workflows and routines as part of coordination

```typescript
interface RunRoutineParams {
    action: "start" | "stop" | "pause" | "resume";
    // For start action:
    routineId?: string;
    inputs?: Record<string, unknown>;
    // For other actions:
    runId?: string;
}

// Example: Start a routine
await run_routine({
    action: "start",
    routineId: "financial_data_collection_v3",
    inputs: {
        timeframe: "Q1_2024",
        sources: ["yahoo_finance", "alpha_vantage"],
        format: "json"
    }
});

// Example: Stop a running routine
await run_routine({
    action: "stop",
    runId: "run_abc123"
});
```

### **🐝 spawn_swarm**

**Purpose**: Start new swarm sessions with bots or teams

**Usage Pattern**: Create new swarms for complex multi-agent tasks

```typescript
interface SpawnSwarmParams {
    kind: "simple" | "team";
    goal: string;
    // For simple swarms:
    swarmLeader?: string;     // Bot ID
    // For team swarms:
    teamId?: string;
}

// Example: Spawn simple swarm
await spawn_swarm({
    kind: "simple",
    swarmLeader: "analyst_bot_001",
    goal: "Analyze Q1 financial performance and identify trends"
});

// Example: Spawn team swarm
await spawn_swarm({
    kind: "team",
    teamId: "financial_analysis_team",
    goal: "Complete comprehensive market analysis for board presentation"
});
```

---

## 🐝 Swarm-Specific Tools

### **🔄 update_swarm_shared_state**

**Purpose**: Update swarm's shared state including subtasks, blackboard, and team configuration

**Usage Pattern**: Primary tool for swarm coordination - manages all shared state

```typescript
interface UpdateSwarmSharedStateParams {
    subTasks?: {
        set?: SwarmSubTask[];     // Add or update subtasks
        delete?: string[];        // Remove subtasks by ID
    };
    blackboard?: {
        set?: Array<{ id: string; value: unknown }>;  // Add/update blackboard items
        delete?: string[];                            // Remove blackboard items by ID
    };
    teamConfig?: {
        structure?: {
            type?: string;        // e.g., "MOISE+"
            version?: string;     // e.g., "1.0"
            content?: string;     // MOISE+ specification content
        };
    };
}

// Example: Add subtasks and update blackboard
await update_swarm_shared_state({
    subTasks: {
        set: [
            {
                id: "T1",
                description: "Collect financial data from APIs",
                status: "todo",
                priority: "high",
                assignee_bot_id: "data_specialist_001",
                created_at: new Date().toISOString()
            },
            {
                id: "T2",
                description: "Analyze trends and patterns", 
                status: "todo",
                depends_on: ["T1"],
                assignee_bot_id: "analyst_bot_002",
                priority: "medium",
                created_at: new Date().toISOString()
            }
        ]
    },
    blackboard: {
        set: [
            { id: "data_sources", value: ["yahoo_finance", "alpha_vantage", "fred"] },
            { id: "analysis_timeframe", value: "Q1_2024" }
        ]
    }
});

// Example: Update team organizational structure
await update_swarm_shared_state({
    teamConfig: {
        structure: {
            type: "MOISE+",
            version: "1.0",
            content: `
                structure AnalysisTeam {
                    group Research {
                        role leader cardinality 1..1
                        role analyst cardinality 2..3
                    }
                }
            `
        }
    }
});
```

### **🛑 end_swarm**

**Purpose**: End the swarm session when goal is complete or limits are reached

**Usage Pattern**: Gracefully or forcefully terminate swarm operations

```typescript
interface EndSwarmParams {
    mode?: "graceful" | "force";
    reason?: string;
}

// Example: Gracefully end swarm
await end_swarm({
    mode: "graceful",
    reason: "Goal completed successfully - market analysis report generated"
});

// Example: Force end swarm due to resource limits
await end_swarm({
    mode: "force", 
    reason: "Credit limit exceeded"
});
```

---

## 🔧 Tool Integration Patterns

### **Resource Discovery and Execution Pattern**

```typescript
// 1. Find existing resources
const existingRoutines = await resource_manage({
    op: "find",
    resource_type: "Routine",
    filters: { name: "market analysis" }
});

// 2. Run appropriate routine
if (existingRoutines.length > 0) {
    await run_routine({
        action: "start",
        routineId: existingRoutines[0].id,
        inputs: { timeframe: "Q1_2024" }
    });
} else {
    // 3. Create new resource if needed
    await resource_manage({
        op: "add",
        resource_type: "Note",
        attributes: {
            name: "Market Analysis TODO",
            content: "Need to create market analysis routine"
        }
    });
}
```

### **Swarm Coordination Pattern**

```typescript
// 1. Update shared state with new tasks
await update_swarm_shared_state({
    subTasks: {
        set: [
            { id: "T1", description: "Data collection", status: "todo" },
            { id: "T2", description: "Analysis", status: "todo", depends_on: ["T1"] }
        ]
    }
});

// 2. Execute routines for subtasks
await run_routine({
    action: "start", 
    routineId: "data_collection_routine",
    inputs: { taskId: "T1" }
});

// 3. Update progress
await update_swarm_shared_state({
    subTasks: {
        set: [
            { id: "T1", description: "Data collection", status: "in_progress" }
        ]
    },
    blackboard: {
        set: [
            { id: "T1_progress", value: "Started data collection from 3 sources" }
        ]
    }
});
```

### **Communication and Delegation Pattern**

```typescript
// 1. Send status update to stakeholders
await send_message({
    recipient: { kind: "chat", chatId: "project_updates_chat" },
    content: "Market analysis swarm initiated. Expected completion in 2 hours.",
    metadata: {
        mcpLlmModel: "gpt-4",
        messageConfig: {
            __version: "1.0",
            resources: [],
            role: "assistant"
        }
    }
});

// 2. Spawn specialized sub-swarm if needed
await spawn_swarm({
    kind: "simple",
    swarmLeader: "risk_analysis_specialist",
    goal: "Perform detailed risk assessment of identified market trends"
});
```

---

## 📚 Tool Selection Guidelines

| Use Case | Primary Tool | Supporting Tools |
|----------|-------------|------------------|
| **Task Management** | `update_swarm_shared_state` | `send_message` |
| **Resource Discovery** | `resource_manage` (op: "find") | `define_tool` |
| **Resource Creation** | `resource_manage` (op: "add") | `define_tool` |
| **Workflow Execution** | `run_routine` | `update_swarm_shared_state` |
| **Team Formation** | `spawn_swarm` | `update_swarm_shared_state` |
| **Communication** | `send_message` | `update_swarm_shared_state` |
| **Swarm Termination** | `end_swarm` | `send_message` |

---

## ⚠️ Implementation Notes

### **Current Limitations**
- `send_message` only supports chat recipients; bot, user, and topic recipients are not yet implemented
- `run_routine` operations other than "start" are not fully implemented
- `spawn_swarm` is not fully implemented yet

### **Error Handling**
All tools return consistent response formats:
```typescript
interface ToolResponse {
    isError: boolean;
    content: Array<{ type: "text"; text: string }>;
}
```

### **Authentication**
- Most tools require authenticated users with appropriate permissions
- Swarm tools require the user to be the swarm initiator or admin for sensitive operations

---

## 🔗 Related Documentation

- **[Swarm State Machine](swarm-state-machine.md)** - How tools integrate with state management
- **[MOISE+ Comprehensive Guide](moise-comprehensive-guide.md)** - Team configuration structure for `update_swarm_shared_state`
- **[Implementation Architecture](implementation-architecture.md)** - Technical tool routing and execution
- **[Autonomous Operations](autonomous-operations.md)** - How tools enable self-directed coordination

---

> 💡 **Important**: This documentation reflects the actual MCP tools implementation. The tools are designed to be composable - complex coordination behaviors emerge from combining these foundational tools rather than having specialized coordination tools for every use case. 