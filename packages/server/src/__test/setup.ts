/**
 * Per-test-file setup that runs after global setup
 * Global setup handles containers and migrations
 * This handles ModelMap, DbProvider, and mocks initialization
 */

import { vi, beforeAll, afterAll, expect } from "vitest";

// Note: Sharp mocking is handled in vitest-sharp-mock-simple.ts

// Pre-import services to avoid dynamic import deadlocks in cleanup
let CacheService: any;
let QueueService: any;
let BusService: any;
let DbProvider: any;

// Import these lazily but store references
async function preloadServices() {
    try {
        ({ CacheService } = await import("../redisConn.js"));
        ({ QueueService } = await import("../tasks/queues.js"));
        ({ BusService } = await import("../services/bus.js"));
        ({ DbProvider } = await import("../db/provider.js"));
    } catch (e) {
        console.error("Failed to preload services:", e);
    }
}

const componentsInitialized = {
    modelMap: false,
    dbProvider: false,
    mocks: false,
    idGenerator: false,
};

beforeAll(async function setup() {
    console.log("=== Per-File Test Setup Starting ===");
    
    try {
        // Debug environment variables
        console.log("Environment check:", {
            REDIS_URL: process.env.REDIS_URL ? "SET" : "NOT SET",
            DB_URL: process.env.DB_URL ? "SET" : "NOT SET",
            NODE_ENV: process.env.NODE_ENV,
            VITEST: process.env.VITEST,
        });
        
        // Preload services to avoid dynamic import issues in cleanup
        await preloadServices();
        
        // Note: Environment vars, containers, and Prisma are already set up by global setup
        
        // Step 1: Initialize ModelMap (this is what the tests need)
        console.log("Step 1: Initializing ModelMap...");
        try {
            const { ModelMap } = await import("../models/base/index.js");
            // ModelMap.init() is already thread-safe and idempotent
            await ModelMap.init();
            componentsInitialized.modelMap = true;
            console.log("✓ ModelMap ready");
        } catch (error) {
            console.error("ModelMap initialization failed:", error);
            // Don't throw - continue without it
        }
        
        // Step 2: Initialize DbProvider
        console.log("Step 2: Initializing DbProvider...");
        try {
            if (!DbProvider) {
                ({ DbProvider } = await import("../db/provider.js"));
            }
            await DbProvider.init();
            componentsInitialized.dbProvider = true;
            console.log("✓ DbProvider ready");
        } catch (error) {
            console.error("DbProvider initialization failed:", error);
            // Don't throw - continue without it
        }
        
        // Step 3: Setup LLM service mocks
        // TODO: Replace with new LLM mock implementation
        // console.log("Step 3: Setting up LLM mocks...");
        // try {
        //     await setupLlmServiceMocks();
        //     componentsInitialized.mocks = true;
        //     console.log("✓ LLM mocks ready");
        // } catch (error) {
        //     console.error("LLM mock setup failed:", error);
        // }
        
        console.log("=== Per-File Test Setup Complete ===");
        console.log("Initialized components:", Object.entries(componentsInitialized)
            .filter(([, enabled]) => enabled)
            .map(([name]) => name)
            .join(", "));
        
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
        // This distributes memory load and prevents crashes. See ID_GENERATOR_CRASH_ROOT_CAUSE.md
        // for detailed analysis.
        
    } catch (error) {
        console.error("=== Setup Failed ===");
        console.error(error);
        await cleanup();
        throw error;
    }
}, 300000);

