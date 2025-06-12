/**
 * RollingHistoryAdapter - Backward compatibility adapter for RollingHistory
 * 
 * This adapter maintains the exact same interface as the original RollingHistory
 * class but routes all operations through the UnifiedMonitoringService for
 * centralized monitoring data storage and analysis.
 * 
 * Key features:
 * - Maintains 100% API compatibility with RollingHistory
 * - Routes data through UnifiedMonitoringService
 * - Converts between HistoryEvent and UnifiedMetric formats
 * - Provides same pattern detection capabilities
 * - Preserves circular buffer behavior semantics
 */

import { type EventBus, type BaseEvent } from "../../cross-cutting/events/eventBus.js";
import { type Logger } from "winston";
import { generatePK } from "@vrooli/shared";
import { UnifiedMonitoringService } from "../UnifiedMonitoringService.js";
import { type UnifiedMetric } from "../types.js";

/**
 * Event type for rolling history (maintained for compatibility)
 */
export interface HistoryEvent {
    timestamp: Date;
    type: string;
    tier: 'tier1' | 'tier2' | 'tier3';
    component: string;
    data: Record<string, unknown>;
    metadata?: Record<string, unknown>;
}

/**
 * Configuration for rolling history (maintained for compatibility)
 */
export interface RollingHistoryConfig {
    maxSize: number;
    maxAge?: number; // Maximum age in milliseconds
    persistInterval?: number; // How often to persist history
}

/**
 * Pattern detection result (maintained for compatibility)
 */
export interface PatternDetectionResult {
    totalEvents: number;
    windowMs: number;
    eventsPerMinute: number;
    hasHighActivity: boolean;
    topEventTypes: Array<[string, number]>;
    tierDistribution: Record<string, number>;
    componentDistribution: Record<string, number>;
}

/**
 * History statistics (maintained for compatibility)
 */
export interface HistoryStats {
    currentSize: number;
    maxSize: number;
    oldestEvent: Date | null;
    newestEvent: Date | null;
    eventsPerSecond: number;
}

/**
 * RollingHistoryAdapter - Maintains exact same interface as RollingHistory
 * but routes through UnifiedMonitoringService for centralized storage
 */
export class RollingHistoryAdapter {
    private readonly eventBus: EventBus;
    private readonly config: Required<RollingHistoryConfig>;
    private readonly unifiedService: UnifiedMonitoringService;
    private readonly logger?: Logger;
    private lastPersist = Date.now();
    
    // Local cache for synchronous access to recent events
    private localCache: HistoryEvent[] = [];
    private cacheUpdateTimer?: NodeJS.Timeout;
    
    // Event subscription management
    private subscriptionIds: string[] = [];

    constructor(
        eventBus: EventBus, 
        config: RollingHistoryConfig,
        logger?: Logger
    ) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.config = {
            maxSize: config.maxSize,
            maxAge: config.maxAge || 3600000, // 1 hour default
            persistInterval: config.persistInterval || 300000, // 5 minutes default
        };

        // Get unified monitoring service instance
        this.unifiedService = UnifiedMonitoringService.getInstance(
            {
                maxOverheadMs: 5,
                eventBusEnabled: true,
                mcpToolsEnabled: true,
            },
            this.eventBus
        );

        // Subscribe to all telemetry events
        this.subscribeToEvents();
        
        // Start cache update timer (update cache every 30 seconds)
        this.cacheUpdateTimer = setInterval(() => {
            this.updateLocalCache().catch(error => {
                this.logger?.warn('[RollingHistoryAdapter] Failed to update local cache:', error);
            });
        }, 30000);
        
