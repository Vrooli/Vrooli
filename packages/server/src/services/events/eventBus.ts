/**
 * Enhanced Event Bus Implementation
 * 
 * Provides unified event publishing and subscription for the execution architecture
 * with support for delivery guarantees and barrier synchronization.
 * 
 * This replaces the fragmented event system with a single, consistent interface
 * that enables true emergent capabilities through agent-extensible event types.
 */

import { MINUTES_1_MS, nanoid } from "@vrooli/shared";
import { EventEmitter } from "events";
import { logger } from "../../events/logger.js";
import { SocketService } from "../../sockets/io.js";
import { EVENT_BUS_CONSTANTS } from "./constants.js";
import { EventBusMonitor } from "./EventBusMonitor.js";
import { EventBusRateLimiter, type EventPublishResultWithRateLimit } from "./rateLimiter.js";
import { getEventBehavior } from "./registry.js";
import type { BarrierSyncConfig, BotEventResponse, EventHandler, EventPublishResult, EventSubscription, EventSubscriptionId, IEventBus, ProgressionControl, ServiceEvent, SubscriptionOptions } from "./types.js";
import { EventMode } from "./types.js";

// Dynamic import type for mqtt-match
type MqttMatchFunction = (pattern: string, topic: string) => boolean;

/** High pending barrier count threshold for backpressure monitoring */
const HIGH_PENDING_BARRIER_THRESHOLD = 50;
/** Subscription warning threshold as percentage of max listeners */
const SUBSCRIPTION_WARNING_THRESHOLD = 0.8;

/**
 * Pending barrier sync operation
 */
interface PendingBarrier {
    eventId: string;
    event: ServiceEvent;
    responses: Array<{
        responderId: string;
        response: BotEventResponse;
        timestamp: Date;
    }>;
    resolve: (result: EventPublishResult) => void;
    reject: (error: Error) => void;
    timeoutId: NodeJS.Timeout;
    startTime: number;
    isCompleted: boolean; // Track completion state to prevent double cleanup
    barrierConfig: BarrierSyncConfig;
}

/**
 * Enhanced event bus implementation
 */
export class EventBus implements IEventBus {
    private readonly emitter: EventEmitter;
    private readonly subscriptions = new Map<EventSubscriptionId, EventSubscription>();
    private readonly pendingBarriers = new Map<string, PendingBarrier>();
    private readonly rateLimiter: EventBusRateLimiter;
    private readonly monitor: EventBusMonitor;
    private isStarted = false;
    private mqttMatch: MqttMatchFunction | null = null;

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
        this.rateLimiter = rateLimiter || new EventBusRateLimiter();

