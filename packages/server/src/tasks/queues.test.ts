import { describe, it, expect, beforeEach, afterEach, beforeAll, afterAll, vi } from "vitest";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import IORedis from "ioredis";
import { QueueService } from "./queues.js";
import { QueueTaskType } from "./taskTypes.js";
import type { EmailTask, RunTask, SwarmTask } from "./taskTypes.js";
import { logger } from "../events/logger.js";

describe("QueueService", () => {
    let redisContainer: StartedTestContainer;
    let redisUrl: string;
    let queueService: QueueService;

    beforeAll(async () => {
        // Start a Redis container specifically for queue tests
        redisContainer = await new GenericContainer("redis:7-alpine")
            .withExposedPorts(6379)
            .start();

        const redisHost = redisContainer.getHost();
        const redisPort = redisContainer.getMappedPort(6379);
        redisUrl = `redis://${redisHost}:${redisPort}`;
    }, 60000);

    afterAll(async () => {
        if (redisContainer) {
            await redisContainer.stop();
        }
    });

    beforeEach(() => {
        // Get fresh instance for each test
        queueService = QueueService.get();
    });

    afterEach(async () => {
        // Ensure clean shutdown after each test
        await queueService.shutdown();
        // Clear singleton instance to ensure fresh state
        (QueueService as any).instance = null;
    });

    describe("Singleton pattern", () => {
        it("should return the same instance", () => {
            const instance1 = QueueService.get();
            const instance2 = QueueService.get();
            expect(instance1).toBe(instance2);
        });

        it("should return new instance after clearing", () => {
            const instance1 = QueueService.get();
            (QueueService as any).instance = null;
            const instance2 = QueueService.get();
            expect(instance1).not.toBe(instance2);
        });
    });

    describe("Initialization", () => {
        it("should initialize with valid Redis URL", async () => {
            await expect(queueService.init(redisUrl)).resolves.not.toThrow();
            
            // Verify connection is established
            const connection = (queueService as any).connection as IORedis;
            expect(connection).toBeDefined();
            expect(connection.status).toBe("ready");
        });

        it("should handle multiple concurrent init calls", async () => {
            // Start multiple init calls concurrently
            const initPromises = [
                queueService.init(redisUrl),
                queueService.init(redisUrl),
                queueService.init(redisUrl),
            ];

            // All should resolve without error
            await expect(Promise.all(initPromises)).resolves.not.toThrow();

            // Should have only one connection
            const connection = (queueService as any).connection as IORedis;
            expect(connection).toBeDefined();
            expect(connection.status).toBe("ready");
        });

        it("should handle init after shutdown", async () => {
            // First init
            await queueService.init(redisUrl);
            
            // Shutdown
            await queueService.shutdown();
            
            // Re-init should work
            await expect(queueService.init(redisUrl)).resolves.not.toThrow();
            
            const connection = (queueService as any).connection as IORedis;
            expect(connection).toBeDefined();
            expect(connection.status).toBe("ready");
        });

        it("should throw error with invalid Redis URL", async () => {
            const invalidUrl = "redis://invalid-host:6379";
            await expect(queueService.init(invalidUrl)).rejects.toThrow();
        });
    });

    describe("Queue operations", () => {
        beforeEach(async () => {
            await queueService.init(redisUrl);
        });

        describe("Email queue", () => {
            it("should add email task successfully", async () => {
                const emailTask: EmailTask = {
                    taskType: QueueTaskType.Email,
                    to: ["test@example.com"],
                    subject: "Test Email",
                    text: "This is a test email",
                    html: "<p>This is a test email</p>",
                };

                const job = await queueService.email.add(emailTask);
                expect(job).toBeDefined();
                expect(job.id).toBeDefined();
                expect(job.data).toEqual(emailTask);
            });

            it("should handle email task with delay", async () => {
                const emailTask: EmailTask = {
                    taskType: QueueTaskType.Email,
                    to: ["delayed@example.com"],
                    subject: "Delayed Email",
                    text: "This email should be delayed",
                };

                const delay = 5000; // 5 seconds
                const job = await queueService.email.add(emailTask, { delay });
                
                expect(job).toBeDefined();
                expect(job.opts.delay).toBe(delay);
            });

            it("should process email task", async () => {
                // Mock the email process function
                const processEmailMock = vi.fn().mockResolvedValue(undefined);
                vi.doMock("./email/process.js", () => ({
                    emailProcess: processEmailMock,
                }));

                const emailTask: EmailTask = {
                    taskType: QueueTaskType.Email,
                    to: ["process@example.com"],
                    subject: "Process Test",
                    text: "Testing email processing",
                };

                // Add task
                const job = await queueService.email.add(emailTask);
                
                // Wait a bit for processing
                await new Promise(resolve => setTimeout(resolve, 1000));

                // Verify the job was processed
                const completedJob = await queueService.email.queue.getJob(job.id!);
                expect(completedJob).toBeDefined();
            });
        });

        describe("Run queue", () => {
            it("should add run task with priority", async () => {
                const runTask: RunTask = {
                    taskType: QueueTaskType.Run,
                    runId: "test-run-123",
                    userId: "user-123",
                    hasPremium: true,
                };

                const priority = 10;
                const job = await queueService.run.add(runTask, { priority });
                
                expect(job).toBeDefined();
                expect(job.data).toEqual(runTask);
                expect(job.opts.priority).toBe(priority);
            });

            it("should respect run queue limits", async () => {
                // Add multiple run tasks
                const tasks: Promise<any>[] = [];
                for (let i = 0; i < 10; i++) {
                    const runTask: RunTask = {
                        taskType: QueueTaskType.Run,
                        runId: `run-${i}`,
                        userId: `user-${i}`,
                        hasPremium: false,
                    };
                    tasks.push(queueService.run.add(runTask));
                }

                // All should be added successfully
                const jobs = await Promise.all(tasks);
                expect(jobs).toHaveLength(10);
                jobs.forEach(job => expect(job).toBeDefined());
            });
        });

        describe("Swarm queue", () => {
            it("should add swarm task", async () => {
                const swarmTask: SwarmTask = {
                    taskType: QueueTaskType.Swarm,
                    conversationId: "conv-123",
                    userId: "user-123",
                    hasPremium: false,
                    swarmModel: "gpt-4",
                    execType: "chat",
                };

                const job = await queueService.swarm.add(swarmTask);
                expect(job).toBeDefined();
                expect(job.data).toEqual(swarmTask);
            });
        });
    });

    describe("Task status operations", () => {
        beforeEach(async () => {
            await queueService.init(redisUrl);
        });

        it("should get task status", async () => {
            // Add a task
            const emailTask: EmailTask = {
                taskType: QueueTaskType.Email,
                to: ["status@example.com"],
                subject: "Status Test",
                text: "Testing status",
            };
            const job = await queueService.email.add(emailTask);

            // Get status
            const statuses = await queueService.getTaskStatuses("user-123", [job.id!]);
            expect(statuses).toHaveLength(1);
            expect(statuses[0].id).toBe(job.id);
            expect(statuses[0].status).toBeDefined();
        });

        it("should return null status for non-existent task", async () => {
            const statuses = await queueService.getTaskStatuses("user-123", ["non-existent-id"]);
            expect(statuses).toHaveLength(1);
            expect(statuses[0].id).toBe("non-existent-id");
            expect(statuses[0].status).toBeNull();
        });

        it("should handle multiple task status queries", async () => {
            // Add multiple tasks
            const tasks = await Promise.all([
                queueService.email.add({
                    taskType: QueueTaskType.Email,
                    to: ["test1@example.com"],
                    subject: "Test 1",
                    text: "Test 1",
                }),
                queueService.email.add({
                    taskType: QueueTaskType.Email,
                    to: ["test2@example.com"],
                    subject: "Test 2", 
                    text: "Test 2",
                }),
            ]);

            const taskIds = tasks.map(t => t.id!);
            const statuses = await queueService.getTaskStatuses("user-123", taskIds);
            
            expect(statuses).toHaveLength(2);
            statuses.forEach((status, index) => {
                expect(status.id).toBe(taskIds[index]);
                expect(status.status).toBeDefined();
            });
        });
    });

    describe("Shutdown and cleanup", () => {
        it("should shutdown gracefully", async () => {
            await queueService.init(redisUrl);
            
            // Add some tasks
            await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["shutdown@example.com"],
                subject: "Shutdown test",
                text: "Testing shutdown",
            });

            // Shutdown should not throw
            await expect(queueService.shutdown()).resolves.not.toThrow();

            // Connection should be null after shutdown
            expect((queueService as any).connection).toBeNull();
            expect((queueService as any).queueInstances).toEqual({});
        });

        it("should handle shutdown without initialization", async () => {
            // Shutdown on uninitialized service should not throw
            await expect(queueService.shutdown()).resolves.not.toThrow();
        });

        it("should handle multiple shutdown calls", async () => {
            await queueService.init(redisUrl);
            
            // Multiple shutdowns should not throw
            await expect(queueService.shutdown()).resolves.not.toThrow();
            await expect(queueService.shutdown()).resolves.not.toThrow();
        });
    });

    describe("Reset functionality", () => {
        it("should reset and reinitialize", async () => {
            await queueService.init(redisUrl);
            
            // Add a task before reset
            const beforeJob = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["before@example.com"],
                subject: "Before reset",
                text: "Before reset",
            });
            expect(beforeJob).toBeDefined();

            // Reset
            await queueService.reset();

            // Should be able to add tasks after reset
            const afterJob = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["after@example.com"],
                subject: "After reset",
                text: "After reset",
            });
            expect(afterJob).toBeDefined();
        });
    });

    describe("Error handling", () => {
        it("should handle Redis connection errors gracefully", async () => {
            const spy = vi.spyOn(logger, "error");
            
            // Try to connect to invalid Redis
            await expect(queueService.init("redis://invalid:6379")).rejects.toThrow();
            
            // Should have logged error
            expect(spy).toHaveBeenCalled();
            spy.mockRestore();
        });

        it("should handle worker errors", async () => {
            await queueService.init(redisUrl);
            
            // Mock a failing processor
            const failingProcessor = vi.fn().mockRejectedValue(new Error("Processing failed"));
            vi.doMock("./email/process.js", () => ({
                emailProcess: failingProcessor,
            }));

            // Add task that will fail
            const job = await queueService.email.add({
                taskType: QueueTaskType.Email,
                to: ["fail@example.com"],
                subject: "Fail test",
                text: "This should fail",
            });

            // Wait for processing attempt
            await new Promise(resolve => setTimeout(resolve, 1000));

            // Job should be in failed state
            const failedJob = await queueService.email.queue.getJob(job.id!);
            expect(failedJob).toBeDefined();
        });
    });

    describe("Concurrent operations", () => {
        it("should handle concurrent queue additions", async () => {
            await queueService.init(redisUrl);
            
            // Add many tasks concurrently
            const promises: Promise<any>[] = [];
            for (let i = 0; i < 50; i++) {
                promises.push(
                    queueService.email.add({
                        taskType: QueueTaskType.Email,
                        to: [`concurrent${i}@example.com`],
                        subject: `Concurrent ${i}`,
                        text: `Concurrent test ${i}`,
                    })
                );
            }

            const jobs = await Promise.all(promises);
            expect(jobs).toHaveLength(50);
            jobs.forEach(job => {
                expect(job).toBeDefined();
                expect(job.id).toBeDefined();
            });
        });

        it("should handle concurrent status queries", async () => {
            await queueService.init(redisUrl);
            
            // Add tasks
            const jobs = await Promise.all(
                Array.from({ length: 20 }, (_, i) =>
                    queueService.email.add({
                        taskType: QueueTaskType.Email,
                        to: [`status${i}@example.com`],
                        subject: `Status ${i}`,
                        text: `Status test ${i}`,
                    })
                )
            );

            const jobIds = jobs.map(j => j.id!);

            // Query statuses concurrently
            const statusPromises = jobIds.map(id => 
                queueService.getTaskStatuses("user-123", [id])
            );

            const results = await Promise.all(statusPromises);
            expect(results).toHaveLength(20);
            results.forEach(statuses => {
                expect(statuses).toHaveLength(1);
                expect(statuses[0].status).toBeDefined();
            });
        });
    });
});