        // Initial cache population
        this.updateLocalCache().catch(error => {
            this.logger?.warn('[RollingHistoryAdapter] Failed initial cache population:', error);
        });
    }

    /**
     * Add an event to the rolling history
     * This is a fire-and-forget operation with minimal overhead
     */
    addEvent(event: HistoryEvent): void {
        try {
            // Convert HistoryEvent to UnifiedMetric
            const metric = this.convertToUnifiedMetric(event);
            
            // Record through unified service
            this.unifiedService.record(metric).catch(error => {
                // Never let history tracking break execution
                this.logger?.error('[RollingHistoryAdapter] Failed to record metric:', error);
            });
            
            // Add to local cache immediately for synchronous access
            this.addToLocalCache(event);

            // Check if we should persist
            if (this.config.persistInterval && 
                Date.now() - this.lastPersist > this.config.persistInterval) {
                this.persistHistory();
            }
        } catch (error) {
            // Never let history tracking break execution
            this.logger?.error('[RollingHistoryAdapter] Failed to add event:', error);
            
            // Record error metric for observability
            this.recordErrorMetric('addEvent', error as Error);
        }
    }

    /**
     * Get recent events within a time window
     */
    getRecentEvents(windowMs?: number): HistoryEvent[] {
        try {
            const cutoff = windowMs ? Date.now() - windowMs : 0;
            const endTime = Date.now();
            
            return this.queryEventsFromUnified(cutoff, endTime);
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to get recent events:', error);
            return [];
        }
    }

    /**
     * Get events by tier
     */
    getEventsByTier(tier: 'tier1' | 'tier2' | 'tier3', limit?: number): HistoryEvent[] {
        try {
            const events = this.getRecentEvents()
                .filter(e => e.tier === tier);
            
            return limit ? events.slice(-limit) : events;
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to get events by tier:', error);
            return [];
        }
    }

    /**
     * Get events by type pattern
     */
    getEventsByType(typePattern: string | RegExp, limit?: number): HistoryEvent[] {
        try {
            const regex = typeof typePattern === 'string' 
                ? new RegExp(typePattern) 
                : typePattern;

            const events = this.getRecentEvents()
                .filter(e => regex.test(e.type));
            
            return limit ? events.slice(-limit) : events;
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to get events by type:', error);
            return [];
        }
    }

    /**
     * Detect patterns in recent history
     */
    detectPatterns(windowMs: number = 300000): PatternDetectionResult {
        try {
            const events = this.getRecentEvents(windowMs);
            
            // Count events by type
            const typeCounts = new Map<string, number>();
            const tierCounts = new Map<string, number>();
            const componentCounts = new Map<string, number>();
            
            for (const event of events) {
                typeCounts.set(event.type, (typeCounts.get(event.type) || 0) + 1);
                tierCounts.set(event.tier, (tierCounts.get(event.tier) || 0) + 1);
                componentCounts.set(event.component, (componentCounts.get(event.component) || 0) + 1);
            }

            // Detect anomalies (simple threshold-based)
            const avgEventsPerMinute = events.length / (windowMs / 60000);
            const hasHighActivity = avgEventsPerMinute > 100;
            
            // Find most common patterns
            const topTypes = Array.from(typeCounts.entries())
                .sort((a, b) => b[1] - a[1])
                .slice(0, 5);

            return {
                totalEvents: events.length,
                windowMs,
                eventsPerMinute: avgEventsPerMinute,
                hasHighActivity,
                topEventTypes: topTypes,
                tierDistribution: Object.fromEntries(tierCounts),
                componentDistribution: Object.fromEntries(componentCounts),
            };
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to detect patterns:', error);
            return {
                totalEvents: 0,
                windowMs,
                eventsPerMinute: 0,
                hasHighActivity: false,
                topEventTypes: [],
                tierDistribution: {},
                componentDistribution: {},
            };
        }
    }

    /**
     * Get events within a specific time range
     */
    getEventsInTimeRange(startTime: number, endTime: number): HistoryEvent[] {
        try {
            return this.queryEventsFromUnified(startTime, endTime);
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to get events in time range:', error);
            return [];
        }
    }

    /**
     * Get all events in the buffer
     */
    getAllEvents(): HistoryEvent[] {
        try {
            // Get all events from the last maxAge period
            const startTime = Date.now() - this.config.maxAge;
            const endTime = Date.now();
            
            const events = this.queryEventsFromUnified(startTime, endTime);
            
            // Limit to maxSize (most recent)
            return events.slice(-this.config.maxSize);
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to get all events:', error);
            return [];
        }
    }

    /**
     * Get valid events (alias for getAllEvents for backward compatibility)
     */
    getValid(): HistoryEvent[] {
        return this.getAllEvents();
    }

    /**
     * Get execution flow for a specific execution ID
     */
    getExecutionFlow(executionId: string): HistoryEvent[] {
        try {
            return this.getRecentEvents()
                .filter(e => 
                    e.data.executionId === executionId ||
                    e.data.runId === executionId ||
                    e.data.swarmId === executionId
                )
                .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to get execution flow:', error);
            return [];
        }
    }

    /**
     * Clear old events based on age
     */
    evictOldEvents(): void {
        // This is handled automatically by the UnifiedMonitoringService
        // through its retention policies, so this is a no-op for compatibility
        this.logger?.debug('[RollingHistoryAdapter] evictOldEvents called (handled by UnifiedMonitoringService)');
    }

    /**
     * Get current buffer stats
     */
    getStats(): HistoryStats {
        try {
            const events = this.getAllEvents();
            const sortedEvents = events.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
            
            const currentSize = events.length;
            const oldestEvent = sortedEvents.length > 0 ? sortedEvents[0].timestamp : null;
            const newestEvent = sortedEvents.length > 0 ? sortedEvents[sortedEvents.length - 1].timestamp : null;
            
            let eventsPerSecond = 0;
            if (sortedEvents.length >= 2) {
                const timeSpan = newestEvent!.getTime() - oldestEvent!.getTime();
                eventsPerSecond = timeSpan > 0 ? (currentSize * 1000) / timeSpan : 0;
            }

            return {
                currentSize,
                maxSize: this.config.maxSize,
                oldestEvent,
                newestEvent,
                eventsPerSecond,
            };
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to get stats:', error);
            return {
                currentSize: 0,
                maxSize: this.config.maxSize,
                oldestEvent: null,
                newestEvent: null,
                eventsPerSecond: 0,
            };
        }
    }

    /**
     * Private helper methods
     */

    /**
     * Subscribe to telemetry events from all tiers
     */
    private subscribeToEvents(): void {
        // Subscribe to all telemetry channels
        const channels = ['telemetry.perf', 'telemetry.biz', 'telemetry.safety'];
        
        for (const channel of channels) {
            const subscriptionId = `rolling_history_adapter_${channel}_${Date.now()}`;
            this.subscriptionIds.push(subscriptionId);
            
            this.eventBus.subscribe({
                id: subscriptionId,
                pattern: channel,
                handler: async (event: BaseEvent) => {
                    // Convert to history event
                    const historyEvent: HistoryEvent = {
                        timestamp: event.timestamp || new Date(),
                        type: event.type || 'unknown',
                        tier: this.extractTier(event),
                        component: (event.metadata as any)?.component || 'unknown',
                        data: event.data || event,
                        metadata: event.metadata,
                    };

                    this.addEvent(historyEvent);
                },
            }).catch(error => {
                this.logger?.error(`[RollingHistoryAdapter] Failed to subscribe to ${channel}:`, error);
            });
        }

        // Subscribe to execution events
        const executionSubscriptionId = `rolling_history_adapter_execution_${Date.now()}`;
        this.subscriptionIds.push(executionSubscriptionId);
        
        this.eventBus.subscribe({
            id: executionSubscriptionId,
            pattern: 'execution.*',
            handler: async (event: BaseEvent) => {
                const historyEvent: HistoryEvent = {
                    timestamp: event.timestamp || new Date(),
                    type: `execution.${event.type}`,
                    tier: this.extractTier(event),
                    component: (event as any).component || 'unknown',
                    data: event.data || event,
                };

                this.addEvent(historyEvent);
            },
        }).catch(error => {
            this.logger?.error('[RollingHistoryAdapter] Failed to subscribe to execution events:', error);
        });
    }

    /**
     * Extract tier from event metadata with proper type validation
     */
    private extractTier(event: BaseEvent): 'tier1' | 'tier2' | 'tier3' {
        // Type-safe tier extraction from metadata
        const tier = event.metadata?.tier;
        if (tier === 'tier1' || tier === 'tier2' || tier === 'tier3') {
            return tier;
        }
        
        // Check event data for tier hints with type guards
        const data = event.data;
        if (data && typeof data === 'object') {
            if ('swarmId' in data) return 'tier1';
            if ('runId' in data) return 'tier2';
            if ('stepId' in data) return 'tier3';
        }
        
        // Fallback based on event type
        if (event.type) {
            if (event.type.includes('swarm') || event.type.includes('coordination')) return 'tier1';
            if (event.type.includes('run') || event.type.includes('orchestration')) return 'tier2';
        }
        
        return 'tier3'; // Default fallback
    }

    /**
     * Convert HistoryEvent to UnifiedMetric
     */
    private convertToUnifiedMetric(event: HistoryEvent): Omit<UnifiedMetric, 'id' | 'timestamp'> {
        // Determine metric type based on event type
        let metricType: 'performance' | 'business' | 'health' | 'safety';
        if (event.type.includes('perf') || event.type.includes('duration') || event.type.includes('latency')) {
            metricType = 'performance';
        } else if (event.type.includes('error') || event.type.includes('health') || event.type.includes('status')) {
            metricType = 'health';
        } else if (event.type.includes('safety') || event.type.includes('security') || event.type.includes('validation')) {
            metricType = 'safety';
        } else {
            metricType = 'business';
        }

        // Extract numeric value if possible
        let value: number | string | boolean = 1; // Default count
        if (event.data.value !== undefined) {
            value = event.data.value as number | string | boolean;
        } else if (event.data.duration !== undefined) {
            value = event.data.duration as number;
        } else if (event.data.count !== undefined) {
            value = event.data.count as number;
        }

        return {
            tier: event.tier === 'tier1' ? 1 : event.tier === 'tier2' ? 2 : 3,
            component: event.component,
            type: metricType,
            name: event.type,
            value,
            executionId: (event.data.executionId || event.data.runId || event.data.swarmId) as string,
            metadata: {
                ...event.metadata,
                originalData: event.data,
                source: 'rolling_history_adapter',
            },
        };
    }

    /**
     * Query events from local cache (synchronous access to recent data)
     * Uses local cache populated by addEvent() and periodic updates
     */
    private queryEventsFromUnified(startTime: number, endTime: number): HistoryEvent[] {
        try {
            // Filter events by time range from local cache
            return this.localCache.filter(event => {
                const eventTime = event.timestamp.getTime();
                return eventTime >= startTime && eventTime <= endTime;
            });
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Error querying from cache:', error);
            return [];
        }
    }

    /**
     * Convert UnifiedMetric back to HistoryEvent format
     */
    private convertToHistoryEvent(metric: UnifiedMetric): HistoryEvent {
        const tier = metric.tier === 1 ? 'tier1' : metric.tier === 2 ? 'tier2' : 'tier3';
        
        return {
            timestamp: metric.timestamp,
            type: metric.name,
            tier,
            component: metric.component,
            data: {
                value: metric.value,
                executionId: metric.executionId,
                ...(metric.metadata?.originalData as Record<string, unknown> || {}),
            },
            metadata: metric.metadata,
        };
    }

    /**
     * Add event to local cache with size and age limits
     */
    private addToLocalCache(event: HistoryEvent): void {
        // Add to front of cache
        this.localCache.unshift(event);
        
        // Hard limit enforcement to prevent unbounded growth if timer fails
        const hardLimit = this.config.maxSize * 2;
        if (this.localCache.length > hardLimit) {
            this.localCache = this.localCache.slice(0, this.config.maxSize);
            this.logger?.warn('[RollingHistoryAdapter] Local cache hit hard limit, force trimming');
        }
        
        // Enforce size limit
        if (this.localCache.length > this.config.maxSize) {
            this.localCache = this.localCache.slice(0, this.config.maxSize);
        }
        
        // Enforce age limit
        const cutoffTime = Date.now() - this.config.maxAge;
        this.localCache = this.localCache.filter(e => e.timestamp.getTime() > cutoffTime);
    }
    
    /**
     * Update local cache from UnifiedMonitoringService (async operation)
     */
    private async updateLocalCache(): Promise<void> {
        try {
            // For now, we maintain the cache through addEvent() calls
            // In future versions, we could query the UnifiedMonitoringService here
            
            // Clean up old events from cache
            const cutoffTime = Date.now() - this.config.maxAge;
            this.localCache = this.localCache.filter(e => e.timestamp.getTime() > cutoffTime);
            
            // Sort by timestamp (newest first)
            this.localCache.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime());
            
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Error updating local cache:', error);
        }
    }
    
    /**
     * Record error metrics for monitoring service health
     */
    private recordErrorMetric(operation: string, error: Error): void {
        try {
            const errorMetric = {
                tier: 'cross-cutting' as const,
                component: 'rolling_history_adapter',
                type: 'health' as const,
                name: 'adapter_error',
                value: 1,
                metadata: {
                    operation,
                    errorMessage: error.message,
                    errorStack: error.stack?.substring(0, 500), // Truncate stack trace
                },
            };
            
            this.unifiedService.record(errorMetric).catch(() => {
                // If we can't record the error metric, just log it
                this.logger?.warn('[RollingHistoryAdapter] Failed to record error metric');
            });
        } catch (metricError) {
            // Don't let error metric recording cause more errors
            this.logger?.warn('[RollingHistoryAdapter] Error in recordErrorMetric:', metricError);
        }
    }

    /**
     * Cleanup resources including timer and event subscriptions
     */
    async cleanup(): Promise<void> {
        // Clear timer
        if (this.cacheUpdateTimer) {
            clearInterval(this.cacheUpdateTimer);
            this.cacheUpdateTimer = undefined;
        }
        
        // Unsubscribe from all event subscriptions
        for (const subscriptionId of this.subscriptionIds) {
            try {
                await this.eventBus.unsubscribe(subscriptionId);
            } catch (error) {
                this.logger?.warn(`[RollingHistoryAdapter] Failed to unsubscribe ${subscriptionId}:`, error);
            }
        }
        this.subscriptionIds = [];
        
        // Clear cache
        this.localCache = [];
    }

    /**
     * Persist history to event bus for external consumption
     */
    private persistHistory(): void {
        try {
            const snapshot = {
                timestamp: new Date(),
                events: this.getRecentEvents(),
                patterns: this.detectPatterns(),
            };

            this.eventBus.publish({
                id: generatePK().toString(),
                type: 'history.snapshot',
                timestamp: new Date(),
                source: {
                    tier: 'cross-cutting',
                    component: 'RollingHistoryAdapter',
                    instanceId: 'rolling_history_adapter',
                },
                data: snapshot,
            });
            this.lastPersist = Date.now();
        } catch (error) {
            this.logger?.error('[RollingHistoryAdapter] Failed to persist history:', error);
        }
    }
}