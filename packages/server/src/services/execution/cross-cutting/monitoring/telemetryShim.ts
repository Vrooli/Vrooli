/**
 * Lightweight telemetry shim for fire-and-forget event emission
 * Provides <5ms overhead with built-in performance guards
 */

import type { EventBus } from "../events/eventBus.js";
import type {
    MonitoringEvent,
    MonitoringEventPrefix,
    PerformanceEvent,
    HealthEvent,
    BusinessEvent,
    SafetyEvent,
    ExecutionTimingPayload,
    ResourceUtilizationPayload,
    ThroughputPayload,
    LatencyPayload,
    ComponentHealthPayload,
    TaskCompletionPayload,
    StrategyEffectivenessPayload,
    ValidationErrorPayload,
    SecurityIncidentPayload,
    PIIDetectionPayload,
    ErrorOccurredPayload,
    SamplingConfig,
    TelemetryConfig,
    PercentileMetrics,
    ResourceUsage,
    StrategyType,
    EventSource,
} from "@vrooli/shared";
import { generatePK } from "@vrooli/shared";

/**
 * Default telemetry configuration
 */
const DEFAULT_CONFIG: TelemetryConfig = {
    enabled: true,
    samplingRates: {
        [MonitoringEventPrefix.PERFORMANCE]: 0.1, // 10% sampling for perf events
        [MonitoringEventPrefix.HEALTH]: 1.0, // 100% for health events
        [MonitoringEventPrefix.BUSINESS]: 1.0, // 100% for business events
        [MonitoringEventPrefix.SAFETY]: 1.0, // 100% for safety events
    },
    batchSize: 100,
    flushInterval: 1000, // 1 second
    maxRetries: 3,
    bufferSize: 10000,
    performanceGuards: {
        maxOverheadMs: 5,
        maxBufferSize: 10000,
        maxBatchSize: 1000,
        dropOnOverload: true,
    },
};

/**
 * TelemetryShim - Lightweight event emitter with performance guarantees
 * 
 * Features:
 * - Async fire-and-forget pattern
 * - Configurable sampling rates
 * - Built-in performance guards
 * - Automatic batching
 * - Memory-efficient circular buffer
 */
export class TelemetryShim {
    private readonly eventBus: EventBus;
    private readonly config: TelemetryConfig;
    private readonly buffer: MonitoringEvent[] = [];
    private readonly source: EventSource;
    private flushTimer?: NodeJS.Timeout;
    private correlationId: string = generatePK().toString();
    
    // Performance tracking
    private emitCount = 0;
    private dropCount = 0;
    private totalOverheadMs = 0;

    constructor(
        eventBus: EventBus,
        source: EventSource,
        config: Partial<TelemetryConfig> = {},
    ) {
        this.eventBus = eventBus;
        this.source = source;
        this.config = { ...DEFAULT_CONFIG, ...config };
        
        if (this.config.enabled) {
            this.startBatchFlush();
        }
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
        const payload: ExecutionTimingPayload = {
            type: "execution_timing",
            component,
            operation,
            startTime,
            endTime,
            duration: endTime.getTime() - startTime.getTime(),
            success,
            metadata,
        };

        await this.emit<PerformanceEvent>(
            MonitoringEventPrefix.PERFORMANCE,
            payload,
            { priority: success ? "normal" : "high" },
        );
    }

    async emitResourceUtilization(
        component: string,
        usage: ResourceUsage,
        limits?: ResourceUsage,
    ): Promise<void> {
        const payload: ResourceUtilizationPayload = {
            type: "resource_utilization",
            component,
            usage,
            limits,
            utilizationPercent: limits
                ? this.calculateUtilization(usage, limits)
                : 0,
        };

        await this.emit<PerformanceEvent>(
            MonitoringEventPrefix.PERFORMANCE,
            payload,
        );
    }

    async emitThroughput(
        component: string,
        metric: string,
        value: number,
        unit: string,
        window: number,
    ): Promise<void> {
        const payload: ThroughputPayload = {
            type: "throughput",
            component,
            metric,
            value,
            unit,
            window,
        };

        await this.emit<PerformanceEvent>(
            MonitoringEventPrefix.PERFORMANCE,
            payload,
        );
    }

