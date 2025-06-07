/**
 * Core event bus interface
 * Defines the contract for event-driven communication between tiers
 */

import { type BaseEvent, type EventFilter, type EventQuery, type EventSubscription } from "../types/events.js";

/**
 * Event bus interface
 * All implementations must conform to this interface
 */
export interface IEventBus {
    // Publishing
    publish(event: BaseEvent): Promise<void>;
    publishBatch(events: BaseEvent[]): Promise<void>;

    // Subscribing
    subscribe(subscription: EventSubscription): Promise<void>;
    unsubscribe(subscriptionId: string): Promise<void>;

    // Query
    getEvents(query: EventQuery): Promise<BaseEvent[]>;
    getEventStream(query: EventQuery): AsyncIterableIterator<BaseEvent>;

    // Lifecycle
    start(): Promise<void>;
    stop(): Promise<void>;
}

/**
 * Event builder for creating well-formed events
 */
export class EventBuilder<T extends BaseEvent = BaseEvent> {
    private event: Partial<T>;

    constructor() {
        this.event = {
            timestamp: new Date(),
            metadata: {
                version: "1.0.0",
                tags: [],
                priority: "NORMAL" as any,
            },
        } as Partial<T>;
    }

    withId(id: string): this {
        this.event.id = id;
        return this;
    }

    withType(type: string): this {
        this.event.type = type;
        return this;
    }

    withSource(tier: 1 | 2 | 3 | "cross-cutting", component: string, instanceId: string): this {
        this.event.source = { tier, component, instanceId };
        return this;
    }

    withCorrelationId(correlationId: string): this {
        this.event.correlationId = correlationId;
        return this;
    }

    withCausationId(causationId: string): this {
        this.event.causationId = causationId;
        return this;
    }

    withUserId(userId: string): this {
        if (!this.event.metadata) {
            this.event.metadata = {} as any;
        }
        this.event.metadata.userId = userId;
        return this;
    }

    withSessionId(sessionId: string): this {
        if (!this.event.metadata) {
            this.event.metadata = {} as any;
        }
        this.event.metadata.sessionId = sessionId;
        return this;
    }

    withRequestId(requestId: string): this {
        if (!this.event.metadata) {
            this.event.metadata = {} as any;
        }
        this.event.metadata.requestId = requestId;
        return this;
    }

    withPriority(priority: "LOW" | "NORMAL" | "HIGH" | "CRITICAL"): this {
        if (!this.event.metadata) {
            this.event.metadata = {} as any;
        }
        this.event.metadata.priority = priority as any;
        return this;
    }

    withTags(...tags: string[]): this {
        if (!this.event.metadata) {
            this.event.metadata = {} as any;
        }
        this.event.metadata.tags = [...(this.event.metadata.tags || []), ...tags];
        return this;
    }

    withTTL(seconds: number): this {
        if (!this.event.metadata) {
            this.event.metadata = {} as any;
        }
        this.event.metadata.ttl = seconds;
        return this;
    }

    withPayload(payload: any): this {
        (this.event as any).payload = payload;
        return this;
    }

    build(): T {
        // Validate required fields
        if (!this.event.id) {
            throw new Error("Event ID is required");
        }
        if (!this.event.type) {
            throw new Error("Event type is required");
        }
        if (!this.event.source) {
            throw new Error("Event source is required");
        }
        if (!this.event.correlationId) {
            throw new Error("Correlation ID is required");
        }

        return this.event as T;
    }
}

/**
 * Event filter utility functions
 */
export class EventFilterUtils {
    static matches(event: BaseEvent, filter: EventFilter): boolean {
        const value = this.getFieldValue(event, filter.field);

        switch (filter.operator) {
            case "equals":
                return value === filter.value;

            case "contains":
                return String(value).includes(String(filter.value));

            case "startsWith":
                return String(value).startsWith(String(filter.value));

            case "endsWith":
                return String(value).endsWith(String(filter.value));

            case "in":
                return Array.isArray(filter.value) && filter.value.includes(value);

            case "regex":
                return new RegExp(String(filter.value)).test(String(value));

            default:
                return false;
        }
    }

    static matchesAll(event: BaseEvent, filters: EventFilter[]): boolean {
        return filters.every(filter => this.matches(event, filter));
    }

    private static getFieldValue(obj: any, field: string): any {
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

/**
 * Event correlation tracker
 */
export class EventCorrelator {
    private correlations = new Map<string, Set<string>>();

    addEvent(event: BaseEvent): void {
        const { correlationId, id } = event;

        if (!this.correlations.has(correlationId)) {
            this.correlations.set(correlationId, new Set());
        }

        this.correlations.get(correlationId)!.add(id);

        // Also track causation chains
        if (event.causationId) {
            this.addCausation(event.causationId, id);
        }
    }

    getCorrelatedEvents(correlationId: string): string[] {
        return Array.from(this.correlations.get(correlationId) || []);
    }

    private addCausation(parentId: string, childId: string): void {
        // Find the correlation that contains the parent
        for (const [_corrId, eventIds] of this.correlations.entries()) {
            if (eventIds.has(parentId)) {
                eventIds.add(childId);
                break;
            }
        }
    }

    clear(): void {
        this.correlations.clear();
    }
}

/**
 * Event replay utility
 */
export class EventReplayer {
    constructor(private eventBus: IEventBus) { }

    async replay(
        events: BaseEvent[],
        speed = 1,
        onEvent?: (event: BaseEvent) => void,
    ): Promise<void> {
        if (events.length === 0) return;

        // Sort events by timestamp
        const sortedEvents = [...events].sort(
            (a, b) => a.timestamp.getTime() - b.timestamp.getTime(),
        );

        const startTime = sortedEvents[0].timestamp.getTime();
        const replayStartTime = Date.now();

        for (const event of sortedEvents) {
            // Calculate when this event should be replayed
            const eventOffset = event.timestamp.getTime() - startTime;
            const replayTime = replayStartTime + (eventOffset / speed);
            const delay = replayTime - Date.now();

            if (delay > 0) {
                await new Promise(resolve => setTimeout(resolve, delay));
            }

            // Replay the event
            await this.eventBus.publish(event);

            if (onEvent) {
                onEvent(event);
            }
        }
    }
}
