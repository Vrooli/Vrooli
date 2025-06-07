/**
 * Type definitions for the monitoring and telemetry system
 * Provides structured schemas for performance, health, and business events
 */

import type { BaseEvent } from "./events.js";
import type { ResourceUsage, StrategyType } from "./core.js";

/**
 * Monitoring event categories with standardized prefixes
 */
export enum MonitoringEventPrefix {
    PERFORMANCE = "perf",
    HEALTH = "health",
    BUSINESS = "biz",
    SAFETY = "safety",
}

/**
 * Base monitoring event extending the core event structure
 */
export interface MonitoringEvent extends BaseEvent {
    category: MonitoringEventPrefix;
    metrics?: MonitoringMetrics;
    sampling?: SamplingConfig;
}

/**
 * Standard metrics included with monitoring events
 */
export interface MonitoringMetrics {
    timestamp: Date;
    duration?: number;
    count?: number;
    rate?: number;
    percentile?: PercentileMetrics;
    resource?: ResourceMetrics;
}

/**
 * Percentile metrics for performance analysis
 */
export interface PercentileMetrics {
    p50: number;
    p90: number;
    p95: number;
    p99: number;
    p999: number;
}

/**
 * Resource usage metrics
 */
export interface ResourceMetrics {
    cpu?: number;
    memory?: number;
    tokens?: number;
    credits?: number;
    requests?: number;
}

/**
 * Sampling configuration for telemetry events
 */
export interface SamplingConfig {
    rate: number; // 0.0 to 1.0
    priority: "always" | "high" | "normal" | "low";
    conditions?: SamplingCondition[];
}

/**
 * Conditional sampling rules
 */
export interface SamplingCondition {
    field: string;
    operator: "equals" | "greater" | "less" | "contains";
    value: unknown;
}

/**
 * Performance event types
 */
export interface PerformanceEvent extends MonitoringEvent {
    category: MonitoringEventPrefix.PERFORMANCE;
    payload: PerformancePayload;
}

export type PerformancePayload =
    | ExecutionTimingPayload
    | ResourceUtilizationPayload
    | ThroughputPayload
    | LatencyPayload
    | CachePerformancePayload;

export interface ExecutionTimingPayload {
    type: "execution_timing";
    component: string;
    operation: string;
    startTime: Date;
    endTime: Date;
    duration: number;
    success: boolean;
    metadata?: Record<string, unknown>;
}

export interface ResourceUtilizationPayload {
    type: "resource_utilization";
    component: string;
    usage: ResourceUsage;
    limits?: ResourceUsage;
    utilizationPercent: number;
}

export interface ThroughputPayload {
    type: "throughput";
    component: string;
    metric: string;
    value: number;
    unit: string;
    window: number; // seconds
}

export interface LatencyPayload {
    type: "latency";
    component: string;
    operation: string;
    value: number;
    percentiles: PercentileMetrics;
}

export interface CachePerformancePayload {
    type: "cache_performance";
    cache: string;
    hits: number;
    misses: number;
    hitRate: number;
    evictions: number;
}

/**
 * Health event types
 */
export interface HealthEvent extends MonitoringEvent {
    category: MonitoringEventPrefix.HEALTH;
    payload: HealthPayload;
}

export type HealthPayload =
    | ComponentHealthPayload
    | SystemHealthPayload
    | DependencyHealthPayload
    | CircuitBreakerPayload;

export interface ComponentHealthPayload {
    type: "component_health";
    component: string;
    status: "healthy" | "degraded" | "unhealthy";
    checks: HealthCheck[];
    lastHealthyTime?: Date;
}

export interface HealthCheck {
    name: string;
    status: "pass" | "warn" | "fail";
    message?: string;
    duration?: number;
    metadata?: Record<string, unknown>;
}

export interface SystemHealthPayload {
    type: "system_health";
    tier: 1 | 2 | 3;
    status: "healthy" | "degraded" | "unhealthy";
    components: ComponentStatus[];
    alerts: HealthAlert[];
}

export interface ComponentStatus {
    name: string;
    status: "up" | "down" | "degraded";
    lastSeen: Date;
    errorRate?: number;
}

export interface HealthAlert {
    severity: "info" | "warning" | "error" | "critical";
    message: string;
    component?: string;
    timestamp: Date;
}

export interface DependencyHealthPayload {
    type: "dependency_health";
    dependency: string;
    status: "available" | "degraded" | "unavailable";
    responseTime?: number;
    errorRate?: number;
    lastChecked: Date;
}

export interface CircuitBreakerPayload {
    type: "circuit_breaker";
    service: string;
    state: "closed" | "open" | "half_open";
    failures: number;
    threshold: number;
    lastFailure?: Date;
    nextRetry?: Date;
}

/**
 * Business event types
 */
export interface BusinessEvent extends MonitoringEvent {
    category: MonitoringEventPrefix.BUSINESS;
    payload: BusinessPayload;
}

export type BusinessPayload =
    | TaskCompletionPayload
    | GoalProgressPayload
    | StrategyEffectivenessPayload
    | UserInteractionPayload
    | CostTrackingPayload;

export interface TaskCompletionPayload {
    type: "task_completion";
    taskId: string;
    taskType: string;
    result: "success" | "failure" | "partial";
    duration: number;
    retries?: number;
    resourceCost?: number;
}

export interface GoalProgressPayload {
    type: "goal_progress";
    goalId: string;
    progress: number; // 0-100
    milestones: MilestoneStatus[];
    estimatedCompletion?: Date;
}

