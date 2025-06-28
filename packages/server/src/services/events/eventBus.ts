/**
 * Enhanced Event Bus Implementation
 * 
 * Provides unified event publishing and subscription for the execution architecture
 * with support for delivery guarantees and barrier synchronization.
 * 
 * This replaces the fragmented event system with a single, consistent interface
 * that enables true emergent capabilities through agent-extensible event types.
 */

import { EventEmitter } from "events";
import { type Logger } from "winston";
import { nanoid } from "@vrooli/shared";
import {
  type BaseEvent,
  type EventHandler,
  type SubscriptionOptions,
  type SubscriptionId,
  type PublishResult,
  type BarrierSyncResult,
  type SafetyEvent,
  type EventSource,
} from "./types.js";
import {
  EVENT_BUS_CONSTANTS,
  COMPONENT_NAMES,
  DELIVERY_GUARANTEES,
} from "./constants.js";
import {
  withRetry,
  DEFAULT_RETRY_STRATEGIES,
  createPatternMatcher,
  type EventPatternMatcher,
  formatEventForLogging,
} from "./utils.js";

/**
 * Enhanced event bus interface supporting delivery guarantees and barrier synchronization
 */
export interface IEventBus {
  /**
   * Publish an event with specified delivery guarantee
   */
  publish<T extends BaseEvent>(event: T): Promise<PublishResult>;
  
  /**
   * Subscribe to event patterns with optional filtering
   */
  subscribe<T extends BaseEvent>(
    pattern: string | string[],
    handler: EventHandler<T>,
    options?: SubscriptionOptions
  ): Promise<SubscriptionId>;
  
  /**
   * Unsubscribe from events
   */
  unsubscribe(subscriptionId: SubscriptionId): Promise<void>;
  
  /**
   * Handle barrier sync events (blocking until responses received)
   */
  publishBarrierSync<T extends SafetyEvent>(
    event: T
  ): Promise<BarrierSyncResult>;
  
  /**
   * Start the event bus
   */
  start(): Promise<void>;
  
  /**
   * Stop the event bus
   */
  stop(): Promise<void>;
}

/**
 * Subscription information
 */
interface Subscription {
  id: SubscriptionId;
  patterns: string[];
  handler: EventHandler;
  options: SubscriptionOptions;
  createdAt: Date;
}

/**
 * Pending barrier sync operation
 */
interface PendingBarrier {
  eventId: string;
  event: SafetyEvent;
  responses: Array<{ responderId: string; response: "OK" | "ALARM"; reason?: string }>;
  resolve: (result: BarrierSyncResult) => void;
  reject: (error: Error) => void;
  timeoutId: NodeJS.Timeout;
  startTime: number;
}

/**
 * Enhanced event bus implementation
 */
export class EventBus implements IEventBus {
  private readonly emitter: EventEmitter;
  private readonly subscriptions = new Map<SubscriptionId, Subscription>();
  private readonly pendingBarriers = new Map<string, PendingBarrier>();
  private readonly eventHistory: BaseEvent[] = [];
  private readonly patternCache = new Map<string, EventPatternMatcher>();
  private isStarted = false;

  // Metrics
  private metrics = {
    eventsPublished: 0,
    eventsDelivered: 0,
    eventsFailed: 0,
    barrierSyncsCompleted: 0,
    barrierSyncsTimedOut: 0,
    activeSubscriptions: 0,
    lastEventTime: 0,
  };

  constructor(private readonly logger: Logger) {
    this.emitter = new EventEmitter();
    this.emitter.setMaxListeners(EVENT_BUS_CONSTANTS.MAX_EVENT_LISTENERS); // Allow many subscribers
  }

  /**
   * Start the event bus
   */
  async start(): Promise<void> {
    if (this.isStarted) {
      this.logger.warn("[EventBus] Already started");
      return;
    }

    this.isStarted = true;
    this.logger.info("[EventBus] Started enhanced event bus");
  }

  /**
   * Stop the event bus
   */
  async stop(): Promise<void> {
    if (!this.isStarted) {
      return;
    }

    this.isStarted = false;

    // Clear pending barriers
    for (const barrier of this.pendingBarriers.values()) {
      clearTimeout(barrier.timeoutId);
      barrier.reject(new Error("Event bus stopped"));
    }
    this.pendingBarriers.clear();

    // Clear subscriptions
    this.emitter.removeAllListeners();
    this.subscriptions.clear();

    this.logger.info("[EventBus] Stopped event bus");
  }

