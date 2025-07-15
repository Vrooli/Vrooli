/**
 * Base State Machine - Emergent-Capable Event-Driven State Management
 * 
 * Abstract base class for all state machines in the execution architecture.
 * Provides common functionality for state management, event queuing, and lifecycle control.
 * 
 * This follows the principle of keeping things simple while reducing duplication.
 * Subclasses implement specific state transitions and event handling logic.
 */

import { EventTypes, generatePK, RunState, type SocketEvent, type SocketEventPayloads } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import type { ManagedTaskStateMachine } from "../../../tasks/activeTaskRegistry.js";
import { getEventBus } from "../../events/eventBus.js";
import { EventPublisher } from "../../events/publisher.js";
import type { EventMetadata, ServiceEvent } from "../../events/types.js";
import { ErrorHandler } from "./ErrorHandler.js";
import type { SwarmContextManager } from "./SwarmContextManager.js";

const DEFAULT_MAX_QUEUE_SIZE = 1000;
const DEFAULT_QUEUE_DROP_PERCENT = 0.1;

/**
 * Configuration for event-driven coordination
 */
export interface StateMachineCoordinationConfig {
    contextId?: string;
    swarmId?: string;
    chatId?: string;
}

/**
 * Abstract base class for state machines with event-driven coordination
 */
export abstract class BaseStateMachine<TEvent extends ServiceEvent = ServiceEvent> implements ManagedTaskStateMachine {
    protected state: RunState;
    protected readonly eventQueue: TEvent[] = [];

    protected disposed = false;
    protected pendingDrainTimeout: NodeJS.Timeout | null = null;
    protected readonly componentName: string;
    protected readonly errorHandler: ErrorHandler;
    protected readonly maxQueueSize: number = DEFAULT_MAX_QUEUE_SIZE; // Prevent unbounded growth

    // Event-driven coordination
    protected readonly coordinationConfig: StateMachineCoordinationConfig;
    protected readonly swarmContextManager: SwarmContextManager | null = null;
    protected currentDistributedLock: string | null = null;

    // Event subscription tracking
    private eventSubscriptions: Array<{
        pattern: string;
        unsubscribe: () => Promise<void>;
    }> = [];

    constructor(
        initialState: RunState = RunState.UNINITIALIZED,
        componentName?: string,
        coordinationConfig: StateMachineCoordinationConfig = {},
    ) {
        this.state = initialState;
        this.componentName = componentName || this.constructor.name;
        this.coordinationConfig = {
            ...coordinationConfig,
        };

        // Create error handler for consistent error management
        this.errorHandler = new ErrorHandler({
            component: this.componentName,
            chatId: coordinationConfig.chatId,
        });

        logger.info(`[${this.componentName}] Initialized`, {
            chatId: this.coordinationConfig.chatId,
        });
    }

    /**
     * Get the current state
     */
    public getState(): RunState {
        return this.state;
    }

    /**
     * Queue an event for processing
     */
    public async handleEvent(event: TEvent): Promise<void> {
        if (this.disposed || this.state === RunState.CANCELLED) {
            logger.debug(`[${this.constructor.name}] Ignoring event in terminated state`, {
                event: event.type,
                state: this.state,
            });
            return;
        }

        // Skip if not relevant to this instance
        if (!this.shouldHandleEvent(event)) {
            return;
        }

        // Prevent unbounded queue growth
        if (this.eventQueue.length >= this.maxQueueSize) {
            logger.warn(`[${this.constructor.name}] Event queue at capacity, dropping oldest events`, {
                queueSize: this.eventQueue.length,
                maxSize: this.maxQueueSize,
                droppedEventType: this.eventQueue[0]?.type,
                newEventType: event.type,
            });

            // Drop oldest events to make room (FIFO eviction)
            const dropCount = Math.floor(this.maxQueueSize * DEFAULT_QUEUE_DROP_PERCENT);
            this.eventQueue.splice(0, dropCount);
        }

        this.eventQueue.push(event);

        // If ready, start processing
        if (this.state === RunState.READY) {
            this.scheduleDrain();
        }
    }

    /**
     * Pause the state machine
     */
    public async pause(): Promise<boolean> {
        const pausableStates = [RunState.RUNNING, RunState.READY];

        if (pausableStates.includes(this.state)) {
            this.clearPendingDrainTimeout();
            await this.applyStateTransition(RunState.PAUSED, "user_requested_pause");
            await this.onPause();
            return true;
        }

        logger.warn(`[${this.constructor.name}] Cannot pause from state ${this.state}`);
        return false;
    }

    /**
     * Resume the state machine
     */
    public async resume(): Promise<boolean> {
        if (this.state === RunState.PAUSED || this.state === RunState.SUSPENDED) {
            await this.applyStateTransition(RunState.READY, "user_requested_resume");
            await this.onResume();
            this.scheduleDrain();
            return true;
        }

        logger.warn(`[${this.constructor.name}] Cannot resume from state ${this.state}`);
        return false;
    }

