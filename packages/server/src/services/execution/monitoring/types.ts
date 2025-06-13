/**
 * Unified monitoring types for the execution architecture.
 * Consolidates metrics from all tiers while maintaining extensibility.
 */

import { type BaseEvent } from "../cross-cutting/events/eventBus.js";

/**
 * Core metric type that unifies all monitoring data across tiers
 */
export interface UnifiedMetric {
    // Core identification
    id: string;
    timestamp: Date;
    tier: 1 | 2 | 3 | "cross-cutting";
    component: string;
    
    // Metric data
    type: MetricType;
    name: string;
    value: number | string | boolean | Record<string, unknown>;
    unit?: string;
    
    // Execution context
    executionId?: string;  // swarmId, runId, or stepId
    userId?: string;
    teamId?: string;
    sessionId?: string;
    
    // Additional context
    metadata?: Record<string, unknown>;
    tags?: string[];
    
    // Sampling and aggregation
    sampleRate?: number;
    aggregationType?: AggregationType;
}

/**
 * Types of metrics collected across the system
 */
export type MetricType = 
    | "performance"     // Execution time, latency, throughput
    | "resource"        // Credits, tokens, memory, CPU
    | "health"          // Error rates, availability, circuit breaker state
    | "business"        // User actions, feature usage, completion rates
    | "safety"          // Security events, compliance, risk scores
    | "quality"         // Accuracy, confidence, validation results
    | "efficiency"      // Cost per operation, resource utilization
    | "intelligence";   // Metacognitive insights, strategy performance

/**
 * How metrics should be aggregated
 */
export type AggregationType = 
    | "gauge"       // Point-in-time value (e.g., memory usage)
    | "counter"     // Cumulative value (e.g., total requests)
    | "histogram"   // Distribution of values (e.g., response times)
    | "summary"     // Statistical summary (e.g., percentiles)
    | "rate";       // Change over time (e.g., requests per second)

/**
 * Retention policy for metrics
 */
export interface RetentionPolicy {
    tier: number | "cross-cutting";
    metricType: MetricType;
    retentionDays: number;
    downsamplingStrategy?: DownsamplingStrategy;
}

export interface DownsamplingStrategy {
    afterDays: number;
    aggregationMethod: "avg" | "max" | "min" | "sum" | "last";
    targetResolution: "1m" | "5m" | "1h" | "1d";
}

/**
 * Query parameters for retrieving metrics
 */
export interface MetricQuery {
    // Time range
    startTime?: Date;
    endTime?: Date;
    
    // Filters
    tier?: Array<1 | 2 | 3 | "cross-cutting">;
    component?: string[];
    type?: MetricType[];
    name?: string | string[];
    executionId?: string;
    userId?: string;
    teamId?: string;
    tags?: string[];
    
    // Aggregation
    groupBy?: Array<"tier" | "component" | "type" | "name" | "executionId" | "userId" | "teamId">;
    aggregation?: {
        method: "avg" | "sum" | "min" | "max" | "count" | "percentile";
        percentile?: number; // For percentile aggregation
        window?: string;     // Time window (e.g., "5m", "1h")
    };
    
    // Pagination
    limit?: number;
    offset?: number;
    
    // Sorting
    orderBy?: "timestamp" | "value" | "name";
    order?: "asc" | "desc";
}

/**
 * Result of a metric query
 */
export interface MetricQueryResult {
    metrics: UnifiedMetric[];
    total: number;
    query: MetricQuery;
    executionTime: number;
}

/**
 * Performance snapshot for swarm execution (Tier 1)
 */
export interface SwarmPerformanceSnapshot {
    swarmId: string;
    timestamp: Date;
    
    // Task metrics
    tasksCompleted: number;
    tasksTotal: number;
    completionRate: number;
    avgTaskDuration: number;
    
    // Resource metrics
    creditsUsed: number;
    tokensConsumed: number;
    toolCallsExecuted: number;
    
    // Quality metrics
    errorRate: number;
    retryCount: number;
    confidenceScore: number;
    
    // Metacognitive insights
    decisionQuality: number;
    adaptationRate: number;
    learningProgress: number;
}

/**
 * Run performance metrics (Tier 2)
 */
export interface RunPerformanceMetrics {
    runId: string;
    routineId: string;
    
    // Timing
    startTime: Date;
    endTime?: Date;
    duration?: number;
    
    // Progress
    stepsCompleted: number;
    stepsTotal: number;
    currentStep?: string;
    
