/**
 * ResourceMetricsAdapter - Adapter for cross-cutting ResourceMetrics
 * 
 * Maintains backward compatibility with the original ResourceMetrics
 * while routing all operations through UnifiedMonitoringService.
 */

import { BaseMonitoringAdapter } from "./BaseMonitoringAdapter.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type Logger } from "winston";

/**
 * Efficiency metrics structure (from original ResourceMetrics)
 */
export interface EfficiencyMetrics {
    utilizationRate: number;
    wasteRate: number;
    optimizationScore: number;
    costPerOutcome: number;
}

/**
 * Resource health metrics
 */
export interface ResourceHealthMetrics {
    availability: number;
    performance: number;
    errorRate: number;
    congestion: number;
}

/**
 * Cost analysis structure
 */
export interface CostAnalysis {
    totalCost: number;
    breakdown: {
        llmTokens: number;
        toolCalls: number;
        storage: number;
        compute: number;
    };
    trends: {
        hourly: number[];
        daily: number[];
        projectedMonthly: number;
    };
    optimizationPotential: number;
}

/**
 * Bottleneck analysis structure
 */
export interface BottleneckAnalysis {
    resource: string;
    severity: 'low' | 'medium' | 'high' | 'critical';
    impact: {
        throughputReduction: number;
        additionalLatency: number;
        affectedOperations: string[];
    };
    recommendations: string[];
}

/**
 * Resource execution record
 */
interface ExecutionRecord {
    timestamp: Date;
    usage: {
        credits: number;
        tokens: number;
        toolCalls: number;
        memory?: number;
        cpu?: number;
    };
    duration: number;
    success: boolean;
    operation?: string;
}

/**
 * ResourceMetricsAdapter - Routes cross-cutting resource metrics through unified monitoring
 */
export class ResourceMetricsAdapter extends BaseMonitoringAdapter {
    private executionHistory: ExecutionRecord[] = [];
    private readonly maxHistorySize: number = 10000;

    constructor(
        eventBus: EventBus,
        logger: Logger
    ) {
        super(
            {
                componentName: "ResourceMetrics",
                tier: "cross-cutting",
                eventChannels: ["resource.usage", "resource.health", "resource.allocation"],
                enableEventBus: true,
            },
            logger,
            eventBus
        );
    }

    /**
     * Record a resource metric
     */
    async recordMetric(
        name: string,
        value: number,
        unit?: string,
        metadata?: Record<string, any>
    ): Promise<void> {
        await super.recordMetric(
            `resource.${name}`,
            value,
            this.getMetricType(name),
            {
                unit,
                ...metadata,
            }
        );

        // Emit event for monitoring
        await this.emitEvent('resource.metric.recorded', {
            name,
            value,
            unit,
            metadata,
            timestamp: new Date(),
        });
    }

    /**
     * Record efficiency metrics
     */
    async recordEfficiency(efficiency: EfficiencyMetrics): Promise<void> {
        await super.recordMetric(
            'resource.utilization_rate',
            efficiency.utilizationRate,
            'performance',
            { type: 'efficiency' }
        );

        await super.recordMetric(
            'resource.waste_rate',
            efficiency.wasteRate,
            'performance',
            { type: 'efficiency' }
        );

        await super.recordMetric(
            'resource.optimization_score',
            efficiency.optimizationScore,
            'business',
            { type: 'efficiency' }
        );

        await super.recordMetric(
            'resource.cost_per_outcome',
            efficiency.costPerOutcome,
            'business',
            { type: 'efficiency' }
        );

        // Emit efficiency update
        await this.emitEvent('resource.efficiency.updated', {
            efficiency,
            timestamp: new Date(),
        });
    }

    /**
     * Record health metrics
     */
    async recordHealth(health: ResourceHealthMetrics): Promise<void> {
        await super.recordMetric(
            'resource.availability',
            health.availability,
            'health',
            { type: 'health' }
        );

        await super.recordMetric(
            'resource.performance',
            health.performance,
            'performance',
            { type: 'health' }
        );

        await super.recordMetric(
            'resource.error_rate',
            health.errorRate,
            'health',
            { type: 'health' }
        );

        await super.recordMetric(
            'resource.congestion',
            health.congestion,
            'performance',
            { type: 'health' }
        );

        // Check for health issues
        if (health.availability < 0.95 || health.errorRate > 0.05) {
            await this.emitEvent('resource.health.degraded', {
                health,
                timestamp: new Date(),
            });
        }
    }

