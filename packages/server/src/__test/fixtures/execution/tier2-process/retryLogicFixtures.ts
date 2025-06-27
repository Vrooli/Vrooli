/**
 * Execution Fixtures for Branch Coordinator Retry Logic Testing
 * 
 * These fixtures test the retry functionality implementation in BranchCoordinator,
 * specifically targeting the TODO at line 1157. They provide comprehensive
 * test scenarios for exponential backoff, failure recovery, and error handling.
 */

import { describe, it, expect } from "vitest";
import { routineConfigFixtures } from "@vrooli/shared";
import {
    type RoutineFixture,
    type ValidationResult,
    runComprehensiveExecutionTests,
    executionErrorScenarios,
    FixtureCreationUtils,
} from "../executionValidationUtils.js";

// ================================================================================================
// Retry Logic Test Configuration
// ================================================================================================

export interface RetryConfig {
    maxRetries: number;
    baseDelayMs: number;
    maxDelayMs: number;
    backoffMultiplier: number;
    jitterRange?: number; // 0-1, adds randomization to prevent thundering herd
}

export interface RetryTestScenario {
    name: string;
    config: RetryConfig;
    failurePattern: number[]; // Which attempts should fail (0-indexed)
    expectedAttempts: number;
    expectedSuccess: boolean;
    expectedTotalDelay: number; // Approximate expected delay in ms
    errorType: "timeout" | "network_error" | "tool_failure" | "resource_exhaustion";
}

export interface BranchExecutionResult {
    success: boolean;
    attempts: number;
    totalDelay: number;
    finalError?: string;
    retryDelays: number[];
}

// ================================================================================================
// Core Retry Logic Fixtures
// ================================================================================================

/**
 * Basic retry configuration with conservative defaults
 */
export const basicRetryConfig: RetryConfig = {
    maxRetries: 3,
    baseDelayMs: 100,
    maxDelayMs: 5000,
    backoffMultiplier: 2.0,
    jitterRange: 0.1,
};

/**
 * Aggressive retry configuration for critical operations
 */
export const aggressiveRetryConfig: RetryConfig = {
    maxRetries: 5,
    baseDelayMs: 50,
    maxDelayMs: 10000,
    backoffMultiplier: 2.5,
    jitterRange: 0.2,
};

/**
 * Conservative retry configuration for resource-intensive operations
 */
export const conservativeRetryConfig: RetryConfig = {
    maxRetries: 2,
    baseDelayMs: 500,
    maxDelayMs: 2000,
    backoffMultiplier: 1.5,
    jitterRange: 0.05,
};

// ================================================================================================
// Retry Test Scenarios
// ================================================================================================