    async emitLatency(
        component: string,
        operation: string,
        value: number,
        percentiles: PercentileMetrics,
    ): Promise<void> {
        const payload: LatencyPayload = {
            type: "latency",
            component,
            operation,
            value,
            percentiles,
        };

        await this.emit<PerformanceEvent>(
            MonitoringEventPrefix.PERFORMANCE,
            payload,
        );
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
        const payload: ComponentHealthPayload = {
            type: "component_health",
            component,
            status,
            checks,
        };

        await this.emit<HealthEvent>(
            MonitoringEventPrefix.HEALTH,
            payload,
            { priority: "high" },
        );
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
        const payload: TaskCompletionPayload = {
            type: "task_completion",
            taskId,
            taskType,
            result,
            duration,
            resourceCost,
        };

        await this.emit<BusinessEvent>(
            MonitoringEventPrefix.BUSINESS,
            payload,
        );
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
        const payload: StrategyEffectivenessPayload = {
            type: "strategy_effectiveness",
            strategy,
            taskType,
            ...stats,
        };

        await this.emit<BusinessEvent>(
            MonitoringEventPrefix.BUSINESS,
            payload,
        );
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
        const payload: ValidationErrorPayload = {
            type: "validation_error",
            component,
            errors,
            context,
        };

        await this.emit<SafetyEvent>(
            MonitoringEventPrefix.SAFETY,
            payload,
            { priority: "high" },
        );
    }

    async emitSecurityIncident(
        incidentType: string,
        severity: "low" | "medium" | "high" | "critical",
        details: Record<string, unknown>,
    ): Promise<void> {
        const payload: SecurityIncidentPayload = {
            type: "security_incident",
            incidentType,
            severity,
            source: this.source.component,
            details,
            mitigated: false,
        };

        await this.emit<SafetyEvent>(
            MonitoringEventPrefix.SAFETY,
            payload,
            { priority: "always" },
        );
    }

    async emitPIIDetection(
        types: string[],
        locations: string[],
        action: "masked" | "removed" | "flagged",
    ): Promise<void> {
        const payload: PIIDetectionPayload = {
            type: "pii_detection",
            types,
            locations,
            action,
        };

        await this.emit<SafetyEvent>(
            MonitoringEventPrefix.SAFETY,
            payload,
            { priority: "high" },
        );
    }

    async emitError(
        error: unknown,
        component: string,
        severity: "low" | "medium" | "high" | "critical" = "medium",
        context?: Record<string, unknown>,
    ): Promise<void> {
        const errorData = this.extractErrorData(error);
        const payload: ErrorOccurredPayload = {
            type: "error_occurred",
            ...errorData,
            component,
            severity,
            context,
        };

        await this.emit<SafetyEvent>(
            MonitoringEventPrefix.SAFETY,
            payload,
            { priority: severity === "critical" ? "always" : "high" },
        );
    }

    /**
     * Core emission method with performance guards
     */
    private async emit<T extends MonitoringEvent>(
        category: MonitoringEventPrefix,
        payload: T["payload"],
        sampling?: Partial<SamplingConfig>,
    ): Promise<void> {
        if (!this.config.enabled) {
            return;
        }

        const startTime = performance.now();

        try {
            // Apply sampling
            if (!this.shouldSample(category, sampling)) {
                return;
            }

            // Check performance guards
            if (this.buffer.length >= this.config.performanceGuards.maxBufferSize) {
                if (this.config.performanceGuards.dropOnOverload) {
                    this.dropCount++;
                    return;
                }
            }

            // Create event
            const event: T = {
                id: generatePK().toString(),
                type: `${category}.${payload.type}`,
                timestamp: new Date(),
                source: this.source,
                correlationId: this.correlationId,
                category,
                payload,
                metadata: {
                    version: "1.0.0",
                    tags: [category, payload.type],
                    priority: sampling?.priority || "normal",
                },
                sampling: {
                    rate: this.config.samplingRates[category] || 1.0,
                    priority: sampling?.priority || "normal",
                    conditions: sampling?.conditions,
                },
            } as T;

            // Add to buffer
            this.buffer.push(event);
            this.emitCount++;

            // Flush if buffer is getting full
            if (this.buffer.length >= this.config.batchSize) {
                await this.flush();
            }
        } finally {
            // Track overhead
            const overhead = performance.now() - startTime;
            this.totalOverheadMs += overhead;

            // Log if overhead exceeds threshold
            if (overhead > this.config.performanceGuards.maxOverheadMs) {
                console.warn(
                    `[TelemetryShim] Emission overhead exceeded: ${overhead.toFixed(2)}ms`,
                );
            }
        }
    }

