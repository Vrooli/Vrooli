/**
 * Strategy Event Emitter
 * 
 * Emits strategy performance events to the event bus for consumption by learning agents.
 * This enables event-driven learning and optimization rather than direct strategy modification.
 */

import { type Logger } from "winston";
import { StrategyType } from "@vrooli/shared";

/**
 * Strategy performance event data
 */
export interface StrategyPerformanceEvent {
    strategyType: StrategyType;
    strategyName: string;
    strategyVersion: string;
    execution: {
        stepId: string;
        stepType: string;
        success: boolean;
        executionTime: number;
        resourceUsage: {
            credits: number;
            tokens: number;
            apiCalls: number;
            cost: number;
        };
        confidence?: number;
        outputs?: unknown;
        error?: string;
    };
    context: {
        stepType: string;
        inputComplexity: number;
        constraintCount: number;
        historyDepth: number;
    };
    timestamp: Date;
}

/**
 * Strategy failure event data
 */
export interface StrategyFailureEvent {
    strategyType: StrategyType;
    strategyName: string;
    strategyVersion: string;
    execution: {
        stepId: string;
        stepType: string;
        success: false;
        executionTime: number;
        resourceUsage: {
            credits: number;
            tokens: number;
            apiCalls: number;
            cost: number;
        };
        error: string;
    };
    context: {
        stepType: string;
        inputComplexity: number;
        constraintCount: number;
        historyDepth: number;
    };
    timestamp: Date;
}

/**
 * Strategy feedback event data
 */
export interface StrategyFeedbackEvent {
    strategyType: StrategyType;
    strategyName: string;
    strategyVersion: string;
    feedback: {
        outcome: "success" | "partial" | "failure";
        userSatisfaction?: number;
        performanceScore: number;
        issues?: string[];
        improvements?: string[];
    };
    timestamp: Date;
}

/**
 * Strategy threshold event (when performance crosses significant thresholds)
 */
export interface StrategyThresholdEvent {
    strategyType: StrategyType;
    strategyName: string;
    strategyVersion: string;
    threshold: {
        type: "success_rate" | "execution_time" | "confidence" | "cost_efficiency";
        direction: "above" | "below";
        value: number;
        previous: number;
        significance: "low" | "medium" | "high";
    };
    context: {
        totalExecutions: number;
        recentPerformance: {
            successRate: number;
            averageExecutionTime: number;
            averageConfidence: number;
            averageCost: number;
        };
    };
    timestamp: Date;
}

/**
 * Interface for event bus integration
 */
interface IEventBus {
    publish(event: any): Promise<void>;
}

/**
 * Simple event bus interface for backward compatibility
 */
interface ISimpleEventBus {
    publish(event: any): Promise<void>;
    on(eventType: string, handler: (event: any) => void): void;
}

/**
 * Strategy event emitter for performance and learning events
 */
export class StrategyEventEmitter {
    private readonly logger: Logger;
    private eventBus?: IEventBus | ISimpleEventBus;
    private eventQueue: any[] = [];
    private readonly maxQueueSize = 100;

    // Threshold tracking for significant changes
    private lastMetrics = {
        successRate: 0,
        averageExecutionTime: 0,
        averageConfidence: 0,
        averageCost: 0,
    };

    constructor(logger: Logger, eventBus?: IEventBus | ISimpleEventBus) {
        this.logger = logger;
        this.eventBus = eventBus;
    }

    /**
     * Set event bus for integration
     */
    setEventBus(eventBus: IEventBus | ISimpleEventBus): void {
        this.eventBus = eventBus;
        
        // Flush queued events
        this.flushEventQueue().catch(error => {
            this.logger.error("[StrategyEventEmitter] Failed to flush event queue", { error });
        });
    }

