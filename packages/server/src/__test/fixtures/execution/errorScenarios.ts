/**
 * Error Scenario Testing Framework (Phase 3 Improvement)
 * 
 * Provides comprehensive error scenario testing for execution fixtures,
 * validating graceful degradation, error recovery, and resilience patterns.
 * 
 * This ensures the AI execution architecture handles failures gracefully
 * and maintains system stability under adverse conditions.
 */

import { describe, it, expect } from "vitest";
import type { 
    BaseConfigObject,
    ChatConfigObject,
    RoutineConfigObject,
    RunConfigObject,
} from "@vrooli/shared";
import type { 
    ExecutionFixture,
    SwarmFixture,
    RoutineFixture,
    ExecutionContextFixture,
    ValidationResult,
} from "./executionValidationUtils.js";
import type { 
    ExecutionResult,
    ExecutionOptions,
    ErrorCondition,
} from "./executionRunner.js";
import { ExecutionFixtureRunner } from "./executionRunner.js";

// ================================================================================================
// Error Scenario Types
// ================================================================================================

export interface ErrorScenarioFixture<T extends BaseConfigObject> {
    /** Base fixture that will be tested under error conditions */
    baseFixture: ExecutionFixture<T>;
    /** Error condition to inject */
    errorCondition: DetailedErrorCondition;
    /** Expected behavior under the error condition */
    expectedBehavior: ExpectedErrorBehavior;
    /** Recovery actions that should be attempted */
    recoveryStrategy?: RecoveryStrategy;
    /** Test metadata */
    metadata?: {
        severity: "low" | "medium" | "high" | "critical";
        category: "config" | "resource" | "network" | "ai_model" | "system";
        description: string;
    };
}

export interface DetailedErrorCondition {
    /** Type of error to inject */
    type: "config_invalid" | "resource_limit" | "network_failure" | "ai_model_error" | 
          "timeout" | "memory_limit" | "token_limit" | "permission_denied" | 
          "external_service_down" | "data_corruption";
    /** Human-readable description */
    description: string;
    /** When to inject the error */
    injectionPoint: "validation" | "initialization" | "execution" | "emergence" | 
                   "integration" | "completion" | "cleanup";
    /** Error parameters */
    parameters?: {
        /** For resource limits */
        memoryLimit?: number;
        tokenLimit?: number;
        timeoutMs?: number;
        
        /** For network failures */
        networkLatency?: number;
        failureRate?: number;
        
        /** For AI model errors */
        modelErrorType?: "unavailable" | "rate_limited" | "invalid_response" | "hallucination";
        
        /** For data corruption */
        corruptionType?: "config" | "input" | "output" | "state";
        corruptionRate?: number;
    };
}

export interface ExpectedErrorBehavior {
    /** Whether execution should fail completely */
    shouldFail: boolean;
    /** Expected error message pattern (regex) */
    errorMessagePattern?: string;
    /** Capabilities that should gracefully degrade */
    gracefulDegradation?: string[];
    /** Fallback behaviors that should activate */
    fallbackBehaviors?: string[];
    /** Maximum acceptable latency under error conditions */
    maxLatencyMs?: number;
    /** Whether system should attempt recovery */
    shouldAttemptRecovery?: boolean;
    /** Minimum success rate acceptable under error */
    minSuccessRate?: number;
}

export interface RecoveryStrategy {
    /** Recovery actions to attempt in order */
    actions: RecoveryAction[];
    /** Maximum time to spend on recovery */
    maxRecoveryTimeMs: number;
    /** Whether to fail fast or keep retrying */
    strategy: "fail_fast" | "retry_with_backoff" | "graceful_degradation";
}

export interface RecoveryAction {
    type: "retry" | "fallback" | "reduce_complexity" | "use_cache" | "request_help";
    description: string;
    parameters?: Record<string, any>;
}

export interface ErrorTestResult {
    /** Basic execution result */
    executionResult: ExecutionResult;
    /** Whether error handling was correct */
    errorHandlingCorrect: boolean;
    /** Recovery actions that were attempted */
    recoveryActionsAttempted: string[];
    /** Whether graceful degradation occurred */
    gracefulDegradationOccurred: boolean;
    /** Error handling performance metrics */
    errorMetrics: {
        timeToDetectError: number;
        timeToStartRecovery: number;
        totalRecoveryTime: number;
        finalSuccessRate: number;
    };
}

// ================================================================================================
// Error Scenario Test Runner
// ================================================================================================

export class ErrorScenarioRunner {
    private executionRunner: ExecutionFixtureRunner;

    constructor() {
        this.executionRunner = new ExecutionFixtureRunner();
    }

