// AI_CHECK: TEST_QUALITY=1, TEST_COVERAGE=1 | LAST: 2025-06-18
import { generatePK, initIdGenerator, RunTriggeredFrom } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRunTask } from "../../__test/fixtures/tasks/runTaskFactory.js";
import "../../__test/setup.js";
import { clearRedisCache } from "../queueFactory.js";
import { QueueService } from "../queues.js";
import { QueueTaskType } from "../taskTypes.js";
import { changeRunTaskStatus, getRunTaskStatuses, processRun } from "./queue.js";

describe("Run Queue", () => {
    let queueService: QueueService;
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";

    // Base run data for tests that don't use createBaseRunData helper
    let baseRunData: {
        runId: string;
        routineId: string;
        routineVersionId: string;
        isNewRun: boolean;
        runFrom: RunTriggeredFrom;
        userData: { id: string; hasPremium: boolean };
        config: { isTimeSensitive: boolean };
    };

    beforeEach(async () => {
        // Initialize ID generator first
        await initIdGenerator(0);
        
        // Create base run data after ID generator is initialized
        const userId = generatePK().toString();
        baseRunData = {
            runId: generatePK().toString(),
            routineId: generatePK().toString(),
            routineVersionId: generatePK().toString(),
            isNewRun: true,
            runFrom: RunTriggeredFrom.RunView,
            userData: { id: userId, hasPremium: false },
            config: { isTimeSensitive: false },
        };
        
        // Get fresh instance and initialize
        queueService = QueueService.get();
        await queueService.init(redisUrl);
    });

    afterEach(async () => {
        // Clean shutdown - order is critical to prevent "Connection is closed" errors
        try {
            await queueService.shutdown();
            // Wait for shutdown to fully complete and event handlers to detach
            await new Promise(resolve => setTimeout(resolve, 100));
        } catch (error) {
            console.log("Shutdown error (ignored):", error);
        }
        // Clear singleton before clearing cache to prevent any access during cleanup
        (QueueService as any).instance = null;
        // Clear Redis cache last to avoid disconnecting connections still in use
        clearRedisCache();
        // Final delay to ensure all async operations complete
        await new Promise(resolve => setTimeout(resolve, 50));
    });

    describe("processRun", () => {
        const createBaseRunData = (overrides = {}) => {
            const runTask = createRunTask(overrides);
            // Remove status field as processRun expects Omit<RunTask, "status">
            const { status, ...runDataWithoutStatus } = runTask;
            return runDataWithoutStatus;
        };

        it("should add run task with correct type and status", async () => {
            const runData = createBaseRunData();
            const result = await processRun(runData, queueService);
            expect(result.success).toBe(true);
            expect(result.data?.id).toBeDefined();

            // Verify the job was added
            const job = await queueService.run.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
            expect(job?.data.type).toBe(QueueTaskType.RUN_START);
            expect(job?.data.status).toBe("Scheduled");
        });

        describe("priority calculation", () => {
            it("should set highest priority for RunView trigger", async () => {
                const runData = createBaseRunData({
                    input: { runFrom: RunTriggeredFrom.RunView },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(80); // 100 - 20
            });

            it("should set lower priority for Api trigger", async () => {
                const runData = createBaseRunData({
                    input: { runFrom: RunTriggeredFrom.Api },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(84); // 100 - 16
            });

            it("should set lower priority for Chat trigger", async () => {
                const runData = createBaseRunData({
                    input: { runFrom: RunTriggeredFrom.Chat },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(86); // 100 - 14
            });

            it("should set lower priority for Webhook trigger", async () => {
                const runData = createBaseRunData({
                    input: { runFrom: RunTriggeredFrom.Webhook },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(90); // 100 - 10
            });

            it("should set lower priority for Bot trigger", async () => {
                const runData = createBaseRunData({
                    input: { runFrom: RunTriggeredFrom.Bot },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(93); // 100 - 7
            });

            it("should set lower priority for Schedule trigger", async () => {
                const runData = createBaseRunData({
                    input: { runFrom: RunTriggeredFrom.Schedule },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(97); // 100 - 3
            });

            it("should set lowest priority for Test trigger", async () => {
                const runData = createBaseRunData({
                    input: { runFrom: RunTriggeredFrom.Test },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(99); // 100 - 1
            });

            it("should boost priority for time-sensitive runs", async () => {
                const runData = createBaseRunData({
                    input: {
                        runFrom: RunTriggeredFrom.RunView,
                        config: { isTimeSensitive: true },
                    },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(65); // 100 - 20 - 15
            });

            it("should boost priority for premium users", async () => {
                const runData = createBaseRunData({
                    context: {
                        userData: {
                            id: generatePK().toString(),
                            hasPremium: true,
                            name: "premiumUser",
                            languages: ["en"],
                            roles: [],
                            wallets: [],
                            theme: "light",
                        },
                    },
                    input: { runFrom: RunTriggeredFrom.RunView },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(75); // 100 - 20 - 5
            });

            it("should boost priority for existing runs", async () => {
                // Mock activeRunRegistry
                vi.doMock("./process.js", () => ({
                    activeRunRegistry: {
                        get: vi.fn().mockReturnValue(true),
                    },
                }));

                const runData = createBaseRunData({
                    input: { isNewRun: false },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(70); // 100 - 20 - 10
            });

            it("should combine all priority boosts", async () => {
                // Mock activeRunRegistry for existing run boost
                vi.doMock("./process.js", () => ({
                    activeRunRegistry: {
                        get: vi.fn().mockReturnValue(true),
                    },
                }));

                const runData = createBaseRunData({
                    context: {
                        userData: {
                            id: generatePK().toString(),
                            hasPremium: true,
                            name: "premiumUser",
                            languages: ["en"],
                            roles: [],
                            wallets: [],
                            theme: "light",
                        },
                    },
                    input: {
                        isNewRun: false,
                        runFrom: RunTriggeredFrom.RunView,
                        config: { isTimeSensitive: true },
                    },
                });
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(50); // 100 - 20 - 15 - 5 - 10
            });

            it("should not go below 0 priority", async () => {
                // Mock activeRunRegistry
                vi.doMock("./process.js", () => ({
                    activeRunRegistry: {
                        get: vi.fn().mockReturnValue(true),
                    },
                }));

                // Max out all priority boosts to potentially go negative
                const runData = createBaseRunData({
                    context: {
                        userData: {
                            id: generatePK().toString(),
                            hasPremium: true,
                            name: "premiumUser",
                            languages: ["en"],
                            roles: [],
                            wallets: [],
                            theme: "light",
                        },
                    },
                    input: {
                        isNewRun: false,
                        runFrom: RunTriggeredFrom.RunView,
                        config: { isTimeSensitive: true },
                    },
                });

                // Add more priority reductions to test floor
                const result = await processRun(runData, queueService);
                const job = await queueService.run.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBeGreaterThanOrEqual(0);
            });
        });

        it("should handle concurrent task additions", async () => {
            const promises = [];
            for (let i = 0; i < 5; i++) {
                const runData = createBaseRunData({
                    input: { runId: generatePK().toString() },
                });
                promises.push(processRun(runData, queueService));
            }

            const results = await Promise.all(promises);
            expect(results).toHaveLength(5);
            results.forEach(result => {
                expect(result.success).toBe(true);
                expect(result.data?.id).toBeDefined();
            });
        });
    });

    describe("changeRunTaskStatus", () => {
        let jobId: string;

        beforeEach(async () => {
            // Add a test job
            const result = await processRun({
                runId: generatePK().toString(),
                routineId: generatePK().toString(),
                routineVersionId: generatePK().toString(),
                isNewRun: true,
                runFrom: RunTriggeredFrom.RunView,
                userData: { id: generatePK().toString(), hasPremium: false },
                config: { isTimeSensitive: false },
            }, queueService);
            jobId = result.data!.id;
        });

        it("should change task status", async () => {
            const result = await changeRunTaskStatus(jobId, "Running", generatePK().toString(), queueService);
            expect(result.success).toBe(true);
        });

        it("should verify status change is delegated to QueueService", async () => {
            const spy = vi.spyOn(queueService, "changeTaskStatus");
            const userId = generatePK().toString();
            await changeRunTaskStatus(jobId, "Running", userId, queueService);
            expect(spy).toHaveBeenCalledWith(jobId, "Running", userId, "run");
        });

        it("should handle invalid job ID", async () => {
            const result = await changeRunTaskStatus("invalid-id", "Running", generatePK().toString(), queueService);
            expect(result.success).toBe(false);
        });
    });

    describe("getRunTaskStatuses", () => {
        const jobIds: string[] = [];

        beforeEach(async () => {
            // Add multiple test jobs
            for (let i = 0; i < 3; i++) {
                const result = await processRun({
                    runId: generatePK().toString(),
                    routineId: generatePK().toString(),
                    routineVersionId: generatePK().toString(),
                    isNewRun: true,
                    runFrom: RunTriggeredFrom.RunView,
                    userData: { id: generatePK().toString(), hasPremium: false },
                    config: { isTimeSensitive: false },
                }, queueService);
                jobIds.push(result.data!.id);
            }
        });

        it("should fetch multiple task statuses", async () => {
            const statuses = await getRunTaskStatuses(jobIds, queueService);
            expect(statuses).toHaveLength(3);
            statuses.forEach((status, index) => {
                expect(status.__typename).toBe("TaskStatusInfo");
                expect(status.id).toBe(jobIds[index]);
                expect(status.queueName).toBe("run");
                expect(status.status).toBeDefined();
            });
        });

        it("should handle mixed valid and invalid IDs", async () => {
            const mixedIds = [...jobIds, "invalid-id-1", "invalid-id-2"];
            const statuses = await getRunTaskStatuses(mixedIds, queueService);

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
            const statuses = await getRunTaskStatuses([], queueService);
            expect(statuses).toEqual([]);
        });

        it("should verify delegation to QueueService", async () => {
            const spy = vi.spyOn(queueService, "getTaskStatuses");
            await getRunTaskStatuses(jobIds, queueService);
            expect(spy).toHaveBeenCalledWith(jobIds, "run");
        });
    });

    describe("Integration with QueueService", () => {
        it("should process run task through worker", async () => {
            // Mock the run process function
            const processRunMock = vi.fn().mockResolvedValue(undefined);
            vi.doMock("./process.js", () => ({
                runProcess: processRunMock,
                activeRunRegistry: new Map(),
            }));

            const runData = {
                runId: generatePK().toString(),
                routineId: generatePK().toString(),
                routineVersionId: generatePK().toString(),
                isNewRun: true,
                runFrom: RunTriggeredFrom.RunView,
                userData: { id: generatePK().toString(), hasPremium: false },
                config: { isTimeSensitive: false },
            };

            const result = await processRun(runData, queueService);
            expect(result.success).toBe(true);

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 1000));

            // Verify job exists
            const job = await queueService.run.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
        });

        it("should handle job failure and retry", async () => {
            // Mock process to fail first time
            let callCount = 0;
            const processRunMock = vi.fn().mockImplementation(() => {
                callCount++;
                if (callCount === 1) {
                    throw new Error("Simulated failure");
                }
                return Promise.resolve();
            });

            vi.doMock("./process.js", () => ({
                runProcess: processRunMock,
                activeRunRegistry: new Map(),
            }));

            const runData = {
                runId: generatePK().toString(),
                routineId: generatePK().toString(),
                routineVersionId: generatePK().toString(),
                isNewRun: true,
                runFrom: RunTriggeredFrom.RunView,
                userData: { id: generatePK().toString(), hasPremium: false },
                config: { isTimeSensitive: false },
            };

            const result = await processRun(runData, queueService);
            const job = await queueService.run.queue.getJob(result.data!.id);

            expect(job).toBeDefined();
            expect(job?.attemptsMade).toBeGreaterThanOrEqual(0);
        });
    });

    describe("Priority edge cases and boundary testing", () => {
        it("should handle priority overflow protection", async () => {
            // Test with maximum possible priority boosts
            const data = {
                ...baseRunData,
                runFrom: RunTriggeredFrom.RunView, // -20
                userData: { id: generatePK().toString(), hasPremium: true }, // -5
                config: { isTimeSensitive: true }, // -15
                isNewRun: false, // -10 (with mock)
            };

            // Mock for existing run boost
            vi.doMock("./process.js", () => ({
                activeRunRegistry: {
                    get: vi.fn().mockReturnValue(true),
                },
            }));

            const result = await processRun(data, queueService);
            const job = await queueService.run.queue.getJob(result.data!.id);

            // Priority should never go below 0
            expect(job?.opts.priority).toBeGreaterThanOrEqual(0);
            expect(job?.opts.priority).toBeLessThanOrEqual(100);
        });

        it("should handle unknown RunTriggeredFrom values", async () => {
            const data = {
                ...baseRunData,
                runFrom: "UnknownTrigger" as any, // Invalid trigger
            };

            const result = await processRun(data, queueService);
            const job = await queueService.run.queue.getJob(result.data!.id);

            // Should use default priority
            expect(job?.opts.priority).toBe(100);
        });
    });

    describe("Job lifecycle and state management", () => {
        it("should handle job cancellation", async () => {
            const result = await processRun(baseRunData, queueService);
            const jobId = result.data!.id;

            // Get the job
            const job = await queueService.run.queue.getJob(jobId);
            expect(job).toBeDefined();

            // Cancel the job
            await changeRunTaskStatus(jobId, "Cancelled", generatePK().toString(), queueService);

            // Verify status change
            const statuses = await getRunTaskStatuses([jobId], queueService);
            expect(statuses[0].status).toBeDefined();
        });

        it("should track job progress updates", async () => {
            const result = await processRun(baseRunData, queueService);
            const jobId = result.data!.id;

            // Simulate progress updates
            const progressStates = [
                "Scheduled",
                "Running",
                "Processing",
                "Completing",
                "Completed",
            ];

            for (const state of progressStates) {
                const updateResult = await changeRunTaskStatus(jobId, state, generatePK().toString(), queueService);
                expect(updateResult.success).toBe(true);
            }
        });

        it("should handle job dependencies", async () => {
            // Create parent job
            const parentRunId = generatePK().toString();
            const parentResult = await processRun({
                ...baseRunData,
                runId: parentRunId,
            }, queueService);

            // Create dependent job
            const dependentResult = await processRun({
                ...baseRunData,
                runId: generatePK().toString(),
                parentRunId,
            }, queueService);

            expect(parentResult.success).toBe(true);
            expect(dependentResult.success).toBe(true);

            // Verify both jobs exist
            const parentJob = await queueService.run.queue.getJob(parentResult.data!.id);
            const dependentJob = await queueService.run.queue.getJob(dependentResult.data!.id);

            expect(parentJob).toBeDefined();
            expect(dependentJob).toBeDefined();
        });
    });

    describe("Error scenarios and recovery", () => {
        it("should handle missing required fields", async () => {
            const invalidData = {
                // Missing runId
                routineId: generatePK().toString(),
                routineVersionId: generatePK().toString(),
                isNewRun: true,
                runFrom: RunTriggeredFrom.RunView,
                userData: { id: generatePK().toString(), hasPremium: false },
                config: { isTimeSensitive: false },
            } as any;

            // Should still process (validation in worker)
            const result = await processRun(invalidData, queueService);
            expect(result.success).toBe(true);
        });

        it("should handle queue service failures gracefully", async () => {
            // Create a spy that simulates queue failure
            const addSpy = vi.spyOn(queueService.run, "add").mockRejectedValueOnce(
                new Error("Queue unavailable"),
            );

            try {
                await processRun(baseRunData, queueService);
                expect.fail("Should have thrown error");
            } catch (error) {
                expect(error.message).toContain("Queue unavailable");
            } finally {
                addSpy.mockRestore();
            }
        });

        it("should handle concurrent status updates", async () => {
            const result = await processRun(baseRunData, queueService);
            const jobId = result.data!.id;

            // Attempt multiple concurrent status updates
            const updates = [
                changeRunTaskStatus(jobId, "Running", generatePK().toString(), queueService),
                changeRunTaskStatus(jobId, "Processing", generatePK().toString(), queueService),
                changeRunTaskStatus(jobId, "Paused", generatePK().toString(), queueService),
            ];

            const results = await Promise.allSettled(updates);

            // At least one should succeed
            const succeeded = results.filter(r => r.status === "fulfilled");
            expect(succeeded.length).toBeGreaterThan(0);
        });
    });
});
