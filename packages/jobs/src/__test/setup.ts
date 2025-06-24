/**
 * Per-test-file setup that runs after global setup
 * Global setup handles containers and migrations
 * This handles ModelMap, DbProvider, and mocks initialization for jobs package
 */

import { vi, beforeAll, afterAll, expect } from "vitest";

// Note: Sharp mocking is handled in vitest-sharp-mock-simple.ts

// Services will be imported when needed to avoid dependency issues

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
        
        // Services will be imported directly when needed
        
        // Note: Environment vars, containers, and Prisma are already set up by global setup
        
        // Step 1: Initialize ModelMap (this is what the tests need)
        console.log("Step 1: Initializing ModelMap...");
        try {
            const { ModelMap } = await import("@vrooli/server");
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
            const { DbProvider } = await import("@vrooli/server");
            await DbProvider.init();
            componentsInitialized.dbProvider = true;
            console.log("✓ DbProvider ready");
        } catch (error) {
            console.error("DbProvider initialization failed:", error);
            // Don't throw - continue without it
        }
        
        // Step 3: Setup LLM service mocks
        console.log("Step 3: Setting up LLM mocks...");
        try {
            await setupLlmServiceMocks();
            componentsInitialized.mocks = true;
            console.log("✓ LLM mocks ready");
        } catch (error) {
            console.error("LLM mock setup failed:", error);
        }
        
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
        // This distributes memory load and prevents crashes.
        
    } catch (error) {
        console.error("=== Setup Failed ===");
        console.error(error);
        await cleanup();
        throw error;
    }
}, 300000);

async function setupLlmServiceMocks() {
    try {
        // Mock LLM services that might be used by jobs
        const mockResponse = {
            attempts: 1,
            message: "Mocked LLM response",
            cost: 0.001,
        };
        
        const mockEstimateTokens = vi.fn().mockReturnValue({ model: "default", tokens: 10 });
        
        // Try to mock LLM services if they exist
        try {
            const server = await import("@vrooli/server");
            if (server.OpenAIService) {
                vi.spyOn(server.OpenAIService.prototype, "generateResponse" as any).mockResolvedValue(mockResponse);
                vi.spyOn(server.OpenAIService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
            }
            if (server.AnthropicService) {
                vi.spyOn(server.AnthropicService.prototype, "generateResponse" as any).mockResolvedValue(mockResponse);
                vi.spyOn(server.AnthropicService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
            }
            if (server.MistralService) {
                vi.spyOn(server.MistralService.prototype, "generateResponse" as any).mockResolvedValue(mockResponse);
                vi.spyOn(server.MistralService.prototype, "estimateTokens" as any).mockImplementation(mockEstimateTokens);
            }
        } catch (e) {
            // Services might not exist or be available
        }
        
    } catch (error) {
        console.warn("Could not setup LLM mocks:", error);
    }
}

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
    console.log("Resetting services...");
    try {
        const server = await import("@vrooli/server");
        if (server.CacheService && server.CacheService.reset) {
            const timeoutPromise = new Promise((_, reject) => 
                setTimeout(() => reject(new Error("CacheService reset timeout")), 10000),
            );
            await Promise.race([server.CacheService.reset(), timeoutPromise]);
            console.log("✓ CacheService reset");
        }
        if (server.QueueService && server.QueueService.reset) {
            await server.QueueService.reset();
            console.log("✓ QueueService reset");
        }
        if (server.BusService && server.BusService.reset) {
            await server.BusService.reset();
            console.log("✓ BusService reset");
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
                console.log("✓ DbProvider reset");
            }
        } catch (e) {
            console.log("⚠ DbProvider not available or failed to reset:", e.message);
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
