import { type Logger } from "winston";
import { type EventBus } from "../events/eventBus.js";
import {
    type ExecutionResourceUsage,
    type ResourceType,
    generatePk,
} from "@vrooli/shared";

/**
 * Usage tracking configuration
 */
export interface UsageTrackerConfig {
    trackerId: string;
    scope: "user" | "team" | "swarm" | "run" | "step";
    scopeId: string;
    windowSize?: number; // milliseconds for rolling window
    aggregationInterval?: number; // milliseconds
}

/**
 * Usage record
 */
export interface UsageRecord {
    id: string;
    timestamp: Date;
    resourceType: ResourceType;
    amount: number;
    cost?: number;
    metadata?: Record<string, unknown>;
}

/**
 * Aggregated usage metrics
 */
export interface AggregatedUsage {
    resourceType: ResourceType;
    total: number;
    average: number;
    peak: number;
    count: number;
    cost: number;
    firstSeen: Date;
    lastSeen: Date;
}

/**
 * Usage summary
 */
export interface UsageSummary {
    scope: string;
    scopeId: string;
    period: {
        start: Date;
        end: Date;
    };
    aggregated: Map<ResourceType, AggregatedUsage>;
    totalCost: number;
    efficiency?: number;
}

/**
 * Shared usage tracking component
 * 
 * Tracks resource usage with:
 * - Real-time recording
 * - Aggregation and rollups
 * - Trend analysis
 * - Cost tracking
 * - Event notifications
 */
export class UsageTracker {
    private readonly logger: Logger;
    private readonly eventBus?: EventBus;
    private readonly config: UsageTrackerConfig;
    private readonly records: UsageRecord[] = [];
    private readonly aggregated: Map<ResourceType, AggregatedUsage> = new Map();
    private aggregationTimer?: NodeJS.Timeout;
    private windowStartTime: Date;
    
    constructor(
        config: UsageTrackerConfig,
        logger: Logger,
        eventBus?: EventBus,
    ) {
        this.config = config;
        this.logger = logger;
        this.eventBus = eventBus;
        this.windowStartTime = new Date();
        
        // Start aggregation timer if configured
        if (config.aggregationInterval) {
            this.startAggregationTimer();
        }
        
        this.logger.debug("[UsageTracker] Initialized", {
            trackerId: config.trackerId,
            scope: config.scope,
            scopeId: config.scopeId,
        });
    }
    
    /**
     * Record resource usage
     */
    async record(
        resourceType: ResourceType,
        amount: number,
        cost?: number,
        metadata?: Record<string, unknown>,
    ): Promise<void> {
        const record: UsageRecord = {
            id: generatePk(),
            timestamp: new Date(),
            resourceType,
            amount,
            cost,
            metadata,
        };
        
        // Add to records
        this.records.push(record);
        
        // Update aggregated metrics
        this.updateAggregatedMetrics(record);
        
        // Cleanup old records if window configured
        if (this.config.windowSize) {
            this.cleanupOldRecords();
        }
        
        // Emit usage event
        if (this.eventBus) {
            await this.eventBus.emit({
                type: 'usage.recorded',
                timestamp: new Date(),
                data: {
                    trackerId: this.config.trackerId,
                    scope: this.config.scope,
                    scopeId: this.config.scopeId,
                    resourceType,
                    amount,
                    cost,
                },
            });
        }
        
        this.logger.debug("[UsageTracker] Recorded usage", {
            trackerId: this.config.trackerId,
            resourceType,
            amount,
            cost,
        });
    }
    
    /**
     * Record execution usage
     */
    async recordExecution(usage: ExecutionResourceUsage): Promise<void> {
        // Record credits
        if (usage.creditsUsed) {
            await this.record(
                ResourceType.CREDITS,
                Number(usage.creditsUsed),
                usage.cost,
            );
        }
        
        // Record time
        if (usage.durationMs) {
            await this.record(
                ResourceType.TIME,
                usage.durationMs,
            );
        }
        
        // Record memory
        if (usage.memoryUsedMB) {
            await this.record(
                ResourceType.MEMORY,
                usage.memoryUsedMB,
            );
        }
        
        // Record tokens
        if (usage.tokens) {
            await this.record(
                ResourceType.TOKENS,
                usage.tokens,
            );
        }
        
        // Record API calls
        if (usage.apiCalls) {
            await this.record(
                ResourceType.API_CALLS,
                usage.apiCalls,
            );
        }
    }
    
    /**
     * Get current usage summary
     */
    getSummary(): UsageSummary {
        const now = new Date();
        const period = {
            start: this.windowStartTime,
            end: now,
        };
        
        let totalCost = 0;
        for (const metrics of this.aggregated.values()) {
            totalCost += metrics.cost;
        }
        
        return {
            scope: this.config.scope,
            scopeId: this.config.scopeId,
            period,
            aggregated: new Map(this.aggregated),
            totalCost,
            efficiency: this.calculateEfficiency(),
        };
    }
    
