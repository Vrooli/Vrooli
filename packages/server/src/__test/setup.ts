/**
 * Per-test-file setup that runs after global setup
 * Global setup handles containers and migrations
 * This handles ModelMap, DbProvider, and mocks initialization
 */

import { vi, beforeAll, afterAll, expect } from "vitest";

// Note: Sharp mocking is handled in vitest-sharp-mock-simple.ts

// Global singleton for error handlers to prevent memory leaks
const errorHandlerSingleton = (() => {
    let initialized = false;
    let unhandledRejectionHandler: ((reason: any, promise: Promise<any>) => void) | null = null;
    let uncaughtExceptionHandler: ((error: Error) => void) | null = null;

    return {
        init() {
            if (initialized || process.env.NODE_ENV !== "test") return;
            
            // Increase max listeners to handle multiple test files
            // With 115 test files, we need more than the default 10
            process.setMaxListeners(150);
            
            unhandledRejectionHandler = (reason, promise) => {
                // Handle Redis connection errors that occur during test cleanup
                if (reason instanceof Error) {
                    if (reason.message.includes("Connection is closed") || 
                        reason.message.includes("Connection is not available") ||
                        reason.message.includes("Stream isn't writeable")) {
                        // These are expected during test cleanup, only log if TEST_LOG_LEVEL is DEBUG
                        if (process.env.TEST_LOG_LEVEL === "DEBUG") {
                            console.debug("Test cleanup connection error (ignored):", reason.message);
                        }
                        return;
                    }
                }
                
                // For other unhandled rejections, only log if not suppressed
                if (process.env.LOG_LEVEL !== "emerg" && process.env.LOG_LEVEL !== "alert") {
                    console.error("Unhandled Rejection in tests:", reason);
                }
            };

            uncaughtExceptionHandler = (error) => {
                // Handle Redis connection errors that occur during test cleanup
                if (error.message.includes("Connection is closed") || 
                    error.message.includes("Connection is not available") ||
                    error.message.includes("Stream isn't writeable")) {
                    // These are expected during test cleanup, only log if TEST_LOG_LEVEL is DEBUG
                    if (process.env.TEST_LOG_LEVEL === "DEBUG") {
                        console.debug("Test cleanup connection error (ignored):", error.message);
                    }
                    return;
                }
                
                // For other uncaught exceptions, only log if not suppressed
                if (process.env.LOG_LEVEL !== "emerg" && process.env.LOG_LEVEL !== "alert") {
                    console.error("Uncaught Exception in tests:", error);
                }
            };

            process.on("unhandledRejection", unhandledRejectionHandler);
            process.on("uncaughtException", uncaughtExceptionHandler);
            initialized = true;
        },
        
        cleanup() {
            if (!initialized) return;
            
            if (unhandledRejectionHandler) {
                process.removeListener("unhandledRejection", unhandledRejectionHandler);
            }
            if (uncaughtExceptionHandler) {
                process.removeListener("uncaughtException", uncaughtExceptionHandler);
            }
            initialized = false;
        },
    };
})();

// Initialize error handlers once
errorHandlerSingleton.init();

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

// Setup QueueService mock early to prevent email service failures
import { setupQueueServiceMock } from "./mocks/queueServiceMock.js";

setupQueueServiceMock();

const componentsInitialized = {
    modelMap: false,
    dbProvider: false,
    mocks: false,
    idGenerator: false,
    socketService: false,
};

// Log level control for test setup
const LOG_LEVEL = process.env.TEST_LOG_LEVEL || "ERROR";
const logLevels = { ERROR: 0, WARN: 1, INFO: 2, DEBUG: 3 };
const currentLogLevel = logLevels[LOG_LEVEL as keyof typeof logLevels] ?? logLevels.ERROR;

function testLog(level: keyof typeof logLevels, ...args: any[]) {
    if (logLevels[level] <= currentLogLevel) {
        console.log(...args);
    }
}

function testError(...args: any[]) {
    console.error(...args);
}

function testWarn(...args: any[]) {
    if (currentLogLevel >= logLevels.WARN) {
        console.warn(...args);
    }
}