// TODO: Replace with new LLM mock implementation
// async function setupLlmServiceMocks() {
//     try {
//         const [
//             { OpenAIService },
//             { AnthropicService }, 
//             { MistralService }
//         ] = await Promise.all([
//             import("../tasks/llm/services/openai.js"),
//             import("../tasks/llm/services/anthropic.js"),
//             import("../tasks/llm/services/mistral.js"),
//         ]);
//         
//         vi.spyOn(OpenAIService.prototype, "generateResponse" as any).mockResolvedValue({
//             attempts: 1,
//             message: "Mocked OpenAI response",
//             cost: 0.001,
//         });
//         
//         vi.spyOn(AnthropicService.prototype, "generateResponse" as any).mockResolvedValue({
//             attempts: 1,
//             message: "Mocked Anthropic response",
//             cost: 0.001,
//         });
//         
//         vi.spyOn(MistralService.prototype, "generateResponse" as any).mockResolvedValue({
//             attempts: 1,
//             message: "Mocked Mistral response",
//             cost: 0.001,
//         });
//         
//         const mockEstimateTokens = vi.fn().mockReturnValue({ model: "default", tokens: 10 });
//         vi.spyOn(OpenAIService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
//         vi.spyOn(AnthropicService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
//         vi.spyOn(MistralService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
//     } catch (error) {
//         console.warn("Could not setup LLM mocks:", error);
//     }
// }

async function cleanup() {
    console.log("\n=== Per-File Cleanup Starting ===");
    
    // Restore mocks with timeout protection
    console.log("Restoring all mocks...");
    try {
        // Add timeout protection for vi.restoreAllMocks
        const restorePromise = new Promise<void>((resolve) => {
            vi.restoreAllMocks();
            resolve();
        });
        const timeoutPromise = new Promise<void>((_, reject) => 
            setTimeout(() => reject(new Error("Mock restore timeout")), 5000),
        );
        await Promise.race([restorePromise, timeoutPromise]);
        console.log("✓ Mocks restored");
    } catch (e) {
        console.error("Mock restore error:", e);
        // Force clear all mocks even if restore fails
        try {
            vi.clearAllMocks();
            vi.unstubAllGlobals();
        } catch (clearError) {
            console.error("Mock clear error:", clearError);
        }
    }
    
    // Reset singleton services to prevent stale connections
    console.log("Resetting CacheService...");
    try {
        if (CacheService && CacheService.reset) {
            const timeoutPromise = new Promise((_, reject) => 
                setTimeout(() => reject(new Error("CacheService reset timeout")), 10000),
            );
            await Promise.race([CacheService.reset(), timeoutPromise]);
            console.log("✓ CacheService reset");
        } else {
            console.log("⚠ CacheService not available");
        }
    } catch (e) {
        console.error("CacheService reset error:", e);
    }
    
    try {
        if (QueueService && QueueService.reset) {
            await QueueService.reset();
            console.log("✓ QueueService reset");
        } else {
            console.log("⚠ QueueService not available");
        }
    } catch (e) {
        console.error("QueueService reset error:", e);
    }
    
    try {
        if (BusService && BusService.reset) {
            await BusService.reset();
            console.log("✓ BusService reset");
        } else {
            console.log("⚠ BusService not available");
        }
    } catch (e) {
        console.error("BusService reset error:", e);
    }
    
    // Clean up based on what was initialized
    if (componentsInitialized.dbProvider) {
        try {
            if (DbProvider && DbProvider.reset) {
                await DbProvider.reset();
                console.log("✓ DbProvider reset");
            } else {
                console.log("⚠ DbProvider not available");
            }
        } catch (e) {
            console.error("DbProvider reset error:", e);
        }
    }
    
    // Note: Containers are managed by global teardown
    console.log("=== Per-File Cleanup Complete ===");
}

afterAll(async () => {
    const testFile = expect.getState().testPath?.split("/").pop() || "unknown";
    console.log(`[${new Date().toISOString()}] afterAll cleanup starting for ${testFile}`);
    const timeoutId = setTimeout(() => {
        console.error(`[${new Date().toISOString()}] CLEANUP TIMEOUT WARNING for ${testFile} - cleanup taking too long!`);
    }, 30000);
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
export const getSetupStatus = () => ({ ...componentsInitialized });