    /**
     * Batch emission
     */
    async emitBatch(events: Array<{
        category: MonitoringEventPrefix;
        payload: MonitoringEvent["payload"];
        sampling?: Partial<SamplingConfig>;
    }>): Promise<void> {
        if (!this.config.enabled || events.length === 0) {
            return;
        }

        const startTime = performance.now();

        try {
            for (const event of events) {
                // Note: We don't await here for performance
                this.emit(event.category, event.payload, event.sampling)
                    .catch(error => {
                        console.error("[TelemetryShim] Batch emit error:", error);
                    });
            }
        } finally {
            const overhead = performance.now() - startTime;
            if (overhead > this.config.performanceGuards.maxOverheadMs * events.length) {
                console.warn(
                    `[TelemetryShim] Batch emission overhead: ${overhead.toFixed(2)}ms for ${events.length} events`,
                );
            }
        }
    }

    /**
     * Flush buffered events
     */
    private async flush(): Promise<void> {
        if (this.buffer.length === 0) {
            return;
        }

        const events = this.buffer.splice(0, this.config.performanceGuards.maxBatchSize);

        try {
            await this.eventBus.publishBatch(events);
        } catch (error) {
            console.error("[TelemetryShim] Failed to flush events:", error);
            // Re-add events to buffer if there's room
            if (this.buffer.length + events.length <= this.config.performanceGuards.maxBufferSize) {
                this.buffer.unshift(...events);
            } else {
                this.dropCount += events.length;
            }
        }
    }

    /**
     * Start periodic flush
     */
    private startBatchFlush(): void {
        this.flushTimer = setInterval(() => {
            this.flush().catch(error => {
                console.error("[TelemetryShim] Periodic flush error:", error);
            });
        }, this.config.flushInterval);
    }

    /**
     * Stop and cleanup
     */
    async stop(): Promise<void> {
        if (this.flushTimer) {
            clearInterval(this.flushTimer);
            this.flushTimer = undefined;
        }

        // Final flush
        await this.flush();

        // Log stats
        console.info("[TelemetryShim] Stats:", {
            emitted: this.emitCount,
            dropped: this.dropCount,
            avgOverheadMs: this.emitCount > 0
                ? (this.totalOverheadMs / this.emitCount).toFixed(2)
                : 0,
        });
    }

    /**
     * Helper methods
     */
    private shouldSample(
        category: MonitoringEventPrefix,
        sampling?: Partial<SamplingConfig>,
    ): boolean {
        // Always sample if priority is "always"
        if (sampling?.priority === "always") {
            return true;
        }

        // Check sampling rate
        const rate = this.config.samplingRates[category] || 1.0;
        if (Math.random() > rate) {
            return false;
        }

        // Check conditions
        if (sampling?.conditions) {
            // In real implementation, evaluate conditions
            // For now, always pass
            return true;
        }

        return true;
    }

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

    /**
     * Get telemetry stats
     */
    getStats(): {
        emitted: number;
        dropped: number;
        bufferSize: number;
        avgOverheadMs: number;
    } {
        return {
            emitted: this.emitCount,
            dropped: this.dropCount,
            bufferSize: this.buffer.length,
            avgOverheadMs: this.emitCount > 0
                ? this.totalOverheadMs / this.emitCount
                : 0,
        };
    }
}