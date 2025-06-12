/**
 * PatternDetector provides anomaly detection and pattern recognition
 * for monitoring metrics.
 */

import { UnifiedMetric, AnomalyDetection } from "../types";
import { StatisticalEngine } from "./StatisticalEngine";

/**
 * Pattern detection and anomaly identification
 */
export class PatternDetector {
    constructor(private readonly stats: StatisticalEngine) {}
    
    /**
     * Detect anomalies in a series of metrics
     */
    detectAnomalies(metrics: UnifiedMetric[]): UnifiedMetric[] {
        if (metrics.length < 5) return [];
        
        // Group by metric name and detect anomalies for each group
        const grouped = this.groupByName(metrics);
        const anomalies: UnifiedMetric[] = [];
        
        for (const [name, group] of grouped) {
            const groupAnomalies = this.detectAnomaliesInGroup(group);
            anomalies.push(...groupAnomalies);
        }
        
        return anomalies;
    }
    
    /**
     * Detect anomalies within a single metric group
     */
    private detectAnomaliesInGroup(metrics: UnifiedMetric[]): UnifiedMetric[] {
        const numericMetrics = metrics.filter(m => typeof m.value === "number");
        if (numericMetrics.length < 5) return [];
        
        const values = numericMetrics.map(m => m.value as number);
        
        // Use multiple detection methods and combine results
        const zScoreAnomalies = this.stats.detectAnomaliesZScore(values, 2.5);
        const madAnomalies = this.stats.detectAnomaliesMAD(values, 2.5);
        const changePointAnomalies = this.detectChangePoints(values);
        
        // Combine all anomaly indices
        const allAnomalyIndices = new Set([
            ...zScoreAnomalies,
            ...madAnomalies,
            ...changePointAnomalies,
        ]);
        
        return Array.from(allAnomalyIndices).map(index => numericMetrics[index]);
    }
    
    /**
     * Detect sudden change points in data
     */
    private detectChangePoints(values: number[]): number[] {
        if (values.length < 10) return [];
        
        const changePoints: number[] = [];
        const windowSize = Math.min(5, Math.floor(values.length / 4));
        
        for (let i = windowSize; i < values.length - windowSize; i++) {
            const before = values.slice(i - windowSize, i);
            const after = values.slice(i, i + windowSize);
            
            const beforeMean = before.reduce((a, b) => a + b, 0) / before.length;
            const afterMean = after.reduce((a, b) => a + b, 0) / after.length;
            
            const beforeStd = this.calculateStandardDeviation(before);
            const afterStd = this.calculateStandardDeviation(after);
            
            // Calculate t-statistic for difference in means
            const pooledStd = Math.sqrt((beforeStd * beforeStd + afterStd * afterStd) / 2);
            
            if (pooledStd > 0) {
                const tStatistic = Math.abs(beforeMean - afterMean) / (pooledStd * Math.sqrt(2 / windowSize));
                
                // Threshold for significant change
                if (tStatistic > 2.5) {
                    changePoints.push(i);
                }
            }
        }
        
        return changePoints;
    }
    
    /**
     * Detect patterns in metric sequences
     */
    detectPatterns(metrics: UnifiedMetric[]): {
        type: "periodic" | "trend" | "spike" | "drop" | "oscillation";
        confidence: number;
        description: string;
        metrics: UnifiedMetric[];
    }[] {
        const patterns: {
            type: "periodic" | "trend" | "spike" | "drop" | "oscillation";
            confidence: number;
            description: string;
            metrics: UnifiedMetric[];
        }[] = [];
        
        const grouped = this.groupByName(metrics);
        
        for (const [name, group] of grouped) {
            if (group.length < 10) continue;
            
            const numericGroup = group.filter(m => typeof m.value === "number");
            if (numericGroup.length < 10) continue;
            
            const values = numericGroup.map(m => m.value as number);
            
            // Detect different pattern types
            const trendPattern = this.detectTrendPattern(numericGroup, values);
            if (trendPattern) patterns.push(trendPattern);
            
            const spikePattern = this.detectSpikePattern(numericGroup, values);
            if (spikePattern) patterns.push(spikePattern);
            
            const periodicPattern = this.detectPeriodicPattern(numericGroup, values);
            if (periodicPattern) patterns.push(periodicPattern);
            
            const oscillationPattern = this.detectOscillationPattern(numericGroup, values);
            if (oscillationPattern) patterns.push(oscillationPattern);
        }
        
        return patterns;
    }
    
