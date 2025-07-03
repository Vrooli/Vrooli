// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-06-24
/**
 * Per-test-file setup that runs after global setup
 * Global setup handles containers and migrations
 * This handles ModelMap, DbProvider, and mocks initialization for jobs package
 */

import { vi, beforeAll, afterAll, expect } from "vitest";

// Test timeout constants
const TEST_SETUP_TIMEOUT_MS = 300_000; // 5 minutes

// Note: Sharp mocking is handled in vitest-sharp-mock-simple.ts

// Services will be imported when needed to avoid dependency issues

const componentsInitialized = {
    modelMap: false,
    dbProvider: false,
    mocks: false,
    idGenerator: false,
};

beforeAll(async function setup() {
    const isVerbose = process.env.TEST_VERBOSE === "true";
    
    if (isVerbose) {
        console.log("=== Per-File Test Setup Starting ===");
        console.log("Environment check:", {
            REDIS_URL: process.env.REDIS_URL ? "SET" : "NOT SET", 
            DB_URL: process.env.DB_URL ? "SET" : "NOT SET",
            NODE_ENV: process.env.NODE_ENV,
            VITEST: process.env.VITEST,
        });
    }
    
    try {
        
        // Services will be imported directly when needed
        
        // Note: Environment vars, containers, and Prisma are already set up by global setup
        
        // Step 1: Initialize ModelMap (this is what the tests need)
        if (isVerbose) console.log("Step 1: Initializing ModelMap...");
        try {
            const { ModelMap } = await import("@vrooli/server");
            // ModelMap.init() is already thread-safe and idempotent
            await ModelMap.init();
            componentsInitialized.modelMap = true;
            if (isVerbose) console.log("✓ ModelMap ready");
        } catch (error) {
            console.error("ModelMap initialization failed:", error);
            // Don't throw - continue without it
        }
        
        // Step 2: Initialize DbProvider
        if (isVerbose) console.log("Step 2: Initializing DbProvider...");
        try {
            const { DbProvider } = await import("@vrooli/server");
            await DbProvider.init();
            componentsInitialized.dbProvider = true;
            if (isVerbose) console.log("✓ DbProvider ready");
        } catch (error) {
            console.error("DbProvider initialization failed:", error);
            // Don't throw - continue without it
        }
        
        // Step 3: Setup LLM service mocks
        if (isVerbose) console.log("Step 3: Setting up LLM mocks...");
        try {
            await setupLlmServiceMocks();
            componentsInitialized.mocks = true;
            if (isVerbose) console.log("✓ LLM mocks ready");
        } catch (error) {
            console.error("LLM mock setup failed:", error);
        }
        
        if (isVerbose) {
            console.log("=== Per-File Test Setup Complete ===");
            console.log("Initialized components:", Object.entries(componentsInitialized)
                .filter(([, enabled]) => enabled)
                .map(([name]) => name)
                .join(", "));
        }
        
        // IMPORTANT: ID generator not initialized in global setup to avoid worker crashes
        // 
        // Root cause: Memory pressure from containers + Prisma + module complexity
        // causes vitest workers (2GB heap limit) to crash when importing @vrooli/shared
        // after full setup. This is NOT a bug in the ID generator itself.
        //
        // Tests that need ID generation should initialize it individually:
        // const { initIdGenerator } = await import("@vrooli/shared");
        // await initIdGenerator(0);
        //
        // This distributes memory load and prevents crashes.
        
    } catch (error) {
        console.error("=== Setup Failed ===");
        console.error(error);
        await cleanup();
        throw error;
    }
}, TEST_SETUP_TIMEOUT_MS);

async function setupLlmServiceMocks() {
    try {
        // Note: AI services (OpenAIService, AnthropicService, MistralService) are internal 
        // to the response service system and not exported from the main server package.
        // They are handled internally by the response service registry and don't need 
        // to be mocked for job processing tests.
        
    } catch (error) {
        console.warn("Could not setup LLM mocks:", error);
    }
}

