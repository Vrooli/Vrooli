import { type Logger } from "winston";
import { type SessionUser } from "@vrooli/shared";
import { type ToolResponse } from "../../../mcp/types.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type RollingHistoryAdapter as RollingHistory } from "../../monitoring/adapters/RollingHistoryAdapter.js";

/**
 * Monitoring tool parameters
 */
export interface QueryMetricsParams {
    // Time range for query
    timeRange?: {
        start?: Date;
        end?: Date;
        duration?: number; // Duration in milliseconds from now
    };
    // Event types to filter
    eventTypes?: string[];
    // Components to filter
    components?: string[];
    // Tiers to filter
    tiers?: ("tier1" | "tier2" | "tier3")[];
    // Limit number of results
    limit?: number;
    // Aggregation window (for time-series data)
    aggregationWindow?: number; // milliseconds
}

export interface AnalyzeHistoryParams {
    // Pattern detection parameters
    patterns?: {
        type: "bottleneck" | "error_cluster" | "resource_spike" | "strategy_effectiveness";
        threshold?: number;
    };
    // Time window for analysis
    timeWindow?: number; // milliseconds
    // Minimum occurrence for pattern detection
    minOccurrence?: number;
}

export interface AggregateDataParams {
    // Aggregation type
    operation: "sum" | "avg" | "min" | "max" | "count" | "percentile";
    // Field to aggregate
    field: string;
    // Group by field
    groupBy?: string;
    // Time bucket size (for time-series aggregation)
    bucketSize?: number; // milliseconds
    // Percentile value (if operation is percentile)
    percentile?: number;
    // Filter criteria
    filter?: {
        eventType?: string;
        component?: string;
        tier?: string;
        timeRange?: {
            start?: Date;
            end?: Date;
        };
    };
}

export interface PublishReportParams {
    // Report type
    type: "performance" | "health" | "slo" | "custom";
    // Report data
    data: Record<string, unknown>;
    // Severity level
    severity?: "info" | "warning" | "error" | "critical";
    // Recipients (optional)
    recipients?: string[];
    // Tags for categorization
    tags?: string[];
}

export interface DetectAnomaliesParams {
    // Metric to analyze
    metric: string;
    // Detection method
    method?: "zscore" | "mad" | "isolation_forest" | "percentile";
    // Sensitivity threshold
    sensitivity?: number; // 0-1, where 1 is most sensitive
    // Time window for baseline
    baselineWindow?: number; // milliseconds
    // Whether to include historical context
    includeContext?: boolean;
}

export interface CalculateSLOParams {
    // SLO definition
    slo: {
        name: string;
        metric: string;
        target: number; // Target value
        comparison: "gte" | "lte" | "eq"; // Greater than or equal, less than or equal, equal
    };
    // Time window for calculation
    timeWindow?: number; // milliseconds
    // Whether to include breakdown by component
    breakdown?: boolean;
}

/**
 * Monitoring tools for swarm intelligence
 * 
 * These tools enable swarms to monitor and analyze system performance,
 * detect patterns, and make data-driven decisions. They integrate with
 * the telemetry shim and rolling history to provide observability.
 */
export class MonitoringTools {
    private readonly user: SessionUser;
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly rollingHistory?: RollingHistory;

    constructor(
        user: SessionUser,
        logger: Logger,
        eventBus: EventBus,
        rollingHistory?: RollingHistory,
    ) {
        this.user = user;
        this.logger = logger;
        this.eventBus = eventBus;
        this.rollingHistory = rollingHistory;
    }

