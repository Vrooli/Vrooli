/**
 * Comprehensive tests for cross-tier communication type guards
 * 
 * These tests ensure type safety and proper validation of all
 * cross-tier communication input types and execution contexts.
 */

import { describe, expect, it } from "vitest";
import type {
    RoutineExecutionInput,
    StepExecutionInput,
    SwarmExecutionInput,
    TierExecutionRequest,
} from "./communication.js";
import type {
    ExecutionContext,
} from "./core.js";
import {
    discriminateTierInput,
    isRoutineExecutionInput,
    isStepExecutionInput,
    isSwarmCoordinationInput,
    isValidExecutionContext,
    isValidStepType,
    isValidStrategy,
    isValidTierExecutionRequest,
    VALID_STEP_TYPES,
    VALID_STRATEGIES,
} from "./typeGuards.js";

describe("SwarmExecutionInput Type Guard", () => {
    const validSwarmInput = {
        goal: "Complete project analysis",
        swarmId: "swarm-123",
        teamConfiguration: {
            leaderAgentId: "agent-1",
            preferredTeamSize: 3,
            requiredSkills: ["analysis", "coordination"],
        },
        availableTools: [
            {
                name: "data_processor",
                description: "Process and analyze data",
            },
        ],
        executionConfig: {
            model: "gpt-4",
            temperature: 0.7,
            parallelExecutionLimit: 5,
        },
    };

    it("should validate correct SwarmExecutionInput", () => {
        expect(isSwarmCoordinationInput(validSwarmInput)).toBe(true);
    });

    it("should reject null or undefined input", () => {
        expect(isSwarmCoordinationInput(null)).toBe(false);
        expect(isSwarmCoordinationInput(undefined)).toBe(false);
    });

    it("should reject non-object input", () => {
        expect(isSwarmCoordinationInput("string")).toBe(false);
        expect(isSwarmCoordinationInput(123)).toBe(false);
        expect(isSwarmCoordinationInput([])).toBe(false);
    });

    it("should reject input without required goal", () => {
        const invalidInput = { ...validSwarmInput };
        delete (invalidInput as any).goal;
        expect(isSwarmCoordinationInput(invalidInput)).toBe(false);
    });

    it("should reject input with empty goal", () => {
        const invalidInput = { ...validSwarmInput, goal: "" };
        expect(isSwarmCoordinationInput(invalidInput)).toBe(false);

        const whitespaceInput = { ...validSwarmInput, goal: "   " };
        expect(isSwarmCoordinationInput(whitespaceInput)).toBe(false);
    });

    it("should validate input without optional teamConfiguration", () => {
        const minimalInput = {
            goal: "Test goal",
        };
        expect(isSwarmCoordinationInput(minimalInput)).toBe(true);
    });

    it("should reject input with invalid teamConfiguration", () => {
        const invalidInput = {
            ...validSwarmInput,
            teamConfiguration: {
                // Missing required fields
                preferredTeamSize: 3,
                requiredSkills: ["analysis"],
            },
        };
        expect(isSwarmCoordinationInput(invalidInput)).toBe(false);
    });

    it("should accept input without optional fields", () => {
        const minimalInput = {
            goal: "Test goal",
        };
        expect(isSwarmCoordinationInput(minimalInput)).toBe(true);
    });
});

describe("RoutineExecutionInput Type Guard", () => {
    const validRoutineInput: RoutineExecutionInput = {
        routineId: "routine-123",
        parameters: { input: "test data" },
        workflow: {
            steps: [
                {
                    id: "step-1",
                    name: "Process Data",
                    toolName: "data_processor",
                    parameters: { format: "json" },
                    strategy: "deterministic",
                },
            ],
            dependencies: [
                {
                    stepId: "step-1",
                    dependsOn: [],
                },
            ],
        },
    };

    it("should validate correct RoutineExecutionInput", () => {
        expect(isRoutineExecutionInput(validRoutineInput)).toBe(true);
    });

    it("should reject null or undefined input", () => {
        expect(isRoutineExecutionInput(null)).toBe(false);
        expect(isRoutineExecutionInput(undefined)).toBe(false);
    });

    it("should reject input without routineId", () => {
        const invalidInput = { ...validRoutineInput };
        delete (invalidInput as any).routineId;
        expect(isRoutineExecutionInput(invalidInput)).toBe(false);
    });

    it("should reject input with empty routineId", () => {
        const invalidInput = { ...validRoutineInput, routineId: "" };
        expect(isRoutineExecutionInput(invalidInput)).toBe(false);
    });

    it("should reject input without parameters", () => {
        const invalidInput = { ...validRoutineInput };
        delete (invalidInput as any).parameters;
        expect(isRoutineExecutionInput(invalidInput)).toBe(false);
    });

    it("should reject input with null parameters", () => {
        const invalidInput = { ...validRoutineInput, parameters: null };
        expect(isRoutineExecutionInput(invalidInput)).toBe(false);
    });

    it("should reject input without workflow", () => {
        const invalidInput = { ...validRoutineInput };
        delete (invalidInput as any).workflow;
        expect(isRoutineExecutionInput(invalidInput)).toBe(false);
    });

    it("should reject input with invalid workflow", () => {
        const invalidInput = {
            ...validRoutineInput,
            workflow: {
                steps: "invalid", // Should be array
                dependencies: [],
            },
        };
        expect(isRoutineExecutionInput(invalidInput)).toBe(false);
    });
});

