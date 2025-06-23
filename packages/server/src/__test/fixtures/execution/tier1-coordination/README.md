# Tier 1: Coordination Intelligence Fixtures

This directory contains fixtures that test the **Coordination Intelligence layer** (`/services/execution/tier1/`) - the strategic swarm coordination tier. These fixtures serve as **building blocks** that are reused by emergent-capabilities and integration-scenarios.

## 🧠 Overview

Tier 1 represents the "thinking about thinking" layer where:
- Swarms of agents coordinate to achieve complex goals
- Organizational structures emerge based on task requirements
- Strategic decisions are made about resource allocation
- Team dynamics evolve through experience

## 🧩 How This Directory Fits

### **Component Testing Focus**
This directory tests individual Tier 1 components:
- `SwarmStateMachine` coordination logic
- MOISE+ organizational structures
- MCP tool integrations
- Agent collaboration patterns

### **Reused by Higher-Level Testing**
These fixtures are composed by other directories:

```typescript
// emergent-capabilities reuses tier1 fixtures
const learningSwarm = {
    baseConfig: securitySwarmFixture.config,  // From this directory
    learning: { ... }                         // Adds emergence testing
};

// integration-scenarios composes tier1 with other tiers
const healthcareScenario = {
    tier1: complianceSwarmFixture,  // From this directory
    tier2: medicalRoutineFixture,   // From tier2-process/
    tier3: secureExecutionContext   // From tier3-execution/
};
```

## 📁 Directory Structure

```
tier1-coordination/
├── swarms/                    # Multi-agent swarm configurations
│   ├── customer-support/      # Domain-specific swarm examples
│   ├── security/             
│   ├── financial/            
│   └── swarmFixtures.ts      # Base swarm patterns and utilities
├── moise-organizations/       # MOISE+ organizational structures
│   ├── hierarchical/         # Top-down command structures
│   ├── flat/                 # Peer-to-peer organizations
│   ├── matrix/               # Cross-functional teams
│   └── organizationFixtures.ts
└── coordination-tools/        # MCP tools for agent coordination
    ├── blackboard/           # Shared knowledge patterns
    ├── messaging/            # Inter-agent communication
    └── resource-management/  # Resource allocation strategies
```

## 🐝 Swarms

Swarms are dynamic collections of agents that coordinate through natural language to achieve complex goals.

### Swarm Configuration Structure
```typescript
const swarmFixture: SwarmFixture = {
    config: {
        __version: "1.0.0",
        id: testIdGenerator.next("CHAT"),
        
        // High-level swarm goal
        swarmTask: "Provide 24/7 customer support across all channels",
        
        // Decomposed subtasks
        swarmSubTasks: [
            { id: "1", task: "Monitor incoming requests", status: "pending" },
            { id: "2", task: "Classify and route issues", status: "pending" },
            { id: "3", task: "Resolve or escalate", status: "pending" }
        ],
        
        // Agent configuration
        botSettings: {
            occupation: "support_coordinator",
            persona: {
                tone: "helpful",
                verbosity: "balanced",
                traits: ["empathetic", "solution-oriented"]
            }
        },
        
        // Coordination settings
        eventSubscriptions: {
            "customer.request.*": true,
            "agent.available": true,
            "escalation.needed": true
        },
        
        // Shared state
        blackboard: [
            { key: "active_issues", value: {}, timestamp: Date.now() },
            { key: "agent_expertise", value: {}, timestamp: Date.now() }
        ]
    },
    
    emergence: {
        capabilities: [
            "load_balancing",      // Distribute work efficiently
            "skill_matching",      // Route to best agent
            "collective_learning"  // Share solutions
        ],
        evolutionPath: "reactive → proactive → predictive"
    },
    
    integration: {
        tier: "tier1",
        producedEvents: [
            "tier1.swarm.task_assigned",
            "tier1.swarm.knowledge_shared"
        ],
        consumedEvents: [
            "tier2.routine.completed",
            "tier3.execution.metrics"
        ]
    }
};
```

### Swarm Patterns

#### Hierarchical Swarm
```typescript
swarmMetadata: {
    formation: "hierarchical",
    coordinationPattern: "delegation",
    expectedAgentCount: 10,
    minViableAgents: 3,
    roles: [
        { role: "coordinator", count: 1 },
        { role: "specialist", count: 6 },
        { role: "trainee", count: 3 }
    ]
}
```

