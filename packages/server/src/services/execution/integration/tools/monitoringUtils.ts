/**
 * Utility functions for monitoring operations
 * Provides common statistical and analytical functions for monitoring tools
 */

/**
 * Time series data point
 */
export interface TimeSeriesPoint {
    timestamp: Date;
    value: number;
    metadata?: Record<string, unknown>;
}

/**
 * Statistical summary
 */
export interface StatisticalSummary {
    count: number;
    sum: number;
    mean: number;
    median: number;
    min: number;
    max: number;
    stdDev: number;
    variance: number;
    percentiles: {
        p25: number;
        p50: number;
        p75: number;
        p90: number;
        p95: number;
        p99: number;
    };
}

/**
 * Anomaly detection result
 */
export interface AnomalyResult {
    isAnomaly: boolean;
    score: number;
    severity: "low" | "medium" | "high" | "critical";
    method: string;
    threshold: number;
    context?: Record<string, unknown>;
}

/**
 * Pattern detection result
 */
export interface PatternResult {
    type: string;
    confidence: number;
    description: string;
    timeRange: {
        start: Date;
        end: Date;
    };
    samples: any[];
    metadata: Record<string, unknown>;
}

/**
 * MonitoringUtils - Common utilities for monitoring operations
 */
export class MonitoringUtils {
    /**
     * Calculate comprehensive statistical summary
     */
    static calculateStatistics(values: number[]): StatisticalSummary {
        if (values.length === 0) {
            throw new Error("Cannot calculate statistics on empty array");
        }

        const sorted = [...values].sort((a, b) => a - b);
        const count = values.length;
        const sum = values.reduce((a, b) => a + b, 0);
        const mean = sum / count;

        // Calculate variance and standard deviation
        const variance = values.reduce((acc, val) => acc + Math.pow(val - mean, 2), 0) / count;
        const stdDev = Math.sqrt(variance);

        // Calculate percentiles
        const percentiles = {
            p25: this.calculatePercentile(sorted, 25),
            p50: this.calculatePercentile(sorted, 50),
            p75: this.calculatePercentile(sorted, 75),
            p90: this.calculatePercentile(sorted, 90),
            p95: this.calculatePercentile(sorted, 95),
            p99: this.calculatePercentile(sorted, 99),
        };

        return {
            count,
            sum,
            mean,
            median: percentiles.p50,
            min: sorted[0],
            max: sorted[count - 1],
            stdDev,
            variance,
            percentiles,
        };
    }

    /**
     * Calculate percentile value
     */
    static calculatePercentile(sortedValues: number[], percentile: number): number {
        if (percentile < 0 || percentile > 100) {
            throw new Error("Percentile must be between 0 and 100");
        }

        const index = (percentile / 100) * (sortedValues.length - 1);
        const lower = Math.floor(index);
        const upper = Math.ceil(index);
        const weight = index - lower;

        if (lower === upper) {
            return sortedValues[lower];
        }

        return sortedValues[lower] * (1 - weight) + sortedValues[upper] * weight;
    }

    /**
     * Detect anomalies using Z-score method
     */
    static detectZScoreAnomalies(
        values: number[],
        threshold: number = 3,
    ): AnomalyResult[] {
        if (values.length < 3) {
            return [];
        }

        const stats = this.calculateStatistics(values);
        const results: AnomalyResult[] = [];

        for (let i = 0; i < values.length; i++) {
            const zScore = Math.abs((values[i] - stats.mean) / stats.stdDev);
            const isAnomaly = zScore > threshold;

            if (isAnomaly) {
                let severity: "low" | "medium" | "high" | "critical";
                if (zScore > 5) severity = "critical";
                else if (zScore > 4) severity = "high";
                else if (zScore > 3) severity = "medium";
                else severity = "low";

                results.push({
                    isAnomaly,
                    score: zScore,
                    severity,
                    method: "zscore",
                    threshold,
                    context: {
                        index: i,
                        value: values[i],
                        mean: stats.mean,
                        stdDev: stats.stdDev,
                        deviation: values[i] - stats.mean,
                    },
                });
            }
        }

        return results;
    }

