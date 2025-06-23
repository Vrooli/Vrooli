# Execution Architecture Test Fixtures

This directory contains comprehensive test fixtures for Vrooli's three-tier execution architecture. The fixtures are designed to validate emergent capabilities, cross-tier integration, and the evolution of AI-driven systems.

## 🎯 Purpose

These fixtures serve multiple critical purposes:

1. **Validate Emergent Capabilities**: Test that capabilities emerge from configuration and agent intelligence, not hard-coded logic
2. **Ensure Type Safety**: Provide compile-time guarantees that fixtures match production schemas
3. **Test Integration**: Verify cross-tier communication and event flows
4. **Document Patterns**: Serve as living examples of execution architecture patterns
5. **Enable Evolution Testing**: Track how routines evolve from conversational to deterministic

## 🏗️ Architecture Overview

The fixtures are organized into two categories that serve different testing purposes:

### **Component Testing (Tier Directories)**
These test individual tiers and map directly to the codebase:
```
├── tier1-coordination/     # Tests /services/execution/tier1/ 
├── tier2-process/         # Tests /services/execution/tier2/
└── tier3-execution/       # Tests /services/execution/tier3/
```

### **System Testing (Cross-Cutting Directories)**  
These compose tier fixtures to test higher-level behaviors:
```
├── emergent-capabilities/ # Tests learned behaviors across all tiers
└── integration-scenarios/ # Tests complete end-to-end workflows
```

## 🧩 How Fixture Groups Relate

### **Hierarchical Testing Strategy**
The relationship is hierarchical - each level builds on the previous:

1. **Tier fixtures** = Test individual components (engines, transmissions, brakes)
2. **Emergent capabilities** = Test learned behaviors (how the car learns optimal shifting)  
3. **Integration scenarios** = Test real-world workflows (complete driving scenarios)

### **Fixture Composition Pattern**
Higher-level directories reuse and compose lower-level fixtures:

```typescript
// integration-scenarios reuses tier fixtures
const healthcareScenario: IntegrationScenario = {
    tier1: securitySwarmFixture,      // From tier1-coordination/
    tier2: medicalRoutineFixture,     // From tier2-process/ 
    tier3: hipaaExecutionContext,     // From tier3-execution/
    
    // Adds integration-specific testing
    expectedEvents: ["tier1.swarm.initialized", "tier2.routine.started"],
    emergentCapabilities: ["end_to_end_compliance", "audit_automation"]
};

// emergent-capabilities adds learning to base fixtures
const learningAgent: EmergentAgentFixture = {
    baseConfig: securitySwarmFixture.config,  // Reuses tier1 fixture
    learning: { patternRecognition: true },   // Adds emergence testing
    validation: "cross_tier_intelligence"     // Tests behavior spanning tiers
};
```

### **No Duplication, Smart Composition**
- ✅ Tier directories provide reusable "building blocks"
- ✅ Cross-cutting directories compose blocks for system testing
- ✅ Shared utilities ensure consistency across all groups
- ✅ No duplicate test code or configuration objects

## 📋 Fixture Structure

Each fixture follows a standardized structure that ensures consistency:

```typescript
interface ExecutionFixture<TConfig> {
    // Data-driven configuration (from @vrooli/shared configs)
    config: TConfig;
    
    // Expected emergent behaviors
    emergence: {
        capabilities: string[];        // What emerges
        eventPatterns?: string[];      // Triggering events
        evolutionPath?: string;        // How it improves
        emergenceConditions?: {...}    // When it emerges
    };
    
    // Integration with other components
    integration: {
        tier: "tier1" | "tier2" | "tier3" | "cross-tier";
        producedEvents?: string[];
        consumedEvents?: string[];
        sharedResources?: string[];
        crossTierDependencies?: {...};
    };
    
    // Validation and metadata
    validation?: ValidationDefinition;
    metadata?: TestMetadata;
}
```

## 🔍 Key Concepts

### Emergent Capabilities

Fixtures demonstrate how capabilities emerge from agent collaboration rather than being hard-coded:

- **Pattern Recognition**: Agents learn from execution patterns
- **Performance Optimization**: Systems improve through experience
- **Domain Expertise**: Specialized knowledge develops over time
- **Resilience**: Error recovery strategies evolve

### Configuration-Driven Design