    /**
     * Execute a single error scenario and validate behavior
     */
    async executeErrorScenario<T extends BaseConfigObject>(
        scenario: ErrorScenarioFixture<T>,
        inputData: any,
    ): Promise<ErrorTestResult> {
        const startTime = performance.now();
        let errorDetectedAt = 0;
        let recoveryStartedAt = 0;
        
        try {
            // Prepare execution options with error injection
            const options: ExecutionOptions = {
                testConditions: {
                    errorInjection: scenario.errorCondition,
                    expectedBehavior: scenario.expectedBehavior,
                },
                validateEmergence: true,
                mockExternalDeps: true,
            };

            // Execute with error injection
            const executionResult = await this.executionRunner.executeWithErrorInjection(
                scenario.baseFixture,
                scenario.errorCondition,
                inputData,
            );

            errorDetectedAt = this.extractErrorDetectionTime(executionResult);
            recoveryStartedAt = this.extractRecoveryStartTime(executionResult);

            // Validate error handling behavior
            const errorHandlingCorrect = this.validateErrorHandling(
                executionResult,
                scenario.expectedBehavior,
            );

            // Check graceful degradation
            const gracefulDegradationOccurred = this.checkGracefulDegradation(
                executionResult,
                scenario.expectedBehavior,
            );

            // Extract recovery actions
            const recoveryActionsAttempted = this.extractRecoveryActions(executionResult);

            const endTime = performance.now();

            return {
                executionResult,
                errorHandlingCorrect,
                recoveryActionsAttempted,
                gracefulDegradationOccurred,
                errorMetrics: {
                    timeToDetectError: errorDetectedAt - startTime,
                    timeToStartRecovery: recoveryStartedAt - startTime,
                    totalRecoveryTime: endTime - recoveryStartedAt,
                    finalSuccessRate: this.calculateSuccessRate(executionResult),
                },
            };
        } catch (error) {
            const endTime = performance.now();
            
            return {
                executionResult: {
                    success: false,
                    error: error instanceof Error ? error.message : String(error),
                    detectedCapabilities: [],
                    performanceMetrics: {
                        latency: endTime - startTime,
                        resourceUsage: { memory: 0, cpu: 0, tokens: 0 },
                    },
                },
                errorHandlingCorrect: false,
                recoveryActionsAttempted: [],
                gracefulDegradationOccurred: false,
                errorMetrics: {
                    timeToDetectError: errorDetectedAt - startTime,
                    timeToStartRecovery: 0,
                    totalRecoveryTime: 0,
                    finalSuccessRate: 0,
                },
            };
        }
    }

    /**
     * Execute multiple error scenarios and generate comprehensive report
     */
    async executeErrorScenarioSuite<T extends BaseConfigObject>(
        scenarios: ErrorScenarioFixture<T>[],
        inputData: any,
    ): Promise<ErrorScenarioSuiteResult> {
        const results: ErrorTestResult[] = [];
        const categoryResults: Record<string, ErrorTestResult[]> = {};
        
        for (const scenario of scenarios) {
            const result = await this.executeErrorScenario(scenario, inputData);
            results.push(result);
            
            const category = scenario.metadata?.category || "unknown";
            if (!categoryResults[category]) {
                categoryResults[category] = [];
            }
            categoryResults[category].push(result);
        }

        // Generate summary statistics
        const summary = this.generateErrorTestSummary(results);
        
        return {
            results,
            categoryResults,
            summary,
            overallResilience: this.calculateOverallResilience(results),
        };
    }

    // ================================================================================================
    // Private Helper Methods
    // ================================================================================================

    private validateErrorHandling(
        result: ExecutionResult,
        expected: ExpectedErrorBehavior,
    ): boolean {
        // Check if failure behavior matches expectation
        if (expected.shouldFail && result.success) {
            return false;
        }
        if (!expected.shouldFail && !result.success) {
            return false;
        }

        // Check error message pattern
        if (expected.errorMessagePattern && result.error) {
            const regex = new RegExp(expected.errorMessagePattern);
            if (!regex.test(result.error)) {
                return false;
            }
        }

        // Check latency constraints
        if (expected.maxLatencyMs && result.performanceMetrics.latency > expected.maxLatencyMs) {
            return false;
        }

        return true;
    }

    private checkGracefulDegradation(
        result: ExecutionResult,
        expected: ExpectedErrorBehavior,
    ): boolean {
        if (!expected.gracefulDegradation) {
            return true; // No graceful degradation expected
        }

        // Check if expected degradation capabilities are still present
        return expected.gracefulDegradation.every(capability =>
            result.detectedCapabilities.includes(capability),
        );
    }

