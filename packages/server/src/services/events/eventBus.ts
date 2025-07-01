/**
 * Enhanced Event Bus Implementation
 * 
 * Provides unified event publishing and subscription for the execution architecture
 * with support for delivery guarantees and barrier synchronization.
 * 
 * This replaces the fragmented event system with a single, consistent interface
 * that enables true emergent capabilities through agent-extensible event types.
 */

import { MINUTES_1_MS } from "@vrooli/shared";
import { EventEmitter } from "events";
import mqttMatch from "mqtt-match";
import { nanoid } from "nanoid";
import { logger } from "../../events/logger.js";
import { SocketService } from "../../sockets/io.js";
import {
    EVENT_BUS_CONSTANTS,
} from "./constants.js";
import {
    EventBusRateLimiter,
    type EventPublishResultWithRateLimit,
} from "./rateLimiter.js";
import {
    type BaseEvent,
    type EventBarrierSyncResult,
    type EventHandler,
    type EventPublishResult,
    type EventSubscription,
    type EventSubscriptionId,
    type IEventBus,
    type SafetyEvent,
    type SubscriptionOptions,
} from "./types.js";

/**
 * Pending barrier sync operation
 */
interface PendingBarrier {
    eventId: string;
    event: SafetyEvent;
    responses: Array<{ responderId: string; response: "OK" | "ALARM"; reason?: string }>;
    resolve: (result: EventBarrierSyncResult) => void;
    reject: (error: Error) => void;
    timeoutId: NodeJS.Timeout;
    startTime: number;
    isCompleted: boolean; // Track completion state to prevent double cleanup
}

/**
 * Enhanced event bus implementation
 */
export class EventBus implements IEventBus {
    private readonly emitter: EventEmitter;
    private readonly subscriptions = new Map<EventSubscriptionId, EventSubscription>();
    private readonly pendingBarriers = new Map<string, PendingBarrier>();
    private readonly rateLimiter: EventBusRateLimiter;
    private isStarted = false;

    // Metrics
    private metrics = {
        eventsPublished: 0,
        eventsDelivered: 0,
        eventsFailed: 0,
        eventsRateLimited: 0,
        barrierSyncsCompleted: 0,
        barrierSyncsTimedOut: 0,
        activeSubscriptions: 0,
        lastEventTime: 0,
    };

    constructor(rateLimiter?: EventBusRateLimiter) {
        this.emitter = new EventEmitter();
        this.emitter.setMaxListeners(EVENT_BUS_CONSTANTS.MAX_EVENT_LISTENERS); // Allow many subscribers

        // Initialize rate limiter
        this.rateLimiter = rateLimiter || new EventBusRateLimiter(logger);

        // Set up backpressure monitoring
        this.setupBackpressureMonitoring();
    }

    /**
     * Start the event bus
     */
    async start(): Promise<void> {
        if (this.isStarted) {
            logger.warn("[EventBus] Already started");
            return;
        }

        this.isStarted = true;
        logger.info("[EventBus] Started enhanced event bus");
    }

    /**
     * Stop the event bus
     */
    async stop(): Promise<void> {
        if (!this.isStarted) {
            return;
        }

        this.isStarted = false;

        // Clear pending barriers with proper cleanup
        for (const barrier of this.pendingBarriers.values()) {
            if (!barrier.isCompleted) {
                clearTimeout(barrier.timeoutId);
                barrier.isCompleted = true;
                barrier.reject(new Error("Event bus stopped"));
            }
        }
        this.pendingBarriers.clear();

        // Clear subscriptions
        this.emitter.removeAllListeners();
        this.subscriptions.clear();

        logger.info("[EventBus] Stopped event bus");
    }

