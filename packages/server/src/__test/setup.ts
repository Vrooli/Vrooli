/**
 * Per-test-file setup that runs after global setup
 * Global setup handles containers and migrations
 * This handles ModelMap, DbProvider, and mocks initialization
 */

import { afterAll, beforeAll, vi } from "vitest";

// Note: Sharp mocking is handled in vitest-sharp-mock-simple.ts
// Note: isolated-vm now works with proper native build environment

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

// Track if services have been loaded to avoid re-importing
let servicesLoaded = false;

// Import these lazily but store references
async function preloadServices() {
    if (servicesLoaded) return;

    try {
        ({ CacheService } = await import("../redisConn.js"));
        ({ QueueService } = await import("../tasks/queues.js"));
        ({ BusService } = await import("../services/bus.js"));
        ({ DbProvider } = await import("../db/provider.js"));
        servicesLoaded = true;
    } catch (e) {
        console.error("Failed to preload services:", e);
    }
}

// Setup QueueService mock early to prevent email service failures
import { SECONDS_30_MS } from "@vrooli/shared";
import { setupQueueServiceMock } from "./mocks/queueServiceMock.js";

setupQueueServiceMock();

// Process-specific state to track what's been initialized in THIS test process
// This fixes the ModelMap singleton issue where parallel test processes
// incorrectly shared initialization state across separate process boundaries
const processTestState = (() => {
    const processId = process.pid;
    const PROCESS_STATE_KEY = `__TEST_STATE_PID_${processId}__`;
    
    const state = (global as any)[PROCESS_STATE_KEY] || {
        componentsInitialized: {
            modelMap: false,
            dbProvider: false,
            mocks: false,
            idGenerator: false,
            socketService: false,
            bcryptService: false,
            i18n: false,
        },
        initCount: 0,
        processId: processId,
    };
    (global as any)[PROCESS_STATE_KEY] = state;
    
    // Debug log for parallel execution troubleshooting
    if (process.env.TEST_LOG_LEVEL === "DEBUG") {
        console.log(`Test setup using process-specific state for PID ${processId}`);
    }
    
    return state;
})();

const componentsInitialized = processTestState.componentsInitialized;

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

