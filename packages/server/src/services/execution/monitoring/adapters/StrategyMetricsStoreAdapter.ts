/**
 * StrategyMetricsStoreAdapter - Adapter for Tier 3 StrategyMetricsStore
 * 
 * Maintains backward compatibility with the original StrategyMetricsStore
 * while routing all operations through UnifiedMonitoringService.
 */

import { BaseMonitoringAdapter } from "./BaseMonitoringAdapter.js";
import { type Logger } from "winston";

/**
 * Execution record structure (from original StrategyMetricsStore)
 */
export interface ExecutionRecord {
    timestamp: Date;
    stepType: string;
    strategy: string;
    executionTime: number;
    success: boolean;
    resourceUsage: {
        llmTokens: number;
        toolCalls: number;
        totalCost: number;
    };
    inputComplexity: number;
    outputQuality: number;
    errorType?: string;
    metadata?: Record<string, any>;
}

/**
 * Aggregated metrics structure
 */
export interface AggregatedMetrics {
    totalExecutions: number;
    successRate: number;
    averageExecutionTime: number;
    totalResourceUsage: {
        llmTokens: number;
        toolCalls: number;
        totalCost: number;
    };
    byStepType: Record<string, {
        count: number;
        successRate: number;
        avgExecutionTime: number;
        avgResourceUsage: {
            llmTokens: number;
            toolCalls: number;
            totalCost: number;
        };
    }>;
    byStrategy: Record<string, {
        count: number;
        successRate: number;
        avgExecutionTime: number;
        avgResourceUsage: {
            llmTokens: number;
            toolCalls: number;
            totalCost: number;
        };
    }>;
    trends: {
        executionTimeMovingAvg: number[];
        successRateMovingAvg: number[];
        resourceUsageMovingAvg: number[];
    };
}

/**
 * StrategyMetricsStoreAdapter - Routes Tier 3 strategy metrics through unified monitoring
 */
export class StrategyMetricsStoreAdapter extends BaseMonitoringAdapter {
    private readonly maxRecords: number;
    private readonly windowSize: number;
    private records: ExecutionRecord[] = [];

    constructor(
        maxRecords: number = 10000,
        windowSize: number = 100,
        logger?: Logger
    ) {
        super(
            {
                componentName: "StrategyMetricsStore",
                tier: 3,
                enableEventBus: false, // StrategyMetricsStore doesn't use event bus
            },
            logger
        );
        
        this.maxRecords = maxRecords;
        this.windowSize = windowSize;
    }

    /**
     * Record an execution
     */
    async recordExecution(record: ExecutionRecord): Promise<void> {
        // Store locally for immediate access
        this.records.push(record);
        if (this.records.length > this.maxRecords) {
            this.records.shift();
        }

        // Record to unified service
        await this.recordMetric(
            `strategy.${record.strategy}.${record.stepType}`,
            record.executionTime,
            'performance',
            {
                success: record.success,
                resourceUsage: record.resourceUsage,
                inputComplexity: record.inputComplexity,
                outputQuality: record.outputQuality,
                errorType: record.errorType,
                timestamp: record.timestamp,
                ...record.metadata,
            }
        );

        // Record resource usage metrics
        await this.recordMetric(
            `resource.${record.strategy}.llm_tokens`,
            record.resourceUsage.llmTokens,
            'performance',
            { stepType: record.stepType }
        );

        await this.recordMetric(
            `resource.${record.strategy}.tool_calls`,
            record.resourceUsage.toolCalls,
            'performance',
            { stepType: record.stepType }
        );

        await this.recordMetric(
            `cost.${record.strategy}.total`,
            record.resourceUsage.totalCost,
            'business',
            { stepType: record.stepType }
        );

        // Record success/failure
        await this.recordMetric(
            record.success ? 'strategy.success' : 'strategy.failure',
            1,
            record.success ? 'business' : 'health',
            {
                strategy: record.strategy,
                stepType: record.stepType,
                errorType: record.errorType,
            }
        );

        // Record quality metrics
        await this.recordMetric(
            `quality.${record.strategy}.output`,
            record.outputQuality,
            'business',
            {
                stepType: record.stepType,
                inputComplexity: record.inputComplexity,
            }
        );
    }

    /**
     * Get aggregated metrics
     */
    getAggregatedMetrics(): AggregatedMetrics {
        if (this.records.length === 0) {
            return this.getEmptyMetrics();
        }

        const byStepType: Record<string, any> = {};
        const byStrategy: Record<string, any> = {};

        // Aggregate by step type and strategy
        for (const record of this.records) {
            this.updateAggregation(byStepType, record.stepType, record);
            this.updateAggregation(byStrategy, record.strategy, record);
        }

        // Calculate final metrics
        for (const key of Object.keys(byStepType)) {
            byStepType[key] = this.finalizeAggregation(byStepType[key]);
        }
        for (const key of Object.keys(byStrategy)) {
            byStrategy[key] = this.finalizeAggregation(byStrategy[key]);
        }

        // Calculate trends
        const trends = this.calculateTrends();

        // Calculate totals
        const totalResourceUsage = this.records.reduce(
            (acc, record) => ({
                llmTokens: acc.llmTokens + record.resourceUsage.llmTokens,
                toolCalls: acc.toolCalls + record.resourceUsage.toolCalls,
                totalCost: acc.totalCost + record.resourceUsage.totalCost,
            }),
            { llmTokens: 0, toolCalls: 0, totalCost: 0 }
        );

        const successfulExecutions = this.records.filter(r => r.success).length;
        const totalExecutionTime = this.records.reduce((sum, r) => sum + r.executionTime, 0);

        return {
            totalExecutions: this.records.length,
            successRate: successfulExecutions / this.records.length,
            averageExecutionTime: totalExecutionTime / this.records.length,
            totalResourceUsage,
            byStepType,
            byStrategy,
            trends,
        };
    }

