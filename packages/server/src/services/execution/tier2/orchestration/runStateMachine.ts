/**
 * Run State Machine - Simple state management for routine runs
 * 
 * This component manages run state transitions and emits events.
 * Complex orchestration logic emerges from orchestration agents
 * analyzing execution patterns.
 */

import { type Logger } from "winston";
import {
    type Run,
    type RunState,
    type RunConfig,
    type RunProgress,
    type RunEvent,
    type RunEventType,
    type Location,
    type TierCommunicationInterface,
    type TierExecutionRequest,
    type ExecutionResult,
    type ExecutionStatus,
    RunState as RunStateEnum,
    RunEventType as RunEventTypeEnum,
    generatePk,
} from "@vrooli/shared";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type IRunStateStore } from "../state/runStateStore.js";
import { BaseStateMachine, BaseStates, type BaseState, type BaseEvent } from "../../shared/BaseStateMachine.js";

/**
 * Run initialization parameters
 */
export interface RunInitParams {
    routineId: string;
    userId: string;
    inputs: Record<string, unknown>;
    config?: Partial<RunConfig>;
    parentRunId?: string;
    swarmId?: string;
}

/**
 * Run event for state machine
 */
interface RunStateMachineEvent extends BaseEvent {
    runId: string;
    executionRequest?: TierExecutionRequest;
    executionResult?: ExecutionResult;
    error?: unknown;
}

/**
 * Run State Machine
 * 
 * Manages basic run lifecycle and state transitions.
 * Delegates execution to Tier 3 and emits events for monitoring.
 * 
 * Now extends BaseStateMachine for consistency and shared functionality.
 * 
 * Does NOT implement:
 * - Complex orchestration logic
 * - Path optimization algorithms
 * - Performance analysis
 * - Branch coordination strategies
 * 
 * These capabilities emerge from orchestration agents.
 */
export class TierTwoRunStateMachine extends BaseStateMachine<BaseState, RunStateMachineEvent> implements TierCommunicationInterface {
    private readonly stateStore: IRunStateStore;
    private readonly tier3Executor: TierCommunicationInterface;
    private readonly activeRuns: Map<string, Run> = new Map();
    private currentRunId: string | null = null;

    constructor(
        logger: Logger,
        eventBus: EventBus,
        stateStore: IRunStateStore,
        tier3Executor: TierCommunicationInterface,
    ) {
        super(logger, eventBus, BaseStates.UNINITIALIZED);
        this.stateStore = stateStore;
        this.tier3Executor = tier3Executor;

        // Subscribe to execution events
        this.subscribeToEvents();
    }

