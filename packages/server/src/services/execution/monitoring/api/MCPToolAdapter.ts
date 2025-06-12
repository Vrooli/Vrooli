/**
 * MCPToolAdapter provides monitoring tools for MCP (Model Context Protocol)
 * integration, allowing AI agents to query and analyze monitoring data.
 */

import { QueryInterface } from "./QueryInterface.js";
import { PatternDetector } from "../analytics/PatternDetector.js";
import { 
    UnifiedMetric, 
    MetricQuery, 
    MetricSummary,
    MetricQueryResult,
} from "../types.js";

/**
 * MCP tool definitions for monitoring
 */
export interface MCPTool {
    name: string;
    description: string;
    parameters: Record<string, any>;
}

/**
 * MCP tool adapter for monitoring functionality
 */
export class MCPToolAdapter {
    constructor(
        private readonly query: QueryInterface,
        private readonly patterns: PatternDetector
    ) {}
    
    /**
     * Get available MCP tools
     */
    getAvailableTools(): MCPTool[] {
        return [
            {
                name: "query_metrics",
                description: "Query monitoring metrics with filters and aggregation",
                parameters: {
                    type: "object",
                    properties: {
                        startTime: { type: "string", format: "date-time", description: "Start time for query" },
                        endTime: { type: "string", format: "date-time", description: "End time for query" },
                        tier: { type: "array", items: { enum: [1, 2, 3, "cross-cutting"] }, description: "Filter by tier" },
                        component: { type: "array", items: { type: "string" }, description: "Filter by component" },
                        type: { type: "array", items: { enum: ["performance", "resource", "health", "business", "safety", "quality", "efficiency", "intelligence"] }, description: "Filter by metric type" },
                        name: { type: "string", description: "Filter by metric name" },
                        executionId: { type: "string", description: "Filter by execution ID" },
                        limit: { type: "number", description: "Maximum number of results" },
                        aggregation: {
                            type: "object",
                            properties: {
                                method: { enum: ["avg", "sum", "min", "max", "count", "percentile"] },
                                percentile: { type: "number", minimum: 0, maximum: 100 },
                                window: { type: "string" }
                            }
                        }
                    }
                }
            },
            {
                name: "get_metric_summary",
                description: "Get statistical summary for specific metrics",
                parameters: {
                    type: "object",
                    properties: {
                        metricNames: { type: "array", items: { type: "string" }, description: "Names of metrics to summarize" },
                        startTime: { type: "string", format: "date-time", description: "Start time for analysis" },
                        endTime: { type: "string", format: "date-time", description: "End time for analysis" },
                        groupBy: { type: "array", items: { type: "string" }, description: "Fields to group by" }
                    },
                    required: ["metricNames", "startTime", "endTime"]
                }
            },
            {
                name: "detect_anomalies",
                description: "Detect anomalies in monitoring metrics",
                parameters: {
                    type: "object",
                    properties: {
                        metricName: { type: "string", description: "Name of metric to analyze" },
                        lookbackMinutes: { type: "number", default: 60, description: "How far back to look for anomalies" },
                        sensitivity: { type: "number", minimum: 1, maximum: 5, default: 3, description: "Anomaly detection sensitivity (1=very sensitive, 5=less sensitive)" }
                    },
                    required: ["metricName"]
                }
            },
            {
                name: "detect_patterns",
                description: "Detect patterns in monitoring metrics",
                parameters: {
                    type: "object",
                    properties: {
                        startTime: { type: "string", format: "date-time", description: "Start time for pattern analysis" },
                        endTime: { type: "string", format: "date-time", description: "End time for pattern analysis" },
                        tier: { type: "array", items: { enum: [1, 2, 3, "cross-cutting"] }, description: "Filter by tier" },
                        component: { type: "string", description: "Filter by component" },
                        minConfidence: { type: "number", minimum: 0, maximum: 1, default: 0.7, description: "Minimum confidence for pattern detection" }
                    },
                    required: ["startTime", "endTime"]
                }
            },
            {
                name: "get_execution_metrics",
                description: "Get all metrics for a specific execution",
                parameters: {
                    type: "object",
                    properties: {
                        executionId: { type: "string", description: "ID of the execution (swarm, run, or step)" },
                        limit: { type: "number", description: "Maximum number of metrics to return" }
                    },
                    required: ["executionId"]
                }
            },
            {
                name: "get_component_health",
                description: "Get health status and metrics for a specific component",
                parameters: {
                    type: "object",
                    properties: {
                        component: { type: "string", description: "Name of the component" },
                        timeRange: { type: "string", enum: ["1h", "6h", "24h", "7d"], default: "1h", description: "Time range for health analysis" }
                    },
                    required: ["component"]
                }
            },
            {
                name: "get_performance_trends",
                description: "Analyze performance trends over time",
                parameters: {
                    type: "object",
                    properties: {
                        metricName: { type: "string", description: "Performance metric to analyze" },
                        timeRange: { type: "string", enum: ["1h", "6h", "24h", "7d", "30d"], default: "24h", description: "Time range for trend analysis" },
                        tier: { enum: [1, 2, 3, "cross-cutting"], description: "Filter by specific tier" }
                    },
                    required: ["metricName"]
                }
            },
            {
                name: "compare_metrics",
                description: "Compare metrics across different time periods or contexts",
                parameters: {
                    type: "object",
                    properties: {
                        metricName: { type: "string", description: "Metric to compare" },
                        baselineStart: { type: "string", format: "date-time", description: "Start of baseline period" },
                        baselineEnd: { type: "string", format: "date-time", description: "End of baseline period" },
                        comparisonStart: { type: "string", format: "date-time", description: "Start of comparison period" },
                        comparisonEnd: { type: "string", format: "date-time", description: "End of comparison period" },
                        groupBy: { type: "string", enum: ["tier", "component", "executionId"], description: "How to group the comparison" }
                    },
                    required: ["metricName", "baselineStart", "baselineEnd", "comparisonStart", "comparisonEnd"]
                }
            }
        ];
    }
    