export const retryTestScenarios: Record<string, RetryTestScenario> = {
    /**
     * Immediate success - no retries needed
     */
    immediateSuccess: {
        name: "Immediate Success",
        config: basicRetryConfig,
        failurePattern: [], // No failures
        expectedAttempts: 1,
        expectedSuccess: true,
        expectedTotalDelay: 0,
        errorType: "timeout",
    },

    /**
     * Single failure followed by success
     */
    singleFailureRecovery: {
        name: "Single Failure Recovery",
        config: basicRetryConfig,
        failurePattern: [0], // First attempt fails
        expectedAttempts: 2,
        expectedSuccess: true,
        expectedTotalDelay: 100, // Base delay
        errorType: "network_error",
    },

    /**
     * Multiple failures with eventual success
     */
    multipleFailureRecovery: {
        name: "Multiple Failure Recovery",
        config: basicRetryConfig,
        failurePattern: [0, 1], // First two attempts fail
        expectedAttempts: 3,
        expectedSuccess: true,
        expectedTotalDelay: 300, // 100 + 200 (exponential backoff)
        errorType: "tool_failure",
    },

    /**
     * Maximum retries exhausted - final failure
     */
    exhaustedRetries: {
        name: "Exhausted Retries",
        config: basicRetryConfig,
        failurePattern: [0, 1, 2, 3], // All attempts fail
        expectedAttempts: 4, // Initial + 3 retries
        expectedSuccess: false,
        expectedTotalDelay: 700, // 100 + 200 + 400 (capped by maxRetries)
        errorType: "resource_exhaustion",
    },

    /**
     * Aggressive retry with eventual success
     */
    aggressiveRetrySuccess: {
        name: "Aggressive Retry Success",
        config: aggressiveRetryConfig,
        failurePattern: [0, 1, 2, 3], // Four failures before success
        expectedAttempts: 5,
        expectedSuccess: true,
        expectedTotalDelay: 387, // 50 + 125 + 312 (50 * 2.5^n with jitter)
        errorType: "timeout",
    },

    /**
     * Conservative retry with quick failure
     */
    conservativeRetryFailure: {
        name: "Conservative Retry Failure",
        config: conservativeRetryConfig,
        failurePattern: [0, 1, 2], // All attempts fail with conservative config
        expectedAttempts: 3, // Initial + 2 retries
        expectedSuccess: false,
        expectedTotalDelay: 1250, // 500 + 750 (500 * 1.5)
        errorType: "network_error",
    },

    /**
     * Delay cap testing - ensures delays don't exceed maximum
     */
    delayCappingTest: {
        name: "Delay Capping Test",
        config: {
            maxRetries: 10,
            baseDelayMs: 1000,
            maxDelayMs: 3000, // Lower cap to test capping
            backoffMultiplier: 3.0, // High multiplier to trigger cap
            jitterRange: 0,
        },
        failurePattern: [0, 1, 2, 3, 4], // Multiple failures to test capping
        expectedAttempts: 6,
        expectedSuccess: true,
        expectedTotalDelay: 15000, // Should be capped: 1000 + 3000 + 3000 + 3000 + 3000 + 3000
        errorType: "timeout",
    },
};

// ================================================================================================
// Branch Coordinator Fixtures for Retry Testing
// ================================================================================================

/**
 * Basic routine fixture for retry testing
 */
