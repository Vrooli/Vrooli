import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { type Logger } from "winston";
import {
    ErrorSeverity,
    ErrorCategory,
    ErrorRecoverability,
    RecoveryStrategy,
    BackoffType,
    generatePK,
} from "@vrooli/shared";
import { SimpleRecoveryProvider } from "./simpleRecoveryProvider.js";
import { type EventBus } from "../events/eventBus.js";

describe("SimpleRecoveryProvider", () => {
    let provider: SimpleRecoveryProvider;
    let logger: Logger;
    let eventBus: EventBus;

    beforeEach(() => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;

        eventBus = {
            publish: vi.fn(),
            subscribe: vi.fn(),
            unsubscribe: vi.fn(),
            on: vi.fn(),
            off: vi.fn(),
        } as unknown as EventBus;

        provider = new SimpleRecoveryProvider(eventBus, logger);

        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("Basic Strategy Lookup", () => {
        it("should select strategy based on error classification", async () => {
            const classification = {
                severity: ErrorSeverity.WARNING,
                category: ErrorCategory.TRANSIENT,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "llm-client",
                operation: "chat",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 1000,
                    errorSeverity: ErrorSeverity.WARNING,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            expect(strategy.strategyType).toBe(RecoveryStrategy.RETRY_SAME);
            expect(strategy.maxAttempts).toBeGreaterThan(0);
            expect(strategy.backoffStrategy).toBeDefined();
        });

        it("should emit strategy selection event for agent analysis", async () => {
            const classification = {
                severity: ErrorSeverity.ERROR,
                category: ErrorCategory.RESOURCE,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 2,
                component: "resource-manager",
                operation: "allocate",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 500,
                    errorSeverity: ErrorSeverity.ERROR,
                },
            };

            await provider.getStrategy(classification, context);

            expect(eventBus.publish).toHaveBeenCalledWith("recovery/strategy_selected", expect.objectContaining({
                classification: {
                    severity: ErrorSeverity.ERROR,
                    category: ErrorCategory.RESOURCE,
                    recoverability: ErrorRecoverability.AUTOMATIC,
                },
                context: {
                    tier: 2,
                    component: "resource-manager",
                    attemptCount: 1,
                },
                selectedStrategy: expect.any(String),
                strategyKey: "ERROR:RESOURCE",
                selectionReason: "basic_lookup",
                timestamp: expect.any(Date),
            }));
        });
    });

    describe("Critical Error Handling", () => {
        it("should select emergency stop for fatal errors", async () => {
            const classification = {
                severity: ErrorSeverity.FATAL,
                category: ErrorCategory.SECURITY,
                recoverability: ErrorRecoverability.NONE,
                securityRisk: true,
                dataRisk: true,
                multipleComponentsAffected: true,
            };

            const context = {
                tier: 3,
                component: "security-validator",
                operation: "validate",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 100,
                    errorSeverity: ErrorSeverity.FATAL,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            expect(strategy.strategyType).toBe(RecoveryStrategy.EMERGENCY_STOP);
            expect(strategy.maxAttempts).toBe(1);
            expect(strategy.timeoutMs).toBeLessThanOrEqual(1000);
        });

        it("should escalate to human for security risks", async () => {
            const classification = {
                severity: ErrorSeverity.CRITICAL,
                category: ErrorCategory.SECURITY,
                recoverability: ErrorRecoverability.MANUAL,
                securityRisk: true,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 1,
                component: "auth-service",
                operation: "authenticate",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 200,
                    errorSeverity: ErrorSeverity.CRITICAL,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            expect(strategy.strategyType).toBe(RecoveryStrategy.ESCALATE_TO_HUMAN);
            expect(strategy.resourceRequirements).toHaveProperty("human_attention");
        });
    });

    describe("Transient Error Recovery", () => {
        it("should use retry strategy for transient errors", async () => {
            const classification = {
                severity: ErrorSeverity.WARNING,
                category: ErrorCategory.TRANSIENT,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "network-client",
                operation: "request",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 1500,
                    errorSeverity: ErrorSeverity.WARNING,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            expect(strategy.strategyType).toBe(RecoveryStrategy.RETRY_SAME);
            expect(strategy.backoffStrategy.type).toBe(BackoffType.EXPONENTIAL);
            expect(strategy.maxAttempts).toBeGreaterThan(1);
        });

        it("should use wait and retry for resource errors", async () => {
            const classification = {
                severity: ErrorSeverity.WARNING,
                category: ErrorCategory.RESOURCE,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 2,
                component: "resource-pool",
                operation: "acquire",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 2000,
                    errorSeverity: ErrorSeverity.WARNING,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            expect(strategy.strategyType).toBe(RecoveryStrategy.WAIT_AND_RETRY);
            expect(strategy.backoffStrategy.type).toBe(BackoffType.EXPONENTIAL_JITTER);
            expect(strategy.backoffStrategy.initialDelayMs).toBeGreaterThan(500);
        });
    });

    describe("Strategy Adaptation", () => {
        it("should adapt strategy for critical errors", async () => {
            const classification = {
                severity: ErrorSeverity.CRITICAL,
                category: ErrorCategory.LOGIC,
                recoverability: ErrorRecoverability.PARTIAL,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "logic-processor",
                operation: "process",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 500,
                    errorSeverity: ErrorSeverity.CRITICAL,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            // Critical errors should have reduced max attempts
            expect(strategy.maxAttempts).toBeLessThanOrEqual(2);
        });

        it("should reduce attempts for repeated failures", async () => {
            const classification = {
                severity: ErrorSeverity.ERROR,
                category: ErrorCategory.TRANSIENT,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "retry-test",
                operation: "test",
                attemptCount: 3, // Multiple previous attempts
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 1000,
                    errorSeverity: ErrorSeverity.ERROR,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            // Should reduce max attempts based on previous attempt count
            expect(strategy.maxAttempts).toBeLessThan(3);
        });
    });

    describe("Default Strategy Selection", () => {
        it("should use emergency strategy for unmatched fatal errors", async () => {
            const classification = {
                severity: ErrorSeverity.FATAL,
                category: ErrorCategory.UNKNOWN as any,
                recoverability: ErrorRecoverability.NONE,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "unknown",
                operation: "unknown",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 0,
                    errorSeverity: ErrorSeverity.FATAL,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            expect(strategy.strategyType).toBe(RecoveryStrategy.EMERGENCY_STOP);
        });

        it("should escalate security risks to human", async () => {
            const classification = {
                severity: ErrorSeverity.WARNING,
                category: ErrorCategory.UNKNOWN as any,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: true, // Security risk flag set
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 2,
                component: "security-check",
                operation: "validate",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 100,
                    errorSeverity: ErrorSeverity.WARNING,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            expect(strategy.strategyType).toBe(RecoveryStrategy.ESCALATE_TO_HUMAN);
        });

        it("should use retry as general fallback", async () => {
            const classification = {
                severity: ErrorSeverity.WARNING,
                category: ErrorCategory.UNKNOWN as any,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "general-service",
                operation: "process",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 500,
                    errorSeverity: ErrorSeverity.WARNING,
                },
            };

            const strategy = await provider.getStrategy(classification, context);

            expect(strategy.strategyType).toBe(RecoveryStrategy.RETRY_SAME);
        });
    });

    describe("Outcome Recording", () => {
        it("should record successful recovery outcomes", async () => {
            const classification = {
                severity: ErrorSeverity.WARNING,
                category: ErrorCategory.TRANSIENT,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "test-service",
                operation: "test",
                attemptCount: 2,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 1000,
                    errorSeverity: ErrorSeverity.WARNING,
                },
            };

            await provider.recordOutcome(
                RecoveryStrategy.RETRY_SAME,
                classification,
                context,
                true, // success
                1500, // duration
                0.02, // resourceCost
            );

            expect(eventBus.publish).toHaveBeenCalledWith("recovery/outcome", expect.objectContaining({
                strategyType: RecoveryStrategy.RETRY_SAME,
                classification: {
                    severity: ErrorSeverity.WARNING,
                    category: ErrorCategory.TRANSIENT,
                    recoverability: ErrorRecoverability.AUTOMATIC,
                },
                context: {
                    tier: 3,
                    component: "test-service",
                    attemptCount: 2,
                },
                success: true,
                duration: 1500,
                resourceCost: 0.02,
                timestamp: expect.any(Date),
            }));
        });

        it("should record failed recovery outcomes", async () => {
            const classification = {
                severity: ErrorSeverity.ERROR,
                category: ErrorCategory.LOGIC,
                recoverability: ErrorRecoverability.PARTIAL,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 2,
                component: "logic-service",
                operation: "validate",
                attemptCount: 3,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 2000,
                    errorSeverity: ErrorSeverity.ERROR,
                },
            };

            await provider.recordOutcome(
                RecoveryStrategy.RETRY_MODIFIED,
                classification,
                context,
                false, // failed
                3000, // duration
                0.05, // resourceCost
            );

            expect(eventBus.publish).toHaveBeenCalledWith("recovery/outcome", expect.objectContaining({
                success: false,
                duration: 3000,
                resourceCost: 0.05,
            }));
        });
    });

    describe("Strategy Configuration", () => {
        it("should provide consistent strategy configurations", async () => {
            const classification = {
                severity: ErrorSeverity.WARNING,
                category: ErrorCategory.TRANSIENT,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "test",
                operation: "test",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 1000,
                    errorSeverity: ErrorSeverity.WARNING,
                },
            };

            const strategy1 = await provider.getStrategy(classification, context);
            const strategy2 = await provider.getStrategy(classification, context);

            // Same classification should yield same strategy type
            expect(strategy1.strategyType).toBe(strategy2.strategyType);
            expect(strategy1.backoffStrategy.type).toBe(strategy2.backoffStrategy.type);
        });

        it("should provide different strategies for different classifications", async () => {
            const transientClassification = {
                severity: ErrorSeverity.WARNING,
                category: ErrorCategory.TRANSIENT,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const resourceClassification = {
                severity: ErrorSeverity.ERROR,
                category: ErrorCategory.RESOURCE,
                recoverability: ErrorRecoverability.AUTOMATIC,
                securityRisk: false,
                dataRisk: false,
                multipleComponentsAffected: false,
            };

            const context = {
                tier: 3,
                component: "test",
                operation: "test",
                attemptCount: 1,
                previousStrategies: [],
                resourceState: {},
                performanceMetrics: {
                    averageResponseTime: 1000,
                    errorSeverity: ErrorSeverity.WARNING,
                },
            };

            const strategy1 = await provider.getStrategy(transientClassification, context);
            const strategy2 = await provider.getStrategy(resourceClassification, context);

            // Different classifications should yield different strategies
            expect(strategy1.strategyType).not.toBe(strategy2.strategyType);
        });
    });
});
