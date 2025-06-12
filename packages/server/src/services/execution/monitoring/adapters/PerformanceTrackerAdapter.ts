/**
 * PerformanceTrackerAdapter - Adapter for Tier 3 PerformanceTracker
 * 
 * Maintains backward compatibility with the original PerformanceTracker
 * while routing all operations through UnifiedMonitoringService.
 */

import { BaseMonitoringAdapter } from "./BaseMonitoringAdapter.js";
import { type Logger } from "winston";

/**
 * Performance entry structure (from original PerformanceTracker)
 */
export interface PerformanceEntry {
    timestamp: Date;
    duration: number;
    success: boolean;
    stepType?: string;
    complexity?: number;
    inputSize?: number;
    outputSize?: number;
    resourcesUsed?: {
        llmTokens?: number;
        toolCalls?: number;
        memoryMB?: number;
    };
    error?: string;
}

/**
 * Performance metrics structure
 */
export interface PerformanceMetrics {
    totalExecutions: number;
    successfulExecutions: number;
    failedExecutions: number;
    successRate: number;
    averageDuration: number;
    p95Duration: number;
    p99Duration: number;
    totalResourcesUsed: {
        llmTokens: number;
        toolCalls: number;
        memoryMB: number;
    };
}

/**
 * Performance feedback structure
 */
export interface PerformanceFeedback {
    isPerformingWell: boolean;
    successRate: number;
    averageDuration: number;
    trend: 'improving' | 'stable' | 'degrading';
    recommendations: string[];
}

/**
 * PerformanceTrackerAdapter - Routes Tier 3 performance tracking through unified monitoring
 */
export class PerformanceTrackerAdapter extends BaseMonitoringAdapter {
    private readonly maxHistorySize: number;
    private recentEntries: PerformanceEntry[] = [];

    constructor(
        maxHistorySize: number = 1000,
        logger?: Logger
    ) {
        super(
            {
                componentName: "PerformanceTracker",
                tier: 3,
                enableEventBus: false, // PerformanceTracker doesn't use event bus
            },
            logger
        );
        
        this.maxHistorySize = maxHistorySize;
    }

    /**
     * Record a performance entry
     */
    async recordPerformance(entry: PerformanceEntry): Promise<void> {
        // Store locally for immediate access
        this.recentEntries.push(entry);
        if (this.recentEntries.length > this.maxHistorySize) {
            this.recentEntries.shift();
        }

        // Record to unified service
        await this.recordMetric(
            `performance.${entry.stepType || 'unknown'}`,
            entry.duration,
            'performance',
            {
                success: entry.success,
                complexity: entry.complexity,
                inputSize: entry.inputSize,
                outputSize: entry.outputSize,
                resourcesUsed: entry.resourcesUsed,
                error: entry.error,
                timestamp: entry.timestamp,
            }
        );

        // Record resource usage if available
        if (entry.resourcesUsed) {
            if (entry.resourcesUsed.llmTokens) {
                await this.recordMetric(
                    'resource.llm_tokens',
                    entry.resourcesUsed.llmTokens,
                    'performance',
                    { stepType: entry.stepType }
                );
            }
            if (entry.resourcesUsed.toolCalls) {
                await this.recordMetric(
                    'resource.tool_calls',
                    entry.resourcesUsed.toolCalls,
                    'performance',
                    { stepType: entry.stepType }
                );
            }
            if (entry.resourcesUsed.memoryMB) {
                await this.recordMetric(
                    'resource.memory_mb',
                    entry.resourcesUsed.memoryMB,
                    'performance',
                    { stepType: entry.stepType }
                );
            }
        }

        // Record success/failure
        await this.recordMetric(
            entry.success ? 'execution.success' : 'execution.failure',
            1,
            entry.success ? 'business' : 'health',
            {
                stepType: entry.stepType,
                duration: entry.duration,
                error: entry.error,
            }
        );
    }