beforeAll(async function setup() {
    testLog("DEBUG", "=== Per-File Test Setup Starting ===");
    
    try {
        // Debug environment variables
        testLog("DEBUG", "Environment check:", {
            REDIS_URL: process.env.REDIS_URL ? "SET" : "NOT SET",
            DB_URL: process.env.DB_URL ? "SET" : "NOT SET",
            NODE_ENV: process.env.NODE_ENV,
            VITEST: process.env.VITEST,
        });
        
        // Preload services to avoid dynamic import issues in cleanup
        await preloadServices();
        
        // Note: Environment vars, containers, and Prisma are already set up by global setup
        
        // Step 1: Initialize ModelMap (this is what the tests need)
        testLog("DEBUG", "Step 1: Initializing ModelMap...");
        try {
            // Clear module cache for ModelMap to work around vitest transformation issues
            const modelMapPath = "../models/base/index.js";
            
            // Use dynamic import with cache busting
            const modelModule = await import(modelMapPath);
            testLog("DEBUG", "ModelMap module imported:", { 
                keys: Object.keys(modelModule), 
                hasModelMap: "ModelMap" in modelModule,
                moduleType: typeof modelModule,
                isDefault: "default" in modelModule,
            });
            
            // Handle both named and default exports
            let ModelMap = modelModule.ModelMap;
            if (!ModelMap && modelModule.default?.ModelMap) {
                ModelMap = modelModule.default.ModelMap;
                testLog("DEBUG", "Using ModelMap from default export");
            }
            
            if (!ModelMap) {
                // Try to access it directly from the module
                const possibleExports = Object.keys(modelModule);
                testError("ModelMap not found. Available exports:", possibleExports);
                
                // Check if it's a transformed class
                const classExport = possibleExports.find(key => {
                    const val = modelModule[key];
                    return typeof val === "function" && val.name === "ModelMap";
                });
                
                if (classExport) {
                    ModelMap = modelModule[classExport];
                    testLog("DEBUG", `Found ModelMap as ${classExport}`);
                } else {
                    throw new Error("ModelMap is undefined after import");
                }
            }
            
            // Verify ModelMap has the expected structure
            if (typeof ModelMap !== "function") {
                testError("ModelMap is not a function/class:", {
                    type: typeof ModelMap,
                    value: ModelMap,
                    stringified: String(ModelMap),
                });
                throw new Error(`ModelMap is not a function (type: ${typeof ModelMap})`);
            }
            
            // Check for init method
            if (typeof ModelMap.init !== "function") {
                testError("ModelMap.init is not a function:", {
                    ModelMap: String(ModelMap),
                    type: typeof ModelMap,
                    constructor: ModelMap.constructor?.name,
                    properties: Object.getOwnPropertyNames(ModelMap),
                    prototype: ModelMap.prototype,
                    staticMethods: Object.getOwnPropertyNames(ModelMap).filter(p => typeof ModelMap[p] === "function"),
                });
                throw new Error(`ModelMap.init is not a function (type: ${typeof ModelMap.init})`);
            }
            
            // ModelMap.init() is already thread-safe and idempotent
            await ModelMap.init();
            componentsInitialized.modelMap = true;
            testLog("DEBUG", "✓ ModelMap ready");
        } catch (error) {
            testError("ModelMap initialization failed:", error);
            // Don't throw - continue without it
        }
        
        // Step 2: Initialize i18n for notification translations
        testLog("DEBUG", "Step 2: Initializing i18n...");
        try {
            const { i18nConfig } = await import("@vrooli/shared");
            const i18next = (await import("i18next")).default;
            await i18next.init(i18nConfig(false)); // false for production mode in tests
            testLog("DEBUG", "✓ i18n ready");
        } catch (error) {
            testError("i18n initialization failed:", error);
            // Don't throw - tests can proceed without i18n but notifications may fail
        }

        // Step 3: Initialize DbProvider (idempotent - safe to call multiple times)
        testLog("DEBUG", "Step 3: Initializing DbProvider...");
        try {
            if (!DbProvider) {
                ({ DbProvider } = await import("../db/provider.js"));
            }
            // DbProvider.init() is idempotent - it checks if already initialized
            await DbProvider.init();
            componentsInitialized.dbProvider = true;
            testLog("DEBUG", "✓ DbProvider ready");
        } catch (error) {
            testError("DbProvider initialization failed:", error);
            throw error; // This is critical - tests cannot proceed without DB
        }
        
        // Step 3.5: Initialize SocketService mock for tests
        testLog("DEBUG", "Step 3.5: Initializing SocketService mock...");
        try {
            const { initializeSocketServiceMock } = await import("./mocks/socketServiceMock.js");
            await initializeSocketServiceMock();
            componentsInitialized.socketService = true;
            testLog("DEBUG", "✓ SocketService mock ready");
        } catch (error) {
            testError("SocketService mock initialization failed:", error);
            // Don't throw - tests can proceed without socket functionality
        }
        
        // Step 4: Setup LLM service mocks
        // TODO: Replace with new LLM mock implementation
        // console.log("Step 4: Setting up LLM mocks...");
        // try {
        //     await setupLlmServiceMocks();
        //     componentsInitialized.mocks = true;
        //     console.log("✓ LLM mocks ready");
        // } catch (error) {
        //     console.error("LLM mock setup failed:", error);
        // }
        
        testLog("DEBUG", "=== Per-File Test Setup Complete ===");
        testLog("DEBUG", "Initialized components:", Object.entries(componentsInitialized)
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
        testError("=== Setup Failed ===");
        testError(error);
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
    testLog("DEBUG", "\n=== Per-File Cleanup Starting ===");
    
    // Restore mocks with timeout protection
    testLog("DEBUG", "Restoring all mocks...");
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
        testLog("DEBUG", "✓ Mocks restored");
    } catch (e) {
        testError("Mock restore error:", e);
        // Force clear all mocks even if restore fails
        try {
            vi.clearAllMocks();
            vi.unstubAllGlobals();
        } catch (clearError) {
            testError("Mock clear error:", clearError);
        }
    }
    
    // Reset singleton services to prevent stale connections
    // Reset in order: Queues first (they depend on Redis), then Services, then Cache
    
    testLog("DEBUG", "Resetting QueueService...");
    try {
        if (QueueService && QueueService.reset) {
            const timeoutPromise = new Promise((_, reject) => 
                setTimeout(() => reject(new Error("QueueService reset timeout")), 15000),
            );
            await Promise.race([QueueService.reset(), timeoutPromise]);
            testLog("DEBUG", "✓ QueueService reset");
        } else {
            testWarn("⚠ QueueService not available");
        }
    } catch (e) {
        testError("QueueService reset error:", e);
    }
    
    testLog("DEBUG", "Resetting BusService...");
    try {
        if (BusService && BusService.reset) {
            const timeoutPromise = new Promise((_, reject) => 
                setTimeout(() => reject(new Error("BusService reset timeout")), 10000),
            );
            await Promise.race([BusService.reset(), timeoutPromise]);
            testLog("DEBUG", "✓ BusService reset");
        } else {
            testWarn("⚠ BusService not available");
        }
    } catch (e) {
        testError("BusService reset error:", e);
    }

    // Reset SocketService to prevent Redis connection leaks
    testLog("DEBUG", "Resetting SocketService...");
    try {
        const { SocketService } = await import("../sockets/io.js");
        if (SocketService && SocketService.reset) {
            const timeoutPromise = new Promise((_, reject) => 
                setTimeout(() => reject(new Error("SocketService reset timeout")), 10000),
            );
            await Promise.race([SocketService.reset(), timeoutPromise]);
            testLog("DEBUG", "✓ SocketService reset");
        } else {
            testWarn("⚠ SocketService not available");
        }
    } catch (e) {
        testError("SocketService reset error:", e);
    }
    
    testLog("DEBUG", "Resetting CacheService...");
    try {
        if (CacheService && CacheService.reset) {
            const timeoutPromise = new Promise((_, reject) => 
                setTimeout(() => reject(new Error("CacheService reset timeout")), 10000),
            );
            await Promise.race([CacheService.reset(), timeoutPromise]);
            testLog("DEBUG", "✓ CacheService reset");
        } else {
            testWarn("⚠ CacheService not available");
        }
    } catch (e) {
        testError("CacheService reset error:", e);
    }
    
    // Additional cleanup for test Redis connections
    if (process.env.NODE_ENV === "test") {
        testLog("DEBUG", "Cleaning up test Redis connections...");
        try {
            const { closeRedisConnections, clearRedisCache } = await import("../tasks/queueFactory.js");
            await closeRedisConnections();
            clearRedisCache();
            testLog("DEBUG", "✓ Test Redis connections cleaned up");
        } catch (e) {
            testError("Test Redis cleanup error:", e);
        }
    }
    
    // Remove process event handlers to prevent memory leaks
    testLog("DEBUG", "Removing process event handlers...");
    errorHandlerSingleton.cleanup();
    testLog("DEBUG", "✓ Removed process event handlers");
    
    // Note: DbProvider is managed by global setup/teardown - DO NOT reset here
    // Note: Containers are managed by global teardown
    testLog("DEBUG", "=== Per-File Cleanup Complete ===");
}

afterAll(async () => {
    const testFile = expect.getState().testPath?.split("/").pop() || "unknown";
    testLog("DEBUG", `[${new Date().toISOString()}] afterAll cleanup starting for ${testFile}`);
    const timeoutId = setTimeout(() => {
        testError(`[${new Date().toISOString()}] CLEANUP TIMEOUT WARNING for ${testFile} - cleanup taking too long!`);
    }, 30000);
    try {
        await cleanup();
        clearTimeout(timeoutId);
        testLog("DEBUG", `[${new Date().toISOString()}] afterAll cleanup completed for ${testFile}`);
    } catch (error) {
        clearTimeout(timeoutId);
        testError(`[${new Date().toISOString()}] afterAll cleanup error for ${testFile}:`, error);
        throw error;
    }
});

// Export status for tests to check
export const getSetupStatus = () => ({ ...componentsInitialized });