#### Consensus Swarm
```typescript
swarmMetadata: {
    formation: "flat",
    coordinationPattern: "consensus",
    votingThreshold: 0.7,
    deliberationTimeout: 30000  // 30 seconds
}
```

#### Dynamic Swarm
```typescript
swarmMetadata: {
    formation: "dynamic",
    coordinationPattern: "emergence",
    adaptationTriggers: [
        "workload > threshold",
        "new_domain_detected",
        "performance < target"
    ]
}
```

## 🏢 MOISE+ Organizations

MOISE+ (Model of Organization for multI-agent SystEms) provides formal organizational structures:

### Organization Structure
```typescript
const moiseOrganization: MOISEOrganization = {
    id: "customer_support_org",
    
    // Structural specification
    structure: {
        roles: [
            {
                id: "support_lead",
                name: "Support Team Lead",
                capabilities: ["escalation", "training", "reporting"],
                cardinality: { min: 1, max: 1 }
            },
            {
                id: "support_agent",
                name: "Support Agent",
                capabilities: ["issue_resolution", "customer_comm"],
                cardinality: { min: 3, max: 20 }
            }
        ],
        
        groups: [
            {
                id: "tier1_support",
                name: "First Line Support",
                roles: ["support_agent"],
                subgroups: ["product", "billing", "technical"]
            }
        ],
        
        links: [
            {
                type: "authority",
                from: "support_lead",
                to: "support_agent"
            }
        ]
    },
    
    // Functional specification
    functional: {
        missions: [
            {
                id: "resolve_customer_issues",
                name: "Resolve Customer Issues",
                goals: ["identify_issue", "provide_solution", "ensure_satisfaction"],
                minAgents: 1,
                maxAgents: 3
            }
        ],
        
        goals: [
            {
                id: "identify_issue",
                name: "Identify Customer Issue",
                description: "Understand what the customer needs",
                type: "achievement"
            }
        ]
    },
    
    // Normative specification
    normative: {
        norms: [
            {
                id: "response_time_norm",
                type: "obligation",
                role: "support_agent",
                mission: "resolve_customer_issues",
                deadline: 300000  // 5 minutes
            }
        ]
    }
};
```

### Organizational Patterns

#### Role Evolution
```typescript
roleEvolution: {
    trigger: "performance_metrics",
    progression: [
        { from: "trainee", to: "junior_agent", criteria: "accuracy > 0.8" },
        { from: "junior_agent", to: "senior_agent", criteria: "resolved > 100" },
        { from: "senior_agent", to: "team_lead", criteria: "mentored > 5" }
    ]
}
```

## 🛠️ Coordination Tools

MCP (Model Context Protocol) tools enable agent coordination:

### Available Tools
```typescript
coordinationTools: {
    // Swarm state management
    "update_swarm_shared_state": {
        description: "Update shared blackboard",
        parameters: ["key", "value", "timestamp"]
    },
    
    // Resource allocation
    "resource_manage": {
        description: "Allocate resources to agents",
        parameters: ["resource_type", "agent_id", "amount"]
    },
    
    // Task execution
    "run_routine": {
        description: "Execute Tier 2 routines",
        parameters: ["routine_id", "inputs", "strategy"]
    },
    
    // Communication
    "send_message": {
        description: "Send messages between agents",
        parameters: ["to", "content", "priority"]
    }
}
```

### Coordination Patterns

#### Blackboard Pattern
```typescript
const blackboardPattern = {
    sharedState: {
        "problem_space": {},
        "partial_solutions": [],
        "agent_contributions": {},
        "consensus_state": null
    },
    
    updateRules: [
        "atomic_updates_only",
        "conflict_resolution_by_timestamp",
        "periodic_garbage_collection"
    ]
};
```

#### Task Auction Pattern
```typescript
const taskAuctionPattern = {
    auctioneer: "coordinator_agent",
    bidding: {
        factors: ["expertise", "availability", "past_performance"],
        strategy: "sealed_bid",
        timeout: 5000
    },
    assignment: "highest_composite_score"
};
```