    /**
     * Query performance metrics from telemetry
     */
    async queryMetrics(params: QueryMetricsParams): Promise<ToolResponse> {
        try {
            if (!this.rollingHistory) {
                return {
                    isError: true,
                    content: [{
                        type: "text",
                        text: "Rolling history not available for metrics query",
                    }],
                };
            }

            // Determine time range
            const now = Date.now();
            const timeRange = params.timeRange || {};
            const startTime = timeRange.start?.getTime() || 
                (timeRange.duration ? now - timeRange.duration : now - 3600000); // Default 1 hour
            const endTime = timeRange.end?.getTime() || now;

            // Get events from rolling history
            const allEvents = this.rollingHistory.getEventsInTimeRange(startTime, endTime);

            // Apply filters
            let filteredEvents = allEvents;
            
            if (params.eventTypes && params.eventTypes.length > 0) {
                filteredEvents = filteredEvents.filter(event => 
                    params.eventTypes!.some(type => event.type.includes(type))
                );
            }

            if (params.components && params.components.length > 0) {
                filteredEvents = filteredEvents.filter(event => 
                    params.components!.includes(event.component)
                );
            }

            if (params.tiers && params.tiers.length > 0) {
                filteredEvents = filteredEvents.filter(event => 
                    params.tiers!.includes(event.tier as any)
                );
            }

            // Apply limit
            if (params.limit) {
                filteredEvents = filteredEvents.slice(0, params.limit);
            }

            // Calculate metrics
            const metrics = this.calculateMetrics(filteredEvents, params.aggregationWindow);

            return {
                isError: false,
                content: [{
                    type: "text",
                    text: JSON.stringify({
                        timeRange: {
                            start: new Date(startTime).toISOString(),
                            end: new Date(endTime).toISOString(),
                        },
                        totalEvents: filteredEvents.length,
                        metrics,
                        events: filteredEvents.slice(0, 100), // Limit raw events
                    }, null, 2),
                }],
            };

        } catch (error) {
            this.logger.error("[MonitoringTools] Error querying metrics", error);
            return {
                isError: true,
                content: [{
                    type: "text",
                    text: `Error querying metrics: ${error instanceof Error ? error.message : String(error)}`,
                }],
            };
        }
    }

    /**
     * Analyze execution history for patterns
     */
    async analyzeHistory(params: AnalyzeHistoryParams): Promise<ToolResponse> {
        try {
            if (!this.rollingHistory) {
                return {
                    isError: true,
                    content: [{
                        type: "text",
                        text: "Rolling history not available for analysis",
                    }],
                };
            }

            const timeWindow = params.timeWindow || 3600000; // Default 1 hour
            const minOccurrence = params.minOccurrence || 3;

            let analysis: any = {};

            if (!params.patterns || params.patterns.type === "bottleneck") {
                analysis.bottlenecks = this.detectBottlenecks(timeWindow, params.patterns?.threshold);
            }

            if (!params.patterns || params.patterns.type === "error_cluster") {
                analysis.errorClusters = this.detectErrorClusters(timeWindow, minOccurrence);
            }

            if (!params.patterns || params.patterns.type === "resource_spike") {
                analysis.resourceSpikes = this.detectResourceSpikes(timeWindow, params.patterns?.threshold);
            }

            if (!params.patterns || params.patterns.type === "strategy_effectiveness") {
                analysis.strategyEffectiveness = this.analyzeStrategyEffectiveness(timeWindow);
            }

            // Overall patterns
            const patterns = this.rollingHistory.detectPatterns(timeWindow);
            analysis.overallPatterns = patterns;

            return {
                isError: false,
                content: [{
                    type: "text",
                    text: JSON.stringify(analysis, null, 2),
                }],
            };

        } catch (error) {
            this.logger.error("[MonitoringTools] Error analyzing history", error);
            return {
                isError: true,
                content: [{
                    type: "text",
                    text: `Error analyzing history: ${error instanceof Error ? error.message : String(error)}`,
                }],
            };
        }
    }

    /**
     * Perform statistical aggregation on metrics
     */
    async aggregateData(params: AggregateDataParams): Promise<ToolResponse> {
        try {
            if (!this.rollingHistory) {
                return {
                    isError: true,
                    content: [{
                        type: "text",
                        text: "Rolling history not available for aggregation",
                    }],
                };
            }

            // Get filtered events
            let events = this.rollingHistory.getAllEvents();

            // Apply filters
            if (params.filter) {
                events = events.filter(event => {
                    if (params.filter!.eventType && !event.type.includes(params.filter!.eventType)) {
                        return false;
                    }
                    if (params.filter!.component && event.component !== params.filter!.component) {
                        return false;
                    }
                    if (params.filter!.tier && event.tier !== params.filter!.tier) {
                        return false;
                    }
                    if (params.filter!.timeRange) {
                        const eventTime = event.timestamp.getTime();
                        const start = params.filter!.timeRange.start?.getTime() || 0;
                        const end = params.filter!.timeRange.end?.getTime() || Date.now();
                        if (eventTime < start || eventTime > end) {
                            return false;
                        }
                    }
                    return true;
                });
            }

            // Perform aggregation
            const result = this.performAggregation(events, params);

            return {
                isError: false,
                content: [{
                    type: "text",
                    text: JSON.stringify(result, null, 2),
                }],
            };

        } catch (error) {
            this.logger.error("[MonitoringTools] Error aggregating data", error);
            return {
                isError: true,
                content: [{
                    type: "text",
                    text: `Error aggregating data: ${error instanceof Error ? error.message : String(error)}`,
                }],
            };
        }
    }