    /**
     * TierCommunicationInterface implementation
     */
    async execute(request: TierExecutionRequest): Promise<ExecutionResult> {
        this.logger.info("[RunStateMachine] Executing request", {
            executionId: request.executionId,
            type: request.type,
        });

        try {
            if (request.type === "routine") {
                const run = await this.createRun({
                    routineId: request.payload.routineId,
                    userId: request.payload.userId || "system",
                    inputs: request.payload.inputs || {},
                    config: request.payload.config,
                    parentRunId: request.payload.parentRunId,
                    swarmId: request.metadata?.swarmId,
                });

                await this.startRun(run.id);

                return {
                    executionId: request.executionId,
                    status: ExecutionStatus.IN_PROGRESS,
                    result: {
                        runId: run.id,
                        state: run.state,
                    },
                };
            }

            throw new Error(`Unsupported execution type: ${request.type}`);
        } catch (error) {
            this.logger.error("[RunStateMachine] Execution failed", {
                executionId: request.executionId,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                executionId: request.executionId,
                status: ExecutionStatus.FAILED,
                error: error instanceof Error ? error.message : String(error),
            };
        }
    }

    async getCapabilities(): Promise<{ supported: string[] }> {
        return {
            supported: ["routine", "workflow", "subroutine"],
        };
    }

    /**
     * Creates a new run and queues start event
     */
    async createRun(params: RunInitParams): Promise<Run> {
        const runId = generatePk();
        
        this.logger.info("[RunStateMachine] Creating run", {
            runId,
            routineId: params.routineId,
            userId: params.userId,
        });

        const run: Run = {
            id: runId,
            routineId: params.routineId,
            userId: params.userId,
            state: RunStateEnum.Waiting,
            inputs: params.inputs,
            outputs: {},
            progress: {
                currentStepId: null,
                completedSteps: [],
                totalSteps: 0,
                percentComplete: 0,
            },
            context: {
                variables: {},
                outputs: {},
            },
            config: {
                maxRetries: 3,
                timeout: 3600000, // 1 hour default
                parallel: true,
                ...params.config,
            },
            createdAt: new Date(),
            updatedAt: new Date(),
            parentRunId: params.parentRunId,
            swarmId: params.swarmId,
        };

        // Store run
        await this.stateStore.saveRun(run);
        this.activeRuns.set(runId, run);
        this.currentRunId = runId;

        // Emit creation event
        await this.emitRunEvent(run, RunEventTypeEnum.Created);

        return run;
    }

    /**
     * Starts run execution by queuing start event
     */
    async startRun(runId: string): Promise<void> {
        const run = await this.getRunState(runId);
        if (!run) {
            throw new Error(`Run not found: ${runId}`);
        }

        this.logger.info("[RunStateMachine] Queuing start for run", { runId });

        // Queue execution start event
        await this.handleEvent({
            type: "START_EXECUTION",
            runId,
            timestamp: new Date(),
        });
    }

    /**
     * Updates run state
     */
    async updateRunState(runId: string, newState: RunState): Promise<void> {
        const run = await this.getRunState(runId);
        if (!run) {
            throw new Error(`Run not found: ${runId}`);
        }

        const oldState = run.state;
        run.state = newState;
        run.updatedAt = new Date();

        // Save to store
        await this.stateStore.saveRun(run);
        this.activeRuns.set(runId, run);

        // Emit state change event
        await this.emitRunEvent(run, RunEventTypeEnum.StateChanged, {
            oldState,
            newState,
        });

        this.logger.debug("[RunStateMachine] State updated", {
            runId,
            oldState,
            newState,
        });
    }

    /**
     * Updates run progress
     */
    async updateProgress(runId: string, progress: Partial<RunProgress>): Promise<void> {
        const run = await this.getRunState(runId);
        if (!run) {
            throw new Error(`Run not found: ${runId}`);
        }

        run.progress = { ...run.progress, ...progress };
        run.updatedAt = new Date();

        // Save to store
        await this.stateStore.saveRun(run);

        // Emit progress event
        await this.emitRunEvent(run, RunEventTypeEnum.ProgressUpdated, progress);
    }

    /**
     * Completes a run
     */
    async completeRun(runId: string, outputs: Record<string, unknown>): Promise<void> {
        const run = await this.getRunState(runId);
        if (!run) {
            throw new Error(`Run not found: ${runId}`);
        }

        run.outputs = outputs;
        run.completedAt = new Date();
        
        await this.updateRunState(runId, RunStateEnum.Completed);
        await this.emitRunEvent(run, RunEventTypeEnum.Completed);

        // Clean up
        this.activeRuns.delete(runId);

        this.logger.info("[RunStateMachine] Run completed", { runId });
    }

    /**
     * Fails a run
     */
    async failRun(runId: string, error: string): Promise<void> {
        const run = await this.getRunState(runId);
        if (!run) {
            throw new Error(`Run not found: ${runId}`);
        }

        run.error = error;
        run.completedAt = new Date();
        
        await this.updateRunState(runId, RunStateEnum.Failed);
        await this.emitRunEvent(run, RunEventTypeEnum.Failed, { error });

        // Clean up
        this.activeRuns.delete(runId);

        this.logger.error("[RunStateMachine] Run failed", { runId, error });
    }

    /**
     * Private helper methods
     */
    private async getRunState(runId: string): Promise<Run | null> {
        // Check cache first
        if (this.activeRuns.has(runId)) {
            return this.activeRuns.get(runId)!;
        }

        // Load from store
        return await this.stateStore.getRun(runId);
    }

    private async emitRunEvent(
        run: Run,
        type: RunEventType,
        metadata?: any,
    ): Promise<void> {
        const event: RunEvent = {
            id: generatePk(),
            runId: run.id,
            type,
            timestamp: new Date(),
            metadata: {
                ...metadata,
                state: run.state,
                progress: run.progress,
            },
        };

        await this.eventBus.publish("run.events", event);
    }

    private async handleExecutionResult(
        runId: string,
        result: ExecutionResult,
    ): Promise<void> {
        if (result.status === ExecutionStatus.COMPLETED) {
            await this.completeRun(runId, result.result || {});
        } else if (result.status === ExecutionStatus.FAILED) {
            await this.failRun(runId, result.error || "Execution failed");
        }
        // IN_PROGRESS is handled by progress updates
    }

    private async handleExecutionError(runId: string, error: any): Promise<void> {
        const errorMessage = error instanceof Error ? error.message : String(error);
        await this.failRun(runId, errorMessage);
    }

    private subscribeToEvents(): void {
        // Subscribe to step completion events
        this.eventBus.subscribe("execution.step.completed", async (event) => {
            const runId = event.metadata?.runId;
            if (runId && this.activeRuns.has(runId)) {
                await this.updateProgress(runId, {
                    currentStepId: event.metadata.stepId,
                    completedSteps: event.metadata.completedSteps || [],
                });
            }
        });

        // Subscribe to execution status updates
        this.eventBus.subscribe("execution.status.changed", async (event) => {
            const runId = event.metadata?.runId;
            if (runId && this.activeRuns.has(runId)) {
                if (event.metadata.status === ExecutionStatus.COMPLETED) {
                    await this.handleExecutionResult(runId, {
                        executionId: event.metadata.executionId,
                        status: ExecutionStatus.COMPLETED,
                        result: event.metadata.result,
                    });
                } else if (event.metadata.status === ExecutionStatus.FAILED) {
                    await this.handleExecutionResult(runId, {
                        executionId: event.metadata.executionId,
                        status: ExecutionStatus.FAILED,
                        error: event.metadata.error,
                    });
                }
            }
        });
    }

    /**
     * Stops the state machine
     */
    async stop(): Promise<void> {
        // Cancel all active runs
        for (const runId of this.activeRuns.keys()) {
            await this.failRun(runId, "State machine stopped");
        }

        this.logger.info("[RunStateMachine] Stopped");
    }

    //
    // BaseStateMachine abstract method implementations
    //

    /**
     * Get the task ID for this state machine
     */
    public getTaskId(): string {
        return this.currentRunId || "no-active-run";
    }

    /**
     * Process a single event
     */
    protected async processEvent(event: RunStateMachineEvent): Promise<void> {
        switch (event.type) {
            case "START_EXECUTION":
                await this.handleStartExecution(event);
                break;
            case "EXECUTION_RESULT":
                await this.handleExecutionResultEvent(event);
                break;
            case "EXECUTION_ERROR":
                await this.handleExecutionErrorEvent(event);
                break;
            default:
                this.logger.warn("[RunStateMachine] Unknown event type", { type: event.type });
        }
    }

    /**
     * Called when entering idle state
     */
    protected async onIdle(): Promise<void> {
        this.logger.debug("[RunStateMachine] Entered idle state");
    }

    /**
     * Called when pausing
     */
    protected async onPause(): Promise<void> {
        this.logLifecycleEvent("Paused", {
            activeRunsCount: this.activeRuns.size,
        });
        // Pause all active runs
        for (const run of this.activeRuns.values()) {
            if (run.state === RunStateEnum.InProgress) {
                await this.updateRunState(run.id, RunStateEnum.Paused);
            }
        }
    }

    /**
     * Called when resuming
     */
    protected async onResume(): Promise<void> {
        this.logLifecycleEvent("Resumed", {
            activeRunsCount: this.activeRuns.size,
        });
        // Resume all paused runs
        for (const run of this.activeRuns.values()) {
            if (run.state === RunStateEnum.Paused) {
                await this.updateRunState(run.id, RunStateEnum.InProgress);
            }
        }
    }

    /**
     * Called when stopping - return final state data
     */
    protected async onStop(mode: "graceful" | "force", reason?: string): Promise<unknown> {
        this.logLifecycleEvent("Stopping", { 
            mode, 
            reason,
            activeRunsCount: this.activeRuns.size,
        });
        
        // Cancel all active runs
        for (const runId of this.activeRuns.keys()) {
            await this.failRun(runId, `State machine stopped: ${reason || "No reason"}`);
        }

        return {
            activeRunsCount: this.activeRuns.size,
            stopReason: reason,
            stopMode: mode,
        };
    }

    /**
     * Determine if an error is fatal
     */
    protected async isErrorFatal(error: unknown, event: RunStateMachineEvent): Promise<boolean> {
        // Most errors are not fatal for the state machine itself
        // Individual runs may fail, but the machine continues
        this.logger.error(`[${this.constructor.name}] Non-fatal error during event processing`, {
            eventType: event.type,
            runId: event.runId,
            error: error instanceof Error ? error.message : String(error),
        });
        
        // Only consider fatal if it's a critical system error
        return false;
    }

    /**
     * Helper method to get associated user ID (for ManagedTaskStateMachine interface)
     */
    public getAssociatedUserId(): string | undefined {
        if (this.currentRunId) {
            const run = this.activeRuns.get(this.currentRunId);
            return run?.userId;
        }
        return undefined;
    }

    //
    // Event handling methods for the new event-driven approach
    //

    private async handleStartExecution(event: RunStateMachineEvent): Promise<void> {
        const run = await this.getRunState(event.runId);
        if (!run) {
            throw new Error(`Run not found: ${event.runId}`);
        }

        this.logger.info("[RunStateMachine] Starting execution for run", { runId: event.runId });

        // Update state
        await this.updateRunState(event.runId, RunStateEnum.InProgress);

        // Emit start event
        await this.emitRunEvent(run, RunEventTypeEnum.Started);

        // Delegate execution to Tier 3
        const executionRequest: TierExecutionRequest = {
            executionId: `exec-${event.runId}`,
            tierOrigin: 2,
            tierTarget: 3,
            type: "routine",
            payload: {
                routineId: run.routineId,
                inputs: run.inputs,
                context: run.context,
            },
            metadata: {
                runId: event.runId,
                userId: run.userId,
                swarmId: run.swarmId,
            },
        };

        // Execute asynchronously and queue result events
        this.tier3Executor.execute(executionRequest)
            .then(result => this.handleEvent({
                type: "EXECUTION_RESULT",
                runId: event.runId,
                executionResult: result,
                timestamp: new Date(),
            }))
            .catch(error => this.handleEvent({
                type: "EXECUTION_ERROR",
                runId: event.runId,
                error,
                timestamp: new Date(),
            }));
    }

    private async handleExecutionResultEvent(event: RunStateMachineEvent): Promise<void> {
        if (!event.executionResult) {
            throw new Error("Missing execution result");
        }
        await this.handleExecutionResult(event.runId, event.executionResult);
    }

    private async handleExecutionErrorEvent(event: RunStateMachineEvent): Promise<void> {
        await this.handleExecutionError(event.runId, event.error);
    }
}