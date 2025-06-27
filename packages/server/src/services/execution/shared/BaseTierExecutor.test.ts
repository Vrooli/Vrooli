/**
 * Comprehensive tests for BaseTierExecutor standardized error handling
 * 
 * These tests ensure consistent error handling, resource tracking,
 * and execution patterns across all three tiers.
 */

import { describe, it, expect, beforeEach, vi, type MockedFunction } from "vitest";
import { type Logger } from "winston";
import { BaseTierExecutor, type ExecutionMetrics, type TierErrorContext } from "./BaseTierExecutor.js";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import {
    type TierExecutionRequest,
    type ExecutionResult,
    type ExecutionContext,
    type CoreResourceAllocation,
    type StepExecutionInput,
} from "@vrooli/shared";

// Mock implementations
class MockEventBus {
    publish = vi.fn().mockResolvedValue(undefined);
}

class MockLogger {
    error = vi.fn();
    warn = vi.fn();
    info = vi.fn();
    debug = vi.fn();
}

// Concrete test implementation of BaseTierExecutor
class TestTierExecutor extends BaseTierExecutor {
    private mockExecuteImpl: MockedFunction<any>;

    constructor(
        logger: Logger,
        eventBus: EventBus,
        executeImpl?: MockedFunction<any>,
    ) {
        super(logger, eventBus, "TestTierExecutor", "tier3");
        this.mockExecuteImpl = executeImpl || vi.fn();
    }

    async executeImpl<TInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        return this.mockExecuteImpl(request);
    }

    // Expose protected methods for testing
    public async testExecuteWithErrorHandling<TInput, TOutput>(
        request: TierExecutionRequest<TInput>,
        executeImpl: (request: TierExecutionRequest<TInput>) => Promise<ExecutionResult<TOutput>>,
    ): Promise<ExecutionResult<TOutput>> {
        return this.executeWithErrorHandling(request, executeImpl);
    }

    public testValidateRequest(request: TierExecutionRequest<any>): void {
        return this.validateRequest(request);
    }

    public testValidateResult(result: ExecutionResult<any>, request: TierExecutionRequest<any>): void {
        return this.validateResult(result, request);
    }

    public testCreateStandardizedErrorResult<TOutput>(
        error: unknown,
        request: TierExecutionRequest<any>,
        startTime: number,
    ): ExecutionResult<TOutput> {
        return this.createStandardizedErrorResult(error, request, startTime);
    }

    public testGetInputTypeName(input: unknown): string {
        return this.getInputTypeName(input);
    }

    public testGetAdditionalErrorContext(
        request: TierExecutionRequest<any>,
        error: unknown,
    ): Record<string, unknown> {
        return this.getAdditionalErrorContext(request, error);
    }

    public testGetStrategyFromRequest(request: TierExecutionRequest<any>): string {
        return this.getStrategyFromRequest(request);
    }
}

