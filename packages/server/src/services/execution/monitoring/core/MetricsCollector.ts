/**
 * MetricsCollector replaces TelemetryShim with enhanced functionality
 * while maintaining the same performance guarantees (<5ms overhead).
 * 
 * Features:
 * - Async fire-and-forget pattern
 * - Configurable sampling rates
 * - Automatic batching
 * - Performance guards
 */

import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { generatePK } from "@vrooli/shared";
import { CircularBuffer } from "../storage/CircularBuffer.js";
import {
    type UnifiedMetric,
    type MonitoringConfig,
    type MetricType,
    type MonitoringEvent,
} from "../types.js";

/**
 * Performance guard to ensure overhead stays under limit
 */
class PerformanceGuard {
    private recentOverheads: number[] = [];
    private readonly maxSamples = 100;
    
    constructor(private readonly maxOverheadMs: number) {}
    
    recordOverhead(overheadMs: number): void {
        this.recentOverheads.push(overheadMs);
        if (this.recentOverheads.length > this.maxSamples) {
            this.recentOverheads.shift();
        }
    }
    
    shouldThrottle(): boolean {
        if (this.recentOverheads.length < 10) return false;
        
        const avgOverhead = this.recentOverheads.reduce((a, b) => a + b, 0) / this.recentOverheads.length;
        return avgOverhead > this.maxOverheadMs;
    }
    
    getAverageOverhead(): number {
        if (this.recentOverheads.length === 0) return 0;
        return this.recentOverheads.reduce((a, b) => a + b, 0) / this.recentOverheads.length;
    }
}

/**
 * Batch collector for efficient metric processing
 */
class BatchCollector {
    private batch: UnifiedMetric[] = [];
    private timer?: NodeJS.Timeout;
    
    constructor(
        private readonly batchSize: number,
        private readonly flushIntervalMs: number,
        private readonly onFlush: (metrics: UnifiedMetric[]) => Promise<void>
    ) {}
    
    add(metric: UnifiedMetric): void {
        this.batch.push(metric);
        
        if (this.batch.length >= this.batchSize) {
            this.flush();
        } else if (!this.timer) {
            this.timer = setTimeout(() => this.flush(), this.flushIntervalMs);
        }
    }
    
    async flush(): Promise<void> {
        if (this.timer) {
            clearTimeout(this.timer);
            this.timer = undefined;
        }
        
        if (this.batch.length === 0) return;
        
        const metricsToFlush = [...this.batch];
        this.batch = [];
        
        try {
            await this.onFlush(metricsToFlush);
        } catch (error) {
            console.error("Error flushing metrics batch", { error, count: metricsToFlush.length });
        }
    }
}

/**
 * MetricsCollector - High-performance metric collection with minimal overhead
 */
export class MetricsCollector {
    private readonly eventBus?: EventBus;
    private readonly performanceGuard: PerformanceGuard;
    private readonly batchCollector: BatchCollector;
    private readonly buffer: CircularBuffer<UnifiedMetric>;
    private readonly samplingRates: Record<MetricType, number>;
    private isShuttingDown = false;
    
    constructor(
        private readonly config: MonitoringConfig,
        eventBus?: EventBus
    ) {
        this.eventBus = eventBus;
        this.performanceGuard = new PerformanceGuard(config.maxOverheadMs);
        this.samplingRates = config.samplingRates;
        
        // Initialize circular buffer for temporary storage
        this.buffer = new CircularBuffer<UnifiedMetric>(1000);
        
        // Initialize batch collector
        this.batchCollector = new BatchCollector(
            100,  // Batch size
            50,   // Flush interval (ms)
            async (metrics) => this.emitMetrics(metrics)
        );
    }
    
    /**
     * Record a single metric with fire-and-forget pattern
     */
    async record(metric: Omit<UnifiedMetric, "id" | "timestamp">): Promise<void> {
        // Quick return if shutting down
        if (this.isShuttingDown) return;
        
        const startTime = process.hrtime.bigint();
        
        try {
            // Apply sampling
            if (!this.shouldSample(metric.type)) {
                return;
            }
            
            // Check performance guard
            if (this.performanceGuard.shouldThrottle()) {
                console.warn("MetricsCollector throttling due to high overhead", {
                    avgOverhead: this.performanceGuard.getAverageOverhead(),
                });
                return;
            }
            
            // Create full metric
            const fullMetric: UnifiedMetric = {
                id: generatePK().toString(),
                timestamp: new Date(),
                ...metric,
            };
            
            // Add to buffer for immediate local access
            this.buffer.add(fullMetric);
            
            // Add to batch for async emission
            this.batchCollector.add(fullMetric);
            
        } finally {
            // Record overhead
            const endTime = process.hrtime.bigint();
            const overheadMs = Number(endTime - startTime) / 1_000_000;
            this.performanceGuard.recordOverhead(overheadMs);
        }
    }
    
    /**
     * Record multiple metrics in batch
     */
    async recordBatch(metrics: Array<Omit<UnifiedMetric, "id" | "timestamp">>): Promise<void> {
        if (this.isShuttingDown || metrics.length === 0) return;
        
        const startTime = process.hrtime.bigint();
        
        try {
            const timestamp = new Date();
            const fullMetrics: UnifiedMetric[] = [];
            
            for (const metric of metrics) {
                if (this.shouldSample(metric.type)) {
                    fullMetrics.push({
                        id: generatePK().toString(),
                        timestamp,
                        ...metric,
                    });
                }
            }
            
            if (fullMetrics.length > 0) {
                // Add to buffer
                fullMetrics.forEach(m => this.buffer.add(m));
                
                // Emit directly for large batches
                if (fullMetrics.length > 50) {
                    await this.emitMetrics(fullMetrics);
                } else {
                    // Add to batch collector for small batches
                    fullMetrics.forEach(m => this.batchCollector.add(m));
                }
            }
        } finally {
            const endTime = process.hrtime.bigint();
            const overheadMs = Number(endTime - startTime) / 1_000_000;
            this.performanceGuard.recordOverhead(overheadMs);
        }
    }
    
