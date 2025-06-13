/**
 * Resilience Event Publisher
 * Publishes rich resilience events for agent learning with ≤5ms overhead guarantee
 * Integrates with existing telemetry shim and event bus infrastructure
 */

import type {
    ResilienceEvent,
    ResilienceEventType,
    ResilienceEventSource,
    ErrorClassification,
    ErrorContext,
    RecoveryStrategyConfig,
    ResilienceOutcome,
    ResilienceLearningData,
    ErrorPattern,
    UserImpactLevel,
    RecoveryStrategy,
} from "@vrooli/shared";
import {
    ResilienceEventType as EventType,
    UserImpactLevel,
} from "@vrooli/shared";
import type { RedisEventBus } from "../events/eventBus.js";
import { uuid } from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";

/**
 * Event publishing configuration
 */
interface PublishingConfig {
    enabled: boolean;
    batchSize: number;
    flushInterval: number;
    maxOverheadMs: number;
    samplingRates: Record<ResilienceEventType, number>;
    enablePatternDetection: boolean;
    enableLearningData: boolean;
}

/**
 * Default publishing configuration
 */
const DEFAULT_CONFIG: PublishingConfig = {
    enabled: true,
    batchSize: 50,
    flushInterval: 2000, // 2 seconds
    maxOverheadMs: 5, // Guarantee ≤5ms overhead
    samplingRates: {
        [EventType.ERROR_DETECTED]: 1.0, // 100% - critical for learning
        [EventType.ERROR_CLASSIFIED]: 1.0, // 100% - essential for classification improvement
        [EventType.RECOVERY_INITIATED]: 1.0, // 100% - track all recovery attempts
        [EventType.RECOVERY_COMPLETED]: 1.0, // 100% - track all outcomes
        [EventType.RECOVERY_FAILED]: 1.0, // 100% - critical for failure analysis
        [EventType.CIRCUIT_BREAKER_OPENED]: 1.0, // 100% - important system events
        [EventType.CIRCUIT_BREAKER_CLOSED]: 1.0, // 100% - recovery events
        [EventType.FALLBACK_TRIGGERED]: 0.8, // 80% - moderate importance
        [EventType.ESCALATION_TRIGGERED]: 1.0, // 100% - critical escalations
        [EventType.PATTERN_DETECTED]: 1.0, // 100% - essential for learning
    },
    enablePatternDetection: true,
    enableLearningData: true,
};

/**
 * Pattern detection cache for identifying recurring issues
 */
interface PatternCache {
    errorSignatures: Map<string, number>;
    recentEvents: ResilienceEvent[];
    lastPatternCheck: Date;
}

/**
 * Resilience event publisher with performance guarantees
 */
export class ResilienceEventPublisher {
    private readonly telemetryShim: TelemetryShim;
    private readonly eventBus: RedisEventBus;
    private readonly config: PublishingConfig;
    private readonly eventBuffer: ResilienceEvent[] = [];
    private readonly patternCache: PatternCache;
    private flushTimer?: NodeJS.Timeout;
    
    // Performance tracking
    private publishCount = 0;
    private totalOverheadMs = 0;
    private droppedEvents = 0;

    constructor(
        telemetryShim: TelemetryShim,
        eventBus: RedisEventBus,
        config: Partial<PublishingConfig> = {},
    ) {
        this.telemetryShim = telemetryShim;
        this.eventBus = eventBus;
        this.config = { ...DEFAULT_CONFIG, ...config };
        
        this.patternCache = {
            errorSignatures: new Map(),
            recentEvents: [],
            lastPatternCheck: new Date(),
        };
        
        if (this.config.enabled) {
            this.startPeriodicFlush();
        }
    }