    /**
     * Publish an event with specified delivery guarantee and rate limiting
     */
    async publish<T extends BaseEvent>(event: T): Promise<EventPublishResult> {
        const startTime = performance.now();

        if (!this.isStarted) {
            return {
                success: false,
                error: new Error("Event bus not started"),
                duration: performance.now() - startTime,
            };
        }

        try {
            // 🔥 RATE LIMITING - Check before processing event
            const rateLimitResult = await this.rateLimiter.checkEventRateLimit(event);

            if (!rateLimitResult.allowed) {
                // Rate limit exceeded - emit rate limited event and reject original
                this.metrics.eventsRateLimited++;

                const rateLimitedEvent = this.rateLimiter.createRateLimitedEvent(event, rateLimitResult);

                // Emit the rate limited event (bypasses rate limiting)
                await this.publishBypassRateLimit(rateLimitedEvent);

                logger.warn("[EventBus] Event rate limited", {
                    eventId: event.id,
                    eventType: event.type,
                    limitType: rateLimitResult.limitType,
                    retryAfterMs: rateLimitResult.retryAfterMs,
                });

                return {
                    success: false,
                    error: new Error(`Rate limit exceeded: ${rateLimitResult.limitType}`),
                    duration: performance.now() - startTime,
                } as EventPublishResultWithRateLimit;
            }

            // Update metrics
            this.metrics.eventsPublished++;
            this.metrics.lastEventTime = Date.now();

            // Handle different delivery guarantees
            const deliveryGuarantee = event.metadata?.deliveryGuarantee || "fire-and-forget";

            switch (deliveryGuarantee) {
                case "fire-and-forget":
                    await this.publishFireAndForget(event);
                    break;
                case "reliable":
                    await this.publishReliable(event);
                    break;
                case "barrier-sync":
                    // Barrier sync events should use publishBarrierSync method
                    throw new Error("Barrier sync events must use publishBarrierSync method");
                default:
                    throw new Error(`Unknown delivery guarantee: ${deliveryGuarantee}`);
            }

            this.metrics.eventsDelivered++;

            logger.debug("[EventBus] Published event", {
                eventId: event.id,
                eventType: event.type,
                deliveryGuarantee,
                rateLimitStatus: "allowed",
            });

            return {
                success: true,
                duration: performance.now() - startTime,
                remainingQuota: rateLimitResult.remainingQuota,
                resetTime: rateLimitResult.resetTime,
            } as EventPublishResultWithRateLimit;

        } catch (error) {
            this.metrics.eventsFailed++;

            logger.error("[EventBus] Failed to publish event", {
                eventId: event.id,
                eventType: event.type,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: error instanceof Error ? error : new Error(String(error)),
                duration: performance.now() - startTime,
            };
        }
    }

    /**
     * Publish an event bypassing rate limits (for internal system events)
     */
    private async publishBypassRateLimit<T extends BaseEvent>(event: T): Promise<void> {
        // Handle delivery without rate limiting
        const deliveryGuarantee = event.metadata?.deliveryGuarantee || "fire-and-forget";

        switch (deliveryGuarantee) {
            case "fire-and-forget":
                await this.publishFireAndForget(event);
                break;
            case "reliable":
                await this.publishReliable(event);
                break;
            default:
                await this.publishFireAndForget(event);
        }

        logger.debug("[EventBus] Published system event bypassing rate limits", {
            eventId: event.id,
            eventType: event.type,
        });
    }

    /**
     * Handle barrier sync events (blocking until responses received)
     */
    async publishBarrierSync<T extends SafetyEvent>(event: T): Promise<EventBarrierSyncResult> {
        const startTime = performance.now();

        if (!this.isStarted) {
            throw new Error("Event bus not started");
        }

        const barrierConfig = event.metadata.barrierConfig;
        if (!barrierConfig) {
            throw new Error("Barrier sync event must have barrierConfig");
        }

        return new Promise<EventBarrierSyncResult>((resolve, reject) => {
            const pendingBarrier: PendingBarrier = {
                eventId: event.id,
                event,
                responses: [],
                resolve,
                reject,
                timeoutId: setTimeout(() => {
                    this.handleBarrierTimeout(event.id, startTime);
                }, barrierConfig.timeoutMs),
                startTime,
                isCompleted: false,
            };

            this.pendingBarriers.set(event.id, pendingBarrier);

            // Emit the event for safety agents to respond
            this.emitToSubscribers(event);

            logger.debug("[EventBus] Published barrier sync event", {
                eventId: event.id,
                eventType: event.type,
                quorum: barrierConfig.quorum,
                timeoutMs: barrierConfig.timeoutMs,
            });
        });
    }

    /**
     * Subscribe to event patterns with optional filtering
     */
    async subscribe<T extends BaseEvent>(
        pattern: string | string[],
        handler: EventHandler<T>,
        options: SubscriptionOptions = {},
    ): Promise<EventSubscriptionId> {
        const subscriptionId = nanoid();
        const patterns = Array.isArray(pattern) ? pattern : [pattern];

        const subscription: EventSubscription = {
            id: subscriptionId,
            patterns,
            handler: handler as EventHandler,
            options,
            createdAt: new Date(),
        };

        this.subscriptions.set(subscriptionId, subscription);
        this.metrics.activeSubscriptions = this.subscriptions.size;

        logger.info("[EventBus] Added subscription", {
            subscriptionId,
            patterns,
        });

        return subscriptionId;
    }

    /**
     * Unsubscribe from events
     */
    async unsubscribe(subscriptionId: EventSubscriptionId): Promise<void> {
        const removed = this.subscriptions.delete(subscriptionId);
        this.metrics.activeSubscriptions = this.subscriptions.size;

        if (removed) {
            logger.info("[EventBus] Removed subscription", {
                subscriptionId,
            });
        }
    }

    /**
     * Respond to a barrier sync event (for safety agents)
     */
    async respondToBarrierSync(
        eventId: string,
        responderId: string,
        response: "OK" | "ALARM",
        reason?: string,
    ): Promise<void> {
        const barrier = this.pendingBarriers.get(eventId);
        if (!barrier) {
            logger.warn("[EventBus] Response to unknown barrier sync event", {
                eventId,
                responderId,
                response,
            });
            return;
        }

        // Add response
        barrier.responses.push({ responderId, response, reason });

        logger.debug("[EventBus] Received barrier sync response", {
            eventId,
            responderId,
            response,
            reason,
            responseCount: barrier.responses.length,
            requiredQuorum: barrier.event.metadata.barrierConfig.quorum,
        });

        // Check if we have enough responses
        const config = barrier.event.metadata.barrierConfig;
        const okResponses = barrier.responses.filter(r => r.response === "OK").length;
        const alarmResponses = barrier.responses.filter(r => r.response === "ALARM").length;

        // If we have any ALARM responses, reject immediately
        if (alarmResponses > 0) {
            this.completeBarrierSync(eventId, false, "ALARM response received");
            return;
        }

        // If we have enough OK responses, approve
        if (okResponses >= config.quorum) {
            this.completeBarrierSync(eventId, true, "Quorum reached");
            return;
        }

        // If we've heard from all required responders and don't have quorum, reject
        if (config.requiredResponders &&
            barrier.responses.length >= config.requiredResponders.length) {
            this.completeBarrierSync(eventId, false, "All responders replied without reaching quorum");
        }
    }

    /**
     * Get event bus metrics including rate limiting statistics
     */
    getMetrics(): typeof this.metrics & {
        pendingBarriers: number;
        rateLimitingEnabled: boolean;
    } {
        return {
            ...this.metrics,
            pendingBarriers: this.pendingBarriers.size,
            rateLimitingEnabled: true,
        };
    }

    /**
     * Get detailed rate limiting status for monitoring
     */
    async getRateLimitStatus(userId?: string, eventType?: string): Promise<{
        global: { allowed: number; total: number };
        user?: { allowed: number; total: number };
        eventType?: { allowed: number; total: number };
    }> {
        return this.rateLimiter.getRateLimitStatus(userId, eventType);
    }

    /**
     * Private methods
     */

    private async publishFireAndForget<T extends BaseEvent>(event: T): Promise<void> {
        // Fire and forget - emit and don't wait for delivery confirmation
        this.emitToSubscribers(event);

        // Also emit to socket clients if socket service is configured
        this.emitToSocketClients(event);
    }

    private async publishReliable<T extends BaseEvent>(event: T): Promise<void> {
        // Reliable delivery - emit and ensure delivery to all subscribers
        await this.emitToSubscribersReliably(event);

        // Also emit to socket clients if socket service is configured
        this.emitToSocketClients(event);
    }

    private emitToSubscribers<T extends BaseEvent>(event: T): void {
        let deliveredCount = 0;

        for (const subscription of this.subscriptions.values()) {
            if (this.matchesSubscription(event, subscription)) {
                // Don't await - fire and forget
                this.callSubscriptionHandler(subscription, event)
                    .then(() => deliveredCount++)
                    .catch(error => {
                        logger.error("[EventBus] Subscription handler error", {
                            subscriptionId: subscription.id,
                            eventId: event.id,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    });
            }
        }
    }

    private async emitToSubscribersReliably<T extends BaseEvent>(event: T): Promise<void> {
        const promises: Promise<void>[] = [];

        for (const subscription of this.subscriptions.values()) {
            if (this.matchesSubscription(event, subscription)) {
                promises.push(
                    this.callSubscriptionHandler(subscription, event).catch(error => {
                        logger.error("[EventBus] Subscription handler error", {
                            subscriptionId: subscription.id,
                            eventId: event.id,
                            error: error instanceof Error ? error.message : String(error),
                        });
                        // Re-throw to indicate delivery failure
                        throw error;
                    }),
                );
            }
        }

        // Wait for all deliveries to complete
        await Promise.all(promises);
    }

    private async callSubscriptionHandler(
        subscription: EventSubscription,
        event: BaseEvent,
    ): Promise<void> {
        // Apply filter if provided
        if (subscription.options.filter && !subscription.options.filter(event)) {
            return;
        }

        // Handle retries
        const maxRetries = subscription.options.maxRetries || 0;
        let lastError: Error | undefined;

        for (let attempt = 0; attempt <= maxRetries; attempt++) {
            try {
                await subscription.handler(event);
                return; // Success
            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));

                if (attempt < maxRetries) {
                    // Wait before retry
                    await new Promise(resolve => setTimeout(resolve, EVENT_BUS_CONSTANTS.RETRY_BASE_DELAY_MS * Math.pow(EVENT_BUS_CONSTANTS.RETRY_EXPONENTIAL_FACTOR, attempt)));
                }
            }
        }

        // All retries failed
        if (lastError) {
            throw lastError;
        }
    }

    private matchesSubscription(event: BaseEvent, subscription: EventSubscription): boolean {
        return subscription.patterns.some(pattern => this.matchesPattern(event.type, pattern));
    }

    private matchesPattern(eventType: string, pattern: string): boolean {
        // Use mqtt-match library for proper MQTT pattern matching
        // This handles +, #, and exact matching according to MQTT standards
        return mqttMatch(pattern, eventType);
    }

    private handleBarrierTimeout(eventId: string, startTime: number): void {
        const barrier = this.pendingBarriers.get(eventId);
        if (!barrier) return;

        const timeoutAction = barrier.event.metadata.barrierConfig.timeoutAction;

        switch (timeoutAction) {
            case "auto-approve":
                this.completeBarrierSync(eventId, true, "Auto-approved on timeout");
                break;
            case "auto-reject":
                this.completeBarrierSync(eventId, false, "Auto-rejected on timeout");
                break;
            case "keep-pending":
                // Don't complete - leave pending for manual resolution
                logger.warn("[EventBus] Barrier sync timed out but keeping pending", {
                    eventId,
                    duration: performance.now() - startTime,
                });
                break;
        }
    }

    private completeBarrierSync(eventId: string, success: boolean, reason: string): void {
        const barrier = this.pendingBarriers.get(eventId);
        if (!barrier || barrier.isCompleted) return;

        // Mark as completed first to prevent double cleanup
        barrier.isCompleted = true;
        clearTimeout(barrier.timeoutId);
        this.pendingBarriers.delete(eventId);

        const result: EventBarrierSyncResult = {
            success,
            responses: barrier.responses,
            timedOut: false,
            duration: performance.now() - barrier.startTime,
        };

        if (success) {
            this.metrics.barrierSyncsCompleted++;
        } else {
            this.metrics.barrierSyncsTimedOut++;
        }

        logger.debug("[EventBus] Completed barrier sync", {
            eventId,
            success,
            reason,
            responseCount: barrier.responses.length,
            duration: result.duration,
        });

        barrier.resolve(result);
    }

    /**
     * Emit event to socket clients if socket service is configured
     */
    private emitToSocketClients<T extends BaseEvent>(event: T): void {
        try {
            // Extract the appropriate room ID from the event based on its type
            const roomId = this.extractRoomId(event);
            if (!roomId) {
                logger.debug("[EventBus] No room ID found for socket emission", {
                    eventType: event.type,
                    eventId: event.id,
                });
                return;
            }

            // Create the socket event payload directly from the event
            // This maintains perfect type alignment between event bus and socket events
            const socketEvent = {
                id: event.id,
                type: event.type,
                timestamp: event.timestamp,
                data: event.data,
            };

            // Forward the event directly to Socket.IO with no transformation
            SocketService.get().emitSocketEvent(event.type, roomId, socketEvent);

            logger.debug("[EventBus] Emitted event to socket clients", {
                eventType: event.type,
                roomId,
                eventId: event.id,
            });
        } catch (error) {
            logger.error("[EventBus] Failed to emit event to socket clients", {
                eventType: event.type,
                eventId: event.id,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Extract the appropriate room ID from an event based on its type and data
     */
    private extractRoomId(event: BaseEvent): string | null {
        // For chat, swarm, bot, tool, response, reasoning, and cancellation events - use chat room
        if (event.type.startsWith("chat/") ||
            event.type.startsWith("swarm/") ||
            event.type.startsWith("bot/") ||
            event.type.startsWith("tool/") ||
            event.type.startsWith("response/") ||
            event.type.startsWith("reasoning/") ||
            event.type.startsWith("cancellation/")) {
            return this.extractChatId(event);
        }

        // For run and step events - use run room
        if (event.type.startsWith("run/") || event.type.startsWith("step/")) {
            return (event.data as any)?.runId || null;
        }

        // For user events - use user room
        if (event.type.startsWith("user/")) {
            return (event.data as any)?.userId || null;
        }

        // For room events - extract room ID from the event data
        if (event.type.startsWith("room/")) {
            return (event.data as any)?.roomId || null;
        }

        // Unknown event type pattern
        logger.debug("[EventBus] Unknown event type pattern for socket emission", {
            eventType: event.type,
            eventId: event.id,
        });
        return null;
    }

    /**
     * Extract chat ID from event data
     */
    private extractChatId(event: BaseEvent): string | null {
        // Try multiple possible locations for chat ID
        const data = event.data as any;
        return data?.chatId ||
            data?.conversationId ||
            null;
    }

    /**
     * Set up backpressure monitoring to prevent system overload
     */
    private setupBackpressureMonitoring(): void {
        // Monitor subscription count and warn if too many
        const monitoringInterval = setInterval(() => {
            if (!this.isStarted) {
                clearInterval(monitoringInterval);
                return;
            }

            const subscriptionCount = this.subscriptions.size;
            const pendingBarriers = this.pendingBarriers.size;

            if (subscriptionCount > EVENT_BUS_CONSTANTS.MAX_EVENT_LISTENERS * 0.8) {
                logger.warn("[EventBus] High subscription count detected", {
                    subscriptionCount,
                    maxListeners: EVENT_BUS_CONSTANTS.MAX_EVENT_LISTENERS,
                });
            }

            if (pendingBarriers > 50) {
                logger.warn("[EventBus] High pending barrier count detected", {
                    pendingBarriers,
                });
            }
        }, MINUTES_1_MS);
    }
}

/**
 * Singleton instance of the event bus
 */
let eventBus: EventBus | null = null;

/**
 * Get the singleton instance of the event bus
 */
export function getEventBus(): EventBus {
    if (!eventBus) {
        eventBus = new EventBus();
    }
    return eventBus;
}