    /**
     * Publish monitoring report
     */
    async publishReport(params: PublishReportParams): Promise<ToolResponse> {
        try {
            const report = {
                id: `report_${Date.now()}`,
                type: params.type,
                timestamp: new Date(),
                severity: params.severity || "info",
                data: params.data,
                tags: params.tags || [],
                publishedBy: this.user.id,
            };

            // Publish to event bus
            await this.eventBus.publish({
                id: report.id,
                type: "monitoring.report",
                timestamp: report.timestamp,
                source: {
                    tier: "cross-cutting" as const,
                    component: "monitoring-tools",
                    instanceId: this.user.id,
                },
                data: report,
            });

            // Log the report
            this.logger.info(`[MonitoringTools] Published ${params.type} report`, {
                reportId: report.id,
                severity: report.severity,
                tags: report.tags,
            });

            return {
                isError: false,
                content: [{
                    type: "text",
                    text: JSON.stringify({
                        success: true,
                        reportId: report.id,
                        message: `${params.type} report published successfully`,
                    }, null, 2),
                }],
            };

        } catch (error) {
            this.logger.error("[MonitoringTools] Error publishing report", error);
            return {
                isError: true,
                content: [{
                    type: "text",
                    text: `Error publishing report: ${error instanceof Error ? error.message : String(error)}`,
                }],
            };
        }
    }

    /**
     * Detect anomalies in metrics
     */
    async detectAnomalies(params: DetectAnomaliesParams): Promise<ToolResponse> {
        try {
            if (!this.rollingHistory) {
                return {
                    isError: true,
                    content: [{
                        type: "text",
                        text: "Rolling history not available for anomaly detection",
                    }],
                };
            }

            const method = params.method || "zscore";
            const sensitivity = params.sensitivity || 0.5;
            const baselineWindow = params.baselineWindow || 3600000; // 1 hour default

            // Get metric values from history
            const metricValues = this.extractMetricValues(params.metric, baselineWindow);

            if (metricValues.length < 10) {
                return {
                    isError: false,
                    content: [{
                        type: "text",
                        text: JSON.stringify({
                            anomalies: [],
                            message: "Insufficient data for anomaly detection",
                        }, null, 2),
                    }],
                };
            }

            // Detect anomalies based on method
            let anomalies: any[] = [];
            
            switch (method) {
                case "zscore":
                    anomalies = this.detectZScoreAnomalies(metricValues, sensitivity);
                    break;
                case "mad":
                    anomalies = this.detectMADAnomalies(metricValues, sensitivity);
                    break;
                case "percentile":
                    anomalies = this.detectPercentileAnomalies(metricValues, sensitivity);
                    break;
                default:
                    anomalies = this.detectZScoreAnomalies(metricValues, sensitivity);
            }

            // Add context if requested
            if (params.includeContext && anomalies.length > 0) {
                anomalies = anomalies.map(anomaly => ({
                    ...anomaly,
                    context: this.getAnomalyContext(anomaly, params.metric),
                }));
            }

            return {
                isError: false,
                content: [{
                    type: "text",
                    text: JSON.stringify({
                        metric: params.metric,
                        method,
                        sensitivity,
                        totalDataPoints: metricValues.length,
                        anomalyCount: anomalies.length,
                        anomalies,
                    }, null, 2),
                }],
            };

        } catch (error) {
            this.logger.error("[MonitoringTools] Error detecting anomalies", error);
            return {
                isError: true,
                content: [{
                    type: "text",
                    text: `Error detecting anomalies: ${error instanceof Error ? error.message : String(error)}`,
                }],
            };
        }
    }