    /**
     * Get usage trend
     */
    getTrend(
        resourceType: ResourceType,
        intervals: number = 10,
    ): Array<{ timestamp: Date; value: number }> {
        const records = this.records.filter(r => r.resourceType === resourceType);
        if (records.length === 0) {
            return [];
        }
        
        const oldest = records[0].timestamp.getTime();
        const newest = records[records.length - 1].timestamp.getTime();
        const intervalSize = (newest - oldest) / intervals;
        
        const trend: Array<{ timestamp: Date; value: number }> = [];
        
        for (let i = 0; i < intervals; i++) {
            const start = oldest + (i * intervalSize);
            const end = start + intervalSize;
            
            const intervalRecords = records.filter(r => {
                const time = r.timestamp.getTime();
                return time >= start && time < end;
            });
            
            const total = intervalRecords.reduce((sum, r) => sum + r.amount, 0);
            
            trend.push({
                timestamp: new Date(start + intervalSize / 2),
                value: total,
            });
        }
        
        return trend;
    }
    
    /**
     * Reset tracker
     */
    reset(): void {
        this.records.length = 0;
        this.aggregated.clear();
        this.windowStartTime = new Date();
        
        this.logger.debug("[UsageTracker] Reset", {
            trackerId: this.config.trackerId,
        });
    }
    
    /**
     * Shutdown tracker
     */
    shutdown(): void {
        if (this.aggregationTimer) {
            clearInterval(this.aggregationTimer);
            this.aggregationTimer = undefined;
        }
        
        // Emit final summary
        if (this.eventBus) {
            const summary = this.getSummary();
            this.eventBus.emit({
                type: 'usage.summary',
                timestamp: new Date(),
                data: {
                    trackerId: this.config.trackerId,
                    summary,
                },
            }).catch(err => {
                this.logger.error("[UsageTracker] Failed to emit summary", { error: err });
            });
        }
        
        this.logger.info("[UsageTracker] Shutdown", {
            trackerId: this.config.trackerId,
            recordCount: this.records.length,
        });
    }
    
    /**
     * Update aggregated metrics
     */
    private updateAggregatedMetrics(record: UsageRecord): void {
        let metrics = this.aggregated.get(record.resourceType);
        
        if (!metrics) {
            metrics = {
                resourceType: record.resourceType,
                total: 0,
                average: 0,
                peak: 0,
                count: 0,
                cost: 0,
                firstSeen: record.timestamp,
                lastSeen: record.timestamp,
            };
            this.aggregated.set(record.resourceType, metrics);
        }
        
        // Update metrics
        metrics.total += record.amount;
        metrics.count++;
        metrics.average = metrics.total / metrics.count;
        metrics.peak = Math.max(metrics.peak, record.amount);
        metrics.cost += record.cost || 0;
        metrics.lastSeen = record.timestamp;
    }
    
    /**
     * Clean up old records outside window
     */
    private cleanupOldRecords(): void {
        if (!this.config.windowSize) {
            return;
        }
        
        const cutoff = Date.now() - this.config.windowSize;
        const oldCount = this.records.length;
        
        // Remove old records
        let i = 0;
        while (i < this.records.length && this.records[i].timestamp.getTime() < cutoff) {
            i++;
        }
        
        if (i > 0) {
            this.records.splice(0, i);
            
            // Recalculate aggregated metrics
            this.recalculateAggregatedMetrics();
            
            this.logger.debug("[UsageTracker] Cleaned up old records", {
                trackerId: this.config.trackerId,
                removed: i,
                remaining: this.records.length,
            });
        }
    }
    
    /**
     * Recalculate aggregated metrics from scratch
     */
    private recalculateAggregatedMetrics(): void {
        this.aggregated.clear();
        
        for (const record of this.records) {
            this.updateAggregatedMetrics(record);
        }
    }
    
    /**
     * Calculate efficiency score
     */
    private calculateEfficiency(): number {
        // Simple efficiency calculation based on resource utilization
        // Can be customized based on specific needs
        
        const credits = this.aggregated.get(ResourceType.CREDITS);
        const time = this.aggregated.get(ResourceType.TIME);
        
        if (!credits || !time || credits.count === 0 || time.count === 0) {
            return 1.0; // No data, assume efficient
        }
        
        // Credits per millisecond
        const creditsPerMs = credits.total / time.total;
        
        // Compare to expected baseline (customize as needed)
        const baselineCreditsPerMs = 0.001; // 1 credit per second
        
        // Efficiency is inverse of cost - lower is better
        const efficiency = Math.min(1.0, baselineCreditsPerMs / creditsPerMs);
        
        return efficiency;
    }
    
    /**
     * Start aggregation timer
     */
    private startAggregationTimer(): void {
        if (!this.config.aggregationInterval) {
            return;
        }
        
        this.aggregationTimer = setInterval(() => {
            this.performAggregation();
        }, this.config.aggregationInterval);
    }
    
    /**
     * Perform periodic aggregation
     */
    private async performAggregation(): Promise<void> {
        const summary = this.getSummary();
        
        // Emit aggregated event
        if (this.eventBus) {
            await this.eventBus.emit({
                type: 'usage.aggregated',
                timestamp: new Date(),
                data: {
                    trackerId: this.config.trackerId,
                    summary,
                },
            });
        }
        
        this.logger.debug("[UsageTracker] Performed aggregation", {
            trackerId: this.config.trackerId,
            totalCost: summary.totalCost,
            efficiency: summary.efficiency,
        });
    }
}