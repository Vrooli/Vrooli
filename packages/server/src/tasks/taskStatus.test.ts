import { describe, it, expect, beforeEach, afterEach, afterAll, vi } from "vitest";
import type IORedis from "ioredis";
import { Job } from "bullmq";
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
        // Clean up
        await queueService.shutdown();
        (QueueService as any).instance = null;
        
        // Close the redis client created in beforeEach
        if (redisClient && redisClient.status !== "end") {
            await redisClient.quit();
        }
        
        // Add delay to ensure all Redis operations and worker threads complete
        await new Promise(resolve => setTimeout(resolve, 200));
    });

    describe("QueueService.getTaskStatuses", () => {
        it("should get statuses for existing tasks", async () => {
            // Add tasks to different queues
            const emailJob = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["test@example.com"],
                subject: "Test",
                text: "Test email",
                userId: "user-1",
            });

            const runJob = await queueService.run.add({
                taskType: QueueTaskType.Run,
                runId: "run-123",
                userId: "user-1",
                hasPremium: false,
                status: "running",
            });

            // Get statuses
            const statuses = await queueService.getTaskStatuses([emailJob.id!, runJob.id!]);

            expect(statuses).toHaveLength(2);
            expect(statuses[0]).toEqual({
                id: emailJob.id,
                status: expect.any(String),
                queueName: "email-send",
            });
            expect(statuses[1]).toEqual({
                id: runJob.id,
                status: "Running", // Should be normalized to PascalCase
                queueName: "run-start",
            });
        });

        it("should return null status for non-existent tasks", async () => {
            const statuses = await queueService.getTaskStatuses(["non-existent-1", "non-existent-2"]);

            expect(statuses).toEqual([
                { id: "non-existent-1", status: null },
                { id: "non-existent-2", status: null },
            ]);
        });

        it("should get statuses from specific queue", async () => {
            // Add tasks to email queue
            const job1 = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["test1@example.com"],
                subject: "Test 1",
                text: "Test 1",
            });

            const job2 = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["test2@example.com"],
                subject: "Test 2",
                text: "Test 2",
            });

            // Get statuses from email queue only
            const statuses = await queueService.getTaskStatuses([job1.id!, job2.id!], "email-send");

            expect(statuses).toHaveLength(2);
            expect(statuses[0].queueName).toBe("email-send");
            expect(statuses[1].queueName).toBe("email-send");
        });

        it("should handle mixed existing and non-existent tasks", async () => {
            const job = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["mixed@example.com"],
                subject: "Mixed test",
                text: "Mixed test",
            });

            const statuses = await queueService.getTaskStatuses([
                "non-existent",
                job.id!,
                "another-non-existent",
            ]);

            expect(statuses).toHaveLength(3);
            expect(statuses[0]).toEqual({ id: "non-existent", status: null });
            expect(statuses[1].id).toBe(job.id);
            expect(statuses[1].status).toBeDefined();
            expect(statuses[2]).toEqual({ id: "another-non-existent", status: null });
        });
    });

    describe("QueueService.changeTaskStatus", () => {
        it("should change status for owned task", async () => {
            const userId = "user-123";
            const job = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["owned@example.com"],
                subject: "Owned task",
                text: "Testing ownership",
                userId,
            });

            const result = await queueService.changeTaskStatus(
                job.id!,
                "processing",
                userId,
                "email-send",
            );

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify status was updated
            const updatedJob = await queueService.email.queue.getJob(job.id!);
            expect(updatedJob?.data.status).toBe("processing");
        });

        it("should reject status change from non-owner", async () => {
            const errorSpy = vi.spyOn(logger, "error");
            const ownerId = "owner-123";
            const otherId = "other-456";

            const job = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["reject@example.com"],
                subject: "Reject test",
                text: "Testing rejection",
                userId: ownerId,
            });

            const result = await queueService.changeTaskStatus(
                job.id!,
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
                taskType: QueueTaskType.Email,
                to: ["no-owner@example.com"],
                subject: "No owner",
                text: "No owner test",
            });

            const result = await queueService.changeTaskStatus(
                job.id!,
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
            const userId = "user-123";
            
            // Add task to run queue
            const job = await queueService.run.add({
                taskType: QueueTaskType.Run,
                runId: "run-456",
                userId,
                hasPremium: false,
            });

            // Change status without specifying queue
            const result = await queueService.changeTaskStatus(
                job.id!,
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
                "user-123",
            );

            expect(result).toEqual({ __typename: "Success", success: true });
        });

        it("should fail for non-terminal status on non-existent task", async () => {
            const result = await queueService.changeTaskStatus(
                "non-existent-task",
                "processing",
                "user-123",
            );

            expect(result).toEqual({ __typename: "Success", success: false });
        });

        it("should normalize status to lowercase", async () => {
            const userId = "user-123";
            const job = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["normalize@example.com"],
                subject: "Normalize test",
                text: "Testing normalization",
                userId,
            });

            const result = await queueService.changeTaskStatus(
                job.id!,
                "PROCESSING",
                userId,
                "email-send",
            );

            expect(result.success).toBe(true);

            // Verify status was normalized
            const updatedJob = await queueService.email.queue.getJob(job.id!);
            expect(updatedJob?.data.status).toBe("processing");
        });
    });

    describe("ManagedQueue task status operations", () => {
        let testQueue: ManagedQueue<TestTaskData>;
        let testQueueRedisClient: IORedis;

        beforeEach(async () => {
            // Create a separate Redis connection for this test queue
            testQueueRedisClient = await buildRedis(redisUrl);
            
            const processorMock = vi.fn().mockResolvedValue({ processed: true });
            
            const config: BaseQueueConfig<TestTaskData> = {
                name: "test-status-queue",
                processor: processorMock,
            };

            testQueue = new ManagedQueue(config, testQueueRedisClient);
            await testQueue.ready;
        });

        afterEach(async () => {
            // Use the proper close method which handles cleanup with timeouts
            await testQueue.close();
            // Close the separate Redis connection
            if (testQueueRedisClient && testQueueRedisClient.status !== "end") {
                await testQueueRedisClient.quit();
            }
            // Add a small delay to ensure all Redis operations complete
            await new Promise(resolve => setTimeout(resolve, 100));
        });

        describe("getTaskStatuses", () => {
            it("should get multiple task statuses", async () => {
                const jobs = await Promise.all([
                    testQueue.add({
                        taskType: QueueTaskType.Email,
                        to: ["test1@example.com"],
                        subject: "Test 1",
                        text: "Test 1",
                        userId: "user-1",
                    }),
                    testQueue.add({
                        taskType: QueueTaskType.Email,
                        to: ["test2@example.com"],
                        subject: "Test 2",
                        text: "Test 2",
                        userId: "user-2",
                    }),
                ]);

                const jobIds = jobs.map(j => j.id!);
                const statuses = await testQueue.getTaskStatuses(jobIds);

                expect(statuses).toHaveLength(2);
                statuses.forEach((status, index) => {
                    expect(status.id).toBe(jobIds[index]);
                    expect(status.status).toBeDefined();
                });
            });

            it("should use data status over BullMQ state", async () => {
                const job = await testQueue.add({
                    taskType: QueueTaskType.Email,
                    to: ["custom-status@example.com"],
                    subject: "Custom status",
                    text: "Testing custom status",
                    status: "custom-processing",
                });

                const statuses = await testQueue.getTaskStatuses([job.id!]);
                
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
                const userId = "user-123";
                const job = await testQueue.add({
                    taskType: QueueTaskType.Email,
                    to: ["update@example.com"],
                    subject: "Update test",
                    text: "Testing update",
                    userId,
                });

                const result = await testQueue.changeTaskStatus(
                    job.id!,
                    "processing",
                    userId,
                );

                expect(result).toEqual({ __typename: "Success", success: true });

                // Verify job data was updated
                const updatedJob = await testQueue.queue.getJob(job.id!);
                expect(updatedJob?.data.status).toBe("processing");
            });

            it("should handle update errors", async () => {
                const errorSpy = vi.spyOn(logger, "error");
                const userId = "user-123";
                
                const job = await testQueue.add({
                    taskType: QueueTaskType.Email,
                    to: ["error@example.com"],
                    subject: "Error test",
                    text: "Testing error",
                    userId,
                });

                // Mock job.update to throw error
                const jobInstance = await testQueue.queue.getJob(job.id!);
                if (jobInstance) {
                    jobInstance.update = vi.fn().mockRejectedValue(new Error("Update failed"));
                }

                const result = await testQueue.changeTaskStatus(
                    job.id!,
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
                // Create a separate Redis connection for this queue to avoid connection conflicts
                const runRedisClient = await buildRedis(redisUrl);
                const runQueue = new ManagedQueue<RunTask>({
                    name: "test-run-queue",
                    processor: vi.fn(),
                }, runRedisClient);

                const startedById = "starter-123";
                const job = await runQueue.add({
                    taskType: QueueTaskType.Run,
                    runId: "run-789",
                    startedById,
                    hasPremium: false,
                });

                const result = await runQueue.changeTaskStatus(
                    job.id!,
                    "completed",
                    startedById,
                );

                expect(result.success).toBe(true);

                // Try with wrong user
                const wrongResult = await runQueue.changeTaskStatus(
                    job.id!,
                    "failed",
                    "wrong-user",
                );

                expect(wrongResult.success).toBe(false);

                // Use the proper close method which handles cleanup with timeouts
                await runQueue.close();
                // Close the separate Redis connection
                if (runRedisClient && runRedisClient.status !== "end") {
                    await runRedisClient.quit();
                }
                // Add a small delay to ensure all Redis operations complete
                await new Promise(resolve => setTimeout(resolve, 100));
            });
        });

        describe("getTaskOwner helper", () => {
            it("should identify owner from userId", () => {
                const task: EmailTask = {
                    taskType: QueueTaskType.Email,
                    to: ["test@example.com"],
                    subject: "Test",
                    text: "Test",
                    userId: "user-123",
                };

                const owner = ManagedQueue.getTaskOwner(task);
                expect(owner).toBe("user-123");
            });

            it("should identify owner from startedById", () => {
                const task: RunTask = {
                    taskType: QueueTaskType.Run,
                    runId: "run-123",
                    startedById: "starter-456",
                    hasPremium: false,
                };

                const owner = ManagedQueue.getTaskOwner(task);
                expect(owner).toBe("starter-456");
            });

            it("should identify owner from userData.id", () => {
                const task: SwarmTask = {
                    taskType: QueueTaskType.Swarm,
                    conversationId: "conv-123",
                    hasPremium: false,
                    swarmModel: "gpt-4",
                    execType: "chat",
                    userData: { id: "data-user-789" },
                };

                const owner = ManagedQueue.getTaskOwner(task);
                expect(owner).toBe("data-user-789");
            });

            it("should return undefined for no owner", () => {
                const task: EmailTask = {
                    taskType: QueueTaskType.Email,
                    to: ["no-owner@example.com"],
                    subject: "No owner",
                    text: "No owner",
                };

                const owner = ManagedQueue.getTaskOwner(task);
                expect(owner).toBeUndefined();
            });
        });
    });
});