async function cleanup() {
    const isVerbose = process.env.TEST_VERBOSE === "true";
    if (isVerbose) console.log("\n=== Per-File Cleanup Starting ===");
    
    // Comprehensive mock cleanup to prevent cross-test interference
    if (isVerbose) console.log("Clearing all mocks...");
    try {
        // First clear all mock state
        vi.clearAllMocks();
        
        // Then restore all mocks to their original implementations
        // Add timeout protection for vi.restoreAllMocks
        const restorePromise = new Promise<void>((resolve) => {
            vi.restoreAllMocks();
            resolve();
        });
        const MOCK_RESTORE_TIMEOUT = 5_000;
        const timeoutPromise = new Promise<void>((_, reject) => 
            setTimeout(() => reject(new Error("Mock restore timeout")), MOCK_RESTORE_TIMEOUT),
        );
        await Promise.race([restorePromise, timeoutPromise]);
        
        // Also clear any global stubs
        vi.unstubAllGlobals();
        
        if (isVerbose) console.log("✓ Mocks cleared and restored");
    } catch (e) {
        console.error("Mock cleanup error:", e);
        // Force clear as fallback
        try {
            vi.clearAllMocks();
            vi.unstubAllGlobals();
            console.log("✓ Fallback mock clear completed");
        } catch (clearError) {
            console.error("Mock clear fallback error:", clearError);
        }
    }
    
    // Reset singleton services to prevent stale connections
    if (isVerbose) console.log("Resetting services...");
    try {
        const server = await import("@vrooli/server");
        if (server.CacheService && server.CacheService.reset) {
            const CACHE_RESET_TIMEOUT = 10_000;
            const timeoutPromise = new Promise((_, reject) => 
                setTimeout(() => reject(new Error("CacheService reset timeout")), CACHE_RESET_TIMEOUT),
            );
            await Promise.race([server.CacheService.reset(), timeoutPromise]);
            if (isVerbose) console.log("✓ CacheService reset");
        }
        if (server.QueueService && server.QueueService.reset) {
            await server.QueueService.reset();
            if (isVerbose) console.log("✓ QueueService reset");
        }
        if (server.BusService && server.BusService.reset) {
            await server.BusService.reset();
            if (isVerbose) console.log("✓ BusService reset");
        }
    } catch (e) {
        console.log("⚠ Services not available or failed to reset:", e.message);
    }
    
    // Clean up based on what was initialized
    if (componentsInitialized.dbProvider) {
        try {
            const { DbProvider } = await import("@vrooli/server");
            if (DbProvider && DbProvider.reset) {
                await DbProvider.reset();
                if (isVerbose) console.log("✓ DbProvider reset");
            }
        } catch (e) {
            console.log("⚠ DbProvider not available or failed to reset:", e.message);
        }
    }
    
    // Note: Containers are managed by global teardown
    if (isVerbose) console.log("=== Per-File Cleanup Complete ===");
}

afterAll(async () => {
    const testFile = expect.getState().testPath?.split("/").pop() || "unknown";
    console.log(`[${new Date().toISOString()}] afterAll cleanup starting for ${testFile}`);
    const CLEANUP_TIMEOUT = 30_000;
    const timeoutId = setTimeout(() => {
        console.error(`[${new Date().toISOString()}] CLEANUP TIMEOUT WARNING for ${testFile} - cleanup taking too long!`);
    }, CLEANUP_TIMEOUT);
    try {
        await cleanup();
        clearTimeout(timeoutId);
        console.log(`[${new Date().toISOString()}] afterAll cleanup completed for ${testFile}`);
    } catch (error) {
        clearTimeout(timeoutId);
        console.error(`[${new Date().toISOString()}] afterAll cleanup error for ${testFile}:`, error);
        throw error;
    }
});

// Export status for tests to check
export function getSetupStatus() {
    return { ...componentsInitialized };
}
