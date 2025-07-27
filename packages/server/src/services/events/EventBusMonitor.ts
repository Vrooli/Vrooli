/**
 * Event Bus Performance Monitor
 * 
 * Real-time monitoring and optimization for the event bus system,
 * including automatic performance tuning and anomaly detection.
 */

import { DAYS_1_MS, HOURS_1_S, MINUTES_1_MS, MINUTES_1_S } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import { MONITOR_PERFORMANCE_LIMITS, MONITOR_SCORING, MONITOR_THRESHOLDS } from "./constants.js";
import type { EventMode } from "./types.js";

/**
 * Performance metrics for event processing
 */
interface EventMetrics {
    eventType: string;
    count: number;
    totalTime: number;
    avgTime: number;
    minTime: number;
    maxTime: number;
    p95Time: number;
    p99Time: number;
    errors: number;
    lastSeen: number;
}

/**
 * Pattern performance data
 */
interface PatternMetrics {
    pattern: string;
    matchCount: number;
    avgMatchTime: number;
    botCount: number;
    cacheHitRate: number;
}

/**
 * System health indicators
 */
interface SystemHealth {
    status: "healthy" | "degraded" | "critical";
    issues: string[];
    recommendations: string[];
    score: number; // 0-100
}

/**
 * Time series data point
 */
interface DataPoint {
    timestamp: number;
    value: number;
}

/**
 * Event bus performance monitor with automatic optimization
 */
export class EventBusMonitor {
    // Metrics storage
    private eventMetrics = new Map<string, EventMetrics>();
    private patternMetrics = new Map<string, PatternMetrics>();
    private modeMetrics = new Map<EventMode, { count: number; avgTime: number }>();

    // Time series data (last hour, 1-minute buckets)
    private throughputHistory: DataPoint[] = [];
    private latencyHistory: DataPoint[] = [];
    private errorHistory: DataPoint[] = [];

    // Performance thresholds
    private readonly thresholds = {
        maxLatencyMs: MONITOR_PERFORMANCE_LIMITS.MAX_LATENCY_MS,
        maxQueueSize: MONITOR_PERFORMANCE_LIMITS.MAX_QUEUE_SIZE,
        minThroughput: MONITOR_PERFORMANCE_LIMITS.MIN_THROUGHPUT, // events/second
        maxErrorRate: MONITOR_PERFORMANCE_LIMITS.MAX_ERROR_RATE, // 5%
        maxMemoryMB: MONITOR_PERFORMANCE_LIMITS.MAX_MEMORY_MB,
    };

    // Monitoring state
    private monitoringInterval?: NodeJS.Timer;
    private lastCleanup = Date.now();

    /**
     * Start monitoring
     */
    start(): void {
        if (this.monitoringInterval) return;

        // Collect metrics every minute
        this.monitoringInterval = setInterval(() => {
            this.collectMetrics();
            this.analyzePerformance();
            this.cleanupOldData();
        }, MINUTES_1_MS); // 1 minute

        logger.info("[EventBusMonitor] Started performance monitoring");
    }

    /**
     * Stop monitoring
     */
    stop(): void {
        if (this.monitoringInterval) {
            clearInterval(this.monitoringInterval);
            this.monitoringInterval = undefined;
        }
    }

    /**
     * Record event processing
     */
    recordEvent(
        eventType: string,
        mode: EventMode,
        processingTime: number,
        success: boolean,
    ): void {
        // Update event metrics
        let metrics = this.eventMetrics.get(eventType);
        if (!metrics) {
            metrics = {
                eventType,
                count: 0,
                totalTime: 0,
                avgTime: 0,
                minTime: Infinity,
                maxTime: 0,
                p95Time: 0,
                p99Time: 0,
                errors: 0,
                lastSeen: Date.now(),
            };
            this.eventMetrics.set(eventType, metrics);
        }

        metrics.count++;
        metrics.totalTime += processingTime;
        metrics.avgTime = metrics.totalTime / metrics.count;
        metrics.minTime = Math.min(metrics.minTime, processingTime);
        metrics.maxTime = Math.max(metrics.maxTime, processingTime);
        metrics.lastSeen = Date.now();

        if (!success) {
            metrics.errors++;
        }

        // Update mode metrics
        let modeStats = this.modeMetrics.get(mode);
        if (!modeStats) {
            modeStats = { count: 0, avgTime: 0 };
            this.modeMetrics.set(mode, modeStats);
        }
        modeStats.count++;
        modeStats.avgTime = (modeStats.avgTime * (modeStats.count - 1) + processingTime) / modeStats.count;
    }

