/**
 * StatisticalEngine consolidates statistical functions from MonitoringUtils
 * and provides advanced analytics capabilities.
 */

import { UnifiedMetric, MetricSummary } from "../types";

/**
 * Statistical engine for metric analysis
 */
export class StatisticalEngine {
    /**
     * Generate statistical summaries for metrics
     */
    generateSummaries(metrics: UnifiedMetric[]): MetricSummary[] {
        const grouped = this.groupMetricsByName(metrics);
        const summaries: MetricSummary[] = [];
        
        for (const [name, metricGroup] of grouped) {
            const numericValues = metricGroup
                .map(m => typeof m.value === "number" ? m.value : NaN)
                .filter(v => !isNaN(v));
                
            if (numericValues.length === 0) continue;
            
            const summary = this.calculateBasicStats(name, numericValues);
            summary.trend = this.detectTrend(metricGroup);
            summary.changeRate = this.calculateChangeRate(metricGroup);
            summary.anomalyCount = 0; // Will be set by anomaly detection
            summary.anomalyScore = 0;
            
            summaries.push(summary);
        }
        
        return summaries;
    }
    
    /**
     * Calculate basic statistical measures
     */
    calculateBasicStats(name: string, values: number[]): MetricSummary {
        const sorted = [...values].sort((a, b) => a - b);
        const count = values.length;
        const sum = values.reduce((a, b) => a + b, 0);
        const avg = sum / count;
        
        // Calculate standard deviation
        const variance = values.reduce((acc, val) => acc + Math.pow(val - avg, 2), 0) / count;
        const stdDev = Math.sqrt(variance);
        
        return {
            name,
            period: "dynamic", // Will be set by caller
            count,
            sum,
            avg,
            min: sorted[0],
            max: sorted[sorted.length - 1],
            stdDev,
            p50: this.percentile(sorted, 0.5),
            p90: this.percentile(sorted, 0.9),
            p95: this.percentile(sorted, 0.95),
            p99: this.percentile(sorted, 0.99),
            trend: "stable", // Will be calculated
            changeRate: 0, // Will be calculated
            anomalyCount: 0,
            anomalyScore: 0,
        };
    }
    
    /**
     * Calculate percentile
     */
    private percentile(sortedValues: number[], percentile: number): number {
        const index = (sortedValues.length - 1) * percentile;
        const lower = Math.floor(index);
        const upper = Math.ceil(index);
        const weight = index % 1;
        
        if (upper >= sortedValues.length) return sortedValues[sortedValues.length - 1];
        
        return sortedValues[lower] * (1 - weight) + sortedValues[upper] * weight;
    }
    
    /**
     * Group metrics by name
     */
    private groupMetricsByName(metrics: UnifiedMetric[]): Map<string, UnifiedMetric[]> {
        const grouped = new Map<string, UnifiedMetric[]>();
        
        for (const metric of metrics) {
            const existing = grouped.get(metric.name) || [];
            existing.push(metric);
            grouped.set(metric.name, existing);
        }
        
        return grouped;
    }
    
    /**
     * Detect trend in time series data
     */
    private detectTrend(metrics: UnifiedMetric[]): "increasing" | "decreasing" | "stable" {
        if (metrics.length < 3) return "stable";
        
        // Sort by timestamp
        const sorted = metrics
            .filter(m => typeof m.value === "number")
            .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
            
        if (sorted.length < 3) return "stable";
        
        // Simple linear regression
        const n = sorted.length;
        const sumX = sorted.reduce((sum, _, i) => sum + i, 0);
        const sumY = sorted.reduce((sum, m) => sum + (m.value as number), 0);
        const sumXY = sorted.reduce((sum, m, i) => sum + i * (m.value as number), 0);
        const sumXX = sorted.reduce((sum, _, i) => sum + i * i, 0);
        
        const slope = (n * sumXY - sumX * sumY) / (n * sumXX - sumX * sumX);
        
        if (Math.abs(slope) < 0.1) return "stable";
        return slope > 0 ? "increasing" : "decreasing";
    }
    
