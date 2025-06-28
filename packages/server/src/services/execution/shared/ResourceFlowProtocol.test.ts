import { describe, it, expect, beforeEach, vi } from "vitest";
import { ResourceFlowProtocol } from "./ResourceFlowProtocol.js";
import type { ResourceFlowConfig } from "./ResourceFlowProtocol.js";
import type { RunExecutionContext } from "../tier2/types.js";
import type { StepInfo } from "../tier3/types.js";
import { logger } from "../../../logger.js";

describe("ResourceFlowProtocol", () => {
    let protocol: ResourceFlowProtocol;
    let mockLogger: typeof logger;

    beforeEach(() => {
        mockLogger = {
            ...logger,
            info: vi.fn(),
            debug: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        };

        protocol = new ResourceFlowProtocol(mockLogger);
    });

    describe("Tier 3 Execution Request Creation", () => {
        it("should create properly formatted TierExecutionRequest for Tier 3", () => {
            const context: Partial<RunExecutionContext> = {
                runId: "test-run-001",
                routineVersionId: "routine-v1",
                userId: "user-123",
                organizationId: "org-456",
                inputs: {
                    inputValue: "test input",
                    numberValue: 42,
                },
                variables: {
                    stepVariable: "step value",
                },
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "step-001",
                stepType: "api",
                nodeId: "node-001",
                name: "Test API Step",
                description: "Test step for protocol validation",
                toolName: "http_request",
                inputs: {
                    url: "https://api.example.com/test",
                    method: "GET",
                },
            };

            const parentAllocation = {
                maxCredits: "5000",
                maxDurationMs: 60000,
                maxMemoryMB: 512,
                maxConcurrentSteps: 3,
            };

            const request = protocol.createTier3ExecutionRequest(
                context as RunExecutionContext,
                stepInfo as StepInfo,
                parentAllocation,
            );

            // Verify proper TierExecutionRequest format
            expect(request).toHaveProperty("context");
            expect(request).toHaveProperty("input");
            expect(request).toHaveProperty("allocation");
            expect(request).toHaveProperty("options");

            // Verify context is properly formatted
            expect(request.context.runId).toBe("test-run-001");
            expect(request.context.userId).toBe("user-123");
            expect(request.context.organizationId).toBe("org-456");

            // Verify input is properly formatted
            expect(request.input.stepId).toBe("step-001");
            expect(request.input.stepType).toBe("api");
            expect(request.input.toolName).toBe("http_request");

            // Verify allocation is included and properly formatted
            expect(request.allocation).toBeTruthy();
            expect(request.allocation.maxCredits).toBeTruthy();
            expect(request.allocation.maxDurationMs).toBeTruthy();

            // Verify this is NOT the old broken format
            expect(request).not.toHaveProperty("payload");
            expect(request).not.toHaveProperty("metadata");
        });

        it("should estimate resources based on step type and complexity", () => {
            const simpleContext: Partial<RunExecutionContext> = {
                runId: "simple-run",
                routineVersionId: "routine-v1",
            };

            const simpleStep: Partial<StepInfo> = {
                stepId: "simple-step",
                stepType: "data",
                toolName: "text_transform",
                inputs: { text: "simple text" },
            };

            const complexStep: Partial<StepInfo> = {
                stepId: "complex-step", 
                stepType: "llm",
                toolName: "claude_chat",
                inputs: {
                    model: "claude-3-5-sonnet-20241022",
                    messages: [{ role: "user", content: "Complex analysis task" }],
                    maxTokens: 4000,
                },
            };

            const parentAllocation = {
                maxCredits: "10000",
                maxDurationMs: 120000,
                maxMemoryMB: 1024,
                maxConcurrentSteps: 5,
            };

            const simpleRequest = protocol.createTier3ExecutionRequest(
                simpleContext as RunExecutionContext,
                simpleStep as StepInfo,
                parentAllocation,
            );

            const complexRequest = protocol.createTier3ExecutionRequest(
                simpleContext as RunExecutionContext,
                complexStep as StepInfo,
                parentAllocation,
            );

            // Complex LLM steps should get more resources than simple data steps
            const simpleCredits = BigInt(simpleRequest.allocation.maxCredits);
            const complexCredits = BigInt(complexRequest.allocation.maxCredits);

            expect(complexCredits).toBeGreaterThan(simpleCredits);
            expect(complexRequest.allocation.maxDurationMs).toBeGreaterThan(simpleRequest.allocation.maxDurationMs);
        });

        it("should handle missing parent allocation gracefully", () => {
            const context: Partial<RunExecutionContext> = {
                runId: "test-run-002",
                routineVersionId: "routine-v1",
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "step-002",
                stepType: "validation",
                toolName: "data_validator",
            };

            // Test with undefined parent allocation
            const request = protocol.createTier3ExecutionRequest(
                context as RunExecutionContext,
                stepInfo as StepInfo,
                undefined as any,
            );

            // Should create conservative fallback allocation
            expect(request.allocation).toBeTruthy();
            expect(request.allocation.maxCredits).toBeTruthy();
            expect(request.allocation.maxDurationMs).toBeTruthy();
            expect(request.allocation.maxMemoryMB).toBeTruthy();
        });

        it("should apply data-driven allocation strategies", () => {
            const config: ResourceFlowConfig = {
                allocationStrategies: {
                    api: {
                        creditMultiplier: 1.5,
                        timeoutMultiplier: 2.0,
                        memoryMultiplier: 1.2,
                    },
                    llm: {
                        creditMultiplier: 3.0,
                        timeoutMultiplier: 4.0,
                        memoryMultiplier: 2.0,
                    },
                },
                defaultStrategy: {
                    creditMultiplier: 1.0,
                    timeoutMultiplier: 1.0,
                    memoryMultiplier: 1.0,
                },
            };

            const protocolWithConfig = new ResourceFlowProtocol(mockLogger, config);

            const context: Partial<RunExecutionContext> = {
                runId: "config-test-run",
                routineVersionId: "routine-v1",
            };

            const apiStep: Partial<StepInfo> = {
                stepId: "api-step",
                stepType: "api",
                toolName: "http_request",
            };

            const llmStep: Partial<StepInfo> = {
                stepId: "llm-step",
                stepType: "llm",
                toolName: "claude_chat",
            };

            const parentAllocation = {
                maxCredits: "6000",
                maxDurationMs: 60000,
                maxMemoryMB: 256,
                maxConcurrentSteps: 2,
            };

            const apiRequest = protocolWithConfig.createTier3ExecutionRequest(
                context as RunExecutionContext,
                apiStep as StepInfo,
                parentAllocation,
            );

            const llmRequest = protocolWithConfig.createTier3ExecutionRequest(
                context as RunExecutionContext,
                llmStep as StepInfo,
                parentAllocation,
            );

            // LLM should get 3x credits vs API 1.5x, so LLM should get 2x API
            const apiCredits = BigInt(apiRequest.allocation.maxCredits);
            const llmCredits = BigInt(llmRequest.allocation.maxCredits);

            expect(llmCredits).toBeGreaterThan(apiCredits);
            expect(llmRequest.allocation.maxDurationMs).toBeGreaterThan(apiRequest.allocation.maxDurationMs);
            expect(llmRequest.allocation.maxMemoryMB).toBeGreaterThan(apiRequest.allocation.maxMemoryMB);
        });

        it("should preserve BigInt precision in credit calculations", () => {
            const context: Partial<RunExecutionContext> = {
                runId: "precision-test",
                routineVersionId: "routine-v1",
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "precision-step",
                stepType: "llm",
                toolName: "claude_chat",
            };

            const parentAllocation = {
                maxCredits: "999999999999999999", // Large BigInt value
                maxDurationMs: 60000,
                maxMemoryMB: 256,
                maxConcurrentSteps: 1,
            };

            const request = protocol.createTier3ExecutionRequest(
                context as RunExecutionContext,
                stepInfo as StepInfo,
                parentAllocation,
            );

            // Should handle large BigInt values without precision loss
            const allocatedCredits = BigInt(request.allocation.maxCredits);
            expect(allocatedCredits).toBeGreaterThan(BigInt(0));
            expect(allocatedCredits).toBeLessThanOrEqual(BigInt("999999999999999999"));

            // Verify it's a valid BigInt string
            expect(() => BigInt(request.allocation.maxCredits)).not.toThrow();
        });
    });

    describe("Hierarchical Resource Allocation", () => {
        it("should allocate resources hierarchically from parent to child", async () => {
            const parentAllocation = {
                maxCredits: "10000",
                maxDurationMs: 120000,
                maxMemoryMB: 1024,
                maxConcurrentSteps: 5,
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "hierarchical-step",
                stepType: "routine",
                toolName: "execute_routine",
                inputs: {
                    routineId: "child-routine-001",
                },
            };

            const context: Partial<RunExecutionContext> = {
                runId: "parent-run",
                routineVersionId: "parent-routine-v1",
            };

            const allocation = await protocol.allocateFromParent(
                parentAllocation,
                stepInfo as StepInfo,
                context as RunExecutionContext,
            );

            expect(allocation.allocated.credits).toBeLessThanOrEqual(BigInt(parentAllocation.maxCredits));
            expect(allocation.allocated.timeoutMs).toBeLessThanOrEqual(parentAllocation.maxDurationMs);
            expect(allocation.allocated.memoryMB).toBeLessThanOrEqual(parentAllocation.maxMemoryMB);
            expect(allocation.allocated.concurrentExecutions).toBeLessThanOrEqual(parentAllocation.maxConcurrentSteps);
        });

        it("should validate allocation requests against parent limits", async () => {
            const limitedParent = {
                maxCredits: "100",
                maxDurationMs: 5000,
                maxMemoryMB: 64,
                maxConcurrentSteps: 1,
            };

            const expensiveStep: Partial<StepInfo> = {
                stepId: "expensive-step",
                stepType: "llm",
                toolName: "claude_chat",
                inputs: {
                    model: "claude-3-5-sonnet-20241022",
                    maxTokens: 10000, // Very expensive
                },
            };

            const context: Partial<RunExecutionContext> = {
                runId: "validation-test",
                routineVersionId: "test-routine-v1",
            };

            await expect(
                protocol.allocateFromParent(
                    limitedParent,
                    expensiveStep as StepInfo,
                    context as RunExecutionContext,
                ),
            ).rejects.toThrow(/insufficient.*resources/i);
        });

        it("should track allocation hierarchy for proper cleanup", async () => {
            const parentAllocation = {
                maxCredits: "5000",
                maxDurationMs: 60000,
                maxMemoryMB: 512,
                maxConcurrentSteps: 3,
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "tracked-step",
                stepType: "api",
                toolName: "http_request",
            };

            const context: Partial<RunExecutionContext> = {
                runId: "tracking-test",
                routineVersionId: "test-routine-v1",
            };

            const allocation = await protocol.allocateFromParent(
                parentAllocation,
                stepInfo as StepInfo,
                context as RunExecutionContext,
            );

            // Allocation should include hierarchy information
            expect(allocation.parentAllocationId).toBeTruthy();
            expect(allocation.allocationPath).toBeTruthy();
            expect(allocation.allocationPath.length).toBeGreaterThan(0);
        });

        it("should release resources back to parent allocation", async () => {
            const parentAllocation = {
                maxCredits: "3000",
                maxDurationMs: 45000,
                maxMemoryMB: 384,
                maxConcurrentSteps: 2,
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "release-test-step",
                stepType: "data",
                toolName: "data_processor",
            };

            const context: Partial<RunExecutionContext> = {
                runId: "release-test",
                routineVersionId: "test-routine-v1",
            };

            const allocation = await protocol.allocateFromParent(
                parentAllocation,
                stepInfo as StepInfo,
                context as RunExecutionContext,
            );

            // Simulate partial resource usage
            const usedResources = {
                credits: BigInt(500),
                timeoutMs: 5000,
                memoryMB: 64,
                concurrentExecutions: 1,
            };

            const released = await protocol.releaseToParent(allocation, usedResources);

            // Should return unused resources to parent
            expect(released.returnedToParent.credits).toBe(
                allocation.allocated.credits - usedResources.credits,
            );
            expect(released.returnedToParent.timeoutMs).toBe(
                allocation.allocated.timeoutMs - usedResources.timeoutMs,
            );
        });
    });

    describe("Data-Driven Configuration", () => {
        it("should support emergent agent modifications to allocation strategies", () => {
            const emergentConfig: ResourceFlowConfig = {
                allocationStrategies: {
                    api: {
                        creditMultiplier: 2.5, // Agent-optimized value
                        timeoutMultiplier: 1.8,
                        memoryMultiplier: 1.4,
                    },
                    llm: {
                        creditMultiplier: 4.5, // Agent-learned optimization
                        timeoutMultiplier: 5.0,
                        memoryMultiplier: 3.0,
                    },
                },
                defaultStrategy: {
                    creditMultiplier: 1.2,
                    timeoutMultiplier: 1.1,
                    memoryMultiplier: 1.0,
                },
                emergentOptimizations: {
                    enabled: true,
                    learningRate: 0.1,
                    confidenceThreshold: 0.8,
                },
            };

            const emergentProtocol = new ResourceFlowProtocol(mockLogger, emergentConfig);

            const context: Partial<RunExecutionContext> = {
                runId: "emergent-test",
                routineVersionId: "test-routine-v1",
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "emergent-step",
                stepType: "llm",
                toolName: "claude_chat",
            };

            const parentAllocation = {
                maxCredits: "4000",
                maxDurationMs: 60000,
                maxMemoryMB: 256,
                maxConcurrentSteps: 2,
            };

            const request = emergentProtocol.createTier3ExecutionRequest(
                context as RunExecutionContext,
                stepInfo as StepInfo,
                parentAllocation,
            );

            // Should apply emergent optimizations
            expect(BigInt(request.allocation.maxCredits)).toBeGreaterThan(BigInt(1000));
            expect(request.allocation.maxDurationMs).toBeGreaterThan(30000);
        });

        it("should handle historical performance data integration", () => {
            const historicalConfig: ResourceFlowConfig = {
                allocationStrategies: {
                    api: {
                        creditMultiplier: 1.3,
                        timeoutMultiplier: 1.5,
                        memoryMultiplier: 1.1,
                    },
                },
                defaultStrategy: {
                    creditMultiplier: 1.0,
                    timeoutMultiplier: 1.0,
                    memoryMultiplier: 1.0,
                },
                historicalPerformanceData: {
                    enabled: true,
                    windowDays: 7,
                    minimumSamples: 10,
                },
            };

            const historicalProtocol = new ResourceFlowProtocol(mockLogger, historicalConfig);

            // Protocol should be able to accept historical data
            expect(historicalProtocol).toBeDefined();
            expect(mockLogger.debug).toHaveBeenCalledWith(
                expect.stringContaining("historical performance"),
            );
        });
    });

    describe("Error Handling and Edge Cases", () => {
        it("should handle invalid step types gracefully", () => {
            const context: Partial<RunExecutionContext> = {
                runId: "invalid-test",
                routineVersionId: "test-routine-v1",
            };

            const invalidStep: Partial<StepInfo> = {
                stepId: "invalid-step",
                stepType: "unknown_type" as any,
                toolName: "unknown_tool",
            };

            const parentAllocation = {
                maxCredits: "1000",
                maxDurationMs: 30000,
                maxMemoryMB: 128,
                maxConcurrentSteps: 1,
            };

            // Should not throw, should use default allocation strategy
            const request = protocol.createTier3ExecutionRequest(
                context as RunExecutionContext,
                invalidStep as StepInfo,
                parentAllocation,
            );

            expect(request.allocation).toBeTruthy();
            expect(BigInt(request.allocation.maxCredits)).toBeGreaterThan(BigInt(0));
        });

        it("should handle zero or negative parent allocations", () => {
            const context: Partial<RunExecutionContext> = {
                runId: "zero-allocation-test",
                routineVersionId: "test-routine-v1",
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "step-001",
                stepType: "data",
                toolName: "data_processor",
            };

            const zeroAllocation = {
                maxCredits: "0",
                maxDurationMs: 0,
                maxMemoryMB: 0,
                maxConcurrentSteps: 0,
            };

            // Should provide minimum viable allocation
            const request = protocol.createTier3ExecutionRequest(
                context as RunExecutionContext,
                stepInfo as StepInfo,
                zeroAllocation,
            );

            expect(BigInt(request.allocation.maxCredits)).toBeGreaterThan(BigInt(0));
            expect(request.allocation.maxDurationMs).toBeGreaterThan(0);
            expect(request.allocation.maxMemoryMB).toBeGreaterThan(0);
        });

        it("should validate BigInt conversion safety", () => {
            const context: Partial<RunExecutionContext> = {
                runId: "bigint-safety-test",
                routineVersionId: "test-routine-v1",
            };

            const stepInfo: Partial<StepInfo> = {
                stepId: "step-001",
                stepType: "data",
                toolName: "data_processor",
            };

            const invalidAllocation = {
                maxCredits: "not-a-number",
                maxDurationMs: 30000,
                maxMemoryMB: 128,
                maxConcurrentSteps: 1,
            };

            // Should handle invalid BigInt strings gracefully
            const request = protocol.createTier3ExecutionRequest(
                context as RunExecutionContext,
                stepInfo as StepInfo,
                invalidAllocation,
            );

            expect(request.allocation).toBeTruthy();
            expect(() => BigInt(request.allocation.maxCredits)).not.toThrow();
        });
    });
});