    /**
     * Detect anomalies using Modified Z-score (MAD) method
     */
    static detectMADAnomalies(
        values: number[],
        threshold: number = 3.5,
    ): AnomalyResult[] {
        if (values.length < 3) {
            return [];
        }

        const median = this.calculatePercentile([...values].sort((a, b) => a - b), 50);
        const deviations = values.map(v => Math.abs(v - median));
        const mad = this.calculatePercentile([...deviations].sort((a, b) => a - b), 50);

        const results: AnomalyResult[] = [];

        for (let i = 0; i < values.length; i++) {
            const modifiedZScore = 0.6745 * (values[i] - median) / mad;
            const isAnomaly = Math.abs(modifiedZScore) > threshold;

            if (isAnomaly) {
                let severity: "low" | "medium" | "high" | "critical";
                const absScore = Math.abs(modifiedZScore);
                if (absScore > 5) severity = "critical";
                else if (absScore > 4) severity = "high";
                else if (absScore > 3.5) severity = "medium";
                else severity = "low";

                results.push({
                    isAnomaly,
                    score: Math.abs(modifiedZScore),
                    severity,
                    method: "mad",
                    threshold,
                    context: {
                        index: i,
                        value: values[i],
                        median,
                        mad,
                        modifiedZScore,
                    },
                });
            }
        }

        return results;
    }

    /**
     * Detect anomalies using percentile method
     */
    static detectPercentileAnomalies(
        values: number[],
        lowerPercentile: number = 5,
        upperPercentile: number = 95,
    ): AnomalyResult[] {
        if (values.length < 3) {
            return [];
        }

        const sorted = [...values].sort((a, b) => a - b);
        const lowerBound = this.calculatePercentile(sorted, lowerPercentile);
        const upperBound = this.calculatePercentile(sorted, upperPercentile);

        const results: AnomalyResult[] = [];

        for (let i = 0; i < values.length; i++) {
            const isAnomaly = values[i] < lowerBound || values[i] > upperBound;

            if (isAnomaly) {
                const distance = values[i] < lowerBound 
                    ? lowerBound - values[i]
                    : values[i] - upperBound;
                
                const range = upperBound - lowerBound;
                const score = distance / range;

                let severity: "low" | "medium" | "high" | "critical";
                if (score > 2) severity = "critical";
                else if (score > 1) severity = "high";
                else if (score > 0.5) severity = "medium";
                else severity = "low";

                results.push({
                    isAnomaly,
                    score,
                    severity,
                    method: "percentile",
                    threshold: score,
                    context: {
                        index: i,
                        value: values[i],
                        lowerBound,
                        upperBound,
                        position: values[i] < lowerBound ? "below" : "above",
                        distance,
                    },
                });
            }
        }

        return results;
    }

    /**
     * Detect trends in time series data
     */
    static detectTrend(timeSeries: TimeSeriesPoint[]): {
        trend: "increasing" | "decreasing" | "stable";
        confidence: number;
        slope: number;
        correlation: number;
    } {
        if (timeSeries.length < 3) {
            return {
                trend: "stable",
                confidence: 0,
                slope: 0,
                correlation: 0,
            };
        }

        // Convert timestamps to numeric values (milliseconds since epoch)
        const x = timeSeries.map(point => point.timestamp.getTime());
        const y = timeSeries.map(point => point.value);

        // Calculate linear regression
        const n = x.length;
        const sumX = x.reduce((a, b) => a + b, 0);
        const sumY = y.reduce((a, b) => a + b, 0);
        const sumXY = x.reduce((sum, xi, i) => sum + xi * y[i], 0);
        const sumXX = x.reduce((sum, xi) => sum + xi * xi, 0);
        const sumYY = y.reduce((sum, yi) => sum + yi * yi, 0);

        const slope = (n * sumXY - sumX * sumY) / (n * sumXX - sumX * sumX);
        const correlation = (n * sumXY - sumX * sumY) / 
            Math.sqrt((n * sumXX - sumX * sumX) * (n * sumYY - sumY * sumY));

        // Determine trend
        let trend: "increasing" | "decreasing" | "stable";
        if (Math.abs(slope) < 1e-10) {
            trend = "stable";
        } else if (slope > 0) {
            trend = "increasing";
        } else {
            trend = "decreasing";
        }

        // Confidence based on correlation strength
        const confidence = Math.abs(correlation);

        return {
            trend,
            confidence,
            slope,
            correlation,
        };
    }

