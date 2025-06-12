/**
 * MetricsStore consolidates storage from RollingHistory, StrategyMetricsStore,
 * and other storage implementations into a unified storage layer.
 */

// logger will be console for now
import { CircularBuffer, TimeBasedCircularBuffer } from "../storage/CircularBuffer";
import { MetricsIndex } from "../storage/MetricsIndex";
import {
    UnifiedMetric,
    MonitoringConfig,
    MetricQuery,
    MetricQueryResult,
    RetentionPolicy,
    MetricType,
} from "../types";

/**
 * Storage bucket for a specific metric type/tier combination
 */
interface StorageBucket {
    buffer: TimeBasedCircularBuffer<UnifiedMetric>;
    retentionPolicy: RetentionPolicy;
    totalStored: number;
    lastAccessed: Date;
}

/**
 * Unified metrics storage with efficient querying and retention management
 */
export class MetricsStore {
    private readonly buckets = new Map<string, StorageBucket>();
    private readonly index: MetricsIndex;
    private isInitialized = false;
    private maintenanceTimer?: NodeJS.Timeout;
    
    constructor(private readonly config: MonitoringConfig) {
        this.index = new MetricsIndex();
    }
    
    /**
     * Initialize the store and create storage buckets
     */
    async initialize(): Promise<void> {
        if (this.isInitialized) return;
        
        try {
            console.debug("Initializing MetricsStore");
            
            // Create storage buckets based on retention policies
            for (const policy of this.config.retentionPolicies) {
                const bucketKey = this.getBucketKey(policy.tier, policy.metricType);
                const capacity = this.config.bufferSizes[`tier-${policy.tier}`] || 10000;
                const maxAgeMs = policy.retentionDays * 24 * 60 * 60 * 1000;
                
                const bucket: StorageBucket = {
                    buffer: new TimeBasedCircularBuffer<UnifiedMetric>(
                        capacity,
                        maxAgeMs,
                        (metric) => metric.timestamp
                    ),
                    retentionPolicy: policy,
                    totalStored: 0,
                    lastAccessed: new Date(),
                };
                
                this.buckets.set(bucketKey, bucket);
                console.debug(`Created storage bucket: ${bucketKey}`, {
                    capacity,
                    retentionDays: policy.retentionDays,
                });
            }
            
            // Start background maintenance
            this.startMaintenance();
            
            this.isInitialized = true;
            console.info("MetricsStore initialized successfully", {
                buckets: this.buckets.size,
            });
            
        } catch (error) {
            console.error("Failed to initialize MetricsStore", { error });
            throw error;
        }
    }
    
    /**
     * Store a metric in the appropriate bucket
     */
    async store(metric: UnifiedMetric): Promise<void> {
        if (!this.isInitialized) {
            throw new Error("MetricsStore not initialized");
        }
        
        try {
            const bucketKey = this.getBucketKey(metric.tier, metric.type);
            let bucket = this.buckets.get(bucketKey);
            
            // Create bucket if it doesn't exist (dynamic bucket creation)
            if (!bucket) {
                bucket = await this.createDynamicBucket(metric.tier, metric.type);
                this.buckets.set(bucketKey, bucket);
            }
            
            // Store metric
            bucket.buffer.add(metric);
            bucket.totalStored++;
            bucket.lastAccessed = new Date();
            
            // Update index for efficient querying
            await this.index.addMetric(metric);
            
        } catch (error) {
            console.error("Error storing metric", { error, metric: metric.name });
            // Don't throw - maintain resilience
        }
    }
    
    /**
     * Store multiple metrics efficiently
     */
    async storeBatch(metrics: UnifiedMetric[]): Promise<void> {
        if (!this.isInitialized || metrics.length === 0) return;
        
        try {
            // Group metrics by bucket
            const bucketGroups = new Map<string, UnifiedMetric[]>();
            
            for (const metric of metrics) {
                const bucketKey = this.getBucketKey(metric.tier, metric.type);
                const group = bucketGroups.get(bucketKey) || [];
                group.push(metric);
                bucketGroups.set(bucketKey, group);
            }
            
            // Store each group
            for (const [bucketKey, groupMetrics] of bucketGroups) {
                let bucket = this.buckets.get(bucketKey);
                
                if (!bucket) {
                    const firstMetric = groupMetrics[0];
                    bucket = await this.createDynamicBucket(firstMetric.tier, firstMetric.type);
                    this.buckets.set(bucketKey, bucket);
                }
                
                bucket.buffer.addBatch(groupMetrics);
                bucket.totalStored += groupMetrics.length;
                bucket.lastAccessed = new Date();
                
                // Update index
                for (const metric of groupMetrics) {
                    await this.index.addMetric(metric);
                }
            }
            
        } catch (error) {
            console.error("Error storing metric batch", { error, count: metrics.length });
        }
    }
    
