# Multi-Agent Workflow Testing Guide

This guide explains how to create comprehensive integration tests for multi-agent workflows in the Vrooli execution system.

## Overview

Multi-agent workflows involve multiple agents coordinating through events and shared blackboard state to accomplish complex tasks. Testing these workflows requires:

1. **Event sequence verification** - Ensuring agents communicate in the expected order
2. **Blackboard state tracking** - Verifying shared data is updated correctly
3. **Coordination pattern validation** - Confirming agents follow expected interaction patterns
4. **Error handling verification** - Testing failure modes and recovery

## Quick Start

### 1. Define Your Scenario

Create a multi-agent scenario using the `MultiAgentScenario` interface:

```typescript
import { multiAgentWorkflowFactory } from "../fixtures/tasks/multiAgentWorkflowFactory.js";

// Use existing scenario
const scenario = multiAgentWorkflowFactory.redisFixLoop();

// Or create custom scenario
const customScenario: MultiAgentScenario = {
    name: "your-workflow",
    description: "Description of what this workflow does",
    agents: [
        {
            id: "agent-1",
            name: "problem-solver",
            role: "fixer",
            subscriptions: ["custom/problem/detected"],
            behaviors: [
                {
                    trigger: { topic: "custom/problem/detected" },
                    action: {
                        type: "routine",
                        label: "problem-analysis",
                        outputOperations: {
                            set: [{ routineOutput: "analysis", blackboardId: "problem_analysis" }]
                        }
                    }
                }
            ]
        }
        // ... more agents
    ],
    routines: [
        {
            id: "routine-1",
            name: "problem-analysis",
            type: "RoutineGenerate",
            config: { /* routine config */ },
            mockResponse: { /* mock response for testing */ }
        }
        // ... more routines
    ],
    workflows: [
        {
            stepNumber: 1,
            expectedTrigger: "custom/problem/detected",
            expectedAction: "problem-analysis routine execution",
            expectedBlackboardUpdates: { problem_analysis: "detailed analysis" },
            expectedEvents: ["custom/problem/analyzed"]
        }
        // ... more steps
    ],
    expectedEvents: [
        {
            eventType: "custom/problem/detected",
            expectedData: { problemType: "connection_issue" },
            order: 1
        }
        // ... more events
    ],
    blackboardState: {
        workflow_config: { max_attempts: 3, timeout: 300000 }
    }
};
```

### 2. Create Integration Test

```typescript
import { createMultiAgentIntegrationTest } from "../helpers/multiAgentTestUtils.js";

describe("Your Multi-Agent Workflow", () => {
    let scenario: MultiAgentScenario;
    let integrationTest: ReturnType<typeof createMultiAgentIntegrationTest>;

    beforeEach(() => {
        scenario = yourScenarioFactory();
        integrationTest = createMultiAgentIntegrationTest(scenario);
    });

    it("should execute complete workflow", async () => {
        const { eventEmit, blackboardUpdate } = integrationTest.setupMocks();
        
        const summary = await integrationTest.runWorkflowTest(async (stepNumber, trigger) => {
            // Implement step execution logic
            switch (stepNumber) {
                case 1:
                    // Execute agent behavior for step 1
                    const result = await executeAgentAction(trigger);
                    await blackboardUpdate("key", result);
                    await eventEmit("next/event", { data: result });
                    return "action description";
                    
                // ... more cases
            }
        });

        // Verify results
        expect(summary.totalEvents).toBe(expectedEventCount);
        expect(summary.stepsExecuted).toBe(expectedStepCount);
    });
});
```

## Architecture Components

### MultiAgentTestOrchestrator

Central coordinator for multi-agent tests:

```typescript
const orchestrator = new MultiAgentTestOrchestrator(scenario);

// Capture events and blackboard changes
const eventMock = orchestrator.createEventCaptureMock();
const blackboardMock = orchestrator.createBlackboardCaptureMock();

// Verify execution
orchestrator.verifyEventSequence();
orchestrator.verifyBlackboardProgression();
orchestrator.verifyWorkflowStep(1, "trigger", "action");
```