    /**
     * Emit strategy performance event
     */
    async emitStrategyPerformance(event: StrategyPerformanceEvent): Promise<void> {
        const eventData = {
            id: `strategy-perf-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            type: "strategy/performance/completed",
            timestamp: event.timestamp,
            source: {
                tier: 3 as const,
                component: "strategy-execution",
                instanceId: event.strategyName,
            },
            correlationId: event.execution.stepId,
            data: event,
            metadata: {
                version: "1.0.0",
                tags: [
                    "strategy",
                    "performance",
                    event.strategyType.toLowerCase(),
                    event.execution.success ? "success" : "failure",
                ],
                priority: event.execution.success ? "NORMAL" : "HIGH",
                userId: event.context.stepType, // Use stepType as a context identifier
            },
        };

        await this.publishEvent(eventData);
    }

    /**
     * Emit strategy failure event
     */
    async emitStrategyFailure(event: StrategyFailureEvent): Promise<void> {
        const eventData = {
            id: `strategy-fail-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            type: "strategy/performance/failed",
            timestamp: event.timestamp,
            source: {
                tier: 3 as const,
                component: "strategy-execution",
                instanceId: event.strategyName,
            },
            correlationId: event.execution.stepId,
            data: event,
            metadata: {
                version: "1.0.0",
                tags: [
                    "strategy",
                    "failure",
                    event.strategyType.toLowerCase(),
                    "error",
                ],
                priority: "HIGH",
                userId: event.context.stepType,
            },
        };

        await this.publishEvent(eventData);
    }

    /**
     * Emit strategy feedback event
     */
    async emitStrategyFeedback(event: StrategyFeedbackEvent): Promise<void> {
        const eventData = {
            id: `strategy-feedback-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            type: "strategy/feedback/received",
            timestamp: event.timestamp,
            source: {
                tier: 3 as const,
                component: "strategy-execution",
                instanceId: event.strategyName,
            },
            correlationId: `feedback-${event.strategyType}`,
            data: event,
            metadata: {
                version: "1.0.0",
                tags: [
                    "strategy",
                    "feedback",
                    event.strategyType.toLowerCase(),
                    event.feedback.outcome,
                ],
                priority: event.feedback.outcome === "failure" ? "HIGH" : "NORMAL",
            },
        };

        await this.publishEvent(eventData);
    }

    /**
     * Emit strategy threshold event
     */
    async emitStrategyThreshold(event: StrategyThresholdEvent): Promise<void> {
        const eventData = {
            id: `strategy-threshold-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            type: "strategy/threshold/crossed",
            timestamp: event.timestamp,
            source: {
                tier: 3 as const,
                component: "strategy-execution",
                instanceId: event.strategyName,
            },
            correlationId: `threshold-${event.strategyType}`,
            data: event,
            metadata: {
                version: "1.0.0",
                tags: [
                    "strategy",
                    "threshold",
                    event.strategyType.toLowerCase(),
                    event.threshold.type,
                    event.threshold.significance,
                ],
                priority: event.threshold.significance === "high" ? "HIGH" : "NORMAL",
            },
        };

        await this.publishEvent(eventData);
    }

    /**
     * Check and emit threshold events based on current metrics
     */
    async checkThresholds(metrics: {
        successRate: number;
        averageExecutionTime: number;
        averageConfidence: number;
        averageCost: number;
        totalExecutions: number;
    }, strategyInfo: {
        type: StrategyType;
        name: string;
        version: string;
    }): Promise<void> {
        const thresholds = this.detectThresholdCrossings(metrics);
        
        for (const threshold of thresholds) {
            await this.emitStrategyThreshold({
                strategyType: strategyInfo.type,
                strategyName: strategyInfo.name,
                strategyVersion: strategyInfo.version,
                threshold,
                context: {
                    totalExecutions: metrics.totalExecutions,
                    recentPerformance: {
                        successRate: metrics.successRate,
                        averageExecutionTime: metrics.averageExecutionTime,
                        averageConfidence: metrics.averageConfidence,
                        averageCost: metrics.averageCost,
                    },
                },
                timestamp: new Date(),
            });
        }

        // Update last metrics for future comparisons
        this.lastMetrics = {
            successRate: metrics.successRate,
            averageExecutionTime: metrics.averageExecutionTime,
            averageConfidence: metrics.averageConfidence,
            averageCost: metrics.averageCost,
        };
    }

    /**
     * Private helper methods
     */
    private async publishEvent(eventData: any): Promise<void> {
        try {
            if (this.eventBus) {
                await this.eventBus.publish(eventData);
                this.logger.debug("[StrategyEventEmitter] Published event", {
                    eventId: eventData.id,
                    eventType: eventData.type,
                });
            } else {
                // Queue event for later if no event bus is available
                this.queueEvent(eventData);
            }
        } catch (error) {
            this.logger.error("[StrategyEventEmitter] Failed to publish event", {
                eventId: eventData.id,
                eventType: eventData.type,
                error: error instanceof Error ? error.message : String(error),
            });
            
            // Queue failed events for retry
            this.queueEvent(eventData);
        }
    }