    /**
     * Calculate service level objectives
     */
    async calculateSLO(params: CalculateSLOParams): Promise<ToolResponse> {
        try {
            if (!this.rollingHistory) {
                return {
                    isError: true,
                    content: [{
                        type: "text",
                        text: "Rolling history not available for SLO calculation",
                    }],
                };
            }

            const timeWindow = params.timeWindow || 86400000; // 24 hours default
            const slo = params.slo;

            // Get metric values
            const metricValues = this.extractMetricValues(slo.metric, timeWindow);

            if (metricValues.length === 0) {
                return {
                    isError: false,
                    content: [{
                        type: "text",
                        text: JSON.stringify({
                            slo: slo.name,
                            message: "No data available for SLO calculation",
                        }, null, 2),
                    }],
                };
            }

            // Calculate SLO compliance
            const compliantValues = metricValues.filter(value => {
                switch (slo.comparison) {
                    case "gte":
                        return value.value >= slo.target;
                    case "lte":
                        return value.value <= slo.target;
                    case "eq":
                        return Math.abs(value.value - slo.target) < 0.001;
                    default:
                        return false;
                }
            });

            const compliance = (compliantValues.length / metricValues.length) * 100;

            // Calculate breakdown if requested
            let breakdown = undefined;
            if (params.breakdown) {
                breakdown = this.calculateSLOBreakdown(metricValues, slo);
            }

            return {
                isError: false,
                content: [{
                    type: "text",
                    text: JSON.stringify({
                        slo: {
                            name: slo.name,
                            metric: slo.metric,
                            target: slo.target,
                            comparison: slo.comparison,
                        },
                        timeWindow: {
                            start: new Date(Date.now() - timeWindow).toISOString(),
                            end: new Date().toISOString(),
                        },
                        compliance: {
                            percentage: compliance.toFixed(2),
                            compliantCount: compliantValues.length,
                            totalCount: metricValues.length,
                        },
                        statistics: {
                            mean: this.calculateMean(metricValues.map(v => v.value)),
                            median: this.calculateMedian(metricValues.map(v => v.value)),
                            min: Math.min(...metricValues.map(v => v.value)),
                            max: Math.max(...metricValues.map(v => v.value)),
                        },
                        breakdown,
                    }, null, 2),
                }],
            };

        } catch (error) {
            this.logger.error("[MonitoringTools] Error calculating SLO", error);
            return {
                isError: true,
                content: [{
                    type: "text",
                    text: `Error calculating SLO: ${error instanceof Error ? error.message : String(error)}`,
                }],
            };
        }
    }

    /**
     * Helper methods
     */

    private calculateMetrics(events: any[], aggregationWindow?: number): any {
        const metrics: any = {
            eventCount: events.length,
            eventTypes: {},
            components: {},
            tiers: {},
            timeSeries: [],
        };

        // Count by type, component, and tier
        for (const event of events) {
            metrics.eventTypes[event.type] = (metrics.eventTypes[event.type] || 0) + 1;
            metrics.components[event.component] = (metrics.components[event.component] || 0) + 1;
            metrics.tiers[event.tier] = (metrics.tiers[event.tier] || 0) + 1;
        }

        // Generate time series if aggregation window specified
        if (aggregationWindow && events.length > 0) {
            const sorted = events.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
            const startTime = sorted[0].timestamp.getTime();
            const endTime = sorted[sorted.length - 1].timestamp.getTime();

            for (let time = startTime; time <= endTime; time += aggregationWindow) {
                const windowEnd = time + aggregationWindow;
                const windowEvents = sorted.filter(e => {
                    const eventTime = e.timestamp.getTime();
                    return eventTime >= time && eventTime < windowEnd;
                });

                metrics.timeSeries.push({
                    timestamp: new Date(time).toISOString(),
                    count: windowEvents.length,
                });
            }
        }

        return metrics;
    }

    private detectBottlenecks(timeWindow: number, threshold = 30000): any[] {
        if (!this.rollingHistory) return [];

        const events = this.rollingHistory.getEventsInTimeRange(
            Date.now() - timeWindow,
            Date.now()
        );

        const stepEvents = events.filter(e => e.type.includes("step_completed"));
        const componentDurations = new Map<string, number[]>();

        for (const event of stepEvents) {
            const component = event.component;
            const duration = event.data.duration as number || 0;

            if (!componentDurations.has(component)) {
                componentDurations.set(component, []);
            }
            componentDurations.get(component)!.push(duration);
        }

        const bottlenecks: any[] = [];
        for (const [component, durations] of componentDurations) {
            const sorted = durations.sort((a, b) => a - b);
            const p95Index = Math.floor(durations.length * 0.95);
            const p95 = sorted[p95Index] || sorted[sorted.length - 1];

            if (p95 > threshold) {
                bottlenecks.push({
                    component,
                    avgDuration: this.calculateMean(durations),
                    p95Duration: p95,
                    maxDuration: Math.max(...durations),
                    sampleCount: durations.length,
                });
            }
        }

        return bottlenecks;
    }

