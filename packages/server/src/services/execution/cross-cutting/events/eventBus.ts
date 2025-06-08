/**
 * Intelligent Event Bus Implementation
 * Provides event-driven communication with delivery guarantees, agent intelligence, and safety barriers
 */

import type { Redis } from "ioredis";
import type { 
    IEventBus, 
    BaseEvent, 
    EventSubscription, 
    EventQuery,
} from "@vrooli/shared";
import {
    EventFilterUtils,
    EventValidator,
    EventSanitizer,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { getRedisConnection } from "../../../../redisConn.js";
import { EventAgent } from "../agents/eventAgent.js";
import { BarrierSynchronizer } from "../safety/barrierSynchronizer.js";
import { ToolApprovalManager } from "../approval/toolApprovalManager.js";

/**
 * Enhanced event types with delivery guarantees
 */
export enum DeliveryGuarantee {
    FIRE_AND_FORGET = "fire-and-forget",
    RELIABLE = "reliable",
    BARRIER_SYNC = "barrier_sync",
}

export enum EventPriority {
    LOW = "low",
    MEDIUM = "medium", 
    HIGH = "high",
    CRITICAL = "critical",
    EMERGENCY = "emergency",
}

/**
 * Enhanced event interface with rich metadata
 */
export interface IntelligentEvent extends BaseEvent {
    // Delivery configuration
    deliveryGuarantee: DeliveryGuarantee;
    priority: EventPriority;
    
    // Barrier sync configuration
    barrierTimeout?: number;
    barrierQuorum?: number;
    
    // Routing and filtering
    tier: 1 | 2 | 3;
    category: string;
    subcategory?: string;
    
    // Rich context
    correlationChain?: string[];
    causedBy?: string;
    relatedEvents?: string[];
    
    // Agent processing metadata
    agentContext?: {
        requiredCapabilities?: string[];
        targetAgents?: string[];
        excludedAgents?: string[];
        confidenceThreshold?: number;
    };
    
    // Safety and compliance
    securityLevel?: "public" | "internal" | "confidential" | "secret";
    complianceRequired?: boolean;
    humanApprovalRequired?: boolean;
}

/**
 * Event agent subscription with intelligence
 */
export interface AgentSubscription {
    agentId: string;
    eventPatterns: string[];
    capabilities: string[];
    priority: number;
    handler: (event: IntelligentEvent) => Promise<AgentResponse>;
    filterPredicate?: (event: IntelligentEvent) => boolean;
    learningEnabled?: boolean;
}

/**
 * Agent response to events
 */
export interface AgentResponse {
    status: "OK" | "ALARM" | "DEFER" | "ESCALATE";
    confidence: number;
    reasoning?: string;
    suggestedActions?: string[];
    metadata?: Record<string, unknown>;
    newEvents?: IntelligentEvent[];
}

/**
 * Intelligent Event Bus Implementation
 * Supports delivery guarantees, agent intelligence, and safety barriers
 */
export class IntelligentEventBus implements IEventBus {
    private publisher: Redis | null = null;
    private subscriber: Redis | null = null;
    private subscriptions = new Map<string, EventSubscription>();
    private agentSubscriptions = new Map<string, AgentSubscription>();
    private handlers = new Map<string, (event: BaseEvent) => Promise<void>>();
    private eventAgents = new Map<string, EventAgent>();
    private validator = new EventValidator();
    private sanitizer = new EventSanitizer();
    private barrierSynchronizer: BarrierSynchronizer;
    private toolApprovalManager: ToolApprovalManager;
    private running = false;
    
    // Enhanced streams for different delivery guarantees
    private readonly RELIABLE_STREAM = "execution:events:reliable";
    private readonly BARRIER_STREAM = "execution:events:barrier";
    private readonly AGENT_STREAM = "execution:events:agents";
    
    private readonly STREAM_KEY = "execution:events";
    private readonly CONSUMER_GROUP = "execution-consumers";
    private readonly CONSUMER_NAME = `consumer-${process.pid}`;
    
    // Timing constants
    private readonly BLOCK_TIMEOUT_MS = 1000; // 1 second block timeout
    private readonly CONSUMER_BATCH_SIZE = 10; // Events per batch
    
    constructor() {
        this.barrierSynchronizer = new BarrierSynchronizer(this);
        this.toolApprovalManager = new ToolApprovalManager(this);
    }
    
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
    
    async publish(event: BaseEvent | IntelligentEvent): Promise<void> {
        if (!this.publisher) {
            throw new Error("EventBus not started");
        }
        
        // Validate event
        const validation = this.validator.validate(event);
        if (!validation.valid) {
            throw new Error(
                `Invalid event: ${validation.errors.map(e => e.message).join(", ")}`,
            );
        }
        
        // Sanitize event
        const sanitized = this.sanitizer.sanitize(event);
        
        // Determine target stream based on delivery guarantee
        const intelligentEvent = sanitized as IntelligentEvent;
        const streamKey = this.getStreamKey(intelligentEvent.deliveryGuarantee);
        
        // Handle barrier synchronization events
        if (intelligentEvent.deliveryGuarantee === DeliveryGuarantee.BARRIER_SYNC) {
            await this.handleBarrierSyncEvent(intelligentEvent);
            return;
        }
        
        // Publish to appropriate stream
        const eventData = JSON.stringify(sanitized);
        await this.publisher.xadd(
            streamKey,
            "*",
            "event",
            eventData,
            "priority",
            intelligentEvent.priority || EventPriority.MEDIUM,
            "tier",
            intelligentEvent.tier?.toString() || "0",
        );
        
        // Notify agents if this is an agent-relevant event
        await this.notifyRelevantAgents(intelligentEvent);
        
        logger.debug("Event published", { 
            eventId: sanitized.id,
            eventType: sanitized.type,
            correlationId: sanitized.correlationId,
        });
    }
    
    async publishBatch(events: (BaseEvent | IntelligentEvent)[]): Promise<void> {
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
                    r.validation.errors.map(e => e.message).join(", "),
                ).join("; ")}`,
            );
        }
        
        // Group events by delivery guarantee
        const groupedEvents = new Map<DeliveryGuarantee, IntelligentEvent[]>();
        const barrierEvents: IntelligentEvent[] = [];
        
        for (const event of events) {
            const sanitized = this.sanitizer.sanitize(event) as IntelligentEvent;
            const guarantee = sanitized.deliveryGuarantee || DeliveryGuarantee.FIRE_AND_FORGET;
            
            if (guarantee === DeliveryGuarantee.BARRIER_SYNC) {
                barrierEvents.push(sanitized);
            } else {
                if (!groupedEvents.has(guarantee)) {
                    groupedEvents.set(guarantee, []);
                }
                groupedEvents.get(guarantee)!.push(sanitized);
            }
        }
        
        // Handle barrier events synchronously
        for (const event of barrierEvents) {
            await this.handleBarrierSyncEvent(event);
        }
        
        // Publish other events in batches
        const pipeline = this.publisher.pipeline();
        
        for (const [guarantee, eventGroup] of groupedEvents) {
            const streamKey = this.getStreamKey(guarantee);
            
            for (const event of eventGroup) {
                const eventData = JSON.stringify(event);
                pipeline.xadd(
                    streamKey,
                    "*",
                    "event",
                    eventData,
                    "priority",
                    event.priority || EventPriority.MEDIUM,
                    "tier",
                    event.tier?.toString() || "0",
                );
            }
        }
        
        await pipeline.exec();
        
        // Notify agents for all events
        for (const eventGroup of groupedEvents.values()) {
            for (const event of eventGroup) {
                await this.notifyRelevantAgents(event);
            }
        }
        
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
            query.limit || 100,
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
                    "BLOCK", this.BLOCK_TIMEOUT_MS.toString(),
                    "STREAMS",
                    this.STREAM_KEY,
                    lastId,
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
                "MKSTREAM",
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
                    "BLOCK", this.BLOCK_TIMEOUT_MS.toString(),
                    "COUNT", this.CONSUMER_BATCH_SIZE.toString(),
                    "STREAMS",
                    this.STREAM_KEY,
                    ">",
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
                                id,
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
    
    private async processEvent(event: BaseEvent | IntelligentEvent): Promise<void> {
        const intelligentEvent = event as IntelligentEvent;
        
        // Process through regular handlers
        const handlerPromises: Promise<void>[] = [];
        for (const [subscriptionId, handler] of this.handlers) {
            handlerPromises.push(
                handler(event).catch(error => {
                    logger.error("Handler error", {
                        subscriptionId,
                        eventId: event.id,
                        error,
                    });
                }),
            );
        }
        
        // Process through intelligent agents
        const agentPromises: Promise<void>[] = [];
        for (const [agentId, subscription] of this.agentSubscriptions) {
            if (this.eventMatchesAgentSubscription(intelligentEvent, subscription)) {
                agentPromises.push(
                    this.processEventThroughAgent(intelligentEvent, subscription).catch(error => {
                        logger.error("Agent processing error", {
                            agentId,
                            eventId: event.id,
                            error,
                        });
                    }),
                );
            }
        }
        
        await Promise.all([...handlerPromises, ...agentPromises]);
    }
    
    /**
     * Enhanced agent subscription
     */
    async subscribeAgent(subscription: AgentSubscription): Promise<void> {
        this.agentSubscriptions.set(subscription.agentId, subscription);
        
        // Register the agent
        if (!this.eventAgents.has(subscription.agentId)) {
            const agent = new EventAgent(subscription.agentId, subscription.capabilities);
            this.eventAgents.set(subscription.agentId, agent);
        }
        
        logger.info("Agent subscribed", {
            agentId: subscription.agentId,
            patterns: subscription.eventPatterns,
            capabilities: subscription.capabilities,
        });
    }
    
    async unsubscribeAgent(agentId: string): Promise<void> {
        this.agentSubscriptions.delete(agentId);
        this.eventAgents.delete(agentId);
        
        logger.info("Agent unsubscribed", { agentId });
    }
    
    /**
     * Publish tier-specific events
     */
    async publishTierEvent(tier: 1 | 2 | 3, eventType: string, payload: any, options?: {
        deliveryGuarantee?: DeliveryGuarantee;
        priority?: EventPriority;
        barrierTimeout?: number;
        humanApprovalRequired?: boolean;
    }): Promise<void> {
        const event: IntelligentEvent = {
            id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
            type: eventType,
            timestamp: new Date(),
            source: `tier${tier}`,
            tier,
            category: eventType.split("/")[0],
            subcategory: eventType.split("/")[1],
            data: payload,
            deliveryGuarantee: options?.deliveryGuarantee || DeliveryGuarantee.FIRE_AND_FORGET,
            priority: options?.priority || EventPriority.MEDIUM,
            barrierTimeout: options?.barrierTimeout,
            humanApprovalRequired: options?.humanApprovalRequired,
        };
        
        await this.publish(event);
    }
    
    /**
     * Helper methods for intelligent event processing
     */
    private getStreamKey(guarantee: DeliveryGuarantee): string {
        switch (guarantee) {
            case DeliveryGuarantee.RELIABLE:
                return this.RELIABLE_STREAM;
            case DeliveryGuarantee.BARRIER_SYNC:
                return this.BARRIER_STREAM;
            default:
                return this.STREAM_KEY; // Fire-and-forget
        }
    }
    
    private async handleBarrierSyncEvent(event: IntelligentEvent): Promise<void> {
        try {
            const result = await this.barrierSynchronizer.synchronize(event);
            
            if (result.status !== "OK") {
                logger.warn("Barrier synchronization failed", {
                    eventId: event.id,
                    status: result.status,
                    reason: result.reason,
                });
                
                // Publish barrier failure event
                await this.publishTierEvent(
                    event.tier,
                    "safety/barrier_failed",
                    {
                        originalEvent: event.id,
                        failureReason: result.reason,
                        agentResponses: result.agentResponses,
                    },
                    { deliveryGuarantee: DeliveryGuarantee.RELIABLE },
                );
            }
        } catch (error) {
            logger.error("Barrier synchronization error", {
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }
    
    private async notifyRelevantAgents(event: IntelligentEvent): Promise<void> {
        // Check if event requires tool approval
        if (event.humanApprovalRequired || event.type.includes("tool/approval_required")) {
            await this.toolApprovalManager.processApprovalRequest(event);
        }
        
        // Publish to agent stream for processing
        if (this.agentSubscriptions.size > 0) {
            const agentEventData = JSON.stringify(event);
            await this.publisher.xadd(
                this.AGENT_STREAM,
                "*",
                "event",
                agentEventData,
                "agentCount",
                this.agentSubscriptions.size.toString(),
            );
        }
    }
    
    private eventMatchesAgentSubscription(
        event: IntelligentEvent,
        subscription: AgentSubscription,
    ): boolean {
        // Check event patterns
        const matches = subscription.eventPatterns.some(pattern => {
            if (pattern.includes("*")) {
                const regex = new RegExp(pattern.replace(/\*/g, ".*"));
                return regex.test(event.type);
            }
            return event.type === pattern;
        });
        
        if (!matches) return false;
        
        // Apply custom filter predicate
        if (subscription.filterPredicate) {
            return subscription.filterPredicate(event);
        }
        
        return true;
    }
    
    private async processEventThroughAgent(
        event: IntelligentEvent,
        subscription: AgentSubscription,
    ): Promise<void> {
        try {
            const agent = this.eventAgents.get(subscription.agentId);
            if (!agent) {
                logger.warn("Agent not found for subscription", {
                    agentId: subscription.agentId,
                });
                return;
            }
            
            // Process event through agent
            const response = await subscription.handler(event);
            
            // Handle agent response
            await this.handleAgentResponse(event, subscription, response);
            
            // Update agent learning if enabled
            if (subscription.learningEnabled) {
                await agent.learn(event, response);
            }
            
        } catch (error) {
            logger.error("Agent processing failed", {
                agentId: subscription.agentId,
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }
    
    private async handleAgentResponse(
        originalEvent: IntelligentEvent,
        subscription: AgentSubscription,
        response: AgentResponse,
    ): Promise<void> {
        // Log agent response
        logger.debug("Agent response", {
            agentId: subscription.agentId,
            eventId: originalEvent.id,
            status: response.status,
            confidence: response.confidence,
        });
        
        // Handle escalations
        if (response.status === "ESCALATE") {
            await this.publishTierEvent(
                1, // Escalate to Tier 1 coordination
                "agent/escalation",
                {
                    originalEvent: originalEvent.id,
                    agentId: subscription.agentId,
                    reason: response.reasoning,
                    suggestedActions: response.suggestedActions,
                },
                { deliveryGuarantee: DeliveryGuarantee.RELIABLE },
            );
        }
        
        // Publish any new events generated by the agent
        if (response.newEvents && response.newEvents.length > 0) {
            await this.publishBatch(response.newEvents);
        }
        
        // Handle alarms
        if (response.status === "ALARM") {
            await this.publishTierEvent(
                originalEvent.tier,
                "safety/alarm_raised",
                {
                    originalEvent: originalEvent.id,
                    agentId: subscription.agentId,
                    reason: response.reasoning,
                    confidence: response.confidence,
                },
                { 
                    deliveryGuarantee: DeliveryGuarantee.RELIABLE,
                    priority: EventPriority.HIGH,
                },
            );
        }
    }
}

// Singleton instance
let eventBusInstance: IntelligentEventBus | null = null;

/**
 * Get the intelligent event bus instance
 */
export function getIntelligentEventBus(): IntelligentEventBus {
    if (!eventBusInstance) {
        eventBusInstance = new IntelligentEventBus();
    }
    return eventBusInstance;
}

// Backward compatibility
export function getEventBus(): IntelligentEventBus {
    return getIntelligentEventBus();
}
