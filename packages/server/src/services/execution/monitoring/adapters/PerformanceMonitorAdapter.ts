/**
 * PerformanceMonitorAdapter - Adapter for Tier 2 PerformanceMonitor
 * 
 * Maintains backward compatibility with the original PerformanceMonitor
 * while routing all operations through UnifiedMonitoringService.
 */

import { BaseMonitoringAdapter } from "./BaseMonitoringAdapter.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type Logger } from "winston";

/**
 * Performance metric types (from original PerformanceMonitor)
 */
export type MetricType = 'duration' | 'count' | 'rate' | 'gauge' | 'custom';

/**
 * Resource usage structure
 */
export interface ResourceUsage {
    credits: number;
    tokens: number;
    toolCalls: number;
    memory?: number;
    cpu?: number;
}

/**
 * Run metrics structure
 */
export interface RunMetrics {
    runId: string;
    startTime: Date;
    endTime?: Date;
    totalDuration?: number;
    stepCount: number;
    successfulSteps: number;
    failedSteps: number;
    resourceUsage: ResourceUsage;
    customMetrics: Record<string, number>;
    stepTimings: Array<{
        stepId: string;
        duration: number;
        resources?: ResourceUsage;
    }>;
}

/**
 * Performance analysis request
 */
export interface PerformanceAnalysisRequest {
    runId: string;
    metrics: RunMetrics;
    context: {
        routineId?: string;
        userId?: string;
        complexity?: number;
    };
}

/**
 * PerformanceMonitorAdapter - Routes Tier 2 performance monitoring through unified service
 */
export class PerformanceMonitorAdapter extends BaseMonitoringAdapter {
    private runMetrics: Map<string, RunMetrics> = new Map();

    constructor(
        eventBus: EventBus,
        logger: Logger
    ) {
        super(
            {
                componentName: "PerformanceMonitor",
                tier: 2,
                eventChannels: ["run.performance", "run.metrics"],
                enableEventBus: true,
            },
            logger,
            eventBus
        );
    }

    /**
     * Record a metric for a run
     */
    async recordRunMetric(
        runId: string,
        metric: string,
        value: number,
        stepId?: string,
        metadata?: Record<string, any>
    ): Promise<void> {
        // Initialize run metrics if needed
        if (!this.runMetrics.has(runId)) {
            this.runMetrics.set(runId, {
                runId,
                startTime: new Date(),
                stepCount: 0,
                successfulSteps: 0,
                failedSteps: 0,
                resourceUsage: {
                    credits: 0,
                    tokens: 0,
                    toolCalls: 0,
                },
                customMetrics: {},
                stepTimings: [],
            });
        }

        const runMetrics = this.runMetrics.get(runId)!;
        
        // Store custom metric
        runMetrics.customMetrics[metric] = value;

        // Record to unified service
        await super.recordMetric(
            `run.metric.${metric}`,
            value,
            this.getMetricType(metric),
            {
                runId,
                stepId,
                ...metadata,
            }
        );

        // Emit event for monitoring agents
        await this.emitEvent('run.metrics.recorded', {
            runId,
            metric,
            value,
            stepId,
            metadata,
            timestamp: new Date(),
        });
    }

    /**
     * Record step timing
     */
    async recordStepTiming(
        runId: string,
        stepId: string,
        duration: number,
        metadata?: Record<string, any>
    ): Promise<void> {
        // Update run metrics
        const runMetrics = this.runMetrics.get(runId);
        if (runMetrics) {
            runMetrics.stepTimings.push({
                stepId,
                duration,
                resources: metadata?.resources,
            });
            runMetrics.stepCount++;
            
            if (metadata?.success !== false) {
                runMetrics.successfulSteps++;
            } else {
                runMetrics.failedSteps++;
            }
        }

        // Record to unified service
        await super.recordMetric(
            'run.step.duration',
            duration,
            'performance',
            {
                runId,
                stepId,
                success: metadata?.success !== false,
                ...metadata,
            }
        );

        // Record step completion
        await super.recordMetric(
            metadata?.success !== false ? 'run.step.success' : 'run.step.failure',
            1,
            metadata?.success !== false ? 'business' : 'health',
            {
                runId,
                stepId,
                duration,
                ...metadata,
            }
        );

        // Emit event
        await this.emitEvent('run.step.completed', {
            runId,
            stepId,
            duration,
            success: metadata?.success !== false,
            metadata,
            timestamp: new Date(),
        });
    }

