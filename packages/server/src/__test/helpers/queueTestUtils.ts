import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import IORedis from "ioredis";
import { Queue, Worker, QueueEvents } from "bullmq";
import { logger } from "../../events/logger.js";

/**
 * Test utilities for queue resilience and robustness testing
 */

export interface QueueTestHarness {
    redis: IORedis;
    queues: Queue[];
    workers: Worker[];
    events: QueueEvents[];
    cleanup: () => Promise<void>;
}

/**
 * Creates a test harness for queue testing with automatic cleanup
 */
export async function createQueueTestHarness(redisUrl: string): Promise<QueueTestHarness> {
    const redis = new IORedis(redisUrl, {
        maxRetriesPerRequest: null,
        retryDelayOnFailover: 100,
    });

    const queues: Queue[] = [];
    const workers: Worker[] = [];
    const events: QueueEvents[] = [];

    const cleanup = async () => {
        // Close workers first
        await Promise.all(workers.map(worker => 
            worker.close().catch(e => console.log('Worker close error:', e))
        ));
        
        // Close queue events
        await Promise.all(events.map(event => 
            event.close().catch(e => console.log('Events close error:', e))
        ));
        
        // Close queues
        await Promise.all(queues.map(queue => 
            queue.close().catch(e => console.log('Queue close error:', e))
        ));
        
        // Close Redis connection
        await redis.quit().catch(e => console.log('Redis close error:', e));
    };

    return {
        redis,
        queues,
        workers,
        events,
        cleanup,
    };
}

/**
 * Simulates Redis connection failure by disconnecting the client
 */
export async function simulateRedisDisconnection(redis: IORedis): Promise<void> {
    redis.disconnect();
}

/**
 * Simulates Redis memory exhaustion
 */
export async function simulateRedisMemoryExhaustion(redis: IORedis): Promise<void> {
    // Fill Redis with dummy data until memory is exhausted
    try {
        for (let i = 0; i < 10000; i++) {
            await redis.set(`dummy:${i}`, 'x'.repeat(10000));
        }
    } catch (error) {
        // Expected - Redis should run out of memory
    }
}

/**
 * Simulates network partition by blocking Redis commands
 */
export async function simulateNetworkPartition(redis: IORedis, durationMs: number): Promise<void> {
    const originalSendCommand = redis.sendCommand;
    
    // Block all commands
    redis.sendCommand = () => {
        return new Promise((resolve, reject) => {
            setTimeout(() => reject(new Error('Network timeout')), 5000);
        });
    };
    
    // Restore after duration
    setTimeout(() => {
        redis.sendCommand = originalSendCommand;
    }, durationMs);
}

/**
 * Creates a worker that consumes excessive memory
 */
export function createMemoryHogWorker(queueName: string, redis: IORedis): Worker {
    return new Worker(queueName, async (job) => {
        // Allocate large amounts of memory
        const largeArray = new Array(1000000).fill('x'.repeat(1000));
        
        // Simulate work while holding memory
        await new Promise(resolve => setTimeout(resolve, 5000));
        
        // Keep reference to prevent garbage collection
        (global as any)[`memoryHog_${job.id}`] = largeArray;
        
        return { processed: true };
    }, { connection: redis });
}

/**
 * Creates a worker that crashes randomly
 */
export function createCrashingWorker(queueName: string, redis: IORedis, crashProbability = 0.5): Worker {
    return new Worker(queueName, async (job) => {
        if (Math.random() < crashProbability) {
            throw new Error(`Worker crashed while processing job ${job.id}`);
        }
        
        return { processed: true };
    }, { connection: redis });
}

/**
 * Creates a worker that times out
 */
export function createTimeoutWorker(queueName: string, redis: IORedis, timeoutMs = 10000): Worker {
    return new Worker(queueName, async (job) => {
        // Simulate work that takes longer than timeout
        await new Promise(resolve => setTimeout(resolve, timeoutMs + 1000));
        return { processed: true };
    }, { 
        connection: redis,
        lockDuration: timeoutMs,
    });
}

/**
 * Monitors queue health metrics
 */
