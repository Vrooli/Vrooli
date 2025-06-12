import { type Logger } from "winston";
import { type SessionUser } from "@vrooli/shared";
import { type EventBus } from "../../../cross-cutting/events/eventBus.js";
import { type RollingHistoryAdapter as RollingHistory } from "../../../monitoring/adapters/RollingHistoryAdapter.js";
import { MonitoringTools } from "../monitoringTools.js";
import { MonitoringUtils } from "../monitoringUtils.js";

/**
 * Example: Monitoring Tools Usage
 * 
 * This example demonstrates how swarms can use monitoring tools to:
 * 1. Query performance metrics
 * 2. Analyze execution history for patterns
 * 3. Detect anomalies in system behavior
 * 4. Calculate SLOs and publish reports
 * 5. Make data-driven optimization decisions
 */

export class MonitoringToolsExample {
    private readonly monitoringTools: MonitoringTools;
    private readonly logger: Logger;

    constructor(
        user: SessionUser,
        logger: Logger,
        eventBus: EventBus,
        rollingHistory?: RollingHistory,
    ) {
        this.logger = logger;
        this.monitoringTools = new MonitoringTools(user, logger, eventBus, rollingHistory);
    }

    /**
     * Demonstrate performance monitoring workflow
     */
    async demonstratePerformanceMonitoring(): Promise<void> {
        this.logger.info("[MonitoringExample] Starting performance monitoring demonstration");

        try {
            // 1. Query recent performance metrics
            const metricsResult = await this.monitoringTools.queryMetrics({
                timeRange: {
                    duration: 3600000, // Last hour
                },
                eventTypes: ["step_completed", "tool_call"],
                tiers: ["tier3"],
                aggregationWindow: 300000, // 5-minute buckets
            });

            this.logger.info("[MonitoringExample] Recent performance metrics:", {
                result: metricsResult.content?.[0]?.text,
            });

            // 2. Analyze execution history for bottlenecks
            const historyResult = await this.monitoringTools.analyzeHistory({
                patterns: {
                    type: "bottleneck",
                    threshold: 10000, // 10 seconds
                },
                timeWindow: 7200000, // 2 hours
                minOccurrence: 3,
            });

            this.logger.info("[MonitoringExample] Bottleneck analysis:", {
                result: historyResult.content?.[0]?.text,
            });

            // 3. Detect anomalies in response times
            const anomalyResult = await this.monitoringTools.detectAnomalies({
                metric: "data.duration",
                method: "zscore",
                sensitivity: 0.7,
                baselineWindow: 3600000, // 1 hour baseline
                includeContext: true,
            });

            this.logger.info("[MonitoringExample] Anomaly detection:", {
                result: anomalyResult.content?.[0]?.text,
            });

            // 4. Calculate response time SLO
            const sloResult = await this.monitoringTools.calculateSLO({
                slo: {
                    name: "Response Time SLO",
                    metric: "data.duration",
                    target: 5000, // 5 seconds
                    comparison: "lte",
                },
                timeWindow: 86400000, // 24 hours
                breakdown: true,
            });

            this.logger.info("[MonitoringExample] SLO calculation:", {
                result: sloResult.content?.[0]?.text,
            });

            // 5. Generate and publish performance report
            await this.generatePerformanceReport();

        } catch (error) {
            this.logger.error("[MonitoringExample] Performance monitoring failed", error);
        }
    }

    /**
     * Demonstrate resource monitoring workflow
     */
    async demonstrateResourceMonitoring(): Promise<void> {
        this.logger.info("[MonitoringExample] Starting resource monitoring demonstration");

        try {
            // 1. Aggregate credit usage by component
            const creditAggregation = await this.monitoringTools.aggregateData({
                operation: "sum",
                field: "data.credits",
                groupBy: "component",
                filter: {
                    eventType: "resource_allocated",
                    timeRange: {
                        start: new Date(Date.now() - 86400000), // 24 hours ago
                        end: new Date(),
                    },
                },
            });

            this.logger.info("[MonitoringExample] Credit usage by component:", {
                result: creditAggregation.content?.[0]?.text,
            });

            // 2. Detect resource spikes
            const resourceSpikes = await this.monitoringTools.analyzeHistory({
                patterns: {
                    type: "resource_spike",
                    threshold: 500, // Credits
                },
                timeWindow: 3600000, // 1 hour
            });

            this.logger.info("[MonitoringExample] Resource spike analysis:", {
                result: resourceSpikes.content?.[0]?.text,
            });

            // 3. Calculate resource efficiency metrics
            await this.calculateResourceEfficiency();

        } catch (error) {
            this.logger.error("[MonitoringExample] Resource monitoring failed", error);
        }
    }

