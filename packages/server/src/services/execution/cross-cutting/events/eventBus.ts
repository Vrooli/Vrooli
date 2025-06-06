/**
 * Redis-based event bus implementation
 * Provides reliable event-driven communication between tiers
 */

import { Redis } from "ioredis";
import { 
    IEventBus, 
    BaseEvent, 
    EventSubscription, 
    EventQuery,
    EventFilterUtils,
    EventValidator,
    EventSanitizer
} from "@local/shared";
import { logger } from "../../../../events/logger";
import { getRedisConnection } from "../../../../redisConn";

/**
 * Redis-based event bus implementation
 */
export class RedisEventBus implements IEventBus {
    private publisher: Redis | null = null;
    private subscriber: Redis | null = null;
    private subscriptions = new Map<string, EventSubscription>();
    private handlers = new Map<string, (event: BaseEvent) => Promise<void>>();
    private validator = new EventValidator();
    private sanitizer = new EventSanitizer();
    private running = false;
    
    private readonly STREAM_KEY = "execution:events";
    private readonly CONSUMER_GROUP = "execution-consumers";
    private readonly CONSUMER_NAME = `consumer-${process.pid}`;
    
    constructor() {}
    
    async start(): Promise<void> {
        if (this.running) {
            return;
        }
        
        try {
            // Get Redis connections
            this.publisher = await getRedisConnection();
            this.subscriber = await getRedisConnection();
            
            // Create consumer group
            await this.createConsumerGroup();
            
            // Start consuming events
            this.startConsuming();
            
            this.running = true;
            logger.info("RedisEventBus started");
        } catch (error) {
            logger.error("Failed to start RedisEventBus", error);
            throw error;
        }
    }
    
    async stop(): Promise<void> {
        if (!this.running) {
            return;
        }
        
        this.running = false;
        
        // Close Redis connections
        if (this.publisher) {
            await this.publisher.quit();
            this.publisher = null;
        }
        
        if (this.subscriber) {
            await this.subscriber.quit();
            this.subscriber = null;
        }
        
        // Clear local state
        this.subscriptions.clear();
        this.handlers.clear();
        
        logger.info("RedisEventBus stopped");
    }
    
    async publish(event: BaseEvent): Promise<void> {
        if (!this.publisher) {
            throw new Error("EventBus not started");
        }
        
        // Validate event
        const validation = this.validator.validate(event);
        if (!validation.valid) {
            throw new Error(
                `Invalid event: ${validation.errors.map(e => e.message).join(", ")}`
            );
        }
        
        // Sanitize event
        const sanitized = this.sanitizer.sanitize(event);
        
        // Publish to Redis stream
        const eventData = JSON.stringify(sanitized);
        await this.publisher.xadd(
            this.STREAM_KEY,
            "*",
            "event",
            eventData
        );
        
        logger.debug("Event published", { 
            eventId: sanitized.id,
            eventType: sanitized.type,
            correlationId: sanitized.correlationId,
        });
    }
    
    async publishBatch(events: BaseEvent[]): Promise<void> {
        if (!this.publisher) {
            throw new Error("EventBus not started");
        }
        
        // Validate all events
        const validationResults = events.map(event => ({
            event,
            validation: this.validator.validate(event),
        }));
        
        const invalid = validationResults.filter(r => !r.validation.valid);
        if (invalid.length > 0) {
            throw new Error(
                `Invalid events: ${invalid.map(r => 
                    r.validation.errors.map(e => e.message).join(", ")
                ).join("; ")}`
            );
        }
        
        // Sanitize and publish all events
        const pipeline = this.publisher.pipeline();
        
        for (const event of events) {
            const sanitized = this.sanitizer.sanitize(event);
            const eventData = JSON.stringify(sanitized);
            pipeline.xadd(
                this.STREAM_KEY,
                "*",
                "event",
                eventData
            );
        }
        
        await pipeline.exec();
        
        logger.debug(`Published batch of ${events.length} events`);
    }
    
    async subscribe(subscription: EventSubscription): Promise<void> {
        // Store subscription
        this.subscriptions.set(subscription.id, subscription);
        
        // Create handler function
        const handler = async (event: BaseEvent): Promise<void> => {
            try {
                // Check if event matches subscription filters
                if (!EventFilterUtils.matchesAll(event, subscription.filters)) {
                    return;
                }
                
                // Execute the handler
                // In a real implementation, this would call the actual handler
                logger.debug("Event matched subscription", {
                    subscriptionId: subscription.id,
                    eventId: event.id,
                    eventType: event.type,
                });
                
                // Here you would typically:
                // 1. Look up the handler function by name
                // 2. Execute it with appropriate error handling
                // 3. Handle retries based on subscription.config
            } catch (error) {
                logger.error("Error handling event", {
                    subscriptionId: subscription.id,
                    eventId: event.id,
                    error,
                });
            }
        };
        
        this.handlers.set(subscription.id, handler);
        
        logger.info("Subscription created", { subscriptionId: subscription.id });
    }
    
    async unsubscribe(subscriptionId: string): Promise<void> {
        this.subscriptions.delete(subscriptionId);
        this.handlers.delete(subscriptionId);
        
        logger.info("Subscription removed", { subscriptionId });
    }
    