All fixtures use configuration objects from `@vrooli/shared`:

- `ChatConfigObject` → Swarm configurations (Tier 1)
- `RoutineConfigObject` → Workflow definitions (Tier 2)  
- `RunConfigObject` → Execution contexts (Tier 3)
- `BotConfigObject` → Agent personalities

### Evolution Stages

Routines naturally progress through four stages:

1. **Conversational**: Human-like exploration and understanding
2. **Reasoning**: Structured approaches with chain-of-thought
3. **Deterministic**: Optimized automation for proven patterns
4. **Routing**: Multi-routine coordination for complex tasks

## 🛠️ Validation Approach

The fixtures use a multi-layered validation approach inspired by the shared package patterns:

### 1. Schema Validation

Fixtures are validated against production schemas:

```typescript
import { ChatConfig, RoutineConfig } from "@vrooli/shared";

// Configs are validated using the actual runtime validators
const swarmConfig = new ChatConfig({ config: fixture.config });
```

### 2. Emergence Validation

Tests verify that emergent capabilities are properly defined:

```typescript
runEmergenceValidationTests(fixture, "security-swarm");
// Validates: capabilities, event patterns, evolution paths
```

### 3. Integration Validation

Tests ensure proper cross-tier communication:

```typescript
runIntegrationTests(fixture, "healthcare-workflow");
// Validates: tier assignment, event flow, dependencies
```

### 4. Comprehensive Tests

Complete validation in one call:

```typescript
runComprehensiveExecutionTests(
    fixture,
    ChatConfig,
    "customer-support-swarm"
);
```

## 📂 Directory Structure

### Tier 1: Coordination Intelligence

```
tier1-coordination/
├── swarms/                 # Multi-agent swarm configurations
│   ├── customer-support/   # Domain-specific swarms
│   ├── security/          
│   └── swarmFixtures.ts    # Common swarm patterns
├── moise-organizations/    # Formal organization structures
└── coordination-tools/     # Agent collaboration patterns
```

### Tier 2: Process Intelligence

```
tier2-process/
├── routines/
│   ├── by-domain/         # Medical, security, system, etc.
│   └── by-evolution-stage/ # Conversational → deterministic
├── navigators/            # Workflow format adapters
└── run-states/            # State machine fixtures
```

### Tier 3: Execution Intelligence

```
tier3-execution/
├── strategies/            # Execution strategy patterns
├── context-management/    # Runtime contexts
└── unified-executor/      # Tool orchestration
```

### Emergent Capabilities

```
emergent-capabilities/
├── agent-types/          # Security, quality, monitoring agents
├── evolution-examples/   # How capabilities evolve
└── self-improvement/     # Learning and adaptation patterns
```

### Integration Scenarios

```
integration-scenarios/
├── healthcare-compliance/ # Complete healthcare workflow
├── customer-service/      # End-to-end support flow
└── financial-trading/     # Complex decision scenarios
```

## 🚀 Usage Examples

### Creating a New Swarm Fixture

```typescript
const securitySwarm: SwarmFixture = {
    config: {
        __version: "1.0.0",
        id: testIdGenerator.next("CHAT"),
        swarmTask: "Monitor and respond to security threats",
        swarmSubTasks: [
            { id: "1", task: "Detect anomalies", status: "pending" },
            { id: "2", task: "Analyze threats", status: "pending" }
        ],
        botSettings: {
            occupation: "security",
            persona: { tone: "professional", verbosity: "concise" }
        }
    },
    emergence: {
        capabilities: [
            "threat_detection",
            "pattern_recognition", 
            "automated_response"
        ],
        eventPatterns: ["security/*", "system/error"],
        evolutionPath: "reactive → proactive → predictive"
    },
    integration: {
        tier: "tier1",
        producedEvents: ["security.threat.detected", "security.response.initiated"],
        consumedEvents: ["tier3.execution.anomaly", "system.alert"],
        sharedResources: ["threat_database", "security_blackboard"]
    }
};
```

### Testing a Routine Evolution

```typescript
// Create fixtures for each evolution stage
const customerInquiryV1 = createConversationalRoutine();
const customerInquiryV2 = createReasoningRoutine();
const customerInquiryV3 = createDeterministicRoutine();

// Test the evolution path
const evolutionPath = [customerInquiryV1, customerInquiryV2, customerInquiryV3];
runEvolutionPathTests(evolutionPath, "customer-inquiry");
```

