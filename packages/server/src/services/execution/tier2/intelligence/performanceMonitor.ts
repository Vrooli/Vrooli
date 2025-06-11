/**
 * Performance Monitor - Simple performance data collection for monitoring agents
 * 
 * This component collects performance metrics and emits events.
 * Performance analysis and optimization emerge from monitoring agents.
 */

import { type Logger } from "winston";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";

/**
 * Performance metric
 */
export interface PerformanceMetric {
    runId: string;
    stepId?: string;
    metric: string;
    value: number;
    timestamp: Date;
    metadata?: Record<string, unknown>;
}

/**
 * PerformanceMonitor - Event emitter for performance analysis
 * 
 * Collects performance metrics and emits events for monitoring agents.
 * Does NOT implement analysis algorithms - those emerge from agent analysis.
 */
export class PerformanceMonitor {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly metrics: Map<string, PerformanceMetric[]> = new Map();

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.subscribeToEvents();
    }

    /**
     * Records a performance metric
     */
    async recordMetric(
        runId: string,
        metric: string,
        value: number,
        stepId?: string,
        metadata?: Record<string, unknown>,
    ): Promise<void> {
        const performanceMetric: PerformanceMetric = {
            runId,
            stepId,
            metric,
            value,
            timestamp: new Date(),
            metadata,
        };

        // Store for historical reference
        if (!this.metrics.has(runId)) {
            this.metrics.set(runId, []);
        }
        this.metrics.get(runId)!.push(performanceMetric);

        // Emit performance metric event for monitoring agents
        await this.eventBus.publish("performance.events", {
            type: "METRIC_RECORDED",
            timestamp: new Date(),
            metadata: performanceMetric,
        });

        this.logger.debug("[PerformanceMonitor] Recorded metric", {
            runId,
            stepId,
            metric,
            value,
        });
    }

    /**
     * Records step execution timing
     */
    async recordStepTiming(
        runId: string,
        stepId: string,
        duration: number,
        metadata?: Record<string, unknown>,
    ): Promise<void> {
        await this.recordMetric(runId, "step_duration", duration, stepId, metadata);
    }

    /**
     * Records resource usage
     */
    async recordResourceUsage(
        runId: string,
        stepId: string,
        resources: {
            credits?: number;
            tokens?: number;
            memory?: number;
            apiCalls?: number;
        },
    ): Promise<void> {
        for (const [resource, value] of Object.entries(resources)) {
            if (value !== undefined) {
                await this.recordMetric(runId, `resource_${resource}`, value, stepId);
            }
        }
    }

    /**
     * Gets performance metrics for a run
     */
    getRunMetrics(runId: string): PerformanceMetric[] {
        return this.metrics.get(runId) || [];
    }

    /**
     * Emits performance analysis request for agents
     */
    async requestPerformanceAnalysis(runId: string): Promise<void> {
        const metrics = this.getRunMetrics(runId);
        
        await this.eventBus.publish("performance.events", {
            type: "PERFORMANCE_ANALYSIS_REQUEST",
            timestamp: new Date(),
            metadata: {
                runId,
                metrics,
                metricCount: metrics.length,
            },
        });
    }

    /**
     * Private helper methods
     */
    private subscribeToEvents(): void {
        // Subscribe to run events to automatically track metrics
        this.eventBus.subscribe("run.events", async (event) => {
            const runId = event.runId;
            
            switch (event.type) {
                case "Started":
                    await this.recordMetric(runId, "run_started", 1, undefined, {
                        timestamp: event.timestamp,
                    });
                    break;
                    
                case "Completed":
                    await this.recordMetric(runId, "run_completed", 1, undefined, {
                        timestamp: event.timestamp,
                        duration: event.metadata?.duration,
                    });
                    break;
                    
                case "Failed":
                    await this.recordMetric(runId, "run_failed", 1, undefined, {
                        timestamp: event.timestamp,
                        error: event.metadata?.error,
                    });
                    break;
                    
                case "ProgressUpdated":
                    if (event.metadata?.percentComplete !== undefined) {
                        await this.recordMetric(runId, "progress", event.metadata.percentComplete);
                    }
                    break;
            }
        });

        // Subscribe to step events
        this.eventBus.subscribe("execution.step.completed", async (event) => {
            if (event.metadata?.runId && event.metadata?.stepId) {
                await this.recordMetric(
                    event.metadata.runId,
                    "step_completed",
                    1,
                    event.metadata.stepId,
                    {
                        timestamp: event.timestamp,
                        status: event.metadata.status,
                        duration: event.metadata.duration,
                    },
                );
            }
        });
    }

    /**
     * Cleans up old metrics
     */
    async cleanup(maxAge: number = 86400000): Promise<void> {
        const cutoff = new Date(Date.now() - maxAge);
        let cleaned = 0;

        for (const [runId, metrics] of this.metrics) {
            const filtered = metrics.filter(m => m.timestamp > cutoff);
            if (filtered.length !== metrics.length) {
                this.metrics.set(runId, filtered);
                cleaned += metrics.length - filtered.length;
            }
            
            // Remove empty entries
            if (filtered.length === 0) {
                this.metrics.delete(runId);
            }
        }

        if (cleaned > 0) {
            this.logger.info("[PerformanceMonitor] Cleaned up old metrics", {
                cleaned,
                remaining: Array.from(this.metrics.values()).reduce((sum, m) => sum + m.length, 0),
            });
        }
    }
}