    /**
     * Execute an MCP tool call
     */
    async executeTool(name: string, parameters: Record<string, any>): Promise<any> {
        switch (name) {
            case "query_metrics":
                return await this.executeQueryMetrics(parameters);
            case "get_metric_summary":
                return await this.executeGetMetricSummary(parameters);
            case "detect_anomalies":
                return await this.executeDetectAnomalies(parameters);
            case "detect_patterns":
                return await this.executeDetectPatterns(parameters);
            case "get_execution_metrics":
                return await this.executeGetExecutionMetrics(parameters);
            case "get_component_health":
                return await this.executeGetComponentHealth(parameters);
            case "get_performance_trends":
                return await this.executeGetPerformanceTrends(parameters);
            case "compare_metrics":
                return await this.executeCompareMetrics(parameters);
            default:
                throw new Error(`Unknown tool: ${name}`);
        }
    }
    
    /**
     * Execute query_metrics tool
     */
    private async executeQueryMetrics(params: any): Promise<MetricQueryResult> {
        const query: MetricQuery = {
            startTime: params.startTime ? new Date(params.startTime) : undefined,
            endTime: params.endTime ? new Date(params.endTime) : undefined,
            tier: params.tier,
            component: params.component,
            type: params.type,
            name: params.name,
            executionId: params.executionId,
            limit: params.limit,
            aggregation: params.aggregation,
        };
        
        return await this.query.execute(query);
    }
    
    /**
     * Execute get_metric_summary tool
     */
    private async executeGetMetricSummary(params: any): Promise<MetricSummary[]> {
        return await this.query.getMetricSummaries(
            params.metricNames,
            new Date(params.startTime),
            new Date(params.endTime),
            params.groupBy
        );
    }
    
    /**
     * Execute detect_anomalies tool
     */
    private async executeDetectAnomalies(params: any): Promise<{
        anomalies: UnifiedMetric[];
        totalAnalyzed: number;
        anomalyRate: number;
        summary: string;
    }> {
        const endTime = new Date();
        const startTime = new Date(endTime.getTime() - (params.lookbackMinutes || 60) * 60 * 1000);
        
        const metrics = await this.query.getTimeRangeMetrics(startTime, endTime, {
            name: params.metricName,
        });
        
        const anomalies = this.patterns.detectAnomalies(metrics);
        const anomalyRate = metrics.length > 0 ? anomalies.length / metrics.length : 0;
        
        return {
            anomalies,
            totalAnalyzed: metrics.length,
            anomalyRate,
            summary: `Found ${anomalies.length} anomalies in ${metrics.length} data points (${(anomalyRate * 100).toFixed(1)}% anomaly rate)`,
        };
    }
    
