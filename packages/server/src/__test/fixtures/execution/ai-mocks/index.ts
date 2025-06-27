/**
 * AI Mock Fixtures - Main Entry Point
 * 
 * Comprehensive export of all AI mock functionality for testing AI-powered features.
 */

// Types
export * from "./types.js";

// Factories
export * from "./factories/index.js";

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
    createAIMockResponse,
    createStreamingMock,
    createToolCallResponse,
    createErrorResponse, 
} from "./factories/index.js";

export {
    aiSuccessFixtures,
    aiErrorFixtures,
    aiStreamingFixtures,
    aiToolCallFixtures,
    aiReasoningFixtures,
} from "./fixtures/index.js";

export {
    withAIMocks,
    registerMockBehavior,
    createEmergentMockBehavior,
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
