# Execution Service Test Framework

A comprehensive test framework for validating multi-agent workflows, emergent behaviors, and AI orchestration in the Vrooli execution service.

## Overview

This framework provides a schema-driven approach to testing complex multi-agent scenarios, enabling:

- **Emergent Behavior Validation**: Test how simple agent rules combine to create complex swarm intelligence
- **Multi-Agent Coordination**: Verify event-driven communication and blackboard state management
- **Deterministic AI Testing**: Mock AI responses predictably while testing different decision paths
- **Scenario-Based Testing**: Define complete workflows with expectations and validate outcomes

## Directory Structure

```
execution/
├── schemas/              # JSON schema definitions
│   ├── routines/        # Routine schemas by category
│   ├── agents/          # Agent schemas by role
│   └── swarms/          # Team/swarm schemas
├── factories/           # Object creation factories
│   ├── routine/         # Routine factory system
│   ├── agent/           # Agent factory system
│   ├── swarm/           # Swarm factory system
│   └── scenario/        # Scenario orchestration
├── scenarios/           # Complete test scenarios
│   └── redis-fix-loop/  # Example scenario
├── assertions/          # Custom test assertions
└── mocks/              # Mock systems
```

## Quick Start

### 1. Define Schemas

Create JSON schemas for your test components:

```json
// schemas/agents/fixers/example-fixer.json
{
    "identity": {
        "name": "example-fixer",
        "version": "1.0.0"
    },
    "goal": "Fix example issues",
    "subscriptions": ["custom/example/fix_requested"],
    "behaviors": [...],
    "testConfig": {
        "expectedRoutineCalls": ["example-fix-routine"],
        "expectedBlackboard": {
            "fix_applied": "string"
        }
    }
}
```

### 2. Create a Scenario

```typescript
// scenarios/example-fix/scenario.ts
import type { ScenarioDefinition } from "../../factories/scenario/types.js";

export const exampleScenario: ScenarioDefinition = {
    name: "example-fix",
    description: "Example fixing workflow",
    schemas: {
        routines: ["schemas/routines/actions/example-fix.json"],
        agents: ["schemas/agents/fixers/example-fixer.json"],
        swarms: ["schemas/swarms/development/example-team.json"]
    },
    mockConfig: {
        ai: {
            "example-fix-routine": {
                responses: [{ response: { fixed: true } }]
            }
        }
    },
    expectations: {
        eventSequence: ["custom/example/fixed"],
        finalBlackboard: { "status": "fixed" }
    }
};
```

### 3. Write Tests

```typescript
// scenarios/example-fix/test.ts
import { describe, it, expect } from "vitest";
import { ScenarioFactory, ScenarioRunner } from "../../factories/index.js";
import { initializeExecutionAssertions } from "../../assertions/index.js";
import { exampleScenario } from "./scenario.js";

initializeExecutionAssertions();

describe("Example Fix Scenario", () => {
    it("should fix the example issue", async () => {
        const factory = new ScenarioFactory();
        const scenario = await factory.setupScenario(exampleScenario);
        const runner = new ScenarioRunner(scenario);
        
        const result = await runner.execute({
            initialEvent: {
                type: "custom/example/fix_requested",
                data: { issue: "test" }
            }
        });
        
        expect(result.success).toBe(true);
        expect(result.events).toMatchSequence(["custom/example/fixed"]);
    });
});
```

## Key Components

### Schema System

Schemas define the structure and behavior of test components:

- **Routine Schemas**: Define inputs, outputs, and steps
- **Agent Schemas**: Define subscriptions, behaviors, and resources
- **Swarm Schemas**: Define team composition and coordination

### Factory System

Factories create test objects from schemas:

```typescript
const routineFactory = new RoutineFactory();
const routine = await routineFactory.createFromSchema(
    "schemas/routines/actions/example.json",
    { saveToDb: true, mockResponses: true }
);
```

### Mock Controller

Control AI and routine responses:

```typescript
scenario.mockController.override("routine:example", {
    success: true,
    output: { result: "mocked" }
});
```

### Custom Assertions

Enhanced assertions for multi-agent testing:

```typescript
// Blackboard assertions
expect(blackboard).toHaveValue("key", "expected");
expect(blackboard).toMatchState({ key: "value" });

// Event assertions
expect(events).toMatchSequence(["event1", "event2"]);
expect(events).toHaveEventWithData("topic", { data: "value" });

// Coordination assertions
expect(routineCalls).toHaveRetryPattern(3);
expect(routineCalls).toHaveParallelExecution(["routine1", "routine2"]);

// Outcome assertions
expect(result).toCompleteWithin(5000);
expect(result).toHaveSuccessRate(100);
```

## Advanced Features

### Coordination Patterns

Test complex agent coordination:

```typescript
expect(routineCalls).toHaveCoordinationPattern({
    type: "pipeline",
    config: {
        stages: ["analyze", "fix", "validate"]
    }
});
```

### State Evolution

Track how blackboard state evolves:

```typescript
const stateHistory = scenario.blackboard.getHistory();
expect(stateHistory).toShowLearning({
    metric: "success_rate",
    improvement: 0.2
});
```

### Resource Tracking

Validate resource usage:

```typescript
expect(result).toUseCreditsWithin(1000);
expect(result.resourceUsage).toMatchLimits({
    memory: 512,
    duration: 60000
});
```

## Best Practices

1. **Schema First**: Always define schemas before writing tests
2. **Isolated Scenarios**: Each scenario should be independent
3. **Mock Deterministically**: Use sequence-based mocks for predictable results
4. **Test Edge Cases**: Include failure scenarios and retry logic
5. **Validate Coordination**: Check both event sequences and timing

## Migration Guide

To migrate existing tests:

1. Extract test data into JSON schemas
2. Replace direct object creation with factories
3. Convert assertions to use custom matchers
4. Group related tests into scenarios

Example:

```typescript
// Before
const routine = { name: "test", steps: [...] };
await db.routine.create({ data: routine });

// After
const routine = await routineFactory.createFromSchema(
    "schemas/routines/test.json"
);
```

## Troubleshooting

### Schema Not Found

Ensure schema files are in the correct directory and have `.json` extension.

### Mock Not Working

Check that mock keys match routine/agent IDs exactly.

### Timing Issues

Use `timing` configuration to speed up tests:

```typescript
mockConfig: {
    timing: {
        "validation-delay": 100  // ms
    }
}
```

## Contributing

When adding new test scenarios:

1. Create schemas in appropriate directories
2. Define scenario with clear expectations
3. Write comprehensive tests covering success and failure
4. Document any special mock requirements
5. Add to CI test suite

## Related Documentation

- [Execution Service Architecture](../../../services/execution/README.md)
- [AI Creation System](../../../../../docs/ai-creation/README.md)
- [Event Bus System](../../../../../docs/architecture/core-services/event-bus-system.md)