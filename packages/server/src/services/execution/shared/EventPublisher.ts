import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { type Logger } from "winston";
import { nanoid, type ExecutionEvent } from "@vrooli/shared";

export interface EventPublisherConfig {
    /** Maximum retry attempts for failed publishes */
    maxRetries?: number;
    /** Base delay in ms between retries */
    retryDelay?: number;
    /** Whether to use exponential backoff for retries */
    exponentialBackoff?: boolean;
    /** Whether to log all published events (useful for debugging) */
    logAllEvents?: boolean;
}

export interface PublishOptions {
    /** Override retry count for this specific publish */
    retries?: number;
    /** Whether to throw on failure or silently fail */
    throwOnError?: boolean;
    /** Additional metadata to merge with event */
    metadata?: Record<string, any>;
}

/**
 * Centralized event publishing utility that handles:
 * - Consistent event structure
 * - Automatic retry logic with exponential backoff
 * - Error handling and logging
 * - Event metadata enrichment
 */
export class EventPublisher {
    private readonly config: Required<EventPublisherConfig>;

    constructor(
        private readonly eventBus: EventBus,
        private readonly logger: Logger,
        private readonly source: string,
        config: EventPublisherConfig = {},
    ) {
        this.config = {
            maxRetries: config.maxRetries ?? 3,
            retryDelay: config.retryDelay ?? 100,
            exponentialBackoff: config.exponentialBackoff ?? true,
            logAllEvents: config.logAllEvents ?? false,
        };
    }

    /**
     * Publish an event with automatic retry and error handling
     */
    async publish<T = any>(
        eventType: string,
        payload: T,
        options: PublishOptions = {},
    ): Promise<void> {
        const event = this.createEvent(eventType, payload, options.metadata);
        const maxRetries = options.retries ?? this.config.maxRetries;

        if (this.config.logAllEvents) {
            this.logger.debug(`[EventPublisher] Publishing event`, {
                source: this.source,
                eventType,
                eventId: event.id,
            });
        }

        let lastError: Error | undefined;
        
        for (let attempt = 0; attempt <= maxRetries; attempt++) {
            try {
                await this.eventBus.publish(event);
                
                if (attempt > 0) {
                    this.logger.info(`[EventPublisher] Event published after ${attempt} retries`, {
                        source: this.source,
                        eventType,
                        eventId: event.id,
                    });
                }
                
                return;
            } catch (error) {
                lastError = error instanceof Error ? error : new Error(String(error));
                
                if (attempt < maxRetries) {
                    const delay = this.calculateRetryDelay(attempt);
                    
                    this.logger.warn(`[EventPublisher] Event publish failed, retrying in ${delay}ms`, {
                        source: this.source,
                        eventType,
                        eventId: event.id,
                        attempt: attempt + 1,
                        maxRetries,
                        error: lastError.message,
                    });
                    
                    await this.delay(delay);
                } else {
                    this.logger.error(`[EventPublisher] Failed to publish event after ${maxRetries} retries`, {
                        source: this.source,
                        eventType,
                        eventId: event.id,
                        error: lastError.message,
                        stack: lastError.stack,
                    });
                }
            }
        }

        if (options.throwOnError !== false && lastError) {
            throw lastError;
        }
    }

    /**
     * Publish multiple events as a batch
     */
    async publishBatch(
        events: Array<{
            eventType: string;
            payload: any;
            options?: PublishOptions;
        }>,
    ): Promise<{ successful: number; failed: number }> {
        const results = await Promise.allSettled(
            events.map(({ eventType, payload, options }) =>
                this.publish(eventType, payload, { ...options, throwOnError: false }),
            ),
        );

        const successful = results.filter(r => r.status === "fulfilled").length;
        const failed = results.filter(r => r.status === "rejected").length;

        if (failed > 0) {
            this.logger.warn(`[EventPublisher] Batch publish completed with failures`, {
                source: this.source,
                successful,
                failed,
                total: events.length,
            });
        }

        return { successful, failed };
    }

    /**
     * Create a structured event with consistent format
     */
    private createEvent<T>(
        type: string,
        payload: T,
        additionalMetadata?: Record<string, any>,
    ): ExecutionEvent {
        return {
            id: nanoid(),
            type,
            timestamp: new Date(),
            source: {
                tier: "cross-cutting" as const,
                component: this.source,
                instanceId: nanoid()
            },
            correlationId: nanoid(),
            data: payload,
            metadata: {
                ...additionalMetadata,
            },
        };
    }

    /**
     * Calculate retry delay with optional exponential backoff
     */
    private calculateRetryDelay(attempt: number): number {
        if (!this.config.exponentialBackoff) {
            return this.config.retryDelay;
        }

        // Exponential backoff with jitter
        const exponentialDelay = this.config.retryDelay * Math.pow(2, attempt);
        const jitter = Math.random() * 0.1 * exponentialDelay; // 10% jitter
        
        return Math.min(exponentialDelay + jitter, 10000); // Cap at 10 seconds
    }

    /**
     * Delay helper for retry logic
     */
    private delay(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    /**
     * Create a child publisher with a specific source prefix
     */
    createChild(sourcePrefix: string, config?: EventPublisherConfig): EventPublisher {
        return new EventPublisher(
            this.eventBus,
            this.logger,
            `${this.source}.${sourcePrefix}`,
            { ...this.config, ...config },
        );
    }

    /**
     * Publish a metric event (common pattern)
     */
    async publishMetric(
        metricName: string,
        value: number,
        tags?: Record<string, string>,
    ): Promise<void> {
        await this.publish("metric.recorded", {
            name: metricName,
            value,
            tags: {
                source: this.source,
                ...tags,
            },
        });
    }

    /**
     * Publish a state change event (common pattern)
     */
    async publishStateChange<T extends string>(
        entityType: string,
        entityId: string,
        fromState: T,
        toState: T,
        context?: Record<string, any>,
    ): Promise<void> {
        await this.publish(`${entityType}.state.changed`, {
            entityType,
            entityId,
            fromState,
            toState,
            context,
        });
    }

    /**
     * Publish an error event (common pattern)
     */
    async publishError(
        operation: string,
        error: Error,
        context?: Record<string, any>,
    ): Promise<void> {
        await this.publish("error.occurred", {
            operation,
            error: {
                message: error.message,
                name: error.name,
                stack: error.stack,
            },
            context,
        }, {
            throwOnError: false, // Don't throw when publishing errors
        });
    }
}