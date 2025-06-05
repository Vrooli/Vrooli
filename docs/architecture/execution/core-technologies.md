# Core Technologies

## **Terminology Definitions**

- **Team**: A persistent organizational unit at the strategic level, composed of humans and AI agents working together toward long-term goals. Teams define high-level objectives, allocate resources (credits, compute, personnel), and set overarching policies (e.g. security, compliance). Each Team maintains a shared context (members, roles, norms) that guides the swarms it spawns.  
- **Routine**: A reusable, versioned workflow that combines AI reasoning, API calls, code execution, and human oversight to accomplish specific tasks. Routines are the atomic units of automation in Vrooli. 

    *Note: Routines are always private by default, and can only be shared with other swarms you create unless you explicitly share them with the public.*
    
- **Run**: The execution instance of a routine, managed by the `RunStateMachine`.  
- **Navigator**: A pluggable component that translates between Vrooli's universal execution model and platform-specific workflow formats (BPMN, Langchain, etc.).  
- **Strategy**: The execution approach applied to a routine step (Conversational, Reasoning, or Deterministic), selected based on routine characteristics and context.  
- **Context**: The execution environment containing variables, state, permissions, and shared knowledge available to agents during routine execution.  
- **Chat:** The conversation data (messages, history, and persisted swarm context), which is tied 1-to-1 with a swarm  
- **Swarm:** The running swarm process managed by the `SwarmStateMachine`.

## **Hierarchical Boundaries**

```mermaid
graph TD
    subgraph "Strategic Boundary"
        Teams[Teams<br/>📈 Long-term goals, resource allocation<br/>🔄 Persistent organizational structures]
    end

    subgraph "Tactical Boundary"
        Swarms[Swarms<br/>🐝 Short-term objectives, dynamic coordination<br/>⏱️ Task-specific, disbanded when complete]
    end

    subgraph "Operational Boundary"
        Agents[Agents<br/>🤖 Persistent AI entities, specialized capabilities] 
        Routines[Routines<br/>⚙️ Versioned workflows, atomic automation<br/>📈 Evolve through usage]
    end

    Teams -.->|"Provides resources & strategic direction"| Swarms
    Swarms -.->|"Coordinates & assigns objectives"| Agents
    Agents -.->|"Execute & improve"| Routines

    classDef strategic fill:#e3f2fd,stroke:#1565c0,stroke-width:3px
    classDef tactical fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef operational fill:#e8f5e8,stroke:#2e7d32,stroke-width:2px

    class Teams strategic
    class Swarms tactical
    class Agents,Routines operational
```

## **Cross-Boundary Communication Protocols**

- **Strategic ↔ Tactical**: Resource allocation requests, goal decomposition, performance reports
- **Tactical ↔ Operational**: Task assignments, capability requests, execution status updates
- **Operational ↔ Operational**: Context sharing, routine invocation, result propagation

## Conceptual Foundation

## Core Hierarchy

```mermaid
graph TD
    Teams[Teams<br/>🏢 Organizations & Goals]
    Swarms[Swarms<br/>🐝 Dynamic Task Forces]
    Agents[Agents<br/>🤖 Specialized Workers]
    Routines[Routines<br/>⚙️ Reusable Processes]
    
    Teams --> Swarms
    Swarms --> Agents
    Agents --> Routines
    
    Teams -.->|"Provides resources,<br/>sets strategic goals"| Swarms
    Swarms -.->|"Recruits specialists,<br/>coordinates work"| Agents
    Agents -.->|"Execute processes,<br/>create improvements"| Routines
    
    classDef teams fill:#e3f2fd,stroke:#1565c0,stroke-width:3px
    classDef swarms fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef agents fill:#e8f5e8,stroke:#2e7d32,stroke-width:2px
    classDef routines fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    
    class Teams teams
    class Swarms swarms
    class Agents agents
    class Routines routines
```

### **Teams** (Strategic Level)
- **Purpose**: Long-term goals, resource allocation, strategic direction
- **Composition**: Humans + AI agents organized around business objectives
- **Lifecycle**: Persistent, evolving with organizational needs
- **Examples**: "Customer Success Team," "Product Development Team," "Research Division"

### **Swarms** (Coordination Level)
- **Purpose**: Dynamic task forces assembled for specific complex objectives
- **Composition**: Temporary coalitions of specialized agents
- **Lifecycle**: Created for tasks, disbanded when complete
- **Examples**: "Analyze Market Trends," "Build Customer Onboarding Flow," "Optimize Supply Chain"

