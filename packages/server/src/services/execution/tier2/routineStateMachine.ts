import { RunState } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { getEventBus } from "../../events/eventBus.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import type { IRunContextManager } from "./runContextManager.js";
import type { ResourceUsage, RunAllocation } from "./types.js";

/**
 * RoutineStateMachine - Lean state machine for routine execution
 * 
 * TODO This is a simplified version that doesn't extend BaseStateMachine to avoid
 * circular import issues during the migration phase. Once the architecture
 * is stabilized, this can be updated to extend BaseStateMachine.
 * 
 * ## Design Principles:
 * 1. **Minimal Code**: Only routine-specific logic
 * 2. **No Duplication**: Leverages SwarmContextManager for persistence
 * 3. **Clear Transitions**: Simple, declarative state transition map
 * 4. **Event-Driven**: Manual event emission for state changes
 * 
 * ## State Flow:
 * ```
 * CREATED → NAVIGATOR_SELECTION → PLANNING → EXECUTING → COMPLETED
 *                                    ↓           ↓          ↓
 *                                 FAILED     PAUSED      FAILED
 * ```
 */
export class RoutineStateMachine {
    private state: RunState = RunState.UNINITIALIZED;
    private readonly contextId: string;
    private readonly swarmContextManager?: ISwarmContextManager;
    private readonly runContextManager?: IRunContextManager;

    // Resource tracking
    private currentAllocation?: RunAllocation;
    private resourceUsage: ResourceUsage = {
        credits: "0",
        durationMs: 0,
        memoryMB: 0,
        stepsExecuted: 0,
    };
    private executionStartTime?: Date;

    constructor(
        contextId: string,
        swarmContextManager: ISwarmContextManager | undefined,
        runContextManager?: IRunContextManager,
    ) {
        this.contextId = contextId;
        this.swarmContextManager = swarmContextManager;
        this.runContextManager = runContextManager;
    }

    /**
     * Define routine-specific valid state transitions
     */
    private getValidTransitions(): Map<RunState, RunState[]> {
        return new Map([
            // Initial state transitions
            [RunState.UNINITIALIZED, [RunState.LOADING, RunState.FAILED, RunState.CANCELLED]],
            [RunState.LOADING, [RunState.CONFIGURING, RunState.FAILED, RunState.CANCELLED]],
            [RunState.CONFIGURING, [RunState.READY, RunState.FAILED, RunState.CANCELLED]],
            [RunState.READY, [RunState.RUNNING, RunState.FAILED, RunState.CANCELLED]],

            // Running state transitions
            [RunState.RUNNING, [RunState.COMPLETED, RunState.FAILED, RunState.PAUSED, RunState.SUSPENDED, RunState.CANCELLED]],

            // Paused/suspended can resume or be cancelled/failed
            [RunState.PAUSED, [RunState.RUNNING, RunState.CANCELLED, RunState.FAILED]],
            [RunState.SUSPENDED, [RunState.RUNNING, RunState.CANCELLED, RunState.FAILED]],

            // Terminal states have no transitions
            [RunState.COMPLETED, []],
            [RunState.FAILED, []],
            [RunState.CANCELLED, []],
        ]);
    }

    /**
     * Get the current state
     */
    public getState(): RunState {
        return this.state;
    }

    /**
     * Get the task ID for this state machine
     */
    public getTaskId(): string {
        return this.contextId;
    }

    /**
     * Transition to a new state
     */
    public async transitionTo(newState: RunState): Promise<void> {
        const validTransitions = this.getValidTransitions();
        const currentState = this.getState();
        const allowedStates = validTransitions.get(currentState) || [];

        if (!allowedStates.includes(newState)) {
            throw new Error(
                `Invalid state transition from ${currentState} to ${newState}. ` +
                `Allowed transitions: ${allowedStates.join(", ")}`,
            );
        }

        const previousState = this.state;
        this.state = newState;

        await this.persistState(newState);
        await this.emitStateChange(previousState, newState);

        logger.info("[RoutineStateMachine] State transition", {
            contextId: this.contextId,
            from: previousState,
            to: newState,
        });
    }

    /**
     * Persist state using SwarmContextManager
     */
    private async persistState(state: RunState): Promise<void> {
        if (!this.swarmContextManager) {
            logger.debug("[RoutineStateMachine] No SwarmContextManager, skipping persistence", {
                state,
                contextId: this.contextId,
            });
            return;
        }

        try {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.state": state,
                "execution.lastTransition": new Date().toISOString(),
                "execution.stateHistory": await this.getStateHistory(),
            });