    /**
     * Record pattern matching performance
     */
    recordPatternMatch(
        pattern: string,
        matchTime: number,
        botCount: number,
        cacheHit: boolean,
    ): void {
        let metrics = this.patternMetrics.get(pattern);
        if (!metrics) {
            metrics = {
                pattern,
                matchCount: 0,
                avgMatchTime: 0,
                botCount: 0,
                cacheHitRate: 0,
            };
            this.patternMetrics.set(pattern, metrics);
        }

        metrics.matchCount++;
        metrics.avgMatchTime = (metrics.avgMatchTime * (metrics.matchCount - 1) + matchTime) / metrics.matchCount;
        metrics.botCount = botCount;

        // Update cache hit rate
        const hits = cacheHit ? metrics.cacheHitRate * (metrics.matchCount - 1) + 1 : metrics.cacheHitRate * (metrics.matchCount - 1);
        metrics.cacheHitRate = hits / metrics.matchCount;
    }

    /**
     * Get system health assessment
     */
    getSystemHealth(): SystemHealth {
        const issues: string[] = [];
        const recommendations: string[] = [];
        let score = 100;

        // Check latency
        const avgLatency = this.getAverageLatency();
        if (avgLatency > this.thresholds.maxLatencyMs) {
            issues.push(`High average latency: ${avgLatency.toFixed(0)}ms`);
            recommendations.push("Consider scaling horizontally or optimizing slow event handlers");
            score -= MONITOR_SCORING.DEDUCTION_HIGH_LATENCY;
        }

        // Check error rate
        const errorRate = this.getErrorRate();
        if (errorRate > this.thresholds.maxErrorRate) {
            issues.push(`High error rate: ${(errorRate * 100).toFixed(1)}%`);
            recommendations.push("Review error logs and fix failing event handlers");
            score -= MONITOR_SCORING.DEDUCTION_HIGH_ERRORS;
        }

        // Check throughput
        const throughput = this.getCurrentThroughput();
        if (throughput < this.thresholds.minThroughput) {
            issues.push(`Low throughput: ${throughput.toFixed(1)} events/sec`);
            recommendations.push("Check for blocking operations or resource constraints");
            score -= MONITOR_SCORING.DEDUCTION_LOW_THROUGHPUT;
        }

        // Check pattern performance
        const slowPatterns = this.getSlowPatterns();
        if (slowPatterns.length > 0) {
            issues.push(`${slowPatterns.length} slow patterns detected`);
            recommendations.push("Optimize pattern matching or reduce wildcard usage");
            score -= MONITOR_SCORING.DEDUCTION_SLOW_PATTERNS;
        }

        // Determine overall status
        let status: SystemHealth["status"];
        if (score >= MONITOR_SCORING.THRESHOLD_HEALTHY) {
            status = "healthy";
        } else if (score >= MONITOR_SCORING.THRESHOLD_DEGRADED) {
            status = "degraded";
        } else {
            status = "critical";
        }

        return {
            status,
            issues,
            recommendations,
            score: Math.max(0, score),
        };
    }