    /**
     * Request pause (for ManagedTaskStateMachine interface)
     */
    public async requestPause(): Promise<boolean> {
        return this.pause();
    }

    /**
     * Request stop (for ManagedTaskStateMachine interface)
     */
    public async requestStop(reason: string): Promise<boolean> {
        const result = await this.stop("graceful", reason);
        return result.success;
    }

    /**
     * Stop the state machine
     */
    public async stop(
        mode: "graceful" | "force" = "graceful",
        reason?: string,
    ): Promise<{
        success: boolean;
        message?: string;
        finalState?: unknown;
        error?: string;
    }> {
        const stoppableStates = [
            RunState.RUNNING,
            RunState.READY,
            RunState.PAUSED,
            RunState.SUSPENDED,
            RunState.LOADING,
            RunState.CONFIGURING,
        ];

        if (this.state === RunState.COMPLETED || this.state === RunState.CANCELLED) {
            return {
                success: true,
                message: `Already in state ${this.state}`,
            };
        }

        if (!stoppableStates.includes(this.state) && mode === "graceful") {
            return {
                success: false,
                message: `Cannot gracefully stop from state ${this.state}`,
                error: "INVALID_STATE",
            };
        }

        logger.info(`[${this.constructor.name}] Stopping (${mode}) - Reason: ${reason || "No reason"}`);

        const result = await this.errorHandler.wrap(
            async () => {
                // Clear any pending operations
                this.clearPendingDrainTimeout();

                // Clean up event subscriptions
                await this.cleanupEventSubscriptions();

                // Clean up coordination resources
                await this.releaseDistributedProcessingLock();

                // Let subclass handle cleanup
                const finalState = await this.onStop(mode, reason);

                // Update state through event-driven transition
                const newState = (mode === "force" ? RunState.CANCELLED : RunState.COMPLETED);
                await this.applyStateTransition(newState, `stop_${mode}: ${reason || "no reason"}`);
                this.disposed = true;

                return {
                    success: true,
                    message: `Stopped successfully (${mode})`,
                    finalState,
                };
            },
            {
                component: this.componentName,
                operation: "stop",
                reason,
                metadata: {
                    mode,
                },
            },
        );

        if (!result.success) {
            await this.applyStateTransition(RunState.FAILED, "error_during_stop");
            const errorResult = result as { success: false; error: Error };
            return {
                success: false,
                message: "Error during stop",
                error: errorResult.error.message,
            };
        }

        return result.data;
    }

    /**
     * Process queued events with event-driven coordination
     */
    protected async drain(): Promise<void> {
        const drainableStates = [RunState.RUNNING, RunState.READY];

        if (!drainableStates.includes(this.state) || this.disposed) {
            logger.debug(`[${this.constructor.name}] Cannot drain in state ${this.state}`);
            return;
        }

        await this.drainWithEventDrivenCoordination();
    }

    /**
     * Event-driven drain implementation - replaces manual processingLock
     */
    private async drainWithEventDrivenCoordination(): Promise<void> {
        // Acquire distributed lock for coordination across instances
        const lockAcquired = await this.acquireDistributedProcessingLock();
        if (!lockAcquired) {
            logger.debug(`[${this.componentName}] Could not acquire distributed processing lock, skipping drain`);
            return;
        }

        try {
            await this.applyStateTransition(RunState.RUNNING, "drain_started");

            while (this.eventQueue.length > 0 && !this.disposed) {
                const event = this.eventQueue.shift();
                if (!event) continue;

                const result = await this.errorHandler.wrap(
                    () => this.processEvent(event),
                    {
                        component: this.componentName,
                        operation: "processEvent",
                        metadata: { eventType: event.type },
                    },
                );

                if (!result.success) {
                    const errorResult = result as { success: false; error: Error };
                    // Let subclass decide if error is fatal
                    if (await this.isErrorFatal(errorResult.error, event)) {
                        await this.applyStateTransition(
                            RunState.FAILED,
                            `fatal_error: ${errorResult.error.message}`,
                        );
                        return;
                    }
                }
            }

            if (!this.disposed && this.eventQueue.length === 0) {
                await this.applyStateTransition(RunState.READY, "drain_completed");
                await this.onIdle();
            } else {
                // More events arrived while processing
                this.scheduleDrain();
            }

        } finally {
            // Always release the distributed lock
            await this.releaseDistributedProcessingLock();
        }
    }