    /**
     * Demonstrate error monitoring workflow
     */
    async demonstrateErrorMonitoring(): Promise<void> {
        this.logger.info("[MonitoringExample] Starting error monitoring demonstration");

        try {
            // 1. Analyze error patterns
            const errorAnalysis = await this.monitoringTools.analyzeHistory({
                patterns: {
                    type: "error_cluster",
                },
                timeWindow: 7200000, // 2 hours
                minOccurrence: 2,
            });

            this.logger.info("[MonitoringExample] Error cluster analysis:", {
                result: errorAnalysis.content?.[0]?.text,
            });

            // 2. Calculate error rate metrics
            const errorMetrics = await this.monitoringTools.aggregateData({
                operation: "count",
                field: "type",
                groupBy: "component",
                filter: {
                    eventType: "error_occurred",
                    timeRange: {
                        start: new Date(Date.now() - 3600000), // 1 hour ago
                        end: new Date(),
                    },
                },
            });

            this.logger.info("[MonitoringExample] Error rates by component:", {
                result: errorMetrics.content?.[0]?.text,
            });

            // 3. Calculate error rate SLO
            const errorSLO = await this.monitoringTools.calculateSLO({
                slo: {
                    name: "Error Rate SLO",
                    metric: "data.errorRate",
                    target: 0.01, // 1% error rate
                    comparison: "lte",
                },
                timeWindow: 86400000, // 24 hours
            });

            this.logger.info("[MonitoringExample] Error rate SLO:", {
                result: errorSLO.content?.[0]?.text,
            });

        } catch (error) {
            this.logger.error("[MonitoringExample] Error monitoring failed", error);
        }
    }

    /**
     * Demonstrate strategy effectiveness analysis
     */
    async demonstrateStrategyAnalysis(): Promise<void> {
        this.logger.info("[MonitoringExample] Starting strategy analysis demonstration");

        try {
            // 1. Analyze strategy effectiveness
            const strategyAnalysis = await this.monitoringTools.analyzeHistory({
                patterns: {
                    type: "strategy_effectiveness",
                },
                timeWindow: 86400000, // 24 hours
            });

            this.logger.info("[MonitoringExample] Strategy effectiveness:", {
                result: strategyAnalysis.content?.[0]?.text,
            });

            // 2. Compare strategy performance
            const performanceComparison = await this.monitoringTools.aggregateData({
                operation: "avg",
                field: "data.duration",
                groupBy: "data.strategy",
                filter: {
                    eventType: "step_completed",
                    timeRange: {
                        start: new Date(Date.now() - 86400000), // 24 hours ago
                        end: new Date(),
                    },
                },
            });

            this.logger.info("[MonitoringExample] Strategy performance comparison:", {
                result: performanceComparison.content?.[0]?.text,
            });

            // 3. Generate strategy optimization recommendations
            await this.generateStrategyRecommendations();

        } catch (error) {
            this.logger.error("[MonitoringExample] Strategy analysis failed", error);
        }
    }

    /**
     * Generate comprehensive performance report
     */
    private async generatePerformanceReport(): Promise<void> {
        // Gather multiple metrics
        const metrics = await Promise.all([
            this.monitoringTools.aggregateData({
                operation: "avg",
                field: "data.duration",
                groupBy: "component",
                filter: { eventType: "step_completed" },
            }),
            this.monitoringTools.aggregateData({
                operation: "percentile",
                field: "data.duration",
                percentile: 95,
                filter: { eventType: "step_completed" },
            }),
            this.monitoringTools.aggregateData({
                operation: "count",
                field: "type",
                filter: { eventType: "error_occurred" },
            }),
        ]);

        const reportData = {
            timestamp: new Date().toISOString(),
            summary: "System performance report for the last 24 hours",
            avgResponseTime: metrics[0].content?.[0]?.text,
            p95ResponseTime: metrics[1].content?.[0]?.text,
            errorCount: metrics[2].content?.[0]?.text,
            recommendations: [
                "Monitor high-latency components",
                "Investigate error patterns",
                "Consider auto-scaling for peak loads",
            ],
        };

        await this.monitoringTools.publishReport({
            type: "performance",
            data: reportData,
            severity: "info",
            tags: ["performance", "daily-report"],
        });

        this.logger.info("[MonitoringExample] Published performance report");
    }