    private detectErrorClusters(timeWindow: number, minOccurrence: number): any[] {
        if (!this.rollingHistory) return [];

        const events = this.rollingHistory.getEventsInTimeRange(
            Date.now() - timeWindow,
            Date.now()
        );

        const errorEvents = events.filter(e => 
            e.type.includes("error") || e.type.includes("fail")
        );

        // Group errors by type and component
        const errorGroups = new Map<string, any[]>();
        
        for (const event of errorEvents) {
            const key = `${event.type}_${event.component}`;
            if (!errorGroups.has(key)) {
                errorGroups.set(key, []);
            }
            errorGroups.get(key)!.push(event);
        }

        const clusters: any[] = [];
        for (const [key, errors] of errorGroups) {
            if (errors.length >= minOccurrence) {
                const [type, component] = key.split("_");
                clusters.push({
                    errorType: type,
                    component,
                    count: errors.length,
                    timeRange: {
                        start: new Date(Math.min(...errors.map(e => e.timestamp.getTime()))),
                        end: new Date(Math.max(...errors.map(e => e.timestamp.getTime()))),
                    },
                    samples: errors.slice(0, 5).map(e => ({
                        timestamp: e.timestamp,
                        message: e.data.message || e.data.error,
                    })),
                });
            }
        }

        return clusters;
    }

    private detectResourceSpikes(timeWindow: number, threshold = 1000): any[] {
        if (!this.rollingHistory) return [];

        const events = this.rollingHistory.getEventsInTimeRange(
            Date.now() - timeWindow,
            Date.now()
        );

        const resourceEvents = events.filter(e => 
            e.type.includes("resource_allocated") || e.type.includes("resource_usage")
        );

        const spikes: any[] = [];
        for (const event of resourceEvents) {
            const credits = event.data.credits as number || 0;
            if (credits > threshold) {
                spikes.push({
                    timestamp: event.timestamp,
                    component: event.component,
                    credits,
                    exceedance: credits - threshold,
                    context: {
                        stepId: event.data.stepId,
                        models: event.data.models || [],
                    },
                });
            }
        }

        return spikes;
    }

    private analyzeStrategyEffectiveness(timeWindow: number): any {
        if (!this.rollingHistory) return {};

        const events = this.rollingHistory.getEventsInTimeRange(
            Date.now() - timeWindow,
            Date.now()
        );

        const strategyEvents = events.filter(e => e.type.includes("strategy_selected"));
        const completionEvents = events.filter(e => e.type.includes("step_completed"));

        const strategies = new Map<string, {
            selections: number;
            completions: number;
            durations: number[];
            failures: number;
        }>();

        // Track strategy selections
        for (const event of strategyEvents) {
            const strategy = event.data.selected as string;
            if (!strategies.has(strategy)) {
                strategies.set(strategy, {
                    selections: 0,
                    completions: 0,
                    durations: [],
                    failures: 0,
                });
            }
            strategies.get(strategy)!.selections++;
        }

        // Track completions
        for (const event of completionEvents) {
            const strategy = event.data.strategy as string;
            if (strategy && strategies.has(strategy)) {
                const stats = strategies.get(strategy)!;
                stats.completions++;
                if (event.data.duration) {
                    stats.durations.push(event.data.duration as number);
                }
                if (event.data.error) {
                    stats.failures++;
                }
            }
        }

        const effectiveness: any = {};
        for (const [strategy, stats] of strategies) {
            effectiveness[strategy] = {
                successRate: stats.selections > 0 
                    ? ((stats.completions - stats.failures) / stats.selections) * 100 
                    : 0,
                avgDuration: stats.durations.length > 0 
                    ? this.calculateMean(stats.durations) 
                    : null,
                totalSelections: stats.selections,
                totalCompletions: stats.completions,
                totalFailures: stats.failures,
            };
        }

        return effectiveness;
    }

