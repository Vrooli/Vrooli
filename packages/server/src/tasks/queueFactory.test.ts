import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import IORedis from "ioredis";
import "../__test/setup.js";
import { Job, Queue, Worker, QueueEvents } from "bullmq";
import { 
    buildRedis, 
    ManagedQueue, 
    DEFAULT_JOB_OPTIONS, 
    BASE_WORKER_OPTS,
    getQueueHealth,
    QueueStatus,
    type BaseQueueConfig,
    type BaseTaskData
} from "./queueFactory.js";
import { logger } from "../events/logger.js";

// Test task data type
interface TestTaskData extends BaseTaskData {
    type: "test";
    id: string;
    userId?: string;
    message: string;
    shouldFail?: boolean;
    processingTime?: number;
}

describe("queueFactory", () => {
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";
    let redisClient: IORedis;

    afterEach(async () => {
        // Clean up any remaining Redis connections
        const clients = (buildRedis as any).redisClients;
        const promises = (buildRedis as any).connectionEstablishmentPromises;
        
        if (clients) {
            // First wait for any pending connections to complete or fail
            if (promises) {
                await Promise.allSettled(Object.values(promises));
                Object.keys(promises).forEach(key => delete promises[key]);
            }
            
            // Then close existing connections
            for (const [url, client] of Object.entries(clients)) {
                if (client && typeof client === "object" && "quit" in client) {
                    try {
                        await (client as IORedis).quit();
                    } catch (error) {
                        // Ignore errors during cleanup
                    }
                }
            }
            // Clear the cache
            Object.keys(clients).forEach(key => delete clients[key]);
        }
        
        // Clear any module-level mocks
        vi.restoreAllMocks();
    });

    describe("buildRedis", () => {
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

    describe("ManagedQueue", () => {
        let testQueue: ManagedQueue<TestTaskData>;
        let processedJobs: TestTaskData[] = [];
        let processorMock: vi.Mock;

        beforeEach(async () => {
            redisClient = await buildRedis(redisUrl);
            processedJobs = [];
            
            // Create processor mock - reset for each test
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

            // Use a unique queue name for each test to avoid conflicts
            const uniqueQueueName = `test-queue-${Date.now()}-${Math.random()}`;
            
            const config: BaseQueueConfig<TestTaskData> = {
                name: uniqueQueueName,
                processor: processorMock,
            };

            testQueue = new ManagedQueue(config, redisClient);
            await testQueue.ready;
            
            // Wait for worker to be ready
            await new Promise(resolve => setTimeout(resolve, 100));
        });

        afterEach(async () => {
            // Clean up queue resources with proper sequencing to avoid race conditions
            if (testQueue) {
                // Wait for any pending operations to complete
                await new Promise(resolve => setTimeout(resolve, 100));
                
                // Close in proper order with error handling
                try {
                    await testQueue.worker.close();
                } catch (e) { 
                    // Ignore worker close errors during cleanup
                }
                
                try {
                    await testQueue.events.close();
                } catch (e) { 
                    // Ignore events close errors during cleanup
                }
                
                try {
                    await testQueue.queue.close();
                } catch (e) { 
                    // Ignore queue close errors during cleanup
                }
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
                    id: "test-1",
                    userId: "user-1",
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
                            id: `test-${i}`,
                            message: `Message ${i}`,
                        })
                    )
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
                    id: "default-opts",
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
                    id: "custom-opts",
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
                }, redisClient);
                
                try {
                    const delay = 1000; // 1 second
                    const startTime = Date.now();

                    const job = await delayedQueue.add({
                        type: "test",
                        id: "delayed-job",
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
                }, redisClient);

                try {
                    const job = await failQueue.add({
                        type: "test",
                        id: "failing-job",
                        message: "This will fail",
                        shouldFail: true,
                    }, { jobId: "failing-job" }); // Explicitly set the jobId

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
                        expect.stringContaining("job failed"),
                        expect.objectContaining({
                            jobId: "failing-job",
                            failedReason: expect.stringContaining("Test error"),
                        })
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
                }, redisClient);

                const job = await retryQueue.add({
                    type: "test",
                    id: "retry-job",
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
                }, redisClient);

                const result = await validatedQueue.addTask({
                    type: "test",
                    id: "validated-task",
                    message: "Testing validation",
                });

                expect(result.success).toBe(true);
                expect(validatorMock).toHaveBeenCalledWith({
                    type: "test",
                    id: "validated-task",
                    message: "Testing validation",
                });

                await validatedQueue.worker.close();
                await validatedQueue.events.close();
                await validatedQueue.queue.close();
            });

            it("should reject invalid tasks", async () => {
                const errorSpy = vi.spyOn(logger, "error");
                const validatorMock = vi.fn().mockReturnValue({ 
                    valid: false, 
                    errors: ["Missing required field", "Invalid format"] 
                });
                
                const validatedQueue = new ManagedQueue({
                    name: "validated-queue-reject",
                    processor: processorMock,
                    validator: validatorMock,
                }, redisClient);

                const result = await validatedQueue.addTask({
                    type: "test",
                    id: "invalid-task",
                    message: "Invalid",
                });

                expect(result.success).toBe(false);
                expect(errorSpy).toHaveBeenCalledWith(
                    expect.stringContaining("Task validation failed"),
                    expect.objectContaining({
                        errors: ["Missing required field", "Invalid format"],
                    })
                );

                errorSpy.mockRestore();
                await validatedQueue.worker.close();
                await validatedQueue.events.close();
                await validatedQueue.queue.close();
            });

            it("should get task statuses", async () => {
                // Add some tasks
                const taskIds = ["status-1", "status-2", "status-3"];
                await Promise.all(
                    taskIds.map(id => 
                        testQueue.add({
                            type: "test",
                            id,
                            message: `Status test ${id}`,
                        })
                    )
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
                const statuses = await testQueue.getTaskStatuses(["non-existent"]);
                expect(statuses).toEqual([{ id: "non-existent", status: null }]);
            });
        });

        describe("Worker configuration", () => {
            it("should apply worker options", async () => {
                const workerQueue = new ManagedQueue({
                    name: "worker-queue",
                    processor: processorMock,
                    workerOpts: { concurrency: 10, lockDuration: 5000 },
                }, redisClient);

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
                }, redisClient);

                // Wait for the queue to be ready
                await concurrentQueue.ready;

                // Add 10 jobs
                const jobs = await Promise.all(
                    Array.from({ length: 10 }, (_, i) =>
                        concurrentQueue.add({
                            type: "test",
                            id: `concurrent-${i}`,
                            message: `Concurrent ${i}`,
                        })
                    )
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
                }, redisClient);

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
                }, redisClient);

                try {
                    await expect(failingHookQueue.ready).rejects.toThrow("onReady failed");
                    expect(errorSpy).toHaveBeenCalledWith(
                        expect.stringContaining("onReady hook error"),
                        expect.any(Object)
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
            redisClient = await buildRedis(redisUrl);
            healthQueue = new Queue("health-test", { connection: redisClient });
        });

        afterEach(async () => {
            await healthQueue.close();
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
                    healthQueue.add("test", { data: `job${i}` })
                )
            );

            const health = await getQueueHealth(healthQueue, { backlog: 500, failed: 25 });
            
            expect(health.status).toBe(QueueStatus.Degraded);
            expect(health.jobCounts.waiting + health.jobCounts.delayed).toBeGreaterThan(500);
        });

        it("should report down queue with failed jobs", async () => {
            // Create a worker that always fails
            const failWorker = new Worker("health-test", async () => {
                throw new Error("Always fails");
            }, { connection: redisClient, concurrency: 10 }); // Higher concurrency to process faster

            try {
                // Add jobs that will fail
                const jobs = await Promise.all(
                    Array.from({ length: 30 }, (_, i) =>
                        healthQueue.add("test", { data: `fail${i}` }, { attempts: 1 })
                    )
                );

                // Wait for all jobs to fail
                let failedCount = 0;
                await new Promise<void>((resolve) => {
                    const queueEvents = new QueueEvents("health-test", { connection: redisClient });
                    queueEvents.on("failed", () => {
                        failedCount++;
                        if (failedCount >= 30) {
                            queueEvents.close();
                            resolve();
                        }
                    });
                    // Fallback timeout
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
            // Create a slow worker
            const slowWorker = new Worker("health-test", async () => {
                await new Promise(resolve => setTimeout(resolve, 5000));
            }, { connection: redisClient, concurrency: 2 });

            try {
                // Add jobs
                const jobs = await Promise.all([
                    healthQueue.add("slow-job-1", { data: "slow1" }),
                    healthQueue.add("slow-job-2", { data: "slow2" }),
                ]);

                // Wait for jobs to become active
                await new Promise<void>((resolve) => {
                    const queueEvents = new QueueEvents("health-test", { connection: redisClient });
                    let activeCount = 0;
                    queueEvents.on("active", () => {
                        activeCount++;
                        if (activeCount >= 2) {
                            queueEvents.close();
                            resolve();
                        }
                    });
                    // Fallback timeout
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
    });
});