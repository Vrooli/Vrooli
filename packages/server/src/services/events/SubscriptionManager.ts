/**
 * Subscription Manager
 * 
 * Efficient subscription management with batching, priority queues,
 * and automatic cleanup for the event system.
 */

import { logger } from "../../events/logger.js";
import type {
    EventHandler,
    EventSubscription,
    EventSubscriptionId,
    ServiceEvent,
    SubscriptionOptions,
} from "./types.js";

/**
 * Priority queue item for event processing
 */
interface QueuedEvent {
    event: ServiceEvent;
    priority: number;
    timestamp: number;
    subscriptionId: EventSubscriptionId;
}

/**
 * Subscription statistics for monitoring
 */
interface SubscriptionStats {
    eventsProcessed: number;
    averageProcessingTime: number;
    lastEventTime: number;
    errorCount: number;
}

/**
 * Enhanced subscription with additional metadata
 */
interface EnhancedSubscription extends EventSubscription {
    stats: SubscriptionStats;
    isActive: boolean;
    errorBackoff: number;
}

/**
 * Batch processor for subscription events
 */
class BatchProcessor {
    private queue: QueuedEvent[] = [];
    private processing = false;
    private batchSize: number;
    private batchTimeout: number;
    private timeoutId?: NodeJS.Timeout;

    private static readonly DEFAULT_BATCH_SIZE = 100;
    private static readonly DEFAULT_BATCH_TIMEOUT_MS = 50;

    constructor(
        private handler: (events: QueuedEvent[]) => Promise<void>,
        options: { batchSize?: number; batchTimeout?: number } = {},
    ) {
        this.batchSize = options.batchSize || BatchProcessor.DEFAULT_BATCH_SIZE;
        this.batchTimeout = options.batchTimeout || BatchProcessor.DEFAULT_BATCH_TIMEOUT_MS;
    }

    add(event: QueuedEvent): void {
        this.queue.push(event);

        // Sort by priority (higher first) then timestamp (older first)
        this.queue.sort((a, b) => {
            if (a.priority !== b.priority) {
                return b.priority - a.priority;
            }
            return a.timestamp - b.timestamp;
        });

        // Start processing if batch is full
        if (this.queue.length >= this.batchSize) {
            this.flush();
        } else {
            // Schedule flush after timeout
            this.scheduleFlush();
        }
    }

    private scheduleFlush(): void {
        if (this.timeoutId) return;

        this.timeoutId = setTimeout(() => {
            this.timeoutId = undefined;
            this.flush();
        }, this.batchTimeout);
    }

    private async flush(): Promise<void> {
        if (this.processing || this.queue.length === 0) return;

        this.processing = true;

        // Clear timeout
        if (this.timeoutId) {
            clearTimeout(this.timeoutId);
            this.timeoutId = undefined;
        }

        // Take batch
        const batch = this.queue.splice(0, this.batchSize);

        try {
            await this.handler(batch);
        } catch (error) {
            logger.error("[BatchProcessor] Error processing batch", {
                batchSize: batch.length,
                error: error instanceof Error ? error.message : String(error),
            });
        } finally {
            this.processing = false;

            // Process next batch if queue has items
            if (this.queue.length > 0) {
                setImmediate(() => this.flush());
            }
        }
    }

    async shutdown(): Promise<void> {
        if (this.timeoutId) {
            clearTimeout(this.timeoutId);
        }
        await this.flush();
    }
}

/**
 * Efficient subscription manager with batching and indexing
 */
export class SubscriptionManager {
    private subscriptions = new Map<EventSubscriptionId, EnhancedSubscription>();
    private patternIndex = new Map<string, Set<EventSubscriptionId>>();
    private handlerIndex = new WeakMap<EventHandler, Set<EventSubscriptionId>>();
    private batchProcessors = new Map<EventSubscriptionId, BatchProcessor>();

    // Performance monitoring
    private metrics = {
        totalSubscriptions: 0,
        activeSubscriptions: 0,
        eventsDelivered: 0,
        deliveryErrors: 0,
    };

    private static readonly MAX_BACKOFF_MS = 5000;
    private static readonly RETRY_BACKOFF_BASE_MS = 100;
    private static readonly MAX_RETRY_BACKOFF_MS = 1000;
    private static readonly RANDOM_ID_RADIX = 36;
    private static readonly RANDOM_ID_LENGTH = 7;
    private static readonly INACTIVE_CLEANUP_MS = 3600000;
    private static readonly MAX_ERROR_COUNT = 100;