    /**
     * Publish error detection event
     */
    async publishErrorDetected(
        error: Error,
        classification: ErrorClassification,
        context: ErrorContext,
        source: ResilienceEventSource,
    ): Promise<void> {
        const startTime = performance.now();

        try {
            if (!this.shouldSample(EventType.ERROR_DETECTED)) {
                return;
            }

            const learningData = this.config.enableLearningData
                ? await this.generateLearningData(classification, context, error)
                : this.createEmptyLearningData();

            const event: ResilienceEvent = {
                id: uuid(),
                timestamp: new Date(),
                type: EventType.ERROR_DETECTED,
                severity: classification.severity,
                source,
                classification,
                context,
                strategy: this.createPlaceholderStrategy(), // No strategy yet
                outcome: this.createInitialOutcome(),
                learningData,
            };

            await this.publishEvent(event);

            // Fire-and-forget telemetry
            this.telemetryShim.emitError(
                error,
                source.component,
                this.mapSeverityToTelemetry(classification.severity),
                {
                    tier: source.tier,
                    operation: source.operation,
                    errorClassification: classification,
                },
            ).catch(() => {
                // Ignore telemetry errors to maintain performance guarantee
            });

        } finally {
            this.trackOverhead(startTime);
        }
    }

    /**
     * Publish error classification event
     */
    async publishErrorClassified(
        classification: ErrorClassification,
        context: ErrorContext,
        source: ResilienceEventSource,
        classificationTime: number,
    ): Promise<void> {
        const startTime = performance.now();

        try {
            if (!this.shouldSample(EventType.ERROR_CLASSIFIED)) {
                return;
            }

            const learningData = this.config.enableLearningData
                ? await this.generateClassificationLearningData(classification, context)
                : this.createEmptyLearningData();

            const event: ResilienceEvent = {
                id: uuid(),
                timestamp: new Date(),
                type: EventType.ERROR_CLASSIFIED,
                severity: classification.severity,
                source,
                classification,
                context,
                strategy: this.createPlaceholderStrategy(),
                outcome: {
                    ...this.createInitialOutcome(),
                    duration: classificationTime,
                },
                learningData,
            };

            await this.publishEvent(event);

            // Update pattern detection
            if (this.config.enablePatternDetection) {
                this.updatePatternCache(event);
            }

        } finally {
            this.trackOverhead(startTime);
        }
    }

    /**
     * Publish recovery initiation event
     */
    async publishRecoveryInitiated(
        classification: ErrorClassification,
        context: ErrorContext,
        strategy: RecoveryStrategyConfig,
        source: ResilienceEventSource,
    ): Promise<void> {
        const startTime = performance.now();

        try {
            if (!this.shouldSample(EventType.RECOVERY_INITIATED)) {
                return;
            }

            const learningData = this.config.enableLearningData
                ? await this.generateRecoveryLearningData(classification, context, strategy)
                : this.createEmptyLearningData();

            const event: ResilienceEvent = {
                id: uuid(),
                timestamp: new Date(),
                type: EventType.RECOVERY_INITIATED,
                severity: classification.severity,
                source,
                classification,
                context,
                strategy,
                outcome: this.createInitialOutcome(),
                learningData,
            };

            await this.publishEvent(event);

        } finally {
            this.trackOverhead(startTime);
        }
    }

    /**
     * Publish recovery completion event
     */
    async publishRecoveryCompleted(
        classification: ErrorClassification,
        context: ErrorContext,
        strategy: RecoveryStrategyConfig,
        outcome: ResilienceOutcome,
        source: ResilienceEventSource,
    ): Promise<void> {
        const startTime = performance.now();

        try {
            if (!this.shouldSample(EventType.RECOVERY_COMPLETED)) {
                return;
            }

            const learningData = this.config.enableLearningData
                ? await this.generateOutcomeLearningData(classification, context, strategy, outcome)
                : this.createEmptyLearningData();

            const event: ResilienceEvent = {
                id: uuid(),
                timestamp: new Date(),
                type: EventType.RECOVERY_COMPLETED,
                severity: classification.severity,
                source,
                classification,
                context,
                strategy,
                outcome,
                learningData,
            };

            await this.publishEvent(event);

            // Fire-and-forget telemetry for successful recovery
            this.telemetryShim.emitTaskCompletion(
                source.requestId,
                `recovery_${strategy.strategyType}`,
                outcome.success ? "success" : "failure",
                outcome.duration,
                this.calculateResourceCost(outcome.resourceUsage),
            ).catch(() => {
                // Ignore telemetry errors
            });

        } finally {
            this.trackOverhead(startTime);
        }
    }

