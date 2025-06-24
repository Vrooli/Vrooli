# Tier 1 Coordination Fixtures

This directory contains test fixtures for Tier 1 (Coordination Intelligence) of the execution architecture. These fixtures enable comprehensive testing of emergent swarm capabilities, organizational dynamics, and strategic coordination using the **enhanced factory-based approach** integrated with shared package validation patterns.

## Overview

Tier 1 coordinates high-level strategic planning through:
- **Swarm Coordination**: Multi-agent collaboration patterns validated through `ChatConfig`
- **MOISE+ Organizations**: Structural, functional, and normative specifications
- **Resource Management**: Strategic allocation across the system
- **Emergent Capabilities**: Behaviors that arise from agent interactions with measurable metrics
- **Event Contracts**: Guaranteed cross-tier communication patterns

## Enhanced Factory Architecture

The tier 1 fixtures now use production-grade factories with comprehensive validation:

```typescript
import { SwarmFixtureFactory } from "../executionFactories.js";

// Pre-instantiated factory with built-in validation
const swarmFactory = new SwarmFixtureFactory();

// Create fixtures with validation
const supportSwarm = swarmFactory.createVariant("customerSupport");
const securitySwarm = swarmFactory.createVariant("securityResponse");

// Evolution path testing
const evolutionPath = swarmFactory.createEvolutionPath(3);
```

## Fixture Structure

```
tier1-coordination/
├── swarms/                     # Swarm configuration fixtures
│   ├── swarmFixtures.ts       # Legacy fixtures (being migrated)
│   ├── customer-support/      # Domain-specific swarm patterns
│   ├── security-response/     # Security swarm configurations
│   ├── financial-trading/     # Financial swarm patterns
│   └── agentFixtures.ts       # Agent-specific fixtures
├── moise-organizations/        # MOISE+ organization fixtures
│   ├── moiseTypes.ts          # Type definitions
│   └── organizationFixtures.ts # Organization patterns
├── coordination-tools/         # Agent collaboration patterns
│   ├── agentCollaboration.ts  # Collaboration examples
│   └── resourceManagementExamples.ts # Resource allocation
└── index.ts                    # Exports and utilities
```

## Key Concepts

### 1. Factory-Based Creation with Config Validation
All fixtures use factories that validate against actual `ChatConfig` classes:
```typescript
// NEW: Direct config class validation
const swarmFactory = new SwarmFixtureFactory();
const swarmFixture = swarmFactory.createComplete({
    config: {
        goal: "Provide comprehensive customer support",
        preferredModel: "gpt-4"
    },
    emergence: {
        capabilities: ["task_decomposition", "resource_optimization"],
        eventPatterns: ["swarm.task.completed", "swarm.resource.allocated"]
    }
});

// Automatic validation through factory
const result = await swarmFactory.validateFixture(swarmFixture);
```

### 2. Measurable Emergent Capabilities
Fixtures define measurable capabilities with concrete metrics:
```typescript
// NEW: Measurable capabilities with baseline/target metrics
const swarmWithMeasurableCapabilities = swarmFactory.createComplete({
    emergence: {
        capabilities: ["pattern_recognition", "strategy_formation"],
        measurableCapabilities: [
            createMeasurableCapability(
                "pattern_recognition",
                "accuracy",
                0.6,  // baseline
                0.9,  // target
                "percentage",
                "Identifies recurring task structures"
            ),
            createMeasurableCapability(
                "strategy_formation",
                "execution_time",
                5000, // baseline ms
                2000, // target ms
                "milliseconds",
                "Develops optimal execution strategies"
            )
        ],
        evolutionPath: "reactive → proactive → autonomous"
    }
});
```

### 3. Event Contracts for Guaranteed Communication
All tier interactions use event contracts with delivery guarantees:
```typescript
// NEW: Event contracts with explicit guarantees
const swarmWithEventContracts = swarmFactory.createComplete({
    integration: {
        tier: "tier1",
        producedEvents: ["tier1.swarm.initialized", "tier1.task.assigned"],
        consumedEvents: ["tier2.task.completed", "tier3.execution.result"],
        eventContracts: [
            createEventContract(
                "tier1.task.assigned",
                "tier1.customerSupportSwarm",
                ["tier2.inquiryRoutine", "tier2.escalationRoutine"],
                { taskId: "string", priority: "number", customerId: "string" },
                "exactly-once"  // Guaranteed delivery
            )
        ]
    }
});
```

### 4. Comprehensive Validation with Round-Trip Testing
Fixtures use enhanced validation with config class round-trip testing:
```typescript
// NEW: Comprehensive validation through factory
const swarmFactory = new SwarmFixtureFactory();

// Validates config through ChatConfig class with round-trip testing
const validationResult = await swarmFactory.validateFixture(swarmFixture);

// Run comprehensive tests (82% code reduction)
runComprehensiveExecutionTests(
    swarmFixture,
    "chat",  // Config type for ChatConfig validation
    "customer-support-swarm"
);

// This automatically validates:
// - Config round-trip consistency
// - Emergence pattern validity
// - Event flow consistency
// - Cross-tier dependencies
// - Evolution pathways
```

## Usage

