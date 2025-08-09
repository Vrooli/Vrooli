import { describe, it, expect, beforeEach, afterEach, afterAll, vi } from "vitest";
import type IORedis from "ioredis";
import { Job } from "bullmq";
import { generatePK } from "@vrooli/shared";
import { QueueService } from "./queues.js";
import { ManagedQueue, buildRedis, type BaseQueueConfig } from "./queueFactory.js";
import { QueueTaskType, type EmailTask, type RunTask, type SwarmTask } from "./taskTypes.js";
import { logger } from "../events/logger.js";
import "../__test/setup.js";

// Test task data types
interface TestTaskData extends EmailTask {
    userId?: string;
    status?: string;
}

describe("Task Status Operations", () => {
    let queueService: QueueService;
    let redisClient: IORedis;
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";

    afterAll(async () => {
        // Cleanup any connections created in this test file
        if (redisClient && redisClient.status !== "end") {
            await redisClient.quit();
        }
    });

    beforeEach(async () => {
        // Initialize QueueService
        queueService = QueueService.get();
        await queueService.init(redisUrl);
        redisClient = await buildRedis(redisUrl);
    });

    afterEach(async () => {
        // Ensure clean shutdown after each test
        try {
            await queueService.shutdown();
            // Wait for shutdown to fully complete and event handlers to detach
            await new Promise(resolve => setTimeout(resolve, 100));
        } catch (error) {
            // Ignore shutdown errors in tests
            console.log("Shutdown error (ignored):", error);
        }
        
        // Clear singleton instance before Redis cache to prevent access during cleanup
        (QueueService as any).instance = null;
        
        // Close the redis client created in beforeEach
        if (redisClient && redisClient.status !== "end") {
            await redisClient.quit();
        }
        
        // Final delay to ensure all async operations complete
        await new Promise(resolve => setTimeout(resolve, 50));
    });

    describe("QueueService.getTaskStatuses", () => {
        it("should get statuses for existing tasks", async () => {
            // Add tasks to different queues
            const emailJob = await queueService.email.add({
                type: QueueTaskType.EMAIL_SEND,
                id: generatePK().toString(),
                to: ["test@example.com"],
                subject: "Test",
                text: "Test email",
                userId: generatePK().toString(),
            });

            const runJob = await queueService.run.add({
                type: QueueTaskType.RUN_START,
                id: generatePK().toString(),
                allocation: {
                    maxCredits: "1000",
                    maxDurationMs: 300000,
                    maxMemoryMB: 512,
                    maxConcurrentSteps: 10,
                },
                context: {
                    swarmId: generatePK().toString(),
                    userData: {
                        __typename: "SessionUser" as const,
                        id: generatePK().toString(),
                        credits: "1000",
                        hasPremium: false,
                        hasReceivedPhoneVerificationReward: false,
                        languages: ["en"],
                        phoneNumberVerified: false,
                        publicId: "test-user-111",
                        session: {
                            __typename: "SessionUserSession",
                            id: generatePK().toString(),
                            lastRefreshAt: new Date().toISOString(),
                        },
                        updatedAt: new Date().toISOString(),
                    },
                    timestamp: new Date(),
                },
                input: {
                    runId: generatePK().toString(),
                    resourceVersionId: generatePK().toString(),
                    isNewRun: true,
                    runFrom: "Manual" as const,
                    startedById: generatePK().toString(),
                    status: "running",
                },
            } as RunTask);

            // Get statuses
            const statuses = await queueService.getTaskStatuses([emailJob.id as string, runJob.id as string]);

            expect(statuses).toHaveLength(2);
            expect(statuses[0]).toEqual({
                id: emailJob.id as string,
                status: expect.any(String),
                queueName: "email-send",
            });
            expect(statuses[1]).toEqual({
                id: runJob.id as string,
                status: "Running", // Should be normalized to PascalCase
                queueName: "run-start",
            });
        });

        it("should return null status for non-existent tasks", async () => {
            const nonExistentId1 = generatePK().toString();
            const nonExistentId2 = generatePK().toString();
            const statuses = await queueService.getTaskStatuses([nonExistentId1, nonExistentId2]);

            expect(statuses).toEqual([
                { id: nonExistentId1, status: null },
                { id: nonExistentId2, status: null },
            ]);
        });

        it("should get statuses from specific queue", async () => {
            // Add tasks to email queue
            const job1 = await queueService.email.add({
                type: QueueTaskType.EMAIL_SEND,
                id: generatePK().toString(),
                to: ["test1@example.com"],
                subject: "Test 1",
                text: "Test 1",
            });

            const job2 = await queueService.email.add({
                type: QueueTaskType.EMAIL_SEND,
                id: generatePK().toString(),
                to: ["test2@example.com"],
                subject: "Test 2",
                text: "Test 2",
            });

            // Get statuses from email queue only
            const statuses = await queueService.getTaskStatuses([job1.id as string, job2.id as string], "email-send");

            expect(statuses).toHaveLength(2);
            expect(statuses[0].queueName).toBe("email-send");
            expect(statuses[1].queueName).toBe("email-send");
        });

        it("should handle mixed existing and non-existent tasks", async () => {
            const job = await queueService.email.add({
                type: QueueTaskType.EMAIL_SEND,
                id: generatePK().toString(),
                to: ["mixed@example.com"],
                subject: "Mixed test",
                text: "Mixed test",
            });

            const nonExistentId = generatePK().toString();
            const anotherNonExistentId = generatePK().toString();
            const statuses = await queueService.getTaskStatuses([
                nonExistentId,
                job.id as string,
                anotherNonExistentId,
            ]);

            expect(statuses).toHaveLength(3);
            expect(statuses[0]).toEqual({ id: nonExistentId, status: null });
            expect(statuses[1].id).toBe(job.id as string);
            expect(statuses[1].status).toBeDefined();
            expect(statuses[2]).toEqual({ id: anotherNonExistentId, status: null });
        });
    });

    describe("QueueService.changeTaskStatus", () => {
        it("should change status for owned task", async () => {
            const userId = generatePK().toString();
            const job = await queueService.email.add({
                type: QueueTaskType.EMAIL_SEND,
                id: generatePK().toString(),
                to: ["owned@example.com"],
                subject: "Owned task",
                text: "Testing ownership",
                userId,
            });

            const result = await queueService.changeTaskStatus(
                job.id as string,
                "processing",
                userId,
                "email-send",
            );

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify status was updated
            const updatedJob = await queueService.email.queue.getJob(job.id as string);
            expect(updatedJob?.data.status).toBe("processing");
        });

        it("should reject status change from non-owner", async () => {
            const errorSpy = vi.spyOn(logger, "error");
            const ownerId = generatePK().toString();
            const otherId = generatePK().toString();

            const job = await queueService.email.add({
                type: QueueTaskType.EMAIL_SEND,
                id: generatePK().toString(),
                to: ["reject@example.com"],
                subject: "Reject test",
                text: "Testing rejection",
                userId: ownerId,
            });

            const result = await queueService.changeTaskStatus(
                job.id as string,
                "processing",
                otherId,
                "email-send",
            );

            expect(result).toEqual({ __typename: "Success", success: false });
            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining("not allowed to update"),
                expect.any(Object),
            );

            errorSpy.mockRestore();
        });

        it("should reject status change for task without owner", async () => {
            const errorSpy = vi.spyOn(logger, "error");

            // Add task without userId
            const job = await queueService.email.add({
                type: QueueTaskType.EMAIL_SEND,
                id: generatePK().toString(),
                to: ["no-owner@example.com"],
                subject: "No owner",
                text: "No owner test",
            });

            const result = await queueService.changeTaskStatus(
                job.id as string,
                "processing",
                "any-user",
                "email-send",
            );

            expect(result).toEqual({ __typename: "Success", success: false });
            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining("does not have an owner"),
                expect.any(Object),
            );

            errorSpy.mockRestore();
        });

        it("should find task across queues when queue name not provided", async () => {
            const userId = generatePK().toString();
            
            // Add task to run queue
            const job = await queueService.run.add({
                type: QueueTaskType.RUN_START,
                id: generatePK().toString(),
                allocation: {
                    maxCredits: "1000",
                    maxDurationMs: 300000,
                    maxMemoryMB: 512,
                    maxConcurrentSteps: 10,
                },
                context: {
                    swarmId: generatePK().toString(),
                    userData: {
                        __typename: "SessionUser" as const,
                        id: userId,
                        credits: "1000",
                        hasPremium: false,
                        hasReceivedPhoneVerificationReward: false,
                        languages: ["en"],
                        phoneNumberVerified: false,
                        publicId: "test-user-456",
                        session: {
                            __typename: "SessionUserSession",
                            id: generatePK().toString(),
                            lastRefreshAt: new Date().toISOString(),
                        },
                        updatedAt: new Date().toISOString(),
                    },
                    timestamp: new Date(),
                },
                input: {
                    runId: generatePK().toString(),
                    resourceVersionId: generatePK().toString(),
                    isNewRun: true,
                    runFrom: "Manual" as const,
                    startedById: userId,
                    status: "queued",
                },
            } as RunTask);

            // Change status without specifying queue
            const result = await queueService.changeTaskStatus(
                job.id as string,
                "completed",
                userId,
            );

            expect(result).toEqual({ __typename: "Success", success: true });
        });

        it("should handle terminal status for non-existent task", async () => {
            // Terminal statuses should succeed even if task doesn't exist
            const result = await queueService.changeTaskStatus(
                "non-existent-task",
                "completed",
                generatePK().toString(),
            );

            expect(result).toEqual({ __typename: "Success", success: true });
        });

        it("should fail for non-terminal status on non-existent task", async () => {
            const result = await queueService.changeTaskStatus(
                "non-existent-task",
                "processing",
                generatePK().toString(),
            );

            expect(result).toEqual({ __typename: "Success", success: false });
        });

        it("should normalize status to lowercase", async () => {
            const userId = generatePK().toString();
            const job = await queueService.email.add({
                type: QueueTaskType.EMAIL_SEND,
                id: generatePK().toString(),
                to: ["normalize@example.com"],
                subject: "Normalize test",
                text: "Testing normalization",
                userId,
            });

            const result = await queueService.changeTaskStatus(
                job.id as string,
                "PROCESSING",
                userId,
                "email-send",
            );

            expect(result.success).toBe(true);

            // Verify status was normalized
            const updatedJob = await queueService.email.queue.getJob(job.id as string);
            expect(updatedJob?.data.status).toBe("processing");
        });
    });

    describe("ManagedQueue task status operations", () => {
        let testQueue: ManagedQueue<TestTaskData>;

        beforeEach(async () => {
            const processorMock = vi.fn().mockResolvedValue({ processed: true });
            
            const config: BaseQueueConfig<TestTaskData> = {
                name: "test-status-queue",
                processor: processorMock,
            };

            testQueue = await ManagedQueue.create(config);
            if (testQueue.ready) {
                await testQueue.ready;
            }
        });

        afterEach(async () => {
            // Use the proper close method which handles cleanup with timeouts
            if (testQueue && testQueue.close) {
                await testQueue.close();
            }
            // Add a small delay to ensure all Redis operations complete
            await new Promise(resolve => setTimeout(resolve, 100));
        });

        describe("getTaskStatuses", () => {
            it("should get multiple task statuses", async () => {
                const jobs = await Promise.all([
                    testQueue.add({
                        type: QueueTaskType.EMAIL_SEND,
                        id: generatePK().toString(),
                        to: ["test1@example.com"],
                        subject: "Test 1",
                        text: "Test 1",
                        userId: generatePK().toString(),
                    }),
                    testQueue.add({
                        type: QueueTaskType.EMAIL_SEND,
                        id: generatePK().toString(),
                        to: ["test2@example.com"],
                        subject: "Test 2",
                        text: "Test 2",
                        userId: generatePK().toString(),
                    }),
                ]);

                const jobIds = jobs.map(j => j.id as string);
                const statuses = await testQueue.getTaskStatuses(jobIds);

                expect(statuses).toHaveLength(2);
                statuses.forEach((status, index) => {
                    expect(status.id).toBe(jobIds[index]);
                    expect(status.status).toBeDefined();
                });
            });

            it("should use data status over BullMQ state", async () => {
                const job = await testQueue.add({
                    type: QueueTaskType.EMAIL_SEND,
                    id: generatePK().toString(),
                    to: ["custom-status@example.com"],
                    subject: "Custom status",
                    text: "Testing custom status",
                    status: "custom-processing",
                });

                const statuses = await testQueue.getTaskStatuses([job.id as string]);
                
                expect(statuses[0].status).toBe("custom-processing");
            });

            it("should handle errors gracefully", async () => {
                const errorSpy = vi.spyOn(logger, "error");
                
                // Mock getJob to throw error
                const originalGetJob = testQueue.queue.getJob;
                testQueue.queue.getJob = vi.fn().mockRejectedValue(new Error("Database error"));

                const statuses = await testQueue.getTaskStatuses(["error-task"]);

                expect(statuses).toEqual([{ id: "error-task", status: null }]);
                expect(errorSpy).toHaveBeenCalled();

                // Restore
                testQueue.queue.getJob = originalGetJob;
                errorSpy.mockRestore();
            });
        });

        describe("changeTaskStatus", () => {
            it("should update task status and data", async () => {
                const userId = generatePK().toString();
                const job = await testQueue.add({
                    type: QueueTaskType.EMAIL_SEND,
                    id: generatePK().toString(),
                    to: ["update@example.com"],
                    subject: "Update test",
                    text: "Testing update",
                    userId,
                });

                const result = await testQueue.changeTaskStatus(
                    job.id as string,
                    "processing",
                    userId,
                );

                expect(result).toEqual({ __typename: "Success", success: true });

                // Verify job data was updated
                const updatedJob = await testQueue.queue.getJob(job.id as string);
                expect(updatedJob?.data.status).toBe("processing");
            });

            it("should handle update errors", async () => {
                const errorSpy = vi.spyOn(logger, "error");
                const userId = generatePK().toString();
                
                const job = await testQueue.add({
                    type: QueueTaskType.EMAIL_SEND,
                    id: generatePK().toString(),
                    to: ["error@example.com"],
                    subject: "Error test",
                    text: "Testing error",
                    userId,
                });

                // Mock job.update to throw error
                const jobInstance = await testQueue.queue.getJob(job.id as string);
                if (jobInstance) {
                    jobInstance.update = vi.fn().mockRejectedValue(new Error("Update failed"));
                }

                const result = await testQueue.changeTaskStatus(
                    job.id as string,
                    "processing",
                    userId,
                );

                expect(result).toEqual({ __typename: "Success", success: false });
                expect(errorSpy).toHaveBeenCalledWith(
                    expect.stringContaining("Failed to change status"),
                    expect.any(Object),
                );

                errorSpy.mockRestore();
            });

            it("should check ownership for different task types", async () => {
                // Test with RunTask that uses startedById
                const runQueue = await ManagedQueue.create<RunTask>({
                    name: "test-run-queue",
                    processor: vi.fn(),
                });

                const startedById = generatePK().toString();
                const job = await runQueue.add({
                    type: QueueTaskType.RUN_START,
                    id: generatePK().toString(),
                    allocation: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 512,
                        maxConcurrentSteps: 10,
                    },
                    context: {
                        swarmId: generatePK().toString(),
                        userData: {
                            __typename: "SessionUser" as const,
                            id: startedById,
                            credits: "1000",
                            hasPremium: false,
                            hasReceivedPhoneVerificationReward: false,
                            languages: ["en"],
                            phoneNumberVerified: false,
                            publicId: "test-user-789",
                            session: {
                                __typename: "SessionUserSession",
                                id: generatePK().toString(),
                                lastRefreshAt: new Date().toISOString(),
                            },
                            updatedAt: new Date().toISOString(),
                        },
                        timestamp: new Date(),
                    },
                    input: {
                        runId: generatePK().toString(),
                        resourceVersionId: generatePK().toString(),
                        isNewRun: true,
                        runFrom: "Manual" as const,
                        startedById,
                        status: "queued",
                    },
                } as RunTask);

                const result = await runQueue.changeTaskStatus(
                    job.id as string,
                    "completed",
                    startedById,
                );

                expect(result.success).toBe(true);

                // Try with wrong user
                const wrongResult = await runQueue.changeTaskStatus(
                    job.id as string,
                    "failed",
                    "wrong-user",
                );

                expect(wrongResult.success).toBe(false);

                // Use the proper close method which handles cleanup with timeouts
                await runQueue.close();
                // Add a small delay to ensure all Redis operations complete
                await new Promise(resolve => setTimeout(resolve, 100));
            });
        });

        describe("getTaskOwner helper", () => {
            it("should identify owner from userId", () => {
                const task: EmailTask = {
                    type: QueueTaskType.EMAIL_SEND,
                    id: generatePK().toString(),
                    to: ["test@example.com"],
                    subject: "Test",
                    text: "Test",
                    userId: generatePK().toString(),
                };

                const owner = ManagedQueue.getTaskOwner(task);
                expect(owner).toBe(task.userId);
            });

            it("should identify owner from startedById", () => {
                const startedById = generatePK().toString();
                const task: RunTask = {
                    type: QueueTaskType.RUN_START,
                    id: generatePK().toString(),
                    allocation: {
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                        maxMemoryMB: 512,
                        maxConcurrentSteps: 10,
                    },
                    context: {
                        swarmId: generatePK().toString(),
                        userData: {
                            __typename: "SessionUser",
                            id: generatePK().toString(),
                            credits: "1000",
                            hasPremium: false,
                            hasReceivedPhoneVerificationReward: false,
                            languages: ["en"],
                            phoneNumberVerified: false,
                            publicId: "test-user-456",
                            session: {
                                __typename: "SessionUserSession",
                                id: generatePK().toString(),
                                lastRefreshAt: new Date().toISOString(),
                            },
                            updatedAt: new Date().toISOString(),
                        },
                        timestamp: new Date(),
                    },
                    input: {
                        runId: generatePK().toString(),
                        resourceVersionId: generatePK().toString(),
                        isNewRun: true,
                        runFrom: "Manual",
                        startedById,
                        status: "queued",
                    },
                    startedById, // For extractOwnerId compatibility
                } as any;

                const owner = ManagedQueue.getTaskOwner(task);
                expect(owner).toBe(task.startedById);
            });

            it("should identify owner from userData.id", () => {
                const task: SwarmTask = {
                    type: QueueTaskType.LLM_COMPLETION,
                    id: generatePK().toString(),
                    chatId: generatePK().toString(),
                    messageId: generatePK().toString(), 
                    taskContexts: [],
                    userData: { 
                        __typename: "SessionUser",
                        id: generatePK().toString(),
                        credits: "1000",
                        hasPremium: false,
                        hasReceivedPhoneVerificationReward: false,
                        languages: ["en"],
                        phoneNumberVerified: false,
                        publicId: "test-user-789",
                        session: {
                            __typename: "SessionUserSession",
                            id: generatePK().toString(),
                            lastRefreshAt: new Date().toISOString(),
                        },
                        updatedAt: new Date().toISOString(),
                    },
                    allocation: {
                        maxCredits: "100",
                        maxDurationMs: 30000,
                        maxMemoryMB: 512,
                        maxConcurrentSteps: 10,
                    },
                };

                const owner = ManagedQueue.getTaskOwner(task);
                expect(owner).toBe(task.userData.id);
            });

            it("should return undefined for no owner", () => {
                const task: EmailTask = {
                    type: QueueTaskType.EMAIL_SEND,
                    id: generatePK().toString(),
                    to: ["no-owner@example.com"],
                    subject: "No owner",
                    text: "No owner",
                };

                const owner = ManagedQueue.getTaskOwner(task);
                expect(owner).toBeNull();
            });
        });
    });
});
