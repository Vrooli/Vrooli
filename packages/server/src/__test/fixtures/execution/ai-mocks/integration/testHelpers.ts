/**
 * Test Helpers
 * 
 * Utility functions to simplify AI mock usage in tests.
 */

import type { LLMRequest } from "@vrooli/shared";
import type { 
    AIMockService, 
    MockTestOptions, 
    EmergentMockBehavior,
    MockScenario,
    MockCostTracking,
    AIMockConfig,
} from "../types.js";
import { MockRegistry } from "./mockRegistry.js";
import { LLMIntegrationService } from "../../../../../services/execution/integration/llmIntegrationService.js";
import { createLogger } from "winston";

/**
 * Setup AI mocks for testing
 */
export async function withAIMocks<T>(
    testFn: (mockService: AIMockService) => Promise<T>,
    options?: MockTestOptions,
): Promise<T> {
    const registry = MockRegistry.getInstance();
    
    // Apply options
    if (options?.debug) {
        registry.setDebugMode(true);
    }
    
    // Clear any existing behaviors
    registry.clearBehaviors();
    
    try {
        // Run the test
        const result = await testFn(registry);
        
        // Validate responses if requested
        if (options?.validateResponses) {
            const interactions = registry.getCapturedInteractions();
            for (const { response } of interactions) {
                // Validation would be done here
            }
        }
        
        return result;
    } finally {
        // Cleanup
        registry.clearBehaviors();
        registry.setDebugMode(false);
    }
}

/**
 * Create a mock LLM integration service
 */
export function createMockLLMService(options?: {
    defaultModel?: string;
    registry?: MockRegistry;
}): LLMIntegrationService {
    const registry = options?.registry || MockRegistry.getInstance();
    const logger = createLogger({ silent: true });
    
    // Create a mock service that uses the registry
    const mockService = new LLMIntegrationService(logger);
    
    // Override the execute method to use mock registry
    mockService.executeRequest = async (request, resources, userData) => {
        const mockResult = await registry.execute(request);
        
        return {
            ...mockResult.response,
            resourceUsage: {
                tokens: mockResult.usage?.totalTokens || 0,
                apiCalls: 1,
                computeTime: 50,
                cost: mockResult.usage?.cost || 0,
            },
        };
    };
    
    return mockService;
}

/**
 * Create emergent behavior mocks
 */
export function createEmergentMockBehavior(config: EmergentMockBehavior): {
    getCurrentBehavior: (iteration: number) => AIMockConfig;
    evolve: (iteration: number) => void;
} {
    let currentIteration = 0;
    
    // eslint-disable-next-line func-style
    const getCurrentBehavior = (iteration: number): AIMockConfig => {
        currentIteration = iteration;
        
        // Find the appropriate evolution stage
        const stage = config.evolution
            .filter(e => e.iteration <= iteration)
            .sort((a, b) => b.iteration - a.iteration)[0];
        
        if (!stage) {
            return config.evolution[0].response;
        }
        
        // Check for regression
        if (config.regressionChance && Math.random() < config.regressionChance) {
            const previousStage = config.evolution[Math.max(0, config.evolution.indexOf(stage) - 1)];
            return previousStage.response;
        }
        
        return stage.response;
    };
    
    // eslint-disable-next-line func-style
    const evolve = (iteration: number) => {
        currentIteration = iteration;
    };
    
    // Register with the mock registry
    MockRegistry.getInstance().registerBehavior(`emergent-${config.capability}`, {
        response: (request) => getCurrentBehavior(currentIteration),
        metadata: {
            name: `Emergent: ${config.capability}`,
            description: "Evolving behavior based on iterations",
        },
    });
    
    return { getCurrentBehavior, evolve };
}

/**
 * Run a mock scenario
 */
