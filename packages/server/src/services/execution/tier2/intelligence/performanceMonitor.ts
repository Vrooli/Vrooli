import { type Logger } from "winston";
import {
    type PerformanceAnalysis,
    type OptimizationSuggestion,
    type RunEventType,
    RunEventType as RunEventTypeEnum,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/eventBus.js";

/**
 * Performance metric
 */
export interface PerformanceMetric {
    runId: string;
    stepId: string;
    metric: string;
    value: number;
    timestamp: Date;
    metadata?: Record<string, unknown>;
}

/**
 * Step performance data
 */
export interface StepPerformance {
    stepId: string;
    executionCount: number;
    totalDuration: number;
    averageDuration: number;
    minDuration: number;
    maxDuration: number;
    failureRate: number;
    resourceUsage: {
        avgCredits: number;
        avgTokens: number;
        avgMemory: number;
    };
}

/**
 * PerformanceMonitor - Analyzes execution performance and suggests optimizations
 * 
 * This component provides real-time performance monitoring and optimization
 * suggestions for workflow execution. It tracks:
 * 
 * - Step execution times and resource usage
 * - Bottleneck identification
 * - Failure patterns and recovery times
 * - Resource utilization trends
 * - Optimization opportunities
 * 
 * The monitor uses statistical analysis to identify performance issues and
 * provide actionable suggestions for improvement.
 */
export class PerformanceMonitor {
    private readonly eventBus: EventBus;
    private readonly logger: Logger;
    private readonly metricsBuffer: Map<string, PerformanceMetric[]> = new Map();
    private readonly stepPerformance: Map<string, StepPerformance> = new Map();
    private readonly analysisInterval: number = 60000; // 1 minute
    private analysisTimer?: NodeJS.Timer;

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.startAnalysisLoop();
        this.subscribeToEvents();
    }

    /**
     * Records a performance metric
     */
    async recordMetric(metric: PerformanceMetric): Promise<void> {
        const key = `${metric.runId}:${metric.stepId}`;
        
        if (!this.metricsBuffer.has(key)) {
            this.metricsBuffer.set(key, []);
        }
        
        this.metricsBuffer.get(key)!.push(metric);

        // Update step performance
        await this.updateStepPerformance(metric);
    }

    /**
     * Analyzes performance for a run
     */
    async analyzeRun(runId: string): Promise<PerformanceAnalysis> {
        const bottlenecks = await this.identifyBottlenecks(runId);
        const suggestions = await this.generateOptimizationSuggestions(runId, bottlenecks);
        const efficiency = await this.calculateEfficiency(runId);

        const analysis: PerformanceAnalysis = {
            bottlenecks,
            suggestions,
            overallEfficiency: efficiency,
        };

        this.logger.info("[PerformanceMonitor] Analysis complete", {
            runId,
            bottleneckCount: bottlenecks.length,
            suggestionCount: suggestions.length,
            efficiency,
        });

        return analysis;
    }

    /**
     * Identifies performance bottlenecks
     */
    private async identifyBottlenecks(runId: string): Promise<Array<{
        stepId: string;
        duration: number;
        resourceUsage: Record<string, number>;
    }>> {
        const bottlenecks = [];
        const stepMetrics = new Map<string, PerformanceMetric[]>();

        // Group metrics by step
        for (const [key, metrics] of this.metricsBuffer) {
            if (key.startsWith(runId)) {
                const stepId = key.split(":")[1];
                stepMetrics.set(stepId, metrics);
            }
        }

        // Calculate statistics for each step
        for (const [stepId, metrics] of stepMetrics) {
            const durations = metrics
                .filter(m => m.metric === "duration")
                .map(m => m.value);

            if (durations.length === 0) continue;

            const avgDuration = durations.reduce((a, b) => a + b, 0) / durations.length;
            const stdDev = Math.sqrt(
                durations.reduce((sum, d) => sum + Math.pow(d - avgDuration, 2), 0) / durations.length,
            );

            // Identify bottlenecks (steps with high duration or high variance)
            if (avgDuration > 5000 || stdDev > avgDuration * 0.5) {
                const resourceMetrics = metrics.filter(m => m.metric.startsWith("resource."));
                const resourceUsage: Record<string, number> = {};

                for (const metric of resourceMetrics) {
                    const resourceType = metric.metric.split(".")[1];
                    resourceUsage[resourceType] = metric.value;
                }

                bottlenecks.push({
                    stepId,
                    duration: avgDuration,
                    resourceUsage,
                });
            }
        }

        // Sort by duration (longest first)
        bottlenecks.sort((a, b) => b.duration - a.duration);

        return bottlenecks;
    }

    /**
     * Generates optimization suggestions
     */
    private async generateOptimizationSuggestions(
        runId: string,
        bottlenecks: Array<{ stepId: string; duration: number; resourceUsage: Record<string, number> }>,
    ): Promise<OptimizationSuggestion[]> {
        const suggestions: OptimizationSuggestion[] = [];

        for (const bottleneck of bottlenecks) {
            const perf = this.stepPerformance.get(bottleneck.stepId);
            if (!perf) continue;

            // Suggest parallelization for independent steps
            if (bottleneck.duration > 10000 && !bottleneck.resourceUsage.parallel) {
                suggestions.push({
                    type: "parallelize",
                    targetSteps: [bottleneck.stepId],
                    expectedImprovement: 0.5, // 50% improvement
                    rationale: "Step takes significant time and could be parallelized",
                    risk: "low",
                });
            }

            // Suggest caching for deterministic steps
            if (perf.executionCount > 5 && perf.failureRate < 0.1) {
                suggestions.push({
                    type: "cache",
                    targetSteps: [bottleneck.stepId],
                    expectedImprovement: 0.8, // 80% improvement on cache hits
                    rationale: "Step is executed frequently with low failure rate",
                    risk: "low",
                });
            }

            // Suggest skipping for optional steps with high failure rate
            if (perf.failureRate > 0.5) {
                suggestions.push({
                    type: "skip",
                    targetSteps: [bottleneck.stepId],
                    expectedImprovement: 1.0, // 100% time saved
                    rationale: "Step has high failure rate and may be optional",
                    risk: "medium",
                });
            }

            // Suggest reordering for steps with dependencies
            if (bottleneck.resourceUsage.waitTime > bottleneck.duration * 0.3) {
                suggestions.push({
                    type: "reorder",
                    targetSteps: [bottleneck.stepId],
                    expectedImprovement: 0.3, // 30% improvement
                    rationale: "Step spends significant time waiting for resources",
                    risk: "medium",
                });
            }
        }

        // Sort by expected improvement (highest first)
        suggestions.sort((a, b) => b.expectedImprovement - a.expectedImprovement);

        return suggestions;
    }

    /**
     * Calculates overall efficiency
     */
    private async calculateEfficiency(runId: string): Promise<number> {
        let totalActiveTime = 0;
        let totalWaitTime = 0;
        let totalSteps = 0;

        for (const [key, metrics] of this.metricsBuffer) {
            if (!key.startsWith(runId)) continue;

            const activeTime = metrics
                .filter(m => m.metric === "active_time")
                .reduce((sum, m) => sum + m.value, 0);

            const waitTime = metrics
                .filter(m => m.metric === "wait_time")
                .reduce((sum, m) => sum + m.value, 0);

            totalActiveTime += activeTime;
            totalWaitTime += waitTime;
            totalSteps++;
        }

        if (totalActiveTime + totalWaitTime === 0) {
            return 1.0;
        }

        return totalActiveTime / (totalActiveTime + totalWaitTime);
    }

    /**
     * Updates step performance statistics
     */
    private async updateStepPerformance(metric: PerformanceMetric): Promise<void> {
        const { stepId } = metric;
        
        if (!this.stepPerformance.has(stepId)) {
            this.stepPerformance.set(stepId, {
                stepId,
                executionCount: 0,
                totalDuration: 0,
                averageDuration: 0,
                minDuration: Infinity,
                maxDuration: 0,
                failureRate: 0,
                resourceUsage: {
                    avgCredits: 0,
                    avgTokens: 0,
                    avgMemory: 0,
                },
            });
        }

        const perf = this.stepPerformance.get(stepId)!;

        switch (metric.metric) {
            case "duration":
                perf.executionCount++;
                perf.totalDuration += metric.value;
                perf.averageDuration = perf.totalDuration / perf.executionCount;
                perf.minDuration = Math.min(perf.minDuration, metric.value);
                perf.maxDuration = Math.max(perf.maxDuration, metric.value);
                break;

            case "failure":
                if (metric.value > 0) {
                    perf.failureRate = ((perf.failureRate * (perf.executionCount - 1)) + 1) / perf.executionCount;
                }
                break;

            case "resource.credits":
                perf.resourceUsage.avgCredits = 
                    ((perf.resourceUsage.avgCredits * (perf.executionCount - 1)) + metric.value) / perf.executionCount;
                break;

            case "resource.tokens":
                perf.resourceUsage.avgTokens = 
                    ((perf.resourceUsage.avgTokens * (perf.executionCount - 1)) + metric.value) / perf.executionCount;
                break;

            case "resource.memory":
                perf.resourceUsage.avgMemory = 
                    ((perf.resourceUsage.avgMemory * (perf.executionCount - 1)) + metric.value) / perf.executionCount;
                break;
        }
    }

    /**
     * Starts the analysis loop
     */
    private startAnalysisLoop(): void {
        this.analysisTimer = setInterval(async () => {
            await this.performPeriodicAnalysis();
        }, this.analysisInterval);
    }

    /**
     * Performs periodic analysis
     */
    private async performPeriodicAnalysis(): Promise<void> {
        // Clean up old metrics
        const cutoffTime = Date.now() - 3600000; // 1 hour
        
        for (const [key, metrics] of this.metricsBuffer) {
            const filtered = metrics.filter(m => m.timestamp.getTime() > cutoffTime);
            
            if (filtered.length === 0) {
                this.metricsBuffer.delete(key);
            } else if (filtered.length < metrics.length) {
                this.metricsBuffer.set(key, filtered);
            }
        }

        // Emit performance summaries
        for (const [stepId, perf] of this.stepPerformance) {
            if (perf.executionCount > 0) {
                await this.eventBus.publish("performance.summary", {
                    stepId,
                    performance: perf,
                    timestamp: new Date(),
                });
            }
        }
    }

    /**
     * Subscribes to relevant events
     */
    private subscribeToEvents(): void {
        // Subscribe to step completion events
        this.eventBus.subscribe("run.events", async (event) => {
            if (event.type === RunEventTypeEnum.STEP_COMPLETED) {
                const duration = event.metadata?.duration;
                if (duration) {
                    await this.recordMetric({
                        runId: event.runId,
                        stepId: event.stepId!,
                        metric: "duration",
                        value: duration,
                        timestamp: event.timestamp,
                    });
                }
            } else if (event.type === RunEventTypeEnum.STEP_FAILED) {
                await this.recordMetric({
                    runId: event.runId,
                    stepId: event.stepId!,
                    metric: "failure",
                    value: 1,
                    timestamp: event.timestamp,
                });
            }
        });

        // Subscribe to resource usage events
        this.eventBus.subscribe("telemetry.resources", async (event) => {
            if (event.runId && event.stepId) {
                await this.recordMetric({
                    runId: event.runId,
                    stepId: event.stepId,
                    metric: `resource.${event.resourceType}`,
                    value: event.value,
                    timestamp: new Date(),
                    metadata: event.metadata,
                });
            }
        });
    }

    /**
     * Stops the performance monitor
     */
    async stop(): Promise<void> {
        if (this.analysisTimer) {
            clearInterval(this.analysisTimer);
            this.analysisTimer = undefined;
        }

        this.logger.info("[PerformanceMonitor] Stopped");
    }

    /**
     * Gets performance report for a step
     */
    async getStepPerformanceReport(stepId: string): Promise<StepPerformance | null> {
        return this.stepPerformance.get(stepId) || null;
    }

    /**
     * Exports performance data
     */
    async exportPerformanceData(): Promise<{
        metrics: PerformanceMetric[];
        stepPerformance: StepPerformance[];
    }> {
        const allMetrics: PerformanceMetric[] = [];
        for (const metrics of this.metricsBuffer.values()) {
            allMetrics.push(...metrics);
        }

        return {
            metrics: allMetrics,
            stepPerformance: Array.from(this.stepPerformance.values()),
        };
    }
}