### Event Capture System

Tracks all events emitted during workflow execution:

```typescript
interface EventCapture {
    eventType: string;           // Type of event emitted
    data: Record<string, unknown>; // Event payload
    timestamp: number;           // When event occurred
    order: number;              // Sequence order
}
```

### Blackboard State Tracking

Monitors shared state changes:

```typescript
interface BlackboardStateCapture {
    key: string;        // Blackboard key updated
    value: unknown;     // New value
    timestamp: number;  // When update occurred
    step: number;       // Which workflow step
}
```

## Testing Patterns

### 1. Sequential Coordination

Agents execute in strict order:

```typescript
multiAgentAssertions.expectAgentCoordination(orchestrator, "sequential");
```

### 2. Parallel Execution

Multiple agents work simultaneously:

```typescript
multiAgentAssertions.expectAgentCoordination(orchestrator, "parallel");
```

### 3. Loop Workflows

Agents repeat actions until condition met:

```typescript
multiAgentAssertions.expectAgentCoordination(orchestrator, "loop");
```

### 4. Blackboard Convergence

Verify final shared state:

```typescript
multiAgentAssertions.expectBlackboardConvergence(orchestrator, {
    final_status: "completed",
    result_confidence: 0.95
});
```

## Common Scenarios

### Error Handling Workflow

```typescript
const errorHandlingScenario = {
    agents: [
        { role: "detector", subscriptions: ["run/failed"] },
        { role: "analyzer", subscriptions: ["custom/error/detected"] },
        { role: "fixer", subscriptions: ["custom/error/analyzed"] }
    ],
    workflows: [
        {
            stepNumber: 1,
            expectedTrigger: "run/failed",
            expectedAction: "error detection",
            expectedEvents: ["custom/error/detected"]
        },
        {
            stepNumber: 2,
            expectedTrigger: "custom/error/detected", 
            expectedAction: "error analysis",
            expectedEvents: ["custom/error/analyzed"]
        },
        {
            stepNumber: 3,
            expectedTrigger: "custom/error/analyzed",
            expectedAction: "error fixing",
            expectedEvents: ["custom/error/fixed"]
        }
    ]
};
```

### Data Pipeline Workflow

```typescript
const dataPipelineScenario = {
    agents: [
        { role: "validator", subscriptions: ["data/received"] },
        { role: "processor", subscriptions: ["data/validated"] }, 
        { role: "aggregator", subscriptions: ["data/processed"] }
    ],
    workflows: [
        {
            stepNumber: 1,
            expectedTrigger: "data/received",
            expectedBlackboardUpdates: { validation_result: "passed" }
        },
        {
            stepNumber: 2, 
            expectedTrigger: "data/validated",
            expectedBlackboardUpdates: { processed_count: 100 }
        },
        {
            stepNumber: 3,
            expectedTrigger: "data/processed",
            expectedBlackboardUpdates: { aggregation_complete: true }
        }
    ]
};
```

### Monitoring and Alerting

```typescript
const monitoringScenario = {
    agents: [
        { role: "monitor", subscriptions: ["system/metrics"] },
        { role: "analyzer", subscriptions: ["system/alert"] },
        { role: "responder", subscriptions: ["system/critical"] }
    ],
    workflows: [
        {
            stepNumber: 1,
            expectedTrigger: "system/metrics",
            expectedAction: "threshold checking"
        },
        {
            stepNumber: 2,
            expectedTrigger: "system/alert", 
            expectedAction: "severity analysis"
        },
        {
            stepNumber: 3,
            expectedTrigger: "system/critical",
            expectedAction: "incident response"
        }
    ]
};
```

## Best Practices

### 1. Mock Strategy

```typescript
// Mock external dependencies
const appLifecycleMocks = {
    startApp: vi.fn().mockResolvedValue({ processId: "123" }),
    stopApp: vi.fn().mockResolvedValue({ success: true }),
    checkStatus: vi.fn().mockResolvedValue({ running: true })
};

// Mock routine responses
mockRoutineStateMachine.executeRoutine = vi.fn().mockImplementation(async (routineId) => {
    switch (routineId) {
        case "data-validator":
            return { success: true, result: { isValid: true } };
        case "data-processor":
            return { success: true, result: { processedCount: 100 } };
        default:
            throw new Error(`Unknown routine: ${routineId}`);
    }
});
```