    /**
     * Record resource usage
     */
    async recordResourceUsage(
        runId: string,
        stepId: string,
        resources: ResourceUsage
    ): Promise<void> {
        // Update run metrics
        const runMetrics = this.runMetrics.get(runId);
        if (runMetrics) {
            runMetrics.resourceUsage.credits += resources.credits;
            runMetrics.resourceUsage.tokens += resources.tokens;
            runMetrics.resourceUsage.toolCalls += resources.toolCalls;
            
            if (resources.memory) {
                runMetrics.resourceUsage.memory = (runMetrics.resourceUsage.memory || 0) + resources.memory;
            }
            if (resources.cpu) {
                runMetrics.resourceUsage.cpu = (runMetrics.resourceUsage.cpu || 0) + resources.cpu;
            }
        }

        // Record individual resource metrics
        await super.recordMetric(
            'run.resource.credits',
            resources.credits,
            'business',
            { runId, stepId }
        );

        await super.recordMetric(
            'run.resource.tokens',
            resources.tokens,
            'performance',
            { runId, stepId }
        );

        await super.recordMetric(
            'run.resource.tool_calls',
            resources.toolCalls,
            'performance',
            { runId, stepId }
        );

        if (resources.memory) {
            await super.recordMetric(
                'run.resource.memory',
                resources.memory,
                'performance',
                { runId, stepId }
            );
        }

        if (resources.cpu) {
            await super.recordMetric(
                'run.resource.cpu',
                resources.cpu,
                'performance',
                { runId, stepId }
            );
        }

        // Emit event
        await this.emitEvent('run.resources.used', {
            runId,
            stepId,
            resources,
            timestamp: new Date(),
        });
    }

    /**
     * Get metrics for a run
     */
    getRunMetrics(runId: string): RunMetrics | undefined {
        const metrics = this.runMetrics.get(runId);
        if (metrics && !metrics.endTime) {
            // Calculate current duration
            metrics.totalDuration = Date.now() - metrics.startTime.getTime();
        }
        return metrics;
    }

    /**
     * Mark run as completed
     */
    async completeRun(runId: string): Promise<void> {
        const metrics = this.runMetrics.get(runId);
        if (!metrics) {
            return;
        }

        metrics.endTime = new Date();
        metrics.totalDuration = metrics.endTime.getTime() - metrics.startTime.getTime();

        // Record run completion metrics
        await super.recordMetric(
            'run.completed',
            1,
            'business',
            {
                runId,
                duration: metrics.totalDuration,
                stepCount: metrics.stepCount,
                successRate: metrics.stepCount > 0 ? metrics.successfulSteps / metrics.stepCount : 0,
                totalResources: metrics.resourceUsage,
            }
        );

        await super.recordMetric(
            'run.duration',
            metrics.totalDuration,
            'performance',
            { runId }
        );

        await super.recordMetric(
            'run.success_rate',
            metrics.stepCount > 0 ? metrics.successfulSteps / metrics.stepCount : 0,
            'business',
            { runId }
        );

        // Emit completion event
        await this.emitEvent('run.completed', {
            runId,
            metrics,
            timestamp: new Date(),
        });
    }

    /**
     * Request performance analysis
     */
    async requestPerformanceAnalysis(runId: string, context?: any): Promise<void> {
        const metrics = this.getRunMetrics(runId);
        if (!metrics) {
            this.logger?.warn(`[PerformanceMonitor] No metrics found for run ${runId}`);
            return;
        }

        const request: PerformanceAnalysisRequest = {
            runId,
            metrics,
            context: context || {},
        };

        // Emit event to trigger analysis
        await this.emitEvent('run.performance.analyze', {
            request,
            timestamp: new Date(),
        });

        // Record analysis request
        await super.recordMetric(
            'run.analysis_requested',
            1,
            'business',
            { runId }
        );
    }

    /**
     * Handle incoming events
     */
    protected async handleEvent(channel: string, event: any): Promise<void> {
        switch (channel) {
            case 'run.performance':
                // Handle performance events from runs
                if (event.type === 'step_completed') {
                    await this.recordStepTiming(
                        event.runId,
                        event.stepId,
                        event.duration,
                        event.metadata
                    );
                } else if (event.type === 'resource_used') {
                    await this.recordResourceUsage(
                        event.runId,
                        event.stepId,
                        event.resources
                    );
                }
                break;

            case 'run.metrics':
                // Handle metric events
                if (event.type === 'custom_metric') {
                    await this.recordRunMetric(
                        event.runId,
                        event.metric,
                        event.value,
                        event.stepId,
                        event.metadata
                    );
                }
                break;
        }
    }

    /**
     * Clear metrics for a run (cleanup)
     */
    clearRunMetrics(runId: string): void {
        this.runMetrics.delete(runId);
    }

    /**
     * Get all active runs
     */
    getActiveRuns(): string[] {
        return Array.from(this.runMetrics.keys()).filter(runId => {
            const metrics = this.runMetrics.get(runId);
            return metrics && !metrics.endTime;
        });
    }

    /**
     * Calculate resource efficiency
     */
    async calculateResourceEfficiency(runId: string): Promise<number> {
        const metrics = this.getRunMetrics(runId);
        if (!metrics || metrics.stepCount === 0) {
            return 0;
        }

        const successRate = metrics.successfulSteps / metrics.stepCount;
        const avgCreditsPerStep = metrics.resourceUsage.credits / metrics.stepCount;
        
        // Efficiency = success rate / normalized resource usage
        const efficiency = successRate / Math.max(1, avgCreditsPerStep / 100);
        
        await super.recordMetric(
            'run.resource_efficiency',
            efficiency,
            'performance',
            { runId }
        );

        return efficiency;
    }
}