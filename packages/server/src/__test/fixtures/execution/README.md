# 🧪 Execution Architecture Test Fixtures

This directory contains comprehensive test fixtures for Vrooli's three-tier AI execution system, demonstrating emergent capabilities, agent collaboration, and self-improving intelligence.

## 📚 Architecture Documentation References

Before exploring these fixtures, familiarize yourself with the core architecture:

- **[Main Execution Architecture](../../../../../../docs/architecture/execution/README.md)** - Vision and overview
- **[Architecture Overview](../../../../../../docs/architecture/execution/_ARCHITECTURE_OVERVIEW.md)** - Three-tier quick reference
- **[Core Technologies](../../../../../../docs/architecture/execution/core-technologies.md)** - Foundational concepts
- **[Quick Start Guide](../../../../../../docs/architecture/execution/quick-start-guide.md)** - 15-minute hands-on introduction

## 🏗️ Directory Structure

✅ **Structure Updated** - The fixtures have been reorganized to align with the three-tier architecture:

```
execution/
├── tier1-coordination/              # Tier 1: Coordination Intelligence
│   ├── swarms/                     # Dynamic swarm configurations
│   │   ├── customer-support/       # Domain-specific swarm examples
│   │   │   ├── swarm-config.ts     # Swarm configuration
│   │   │   ├── agent-roles.ts      # Role definitions
│   │   │   └── coordination-patterns.ts
│   │   ├── security-response/
│   │   ├── healthcare-compliance/
│   │   └── financial-trading/
│   ├── moise-organizations/        # MOISE+ organizational structures
│   │   ├── healthcare-org.ts
│   │   ├── financial-org.ts
│   │   └── research-org.ts
│   └── coordination-tools/         # MCP tools for natural coordination
│       ├── shared-state.ts
│       ├── resource-management.ts
│       └── message-passing.ts
│
├── tier2-process/                  # Tier 2: Process Intelligence
│   ├── routines/                   # Versioned workflow definitions
│   │   ├── by-evolution-stage/    # Show progression through strategies
│   │   │   ├── conversational/    # Novel problem-solving routines
│   │   │   ├── reasoning/         # Pattern-based routines
│   │   │   ├── deterministic/     # Optimized automated routines
│   │   │   └── routing/           # Multi-routine coordinators
│   │   └── by-domain/             # Current organization (secondary view)
│   ├── navigators/                # Format-specific translators
│   │   ├── native-vrooli.ts
│   │   ├── bpmn.ts
│   │   └── custom-formats.ts
│   └── run-states/                # RunStateMachine examples
│       ├── sequential.ts
│       ├── parallel.ts
│       └── conditional.ts
│
├── tier3-execution/               # Tier 3: Execution Intelligence
│   ├── strategies/                # Execution strategy examples
│   │   ├── conversational.ts
│   │   ├── reasoning.ts
│   │   ├── deterministic.ts
│   │   └── routing.ts
│   ├── unified-executor/          # UnifiedExecutor configurations
│   │   ├── tool-orchestration.ts
│   │   ├── resource-management.ts
│   │   └── safety-enforcement.ts
│   └── context-management/        # Execution context fixtures
│       ├── run-context.ts
│       ├── swarm-context.ts
│       └── team-context.ts
│
├── emergent-capabilities/         # Cross-tier emergent behaviors
│   ├── agent-types/              # Specialized agent configurations
│   │   ├── security-agents/      # Domain threat detection
│   │   ├── quality-agents/       # Output validation
│   │   ├── optimization-agents/  # Performance enhancement
│   │   └── monitoring-agents/    # Intelligent observability
│   ├── evolution-examples/       # Routine evolution scenarios
│   │   ├── customer-support-evolution.ts
│   │   ├── security-scan-evolution.ts
│   │   └── data-processing-evolution.ts
│   └── self-improvement/         # Recursive capability growth
│       ├── pattern-recognition.ts
│       ├── strategy-proposals.ts
│       └── collaborative-review.ts
│
└── integration-scenarios/        # Complete system examples
    ├── healthcare-compliance/    # Full three-tier integration
    ├── financial-trading/
    └── customer-service/
```

## 🎯 Core Fixture Categories

### 1. **Agent Fixtures** (`emergentAgentFixtures.ts`)

Demonstrates specialized agent types that provide emergent capabilities through event-driven intelligence:

- **Security Agents** - Domain-specific threat detection, HIPAA compliance, API security, GDPR auditing
- **Quality Agents** - Output validation, bias detection, accuracy monitoring
- **Optimization Agents** - Performance enhancement, cost reduction, resource optimization
- **Monitoring Agents** - Intelligent observability, predictive analytics, anomaly detection

These agents represent the key innovation: capabilities emerge from specialized agents analyzing events, not from built-in features.

### 2. **Organization Fixtures** (`organizationFixtures.ts`)

Implements MOISE+ organizational modeling for structured agent collaboration:

