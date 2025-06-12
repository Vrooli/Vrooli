/**
 * MetricsIndex provides efficient querying capabilities for metrics.
 * Maintains indexes for common query patterns to improve performance.
 */

import { UnifiedMetric, MetricType } from "../types";

/**
 * Index entry for fast lookups
 */
interface IndexEntry {
    metric: UnifiedMetric;
    timestamp: number;
}

/**
 * Time-based index bucket
 */
interface TimeBucket {
    startTime: number;
    endTime: number;
    entries: IndexEntry[];
}

/**
 * Efficient metrics indexing for fast queries
 */
export class MetricsIndex {
    // Primary indexes
    private readonly timeIndex = new Map<number, TimeBucket>(); // Bucketed by hour
    private readonly componentIndex = new Map<string, IndexEntry[]>();
    private readonly executionIndex = new Map<string, IndexEntry[]>();
    private readonly typeIndex = new Map<MetricType, IndexEntry[]>();
    private readonly nameIndex = new Map<string, IndexEntry[]>();
    
    // Configuration
    private readonly bucketSizeMs = 60 * 60 * 1000; // 1 hour buckets
    private readonly maxIndexSize = 100000; // Maximum entries per index
    
    /**
     * Add a metric to all relevant indexes
     */
    async addMetric(metric: UnifiedMetric): Promise<void> {
        const entry: IndexEntry = {
            metric,
            timestamp: metric.timestamp.getTime(),
        };
        
        // Add to time index
        this.addToTimeIndex(entry);
        
        // Add to component index
        this.addToIndex(this.componentIndex, metric.component, entry);
        
        // Add to type index
        this.addToIndex(this.typeIndex, metric.type, entry);
        
        // Add to name index
        this.addToIndex(this.nameIndex, metric.name, entry);
        
        // Add to execution index if present
        if (metric.executionId) {
            this.addToIndex(this.executionIndex, metric.executionId, entry);
        }
        
        // Prevent memory leaks by limiting index sizes
        this.enforceIndexLimits();
    }
    
    /**
     * Find metrics by time range
     */
    findByTimeRange(startTime: Date, endTime: Date): UnifiedMetric[] {
        const startMs = startTime.getTime();
        const endMs = endTime.getTime();
        const results: UnifiedMetric[] = [];
        
        // Find relevant time buckets
        const startBucket = Math.floor(startMs / this.bucketSizeMs);
        const endBucket = Math.floor(endMs / this.bucketSizeMs);
        
        for (let bucketKey = startBucket; bucketKey <= endBucket; bucketKey++) {
            const bucket = this.timeIndex.get(bucketKey);
            if (!bucket) continue;
            
            for (const entry of bucket.entries) {
                if (entry.timestamp >= startMs && entry.timestamp <= endMs) {
                    results.push(entry.metric);
                }
            }
        }
        
        return results;
    }
    
    /**
     * Find metrics by component
     */
    findByComponent(component: string): UnifiedMetric[] {
        const entries = this.componentIndex.get(component) || [];
        return entries.map(e => e.metric);
    }
    
    /**
     * Find metrics by execution ID
     */
    findByExecutionId(executionId: string): UnifiedMetric[] {
        const entries = this.executionIndex.get(executionId) || [];
        return entries.map(e => e.metric);
    }
    
    /**
     * Find metrics by type
     */
    findByType(type: MetricType): UnifiedMetric[] {
        const entries = this.typeIndex.get(type) || [];
        return entries.map(e => e.metric);
    }
    
    /**
     * Find metrics by name
     */
    findByName(name: string): UnifiedMetric[] {
        const entries = this.nameIndex.get(name) || [];
        return entries.map(e => e.metric);
    }
    
    /**
     * Get index statistics
     */
    getStats(): {
        timeBuckets: number;
        totalEntries: number;
        componentCount: number;
        executionCount: number;
        typeCount: number;
        nameCount: number;
        memoryUsageEstimate: number;
    } {
        let totalEntries = 0;
        for (const bucket of this.timeIndex.values()) {
            totalEntries += bucket.entries.length;
        }
        
        // Rough memory estimate (each entry ~200 bytes)
        const memoryUsageEstimate = totalEntries * 200;
        
        return {
            timeBuckets: this.timeIndex.size,
            totalEntries,
            componentCount: this.componentIndex.size,
            executionCount: this.executionIndex.size,
            typeCount: this.typeIndex.size,
            nameCount: this.nameIndex.size,
            memoryUsageEstimate,
        };
    }
    
    /**
     * Clear all indexes
     */
    clear(): void {
        this.timeIndex.clear();
        this.componentIndex.clear();
        this.executionIndex.clear();
        this.typeIndex.clear();
        this.nameIndex.clear();
    }
    
    /**
     * Add entry to time-based index
     */
    private addToTimeIndex(entry: IndexEntry): void {
        const bucketKey = Math.floor(entry.timestamp / this.bucketSizeMs);
        
        let bucket = this.timeIndex.get(bucketKey);
        if (!bucket) {
            bucket = {
                startTime: bucketKey * this.bucketSizeMs,
                endTime: (bucketKey + 1) * this.bucketSizeMs,
                entries: [],
            };
            this.timeIndex.set(bucketKey, bucket);
        }
        
        bucket.entries.push(entry);
        
        // Keep entries sorted by timestamp within bucket
        if (bucket.entries.length > 1) {
            bucket.entries.sort((a, b) => a.timestamp - b.timestamp);
        }
    }
    
    /**
     * Add entry to a generic index
     */
    private addToIndex<K>(index: Map<K, IndexEntry[]>, key: K, entry: IndexEntry): void {
        let entries = index.get(key);
        if (!entries) {
            entries = [];
            index.set(key, entries);
        }
        
        entries.push(entry);
        
        // Keep entries sorted by timestamp
        if (entries.length > 1) {
            entries.sort((a, b) => a.timestamp - b.timestamp);
        }
    }
    
    /**
     * Enforce limits on index sizes to prevent memory issues
     */
    private enforceIndexLimits(): void {
        // Limit time index by removing old buckets
        if (this.timeIndex.size > 168) { // Keep ~1 week of hourly buckets
            const buckets = Array.from(this.timeIndex.keys()).sort((a, b) => a - b);
            const bucketsToRemove = buckets.slice(0, buckets.length - 168);
            
            for (const bucketKey of bucketsToRemove) {
                this.timeIndex.delete(bucketKey);
            }
        }
        
        // Limit other indexes by keeping only recent entries
        this.limitIndex(this.componentIndex);
        this.limitIndex(this.executionIndex);
        this.limitIndex(this.typeIndex);
        this.limitIndex(this.nameIndex);
    }
    
    /**
     * Limit an index by keeping only the most recent entries
     */
    private limitIndex<K>(index: Map<K, IndexEntry[]>): void {
        for (const [key, entries] of index) {
            if (entries.length > this.maxIndexSize / index.size) {
                // Keep only the most recent entries
                const maxEntriesPerKey = Math.floor(this.maxIndexSize / index.size);
                entries.splice(0, entries.length - maxEntriesPerKey);
            }
        }
    }
}