beforeAll(async function setup() {
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
        if (!componentsInitialized.modelMap) {
            try {
                const modelModule = await import("../models/base/index.js");

                // Handle both named and default exports
                let ModelMapClass = modelModule.ModelMap;
                if (!ModelMapClass && modelModule.default?.ModelMap) {
                    ModelMapClass = modelModule.default.ModelMap;
                }

                if (!ModelMapClass) {
                    // Try to find ModelMap in exports
                    const possibleExports = Object.keys(modelModule);
                    const classExport = possibleExports.find(key => {
                        const val = modelModule[key];
                        return typeof val === "function" && val.name === "ModelMap";
                    });

                    if (classExport) {
                        ModelMapClass = modelModule[classExport];
                    } else {
                        throw new Error("ModelMap is undefined after import");
                    }
                }

                // Enhanced ModelMap initialization with validation of critical models
                // These models are essential for most tests to function properly
                const criticalModels = [
                    "User",           // Core user functionality
                    "Team",           // Team operations
                    "Chat",           // Chat functionality  
                    "ChatMessage",    // Message handling
                    "Run",            // Routine execution
                    "Resource",       // Resource management
                    "Comment",        // Comments and interactions
                ];

                testLog("DEBUG", "Initializing ModelMap with critical models:", criticalModels.join(", "));
                await ModelMapClass.init(criticalModels);

                // Verify that ModelMap is actually functional
                if (!ModelMapClass.isInitialized()) {
                    throw new Error("ModelMap.init() succeeded but ModelMap.isInitialized() returns false");
                }

                // Additional verification: double-check User model specifically since it's the most commonly failing
                try {
                    const userModel = ModelMapClass.get("User", false);
                    if (!userModel) {
                        throw new Error("User model validation failed - model is null/undefined after successful init");
                    }
                    if (!userModel.__typename || userModel.__typename !== "User") {
                        throw new Error(`User model validation failed - invalid __typename: ${userModel.__typename}`);
                    }
                    testLog("DEBUG", "✓ User model validation passed");
                } catch (testError) {
                    throw new Error(`Critical User model validation failed: ${testError}`);
                }

                componentsInitialized.modelMap = true;
            } catch (error) {
                console.error("ModelMap initialization failed:", error);
                throw error; // Critical - tests cannot proceed without ModelMap
            }
        } else {
            testLog("DEBUG", `Step 1: ModelMap already initialized in process ${processTestState.processId}, skipping...`);
            // In parallel mode, this indicates the process-specific state is working correctly
            // Each process should initialize ModelMap independently, so this branch should rarely execute
            if (process.env.TEST_LOG_LEVEL === "DEBUG") {
                console.log(`Process ${process.pid}: ModelMap initialization skipped (already done in this process)`);
            }
        }

        // Step 2: Initialize i18n for notification translations
        if (!componentsInitialized.i18n) {
            testLog("DEBUG", "Step 2: Initializing i18n...");
            try {
                const { i18nConfig } = await import("@vrooli/shared");
                const i18nextModule = await import("i18next");
                const i18next = i18nextModule.default || i18nextModule;

                if (i18next && typeof i18next.init === "function") {
                    await i18next.init(i18nConfig(false));
                    componentsInitialized.i18n = true;
                    testLog("DEBUG", "✓ i18n ready");
                } else {
                    testLog("DEBUG", "✓ i18n init skipped (likely mocked)");
                }
            } catch (error) {
                testError("i18n initialization failed:", error);
                // Don't throw - tests can proceed without i18n but notifications may fail
            }
        } else {
            testLog("DEBUG", "Step 2: i18n already initialized, skipping...");
        }

        // Step 2.5: Initialize BcryptService for authentication operations
        testLog("DEBUG", "Step 2.5: Initializing BcryptService...");
        try {
            const { initializeBcryptService } = await import("../auth/bcryptWrapper.js");
            await initializeBcryptService();
            componentsInitialized.bcryptService = true;
            testLog("DEBUG", "✓ BcryptService ready");
        } catch (error) {
            testError("BcryptService initialization failed:", error);
            // Don't throw - tests can proceed without auth functionality but password operations will fail
        }

        // Step 3: Initialize DbProvider (idempotent - safe to call multiple times)
        testLog("DEBUG", "Step 3: Initializing DbProvider...");
        try {
            if (!DbProvider) {
                ({ DbProvider } = await import("../db/provider.js"));
            }
            // DbProvider.init() is idempotent - it checks if already initialized
            // Check if init method exists (may be missing in mocked versions)
            if (DbProvider && typeof DbProvider.init === "function") {
                await DbProvider.init();
                componentsInitialized.dbProvider = true;
                testLog("DEBUG", "✓ DbProvider ready");
            } else {
                testLog("DEBUG", "✓ DbProvider init skipped (likely mocked)");
                // When DbProvider is mocked, we don't need real initialization
                componentsInitialized.dbProvider = true;
            }
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

        console.log("=== Per-File Test Setup Complete ===");
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
}, SECONDS_30_MS);

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
    // Balanced cleanup - enough to prevent leaks, minimal enough to prevent hanging
    try {
        vi.clearAllMocks();
        vi.unstubAllGlobals();
        vi.clearAllTimers();

        // Reset services with timeout
        if (BusService?.reset) {
            const resetPromise = BusService.reset();
            const timeoutPromise = new Promise((_, reject) =>
                setTimeout(() => reject(new Error("BusService reset timeout")), 3000),
            );
            await Promise.race([resetPromise, timeoutPromise]).catch(() => {
                // If reset fails, just clear the instance
                (BusService as any).instance = null;
            });
        }

        // Clear singleton instances
        if (CacheService) (CacheService as any).instance = null;
        if (QueueService) (QueueService as any).instance = null;
    } catch (e) {
        // Ignore cleanup errors but try to force clear instances
        try {
            if (BusService) (BusService as any).instance = null;
            if (CacheService) (CacheService as any).instance = null;
            if (QueueService) (QueueService as any).instance = null;
        } catch (ignored) {
            // Final safety net
        }
    }
}

afterAll(async () => {
    try {
        await cleanup();
        if (global.gc) global.gc();
    } catch (error) {
        // Ignore all cleanup errors to prevent test skipping
    }
}, SECONDS_30_MS); // 30 second timeout

// Export status for tests to check
export const getSetupStatus = () => ({ ...componentsInitialized });