    private performAggregation(events: any[], params: AggregateDataParams): any {
        const values = events
            .map(e => this.extractFieldValue(e, params.field))
            .filter(v => v !== undefined && v !== null && !isNaN(Number(v)))
            .map(v => Number(v));

        if (values.length === 0) {
            return {
                operation: params.operation,
                field: params.field,
                result: null,
                message: "No numeric values found for aggregation",
            };
        }

        let result: number;
        switch (params.operation) {
            case "sum":
                result = values.reduce((a, b) => a + b, 0);
                break;
            case "avg":
                result = this.calculateMean(values);
                break;
            case "min":
                result = Math.min(...values);
                break;
            case "max":
                result = Math.max(...values);
                break;
            case "count":
                result = values.length;
                break;
            case "percentile":
                const p = params.percentile || 95;
                result = this.calculatePercentile(values, p);
                break;
            default:
                result = 0;
        }

        // Handle groupBy
        if (params.groupBy) {
            const groups = new Map<string, number[]>();
            for (let i = 0; i < events.length; i++) {
                const groupValue = this.extractFieldValue(events[i], params.groupBy);
                if (groupValue !== undefined) {
                    const value = this.extractFieldValue(events[i], params.field);
                    if (value !== undefined && !isNaN(Number(value))) {
                        const key = String(groupValue);
                        if (!groups.has(key)) {
                            groups.set(key, []);
                        }
                        groups.get(key)!.push(Number(value));
                    }
                }
            }

            const groupResults: any = {};
            for (const [key, groupValues] of groups) {
                let groupResult: number;
                switch (params.operation) {
                    case "sum":
                        groupResult = groupValues.reduce((a, b) => a + b, 0);
                        break;
                    case "avg":
                        groupResult = this.calculateMean(groupValues);
                        break;
                    case "min":
                        groupResult = Math.min(...groupValues);
                        break;
                    case "max":
                        groupResult = Math.max(...groupValues);
                        break;
                    case "count":
                        groupResult = groupValues.length;
                        break;
                    case "percentile":
                        groupResult = this.calculatePercentile(groupValues, params.percentile || 95);
                        break;
                    default:
                        groupResult = 0;
                }
                groupResults[key] = groupResult;
            }

            return {
                operation: params.operation,
                field: params.field,
                groupBy: params.groupBy,
                results: groupResults,
                totalGroups: groups.size,
            };
        }

        return {
            operation: params.operation,
            field: params.field,
            result,
            sampleCount: values.length,
        };
    }

    private extractMetricValues(metric: string, timeWindow: number): Array<{
        timestamp: Date;
        value: number;
        component?: string;
    }> {
        if (!this.rollingHistory) return [];

        const events = this.rollingHistory.getEventsInTimeRange(
            Date.now() - timeWindow,
            Date.now()
        );

        const values: any[] = [];
        for (const event of events) {
            const value = this.extractFieldValue(event, metric);
            if (value !== undefined && !isNaN(Number(value))) {
                values.push({
                    timestamp: event.timestamp,
                    value: Number(value),
                    component: event.component,
                });
            }
        }

        return values;
    }

    private extractFieldValue(obj: any, path: string): any {
        const parts = path.split(".");
        let value = obj;
        for (const part of parts) {
            if (value && typeof value === "object" && part in value) {
                value = value[part];
            } else {
                return undefined;
            }
        }
        return value;
    }

    private detectZScoreAnomalies(values: any[], sensitivity: number): any[] {
        const nums = values.map(v => v.value);
        const mean = this.calculateMean(nums);
        const stdDev = this.calculateStdDev(nums, mean);
        
        const threshold = 3 - (sensitivity * 2); // Sensitivity 0-1 maps to z-score 3-1
        
        const anomalies: any[] = [];
        for (const value of values) {
            const zScore = Math.abs((value.value - mean) / stdDev);
            if (zScore > threshold) {
                anomalies.push({
                    ...value,
                    zScore,
                    deviation: value.value - mean,
                    severity: zScore > 4 ? "critical" : zScore > 3 ? "high" : "medium",
                });
            }
        }
        
        return anomalies;
    }

