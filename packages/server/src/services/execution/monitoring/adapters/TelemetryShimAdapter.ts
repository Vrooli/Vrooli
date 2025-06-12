/**
 * TelemetryShimAdapter provides backward compatibility for existing TelemetryShim usage
 * while routing all metrics through the UnifiedMonitoringService.
 */

import { UnifiedMonitoringService } from "../UnifiedMonitoringService.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import {
    type MonitoringEvent,
    type MonitoringEventPrefix,
    type PerformanceEvent,
    type HealthEvent,
    type BusinessEvent,
    type SafetyEvent,
    type ExecutionTimingPayload,
    type ResourceUtilizationPayload,
    type ThroughputPayload,
    type LatencyPayload,
    type ComponentHealthPayload,
    type TaskCompletionPayload,
    type StrategyEffectivenessPayload,
    type ValidationErrorPayload,
    type SecurityIncidentPayload,
    type PIIDetectionPayload,
    type ErrorOccurredPayload,
    type SamplingConfig,
    type TelemetryConfig,
    type PercentileMetrics,
    type ResourceUsage,
    type StrategyType,
    type EventSource,
} from "@vrooli/shared";

/**
 * Adapter that provides TelemetryShim interface while using UnifiedMonitoringService
 */
export class TelemetryShimAdapter {
    private readonly monitoringService: UnifiedMonitoringService;
    private readonly source: EventSource;
    private correlationId: string = "unknown";
    
    // Performance tracking (for compatibility)
    private emitCount = 0;
    private dropCount = 0;
    private totalOverheadMs = 0;
    
    constructor(
        eventBus: EventBus,
        source: EventSource,
        config: Partial<TelemetryConfig> = {},
    ) {
        this.source = source;
        this.monitoringService = UnifiedMonitoringService.getInstance();
        
        // Initialize monitoring service if needed
        this.monitoringService.initialize().catch(error => {
            console.error("Failed to initialize monitoring service", error);
        });
    }
    
    /**
     * Set correlation ID for request tracing
     */
    setCorrelationId(correlationId: string): void {
        this.correlationId = correlationId;
    }
    
    /**
     * Performance Events
     */
    async emitExecutionTiming(
        component: string,
        operation: string,
        startTime: Date,
        endTime: Date,
        success: boolean,
        metadata?: Record<string, unknown>,
    ): Promise<void> {
        const duration = endTime.getTime() - startTime.getTime();
        
        await this.monitoringService.record({
            tier: "cross-cutting",
            component,
            type: "performance",
            name: `execution.${operation}.duration`,
            value: duration,
            unit: "ms",
            metadata: {
                operation,
                success,
                startTime: startTime.toISOString(),
                endTime: endTime.toISOString(),
                correlationId: this.correlationId,
                ...metadata,
            },
        });
        
        // Also emit success/failure metric
        await this.monitoringService.record({
            tier: "cross-cutting",
            component,
            type: "business",
            name: `execution.${operation}.success`,
            value: success ? 1 : 0,
            metadata: {
                operation,
                duration,
                correlationId: this.correlationId,
            },
        });
    }
    
    async emitResourceUtilization(
        component: string,
        usage: ResourceUsage,
        limits?: ResourceUsage,
    ): Promise<void> {
        const utilizationPercent = limits
            ? this.calculateUtilization(usage, limits)
            : 0;
        
        // Emit individual resource metrics
        if (usage.credits !== undefined) {
            await this.monitoringService.record({
                tier: "cross-cutting",
                component,
                type: "resource",
                name: "resource.credits.usage",
                value: usage.credits,
                unit: "credits",
                metadata: { correlationId: this.correlationId },
            });
        }
        
        if (usage.tokens !== undefined) {
            await this.monitoringService.record({
                tier: "cross-cutting",
                component,
                type: "resource",
                name: "resource.tokens.usage",
                value: usage.tokens,
                unit: "tokens",
                metadata: { correlationId: this.correlationId },
            });
        }
        
        if (usage.memory !== undefined) {
            await this.monitoringService.record({
                tier: "cross-cutting",
                component,
                type: "resource",
                name: "resource.memory.usage",
                value: usage.memory,
                unit: "bytes",
                metadata: { correlationId: this.correlationId },
            });
        }
        
        if (usage.time !== undefined) {
            await this.monitoringService.record({
                tier: "cross-cutting",
                component,
                type: "resource",
                name: "resource.time.usage",
                value: usage.time,
                unit: "ms",
                metadata: { correlationId: this.correlationId },
            });
        }
        
        // Emit utilization percentage if limits are provided
        if (limits && utilizationPercent > 0) {
            await this.monitoringService.record({
                tier: "cross-cutting",
                component,
                type: "efficiency",
                name: "resource.utilization.percent",
                value: utilizationPercent,
                unit: "percentage",
                metadata: { 
                    correlationId: this.correlationId,
                    usage,
                    limits,
                },
            });
        }
    }
    
