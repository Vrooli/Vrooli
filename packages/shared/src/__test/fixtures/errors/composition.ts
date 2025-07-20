/**
 * Error composition utilities for building complex error scenarios
 * 
 * This module provides utilities for chaining, cascading, and composing
 * errors to create realistic error scenarios for testing.
 */

import {
    type CascadeOptions,
    type ComposedError,
    type ErrorContext,
    type ErrorSequence,
    type RecoveryScenario,
    type RecoveryStep,
    type TimingOptions,
} from "./types.js";

/**
 * Error composer for building complex error scenarios
 */
export const errorComposer = {
    /**
     * Chain errors together with a primary error and its causes
     */
    chain<T>(primary: T, ...causes: any[]): ComposedError {
        return {
            primary,
            causes,
            composition: "chain",
            metadata: {
                created: new Date(),
                severity: this.determineSeverity(primary, causes),
                tags: ["chained", "composed"],
            },
        };
    },

    /**
     * Create multi-stage failure scenarios
     */
    cascade(errors: any[], options: CascadeOptions): ErrorSequence {
        return {
            errors,
            timing: options.timing,
            options: {
                failFast: options.failureMode === "fail-fast",
                continueOnError: options.failureMode === "continue",
                timeout: options.maxAttempts ? options.maxAttempts * (options.delay || 1000) : undefined,
            },
        };
    },

    /**
     * Build context-aware errors
     */
    withContext<T>(error: T, context: ErrorContext): T {
        const contextualError = { ...error } as any;
        contextualError.context = context;
        contextualError.telemetry = {
            traceId: this.generateTraceId(),
            spanId: this.generateSpanId(),
            tags: {
                operation: context.operation || "unknown",
                resource: context.resource?.type || "unknown",
                user_role: context.user?.role || "anonymous",
            },
            level: this.mapSeverityToLevel((error as any).severity),
        };
        return contextualError;
    },

    /**
     * Create time-based error scenarios
     */
    withTiming<T>(error: T, timing: TimingOptions): T {
        const timedError = {
            ...error,
            composition: "timed",
            timing: {
                delay: timing.delay,
                duration: timing.duration,
                pattern: timing.pattern || "immediate",
                schedule: timing.schedule,
            },
        } as any;

        return timedError;
    },

    /**
     * Determine severity level based on error composition
     */
    determineSeverity(primary: any, causes: any[]): "low" | "medium" | "high" | "critical" {
        const primarySeverity = primary.severity || primary.userImpact;
        const causesSeverity = causes.map(c => c.severity || c.userImpact);

        if (primarySeverity === "critical" || causesSeverity.includes("critical")) {
            return "critical";
        }
        if (primarySeverity === "blocking" || causesSeverity.includes("blocking")) {
            return "high";
        }
        if (primarySeverity === "degraded" || causesSeverity.includes("degraded")) {
            return "medium";
        }
        return "low";
    },

    /**
     * Generate unique trace ID for telemetry
     */
    generateTraceId(): string {
        return Math.random().toString(36).substring(2, 15) + Math.random().toString(36).substring(2, 15);
    },

    /**
     * Generate unique span ID for telemetry
     */
    generateSpanId(): string {
        return Math.random().toString(36).substring(2, 10);
    },

    /**
     * Map severity to telemetry level
     */
    mapSeverityToLevel(severity: string): "debug" | "info" | "warn" | "error" | "fatal" {
        switch (severity) {
            case "critical": return "fatal";
            case "error":
            case "blocking": return "error";
            case "warning":
            case "degraded": return "warn";
            case "info": return "info";
            default: return "debug";
        }
    },
};

/**
 * Recovery scenario builder for comprehensive testing
 */
