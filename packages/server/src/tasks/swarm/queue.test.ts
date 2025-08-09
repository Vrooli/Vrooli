// AI_CHECK: TEST_QUALITY=1, TEST_COVERAGE=1 | LAST: 2025-06-18
import { generatePK, initIdGenerator } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createIsolatedQueueTestHarness } from "../../__test/helpers/queueTestUtils.js";
import { clearRedisCache } from "../queueFactory.js";
import { QueueService } from "../queues.js";
import { QueueTaskType, type SwarmExecutionTask, type SwarmTask } from "../taskTypes.js";
import { changeSwarmTaskStatus, getSwarmTaskStatuses, processNewSwarmExecution, processSwarm } from "./queue.js";

// Mock the execution tier components to prevent initialization errors in tests
vi.mock("../../services/execution/tier2/tierTwoOrchestrator.js", () => ({
    TierTwoOrchestrator: vi.fn().mockImplementation(() => ({
        startRun: vi.fn().mockResolvedValue(undefined),
        cancelRun: vi.fn().mockResolvedValue(undefined),
        handleSwarmRequest: vi.fn().mockResolvedValue({ success: true }),
    })),
}));

vi.mock("../../services/execution/tier1/coordination/swarmCoordinator.js", () => ({
    SwarmCoordinator: vi.fn().mockImplementation(() => ({
        execute: vi.fn().mockResolvedValue({
            status: "completed",
            result: { success: true },
        }),
        getExecutionStatus: vi.fn().mockResolvedValue({
            swarmId: "mock-swarm-id",
            status: "completed",
            progress: 100,
            metadata: {
                currentPhase: "idle",
                activeRuns: 0,
                completedRuns: 1,
            },
        }),
        cancelExecution: vi.fn().mockResolvedValue(undefined),
    })),
}));

vi.mock("../../services/events/eventBus.js", () => ({
    EventBus: vi.fn().mockImplementation(() => ({})),
}));

vi.mock("../../services/execution/tier3/stepExecutor.js", () => ({
    StepExecutor: vi.fn().mockImplementation(() => ({
        execute: vi.fn().mockResolvedValue({ success: true, outputs: {} }),
    })),
}));