    /**
     * Detect seasonal patterns in time series data
     */
    static detectSeasonality(
        timeSeries: TimeSeriesPoint[],
        expectedPeriod?: number, // in milliseconds
    ): {
        hasSeasonality: boolean;
        period?: number;
        strength: number;
        peaks: number[];
    } {
        if (timeSeries.length < 10) {
            return {
                hasSeasonality: false,
                strength: 0,
                peaks: [],
            };
        }

        const values = timeSeries.map(p => p.value);
        
        // If expected period is provided, check for that specific seasonality
        if (expectedPeriod) {
            const intervals = timeSeries.map((p, i) => 
                i === 0 ? 0 : p.timestamp.getTime() - timeSeries[i - 1].timestamp.getTime()
            );
            const avgInterval = intervals.slice(1).reduce((a, b) => a + b, 0) / (intervals.length - 1);
            const periodPoints = Math.round(expectedPeriod / avgInterval);
            
            if (periodPoints < values.length / 2) {
                const autocorr = this.calculateAutocorrelation(values, periodPoints);
                return {
                    hasSeasonality: Math.abs(autocorr) > 0.5,
                    period: expectedPeriod,
                    strength: Math.abs(autocorr),
                    peaks: autocorr > 0.5 ? [periodPoints] : [],
                };
            }
        }

        // Search for seasonality using autocorrelation
        const maxLag = Math.min(Math.floor(values.length / 2), 50);
        const autocorrelations: number[] = [];
        
        for (let lag = 1; lag <= maxLag; lag++) {
            autocorrelations.push(this.calculateAutocorrelation(values, lag));
        }

        // Find peaks in autocorrelation
        const peaks: number[] = [];
        for (let i = 1; i < autocorrelations.length - 1; i++) {
            if (autocorrelations[i] > autocorrelations[i - 1] && 
                autocorrelations[i] > autocorrelations[i + 1] &&
                autocorrelations[i] > 0.3) {
                peaks.push(i + 1);
            }
        }

        const strength = peaks.length > 0 ? Math.max(...peaks.map(p => autocorrelations[p - 1])) : 0;

        return {
            hasSeasonality: peaks.length > 0 && strength > 0.5,
            period: peaks.length > 0 ? peaks[0] : undefined,
            strength,
            peaks,
        };
    }

    /**
     * Calculate autocorrelation at a specific lag
     */
    private static calculateAutocorrelation(values: number[], lag: number): number {
        if (lag >= values.length) {
            return 0;
        }

        const n = values.length - lag;
        const mean = values.reduce((a, b) => a + b, 0) / values.length;
        
        let numerator = 0;
        let denominator = 0;

        for (let i = 0; i < n; i++) {
            numerator += (values[i] - mean) * (values[i + lag] - mean);
        }

        for (let i = 0; i < values.length; i++) {
            denominator += Math.pow(values[i] - mean, 2);
        }

        return denominator === 0 ? 0 : numerator / denominator;
    }