    /**
     * Add a subscription with enhanced tracking
     */
    addSubscription(
        patterns: string[],
        handler: EventHandler,
        options: SubscriptionOptions = {},
    ): EventSubscriptionId {
        const id = this.generateSubscriptionId();

        const subscription: EnhancedSubscription = {
            id,
            patterns,
            handler,
            options,
            createdAt: new Date(),
            stats: {
                eventsProcessed: 0,
                averageProcessingTime: 0,
                lastEventTime: 0,
                errorCount: 0,
            },
            isActive: true,
            errorBackoff: 0,
        };

        this.subscriptions.set(id, subscription);
        this.metrics.totalSubscriptions++;
        this.metrics.activeSubscriptions++;

        // Update indexes
        for (const pattern of patterns) {
            if (!this.patternIndex.has(pattern)) {
                this.patternIndex.set(pattern, new Set());
            }
            this.patternIndex.get(pattern)!.add(id);
        }

        // Update handler index
        if (!this.handlerIndex.has(handler)) {
            this.handlerIndex.set(handler, new Set());
        }
        this.handlerIndex.get(handler)!.add(id);

        // Create batch processor if batching is requested
        if (options.batchSize && options.batchSize > 1) {
            const processor = new BatchProcessor(
                events => this.processBatch(subscription, events),
                { batchSize: options.batchSize },
            );
            this.batchProcessors.set(id, processor);
        }

        logger.debug("[SubscriptionManager] Added subscription", {
            id,
            patterns,
            batching: !!options.batchSize,
        });

        return id;
    }

    /**
     * Remove a subscription and clean up
     */
    async removeSubscription(id: EventSubscriptionId): Promise<void> {
        const subscription = this.subscriptions.get(id);
        if (!subscription) return;

        // Mark as inactive
        subscription.isActive = false;
        this.metrics.activeSubscriptions--;

        // Flush any pending batches
        const processor = this.batchProcessors.get(id);
        if (processor) {
            await processor.shutdown();
            this.batchProcessors.delete(id);
        }

        // Remove from indexes
        for (const pattern of subscription.patterns) {
            this.patternIndex.get(pattern)?.delete(id);
            if (this.patternIndex.get(pattern)?.size === 0) {
                this.patternIndex.delete(pattern);
            }
        }

        // Remove from handler index
        this.handlerIndex.get(subscription.handler)?.delete(id);

        // Remove subscription
        this.subscriptions.delete(id);

        logger.debug("[SubscriptionManager] Removed subscription", {
            id,
            stats: subscription.stats,
        });
    }

    /**
     * Find subscriptions matching a pattern
     */
    findSubscriptionsByPattern(pattern: string): EnhancedSubscription[] {
        const ids = this.patternIndex.get(pattern);
        if (!ids) return [];

        return Array.from(ids)
            .map(id => this.subscriptions.get(id))
            .filter((sub): sub is EnhancedSubscription => sub !== undefined && sub.isActive);
    }

    /**
     * Deliver event to subscription
     */
    async deliverEvent(
        subscription: EnhancedSubscription,
        event: ServiceEvent,
    ): Promise<void> {
        if (!subscription.isActive) return;

        // Check if batching is enabled
        const processor = this.batchProcessors.get(subscription.id);
        if (processor) {
            // Add to batch queue
            processor.add({
                event,
                priority: this.calculateEventPriority(event),
                timestamp: Date.now(),
                subscriptionId: subscription.id,
            });
            return;
        }

        // Process immediately
        await this.processEvent(subscription, event);
    }