    /**
     * Calculate rate of change
     */
    private calculateChangeRate(metrics: UnifiedMetric[]): number {
        if (metrics.length < 2) return 0;
        
        const sorted = metrics
            .filter(m => typeof m.value === "number")
            .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
            
        if (sorted.length < 2) return 0;
        
        const first = sorted[0].value as number;
        const last = sorted[sorted.length - 1].value as number;
        
        if (first === 0) return last === 0 ? 0 : Infinity;
        
        return ((last - first) / first) * 100; // Percentage change
    }
    
    /**
     * Calculate moving average
     */
    calculateMovingAverage(values: number[], windowSize: number): number[] {
        const result: number[] = [];
        
        for (let i = 0; i < values.length; i++) {
            const start = Math.max(0, i - windowSize + 1);
            const window = values.slice(start, i + 1);
            const avg = window.reduce((a, b) => a + b, 0) / window.length;
            result.push(avg);
        }
        
        return result;
    }
    
    /**
     * Detect anomalies using Z-score method
     */
    detectAnomaliesZScore(values: number[], threshold: number = 2): number[] {
        if (values.length < 3) return [];
        
        const mean = values.reduce((a, b) => a + b, 0) / values.length;
        const variance = values.reduce((acc, val) => acc + Math.pow(val - mean, 2), 0) / values.length;
        const stdDev = Math.sqrt(variance);
        
        if (stdDev === 0) return [];
        
        const anomalies: number[] = [];
        
        for (let i = 0; i < values.length; i++) {
            const zScore = Math.abs((values[i] - mean) / stdDev);
            if (zScore > threshold) {
                anomalies.push(i);
            }
        }
        
        return anomalies;
    }
    
    /**
     * Detect anomalies using Median Absolute Deviation (MAD)
     */
    detectAnomaliesMAD(values: number[], threshold: number = 2): number[] {
        if (values.length < 3) return [];
        
        const sorted = [...values].sort((a, b) => a - b);
        const median = this.percentile(sorted, 0.5);
        
        const deviations = values.map(v => Math.abs(v - median));
        const sortedDeviations = [...deviations].sort((a, b) => a - b);
        const mad = this.percentile(sortedDeviations, 0.5);
        
        if (mad === 0) return [];
        
        const anomalies: number[] = [];
        
        for (let i = 0; i < values.length; i++) {
            const madScore = Math.abs((values[i] - median) / mad);
            if (madScore > threshold) {
                anomalies.push(i);
            }
        }
        
        return anomalies;
    }
    
    /**
     * Calculate correlation between two series
     */
    calculateCorrelation(x: number[], y: number[]): number {
        if (x.length !== y.length || x.length < 2) return 0;
        
        const n = x.length;
        const sumX = x.reduce((a, b) => a + b, 0);
        const sumY = y.reduce((a, b) => a + b, 0);
        const sumXY = x.reduce((sum, xi, i) => sum + xi * y[i], 0);
        const sumXX = x.reduce((sum, xi) => sum + xi * xi, 0);
        const sumYY = y.reduce((sum, yi) => sum + yi * yi, 0);
        
        const numerator = n * sumXY - sumX * sumY;
        const denominator = Math.sqrt((n * sumXX - sumX * sumX) * (n * sumYY - sumY * sumY));
        
        return denominator === 0 ? 0 : numerator / denominator;
    }
    
    /**
     * Detect seasonality in time series
     */
    detectSeasonality(values: number[], period: number): boolean {
        if (values.length < period * 2) return false;
        
        // Calculate autocorrelation at the specified period
        const correlation = this.calculateAutocorrelation(values, period);
        return Math.abs(correlation) > 0.5; // Threshold for seasonality
    }
    
    /**
     * Calculate autocorrelation at a specific lag
     */
    private calculateAutocorrelation(values: number[], lag: number): number {
        if (lag >= values.length || lag <= 0) return 0;
        
        const n = values.length - lag;
        const x = values.slice(0, n);
        const y = values.slice(lag);
        
        return this.calculateCorrelation(x, y);
    }
}