    private queueEvent(eventData: any): void {
        this.eventQueue.push(eventData);
        
        // Prevent queue from growing too large
        if (this.eventQueue.length > this.maxQueueSize) {
            this.eventQueue = this.eventQueue.slice(-this.maxQueueSize);
            this.logger.warn("[StrategyEventEmitter] Event queue overflow, dropped oldest events");
        }
        
        this.logger.debug("[StrategyEventEmitter] Queued event", {
            eventId: eventData.id,
            queueSize: this.eventQueue.length,
        });
    }

    private async flushEventQueue(): Promise<void> {
        if (!this.eventBus || this.eventQueue.length === 0) {
            return;
        }

        const events = [...this.eventQueue];
        this.eventQueue = [];

        this.logger.info("[StrategyEventEmitter] Flushing event queue", {
            eventCount: events.length,
        });

        for (const event of events) {
            try {
                await this.eventBus.publish(event);
            } catch (error) {
                this.logger.error("[StrategyEventEmitter] Failed to flush queued event", {
                    eventId: event.id,
                    error: error instanceof Error ? error.message : String(error),
                });
                // Re-queue failed event
                this.eventQueue.push(event);
            }
        }
    }

    private detectThresholdCrossings(current: {
        successRate: number;
        averageExecutionTime: number;
        averageConfidence: number;
        averageCost: number;
    }): Array<{
        type: "success_rate" | "execution_time" | "confidence" | "cost_efficiency";
        direction: "above" | "below";
        value: number;
        previous: number;
        significance: "low" | "medium" | "high";
    }> {
        const thresholds = [];

        // Success rate thresholds
        if (this.isSignificantChange(current.successRate, this.lastMetrics.successRate, 0.1)) {
            thresholds.push({
                type: "success_rate" as const,
                direction: current.successRate > this.lastMetrics.successRate ? "above" as const : "below" as const,
                value: current.successRate,
                previous: this.lastMetrics.successRate,
                significance: this.getSignificance(current.successRate, this.lastMetrics.successRate),
            });
        }

        // Execution time thresholds (lower is better)
        if (this.isSignificantChange(current.averageExecutionTime, this.lastMetrics.averageExecutionTime, 5000)) {
            thresholds.push({
                type: "execution_time" as const,
                direction: current.averageExecutionTime > this.lastMetrics.averageExecutionTime ? "above" as const : "below" as const,
                value: current.averageExecutionTime,
                previous: this.lastMetrics.averageExecutionTime,
                significance: this.getSignificance(current.averageExecutionTime, this.lastMetrics.averageExecutionTime),
            });
        }

        // Confidence thresholds
        if (this.isSignificantChange(current.averageConfidence, this.lastMetrics.averageConfidence, 0.1)) {
            thresholds.push({
                type: "confidence" as const,
                direction: current.averageConfidence > this.lastMetrics.averageConfidence ? "above" as const : "below" as const,
                value: current.averageConfidence,
                previous: this.lastMetrics.averageConfidence,
                significance: this.getSignificance(current.averageConfidence, this.lastMetrics.averageConfidence),
            });
        }

        // Cost efficiency thresholds (lower is better)
        if (this.isSignificantChange(current.averageCost, this.lastMetrics.averageCost, 0.01)) {
            thresholds.push({
                type: "cost_efficiency" as const,
                direction: current.averageCost > this.lastMetrics.averageCost ? "above" as const : "below" as const,
                value: current.averageCost,
                previous: this.lastMetrics.averageCost,
                significance: this.getSignificance(current.averageCost, this.lastMetrics.averageCost),
            });
        }

        return thresholds;
    }

    private isSignificantChange(current: number, previous: number, threshold: number): boolean {
        if (previous === 0) return current > threshold;
        return Math.abs(current - previous) >= threshold;
    }

    private getSignificance(current: number, previous: number): "low" | "medium" | "high" {
        if (previous === 0) return current > 0.5 ? "high" : "low";
        
        const percentChange = Math.abs((current - previous) / previous);
        
        if (percentChange > 0.3) return "high";
        if (percentChange > 0.15) return "medium";
        return "low";
    }

    /**
     * Get queued event count (useful for debugging)
     */
    getQueuedEventCount(): number {
        return this.eventQueue.length;
    }

    /**
     * Clear event queue (useful for testing)
     */
    clearQueue(): void {
        this.eventQueue = [];
        this.logger.debug("[StrategyEventEmitter] Cleared event queue");
    }
}