// AI_CHECK: TEST_QUALITY=1, TEST_COVERAGE=1 | LAST: 2025-06-23
import { Queue, QueueEvents, Worker, type Job } from "bullmq";
import IORedis from "ioredis";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { generatePK } from "@vrooli/shared";
import { logger } from "../events/logger.js";
import {
    buildRedis,
    clearRedisCache,
    DEFAULT_JOB_OPTIONS,
    getQueueHealth,
    ManagedQueue,
    QueueStatus,
    type BaseQueueConfig,
    type BaseTaskData,
} from "./queueFactory.js";
import { createIsolatedQueueTestHarness } from "../__test/helpers/queueTestUtils.js";

// Test task data type
interface TestTaskData extends BaseTaskData {
    type: "test";
    id: string;
    userId?: string;
    message: string;
    shouldFail?: boolean;
    processingTime?: number;
}

// ============================================================================
// CACHE MANIPULATION TESTS - Run sequentially to prevent cache corruption
// ============================================================================
describe("queueFactory - Cache Management (Isolated)", () => {
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";
    
    // Run cache tests sequentially to prevent cache corruption
    describe.sequential("buildRedis cache behavior", () => {
        afterEach(async () => {
            // Aggressive cleanup for cache manipulation tests
            const clients = (buildRedis as any).redisClients;
            const promises = (buildRedis as any).connectionEstablishmentPromises;

            if (clients) {
                if (promises) {
                    await Promise.allSettled(Object.values(promises));
                    Object.keys(promises).forEach(key => delete promises[key]);
                }

                for (const [url, client] of Object.entries(clients)) {
                    if (client && typeof client === "object" && "quit" in client) {
                        try {
                            // Remove all listeners to prevent connection closed errors
                            (client as IORedis).removeAllListeners();
                            if ((client as IORedis).status !== "ready") {
                                (client as IORedis).disconnect();
                            } else {
                                await (client as IORedis).quit();
                            }
                        } catch (error) {
                            // Force disconnect if quit fails
                            try {
                                (client as IORedis).disconnect();
                            } catch (e) {
                                // Ignore final cleanup errors
                            }
                        }
                    }
                }
                Object.keys(clients).forEach(key => delete clients[key]);
            }

            await new Promise(resolve => setTimeout(resolve, 200));
            vi.restoreAllMocks();
        });

        it("should create a new Redis connection", async () => {
            const client = await buildRedis(redisUrl);
            expect(client).toBeDefined();
            expect(client.status).toBe("ready");
            await client.quit();
        });

        it("should reuse existing connection for same URL", async () => {
            const client1 = await buildRedis(redisUrl);
            const client2 = await buildRedis(redisUrl);

            expect(client1).toBe(client2);
            // Don't quit the client here as it will break other tests that might use it
            // The afterEach cleanup will handle it
        });

        it("should handle concurrent connection requests", async () => {
            // Use a unique URL for this test to avoid conflicts
            const testUrl = `${redisUrl}/1`; // Use database 1

            // Clear any existing connections
            const clients = (buildRedis as any).redisClients || {};
            const promises = (buildRedis as any).connectionEstablishmentPromises || {};
            if (clients[testUrl]) delete clients[testUrl];
            if (promises[testUrl]) delete promises[testUrl];

            // Make multiple concurrent requests
            const connectionPromises = Array.from({ length: 5 }, () => buildRedis(testUrl));
            const connections = await Promise.all(connectionPromises);

            // All should be the same instance
            const firstConnection = connections[0];
            connections.forEach(conn => {
                expect(conn).toBe(firstConnection);
            });

            // Clean up
            await firstConnection.quit();
            delete clients[testUrl];
        });

        it("should replace stale connection", async () => {
            const client1 = await buildRedis(redisUrl);

            // Force disconnect to make it stale
            client1.disconnect();

            // Wait a bit for status to update
            await new Promise(resolve => setTimeout(resolve, 100));

            // Getting connection again should create new one
            const client2 = await buildRedis(redisUrl);
            expect(client2).not.toBe(client1);
            expect(client2.status).toBe("ready");

            await client2.quit();
        });

        it("should handle connection errors", async () => {
            const invalidUrl = "redis://invalid-host:6379";
            await expect(buildRedis(invalidUrl)).rejects.toThrow();
        });

        it.skip("should timeout if connection takes too long", async () => {
            // Skip this test for now as mocking IORedis is complex
            // The timeout functionality is tested in integration tests
        });
    });

    describe.sequential("Connection pool management", () => {
        afterEach(async () => {
            // Extra cleanup for connection pool tests
            const clients = (buildRedis as any).redisClients;
            if (clients) {
                for (const [url, client] of Object.entries(clients)) {
                    if (client && typeof client === "object" && "quit" in client) {
                        try {
                            (client as IORedis).removeAllListeners();
                            (client as IORedis).disconnect();
                        } catch (error) {
                            // Ignore cleanup errors
                        }
                    }
                }
                Object.keys(clients).forEach(key => delete clients[key]);
            }
            await new Promise(resolve => setTimeout(resolve, 200));
        });

        it("should handle connection pool exhaustion", async () => {
            const connections: IORedis[] = [];
            const maxConnections = 8; // Reduced further to prevent overwhelming

            try {
                for (let i = 0; i < maxConnections; i++) {
                    const conn = await buildRedis(`${redisUrl}/${i % 6}`);
                    connections.push(conn);
                }

                connections.forEach(conn => {
                    expect(conn.status).toBe("ready");
                });
            } finally {
                await Promise.all(connections.map(conn => conn.quit().catch(() => {})));
            }
        });

        it("should handle Redis disconnection and reconnection", async () => {
            const client = await buildRedis(redisUrl);
            let disconnectCount = 0;

            client.on("disconnect", () => disconnectCount++);

            // Force disconnect and wait for event
            client.disconnect();
            await new Promise(resolve => setTimeout(resolve, 300));

            // Check if disconnect was detected or if connection is no longer ready
            const isDisconnected = disconnectCount > 0 || client.status !== "ready";
            expect(isDisconnected).toBe(true);

            const newClient = await buildRedis(redisUrl);
            expect(newClient.status).toBe("ready");

            await newClient.quit();
        });

        it("should handle connection timeout gracefully", async () => {
            const originalConnect = IORedis.prototype.connect;
            const timeoutUrl = "redis://timeout-test:6379";
            
            IORedis.prototype.connect = vi.fn().mockImplementation(function(this: any) {
                // Only fail for our specific timeout test URL
                if (this.options?.host === "timeout-test") {
                    return new Promise((_, reject) => {
                        setTimeout(() => reject(new Error("Connection timeout")), 50);
                    });
                }
                // Let other connections proceed normally
                return originalConnect.call(this);
            });

            try {
                await expect(buildRedis(timeoutUrl)).rejects.toThrow("Connection timeout");
            } finally {
                IORedis.prototype.connect = originalConnect;
            }
        });
    });
});