### 2. Error Simulation

```typescript
it("should handle routine failures", async () => {
    // Mock failure
    mockRoutineStateMachine.executeRoutine = vi.fn()
        .mockRejectedValueOnce(new Error("Network timeout"))
        .mockResolvedValue({ success: true, result: { recovered: true } });

    // Test failure handling
    await expect(executeWorkflowStep(1)).rejects.toThrow("Network timeout");
    
    // Test recovery
    const result = await executeWorkflowStep(1);
    expect(result.recovered).toBe(true);
});
```

### 3. Timing and Async Handling

```typescript
it("should handle async coordination", async () => {
    const promises = [];
    
    // Start multiple async operations
    promises.push(eventEmit("parallel/task/1", {}));
    promises.push(eventEmit("parallel/task/2", {}));
    promises.push(eventEmit("parallel/task/3", {}));
    
    // Wait for all to complete
    await Promise.all(promises);
    
    // Verify parallel execution
    multiAgentAssertions.expectAgentCoordination(orchestrator, "parallel");
});
```

### 4. State Verification

```typescript
it("should maintain workflow state", async () => {
    // Execute workflow steps
    await executeStep(1);
    await executeStep(2);
    await executeStep(3);
    
    // Verify state progression
    const blackboardChanges = orchestrator.getCapturedBlackboardChanges();
    expect(blackboardChanges).toContainEqual({
        key: "workflow_status",
        value: "in_progress",
        step: 1
    });
    expect(blackboardChanges).toContainEqual({
        key: "workflow_status", 
        value: "completed",
        step: 3
    });
});
```

## Debugging and Troubleshooting

### Execution Summary

```typescript
const summary = orchestrator.generateExecutionSummary();
console.log({
    totalEvents: summary.totalEvents,
    totalBlackboardUpdates: summary.totalBlackboardUpdates,
    stepsExecuted: summary.stepsExecuted,
    eventTypes: summary.eventTypes,
    blackboardKeys: summary.blackboardKeys,
    executionTime: summary.executionTime
});
```

### Event Debugging

```typescript
const events = orchestrator.getCapturedEvents();
events.forEach((event, index) => {
    console.log(`Event ${index + 1}: ${event.eventType}`, event.data);
});
```

### Blackboard Debugging

```typescript
const blackboardChanges = orchestrator.getCapturedBlackboardChanges();
blackboardChanges.forEach((change, index) => {
    console.log(`Change ${index + 1}: ${change.key} = ${JSON.stringify(change.value)}`);
});
```

## Extension Points

### Custom Assertions

```typescript
export const customAssertions = {
    expectWorkflowCompletion(orchestrator: MultiAgentTestOrchestrator, timeLimit: number) {
        const summary = orchestrator.generateExecutionSummary();
        expect(summary.executionTime).toBeLessThan(timeLimit);
        expect(summary.eventTypes).toContain("workflow/completed");
    },

    expectErrorRecovery(orchestrator: MultiAgentTestOrchestrator) {
        const events = orchestrator.getCapturedEvents();
        const hasError = events.some(e => e.eventType.includes("error"));
        const hasRecovery = events.some(e => e.eventType.includes("recovered"));
        expect(hasError && hasRecovery).toBe(true);
    }
};
```

### Custom Scenario Builders

```typescript
export function createCustomWorkflowScenario(config: {
    workflowType: string;
    agentCount: number;
    maxRetries: number;
}): MultiAgentScenario {
    return {
        name: `${config.workflowType}-workflow`,
        agents: generateAgents(config.agentCount),
        routines: generateRoutines(config.workflowType),
        workflows: generateWorkflowSteps(config.maxRetries),
        // ... rest of scenario
    };
}
```

This testing infrastructure provides comprehensive coverage for multi-agent workflows while maintaining flexibility for different coordination patterns and error scenarios.