    /**
     * Calculate resource efficiency metrics
     */
    private async calculateResourceEfficiency(): Promise<void> {
        const [creditUsage, completionCount] = await Promise.all([
            this.monitoringTools.aggregateData({
                operation: "sum",
                field: "data.credits",
                filter: { eventType: "resource_allocated" },
            }),
            this.monitoringTools.aggregateData({
                operation: "count",
                field: "type",
                filter: { eventType: "step_completed" },
            }),
        ]);

        const efficiency = {
            totalCredits: creditUsage.content?.[0]?.text,
            totalCompletions: completionCount.content?.[0]?.text,
            efficiency: "credits per completion",
        };

        this.logger.info("[MonitoringExample] Resource efficiency metrics:", efficiency);
    }

    /**
     * Generate strategy optimization recommendations
     */
    private async generateStrategyRecommendations(): Promise<void> {
        // This would typically involve more sophisticated analysis
        const recommendations = [
            "Consider caching for deterministic strategy to reduce latency",
            "Optimize reasoning strategy for complex decision-making tasks",
            "Use conversational strategy for user-interactive scenarios",
            "Monitor strategy selection patterns for auto-optimization",
        ];

        await this.monitoringTools.publishReport({
            type: "custom",
            data: {
                title: "Strategy Optimization Recommendations",
                recommendations,
                analysisDate: new Date().toISOString(),
            },
            severity: "info",
            tags: ["strategy", "optimization", "recommendations"],
        });

        this.logger.info("[MonitoringExample] Published strategy recommendations");
    }

    /**
     * Demonstrate real-time monitoring using utilities
     */
    async demonstrateRealTimeAnalysis(): Promise<void> {
        this.logger.info("[MonitoringExample] Starting real-time analysis demonstration");

        // Simulate time series data
        const timeSeries = this.generateSampleTimeSeries();

        // Calculate statistics
        const stats = MonitoringUtils.calculateStatistics(
            timeSeries.map(p => p.value),
        );
        this.logger.info("[MonitoringExample] Time series statistics:", stats);

        // Detect anomalies
        const anomalies = MonitoringUtils.detectZScoreAnomalies(
            timeSeries.map(p => p.value),
            2.5,
        );
        this.logger.info("[MonitoringExample] Detected anomalies:", {
            count: anomalies.length,
            anomalies: anomalies.slice(0, 3), // Show first 3
        });

        // Detect trends
        const trend = MonitoringUtils.detectTrend(timeSeries);
        this.logger.info("[MonitoringExample] Trend analysis:", trend);

        // Detect seasonality
        const seasonality = MonitoringUtils.detectSeasonality(timeSeries);
        this.logger.info("[MonitoringExample] Seasonality analysis:", seasonality);

        // Calculate SLI
        const sli = MonitoringUtils.calculateSLI(
            timeSeries.map(p => p.value),
            100, // Good threshold
            "gte",
        );
        this.logger.info("[MonitoringExample] Service Level Indicator:", sli);
    }

    /**
     * Generate sample time series data for demonstration
     */
    private generateSampleTimeSeries(): Array<{
        timestamp: Date;
        value: number;
    }> {
        const data: Array<{ timestamp: Date; value: number }> = [];
        const now = Date.now();
        const interval = 60000; // 1 minute intervals

        for (let i = 0; i < 100; i++) {
            // Generate realistic performance data with some patterns
            const baseValue = 100;
            const trend = i * 0.5; // Slight upward trend
            const seasonal = 20 * Math.sin((i / 10) * Math.PI); // Seasonal pattern
            const noise = (Math.random() - 0.5) * 30; // Random noise
            const anomaly = i === 75 ? 200 : 0; // Spike anomaly

            const value = Math.max(0, baseValue + trend + seasonal + noise + anomaly);

            data.push({
                timestamp: new Date(now - (100 - i) * interval),
                value,
            });
        }

        return data;
    }
}

/**
 * Example usage function
 */
export async function runMonitoringToolsExample(
    user: SessionUser,
    logger: Logger,
    eventBus: EventBus,
    rollingHistory?: RollingHistory,
): Promise<void> {
    const example = new MonitoringToolsExample(user, logger, eventBus, rollingHistory);

    try {
        // Run all demonstrations
        await example.demonstratePerformanceMonitoring();
        await example.demonstrateResourceMonitoring();
        await example.demonstrateErrorMonitoring();
        await example.demonstrateStrategyAnalysis();
        await example.demonstrateRealTimeAnalysis();

        logger.info("[MonitoringExample] All monitoring tool demonstrations completed");
    } catch (error) {
        logger.error("[MonitoringExample] Demonstration failed", error);
        throw error;
    }
}