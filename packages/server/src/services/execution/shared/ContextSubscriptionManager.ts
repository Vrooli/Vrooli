/**
 * Context Subscription Manager - Live Update Propagation for Emergent Coordination
 * 
 * This component enables real-time communication between swarm components by managing
 * subscriptions to context changes. It's a critical enabler of emergent capabilities
 * because it allows agents and system components to react instantly to configuration
 * changes, resource updates, and emergent events.
 * 
 * ## Key Emergent Capabilities Enabled:
 * 
 * 1. **Live Agent Coordination**: Agents can subscribe to specific context changes
 *    and react immediately, enabling dynamic team formation and role adaptation
 * 
 * 2. **Real-Time Resource Optimization**: Resource management agents can monitor
 *    allocation patterns and adjust strategies in real-time
 * 
 * 3. **Instant Policy Propagation**: Security and organizational policy changes
 *    propagate instantly to all running components without restarts
 * 
 * 4. **Emergent Event Cascades**: Changes from one agent can trigger cascading
 *    optimizations from other agents, creating emergent system behavior
 * 
 * ## Architecture Integration:
 * 
 * ```
 * SwarmContextManager
 *         ↓ publishes updates to
 * ContextSubscriptionManager (This Component)
 *         ↓ notifies subscribers
 * State Machines | Agents | UI Components | Monitoring Systems
 * ```
 * 
 * ## Design Philosophy:
 * 
 * Traditional systems require restarts or polling to pick up configuration changes.
 * This manager enables true live updates where:
 * - Policy changes propagate instantly to running swarms
 * - Resource allocation adjustments take effect immediately
 * - Agent optimizations can trigger real-time system adaptations
 * - UI components stay synchronized with swarm state changes
 * 
 * @see SwarmContextManager - Publishes context updates through this manager
 * @see UnifiedSwarmContext - The context model that drives all updates
 * @see /docs/architecture/execution/swarm-state-management-redesign.md - Complete architecture
 */

import {
    generatePK,
} from "@vrooli/shared";
import { type Redis as RedisClient } from "ioredis";
import { logger } from "../../../events/logger.js";
import {
    type ContextSubscription,
    type ContextUpdateEvent,
    type SwarmId,
    UnifiedSwarmContextGuards,
} from "./UnifiedSwarmContext.js";

/**
 * Subscription filter for fine-grained update control
 */
export interface SubscriptionFilter {
    /** JSONPath patterns to match against change paths */
    pathPatterns: string[];

    /** Change types to include */
    changeTypes?: ("created" | "updated" | "deleted")[];

    /** Only include changes made by specific entities */
    updatedBy?: string[];

    /** Only include emergent (agent-driven) changes */
    emergentOnly?: boolean;

    /** Minimum change severity to include */
    minSeverity?: "low" | "medium" | "high" | "critical";

    /** Rate limiting for high-frequency updates */
    rateLimit?: {
        maxNotificationsPerSecond: number;
        burstAllowance: number;
    };
}

/**
 * Subscription health metrics for monitoring and optimization
 */
export interface SubscriptionMetrics {
    subscriptionId: string;
    swarmId: SwarmId;
    subscriberId: string;

    // Performance metrics
    totalNotifications: number;
    successfulNotifications: number;
    failedNotifications: number;
    averageProcessingTimeMs: number;
    lastNotificationAt?: Date;

    // Rate limiting metrics
    notificationsInLastSecond: number;
    burstTokensUsed: number;
    rateLimitHits: number;

    // Health indicators
    consecutiveFailures: number;
    isHealthy: boolean;
    lastError?: string;
}

/**
 * Configuration for ContextSubscriptionManager
 */
export interface ContextSubscriptionManagerConfig {
    /** Redis channel prefix for pub/sub */
    channelPrefix: string;

    /** Default rate limiting settings */
    defaultRateLimit: {
        maxNotificationsPerSecond: number;
        burstAllowance: number;
    };