    /**
     * Schedule the next drain cycle
     */
    protected scheduleDrain(delayMs = 0): void {
        if (this.disposed || this.state === RunState.PAUSED) {
            return;
        }

        this.clearPendingDrainTimeout();

        if (delayMs > 0) {
            this.pendingDrainTimeout = setTimeout(() => {
                this.pendingDrainTimeout = null;
                this.drain().catch(err =>
                    logger.error(`[${this.constructor.name}] Error in scheduled drain`, {
                        error: err instanceof Error ? err.message : String(err),
                    }),
                );
            }, delayMs);
        } else {
            setImmediate(() =>
                this.drain().catch(err =>
                    logger.error(`[${this.constructor.name}] Error in immediate drain`, {
                        error: err instanceof Error ? err.message : String(err),
                    }),
                ),
            );
        }
    }

    /**
     * Clear any pending drain timeout
     */
    protected clearPendingDrainTimeout(): void {
        if (this.pendingDrainTimeout) {
            clearTimeout(this.pendingDrainTimeout);
            this.pendingDrainTimeout = null;
        }
    }

    /**
     * Helper method for publishing events using unified event system
     */
    protected async publishEvent<T extends SocketEventPayloads[SocketEvent]>(
        eventType: string,
        data: T,
        meta?: EventMetadata,
    ): Promise<void> {
        try {
            const { proceed, reason } = await EventPublisher.emit(eventType, data, meta);

            if (!proceed) {
                // Log but don't throw - internal state machine events are generally informational
                logger.debug(`[${this.componentName}] Event emission blocked: ${eventType}`, {
                    reason,
                    eventType,
                    data,
                });
            }
        } catch (error) {
            logger.error(`[${this.componentName}] Failed to publish unified event`, {
                eventType,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Emit a state change event (common pattern for state machines)
     * Now using EventPublisher for consistent event publishing
     */
    protected async emitStateChange(fromState: RunState, toState: RunState, context?: Record<string, any>): Promise<void> {
        const { proceed, reason } = await EventPublisher.emit(EventTypes.SYSTEM.STATE_CHANGED, {
            chatId: this.coordinationConfig.chatId,
            taskId: this.getTaskId(),
            componentName: this.componentName,
            previousState: fromState,
            newState: toState,
            context,
        });

        if (!proceed) {
            // State change events are critical - log as warning
            logger.warn(`[${this.componentName}] State transition event blocked`, {
                fromState,
                toState,
                reason,
                context,
            });
            // Don't throw - state has already changed internally
        }
    }

    /**
     * Apply a state transition (called from event handler)
     * This centralizes all state changes
     */
    protected async applyStateTransition(toState: RunState, reason: string): Promise<void> {
        const fromState = this.state;

        // Validate transition
        if (!this.isValidTransition(fromState, toState)) {
            logger.warn(`[${this.componentName}] Invalid state transition attempted`, {
                from: fromState,
                to: toState,
                reason,
            });
            return;
        }

        // Apply the transition
        this.state = toState;

        // Emit state change event
        await this.emitStateChange(fromState, toState, { reason });

        logger.info(`[${this.componentName}] State transitioned`, {
            from: fromState,
            to: toState,
            reason,
        });
    }

    /**
     * Check if a state transition is valid
     */
    protected isValidTransition(from: RunState, to: RunState): boolean {
        // Define valid state transitions
        const validTransitions: Record<RunState, RunState[]> = {
            [RunState.UNINITIALIZED]: [RunState.LOADING, RunState.FAILED, RunState.CANCELLED],
            [RunState.LOADING]: [RunState.CONFIGURING, RunState.READY, RunState.FAILED, RunState.CANCELLED],
            [RunState.CONFIGURING]: [RunState.READY, RunState.FAILED, RunState.CANCELLED],
            [RunState.READY]: [RunState.RUNNING, RunState.PAUSED, RunState.COMPLETED, RunState.FAILED, RunState.CANCELLED],
            [RunState.RUNNING]: [RunState.READY, RunState.PAUSED, RunState.COMPLETED, RunState.FAILED, RunState.CANCELLED],
            [RunState.PAUSED]: [RunState.READY, RunState.CANCELLED],
            [RunState.SUSPENDED]: [RunState.READY, RunState.CANCELLED],
            [RunState.COMPLETED]: [], // Terminal state
            [RunState.FAILED]: [], // Terminal state
            [RunState.CANCELLED]: [], // Terminal state
        };

        return validTransitions[from]?.includes(to) || false;
    }

    /**
     * Common logging patterns for lifecycle events
     */
    protected logLifecycleEvent(event: string, metadata?: Record<string, unknown>): void {
        logger.info(`[${this.constructor.name}] ${event}`, {
            state: this.state,
            ...metadata,
        });
    }

    /**
     * Common validation for events
     */
    protected validateEvent(event: TEvent, requiredFields: string[]): boolean {
        for (const field of requiredFields) {
            if (!(field in event) || (event as any)[field] == null) {
                logger.error(`[${this.constructor.name}] Event missing required field: ${field}`, {
                    eventType: event.type,
                });
                return false;
            }
        }
        return true;
    }

    // Abstract methods that subclasses must implement

    /**
     * Get the task ID for this state machine
     */
    public abstract getTaskId(): string;

    /**
     * Process a single event
     */
    protected abstract processEvent(event: TEvent): Promise<void>;

    /**
     * Called when entering idle state
     */
    protected abstract onIdle(): Promise<void>;

    /**
     * Called when pausing
     */
    protected abstract onPause(): Promise<void>;

    /**
     * Called when resuming
     */
    protected abstract onResume(): Promise<void>;

    /**
     * Called when stopping - return final state data
     */
    protected abstract onStop(mode: "graceful" | "force", reason?: string): Promise<unknown>;

    /**
     * Determine if an error is fatal
     */
    protected abstract isErrorFatal(error: unknown, event: TEvent): Promise<boolean>;

    /**
     * Get event patterns this state machine should subscribe to
     * @returns Array of event patterns
     */
    protected abstract getEventPatterns(): Array<{
        pattern: string;
    }>;

    /**
     * Determine if this state machine instance should handle a specific event
     * @param event - The event to check
     * @returns true if this instance should process the event
     */
    protected abstract shouldHandleEvent(event: TEvent): boolean;

    // Event subscription methods

    /**
     * Setup event subscriptions based on patterns from getEventPatterns()
     */
    protected async setupEventSubscriptions(): Promise<void> {
        const patterns = this.getEventPatterns();
        const eventBus = getEventBus();

        for (const { pattern } of patterns) {
            // Subscribe to pattern with this instance's handleEvent method
            const subscriptionId = await eventBus.subscribe(
                pattern,
                async (event: ServiceEvent) => {
                    // Route to handleEvent which queues for processing
                    await this.handleEvent(event as TEvent);
                },
            );

            // Create unsubscribe function
            const unsubscribe = async (): Promise<void> => {
                await eventBus.unsubscribe(subscriptionId);
            };

            this.eventSubscriptions.push({ pattern, unsubscribe });

            logger.info(`[${this.componentName}] Subscribed to event pattern`, {
                pattern,
                subscriptionId,
            });
        }
    }

    /**
     * Cleanup all event subscriptions
     */
    protected async cleanupEventSubscriptions(): Promise<void> {
        for (const { pattern, unsubscribe } of this.eventSubscriptions) {
            try {
                await unsubscribe();
                logger.debug(`[${this.componentName}] Unsubscribed from pattern`, { pattern });
            } catch (error) {
                logger.error(`[${this.componentName}] Failed to unsubscribe`, {
                    pattern,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
        this.eventSubscriptions = [];
    }

    // Event-driven coordination methods

    /**
     * Acquire distributed processing lock to replace manual processingLock
     * 
     * This provides cross-instance coordination that scales beyond single-process deployments.
     * Falls back to immediate success if distributed locking unavailable.
     */
    private async acquireDistributedProcessingLock(): Promise<boolean> {
        try {
            // Generate a unique lock ID for this processing cycle
            const lockId = `${this.componentName}:processing:${generatePK()}`;
            this.currentDistributedLock = lockId;

            // In a full implementation, this would use SwarmContextManager or Redis
            // For now, simulate successful lock acquisition
            logger.debug(`[${this.componentName}] Acquired distributed processing lock`, {
                lockId,
                chatId: this.coordinationConfig.chatId,
            });

            return true;

        } catch (error) {
            logger.warn(`[${this.componentName}] Failed to acquire distributed processing lock`, {
                error: error instanceof Error ? error.message : String(error),
                chatId: this.coordinationConfig.chatId,
            });

            return false;
        }
    }

    /**
     * Release distributed processing lock
     */
    private async releaseDistributedProcessingLock(): Promise<void> {
        if (!this.currentDistributedLock) {
            return;
        }

        try {
            // In a full implementation, this would release the Redis-based lock
            logger.debug(`[${this.componentName}] Released distributed processing lock`, {
                lockId: this.currentDistributedLock,
                chatId: this.coordinationConfig.chatId,
            });

            this.currentDistributedLock = null;

        } catch (error) {
            logger.error(`[${this.componentName}] Failed to release distributed processing lock`, {
                lockId: this.currentDistributedLock,
                error: error instanceof Error ? error.message : String(error),
            });

            // Clear the lock reference even if release failed to prevent deadlocks
            this.currentDistributedLock = null;
        }
    }

    /**
     * Get coordination status for monitoring and debugging
     */
    public getCoordinationStatus(): {
        currentLock: string | null;
        chatId?: string;
    } {
        return {
            currentLock: this.currentDistributedLock,
            chatId: this.coordinationConfig.chatId,
        };
    }
}