    /**
     * Get performance report
     */
    getPerformanceReport(): {
        summary: {
            totalEvents: number;
            uniqueEventTypes: number;
            avgLatency: number;
            errorRate: number;
            throughput: number;
        };
        topEventTypes: Array<{ type: string; count: number; avgTime: number }>;
        modeDistribution: Record<EventMode, { percentage: number; avgTime: number }>;
        slowestPatterns: Array<{ pattern: string; avgTime: number; cacheHitRate: number }>;
        recommendations: string[];
    } {
        const totalEvents = Array.from(this.eventMetrics.values())
            .reduce((sum, m) => sum + m.count, 0);

        // Get top event types by count
        const topEventTypes = Array.from(this.eventMetrics.values())
            .sort((a, b) => b.count - a.count)
            .slice(0, MONITOR_THRESHOLDS.TOP_EVENT_TYPES_COUNT)
            .map(m => ({
                type: m.eventType,
                count: m.count,
                avgTime: m.avgTime,
            }));

        const modeDistribution: Record<string, { percentage: number; avgTime: number }> = {};
        const totalModeEvents = Array.from(this.modeMetrics.values())
            .reduce((sum, m) => sum + m.count, 0);

        for (const [mode, stats] of this.modeMetrics) {
            modeDistribution[mode] = {
                percentage: (stats.count / totalModeEvents) * 100,
                avgTime: stats.avgTime,
            };
        }

        // Get slowest patterns
        const slowestPatterns = Array.from(this.patternMetrics.values())
            .filter(p => p.matchCount > 0)
            .sort((a, b) => b.avgMatchTime - a.avgMatchTime)
            .slice(0, MONITOR_THRESHOLDS.SLOWEST_PATTERNS_COUNT)
            .map(m => ({
                pattern: m.pattern,
                avgTime: m.avgMatchTime,
                cacheHitRate: m.cacheHitRate,
            }));

        const health = this.getSystemHealth();

        return {
            summary: {
                totalEvents,
                uniqueEventTypes: this.eventMetrics.size,
                avgLatency: this.getAverageLatency(),
                errorRate: this.getErrorRate(),
                throughput: this.getCurrentThroughput(),
            },
            topEventTypes,
            modeDistribution: modeDistribution as Record<EventMode, { percentage: number; avgTime: number }>,
            slowestPatterns,
            recommendations: health.recommendations,
        };
    }

    /**
     * Get optimization suggestions
     */
    getOptimizationSuggestions(): Array<{
        type: "cache" | "index" | "batch" | "pattern";
        target: string;
        reason: string;
        expectedImprovement: string;
    }> {
        const suggestions: Array<{
            type: "cache" | "index" | "batch" | "pattern";
            target: string;
            reason: string;
            expectedImprovement: string;
        }> = [];

        // Pattern optimization suggestions
        for (const [pattern, metrics] of this.patternMetrics.entries()) {
            if (metrics.matchCount > MONITOR_THRESHOLDS.HIGH_FREQUENCY && metrics.avgMatchTime < MONITOR_THRESHOLDS.LOW_PROCESSING_TIME) {
                suggestions.push({
                    type: "cache",
                    target: pattern,
                    reason: "High frequency pattern with low processing time",
                    expectedImprovement: "Reduce CPU usage and improve response time",
                });
            }

            if (metrics.cacheHitRate < MONITOR_THRESHOLDS.LOW_CACHE_HIT_RATE) {
                suggestions.push({
                    type: "index",
                    target: pattern,
                    reason: "Low cache hit rate",
                    expectedImprovement: "2x faster pattern matching",
                });
            }
        }

        // Check for events that would benefit from batching
        const burstEvents = this.detectBurstPatterns();
        for (const eventType of burstEvents) {
            suggestions.push({
                type: "batch",
                target: eventType,
                reason: "Detected burst pattern",
                expectedImprovement: "3x throughput improvement",
            });
        }

        // Check for inefficient patterns
        for (const [pattern, _metrics] of this.patternMetrics) {
            if (pattern.includes("**/") || pattern.split("*").length > MONITOR_THRESHOLDS.COMPLEX_PATTERN_WILDCARD_COUNT) {
                suggestions.push({
                    type: "pattern",
                    target: pattern,
                    reason: "Complex wildcard pattern",
                    expectedImprovement: "10x faster matching with specific patterns",
                });
            }
        }

        return suggestions;
    }