    private detectMADAnomalies(values: any[], sensitivity: number): any[] {
        const nums = values.map(v => v.value);
        const median = this.calculateMedian(nums);
        const mad = this.calculateMAD(nums, median);
        
        const threshold = 3 - (sensitivity * 2);
        
        const anomalies: any[] = [];
        for (const value of values) {
            const madScore = Math.abs((value.value - median) / mad);
            if (madScore > threshold) {
                anomalies.push({
                    ...value,
                    madScore,
                    deviation: value.value - median,
                    severity: madScore > 4 ? "critical" : madScore > 3 ? "high" : "medium",
                });
            }
        }
        
        return anomalies;
    }

    private detectPercentileAnomalies(values: any[], sensitivity: number): any[] {
        const nums = values.map(v => v.value);
        const sorted = nums.sort((a, b) => a - b);
        
        const lowerBound = this.calculatePercentile(sorted, 5 - (sensitivity * 4));
        const upperBound = this.calculatePercentile(sorted, 95 + (sensitivity * 4));
        
        const anomalies: any[] = [];
        for (const value of values) {
            if (value.value < lowerBound || value.value > upperBound) {
                anomalies.push({
                    ...value,
                    bounds: { lower: lowerBound, upper: upperBound },
                    position: value.value < lowerBound ? "below" : "above",
                    severity: Math.abs(value.value - (value.value < lowerBound ? lowerBound : upperBound)) > 
                        (upperBound - lowerBound) * 0.5 ? "high" : "medium",
                });
            }
        }
        
        return anomalies;
    }

    private getAnomalyContext(anomaly: any, metric: string): any {
        if (!this.rollingHistory) return {};

        // Get events around the anomaly time (±5 minutes)
        const contextWindow = 300000; // 5 minutes
        const startTime = anomaly.timestamp.getTime() - contextWindow;
        const endTime = anomaly.timestamp.getTime() + contextWindow;

        const contextEvents = this.rollingHistory.getEventsInTimeRange(startTime, endTime);

        return {
            nearbyEvents: contextEvents.length,
            errorCount: contextEvents.filter(e => e.type.includes("error")).length,
            warningCount: contextEvents.filter(e => e.type.includes("warning")).length,
            relatedComponents: [...new Set(contextEvents.map(e => e.component))],
        };
    }

    private calculateSLOBreakdown(values: any[], slo: any): any {
        const breakdown = new Map<string, {
            total: number;
            compliant: number;
            values: number[];
        }>();

        for (const value of values) {
            const component = value.component || "unknown";
            if (!breakdown.has(component)) {
                breakdown.set(component, {
                    total: 0,
                    compliant: 0,
                    values: [],
                });
            }

            const stats = breakdown.get(component)!;
            stats.total++;
            stats.values.push(value.value);

            let isCompliant = false;
            switch (slo.comparison) {
                case "gte":
                    isCompliant = value.value >= slo.target;
                    break;
                case "lte":
                    isCompliant = value.value <= slo.target;
                    break;
                case "eq":
                    isCompliant = Math.abs(value.value - slo.target) < 0.001;
                    break;
            }

            if (isCompliant) {
                stats.compliant++;
            }
        }

        const result: any = {};
        for (const [component, stats] of breakdown) {
            result[component] = {
                compliance: (stats.compliant / stats.total) * 100,
                sampleCount: stats.total,
                mean: this.calculateMean(stats.values),
                min: Math.min(...stats.values),
                max: Math.max(...stats.values),
            };
        }

        return result;
    }

    private calculateMean(values: number[]): number {
        return values.reduce((a, b) => a + b, 0) / values.length;
    }

    private calculateMedian(values: number[]): number {
        const sorted = values.sort((a, b) => a - b);
        const mid = Math.floor(sorted.length / 2);
        return sorted.length % 2 === 0
            ? (sorted[mid - 1] + sorted[mid]) / 2
            : sorted[mid];
    }

    private calculateStdDev(values: number[], mean: number): number {
        const variance = values.reduce((sum, val) => sum + Math.pow(val - mean, 2), 0) / values.length;
        return Math.sqrt(variance);
    }

    private calculateMAD(values: number[], median: number): number {
        const deviations = values.map(v => Math.abs(v - median));
        return this.calculateMedian(deviations);
    }

    private calculatePercentile(values: number[], percentile: number): number {
        const sorted = values.sort((a, b) => a - b);
        const index = (percentile / 100) * (sorted.length - 1);
        const lower = Math.floor(index);
        const upper = Math.ceil(index);
        const weight = index - lower;
        
        return sorted[lower] * (1 - weight) + sorted[upper] * weight;
    }
}