## 🧪 Testing Tier 1 Fixtures

### Swarm Formation Tests
```typescript
describe("Swarm Formation", () => {
    it("should form swarm with minimum viable agents", async () => {
        const swarm = await createSwarm(securitySwarmFixture);
        
        expect(swarm.agents.length).toBeGreaterThanOrEqual(
            swarm.metadata.minViableAgents
        );
        expect(swarm.state).toBe("operational");
    });
    
    it("should adapt formation based on workload", async () => {
        const swarm = await createDynamicSwarm();
        await simulateHighWorkload(swarm);
        
        expect(swarm.formation).toBe("expanded");
        expect(swarm.agents.length).toBeGreaterThan(initialCount);
    });
});
```

### Coordination Tests
```typescript
describe("Agent Coordination", () => {
    it("should achieve consensus on decisions", async () => {
        const decision = await swarm.deliberate({
            topic: "response_strategy",
            options: ["apologize", "offer_refund", "escalate"],
            timeout: 10000
        });
        
        expect(decision.consensus).toBeGreaterThan(0.7);
        expect(decision.selectedOption).toBeDefined();
    });
});
```

## 🎯 Emergent Behaviors

### Expected Emergence at Tier 1

#### Self-Organization
```typescript
emergence: {
    capabilities: [
        "dynamic_role_assignment",
        "adaptive_team_structure",
        "emergent_leadership"
    ],
    indicators: {
        roleFlexibility: "> 0.6",
        structuralAdaptation: "< 1 hour",
        leadershipEmergence: "natural"
    }
}
```

#### Collective Intelligence
```typescript
collectiveIntelligence: {
    mechanisms: [
        "knowledge_aggregation",
        "distributed_problem_solving",
        "swarm_learning"
    ],
    metrics: {
        solutionQuality: "better_than_individual",
        convergenceTime: "< 5 minutes",
        innovationRate: "0.1 novel_solutions/day"
    }
}
```

## 💡 Best Practices

### DO:
- ✅ Use natural language for coordination logic
- ✅ Define clear swarm goals and subtasks
- ✅ Include emergence capabilities
- ✅ Test swarm dynamics over time
- ✅ Allow for role flexibility

### DON'T:
- ❌ Hard-code coordination strategies
- ❌ Create rigid hierarchies
- ❌ Ignore minority agent opinions
- ❌ Skip consensus mechanisms
- ❌ Assume perfect communication

## 🔗 Integration Points

### To Tier 2 (Process)
```typescript
tier1ToTier2: {
    commands: [
        "execute_routine",
        "modify_workflow",
        "optimize_process"
    ],
    feedback: [
        "routine_performance",
        "bottleneck_detected",
        "optimization_opportunity"
    ]
}
```

### From Tier 3 (Execution)
```typescript
tier3ToTier1: {
    metrics: [
        "execution_time",
        "resource_usage",
        "error_rates"
    ],
    alerts: [
        "anomaly_detected",
        "threshold_exceeded",
        "pattern_recognized"
    ]
}
```

## 📊 Performance Benchmarks

### Coordination Efficiency
```typescript
benchmarks: {
    swarmFormationTime: "< 1 second",
    consensusAchievement: "< 10 seconds",
    taskDistribution: "< 100ms",
    knowledgeSharing: "< 500ms",
    adaptationLatency: "< 5 seconds"
}
```

## 🚀 Advanced Patterns

### Multi-Swarm Coordination
```typescript
multiSwarmCoordination: {
    swarms: ["security", "compliance", "operations"],
    coordination: "federated",
    sharedGoals: ["system_security"],
    conflictResolution: "priority_based"
}
```

### Evolutionary Swarms
```typescript
evolutionarySwarm: {
    generationSize: 10,
    selectionCriteria: "performance_based",
    mutation: "strategy_variation",
    crossover: "knowledge_transfer",
    generations: "continuous"
}
```

## 📚 References

- [Swarm Intelligence Patterns](/docs/architecture/execution/tier1-coordination/swarm-patterns.md)
- [MOISE+ Implementation](/docs/architecture/execution/tier1-coordination/moise.md)
- [MCP Tools Documentation](/docs/architecture/execution/tier1-coordination/mcp-tools.md)