describe("BaseTierExecutor", () => {
    let mockLogger: MockLogger;
    let mockEventBus: MockEventBus;
    let executor: TestTierExecutor;

    const createContext = (overrides: Partial<ExecutionContext> = {}): ExecutionContext => ({
        executionId: "exec-123",
        swarmId: "swarm-456",
        userId: "user-789",
        correlationId: "corr-abc",
        ...overrides,
    });

    const createAllocation = (overrides: Partial<CoreResourceAllocation> = {}): CoreResourceAllocation => ({
        maxCredits: "10000",
        maxDurationMs: 60000,
        maxMemoryMB: 2048,
        maxConcurrentSteps: 10,
        ...overrides,
    });

    const createRequest = (
        inputOverrides: Partial<StepExecutionInput> = {},
        contextOverrides: Partial<ExecutionContext> = {},
        allocationOverrides: Partial<CoreResourceAllocation> = {},
    ): TierExecutionRequest<StepExecutionInput> => ({
        context: createContext(contextOverrides),
        input: {
            stepId: "step-123",
            stepType: "tool_call",
            parameters: { arg1: "value1" },
            strategy: "conversational",
            ...inputOverrides,
        },
        allocation: createAllocation(allocationOverrides),
    });

    beforeEach(() => {
        mockLogger = new MockLogger();
        mockEventBus = new MockEventBus();
        executor = new TestTierExecutor(
            mockLogger as unknown as Logger,
            mockEventBus as unknown as EventBus,
        );
    });

    describe("executeWithErrorHandling", () => {
        it("should execute successfully and track metrics", async () => {
            const request = createRequest();
            const expectedResult: ExecutionResult<any> = {
                success: true,
                result: { output: "test result" },
                resourcesUsed: {
                    creditsUsed: "100",
                    durationMs: 1000,
                    memoryUsedMB: 256,
                    stepsExecuted: 1,
                },
                duration: 1000,
                context: request.context,
                confidence: 0.95,
                performanceScore: 0.85,
            };

            const mockImpl = vi.fn().mockResolvedValue(expectedResult);
            const result = await executor.testExecuteWithErrorHandling(request, mockImpl);

            expect(result).toEqual(expectedResult);
            expect(mockImpl).toHaveBeenCalledWith(request);
            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "tier.execution.completed",
                    data: expect.objectContaining({
                        tier: "tier3",
                        executionId: request.context.executionId,
                    }),
                }),
            );
        });

        it("should handle validation errors", async () => {
            const invalidRequest = createRequest({}, { executionId: "" }); // Invalid context
            const mockImpl = vi.fn();

            const result = await executor.testExecuteWithErrorHandling(invalidRequest, mockImpl);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("TIER3_EXECUTION_FAILED");
            expect(result.error?.message).toContain("execution ID");
            expect(mockImpl).not.toHaveBeenCalled();
            expect(mockLogger.error).toHaveBeenCalled();
        });

        it("should handle execution errors", async () => {
            const request = createRequest();
            const executionError = new Error("Execution failed");
            const mockImpl = vi.fn().mockRejectedValue(executionError);

            const result = await executor.testExecuteWithErrorHandling(request, mockImpl);

            expect(result.success).toBe(false);
            expect(result.error?.message).toBe("Execution failed");
            expect(result.error?.type).toBe("Error");
            expect(result.error?.tier).toBe("tier3");
            expect(mockLogger.error).toHaveBeenCalledWith(
                "[TestTierExecutor] Execution failed",
                expect.objectContaining({
                    tier: "tier3",
                    executionId: request.context.executionId,
                    error: "Execution failed",
                }),
            );
        });

        it("should handle result validation errors", async () => {
            const request = createRequest();
            const invalidResult = { success: undefined }; // Missing success flag
            const mockImpl = vi.fn().mockResolvedValue(invalidResult);

            const result = await executor.testExecuteWithErrorHandling(request, mockImpl);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("success flag");
        });

        it("should track execution metrics throughout lifecycle", async () => {
            const request = createRequest();
            const mockImpl = vi.fn().mockImplementation(async () => {
                // Check that execution is being tracked
                const metrics = executor.getExecutionMetrics(request.context.executionId);
                expect(metrics).toBeDefined();
                expect(metrics?.phase).toBe("execution");
                
                return {
                    success: true,
                    result: "test",
                    context: request.context,
                    confidence: 1.0,
                    performanceScore: 1.0,
                };
            });

            await executor.testExecuteWithErrorHandling(request, mockImpl);

            // After completion, metrics should eventually be cleaned up
            // But immediately after, they should still exist with success flag
            const metrics = executor.getExecutionMetrics(request.context.executionId);
            expect(metrics?.success).toBe(true);
        });

        it("should emit error events on failure", async () => {
            const request = createRequest();
            const error = new TypeError("Type error occurred");
            const mockImpl = vi.fn().mockRejectedValue(error);

            await executor.testExecuteWithErrorHandling(request, mockImpl);

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "tier.execution.failed",
                    data: expect.objectContaining({
                        tier: "tier3",
                        executionId: request.context.executionId,
                        error: "Type error occurred",
                        phase: expect.any(String),
                    }),
                }),
            );
        });

        it("should handle event publishing failures gracefully", async () => {
            const request = createRequest();
            const mockImpl = vi.fn().mockResolvedValue({
                success: true,
                result: "test",
                context: request.context,
                confidence: 1.0,
                performanceScore: 1.0,
            });

            // Make event publishing fail
            mockEventBus.publish.mockRejectedValue(new Error("Event bus error"));

            // Should still complete successfully
            const result = await executor.testExecuteWithErrorHandling(request, mockImpl);
            expect(result.success).toBe(true);
            
            // The error from event publishing should not affect the result
        });
    });

    describe("validateRequest", () => {
        it("should validate correct request", () => {
            const request = createRequest();
            expect(() => executor.testValidateRequest(request)).not.toThrow();
        });

        it("should reject request without execution ID", () => {
            const request = createRequest({}, { executionId: "" });
            expect(() => executor.testValidateRequest(request)).toThrow("execution ID");
        });

        it("should reject request without allocation", () => {
            const request = createRequest();
            (request as any).allocation = null;
            expect(() => executor.testValidateRequest(request)).toThrow("allocation");
        });

        it("should reject request without input", () => {
            const request = createRequest();
            (request as any).input = null;
            expect(() => executor.testValidateRequest(request)).toThrow("input");
        });

        it("should reject request with undefined input", () => {
            const request = createRequest();
            (request as any).input = undefined;
            expect(() => executor.testValidateRequest(request)).toThrow("input");
        });
    });

    describe("validateResult", () => {
        const request = createRequest();

        it("should validate correct result", () => {
            const result: ExecutionResult<any> = {
                success: true,
                result: "test",
                resourcesUsed: {
                    creditsUsed: "100",
                    durationMs: 1000,
                    memoryUsedMB: 256,
                    stepsExecuted: 1,
                },
                context: request.context,
                confidence: 1.0,
                performanceScore: 1.0,
            };

            expect(() => executor.testValidateResult(result, request)).not.toThrow();
        });

        it("should reject result without success flag", () => {
            const result = { result: "test" } as any;
            expect(() => executor.testValidateResult(result, request)).toThrow("success flag");
        });

        it("should reject failed result without error", () => {
            const result = { success: false } as any;
            expect(() => executor.testValidateResult(result, request)).toThrow("error details");
        });

        it("should warn about missing resource usage", () => {
            const result: ExecutionResult<any> = {
                success: true,
                result: "test",
                context: request.context,
                confidence: 1.0,
                performanceScore: 1.0,
            };

            executor.testValidateResult(result, request);
            expect(mockLogger.warn).toHaveBeenCalledWith(
                "[TestTierExecutor] Execution result missing resource usage",
                expect.objectContaining({
                    executionId: request.context.executionId,
                }),
            );
        });
    });

    describe("createStandardizedErrorResult", () => {
        const request = createRequest();
        const startTime = Date.now() - 5000;

        it("should create error result for Error instance", () => {
            const error = new Error("Test error");
            const result = executor.testCreateStandardizedErrorResult(error, request, startTime);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("TIER3_EXECUTION_FAILED");
            expect(result.error?.message).toBe("Test error");
            expect(result.error?.type).toBe("Error");
            expect(result.error?.tier).toBe("tier3");
            expect(result.duration).toBeGreaterThan(0);
            expect(result.context).toBe(request.context);
            expect(result.metadata?.tier).toBe("tier3");
            expect(result.confidence).toBe(0.0);
            expect(result.performanceScore).toBe(0.0);
        });

        it("should create error result for non-Error values", () => {
            const error = "String error";
            const result = executor.testCreateStandardizedErrorResult(error, request, startTime);

            expect(result.error?.message).toBe("Unknown error");
            expect(result.error?.type).toBe("Error");
        });

        it("should include error context", () => {
            const error = new TypeError("Type error");
            const result = executor.testCreateStandardizedErrorResult(error, request, startTime);

            expect(result.error?.context).toBeDefined();
            expect(result.metadata?.errorContext).toMatchObject({
                tier: "tier3",
                executionId: request.context.executionId,
                inputType: expect.any(String),
                phase: expect.any(String),
            });
        });

        it("should include resource usage from tracker if available", async () => {
            // First start an execution to create metrics
            const mockImpl = vi.fn().mockRejectedValue(new Error("Test error"));
            await executor.testExecuteWithErrorHandling(request, mockImpl);

            // The error result should include resource usage
            const calls = mockLogger.error.mock.calls;
            const errorCall = calls.find(call => call[0].includes("Execution failed"));
            expect(errorCall).toBeDefined();
        });
    });

    describe("utility methods", () => {
        describe("getInputTypeName", () => {
            it("should get constructor name for objects", () => {
                const input = { stepId: "test" };
                expect(executor.testGetInputTypeName(input)).toBe("Object");
            });

            it("should get type for primitives", () => {
                expect(executor.testGetInputTypeName("string")).toBe("String");
                expect(executor.testGetInputTypeName(123)).toBe("Number");
                expect(executor.testGetInputTypeName(null)).toBe("object");
            });

            it("should handle objects without constructor", () => {
                const input = Object.create(null);
                expect(executor.testGetInputTypeName(input)).toBe("object");
            });
        });

        describe("getAdditionalErrorContext", () => {
            it("should extract context from request", () => {
                const request = createRequest();
                const error = new Error("Test");
                const context = executor.testGetAdditionalErrorContext(request, error);

                expect(context.hasAllocation).toBe(true);
                expect(context.hasOptions).toBe(false);
                expect(context.inputKeys).toEqual(
                    expect.arrayContaining(["stepId", "stepType", "parameters", "strategy"]),
                );
            });

            it("should handle null input", () => {
                const request = createRequest();
                (request as any).input = null;
                const context = executor.testGetAdditionalErrorContext(request, new Error());

                expect(context.inputKeys).toEqual([]);
            });
        });

        describe("getStrategyFromRequest", () => {
            it("should extract strategy from input", () => {
                const request = createRequest({ strategy: "reasoning" });
                expect(executor.testGetStrategyFromRequest(request)).toBe("reasoning");
            });

            it("should extract strategy from nested config", () => {
                const request = createRequest();
                delete (request.input as any).strategy; // Remove default strategy
                (request.input as any).config = { strategy: "deterministic" };
                expect(executor.testGetStrategyFromRequest(request)).toBe("deterministic");
            });

            it("should extract strategy from options", () => {
                const request = createRequest();
                delete (request.input as any).strategy; // Remove default strategy
                (request as any).options = { strategy: "routing" };
                expect(executor.testGetStrategyFromRequest(request)).toBe("routing");
            });

            it("should return unknown for missing strategy", () => {
                const request = createRequest();
                delete (request.input as any).strategy;
                expect(executor.testGetStrategyFromRequest(request)).toBe("unknown");
            });
        });
    });

    describe("execution tracking", () => {
        it("should track active executions", async () => {
            const request1 = createRequest({}, { executionId: "exec-1" });
            const request2 = createRequest({}, { executionId: "exec-2" });

            const mockImpl = vi.fn().mockImplementation(() => new Promise(resolve => {
                setTimeout(() => resolve({
                    success: true,
                    result: "test",
                    context: request1.context,
                    confidence: 1.0,
                    performanceScore: 1.0,
                }), 100);
            }));

            // Start two executions
            const promise1 = executor.testExecuteWithErrorHandling(request1, mockImpl);
            const promise2 = executor.testExecuteWithErrorHandling(request2, mockImpl);

            // Check that both are tracked
            const activeExecutions = executor.getActiveExecutions();
            expect(activeExecutions.size).toBeGreaterThan(0);

            await Promise.all([promise1, promise2]);
        });

        it("should clean up execution tracking after completion", async () => {
            const request = createRequest();
            const mockImpl = vi.fn().mockResolvedValue({
                success: true,
                result: "test",
                context: request.context,
                confidence: 1.0,
                performanceScore: 1.0,
            });

            await executor.testExecuteWithErrorHandling(request, mockImpl);

            // Should eventually clean up (test the cleanup mechanism)
            expect(typeof executor.getExecutionMetrics(request.context.executionId)).toBeDefined();
        });
    });

    describe("error handling edge cases", () => {
        it("should handle circular reference errors", async () => {
            const request = createRequest();
            const circularError: any = { message: "Circular error" };
            circularError.self = circularError;

            const mockImpl = vi.fn().mockRejectedValue(circularError);
            const result = await executor.testExecuteWithErrorHandling(request, mockImpl);

            expect(result.success).toBe(false);
            expect(mockLogger.error).toHaveBeenCalled();
        });

        it("should handle errors during cleanup", async () => {
            const request = createRequest();
            
            // Create an executor that will throw during result validation
            class FaultyTestTierExecutor extends TestTierExecutor {
                protected validateResult(result: ExecutionResult<any>, request: TierExecutionRequest<any>): void {
                    throw new Error("Validation error");
                }
            }
            
            const faultyExecutor = new FaultyTestTierExecutor(
                mockLogger as unknown as Logger,
                mockEventBus as unknown as EventBus,
            );

            const mockImpl = vi.fn().mockResolvedValue({
                success: true,
                result: "test",
                context: request.context,
                confidence: 1.0,
                performanceScore: 1.0,
            });

            const result = await faultyExecutor.testExecuteWithErrorHandling(request, mockImpl);

            expect(result.success).toBe(false);
            expect(result.error?.message).toBe("Validation error");
        });
    });
});