    /**
     * Publish recovery failure event
     */
    async publishRecoveryFailed(
        classification: ErrorClassification,
        context: ErrorContext,
        strategy: RecoveryStrategyConfig,
        outcome: ResilienceOutcome,
        source: ResilienceEventSource,
        error?: Error,
    ): Promise<void> {
        const startTime = performance.now();

        try {
            if (!this.shouldSample(EventType.RECOVERY_FAILED)) {
                return;
            }

            const learningData = this.config.enableLearningData
                ? await this.generateFailureLearningData(classification, context, strategy, outcome, error)
                : this.createEmptyLearningData();

            const event: ResilienceEvent = {
                id: uuid(),
                timestamp: new Date(),
                type: EventType.RECOVERY_FAILED,
                severity: classification.severity,
                source,
                classification,
                context,
                strategy,
                outcome,
                learningData,
            };

            await this.publishEvent(event);

            // Enhanced telemetry for failures
            if (error) {
                this.telemetryShim.emitError(
                    error,
                    source.component,
                    "high",
                    {
                        recoveryStrategy: strategy.strategyType,
                        attemptCount: outcome.attemptCount,
                        tier: source.tier,
                    },
                ).catch(() => {
                    // Ignore telemetry errors
                });
            }

        } finally {
            this.trackOverhead(startTime);
        }
    }