    async emitThroughput(
        component: string,
        metric: string,
        value: number,
        unit: string,
        window: number,
    ): Promise<void> {
        await this.monitoringService.record({
            tier: "cross-cutting",
            component,
            type: "performance",
            name: `throughput.${metric}`,
            value,
            unit: `${unit}/window`,
            metadata: {
                correlationId: this.correlationId,
                window,
                originalUnit: unit,
            },
        });
    }
    
    async emitLatency(
        component: string,
        operation: string,
        value: number,
        percentiles: PercentileMetrics,
    ): Promise<void> {
        // Emit main latency value
        await this.monitoringService.record({
            tier: "cross-cutting",
            component,
            type: "performance",
            name: `latency.${operation}`,
            value,
            unit: "ms",
            metadata: {
                correlationId: this.correlationId,
                operation,
            },
        });
        
        // Emit percentile metrics
        for (const [percentile, latencyValue] of Object.entries(percentiles)) {
            await this.monitoringService.record({
                tier: "cross-cutting",
                component,
                type: "performance",
                name: `latency.${operation}.${percentile}`,
                value: latencyValue,
                unit: "ms",
                metadata: {
                    correlationId: this.correlationId,
                    operation,
                    percentile,
                },
            });
        }
    }
    
    /**
     * Health Events
     */
    async emitComponentHealth(
        component: string,
        status: "healthy" | "degraded" | "unhealthy",
        checks: Array<{
            name: string;
            status: "pass" | "warn" | "fail";
            message?: string;
            duration?: number;
        }>,
    ): Promise<void> {
        // Convert status to numeric value
        const statusValue = status === "healthy" ? 1 : status === "degraded" ? 0.5 : 0;
        
        await this.monitoringService.record({
            tier: "cross-cutting",
            component,
            type: "health",
            name: "component.health.status",
            value: statusValue,
            metadata: {
                correlationId: this.correlationId,
                status,
                checks,
            },
        });
        
        // Emit individual check results
        for (const check of checks) {
            const checkValue = check.status === "pass" ? 1 : check.status === "warn" ? 0.5 : 0;
            
            await this.monitoringService.record({
                tier: "cross-cutting",
                component,
                type: "health",
                name: `health.check.${check.name}`,
                value: checkValue,
                metadata: {
                    correlationId: this.correlationId,
                    checkStatus: check.status,
                    message: check.message,
                    duration: check.duration,
                },
            });
        }
    }
    
    /**
     * Business Events
     */
    async emitTaskCompletion(
        taskId: string,
        taskType: string,
        result: "success" | "failure" | "partial",
        duration: number,
        resourceCost?: number,
    ): Promise<void> {
        const resultValue = result === "success" ? 1 : result === "partial" ? 0.5 : 0;
        
        await this.monitoringService.record({
            tier: "cross-cutting",
            component: this.source.component,
            type: "business",
            name: "task.completion",
            value: resultValue,
            executionId: taskId,
            metadata: {
                correlationId: this.correlationId,
                taskId,
                taskType,
                result,
                duration,
                resourceCost,
            },
        });
        
        // Also emit duration metric
        await this.monitoringService.record({
            tier: "cross-cutting",
            component: this.source.component,
            type: "performance",
            name: `task.${taskType}.duration`,
            value: duration,
            unit: "ms",
            executionId: taskId,
            metadata: {
                correlationId: this.correlationId,
                result,
            },
        });
    }
    
    async emitStrategyEffectiveness(
        strategy: StrategyType,
        taskType: string,
        stats: {
            successRate: number;
            avgDuration: number;
            avgCost: number;
            sampleSize: number;
        },
    ): Promise<void> {
        await this.monitoringService.recordBatch([
            {
                tier: "cross-cutting",
                component: this.source.component,
                type: "intelligence",
                name: `strategy.${strategy}.success_rate`,
                value: stats.successRate,
                unit: "percentage",
                metadata: {
                    correlationId: this.correlationId,
                    strategy,
                    taskType,
                    sampleSize: stats.sampleSize,
                },
            },
            {
                tier: "cross-cutting",
                component: this.source.component,
                type: "performance",
                name: `strategy.${strategy}.avg_duration`,
                value: stats.avgDuration,
                unit: "ms",
                metadata: {
                    correlationId: this.correlationId,
                    strategy,
                    taskType,
                },
            },
            {
                tier: "cross-cutting",
                component: this.source.component,
                type: "efficiency",
                name: `strategy.${strategy}.avg_cost`,
                value: stats.avgCost,
                unit: "credits",
                metadata: {
                    correlationId: this.correlationId,
                    strategy,
                    taskType,
                },
            },
        ]);
    }
    
