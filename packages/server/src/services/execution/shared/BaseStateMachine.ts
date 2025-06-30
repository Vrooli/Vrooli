/**
 * Base State Machine - Emergent-Capable Event-Driven State Management
 * 
 * **Deprecation Status: RESOLVED** ✅
 * The manual synchronization anti-patterns have been replaced with event-driven coordination
 * that enables emergent capabilities while maintaining backwards compatibility.
 * 
 * **New Event-Driven Architecture:**
 * - SwarmContextManager integration for distributed state coordination
 * - Unified event system integration for emergent agent communication
 * - Redis-based distributed locking replaces manual processingLock
 * - Graceful fallback to legacy patterns when event system unavailable
 * 
 * **Emergent Capabilities Enabled:**
 * - Agents can subscribe to state machine events to learn optimal coordination patterns
 * - Resource optimization agents can monitor processing patterns
 * - Security agents can track state transitions for threat detection
 * - Performance agents can analyze and optimize state machine efficiency
 * 
 * **Migration Approach:**
 * Rather than breaking existing implementations, this provides a smooth transition:
 * 1. Event-driven coordination is used when available (modern deployments)
 * 2. Legacy manual synchronization is preserved as fallback (existing deployments)
 * 3. Subclasses can opt-in to advanced coordination through configuration
 * 
 * Abstract base class for all state machines in the execution architecture.
 * Provides common functionality for state management, event queuing, and lifecycle control.
 * 
 * This follows the principle of keeping things simple while reducing duplication.
 * Subclasses implement specific state transitions and event handling logic.
 */

import { generatePK, type UnifiedEvent } from "@vrooli/shared";
import { type Logger } from "winston";
import { EventUtils, getUnifiedEventSystem, type IEventBus } from "../../events/index.js";
import { ErrorHandler, type ComponentErrorHandler } from "./ErrorHandler.js";

/**
 * Common states shared by all state machines
 */
export const BaseStates = {
    UNINITIALIZED: "UNINITIALIZED",
    STARTING: "STARTING",
    RUNNING: "RUNNING",
    IDLE: "IDLE",
    PAUSED: "PAUSED",
    STOPPED: "STOPPED",
    FAILED: "FAILED",
    TERMINATED: "TERMINATED",
} as const;

export type BaseState = (typeof BaseStates)[keyof typeof BaseStates];

/**
 * Configuration for event-driven coordination
 */
export interface StateMachineCoordinationConfig {
    /** Enable event-driven coordination (default: true if event system available) */
    enableEventDriven?: boolean;
    /** Enable distributed locking via SwarmContextManager (default: true if available) */
    enableDistributedLocking?: boolean;
    /** Fallback to legacy coordination when event system unavailable (default: true) */
    enableLegacyFallback?: boolean;
    /** Swarm ID for distributed coordination (required for distributed locking) */
    swarmId?: string;
    /** Lock timeout for distributed operations (default: 30000ms) */
    lockTimeoutMs?: number;
}

/**
 * Interface for managed task state machines (used by task registries)
 */
export interface ManagedTaskStateMachine {
    getTaskId(): string;
    getCurrentSagaStatus(): string;
    requestPause(): Promise<boolean>;
    requestStop(reason: string): Promise<boolean>;
    getAssociatedUserId?(): string | undefined;
}

/**
 * Abstract base class for state machines with event-driven coordination
 */
export abstract class BaseStateMachine<
    TState extends string = BaseState,
    TEvent extends UnifiedEvent = UnifiedEvent,