    // Resource usage
    resourceUsage: {
        credits: number;
        tokens: number;
        memory: number;
        toolCalls: number;
    };
    
    // Quality
    errorCount: number;
    warningCount: number;
    successRate: number;
}

/**
 * Strategy execution metrics (Tier 3)
 */
export interface StrategyExecutionMetrics {
    executionId: string;
    strategy: string;
    
    // Performance
    executionTime: number;
    latency: number;
    throughput: number;
    
    // Resource efficiency
    creditsPerOperation: number;
    tokensPerOperation: number;
    cacheHitRate: number;
    
    // Quality
    accuracy: number;
    confidence: number;
    errorRate: number;
    
    // Adaptation
    strategyScore: number;
    adaptationTriggers: string[];
}

/**
 * Resource usage tracking
 */
export interface ResourceUsageMetrics {
    timestamp: Date;
    tier: 1 | 2 | 3;
    executionId: string;
    
    // Computational resources
    credits: {
        allocated: number;
        used: number;
        remaining: number;
    };
    
    tokens: {
        input: number;
        output: number;
        total: number;
    };
    
    // System resources
    memory: {
        allocated: number;
        used: number;
        peak: number;
    };
    
    // Tool usage
    toolCalls: {
        total: number;
        byTool: Record<string, number>;
        failures: number;
    };
    
    // Efficiency metrics
    efficiency: {
        creditUtilization: number;
        tokenEfficiency: number;
        cacheEffectiveness: number;
    };
}

/**
 * Statistical summary for analytics
 */
export interface MetricSummary {
    name: string;
    period: string;
    
    // Basic statistics
    count: number;
    sum: number;
    avg: number;
    min: number;
    max: number;
    stdDev: number;
    
    // Percentiles
    p50: number;
    p90: number;
    p95: number;
    p99: number;
    
    // Trends
    trend: "increasing" | "decreasing" | "stable";
    changeRate: number;
    
    // Anomalies
    anomalyCount: number;
    anomalyScore: number;
}

/**
 * Monitoring event for the event bus
 */
export interface MonitoringEvent extends BaseEvent {
    type: "monitoring.metric" | "monitoring.alert" | "monitoring.report" | "monitoring.anomaly";
    data: {
        metric?: UnifiedMetric;
        alert?: MonitoringAlert;
        report?: MonitoringReport;
        anomaly?: AnomalyDetection;
    };
}

/**
 * Alert generated by monitoring
 */
export interface MonitoringAlert {
    id: string;
    timestamp: Date;
    severity: "info" | "warning" | "error" | "critical";
    source: string;
    message: string;
    metric: UnifiedMetric;
    threshold?: number;
    actualValue: number;
}

/**
 * Monitoring report for agents
 */
export interface MonitoringReport {
    id: string;
    timestamp: Date;
    period: string;
    tier?: 1 | 2 | 3 | "cross-cutting";
    summaries: MetricSummary[];
    insights: string[];
    recommendations: string[];
}

/**
 * Anomaly detection result
 */
export interface AnomalyDetection {
    timestamp: Date;
    metric: UnifiedMetric;
    score: number;
    type: "spike" | "drop" | "pattern" | "threshold";
    confidence: number;
    context: Record<string, unknown>;
}

/**
 * Configuration for the monitoring service
 */
export interface MonitoringConfig {
    // Performance settings
    maxOverheadMs: number;              // Maximum overhead (default: 5ms)
    samplingRates: Record<MetricType, number>;  // Sampling by metric type
    
    // Storage settings
    bufferSizes: Record<string, number>;         // Buffer sizes by tier/component
    retentionPolicies: RetentionPolicy[];       // Retention configuration
    
    // Feature flags
    enabledMetricTypes: MetricType[];           // Which metrics to collect
    enableAnomalyDetection: boolean;            // Enable anomaly detection
    enableAutoDownsampling: boolean;            // Enable automatic downsampling
    
    // Integration settings
    eventBusEnabled: boolean;                   // Emit events to event bus
    mcpToolsEnabled: boolean;                   // Enable MCP monitoring tools
}

/**
 * Consolidated types for adapter consolidation
 */

/**
 * Unified resource usage tracking (for adapter consolidation)
 */
export interface UnifiedResourceUsage {
    credits: number;
    tokens: number;
    toolCalls: number;
    memory?: number;
    cpu?: number;
    custom?: Record<string, number>;
}

/**
 * Unified efficiency metrics (for adapter consolidation)
 */