    /**
     * Execute detect_patterns tool
     */
    private async executeDetectPatterns(params: any): Promise<{
        patterns: Array<{
            type: string;
            confidence: number;
            description: string;
            metricCount: number;
        }>;
        summary: string;
    }> {
        const metrics = await this.query.getTimeRangeMetrics(
            new Date(params.startTime),
            new Date(params.endTime),
            {
                tier: params.tier,
                component: params.component ? [params.component] : undefined,
            }
        );
        
        const patterns = this.patterns.detectPatterns(metrics);
        const minConfidence = params.minConfidence || 0.7;
        const filteredPatterns = patterns.filter(p => p.confidence >= minConfidence);
        
        return {
            patterns: filteredPatterns.map(p => ({
                type: p.type,
                confidence: p.confidence,
                description: p.description,
                metricCount: p.metrics.length,
            })),
            summary: `Found ${filteredPatterns.length} patterns with confidence >= ${minConfidence}`,
        };
    }
    
    /**
     * Execute get_execution_metrics tool
     */
    private async executeGetExecutionMetrics(params: any): Promise<{
        metrics: UnifiedMetric[];
        summary: {
            totalMetrics: number;
            timeSpan: string;
            components: string[];
            types: string[];
        };
    }> {
        const metrics = await this.query.getExecutionMetrics(params.executionId, params.limit);
        
        const components = [...new Set(metrics.map(m => m.component))];
        const types = [...new Set(metrics.map(m => m.type))];
        const timeSpan = this.calculateTimeSpan(metrics);
        
        return {
            metrics,
            summary: {
                totalMetrics: metrics.length,
                timeSpan,
                components,
                types,
            },
        };
    }
    
    /**
     * Execute get_component_health tool
     */
    private async executeGetComponentHealth(params: any): Promise<{
        status: "healthy" | "degraded" | "unhealthy";
        metrics: UnifiedMetric[];
        healthScore: number;
        issues: string[];
        recommendations: string[];
    }> {
        const timeRange = this.parseTimeRange(params.timeRange || "1h");
        const metrics = await this.query.getComponentMetrics(params.component, timeRange);
        
        // Analyze health metrics
        const healthMetrics = metrics.filter(m => m.type === "health");
        const errorMetrics = metrics.filter(m => m.name.includes("error") || m.name.includes("failure"));
        
        let healthScore = 1.0;
        const issues: string[] = [];
        const recommendations: string[] = [];
        
        // Calculate health score based on various factors
        if (healthMetrics.length > 0) {
            const avgHealth = healthMetrics
                .filter(m => typeof m.value === "number")
                .reduce((sum, m) => sum + (m.value as number), 0) / healthMetrics.length;
            healthScore *= avgHealth;
        }
        
        if (errorMetrics.length > 0) {
            const errorRate = errorMetrics.filter(m => typeof m.value === "number" && m.value > 0).length / errorMetrics.length;
            if (errorRate > 0.1) {
                issues.push(`High error rate: ${(errorRate * 100).toFixed(1)}%`);
                recommendations.push("Investigate recent errors and implement fixes");
            }
            healthScore *= (1 - errorRate);
        }
        
        const status = healthScore > 0.8 ? "healthy" : healthScore > 0.5 ? "degraded" : "unhealthy";
        
        return {
            status,
            metrics,
            healthScore,
            issues,
            recommendations,
        };
    }
    
    /**
     * Execute get_performance_trends tool
     */
    private async executeGetPerformanceTrends(params: any): Promise<{
        trend: "improving" | "degrading" | "stable";
        metrics: UnifiedMetric[];
        trendScore: number;
        insights: string[];
    }> {
        const timeRange = this.parseTimeRange(params.timeRange || "24h");
        const metrics = await this.query.getTimeRangeMetrics(timeRange.start, timeRange.end, {
            name: params.metricName,
            tier: params.tier ? [params.tier] : undefined,
            type: ["performance"],
        });
        
        const patterns = this.patterns.detectPatterns(metrics);
        const trendPatterns = patterns.filter(p => p.type === "trend");
        
        let trend: "improving" | "degrading" | "stable" = "stable";
        let trendScore = 0;
        const insights: string[] = [];
        
        if (trendPatterns.length > 0) {
            const strongestTrend = trendPatterns.reduce((max, p) => p.confidence > max.confidence ? p : max);
            trendScore = strongestTrend.confidence;
            
            if (strongestTrend.description.includes("Increasing")) {
                // For performance metrics, increasing usually means degrading
                trend = params.metricName.includes("latency") || params.metricName.includes("duration") ? "degrading" : "improving";
            } else if (strongestTrend.description.includes("Decreasing")) {
                trend = params.metricName.includes("latency") || params.metricName.includes("duration") ? "improving" : "degrading";
            }
            
            insights.push(strongestTrend.description);
        }
        
        return {
            trend,
            metrics,
            trendScore,
            insights,
        };
    }
    