        // Initialize performance monitor
        this.monitor = new EventBusMonitor();

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
        this.monitor.start();
        logger.info("[EventBus] Started enhanced event bus with performance monitoring");
    }

    /**
     * Stop the event bus
     */
    async stop(): Promise<void> {
        if (!this.isStarted) {
            return;
        }

        this.isStarted = false;
        this.monitor.stop();

        // Clear pending barriers with proper cleanup
        for (const barrier of Array.from(this.pendingBarriers.values())) {
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
     * Publish an event with intelligent mode detection and barrier handling
     */
    async publish(event: Omit<ServiceEvent, "id" | "timestamp">): Promise<EventPublishResult> {
        const startTime = performance.now();

        if (!this.isStarted) {
            return {
                success: false,
                error: new Error("Event bus not started"),
                duration: performance.now() - startTime,
            };
        }

        try {
            // Create full event first
            const fullEvent = this.createFullEvent(event);

            // 🔥 RATE LIMITING - Check before processing event
            const rateLimitResult = await this.rateLimiter.checkEventRateLimit(fullEvent);

            if (!rateLimitResult.allowed) {
                // Rate limit exceeded - emit rate limited event and reject original
                this.metrics.eventsRateLimited++;

                const rateLimitedEvent = this.rateLimiter.createRateLimitedEvent(fullEvent, rateLimitResult);

                // Emit the rate limited event (bypasses rate limiting)
                await this.publishBypassRateLimit(rateLimitedEvent);

                logger.warn("[EventBus] Event rate limited", {
                    eventType: fullEvent.type,
                    limitType: rateLimitResult.limitType,
                    retryAfterMs: rateLimitResult.retryAfterMs,
                });

                return {
                    success: false,
                    error: new Error(`Rate limit exceeded: ${rateLimitResult.limitType}`),
                    duration: performance.now() - startTime,
                } as EventPublishResultWithRateLimit;
            }

            // Get event behavior from registry
            const behavior = getEventBehavior(fullEvent.type);

            // Update metrics
            this.metrics.eventsPublished++;
            this.metrics.lastEventTime = Date.now();

            // Check if we need a barrier
            const needsBarrier = await this.needsBarrier(behavior, fullEvent);

            if (needsBarrier) {
                // Use barrier sync for APPROVAL/CONSENSUS modes or when intercepted
                return this.publishWithBarrier(fullEvent, behavior);
            }

            // Fast path: emit without barrier
            await this.emitToSubscribers(fullEvent);
            this.metrics.eventsDelivered++;

            // Record event in monitor
            const duration = performance.now() - startTime;
            this.monitor.recordEvent(event.type, behavior.mode || EventMode.PASSIVE, duration, true);

            logger.debug("[EventBus] Published event (fast path)", {
                eventType: fullEvent.type,
                mode: behavior.mode,
                rateLimitStatus: "allowed",
            });

            return {
                success: true,
                duration,
                eventId: fullEvent.id,
                remainingQuota: rateLimitResult.remainingQuota,
                resetTime: rateLimitResult.resetTime,
            } as EventPublishResultWithRateLimit;

        } catch (error) {
            this.metrics.eventsFailed++;

            // Record failure in monitor
            const duration = performance.now() - startTime;
            const behavior = getEventBehavior(event.type);
            this.monitor.recordEvent(event.type, behavior.mode || EventMode.PASSIVE, duration, false);

            logger.error("[EventBus] Failed to publish event", {
                eventType: event.type,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: error instanceof Error ? error : new Error(String(error)),
                duration,
            };
        }
    }

    /**
     * Publish an event bypassing rate limits (for internal system events)
     */
    private async publishBypassRateLimit(event: ServiceEvent): Promise<void> {
        // For system events, always use fast path
        await this.emitToSubscribers(event);
        this.emitToSocketClients(event);

        logger.debug("[EventBus] Published system event bypassing rate limits", {
            eventType: event.type,
        });
    }


    /**
     * Subscribe to event patterns with optional filtering
     */
    async subscribe<T extends ServiceEvent>(
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
     * Get comprehensive event bus metrics including rate limiting and performance monitoring
     */
    getMetrics(): typeof this.metrics & {
        pendingBarriers: number;
        rateLimitingEnabled: boolean;
        performance: ReturnType<EventBusMonitor["getPerformanceReport"]>;
        health: ReturnType<EventBusMonitor["getSystemHealth"]>;
        optimizations: ReturnType<EventBusMonitor["getOptimizationSuggestions"]>;
    } {
        return {
            ...this.metrics,
            pendingBarriers: this.pendingBarriers.size,
            rateLimitingEnabled: true,
            performance: this.monitor.getPerformanceReport(),
            health: this.monitor.getSystemHealth(),
            optimizations: this.monitor.getOptimizationSuggestions(),
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

    private async emitToSubscribers<T extends ServiceEvent>(event: T): Promise<void> {
        let deliveredCount = 0;

        for (const subscription of Array.from(this.subscriptions.values())) {
            if (await this.matchesSubscription(event, subscription)) {
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

    private async emitToSubscribersReliably<T extends ServiceEvent>(event: T): Promise<void> {
        const promises: Promise<void>[] = [];

        for (const subscription of Array.from(this.subscriptions.values())) {
            if (await this.matchesSubscription(event, subscription)) {
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
        event: ServiceEvent,
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

    private async matchesSubscription(event: ServiceEvent, subscription: EventSubscription): Promise<boolean> {
        for (const pattern of subscription.patterns) {
            if (await this.matchesPattern(event.type, pattern)) {
                return true;
            }
        }
        return false;
    }

    private async matchesPattern(eventType: string, pattern: string): Promise<boolean> {
        // Use mqtt-match library for proper MQTT pattern matching
        // This handles +, #, and exact matching according to MQTT standards
        if (!this.mqttMatch) {
            try {
                const mqttMatchModule = await import("mqtt-match");
                // Handle different export patterns
                this.mqttMatch = (mqttMatchModule as any).default || mqttMatchModule;
            } catch (error) {
                logger.warn("[EventBus] Failed to load mqtt-match, using fallback pattern matching", {
                    error: error instanceof Error ? error.message : String(error),
                });
                // Fallback to simple wildcard matching
                this.mqttMatch = (pattern: string, topic: string) => {
                    const regexPattern = pattern
                        .replace(/\+/g, "[^/]+")    // + matches single level
                        .replace(/#/g, ".*")        // # matches multi-level
                        .replace(/\*/g, "[^/]*");   // * matches within level
                    const regex = new RegExp(`^${regexPattern}$`);
                    return regex.test(topic);
                };
            }
        }
        if (!this.mqttMatch) {
            throw new Error("mqttMatch not initialized");
        }
        return this.mqttMatch(pattern, eventType);
    }

    private handleBarrierTimeout(eventId: string, startTime: number): void {
        const barrier = this.pendingBarriers.get(eventId);
        if (!barrier || barrier.isCompleted) return;

        const timeoutAction = barrier.barrierConfig?.timeoutAction || "block";

        switch (timeoutAction) {
            case "continue":
                this.completeBarrierWithProgression(eventId, "continue", "Auto-approved on timeout");
                break;
            case "block":
                this.completeBarrierWithProgression(eventId, "block", "Auto-rejected on timeout");
                break;
            case "defer":
                // Don't complete - leave pending for manual resolution
                logger.warn("[EventBus] Barrier sync timed out but keeping pending", {
                    eventId,
                    duration: performance.now() - startTime,
                });
                break;
        }

        this.metrics.barrierSyncsTimedOut++;
    }

    /**
     * Emit event to socket clients if socket service is configured
     */
    private emitToSocketClients<T extends ServiceEvent>(event: T): void {
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
    private extractRoomId(event: Pick<ServiceEvent, "type" | "data">): string | null {
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
            return (event.data as Record<string, unknown>)?.runId as string || null;
        }

        // For user events - use user room
        if (event.type.startsWith("user/")) {
            return (event.data as Record<string, unknown>)?.userId as string || null;
        }

        // For room events - extract room ID from the event data
        if (event.type.startsWith("room/")) {
            return (event.data as Record<string, unknown>)?.roomId as string || null;
        }

        // Unknown event type pattern
        logger.debug("[EventBus] Unknown event type pattern for socket emission", {
            eventType: event.type,
        });
        return null;
    }

    /**
     * Extract chat ID from event data
     */
    private extractChatId(event: Pick<ServiceEvent, "data">): string | null {
        // Try multiple possible locations for chat ID
        const data = event.data as Record<string, unknown>;
        return (data?.chatId as string) ||
            (data?.conversationId as string) ||
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

            if (subscriptionCount > EVENT_BUS_CONSTANTS.MAX_EVENT_LISTENERS * SUBSCRIPTION_WARNING_THRESHOLD) {
                logger.warn("[EventBus] High subscription count detected", {
                    subscriptionCount,
                    maxListeners: EVENT_BUS_CONSTANTS.MAX_EVENT_LISTENERS,
                });
            }

            if (pendingBarriers > HIGH_PENDING_BARRIER_THRESHOLD) {
                logger.warn("[EventBus] High pending barrier count detected", {
                    pendingBarriers,
                });
            }
        }, MINUTES_1_MS);
    }

    /**
     * Create a full event with ID and timestamp
     */
    private createFullEvent(event: Omit<ServiceEvent, "id" | "timestamp">): ServiceEvent {
        return {
            ...event,
            id: nanoid(),
            timestamp: new Date(),
        };
    }

    /**
     * Determine if an event needs barrier synchronization
     */
    private async needsBarrier(behavior: ReturnType<typeof getEventBehavior>, event: ServiceEvent): Promise<boolean> {
        // Check event mode
        if (behavior.mode === EventMode.APPROVAL || behavior.mode === EventMode.CONSENSUS) {
            return true;
        }

        // For INTERCEPTABLE mode, check if there are subscribers
        if (behavior.mode === EventMode.INTERCEPTABLE) {
            const subscriberCount = await this.getSubscriberCountForEvent(event);
            return subscriberCount > 0;
        }

        // PASSIVE mode never needs barrier
        return false;
    }

    /**
     * Get subscriber count for a specific event
     */
    private async getSubscriberCountForEvent(event: ServiceEvent): Promise<number> {
        let count = 0;
        for (const subscription of this.subscriptions.values()) {
            if (await this.matchesSubscription(event, subscription)) {
                count++;
            }
        }
        return count;
    }

    /**
     * Get subscriber count for a pattern (implements IEventBus interface)
     */
    getSubscriberCount(pattern: string): number {
        let count = 0;
        for (const subscription of this.subscriptions.values()) {
            if (subscription.patterns.includes(pattern)) {
                count++;
            }
        }
        return count;
    }

    /**
     * Publish event with barrier synchronization
     */
    private async publishWithBarrier(event: ServiceEvent, behavior: ReturnType<typeof getEventBehavior>): Promise<EventPublishResult> {
        const startTime = performance.now();
        const barrierConfig = behavior.barrierConfig || {
            quorum: 1,
            timeoutMs: EVENT_BUS_CONSTANTS.DEFAULT_BARRIER_TIMEOUT_MS,
            timeoutAction: "block",
        };

        return new Promise<EventPublishResult>((resolve, reject) => {
            const pendingBarrier: PendingBarrier = {
                eventId: event.id,
                event,
                responses: [],
                resolve: (result) => {
                    // Convert to EventPublishResult
                    resolve({
                        ...result,
                        wasBlocking: true,
                    });
                },
                reject,
                timeoutId: setTimeout(() => {
                    this.handleBarrierTimeout(event.id, startTime);
                }, barrierConfig.timeoutMs),
                startTime,
                isCompleted: false,
                barrierConfig,
            };

            this.pendingBarriers.set(event.id, pendingBarrier);

            // Emit the event for agents to respond
            this.emitToSubscribers(event).then(() => {
                logger.debug("[EventBus] Published barrier event", {
                    eventId: event.id,
                    eventType: event.type,
                    mode: behavior.mode,
                    quorum: barrierConfig.quorum,
                    timeoutMs: barrierConfig.timeoutMs,
                });

                // For INTERCEPTABLE mode with no responses, complete immediately
                if (behavior.mode === EventMode.INTERCEPTABLE && pendingBarrier.responses.length === 0) {
                    // No one intercepted, continue
                    this.completeBarrierWithProgression(event.id, "continue", "No interception");
                }
            }).catch(error => {
                logger.error("[EventBus] Failed to emit barrier event", {
                    eventId: event.id,
                    error: error instanceof Error ? error.message : String(error),
                });
                // Don't reject the promise here as the barrier might still complete
            });
        });
    }

    /**
     * Complete barrier with progression control
     */
    private completeBarrierWithProgression(eventId: string, progression: ProgressionControl, reason?: string): void {
        const barrier = this.pendingBarriers.get(eventId);
        if (!barrier || barrier.isCompleted) {
            return;
        }

        barrier.isCompleted = true;
        clearTimeout(barrier.timeoutId);
        this.pendingBarriers.delete(eventId);

        const duration = performance.now() - barrier.startTime;

        const result: EventPublishResult = {
            success: true,
            duration,
            eventId,
            progression,
            responses: barrier.responses,
            wasBlocking: true,
        };

        barrier.resolve(result);
        this.metrics.barrierSyncsCompleted++;

        // Record successful barrier completion in monitor
        const behavior = getEventBehavior(barrier.event.type);
        this.monitor.recordEvent(barrier.event.type, behavior.mode || EventMode.PASSIVE, duration, true);

        logger.debug("[EventBus] Barrier sync completed", {
            eventId,
            progression,
            reason,
            responseCount: barrier.responses.length,
            duration,
        });
    }

    /**
     * Add response to barrier (new method for bot responses)
     */
    async respondToBarrier(eventId: string, responderId: string, response: BotEventResponse): Promise<void> {
        const barrier = this.pendingBarriers.get(eventId);
        if (!barrier || barrier.isCompleted) {
            logger.warn("[EventBus] Received response for unknown or completed barrier", {
                eventId,
                responderId,
            });
            return;
        }

        // Add response
        barrier.responses.push({
            responderId,
            response,
            timestamp: new Date(),
        });

        logger.debug("[EventBus] Received barrier response", {
            eventId,
            responderId,
            progression: response.progression,
            responseCount: barrier.responses.length,
        });

        // Check if we should complete based on response
        const config = barrier.barrierConfig;

        // For blockOnFirst, any block response blocks immediately
        if (config?.blockOnFirst && response.progression === "block") {
            this.completeBarrierWithProgression(eventId, "block", response.reason);
            return;
        }

        // Check if we have quorum
        if (typeof config?.quorum === "number" && barrier.responses.length >= config.quorum) {
            // Aggregate responses
            const progression = this.aggregateProgression(barrier.responses, config);
            this.completeBarrierWithProgression(eventId, progression, "Quorum reached");
        } else if (config?.quorum === "all") {
            // For "all" quorum, need to know total expected responders
            // This would require tracking bot subscriptions
            // For now, we'll rely on timeout
        }
    }

    /**
     * Aggregate progression from multiple responses
     */
    private aggregateProgression(responses: PendingBarrier["responses"], config?: PendingBarrier["barrierConfig"]): ProgressionControl {
        if (responses.length === 0) {
            return "continue";
        }

        // Count progression types
        const counts = responses.reduce((acc, r) => {
            acc[r.response.progression] = (acc[r.response.progression] || 0) + 1;
            return acc;
        }, {} as Record<ProgressionControl, number>);

        // If we have a continue threshold, check if we meet it
        if (config?.continueThreshold !== undefined) {
            const continueCount = counts.continue || 0;
            if (continueCount >= config.continueThreshold) {
                return "continue";
            }
            return "block";
        }

        // Default: Majority wins, ties go to "block" for safety
        const total = responses.length;
        if ((counts.continue || 0) > total / 2) {
            return "continue";
        }
        if ((counts.block || 0) >= total / 2) {
            return "block";
        }
        if ((counts.defer || 0) > total / 2) {
            return "defer";
        }
        if ((counts.retry || 0) > total / 2) {
            return "retry";
        }

        // Fallback to block for safety
        return "block";
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