export const retryTestRoutineFixture: RoutineFixture = FixtureCreationUtils.createCompleteFixture(
    routineConfigFixtures.minimal.valid,
    "routine",
    {
        emergence: {
            capabilities: ["retry_logic", "exponential_backoff", "failure_recovery"],
            evolutionPath: "basic_retry → intelligent_retry → adaptive_retry",
            emergenceConditions: {
                minAgents: 1,
                requiredResources: ["retry_scheduler", "failure_detector"],
                environmentalFactors: ["network_stability"],
            },
            learningMetrics: {
                performanceImprovement: "retry_success_rate",
                adaptationTime: "milliseconds",
                innovationRate: "high",
            },
        },
        integration: {
            tier: "tier2",
            producedEvents: [
                "tier2.branch.retry_started",
                "tier2.branch.retry_succeeded", 
                "tier2.branch.retry_failed",
                "tier2.branch.retry_exhausted",
            ],
            consumedEvents: [
                "tier2.branch.execution_failed",
                "tier2.system.retry_config_updated",
            ],
            sharedResources: ["retry_scheduler", "failure_metrics"],
            crossTierDependencies: {
                dependsOn: ["tier3.execution_engine", "tier2.state_store"],
                provides: ["retry_statistics", "failure_patterns"],
            },
        },
        evolutionStage: {
            current: "reasoning",
            nextStage: "deterministic",
            evolutionTriggers: ["retry_pattern_recognition", "failure_prediction"],
            performanceMetrics: {
                averageExecutionTime: 2500, // Including retry delays
                successRate: 0.95, // High success rate with retries
                costPerExecution: 0.08, // Higher due to retries
            },
        },
        errorScenarios: [
            // Timeout scenario with retry
            {
                errorType: "timeout",
                context: {
                    tier: "tier2",
                    operation: "branch_execution",
                    step: 3,
                },
                expectedError: {
                    code: "operation_timed_out",
                    trace: "T2BR-EXEC",
                    data: {
                        errorType: "timeout",
                        tier: "tier2",
                        component: "branch_coordinator",
                        operation: "branch_execution",
                        timestamp: new Date().toISOString(),
                    },
                    executionContext: {
                        tier: "tier2",
                        component: "branch_coordinator",
                        operation: "branch_execution",
                        timestamp: new Date().toISOString(),
                    },
                    executionImpact: {
                        tierAffected: ["tier2"],
                        resourcesAffected: ["routine_state", "execution_context"],
                        cascadingEffects: false,
                        recoverability: "automatic",
                    },
                    resilienceMetadata: {
                        errorClassification: "execution.tier2.timeout",
                        retryable: true,
                        circuitBreakerTriggered: true,
                        fallbackAvailable: true,
                    },
                    toServerError() {
                        return {
                            trace: this.trace,
                            code: this.code,
                            data: this.data,
                        };
                    },
                    getSeverity() {
                        return "Warning" as const;
                    },
                },
                recovery: {
                    strategy: "retry",
                    maxAttempts: 3,
                    fallbackBehavior: "exponential_backoff",
                },
                validation: {
                    shouldRecover: true,
                    timeoutMs: 10000,
                    expectedFinalState: "retry_succeeded",
                    preserveProgress: true,
                },
            },
            // Resource exhaustion scenario
            {
                errorType: "resource_exhaustion",
                context: {
                    tier: "tier2",
                    operation: "parallel_branch_execution",
                    resource: "memory_pool",
                },
                expectedError: {
                    code: "insufficient_resources",
                    trace: "T2BR-RSRC",
                    data: {
                        errorType: "resource_exhaustion",
                        tier: "tier2",
                        component: "resource_manager",
                        operation: "parallel_branch_execution",
                        timestamp: new Date().toISOString(),
                    },
                    executionContext: {
                        tier: "tier2",
                        component: "resource_manager",
                        operation: "parallel_branch_execution",
                        timestamp: new Date().toISOString(),
                    },
                    executionImpact: {
                        tierAffected: ["tier2"],
                        resourcesAffected: ["memory_pool", "cpu_scheduler"],
                        cascadingEffects: true,
                        recoverability: "automatic",
                    },
                    resilienceMetadata: {
                        errorClassification: "execution.tier2.resource_exhaustion",
                        retryable: true,
                        circuitBreakerTriggered: true,
                        fallbackAvailable: true,
                    },
                    toServerError() {
                        return {
                            trace: this.trace,
                            code: this.code,
                            data: this.data,
                        };
                    },
                    getSeverity() {
                        return "Error" as const;
                    },
                },
                recovery: {
                    strategy: "retry",
                    maxAttempts: 2,
                    fallbackBehavior: "reduce_parallelization",
                },
                validation: {
                    shouldRecover: true,
                    timeoutMs: 15000,
                    expectedFinalState: "degraded_execution",
                    preserveProgress: false,
                },
            },
        ],
        metadata: {
            domain: "process_orchestration",
            complexity: "medium",
            maintainer: "retry_logic_team",
            lastUpdated: new Date().toISOString(),
        },
    },
);

/**
 * Network resilience routine fixture - tests retry under network conditions
 */