describe("StepExecutionInput Type Guard", () => {
    const validStepInput: StepExecutionInput = {
        stepId: "step-123",
        stepType: "tool_call",
        parameters: { arg1: "value1" },
        strategy: "conversational",
        toolName: "test_tool",
    };

    it("should validate correct StepExecutionInput", () => {
        expect(isStepExecutionInput(validStepInput)).toBe(true);
    });

    it("should accept input without optional toolName", () => {
        const inputWithoutTool = { ...validStepInput };
        delete (inputWithoutTool as any).toolName;
        expect(isStepExecutionInput(inputWithoutTool)).toBe(true);
    });

    it("should reject null or undefined input", () => {
        expect(isStepExecutionInput(null)).toBe(false);
        expect(isStepExecutionInput(undefined)).toBe(false);
    });

    it("should reject input without required fields", () => {
        const requiredFields = ["stepId", "stepType", "parameters", "strategy"];

        for (const field of requiredFields) {
            const invalidInput = { ...validStepInput };
            delete (invalidInput as any)[field];
            expect(isStepExecutionInput(invalidInput)).toBe(false);
        }
    });

    it("should reject input with empty string fields", () => {
        expect(isStepExecutionInput({ ...validStepInput, stepId: "" })).toBe(false);
        expect(isStepExecutionInput({ ...validStepInput, stepType: "" })).toBe(false);
        expect(isStepExecutionInput({ ...validStepInput, strategy: "" })).toBe(false);
    });

    it("should reject invalid toolName type", () => {
        const invalidInput = { ...validStepInput, toolName: 123 };
        expect(isStepExecutionInput(invalidInput)).toBe(false);
    });
});

describe("Discriminated Union Type Guard", () => {
    const swarmInput: SwarmExecutionInput = {
        goal: "Test goal",
        availableAgents: [{
            id: "agent-1",
            name: "Test Agent",
            capabilities: ["test"],
            currentLoad: 0,
            maxConcurrentTasks: 1,
        }],
    };

    const routineInput: RoutineExecutionInput = {
        routineId: "routine-123",
        parameters: {},
        workflow: {
            steps: [{
                id: "step-1",
                name: "Test Step",
                toolName: "test_tool",
                parameters: {},
                strategy: "deterministic",
            }],
            dependencies: [{
                stepId: "step-1",
                dependsOn: [],
            }],
        },
    };

    const stepInput: StepExecutionInput = {
        stepId: "step-123",
        stepType: "tool_call",
        parameters: {},
        strategy: "conversational",
    };

    it("should discriminate SwarmExecutionInput", () => {
        const result = discriminateTierInput(swarmInput);
        expect(result).toBe(swarmInput);
    });

    it("should discriminate RoutineExecutionInput", () => {
        const result = discriminateTierInput(routineInput);
        expect(result).toBe(routineInput);
    });

    it("should discriminate StepExecutionInput", () => {
        const result = discriminateTierInput(stepInput);
        expect(result).toBe(stepInput);
    });

    it("should return null for invalid input", () => {
        expect(discriminateTierInput(null)).toBe(null);
        expect(discriminateTierInput({})).toBe(null);
        expect(discriminateTierInput("invalid")).toBe(null);
    });
});