    /**
     * Record execution
     */
    async recordExecution(
        usage: ExecutionRecord['usage'],
        duration: number,
        success: boolean,
        operation?: string
    ): Promise<void> {
        const record: ExecutionRecord = {
            timestamp: new Date(),
            usage,
            duration,
            success,
            operation,
        };

        // Store in history
        this.executionHistory.push(record);
        if (this.executionHistory.length > this.maxHistorySize) {
            this.executionHistory.shift();
        }

        // Record usage metrics
        await super.recordMetric(
            'resource.execution.credits',
            usage.credits,
            'business',
            { operation, success }
        );

        await super.recordMetric(
            'resource.execution.tokens',
            usage.tokens,
            'performance',
            { operation, success }
        );

        await super.recordMetric(
            'resource.execution.tool_calls',
            usage.toolCalls,
            'performance',
            { operation, success }
        );

        await super.recordMetric(
            'resource.execution.duration',
            duration,
            'performance',
            { operation, success }
        );

        // Record success/failure
        await super.recordMetric(
            success ? 'resource.execution.success' : 'resource.execution.failure',
            1,
            success ? 'business' : 'health',
            { operation, usage }
        );
    }

    /**
     * Get cost analysis
     */
    getCostAnalysis(): CostAnalysis {
        const now = Date.now();
        const hourAgo = now - 3600000;
        const dayAgo = now - 86400000;

        // Calculate costs from execution history
        const totalCost = this.executionHistory.reduce((sum, record) => {
            // Simple cost model: 1 credit = $0.01, 1000 tokens = $0.02, 1 tool call = $0.05
            const creditCost = record.usage.credits * 0.01;
            const tokenCost = (record.usage.tokens / 1000) * 0.02;
            const toolCost = record.usage.toolCalls * 0.05;
            return sum + creditCost + tokenCost + toolCost;
        }, 0);

        // Calculate breakdown
        const breakdown = this.executionHistory.reduce(
            (acc, record) => {
                acc.llmTokens += (record.usage.tokens / 1000) * 0.02;
                acc.toolCalls += record.usage.toolCalls * 0.05;
                acc.storage += 0; // Not tracked in this simple model
                acc.compute += record.usage.credits * 0.01;
                return acc;
            },
            { llmTokens: 0, toolCalls: 0, storage: 0, compute: 0 }
        );

        // Calculate hourly trend (last 24 hours)
        const hourlyTrend: number[] = [];
        for (let i = 23; i >= 0; i--) {
            const hourStart = now - (i + 1) * 3600000;
            const hourEnd = now - i * 3600000;
            const hourCost = this.calculateCostInRange(hourStart, hourEnd);
            hourlyTrend.push(hourCost);
        }

        // Calculate daily trend (last 7 days)
        const dailyTrend: number[] = [];
        for (let i = 6; i >= 0; i--) {
            const dayStart = now - (i + 1) * 86400000;
            const dayEnd = now - i * 86400000;
            const dayCost = this.calculateCostInRange(dayStart, dayEnd);
            dailyTrend.push(dayCost);
        }

        // Project monthly cost
        const avgDailyCost = dailyTrend.reduce((a, b) => a + b, 0) / dailyTrend.length;
        const projectedMonthly = avgDailyCost * 30;

        // Calculate optimization potential
        const wastedResources = this.executionHistory.filter(r => !r.success).length;
        const optimizationPotential = (wastedResources / Math.max(1, this.executionHistory.length)) * totalCost;

        return {
            totalCost,
            breakdown,
            trends: {
                hourly: hourlyTrend,
                daily: dailyTrend,
                projectedMonthly,
            },
            optimizationPotential,
        };
    }

