/**
 * Simple Event Bus Implementation
 * 
 * Provides basic event publishing and subscription for the execution architecture.
 * This is a simplified version that can be extended with more sophisticated features
 * like Redis persistence, event replay, etc.
 */

import { type Logger } from "winston";
import { EventEmitter } from "events";
import { type ExecutionEvent } from "@vrooli/shared";

// BaseEvent interface removed - using ExecutionEvent from @vrooli/shared instead

/**
 * Event subscription configuration
 */
export interface EventSubscription {
    id: string;
    pattern: string;
    handler: (event: ExecutionEvent) => Promise<void>;
    filters?: Array<{
        field: string;
        operator: "equals" | "contains" | "startsWith" | "endsWith" | "in" | "regex";
        value: any;
    }>;
}

/**
 * Simple in-memory event bus
 */
export class EventBus {
    private readonly emitter: EventEmitter;
    private readonly logger: Logger;
    private readonly subscriptions = new Map<string, EventSubscription>();
    private readonly eventHistory: ExecutionEvent[] = [];
    private readonly maxHistorySize = 1000;

    constructor(logger: Logger) {
        this.logger = logger;
        this.emitter = new EventEmitter();
        this.emitter.setMaxListeners(100); // Allow many subscribers
    }

    /**
     * Publish an event to all matching subscribers
     */
    async publish(event: ExecutionEvent): Promise<void> {
        try {
            // Store in history
            this.addToHistory(event);

            // Log the event
            this.logger.debug("[EventBus] Publishing event", {
                eventId: event.id,
                eventType: event.type,
                source: event.source,
            });

            // Emit to pattern-based subscribers
            this.emitter.emit("event", event);

            // Also emit to type-specific listeners
            this.emitter.emit(event.type, event);

            // Process subscriptions with filters
            for (const subscription of this.subscriptions.values()) {
                if (this.matchesSubscription(event, subscription)) {
                    try {
                        await subscription.handler(event);
                    } catch (error) {
                        this.logger.error("[EventBus] Subscription handler error", {
                            subscriptionId: subscription.id,
                            eventId: event.id,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

        } catch (error) {
            this.logger.error("[EventBus] Failed to publish event", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Publish multiple events in batch
     */
    async publishBatch(events: ExecutionEvent[]): Promise<void> {
        for (const event of events) {
            await this.publish(event);
        }
    }

    /**
     * Subscribe to events with pattern matching
     */
    async subscribe(subscription: EventSubscription): Promise<void> {
        this.subscriptions.set(subscription.id, subscription);
        
        this.logger.info("[EventBus] Added subscription", {
            subscriptionId: subscription.id,
            pattern: subscription.pattern,
        });
    }

    /**
     * Unsubscribe from events
     */
    async unsubscribe(subscriptionId: string): Promise<void> {
        this.subscriptions.delete(subscriptionId);
        
        this.logger.info("[EventBus] Removed subscription", {
            subscriptionId,
        });
    }

    /**
     * Subscribe to events using Node.js EventEmitter pattern
     */
    on(eventType: string, handler: (event: ExecutionEvent) => void): void {
        this.emitter.on(eventType, handler);
    }

    /**
     * Remove event listener
     */
    off(eventType: string, handler: (event: ExecutionEvent) => void): void {
        this.emitter.off(eventType, handler);
    }

    /**
     * Get recent events matching criteria
     */
    getEvents(criteria?: {
        type?: string;
        source?: {
            tier?: string;
            component?: string;
        };
        since?: Date;
        limit?: number;
    }): ExecutionEvent[] {
        let events = [...this.eventHistory];

        if (criteria) {
            if (criteria.type) {
                events = events.filter(e => e.type === criteria.type);
            }
            if (criteria.source?.tier) {
                events = events.filter(e => e.source.tier === criteria.source!.tier);
            }
            if (criteria.source?.component) {
                events = events.filter(e => e.source.component === criteria.source!.component);
            }
            if (criteria.since) {
                events = events.filter(e => e.timestamp >= criteria.since!);
            }
        }

        // Sort by timestamp, newest first
        events.sort((a, b) => b.timestamp.getTime() - a.timestamp.getTime());

        // Apply limit
        if (criteria?.limit) {
            events = events.slice(0, criteria.limit);
        }

        return events;
    }

    /**
     * Get event stream (simplified - returns recent events)
     */
    async *getEventStream(): AsyncIterableIterator<ExecutionEvent> {
        // For a simple implementation, just yield recent events
        // In a full implementation, this would stream live events
        for (const event of this.eventHistory.slice(-10)) {
            yield event;
        }
    }

    /**
     * Start the event bus (no-op for simple implementation)
     */
    async start(): Promise<void> {
        this.logger.info("[EventBus] Started");
    }

    /**
     * Stop the event bus
     */
    async stop(): Promise<void> {
        this.emitter.removeAllListeners();
        this.subscriptions.clear();
        this.logger.info("[EventBus] Stopped");
    }

    /**
     * Get current metrics
     */
    getMetrics(): {
        totalEvents: number;
        activeSubscriptions: number;
        eventTypes: Map<string, number>;
    } {
        const eventTypes = new Map<string, number>();
        
        for (const event of this.eventHistory) {
            const count = eventTypes.get(event.type) || 0;
            eventTypes.set(event.type, count + 1);
        }

        return {
            totalEvents: this.eventHistory.length,
            activeSubscriptions: this.subscriptions.size,
            eventTypes,
        };
    }

    /**
     * Private helper methods
     */
    private addToHistory(event: ExecutionEvent): void {
        this.eventHistory.push(event);
        
        // Keep history size manageable
        if (this.eventHistory.length > this.maxHistorySize) {
            this.eventHistory.splice(0, this.eventHistory.length - this.maxHistorySize);
        }
    }

    private matchesSubscription(event: ExecutionEvent, subscription: EventSubscription): boolean {
        // Check pattern match
        if (!this.matchesPattern(event.type, subscription.pattern)) {
            return false;
        }

        // Check filters
        if (subscription.filters) {
            for (const filter of subscription.filters) {
                if (!this.matchesFilter(event, filter)) {
                    return false;
                }
            }
        }

        return true;
    }

    private matchesPattern(eventType: string, pattern: string): boolean {
        // Simple glob pattern matching
        if (pattern === "*") return true;
        if (pattern.endsWith("/*")) {
            const prefix = pattern.slice(0, -2);
            return eventType.startsWith(prefix);
        }
        if (pattern.includes("*")) {
            const regex = new RegExp(pattern.replace(/\*/g, ".*"));
            return regex.test(eventType);
        }
        return eventType === pattern;
    }

    private matchesFilter(event: ExecutionEvent, filter: {
        field: string;
        operator: "equals" | "contains" | "startsWith" | "endsWith" | "in" | "regex";
        value: any;
    }): boolean {
        const fieldValue = this.getFieldValue(event, filter.field);

        switch (filter.operator) {
            case "equals":
                return fieldValue === filter.value;
            case "contains":
                return String(fieldValue).includes(String(filter.value));
            case "startsWith":
                return String(fieldValue).startsWith(String(filter.value));
            case "endsWith":
                return String(fieldValue).endsWith(String(filter.value));
            case "in":
                return Array.isArray(filter.value) && filter.value.includes(fieldValue);
            case "regex":
                return new RegExp(String(filter.value)).test(String(fieldValue));
            default:
                return false;
        }
    }

    private getFieldValue(obj: any, field: string): any {
        const parts = field.split(".");
        let value = obj;

        for (const part of parts) {
            if (value == null) {
                return undefined;
            }
            value = value[part];
        }

        return value;
    }
}