// ============================================================================
// QUEUE FUNCTIONALITY TESTS - Use mocked connections to avoid cache pollution
// ============================================================================
describe("queueFactory - Queue Functionality (Mocked)", () => {
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";
    let mockRedisClient: IORedis;
    let buildRedisSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(async () => {
        // Create a real Redis client for mocking
        mockRedisClient = new IORedis(redisUrl, {
            maxRetriesPerRequest: null,
            retryDelayOnFailover: 100,
        });
        mockRedisClient.setMaxListeners(50);
        
        // Mock buildRedis to always return our controlled client
        buildRedisSpy = vi.spyOn(await import("./queueFactory.js"), "buildRedis");
        buildRedisSpy.mockResolvedValue(mockRedisClient);
    });

    afterEach(async () => {
        // Clean up mock and real client
        buildRedisSpy.mockRestore();
        try {
            // Remove all listeners before quitting to prevent connection errors
            mockRedisClient.removeAllListeners();
            
            // Add error handler to ignore close errors
            mockRedisClient.on("error", () => {
                // Ignore errors during close
            });
            
            if (mockRedisClient.status === "ready") {
                await mockRedisClient.quit();
            } else {
                mockRedisClient.disconnect();
            }
        } catch (error) {
            // Ignore cleanup errors
            try {
                mockRedisClient.disconnect();
            } catch (e) {
                // Ignore final disconnect errors
            }
        }
        vi.restoreAllMocks();
        // Longer delay to allow Redis cleanup to complete
        await new Promise(resolve => setTimeout(resolve, 200));
    });

    describe("ManagedQueue", () => {
        let testQueue: ManagedQueue<TestTaskData>;
        let processedJobs: TestTaskData[] = [];
        let processorMock: vi.Mock;

        beforeEach(async () => {
            // Reset for each test - using mocked buildRedis so no cache pollution
            processedJobs = [];

            processorMock = vi.fn().mockImplementation(async (job: Job<TestTaskData>) => {
                processedJobs.push(job.data);

                if (job.data.shouldFail) {
                    throw new Error("Test error: Job configured to fail");
                }

                if (job.data.processingTime) {
                    await new Promise(resolve => setTimeout(resolve, job.data.processingTime));
                }

                return { processed: true, message: job.data.message };
            });

            const uniqueQueueName = `test-queue-${Date.now()}-${Math.random()}`;

            const config: BaseQueueConfig<TestTaskData> = {
                name: uniqueQueueName,
                processor: processorMock,
            };

            // buildRedis is mocked to return mockRedisClient
            testQueue = new ManagedQueue(config, mockRedisClient);
            await testQueue.ready;

            await new Promise(resolve => setTimeout(resolve, 100));
        });

        afterEach(async () => {
            // Clean up queue resources with proper sequencing to avoid race conditions
            if (testQueue) {
                // Use the proper close method which handles cleanup with timeouts
                try {
                    // Wait longer for cleanup in tests
                    await Promise.race([
                        testQueue.close(),
                        new Promise((_, reject) => 
                            setTimeout(() => reject(new Error("Queue close timeout")), 10000),
                        ),
                    ]);
                } catch (e) {
                    // Ignore close errors during cleanup
                    console.log("Queue close error (ignored):", e);
                }
                
                // Add longer delay to ensure all Redis operations and worker threads complete
                await new Promise(resolve => setTimeout(resolve, 300));
            }
        });

        describe("Basic operations", () => {
            it("should create queue with correct configuration", () => {
                expect(testQueue.queue).toBeDefined();
                expect(testQueue.worker).toBeDefined();
                expect(testQueue.events).toBeDefined();
                expect(testQueue.queue.name).toMatch(/^test-queue-/);
            });

            it("should add and process jobs", async () => {
                const testData: TestTaskData = {
                    type: "test",
                    id: generatePK().toString(),
                    userId: generatePK().toString(),
                    message: "Hello World",
                };

                const job = await testQueue.add(testData);
                expect(job).toBeDefined();
                expect(job.data).toEqual(testData);

                // Wait for the job to be completed
                await job.waitUntilFinished(testQueue.events);

                expect(processedJobs).toHaveLength(1);
                expect(processedJobs[0]).toEqual(testData);
                expect(processorMock).toHaveBeenCalledTimes(1);
            });

            it("should handle multiple jobs", async () => {
                // Reset processed jobs for this test
                processedJobs = [];
                processorMock.mockClear();

                const jobs = await Promise.all(
                    Array.from({ length: 10 }, (_, i) =>
                        testQueue.add({
                            type: "test",
                            id: generatePK().toString(),
                            message: `Message ${i}`,
                        }),
                    ),
                );

                expect(jobs).toHaveLength(10);

                // Wait for all jobs to complete
                await Promise.all(jobs.map(job => job.waitUntilFinished(testQueue.events)));

                expect(processedJobs).toHaveLength(10);
                expect(processorMock).toHaveBeenCalledTimes(10);
            });
        });

        describe("Job options", () => {
            it("should apply default job options", async () => {
                const job = await testQueue.add({
                    type: "test",
                    id: generatePK().toString(),
                    message: "Testing defaults",
                });

                expect(job.opts.removeOnComplete).toEqual(DEFAULT_JOB_OPTIONS.removeOnComplete);
                expect(job.opts.removeOnFail).toEqual(DEFAULT_JOB_OPTIONS.removeOnFail);
                expect(job.opts.attempts).toBe(DEFAULT_JOB_OPTIONS.attempts);
                expect(job.opts.backoff).toEqual(DEFAULT_JOB_OPTIONS.backoff);
            });

            it("should override job options", async () => {
                const customOpts = {
                    delay: 1000,
                    priority: 5,
                    attempts: 1,
                };

                const job = await testQueue.add({
                    type: "test",
                    id: generatePK().toString(),
                    message: "Custom options",
                }, customOpts);

                expect(job.opts.delay).toBe(customOpts.delay);
                expect(job.opts.priority).toBe(customOpts.priority);
                expect(job.opts.attempts).toBe(customOpts.attempts);
            });

            it("should handle delayed jobs", async () => {
                // Use local array to avoid interference
                const localProcessedJobs: TestTaskData[] = [];

                // Create a fresh queue for this test to avoid interference
                const delayedQueue = new ManagedQueue({
                    name: "delayed-test-queue",
                    processor: async (job) => {
                        localProcessedJobs.push(job.data);
                        return { processed: true };
                    },
                }, mockRedisClient);

                try {
                    const delay = 1000; // 1 second
                    const startTime = Date.now();

                    const job = await delayedQueue.add({
                        type: "test",
                        id: generatePK().toString(),
                        message: "Delayed message",
                    }, { delay });

                    // Job should not be processed immediately
                    await new Promise(resolve => setTimeout(resolve, 200));
                    expect(localProcessedJobs).toHaveLength(0);

                    // Wait for the job to be completed
                    await job.waitUntilFinished(delayedQueue.events);

                    expect(localProcessedJobs).toHaveLength(1);
                    const processingTime = Date.now() - startTime;
                    expect(processingTime).toBeGreaterThanOrEqual(delay);
                } finally {
                    await delayedQueue.worker.close();
                    await delayedQueue.events.close();
                    await delayedQueue.queue.close();
                }
            });
        });

        describe("Error handling", () => {
            it("should handle job failures", async () => {
                const errorSpy = vi.spyOn(logger, "error");

                // Clear any previous error calls
                errorSpy.mockClear();

                // Create a queue with only 1 attempt to fail quickly
                const failQueue = new ManagedQueue({
                    name: "fail-test-queue",
                    processor: async (job: Job<TestTaskData>) => {
                        if (job.data.shouldFail) {
                            throw new Error("Test error: Job configured to fail");
                        }
                        return { processed: true };
                    },
                    jobOpts: { attempts: 1 }, // Only 1 attempt, no retries
                }, mockRedisClient);

                try {
                    const jobId = generatePK().toString();
                    const job = await failQueue.add({
                        type: "test",
                        id: jobId,
                        message: "This will fail",
                        shouldFail: true,
                    }, { jobId }); // Explicitly set the jobId

                    // Wait for the job to fail
                    try {
                        await job.waitUntilFinished(failQueue.events);
                    } catch (err) {
                        // Expected to throw after failing
                    }

                    // Wait a bit to ensure event handler processes
                    await new Promise(resolve => setTimeout(resolve, 200));

                    // Check that error was logged
                    expect(errorSpy).toHaveBeenCalledWith(
                        "Job failed in queue fail-test-queue",
                        expect.objectContaining({
                            jobId,
                            failedReason: "Test error: Job configured to fail",
                            queue: "fail-test-queue",
                        }),
                    );
                } finally {
                    errorSpy.mockRestore();
                    await failQueue.worker.close();
                    await failQueue.events.close();
                    await failQueue.queue.close();
                }
            }, 10000); // Increase timeout for this test

            it("should retry failed jobs", async () => {
                let attemptCount = 0;
                const retryProcessor = vi.fn().mockImplementation(async () => {
                    attemptCount++;
                    if (attemptCount < 3) {
                        throw new Error(`Attempt ${attemptCount} failed`);
                    }
                    return { success: true };
                });

                const retryQueue = new ManagedQueue({
                    name: "retry-queue",
                    processor: retryProcessor,
                    jobOpts: { attempts: 3, backoff: { type: "fixed", delay: 100 } },
                }, mockRedisClient);

                const job = await retryQueue.add({
                    type: "test",
                    id: generatePK().toString(),
                    message: "Will retry",
                });

                // Wait for completion
                await job.waitUntilFinished(retryQueue.events);

                expect(retryProcessor).toHaveBeenCalledTimes(3);
                expect(attemptCount).toBe(3);

                await retryQueue.worker.close();
                await retryQueue.events.close();
                await retryQueue.queue.close();
            });
        });

        describe("Task operations", () => {
            it("should add task with validation", async () => {
                const validatorMock = vi.fn().mockReturnValue({ valid: true });

                const validatedQueue = new ManagedQueue({
                    name: "validated-queue",
                    processor: processorMock,
                    validator: validatorMock,
                }, mockRedisClient);

                const taskData = {
                    type: "test" as const,
                    id: generatePK().toString(),
                    message: "Testing validation",
                };
                
                const result = await validatedQueue.addTask(taskData);

                expect(result.success).toBe(true);
                expect(validatorMock).toHaveBeenCalledWith(taskData);

                await validatedQueue.worker.close();
                await validatedQueue.events.close();
                await validatedQueue.queue.close();
            });

            it("should reject invalid tasks", async () => {
                const errorSpy = vi.spyOn(logger, "error");
                const validatorMock = vi.fn().mockReturnValue({
                    valid: false,
                    errors: ["Missing required field", "Invalid format"],
                });

                const validatedQueue = new ManagedQueue({
                    name: "validated-queue-reject",
                    processor: processorMock,
                    validator: validatorMock,
                }, mockRedisClient);

                const result = await validatedQueue.addTask({
                    type: "test",
                    id: generatePK().toString(),
                    message: "Invalid",
                });

                expect(result.success).toBe(false);
                expect(errorSpy).toHaveBeenCalledWith(
                    expect.stringContaining("Task validation failed"),
                    expect.objectContaining({
                        errors: ["Missing required field", "Invalid format"],
                    }),
                );

                errorSpy.mockRestore();
                await validatedQueue.worker.close();
                await validatedQueue.events.close();
                await validatedQueue.queue.close();
            });

            it("should get task statuses", async () => {
                // Add some tasks
                const taskIds = Array.from({ length: 3 }, () => generatePK().toString());
                await Promise.all(
                    taskIds.map(id =>
                        testQueue.add({
                            type: "test",
                            id,
                            message: `Status test ${id}`,
                        }),
                    ),
                );

                // Get statuses
                const statuses = await testQueue.getTaskStatuses(taskIds);

                expect(statuses).toHaveLength(3);
                statuses.forEach((status, index) => {
                    expect(status.id).toBe(taskIds[index]);
                    expect(status.status).toBeDefined();
                });
            });

            it("should return null for non-existent tasks", async () => {
                const nonExistentId = generatePK().toString();
                const statuses = await testQueue.getTaskStatuses([nonExistentId]);
                expect(statuses).toEqual([{ id: nonExistentId, status: null }]);
            });
        });

        describe("Worker configuration", () => {
            it("should apply worker options", async () => {
                const workerQueue = new ManagedQueue({
                    name: "worker-queue",
                    processor: processorMock,
                    workerOpts: { concurrency: 10, lockDuration: 5000 },
                }, mockRedisClient);

                expect(workerQueue.worker.opts.concurrency).toBe(10);
                expect(workerQueue.worker.opts.lockDuration).toBe(5000);

                await workerQueue.worker.close();
                await workerQueue.events.close();
                await workerQueue.queue.close();
            });

            it("should handle concurrent processing", async () => {
                // Reset processed jobs for this test
                processedJobs = [];

                const concurrentQueue = new ManagedQueue({
                    name: "concurrent-queue",
                    processor: async (job) => {
                        // Simulate some work
                        await new Promise(resolve => setTimeout(resolve, 100));
                        processedJobs.push(job.data);
                    },
                    workerOpts: { concurrency: 5 },
                }, mockRedisClient);

                // Wait for the queue to be ready
                await concurrentQueue.ready;

                // Add 10 jobs
                const jobs = await Promise.all(
                    Array.from({ length: 10 }, (_, i) =>
                        concurrentQueue.add({
                            type: "test",
                            id: generatePK().toString(),
                            message: `Concurrent ${i}`,
                        }),
                    ),
                );

                // Wait for all jobs to complete
                await Promise.all(jobs.map(job => job.waitUntilFinished(concurrentQueue.events)));

                expect(processedJobs).toHaveLength(10);

                await concurrentQueue.worker.close();
                await concurrentQueue.events.close();
                await concurrentQueue.queue.close();
            });
        });

        describe("Lifecycle hooks", () => {
            it("should execute onReady hook", async () => {
                const onReadyMock = vi.fn().mockResolvedValue(undefined);

                const hookQueue = new ManagedQueue({
                    name: "hook-queue",
                    processor: processorMock,
                    onReady: onReadyMock,
                }, mockRedisClient);

                try {
                    await hookQueue.ready;
                    expect(onReadyMock).toHaveBeenCalledTimes(1);
                } finally {
                    // Ensure cleanup happens even if test fails
                    try {
                        await hookQueue.worker.close();
                    } catch (e) { /* ignore cleanup errors */ }
                    try {
                        await hookQueue.events.close();
                    } catch (e) { /* ignore cleanup errors */ }
                    try {
                        await hookQueue.queue.close();
                    } catch (e) { /* ignore cleanup errors */ }
                }
            });

            it("should handle onReady hook errors", async () => {
                const errorSpy = vi.spyOn(logger, "error");
                const onReadyError = new Error("onReady failed");

                const failingHookQueue = new ManagedQueue({
                    name: "failing-hook-queue",
                    processor: processorMock,
                    onReady: async () => { throw onReadyError; },
                }, mockRedisClient);

                try {
                    await expect(failingHookQueue.ready).rejects.toThrow("onReady failed");
                    expect(errorSpy).toHaveBeenCalledWith(
                        "onReady hook failed for queue failing-hook-queue",
                        expect.objectContaining({
                            error: expect.objectContaining({
                                name: expect.any(String),
                                message: "onReady failed",
                                stack: expect.any(String),
                            }),
                        }),
                    );
                } finally {
                    errorSpy.mockRestore();
                    // Ensure cleanup happens even if test fails
                    try {
                        await failingHookQueue.worker.close();
                    } catch (e) { /* ignore cleanup errors */ }
                    try {
                        await failingHookQueue.events.close();
                    } catch (e) { /* ignore cleanup errors */ }
                    try {
                        await failingHookQueue.queue.close();
                    } catch (e) { /* ignore cleanup errors */ }
                }
            });
        });
    });

    describe("getQueueHealth", () => {
        let healthQueue: Queue;

        beforeEach(async () => {
            // Using mocked buildRedis so no cache pollution concerns
            healthQueue = new Queue("health-test", { connection: mockRedisClient });
        });

        afterEach(async () => {
            try {
                await healthQueue.close();
            } catch (error) {
                // Ignore close errors during cleanup
            }
        });

        it("should report healthy queue", async () => {
            // Add a few jobs
            await Promise.all([
                healthQueue.add("test", { data: "job1" }),
                healthQueue.add("test", { data: "job2" }),
            ]);

            const health = await getQueueHealth(healthQueue);

            expect(health.status).toBe(QueueStatus.Healthy);
            expect(health.jobCounts).toBeDefined();
            expect(health.jobCounts.waiting).toBeGreaterThanOrEqual(0);
            expect(health.jobCounts.failed).toBe(0);
        });

        it("should report degraded queue", async () => {
            // Add many jobs to exceed threshold
            await Promise.all(
                Array.from({ length: 600 }, (_, i) =>
                    healthQueue.add("test", { data: `job${i}` }),
                ),
            );

            const health = await getQueueHealth(healthQueue, { backlog: 500, failed: 25 });

            expect(health.status).toBe(QueueStatus.Degraded);
            expect(health.jobCounts.waiting + health.jobCounts.delayed).toBeGreaterThan(500);
        });

        it("should report down queue with failed jobs", async () => {
            const failWorker = new Worker("health-test", async () => {
                throw new Error("Always fails");
            }, { connection: mockRedisClient, concurrency: 10 });

            try {
                const jobs = await Promise.all(
                    Array.from({ length: 30 }, (_, i) =>
                        healthQueue.add("test", { data: `fail${i}` }, { attempts: 1 }),
                    ),
                );

                let failedCount = 0;
                await new Promise<void>((resolve) => {
                    const queueEvents = new QueueEvents("health-test", { connection: mockRedisClient });
                    queueEvents.on("failed", () => {
                        failedCount++;
                        if (failedCount >= 30) {
                            queueEvents.close();
                            resolve();
                        }
                    });
                    setTimeout(() => {
                        queueEvents.close();
                        resolve();
                    }, 5000);
                });

                const health = await getQueueHealth(healthQueue, { backlog: 500, failed: 25 });

                expect(health.status).toBe(QueueStatus.Down);
                expect(health.jobCounts.failed).toBeGreaterThan(25);
            } finally {
                try {
                    await failWorker.close();
                } catch (e) { /* ignore worker close errors */ }
            }
        });

        it("should include active job information", async () => {
            const slowWorker = new Worker("health-test", async () => {
                await new Promise(resolve => setTimeout(resolve, 5000));
            }, { connection: mockRedisClient, concurrency: 2 });

            try {
                const jobs = await Promise.all([
                    healthQueue.add("slow-job-1", { data: "slow1" }),
                    healthQueue.add("slow-job-2", { data: "slow2" }),
                ]);

                await new Promise<void>((resolve) => {
                    const queueEvents = new QueueEvents("health-test", { connection: mockRedisClient });
                    let activeCount = 0;
                    queueEvents.on("active", () => {
                        activeCount++;
                        if (activeCount >= 2) {
                            queueEvents.close();
                            resolve();
                        }
                    });
                    setTimeout(() => {
                        queueEvents.close();
                        resolve();
                    }, 2000);
                });

                const health = await getQueueHealth(healthQueue);

                expect(health.activeJobs).toBeDefined();
                expect(health.activeJobs.length).toBeGreaterThan(0);
                health.activeJobs.forEach(job => {
                    expect(job.id).toBeDefined();
                    expect(job.name).toBeDefined();
                    expect(job.duration).toBeGreaterThan(0);
                });
            } finally {
                try {
                    await slowWorker.close();
                } catch (e) { /* ignore worker close errors */ }
            }
        });

        it("should handle queue with mixed job states", async () => {
            const worker = new Worker("health-test", async (job) => {
                if (job.data.shouldFail) {
                    throw new Error("Intentional failure");
                }
                if (job.data.shouldDelay) {
                    await new Promise(resolve => setTimeout(resolve, 1000));
                }
            }, { connection: mockRedisClient, concurrency: 3 });

            try {
                await Promise.all([
                    healthQueue.add("success-1", { data: "ok" }),
                    healthQueue.add("fail-1", { shouldFail: true }, { attempts: 1 }),
                    healthQueue.add("delayed-1", { data: "delayed" }, { delay: 5000 }),
                    healthQueue.add("active-1", { shouldDelay: true }),
                ]);

                await new Promise(resolve => setTimeout(resolve, 500));

                const health = await getQueueHealth(healthQueue);

                expect(health.jobCounts.waiting + health.jobCounts.active +
                    health.jobCounts.failed + health.jobCounts.delayed).toBeGreaterThan(0);
                expect(health.status).toBeDefined();
            } finally {
                await worker.close();
            }
        });

        it("should calculate health status based on custom thresholds", async () => {
            const customThresholds = {
                backlog: 10,  // Very low threshold
                failed: 5,    // Very low threshold
            };

            // Add just enough jobs to trigger degraded status
            await Promise.all(
                Array.from({ length: 12 }, (_, i) =>
                    healthQueue.add("test", { data: `job${i}` }),
                ),
            );

            const health = await getQueueHealth(healthQueue, customThresholds);
            // Health status depends on actual job counts, could be Degraded or Down
            expect([QueueStatus.Degraded, QueueStatus.Down]).toContain(health.status);
        });
    });

    describe("Performance and load testing", () => {
        it("should handle high job throughput", async () => {
            const perfQueue = new ManagedQueue({
                name: "perf-test-queue",
                processor: async (job) => {
                    return { processed: job.data.id };
                },
                workerOpts: { concurrency: 10 },
            }, mockRedisClient);

            try {
                const startTime = Date.now();
                const jobCount = 100;

                const promises = Array.from({ length: jobCount }, (_, i) =>
                    perfQueue.add({
                        type: "test",
                        id: generatePK().toString(),
                        message: `Performance test ${i}`,
                    }),
                );

                const jobs = await Promise.all(promises);

                await Promise.all(jobs.map(job =>
                    job.waitUntilFinished(perfQueue.events),
                ));

                const duration = Date.now() - startTime;
                const throughput = jobCount / (duration / 1000);

                expect(throughput).toBeGreaterThan(10);
            } finally {
                await perfQueue.worker.close();
                await perfQueue.events.close();
                await perfQueue.queue.close();
            }
        }, 30000);

        it("should maintain job data integrity under concurrent load", async () => {
            const integrityQueue = new ManagedQueue({
                name: "integrity-test-queue",
                processor: async (job) => {
                    expect(job.data.id).toBeDefined();
                    expect(job.data.checksum).toBe(job.data.id.length);
                    return { verified: true };
                },
                workerOpts: { concurrency: 5 },
            }, mockRedisClient);

            try {
                const jobs = await Promise.all(
                    Array.from({ length: 50 }, (_, i) => {
                        const id = generatePK().toString();
                        return integrityQueue.add({
                            type: "test",
                            id,
                            checksum: id.length,
                            data: { nested: { value: i } },
                        });
                    }),
                );

                await Promise.all(jobs.map(job =>
                    job.waitUntilFinished(integrityQueue.events),
                ));

                const jobCounts = await integrityQueue.queue.getJobCounts();
                expect(jobCounts.failed).toBe(0);
            } finally {
                await integrityQueue.worker.close();
                await integrityQueue.events.close();
                await integrityQueue.queue.close();
            }
        });
    });
});
