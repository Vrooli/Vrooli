import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import IORedis from "ioredis";
import { Queue, Worker, QueueEvents } from "bullmq";
import { ManagedQueue, buildRedis, clearRedisCache } from "./queueFactory.js";
import { QueueService } from "./queues.js";
import { QueueTaskType } from "./taskTypes.js";
import { logger } from "../events/logger.js";
import {
    createQueueTestHarness,
    createIsolatedQueueTestHarness,
    simulateRedisDisconnection,
    simulateRedisMemoryExhaustion,
    simulateNetworkPartition,
    createMemoryHogWorker,
    createCrashingWorker,
    createTimeoutWorker,
    getQueueHealthMetrics,
    waitForQueueState,
    generateQueueLoad,
    measureQueuePerformance,
    createRedisSpy,
    mockEnvVars,
} from "../__test/helpers/queueTestUtils.js";
import "../__test/setup.js";

// Unmock QueueService for this test since we want to test the real implementation
vi.unmock("./queues.js");

describe("Queue System Resilience Tests", () => {
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";
    let queueService: QueueService;
    let harness: Awaited<ReturnType<typeof createQueueTestHarness>>;

    beforeEach(async () => {
        queueService = QueueService.get();
        await queueService.init(redisUrl);
        harness = await createQueueTestHarness(redisUrl);
    });

    afterEach(async () => {
        await harness.cleanup();
        try {
            await queueService.shutdown();
            // Wait for shutdown to fully complete
            await new Promise(resolve => setTimeout(resolve, 100));
        } catch (error) {
            console.log("Shutdown error (ignored):", error);
        }
        // Clear singleton before clearing cache
        (QueueService as any).instance = null;
        // Clear Redis cache last
        clearRedisCache();
        // Final delay
        await new Promise(resolve => setTimeout(resolve, 50));
    });

    describe("Redis Connection Resilience", () => {
        it("should handle Redis disconnection during job processing", async () => {
            // Use isolated harness for connection manipulation tests
            const isolatedHarness = await createIsolatedQueueTestHarness(redisUrl);
            
            // Create a test queue with a slow processor
            const testQueue = new Queue("redis-disconnect-test", { connection: isolatedHarness.isolatedConnection });
            const processedJobs: any[] = [];
            
            const worker = new Worker("redis-disconnect-test", async (job) => {
                processedJobs.push(job.data);
                // Simulate slow processing
                await new Promise(resolve => setTimeout(resolve, 2000));
                return { processed: true };
            }, { connection: isolatedHarness.isolatedConnection });

            isolatedHarness.queues.push(testQueue);
            isolatedHarness.workers.push(worker);

            // Add jobs before disconnection
            const job1 = await testQueue.add("test-job-1", { message: "Before disconnect" });
            const job2 = await testQueue.add("test-job-2", { message: "During disconnect" });

            // Wait a bit for first job to start
            await new Promise(resolve => setTimeout(resolve, 500));

            // Simulate Redis disconnection
            simulateRedisDisconnection(isolatedHarness.isolatedConnection);

            // Wait for reconnection attempt
            await new Promise(resolve => setTimeout(resolve, 3000));

            // Create new connection and verify job recovery
            const newRedis = new IORedis(redisUrl);
            const recoveryQueue = new Queue("redis-disconnect-test", { connection: newRedis });
            
            // Register for cleanup
            isolatedHarness.queues.push(recoveryQueue);
            
            // Check that jobs are still in the queue or completed
            const jobCounts = await recoveryQueue.getJobCounts();
            expect(jobCounts.waiting + jobCounts.active + jobCounts.completed).toBeGreaterThan(0);

            // Clean up explicitly
            await recoveryQueue.close();
            await newRedis.quit();
            
            // Force cleanup of isolated harness
            await isolatedHarness.forceCleanup();
        }, 15000);

        it("should recover gracefully from Redis memory exhaustion", async () => {
            const errorSpy = vi.spyOn(logger, "error");
            
            try {
                // Create a test queue
                const testQueue = new Queue("memory-exhaustion-test", { connection: harness.redis });
                harness.queues.push(testQueue);

                // Add some jobs first
                await testQueue.add("normal-job", { data: "normal" });

                // Simulate memory exhaustion
                await simulateRedisMemoryExhaustion(harness.redis);

                // Try to add more jobs - should handle gracefully
                try {
                    await testQueue.add("post-exhaustion-job", { data: "post-exhaustion" });
                } catch (error) {
                    // Expected - Redis out of memory
                    expect(error).toBeDefined();
                }

                // Verify error was logged
                expect(errorSpy).toHaveBeenCalled();

            } finally {
                errorSpy.mockRestore();
            }
        }, 20000);

        it("should handle Redis master failover scenarios", async () => {
            // Create a queue with failover configuration
            const testQueue = new Queue("failover-test", { 
                connection: {
                    host: "localhost",
                    port: 6379,
                    retryDelayOnFailover: 100,
                    maxRetriesPerRequest: 3,
                    lazyConnect: true,
                },
            });
            
            harness.queues.push(testQueue);

            // Add jobs before failover
            const job1 = await testQueue.add("pre-failover", { data: "before" });
            expect(job1).toBeDefined();

            // Simulate network partition
            await simulateNetworkPartition(harness.redis, 2000);

            // Try to add jobs during partition - should retry and eventually succeed
            const startTime = Date.now();
            try {
                const job2 = await testQueue.add("during-failover", { data: "during" }, {
                    attempts: 3,
                    backoff: { type: "fixed", delay: 1000 },
                });
                expect(job2).toBeDefined();
            } catch (error) {
                // May fail during partition - that's acceptable
                console.log("Job addition failed during partition (expected):", error.message);
            }

            // Wait for partition to end
            await new Promise(resolve => setTimeout(resolve, 3000));

            // Should be able to add jobs after recovery
            const job3 = await testQueue.add("post-failover", { data: "after" });
            expect(job3).toBeDefined();
        }, 15000);

        it("should detect and handle stale Redis connections", async () => {
            // Create initial connection
            const redis1 = await buildRedis(redisUrl);
            expect(redis1.status).toBe("ready");

            // Force disconnect to make it stale
            redis1.disconnect();
            await new Promise(resolve => setTimeout(resolve, 500));

            // Should detect stale connection and create new one
            const redis2 = await buildRedis(redisUrl);
            expect(redis2.status).toBe("ready");
            expect(redis2).not.toBe(redis1); // Should be different instance

            await redis2.quit();
        });

        it("should handle concurrent Redis connection requests during failures", async () => {
            // Clear any existing connections
            clearRedisCache();

            // Create multiple concurrent connection requests
            const connectionPromises = Array.from({ length: 5 }, () => buildRedis(redisUrl));

            // All should either succeed or fail gracefully
            const results = await Promise.allSettled(connectionPromises);
            
            results.forEach((result, index) => {
                if (result.status === "fulfilled") {
                    expect(result.value.status).toBe("ready");
                } else {
                    console.log(`Connection ${index} failed (acceptable):`, result.reason);
                }
            });

            // Clean up successful connections
            const successfulConnections = results
                .filter((r): r is PromiseFulfilledResult<IORedis> => r.status === "fulfilled")
                .map(r => r.value);

            await Promise.all(successfulConnections.map(conn => conn.quit().catch(() => {})));
        });
    });

    describe("Worker Resource Management", () => {
        it("should handle workers that consume excessive memory", async () => {
            const testQueue = new Queue("memory-hog-test", { connection: harness.redis });
            harness.queues.push(testQueue);

            // Create memory-hogging worker
            const worker = createMemoryHogWorker("memory-hog-test", harness.redis);
            harness.workers.push(worker);

            // Monitor memory usage before adding jobs
            const initialMemory = process.memoryUsage();

            // Add jobs that will consume memory
            const jobs = await Promise.all([
                testQueue.add("memory-job-1", { data: "test1" }),
                testQueue.add("memory-job-2", { data: "test2" }),
            ]);

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 6000));

            // Check memory usage increased significantly
            const finalMemory = process.memoryUsage();
            const memoryIncrease = finalMemory.heapUsed - initialMemory.heapUsed;
            
            // Should have consumed significant memory
            expect(memoryIncrease).toBeGreaterThan(50 * 1024 * 1024); // 50MB

            // Verify jobs were processed despite memory consumption
            const jobCounts = await testQueue.getJobCounts();
            expect(jobCounts.completed + jobCounts.failed).toBe(2);
        }, 15000);

        it("should handle worker process crashes gracefully", async () => {
            const testQueue = new Queue("crash-test", { connection: harness.redis });
            const queueEvents = new QueueEvents("crash-test", { connection: harness.redis });
            
            harness.queues.push(testQueue);
            harness.events.push(queueEvents);

            let failedJobs = 0;
            let completedJobs = 0;

            queueEvents.on("failed", () => { failedJobs++; });
            queueEvents.on("completed", () => { completedJobs++; });

            // Create worker that crashes 70% of the time
            const worker = createCrashingWorker("crash-test", harness.redis, 0.7);
            harness.workers.push(worker);

            // Add multiple jobs
            const jobPromises = Array.from({ length: 10 }, (_, i) =>
                testQueue.add(`crash-job-${i}`, { data: `test${i}` }, {
                    attempts: 3,
                    backoff: { type: "fixed", delay: 500 },
                }),
            );

            await Promise.all(jobPromises);

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 8000));

            // Some jobs should fail, some should succeed after retries
            expect(failedJobs + completedJobs).toBe(10);
            expect(failedJobs).toBeGreaterThan(0); // Some should fail
            expect(completedJobs).toBeGreaterThan(0); // Some should succeed
        }, 15000);

        it("should enforce worker timeout limits", async () => {
            const testQueue = new Queue("timeout-test", { connection: harness.redis });
            const queueEvents = new QueueEvents("timeout-test", { connection: harness.redis });
            
            harness.queues.push(testQueue);
            harness.events.push(queueEvents);

            let timeoutJobs = 0;
            queueEvents.on("failed", (job) => {
                if (job.failedReason?.includes("timeout") || job.failedReason?.includes("stalled")) {
                    timeoutJobs++;
                }
            });

            // Create worker that times out
            const worker = createTimeoutWorker("timeout-test", harness.redis, 2000); // 2s timeout
            harness.workers.push(worker);

            // Add jobs that will timeout
            const jobs = await Promise.all([
                testQueue.add("timeout-job-1", { data: "test1" }),
                testQueue.add("timeout-job-2", { data: "test2" }),
            ]);

            // Wait for timeout to occur
            await new Promise(resolve => setTimeout(resolve, 8000));

            const jobCounts = await testQueue.getJobCounts();
            
            // Jobs should be failed due to timeout
            expect(jobCounts.failed).toBeGreaterThan(0);
            expect(timeoutJobs).toBeGreaterThan(0);
        }, 15000);

        it("should handle worker concurrency limits", async () => {
            const testQueue = new Queue("concurrency-test", { connection: harness.redis });
            harness.queues.push(testQueue);

            let concurrentJobs = 0;
            let maxConcurrency = 0;

            // Worker with concurrency limit of 2
            const worker = new Worker("concurrency-test", async (job) => {
                concurrentJobs++;
                maxConcurrency = Math.max(maxConcurrency, concurrentJobs);
                
                // Simulate work
                await new Promise(resolve => setTimeout(resolve, 1000));
                
                concurrentJobs--;
                return { processed: true };
            }, { 
                connection: harness.redis,
                concurrency: 2, 
            });

            harness.workers.push(worker);

            // Add more jobs than concurrency limit
            const jobPromises = Array.from({ length: 5 }, (_, i) =>
                testQueue.add(`concurrent-job-${i}`, { data: `test${i}` }),
            );

            await Promise.all(jobPromises);

            // Wait for all jobs to complete
            await waitForQueueState(testQueue, { completed: 5 }, 10000);

            // Should not exceed concurrency limit
            expect(maxConcurrency).toBeLessThanOrEqual(2);
        }, 15000);
    });

    describe("Queue Limit Enforcement", () => {
        it("should enforce maximum active jobs limit", async () => {
            // Mock environment variables for strict limits
            const restoreEnv = mockEnvVars({
                WORKER_RUN_MAX_ACTIVE: "3",
                WORKER_RUN_HIGH_LOAD_THRESHOLD_PERCENTAGE: "0.8",
            });

            try {
                // Create run queue with limited capacity
                const runQueue = new Queue("run-limit-test", { connection: harness.redis });
                harness.queues.push(runQueue);

                // Create slow worker to keep jobs active
                const worker = new Worker("run-limit-test", async (job) => {
                    await new Promise(resolve => setTimeout(resolve, 3000));
                    return { processed: true };
                }, { 
                    connection: harness.redis,
                    concurrency: 1, 
                });

                harness.workers.push(worker);

                // Add jobs up to limit
                const jobs = [];
                for (let i = 0; i < 5; i++) {
                    jobs.push(await runQueue.add(`limit-job-${i}`, { data: `test${i}` }));
                }

                // Wait a bit for jobs to start processing
                await new Promise(resolve => setTimeout(resolve, 1000));

                const jobCounts = await runQueue.getJobCounts();
                
                // Should respect concurrency limits
                expect(jobCounts.active).toBeLessThanOrEqual(1); // Worker concurrency is 1
                expect(jobCounts.waiting).toBeGreaterThan(0); // Other jobs should wait

            } finally {
                restoreEnv();
            }
        }, 15000);

        it("should prioritize premium users correctly", async () => {
            // Test with swarm queue which has premium prioritization
            const jobs = [];

            // Add free user jobs
            for (let i = 0; i < 3; i++) {
                const result = await queueService.swarm.addTask({
                    type: QueueTaskType.SWARM_RUN,
                    status: "Scheduled",
                    swarmId: `freeswarm${i}`,
                    routineVersionId: "123",
                    runId: `freerun${i}`,
                    userData: { id: `${i}`, hasPremium: false },
                    inputs: {},
                    model: "gpt-4",
                    teamId: "1001",
                });
                if (result.success) jobs.push({ type: "free", jobId: result.data!.id });
            }

            // Add premium user jobs
            for (let i = 0; i < 3; i++) {
                const result = await queueService.swarm.addTask({
                    type: QueueTaskType.SWARM_RUN,
                    status: "Scheduled",
                    swarmId: `premiumswarm${i}`,
                    routineVersionId: "123",
                    runId: `premiumrun${i}`,
                    userData: { id: `${i + 100}`, hasPremium: true },
                    inputs: {},
                    model: "gpt-4",
                    teamId: "1002",
                });
                if (result.success) jobs.push({ type: "premium", jobId: result.data!.id });
            }

            // Check job priorities
            for (const jobInfo of jobs) {
                const job = await queueService.swarm.queue.getJob(jobInfo.jobId);
                if (job) {
                    if (jobInfo.type === "premium") {
                        expect(job.opts.priority).toBe(80); // Premium priority
                    } else {
                        expect(job.opts.priority).toBe(100); // Free priority
                    }
                }
            }
        });

        it("should handle rate limiting correctly", async () => {
            // Use email queue which has rate limiting
            const jobs = [];
            const startTime = Date.now();

            // Add many jobs rapidly
            for (let i = 0; i < 10; i++) {
                try {
                    const result = await queueService.email.addTask({
                        type: QueueTaskType.EMAIL_SEND,
                        to: [`ratelimit${i}@example.com`],
                        subject: `Rate limit test ${i}`,
                        text: `Test message ${i}`,
                    });
                    
                    if (result.success) {
                        jobs.push(result.data!.id);
                    }
                } catch (error) {
                    // Rate limiting might cause some to fail
                    console.log(`Job ${i} failed due to rate limiting:`, error.message);
                }
            }

            const endTime = Date.now();
            const duration = endTime - startTime;

            // Should not have processed all jobs instantly
            expect(duration).toBeGreaterThan(100); // At least some delay
            expect(jobs.length).toBeGreaterThan(0); // Some should succeed

            // Check queue state
            const jobCounts = await queueService.email.queue.getJobCounts();
            expect(jobCounts.waiting + jobCounts.active + jobCounts.completed).toBeGreaterThan(0);
        });

        it("should handle queue depth monitoring", async () => {
            // Create test queue with monitoring
            const testQueue = new Queue("depth-test", { connection: harness.redis });
            harness.queues.push(testQueue);

            // Add many jobs to create queue depth
            await generateQueueLoad(testQueue, 50, { test: "depth" });

            const metrics = await getQueueHealthMetrics(testQueue);
            
            expect(metrics.queueDepth).toBe(50);
            expect(metrics.counts.waiting).toBe(50);
            expect(metrics.activeJobs).toBe(0); // No worker, so no active jobs
        });

        it("should measure and enforce performance limits", async () => {
            const testQueue = new Queue("performance-test", { connection: harness.redis });
            harness.queues.push(testQueue);

            // Create worker with consistent processing time
            const worker = new Worker("performance-test", async (job) => {
                await new Promise(resolve => setTimeout(resolve, 100)); // 100ms per job
                return { processed: true, jobId: job.id };
            }, { 
                connection: harness.redis,
                concurrency: 2, 
            });

            harness.workers.push(worker);

            // Generate moderate load
            await generateQueueLoad(testQueue, 10, { test: "performance" });

            // Measure performance
            const metrics = await measureQueuePerformance(testQueue, 5000);

            expect(metrics.jobsProcessed).toBeGreaterThan(0);
            expect(metrics.throughputPerSecond).toBeGreaterThan(0);
            expect(metrics.errorRate).toBeLessThan(0.1); // Less than 10% errors
        }, 15000);
    });

    describe("Error Recovery and Cleanup", () => {
        it("should clean up failed jobs automatically", async () => {
            const testQueue = new Queue("cleanup-test", { 
                connection: harness.redis,
                defaultJobOptions: {
                    removeOnFail: 5, // Keep only 5 failed jobs
                    removeOnComplete: 10, // Keep only 10 completed jobs
                },
            });
            
            harness.queues.push(testQueue);

            // Create worker that fails some jobs
            const worker = new Worker("cleanup-test", async (job) => {
                if (job.data.shouldFail) {
                    throw new Error("Intentional failure");
                }
                return { processed: true };
            }, { connection: harness.redis });

            harness.workers.push(worker);

            // Add failing jobs
            for (let i = 0; i < 10; i++) {
                await testQueue.add(`failing-job-${i}`, { shouldFail: true }, {
                    attempts: 1, // No retries
                });
            }

            // Add successful jobs
            for (let i = 0; i < 15; i++) {
                await testQueue.add(`success-job-${i}`, { shouldFail: false });
            }

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 5000));

            const jobCounts = await testQueue.getJobCounts();
            
            // Should have cleaned up excess failed/completed jobs
            expect(jobCounts.failed).toBeLessThanOrEqual(5);
            expect(jobCounts.completed).toBeLessThanOrEqual(10);
        }, 10000);

        it("should handle Redis command monitoring", async () => {
            const redisSpy = createRedisSpy(harness.redis);

            try {
                const testQueue = new Queue("redis-spy-test", { connection: harness.redis });
                harness.queues.push(testQueue);

                // Perform various operations
                await testQueue.add("test-job", { data: "test" });
                await testQueue.getJobCounts();

                const commandCounts = redisSpy.getCommandCounts();
                const commandErrors = redisSpy.getCommandErrors();

                // Should have recorded Redis commands
                expect(Object.keys(commandCounts).length).toBeGreaterThan(0);
                expect(commandCounts.xadd || 0).toBeGreaterThan(0); // Queue operations use streams

                // Should not have errors in normal operation
                expect(Object.keys(commandErrors).length).toBe(0);

            } finally {
                redisSpy.restore();
            }
        });
    });
});