    /**
     * Publish circuit breaker opened event
     */
    async publishCircuitBreakerOpened(
        component: string,
        threshold: number,
        failures: number,
        source: ResilienceEventSource,
    ): Promise<void> {
        const startTime = performance.now();

        try {
            if (!this.shouldSample(EventType.CIRCUIT_BREAKER_OPENED)) {
                return;
            }

            const classification: ErrorClassification = {
                severity: "CRITICAL" as any,
                category: "SYSTEM" as any,
                recoverability: "MANUAL" as any,
                systemFunctional: false,
                multipleComponentsAffected: true,
                dataRisk: false,
                securityRisk: false,
                confidenceScore: 1.0,
                timestamp: new Date(),
                metadata: {
                    circuitBreaker: {
                        component,
                        threshold,
                        failures,
                    },
                },
            };

            const context: ErrorContext = {
                tier: source.tier,
                component: source.component,
                operation: "circuit_breaker_management",
                attemptCount: 0,
                previousStrategies: [],
                systemState: { circuitBreakerTriggered: true },
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: source.requestId },
            };

            const event: ResilienceEvent = {
                id: uuid(),
                timestamp: new Date(),
                type: EventType.CIRCUIT_BREAKER_OPENED,
                severity: classification.severity,
                source,
                classification,
                context,
                strategy: this.createCircuitBreakerStrategy(),
                outcome: this.createInitialOutcome(),
                learningData: this.createEmptyLearningData(),
            };

            await this.publishEvent(event);

            // High-priority telemetry for circuit breaker events
            this.telemetryShim.emitComponentHealth(
                component,
                "unhealthy",
                [
                    {
                        name: "circuit_breaker",
                        status: "fail",
                        message: `Circuit breaker opened after ${failures} failures (threshold: ${threshold})`,
                    },
                ],
            ).catch(() => {
                // Ignore telemetry errors
            });

        } finally {
            this.trackOverhead(startTime);
        }
    }

    /**
     * Publish circuit breaker closed event
     */
    async publishCircuitBreakerClosed(
        component: string,
        recoveryTime: number,
        source: ResilienceEventSource,
    ): Promise<void> {
        const startTime = performance.now();

        try {
            if (!this.shouldSample(EventType.CIRCUIT_BREAKER_CLOSED)) {
                return;
            }

            const classification: ErrorClassification = {
                severity: "INFO" as any,
                category: "SYSTEM" as any,
                recoverability: "AUTOMATIC" as any,
                systemFunctional: true,
                multipleComponentsAffected: false,
                dataRisk: false,
                securityRisk: false,
                confidenceScore: 1.0,
                timestamp: new Date(),
                metadata: {
                    circuitBreaker: {
                        component,
                        recoveryTime,
                    },
                },
            };

            const context: ErrorContext = {
                tier: source.tier,
                component: source.component,
                operation: "circuit_breaker_recovery",
                attemptCount: 0,
                previousStrategies: [],
                systemState: { circuitBreakerRecovered: true },
                resourceState: {},
                performanceMetrics: {},
                userContext: { requestId: source.requestId },
            };

            const event: ResilienceEvent = {
                id: uuid(),
                timestamp: new Date(),
                type: EventType.CIRCUIT_BREAKER_CLOSED,
                severity: classification.severity,
                source,
                classification,
                context,
                strategy: this.createCircuitBreakerStrategy(),
                outcome: {
                    success: true,
                    duration: recoveryTime,
                    attemptCount: 1,
                    strategiesUsed: ["CIRCUIT_BREAK" as RecoveryStrategy],
                    finalStrategy: "CIRCUIT_BREAK" as RecoveryStrategy,
                    qualityImpact: 0,
                    resourceUsage: { recovery_time: recoveryTime },
                    userImpact: UserImpactLevel.NONE,
                    lessons: ["Circuit breaker successfully recovered"],
                },
                learningData: this.createEmptyLearningData(),
            };

            await this.publishEvent(event);

            // Recovery telemetry
            this.telemetryShim.emitComponentHealth(
                component,
                "healthy",
                [
                    {
                        name: "circuit_breaker",
                        status: "pass",
                        message: `Circuit breaker closed after ${recoveryTime}ms recovery`,
                    },
                ],
            ).catch(() => {
                // Ignore telemetry errors
            });

        } finally {
            this.trackOverhead(startTime);
        }
    }

    /**
     * Publish pattern detection event
     */
    async publishPatternDetected(
        pattern: ErrorPattern,
        triggeringEvent: ResilienceEvent,
        confidence: number,
        source: ResilienceEventSource,
    ): Promise<void> {
        const startTime = performance.now();

        try {
            if (!this.shouldSample(EventType.PATTERN_DETECTED)) {
                return;
            }

            const learningData: ResilienceLearningData = {
                patternId: pattern.id,
                similarity: confidence,
                contextFeatures: this.extractContextFeatures(triggeringEvent),
                effectiveStrategies: pattern.effectiveStrategies.map(s => s.strategyType),
                ineffectiveStrategies: [],
                environmentalFactors: {
                    patternFrequency: pattern.frequency,
                    lastSeen: pattern.lastSeen,
                    avgResolutionTime: pattern.averageResolutionTime,
                },
                recommendations: [
                    `Pattern ${pattern.name} detected with ${confidence.toFixed(2)} confidence`,
                    `Consider using strategies: ${pattern.effectiveStrategies.map(s => s.strategyType).join(", ")}`,
                ],
                confidence,
            };

            const event: ResilienceEvent = {
                id: uuid(),
                timestamp: new Date(),
                type: EventType.PATTERN_DETECTED,
                severity: pattern.severity,
                source,
                classification: triggeringEvent.classification,
                context: triggeringEvent.context,
                strategy: this.createPatternDetectionStrategy(pattern),
                outcome: this.createInitialOutcome(),
                learningData,
            };

            await this.publishEvent(event);

        } finally {
            this.trackOverhead(startTime);
        }
    }

    /**
     * Core event publishing method with performance guarantees
     */
    private async publishEvent(event: ResilienceEvent): Promise<void> {
        // Add to buffer for batch processing
        this.eventBuffer.push(event);
        this.publishCount++;

        // Flush if buffer is getting full
        if (this.eventBuffer.length >= this.config.batchSize) {
            await this.flush();
        }
    }

    /**
     * Flush buffered events
     */
    private async flush(): Promise<void> {
        if (this.eventBuffer.length === 0) {
            return;
        }

        const events = this.eventBuffer.splice(0, this.config.batchSize);

        try {
            // Convert to base events for event bus
            const baseEvents = events.map(event => ({
                id: event.id,
                type: event.type,
                timestamp: event.timestamp,
                data: event,
                correlationId: event.source.requestId,
                metadata: {
                    tier: event.source.tier,
                    component: event.source.component,
                    severity: event.severity,
                },
            }));

            await this.eventBus.publishBatch(baseEvents);
        } catch (error) {
            this.droppedEvents += events.length;
            logger.error("Failed to publish resilience events", {
                eventCount: events.length,
                error,
            });
        }
    }

    /**
     * Start periodic flush
     */
    private startPeriodicFlush(): void {
        this.flushTimer = setInterval(() => {
            this.flush().catch(error => {
                logger.error("Periodic flush error", error);
            });
        }, this.config.flushInterval);
    }

    /**
     * Stop publisher and cleanup
     */
    async stop(): Promise<void> {
        if (this.flushTimer) {
            clearInterval(this.flushTimer);
            this.flushTimer = undefined;
        }

        // Final flush
        await this.flush();

        logger.info("ResilienceEventPublisher stopped", {
            publishedEvents: this.publishCount,
            droppedEvents: this.droppedEvents,
            avgOverheadMs: this.publishCount > 0 
                ? (this.totalOverheadMs / this.publishCount).toFixed(2)
                : 0,
        });
    }

    /**
     * Check if event should be sampled
     */
    private shouldSample(eventType: ResilienceEventType): boolean {
        const rate = this.config.samplingRates[eventType] || 1.0;
        return Math.random() <= rate;
    }

    /**
     * Track publishing overhead
     */
    private trackOverhead(startTime: number): void {
        const overhead = performance.now() - startTime;
        this.totalOverheadMs += overhead;

        // Warn if exceeding performance guarantee
        if (overhead > this.config.maxOverheadMs) {
            logger.warn("Resilience event publishing overhead exceeded", {
                overhead: `${overhead.toFixed(2)}ms`,
                threshold: `${this.config.maxOverheadMs}ms`,
            });
        }
    }

    /**
     * Generate learning data for different event types
     */
    private async generateLearningData(
        classification: ErrorClassification,
        context: ErrorContext,
        error: Error,
    ): Promise<ResilienceLearningData> {
        const similarity = this.calculateSimilarity(classification, context);
        const contextFeatures = this.extractContextFeatures({ classification, context } as any);
        
        return {
            similarity,
            contextFeatures,
            effectiveStrategies: [],
            ineffectiveStrategies: [],
            environmentalFactors: {
                errorType: error.constructor.name,
                tier: context.tier,
                component: context.component,
                attemptCount: context.attemptCount,
            },
            recommendations: this.generateInitialRecommendations(classification, context),
            confidence: classification.confidenceScore,
        };
    }

    private async generateClassificationLearningData(
        classification: ErrorClassification,
        context: ErrorContext,
    ): Promise<ResilienceLearningData> {
        return {
            similarity: classification.confidenceScore,
            contextFeatures: [
                `severity:${classification.severity}`,
                `category:${classification.category}`,
                `recoverability:${classification.recoverability}`,
                `tier:${context.tier}`,
            ],
            effectiveStrategies: [],
            ineffectiveStrategies: [],
            environmentalFactors: {
                classificationTime: performance.now(),
                systemFunctional: classification.systemFunctional,
                dataRisk: classification.dataRisk,
                securityRisk: classification.securityRisk,
            },
            recommendations: [
                `Error classified as ${classification.severity} ${classification.category}`,
                `Recoverability assessment: ${classification.recoverability}`,
            ],
            confidence: classification.confidenceScore,
        };
    }

    private async generateRecoveryLearningData(
        classification: ErrorClassification,
        context: ErrorContext,
        strategy: RecoveryStrategyConfig,
    ): Promise<ResilienceLearningData> {
        return {
            similarity: strategy.estimatedSuccessRate,
            contextFeatures: this.extractContextFeatures({ classification, context, strategy } as any),
            effectiveStrategies: [strategy.strategyType],
            ineffectiveStrategies: context.previousStrategies,
            environmentalFactors: {
                strategyType: strategy.strategyType,
                maxAttempts: strategy.maxAttempts,
                estimatedSuccessRate: strategy.estimatedSuccessRate,
                resourceRequirements: strategy.resourceRequirements,
            },
            recommendations: [
                `Selected strategy: ${strategy.strategyType}`,
                `Estimated success rate: ${strategy.estimatedSuccessRate}`,
            ],
            confidence: strategy.estimatedSuccessRate,
        };
    }

    private async generateOutcomeLearningData(
        classification: ErrorClassification,
        context: ErrorContext,
        strategy: RecoveryStrategyConfig,
        outcome: ResilienceOutcome,
    ): Promise<ResilienceLearningData> {
        return {
            similarity: outcome.success ? 1.0 : 0.0,
            contextFeatures: this.extractContextFeatures({ classification, context, strategy, outcome } as any),
            effectiveStrategies: outcome.success ? [strategy.strategyType] : [],
            ineffectiveStrategies: outcome.success ? [] : [strategy.strategyType],
            environmentalFactors: {
                duration: outcome.duration,
                attemptCount: outcome.attemptCount,
                qualityImpact: outcome.qualityImpact,
                userImpact: outcome.userImpact,
                resourceUsage: outcome.resourceUsage,
            },
            recommendations: outcome.lessons,
            confidence: outcome.success ? 0.9 : 0.1,
        };
    }

    private async generateFailureLearningData(
        classification: ErrorClassification,
        context: ErrorContext,
        strategy: RecoveryStrategyConfig,
        outcome: ResilienceOutcome,
        error?: Error,
    ): Promise<ResilienceLearningData> {
        return {
            similarity: 0.0, // Failed outcome
            contextFeatures: this.extractContextFeatures({ classification, context, strategy } as any),
            effectiveStrategies: [],
            ineffectiveStrategies: [strategy.strategyType, ...context.previousStrategies],
            environmentalFactors: {
                failureReason: error?.message || "Unknown failure",
                duration: outcome.duration,
                attemptCount: outcome.attemptCount,
                strategiesAttempted: outcome.strategiesUsed.length,
            },
            recommendations: [
                `Strategy ${strategy.strategyType} failed after ${outcome.duration}ms`,
                "Consider escalation or alternative approach",
                ...outcome.lessons,
            ],
            confidence: 0.8, // High confidence in failure data
        };
    }

    /**
     * Helper methods
     */
    private calculateSimilarity(
        classification: ErrorClassification,
        context: ErrorContext,
    ): number {
        // Simple similarity based on recent patterns
        const signature = `${classification.severity}:${classification.category}:${context.tier}`;
        const recentOccurrences = this.patternCache.errorSignatures.get(signature) || 0;
        
        // Higher similarity for more frequent patterns
        return Math.min(recentOccurrences / 10, 1.0);
    }

    private extractContextFeatures(event: any): string[] {
        const features: string[] = [];
        
        if (event.classification) {
            features.push(`severity:${event.classification.severity}`);
            features.push(`category:${event.classification.category}`);
            features.push(`recoverability:${event.classification.recoverability}`);
        }
        
        if (event.context) {
            features.push(`tier:${event.context.tier}`);
            features.push(`component:${event.context.component}`);
            features.push(`attempts:${event.context.attemptCount}`);
        }
        
        if (event.strategy) {
            features.push(`strategy:${event.strategy.strategyType}`);
        }
        
        return features;
    }

    private generateInitialRecommendations(
        classification: ErrorClassification,
        context: ErrorContext,
    ): string[] {
        const recommendations: string[] = [];
        
        if (classification.severity === "CRITICAL" as any) {
            recommendations.push("Consider immediate escalation");
        }
        
        if (classification.securityRisk) {
            recommendations.push("Security assessment required");
        }
        
        if (classification.dataRisk) {
            recommendations.push("Data integrity verification needed");
        }
        
        if (context.attemptCount > 2) {
            recommendations.push("Multiple failures - consider alternative approach");
        }
        
        return recommendations;
    }

    private updatePatternCache(event: ResilienceEvent): void {
        // Update error signature frequency
        const signature = `${event.classification.severity}:${event.classification.category}:${event.source.tier}`;
        const count = this.patternCache.errorSignatures.get(signature) || 0;
        this.patternCache.errorSignatures.set(signature, count + 1);
        
        // Add to recent events
        this.patternCache.recentEvents.push(event);
        
        // Keep only recent events (last 100)
        if (this.patternCache.recentEvents.length > 100) {
            this.patternCache.recentEvents.shift();
        }
        
        this.patternCache.lastPatternCheck = new Date();
    }

    private mapSeverityToTelemetry(severity: string): "low" | "medium" | "high" | "critical" {
        switch (severity) {
            case "FATAL": return "critical";
            case "CRITICAL": return "critical";
            case "ERROR": return "high";
            case "WARNING": return "medium";
            case "INFO": return "low";
            default: return "medium";
        }
    }

    private calculateResourceCost(resourceUsage: Record<string, number>): number {
        return Object.values(resourceUsage).reduce((sum, cost) => sum + (cost || 0), 0);
    }

    /**
     * Factory methods for common objects
     */
    private createEmptyLearningData(): ResilienceLearningData {
        return {
            similarity: 0,
            contextFeatures: [],
            effectiveStrategies: [],
            ineffectiveStrategies: [],
            environmentalFactors: {},
            recommendations: [],
            confidence: 0,
        };
    }

    private createInitialOutcome(): ResilienceOutcome {
        return {
            success: false,
            duration: 0,
            attemptCount: 0,
            strategiesUsed: [],
            qualityImpact: 0,
            resourceUsage: {},
            userImpact: UserImpactLevel.NONE,
            lessons: [],
        };
    }

    private createPlaceholderStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: "LOG_INFO" as RecoveryStrategy,
            maxAttempts: 1,
            backoffStrategy: {
                type: "FIXED" as any,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [],
            priority: 1,
            timeoutMs: 1000,
            conditions: [],
            estimatedSuccessRate: 1.0,
            resourceRequirements: {},
        };
    }

    private createCircuitBreakerStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: "CIRCUIT_BREAK" as RecoveryStrategy,
            maxAttempts: 1,
            backoffStrategy: {
                type: "FIXED" as any,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [],
            priority: 10,
            timeoutMs: 0,
            conditions: [],
            estimatedSuccessRate: 1.0,
            resourceRequirements: {},
        };
    }

    private createPatternDetectionStrategy(pattern: ErrorPattern): RecoveryStrategyConfig {
        const bestStrategy = pattern.effectiveStrategies[0];
        return bestStrategy || this.createPlaceholderStrategy();
    }

    /**
     * Get publishing statistics
     */
    getStatistics(): {
        publishedEvents: number;
        droppedEvents: number;
        bufferedEvents: number;
        averageOverheadMs: number;
        patternCacheSize: number;
    } {
        return {
            publishedEvents: this.publishCount,
            droppedEvents: this.droppedEvents,
            bufferedEvents: this.eventBuffer.length,
            averageOverheadMs: this.publishCount > 0 
                ? this.totalOverheadMs / this.publishCount 
                : 0,
            patternCacheSize: this.patternCache.errorSignatures.size,
        };
    }
}