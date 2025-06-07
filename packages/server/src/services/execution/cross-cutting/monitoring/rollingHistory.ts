import { type EventBus } from "../events/eventBus.js";

/**
 * Event type for rolling history
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
 * Configuration for rolling history
 */
export interface RollingHistoryConfig {
    maxSize: number;
    maxAge?: number; // Maximum age in milliseconds
    persistInterval?: number; // How often to persist history
}

/**
 * RollingHistory - Maintains a circular buffer of recent execution events
 * 
 * This component provides a fire-and-forget mechanism for tracking recent
 * execution history across all tiers. It maintains a fixed-size buffer that
 * automatically evicts old events when full.
 * 
 * Key features:
 * - Fixed-size circular buffer
 * - Time-based eviction (optional)
 * - Pattern detection support
 * - Zero-overhead event emission
 * - Query capabilities for analysis
 * 
 * The history is designed to enable emergent monitoring capabilities
 * without imposing overhead on the execution path.
 */
export class RollingHistory {
    private readonly eventBus: EventBus;
    private readonly config: RollingHistoryConfig;
    private readonly buffer: HistoryEvent[] = [];
    private writeIndex = 0;
    private size = 0;
    private lastPersist = Date.now();

    constructor(eventBus: EventBus, config: RollingHistoryConfig) {
        this.eventBus = eventBus;
        this.config = {
            maxSize: config.maxSize,
            maxAge: config.maxAge || 3600000, // 1 hour default
            persistInterval: config.persistInterval || 300000, // 5 minutes default
        };

        // Subscribe to all telemetry events
        this.subscribeToEvents();
    }

    /**
     * Add an event to the rolling history
     * This is a fire-and-forget operation with minimal overhead
     */
    addEvent(event: HistoryEvent): void {
        try {
            // Add to circular buffer
            if (this.size < this.config.maxSize) {
                this.buffer.push(event);
                this.size++;
            } else {
                // Overwrite oldest event
                this.buffer[this.writeIndex] = event;
            }

            // Move write index
            this.writeIndex = (this.writeIndex + 1) % this.config.maxSize;

            // Check if we should persist
            if (this.config.persistInterval && 
                Date.now() - this.lastPersist > this.config.persistInterval) {
                this.persistHistory();
            }
        } catch (error) {
            // Never let history tracking break execution
            console.error('[RollingHistory] Failed to add event:', error);
        }
    }

    /**
     * Get recent events within a time window
     */
    getRecentEvents(windowMs?: number): HistoryEvent[] {
        const cutoff = windowMs ? Date.now() - windowMs : 0;
        const events: HistoryEvent[] = [];

        // Collect events newer than cutoff
        for (let i = 0; i < this.size; i++) {
            const event = this.buffer[i];
            if (event && event.timestamp.getTime() > cutoff) {
                events.push(event);
            }
        }

        // Sort by timestamp
        return events.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
    }

    /**
     * Get events by tier
     */
    getEventsByTier(tier: 'tier1' | 'tier2' | 'tier3', limit?: number): HistoryEvent[] {
        const events = this.getRecentEvents()
            .filter(e => e.tier === tier);
        
        return limit ? events.slice(-limit) : events;
    }

    /**
     * Get events by type pattern
     */
    getEventsByType(typePattern: string | RegExp, limit?: number): HistoryEvent[] {
        const regex = typeof typePattern === 'string' 
            ? new RegExp(typePattern) 
            : typePattern;

        const events = this.getRecentEvents()
            .filter(e => regex.test(e.type));
        
        return limit ? events.slice(-limit) : events;
    }

    /**
     * Detect patterns in recent history
     */
    detectPatterns(windowMs: number = 300000): PatternDetectionResult {
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
    }

    /**
     * Get execution flow for a specific execution ID
     */
    getExecutionFlow(executionId: string): HistoryEvent[] {
        return this.getRecentEvents()
            .filter(e => 
                e.data.executionId === executionId ||
                e.data.runId === executionId ||
                e.data.swarmId === executionId
            )
            .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
    }

    /**
     * Clear old events based on age
     */
    evictOldEvents(): void {
        if (!this.config.maxAge) return;

        const cutoff = Date.now() - this.config.maxAge;
        const validEvents: HistoryEvent[] = [];

        for (let i = 0; i < this.size; i++) {
            const event = this.buffer[i];
            if (event && event.timestamp.getTime() > cutoff) {
                validEvents.push(event);
            }
        }

        // Rebuild buffer
        this.buffer.length = 0;
        this.buffer.push(...validEvents);
        this.size = validEvents.length;
        this.writeIndex = this.size % this.config.maxSize;
    }

    /**
     * Subscribe to telemetry events from all tiers
     */
    private subscribeToEvents(): void {
        // Subscribe to all telemetry channels
        const channels = ['telemetry.perf', 'telemetry.biz', 'telemetry.safety'];
        
        for (const channel of channels) {
            this.eventBus.subscribe(channel, (event: any) => {
                // Convert to history event
                const historyEvent: HistoryEvent = {
                    timestamp: event.timestamp || new Date(),
                    type: event.type || 'unknown',
                    tier: this.extractTier(event),
                    component: event.metadata?.component || 'unknown',
                    data: event,
                    metadata: event.metadata,
                };

                this.addEvent(historyEvent);
            });
        }

        // Subscribe to execution events
        this.eventBus.subscribe('execution.*', (event: any) => {
            const historyEvent: HistoryEvent = {
                timestamp: event.timestamp || new Date(),
                type: `execution.${event.type}`,
                tier: this.extractTier(event),
                component: event.component || 'unknown',
                data: event,
            };

            this.addEvent(historyEvent);
        });
    }

    /**
     * Extract tier from event metadata
     */
    private extractTier(event: any): 'tier1' | 'tier2' | 'tier3' {
        if (event.metadata?.tier) {
            return event.metadata.tier;
        }
        if (event.swarmId) return 'tier1';
        if (event.runId) return 'tier2';
        return 'tier3';
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

            this.eventBus.publish('history.snapshot', snapshot);
            this.lastPersist = Date.now();
        } catch (error) {
            console.error('[RollingHistory] Failed to persist history:', error);
        }
    }

    /**
     * Get current buffer stats
     */
    getStats(): HistoryStats {
        return {
            currentSize: this.size,
            maxSize: this.config.maxSize,
            oldestEvent: this.size > 0 ? this.buffer[0].timestamp : null,
            newestEvent: this.size > 0 
                ? this.buffer[(this.writeIndex - 1 + this.config.maxSize) % this.config.maxSize].timestamp 
                : null,
            eventsPerSecond: this.calculateEventRate(),
        };
    }

    /**
     * Calculate current event rate
     */
    private calculateEventRate(): number {
        if (this.size < 2) return 0;

        const oldest = this.buffer[0].timestamp.getTime();
        const newest = this.buffer[(this.writeIndex - 1 + this.config.maxSize) % this.config.maxSize].timestamp.getTime();
        const durationSeconds = (newest - oldest) / 1000;

        return durationSeconds > 0 ? this.size / durationSeconds : 0;
    }
}

/**
 * Pattern detection result
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
 * History statistics
 */
export interface HistoryStats {
    currentSize: number;
    maxSize: number;
    oldestEvent: Date | null;
    newestEvent: Date | null;
    eventsPerSecond: number;
}