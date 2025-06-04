# Tier 1: Coordination Intelligence

**Purpose**: Dynamic swarm coordination through AI metacognition and prompt-based reasoning

Unlike traditional multi-agent systems with rigid coordination services, Vrooli's Tier 1 leverages **AI metacognition** - the ability for agents to reason about their own thinking and coordinate dynamically through natural language understanding. This creates an infinitely flexible coordination layer that evolves with AI capabilities.

## 📚 Documentation Structure

This documentation is organized into several focused sections:

- **[Metacognitive Framework](./metacognitive-framework.md)** - The core advantage of prompt-based coordination
- **[Implementation Architecture](./implementation-architecture.md)** - Technical architecture and components
- **[SwarmStateMachine](./swarm-state-machine.md)** - State machine architecture and lifecycle management
- **[Autonomous Operations](./autonomous-operations.md)** - Self-directed task management and monitoring
- **[Conclusion](./conclusion.md)** - Why prompt-based metacognition wins

## 🧠 Core Architecture Overview

```mermaid
graph TB
    subgraph "Tier 1: Prompt-Based Coordination Intelligence"
        SwarmStateMachine[SwarmStateMachine<br/>🎯 Swarm lifecycle management<br/>📋 State persistence<br/>🔄 Event routing]
        
        subgraph "Metacognitive Framework"
            PromptEngine[Prompt Engine<br/>🧠 Role-aware system prompts<br/>📊 Dynamic context injection<br/>🎯 Goal framing]
            
            MoiseSerializer[MOISE+ Serializer<br/>📦 Inject roles / missions / norms<br/>⬇️ Into leader prompt]
            
            MCPTools[MCP Tool Suite<br/>🔧 update_swarm_shared_state<br/>📋 manage_subtasks<br/>👥 delegate_roles<br/>📢 subscribe_to_events]
            
            SwarmContext[Swarm Context<br/>📊 Current state<br/>🎯 Goals & subtasks<br/>👥 Team structure<br/>📝 Execution history]
        end
        
        subgraph "Dynamic Capabilities (via Prompting)"
            RecruitmentLogic[Recruitment Logic<br/>🔍 Look for suitable team<br/>👥 Create new team if needed<br/>🎯 Domain expertise matching]
            
            TaskDecomposition[Task Decomposition<br/>📋 Break down complex goals<br/>🔗 Identify dependencies<br/>⏱️ Estimate effort]
            
            ResourceAllocation[Resource Allocation<br/>💰 Track credit usage<br/>⏱️ Monitor time limits<br/>🎯 Optimize allocation]
            
            EventCoordination[Event Coordination<br/>📢 Route events to specialists<br/>🔔 Subscribe to topics<br/>🔄 Handle async callbacks]
        end
        
        subgraph "Team Organization (MOISE+)"
            TeamConfig[Team Config<br/>🏗️ Organizational structure<br/>👥 Role definitions<br/>🔗 Authority relations<br/>📋 Norms & obligations]
        end
    end
    
    %% Connections
    SwarmStateMachine --> PromptEngine
    SwarmStateMachine --> SwarmContext
    PromptEngine --> MoiseSerializer
    PromptEngine --> MCPTools
    MCPTools --> SwarmContext
    
    %% Dynamic capabilities emerge from prompting
    PromptEngine -.->|"Enables reasoning about"| RecruitmentLogic
    PromptEngine -.->|"Enables reasoning about"| TaskDecomposition
    PromptEngine -.->|"Enables reasoning about"| ResourceAllocation
    PromptEngine -.->|"Enables reasoning about"| EventCoordination
    
    TeamConfig --> SwarmContext
    TeamConfig -.->|"Informs role behavior"| PromptEngine
    
    classDef orchestrator fill:#e3f2fd,stroke:#1565c0,stroke-width:3px
    classDef framework fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef dynamic fill:#e8f5e8,stroke:#2e7d32,stroke-width:2px,stroke-dasharray:5 5
    classDef team fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    
    class SwarmStateMachine orchestrator
    class PromptEngine,MoiseSerializer,MCPTools,SwarmContext framework
    class RecruitmentLogic,TaskDecomposition,ResourceAllocation,EventCoordination dynamic
    class TeamConfig team
```

## 🎯 Key Concepts

### Metacognitive Coordination
Traditional multi-agent systems hard-code coordination logic into separate services. Vrooli takes a radically different approach: **coordination emerges from AI reasoning**. Agents understand their roles, analyze situations, and coordinate naturally through prompts.

### MOISE+ Organizational Modeling
Teams define rich organizational structures using MOISE+ notation, providing a formal grammar for describing who **may/must/must-not** do any piece of work. This creates sophisticated coordination without hard-coding.

### Dynamic Capabilities
All coordination behaviors emerge from prompting:
- **Hierarchical**: Leader delegates to specialists
- **Peer-to-peer**: Agents collaborate directly via events
- **Emergent**: Patterns evolve based on task success
- **Hybrid**: Mix strategies as needed

### Tool-Mediated Actions
Instead of API calls to coordination services, agents use MCP tools that feel natural:
```typescript
await update_swarm_shared_state({
    subtasks: [
        { id: "T1", description: "Analyze market trends", status: "todo" },
        { id: "T2", description: "Generate report", status: "todo", depends_on: ["T1"] }
    ],
    subtaskLeaders: { "T1": "analyst_bot_123" }
});
```

## 🚀 Get Started

Begin with the **[Metacognitive Framework](./metacognitive-framework.md)** to understand the foundational concepts, then explore the **[Implementation Architecture](./implementation-architecture.md)** for technical details.

For understanding the autonomous operation capabilities, see **[Autonomous Operations](./autonomous-operations.md)**. 