    /**
     * Detect change points in time series data
     */
    static detectChangePoints(
        timeSeries: TimeSeriesPoint[],
        minSegmentLength: number = 5,
    ): {
        changePoints: number[];
        segments: Array<{
            start: number;
            end: number;
            mean: number;
            variance: number;
        }>;
    } {
        if (timeSeries.length < minSegmentLength * 2) {
            return {
                changePoints: [],
                segments: [{
                    start: 0,
                    end: timeSeries.length - 1,
                    mean: this.calculateStatistics(timeSeries.map(p => p.value)).mean,
                    variance: this.calculateStatistics(timeSeries.map(p => p.value)).variance,
                }],
            };
        }

        const values = timeSeries.map(p => p.value);
        const changePoints: number[] = [];

        // Simple change point detection using variance
        for (let i = minSegmentLength; i < values.length - minSegmentLength; i++) {
            const leftSegment = values.slice(0, i);
            const rightSegment = values.slice(i);
            
            const leftStats = this.calculateStatistics(leftSegment);
            const rightStats = this.calculateStatistics(rightSegment);
            
            // Use t-test-like statistic to detect significant difference
            const pooledVariance = ((leftSegment.length - 1) * leftStats.variance + 
                                   (rightSegment.length - 1) * rightStats.variance) /
                                   (leftSegment.length + rightSegment.length - 2);
            
            const standardError = Math.sqrt(pooledVariance * 
                (1 / leftSegment.length + 1 / rightSegment.length));
            
            if (standardError > 0) {
                const tStatistic = Math.abs(leftStats.mean - rightStats.mean) / standardError;
                
                // If t-statistic is large enough, consider it a change point
                if (tStatistic > 2.5) {
                    changePoints.push(i);
                }
            }
        }

        // Create segments based on change points
        const segments: Array<{
            start: number;
            end: number;
            mean: number;
            variance: number;
        }> = [];

        let start = 0;
        for (const changePoint of changePoints) {
            const segmentValues = values.slice(start, changePoint);
            const stats = this.calculateStatistics(segmentValues);
            segments.push({
                start,
                end: changePoint - 1,
                mean: stats.mean,
                variance: stats.variance,
            });
            start = changePoint;
        }

        // Add final segment
        const finalSegmentValues = values.slice(start);
        const finalStats = this.calculateStatistics(finalSegmentValues);
        segments.push({
            start,
            end: values.length - 1,
            mean: finalStats.mean,
            variance: finalStats.variance,
        });

        return {
            changePoints,
            segments,
        };
    }

    /**
     * Calculate service level indicators
     */
    static calculateSLI(
        measurements: number[],
        goodThreshold: number,
        comparison: "gte" | "lte" = "gte",
    ): {
        sli: number; // Percentage of good measurements
        goodEvents: number;
        totalEvents: number;
        worstCase: number;
        bestCase: number;
    } {
        if (measurements.length === 0) {
            return {
                sli: 0,
                goodEvents: 0,
                totalEvents: 0,
                worstCase: 0,
                bestCase: 0,
            };
        }

        const goodEvents = measurements.filter(measurement => {
            return comparison === "gte" 
                ? measurement >= goodThreshold
                : measurement <= goodThreshold;
        }).length;

        const sli = (goodEvents / measurements.length) * 100;

        return {
            sli,
            goodEvents,
            totalEvents: measurements.length,
            worstCase: Math.min(...measurements),
            bestCase: Math.max(...measurements),
        };
    }

    /**
     * Generate monitoring insights from patterns
     */
    static generateInsights(patterns: PatternResult[]): string[] {
        const insights: string[] = [];

        for (const pattern of patterns) {
            switch (pattern.type) {
                case "bottleneck":
                    if (pattern.confidence > 0.8) {
                        insights.push(
                            `High confidence bottleneck detected. ${pattern.description}. ` +
                            `Consider scaling or optimizing affected components.`
                        );
                    }
                    break;

                case "error_cluster":
                    if (pattern.confidence > 0.7) {
                        insights.push(
                            `Error clustering detected. ${pattern.description}. ` +
                            `Investigate potential systemic issues.`
                        );
                    }
                    break;

                case "resource_spike":
                    insights.push(
                        `Resource usage spike detected. ${pattern.description}. ` +
                        `Monitor resource allocation and consider auto-scaling.`
                    );
                    break;

                case "performance_degradation":
                    if (pattern.confidence > 0.6) {
                        insights.push(
                            `Performance degradation trend detected. ${pattern.description}. ` +
                            `Review recent changes and monitor system health.`
                        );
                    }
                    break;

                default:
                    if (pattern.confidence > 0.7) {
                        insights.push(
                            `Pattern detected: ${pattern.type}. ${pattern.description}.`
                        );
                    }
            }
        }

        return insights;
    }
}