export interface UnifiedEfficiencyMetrics {
    utilizationRate: number;
    wasteRate: number;
    optimizationScore: number;
    costPerOutcome: number;
    throughput?: number;
    latency?: number;
}

/**
 * Unified performance metrics (for adapter consolidation)
 */
export interface UnifiedPerformanceMetrics {
    duration: number;
    success: boolean;
    resourceUsage: UnifiedResourceUsage;
    error?: string;
    metadata: Record<string, unknown>;
    timestamp: Date;
}

/**
 * Performance summary with percentiles (for adapter consolidation)
 */
export interface PerformanceSummary {
    count: number;
    successRate: number;
    avgDuration: number;
    p50Duration: number;
    p95Duration: number;
    p99Duration: number;
    avgResourceUsage: UnifiedResourceUsage;
    timeRange: {
        start: Date;
        end: Date;
    };
}

/**
 * Cost breakdown by resource type (for adapter consolidation)
 */
export interface CostBreakdown {
    total: number;
    byResource: Record<string, number>;
    byComponent?: Record<string, number>;
    trends: {
        daily: Array<{ date: string; cost: number }>;
        weekly: Array<{ week: string; cost: number }>;
        monthly: Array<{ month: string; cost: number }>;
    };
}

/**
 * Health status for components (for adapter consolidation)
 */
export interface ComponentHealth {
    component: string;
    status: "healthy" | "degraded" | "unhealthy" | "unknown";
    lastCheck: Date;
    metrics: {
        availability: number;
        errorRate: number;
        responseTime: number;
    };
    issues?: string[];
}

/**
 * Optimization suggestion (for adapter consolidation)
 */
export interface OptimizationSuggestion {
    id: string;
    type: "reduce" | "optimize" | "cache" | "parallelize" | "batch";
    targetResource: string;
    currentUsage: number;
    projectedSavings: number;
    implementation: string;
    risk: "low" | "medium" | "high";
    priority: number;
}

/**
 * Strategy effectiveness metrics (for adapter consolidation)
 */
export interface StrategyEffectiveness {
    strategy: string;
    executions: number;
    successRate: number;
    avgDuration: number;
    avgCost: number;
    efficiency: UnifiedEfficiencyMetrics;
    trend: "improving" | "stable" | "degrading";
    recommendedFor: string[];
    notRecommendedFor: string[];
}

/**
 * Intelligence metrics for AI components (for adapter consolidation)
 */
export interface IntelligenceMetrics {
    component: string;
    tier: 1 | 2 | 3;
    metrics: {
        decisionQuality: number;
        adaptationRate: number;
        goalAchievementRate: number;
        learningCurveSlope: number;
        autonomyLevel: number;
    };
    capabilities: {
        current: string[];
        emerging: string[];
        improving: string[];
    };
}

/**
 * Error analysis summary (for adapter consolidation)
 */
export interface ErrorAnalysis {
    totalErrors: number;
    errorsByType: Record<string, number>;
    errorsByComponent: Record<string, number>;
    errorRate: number;
    trends: {
        hourly: number[];
        daily: number[];
    };
    commonPatterns: Array<{
        pattern: string;
        count: number;
        impact: "low" | "medium" | "high";
        suggestedFix?: string;
    }>;
}

/**
 * Event types for monitoring (for adapter consolidation)
 */
export enum MonitoringEventType {
    // Performance events
    EXECUTION_START = "execution.start",
    EXECUTION_COMPLETE = "execution.complete",
    EXECUTION_FAILED = "execution.failed",
    
    // Resource events
    RESOURCE_ALLOCATED = "resource.allocated",
    RESOURCE_RELEASED = "resource.released",
    RESOURCE_LIMIT_REACHED = "resource.limit_reached",
    
    // Health events
    HEALTH_CHECK = "health.check",
    COMPONENT_ERROR = "component.error",
    COMPONENT_RECOVERED = "component.recovered",
    
    // Intelligence events
    DECISION_MADE = "decision.made",
    GOAL_ACHIEVED = "goal.achieved",
    STRATEGY_ADAPTED = "strategy.adapted",
}

/**
 * Time window for queries (for adapter consolidation)
 */
export interface TimeWindow {
    start: Date;
    end: Date;
    granularity?: "minute" | "hour" | "day" | "week" | "month";
}

/**
 * Query options for metrics (for adapter consolidation)
 */
export interface MetricQueryOptions {
    timeWindow?: TimeWindow;
    filters?: Record<string, any>;
    groupBy?: string[];
    orderBy?: string;
    limit?: number;
    includeMetadata?: boolean;
}