### **Agents** (Execution Level)
- **Purpose**: Specialized workers with specific capabilities and personas
- **Composition**: Individual AI entities with defined roles and skills
- **Lifecycle**: Persistent, but recruited into different swarms as needed
- **Examples**: "Data Analyst," "Content Writer," "API Integration Specialist"

### **Routines** (Process Level)
- **Purpose**: Reusable automation building blocks
- **Composition**: Workflows combining AI reasoning, API calls, code, and human oversight
- **Lifecycle**: Versioned, improved over time through use and feedback
- **Examples**: "Market Research Report," "Customer Sentiment Analysis," "API Integration Template"

## Execution Strategy Evolution

> 📍 **Complete Strategy Details**: For comprehensive information about the four execution strategies (Conversational, Reasoning, Deterministic, Routing) and how routines evolve between them, see **[Strategy Evolution Agents](emergent-capabilities/agent-examples/strategy-evolution-agents.md)**.

**Conversational → Reasoning → Deterministic → Routing**

- **Conversational**: Initial human-like exploration and problem-solving
- **Reasoning**: Structured approaches as patterns emerge  
- **Deterministic**: Optimized automation for proven workflows
- **Routing**: Intelligent coordination for complex multi-phase tasks

### **The Evolution Mechanism: Top-Down Decomposition**

The key insight driving this evolution is **top-down decomposition enabled by recursive routine composition**. 

> 📖 **Implementation Details**: For detailed diagrams, implementation examples, and the complete decomposition process, see **[Strategy Evolution Agents](emergent-capabilities/agent-examples/strategy-evolution-agents.md)**.

### **Key Concepts**

- **Gradual Refinement**: Routines evolve incrementally as patterns emerge
- **Recursive Composition**: Sub-routines can use different strategies within parent routines
- **Context Inheritance**: Sub-routines inherit appropriate context while maintaining security boundaries
- **Learning Propagation**: Insights from execution inform optimization opportunities

This enables Vrooli's **compound knowledge effect** - every routine becomes a building block for more sophisticated automation.

### **The Decomposition Process**

```mermaid
sequenceDiagram
    participant User as User/Swarm
    participant RSM as RunStateMachine
    participant ConvStrat as Conversational Strategy
    participant ReasonStrat as Reasoning Strategy
    participant DetermStrat as Deterministic Strategy
    participant SubExec as Sub-routine Executor

    Note over User,SubExec: Evolution Through Decomposition

    User->>RSM: "Create marketing campaign"
    RSM->>ConvStrat: Initial execution (no patterns exist)
    ConvStrat->>ConvStrat: Natural language reasoning
    ConvStrat->>ConvStrat: Identify: research, content, distribution
    ConvStrat-->>RSM: Success + learned patterns

    Note over RSM: After multiple executions, patterns emerge

    User->>RSM: "Create marketing campaign" (later)
    RSM->>ReasonStrat: Structured execution (patterns identified)
    ReasonStrat->>SubExec: Sub-routine: "Market Research"
    ReasonStrat->>SubExec: Sub-routine: "Content Creation"
    ReasonStrat->>SubExec: Sub-routine: "Distribution Planning"
    SubExec-->>ReasonStrat: All sub-results
    ReasonStrat-->>RSM: Coordinated campaign

    Note over RSM: After many successful executions

    User->>RSM: "Create marketing campaign" (much later)
    RSM->>DetermStrat: Automated execution (best practices proven)
    DetermStrat->>SubExec: Deterministic: API-driven research
    DetermStrat->>SubExec: Deterministic: Template-based content
    DetermStrat->>SubExec: Reasoning: Strategic distribution
    SubExec-->>DetermStrat: Optimized results
    DetermStrat-->>RSM: Fully automated campaign
```

This recursive composition and gradual evolution is what enables Vrooli's **compound knowledge effect** - every routine becomes a building block for more sophisticated automation, creating an exponential growth in capability over time.

### **Context Inheritance and Data Flow**

Each routine execution creates its own **context object** that stores inputs and outputs. When routines contain sub-routines:

- **Context Inheritance**: Sub-routines inherit appropriate context from their parent
- **Selective Output**: Sub-routines specify which outputs to pass to the parent routine  
- **Sensitivity Levels**: Inputs and outputs have sensitivity classifications (public, internal, confidential, secret, PII) that influence data handling and security
- **Hierarchical Scoping**: Parents only receive outputs they're authorized for, maintaining security boundaries

This ensures that complex multi-level routines can share data appropriately while maintaining security and only keeping track of relevant information at each level.