describe("ExecutionContext Validation", () => {
    const validContext: ExecutionContext = {
        executionId: "exec-123",
        swarmId: "swarm-456",
        userId: "user-789",
        correlationId: "corr-abc",
        timestamp: Date.now(),
        parentExecutionId: "parent-exec-123",
        stepId: "step-456",
        routineId: "routine-789",
    };

    it("should validate correct ExecutionContext", () => {
        expect(isValidExecutionContext(validContext)).toBe(true);
    });

    it("should accept context without optional fields", () => {
        const minimalContext = {
            executionId: "exec-123",
            swarmId: "swarm-456",
            userId: "user-789",
            correlationId: "corr-abc",
            timestamp: Date.now(),
        };
        expect(isValidExecutionContext(minimalContext)).toBe(true);
    });

    it("should reject null or undefined context", () => {
        expect(isValidExecutionContext(null)).toBe(false);
        expect(isValidExecutionContext(undefined)).toBe(false);
    });

    it("should reject context without required fields", () => {
        const requiredFields = ["executionId", "swarmId", "userId", "correlationId", "timestamp"];

        for (const field of requiredFields) {
            const invalidContext = { ...validContext };
            delete (invalidContext as any)[field];
            expect(isValidExecutionContext(invalidContext)).toBe(false);
        }
    });

    it("should reject context with empty string fields", () => {
        const stringFields = ["executionId", "swarmId", "userId", "correlationId"];

        for (const field of stringFields) {
            const invalidContext = { ...validContext, [field]: "" };
            expect(isValidExecutionContext(invalidContext)).toBe(false);

            const whitespaceContext = { ...validContext, [field]: "   " };
            expect(isValidExecutionContext(whitespaceContext)).toBe(false);
        }
    });

    it("should reject context with empty optional string fields", () => {
        const optionalFields = ["parentExecutionId", "stepId", "routineId"];

        for (const field of optionalFields) {
            const invalidContext = { ...validContext, [field]: "" };
            expect(isValidExecutionContext(invalidContext)).toBe(false);
        }
    });
});

describe("TierExecutionRequest Validation", () => {
    const validRequest: TierExecutionRequest<StepExecutionInput> = {
        context: {
            executionId: "exec-123",
            swarmId: "swarm-456",
            userId: "user-789",
            correlationId: "corr-abc",
            timestamp: Date.now(),
        },
        input: {
            stepId: "step-123",
            stepType: "tool_call",
            parameters: {},
            strategy: "conversational",
        },
        allocation: {
            maxCredits: "1000",
            maxDurationMs: 60000,
            maxMemoryMB: 512,
            maxConcurrentSteps: 3,
        },
    };

    it("should validate correct TierExecutionRequest", () => {
        expect(isValidTierExecutionRequest(validRequest, isStepExecutionInput)).toBe(true);
    });

    it("should reject request with invalid context", () => {
        const invalidRequest = {
            ...validRequest,
            context: { executionId: "" }, // Invalid context
        };
        expect(isValidTierExecutionRequest(invalidRequest, isStepExecutionInput)).toBe(false);
    });

    it("should reject request with invalid input", () => {
        const invalidRequest = {
            ...validRequest,
            input: { stepId: "" }, // Invalid input
        };
        expect(isValidTierExecutionRequest(invalidRequest, isStepExecutionInput)).toBe(false);
    });

    it("should reject request with invalid allocation", () => {
        const invalidRequest = {
            ...validRequest,
            allocation: { maxCredits: "invalid" }, // Invalid allocation
        };
        expect(isValidTierExecutionRequest(invalidRequest, isStepExecutionInput)).toBe(false);
    });
});

describe("Strategy Validation", () => {
    it("should validate all valid strategies", () => {
        for (const strategy of VALID_STRATEGIES) {
            expect(isValidStrategy(strategy)).toBe(true);
        }
    });

    it("should reject invalid strategies", () => {
        const invalidStrategies = ["invalid", "unknown", "", "CONVERSATIONAL"];

        for (const strategy of invalidStrategies) {
            expect(isValidStrategy(strategy)).toBe(false);
        }
    });
});

describe("StepType Validation", () => {
    it("should validate all valid step types", () => {
        for (const stepType of VALID_STEP_TYPES) {
            expect(isValidStepType(stepType)).toBe(true);
        }
    });

    it("should reject invalid step types", () => {
        const invalidTypes = ["invalid", "unknown", "", "TOOL_CALL"];

        for (const stepType of invalidTypes) {
            expect(isValidStepType(stepType)).toBe(false);
        }
    });
});

describe("Edge Cases and Error Conditions", () => {
    it("should handle deeply nested invalid objects", () => {
        const deeplyNested = {
            goal: "test",
            teamConfiguration: {
                leaderAgentId: "agent-1",
                preferredTeamSize: 3,
                requiredSkills: [123], // Invalid: should be strings
            },
        };
        expect(isSwarmCoordinationInput(deeplyNested)).toBe(false);
    });

    it("should handle circular references gracefully", () => {
        const circular: any = {
            goal: "test",
        };
        circular.self = circular;

        expect(() => isSwarmCoordinationInput(circular)).not.toThrow();
    });

    it("should handle very large objects efficiently", () => {
        const largeToolsList = Array.from({ length: 1000 }, (_, i) => ({
            name: `tool-${i}`,
            description: `Tool ${i} description`,
        }));

        const largeInput = {
            goal: "test",
            availableTools: largeToolsList,
        };

        const startTime = Date.now();
        const result = isSwarmCoordinationInput(largeInput);
        const duration = Date.now() - startTime;

        expect(result).toBe(true);
        expect(duration).toBeLessThan(100); // Should complete within 100ms
    });
});