### Validating Integration Scenarios

```typescript
const healthcareScenario: IntegrationScenario = {
    tier1: complianceSwarmFixture,
    tier2: patientDataRoutineFixture,
    tier3: hipaaExecutionContext,
    expectedEvents: [
        "tier1.swarm.initialized",
        "tier2.routine.started",
        "tier3.compliance.checked",
        "tier1.task.completed"
    ],
    emergence: {
        capabilities: ["hipaa_compliance", "audit_trail", "data_privacy"]
    }
};

runIntegrationScenarioTests(healthcareScenario);
```

## 🧪 Running Tests

```bash
# Run all execution fixture tests
cd packages/server && pnpm test fixtures/execution

# Run specific tier tests
pnpm test tier1-coordination
pnpm test tier2-process
pnpm test tier3-execution

# Run validation tests only
pnpm test fixtures-validation

# Run with coverage
pnpm test-coverage fixtures/execution
```

## 📝 Best Practices

### DO:
- ✅ Use configuration objects from `@vrooli/shared`
- ✅ Define clear emergence capabilities
- ✅ Include event patterns for integration
- ✅ Provide evolution paths for improvements
- ✅ Test both success and failure scenarios
- ✅ Document complex fixtures with comments
- ✅ Use the testIdGenerator for consistent IDs

### DON'T:
- ❌ Hard-code business logic in fixtures
- ❌ Create fixtures without emergence definitions
- ❌ Skip integration metadata
- ❌ Use random or inconsistent IDs
- ❌ Mix concerns between tiers
- ❌ Implement actual execution logic

## 🔄 Evolution Testing

Fixtures support testing how capabilities evolve:

1. **Performance Metrics**: Track improvements across versions
2. **Error Rates**: Validate resilience improvements
3. **Cost Reduction**: Ensure efficiency gains
4. **Quality Scores**: Measure output improvements

Example evolution test:

```typescript
const v1Metrics = { avgDuration: 5000, errorRate: 0.15 };
const v2Metrics = { avgDuration: 3000, errorRate: 0.08 };
const v3Metrics = { avgDuration: 1000, errorRate: 0.02 };

expect(v3Metrics.avgDuration).toBeLessThan(v1Metrics.avgDuration * 0.25);
expect(v3Metrics.errorRate).toBeLessThan(v1Metrics.errorRate * 0.15);
```

## 🎯 Common Patterns

### Self-Improvement Pattern
```typescript
emergence: commonEmergencePatterns.selfImprovement
// Includes: pattern_recognition, performance_optimization, strategy_evolution
```

### Collaboration Pattern
```typescript
emergence: commonEmergencePatterns.collaboration
// Includes: agent_coordination, knowledge_sharing, task_delegation
```

### Domain Expertise Pattern
```typescript
emergence: commonEmergencePatterns.domainExpertise
// Includes: domain_knowledge, context_awareness, specialized_tools
```

### Resilience Pattern
```typescript
emergence: commonEmergencePatterns.resilience
// Includes: error_recovery, self_healing, adaptive_retry
```

## 🤝 Contributing

When adding new fixtures:

1. Follow the standard `ExecutionFixture` interface
2. Include comprehensive emergence definitions
3. Add integration metadata
4. Provide test coverage using the validation utilities
5. Document complex scenarios
6. Update relevant pattern libraries

## 📚 Related Documentation

- [Execution Architecture Overview](/docs/architecture/execution/README.md)
- [Testing Best Practices](/docs/testing/README.md)
- [Emergent Capabilities Guide](/docs/architecture/execution/emergent-capabilities/README.md)
- [Shared Validation Patterns](/packages/shared/src/validation/README.md)

## ⚠️ Important Notes

1. **Fixtures are Data, Not Code**: All behavior emerges from configuration and agent intelligence
2. **Version Everything**: Use `__version` fields for compatibility tracking
3. **Event-Driven**: All integration happens through events, not direct calls
4. **Test the Emergence**: Focus on validating emergent capabilities, not implementation
5. **Evolution is Key**: Always consider how a fixture can evolve to be better

Remember: The goal is to create a system that improves itself through use, not one that requires constant manual updates. These fixtures help ensure we're building toward that vision.