    /**
     * Execute compare_metrics tool
     */
    private async executeCompareMetrics(params: any): Promise<{
        baseline: MetricSummary[];
        comparison: MetricSummary[];
        differences: Array<{
            metric: string;
            baselineValue: number;
            comparisonValue: number;
            percentageChange: number;
            significance: "significant" | "minor" | "negligible";
        }>;
        summary: string;
    }> {
        const baselineMetrics = await this.query.getTimeRangeMetrics(
            new Date(params.baselineStart),
            new Date(params.baselineEnd),
            { name: params.metricName }
        );
        
        const comparisonMetrics = await this.query.getTimeRangeMetrics(
            new Date(params.comparisonStart),
            new Date(params.comparisonEnd),
            { name: params.metricName }
        );
        
        const baseline = await this.query.getMetricSummaries(
            [params.metricName],
            new Date(params.baselineStart),
            new Date(params.baselineEnd),
            params.groupBy ? [params.groupBy] : undefined
        );
        
        const comparison = await this.query.getMetricSummaries(
            [params.metricName],
            new Date(params.comparisonStart),
            new Date(params.comparisonEnd),
            params.groupBy ? [params.groupBy] : undefined
        );
        
        const differences = this.calculateDifferences(baseline, comparison);
        const summary = this.generateComparisonSummary(differences);
        
        return {
            baseline,
            comparison,
            differences,
            summary,
        };
    }
    
    /**
     * Parse time range string to start/end dates
     */
    private parseTimeRange(timeRange: string): { start: Date; end: Date } {
        const end = new Date();
        const start = new Date(end);
        
        switch (timeRange) {
            case "1h":
                start.setHours(start.getHours() - 1);
                break;
            case "6h":
                start.setHours(start.getHours() - 6);
                break;
            case "24h":
                start.setDate(start.getDate() - 1);
                break;
            case "7d":
                start.setDate(start.getDate() - 7);
                break;
            case "30d":
                start.setDate(start.getDate() - 30);
                break;
            default:
                start.setHours(start.getHours() - 1);
        }
        
        return { start, end };
    }
    
    /**
     * Calculate time span for metrics
     */
    private calculateTimeSpan(metrics: UnifiedMetric[]): string {
        if (metrics.length === 0) return "No data";
        
        const times = metrics.map(m => m.timestamp.getTime()).sort((a, b) => a - b);
        const spanMs = times[times.length - 1] - times[0];
        const spanMinutes = spanMs / (1000 * 60);
        
        if (spanMinutes < 60) return `${Math.round(spanMinutes)}m`;
        const spanHours = spanMinutes / 60;
        if (spanHours < 24) return `${Math.round(spanHours)}h`;
        const spanDays = spanHours / 24;
        return `${Math.round(spanDays)}d`;
    }
    
    /**
     * Calculate differences between baseline and comparison summaries
     */
    private calculateDifferences(baseline: MetricSummary[], comparison: MetricSummary[]) {
        const differences: Array<{
            metric: string;
            baselineValue: number;
            comparisonValue: number;
            percentageChange: number;
            significance: "significant" | "minor" | "negligible";
        }> = [];
        
        for (const baseMetric of baseline) {
            const compMetric = comparison.find(c => c.name === baseMetric.name);
            if (!compMetric) continue;
            
            const baseValue = baseMetric.avg;
            const compValue = compMetric.avg;
            const percentageChange = baseValue !== 0 ? ((compValue - baseValue) / baseValue) * 100 : 0;
            
            let significance: "significant" | "minor" | "negligible" = "negligible";
            if (Math.abs(percentageChange) > 20) {
                significance = "significant";
            } else if (Math.abs(percentageChange) > 5) {
                significance = "minor";
            }
            
            differences.push({
                metric: baseMetric.name,
                baselineValue: baseValue,
                comparisonValue: compValue,
                percentageChange,
                significance,
            });
        }
        
        return differences;
    }
    
    /**
     * Generate comparison summary
     */
    private generateComparisonSummary(differences: Array<{ significance: string; percentageChange: number }>): string {
        const significant = differences.filter(d => d.significance === "significant");
        const improving = differences.filter(d => d.percentageChange < -5);
        const degrading = differences.filter(d => d.percentageChange > 5);
        
        let summary = `Analyzed ${differences.length} metrics. `;
        
        if (significant.length > 0) {
            summary += `${significant.length} significant changes detected. `;
        }
        
        if (improving.length > 0) {
            summary += `${improving.length} metrics improved. `;
        }
        
        if (degrading.length > 0) {
            summary += `${degrading.length} metrics degraded. `;
        }
        
        if (significant.length === 0 && improving.length === 0 && degrading.length === 0) {
            summary += "No significant changes detected.";
        }
        
        return summary.trim();
    }
}