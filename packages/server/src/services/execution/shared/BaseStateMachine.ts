/**
 * Base State Machine
 * 
 * Abstract base class for all state machines in the execution architecture.
 * Provides common functionality for state management, event queuing, and lifecycle control.
 * 
 * This follows the principle of keeping things simple while reducing duplication.
 * Subclasses implement specific state transitions and event handling logic.
 */

import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";

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
 * Base event interface that all state machine events must extend
 */
export interface BaseEvent {
    type: string;
    timestamp?: Date;
    metadata?: Record<string, unknown>;
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
 * Abstract base class for state machines
 */
export abstract class BaseStateMachine<
    TState extends string = BaseState,
    TEvent extends BaseEvent = BaseEvent
> implements ManagedTaskStateMachine {
    protected state: TState;
    protected readonly eventQueue: TEvent[] = [];
    protected processingLock = false;
    protected disposed = false;
    protected pendingDrainTimeout: NodeJS.Timeout | null = null;
    
    constructor(
        protected readonly logger: Logger,
        protected readonly eventBus: EventBus,
        initialState: TState = BaseStates.UNINITIALIZED as TState,
    ) {
        this.state = initialState;
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

        try {
            // Clear any pending operations
            this.clearPendingDrainTimeout();
            this.processingLock = false;

            // Let subclass handle cleanup
            const finalState = await this.onStop(mode, reason);

            // Update state
            this.state = (mode === "force" ? BaseStates.TERMINATED : BaseStates.STOPPED) as TState;
            this.disposed = true;

            return {
                success: true,
                message: `Stopped successfully (${mode})`,
                finalState,
            };
        } catch (error) {
            this.logger.error(`[${this.constructor.name}] Error during stop`, {
                error: error instanceof Error ? error.message : String(error),
            });
            
            this.state = BaseStates.FAILED as TState;
            
            return {
                success: false,
                message: "Error during stop",
                error: error instanceof Error ? error.message : "Unknown error",
            };
        }
    }

    /**
     * Process queued events
     */
    protected async drain(): Promise<void> {
        const drainableStates = [BaseStates.RUNNING, BaseStates.IDLE] as TState[];
        
        if (!drainableStates.includes(this.state) || this.disposed) {
            this.logger.debug(`[${this.constructor.name}] Cannot drain in state ${this.state}`);
            return;
        }

        if (this.processingLock) {
            this.logger.debug(`[${this.constructor.name}] Drain already in progress`);
            return;
        }

        this.processingLock = true;
        this.state = BaseStates.RUNNING as TState;

        while (this.eventQueue.length > 0 && !this.disposed) {
            const event = this.eventQueue.shift()!;
            
            try {
                await this.processEvent(event);
            } catch (error) {
                this.logger.error(`[${this.constructor.name}] Error processing event`, {
                    event: event.type,
                    error: error instanceof Error ? error.message : String(error),
                });
                
                // Let subclass decide if error is fatal
                if (await this.isErrorFatal(error, event)) {
                    this.state = BaseStates.FAILED as TState;
                    this.processingLock = false;
                    return;
                }
            }
        }

        this.processingLock = false;
        
        if (!this.disposed && this.eventQueue.length === 0) {
            this.state = BaseStates.IDLE as TState;
            await this.onIdle();
        } else {
            // More events arrived while processing
            this.scheduleDrain();
        }
    }

    /**
     * Schedule the next drain cycle
     */
    protected scheduleDrain(delayMs: number = 0): void {
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
                    })
                );
            }, delayMs);
        } else {
            setImmediate(() => 
                this.drain().catch(err => 
                    this.logger.error(`[${this.constructor.name}] Error in immediate drain`, {
                        error: err instanceof Error ? err.message : String(err),
                    })
                )
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
     * Emit an event to the event bus
     */
    protected async emitEvent(type: string, data: unknown): Promise<void> {
        await this.eventBus.publish(type, data);
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
}