    /**
     * Get performance metrics
     */
    getMetrics(): PerformanceMetrics {
        const entries = this.recentEntries;
        
        if (entries.length === 0) {
            return {
                totalExecutions: 0,
                successfulExecutions: 0,
                failedExecutions: 0,
                successRate: 0,
                averageDuration: 0,
                p95Duration: 0,
                p99Duration: 0,
                totalResourcesUsed: {
                    llmTokens: 0,
                    toolCalls: 0,
                    memoryMB: 0,
                },
            };
        }

        const successfulEntries = entries.filter(e => e.success);
        const failedEntries = entries.filter(e => !e.success);
        const durations = entries.map(e => e.duration).sort((a, b) => a - b);

        const totalResourcesUsed = entries.reduce(
            (acc, entry) => {
                if (entry.resourcesUsed) {
                    acc.llmTokens += entry.resourcesUsed.llmTokens || 0;
                    acc.toolCalls += entry.resourcesUsed.toolCalls || 0;
                    acc.memoryMB += entry.resourcesUsed.memoryMB || 0;
                }
                return acc;
            },
            { llmTokens: 0, toolCalls: 0, memoryMB: 0 }
        );

        return {
            totalExecutions: entries.length,
            successfulExecutions: successfulEntries.length,
            failedExecutions: failedEntries.length,
            successRate: entries.length > 0 ? successfulEntries.length / entries.length : 0,
            averageDuration: durations.reduce((a, b) => a + b, 0) / durations.length,
            p95Duration: durations[Math.floor(durations.length * 0.95)] || 0,
            p99Duration: durations[Math.floor(durations.length * 0.99)] || 0,
            totalResourcesUsed,
        };
    }

    /**
     * Generate performance feedback
     */
    generateFeedback(): PerformanceFeedback {
        const metrics = this.getMetrics();
        const recentTrend = this.getRecentTrend(10);
        
        const recommendations: string[] = [];
        
        if (metrics.successRate < 0.8) {
            recommendations.push("Success rate below 80%. Consider reviewing error patterns.");
        }
        
        if (metrics.p99Duration > metrics.averageDuration * 3) {
            recommendations.push("High variance in execution times. Consider optimizing edge cases.");
        }
        
        if (metrics.totalResourcesUsed.llmTokens > 10000) {
            recommendations.push("High token usage. Consider caching or simplifying prompts.");
        }

        return {
            isPerformingWell: metrics.successRate >= 0.8 && recentTrend !== 'degrading',
            successRate: metrics.successRate,
            averageDuration: metrics.averageDuration,
            trend: recentTrend,
            recommendations,
        };
    }

    /**
     * Get recent performance trend
     */
    getRecentTrend(count: number = 10): 'improving' | 'stable' | 'degrading' {
        if (this.recentEntries.length < count * 2) {
            return 'stable';
        }

        const recent = this.recentEntries.slice(-count);
        const previous = this.recentEntries.slice(-count * 2, -count);

        const recentSuccessRate = recent.filter(e => e.success).length / recent.length;
        const previousSuccessRate = previous.filter(e => e.success).length / previous.length;

        const recentAvgDuration = recent.reduce((sum, e) => sum + e.duration, 0) / recent.length;
        const previousAvgDuration = previous.reduce((sum, e) => sum + e.duration, 0) / previous.length;

        const successImprovement = recentSuccessRate - previousSuccessRate;
        const durationImprovement = previousAvgDuration - recentAvgDuration;

        if (successImprovement > 0.1 || durationImprovement > recentAvgDuration * 0.2) {
            return 'improving';
        } else if (successImprovement < -0.1 || durationImprovement < -recentAvgDuration * 0.2) {
            return 'degrading';
        } else {
            return 'stable';
        }
    }

    /**
     * Clear performance history
     */
    clear(): void {
        this.recentEntries = [];
    }

    /**
     * Get recent entries (for compatibility)
     */
    getRecentEntries(count?: number): PerformanceEntry[] {
        if (count) {
            return this.recentEntries.slice(-count);
        }
        return [...this.recentEntries];
    }

    /**
     * Handle events (not used by PerformanceTracker)
     */
    protected async handleEvent(channel: string, event: any): Promise<void> {
        // PerformanceTracker doesn't subscribe to events
    }
}