export async function getQueueHealthMetrics(queue: Queue) {
    const counts = await queue.getJobCounts();
    const waiting = await queue.getWaiting();
    const active = await queue.getActive();
    const failed = await queue.getFailed();
    const completed = await queue.getCompleted();
    
    return {
        counts,
        queueDepth: counts.waiting + counts.delayed,
        activeJobs: active.length,
        failureRate: counts.failed / (counts.completed + counts.failed || 1),
        oldestWaiting: waiting.length > 0 ? waiting[0].timestamp : null,
        recentFailures: failed.slice(-10).map(job => ({
            id: job.id,
            error: job.failedReason,
            timestamp: job.finishedOn,
        })),
    };
}

/**
 * Waits for queue to reach a specific state
 */
export async function waitForQueueState(
    queue: Queue, 
    expectedCounts: Partial<{ waiting: number; active: number; completed: number; failed: number }>,
    timeoutMs = 10000
): Promise<boolean> {
    const startTime = Date.now();
    
    while (Date.now() - startTime < timeoutMs) {
        const counts = await queue.getJobCounts();
        
        const matches = Object.entries(expectedCounts).every(([key, value]) => {
            return counts[key as keyof typeof counts] === value;
        });
        
        if (matches) return true;
        
        await new Promise(resolve => setTimeout(resolve, 100));
    }
    
    return false;
}

/**
 * Generates load on a queue by adding many jobs
 */
export async function generateQueueLoad(
    queue: Queue,
    jobCount: number,
    jobData = { test: true }
): Promise<string[]> {
    const jobIds: string[] = [];
    
    for (let i = 0; i < jobCount; i++) {
        const job = await queue.add(`load-test-${i}`, {
            ...jobData,
            index: i,
            timestamp: Date.now(),
        });
        
        if (job.id) {
            jobIds.push(job.id);
        }
    }
    
    return jobIds;
}

/**
 * Measures queue performance metrics
 */
export async function measureQueuePerformance(
    queue: Queue,
    testDurationMs = 10000
): Promise<{
    jobsProcessed: number;
    averageProcessingTime: number;
    throughputPerSecond: number;
    errorRate: number;
}> {
    const startTime = Date.now();
    const initialCounts = await queue.getJobCounts();
    
    await new Promise(resolve => setTimeout(resolve, testDurationMs));
    
    const endTime = Date.now();
    const finalCounts = await queue.getJobCounts();
    
    const jobsProcessed = finalCounts.completed - initialCounts.completed;
    const errors = finalCounts.failed - initialCounts.failed;
    const duration = (endTime - startTime) / 1000;
    
    return {
        jobsProcessed,
        averageProcessingTime: duration / jobsProcessed || 0,
        throughputPerSecond: jobsProcessed / duration,
        errorRate: errors / (jobsProcessed + errors || 1),
    };
}

/**
 * Creates a spy for Redis operations
 */
export function createRedisSpy(redis: IORedis) {
    const originalSendCommand = redis.sendCommand;
    const commandCounts = new Map<string, number>();
    const commandErrors = new Map<string, Error[]>();
    
    redis.sendCommand = function(command) {
        const commandName = command.name;
        commandCounts.set(commandName, (commandCounts.get(commandName) || 0) + 1);
        
        try {
            return originalSendCommand.call(this, command);
        } catch (error) {
            const errors = commandErrors.get(commandName) || [];
            errors.push(error as Error);
            commandErrors.set(commandName, errors);
            throw error;
        }
    };
    
    return {
        getCommandCounts: () => Object.fromEntries(commandCounts),
        getCommandErrors: () => Object.fromEntries(commandErrors),
        restore: () => {
            redis.sendCommand = originalSendCommand;
        },
    };
}

/**
 * Mock environment variables for testing
 */
export function mockEnvVars(vars: Record<string, string>): () => void {
    const originalVars: Record<string, string | undefined> = {};
    
    Object.entries(vars).forEach(([key, value]) => {
        originalVars[key] = process.env[key];
        process.env[key] = value;
    });
    
    return () => {
        Object.entries(originalVars).forEach(([key, value]) => {
            if (value === undefined) {
                delete process.env[key];
            } else {
                process.env[key] = value;
            }
        });
    };
}