    private extractRecoveryActions(result: ExecutionResult): string[] {
        // Extract recovery actions from execution trace
        if (!result.executionTrace) {
            return [];
        }

        return result.executionTrace
            .filter(entry => entry.action.includes("recovery") || entry.action.includes("fallback"))
            .map(entry => entry.action);
    }

    private extractErrorDetectionTime(result: ExecutionResult): number {
        if (!result.executionTrace) {
            return 0;
        }

        const errorEntry = result.executionTrace.find(entry => 
            entry.action.includes("error") || entry.action.includes("failure"),
        );
        
        return errorEntry?.timestamp || 0;
    }

    private extractRecoveryStartTime(result: ExecutionResult): number {
        if (!result.executionTrace) {
            return 0;
        }

        const recoveryEntry = result.executionTrace.find(entry => 
            entry.action.includes("recovery") || entry.action.includes("fallback"),
        );
        
        return recoveryEntry?.timestamp || 0;
    }

    private calculateSuccessRate(result: ExecutionResult): number {
        if (result.success) {
            return result.performanceMetrics.accuracy || 1.0;
        }
        return 0;
    }

    private generateErrorTestSummary(results: ErrorTestResult[]): ErrorTestSummary {
        const total = results.length;
        const correctlyHandled = results.filter(r => r.errorHandlingCorrect).length;
        const gracefulDegradations = results.filter(r => r.gracefulDegradationOccurred).length;
        
        const avgRecoveryTime = results.reduce((sum, r) => 
            sum + r.errorMetrics.totalRecoveryTime, 0) / total;
        
        const avgSuccessRate = results.reduce((sum, r) => 
            sum + r.errorMetrics.finalSuccessRate, 0) / total;

        return {
            totalScenarios: total,
            correctlyHandled,
            gracefulDegradations,
            averageRecoveryTime: avgRecoveryTime,
            averageSuccessRate: avgSuccessRate,
            errorHandlingRate: correctlyHandled / total,
            resilienceScore: this.calculateResilienceScore(results),
        };
    }

    private calculateOverallResilience(results: ErrorTestResult[]): ResilienceMetrics {
        const errorCategories = new Set<string>();
        let totalRecoveryActions = 0;
        let successfulRecoveries = 0;

        for (const result of results) {
            totalRecoveryActions += result.recoveryActionsAttempted.length;
            if (result.errorHandlingCorrect) {
                successfulRecoveries++;
            }
        }

        return {
            overallResilienceScore: this.calculateResilienceScore(results),
            errorCoverage: errorCategories.size,
            recoveryEffectiveness: successfulRecoveries / results.length,
            adaptabilityScore: totalRecoveryActions / results.length,
        };
    }

    private calculateResilienceScore(results: ErrorTestResult[]): number {
        // Calculate a composite resilience score (0-1)
        const weights = {
            errorHandling: 0.4,
            gracefulDegradation: 0.3,
            recoveryTime: 0.2,
            successRate: 0.1,
        };

        const errorHandlingScore = results.filter(r => r.errorHandlingCorrect).length / results.length;
        const gracefulDegradationScore = results.filter(r => r.gracefulDegradationOccurred).length / results.length;
        
        const avgRecoveryTime = results.reduce((sum, r) => sum + r.errorMetrics.totalRecoveryTime, 0) / results.length;
        const recoveryTimeScore = Math.max(0, 1 - (avgRecoveryTime / 10000)); // Penalize long recovery times
        
        const avgSuccessRate = results.reduce((sum, r) => sum + r.errorMetrics.finalSuccessRate, 0) / results.length;

        return (
            weights.errorHandling * errorHandlingScore +
            weights.gracefulDegradation * gracefulDegradationScore +
            weights.recoveryTime * recoveryTimeScore +
            weights.successRate * avgSuccessRate
        );
    }
}

// ================================================================================================
// Supporting Types
// ================================================================================================

export interface ErrorScenarioSuiteResult {
    results: ErrorTestResult[];
    categoryResults: Record<string, ErrorTestResult[]>;
    summary: ErrorTestSummary;
    overallResilience: ResilienceMetrics;
}

export interface ErrorTestSummary {
    totalScenarios: number;
    correctlyHandled: number;
    gracefulDegradations: number;
    averageRecoveryTime: number;
    averageSuccessRate: number;
    errorHandlingRate: number;
    resilienceScore: number;
}

export interface ResilienceMetrics {
    overallResilienceScore: number;
    errorCoverage: number;
    recoveryEffectiveness: number;
    adaptabilityScore: number;
}

// ================================================================================================
// Predefined Error Scenarios
// ================================================================================================

/**
 * Create comprehensive error scenarios for a given fixture
 */