    /**
     * Safety Events
     */
    async emitValidationError(
        component: string,
        errors: Array<{
            field?: string;
            rule: string;
            message: string;
            severity: "warning" | "error" | "critical";
        }>,
        context?: Record<string, unknown>,
    ): Promise<void> {
        await this.monitoringService.record({
            tier: "cross-cutting",
            component,
            type: "safety",
            name: "validation.error",
            value: errors.length,
            metadata: {
                correlationId: this.correlationId,
                errors,
                context,
            },
        });
        
        // Emit metrics by severity
        const severityCounts = errors.reduce((acc, error) => {
            acc[error.severity] = (acc[error.severity] || 0) + 1;
            return acc;
        }, {} as Record<string, number>);
        
        for (const [severity, count] of Object.entries(severityCounts)) {
            await this.monitoringService.record({
                tier: "cross-cutting",
                component,
                type: "safety",
                name: `validation.error.${severity}`,
                value: count,
                metadata: {
                    correlationId: this.correlationId,
                },
            });
        }
    }
    
    async emitSecurityIncident(
        incidentType: string,
        severity: "low" | "medium" | "high" | "critical",
        details: Record<string, unknown>,
    ): Promise<void> {
        const severityValue = { low: 1, medium: 2, high: 3, critical: 4 }[severity];
        
        await this.monitoringService.record({
            tier: "cross-cutting",
            component: this.source.component,
            type: "safety",
            name: "security.incident",
            value: severityValue,
            metadata: {
                correlationId: this.correlationId,
                incidentType,
                severity,
                details,
                mitigated: false,
            },
        });
    }
    
    async emitPIIDetection(
        types: string[],
        locations: string[],
        action: "masked" | "removed" | "flagged",
    ): Promise<void> {
        await this.monitoringService.record({
            tier: "cross-cutting",
            component: this.source.component,
            type: "safety",
            name: "pii.detection",
            value: types.length,
            metadata: {
                correlationId: this.correlationId,
                types,
                locations,
                action,
            },
        });
    }
    
    async emitError(
        error: unknown,
        component: string,
        severity: "low" | "medium" | "high" | "critical" = "medium",
        context?: Record<string, unknown>,
    ): Promise<void> {
        const errorData = this.extractErrorData(error);
        const severityValue = { low: 1, medium: 2, high: 3, critical: 4 }[severity];
        
        await this.monitoringService.record({
            tier: "cross-cutting",
            component,
            type: "safety",
            name: "error.occurred",
            value: severityValue,
            metadata: {
                correlationId: this.correlationId,
                ...errorData,
                severity,
                context,
            },
        });
    }
    
    /**
     * Compatibility methods for existing TelemetryShim interface
     */
    async stop(): Promise<void> {
        // No-op for compatibility
        console.info("[TelemetryShimAdapter] Stop called - delegating to UnifiedMonitoringService");
    }
    
    getStats(): {
        emitted: number;
        dropped: number;
        bufferSize: number;
        avgOverheadMs: number;
    } {
        return {
            emitted: this.emitCount,
            dropped: this.dropCount,
            bufferSize: 0, // Not applicable in new architecture
            avgOverheadMs: this.emitCount > 0 ? this.totalOverheadMs / this.emitCount : 0,
        };
    }
    
    /**
     * Helper methods (from original TelemetryShim)
     */
    private calculateUtilization(
        usage: ResourceUsage,
        limits: ResourceUsage,
    ): number {
        const metrics = ["credits", "tokens", "time", "memory"] as const;
        let totalUsage = 0;
        let totalLimit = 0;
        
        for (const metric of metrics) {
            if (usage[metric] && limits[metric]) {
                totalUsage += usage[metric]!;
                totalLimit += limits[metric]!;
            }
        }
        
        return totalLimit > 0 ? (totalUsage / totalLimit) * 100 : 0;
    }
    
    private extractErrorData(error: unknown): {
        errorType: string;
        message: string;
        stack?: string[];
    } {
        if (error instanceof Error) {
            return {
                errorType: error.constructor.name,
                message: error.message,
                stack: error.stack?.split("\n").slice(0, 5), // Top 5 frames
            };
        }
        
        return {
            errorType: typeof error,
            message: String(error),
        };
    }
}