export const recoveryScenarioBuilder = {
    /**
     * Create a basic retry scenario
     */
    retryScenario<T>(
        error: T,
        maxAttempts = 3,
        baseDelay = 1000,
    ): RecoveryScenario<T> {
        const steps: RecoveryStep[] = [];

        for (let i = 1; i <= maxAttempts; i++) {
            steps.push({
                action: "retry",
                delay: baseDelay * Math.pow(2, i - 1), // Exponential backoff
                condition: (error, attempt) => attempt < maxAttempts,
                description: `Retry attempt ${i} of ${maxAttempts}`,
            });
        }

        // Final fallback
        steps.push({
            action: "fallback",
            description: "Use fallback strategy after max retries",
        });

        return {
            error,
            recoverySteps: steps,
            expectedOutcome: "success",
            fallbackBehavior: "graceful-degradation",
            description: `Retry up to ${maxAttempts} times with exponential backoff`,
        };
    },

    /**
     * Create a circuit breaker scenario
     */
    circuitBreakerScenario<T>(
        error: T,
        failureThreshold = 5,
        timeout = 30000,
    ): RecoveryScenario<T> {
        return {
            error,
            recoverySteps: [
                {
                    action: "retry",
                    condition: (error, attempt) => attempt < failureThreshold,
                    description: `Allow retries until ${failureThreshold} failures`,
                },
                {
                    action: "circuit-break",
                    delay: timeout,
                    description: `Open circuit for ${timeout}ms after threshold reached`,
                },
                {
                    action: "retry",
                    condition: () => true,
                    description: "Half-open circuit - allow single test request",
                },
            ],
            expectedOutcome: "partial",
            fallbackBehavior: "circuit-open",
            description: "Circuit breaker pattern with configurable threshold",
        };
    },

    /**
     * Create a fallback scenario
     */
    fallbackScenario<T>(
        error: T,
        fallbackAction: string,
        retryAfterFallback = true,
    ): RecoveryScenario<T> {
        const steps: RecoveryStep[] = [
            {
                action: "fallback",
                description: `Use fallback: ${fallbackAction}`,
            },
        ];

        if (retryAfterFallback) {
            steps.push({
                action: "retry",
                delay: 10000,
                description: "Retry original operation after fallback",
            });
        }

        return {
            error,
            recoverySteps: steps,
            expectedOutcome: "success",
            fallbackBehavior: fallbackAction,
            description: `Immediate fallback to ${fallbackAction}`,
        };
    },

    /**
     * Create a complex multi-stage recovery scenario
     */
    multiStageScenario<T>(
        error: T,
        stages: Array<{
            action: RecoveryStep["action"];
            delay?: number;
            fallback?: string;
            description: string;
        }>,
    ): RecoveryScenario<T> {
        const steps: RecoveryStep[] = stages.map(stage => {
            const step: RecoveryStep = {
                action: stage.action,
                delay: stage.delay,
                description: stage.description,
            };

            if (stage.fallback) {
                step.transform = (e) => {
                    const base = typeof e === 'object' && e !== null ? e : {};
                    return { ...base, fallback: stage.fallback };
                };
            }

            return step;
        });

        return {
            error,
            recoverySteps: steps,
            expectedOutcome: "success",
            fallbackBehavior: "multi-stage-recovery",
            description: "Complex multi-stage recovery process",
        };
    },
};

/**
 * Error scenario factory for common patterns
 */