    /**
     * Get recent metrics from local buffer
     */
    getRecentMetrics(count: number = 100): UnifiedMetric[] {
        return this.buffer.getRecent(count);
    }
    
    /**
     * Emit metrics to event bus
     */
    private async emitMetrics(metrics: UnifiedMetric[]): Promise<void> {
        if (metrics.length === 0 || !this.config.eventBusEnabled) return;
        
        try {
            // Group by metric type for efficient emission
            const grouped = new Map<MetricType, UnifiedMetric[]>();
            
            for (const metric of metrics) {
                const group = grouped.get(metric.type) || [];
                group.push(metric);
                grouped.set(metric.type, group);
            }
            
            // Emit each group
            const promises: Promise<void>[] = [];
            
            for (const [type, groupMetrics] of grouped) {
                const event: MonitoringEvent = {
                    id: generatePK().toString(),
                    type: "monitoring.metric",
                    timestamp: new Date(),
                    source: {
                        tier: "cross-cutting",
                        component: "MetricsCollector",
                        instanceId: generatePK().toString(),
                    },
                    data: {
                        metric: groupMetrics.length === 1 ? groupMetrics[0] : undefined,
                    },
                    metadata: groupMetrics.length > 1 ? { 
                        metrics: groupMetrics,
                        version: "1.0.0",
                        tags: ["monitoring", "metric"],
                    } : {
                        version: "1.0.0",
                        tags: ["monitoring", "metric"],
                    },
                };
                
                if (this.eventBus) {
                    // Wrap in error handling to prevent unhandled rejections
                    const publishPromise = this.eventBus.publish(event).catch(error => {
                        console.error("Failed to publish monitoring event", { error, eventType: event.type });
                        throw error; // Re-throw to be caught by allSettled
                    });
                    promises.push(publishPromise);
                }
            }
            
            // Wait for all emissions with proper error handling
            if (promises.length > 0) {
                const results = await Promise.allSettled(promises);
                const failed = results.filter(r => r.status === 'rejected');
                if (failed.length > 0) {
                    const errors = failed.map(r => (r as PromiseRejectedResult).reason);
                    console.warn("Some event publications failed", { 
                        failedCount: failed.length, 
                        totalCount: promises.length,
                        errors: errors.slice(0, 3) // Log first 3 errors to avoid spam
                    });
                }
            }
            
        } catch (error) {
            // Enhanced error logging with more context
            console.error("Critical error in metric emission", { 
                error: error instanceof Error ? error.message : String(error),
                stack: error instanceof Error ? error.stack : undefined,
                count: metrics.length,
                timestamp: new Date().toISOString()
            });
            // Don't throw - maintain fire-and-forget pattern
        }
    }
    
    /**
     * Check if metric should be sampled based on configuration
     */
    private shouldSample(metricType: MetricType): boolean {
        const rate = this.samplingRates[metricType] ?? 1.0;
        return Math.random() < rate;
    }
    
    /**
     * Flush any pending metrics
     */
    async flush(): Promise<void> {
        await this.batchCollector.flush();
    }
    
    /**
     * Get collector health status
     */
    async getHealth(): Promise<{
        status: "healthy" | "degraded" | "unhealthy";
        avgOverheadMs: number;
        bufferSize: number;
        metricsCollected: number;
    }> {
        const avgOverhead = this.performanceGuard.getAverageOverhead();
        const isThrottling = this.performanceGuard.shouldThrottle();
        
        return {
            status: isThrottling ? "degraded" : avgOverhead > this.config.maxOverheadMs * 0.8 ? "degraded" : "healthy",
            avgOverheadMs: avgOverhead,
            bufferSize: this.buffer.size(),
            metricsCollected: this.buffer.totalAdded(),
        };
    }
    
    /**
     * Shutdown collector gracefully
     */
    async shutdown(): Promise<void> {
        this.isShuttingDown = true;
        await this.flush();
    }
    
    /**
     * Convenience methods for specific metric types (maintaining TelemetryShim compatibility)
     */
    
    async emitPerformance(
        component: string,
        name: string,
        value: number,
        metadata?: Record<string, unknown>
    ): Promise<void> {
        await this.record({
            tier: "cross-cutting",
            component,
            type: "performance",
            name,
            value,
            unit: "ms",
            metadata,
        });
    }
    
    async emitResource(
        component: string,
        name: string,
        value: number,
        metadata?: Record<string, unknown>
    ): Promise<void> {
        await this.record({
            tier: "cross-cutting",
            component,
            type: "resource",
            name,
            value,
            metadata,
        });
    }
    
    async emitHealth(
        component: string,
        name: string,
        value: boolean | number,
        metadata?: Record<string, unknown>
    ): Promise<void> {
        await this.record({
            tier: "cross-cutting",
            component,
            type: "health",
            name,
            value,
            metadata,
        });
    }
    
    async emitBusiness(
        component: string,
        name: string,
        value: number | string,
        metadata?: Record<string, unknown>
    ): Promise<void> {
        await this.record({
            tier: "cross-cutting",
            component,
            type: "business",
            name,
            value,
            metadata,
        });
    }
    
    async emitSafety(
        component: string,
        name: string,
        value: boolean | number,
        metadata?: Record<string, unknown>
    ): Promise<void> {
        await this.record({
            tier: "cross-cutting",
            component,
            type: "safety",
            name,
            value,
            metadata,
        });
    }
}