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
// NOTE: These services don't exist in the shared package - it's a pure utility package
// The shared package doesn't need Redis, Queue, Bus, or DB services

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
    testLog("DEBUG", "=== Shared Package Test Setup Starting ===");
    
    try {
        // The shared package is a pure utility package and doesn't need:
        // - Redis/Queue/Bus services (server-only)
        // - Database connections (server-only)
        // - ModelMap (server-only)
        // - Socket services (server-only)
        
        // Only initialize i18n if it's actually used by shared package tests
        testLog("DEBUG", "Step 1: Initializing i18n...");
        try {
            const { i18nConfig } = await import("@vrooli/shared");
            const i18next = (await import("i18next")).default;
            await i18next.init(i18nConfig(false)); // false for production mode in tests
            testLog("DEBUG", "✓ i18n ready");
        } catch (error) {
            testLog("DEBUG", "i18n initialization skipped (not needed for utility tests):", error);
            // Don't throw - shared package tests don't need i18n
        }
        
        testLog("DEBUG", "=== Shared Package Test Setup Complete ===");
        
    } catch (error) {
        testError("=== Setup Failed ===");
        testError(error);
        throw error;
    }
}, 30000);

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
    // Minimal cleanup for shared package tests
    try {
        // Basic mock cleanup
        vi.clearAllMocks();
        vi.unstubAllGlobals();
        vi.clearAllTimers();
    } catch (e) {
        // Ignore all cleanup errors to prevent test skipping
    }
}

afterAll(async () => {
    try {
        await cleanup();
        // Force garbage collection if available to help with memory
        if (global.gc) global.gc();
    } catch (error) {
        // Never throw errors in afterAll to prevent test skipping
    }
}, 5000); // Short timeout to prevent hanging

// Export status for tests to check
export const getSetupStatus = () => ({ ...componentsInitialized });