    /**
     * Get executions by step type
     */
    getExecutionsByStepType(stepType: string): ExecutionRecord[] {
        return this.records.filter(record => record.stepType === stepType);
    }

    /**
     * Get recent executions
     */
    getRecentExecutions(count: number = 100): ExecutionRecord[] {
        return this.records.slice(-count);
    }

    /**
     * Export metrics for persistence
     */
    exportMetrics(): string {
        const data = {
            records: this.records,
            timestamp: new Date(),
            version: '1.0.0',
        };
        return JSON.stringify(data);
    }

    /**
     * Import metrics from persistence
     */
    importMetrics(data: string): void {
        try {
            const parsed = JSON.parse(data);
            if (parsed.records && Array.isArray(parsed.records)) {
                this.records = parsed.records.map((record: any) => ({
                    ...record,
                    timestamp: new Date(record.timestamp),
                }));
            }
        } catch (error) {
            this.logger?.error('[StrategyMetricsStoreAdapter] Failed to import metrics:', error);
        }
    }

    /**
     * Clear all metrics
     */
    clear(): void {
        this.records = [];
    }

    /**
     * Handle events (not used by StrategyMetricsStore)
     */
    protected async handleEvent(channel: string, event: any): Promise<void> {
        // StrategyMetricsStore doesn't subscribe to events
    }

    /**
     * Private helper methods
     */

    private getEmptyMetrics(): AggregatedMetrics {
        return {
            totalExecutions: 0,
            successRate: 0,
            averageExecutionTime: 0,
            totalResourceUsage: {
                llmTokens: 0,
                toolCalls: 0,
                totalCost: 0,
            },
            byStepType: {},
            byStrategy: {},
            trends: {
                executionTimeMovingAvg: [],
                successRateMovingAvg: [],
                resourceUsageMovingAvg: [],
            },
        };
    }

    private updateAggregation(
        aggregation: Record<string, any>,
        key: string,
        record: ExecutionRecord
    ): void {
        if (!aggregation[key]) {
            aggregation[key] = {
                records: [],
                successCount: 0,
                totalExecutionTime: 0,
                totalResourceUsage: {
                    llmTokens: 0,
                    toolCalls: 0,
                    totalCost: 0,
                },
            };
        }

        const agg = aggregation[key];
        agg.records.push(record);
        if (record.success) {
            agg.successCount++;
        }
        agg.totalExecutionTime += record.executionTime;
        agg.totalResourceUsage.llmTokens += record.resourceUsage.llmTokens;
        agg.totalResourceUsage.toolCalls += record.resourceUsage.toolCalls;
        agg.totalResourceUsage.totalCost += record.resourceUsage.totalCost;
    }

    private finalizeAggregation(agg: any): any {
        const count = agg.records.length;
        return {
            count,
            successRate: agg.successCount / count,
            avgExecutionTime: agg.totalExecutionTime / count,
            avgResourceUsage: {
                llmTokens: agg.totalResourceUsage.llmTokens / count,
                toolCalls: agg.totalResourceUsage.toolCalls / count,
                totalCost: agg.totalResourceUsage.totalCost / count,
            },
        };
    }

    private calculateTrends(): any {
        const windowSize = Math.min(this.windowSize, this.records.length);
        const windows = Math.ceil(this.records.length / windowSize);

        const executionTimeMovingAvg: number[] = [];
        const successRateMovingAvg: number[] = [];
        const resourceUsageMovingAvg: number[] = [];

        for (let i = 0; i < windows; i++) {
            const start = i * windowSize;
            const end = Math.min(start + windowSize, this.records.length);
            const windowRecords = this.records.slice(start, end);

            if (windowRecords.length > 0) {
                const avgExecTime = windowRecords.reduce((sum, r) => sum + r.executionTime, 0) / windowRecords.length;
                const successRate = windowRecords.filter(r => r.success).length / windowRecords.length;
                const avgCost = windowRecords.reduce((sum, r) => sum + r.resourceUsage.totalCost, 0) / windowRecords.length;

                executionTimeMovingAvg.push(avgExecTime);
                successRateMovingAvg.push(successRate);
                resourceUsageMovingAvg.push(avgCost);
            }
        }

        return {
            executionTimeMovingAvg,
            successRateMovingAvg,
            resourceUsageMovingAvg,
        };
    }
}