    /**
     * Detect trend patterns
     */
    private detectTrendPattern(metrics: UnifiedMetric[], values: number[]) {
        const correlation = this.calculateTimeCorrelation(metrics, values);
        const threshold = 0.7;
        
        if (Math.abs(correlation) > threshold) {
            return {
                type: "trend" as const,
                confidence: Math.abs(correlation),
                description: `${correlation > 0 ? "Increasing" : "Decreasing"} trend detected (r=${correlation.toFixed(3)})`,
                metrics,
            };
        }
        
        return null;
    }
    
    /**
     * Detect spike patterns
     */
    private detectSpikePattern(metrics: UnifiedMetric[], values: number[]) {
        const anomalyIndices = this.stats.detectAnomaliesZScore(values, 3);
        
        if (anomalyIndices.length > 0) {
            const anomalyMetrics = anomalyIndices.map(i => metrics[i]);
            const mean = values.reduce((a, b) => a + b, 0) / values.length;
            const maxSpike = Math.max(...anomalyIndices.map(i => Math.abs(values[i] - mean)));
            
            return {
                type: maxSpike > mean ? "spike" as const : "drop" as const,
                confidence: Math.min(1, maxSpike / (mean * 2)),
                description: `${anomalyIndices.length} ${maxSpike > mean ? "spike(s)" : "drop(s)"} detected`,
                metrics: anomalyMetrics,
            };
        }
        
        return null;
    }
    
    /**
     * Detect periodic patterns
     */
    private detectPeriodicPattern(metrics: UnifiedMetric[], values: number[]) {
        // Test for common periods (every hour, day, week)
        const periods = [24, 48, 168]; // hours
        
        for (const period of periods) {
            if (this.stats.detectSeasonality(values, period)) {
                return {
                    type: "periodic" as const,
                    confidence: 0.8,
                    description: `Periodic pattern detected with period of ${period} data points`,
                    metrics,
                };
            }
        }
        
        return null;
    }
    
    /**
     * Detect oscillation patterns
     */
    private detectOscillationPattern(metrics: UnifiedMetric[], values: number[]) {
        if (values.length < 6) return null;
        
        // Count direction changes
        let directionChanges = 0;
        let lastDirection: "up" | "down" | null = null;
        
        for (let i = 1; i < values.length; i++) {
            const currentDirection = values[i] > values[i - 1] ? "up" : "down";
            
            if (lastDirection && currentDirection !== lastDirection) {
                directionChanges++;
            }
            
            lastDirection = currentDirection;
        }
        
        const oscillationRate = directionChanges / (values.length - 1);
        
        if (oscillationRate > 0.4) { // Threshold for oscillation
            return {
                type: "oscillation" as const,
                confidence: Math.min(1, oscillationRate * 2),
                description: `Oscillating pattern detected (${directionChanges} direction changes)`,
                metrics,
            };
        }
        
        return null;
    }
    
    /**
     * Group metrics by name
     */
    private groupByName(metrics: UnifiedMetric[]): Map<string, UnifiedMetric[]> {
        const grouped = new Map<string, UnifiedMetric[]>();
        
        for (const metric of metrics) {
            const existing = grouped.get(metric.name) || [];
            existing.push(metric);
            grouped.set(metric.name, existing);
        }
        
        return grouped;
    }
    
    /**
     * Calculate correlation between time and values
     */
    private calculateTimeCorrelation(metrics: UnifiedMetric[], values: number[]): number {
        const times = metrics.map((m, i) => i); // Use index as time proxy
        return this.stats.calculateCorrelation(times, values);
    }
    
    /**
     * Calculate standard deviation
     */
    private calculateStandardDeviation(values: number[]): number {
        const mean = values.reduce((a, b) => a + b, 0) / values.length;
        const variance = values.reduce((acc, val) => acc + Math.pow(val - mean, 2), 0) / values.length;
        return Math.sqrt(variance);
    }
}