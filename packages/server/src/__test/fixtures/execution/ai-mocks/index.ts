/**
 * AI Mock Fixtures - Main Entry Point
 * 
 * Comprehensive export of all AI mock functionality for testing AI-powered features.
 */

// Types
export * from "./types.js";

// Pre-defined fixtures
export * from "./fixtures/index.js";

// Validators
export * from "./validators/index.js";

// Behavior patterns
export * from "./behaviors/index.js";

// Integration utilities
export * from "./integration/index.js";

// Re-export commonly used items for convenience
export {
    createAIMockResponse, createErrorResponse, createStreamingMock,
    createToolCallResponse,
} from "./factories/index.js";

export {
    aiErrorFixtures, aiReasoningFixtures, aiStreamingFixtures, aiSuccessFixtures, aiToolCallFixtures,
} from "./fixtures/index.js";

export {
    createEmergentMockBehavior, registerMockBehavior, withAIMocks,
} from "./integration/index.js";

/**
 * Quick start helper for common test scenarios
 */
export const setupAIMocks = () => {
    // Import what we need
    const { MockRegistry } = require("./integration/mockRegistry.js");
    const { aiSuccessFixtures } = require("./fixtures/index.js");

    // Set up default behaviors
    const registry = MockRegistry.getInstance();
    registry.clear();

    // Register common patterns
    registry.register("greeting", {
        pattern: /hello|hi|hey/i,
        response: aiSuccessFixtures.greeting(),
    });

    registry.register("help", {
        pattern: /help|assist|support/i,
        response: aiSuccessFixtures.helpfulResponse(),
    });

    registry.register("error", {
        pattern: /error|fail|problem/i,
        response: {
            error: {
                type: "INTERNAL_ERROR",
                message: "Simulated error for testing",
            },
        },
    });

    return registry;
};
