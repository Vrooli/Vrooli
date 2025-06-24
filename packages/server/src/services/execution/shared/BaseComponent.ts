/**
 * Base Component
 * 
 * Abstract base class for all execution architecture components.
 * Reduces constructor boilerplate by providing common infrastructure.
 * 
 * This follows the emergent architecture principle of providing tools, not decisions:
 * - EventPublisher for event infrastructure 
 * - ErrorHandler for error infrastructure
 * - Logger for consistent logging
 * 
 * Components decide how to use these tools based on their specific needs.
 */

import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { EventPublisher } from "./EventPublisher.js";
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
    protected readonly eventPublisher: EventPublisher;
    protected readonly errorHandler: ComponentErrorHandler;
    protected disposed = false;
    
    constructor(
        protected readonly logger: Logger,
        protected readonly eventBus: EventBus,
        componentName?: string,
    ) {
        const name = componentName || this.constructor.name;
        this.eventPublisher = new EventPublisher(eventBus, logger, name);
        this.errorHandler = new ErrorHandler(logger, this.eventPublisher).createComponentHandler(name);
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
     * Helper method for publishing events
     */
    protected async publishEvent(
        eventType: string,
        payload: any,
        metadata?: Record<string, any>,
    ): Promise<void> {
        await this.eventPublisher.publish(eventType, payload, { metadata });
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
        await this.eventPublisher.publishStateChange(
            entityType,
            entityId,
            fromState,
            toState,
            context,
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
        await this.eventPublisher.publishError(operation, error, context);
    }

    /**
     * Helper method for publishing metrics
     */
    protected async publishMetric(
        metricName: string,
        value: number,
        tags?: Record<string, string>,
    ): Promise<void> {
        await this.eventPublisher.publishMetric(metricName, value, tags);
    }
}

/**
 * Factory helper for creating components with consistent initialization
 */
export class ComponentFactory {
    constructor(
        private readonly logger: Logger,
        private readonly eventBus: EventBus,
    ) {}

    /**
     * Create a component instance with automatic initialization
     */
    async create<T extends BaseComponent>(
        ComponentClass: new (logger: Logger, eventBus: EventBus, ...args: any[]) => T,
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
            ComponentClass: new (logger: Logger, eventBus: EventBus, ...args: any[]) => T;
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