export const networkResilienceRoutineFixture: RoutineFixture = FixtureCreationUtils.createCompleteFixture(
    routineConfigFixtures.minimal.valid,
    "routine",
    {
        emergence: {
            capabilities: ["network_resilience", "connection_recovery", "adaptive_timeout"],
            evolutionPath: "fixed_timeout → adaptive_timeout → predictive_recovery",
            emergenceConditions: {
                minAgents: 1,
                requiredResources: ["network_monitor", "connection_pool"],
                environmentalFactors: ["network_stability", "latency_variance"],
            },
        },
        integration: {
            tier: "tier2",
            producedEvents: [
                "tier2.network.connection_lost",
                "tier2.network.connection_recovered",
                "tier2.network.retry_pattern_detected",
            ],
            consumedEvents: [
                "tier2.network.stability_changed",
                "tier3.tool.network_error",
            ],
        },
        evolutionStage: {
            current: "conversational",
            nextStage: "reasoning",
            evolutionTriggers: ["network_pattern_recognition"],
            performanceMetrics: {
                averageExecutionTime: 3000,
                successRate: 0.88, // Lower due to network issues
                costPerExecution: 0.12,
            },
        },
        errorScenarios: [
            {
                errorType: "network_error",
                context: {
                    tier: "tier2",
                    operation: "external_api_call",
                },
                expectedError: {
                    code: "network_connection_failed",
                    trace: "T2NW-CONN",
                    data: {
                        errorType: "network_error",
                        tier: "tier2",
                        component: "network_adapter",
                        operation: "external_api_call",
                        timestamp: new Date().toISOString(),
                    },
                    executionContext: {
                        tier: "tier2",
                        component: "network_adapter",
                        operation: "external_api_call",
                        timestamp: new Date().toISOString(),
                    },
                    executionImpact: {
                        tierAffected: ["tier2"],
                        resourcesAffected: ["network_adapter", "connection_pool"],
                        cascadingEffects: true,
                        recoverability: "automatic",
                    },
                    resilienceMetadata: {
                        errorClassification: "execution.tier2.network_error",
                        retryable: true,
                        circuitBreakerTriggered: false,
                        fallbackAvailable: true,
                    },
                    toServerError() {
                        return {
                            trace: this.trace,
                            code: this.code,
                            data: this.data,
                        };
                    },
                    getSeverity() {
                        return "Warning" as const;
                    },
                },
                recovery: {
                    strategy: "retry",
                    maxAttempts: 5,
                    fallbackBehavior: "exponential_backoff_with_jitter",
                },
                validation: {
                    shouldRecover: true,
                    timeoutMs: 20000,
                    expectedFinalState: "network_recovered",
                    preserveProgress: true,
                },
            },
        ],
    },
);

/**
 * High-throughput routine fixture - tests retry under load
 */
export const highThroughputRoutineFixture: RoutineFixture = FixtureCreationUtils.createCompleteFixture(
    routineConfigFixtures.minimal.valid,
    "routine",
    {
        emergence: {
            capabilities: ["high_throughput", "load_balancing", "circuit_breaking"],
            evolutionPath: "sequential → parallel → adaptive_parallel",
            emergenceConditions: {
                minAgents: 5,
                requiredResources: ["thread_pool", "memory_manager", "circuit_breaker"],
                environmentalFactors: ["system_load", "available_cores"],
            },
        },
        integration: {
            tier: "tier2",
            producedEvents: [
                "tier2.load.threshold_exceeded",
                "tier2.load.circuit_breaker_opened",
                "tier2.load.throughput_degraded",
            ],
            consumedEvents: [
                "tier2.system.load_changed",
                "tier3.resource.exhaustion_warning",
            ],
        },
        evolutionStage: {
            current: "deterministic",
            evolutionTriggers: ["load_optimization", "throughput_improvement"],
            performanceMetrics: {
                averageExecutionTime: 500, // Optimized for speed
                successRate: 0.92,
                costPerExecution: 0.15, // Higher due to resource usage
            },
        },
        errorScenarios: [
            {
                errorType: "resource_exhaustion",
                context: {
                    tier: "tier2",
                    operation: "high_throughput_processing",
                    resource: "thread_pool",
                },
                expectedError: {
                    code: "insufficient_resources",
                    trace: "T2HT-THRD",
                    data: {
                        errorType: "resource_exhaustion",
                        tier: "tier2",
                        component: "thread_manager",
                        operation: "high_throughput_processing",
                        timestamp: new Date().toISOString(),
                    },
                    executionContext: {
                        tier: "tier2",
                        component: "thread_manager",
                        operation: "high_throughput_processing",
                        timestamp: new Date().toISOString(),
                    },
                    executionImpact: {
                        tierAffected: ["tier2"],
                        resourcesAffected: ["thread_pool", "memory_manager"],
                        cascadingEffects: true,
                        recoverability: "automatic",
                    },
                    resilienceMetadata: {
                        errorClassification: "execution.tier2.resource_exhaustion",
                        retryable: true,
                        circuitBreakerTriggered: true,
                        fallbackAvailable: true,
                    },
                    toServerError() {
                        return {
                            trace: this.trace,
                            code: this.code,
                            data: this.data,
                        };
                    },
                    getSeverity() {
                        return "Error" as const;
                    },
                },
                recovery: {
                    strategy: "retry",
                    maxAttempts: 2,
                    fallbackBehavior: "reduce_throughput",
                    escalationTarget: "tier1",
                },
                validation: {
                    shouldRecover: true,
                    timeoutMs: 8000,
                    expectedFinalState: "reduced_throughput",
                    preserveProgress: false,
                },
            },
        ],
    },
);

