/**
 * Strategy Metrics Store
 * 
 * In-memory store for tracking strategy execution metrics.
 * Designed to be simple and fast, with optional Redis backing for persistence.
 */

import { type Logger } from "winston";

/**
 * Individual execution record
 */
export interface ExecutionRecord {
    stepId: string;
    stepType: string;
    success: boolean;
    executionTime: number;
    tokensUsed: number;
    apiCalls: number;
    cost: number;
    confidence: number;
    timestamp: Date;
    error?: string;
    context: {
        inputSize: number;
        constraintsCount: number;
        historySize: number;
    };
}

/**
 * Aggregated metrics
 */
export interface AggregatedMetrics {
    totalExecutions: number;
    successCount: number;
    failureCount: number;
    averageExecutionTime: number;
    averageTokens: number;
    averageApiCalls: number;
    averageCost: number;
    averageConfidence: number;
    
    // Performance trends
    recentSuccessRate: number; // Last 10 executions
    efficiencyTrend: number; // Positive = getting faster, negative = getting slower
    confidenceTrend: number; // Positive = getting more confident
    
    // Context analysis
    stepTypeDistribution: Map<string, number>;
    errorPatterns: Map<string, number>;
    
    // Time-based metrics
    firstExecution?: Date;
    lastExecution?: Date;
}

/**
 * Simple in-memory metrics store for strategy performance tracking
 */
export class StrategyMetricsStore {
    private executions: ExecutionRecord[] = [];
    private readonly maxRecords = 1000; // Keep last 1000 executions
    private readonly logger?: Logger;

    constructor(logger?: Logger) {
        this.logger = logger;
    }

    /**
     * Record a new execution
     */
    recordExecution(record: ExecutionRecord): void {
        this.executions.push(record);

        // Keep only the most recent records
        if (this.executions.length > this.maxRecords) {
            this.executions = this.executions.slice(-this.maxRecords);
        }

        this.logger?.debug("[StrategyMetricsStore] Recorded execution", {
            stepId: record.stepId,
            success: record.success,
            executionTime: record.executionTime,
            totalRecords: this.executions.length,
        });
    }

    /**
     * Get aggregated metrics
     */
    getAggregatedMetrics(): AggregatedMetrics {
        if (this.executions.length === 0) {
            return this.getEmptyMetrics();
        }

        const totalExecutions = this.executions.length;
        const successfulExecutions = this.executions.filter(e => e.success);
        const failedExecutions = this.executions.filter(e => !e.success);

        // Basic counts
        const successCount = successfulExecutions.length;
        const failureCount = failedExecutions.length;

        // Averages (only from successful executions where appropriate)
        const averageExecutionTime = this.calculateAverage(this.executions, 'executionTime');
        const averageTokens = this.calculateAverage(this.executions, 'tokensUsed');
        const averageApiCalls = this.calculateAverage(this.executions, 'apiCalls');
        const averageCost = this.calculateAverage(this.executions, 'cost');
        const averageConfidence = this.calculateAverage(successfulExecutions, 'confidence');

        // Recent performance (last 10 executions)
        const recentExecutions = this.executions.slice(-10);
        const recentSuccesses = recentExecutions.filter(e => e.success);
        const recentSuccessRate = recentSuccesses.length / recentExecutions.length;

        // Trends
        const efficiencyTrend = this.calculateEfficiencyTrend();
        const confidenceTrend = this.calculateConfidenceTrend();

        // Distribution analysis
        const stepTypeDistribution = this.calculateStepTypeDistribution();
        const errorPatterns = this.calculateErrorPatterns();

        // Time bounds
        const firstExecution = this.executions[0]?.timestamp;
        const lastExecution = this.executions[this.executions.length - 1]?.timestamp;

        return {
            totalExecutions,
            successCount,
            failureCount,
            averageExecutionTime,
            averageTokens,
            averageApiCalls,
            averageCost,
            averageConfidence,
            recentSuccessRate,
            efficiencyTrend,
            confidenceTrend,
            stepTypeDistribution,
            errorPatterns,
            firstExecution,
            lastExecution,
        };
    }

    /**
     * Get execution history for a specific step type
     */
    getExecutionsByStepType(stepType: string): ExecutionRecord[] {
        return this.executions.filter(e => e.stepType === stepType);
    }

    /**
     * Get recent executions (last N)
     */
    getRecentExecutions(count: number = 10): ExecutionRecord[] {
        return this.executions.slice(-count);
    }

    /**
     * Get executions within a time range
     */
    getExecutionsInRange(startTime: Date, endTime: Date): ExecutionRecord[] {
        return this.executions.filter(e => 
            e.timestamp >= startTime && e.timestamp <= endTime
        );
    }

