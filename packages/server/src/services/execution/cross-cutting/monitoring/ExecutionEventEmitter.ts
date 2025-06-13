/**
 * Execution Event Emitter - Minimal event emission for monitoring
 * 
 * This component replaces the complex UnifiedMonitoringService with simple
 * event emission. All monitoring intelligence emerges from agents that
 * subscribe to these events.
 * 
 * IMPORTANT: This component does NOT:
 * - Analyze metrics
 * - Detect patterns
 * - Calculate statistics
 * - Make decisions
 * 
 * It ONLY emits raw events for agents to process.
 */

import { type Logger } from "winston";
import { type EventBus } from "../events/eventBus.js";
import { EventPublisher } from "../../shared/EventPublisher.js";
import { ErrorHandler, ComponentErrorHandler } from "../../shared/ErrorHandler.js";

/**
 * Raw metric event - no analysis, just data
 */
export interface RawMetricEvent {
    tier: 1 | 2 | 3 | "cross-cutting";
    component: string;
    metricType: "performance" | "resource" | "health" | "business" | "safety" | "quality" | "efficiency" | "intelligence";
    name: string;
    value: number;
    unit?: string;
    tags?: Record<string, string>;
    timestamp?: Date;
}

/**
 * Raw execution event - no interpretation
 */
export interface RawExecutionEvent {
    executionId: string;
    event: "started" | "completed" | "failed" | "progress";
    tier: 1 | 2 | 3;
    component: string;
    data: Record<string, unknown>;
    timestamp?: Date;
}

/**
 * Execution Event Emitter
 * 
 * Provides minimal event emission for the execution architecture.
 * Monitoring agents subscribe to these events to provide analytics,
 * pattern detection, anomaly detection, and other intelligence.
 */
export class ExecutionEventEmitter {
    private readonly eventPublisher: EventPublisher;
    private readonly errorHandler: ComponentErrorHandler;
    
    constructor(
        private readonly logger: Logger,
        private readonly eventBus: EventBus,
        componentName: string = "ExecutionEventEmitter"
    ) {
        this.eventPublisher = new EventPublisher(eventBus, logger, componentName);
        this.errorHandler = new ErrorHandler(logger, this.eventPublisher).createComponentHandler(componentName);
    }
    
    /**
     * Emit a raw metric event
     * No analysis, no aggregation, just emission
     */
    async emitMetric(metric: RawMetricEvent): Promise<void> {
        await this.errorHandler.execute(
            async () => {
                // Add timestamp if not provided
                const eventData = {
                    ...metric,
                    timestamp: metric.timestamp || new Date(),
                };
                
                // Emit to tier-specific channel for efficient routing
                const channel = `execution.metrics.${metric.tier}`;
                await this.eventPublisher.publish(channel, eventData);
                
                // Also emit to metric type channel for specialized agents
                const typeChannel = `execution.metrics.type.${metric.metricType}`;
                await this.eventPublisher.publish(typeChannel, eventData);
            },
            "emitMetric",
            { metricName: metric.name, tier: metric.tier }
        );
    }
    
    /**
     * Emit a raw execution event
     * No processing, no decisions, just emission
     */
    async emitExecutionEvent(event: RawExecutionEvent): Promise<void> {
        await this.errorHandler.execute(
            async () => {
                // Add timestamp if not provided
                const eventData = {
                    ...event,
                    timestamp: event.timestamp || new Date(),
                };
                
                // Emit to execution channel
                const channel = `execution.events.${event.tier}`;
                await this.eventPublisher.publish(channel, eventData);
                
                // Also emit to event type channel
                const typeChannel = `execution.events.type.${event.event}`;
                await this.eventPublisher.publish(typeChannel, eventData);
            },
            "emitExecutionEvent",
            { executionId: event.executionId, eventType: event.event }
        );
    }
    
    /**
     * Batch emit metrics for efficiency
     * Still no analysis, just efficient emission
     */
    async emitMetricsBatch(metrics: RawMetricEvent[]): Promise<void> {
        if (metrics.length === 0) return;
        
        await this.errorHandler.execute(
            async () => {
                // Group by channel for efficient publishing
                const byTier = new Map<string, RawMetricEvent[]>();
                const byType = new Map<string, RawMetricEvent[]>();
                
                for (const metric of metrics) {
                    const tierChannel = `execution.metrics.${metric.tier}`;
                    const typeChannel = `execution.metrics.type.${metric.metricType}`;
                    
                    if (!byTier.has(tierChannel)) {
                        byTier.set(tierChannel, []);
                    }
                    if (!byType.has(typeChannel)) {
                        byType.set(typeChannel, []);
                    }
                    
                    const eventData = {
                        ...metric,
                        timestamp: metric.timestamp || new Date(),
                    };
                    
                    byTier.get(tierChannel)!.push(eventData);
                    byType.get(typeChannel)!.push(eventData);
                }
                
                // Publish batches
                const publishPromises: Promise<void>[] = [];
                
                for (const [channel, channelMetrics] of byTier) {
                    publishPromises.push(
                        this.eventPublisher.publish(channel, {
                            type: "batch",
                            metrics: channelMetrics,
                            count: channelMetrics.length,
                        })
                    );
                }
                
                for (const [channel, channelMetrics] of byType) {
                    publishPromises.push(
                        this.eventPublisher.publish(channel, {
                            type: "batch",
                            metrics: channelMetrics,
                            count: channelMetrics.length,
                        })
                    );
                }
                
                await Promise.all(publishPromises);
            },
            "emitMetricsBatch",
            { count: metrics.length }
        );
    }
    
    /**
     * Helper to create a child emitter for a specific component
     * Makes it easier for components to emit with consistent metadata
     */
    createComponentEmitter(tier: 1 | 2 | 3 | "cross-cutting", componentName: string): ComponentEventEmitter {
        return new ComponentEventEmitter(this, tier, componentName);
    }
}

/**
 * Component-specific event emitter
 * Automatically adds tier and component metadata
 */
export class ComponentEventEmitter {
    constructor(
        private readonly parent: ExecutionEventEmitter,
        private readonly tier: 1 | 2 | 3 | "cross-cutting",
        private readonly component: string
    ) {}
    
    async emitMetric(
        metricType: RawMetricEvent["metricType"],
        name: string,
        value: number,
        unit?: string,
        tags?: Record<string, string>
    ): Promise<void> {
        await this.parent.emitMetric({
            tier: this.tier,
            component: this.component,
            metricType,
            name,
            value,
            unit,
            tags,
        });
    }
    
    async emitExecutionEvent(
        executionId: string,
        event: RawExecutionEvent["event"],
        data: Record<string, unknown>
    ): Promise<void> {
        await this.parent.emitExecutionEvent({
            executionId,
            event,
            tier: this.tier as 1 | 2 | 3, // cross-cutting doesn't emit execution events
            component: this.component,
            data,
        });
    }
}

/**
 * Example monitoring agent that would subscribe to these events:
 * 
 * ```typescript
 * // This would be a routine deployed by teams, NOT hardcoded
 * class PerformanceMonitoringAgent {
 *     async processMetric(metric: RawMetricEvent) {
 *         // Agent logic to analyze performance
 *         if (metric.metricType === "performance") {
 *             // Detect anomalies, calculate statistics, etc.
 *             // All intelligence emerges here, not in infrastructure
 *         }
 *     }
 * }
 * ```
 */