  /**
   * Publish an event with specified delivery guarantee
   */
  async publish<T extends BaseEvent>(event: T): Promise<PublishResult> {
    const startTime = performance.now();

    if (!this.isStarted) {
      return {
        success: false,
        error: new Error("Event bus not started"),
        duration: performance.now() - startTime,
      };
    }

    try {
      // Add to history
      this.addToHistory(event);

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

      this.logger.debug("[EventBus] Published event", {
        eventId: event.id,
        eventType: event.type,
        deliveryGuarantee,
      });

      return {
        success: true,
        duration: performance.now() - startTime,
      };

    } catch (error) {
      this.metrics.eventsFailed++;
      
      this.logger.error("[EventBus] Failed to publish event", {
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
   * Handle barrier sync events (blocking until responses received)
   */
  async publishBarrierSync<T extends SafetyEvent>(event: T): Promise<BarrierSyncResult> {
    const startTime = performance.now();

    if (!this.isStarted) {
      throw new Error("Event bus not started");
    }

    const barrierConfig = event.metadata.barrierConfig;
    if (!barrierConfig) {
      throw new Error("Barrier sync event must have barrierConfig");
    }

    return new Promise<BarrierSyncResult>((resolve, reject) => {
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
      };

      this.pendingBarriers.set(event.id, pendingBarrier);

      // Add to history
      this.addToHistory(event);

      // Emit the event for safety agents to respond
      this.emitToSubscribers(event);

      this.logger.debug("[EventBus] Published barrier sync event", {
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
  ): Promise<SubscriptionId> {
    const subscriptionId = nanoid();
    const patterns = Array.isArray(pattern) ? pattern : [pattern];

    const subscription: Subscription = {
      id: subscriptionId,
      patterns,
      handler: handler as EventHandler,
      options,
      createdAt: new Date(),
    };

    this.subscriptions.set(subscriptionId, subscription);
    this.metrics.activeSubscriptions = this.subscriptions.size;

    this.logger.info("[EventBus] Added subscription", {
      subscriptionId,
      patterns,
    });

    return subscriptionId;
  }

  /**
   * Unsubscribe from events
   */
  async unsubscribe(subscriptionId: SubscriptionId): Promise<void> {
    const removed = this.subscriptions.delete(subscriptionId);
    this.metrics.activeSubscriptions = this.subscriptions.size;

    if (removed) {
      this.logger.info("[EventBus] Removed subscription", {
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
      this.logger.warn("[EventBus] Response to unknown barrier sync event", {
        eventId,
        responderId,
        response,
      });
      return;
    }

    // Add response
    barrier.responses.push({ responderId, response, reason });

    this.logger.debug("[EventBus] Received barrier sync response", {
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
   * Get event bus metrics
   */
  getMetrics(): typeof this.metrics & {
    pendingBarriers: number;
    historySize: number;
  } {
    return {
      ...this.metrics,
      pendingBarriers: this.pendingBarriers.size,
      historySize: this.eventHistory.length,
    };
  }

  /**
   * Private methods
   */

  private async publishFireAndForget<T extends BaseEvent>(event: T): Promise<void> {
    // Fire and forget - emit and don't wait for delivery confirmation
    this.emitToSubscribers(event);
  }

  private async publishReliable<T extends BaseEvent>(event: T): Promise<void> {
    // Reliable delivery - emit and ensure delivery to all subscribers
    await this.emitToSubscribersReliably(event);
  }

  private emitToSubscribers<T extends BaseEvent>(event: T): void {
    let deliveredCount = 0;

    for (const subscription of this.subscriptions.values()) {
      if (this.matchesSubscription(event, subscription)) {
        // Don't await - fire and forget
        this.callSubscriptionHandler(subscription, event)
          .then(() => deliveredCount++)
          .catch(error => {
            this.logger.error("[EventBus] Subscription handler error", {
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
            this.logger.error("[EventBus] Subscription handler error", {
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
    subscription: Subscription,
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

  private matchesSubscription(event: BaseEvent, subscription: Subscription): boolean {
    return subscription.patterns.some(pattern => this.matchesPattern(event.type, pattern));
  }

  private matchesPattern(eventType: string, pattern: string): boolean {
    // MQTT-style pattern matching
    if (pattern === "#") return true; // Match all
    if (pattern === eventType) return true; // Exact match
    
    // Wildcard patterns
    if (pattern.includes("+") || pattern.includes("*")) {
      const regex = pattern
        .replace(/\+/g, "[^/]+") // + matches one level
        .replace(/\*/g, ".*");    // * matches everything
      return new RegExp(`^${regex}$`).test(eventType);
    }

    // Hierarchical prefix matching
    if (pattern.endsWith("/#")) {
      const prefix = pattern.slice(0, pattern.length - EVENT_BUS_CONSTANTS.PATTERN_SUFFIX_LENGTH);
      return eventType.startsWith(prefix + "/") || eventType === prefix;
    }

    return false;
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
        this.logger.warn("[EventBus] Barrier sync timed out but keeping pending", {
          eventId,
          duration: performance.now() - startTime,
        });
        break;
    }
  }

  private completeBarrierSync(eventId: string, success: boolean, reason: string): void {
    const barrier = this.pendingBarriers.get(eventId);
    if (!barrier) return;

    clearTimeout(barrier.timeoutId);
    this.pendingBarriers.delete(eventId);

    const result: BarrierSyncResult = {
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

    this.logger.debug("[EventBus] Completed barrier sync", {
      eventId,
      success,
      reason,
      responseCount: barrier.responses.length,
      duration: result.duration,
    });

    barrier.resolve(result);
  }

  private addToHistory(event: BaseEvent): void {
    this.eventHistory.push(event);
    
    // Keep history size manageable
    if (this.eventHistory.length > EVENT_BUS_CONSTANTS.MAX_EVENT_HISTORY) {
      this.eventHistory.splice(0, this.eventHistory.length - EVENT_BUS_CONSTANTS.MAX_EVENT_HISTORY);
    }
  }
}

/**
 * Create an enhanced event bus instance
 */
export function createEventBus(logger: Logger): EventBus {
  return new EventBus(logger);
}
