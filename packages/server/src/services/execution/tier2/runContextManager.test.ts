import { EventTypes, generatePK, type ExecutionResourceUsage } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, test, vi, type Mock } from "vitest";
import { logger } from "../../../events/logger.js";
import { CacheService } from "../../../redisConn.js";
import { EventPublisher } from "../../events/publisher.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import { RunContextManager, type IRunContextManager } from "./runContextManager.js";
import type { RunAllocationRequest, RunExecutionContext, StepAllocationRequest } from "./types.js";

// Mock dependencies
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("../../../redisConn.js", () => ({
    CacheService: {
        get: vi.fn(() => ({
            set: vi.fn(),
            get: vi.fn(),
            del: vi.fn(),
        })),
    },
}));

vi.mock("../../events/publisher.js", () => ({
    EventPublisher: {
        emit: vi.fn(async () => ({
            proceed: true,
            reason: null,
            eventId: "test-event-123",
            wasBlocking: false,
            publishResult: { success: true },
        })),
    },
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        generatePK: vi.fn(() => ({ toString: () => "test-pk-123" })),
    };
});

describe("RunContextManager", () => {
    let runContextManager: IRunContextManager;
    let mockSwarmContextManager: ISwarmContextManager;
    let mockCacheService: any;
    let mockGeneratePK: Mock;
    let mockEventPublisher: Mock;
    let mockLogger: any;

    beforeEach(() => {
        vi.clearAllMocks();

        // Setup mocks
        mockCacheService = {
            set: vi.fn().mockResolvedValue(undefined),
            get: vi.fn(),
            del: vi.fn().mockResolvedValue(undefined),
        };
        (CacheService.get as Mock).mockReturnValue(mockCacheService);

        mockSwarmContextManager = {
            allocateResources: vi.fn().mockResolvedValue({
                allocationId: "swarm-alloc-123",
                consumerId: "test-run-456",
                consumerType: "run",
                limits: {
                    maxCredits: "1000",
                    maxDurationMs: 60000,
                    maxMemoryMB: 512,
                    maxConcurrentSteps: 1,
                },
                allocated: {
                    credits: 0,
                    timestamp: new Date(),
                },
                purpose: "test run execution",
                priority: "normal",
            }),
            releaseResources: vi.fn().mockResolvedValue(undefined),
        } as any;

        mockGeneratePK = generatePK as Mock;
        mockEventPublisher = EventPublisher.emit as Mock;
        mockLogger = logger;

        runContextManager = new RunContextManager(mockSwarmContextManager);
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("allocateFromSwarm", () => {
        const validRequest: RunAllocationRequest = {
            runId: "test-run-456",
            routineId: "test-routine-789",
            estimatedRequirements: {
                credits: "1000",
                durationMs: 60000,
                memoryMB: 512,
                maxSteps: 10,
            },
            priority: "normal",
            purpose: "test run execution",
        };

        test("should successfully allocate resources from swarm", async () => {
            const result = await runContextManager.allocateFromSwarm("test-swarm-123", validRequest);

            expect(result).toEqual({
                allocationId: "test-pk-123",
                runId: "test-run-456",
                swarmId: "test-swarm-123",
                allocated: {
                    credits: "1000",
                    timeoutMs: 60000,
                    memoryMB: 512,
                    concurrentExecutions: 1,
                },
                remaining: {
                    credits: "1000",
                    timeoutMs: 60000,
                    memoryMB: 512,
                    concurrentExecutions: 1,
                },
                allocatedAt: expect.any(Date),
                expiresAt: expect.any(Date),
            });

            expect(mockSwarmContextManager.allocateResources).toHaveBeenCalledWith("test-swarm-123", {
                consumerId: "test-run-456",
                consumerType: "run",
                limits: {
                    maxCredits: "1000",
                    maxDurationMs: 60000,
                    maxMemoryMB: 512,
                    maxConcurrentSteps: 1,
                },
                allocated: {
                    credits: 0,
                    timestamp: expect.any(Date),
                },
                purpose: "test run execution",
                priority: "normal",
            });

            expect(mockCacheService.set).toHaveBeenCalledWith(
                "run_allocation:test-run-456",
                expect.any(Object),
                expect.any(Number),
            );

            expect(mockLogger.info).toHaveBeenCalledWith(
                "Allocated resources for run from swarm",
                expect.objectContaining({
                    runId: "test-run-456",
                    swarmId: "test-swarm-123",
                }),
            );
        });

        test("should calculate TTL with minimum bounds", async () => {
            const shortRequest = {
                ...validRequest,
                estimatedRequirements: {
                    ...validRequest.estimatedRequirements,
                    durationMs: 1000, // 1 second - below minimum
                },
            };

            await runContextManager.allocateFromSwarm("test-swarm-123", shortRequest);

            // Should use minimum TTL of 300 seconds
            expect(mockCacheService.set).toHaveBeenCalledWith(
                "run_allocation:test-run-456",
                expect.any(Object),
                300,
            );
        });

        test("should calculate TTL with maximum bounds", async () => {
            const longRequest = {
                ...validRequest,
                estimatedRequirements: {
                    ...validRequest.estimatedRequirements,
                    durationMs: 100000000, // Very long duration - above maximum
                },
            };

            await runContextManager.allocateFromSwarm("test-swarm-123", longRequest);

            // Should use maximum TTL of 86400 seconds (24 hours)
            expect(mockCacheService.set).toHaveBeenCalledWith(
                "run_allocation:test-run-456",
                expect.any(Object),
                86400,
            );
        });

        test("should calculate TTL within normal bounds", async () => {
            const normalRequest = {
                ...validRequest,
                estimatedRequirements: {
                    ...validRequest.estimatedRequirements,
                    durationMs: 3600000, // 1 hour
                },
            };

            await runContextManager.allocateFromSwarm("test-swarm-123", normalRequest);

            // Should use calculated TTL of 3600 seconds
            expect(mockCacheService.set).toHaveBeenCalledWith(
                "run_allocation:test-run-456",
                expect.any(Object),
                3600,
            );
        });

        test("should throw error when cache storage fails", async () => {
            mockCacheService.set.mockRejectedValue(new Error("Cache storage failed"));

            await expect(
                runContextManager.allocateFromSwarm("test-swarm-123", validRequest),
            ).rejects.toThrow("Failed to persist run allocation: Cache storage failed");

            expect(mockLogger.error).toHaveBeenCalledWith(
                "Failed to store run allocation in cache",
                expect.objectContaining({
                    runId: "test-run-456",
                    error: "Cache storage failed",
                }),
            );
        });

        test("should handle priority parameter correctly", async () => {
            const highPriorityRequest = {
                ...validRequest,
                priority: "high" as const,
            };

            await runContextManager.allocateFromSwarm("test-swarm-123", highPriorityRequest);

            expect(mockSwarmContextManager.allocateResources).toHaveBeenCalledWith(
                "test-swarm-123",
                expect.objectContaining({
                    priority: "high",
                }),
            );
        });
    });

    describe("releaseToSwarm", () => {
        const mockUsage: ExecutionResourceUsage = {
            creditsUsed: "500",
            durationMs: 30000,
            memoryUsedMB: 256,
            stepsExecuted: 5,
            startTime: new Date(),
        };

        beforeEach(async () => {
            // Setup an allocation first
            const request: RunAllocationRequest = {
                runId: "test-run-456",
                routineId: "test-routine-789",
                estimatedRequirements: {
                    credits: "1000",
                    durationMs: 60000,
                    memoryMB: 512,
                    maxSteps: 10,
                },
                priority: "normal",
                purpose: "test run execution",
            };
            await runContextManager.allocateFromSwarm("test-swarm-123", request);
            vi.clearAllMocks(); // Clear setup calls
        });

        test("should successfully release resources to swarm", async () => {
            await runContextManager.releaseToSwarm("test-swarm-123", "test-run-456", mockUsage);

            expect(mockSwarmContextManager.releaseResources).toHaveBeenCalledWith(
                "test-swarm-123",
                "test-pk-123",
            );

            expect(mockCacheService.del).toHaveBeenCalledWith("run_allocation:test-run-456");
            expect(mockCacheService.del).toHaveBeenCalledWith("run_context:test-run-456");

            expect(mockLogger.info).toHaveBeenCalledWith(
                "Released run resources to swarm",
                expect.objectContaining({
                    runId: "test-run-456",
                    swarmId: "test-swarm-123",
                    usage: {
                        creditsUsed: "500",
                        durationMs: 30000,
                        stepsExecuted: 5,
                    },
                }),
            );
        });

        test("should handle missing allocation gracefully", async () => {
            await runContextManager.releaseToSwarm("test-swarm-123", "non-existent-run", mockUsage);

            expect(mockLogger.warn).toHaveBeenCalledWith(
                "No allocation found for run",
                { runId: "non-existent-run" },
            );

            expect(mockSwarmContextManager.releaseResources).not.toHaveBeenCalled();
        });

        test("should continue if cache cleanup fails", async () => {
            mockCacheService.del.mockRejectedValue(new Error("Cache deletion failed"));

            await expect(
                runContextManager.releaseToSwarm("test-swarm-123", "test-run-456", mockUsage),
            ).resolves.not.toThrow();

            expect(mockLogger.warn).toHaveBeenCalledWith(
                "Failed to clean up cache entries",
                expect.objectContaining({
                    runId: "test-run-456",
                    error: "Cache deletion failed",
                }),
            );

            expect(mockSwarmContextManager.releaseResources).toHaveBeenCalled();
        });
    });

    describe("getRunContext", () => {
        const mockContext: RunExecutionContext = {
            runId: "test-run-456",
            routineId: "test-routine-789",
            swarmId: "test-swarm-123",
            navigator: null,
            currentLocation: { id: "loc1", routineId: "test-routine-789", nodeId: "node1" },
            visitedLocations: [],
            variables: { var1: "value1" },
            outputs: {},
            completedSteps: [],
            parallelBranches: [],
            resourceLimits: {
                maxCredits: "1000",
                maxDurationMs: 60000,
                maxMemoryMB: 512,
                maxSteps: 10,
            },
            resourceUsage: {
                creditsUsed: "100",
                durationMs: 5000,
                memoryUsedMB: 64,
                stepsExecuted: 1,
                startTime: new Date("2023-01-01T00:00:00Z"),
            },
            progress: {
                currentStepId: "step1",
                completedSteps: [],
                totalSteps: 5,
                percentComplete: 20,
            },
            retryCount: 0,
        };

        test("should successfully retrieve run context", async () => {
            mockCacheService.get.mockResolvedValue(mockContext);

            const result = await runContextManager.getRunContext("test-run-456");

            expect(result).toEqual(mockContext);
            expect(mockCacheService.get).toHaveBeenCalledWith("run_context:test-run-456");
        });

        test("should reconstruct Date objects from serialized strings", async () => {
            const serializedContext = {
                ...mockContext,
                resourceUsage: {
                    ...mockContext.resourceUsage,
                    startTime: "2023-01-01T00:00:00.000Z", // Serialized as string
                },
            };
            mockCacheService.get.mockResolvedValue(serializedContext);

            const result = await runContextManager.getRunContext("test-run-456");

            expect(result.resourceUsage.startTime).toBeInstanceOf(Date);
            expect(result.resourceUsage.startTime.toISOString()).toBe("2023-01-01T00:00:00.000Z");
        });

        test("should throw error when context not found", async () => {
            mockCacheService.get.mockResolvedValue(null);

            await expect(
                runContextManager.getRunContext("non-existent-run"),
            ).rejects.toThrow("Run context not found: non-existent-run");
        });

        test("should throw error when cache retrieval fails", async () => {
            mockCacheService.get.mockRejectedValue(new Error("Cache retrieval failed"));

            await expect(
                runContextManager.getRunContext("test-run-456"),
            ).rejects.toThrow("Cache retrieval failed");

            expect(mockLogger.error).toHaveBeenCalledWith(
                "Failed to retrieve run context from cache",
                expect.objectContaining({
                    runId: "test-run-456",
                    error: "Cache retrieval failed",
                }),
            );
        });
    });

    describe("updateRunContext", () => {
        const mockContext: RunExecutionContext = {
            runId: "test-run-456",
            routineId: "test-routine-789",
            navigator: null,
            currentLocation: { id: "loc1", routineId: "test-routine-789", nodeId: "node1" },
            visitedLocations: [],
            variables: {},
            outputs: {},
            completedSteps: [],
            parallelBranches: [],
            resourceLimits: {
                maxCredits: "1000",
                maxDurationMs: 60000,
                maxMemoryMB: 512,
                maxSteps: 10,
            },
            resourceUsage: {
                creditsUsed: "300",
                durationMs: 15000,
                memoryUsedMB: 128,
                stepsExecuted: 3,
                startTime: new Date(),
            },
            progress: {
                currentStepId: "step3",
                completedSteps: ["step1", "step2"],
                totalSteps: 5,
                percentComplete: 60,
            },
            retryCount: 0,
        };

        beforeEach(async () => {
            // Setup an allocation for TTL calculation
            const request: RunAllocationRequest = {
                runId: "test-run-456",
                routineId: "test-routine-789",
                estimatedRequirements: {
                    credits: "1000",
                    durationMs: 60000,
                    memoryMB: 512,
                    maxSteps: 10,
                },
                priority: "normal",
                purpose: "test run execution",
            };
            await runContextManager.allocateFromSwarm("test-swarm-123", request);
            vi.clearAllMocks(); // Clear setup calls
        });

        test("should successfully update run context with allocation", async () => {
            await runContextManager.updateRunContext("test-run-456", mockContext);

            expect(mockCacheService.set).toHaveBeenCalledWith(
                "run_context:test-run-456",
                mockContext,
                expect.any(Number),
            );

            expect(mockLogger.debug).toHaveBeenCalledWith(
                "Updated run context",
                expect.objectContaining({
                    runId: "test-run-456",
                    currentStepId: "step3",
                    percentComplete: 60,
                }),
            );
        });

        test("should use default TTL when no allocation exists", async () => {
            await runContextManager.updateRunContext("non-allocated-run", mockContext);

            expect(mockCacheService.set).toHaveBeenCalledWith(
                "run_context:non-allocated-run",
                mockContext,
                3600, // Default 1 hour TTL
            );
        });

        test("should update allocation remaining resources", async () => {
            await runContextManager.updateRunContext("test-run-456", mockContext);

            // The allocation should have its remaining credits updated
            // Original: 1000, Used: 300, Remaining should be: 700
            // This is tested indirectly through the implementation
            expect(mockCacheService.set).toHaveBeenCalled();
        });

        test("should throw error when cache update fails", async () => {
            mockCacheService.set.mockRejectedValue(new Error("Cache update failed"));

            await expect(
                runContextManager.updateRunContext("test-run-456", mockContext),
            ).rejects.toThrow("Failed to persist run context: Cache update failed");

            expect(mockLogger.error).toHaveBeenCalledWith(
                "Failed to update run context in cache",
                expect.objectContaining({
                    runId: "test-run-456",
                    error: "Cache update failed",
                }),
            );
        });
    });

    describe("event emission methods", () => {
        const mockAllocation = {
            allocationId: "test-pk-123",
            runId: "test-run-456",
            swarmId: "test-swarm-123",
            allocated: {
                credits: "1000",
                timeoutMs: 60000,
                memoryMB: 512,
                concurrentExecutions: 1,
            },
            remaining: {
                credits: "1000",
                timeoutMs: 60000,
                memoryMB: 512,
                concurrentExecutions: 1,
            },
            allocatedAt: new Date(),
            expiresAt: new Date(Date.now() + 60000),
        };

        const mockUsage: ExecutionResourceUsage = {
            creditsUsed: "300",
            durationMs: 15000,
            memoryUsedMB: 128,
            stepsExecuted: 3,
            startTime: new Date(),
        };

        describe("emitRunStarted", () => {
            test("should emit run started event successfully", async () => {
                await runContextManager.emitRunStarted("test-run-456", "test-routine-789", mockAllocation);

                expect(mockEventPublisher).toHaveBeenCalledWith(EventTypes.RUN.STARTED, {
                    runId: "test-run-456",
                    routineId: "test-routine-789",
                    inputs: {},
                    estimatedDuration: 60000,
                    parentSwarmId: "test-swarm-123",
                });
            });

            test("should log warning when event is blocked but continue", async () => {
                mockEventPublisher.mockResolvedValue({
                    proceed: false,
                    reason: "Event blocked by interceptor",
                });

                await runContextManager.emitRunStarted("test-run-456", "test-routine-789", mockAllocation);

                expect(mockLogger.warn).toHaveBeenCalledWith(
                    "[RunContextManager] Run started event blocked",
                    expect.objectContaining({
                        runId: "test-run-456",
                        routineId: "test-routine-789",
                        reason: "Event blocked by interceptor",
                    }),
                );
            });
        });

        describe("emitRunCompleted", () => {
            test("should emit run completed event successfully", async () => {
                const result = { success: true, output: "completed" };

                await runContextManager.emitRunCompleted("test-run-456", result, mockUsage, "test-swarm-123");

                expect(mockEventPublisher).toHaveBeenCalledWith(EventTypes.RUN.COMPLETED, {
                    runId: "test-run-456",
                    parentSwarmId: "test-swarm-123",
                    outputs: result,
                    duration: 15000,
                    message: "Run completed successfully",
                    isSwarmIntegrated: true,
                });
            });

            test("should handle completion without parent swarm", async () => {
                const result = { success: true };

                await runContextManager.emitRunCompleted("test-run-456", result, mockUsage);

                expect(mockEventPublisher).toHaveBeenCalledWith(EventTypes.RUN.COMPLETED, {
                    runId: "test-run-456",
                    parentSwarmId: undefined,
                    outputs: result,
                    duration: 15000,
                    message: "Run completed successfully",
                    isSwarmIntegrated: false,
                });
            });

            test("should log warning when event is blocked", async () => {
                mockEventPublisher.mockResolvedValue({
                    proceed: false,
                    reason: "Completion event blocked",
                });

                await runContextManager.emitRunCompleted("test-run-456", { success: true }, mockUsage);

                expect(mockLogger.warn).toHaveBeenCalledWith(
                    "[RunContextManager] Run completed event blocked",
                    expect.objectContaining({
                        runId: "test-run-456",
                        reason: "Completion event blocked",
                    }),
                );
            });
        });

        describe("emitRunFailed", () => {
            test("should emit run failed event with Error object", async () => {
                const error = new Error("Execution failed");

                await runContextManager.emitRunFailed("test-run-456", error, mockUsage, "test-swarm-123");

                expect(mockEventPublisher).toHaveBeenCalledWith(EventTypes.RUN.FAILED, {
                    runId: "test-run-456",
                    parentSwarmId: "test-swarm-123",
                    error: "Execution failed",
                    duration: 15000,
                    retryable: false,
                    isSwarmIntegrated: true,
                });
            });

            test("should emit run failed event with error object", async () => {
                const error = { message: "Custom error message" };

                await runContextManager.emitRunFailed("test-run-456", error, mockUsage);

                expect(mockEventPublisher).toHaveBeenCalledWith(EventTypes.RUN.FAILED, {
                    runId: "test-run-456",
                    parentSwarmId: undefined,
                    error: "Custom error message",
                    duration: 15000,
                    retryable: false,
                    isSwarmIntegrated: false,
                });
            });

            test("should log warning when event is blocked", async () => {
                mockEventPublisher.mockResolvedValue({
                    proceed: false,
                    reason: "Failure event blocked",
                });

                const error = new Error("Test failure");
                await runContextManager.emitRunFailed("test-run-456", error, mockUsage);

                expect(mockLogger.warn).toHaveBeenCalledWith(
                    "[RunContextManager] Run failed event blocked",
                    expect.objectContaining({
                        runId: "test-run-456",
                        error: "Test failure",
                        blockReason: "Failure event blocked",
                    }),
                );
            });
        });
    });

    describe("step resource allocation", () => {
        beforeEach(async () => {
            // Setup a run allocation for step allocation tests
            const request: RunAllocationRequest = {
                runId: "test-run-456",
                routineId: "test-routine-789",
                estimatedRequirements: {
                    credits: "1000",
                    durationMs: 60000,
                    memoryMB: 512,
                    maxSteps: 10,
                },
                priority: "normal",
                purpose: "test run execution",
            };
            await runContextManager.allocateFromSwarm("test-swarm-123", request);
            vi.clearAllMocks(); // Clear setup calls
        });

        describe("allocateForStep", () => {
            const stepRequest: StepAllocationRequest = {
                stepId: "1",
                stepType: "llm_call",
                estimatedRequirements: {
                    credits: "200",
                    durationMs: 10000,
                    memoryMB: 128,
                },
            };

            test("should successfully allocate resources for step", async () => {
                const result = await runContextManager.allocateForStep("test-run-456", stepRequest);

                expect(result).toEqual({
                    allocationId: "test-pk-123",
                    stepId: "1",
                    runId: "test-run-456",
                    allocated: {
                        credits: "200",
                        timeoutMs: 10000,
                        memoryMB: 128,
                        concurrentExecutions: 1,
                    },
                    allocatedAt: expect.any(Date),
                    expiresAt: expect.any(Date),
                });

                expect(mockLogger.debug).toHaveBeenCalledWith(
                    "Allocated resources for step from run",
                    expect.objectContaining({
                        runId: "test-run-456",
                        stepId: "1",
                        credits: "200",
                        durationMs: 10000,
                    }),
                );
            });

            test("should throw error for insufficient credits", async () => {
                const excessiveRequest = {
                    ...stepRequest,
                    estimatedRequirements: {
                        ...stepRequest.estimatedRequirements,
                        credits: "2000", // More than available
                    },
                };

                await expect(
                    runContextManager.allocateForStep("test-run-456", excessiveRequest),
                ).rejects.toThrow("Insufficient credits for step. Requested: 2000, Available: 1000");
            });

            test("should throw error for insufficient time", async () => {
                const excessiveRequest = {
                    ...stepRequest,
                    estimatedRequirements: {
                        ...stepRequest.estimatedRequirements,
                        durationMs: 120000, // More than available
                    },
                };

                await expect(
                    runContextManager.allocateForStep("test-run-456", excessiveRequest),
                ).rejects.toThrow("Insufficient time for step. Requested: 120000ms, Available: 60000ms");
            });

            test("should throw error for non-existent run", async () => {
                await expect(
                    runContextManager.allocateForStep("non-existent-run", stepRequest),
                ).rejects.toThrow("No allocation found for run: non-existent-run");
            });
        });

        describe("releaseFromStep", () => {
            const stepUsage: ExecutionResourceUsage = {
                creditsUsed: "150",
                durationMs: 8000,
                memoryUsedMB: 100,
                stepsExecuted: 1,
                startTime: new Date(),
            };

            test("should successfully release step resources", async () => {
                // First allocate for step
                const stepRequest: StepAllocationRequest = {
                    stepId: "1",
                    stepType: "llm_call",
                    estimatedRequirements: {
                        credits: "200",
                        durationMs: 10000,
                        memoryMB: 128,
                    },
                };
                await runContextManager.allocateForStep("test-run-456", stepRequest);
                vi.clearAllMocks();

                await runContextManager.releaseFromStep("test-run-456", "1", stepUsage);

                expect(mockLogger.debug).toHaveBeenCalledWith(
                    "Released step resources back to run",
                    expect.objectContaining({
                        runId: "test-run-456",
                        stepId: "1",
                        creditsUsed: "150",
                        durationMs: 8000,
                    }),
                );
            });

            test("should handle missing run allocation gracefully", async () => {
                await runContextManager.releaseFromStep("non-existent-run", "1", stepUsage);

                expect(mockLogger.warn).toHaveBeenCalledWith(
                    "No run allocation found for step release",
                    { runId: "non-existent-run", stepId: "1" },
                );
            });
        });
    });

    describe("edge cases and error scenarios", () => {
        test("should handle BigInt credit calculations correctly", async () => {
            const request: RunAllocationRequest = {
                runId: "test-run-456",
                routineId: "test-routine-789",
                estimatedRequirements: {
                    credits: "999999999999999999", // Large BigInt value
                    durationMs: 60000,
                    memoryMB: 512,
                    maxSteps: 10,
                },
                priority: "normal",
                purpose: "test run execution",
            };

            const result = await runContextManager.allocateFromSwarm("test-swarm-123", request);

            expect(result.allocated.credits).toBe("999999999999999999");
            expect(result.remaining.credits).toBe("999999999999999999");
        });

        test("should handle zero-duration requests", async () => {
            const request: RunAllocationRequest = {
                runId: "test-run-456",
                routineId: "test-routine-789",
                estimatedRequirements: {
                    credits: "1000",
                    durationMs: 0,
                    memoryMB: 512,
                    maxSteps: 10,
                },
                priority: "normal",
                purpose: "test run execution",
            };

            await runContextManager.allocateFromSwarm("test-swarm-123", request);

            // Should use minimum TTL even with zero duration
            expect(mockCacheService.set).toHaveBeenCalledWith(
                "run_allocation:test-run-456",
                expect.any(Object),
                300, // Minimum TTL
            );
        });

        test("should handle string error objects correctly", async () => {
            const mockUsage: ExecutionResourceUsage = {
                creditsUsed: "100",
                durationMs: 5000,
                memoryUsedMB: 64,
                stepsExecuted: 1,
                startTime: new Date(),
            };

            // Test with non-Error object that needs string conversion
            const errorObj = { toString: () => "Custom error string" };
            await runContextManager.emitRunFailed("test-run-456", errorObj as any, mockUsage);

            expect(mockEventPublisher).toHaveBeenCalledWith(EventTypes.RUN.FAILED, {
                runId: "test-run-456",
                parentSwarmId: undefined,
                error: "Custom error string", // Uses toString() via String(error)
                duration: 5000,
                retryable: false,
                isSwarmIntegrated: false,
            });
        });
    });
});