// ================================================================================================
// Retry Logic Validation Utilities
// ================================================================================================

/**
 * Validates retry configuration for soundness
 */
export function validateRetryConfig(config: RetryConfig): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];

    // Basic validation
    if (config.maxRetries < 0) {
        errors.push("maxRetries must be non-negative");
    }
    if (config.baseDelayMs <= 0) {
        errors.push("baseDelayMs must be positive");
    }
    if (config.maxDelayMs <= 0) {
        errors.push("maxDelayMs must be positive");
    }
    if (config.backoffMultiplier <= 1.0) {
        errors.push("backoffMultiplier must be greater than 1.0");
    }

    // Logical consistency
    if (config.maxDelayMs < config.baseDelayMs) {
        errors.push("maxDelayMs must be >= baseDelayMs");
    }

    // Performance warnings
    const MAX_RETRIES_WARNING_THRESHOLD = 10;
    const BASE_DELAY_WARNING_THRESHOLD = 1000;
    if (config.maxRetries > MAX_RETRIES_WARNING_THRESHOLD) {
        warnings.push("Very high maxRetries may cause long delays");
    }
    if (config.baseDelayMs > BASE_DELAY_WARNING_THRESHOLD) {
        warnings.push("High baseDelayMs may impact user experience");
    }
    if (config.backoffMultiplier > 3.0) {
        warnings.push("High backoffMultiplier may cause very long delays");
    }

    // Jitter validation
    if (config.jitterRange !== undefined) {
        if (config.jitterRange < 0 || config.jitterRange > 1) {
            errors.push("jitterRange must be between 0 and 1");
        }
    }

    return {
        pass: errors.length === 0,
        message: errors.length === 0 ? "Retry config validation passed" : "Retry config validation failed",
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
    };
}

/**
 * Validates retry test scenario for completeness
 */
export function validateRetryTestScenario(scenario: RetryTestScenario): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];

    // Validate config
    const configResult = validateRetryConfig(scenario.config);
    if (!configResult.pass) {
        errors.push(`Config validation failed: ${configResult.message}`);
    }

    // Validate failure pattern
    if (scenario.failurePattern.some(attempt => attempt < 0)) {
        errors.push("Failure pattern cannot contain negative attempt numbers");
    }

    // Check logical consistency
    const maxPossibleAttempts = scenario.config.maxRetries + 1;
    if (scenario.expectedAttempts > maxPossibleAttempts) {
        errors.push(`Expected attempts (${scenario.expectedAttempts}) exceeds max possible (${maxPossibleAttempts})`);
    }

    // Validate success expectations
    const allAttemptsWillFail = scenario.failurePattern.length >= scenario.expectedAttempts;
    if (scenario.expectedSuccess && allAttemptsWillFail) {
        errors.push("Cannot expect success if all attempts will fail");
    }

    // Performance warnings
    const LONG_DELAY_WARNING_THRESHOLD = 30000;
    if (scenario.expectedTotalDelay > LONG_DELAY_WARNING_THRESHOLD) {
        warnings.push("Very long expected delay may impact test performance");
    }

    return {
        pass: errors.length === 0,
        message: errors.length === 0 ? "Scenario validation passed" : "Scenario validation failed",
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
    };
}

