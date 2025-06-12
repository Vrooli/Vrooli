/**
 * QueryInterface provides a unified API for querying metrics
 * across all storage backends.
 */

import { MetricsStore } from "../core/MetricsStore";
import { StatisticalEngine } from "../analytics/StatisticalEngine";
import { 
    MetricQuery, 
    MetricQueryResult, 
    UnifiedMetric,
    MetricSummary,
} from "../types";

/**
 * Unified query interface for metrics
 */
export class QueryInterface {
    constructor(
        private readonly store: MetricsStore,
        private readonly stats: StatisticalEngine
    ) {}
    
    /**
     * Execute a metric query
     */
    async execute(query: MetricQuery): Promise<MetricQueryResult> {
        return await this.store.query(query);
    }
    
    /**
     * Get metrics for a specific execution context
     */
    async getExecutionMetrics(
        executionId: string,
        limit?: number
    ): Promise<UnifiedMetric[]> {
        return await this.store.getByExecutionId(executionId, limit);
    }
    
    /**
     * Get recent metrics by type
     */
    async getRecentMetrics(
        type: "performance" | "resource" | "health" | "business" | "safety" | "quality" | "efficiency" | "intelligence",
        count: number = 100,
        tier?: 1 | 2 | 3 | "cross-cutting"
    ): Promise<UnifiedMetric[]> {
        return await this.store.getRecent(type, count, tier);
    }
    
    /**
     * Get metrics for a specific time range
     */
    async getTimeRangeMetrics(
        startTime: Date,
        endTime: Date,
        filters?: Partial<MetricQuery>
    ): Promise<UnifiedMetric[]> {
        const query: MetricQuery = {
            startTime,
            endTime,
            ...filters,
        };
        
        const result = await this.execute(query);
        return result.metrics;
    }
    
    /**
     * Get metric summaries with statistical analysis
     */
    async getMetricSummaries(
        metricNames: string[],
        startTime: Date,
        endTime: Date,
        groupBy?: string[]
    ): Promise<MetricSummary[]> {
        const metrics = await this.getTimeRangeMetrics(startTime, endTime, {
            name: metricNames,
            groupBy: groupBy as any,
        });
        
        return this.stats.generateSummaries(metrics);
    }
    
    /**
     * Get aggregated metrics
     */
    async getAggregatedMetrics(
        query: MetricQuery & {
            aggregation: {
                method: "avg" | "sum" | "min" | "max" | "count" | "percentile";
                percentile?: number;
                window?: string;
            };
        }
    ): Promise<{
        name: string;
        value: number;
        aggregation: string;
        period: string;
        dataPoints: number;
    }[]> {
        const result = await this.execute(query);
        const metrics = result.metrics;
        
        if (metrics.length === 0) return [];
        
        // Group metrics for aggregation
        const grouped = this.groupMetrics(metrics, query.groupBy || ["name"]);
        const aggregated: {
            name: string;
            value: number;
            aggregation: string;
            period: string;
            dataPoints: number;
        }[] = [];
        
        for (const [groupKey, groupMetrics] of grouped) {
            const numericValues = groupMetrics
                .map(m => typeof m.value === "number" ? m.value : NaN)
                .filter(v => !isNaN(v));
                
            if (numericValues.length === 0) continue;
            
            let aggregatedValue: number;
            
            switch (query.aggregation.method) {
                case "avg":
                    aggregatedValue = numericValues.reduce((a, b) => a + b, 0) / numericValues.length;
                    break;
                case "sum":
                    aggregatedValue = numericValues.reduce((a, b) => a + b, 0);
                    break;
                case "min":
                    aggregatedValue = Math.min(...numericValues);
                    break;
                case "max":
                    aggregatedValue = Math.max(...numericValues);
                    break;
                case "count":
                    aggregatedValue = numericValues.length;
                    break;
                case "percentile":
                    const percentile = query.aggregation.percentile || 50;
                    aggregatedValue = this.calculatePercentile(numericValues, percentile / 100);
                    break;
                default:
                    aggregatedValue = numericValues.reduce((a, b) => a + b, 0) / numericValues.length;
            }
            
            aggregated.push({
                name: groupKey,
                value: aggregatedValue,
                aggregation: query.aggregation.method,
                period: this.formatPeriod(query.startTime, query.endTime),
                dataPoints: numericValues.length,
            });
        }
        
        return aggregated;
    }
    