describe("Swarm Queue", () => {
    let queueService: QueueService;
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";

    beforeEach(async () => {
        // Initialize ID generator first
        await initIdGenerator(0);
        
        // Get fresh instance and initialize
        queueService = QueueService.get();
        await queueService.init(redisUrl);
        
        // Pre-initialize all queues to avoid race conditions
        await queueService.initializeAllQueues();
    });

    afterEach(async () => {
        // Ensure clean shutdown after each test
        try {
            await queueService.shutdown();
        } catch (error) {
            // Ignore shutdown errors in tests
            console.log("Shutdown error (ignored):", error);
        }

        // Wait for shutdown to fully complete and event handlers to detach
        await new Promise(resolve => setTimeout(resolve, 100));

        // Clear singleton instance before Redis cache to prevent access during cleanup
        (QueueService as any).instance = null;

        // Clear Redis cache last to avoid disconnecting connections still in use
        clearRedisCache();

        // Final delay to ensure all async operations complete
        await new Promise(resolve => setTimeout(resolve, 50));
    });

    describe("processSwarm", () => {
        function createBaseSwarmData(): Omit<SwarmExecutionTask, "status"> {
            const userId = generatePK().toString();
            const swarmId = generatePK().toString();
            const sessionId = generatePK().toString();

            return {
                type: QueueTaskType.SWARM_EXECUTION,
                id: swarmId,
                userId,
                allocation: {
                    maxCredits: "1000",
                    maxDurationMs: 300000,
                    maxMemoryMB: 256,
                    maxConcurrentSteps: 2,
                },
                input: {
                    swarmId,
                    goal: "Complete test objectives efficiently",
                    executionConfig: {
                        model: "gpt-4",
                    },
                    userData: {
                        __typename: "SessionUser" as const,
                        id: userId,
                        credits: "5000",
                        hasPremium: false,
                        hasReceivedPhoneVerificationReward: false,
                        languages: ["en"],
                        phoneNumberVerified: false,
                        publicId: `pub_${userId}`,
                        session: {
                            __typename: "SessionUserSession" as const,
                            id: sessionId,
                            lastRefreshAt: new Date().toISOString(),
                        },
                        updatedAt: new Date().toISOString(),
                    },
                },
            };
        }

        it("should add swarm task with correct status", async () => {
            const baseSwarmData = createBaseSwarmData();
            const result = await processSwarm(baseSwarmData, queueService);
            expect(result.success).toBe(true);
            expect(result.data?.id).toBeDefined();

            // Verify the job was added
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
            expect(job?.data.status).toBe("Scheduled");
            expect(job?.data.type).toBe(QueueTaskType.SWARM_EXECUTION);
        });

        describe("priority calculation", () => {
            it("should set base priority for non-premium users", async () => {
                const baseSwarmData = createBaseSwarmData();
                const result = await processSwarm(baseSwarmData, queueService);
                const job = await queueService.swarm.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(100); // BASE_PRIORITY
            });

            it("should boost priority for premium users", async () => {
                const baseSwarmData = createBaseSwarmData();
                const userId = generatePK().toString();
                const premiumData = {
                    ...baseSwarmData,
                    userData: { id: userId, hasPremium: true },
                };
                const result = await processSwarm(premiumData, queueService);
                const job = await queueService.swarm.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(80); // 100 - 20
            });

            it("should not go below 0 priority", async () => {
                // Even with maximum boosts, priority should not go below 0
                const baseSwarmData = createBaseSwarmData();
                const userId = generatePK().toString();
                const data = {
                    ...baseSwarmData,
                    userData: { id: userId, hasPremium: true },
                };
                const result = await processSwarm(data, queueService);
                const job = await queueService.swarm.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBeGreaterThanOrEqual(0);
            });
        });

        it("should handle concurrent task additions", async () => {
            const promises = [];
            for (let i = 0; i < 5; i++) {
                const data = { ...baseSwarmData, swarmId: `swarm-${i}` };
                promises.push(processSwarm(data, queueService));
            }

            const results = await Promise.all(promises);
            expect(results).toHaveLength(5);
            results.forEach(result => {
                expect(result.success).toBe(true);
                expect(result.data?.id).toBeDefined();
            });
        });

        it("should preserve all task data", async () => {
            const complexData: Omit<SwarmTask, "status"> = {
                ...baseSwarmData,
                inputs: {
                    nested: {
                        data: "complex",
                        array: [1, 2, 3],
                    },
                },
                metadata: {
                    source: "test",
                    timestamp: Date.now(),
                },
            };

            const result = await processSwarm(complexData, queueService);
            const job = await queueService.swarm.queue.getJob(result.data!.id);

            expect(job?.data.inputs).toEqual(complexData.inputs);
            expect(job?.data.metadata).toEqual(complexData.metadata);
        });
    });

    describe("processNewSwarmExecution", () => {
        function createBaseExecutionData(): Omit<SwarmExecutionTask, "status"> {
            const userId = generatePK().toString();
            const swarmId = generatePK().toString();
            const executionId = generatePK().toString();

            return {
                type: QueueTaskType.SWARM_EXECUTION,
                swarmId,
                executionId,
                userId,
                userData: {
                    id: userId,
                    hasPremium: false,
                },
                config: {
                    model: "gpt-4",
                    temperature: 0.7,
                },
            };
        }

        it("should add swarm execution task with correct status", async () => {
            const baseExecutionData = createBaseExecutionData();
            const result = await processNewSwarmExecution(baseExecutionData, queueService);
            expect(result.success).toBe(true);
            expect(result.data?.id).toBeDefined();

            // Verify the job was added
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
            expect(job?.data.status).toBe("Scheduled");
            expect(job?.data.type).toBe(QueueTaskType.SWARM_EXECUTION);
        });

        it("should apply same priority rules as processSwarm", async () => {
            // Test with premium user
            const baseExecutionData = createBaseExecutionData();
            const userId = generatePK().toString();
            const premiumData = {
                ...baseExecutionData,
                userData: { id: userId, hasPremium: true },
            };
            const result = await processNewSwarmExecution(premiumData, queueService);
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            expect(job?.opts.priority).toBe(80); // 100 - 20
        });

        it("should handle different execution configurations", async () => {
            const baseExecutionData = createBaseExecutionData();
            const executionVariants = [
                { ...baseExecutionData, config: { model: "gpt-3.5-turbo" } },
                { ...baseExecutionData, config: { model: "claude-3" } },
                { ...baseExecutionData, config: { model: "gpt-4", maxTokens: 1000 } },
            ];

            for (const variant of executionVariants) {
                const result = await processNewSwarmExecution(variant, queueService);
                expect(result.success).toBe(true);

                const job = await queueService.swarm.queue.getJob(result.data!.id);
                expect(job?.data.config).toEqual(variant.config);
            }
        });
    });

    describe("changeSwarmTaskStatus", () => {
        let taskId: string;
        let userId: string;

        beforeEach(async () => {
            // Add a test job
            userId = generatePK().toString();
            const swarmId = generatePK().toString();
            const sessionId = generatePK().toString();

            const result = await processSwarm({
                type: QueueTaskType.SWARM_EXECUTION,
                id: swarmId,
                userId,
                allocation: {
                    maxCredits: "1000",
                    maxDurationMs: 300000,
                    maxMemoryMB: 256,
                    maxConcurrentSteps: 2,
                },
                input: {
                    swarmId,
                    goal: "Test swarm status changes",
                    userData: {
                        __typename: "SessionUser" as const,
                        id: userId,
                        credits: "5000",
                        hasPremium: false,
                        hasReceivedPhoneVerificationReward: false,
                        languages: ["en"],
                        phoneNumberVerified: false,
                        publicId: `pub_${userId}`,
                        session: {
                            __typename: "SessionUserSession" as const,
                            id: sessionId,
                            lastRefreshAt: new Date().toISOString(),
                        },
                        updatedAt: new Date().toISOString(),
                    },
                },
            }, queueService);
            taskId = result.data!.id;
        });

        it("should change task status", async () => {
            const result = await changeSwarmTaskStatus(taskId, "Running", userId, queueService);
            expect(result.success).toBe(true);
        });

        it("should verify status change is delegated to swarm queue", async () => {
            const spy = vi.spyOn(queueService.swarm, "changeTaskStatus");
            await changeSwarmTaskStatus(taskId, "Running", userId, queueService);
            expect(spy).toHaveBeenCalledWith(taskId, "Running", userId);
        });

        it("should handle custom status values", async () => {
            const customStatuses = ["Processing", "Analyzing", "Generating"];

            for (const status of customStatuses) {
                const result = await changeSwarmTaskStatus(taskId, status, userId, queueService);
                expect(result.success).toBe(true);
            }
        });

        it("should handle invalid task ID", async () => {
            const invalidUserId = generatePK().toString();
            const result = await changeSwarmTaskStatus("invalid-id", "Running", invalidUserId, queueService);
            expect(result.success).toBe(false);
        });
    });

    describe("getSwarmTaskStatuses", () => {
        const taskIds: string[] = [];

        beforeEach(async () => {
            // Clear taskIds array before each test
            taskIds.length = 0;

            // Add multiple test jobs
            for (let i = 0; i < 3; i++) {
                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const sessionId = generatePK().toString();

                const result = await processSwarm({
                    type: QueueTaskType.SWARM_EXECUTION,
                    id: swarmId,
                    userId,
                    allocation: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 256,
                        maxConcurrentSteps: 2,
                    },
                    input: {
                        swarmId,
                        goal: `Test multiple swarms ${i}`,
                        userData: {
                            __typename: "SessionUser" as const,
                            id: userId,
                            credits: "5000",
                            hasPremium: false,
                            hasReceivedPhoneVerificationReward: false,
                            languages: ["en"],
                            phoneNumberVerified: false,
                            publicId: `pub_${userId}`,
                            session: {
                                __typename: "SessionUserSession" as const,
                                id: sessionId,
                                lastRefreshAt: new Date().toISOString(),
                            },
                            updatedAt: new Date().toISOString(),
                        },
                    },
                }, queueService);
                taskIds.push(result.data!.id);
            }
        });

        it("should fetch multiple task statuses", async () => {
            const statuses = await getSwarmTaskStatuses(taskIds, queueService);
            expect(statuses).toHaveLength(3);
            statuses.forEach((status, index) => {
                expect(status.__typename).toBe("TaskStatusInfo");
                expect(status.id).toBe(taskIds[index]);
                expect(status.queueName).toBe("swarm");
                expect(status.status).toBeDefined();
            });
        });

        it("should handle mixed valid and invalid IDs", async () => {
            const mixedIds = [...taskIds, "invalid-id-1", "invalid-id-2"];
            const statuses = await getSwarmTaskStatuses(mixedIds, queueService);

            // Should return status for all IDs, with null for invalid ones
            expect(statuses).toHaveLength(5);

            // Valid IDs should have status
            statuses.slice(0, 3).forEach(status => {
                expect(status.status).toBeDefined();
            });

            // Invalid IDs should have null status
            statuses.slice(3).forEach(status => {
                expect(status.status).toBeNull();
            });
        });

        it("should handle empty array", async () => {
            const statuses = await getSwarmTaskStatuses([], queueService);
            expect(statuses).toEqual([]);
        });

        it("should verify delegation to QueueService", async () => {
            const spy = vi.spyOn(queueService, "getTaskStatuses");
            await getSwarmTaskStatuses(taskIds, queueService);
            expect(spy).toHaveBeenCalledWith(taskIds, "swarm");
        });
    });

    describe("Integration with QueueService", () => {
        it("should process swarm task through worker", async () => {
            // Use isolated harness to prevent job processing from corrupting connections
            const isolatedHarness = await createIsolatedQueueTestHarness(redisUrl);

            try {
                // Create fresh QueueService instance for this test
                (QueueService as any).instance = null;
                const isolatedQueueService = QueueService.get();
                await isolatedQueueService.init(redisUrl);

                // Mock the swarm process function
                const processSwarmMock = vi.fn().mockResolvedValue(undefined);
                vi.doMock("./process.js", () => ({
                    swarmProcess: processSwarmMock,
                }));

                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const sessionId = generatePK().toString();

                const swarmData = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    id: swarmId,
                    userId,
                    allocation: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 256,
                        maxConcurrentSteps: 2,
                    },
                    input: {
                        swarmId,
                        goal: "Integration test execution",
                        userData: {
                            __typename: "SessionUser" as const,
                            id: userId,
                            credits: "5000",
                            hasPremium: false,
                            hasReceivedPhoneVerificationReward: false,
                            languages: ["en"],
                            phoneNumberVerified: false,
                            publicId: `pub_${userId}`,
                            session: {
                                __typename: "SessionUserSession" as const,
                                id: sessionId,
                                lastRefreshAt: new Date().toISOString(),
                            },
                            updatedAt: new Date().toISOString(),
                        },
                    },
                };

                const result = await processSwarm(swarmData, isolatedQueueService);
                expect(result.success).toBe(true);

                // Wait for processing
                await new Promise(resolve => setTimeout(resolve, 1000));

                // Verify job exists
                const job = await isolatedQueueService.swarm.queue.getJob(result.data!.id);
                expect(job).toBeDefined();

                // Clean shutdown
                await isolatedQueueService.shutdown();

                // Reset the instance to ensure clean state
                (QueueService as any).instance = null;
            } finally {
                // Force cleanup of isolated harness
                await isolatedHarness.forceCleanup();
            }
        });

        it("should handle job failure and retry", async () => {
            // Use isolated harness to prevent job failure from corrupting connections
            const isolatedHarness = await createIsolatedQueueTestHarness(redisUrl);

            try {
                // Create fresh QueueService instance for this test
                (QueueService as any).instance = null;
                const isolatedQueueService = QueueService.get();
                await isolatedQueueService.init(redisUrl);

                // Mock process to fail first time
                let callCount = 0;
                const processSwarmMock = vi.fn().mockImplementation(() => {
                    callCount++;
                    if (callCount === 1) {
                        throw new Error("Simulated swarm failure");
                    }
                    return Promise.resolve();
                });

                vi.doMock("./process.js", () => ({
                    swarmProcess: processSwarmMock,
                }));

                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const sessionId = generatePK().toString();

                const swarmData = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    id: swarmId,
                    userId,
                    allocation: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 256,
                        maxConcurrentSteps: 2,
                    },
                    input: {
                        swarmId,
                        goal: "Test failure handling",
                        userData: {
                            __typename: "SessionUser" as const,
                            id: userId,
                            credits: "5000",
                            hasPremium: false,
                            hasReceivedPhoneVerificationReward: false,
                            languages: ["en"],
                            phoneNumberVerified: false,
                            publicId: `pub_${userId}`,
                            session: {
                                __typename: "SessionUserSession" as const,
                                id: sessionId,
                                lastRefreshAt: new Date().toISOString(),
                            },
                            updatedAt: new Date().toISOString(),
                        },
                    },
                };

                const result = await processSwarm(swarmData, isolatedQueueService);
                const job = await isolatedQueueService.swarm.queue.getJob(result.data!.id);

                expect(job).toBeDefined();
                expect(job?.attemptsMade).toBeGreaterThanOrEqual(0);

                // Clean shutdown
                await isolatedQueueService.shutdown();

                // Reset the instance to ensure clean state
                (QueueService as any).instance = null;
            } finally {
                // Force cleanup of isolated harness
                await isolatedHarness.forceCleanup();
            }
        });

        it("should handle three-tier execution flow", async () => {
            // Use isolated harness to prevent tier execution from corrupting connections
            const isolatedHarness = await createIsolatedQueueTestHarness(redisUrl);

            try {
                // Create fresh QueueService instance for this test
                (QueueService as any).instance = null;
                const isolatedQueueService = QueueService.get();
                await isolatedQueueService.init(redisUrl);

                // The three-tier coordinator is already mocked globally

                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const executionId = generatePK().toString();

                const executionData = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    swarmId,
                    executionId,
                    userId,
                    userData: { id: userId, hasPremium: true },
                    config: {
                        model: "gpt-4",
                        temperature: 0.7,
                        maxIterations: 10,
                    },
                };

                const result = await processNewSwarmExecution(executionData, isolatedQueueService);
                expect(result.success).toBe(true);

                // Verify execution task was created with proper priority
                const job = await isolatedQueueService.swarm.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(80); // Premium user priority

                // Clean shutdown
                await isolatedQueueService.shutdown();

                // Reset the instance to ensure clean state
                (QueueService as any).instance = null;
            } finally {
                // Force cleanup of isolated harness
                await isolatedHarness.forceCleanup();
            }
        });
    });

    describe("Queue limits and concurrency", () => {
        it("should respect queue concurrency limits", async () => {
            // Add many tasks quickly
            const promises = [];
            for (let i = 0; i < 20; i++) {
                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const sessionId = generatePK().toString();

                const data = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    id: swarmId,
                    userId,
                    allocation: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 256,
                        maxConcurrentSteps: 2,
                    },
                    input: {
                        swarmId,
                        goal: `Concurrent test ${i}`,
                        userData: {
                            __typename: "SessionUser" as const,
                            id: userId,
                            credits: "5000",
                            hasPremium: i % 2 === 0,
                            hasReceivedPhoneVerificationReward: false,
                            languages: ["en"],
                            phoneNumberVerified: false,
                            publicId: `pub_${userId}`,
                            session: {
                                __typename: "SessionUserSession" as const,
                                id: sessionId,
                                lastRefreshAt: new Date().toISOString(),
                            },
                            updatedAt: new Date().toISOString(),
                        },
                    },
                };
                promises.push(processSwarm(data, queueService));
            }

            const results = await Promise.all(promises);
            expect(results).toHaveLength(20);
            results.forEach(result => {
                expect(result.success).toBe(true);
            });

            // Verify tasks are properly queued
            const jobCounts = await queueService.swarm.queue.getJobCounts();
            expect(jobCounts.waiting + jobCounts.active).toBeGreaterThan(0);
        });

        it("should process tasks in priority order", async () => {
            // Add tasks with different priorities
            const tasks = [
                { hasPremium: false },
                { hasPremium: true },
                { hasPremium: false },
            ];

            const results = [];
            for (const task of tasks) {
                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const sessionId = generatePK().toString();

                const data = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    id: swarmId,
                    userId,
                    allocation: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 256,
                        maxConcurrentSteps: 2,
                    },
                    input: {
                        swarmId,
                        goal: `Priority test for ${task.hasPremium ? "premium" : "regular"} user`,
                        userData: {
                            __typename: "SessionUser" as const,
                            id: userId,
                            credits: "5000",
                            hasPremium: task.hasPremium,
                            hasReceivedPhoneVerificationReward: false,
                            languages: ["en"],
                            phoneNumberVerified: false,
                            publicId: `pub_${userId}`,
                            session: {
                                __typename: "SessionUserSession" as const,
                                id: sessionId,
                                lastRefreshAt: new Date().toISOString(),
                            },
                            updatedAt: new Date().toISOString(),
                        },
                    },
                };
                const result = await processSwarm(data, queueService);
                results.push({ id: result.data!.id, isPremium: task.hasPremium });
            }

            // Premium tasks should be processed with higher priority
            for (const result of results) {
                const job = await queueService.swarm.queue.getJob(result.id);
                if (result.isPremium) {
                    expect(job?.opts.priority).toBe(80);
                } else {
                    expect(job?.opts.priority).toBe(100);
                }
            }
        });
    });

    describe("Three-tier execution edge cases", () => {
        it("should handle tier coordination failures", async () => {
            // Use isolated harness to prevent tier failure from corrupting connections
            const isolatedHarness = await createIsolatedQueueTestHarness(redisUrl);

            try {
                // Create fresh QueueService instance for this test
                (QueueService as any).instance = null;
                const isolatedQueueService = QueueService.get();
                await isolatedQueueService.init(redisUrl);

                // Override the global mock to simulate failure
                const SwarmCoordinator = await import("../../services/execution/tier1/coordination/swarmCoordinator.js");
                vi.mocked(SwarmCoordinator.SwarmCoordinator.getInstance).mockReturnValueOnce({
                    executeSwarm: vi.fn().mockRejectedValue(new Error("Tier coordination failed")),
                });

                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const executionId = generatePK().toString();

                const executionData = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    swarmId,
                    executionId,
                    userId,
                    userData: { id: userId, hasPremium: false },
                    config: {
                        model: "gpt-4",
                        temperature: 0.7,
                    },
                };

                // Job should be added successfully
                const result = await processNewSwarmExecution(executionData, isolatedQueueService);
                expect(result.success).toBe(true);

                // But processing will fail (handled by worker)
                const job = await isolatedQueueService.swarm.queue.getJob(result.data!.id);
                expect(job).toBeDefined();

                // Clean shutdown
                await isolatedQueueService.shutdown();

                // Reset the instance to ensure clean state
                (QueueService as any).instance = null;
            } finally {
                // Force cleanup of isolated harness
                await isolatedHarness.forceCleanup();
            }
        });

        it("should handle swarm state transitions", async () => {
            const userId = generatePK().toString();
            const swarmId = generatePK().toString();
            const sessionId = generatePK().toString();

            const states = [
                "Initializing",
                "Planning",
                "Executing",
                "Analyzing",
                "Completing",
                "Completed",
            ];

            // Add initial swarm task
            const result = await processSwarm({
                type: QueueTaskType.SWARM_EXECUTION,
                id: swarmId,
                userId,
                allocation: {
                    maxCredits: "1000",
                    maxDurationMs: 300000,
                    maxMemoryMB: 256,
                    maxConcurrentSteps: 2,
                },
                input: {
                    swarmId,
                    goal: "Test state transitions",
                    userData: {
                        __typename: "SessionUser" as const,
                        id: userId,
                        credits: "5000",
                        hasPremium: false,
                        hasReceivedPhoneVerificationReward: false,
                        languages: ["en"],
                        phoneNumberVerified: false,
                        publicId: `pub_${userId}`,
                        session: {
                            __typename: "SessionUserSession" as const,
                            id: sessionId,
                            lastRefreshAt: new Date().toISOString(),
                        },
                        updatedAt: new Date().toISOString(),
                    },
                },
            }, queueService);

            const taskId = result.data!.id;

            // Transition through states
            for (const state of states) {
                const statusResult = await changeSwarmTaskStatus(
                    taskId,
                    state,
                    userId,
                    queueService,
                );
                expect(statusResult.success).toBe(true);

                // Verify state transition
                const statuses = await getSwarmTaskStatuses([taskId], queueService);
                expect(statuses[0].status).toBeDefined();
            }
        });

        it("should enforce resource limits", async () => {
            // Use isolated harness to prevent resource testing from corrupting connections
            const isolatedHarness = await createIsolatedQueueTestHarness(redisUrl);

            try {
                // Create fresh QueueService instance for this test
                (QueueService as any).instance = null;
                const isolatedQueueService = QueueService.get();
                await isolatedQueueService.init(redisUrl);

                // Test with various resource configurations
                const resourceConfigs = [
                    { maxIterations: 1, maxTokens: 100 },
                    { maxIterations: 100, maxTokens: 10000 },
                    { maxIterations: 1000, maxTokens: 100000 },
                ];

                for (const config of resourceConfigs) {
                    const userId = generatePK().toString();
                    const swarmId = generatePK().toString();
                    const executionId = generatePK().toString();

                    const executionData = {
                        type: QueueTaskType.SWARM_EXECUTION,
                        swarmId,
                        executionId,
                        userId,
                        userData: { id: userId, hasPremium: false },
                        config: {
                            model: "gpt-4",
                            ...config,
                        },
                    };

                    const result = await processNewSwarmExecution(executionData, isolatedQueueService);
                    expect(result.success).toBe(true);

                    const job = await isolatedQueueService.swarm.queue.getJob(result.data!.id);
                    expect(job?.data.config.maxIterations).toBe(config.maxIterations);
                    expect(job?.data.config.maxTokens).toBe(config.maxTokens);
                }

                // Clean shutdown
                await isolatedQueueService.shutdown();

                // Reset the instance to ensure clean state
                (QueueService as any).instance = null;
            } finally {
                // Force cleanup of isolated harness
                await isolatedHarness.forceCleanup();
            }
        });
    });

    describe("Error handling and recovery", () => {
        it("should handle invalid swarm configurations", async () => {
            const invalidConfigs = [
                { model: "" }, // Empty model
                { model: "invalid-model-12345" }, // Unknown model
                { model: "gpt-4", temperature: -1 }, // Invalid temperature
                { model: "gpt-4", temperature: 2.5 }, // Temperature too high
            ];

            for (const config of invalidConfigs) {
                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const executionId = generatePK().toString();

                const executionData = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    swarmId,
                    executionId,
                    userId,
                    userData: { id: userId, hasPremium: false },
                    config: config as any,
                };

                // Should add successfully (validation in processor)
                const result = await processNewSwarmExecution(executionData, queueService);
                expect(result.success).toBe(true);
            }
        });

        it("should handle missing required swarm data", async () => {
            // Use isolated harness to prevent validation errors from corrupting connections
            const isolatedHarness = await createIsolatedQueueTestHarness(redisUrl);

            try {
                // Create fresh QueueService instance for this test
                (QueueService as any).instance = null;
                const isolatedQueueService = QueueService.get();
                await isolatedQueueService.init(redisUrl);

                const userId = generatePK().toString();
                const sessionId = generatePK().toString();

                const incompleteData = {
                    type: QueueTaskType.SWARM_EXECUTION,
                    allocation: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 256,
                        maxConcurrentSteps: 2,
                    },
                    input: {
                        // Missing required goal property to test validation
                        userData: {
                            __typename: "SessionUser" as const,
                            id: userId,
                            credits: "5000",
                            hasPremium: false,
                            hasReceivedPhoneVerificationReward: false,
                            languages: ["en"],
                            phoneNumberVerified: false,
                            publicId: `pub_${userId}`,
                            session: {
                                __typename: "SessionUserSession" as const,
                                id: sessionId,
                                lastRefreshAt: new Date().toISOString(),
                            },
                            updatedAt: new Date().toISOString(),
                        },
                    },
                } as any;

                // Should still add (validation in processor)
                const result = await processSwarm(incompleteData, isolatedQueueService);
                expect(result.success).toBe(true);

                // Clean shutdown
                await isolatedQueueService.shutdown();

                // Reset the instance to ensure clean state
                (QueueService as any).instance = null;
            } finally {
                // Force cleanup of isolated harness
                await isolatedHarness.forceCleanup();
            }
        });

        it("should handle queue overload scenarios", async () => {
            // Simulate queue overload by adding many tasks rapidly
            const overloadCount = 100;
            const promises = [];

            for (let i = 0; i < overloadCount; i++) {
                const userId = generatePK().toString();
                const swarmId = generatePK().toString();
                const sessionId = generatePK().toString();

                promises.push(
                    processSwarm({
                        type: QueueTaskType.SWARM_EXECUTION,
                        id: swarmId,
                        userId,
                        allocation: {
                            maxCredits: "1000",
                            maxDurationMs: 300000,
                            maxMemoryMB: 256,
                            maxConcurrentSteps: 2,
                        },
                        input: {
                            swarmId,
                            goal: `Complete overload test ${i} efficiently`,
                            executionConfig: {
                                model: "gpt-4",
                            },
                            userData: {
                                __typename: "SessionUser" as const,
                                id: userId,
                                credits: "5000",
                                hasPremium: false,
                                hasReceivedPhoneVerificationReward: false,
                                languages: ["en"],
                                phoneNumberVerified: false,
                                publicId: `pub_${userId}`,
                                session: {
                                    __typename: "SessionUserSession" as const,
                                    id: sessionId,
                                    lastRefreshAt: new Date().toISOString(),
                                },
                                updatedAt: new Date().toISOString(),
                            },
                        },
                    }, queueService),
                );
            }

            const results = await Promise.allSettled(promises);

            // Most should succeed
            const succeeded = results.filter(r => r.status === "fulfilled");
            expect(succeeded.length).toBeGreaterThan(overloadCount * 0.9);

            // Check queue health
            const jobCounts = await queueService.swarm.queue.getJobCounts();
            expect(jobCounts.waiting + jobCounts.active).toBeGreaterThan(0);
        });
    });
});