    /**
     * Query metrics with filtering and pagination
     */
    async query(query: MetricQuery): Promise<MetricQueryResult> {
        if (!this.isInitialized) {
            throw new Error("MetricsStore not initialized");
        }
        
        const startTime = process.hrtime.bigint();
        
        try {
            let allMetrics: UnifiedMetric[] = [];
            
            // Determine which buckets to search
            const bucketsToSearch = this.getBucketsForQuery(query);
            
            // Collect metrics from relevant buckets
            for (const bucket of bucketsToSearch) {
                bucket.lastAccessed = new Date();
                
                let bucketMetrics: UnifiedMetric[];
                
                if (query.startTime || query.endTime) {
                    const start = query.startTime || new Date(0);
                    const end = query.endTime || new Date();
                    bucketMetrics = bucket.buffer.getInTimeRange(start, end, m => m.timestamp);
                } else {
                    bucketMetrics = bucket.buffer.getAll();
                }
                
                allMetrics.push(...bucketMetrics);
            }
            
            // Apply filters
            allMetrics = this.applyFilters(allMetrics, query);
            
            // Apply sorting
            allMetrics = this.applySorting(allMetrics, query);
            
            // Apply pagination
            const total = allMetrics.length;
            const offset = query.offset || 0;
            const limit = query.limit || 1000;
            const paginatedMetrics = allMetrics.slice(offset, offset + limit);
            
            const endTime = process.hrtime.bigint();
            const executionTime = Number(endTime - startTime) / 1_000_000;
            
            return {
                metrics: paginatedMetrics,
                total,
                query,
                executionTime,
            };
            
        } catch (error) {
            console.error("Error querying metrics", { error, query });
            throw error;
        }
    }
    
    /**
     * Get metrics for a specific execution context
     */
    async getByExecutionId(executionId: string, limit?: number): Promise<UnifiedMetric[]> {
        return (await this.query({
            executionId,
            limit,
            orderBy: "timestamp",
            order: "desc",
        })).metrics;
    }
    
    /**
     * Get recent metrics of a specific type
     */
    async getRecent(
        type: MetricType,
        count: number = 100,
        tier?: 1 | 2 | 3 | "cross-cutting"
    ): Promise<UnifiedMetric[]> {
        return (await this.query({
            type: [type],
            tier: tier ? [tier] : undefined,
            limit: count,
            orderBy: "timestamp",
            order: "desc",
        })).metrics;
    }
    
    /**
     * Enforce retention policies by cleaning old data
     */
    async enforceRetentionPolicies(): Promise<void> {
        console.debug("Enforcing retention policies");
        
        let totalEvicted = 0;
        
        for (const [bucketKey, bucket] of this.buckets) {
            const beforeSize = bucket.buffer.size();
            
            // Time-based buffers automatically evict old entries
            // Just need to trigger cleanup
            const validMetrics = bucket.buffer.getValid();
            
            const afterSize = validMetrics.length;
            const evicted = beforeSize - afterSize;
            totalEvicted += evicted;
            
            if (evicted > 0) {
                console.debug(`Evicted ${evicted} metrics from bucket ${bucketKey}`);
            }
        }
        
        if (totalEvicted > 0) {
            console.info(`Retention policy enforcement complete`, { totalEvicted });
        }
    }
    
    /**
     * Downsample old metrics to reduce storage
     */
    async downsampleOldMetrics(): Promise<void> {
        if (!this.config.enableAutoDownsampling) return;
        
        console.debug("Starting metric downsampling");
        
        // Implementation for downsampling would go here
        // For now, just log that it's not implemented
        console.debug("Metric downsampling not yet implemented");
    }
    
    /**
     * Get storage statistics
     */
    async getStats(): Promise<{
        buckets: number;
        totalMetrics: number;
        memoryUsage: number;
        oldestMetric?: Date;
        newestMetric?: Date;
    }> {
        let totalMetrics = 0;
        let oldestMetric: Date | undefined;
        let newestMetric: Date | undefined;
        
        for (const bucket of this.buckets.values()) {
            totalMetrics += bucket.buffer.size();
            
            const bucketMetrics = bucket.buffer.getAll();
            if (bucketMetrics.length > 0) {
                const bucketOldest = bucketMetrics[0].timestamp;
                const bucketNewest = bucketMetrics[bucketMetrics.length - 1].timestamp;
                
                if (!oldestMetric || bucketOldest < oldestMetric) {
                    oldestMetric = bucketOldest;
                }
                if (!newestMetric || bucketNewest > newestMetric) {
                    newestMetric = bucketNewest;
                }
            }
        }
        
        // Rough memory usage estimation
        const memoryUsage = totalMetrics * 500; // ~500 bytes per metric estimate
        
        return {
            buckets: this.buckets.size,
            totalMetrics,
            memoryUsage,
            oldestMetric,
            newestMetric,
        };
    }
    
    /**
     * Get health status of the store
     */
    async getHealth(): Promise<{
        status: "healthy" | "degraded" | "unhealthy";
        details: Record<string, any>;
    }> {
        const stats = await this.getStats();
        const maxMemoryMB = 1000; // 1GB limit
        const memoryUsageMB = stats.memoryUsage / (1024 * 1024);
        
        let status: "healthy" | "degraded" | "unhealthy" = "healthy";
        
        if (memoryUsageMB > maxMemoryMB * 0.9) {
            status = "unhealthy";
        } else if (memoryUsageMB > maxMemoryMB * 0.7) {
            status = "degraded";
        }
        
        return {
            status,
            details: {
                ...stats,
                memoryUsageMB,
                maxMemoryMB,
                initialized: this.isInitialized,
            },
        };
    }
    