### Creating Fixtures with Enhanced Factory
```typescript
import { SwarmFixtureFactory, swarmFactory } from "../executionFactories.js";

// Method 1: Use pre-instantiated factory
const supportSwarm = swarmFactory.createVariant("customerSupport");

// Method 2: Create custom swarm with overrides
const customSwarm = swarmFactory.createComplete({
    config: {
        goal: "Monitor system security with AI agents"
    },
    emergence: {
        capabilities: ["threat_detection", "automated_response"],
        evolutionPath: "reactive → proactive → predictive"
    }
});

// Method 3: Create evolution path for testing improvement
const evolutionStages = swarmFactory.createEvolutionPath(3); // 3 stages
// Stage 0: reactive (flat formation, 2 agents)
// Stage 1: proactive (hierarchical formation, 3 agents)  
// Stage 2: predictive (dynamic formation, 4 agents)

// All fixtures are automatically validated during creation
```

### Running Tests
```typescript
// Automatic test generation (82% code reduction)
runTier1ComprehensiveTests(swarmFixture, 'Learning Swarm');

// This generates tests for:
// - Config validation
// - Emergence patterns
// - Event flows
// - Integration points
// - Evolution paths
```

### Testing Emergence
```typescript
// Test that capabilities emerge from configuration
it('should develop pattern recognition from event analysis', async () => {
    const swarm = createSwarmWithEventAnalysis();
    const result = await simulateSwarmExecution(swarm);
    
    expect(result.emergentCapabilities).toContain('pattern_recognition');
    expect(result.metrics.patternAccuracy).toBeGreaterThan(0.8);
});
```

## Integration with Shared Fixtures

Tier 1 fixtures build on shared package patterns:

- **Chat Configs** → Swarm definitions
- **Bot Configs** → Agent specifications  
- **Routine Configs** → Strategic planning templates
- **Run Configs** → Execution tracking

Example:
```typescript
import { chatConfig } from '@vrooli/shared/__test/fixtures/config';

const swarmFixture = SwarmFixtureFactory.fromChatConfig(
    chatConfig.fixtures.complete,
    { emergentCapabilities: ['distributed_reasoning'] }
);
```

## Validation Approach

### 1. Schema Validation
All fixtures validate against actual runtime schemas:
```typescript
validateAgainstSchema(ChatConfigObject, fixture.config);
```

### 2. Round-Trip Consistency
Configs maintain consistency through serialization:
```typescript
const exported = config.export();
const reimported = new ChatConfigObject(exported);
expect(reimported.export()).toEqual(exported);
```

### 3. Emergence Validation
Validate emergence definitions:
```typescript
validateEmergencePattern({
    capabilities: fixture.emergence.capabilities,
    eventPatterns: fixture.emergence.eventPatterns,
    evolutionPath: fixture.emergence.evolutionPath
});
```

### 4. Integration Validation
Ensure proper event flows:
```typescript
validateEventFlow({
    produced: fixture.integration.producedEvents,
    consumed: fixture.integration.consumedEvents
});
```

## Best Practices

1. **Test Emergence, Not Implementation**: Focus on what capabilities emerge
2. **Use Real Schemas**: Always validate against actual system schemas
3. **Event-Only Communication**: No direct function calls between tiers
4. **Evolution Testing**: Include fixtures for each evolution stage
5. **Comprehensive Scenarios**: Test complete flows, not isolated components

## Common Patterns

### Basic Swarm
```typescript
fixtures.minimal: {
    config: minimalChatConfig,
    emergence: {
        capabilities: ["task_coordination"],
        criteria: { "task_coordination": "Assigns tasks to agents" }
    }
}
```

### Learning Swarm
```typescript
fixtures.variants.learning: {
    config: learningEnabledChatConfig,
    emergence: {
        capabilities: ["pattern_learning", "strategy_optimization"],
        evolutionPath: "reactive -> adaptive -> autonomous"
    }
}
```

### Hierarchical Organization
```typescript
organization.variants.hierarchical: {
    structural: {
        roles: ["coordinator", "specialist", "executor"],
        groups: [{ id: "strategic", roles: ["coordinator"] }]
    },
    functional: {
        goals: ["optimize_performance", "maintain_quality"],
        plans: ["top_down_delegation", "bottom_up_feedback"]
    }
}
```

## Testing Scenarios

### 1. Basic Execution
Tests fundamental swarm coordination without advanced features.

### 2. Adaptive Organization
Tests dynamic restructuring based on performance metrics.

### 3. Emergent Learning
Tests how swarms develop new strategies through experience.

### 4. Cross-Tier Integration
Tests complete flows from tier 1 coordination through tier 3 execution.

## Troubleshooting

### Common Issues

1. **Type Mismatches**: Ensure fixtures match both TypeScript types and runtime schemas
2. **Invalid Events**: Check event naming conventions (tier.component.action)
3. **Missing Capabilities**: Verify emergence definitions include all required capabilities
4. **Integration Failures**: Confirm event producers/consumers are properly matched

### Validation Errors

Run comprehensive validation to identify issues:
```typescript
pnpm test -- tier1-coordination/validation
```

This will check:
- Schema compliance
- Type safety
- Event flow consistency
- Emergence pattern validity
- Integration completeness