    /** Health check settings */
    healthCheck: {
        maxConsecutiveFailures: number;
        unhealthySubscriptionTTL: number; // seconds
        cleanupInterval: number; // seconds
    };

    /** Performance optimization settings */
    optimization: {
        batchNotifications: boolean;
        batchSize: number;
        batchTimeoutMs: number;
        enableMetrics: boolean;
    };

    /** Pub/sub settings */
    pubsub: {
        maxRetries: number;
        retryDelayMs: number;
        connectionTimeout: number;
    };
}

/**
 * Notification batch for optimized delivery
 */
interface NotificationBatch {
    subscriptionId: string;
    events: ContextUpdateEvent[];
    scheduledAt: Date;
    timeoutHandle: NodeJS.Timeout;
}

/**
 * Rate limiter state for subscription management
 */
interface RateLimiterState {
    tokens: number;
    lastRefill: Date;
    burstTokens: number;
}

/**
 * Context Subscription Manager Implementation
 * 
 * Manages real-time propagation of context updates to enable emergent coordination.
 * Provides efficient, filtered, rate-limited notifications with health monitoring.
 */
export class ContextSubscriptionManager {
    private readonly config: ContextSubscriptionManagerConfig;

    // Subscription management
    private readonly subscriptions = new Map<string, ContextSubscription & { filter?: SubscriptionFilter }>();
    private readonly subscriptionMetrics = new Map<string, SubscriptionMetrics>();
    private readonly rateLimiters = new Map<string, RateLimiterState>();

    // Notification batching
    private readonly notificationBatches = new Map<string, NotificationBatch>();

    // Redis pub/sub
    private subscriberClient?: RedisClient;
    private readonly subscribedChannels = new Set<string>();

    // Health monitoring
    private healthCheckInterval?: NodeJS.Timeout;
    private cleanupInterval?: NodeJS.Timeout;

    // Performance tracking
    private readonly globalMetrics = {
        totalSubscriptions: 0,
        activeSubscriptions: 0,
        totalNotifications: 0,
        failedNotifications: 0,
        averageProcessingTimeMs: 0,
        rateLimitHits: 0,
    };

    constructor(
        redis: RedisClient,
        config: Partial<ContextSubscriptionManagerConfig> = {},
    ) {
        this.config = {
            channelPrefix: "swarm_context:",
            defaultRateLimit: {
                maxNotificationsPerSecond: 10,
                burstAllowance: 5,
            },
            healthCheck: {
                maxConsecutiveFailures: 5,
                unhealthySubscriptionTTL: 300, // 5 minutes
                cleanupInterval: 60, // 1 minute
            },
            optimization: {
                batchNotifications: true,
                batchSize: 10,
                batchTimeoutMs: 100,
                enableMetrics: true,
            },
            pubsub: {
                maxRetries: 3,
                retryDelayMs: 1000,
                connectionTimeout: 5000,
            },
            ...config,
        };

        logger.info("[ContextSubscriptionManager] Initialized for emergent coordination", {
            config: this.config,
        });
    }