export interface MilestoneStatus {
    id: string;
    name: string;
    completed: boolean;
    completedAt?: Date;
}

export interface StrategyEffectivenessPayload {
    type: "strategy_effectiveness";
    strategy: StrategyType;
    taskType: string;
    successRate: number;
    avgDuration: number;
    avgCost: number;
    sampleSize: number;
}

export interface UserInteractionPayload {
    type: "user_interaction";
    action: string;
    context: string;
    userId?: string;
    sessionId?: string;
    metadata?: Record<string, unknown>;
}

export interface CostTrackingPayload {
    type: "cost_tracking";
    resource: string;
    amount: number;
    unit: string;
    consumer: string;
    operation?: string;
}

/**
 * Safety event types
 */
export interface SafetyEvent extends MonitoringEvent {
    category: MonitoringEventPrefix.SAFETY;
    payload: SafetyPayload;
}

export type SafetyPayload =
    | ValidationErrorPayload
    | SecurityIncidentPayload
    | PIIDetectionPayload
    | ErrorOccurredPayload
    | AnomalyDetectedPayload;

export interface ValidationErrorPayload {
    type: "validation_error";
    component: string;
    errors: ValidationError[];
    context?: Record<string, unknown>;
}

export interface ValidationError {
    field?: string;
    rule: string;
    message: string;
    severity: "warning" | "error" | "critical";
}

export interface SecurityIncidentPayload {
    type: "security_incident";
    incidentType: string;
    severity: "low" | "medium" | "high" | "critical";
    source: string;
    target?: string;
    details: Record<string, unknown>;
    mitigated: boolean;
}

export interface PIIDetectionPayload {
    type: "pii_detection";
    types: string[];
    locations: string[];
    action: "masked" | "removed" | "flagged";
    context?: string;
}

export interface ErrorOccurredPayload {
    type: "error_occurred";
    errorType: string;
    message: string;
    stack?: string[];
    component: string;
    severity: "low" | "medium" | "high" | "critical";
    context?: Record<string, unknown>;
}

export interface AnomalyDetectedPayload {
    type: "anomaly_detected";
    anomalyType: string;
    component: string;
    metric: string;
    expected: number | string;
    actual: number | string;
    deviation: number;
    confidence: number;
}

/**
 * Tier-specific metric types
 */
export interface TierMetrics {
    tier: 1 | 2 | 3;
    component: string;
    metrics: TierSpecificMetrics;
}

export type TierSpecificMetrics =
    | Tier1Metrics
    | Tier2Metrics
    | Tier3Metrics;

export interface Tier1Metrics {
    tier: 1;
    swarms: {
        active: number;
        total: number;
        avgAgentsPerSwarm: number;
    };
    coordination: {
        decisionsPerMinute: number;
        consensusTime: number;
        conflictRate: number;
    };
    resources: {
        totalAllocated: number;
        utilizationRate: number;
        contentionEvents: number;
    };
}

export interface Tier2Metrics {
    tier: 2;
    runs: {
        active: number;
        completed: number;
        failed: number;
        avgDuration: number;
    };
    navigation: {
        pathOptimizations: number;
        branchingFactor: number;
        backtrackRate: number;
    };
    performance: {
        stepThroughput: number;
        avgStepDuration: number;
        cacheHitRate: number;
    };
}

export interface Tier3Metrics {
    tier: 3;
    execution: {
        stepsPerMinute: number;
        strategyDistribution: Record<StrategyType, number>;
        avgExecutionTime: number;
    };
    tools: {
        callsPerMinute: number;
        successRate: number;
        avgLatency: number;
        topTools: Array<{ name: string; calls: number }>;
    };
    resources: {
        tokensUsed: number;
        creditsConsumed: number;
        costPerStep: number;
    };
}

/**
 * Telemetry configuration
 */
export interface TelemetryConfig {
    enabled: boolean;
    samplingRates: Record<string, number>;
    batchSize: number;
    flushInterval: number;
    maxRetries: number;
    bufferSize: number;
    performanceGuards: PerformanceGuards;
}

export interface PerformanceGuards {
    maxOverheadMs: number;
    maxBufferSize: number;
    maxBatchSize: number;
    dropOnOverload: boolean;
}

/**
 * Rolling history configuration
 */
export interface RollingHistoryConfig {
    maxEntries: number;
    maxAgeMs: number;
    compressionEnabled: boolean;
    indexFields: string[];
}

/**
 * History entry for execution tracking
 */
export interface HistoryEntry {
    id: string;
    timestamp: Date;
    type: string;
    tier: 1 | 2 | 3;
    component: string;
    operation: string;
    duration?: number;
    success: boolean;
    metadata: Record<string, unknown>;
}

/**
 * Query interface for history
 */
export interface HistoryQuery {
    timeRange?: TimeRange;
    types?: string[];
    tiers?: Array<1 | 2 | 3>;
    components?: string[];
    success?: boolean;
    limit?: number;
    orderBy?: "timestamp" | "duration";
    orderDirection?: "asc" | "desc";
}

export interface TimeRange {
    start: Date;
    end: Date;
}

/**
 * Aggregation results for history analysis
 */
export interface HistoryAggregation {
    timeRange: TimeRange;
    totalEntries: number;
    successRate: number;
    avgDuration: number;
    byType: Record<string, AggregationStats>;
    byTier: Record<string, AggregationStats>;
    byComponent: Record<string, AggregationStats>;
}

export interface AggregationStats {
    count: number;
    successCount: number;
    failureCount: number;
    avgDuration: number;
    minDuration: number;
    maxDuration: number;
}
