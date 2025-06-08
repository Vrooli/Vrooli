/**
 * Shared performance tracking for execution strategies
 * Extracted from common patterns in ConversationalStrategy and DeterministicStrategy
 */

/**
 * Performance tracking entry
 */
export interface PerformanceEntry {
    timestamp: Date;
    executionTime: number;
    success: boolean;
    confidence: number;
    // Strategy-specific data
    metadata?: Record<string, unknown>;
}

/**
 * Performance metrics and analytics
 */
export interface PerformanceMetrics {
    totalExecutions: number;
    successRate: number;
    averageExecutionTime: number;
    averageConfidence: number;
    recentPerformance: PerformanceEntry[];
    trends: {
        executionTimeImprovement: number; // Percentage change over time
        confidenceImprovement: number;
        successRateImprovement: number;
    };
}

/**
 * Performance feedback for strategy learning
 */
export interface PerformanceFeedback {
    shouldAdapt: boolean;
    recommendations: string[];
    optimizationPotential: number; // 0-1 scale
    riskLevel: 'low' | 'medium' | 'high';
}

/**
 * Shared performance tracking utility for execution strategies
 */
export class PerformanceTracker {
    private readonly history: PerformanceEntry[] = [];
    private readonly maxHistorySize: number;
    private readonly analysisWindow: number; // Number of entries to analyze for trends

    constructor(
        maxHistorySize = 100,
        analysisWindow = 20
    ) {
        this.maxHistorySize = maxHistorySize;
        this.analysisWindow = analysisWindow;
    }

    /**
     * Record a performance entry
     */
    recordPerformance(entry: PerformanceEntry): void {
        this.history.push(entry);
        
        // Trim history if too large
        if (this.history.length > this.maxHistorySize) {
            this.history.splice(0, this.history.length - this.maxHistorySize);
        }
    }

    /**
     * Get performance metrics and analytics
     */
    getMetrics(): PerformanceMetrics {
        if (this.history.length === 0) {
            return {
                totalExecutions: 0,
                successRate: 0,
                averageExecutionTime: 0,
                averageConfidence: 0,
                recentPerformance: [],
                trends: {
                    executionTimeImprovement: 0,
                    confidenceImprovement: 0,
                    successRateImprovement: 0,
                },
            };
        }

        const totalExecutions = this.history.length;
        const successCount = this.history.filter(e => e.success).length;
        const successRate = successCount / totalExecutions;
        
        const averageExecutionTime = this.history.reduce((sum, e) => sum + e.executionTime, 0) / totalExecutions;
        const averageConfidence = this.history.reduce((sum, e) => sum + e.confidence, 0) / totalExecutions;
        
        const recentPerformance = this.history.slice(-10); // Last 10 entries
        const trends = this.calculateTrends();

        return {
            totalExecutions,
            successRate,
            averageExecutionTime,
            averageConfidence,
            recentPerformance,
            trends,
        };
    }

    /**
     * Generate performance feedback for strategy adaptation
     */
    generateFeedback(): PerformanceFeedback {
        const metrics = this.getMetrics();
        const recommendations: string[] = [];
        let optimizationPotential = 0;
        let riskLevel: 'low' | 'medium' | 'high' = 'low';

        // Analyze success rate
        if (metrics.successRate < 0.7) {
            recommendations.push("Consider adjusting strategy parameters or validation criteria");
            optimizationPotential += 0.3;
            riskLevel = 'high';
        } else if (metrics.successRate < 0.85) {
            recommendations.push("Minor optimization opportunities available");
            optimizationPotential += 0.1;
            riskLevel = 'medium';
        }

        // Analyze execution time trends
        if (metrics.trends.executionTimeImprovement < -0.1) {
            recommendations.push("Execution time is degrading - investigate performance bottlenecks");
            optimizationPotential += 0.2;
            riskLevel = riskLevel === 'low' ? 'medium' : 'high';
        }

        // Analyze confidence trends
        if (metrics.trends.confidenceImprovement < -0.05) {
            recommendations.push("Confidence levels decreasing - review input quality or model selection");
            optimizationPotential += 0.1;
        }

        const shouldAdapt = optimizationPotential > 0.15 || riskLevel === 'high';

        return {
            shouldAdapt,
            recommendations,
            optimizationPotential: Math.min(optimizationPotential, 1.0),
            riskLevel,
        };
    }

    /**
     * Get recent performance for immediate decision making
     */
    getRecentTrend(count = 5): {
        averageSuccessRate: number;
        averageExecutionTime: number;
        averageConfidence: number;
    } {
        const recent = this.history.slice(-count);
        
        if (recent.length === 0) {
            return {
                averageSuccessRate: 0,
                averageExecutionTime: 0,
                averageConfidence: 0,
            };
        }

        const successCount = recent.filter(e => e.success).length;
        const averageSuccessRate = successCount / recent.length;
        const averageExecutionTime = recent.reduce((sum, e) => sum + e.executionTime, 0) / recent.length;
        const averageConfidence = recent.reduce((sum, e) => sum + e.confidence, 0) / recent.length;

        return {
            averageSuccessRate,
            averageExecutionTime,
            averageConfidence,
        };
    }

    /**
     * Calculate performance trends over the analysis window
     */
    private calculateTrends(): {
        executionTimeImprovement: number;
        confidenceImprovement: number;
        successRateImprovement: number;
    } {
        if (this.history.length < this.analysisWindow) {
            return {
                executionTimeImprovement: 0,
                confidenceImprovement: 0,
                successRateImprovement: 0,
            };
        }

        const recent = this.history.slice(-this.analysisWindow);
        const older = this.history.slice(-this.analysisWindow * 2, -this.analysisWindow);

        if (older.length === 0) {
            return {
                executionTimeImprovement: 0,
                confidenceImprovement: 0,
                successRateImprovement: 0,
            };
        }

        // Calculate averages for both periods
        const recentAvgTime = recent.reduce((sum, e) => sum + e.executionTime, 0) / recent.length;
        const olderAvgTime = older.reduce((sum, e) => sum + e.executionTime, 0) / older.length;
        
        const recentAvgConfidence = recent.reduce((sum, e) => sum + e.confidence, 0) / recent.length;
        const olderAvgConfidence = older.reduce((sum, e) => sum + e.confidence, 0) / older.length;
        
        const recentSuccessRate = recent.filter(e => e.success).length / recent.length;
        const olderSuccessRate = older.filter(e => e.success).length / older.length;

        // Calculate improvement (negative means degradation)
        const executionTimeImprovement = olderAvgTime > 0 ? (olderAvgTime - recentAvgTime) / olderAvgTime : 0;
        const confidenceImprovement = olderAvgConfidence > 0 ? (recentAvgConfidence - olderAvgConfidence) / olderAvgConfidence : 0;
        const successRateImprovement = olderSuccessRate > 0 ? (recentSuccessRate - olderSuccessRate) / olderSuccessRate : 0;

        return {
            executionTimeImprovement,
            confidenceImprovement,
            successRateImprovement,
        };
    }

    /**
     * Clear performance history (useful for testing or major strategy changes)
     */
    clearHistory(): void {
        this.history.length = 0;
    }

    /**
     * Get raw performance history (for detailed analysis)
     */
    getHistory(): readonly PerformanceEntry[] {
        return [...this.history];
    }
}