export function createStandardErrorScenarios<T extends BaseConfigObject>(
    baseFixture: ExecutionFixture<T>,
): ErrorScenarioFixture<T>[] {
    return [
        // Resource limitation scenarios
        {
            baseFixture,
            errorCondition: {
                type: "memory_limit",
                description: "Memory usage exceeds available resources",
                injectionPoint: "execution",
                parameters: { memoryLimit: 512 }, // 512MB limit
            },
            expectedBehavior: {
                shouldFail: false,
                gracefulDegradation: ["basic_operation"],
                fallbackBehaviors: ["reduce_complexity", "use_simpler_model"],
                maxLatencyMs: 10000,
            },
            metadata: {
                severity: "medium",
                category: "resource",
                description: "Tests behavior under memory constraints",
            },
        },

        // Network failure scenarios
        {
            baseFixture,
            errorCondition: {
                type: "network_failure",
                description: "Network connectivity is intermittent",
                injectionPoint: "integration",
                parameters: { networkLatency: 5000, failureRate: 0.3 },
            },
            expectedBehavior: {
                shouldFail: false,
                gracefulDegradation: ["offline_operation"],
                fallbackBehaviors: ["use_cache", "retry_with_backoff"],
                maxLatencyMs: 15000,
                shouldAttemptRecovery: true,
            },
            metadata: {
                severity: "high",
                category: "network",
                description: "Tests resilience to network issues",
            },
        },

        // AI model failure scenarios
        {
            baseFixture,
            errorCondition: {
                type: "ai_model_error",
                description: "AI model returns invalid responses",
                injectionPoint: "execution",
                parameters: { modelErrorType: "invalid_response" },
            },
            expectedBehavior: {
                shouldFail: false,
                gracefulDegradation: ["basic_response"],
                fallbackBehaviors: ["use_fallback_model", "template_response"],
                shouldAttemptRecovery: true,
            },
            metadata: {
                severity: "high",
                category: "ai_model",
                description: "Tests handling of AI model failures",
            },
        },

        // Configuration corruption scenarios
        {
            baseFixture,
            errorCondition: {
                type: "data_corruption",
                description: "Configuration data is corrupted",
                injectionPoint: "validation",
                parameters: { corruptionType: "config", corruptionRate: 0.1 },
            },
            expectedBehavior: {
                shouldFail: true,
                errorMessagePattern: ".*config.*corrupt.*|.*validation.*failed.*",
                shouldAttemptRecovery: false,
            },
            metadata: {
                severity: "critical",
                category: "config",
                description: "Tests handling of corrupted configuration",
            },
        },
    ];
}

// ================================================================================================
// Test Utilities
// ================================================================================================

/**
 * Run comprehensive error scenario tests for a fixture
 */
export function runErrorScenarioTests<T extends BaseConfigObject>(
    scenarios: ErrorScenarioFixture<T>[],
    fixtureName: string,
    inputData: any = { query: "test input" },
): void {
    describe(`${fixtureName} error scenarios`, () => {
        const runner = new ErrorScenarioRunner();

        scenarios.forEach((scenario, index) => {
            const testName = `should handle ${scenario.errorCondition.type} ${scenario.metadata?.severity ? `(${scenario.metadata.severity})` : ""}`;
            
            it(testName, async () => {
                const result = await runner.executeErrorScenario(scenario, inputData);
                
                // Validate error handling
                expect(result.errorHandlingCorrect).toBe(true);
                
                // Validate graceful degradation if expected
                if (scenario.expectedBehavior.gracefulDegradation) {
                    expect(result.gracefulDegradationOccurred).toBe(true);
                }
                
                // Validate recovery if expected
                if (scenario.expectedBehavior.shouldAttemptRecovery) {
                    expect(result.recoveryActionsAttempted.length).toBeGreaterThan(0);
                }
                
                // Validate performance constraints
                if (scenario.expectedBehavior.maxLatencyMs) {
                    expect(result.executionResult.performanceMetrics.latency)
                        .toBeLessThanOrEqual(scenario.expectedBehavior.maxLatencyMs);
                }
            }, 30000); // 30 second timeout for error scenarios
        });

        it("should maintain overall resilience across all error scenarios", async () => {
            const suiteResult = await runner.executeErrorScenarioSuite(scenarios, inputData);
            
            // Overall resilience should be above threshold
            expect(suiteResult.overallResilience.overallResilienceScore).toBeGreaterThan(0.7);
            
            // Should handle majority of errors correctly
            expect(suiteResult.summary.errorHandlingRate).toBeGreaterThan(0.8);
            
            // Should demonstrate recovery effectiveness
            expect(suiteResult.overallResilience.recoveryEffectiveness).toBeGreaterThan(0.6);
        }, 60000); // 1 minute timeout for full suite
    });
}
