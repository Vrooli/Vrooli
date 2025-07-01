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
import { type Logger } from "winston";
import { EventUtils, getUnifiedEventSystem, type IEventBus } from "../../events/index.js";
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
    protected readonly unifiedEventBus: IEventBus | null;
    protected readonly componentName: string;
    protected disposed = false;

    constructor(
        protected readonly eventBus: IEventBus,
        componentName?: string,
    ) {
        this.componentName = componentName || this.constructor.name;
        this.errorHandler = new ErrorHandler(this.eventPublisher).createComponentHandler(this.componentName);

        // Get unified event system for modern event publishing
        this.unifiedEventBus = getUnifiedEventSystem();
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
        this.logger.debug(`[${this.getComponentName()}] Component disposed`);
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
    protected async publishUnifiedEvent(
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
        if (!this.unifiedEventBus) {
            this.logger.debug(`[${this.componentName}] Unified event bus not available, skipping event publication`);
            return;
        }

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

            await this.unifiedEventBus.publish(event);

            this.logger.debug(`[${this.componentName}] Published unified event`, {
                eventType,
                deliveryGuarantee: options?.deliveryGuarantee,
                priority: options?.priority,
            });

        } catch (eventError) {
            this.logger.error(`[${this.componentName}] Failed to publish unified event`, {
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
        await this.publishUnifiedEvent(
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
        await this.publishUnifiedEvent(
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
        await this.publishUnifiedEvent(
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

/**
 * Factory helper for creating components with consistent initialization
 */
export class ComponentFactory {
    constructor(
        private readonly logger: Logger,
        private readonly eventBus: IEventBus,
    ) { }

    /**
     * Create a component instance with automatic initialization
     */
    async create<T extends BaseComponent>(
        ComponentClass: new (logger: Logger, eventBus: IEventBus, ...args: any[]) => T,
        ...additionalArgs: any[]
    ): Promise<T> {
        const component = new ComponentClass(this.logger, this.eventBus, ...additionalArgs);

        if (component.initialize) {
            await component.initialize();
        }

        return component;
    }

    /**
     * Create multiple components in parallel
     */
    async createBatch<T extends BaseComponent>(
        componentSpecs: Array<{
            ComponentClass: new (logger: Logger, eventBus: IEventBus, ...args: any[]) => T;
            args?: any[];
        }>,
    ): Promise<T[]> {
        const creationPromises = componentSpecs.map(spec =>
            this.create(spec.ComponentClass, ...(spec.args || [])),
        );

        return Promise.all(creationPromises);
    }

    /**
     * Create a child factory with a prefixed logger
     */
    createChild(prefix: string): ComponentFactory {
        const childLogger = this.logger.child({ component: prefix });
        return new ComponentFactory(childLogger, this.eventBus);
    }
}