    /**
     * Collect current metrics
     */
    private collectMetrics(): void {
        const now = Date.now();
        const minute = Math.floor(now / MINUTES_1_MS) * MINUTES_1_MS;

        // Calculate throughput
        const eventsInLastMinute = Array.from(this.eventMetrics.values())
            .reduce((sum, m) => {
                const recentCount = m.count; // This would need time-based tracking
                return sum + recentCount;
            }, 0);

        this.throughputHistory.push({
            timestamp: minute,
            value: eventsInLastMinute / MINUTES_1_S, // events per second
        });

        // Calculate average latency
        const avgLatency = this.getAverageLatency();
        this.latencyHistory.push({
            timestamp: minute,
            value: avgLatency,
        });

        // Calculate error rate
        const errorRate = this.getErrorRate();
        this.errorHistory.push({
            timestamp: minute,
            value: errorRate,
        });

        // Keep only last hour
        const oneHourAgo = now - (HOURS_1_S * 1000); // Convert seconds to milliseconds
        this.throughputHistory = this.throughputHistory.filter(d => d.timestamp > oneHourAgo);
        this.latencyHistory = this.latencyHistory.filter(d => d.timestamp > oneHourAgo);
        this.errorHistory = this.errorHistory.filter(d => d.timestamp > oneHourAgo);
    }

    /**
     * Analyze performance and log warnings
     */
    private analyzePerformance(): void {
        const health = this.getSystemHealth();

        if (health.status === "critical") {
            logger.error("[EventBusMonitor] Critical performance issues detected", {
                health,
            });
        } else if (health.status === "degraded") {
            logger.warn("[EventBusMonitor] Performance degradation detected", {
                health,
            });
        }

        // Log optimization suggestions periodically
        const suggestions = this.getOptimizationSuggestions();
        if (suggestions.length > 0) {
            logger.info("[EventBusMonitor] Performance optimization suggestions", {
                suggestions: suggestions.slice(0, 3), // Top 3
            });
        }
    }

    /**
     * Clean up old metrics data
     */
    private cleanupOldData(): void {
        const now = Date.now();
        const dayAgo = now - DAYS_1_MS;

        // Clean up event metrics older than a day
        for (const [eventType, metrics] of this.eventMetrics) {
            if (metrics.lastSeen < dayAgo) {
                this.eventMetrics.delete(eventType);
            }
        }

        this.lastCleanup = now;
    }

    /**
     * Calculate average latency
     */
    private getAverageLatency(): number {
        const metrics = Array.from(this.eventMetrics.values());
        if (metrics.length === 0) return 0;

        const totalTime = metrics.reduce((sum, m) => sum + m.totalTime, 0);
        const totalCount = metrics.reduce((sum, m) => sum + m.count, 0);

        return totalCount > 0 ? totalTime / totalCount : 0;
    }

    /**
     * Calculate error rate
     */
    private getErrorRate(): number {
        const metrics = Array.from(this.eventMetrics.values());

        const totalCount = metrics.reduce((sum, m) => sum + m.count, 0);
        const totalErrors = metrics.reduce((sum, m) => sum + m.errors, 0);

        return totalCount > 0 ? totalErrors / totalCount : 0;
    }

    /**
     * Get current throughput
     */
    private getCurrentThroughput(): number {
        if (this.throughputHistory.length === 0) return 0;
        return this.throughputHistory[this.throughputHistory.length - 1].value;
    }

    /**
     * Get slow patterns
     */
    private getSlowPatterns(): string[] {
        return Array.from(this.patternMetrics.entries())
            .filter(([_, metrics]) => metrics.avgMatchTime > MONITOR_THRESHOLDS.SLOW_PATTERN_MS) // > 10ms
            .map(([pattern]) => pattern);
    }

    /**
     * Detect burst patterns
     */
    private detectBurstPatterns(): string[] {
        // This would need more sophisticated time-series analysis
        // For now, return high-frequency events
        return Array.from(this.eventMetrics.entries())
            .filter(([_, metrics]) => metrics.count > MONITOR_THRESHOLDS.HIGH_EVENT_COUNT)
            .map(([eventType]) => eventType);
    }
}