            logger.debug("[RoutineStateMachine] State persisted", {
                state,
                contextId: this.contextId,
            });
        } catch (error) {
            logger.error("[RoutineStateMachine] Failed to persist state", {
                state,
                contextId: this.contextId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Emit state change event
     */
    private async emitStateChange(fromState: RunState, toState: RunState): Promise<void> {
        try {
            await getEventBus().publish({
                type: "routine.state.changed",
                source: { tier: 2, component: "RoutineStateMachine" },
                data: {
                    contextId: this.contextId,
                    fromState,
                    toState,
                    timestamp: new Date().toISOString(),
                },
            });
        } catch (error) {
            logger.error("[RoutineStateMachine] Failed to emit state change event", {
                contextId: this.contextId,
                fromState,
                toState,
                error: error instanceof Error ? error.message : String(error),
            });
            // Don't throw - state change is more important than event emission
        }
    }

    /**
     * Get state history for persistence
     */
    private async getStateHistory(): Promise<Array<{ state: RunState; timestamp: string }>> {
        // For now, return current state as history
        // In a full implementation, this would track history
        const currentState = this.getState();
        return [{
            state: currentState,
            timestamp: new Date().toISOString(),
        }];
    }

    /**
     * Initialize execution with proper state and resource allocation
     */
    async initializeExecution(executionId: string, routineId: string, swarmId?: string): Promise<void> {
        // Transition through initialization states
        await this.transitionTo(RunState.LOADING);

        // Allocate resources if RunContextManager is available
        if (this.runContextManager && swarmId) {
            try {
                this.currentAllocation = await this.runContextManager.allocateFromSwarm(swarmId, {
                    runId: executionId,
                    routineId,
                    estimatedRequirements: {
                        credits: "1000", // Default allocation
                        durationMs: 300000, // 5 minutes
                        memoryMB: 512,
                        maxSteps: 50,
                    },
                    priority: "medium",
                    purpose: `Routine execution: ${routineId}`,
                });

                logger.info("[RoutineStateMachine] Resources allocated", {
                    executionId,
                    routineId,
                    allocation: this.currentAllocation,
                });
            } catch (error) {
                logger.error("[RoutineStateMachine] Failed to allocate resources", {
                    executionId,
                    routineId,
                    error: error instanceof Error ? error.message : String(error),
                });
                await this.transitionTo(RunState.FAILED);
                throw error;
            }
        }

        await this.transitionTo(RunState.CONFIGURING);
        await this.transitionTo(RunState.READY);

        // Persist initial context if SwarmContextManager available
        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.id": executionId,
                "execution.routineId": routineId,
                "execution.startTime": new Date().toISOString(),
                "execution.state": RunState.READY,
                "execution.allocation": this.currentAllocation ? {
                    allocationId: this.currentAllocation.allocationId,
                    swarmId: this.currentAllocation.swarmId,
                    allocatedCredits: this.currentAllocation.allocated.credits,
                } : null,
            });
        }

        logger.info("[RoutineStateMachine] Execution initialized", {
            executionId,
            routineId,
            contextId: this.contextId,
            hasAllocation: !!this.currentAllocation,
        });
    }

    /**
     * Helper method to check if execution can proceed
     */
    async canProceed(): Promise<boolean> {
        const currentState = this.getState();
        return ![
            RunState.COMPLETED,
            RunState.FAILED,
            RunState.CANCELLED,
        ].includes(currentState);
    }

    /**
     * Helper method to check if execution is in terminal state
     */
    async isTerminal(): Promise<boolean> {
        const currentState = this.getState();
        return [
            RunState.COMPLETED,
            RunState.FAILED,
            RunState.CANCELLED,
        ].includes(currentState);
    }

    /**
     * Start execution (transition from READY to RUNNING)
     */
    async start(): Promise<void> {
        if (this.getState() !== RunState.READY) {
            throw new Error(`Cannot start from state ${this.getState()}`);
        }

        // Mark execution start time for resource tracking
        this.executionStartTime = new Date();

        await this.transitionTo(RunState.RUNNING);

        // Emit run started event through RunContextManager
        if (this.runContextManager && this.currentAllocation) {
            const routineId = await this.getRoutineId();
            await this.runContextManager.emitRunStarted(
                this.contextId,
                routineId || "unknown",
                this.currentAllocation,
            );
        }

        logger.info("[RoutineStateMachine] Execution started", {
            contextId: this.contextId,
            startTime: this.executionStartTime,
        });
    }

    /**
     * Stop the state machine (transition to cancelled)
     */
    async stop(reason?: string): Promise<void> {
        if (await this.isTerminal()) {
            logger.debug("[RoutineStateMachine] Already in terminal state, ignoring stop", {
                contextId: this.contextId,
                currentState: this.getState(),
            });
            return;
        }

        await this.transitionTo(RunState.CANCELLED);

        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.stopReason": reason || "Manual stop",
                "execution.stopTime": new Date().toISOString(),
            });
        }

        logger.info("[RoutineStateMachine] Execution stopped", {
            contextId: this.contextId,
            reason,
        });
    }

    /**
     * Pause the state machine (only valid during execution)
     */
    async pause(): Promise<void> {
        if (this.getState() !== RunState.RUNNING) {
            throw new Error(`Cannot pause from state ${this.getState()}`);
        }

        await this.transitionTo(RunState.PAUSED);
        logger.info("[RoutineStateMachine] Execution paused", {
            contextId: this.contextId,
        });
    }

    /**
     * Resume the state machine (only valid when paused or suspended)
     */
    async resume(): Promise<void> {
        const currentState = this.getState();
        if (currentState !== RunState.PAUSED && currentState !== RunState.SUSPENDED) {
            throw new Error(`Cannot resume from state ${currentState}`);
        }

        await this.transitionTo(RunState.RUNNING);
        logger.info("[RoutineStateMachine] Execution resumed", {
            contextId: this.contextId,
        });
    }

    /**
     * Suspend the state machine (only valid during execution)
     */
    async suspend(): Promise<void> {
        if (this.getState() !== RunState.RUNNING) {
            throw new Error(`Cannot suspend from state ${this.getState()}`);
        }

        await this.transitionTo(RunState.SUSPENDED);
        logger.info("[RoutineStateMachine] Execution suspended", {
            contextId: this.contextId,
        });
    }

    /**
     * Mark execution as failed
     */
    async fail(error: string): Promise<void> {
        if (await this.isTerminal()) {
            logger.debug("[RoutineStateMachine] Already in terminal state, ignoring fail", {
                contextId: this.contextId,
                currentState: this.getState(),
            });
            return;
        }

        // Calculate resource usage up to failure
        this.updateResourceUsage();

        await this.transitionTo(RunState.FAILED);

        // Emit failure event and release resources
        if (this.runContextManager && this.currentAllocation) {
            await this.runContextManager.emitRunFailed(
                this.contextId,
                new Error(error),
                this.resourceUsage,
            );

            await this.runContextManager.releaseToSwarm(
                this.currentAllocation.swarmId,
                this.contextId,
                this.resourceUsage,
            );
        }

        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.error": error,
                "execution.failureTime": new Date().toISOString(),
                "execution.resourceUsage": this.resourceUsage,
            });
        }

        logger.error("[RoutineStateMachine] Execution failed", {
            contextId: this.contextId,
            error,
            resourceUsage: this.resourceUsage,
        });
    }

    /**
     * Mark execution as completed
     */
    async complete(result?: any): Promise<void> {
        if (this.getState() !== RunState.RUNNING) {
            throw new Error(`Cannot complete from state ${this.getState()}`);
        }

        // Calculate final resource usage
        this.updateResourceUsage();

        await this.transitionTo(RunState.COMPLETED);

        // Emit completion event and release resources
        if (this.runContextManager && this.currentAllocation) {
            await this.runContextManager.emitRunCompleted(
                this.contextId,
                result || { success: true },
                this.resourceUsage,
            );

            await this.runContextManager.releaseToSwarm(
                this.currentAllocation.swarmId,
                this.contextId,
                this.resourceUsage,
            );
        }

        if (this.swarmContextManager) {
            await this.swarmContextManager.updateContext(this.contextId, {
                "execution.completionTime": new Date().toISOString(),
                "execution.resourceUsage": this.resourceUsage,
                "execution.result": result,
            });
        }

        logger.info("[RoutineStateMachine] Execution completed", {
            contextId: this.contextId,
            resourceUsage: this.resourceUsage,
        });
    }

    /**
     * Get routine ID from context
     */
    private async getRoutineId(): Promise<string | null> {
        if (!this.swarmContextManager) return null;

        try {
            const context = await this.swarmContextManager.getContext(this.contextId);
            return context?.["execution.routineId"] as string || null;
        } catch (error) {
            logger.debug("[RoutineStateMachine] Could not get routine ID", {
                contextId: this.contextId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    /**
     * Update resource usage tracking
     */
    private updateResourceUsage(): void {
        if (this.executionStartTime) {
            this.resourceUsage.durationMs = Date.now() - this.executionStartTime.getTime();
        }

        // Update other resource usage metrics here as needed
        // In a real implementation, this would track actual resource consumption
    }

    /**
     * Add credits to resource usage
     */
    public addCreditsUsed(credits: string): void {
        const current = BigInt(this.resourceUsage.credits);
        const additional = BigInt(credits);
        this.resourceUsage.credits = (current + additional).toString();
    }

    /**
     * Increment step count
     */
    public incrementStepCount(): void {
        this.resourceUsage.stepsExecuted += 1;
    }

    /**
     * Get current resource allocation
     */
    public getCurrentAllocation(): RunAllocation | undefined {
        return this.currentAllocation;
    }

    /**
     * Get current resource usage
     */
    public getResourceUsage(): ResourceUsage {
        this.updateResourceUsage();
        return { ...this.resourceUsage };
    }
} 
