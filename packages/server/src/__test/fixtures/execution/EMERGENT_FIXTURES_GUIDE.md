# Emergent Fixtures Guide

This guide explains how to create test fixtures that demonstrate true emergent capabilities in Vrooli's execution architecture.

## Key Principle: Emergence, Not Hard-Coding

Capabilities must **emerge** from configuration and agent interaction, not be programmed directly.

### ❌ Wrong: Hard-Coded Capabilities
```typescript
// BAD: Capabilities are directly specified
const fixture = {
    config: {
        capabilities: ["threat_detection", "pattern_analysis"], // NO!
        algorithm: "neural_network", // NO!
        optimizationMethod: "gradient_descent", // NO!
    }
};
```

### ✅ Right: Emergent Capabilities
```typescript
// GOOD: Capabilities emerge from interaction
const fixture = {
    config: {
        goal: "Monitor system security", // Simple goal
        // No HOW - agents figure it out
    },
    emergence: {
        capabilities: ["threat_detection"], // What EMERGES
        emergenceConditions: {
            minAgents: 3, // Need collaboration
            minEvents: 20, // Need experience
        }
    }
};
```

## Using the Helpers

### 1. Creating Fixtures with Emergent Patterns

```typescript
import { 
    EMERGENCE_PATTERNS,
    createMeasurableEmergence 
} from "./emergentCapabilityHelpers.js";

const researchSwarm = swarmFactory.createComplete({
    config: {
        goal: "Research quantum computing", // Just the goal
    },
    emergence: createMeasurableEmergence(
        EMERGENCE_PATTERNS.collaborativeLearning,
        {
            capabilities: [
                "interdisciplinary_synthesis", // Emerges from diverse agents
                "hypothesis_generation", // Emerges from pattern correlation
            ]
        }
    )
});
```

### 2. Defining Evolution Triggers

```typescript
import { EVOLUTION_TRIGGERS } from "./emergentCapabilityHelpers.js";

const evolutionTriggers = [
    EVOLUTION_TRIGGERS.performanceThreshold("accuracy", 0.9),
    EVOLUTION_TRIGGERS.errorReduction(0.05),
    EVOLUTION_TRIGGERS.usagePatternShift(),
];
```

### 3. Creating Measurable Capabilities

```typescript
import { 
    createCollaborativeCapability,
    createCompoundGrowthCapability 
} from "./emergentCapabilityHelpers.js";

const capabilities = {
    // Emerges from multiple agents
    swarmIntelligence: createCollaborativeCapability("swarm_optimization", 5),
    
    // Grows exponentially with usage
    learningRate: createCompoundGrowthCapability(
        "pattern_recognition",
        0.1, // baseline
        0.15, // 15% growth per cycle
        "patterns/hour"
    ),
};
```

## Examples

### Research Swarm (Tier 1)
- **Config**: Simple research goal
- **Emergence**: Interdisciplinary synthesis, hypothesis generation
- **Evidence**: Different domains → different emergent capabilities

### Evolving Routine (Tier 2)
- **Config**: Basic workflow steps
- **Emergence**: Pattern discovery, automated optimization
- **Evidence**: Performance improves through usage, not updates

### Cross-Tier Integration
- **Config**: Three simple tiers
- **Emergence**: Capabilities impossible for any single tier
- **Evidence**: Synergy metrics > sum of parts

## Validation Checklist

✅ **Valid Emergent Fixture**:
- [ ] Config contains only goals/objectives, not methods
- [ ] Capabilities listed in `emergence`, not `config`
- [ ] Requires multiple agents/events for emergence
- [ ] Includes measurable learning metrics
- [ ] Shows evolution pathway
- [ ] Performance improves through usage

❌ **Invalid (Hard-Coded) Fixture**:
- [ ] Config specifies algorithms or methods
- [ ] Capabilities are input parameters
- [ ] Works with single agent/event
- [ ] No learning metrics
- [ ] Static performance
- [ ] Pre-programmed behaviors

## Testing Emergence

```typescript
// Test that capabilities truly emerge
it("should demonstrate emergent capabilities", () => {
    // 1. Verify config doesn't contain capabilities
    expect(fixture.config).not.toHaveProperty("capabilities");
    
    // 2. Check emergence conditions
    expect(fixture.emergence.emergenceConditions.minAgents).toBeGreaterThan(1);
    
    // 3. Validate evolution path
    expect(fixture.emergence.evolutionPath).toContain("→");
    
    // 4. Measure synergy
    const integrated = measureIntegratedPerformance(fixtures);
    const individual = fixtures.map(measureIndividualPerformance);
    expect(integrated).toBeGreaterThan(Math.max(...individual));
});
```

## Philosophy

Remember: We're testing that the **architecture enables emergence**, not implementing the emergent behaviors directly. The magic happens at runtime through agent collaboration, event-driven learning, and evolutionary pressure.

The fixtures demonstrate **what emerges**, not **how it's implemented**.