/**
 * Calculates expected delay for retry scenario (without jitter)
 */
export function calculateExpectedDelay(config: RetryConfig, retryCount: number): number {
    let totalDelay = 0;
    
    for (let i = 0; i < retryCount; i++) {
        const delay = Math.min(
            config.baseDelayMs * Math.pow(config.backoffMultiplier, i),
            config.maxDelayMs,
        );
        totalDelay += delay;
    }
    
    return totalDelay;
}

/**
 * Simulates retry execution for testing
 */
export function simulateRetryExecution(
    scenario: RetryTestScenario,
    includeJitter = false,
): BranchExecutionResult {
    const { config, failurePattern } = scenario;
    const retryDelays: number[] = [];
    let attempts = 0;
    let totalDelay = 0;
    
    while (attempts <= config.maxRetries) {
        const isFailure = failurePattern.includes(attempts);
        
        if (!isFailure) {
            // Success
            return {
                success: true,
                attempts: attempts + 1,
                totalDelay,
                retryDelays,
            };
        }
        
        attempts++;
        
        // Calculate delay for next attempt (if not last)
        if (attempts <= config.maxRetries) {
            let delay = Math.min(
                config.baseDelayMs * Math.pow(config.backoffMultiplier, attempts - 1),
                config.maxDelayMs,
            );
            
            // Add jitter if requested
            if (includeJitter && config.jitterRange) {
                const jitter = (Math.random() - 0.5) * 2 * config.jitterRange * delay;
                delay = Math.max(0, delay + jitter);
            }
            
            retryDelays.push(delay);
            totalDelay += delay;
        }
    }
    
    // All retries exhausted
    return {
        success: false,
        attempts,
        totalDelay,
        finalError: `Max retries (${config.maxRetries}) exhausted`,
        retryDelays,
    };
}

// ================================================================================================
// Comprehensive Test Suite for Retry Logic
// ================================================================================================

/**
 * Runs comprehensive tests for retry logic fixtures
 */