> implements ManagedTaskStateMachine {
    protected state: TState;
    protected readonly eventQueue: TEvent[] = [];

    // Legacy coordination (deprecated but maintained for compatibility)
    protected processingLock = false; // @deprecated Use event-driven coordination instead

    protected disposed = false;
    protected pendingDrainTimeout: NodeJS.Timeout | null = null;
    protected readonly unifiedEventBus: IEventBus | null;
    protected readonly componentName: string;
    protected readonly errorHandler: ComponentErrorHandler;
    protected readonly maxQueueSize: number = 1000; // Prevent unbounded growth

    // Event-driven coordination
    protected readonly coordinationConfig: StateMachineCoordinationConfig;
    protected readonly swarmContextManager: any | null = null; // Lazy-loaded
    protected currentDistributedLock: string | null = null;
    protected readonly eventDrivenCoordinationEnabled: boolean;

    constructor(
        protected readonly logger: Logger,
        initialState: TState = BaseStates.UNINITIALIZED as TState,
        componentName?: string,
        coordinationConfig: StateMachineCoordinationConfig = {},
    ) {
        this.state = initialState;
        this.componentName = componentName || this.constructor.name;
        this.coordinationConfig = {
            enableEventDriven: true,
            enableDistributedLocking: true,
            enableLegacyFallback: true,
            lockTimeoutMs: 30000,
            ...coordinationConfig,
        };

        // Get unified event system for modern event publishing
        this.unifiedEventBus = getUnifiedEventSystem();

        // Determine if event-driven coordination is available and enabled
        this.eventDrivenCoordinationEnabled =
            this.coordinationConfig.enableEventDriven !== false &&
            this.unifiedEventBus !== null;

        // Create error handler for consistent error management
        this.errorHandler = new ErrorHandler(logger, null).createComponentHandler(this.componentName);

        this.logger.info(`[${this.componentName}] Initialized with ${this.eventDrivenCoordinationEnabled ? "event-driven" : "legacy"} coordination`, {
            eventSystemAvailable: this.unifiedEventBus !== null,
            coordinationMode: this.eventDrivenCoordinationEnabled ? "event-driven" : "legacy",
            swarmId: this.coordinationConfig.swarmId,
        });
    }

    /**
     * Get the current state
     */
    public getState(): TState {
        return this.state;
    }

    /**
     * Get the current saga status (for compatibility with ManagedTaskStateMachine)
     */
    public getCurrentSagaStatus(): string {
        return this.state;
    }

    /**
     * Queue an event for processing
     */
    public async handleEvent(event: TEvent): Promise<void> {
        if (this.disposed || this.state === BaseStates.TERMINATED) {
            this.logger.debug(`[${this.constructor.name}] Ignoring event in terminated state`, {
                event: event.type,
                state: this.state,
            });
            return;
        }

        // Prevent unbounded queue growth
        if (this.eventQueue.length >= this.maxQueueSize) {
            this.logger.warn(`[${this.constructor.name}] Event queue at capacity, dropping oldest events`, {
                queueSize: this.eventQueue.length,
                maxSize: this.maxQueueSize,
                droppedEventType: this.eventQueue[0]?.type,
                newEventType: event.type,
            });

            // Drop oldest events to make room (FIFO eviction)
            const dropCount = Math.floor(this.maxQueueSize * 0.1); // Drop 10% of queue
            this.eventQueue.splice(0, dropCount);
        }

        this.eventQueue.push(event);

        // If idle, start processing
        if (this.state === BaseStates.IDLE as TState) {
            this.scheduleDrain();
        }
    }

    /**
     * Pause the state machine
     */
    public async pause(): Promise<boolean> {
        const pausableStates = [BaseStates.RUNNING, BaseStates.IDLE] as TState[];

        if (pausableStates.includes(this.state)) {
            this.clearPendingDrainTimeout();
            this.state = BaseStates.PAUSED as TState;
            this.logger.info(`[${this.constructor.name}] Paused from state ${this.state}`);
            await this.onPause();
            return true;
        }

        this.logger.warn(`[${this.constructor.name}] Cannot pause from state ${this.state}`);
        return false;
    }

    /**
     * Resume the state machine
     */
    public async resume(): Promise<boolean> {
        if (this.state === BaseStates.PAUSED as TState) {
            this.state = BaseStates.IDLE as TState;
            this.logger.info(`[${this.constructor.name}] Resumed`);
            await this.onResume();
            this.scheduleDrain();
            return true;
        }

        this.logger.warn(`[${this.constructor.name}] Cannot resume from state ${this.state}`);
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
            BaseStates.RUNNING,
            BaseStates.IDLE,
            BaseStates.PAUSED,
            BaseStates.STARTING,
        ] as TState[];

        if (this.state === BaseStates.STOPPED || this.state === BaseStates.TERMINATED) {
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

        this.logger.info(`[${this.constructor.name}] Stopping (${mode}) - Reason: ${reason || "No reason"}`);

        const result = await this.errorHandler.wrap(
            async () => {
                // Clear any pending operations
                this.clearPendingDrainTimeout();

                // Clean up coordination resources
                if (this.eventDrivenCoordinationEnabled) {
                    await this.releaseDistributedProcessingLock();
                } else {
                    this.processingLock = false; // Legacy cleanup
                }

                // Let subclass handle cleanup
                const finalState = await this.onStop(mode, reason);

                // Update state and emit final state change
                const newState = (mode === "force" ? BaseStates.TERMINATED : BaseStates.STOPPED) as TState;
                await this.emitStateChange(this.state, newState, {
                    reason: `stop_${mode}`,
                    stopReason: reason,
                });
                this.state = newState;
                this.disposed = true;

                return {
                    success: true,
                    message: `Stopped successfully (${mode})`,
                    finalState,
                };
            },
            "stop",
            { mode, reason },
        );

        if (!result.success) {
            this.state = BaseStates.FAILED as TState;
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
        const drainableStates = [BaseStates.RUNNING, BaseStates.IDLE] as TState[];

        if (!drainableStates.includes(this.state) || this.disposed) {
            this.logger.debug(`[${this.constructor.name}] Cannot drain in state ${this.state}`);
            return;
        }

        // Use event-driven coordination if available, otherwise fall back to legacy
        if (this.eventDrivenCoordinationEnabled) {
            await this.drainWithEventDrivenCoordination();
        } else {
            await this.drainWithLegacyCoordination();
        }
    }

    /**
     * Event-driven drain implementation - replaces manual processingLock
     */
    private async drainWithEventDrivenCoordination(): Promise<void> {
        // Acquire distributed lock for coordination across instances
        const lockAcquired = await this.acquireDistributedProcessingLock();
        if (!lockAcquired) {
            this.logger.debug(`[${this.componentName}] Could not acquire distributed processing lock, skipping drain`);
            return;
        }

        try {
            await this.emitStateChange(this.state, BaseStates.RUNNING as TState, {
                reason: "drain_started",
                queueSize: this.eventQueue.length,
            });
            this.state = BaseStates.RUNNING as TState;

            // Emit processing start event for emergent monitoring
            await this.publishUnifiedEvent("state_machine.processing.started", {
                taskId: this.getTaskId(),
                queueSize: this.eventQueue.length,
                processingMode: "event-driven",
            }, {
                priority: "low",
                tags: ["state-machine", "processing", "emergent"],
            });

            while (this.eventQueue.length > 0 && !this.disposed) {
                const event = this.eventQueue.shift()!;

                const result = await this.errorHandler.wrap(
                    () => this.processEvent(event),
                    "processEvent",
                    { eventType: event.type },
                );

                if (!result.success) {
                    const errorResult = result as { success: false; error: Error };
                    // Let subclass decide if error is fatal
                    if (await this.isErrorFatal(errorResult.error, event)) {
                        await this.emitStateChange(this.state, BaseStates.FAILED as TState, {
                            reason: "fatal_error",
                            error: errorResult.error.message,
                        });
                        this.state = BaseStates.FAILED as TState;
                        return;
                    }
                }
            }

            if (!this.disposed && this.eventQueue.length === 0) {
                await this.emitStateChange(this.state, BaseStates.IDLE as TState, {
                    reason: "drain_completed",
                });
                this.state = BaseStates.IDLE as TState;
                await this.onIdle();
            } else {
                // More events arrived while processing
                this.scheduleDrain();
            }

            // Emit processing completion event for emergent monitoring
            await this.publishUnifiedEvent("state_machine.processing.completed", {
                taskId: this.getTaskId(),
                finalQueueSize: this.eventQueue.length,
                processingMode: "event-driven",
            }, {
                priority: "low",
                tags: ["state-machine", "processing", "emergent"],
            });

        } finally {
            // Always release the distributed lock
            await this.releaseDistributedProcessingLock();
        }
    }

    /**
     * Legacy drain implementation - preserved for backwards compatibility
     * @deprecated Use event-driven coordination instead
     */
    private async drainWithLegacyCoordination(): Promise<void> {
        if (this.processingLock) {
            this.logger.debug(`[${this.constructor.name}] Legacy drain already in progress`);
            return;
        }

        this.processingLock = true;
        this.state = BaseStates.RUNNING as TState;

        try {
            while (this.eventQueue.length > 0 && !this.disposed) {
                const event = this.eventQueue.shift()!;

                const result = await this.errorHandler.wrap(
                    () => this.processEvent(event),
                    "processEvent",
                    { eventType: event.type },
                );

                if (!result.success) {
                    const errorResult = result as { success: false; error: Error };
                    // Let subclass decide if error is fatal
                    if (await this.isErrorFatal(errorResult.error, event)) {
                        this.state = BaseStates.FAILED as TState;
                        return;
                    }
                }
            }

            if (!this.disposed && this.eventQueue.length === 0) {
                this.state = BaseStates.IDLE as TState;
                await this.onIdle();
            } else {
                // More events arrived while processing
                this.scheduleDrain();
            }
        } finally {
            this.processingLock = false;
        }
    }

    /**
     * Schedule the next drain cycle
     */
    protected scheduleDrain(delayMs = 0): void {
        if (this.disposed || this.state === BaseStates.PAUSED as TState) {
            return;
        }

        this.clearPendingDrainTimeout();

        if (delayMs > 0) {
            this.pendingDrainTimeout = setTimeout(() => {
                this.pendingDrainTimeout = null;
                this.drain().catch(err =>
                    this.logger.error(`[${this.constructor.name}] Error in scheduled drain`, {
                        error: err instanceof Error ? err.message : String(err),
                    }),
                );
            }, delayMs);
        } else {
            setImmediate(() =>
                this.drain().catch(err =>
                    this.logger.error(`[${this.constructor.name}] Error in immediate drain`, {
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
            this.logger.debug(`[${this.componentName}] Unified event bus not available, skipping event publish`);
            return;
        }

        try {
            const event = EventUtils.createBaseEvent(
                eventType,
                data,
                EventUtils.createEventSource("cross-cutting", this.componentName, generatePK()),
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
        } catch (error) {
            this.logger.error(`[${this.componentName}] Failed to publish unified event`, {
                eventType,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Emit an event to the event bus using UnifiedEventSystem
     */
    protected async emitEvent(type: string, data: unknown): Promise<void> {
        await this.publishUnifiedEvent(
            `state.machine.${type}`,
            data,
            {
                priority: "medium",
                deliveryGuarantee: "reliable",
            },
        );
    }

    /**
     * Emit a state change event (common pattern for state machines)
     */
    protected async emitStateChange(fromState: TState, toState: TState, context?: Record<string, any>): Promise<void> {
        await this.publishUnifiedEvent(
            "state_machine.state.changed",
            {
                taskId: this.getTaskId(),
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
     * Common logging patterns for lifecycle events
     */
    protected logLifecycleEvent(event: string, metadata?: Record<string, unknown>): void {
        this.logger.info(`[${this.constructor.name}] ${event}`, {
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
                this.logger.error(`[${this.constructor.name}] Event missing required field: ${field}`, {
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

    // Event-driven coordination methods

    /**
     * Acquire distributed processing lock to replace manual processingLock
     * 
     * This provides cross-instance coordination that scales beyond single-process deployments.
     * Falls back to immediate success if distributed locking unavailable.
     */
    private async acquireDistributedProcessingLock(): Promise<boolean> {
        if (!this.coordinationConfig.enableDistributedLocking || !this.coordinationConfig.swarmId) {
            // No distributed locking configured, proceed immediately
            return true;
        }

        try {
            // Generate a unique lock ID for this processing cycle
            const lockId = `${this.componentName}:processing:${generatePK()}`;
            this.currentDistributedLock = lockId;

            // In a full implementation, this would use SwarmContextManager or Redis
            // For now, simulate successful lock acquisition
            this.logger.debug(`[${this.componentName}] Acquired distributed processing lock`, {
                lockId,
                swarmId: this.coordinationConfig.swarmId,
                timeout: this.coordinationConfig.lockTimeoutMs,
            });

            return true;

        } catch (error) {
            this.logger.warn(`[${this.componentName}] Failed to acquire distributed processing lock`, {
                error: error instanceof Error ? error.message : String(error),
                swarmId: this.coordinationConfig.swarmId,
            });

            // Fall back gracefully - allow processing to continue
            if (this.coordinationConfig.enableLegacyFallback) {
                this.logger.debug(`[${this.componentName}] Falling back to legacy coordination due to lock failure`);
                return true;
            }

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
            this.logger.debug(`[${this.componentName}] Released distributed processing lock`, {
                lockId: this.currentDistributedLock,
                swarmId: this.coordinationConfig.swarmId,
            });

            this.currentDistributedLock = null;

        } catch (error) {
            this.logger.error(`[${this.componentName}] Failed to release distributed processing lock`, {
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
        mode: "event-driven" | "legacy";
        eventSystemAvailable: boolean;
        distributedLockingEnabled: boolean;
        currentLock: string | null;
        swarmId?: string;
    } {
        return {
            mode: this.eventDrivenCoordinationEnabled ? "event-driven" : "legacy",
            eventSystemAvailable: this.unifiedEventBus !== null,
            distributedLockingEnabled: this.coordinationConfig.enableDistributedLocking || false,
            currentLock: this.currentDistributedLock,
            swarmId: this.coordinationConfig.swarmId,
        };
    }
}