    /**
     * Process a single event
     */
    private async processEvent(
        subscription: EnhancedSubscription,
        event: ServiceEvent,
    ): Promise<void> {
        const startTime = Date.now();

        try {
            // Apply filter if configured
            if (subscription.options.filter && !subscription.options.filter(event)) {
                return;
            }

            // Apply backoff if errors occurred
            if (subscription.errorBackoff > 0) {
                await this.delay(subscription.errorBackoff);
            }

            // Call handler with retry logic
            await this.callHandlerWithRetry(subscription, event);

            // Update statistics
            const processingTime = Date.now() - startTime;
            subscription.stats.eventsProcessed++;
            subscription.stats.averageProcessingTime =
                (subscription.stats.averageProcessingTime * (subscription.stats.eventsProcessed - 1) + processingTime) /
                subscription.stats.eventsProcessed;
            subscription.stats.lastEventTime = Date.now();

            // Reset error backoff on success
            subscription.errorBackoff = 0;

            this.metrics.eventsDelivered++;

        } catch (error) {
            subscription.stats.errorCount++;
            this.metrics.deliveryErrors++;

            // Exponential backoff
            subscription.errorBackoff = Math.min(
                subscription.errorBackoff > 0 ? subscription.errorBackoff * 2 : SubscriptionManager.RETRY_BACKOFF_BASE_MS,
                SubscriptionManager.MAX_BACKOFF_MS,
            );

            logger.error("[SubscriptionManager] Event delivery failed", {
                subscriptionId: subscription.id,
                eventType: event.type,
                error: error instanceof Error ? error.message : String(error),
                errorCount: subscription.stats.errorCount,
                backoff: subscription.errorBackoff,
            });

            // Disable subscription if too many errors
            if (subscription.stats.errorCount > SubscriptionManager.MAX_ERROR_COUNT) {
                subscription.isActive = false;
                this.metrics.activeSubscriptions--;
                logger.error("[SubscriptionManager] Subscription disabled due to errors", {
                    subscriptionId: subscription.id,
                    errorCount: subscription.stats.errorCount,
                });
            }
        }
    }

    /**
     * Process a batch of events
     */
    private async processBatch(
        subscription: EnhancedSubscription,
        events: QueuedEvent[],
    ): Promise<void> {
        // Process events in order
        for (const { event } of events) {
            await this.processEvent(subscription, event);
        }
    }

    /**
     * Call handler with retry logic
     */
    private async callHandlerWithRetry(
        subscription: EnhancedSubscription,
        event: ServiceEvent,
    ): Promise<void> {
        const maxRetries = subscription.options.maxRetries || 0;
        let lastError: Error | undefined;

        for (let attempt = 0; attempt <= maxRetries; attempt++) {
            try {
                await subscription.handler(event);
                return; // Success
            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));

                if (attempt < maxRetries) {
                    // Exponential backoff between retries
                    const backoffMs = Math.min(SubscriptionManager.RETRY_BACKOFF_BASE_MS * Math.pow(2, attempt), SubscriptionManager.MAX_RETRY_BACKOFF_MS);
                    await this.delay(backoffMs);
                }
            }
        }

        // All retries failed
        if (lastError) {
            throw lastError;
        }
    }

    /**
     * Calculate event priority for queue ordering
     */
    private calculateEventPriority(event: ServiceEvent): number {
        const priorityScores = {
            critical: 1000,
            high: 100,
            medium: 10,
            low: 1,
        };

        return priorityScores[event.metadata?.priority || "medium"];
    }

    /**
     * Generate unique subscription ID
     */
    private generateSubscriptionId(): EventSubscriptionId {
        return `sub_${Date.now()}_${Math.random().toString(SubscriptionManager.RANDOM_ID_RADIX).substring(2, 2 + SubscriptionManager.RANDOM_ID_LENGTH)}`;
    }

    /**
     * Delay helper
     */
    private delay(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    /**
     * Get subscription statistics
     */
    getSubscriptionStats(id: EventSubscriptionId): SubscriptionStats | undefined {
        return this.subscriptions.get(id)?.stats;
    }

    /**
     * Get overall metrics
     */
    getMetrics(): typeof this.metrics {
        return { ...this.metrics };
    }

    /**
     * Clean up inactive subscriptions
     */
    async cleanup(): Promise<number> {
        let cleaned = 0;

        for (const [id, subscription] of this.subscriptions) {
            // Remove subscriptions that have been inactive for over an hour
            if (!subscription.isActive &&
                Date.now() - subscription.stats.lastEventTime > SubscriptionManager.INACTIVE_CLEANUP_MS) {
                await this.removeSubscription(id);
                cleaned++;
            }
        }

        return cleaned;
    }

    /**
     * Get subscription by handler (useful for cleanup)
     */
    getSubscriptionsByHandler(handler: EventHandler): EventSubscriptionId[] {
        return Array.from(this.handlerIndex.get(handler) || []);
    }
}
