/**
 * Resilience Infrastructure Integration Tests
 * Tests the complete error classification and recovery flow
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import type {
    ErrorContext,
    ResilienceEventSource,
    ErrorSeverity,
    ErrorCategory,
    RecoveryStrategy,
} from "@vrooli/shared";
import {
    ErrorSeverity as Severity,
    ErrorCategory as Category,
    RecoveryStrategy as Strategy,
} from "@vrooli/shared";
import { ErrorClassifier } from "../errorClassifier.js";
import { RecoverySelector } from "../recoverySelector.js";
import { ResilienceInfrastructure } from "../index.js";

// Mock telemetry shim
const mockTelemetryShim = {
    emitError: vi.fn().mockResolvedValue(undefined),
    emitTaskCompletion: vi.fn().mockResolvedValue(undefined),
    emitComponentHealth: vi.fn().mockResolvedValue(undefined),
};

// Mock event bus
const mockEventBus = {
    publishBatch: vi.fn().mockResolvedValue(undefined),
    publish: vi.fn().mockResolvedValue(undefined),
};

describe("Resilience Infrastructure Integration", () => {
    let resilience: ResilienceInfrastructure;
    let classifier: ErrorClassifier;
    let selector: RecoverySelector;

    beforeEach(() => {
        vi.clearAllMocks();
        resilience = new ResilienceInfrastructure(mockTelemetryShim, mockEventBus);
        classifier = new ErrorClassifier();
        selector = new RecoverySelector();
    });

    afterEach(async () => {
        await resilience.shutdown();
    });

    describe("Error Classification", () => {
        it("should classify network timeout as transient error", async () => {
            const error = new Error("Connection timeout after 5000ms");
            const context: ErrorContext = {
                tier: 3,
                component: "api-client",
                operation: "fetchData",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "test-123" },
            };

            const classification = await classifier.classify(error, context);

            expect(classification.category).toBe(Category.TRANSIENT);
            expect(classification.severity).toBe(Severity.WARNING);
            expect(classification.recoverability).toBe("AUTOMATIC");
            expect(classification.systemFunctional).toBe(true);
            expect(classification.confidenceScore).toBeGreaterThan(0.8);
        });

        it("should classify database connection failure as system error", async () => {
            const error = new Error("Database connection failed: Connection refused");
            const context: ErrorContext = {
                tier: 2,
                component: "database-manager",
                operation: "connect",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "test-456" },
            };

            const classification = await classifier.classify(error, context);

            expect(classification.category).toBe(Category.SYSTEM);
            expect(classification.severity).toBe(Severity.CRITICAL);
            expect(classification.systemFunctional).toBe(false);
            expect(classification.multipleComponentsAffected).toBe(false);
        });

        it("should classify authentication error as security risk", async () => {
            const error = new Error("Invalid authentication token");
            const context: ErrorContext = {
                tier: 1,
                component: "auth-service",
                operation: "validateToken",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "test-789" },
            };

            const classification = await classifier.classify(error, context);

            expect(classification.category).toBe(Category.SECURITY);
            expect(classification.securityRisk).toBe(true);
            expect(classification.severity).toBe(Severity.CRITICAL);
        });

        it("should escalate severity after multiple attempts", async () => {
            const error = new Error("Temporary service unavailable");
            const context: ErrorContext = {
                tier: 3,
                component: "external-api",
                operation: "callService",
                attemptCount: 4, // Multiple attempts
                previousStrategies: [Strategy.RETRY_SAME, Strategy.WAIT_AND_RETRY],
                systemState: {},
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "test-multi" },
            };

            const classification = await classifier.classify(error, context);

            expect(classification.severity).toBe(Severity.ERROR);
            expect(classification.confidenceScore).toBeLessThan(0.9); // Lower confidence after multiple attempts
        });
    });

    describe("Recovery Strategy Selection", () => {
        it("should select retry strategy for transient errors", async () => {
            const classification = {
                severity: Severity.WARNING,
                category: Category.TRANSIENT,
                recoverability: "AUTOMATIC" as any,
                systemFunctional: true,
                multipleComponentsAffected: false,
                dataRisk: false,
                securityRisk: false,
                confidenceScore: 0.9,
                timestamp: new Date(),
                metadata: {},
            };

            const context: ErrorContext = {
                tier: 3,
                component: "api-client",
                operation: "fetchData",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: { memoryUsage: 0.3, cpuUsage: 0.4 },
                performanceMetrics: {},
                userContext: { requestId: "test-retry" },
            };

            const strategy = await selector.selectStrategy(classification, context);

            expect(strategy.strategyType).toBe(Strategy.RETRY_SAME);
            expect(strategy.maxAttempts).toBeGreaterThan(1);
            expect(strategy.estimatedSuccessRate).toBeGreaterThan(0.5);
        });

        it("should select escalation for security risks", async () => {
            const classification = {
                severity: Severity.CRITICAL,
                category: Category.SECURITY,
                recoverability: "MANUAL" as any,
                systemFunctional: true,
                multipleComponentsAffected: false,
                dataRisk: false,
                securityRisk: true,
                confidenceScore: 0.95,
                timestamp: new Date(),
                metadata: {},
            };

            const context: ErrorContext = {
                tier: 2,
                component: "auth-service",
                operation: "authenticate",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "test-security" },
            };

            const strategy = await selector.selectStrategy(classification, context);

            expect(strategy.strategyType).toBe(Strategy.ESCALATE_TO_HUMAN);
            expect(strategy.maxAttempts).toBe(1);
        });

        it("should select emergency stop for fatal errors", async () => {
            const classification = {
                severity: Severity.FATAL,
                category: Category.SYSTEM,
                recoverability: "NONE" as any,
                systemFunctional: false,
                multipleComponentsAffected: true,
                dataRisk: false,
                securityRisk: false,
                confidenceScore: 1.0,
                timestamp: new Date(),
                metadata: {},
            };

            const context: ErrorContext = {
                tier: 1,
                component: "system-core",
                operation: "startup",
                attemptCount: 1,
                previousStrategies: [],
                systemState: { tierFailures: 2 },
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "test-fatal" },
            };

            const strategy = await selector.selectStrategy(classification, context);

            expect(strategy.strategyType).toBe(Strategy.EMERGENCY_STOP);
        });

        it("should adapt strategy based on resource constraints", async () => {
            const classification = {
                severity: Severity.ERROR,
                category: Category.RESOURCE,
                recoverability: "AUTOMATIC" as any,
                systemFunctional: true,
                multipleComponentsAffected: false,
                dataRisk: false,
                securityRisk: false,
                confidenceScore: 0.8,
                timestamp: new Date(),
                metadata: {},
            };

            const context: ErrorContext = {
                tier: 3,
                component: "resource-manager",
                operation: "allocateMemory",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: { memoryUsage: 0.9, cpuUsage: 0.8 }, // High resource usage
                performanceMetrics: {},
                userContext: { requestId: "test-resource" },
            };

            const strategy = await selector.selectStrategy(classification, context);

            expect([Strategy.REDUCE_SCOPE, Strategy.WAIT_AND_RETRY]).toContain(strategy.strategyType);
        });
    });

    describe("Complete Resilience Flow", () => {
        it("should handle complete error-to-recovery flow", async () => {
            const error = new Error("Network timeout during API call");
            const context: ErrorContext = {
                tier: 3,
                component: "external-api",
                operation: "fetchUserData",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: {},
                performanceMetrics: { averageResponseTime: 1000 },
                userContext: { requestId: "flow-test-123" },
            };

            const source: ResilienceEventSource = {
                tier: 3,
                component: "external-api",
                operation: "fetchUserData",
                requestId: "flow-test-123",
            };

            // Step 1: Handle initial error
            const { classification, strategy } = await resilience.handleError(error, context, source);

            expect(classification).toBeDefined();
            expect(strategy).toBeDefined();
            expect(classification.category).toBe(Category.TRANSIENT);
            expect(strategy.strategyType).toBe(Strategy.RETRY_SAME);

            // Step 2: Simulate successful recovery
            const successfulOutcome = {
                success: true,
                duration: 2500,
                attemptCount: 2,
                strategiesUsed: [Strategy.RETRY_SAME],
                finalStrategy: Strategy.RETRY_SAME,
                qualityImpact: 0.1,
                resourceUsage: { retries: 1, time: 2500 },
                userImpact: "MINIMAL" as any,
                lessons: ["Retry successful after network recovery"],
            };

            await resilience.recordRecoveryOutcome(
                classification,
                context,
                strategy,
                successfulOutcome,
                source,
            );

            // Verify events were published
            expect(mockEventBus.publishBatch).toHaveBeenCalled();
            expect(mockTelemetryShim.emitError).toHaveBeenCalled();
            expect(mockTelemetryShim.emitTaskCompletion).toHaveBeenCalled();
        });

        it("should record strategy effectiveness for learning", async () => {
            const classification = {
                severity: Severity.WARNING,
                category: Category.TRANSIENT,
                recoverability: "AUTOMATIC" as any,
                systemFunctional: true,
                multipleComponentsAffected: false,
                dataRisk: false,
                securityRisk: false,
                confidenceScore: 0.9,
                timestamp: new Date(),
                metadata: {},
            };

            const context: ErrorContext = {
                tier: 3,
                component: "test-component",
                operation: "test-operation",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "learning-test" },
            };

            // Record multiple outcomes to test learning
            const outcomes = [
                { success: true, duration: 1000, resourceCost: 10 },
                { success: true, duration: 1200, resourceCost: 12 },
                { success: false, duration: 5000, resourceCost: 50 },
                { success: true, duration: 800, resourceCost: 8 },
            ];

            for (const outcome of outcomes) {
                selector.recordOutcome(
                    Strategy.RETRY_SAME,
                    classification,
                    context,
                    outcome.success,
                    outcome.duration,
                    outcome.resourceCost,
                );
            }

            const stats = selector.getEffectivenessStatistics();
            expect(stats.totalOutcomes).toBe(4);
            expect(stats.averageSuccessRate).toBe(0.75); // 3 out of 4 successful
        });
    });

    describe("Pattern Learning", () => {
        it("should improve classification with learned patterns", async () => {
            const pattern = {
                id: "test-timeout-pattern",
                name: "Test Timeout Pattern",
                description: "Recurring timeout pattern for testing",
                frequency: 10,
                severity: Severity.WARNING,
                category: Category.TRANSIENT,
                triggerConditions: [
                    {
                        field: "errorMessage",
                        operator: "CONTAINS" as any,
                        value: "timeout",
                        weight: 0.8,
                    },
                    {
                        field: "tier",
                        operator: "EQUALS" as any,
                        value: 3,
                        weight: 0.6,
                    },
                ],
                commonContext: {},
                effectiveStrategies: [],
                successRate: 0.85,
                averageResolutionTime: 3000,
                lastSeen: new Date(),
                confidence: 0.9,
            };

            resilience.addErrorPattern(pattern);

            const error = new Error("Connection timeout occurred");
            const context: ErrorContext = {
                tier: 3,
                component: "test-api",
                operation: "call",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "pattern-test" },
            };

            const classification = await classifier.classify(error, context);

            // Classification should be enhanced by pattern matching
            expect(classification.metadata.matchingPatterns).toBeDefined();
            expect(classification.confidenceScore).toBeGreaterThan(0.8);
        });
    });

    describe("Performance Characteristics", () => {
        it("should maintain low overhead during high-frequency errors", async () => {
            const error = new Error("High frequency test error");
            const context: ErrorContext = {
                tier: 3,
                component: "performance-test",
                operation: "highFrequency",
                attemptCount: 1,
                previousStrategies: [],
                systemState: {},
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: "perf-test" },
            };

            const startTime = performance.now();

            // Simulate high frequency of errors
            const promises = [];
            for (let i = 0; i < 100; i++) {
                promises.push(classifier.classify(error, {
                    ...context,
                    userContext: { requestId: `perf-test-${i}` },
                }));
            }

            await Promise.all(promises);

            const duration = performance.now() - startTime;
            const averagePerError = duration / 100;

            // Should maintain reasonable performance even under load
            expect(averagePerError).toBeLessThan(50); // Less than 50ms per classification
        });

        it("should maintain event publishing performance guarantees", async () => {
            const stats = resilience.getStatistics();
            
            // Publishing overhead should stay within limits
            expect(stats.publishing.averageOverheadMs).toBeLessThan(5);
            
            // Should track published events
            expect(stats.publishing.publishedEvents).toBeGreaterThanOrEqual(0);
            expect(stats.publishing.droppedEvents).toBe(0);
        });
    });
});

describe("Error Feature Extraction", () => {
    let classifier: ErrorClassifier;

    beforeEach(() => {
        classifier = new ErrorClassifier();
    });

    it("should detect network errors correctly", async () => {
        const networkErrors = [
            new Error("Connection timeout"),
            new Error("Network unreachable"),
            new Error("Socket error occurred"),
        ];

        const context: ErrorContext = {
            tier: 3,
            component: "network-test",
            operation: "connect",
            attemptCount: 1,
            previousStrategies: [],
            systemState: {},
            resourceState: {},
            performanceMetrics: {},
            userContext: { requestId: "network-test" },
        };

        for (const error of networkErrors) {
            const classification = await classifier.classify(error, context);
            expect(classification.category).toBe(Category.TRANSIENT);
        }
    });

    it("should detect database errors correctly", async () => {
        const dbErrors = [
            new Error("Prisma connection failed"),
            new Error("PostgreSQL: connection refused"),
            new Error("Database timeout"),
        ];

        const context: ErrorContext = {
            tier: 2,
            component: "database",
            operation: "query",
            attemptCount: 1,
            previousStrategies: [],
            systemState: {},
            resourceState: {},
            performanceMetrics: {},
            userContext: { requestId: "db-test" },
        };

        for (const error of dbErrors) {
            const classification = await classifier.classify(error, context);
            expect(classification.category).toBe(Category.SYSTEM);
        }
    });

    it("should detect validation errors correctly", async () => {
        const validationErrors = [
            new Error("Validation failed: invalid input"),
            new Error("Schema validation error"),
            new Error("Invalid parameter provided"),
        ];

        const context: ErrorContext = {
            tier: 3,
            component: "validator",
            operation: "validate",
            attemptCount: 1,
            previousStrategies: [],
            systemState: {},
            resourceState: {},
            performanceMetrics: {},
            userContext: { requestId: "validation-test" },
        };

        for (const error of validationErrors) {
            const classification = await classifier.classify(error, context);
            expect(classification.category).toBe(Category.LOGIC);
        }
    });
});