    /**
     * Clear all metrics (useful for testing)
     */
    clear(): void {
        this.executions = [];
        this.logger?.debug("[StrategyMetricsStore] Cleared all metrics");
    }

    /**
     * Get raw execution count
     */
    getExecutionCount(): number {
        return this.executions.length;
    }

    /**
     * Private helper methods
     */
    private getEmptyMetrics(): AggregatedMetrics {
        return {
            totalExecutions: 0,
            successCount: 0,
            failureCount: 0,
            averageExecutionTime: 0,
            averageTokens: 0,
            averageApiCalls: 0,
            averageCost: 0,
            averageConfidence: 0,
            recentSuccessRate: 0,
            efficiencyTrend: 0,
            confidenceTrend: 0,
            stepTypeDistribution: new Map(),
            errorPatterns: new Map(),
        };
    }

    private calculateAverage(records: ExecutionRecord[], field: keyof ExecutionRecord): number {
        if (records.length === 0) return 0;
        
        const sum = records.reduce((total, record) => {
            const value = record[field];
            return total + (typeof value === 'number' ? value : 0);
        }, 0);
        
        return sum / records.length;
    }

    private calculateEfficiencyTrend(): number {
        if (this.executions.length < 6) return 0;

        // Compare first half vs second half of recent executions
        const recentExecutions = this.executions.slice(-20);
        const midpoint = Math.floor(recentExecutions.length / 2);
        
        const firstHalf = recentExecutions.slice(0, midpoint);
        const secondHalf = recentExecutions.slice(midpoint);

        const firstHalfAvg = this.calculateAverage(firstHalf, 'executionTime');
        const secondHalfAvg = this.calculateAverage(secondHalf, 'executionTime');

        if (firstHalfAvg === 0) return 0;

        // Negative trend = getting faster (better), positive = getting slower (worse)
        return (secondHalfAvg - firstHalfAvg) / firstHalfAvg;
    }

    private calculateConfidenceTrend(): number {
        if (this.executions.length < 6) return 0;

        const successfulExecutions = this.executions.filter(e => e.success);
        if (successfulExecutions.length < 6) return 0;

        // Compare first half vs second half of successful executions
        const recentSuccesses = successfulExecutions.slice(-20);
        const midpoint = Math.floor(recentSuccesses.length / 2);
        
        const firstHalf = recentSuccesses.slice(0, midpoint);
        const secondHalf = recentSuccesses.slice(midpoint);

        const firstHalfAvg = this.calculateAverage(firstHalf, 'confidence');
        const secondHalfAvg = this.calculateAverage(secondHalf, 'confidence');

        if (firstHalfAvg === 0) return 0;

        // Positive trend = getting more confident (better)
        return (secondHalfAvg - firstHalfAvg) / firstHalfAvg;
    }

    private calculateStepTypeDistribution(): Map<string, number> {
        const distribution = new Map<string, number>();
        
        for (const execution of this.executions) {
            const count = distribution.get(execution.stepType) || 0;
            distribution.set(execution.stepType, count + 1);
        }
        
        return distribution;
    }

    private calculateErrorPatterns(): Map<string, number> {
        const patterns = new Map<string, number>();
        
        for (const execution of this.executions) {
            if (execution.error) {
                // Extract error type (first word of error message)
                const errorType = execution.error.split(' ')[0] || 'Unknown';
                const count = patterns.get(errorType) || 0;
                patterns.set(errorType, count + 1);
            }
        }
        
        return patterns;
    }

    /**
     * Export metrics to JSON (useful for persistence or debugging)
     */
    exportMetrics(): {
        executions: ExecutionRecord[];
        aggregated: AggregatedMetrics;
        exportedAt: Date;
    } {
        return {
            executions: [...this.executions],
            aggregated: this.getAggregatedMetrics(),
            exportedAt: new Date(),
        };
    }

    /**
     * Import metrics from JSON (useful for loading persisted data)
     */
    importMetrics(data: { executions: ExecutionRecord[] }): void {
        this.executions = data.executions.map(e => ({
            ...e,
            timestamp: new Date(e.timestamp), // Ensure timestamp is a Date object
        }));

        // Sort by timestamp to maintain order
        this.executions.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());

        // Trim to max records
        if (this.executions.length > this.maxRecords) {
            this.executions = this.executions.slice(-this.maxRecords);
        }

        this.logger?.info("[StrategyMetricsStore] Imported metrics", {
            executionCount: this.executions.length,
        });
    }
}