    /**
     * Close the store and cleanup resources
     */
    async close(): Promise<void> {
        console.info("Closing MetricsStore");
        
        if (this.maintenanceTimer) {
            clearInterval(this.maintenanceTimer);
            this.maintenanceTimer = undefined;
        }
        
        // Stop all time-based buffers
        for (const bucket of this.buckets.values()) {
            bucket.buffer.stop();
        }
        
        this.buckets.clear();
        this.isInitialized = false;
    }
    
    /**
     * Get bucket key for tier/type combination
     */
    private getBucketKey(tier: 1 | 2 | 3 | "cross-cutting", type: MetricType): string {
        return `${tier}-${type}`;
    }
    
    /**
     * Create a dynamic bucket for unknown tier/type combinations
     */
    private async createDynamicBucket(
        tier: 1 | 2 | 3 | "cross-cutting",
        type: MetricType
    ): Promise<StorageBucket> {
        const capacity = this.config.bufferSizes[`tier-${tier}`] || 10000;
        const retentionDays = 7; // Default retention
        const maxAgeMs = retentionDays * 24 * 60 * 60 * 1000;
        
        const policy: RetentionPolicy = {
            tier,
            metricType: type,
            retentionDays,
        };
        
        const bucket: StorageBucket = {
            buffer: new TimeBasedCircularBuffer<UnifiedMetric>(
                capacity,
                maxAgeMs,
                (metric) => metric.timestamp
            ),
            retentionPolicy: policy,
            totalStored: 0,
            lastAccessed: new Date(),
        };
        
        console.debug(`Created dynamic bucket for ${tier}-${type}`, {
            capacity,
            retentionDays,
        });
        
        return bucket;
    }
    
    /**
     * Get buckets relevant to a query
     */
    private getBucketsForQuery(query: MetricQuery): StorageBucket[] {
        const buckets: StorageBucket[] = [];
        
        for (const [bucketKey, bucket] of this.buckets) {
            const [tierStr, typeStr] = bucketKey.split('-');
            const tier = tierStr === "cross-cutting" ? tierStr : parseInt(tierStr) as 1 | 2 | 3;
            const type = typeStr as MetricType;
            
            // Check if bucket matches query filters
            if (query.tier && !query.tier.includes(tier)) continue;
            if (query.type && !query.type.includes(type)) continue;
            
            buckets.push(bucket);
        }
        
        return buckets;
    }
    
    /**
     * Apply query filters to metrics
     */
    private applyFilters(metrics: UnifiedMetric[], query: MetricQuery): UnifiedMetric[] {
        return metrics.filter(metric => {
            if (query.component && !query.component.includes(metric.component)) return false;
            if (query.name && (Array.isArray(query.name) ? !query.name.includes(metric.name) : metric.name !== query.name)) return false;
            if (query.executionId && metric.executionId !== query.executionId) return false;
            if (query.userId && metric.userId !== query.userId) return false;
            if (query.teamId && metric.teamId !== query.teamId) return false;
            if (query.tags && (!metric.tags || !query.tags.some(tag => metric.tags!.includes(tag)))) return false;
            
            return true;
        });
    }
    
    /**
     * Apply sorting to metrics
     */
    private applySorting(metrics: UnifiedMetric[], query: MetricQuery): UnifiedMetric[] {
        const orderBy = query.orderBy || "timestamp";
        const order = query.order || "desc";
        
        return metrics.sort((a, b) => {
            let aVal: any, bVal: any;
            
            switch (orderBy) {
                case "timestamp":
                    aVal = a.timestamp.getTime();
                    bVal = b.timestamp.getTime();
                    break;
                case "value":
                    aVal = typeof a.value === "number" ? a.value : 0;
                    bVal = typeof b.value === "number" ? b.value : 0;
                    break;
                case "name":
                    aVal = a.name;
                    bVal = b.name;
                    break;
                default:
                    aVal = a.timestamp.getTime();
                    bVal = b.timestamp.getTime();
            }
            
            if (order === "asc") {
                return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
            } else {
                return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
            }
        });
    }
    
    /**
     * Start background maintenance tasks
     */
    private startMaintenance(): void {
        // Run maintenance every 5 minutes
        this.maintenanceTimer = setInterval(async () => {
            try {
                await this.enforceRetentionPolicies();
                
                // Clean up unused buckets (not accessed in last hour)
                const oneHourAgo = new Date(Date.now() - 60 * 60 * 1000);
                for (const [bucketKey, bucket] of this.buckets) {
                    if (bucket.lastAccessed < oneHourAgo && bucket.buffer.isEmpty()) {
                        bucket.buffer.stop();
                        this.buckets.delete(bucketKey);
                        console.debug(`Cleaned up unused bucket: ${bucketKey}`);
                    }
                }
            } catch (error) {
                console.error("Error in MetricsStore maintenance", { error });
            }
        }, 5 * 60 * 1000);
    }
}