    /**
     * Search metrics by text
     */
    async searchMetrics(
        searchText: string,
        limit: number = 100
    ): Promise<UnifiedMetric[]> {
        const query: MetricQuery = {
            limit,
            orderBy: "timestamp",
            order: "desc",
        };
        
        const result = await this.execute(query);
        
        // Filter by search text
        const searchLower = searchText.toLowerCase();
        return result.metrics.filter(metric =>
            metric.name.toLowerCase().includes(searchLower) ||
            metric.component.toLowerCase().includes(searchLower) ||
            (typeof metric.value === "string" && metric.value.toLowerCase().includes(searchLower)) ||
            metric.tags?.some(tag => tag.toLowerCase().includes(searchLower))
        );
    }
    
    /**
     * Get top metrics by value
     */
    async getTopMetrics(
        metricName: string,
        limit: number = 10,
        timeRange?: { start: Date; end: Date }
    ): Promise<UnifiedMetric[]> {
        const query: MetricQuery = {
            name: metricName,
            startTime: timeRange?.start,
            endTime: timeRange?.end,
            orderBy: "value",
            order: "desc",
            limit,
        };
        
        const result = await this.execute(query);
        return result.metrics.filter(m => typeof m.value === "number");
    }
    
    /**
     * Get metrics by component
     */
    async getComponentMetrics(
        component: string,
        timeRange?: { start: Date; end: Date },
        limit?: number
    ): Promise<UnifiedMetric[]> {
        const query: MetricQuery = {
            component: [component],
            startTime: timeRange?.start,
            endTime: timeRange?.end,
            limit,
            orderBy: "timestamp",
            order: "desc",
        };
        
        const result = await this.execute(query);
        return result.metrics;
    }
    
    /**
     * Get metrics by tier
     */
    async getTierMetrics(
        tier: 1 | 2 | 3 | "cross-cutting",
        timeRange?: { start: Date; end: Date },
        limit?: number
    ): Promise<UnifiedMetric[]> {
        const query: MetricQuery = {
            tier: [tier],
            startTime: timeRange?.start,
            endTime: timeRange?.end,
            limit,
            orderBy: "timestamp",
            order: "desc",
        };
        
        const result = await this.execute(query);
        return result.metrics;
    }
    
    /**
     * Group metrics by specified fields
     */
    private groupMetrics(
        metrics: UnifiedMetric[], 
        groupBy: string[]
    ): Map<string, UnifiedMetric[]> {
        const grouped = new Map<string, UnifiedMetric[]>();
        
        for (const metric of metrics) {
            const groupKey = groupBy.map(field => {
                switch (field) {
                    case "tier": return metric.tier;
                    case "component": return metric.component;
                    case "type": return metric.type;
                    case "name": return metric.name;
                    case "executionId": return metric.executionId || "unknown";
                    case "userId": return metric.userId || "unknown";
                    case "teamId": return metric.teamId || "unknown";
                    default: return "unknown";
                }
            }).join("|");
            
            const existing = grouped.get(groupKey) || [];
            existing.push(metric);
            grouped.set(groupKey, existing);
        }
        
        return grouped;
    }
    
    /**
     * Calculate percentile
     */
    private calculatePercentile(values: number[], percentile: number): number {
        const sorted = [...values].sort((a, b) => a - b);
        const index = (sorted.length - 1) * percentile;
        const lower = Math.floor(index);
        const upper = Math.ceil(index);
        const weight = index % 1;
        
        if (upper >= sorted.length) return sorted[sorted.length - 1];
        
        return sorted[lower] * (1 - weight) + sorted[upper] * weight;
    }
    
    /**
     * Format time period for display
     */
    private formatPeriod(startTime?: Date, endTime?: Date): string {
        if (!startTime || !endTime) return "all-time";
        
        const durationMs = endTime.getTime() - startTime.getTime();
        const hours = durationMs / (1000 * 60 * 60);
        
        if (hours < 1) return `${Math.round(hours * 60)}m`;
        if (hours < 24) return `${Math.round(hours)}h`;
        const days = Math.round(hours / 24);
        if (days < 7) return `${days}d`;
        const weeks = Math.round(days / 7);
        return `${weeks}w`;
    }
}