    /**
     * Detect bottlenecks
     */
    detectBottlenecks(): BottleneckAnalysis[] {
        const bottlenecks: BottleneckAnalysis[] = [];

        // Analyze recent executions
        const recentExecutions = this.executionHistory.slice(-100);
        
        // Check for high failure rate
        const failureRate = recentExecutions.filter(e => !e.success).length / recentExecutions.length;
        if (failureRate > 0.2) {
            bottlenecks.push({
                resource: 'execution_reliability',
                severity: failureRate > 0.5 ? 'critical' : failureRate > 0.3 ? 'high' : 'medium',
                impact: {
                    throughputReduction: failureRate,
                    additionalLatency: 0,
                    affectedOperations: [...new Set(recentExecutions.map(e => e.operation || 'unknown'))],
                },
                recommendations: [
                    'Investigate root cause of failures',
                    'Implement retry mechanisms',
                    'Review resource allocation strategies',
                ],
            });
        }

        // Check for resource exhaustion
        const avgCreditsPerExec = recentExecutions.reduce((sum, e) => sum + e.usage.credits, 0) / recentExecutions.length;
        if (avgCreditsPerExec > 1000) {
            bottlenecks.push({
                resource: 'credit_usage',
                severity: avgCreditsPerExec > 5000 ? 'critical' : avgCreditsPerExec > 2000 ? 'high' : 'medium',
                impact: {
                    throughputReduction: 0.3,
                    additionalLatency: 0,
                    affectedOperations: ['all'],
                },
                recommendations: [
                    'Optimize credit allocation',
                    'Implement credit budgeting',
                    'Consider caching strategies',
                ],
            });
        }

        // Check for performance degradation
        const avgDuration = recentExecutions.reduce((sum, e) => sum + e.duration, 0) / recentExecutions.length;
        if (avgDuration > 30000) { // 30 seconds
            bottlenecks.push({
                resource: 'execution_time',
                severity: avgDuration > 60000 ? 'high' : 'medium',
                impact: {
                    throughputReduction: 0.5,
                    additionalLatency: avgDuration - 10000,
                    affectedOperations: ['all'],
                },
                recommendations: [
                    'Profile slow operations',
                    'Implement timeouts',
                    'Consider parallel execution',
                ],
            });
        }

        return bottlenecks;
    }

    /**
     * Handle incoming events
     */
    protected async handleEvent(channel: string, event: any): Promise<void> {
        switch (channel) {
            case 'resource.usage':
                if (event.type === 'resource_consumed') {
                    await this.recordExecution(
                        event.usage,
                        event.duration || 0,
                        event.success !== false,
                        event.operation
                    );
                }
                break;

            case 'resource.health':
                if (event.type === 'health_check') {
                    await this.recordHealth(event.health);
                }
                break;

            case 'resource.allocation':
                if (event.type === 'efficiency_report') {
                    await this.recordEfficiency(event.efficiency);
                }
                break;
        }
    }

    /**
     * Calculate cost in time range
     */
    private calculateCostInRange(startTime: number, endTime: number): number {
        return this.executionHistory
            .filter(record => {
                const time = record.timestamp.getTime();
                return time >= startTime && time <= endTime;
            })
            .reduce((sum, record) => {
                const creditCost = record.usage.credits * 0.01;
                const tokenCost = (record.usage.tokens / 1000) * 0.02;
                const toolCost = record.usage.toolCalls * 0.05;
                return sum + creditCost + tokenCost + toolCost;
            }, 0);
    }

    /**
     * Get resource utilization summary
     */
    getUtilizationSummary(): any {
        const recentExecutions = this.executionHistory.slice(-100);
        
        if (recentExecutions.length === 0) {
            return {
                utilizationRate: 0,
                efficiency: 0,
                averageLoad: 0,
            };
        }

        const totalDuration = recentExecutions.reduce((sum, e) => sum + e.duration, 0);
        const totalTime = Date.now() - recentExecutions[0].timestamp.getTime();
        const utilizationRate = totalDuration / totalTime;

        const successRate = recentExecutions.filter(e => e.success).length / recentExecutions.length;
        const avgCredits = recentExecutions.reduce((sum, e) => sum + e.usage.credits, 0) / recentExecutions.length;
        const efficiency = successRate / Math.max(1, avgCredits / 100);

        return {
            utilizationRate,
            efficiency,
            averageLoad: avgCredits,
            successRate,
            totalExecutions: recentExecutions.length,
        };
    }
}