    /**
     * Start the subscription manager and setup Redis pub/sub
     */
    async start(): Promise<void> {
        try {
            // Create dedicated Redis client for pub/sub
            this.subscriberClient = this.redis.duplicate();

            // Setup message handler
            this.subscriberClient.on("message", this.handleRedisMessage.bind(this));

            // Start health monitoring
            this.startHealthMonitoring();

            logger.info("[ContextSubscriptionManager] Started successfully");

        } catch (error) {
            logger.error("[ContextSubscriptionManager] Failed to start", {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Stop the subscription manager and cleanup resources
     */
    async stop(): Promise<void> {
        try {
            // Stop health monitoring
            if (this.healthCheckInterval) {
                clearInterval(this.healthCheckInterval);
            }
            if (this.cleanupInterval) {
                clearInterval(this.cleanupInterval);
            }

            // Clear notification batches
            for (const batch of this.notificationBatches.values()) {
                clearTimeout(batch.timeoutHandle);
            }
            this.notificationBatches.clear();

            // Disconnect pub/sub client
            if (this.subscriberClient) {
                await this.subscriberClient.quit();
            }

            // Clear state
            this.subscriptions.clear();
            this.subscriptionMetrics.clear();
            this.rateLimiters.clear();

            logger.info("[ContextSubscriptionManager] Stopped successfully");

        } catch (error) {
            logger.error("[ContextSubscriptionManager] Error during stop", {
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Create a subscription for live context updates
     */
    async createSubscription(
        subscription: Omit<ContextSubscription, "id" | "metadata">,
        filter?: SubscriptionFilter,
    ): Promise<string> {
        const subscriptionId = generatePK();

        const fullSubscription: ContextSubscription & { filter?: SubscriptionFilter } = {
            ...subscription,
            id: subscriptionId,
            filter,
            metadata: {
                createdAt: new Date(),
                lastNotified: new Date(),
                totalNotifications: 0,
                subscriptionType: "agent", // Default for emergent capabilities
            },
        };

        // Initialize metrics
        const metrics: SubscriptionMetrics = {
            subscriptionId,
            swarmId: subscription.swarmId,
            subscriberId: subscription.subscriberId,
            totalNotifications: 0,
            successfulNotifications: 0,
            failedNotifications: 0,
            averageProcessingTimeMs: 0,
            notificationsInLastSecond: 0,
            burstTokensUsed: 0,
            rateLimitHits: 0,
            consecutiveFailures: 0,
            isHealthy: true,
        };

        // Initialize rate limiter
        const rateLimit = filter?.rateLimit || this.config.defaultRateLimit;
        const rateLimiter: RateLimiterState = {
            tokens: rateLimit.maxNotificationsPerSecond,
            lastRefill: new Date(),
            burstTokens: rateLimit.burstAllowance,
        };

        // Store subscription and related data
        this.subscriptions.set(subscriptionId, fullSubscription);
        this.subscriptionMetrics.set(subscriptionId, metrics);
        this.rateLimiters.set(subscriptionId, rateLimiter);

        // Subscribe to Redis channel for this swarm
        await this.subscribeToSwarmChannel(subscription.swarmId);

        // Update global metrics
        this.globalMetrics.totalSubscriptions++;
        this.globalMetrics.activeSubscriptions++;

        logger.debug("[ContextSubscriptionManager] Created subscription for emergent coordination", {
            subscriptionId,
            swarmId: subscription.swarmId,
            subscriberId: subscription.subscriberId,
            watchPaths: subscription.watchPaths,
            hasFilter: !!filter,
            rateLimit,
        });

        return subscriptionId;
    }

    /**
     * Remove a subscription
     */
    async removeSubscription(subscriptionId: string): Promise<void> {
        const subscription = this.subscriptions.get(subscriptionId);
        if (!subscription) {
            return;
        }

        // Remove from all maps
        this.subscriptions.delete(subscriptionId);
        this.subscriptionMetrics.delete(subscriptionId);
        this.rateLimiters.delete(subscriptionId);

        // Clear any pending notification batch
        const batch = this.notificationBatches.get(subscriptionId);
        if (batch) {
            clearTimeout(batch.timeoutHandle);
            this.notificationBatches.delete(subscriptionId);
        }

        // Check if we should unsubscribe from Redis channel
        await this.checkChannelUnsubscription(subscription.swarmId);

        // Update global metrics
        this.globalMetrics.activeSubscriptions--;

        logger.debug("[ContextSubscriptionManager] Removed subscription", {
            subscriptionId,
            swarmId: subscription.swarmId,
            subscriberId: subscription.subscriberId,
        });
    }

    /**
     * Publish a context update event to all relevant subscribers
     */
    async publishUpdate(event: ContextUpdateEvent): Promise<void> {
        logger.debug("[ContextSubscriptionManager] Publishing context update", {
            swarmId: event.swarmId,
            version: event.newVersion,
            changesCount: event.changes.length,
            emergent: event.emergent,
        });

        try {
            // Validate event
            if (!UnifiedSwarmContextGuards.isContextUpdateEvent(event)) {
                throw new Error("Invalid context update event format");
            }

            // Publish to Redis for cross-instance coordination
            const channel = `${this.config.channelPrefix}${event.swarmId}`;
            const message = JSON.stringify(event);
            await this.redis.publish(channel, message);

            // Process local subscriptions immediately
            await this.processLocalSubscriptions(event);

        } catch (error) {
            logger.error("[ContextSubscriptionManager] Failed to publish update", {
                swarmId: event.swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Get subscription metrics for monitoring and optimization
     */
    getSubscriptionMetrics(subscriptionId?: string): SubscriptionMetrics | SubscriptionMetrics[] {
        if (subscriptionId) {
            const metrics = this.subscriptionMetrics.get(subscriptionId);
            if (!metrics) {
                throw new Error(`Subscription not found: ${subscriptionId}`);
            }
            return metrics;
        }

        return Array.from(this.subscriptionMetrics.values());
    }

    /**
     * Get global performance metrics
     */
    getGlobalMetrics(): typeof this.globalMetrics {
        return { ...this.globalMetrics };
    }

    /**
     * Get health status of all subscriptions
     */
    getHealthStatus(): {
        healthy: boolean;
        totalSubscriptions: number;
        healthySubscriptions: number;
        unhealthySubscriptions: number;
        details: { subscriptionId: string; healthy: boolean; lastError?: string }[];
    } {
        const healthyCount = Array.from(this.subscriptionMetrics.values())
            .filter(m => m.isHealthy).length;

        const unhealthyCount = this.subscriptionMetrics.size - healthyCount;

        return {
            healthy: unhealthyCount === 0,
            totalSubscriptions: this.subscriptionMetrics.size,
            healthySubscriptions: healthyCount,
            unhealthySubscriptions: unhealthyCount,
            details: Array.from(this.subscriptionMetrics.values()).map(m => ({
                subscriptionId: m.subscriptionId,
                healthy: m.isHealthy,
                lastError: m.lastError,
            })),
        };
    }

    // Private implementation methods

    private async subscribeToSwarmChannel(swarmId: SwarmId): Promise<void> {
        if (!this.subscriberClient) {
            throw new Error("Subscriber client not initialized");
        }

        const channel = `${this.config.channelPrefix}${swarmId}`;

        try {
            if (!this.subscribedChannels.has(channel)) {
                await this.subscriberClient.subscribe(channel);
                this.subscribedChannels.add(channel);

                logger.debug("[ContextSubscriptionManager] Subscribed to Redis channel", {
                    channel,
                    swarmId,
                    totalChannels: this.subscribedChannels.size,
                });
            }
        } catch (error) {
            logger.error("[ContextSubscriptionManager] Failed to subscribe to Redis channel", {
                channel,
                swarmId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    private async handleRedisMessage(channel: string, message: string): Promise<void> {
        try {
            // Parse and validate the event
            let event: ContextUpdateEvent;
            try {
                const parsed = JSON.parse(message);
                if (!UnifiedSwarmContextGuards.isContextUpdateEvent(parsed)) {
                    logger.error("[ContextSubscriptionManager] Invalid event format in Redis message", {
                        channel,
                        messageLength: message.length,
                        parsedKeys: Object.keys(parsed),
                    });
                    return;
                }
                event = parsed;
            } catch (parseError) {
                logger.error("[ContextSubscriptionManager] Failed to parse Redis message", {
                    channel,
                    error: parseError instanceof Error ? parseError.message : String(parseError),
                    messageLength: message.length,
                });
                return;
            }

            // Extract swarm ID from channel
            const swarmId = channel.replace(this.config.channelPrefix, "");

            if (event.swarmId !== swarmId) {
                logger.warn("[ContextSubscriptionManager] Swarm ID mismatch in Redis message", {
                    channelSwarmId: swarmId,
                    eventSwarmId: event.swarmId,
                });
                return;
            }

            // Process local subscriptions
            await this.processLocalSubscriptions(event);

        } catch (error) {
            logger.error("[ContextSubscriptionManager] Failed to handle Redis message", {
                channel,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    private async processLocalSubscriptions(event: ContextUpdateEvent): Promise<void> {
        const swarmSubscriptions = Array.from(this.subscriptions.values())
            .filter(sub => sub.swarmId === event.swarmId);

        for (const subscription of swarmSubscriptions) {
            try {
                // Check if subscription is interested in this update
                if (!this.isSubscriptionInterestedInEvent(subscription, event)) {
                    continue;
                }

                // Apply rate limiting
                if (!this.checkRateLimit(subscription.id)) {
                    continue;
                }

                // Deliver notification (batched or immediate)
                await this.deliverNotification(subscription, event);

            } catch (error) {
                await this.handleSubscriptionError(subscription.id, error);
            }
        }

        this.globalMetrics.totalNotifications++;
    }

    private isSubscriptionInterestedInEvent(
        subscription: ContextSubscription & { filter?: SubscriptionFilter },
        event: ContextUpdateEvent,
    ): boolean {
        // Check path patterns from subscription
        const pathMatch = event.changes.some(change =>
            subscription.watchPaths.some(pattern =>
                this.matchesPattern(change.path, pattern),
            ),
        );

        if (!pathMatch) {
            return false;
        }

        // Apply additional filters if present
        const filter = subscription.filter;
        if (!filter) {
            return true;
        }

        // Check change types
        if (filter.changeTypes && filter.changeTypes.length > 0) {
            const changeTypeMatch = event.changes.some(change =>
                filter.changeTypes!.includes(change.changeType),
            );
            if (!changeTypeMatch) {
                return false;
            }
        }

        // Check updatedBy filter
        if (filter.updatedBy && filter.updatedBy.length > 0) {
            if (!filter.updatedBy.includes(event.updatedBy)) {
                return false;
            }
        }

        // Check emergent filter
        if (filter.emergentOnly && !event.emergent) {
            return false;
        }

        return true;
    }

    private matchesPattern(path: string, pattern: string): boolean {
        // Simple pattern matching - supports wildcards
        const regex = new RegExp(
            "^" + pattern.replace(/\*/g, ".*").replace(/\?/g, ".") + "$",
        );
        return regex.test(path);
    }

    private checkRateLimit(subscriptionId: string): boolean {
        const rateLimiter = this.rateLimiters.get(subscriptionId);
        const subscription = this.subscriptions.get(subscriptionId);

        if (!rateLimiter || !subscription) {
            return false;
        }

        const rateLimit = subscription.filter?.rateLimit || this.config.defaultRateLimit;
        const now = new Date();

        // Refill tokens based on time elapsed
        const timeSinceLastRefill = now.getTime() - rateLimiter.lastRefill.getTime();
        const tokensToAdd = Math.floor(timeSinceLastRefill / 1000) * rateLimit.maxNotificationsPerSecond;

        if (tokensToAdd > 0) {
            rateLimiter.tokens = Math.min(
                rateLimit.maxNotificationsPerSecond,
                rateLimiter.tokens + tokensToAdd,
            );
            rateLimiter.lastRefill = now;
        }

        // Check if we have tokens
        if (rateLimiter.tokens > 0) {
            rateLimiter.tokens--;
            return true;
        }

        // Try burst tokens
        if (rateLimiter.burstTokens > 0) {
            rateLimiter.burstTokens--;
            return true;
        }

        // Rate limited
        const metrics = this.subscriptionMetrics.get(subscriptionId);
        if (metrics) {
            metrics.rateLimitHits++;
        }
        this.globalMetrics.rateLimitHits++;

        return false;
    }

    private async deliverNotification(
        subscription: ContextSubscription,
        event: ContextUpdateEvent,
    ): Promise<void> {
        if (this.config.optimization.batchNotifications) {
            await this.addToBatch(subscription.id, event);
        } else {
            await this.deliverImmediateNotification(subscription, event);
        }
    }

    private async addToBatch(subscriptionId: string, event: ContextUpdateEvent): Promise<void> {
        let batch = this.notificationBatches.get(subscriptionId);

        if (!batch) {
            // Create new batch
            batch = {
                subscriptionId,
                events: [event],
                scheduledAt: new Date(),
                timeoutHandle: setTimeout(
                    () => this.flushBatch(subscriptionId),
                    this.config.optimization.batchTimeoutMs,
                ),
            };
            this.notificationBatches.set(subscriptionId, batch);
        } else {
            // Add to existing batch (check bounds to prevent memory leak)
            if (batch.events.length < this.config.optimization.batchSize * 2) {
                batch.events.push(event);
            } else {
                // Force flush if batch is growing too large
                logger.warn("[ContextSubscriptionManager] Forced batch flush due to size limit", {
                    subscriptionId,
                    eventCount: batch.events.length,
                });
                clearTimeout(batch.timeoutHandle);
                await this.flushBatch(subscriptionId);

                // Start new batch with this event
                const newBatch: NotificationBatch = {
                    subscriptionId,
                    events: [event],
                    scheduledAt: new Date(),
                    timeoutHandle: setTimeout(
                        () => this.flushBatch(subscriptionId),
                        this.config.optimization.batchTimeoutMs,
                    ),
                };
                this.notificationBatches.set(subscriptionId, newBatch);
                return;
            }

            // Flush if batch is full
            if (batch.events.length >= this.config.optimization.batchSize) {
                clearTimeout(batch.timeoutHandle);
                await this.flushBatch(subscriptionId);
            }
        }
    }

    private async flushBatch(subscriptionId: string): Promise<void> {
        const batch = this.notificationBatches.get(subscriptionId);
        if (!batch) {
            return;
        }

        this.notificationBatches.delete(subscriptionId);

        const subscription = this.subscriptions.get(subscriptionId);
        if (!subscription) {
            return;
        }

        try {
            // Deliver all events in batch
            for (const event of batch.events) {
                await this.deliverImmediateNotification(subscription, event);
            }
        } catch (error) {
            await this.handleSubscriptionError(subscriptionId, error);
        }
    }

    private async deliverImmediateNotification(
        subscription: ContextSubscription,
        event: ContextUpdateEvent,
    ): Promise<void> {
        const startTime = Date.now();

        try {
            await subscription.handler(event);

            // Update metrics on success
            const metrics = this.subscriptionMetrics.get(subscription.id);
            if (metrics) {
                metrics.successfulNotifications++;
                metrics.totalNotifications++;
                metrics.lastNotificationAt = new Date();
                metrics.consecutiveFailures = 0;
                metrics.isHealthy = true;

                const processingTime = Date.now() - startTime;
                metrics.averageProcessingTimeMs =
                    (metrics.averageProcessingTimeMs * (metrics.totalNotifications - 1) + processingTime) /
                    metrics.totalNotifications;
            }

            // Update subscription metadata
            subscription.metadata.totalNotifications++;
            subscription.metadata.lastNotified = new Date();

        } catch (error) {
            await this.handleSubscriptionError(subscription.id, error);
            throw error;
        }
    }

    private async handleSubscriptionError(subscriptionId: string, error: unknown): Promise<void> {
        const metrics = this.subscriptionMetrics.get(subscriptionId);
        if (!metrics) {
            return;
        }

        metrics.failedNotifications++;
        metrics.totalNotifications++;
        metrics.consecutiveFailures++;
        metrics.lastError = error instanceof Error ? error.message : String(error);

        // Mark as unhealthy if too many consecutive failures
        if (metrics.consecutiveFailures >= this.config.healthCheck.maxConsecutiveFailures) {
            metrics.isHealthy = false;

            logger.warn("[ContextSubscriptionManager] Subscription marked unhealthy", {
                subscriptionId,
                swarmId: metrics.swarmId,
                subscriberId: metrics.subscriberId,
                consecutiveFailures: metrics.consecutiveFailures,
                lastError: metrics.lastError,
            });
        }

        this.globalMetrics.failedNotifications++;

        logger.error("[ContextSubscriptionManager] Subscription notification failed", {
            subscriptionId,
            swarmId: metrics.swarmId,
            error: metrics.lastError,
        });
    }

    private startHealthMonitoring(): void {
        // Health check interval
        this.healthCheckInterval = setInterval(() => {
            this.performHealthCheck();
        }, this.config.healthCheck.cleanupInterval * 1000);

        // Cleanup interval
        this.cleanupInterval = setInterval(() => {
            this.cleanupUnhealthySubscriptions();
        }, this.config.healthCheck.cleanupInterval * 1000);
    }

    private performHealthCheck(): void {
        const now = Date.now();
        let healthyCount = 0;
        let unhealthyCount = 0;

        for (const metrics of this.subscriptionMetrics.values()) {
            if (metrics.isHealthy) {
                healthyCount++;
            } else {
                unhealthyCount++;
            }

            // Reset per-second counters
            metrics.notificationsInLastSecond = 0;

            // Restore burst tokens only if depleted (prevent memory leak)
            const rateLimiter = this.rateLimiters.get(metrics.subscriptionId);
            if (rateLimiter && rateLimiter.burstTokens < rateLimiter.tokens) {
                const subscription = this.subscriptions.get(metrics.subscriptionId);
                const rateLimit = subscription?.filter?.rateLimit || this.config.defaultRateLimit;
                // Only restore up to the configured allowance to prevent memory leak
                rateLimiter.burstTokens = Math.min(rateLimit.burstAllowance, rateLimiter.burstTokens + 1);
            }
        }

        logger.debug("[ContextSubscriptionManager] Health check completed", {
            totalSubscriptions: this.subscriptionMetrics.size,
            healthySubscriptions: healthyCount,
            unhealthySubscriptions: unhealthyCount,
            globalMetrics: this.globalMetrics,
        });
    }

    private async checkChannelUnsubscription(swarmId: string): Promise<void> {
        // Check if there are any remaining subscriptions for this swarm
        const hasSubscriptionsForSwarm = Array.from(this.subscriptions.values())
            .some(sub => sub.swarmId === swarmId);

        if (!hasSubscriptionsForSwarm && this.subscriberClient) {
            // No more subscriptions for this swarm, unsubscribe from Redis channel
            const channel = `${this.config.channelPrefix}${swarmId}`;

            try {
                await this.subscriberClient.unsubscribe(channel);
                this.subscribedChannels.delete(channel);

                logger.debug("[ContextSubscriptionManager] Unsubscribed from Redis channel", {
                    channel,
                    swarmId,
                    remainingChannels: this.subscribedChannels.size,
                });
            } catch (error) {
                logger.error("[ContextSubscriptionManager] Failed to unsubscribe from Redis channel", {
                    channel,
                    swarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
    }

    private cleanupUnhealthySubscriptions(): void {
        const now = Date.now();
        const unhealthyThreshold = this.config.healthCheck.unhealthySubscriptionTTL * 1000;

        const subscriptionsToRemove: string[] = [];

        for (const [subscriptionId, metrics] of this.subscriptionMetrics.entries()) {
            if (!metrics.isHealthy && metrics.lastNotificationAt) {
                const timeSinceLastNotification = now - metrics.lastNotificationAt.getTime();

                if (timeSinceLastNotification > unhealthyThreshold) {
                    logger.warn("[ContextSubscriptionManager] Removing unhealthy subscription", {
                        subscriptionId,
                        swarmId: metrics.swarmId,
                        subscriberId: metrics.subscriberId,
                        timeSinceLastNotification,
                        consecutiveFailures: metrics.consecutiveFailures,
                    });

                    subscriptionsToRemove.push(subscriptionId);
                }
            }
        }

        // Remove unhealthy subscriptions after iteration to avoid modification during iteration
        for (const subscriptionId of subscriptionsToRemove) {
            this.removeSubscription(subscriptionId);
        }
    }
}