export const errorScenarios = {
    /**
     * Database connection failure with retry and fallback
     */
    databaseFailure: {
        description: "Database connection lost with automatic recovery",
        compose: (databaseError: any, networkError: any) => errorComposer.chain(
            databaseError,
            networkError,
        ),
        recovery: (error: any) => recoveryScenarioBuilder.retryScenario(error, 3, 2000),
    },

    /**
     * API rate limiting with backoff
     */
    rateLimitExceeded: {
        description: "API rate limit exceeded with exponential backoff",
        compose: (rateLimitError: any) => rateLimitError,
        recovery: (error: any) => recoveryScenarioBuilder.retryScenario(error, 5, 1000),
    },

    /**
     * Service temporarily down with circuit breaker
     */
    serviceUnavailable: {
        description: "External service unavailable with circuit breaker pattern",
        compose: (serviceError: any, networkError: any) => errorComposer.cascade(
            [serviceError, networkError],
            { timing: "sequential", failureMode: "fail-fast" },
        ),
        recovery: (error: any) => recoveryScenarioBuilder.circuitBreakerScenario(error, 3, 60000),
    },

    /**
     * Authentication failure with lockout
     */
    authenticationFailure: {
        description: "Multiple authentication failures leading to account lockout",
        compose: (authError: any, lockoutError: any) => errorComposer.cascade(
            [authError, authError, authError, lockoutError],
            { timing: "sequential", failureMode: "continue" },
        ),
        recovery: (error: any) => recoveryScenarioBuilder.fallbackScenario(
            error,
            "account-recovery-flow",
            false,
        ),
    },

    /**
     * Validation errors with field-specific recovery
     */
    validationFailure: {
        description: "Form validation failures with field-specific guidance",
        compose: (validationError: any) => validationError,
        recovery: (error: any) => ({
            error,
            recoverySteps: [{
                action: "fallback" as const,
                description: "Show field-specific validation guidance",
            }],
            expectedOutcome: "success" as const,
            fallbackBehavior: "inline-validation-help",
            description: "Field-by-field validation guidance",
        }),
    },

    /**
     * Network timeout with progressive fallback
     */
    networkTimeout: {
        description: "Network timeout with progressive quality degradation",
        compose: (timeoutError: any, qualityError: any) => errorComposer.chain(
            timeoutError,
            qualityError,
        ),
        recovery: (error: any) => recoveryScenarioBuilder.multiStageScenario(error, [
            { action: "retry", delay: 2000, description: "Quick retry" },
            { action: "retry", delay: 5000, description: "Slower retry" },
            { action: "fallback", fallback: "offline-mode", description: "Switch to offline mode" },
        ]),
    },
};

/**
 * Utilities for testing error scenarios
 */
export const errorTestUtils = {
    /**
     * Execute a recovery scenario and return test results
     */
    async executeRecoveryScenario<T>(
        scenario: RecoveryScenario<T>,
        mockOperation: (attempt: number) => Promise<any>,
    ): Promise<{
        success: boolean;
        attempts: number;
        finalError?: any;
        executionTime: number;
        stepResults: Array<{ step: RecoveryStep; success: boolean; duration: number }>;
    }> {
        const startTime = Date.now();
        const stepResults: Array<{ step: RecoveryStep; success: boolean; duration: number }> = [];
        let attempts = 0;
        let lastError: any;

        for (const step of scenario.recoverySteps) {
            const stepStart = Date.now();
            attempts++;

            try {
                if (step.delay) {
                    await new Promise(resolve => setTimeout(resolve, step.delay));
                }

                if (step.condition && !step.condition(lastError, attempts)) {
                    stepResults.push({
                        step,
                        success: false,
                        duration: Date.now() - stepStart,
                    });
                    continue;
                }

                const result = await mockOperation(attempts);

                stepResults.push({
                    step,
                    success: true,
                    duration: Date.now() - stepStart,
                });

                // If we got here, the operation succeeded
                return {
                    success: true,
                    attempts,
                    executionTime: Date.now() - startTime,
                    stepResults,
                };

            } catch (error) {
                lastError = step.transform ? step.transform(error) : error;
                stepResults.push({
                    step,
                    success: false,
                    duration: Date.now() - stepStart,
                });
            }
        }

        return {
            success: false,
            attempts,
            finalError: lastError,
            executionTime: Date.now() - startTime,
            stepResults,
        };
    },

    /**
     * Validate error composition structure
     */
    validateComposition(composed: ComposedError): {
        valid: boolean;
        issues: string[];
    } {
        const issues: string[] = [];

        if (!composed.primary) {
            issues.push("Missing primary error");
        }

        if (!Array.isArray(composed.causes)) {
            issues.push("Causes must be an array");
        }

        if (!composed.metadata?.created) {
            issues.push("Missing creation timestamp");
        }

        if (!["low", "medium", "high", "critical"].includes(composed.metadata?.severity)) {
            issues.push("Invalid severity level");
        }

        return {
            valid: issues.length === 0,
            issues,
        };
    },
};