    async getEvents(query: EventQuery): Promise<BaseEvent[]> {
        if (!this.publisher) {
            throw new Error("EventBus not started");
        }
        
        const events: BaseEvent[] = [];
        
        // Calculate time range for query
        const endTime = query.timeRange?.end || "+";
        const startTime = query.timeRange?.start || "-";
        
        // Read from stream
        const entries = await this.publisher.xrange(
            this.STREAM_KEY,
            startTime === "-" ? "-" : startTime.getTime().toString(),
            endTime === "+" ? "+" : endTime.getTime().toString(),
            "COUNT",
            query.limit || 100
        );
        
        // Parse events and apply filters
        for (const [, fields] of entries) {
            try {
                const eventData = fields.find((_, i) => i % 2 === 1); // Get value, not key
                if (eventData) {
                    const event = JSON.parse(eventData) as BaseEvent;
                    
                    // Apply query filters
                    if (!query.filters || EventFilterUtils.matchesAll(event, query.filters)) {
                        events.push(event);
                    }
                }
            } catch (error) {
                logger.error("Error parsing event from stream", error);
            }
        }
        
        // Apply ordering
        if (query.orderBy) {
            events.sort((a, b) => {
                const aValue = (a as any)[query.orderBy!];
                const bValue = (b as any)[query.orderBy!];
                const direction = query.orderDirection === "desc" ? -1 : 1;
                return aValue > bValue ? direction : -direction;
            });
        }
        
        // Apply offset
        if (query.offset) {
            return events.slice(query.offset);
        }
        
        return events;
    }
    
    async *getEventStream(query: EventQuery): AsyncIterableIterator<BaseEvent> {
        if (!this.subscriber) {
            throw new Error("EventBus not started");
        }
        
        // Start from latest if no start time specified
        let lastId = "$";
        
        while (this.running) {
            try {
                // Read new events
                const entries = await this.subscriber.xread(
                    "BLOCK", 1000, // Block for 1 second
                    "STREAMS",
                    this.STREAM_KEY,
                    lastId
                );
                
                if (!entries || entries.length === 0) {
                    continue;
                }
                
                const [, messages] = entries[0];
                
                for (const [id, fields] of messages) {
                    try {
                        const eventData = fields.find((_, i) => i % 2 === 1);
                        if (eventData) {
                            const event = JSON.parse(eventData) as BaseEvent;
                            
                            // Apply query filters
                            if (!query.filters || EventFilterUtils.matchesAll(event, query.filters)) {
                                yield event;
                            }
                        }
                        
                        lastId = id;
                    } catch (error) {
                        logger.error("Error parsing event from stream", error);
                    }
                }
            } catch (error) {
                if ((error as any).message?.includes("BLOCK")) {
                    // Timeout is normal, continue
                    continue;
                }
                logger.error("Error reading event stream", error);
                throw error;
            }
        }
    }
    
    private async createConsumerGroup(): Promise<void> {
        if (!this.publisher) {
            return;
        }
        
        try {
            await this.publisher.xgroup(
                "CREATE",
                this.STREAM_KEY,
                this.CONSUMER_GROUP,
                "$",
                "MKSTREAM"
            );
            logger.info("Consumer group created", { group: this.CONSUMER_GROUP });
        } catch (error: any) {
            if (!error.message?.includes("BUSYGROUP")) {
                throw error;
            }
            // Group already exists, which is fine
        }
    }
    
    private async startConsuming(): Promise<void> {
        if (!this.subscriber) {
            return;
        }
        
        // Start background consumer
        this.consumeEvents().catch(error => {
            logger.error("Error in event consumer", error);
        });
    }
    
    private async consumeEvents(): Promise<void> {
        while (this.running && this.subscriber) {
            try {
                // Read events from consumer group
                const entries = await this.subscriber.xreadgroup(
                    "GROUP",
                    this.CONSUMER_GROUP,
                    this.CONSUMER_NAME,
                    "BLOCK", 1000,
                    "COUNT", 10,
                    "STREAMS",
                    this.STREAM_KEY,
                    ">"
                );
                
                if (!entries || entries.length === 0) {
                    continue;
                }
                
                const [, messages] = entries[0];
                
                for (const [id, fields] of messages) {
                    try {
                        const eventData = fields.find((_, i) => i % 2 === 1);
                        if (eventData) {
                            const event = JSON.parse(eventData) as BaseEvent;
                            
                            // Process event through all matching handlers
                            await this.processEvent(event);
                            
                            // Acknowledge event
                            await this.subscriber.xack(
                                this.STREAM_KEY,
                                this.CONSUMER_GROUP,
                                id
                            );
                        }
                    } catch (error) {
                        logger.error("Error processing event", { id, error });
                    }
                }
            } catch (error: any) {
                if (!error.message?.includes("BLOCK")) {
                    logger.error("Error in consume loop", error);
                }
            }
        }
    }
    
    private async processEvent(event: BaseEvent): Promise<void> {
        const promises: Promise<void>[] = [];
        
        for (const [subscriptionId, handler] of this.handlers) {
            promises.push(
                handler(event).catch(error => {
                    logger.error("Handler error", {
                        subscriptionId,
                        eventId: event.id,
                        error,
                    });
                })
            );
        }
        
        await Promise.all(promises);
    }
}

// Singleton instance
let eventBusInstance: RedisEventBus | null = null;

/**
 * Get the event bus instance
 */
export function getEventBus(): RedisEventBus {
    if (!eventBusInstance) {
        eventBusInstance = new RedisEventBus();
    }
    return eventBusInstance;
}