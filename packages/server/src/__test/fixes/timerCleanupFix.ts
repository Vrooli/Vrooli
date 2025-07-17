/**
 * Fixes for timer accumulation issues in test environment
 */

import { vi } from "vitest";

// Store active sleep promises so they can be cancelled
const activeSleepPromises = new Set<{
    promise: Promise<void>;
    timeoutId: NodeJS.Timeout;
    reject: (reason?: any) => void;
}>();

/**
 * Cancellable sleep function that can be interrupted
 */
export function cancellableSleep(ms: number): Promise<void> & { cancel: () => void } {
    let timeoutId: NodeJS.Timeout;
    let rejectFn: ((reason?: any) => void) | null = null;
    
    const promise = new Promise<void>((resolve, reject) => {
        rejectFn = reject;
        timeoutId = setTimeout(() => {
            activeSleepPromises.delete(entry);
            resolve();
        }, ms);
    }) as Promise<void> & { cancel: () => void };
    
    const entry = {
        promise,
        timeoutId: timeoutId!,
        reject: rejectFn!,
    };
    
    activeSleepPromises.add(entry);
    
    // Add cancel method to the promise
    promise.cancel = () => {
        clearTimeout(entry.timeoutId);
        activeSleepPromises.delete(entry);
        entry.reject(new Error("Sleep cancelled"));
    };
    
    return promise;
}

/**
 * Cancel all active sleep timers
 */
export function cancelAllSleeps() {
    activeSleepPromises.forEach(entry => {
        clearTimeout(entry.timeoutId);
        entry.reject(new Error("Sleep cancelled during cleanup"));
    });
    activeSleepPromises.clear();
}

/**
 * Enhanced Redis connection options for test environment
 */
export function getTestRedisOptions() {
    return {
        // Disable keepalive in tests to prevent timer accumulation
        keepAlive: 0,
        // Faster retry for tests
        retryStrategy: (retries: number) => {
            if (retries > 3) return null; // Stop after 3 retries in tests
            return Math.min(retries * 100, 500); // Max 500ms delay
        },
        // Shorter timeouts for tests
        connectTimeout: 5000,
        commandTimeout: 5000,
        // Disable lazy connect to ensure immediate connection
        lazyConnect: false,
    };
}

/**
 * Test-optimized stream bus options
 */
export function getTestStreamBusOptions() {
    return {
        autoClaimEveryMs: 1000, // 1 second instead of 1 minute
        batchSize: 10,
        blockMs: 100, // 100ms instead of 5 seconds
        closeTimeoutMs: 100, // Faster shutdown
        groupName: "test-workers",
        maxReconnectDelayMs: 500,
        maxReconnectAttempts: 3,
        maxStreamSize: 1000,
        reconnectDelayMultiplier: 100,
        retryCount: 1,
        streamName: "test.events",
    };
}

/**
 * Enhanced cleanup for Redis-based services
 */
export async function cleanupRedisServices() {
    console.log("[CLEANUP] Starting Redis service cleanup...");
    
    try {
        // Cancel all active sleeps first
        cancelAllSleeps();
        
        // Reset BusService
        const { BusService } = await import("../../services/bus.js");
        await BusService.reset();
        
        // Reset CacheService
        const { CacheService } = await import("../../redisConn.js");
        if ((CacheService as any).instance) {
            const instance = (CacheService as any).instance;
            if (instance.client) {
                await instance.client.quit();
            }
            (CacheService as any).instance = null;
        }
        
        // Reset QueueService
        const { QueueService } = await import("../../tasks/queues.js");
        if ((QueueService as any).instance) {
            await (QueueService as any).instance.close();
            (QueueService as any).instance = null;
        }
        
        // Clear all remaining timers
        const maxTimerId = setTimeout(() => {}, 0);
        for (let i = 0; i < maxTimerId; i++) {
            clearTimeout(i);
            clearInterval(i);
        }
        
        console.log("[CLEANUP] Redis service cleanup complete");
    } catch (error) {
        console.error("[CLEANUP] Error during Redis service cleanup:", error);
    }
}

/**
 * Setup test environment optimizations
 */
export function setupTestEnvironmentOptimizations() {
    // Override default options for test environment
    process.env.REDIS_AUTO_CLAIM_EVERY_MS = "1000"; // 1 second
    process.env.REDIS_BLOCK_MS = "100"; // 100ms
    process.env.REDIS_CLOSE_TIMEOUT_MS = "100"; // 100ms
    process.env.REDIS_MAX_RECONNECT_ATTEMPTS = "3";
    
    // Disable TCP keepalive in tests
    process.env.REDIS_KEEPALIVE = "0";
    
    console.log("[TEST SETUP] Applied test environment optimizations");
}

/**
 * Monkey-patch the sleep function in bus.ts to use cancellable version
 */
export async function patchBusSleepFunction() {
    try {
        const busModule = await import("../../services/bus.js");
        
        // Replace the module's sleep function with our cancellable version
        if (busModule) {
            (busModule as any).sleep = cancellableSleep;
            console.log("[PATCH] Replaced bus.ts sleep function with cancellable version");
        }
    } catch (error) {
        console.error("[PATCH] Failed to patch bus sleep function:", error);
    }
}