export function runRetryLogicTests(): void {
    describe("Branch Coordinator Retry Logic Fixtures", () => {
        // Test basic fixtures
        describe("basic retry fixtures", () => {
            runComprehensiveExecutionTests(
                retryTestRoutineFixture,
                "routine",
                "retry-test-routine",
            );
            
            runComprehensiveExecutionTests(
                networkResilienceRoutineFixture,
                "routine", 
                "network-resilience-routine",
            );
            
            runComprehensiveExecutionTests(
                highThroughputRoutineFixture,
                "routine",
                "high-throughput-routine",
            );
        });

        // Test retry configurations
        describe("retry configuration validation", () => {
            it("should validate basic retry config", () => {
                const result = validateRetryConfig(basicRetryConfig);
                expect(result.pass).toBe(true);
            });

            it("should validate aggressive retry config", () => {
                const result = validateRetryConfig(aggressiveRetryConfig);
                expect(result.pass).toBe(true);
            });

            it("should validate conservative retry config", () => {
                const result = validateRetryConfig(conservativeRetryConfig);
                expect(result.pass).toBe(true);
            });

            it("should reject invalid retry config", () => {
                const invalidConfig: RetryConfig = {
                    maxRetries: -1,
                    baseDelayMs: 0,
                    maxDelayMs: -100,
                    backoffMultiplier: 0.5,
                };
                const result = validateRetryConfig(invalidConfig);
                expect(result.pass).toBe(false);
                expect(result.errors?.length).toBeGreaterThan(0);
            });
        });

        // Test retry scenarios
        describe("retry scenario validation", () => {
            Object.entries(retryTestScenarios).forEach(([scenarioName, scenario]) => {
                it(`should validate ${scenarioName} scenario`, () => {
                    const result = validateRetryTestScenario(scenario);
                    if (result.errors) {
                        console.error(`Scenario ${scenarioName} validation errors:`, result.errors);
                    }
                    expect(result.pass).toBe(true);
                });
            });
        });

        // Test retry simulation
        describe("retry simulation", () => {
            Object.entries(retryTestScenarios).forEach(([scenarioName, scenario]) => {
                it(`should simulate ${scenarioName} correctly`, () => {
                    const result = simulateRetryExecution(scenario);
                    
                    expect(result.attempts).toBe(scenario.expectedAttempts);
                    expect(result.success).toBe(scenario.expectedSuccess);
                    
                    // Allow some tolerance for expected delay due to algorithm differences
                    const DELAY_TOLERANCE_FACTOR = 0.2;
                    const delayTolerance = scenario.expectedTotalDelay * DELAY_TOLERANCE_FACTOR;
                    expect(result.totalDelay).toBeGreaterThanOrEqual(scenario.expectedTotalDelay - delayTolerance);
                    expect(result.totalDelay).toBeLessThanOrEqual(scenario.expectedTotalDelay + delayTolerance);
                });
            });
        });

        // Test delay calculations
        describe("delay calculations", () => {
            it("should calculate exponential backoff correctly", () => {
                const config: RetryConfig = {
                    maxRetries: 3,
                    baseDelayMs: 100,
                    maxDelayMs: 5000,
                    backoffMultiplier: 2.0,
                };

                const EXPECTED_DELAY_0_RETRIES = 0;
                const EXPECTED_DELAY_1_RETRY = 100;
                const EXPECTED_DELAY_2_RETRIES = 300; // 100 + 200
                const EXPECTED_DELAY_3_RETRIES = 700; // 100 + 200 + 400
                expect(calculateExpectedDelay(config, 0)).toBe(EXPECTED_DELAY_0_RETRIES);
                expect(calculateExpectedDelay(config, 1)).toBe(EXPECTED_DELAY_1_RETRY);
                expect(calculateExpectedDelay(config, 2)).toBe(EXPECTED_DELAY_2_RETRIES);
                expect(calculateExpectedDelay(config, 3)).toBe(EXPECTED_DELAY_3_RETRIES);
            });

            it("should respect delay caps", () => {
                const config: RetryConfig = {
                    maxRetries: 5,
                    baseDelayMs: 1000,
                    maxDelayMs: 2000,
                    backoffMultiplier: 3.0,
                };

                // Without cap: 1000 + 3000 + 9000 = 13000
                // With cap: 1000 + 2000 + 2000 = 5000
                const EXPECTED_CAPPED_DELAY = 5000;
                expect(calculateExpectedDelay(config, 3)).toBe(EXPECTED_CAPPED_DELAY);
            });
        });

        // Test error scenario integration
        describe("error scenario integration", () => {
            it("should have consistent error scenarios", () => {
                const fixture = retryTestRoutineFixture;
                expect(fixture.errorScenarios).toBeDefined();
                expect(fixture.errorScenarios!.length).toBeGreaterThan(0);

                fixture.errorScenarios!.forEach((scenario) => {
                    expect(scenario.recovery.strategy).toBe("retry");
                    expect(scenario.validation.shouldRecover).toBe(true);
                    expect(scenario.expectedError.resilienceMetadata?.retryable).toBe(true);
                });
            });
        });
    });
}

// ================================================================================================
// Export Collections for Easy Testing
// ================================================================================================

export const retryLogicFixtures = {
    routines: {
        basic: retryTestRoutineFixture,
        networkResilience: networkResilienceRoutineFixture,
        highThroughput: highThroughputRoutineFixture,
    },
    configs: {
        basic: basicRetryConfig,
        aggressive: aggressiveRetryConfig,
        conservative: conservativeRetryConfig,
    },
    scenarios: retryTestScenarios,
    errorScenarios: executionErrorScenarios,
};

export default retryLogicFixtures;