- Healthcare compliance organizations with multi-role hierarchies
- Financial trading teams with risk management structures
- Research laboratory structures with collaborative workflows
- Complete specifications including:
  - **Structural**: Roles, groups, and communication links
  - **Functional**: Goals, missions, and schemas
  - **Normative**: Obligations, permissions, and prohibitions

### 3. **Swarm Fixtures** (`swarmFixtures.ts`)

Demonstrates Tier 1 coordination through dynamic swarm formations:

- **Customer Support Swarms** - Multi-agent teams handling complex queries
- **Security Response Teams** - Rapid threat detection and mitigation
- **Healthcare Diagnostic Collaborations** - Coordinated medical analysis
- **Financial Risk Assessment Consortiums** - Collaborative trading decisions

Key features:
- Natural language coordination through MCP tools
- Dynamic agent recruitment based on capabilities
- Shared state management via blackboard pattern
- Consensus-based decision making

### 4. **Routine Fixtures** (`routines/`)

Organized by domain, demonstrating routine evolution through execution strategies:

#### Security Routines (4)
- `HIPAA_COMPLIANCE_CHECK` - PHI detection in AI outputs (Deterministic)
- `API_SECURITY_SCAN` - Security threat analysis (Reasoning)
- `GDPR_DATA_AUDIT` - Privacy compliance auditing (Reasoning)
- `TRADING_PATTERN_ANALYSIS` - Financial fraud detection (Conversational)

#### Medical Routines (1)
- `MEDICAL_DIAGNOSIS_VALIDATION` - Clinical guideline validation (Reasoning)

#### Performance Routines (3)
- `PERFORMANCE_BOTTLENECK_DETECTION` - Execution optimization (Reasoning)
- `COST_ANALYSIS` - Operational cost tracking (Deterministic)
- `OUTPUT_QUALITY_ASSESSMENT` - Quality and bias evaluation (Reasoning)

#### System Routines (2)
- `SYSTEM_FAILURE_ANALYSIS` - Failure pattern identification (Conversational)
- `SYSTEM_HEALTH_CHECK` - Comprehensive monitoring (Deterministic)

#### BPMN Workflows (3)
- `COMPREHENSIVE_SECURITY_AUDIT` - Parallel security analysis (Routing)
- `MEDICAL_TREATMENT_VALIDATION` - Sequential compliance validation (Routing)
- `RESILIENCE_OPTIMIZATION_WORKFLOW` - Combined analysis flows (Routing)

#### Bootstrap Routines (5+)
- API integration routines for external service connections
- Data transformation routines for format conversions
- Document processing routines for content extraction

## 🏭 Tier-Specific Factory Patterns

### Tier 1 Factory - Coordination Intelligence

```typescript
export class Tier1Factory {
    // Create swarms with natural language coordination
    static createSwarm(config: SwarmConfig): Swarm {
        return {
            id: generatePK(),
            config: {
                maxAgents: config.maxAgents || 10,
                consensusThreshold: config.consensusThreshold || 0.7,
                coordinationTools: [
                    "update_swarm_shared_state",
                    "resource_manage",
                    "send_message"
                ]
            },
            // ... swarm configuration
        };
    }
    
    // Create MOISE+ organizations
    static createMOISEOrg(spec: MOISEPlusSpecification): Organization {
        return {
            structural: spec.roles,
            functional: spec.goals,
            normative: spec.norms
        };
    }
}
```

### Tier 2 Factory - Process Intelligence

```typescript
export class Tier2Factory {
    // Create routines at different evolution stages
    static createRoutine(stage: ExecutionStrategy): Routine {
        const strategies = {
            conversational: { avgTime: "45s", cost: "$0.12" },
            reasoning: { avgTime: "15s", cost: "$0.08" },
            deterministic: { avgTime: "2s", cost: "$0.02" },
            routing: { avgTime: "varies", cost: "varies" }
        };
        
        return {
            executionStrategy: stage,
            performance: strategies[stage],
            // ... routine configuration
        };
    }
    
    // Create run state machines
    static createRunState(type: RunType): RunStateMachine {
        return new RunStateMachine({
            type,
            navigatorPlugin: type === "bpmn" ? "BPMNNavigator" : "NativeNavigator"
        });
    }
}
```

### Tier 3 Factory - Execution Intelligence

```typescript
export class Tier3Factory {
    // Create unified executors with strategy selection
    static createUnifiedExecutor(strategy: ExecutionStrategy): UnifiedExecutor {
        return {
            strategy,
            toolOrchestration: strategy === "deterministic" ? "optimized" : "flexible",
            resourceManagement: "credit-based",
            safetyEnforcement: "synchronous"
        };
    }
    
    // Create execution contexts
    static createContext(type: ContextType): ExecutionContext {
        return {
            type,
            sharedState: {},
            resourceLimits: { credits: 1000, timeout: 30000 }
        };
    }
}
```

## 📈 Advanced Examples

### Event-Driven Intelligence (`eventDrivenAgentExamples.ts`)

Shows how agents process events to deliver emergent capabilities:

```typescript
// Example: Security agent evolving through pattern recognition
const securityAgent = {
    observes: ["api_calls", "data_access", "authentication"],
    patterns: {
        suspicious: ["rapid_requests", "unusual_access_patterns"],
        learned: ["new_threat_signatures", "false_positive_patterns"]
    },
    evolution: {
        v1: "alert_on_threshold",
        v2: "pattern_matching",
        v3: "predictive_detection"
    }
};
```

### Routine Evolution (`evolutionFixtures.ts`)

Demonstrates how routines evolve through execution strategies:

```typescript
// Example: Customer support routine evolution
export const customerSupportEvolution = {
    v1_conversational: {
        strategy: "conversational",
        performance: { time: "45s", cost: "$0.12", accuracy: "92%" }
    },
    v2_reasoning: {
        strategy: "reasoning",
        performance: { time: "15s", cost: "$0.08", accuracy: "95%" },
        improvements: ["template_responses", "pattern_recognition"]
    },
    v3_deterministic: {
        strategy: "deterministic",
        performance: { time: "2s", cost: "$0.02", accuracy: "99%" },
        improvements: ["direct_lookups", "cached_responses"]
    }
};
```

## 🚀 Usage Examples

### Testing Three-Tier Integration

```typescript
// Tier 1: Form a swarm for customer support
const swarm = Tier1Factory.createSwarm({
    maxAgents: 5,
    consensusThreshold: 0.8,
    goal: "Handle customer inquiries efficiently"
});

// Tier 2: Select appropriate routine based on query complexity
const routine = Tier2Factory.createRoutine(
    queryComplexity === "simple" ? "deterministic" : "reasoning"
);

// Tier 3: Execute with appropriate strategy
const executor = Tier3Factory.createUnifiedExecutor(routine.executionStrategy);
const result = await executor.execute(routine, swarm.context);
```

### Testing Routine Evolution

```typescript
import { customerSupportEvolution } from './evolutionFixtures';

// Track performance improvements through evolution
const v1Performance = customerSupportEvolution.v1_conversational.performance;
const v3Performance = customerSupportEvolution.v3_deterministic.performance;

console.log(`Speed improvement: ${v1Performance.time} → ${v3Performance.time}`);
console.log(`Cost reduction: ${v1Performance.cost} → ${v3Performance.cost}`);
console.log(`Accuracy gain: ${v1Performance.accuracy} → ${v3Performance.accuracy}`);
```

### Testing Emergent Capabilities

```typescript
// Deploy specialized agents that provide emergent capabilities
const securitySwarm = {
    agents: [
        { type: "threat_detector", observes: ["api_calls", "auth_attempts"] },
        { type: "compliance_monitor", validates: ["hipaa", "gdpr"] },
        { type: "pattern_learner", evolves: ["threat_signatures"] }
    ],
    emergentCapability: "Proactive security threat prevention"
};

// Capabilities emerge from agent collaboration, not built-in features
```

## 📊 Test Coverage

- **Unit Tests**: Individual tier components (swarms, routines, executors)
- **Integration Tests**: Cross-tier communication and coordination
- **Evolution Tests**: Routine progression through execution strategies
- **Emergence Tests**: Capability development through event-driven agents
- **Performance Tests**: Strategy optimization and cost reduction

## 🛡️ Safety Architecture

Multi-layered safety approach aligned with the three-tier system:

### Synchronous Safety (<10ms)
- **Input Validation**: Block malicious inputs at entry
- **Resource Limits**: Hard caps on credits and execution time
- **Emergency Stop**: Immediate termination capabilities

### Asynchronous Safety (Event-Driven)
- **Compliance Agents**: Monitor for regulatory violations
- **Quality Agents**: Detect bias and accuracy issues
- **Security Agents**: Identify threat patterns
- **Audit Agents**: Maintain decision trails

### Team-Specific Safety
- **Domain Expertise**: Healthcare, financial, legal compliance
- **Adaptive Learning**: Improve safety patterns over time
- **Collaborative Review**: Multi-agent validation of critical decisions

## 🔧 Contributing

When adding new fixtures:

1. **Understand the Tiers**: Know which tier your fixture belongs to
2. **Show Evolution**: Demonstrate progression through execution strategies
3. **Use Factories**: Leverage tier-specific factory patterns
4. **Document Emergence**: Explain how capabilities emerge from collaboration
5. **Include Metrics**: Show performance, cost, and quality improvements
6. **Test Integration**: Ensure fixtures work across tier boundaries

### Fixture Checklist

- [ ] Correct tier placement (Tier 1, 2, or 3)
- [ ] Execution strategy specified (if routine)
- [ ] Evolution path documented (if applicable)
- [ ] Integration with other tiers shown
- [ ] Performance metrics included
- [ ] Safety considerations addressed
- [ ] TypeScript types properly defined

---

*Remember: The three-tier architecture enables compound intelligence through emergent agent collaboration, not through built-in features. Every fixture should demonstrate this principle.*