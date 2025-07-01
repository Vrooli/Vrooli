/**
 * Base Component
 * 
 * Abstract base class for all execution architecture components.
 * Reduces constructor boilerplate by providing common infrastructure.
 * 
 * This follows the emergent architecture principle of providing tools, not decisions:
 * - ErrorHandler for error infrastructure
 * - Logger for consistent logging
 * 
 * Components decide how to use these tools based on their specific needs.
 */

import { nanoid } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { getEventBus } from "../../events/eventBus.js";
import { EventUtils } from "../../events/index.js";
import { ErrorHandler, type ComponentErrorHandler } from "./ErrorHandler.js";

/**
 * Base component interface
 */
export interface IBaseComponent {
    /** Get component name for identification */
    getComponentName(): string;
    /** Initialize component (called after construction) */
    initialize?(): Promise<void>;
    /** Cleanup component resources */
    dispose?(): Promise<void>;
}

/**
 * Abstract base class for components
 */
export abstract class BaseComponent implements IBaseComponent {
    protected readonly errorHandler: ComponentErrorHandler;
    protected readonly componentName: string;
    protected disposed = false;

    constructor(
        componentName?: string,
    ) {
        this.componentName = componentName || this.constructor.name;
        this.errorHandler = new ErrorHandler().createComponentHandler(this.componentName);
    }

    /**
     * Get component name for identification
     */
    public getComponentName(): string {
        return this.constructor.name;
    }

    /**
     * Initialize component (override if needed)
     */
    public async initialize(): Promise<void> {
        // Default implementation - no initialization needed
    }

    /**
     * Cleanup component resources (override if needed)
     */
    public async dispose(): Promise<void> {
        if (this.disposed) {
            return;
        }

        this.disposed = true;
        logger.debug(`[${this.getComponentName()}] Component disposed`);
    }

    /**
     * Check if component is disposed
     */
    protected isDisposed(): boolean {
        return this.disposed;
    }

    /**
     * Helper method for safe async operations with error handling
     */
    protected async executeWithErrorHandling<T>(
        operation: () => Promise<T>,
        operationName: string,
        context?: Record<string, any>,
    ): Promise<T> {
        return this.errorHandler.execute(operation, {
            operation: operationName,
            component: this.getComponentName(),
            metadata: context,
        });
    }

    /**
     * Helper method for publishing events using unified event system
     */
    protected async publishEvent(
        eventType: string,
        data: any,
        options?: {
            deliveryGuarantee?: "fire-and-forget" | "reliable" | "barrier-sync";
            priority?: "low" | "medium" | "high" | "critical";
            tags?: string[];
            userId?: string;
            conversationId?: string;
        },
    ): Promise<void> {
        try {
            const event = EventUtils.createBaseEvent(
                eventType,
                data,
                EventUtils.createEventSource("cross-cutting", this.componentName, nanoid()),
                EventUtils.createEventMetadata(
                    options?.deliveryGuarantee || "fire-and-forget",
                    options?.priority || "medium",
                    {
                        tags: options?.tags,
                        userId: options?.userId,
                        conversationId: options?.conversationId,
                    },
                ),
            );

            await getEventBus().publish(event);

            logger.debug(`[${this.componentName}] Published unified event`, {
                eventType,
                deliveryGuarantee: options?.deliveryGuarantee,
                priority: options?.priority,
            });

        } catch (eventError) {
            logger.error(`[${this.componentName}] Failed to publish unified event`, {
                eventType,
                error: eventError instanceof Error ? eventError.message : String(eventError),
            });
        }
    }

    /**
     * Helper method for publishing state changes
     */
    protected async publishStateChange<T extends string>(
        entityType: string,
        entityId: string,
        fromState: T,
        toState: T,
        context?: Record<string, any>,
    ): Promise<void> {
        await this.publishEvent(
            `${entityType}.state.changed`,
            {
                entityId,
                entityType,
                previousState: fromState,
                newState: toState,
                ...context,
            },
            {
                priority: "medium",
                deliveryGuarantee: "reliable",
            },
        );
    }

    /**
     * Helper method for publishing errors
     */
    protected async publishError(
        operation: string,
        error: Error,
        context?: Record<string, any>,
    ): Promise<void> {
        await this.publishEvent(
            "component.error",
            {
                component: this.componentName,
                operation,
                error: {
                    name: error.name,
                    message: error.message,
                    stack: error.stack,
                },
                ...context,
            },
            {
                priority: "high",
                deliveryGuarantee: "reliable",
            },
        );
    }

    /**
     * Helper method for publishing metrics
     */
    protected async publishMetric(
        metricName: string,
        value: number,
        tags?: Record<string, string>,
    ): Promise<void> {
        await this.publishEvent(
            "component.metric",
            {
                component: this.componentName,
                metricName,
                value,
                timestamp: Date.now(),
                tags,
            },
            {
                priority: "low",
                deliveryGuarantee: "fire-and-forget",
            },
        );
    }
}