export async function runMockScenario(
    scenario: MockScenario,
    options?: MockTestOptions,
): Promise<{
    success: boolean;
    results: Array<{ step: string; passed: boolean; error?: string }>;
}> {
    const results: Array<{ step: string; passed: boolean; error?: string }> = [];
    
    await withAIMocks(async (mockService) => {
        for (const step of scenario.steps) {
            try {
                // Register the mock for this step
                mockService.registerBehavior(`${scenario.name}-step-${scenario.steps.indexOf(step)}`, {
                    response: step.mockConfig,
                });
                
                // Create request
                const request: LLMRequest = {
                    model: step.input.model || "gpt-4o-mini",
                    messages: step.input.messages || [],
                    ...step.input,
                };
                
                // Execute
                const result = await mockService.execute(request);
                
                // Run assertions if provided
                if (step.assertions) {
                    step.assertions(result.response);
                }
                
                results.push({
                    step: step.expectedBehavior,
                    passed: true,
                });
            } catch (error) {
                results.push({
                    step: step.expectedBehavior,
                    passed: false,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
    }, options);
    
    const success = results.every(r => r.passed);
    return { success, results };
}

/**
 * Enable debug logging for AI mocks
 */
export function enableAIMockDebug(enabled = true): void {
    MockRegistry.getInstance().setDebugMode(enabled);
}

/**
 * Cost tracking wrapper
 */
export async function withCostTracking<T>(
    testFn: (tracker: CostTracker) => Promise<T>,
): Promise<T> {
    const tracker = new CostTracker();
    const registry = MockRegistry.getInstance();
    
    // Intercept mock responses to track costs
    const originalExecute = registry.execute.bind(registry);
    registry.execute = async (request) => {
        const result = await originalExecute(request);
        tracker.addUsage(result);
        return result;
    };
    
    try {
        return await testFn(tracker);
    } finally {
        // Restore original execute method
        registry.execute = originalExecute;
    }
}

/**
 * Cost tracker implementation
 */
class CostTracker {
    private tracking: MockCostTracking = {
        totalCost: 0,
        inputTokens: 0,
        outputTokens: 0,
        requests: 0,
        breakdown: [],
    };
    
    addUsage(result: any): void {
        const usage = result.usage;
        if (!usage) return;
        
        this.tracking.inputTokens += usage.promptTokens || 0;
        this.tracking.outputTokens += usage.completionTokens || 0;
        this.tracking.totalCost += usage.cost || 0;
        this.tracking.requests++;
        
        this.tracking.breakdown.push({
            model: result.response?.model || "unknown",
            inputTokens: usage.promptTokens || 0,
            outputTokens: usage.completionTokens || 0,
            cost: usage.cost || 0,
            timestamp: new Date(),
        });
    }
    
    getTotalCost(): number {
        return this.tracking.totalCost;
    }
    
    getBreakdown(): MockCostTracking["breakdown"] {
        return [...this.tracking.breakdown];
    }
    
    getSummary(): Omit<MockCostTracking, "breakdown"> {
        return {
            totalCost: this.tracking.totalCost,
            inputTokens: this.tracking.inputTokens,
            outputTokens: this.tracking.outputTokens,
            requests: this.tracking.requests,
        };
    }
    
    reset(): void {
        this.tracking = {
            totalCost: 0,
            inputTokens: 0,
            outputTokens: 0,
            requests: 0,
            breakdown: [],
        };
    }
}

/**
 * Create a test request
 */
export function createTestRequest(overrides?: Partial<LLMRequest>): LLMRequest {
    return {
        model: "gpt-4o-mini",
        messages: [
            { role: "user", content: "Test message" },
        ],
        ...overrides,
    };
}

/**
 * Assert mock behavior was used
 */
export function assertBehaviorUsed(behaviorId: string, minTimes = 1): void {
    const stats = MockRegistry.getInstance().getStats();
    const uses = stats.behaviorHits[behaviorId] || 0;
    
    if (uses < minTimes) {
        throw new Error(`Expected behavior '${behaviorId}' to be used at least ${minTimes} times, but was used ${uses} times`);
    }
}

/**
 * Get mock interaction history
 */
export function getMockInteractionHistory(): Array<{
    request: LLMRequest;
    response: any;
}> {
    return MockRegistry.getInstance().getCapturedInteractions();
}

/**
 * Clear mock interaction history
 */
export function clearMockInteractionHistory(): void {
    MockRegistry.getInstance().clearCapturedInteractions();
}

/**
 * Wait for mock condition
 */
export async function waitForMockCondition(
    condition: () => boolean,
    timeout = 5000,
    interval = 100,
): Promise<void> {
    const startTime = Date.now();
    
    while (!condition()) {
        if (Date.now() - startTime > timeout) {
            throw new Error("Timeout waiting for mock condition");
        }
        await new Promise(resolve => setTimeout(resolve, interval));
    }
}

/**
 * Create a mock response matcher
 */
export function createResponseMatcher(expected: Partial<AIMockConfig>): (actual: any) => boolean {
    return (actual: any) => {
        if (expected.content && actual.content !== expected.content) return false;
        if (expected.confidence && actual.confidence !== expected.confidence) return false;
        if (expected.model && actual.model !== expected.model) return false;
        if (expected.toolCalls && JSON.stringify(actual.toolCalls) !== JSON.stringify(expected.toolCalls)) return false;
        return true;
    };
}
