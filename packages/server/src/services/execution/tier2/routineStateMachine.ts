import { RunState, generatePK, getDotNotationValue, type BlackboardItem, type ExecutionResourceUsage, type RoutineAction, type RoutineExecutionInput, type TierExecutionRequest } from "@vrooli/shared";
import { DbProvider } from "../../../db/provider.js";
import { logger } from "../../../events/logger.js";
import type { ServiceEvent } from "../../events/types.js";
import type { BlackboardPatch } from "../../types/tools.js";
import { BaseStateMachine } from "../shared/BaseStateMachine.js";
import type { ISwarmContextManager, SwarmContextManager } from "../shared/SwarmContextManager.js";
import type { IRunContextManager } from "./runContextManager.js";
import type { RunAllocation, RunExecutionContext } from "./types.js";

/**
 * RoutineStateMachine - Event-driven state machine for routine execution
 * 
 * Now extends BaseStateMachine for unified state management, event handling,
 * and integration with the active task registry.
 * 
 * ## Design Principles:
 * 1. **Event-Driven**: Leverages BaseStateMachine's event queue and processing
 * 2. **Unified State Management**: Uses RunState directly for consistency
 * 3. **Registry Integration**: Implements ManagedTaskStateMachine via BaseStateMachine
 * 4. **SwarmContextManager**: Uses unified context for persistence
 * 
 * ## State Flow:
 * ```
 * UNINITIALIZED → LOADING → CONFIGURING → READY → RUNNING → COMPLETED
 *                     ↓          ↓          ↓        ↓          ↓
 *                  FAILED     FAILED     FAILED   PAUSED    FAILED
 * ```
 */
export class RoutineStateMachine extends BaseStateMachine<ServiceEvent> {
    private readonly contextId: string;
    protected readonly swarmContextManager: SwarmContextManager | null = null;
    private readonly runContextManager?: IRunContextManager;
    private readonly parentSwarmId?: string;
    private userId?: string;

    // Execution request metadata for outputOperations processing
    private executionRequest?: TierExecutionRequest<RoutineExecutionInput>;

    // Resource tracking
    private currentAllocation?: RunAllocation;
    private resourceUsage: ExecutionResourceUsage = {
        creditsUsed: "0",
        durationMs: 0,
        memoryUsedMB: 0,
        stepsExecuted: 0,
        toolCalls: 0,
    };
    private executionStartTime?: Date;

    constructor(
        contextId: string,
        swarmContextManager: ISwarmContextManager | undefined,
        runContextManager?: IRunContextManager,
        userId?: string,
        parentSwarmId?: string,
    ) {
        super(RunState.UNINITIALIZED, "RoutineStateMachine", {
            contextId, // Use contextId for routine coordination
            swarmId: parentSwarmId, // Use actual swarmId for swarm coordination
        });
        this.contextId = contextId;
        this.swarmContextManager = swarmContextManager as SwarmContextManager | null;
        this.runContextManager = runContextManager;
        this.parentSwarmId = parentSwarmId;
        this.userId = userId;
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
     * Get the task ID for this state machine
     */
    public getTaskId(): string {
        return this.contextId;
    }

    /**
     * Set the execution request for outputOperations processing
     */
    public setExecutionRequest(request: TierExecutionRequest<RoutineExecutionInput>): void {
        this.executionRequest = request;
    }

    /**
     * Transition to a new RunState
     */
    public async transitionTo(newState: RunState): Promise<void> {
        const validTransitions = this.getValidTransitions();
        const currentRunState = this.getState();
        const allowedStates = validTransitions.get(currentRunState) || [];

        if (!allowedStates.includes(newState)) {
            throw new Error(
                `Invalid state transition from ${currentRunState} to ${newState}. ` +
                `Allowed transitions: ${allowedStates.join(", ")}`,
            );
        }

        const previousRunState = this.getState();
        this.state = newState;

        await this.persistState(newState);

        logger.info("[RoutineStateMachine] State transition", {
            contextId: this.contextId,
            from: previousRunState,
            to: newState,
        });
    }

    /**
     * Persist routine execution state using RunContextManager
     */
    private async persistState(state: RunState): Promise<void> {
        if (!this.runContextManager) {
            logger.debug("[RoutineStateMachine] No RunContextManager, skipping routine state persistence", {
                state,
                contextId: this.contextId,
            });
            return;
        }

        try {
            // Get current run context and update execution state
            const currentContext = await this.runContextManager.getRunContext(this.contextId);

            // Update the execution state in routine context
            const updatedContext = {
                ...currentContext,
                progress: {
                    ...currentContext.progress,
                    currentStepId: currentContext.progress?.currentStepId || null,
                },
                // Store state transition information in routine context
                lastStateTransition: {
                    state,
                    timestamp: new Date().toISOString(),
                    previousState: this.getState(),
                },
            };

            await this.runContextManager.updateRunContext(this.contextId, updatedContext);

            logger.debug("[RoutineStateMachine] Routine state persisted", {
                state,
                contextId: this.contextId,
            });
        } catch (error) {
            // Fallback: if RunContextManager fails, we can continue execution
            // This ensures routines don't fail due to context persistence issues
            logger.warn("[RoutineStateMachine] Failed to persist routine state, continuing execution", {
                state,
                contextId: this.contextId,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Initialize execution with proper state and resource allocation
     * Supports both new execution and resumption based on runId parameter
     */
    async initializeExecution(executionId: string, resourceVersionId: string, swarmId?: string, runId?: string): Promise<void> {
        // Check if this is a resumption request
        if (runId) {
            await this.loadAndResumeExistingRun(runId, resourceVersionId, swarmId);
            return;
        }

        // NEW EXECUTION PATH - existing logic
        // Transition through initialization states
        await this.transitionTo(RunState.LOADING);

        // Allocate resources if RunContextManager is available
        if (this.runContextManager && swarmId) {
            try {
                this.currentAllocation = await this.runContextManager.allocateFromSwarm(swarmId, {
                    runId: executionId,
                    routineId: resourceVersionId,
                    estimatedRequirements: {
                        credits: "1000", // Default allocation
                        durationMs: 300000, // 5 minutes
                        memoryMB: 512,
                        maxSteps: 50,
                    },
                    priority: "medium",
                    purpose: `Routine execution: ${resourceVersionId}`,
                });

                logger.info("[RoutineStateMachine] Resources allocated", {
                    executionId,
                    resourceVersionId,
                    allocation: this.currentAllocation,
                });
            } catch (error) {
                logger.error("[RoutineStateMachine] Failed to allocate resources", {
                    executionId,
                    resourceVersionId,
                    error: error instanceof Error ? error.message : String(error),
                });
                await this.transitionTo(RunState.FAILED);
                throw error;
            }
        }

        await this.transitionTo(RunState.CONFIGURING);
        await this.transitionTo(RunState.READY);

        // Setup event subscriptions now that we have the execution ID
        await this.setupEventSubscriptions();

        // Initialize routine execution context
        if (this.runContextManager) {
            try {
                // Create initial run execution context
                const initialContext: RunExecutionContext = {
                    runId: executionId,
                    routineId: resourceVersionId,
                    swarmId,

                    // Navigation state (will be populated by navigator)
                    navigator: null,
                    currentLocation: { id: "start", routineId: resourceVersionId, nodeId: "start" },
                    visitedLocations: [],

                    // Execution state
                    variables: {},
                    outputs: {},
                    completedSteps: [],
                    parallelBranches: [],

                    // Swarm inheritance
                    parentContext: swarmId ? { swarmId } : undefined,
                    availableResources: this.currentAllocation ? [this.currentAllocation] : [],
                    sharedKnowledge: {},

                    // Resource tracking
                    resourceLimits: {
                        maxCredits: this.currentAllocation?.allocated.credits.toString() || "1000",
                        maxDurationMs: 300000, // 5 minutes default
                        maxMemoryMB: 512,
                        maxSteps: 50,
                    },
                    resourceUsage: {
                        creditsUsed: "0",
                        durationMs: 0,
                        memoryUsedMB: 0,
                        stepsExecuted: 0,
                        startTime: new Date(),
                    },

                    // Progress tracking
                    progress: {
                        currentStepId: null,
                        completedSteps: [],
                        totalSteps: 0,
                        percentComplete: 0,
                    },

                    // Error handling
                    retryCount: 0,
                };

                await this.runContextManager.updateRunContext(this.contextId, initialContext);

                logger.debug("[RoutineStateMachine] Run execution context initialized", {
                    executionId,
                    resourceVersionId,
                    contextId: this.contextId,
                });
            } catch (error) {
                logger.warn("[RoutineStateMachine] Failed to initialize run context, continuing", {
                    executionId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        // Only notify swarm of routine start (minimal coordination info)
        if (this.swarmContextManager && this.parentSwarmId) {
            try {
                await this.swarmContextManager.updateContext(this.parentSwarmId, {
                    execution: {
                        status: RunState.RUNNING,
                        agents: [],
                        startedAt: new Date(),
                        lastActivityAt: new Date(),
                        activeRuns: [{
                            runId: this.contextId,
                            routineId: resourceVersionId,
                            agentId: "tier2-routine-executor",
                            startedAt: new Date(),
                            status: "running",
                        }],
                    },
                });

                logger.debug("[RoutineStateMachine] Notified swarm of routine start", {
                    swarmId: this.parentSwarmId,
                    routineExecutionId: this.contextId,
                });
            } catch (error) {
                logger.warn("[RoutineStateMachine] Failed to notify swarm of routine start", {
                    swarmId: this.parentSwarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        logger.info("[RoutineStateMachine] Execution initialized", {
            executionId,
            resourceVersionId,
            contextId: this.contextId,
            hasAllocation: !!this.currentAllocation,
        });
    }

    /**
     * Load run from database for resumption
     */
    private async loadRunFromDatabase(runId: string) {
        const run = await DbProvider.get().run.findUnique({
            where: { id: BigInt(runId) },
            select: {
                id: true,
                status: true,
                isPrivate: true,
                name: true,
                data: true,
                startedAt: true,
                completedAt: true,
                timeElapsed: true,
                completedComplexity: true,
                contextSwitches: true,
            },
        });

        if (!run) {
            throw new Error(`Run ${runId} not found in database`);
        }

        return run;
    }

    /**
     * Load and resume existing run from database
     */
    private async loadAndResumeExistingRun(runId: string, resourceVersionId: string, _swarmId?: string): Promise<void> {
        logger.info("[RoutineStateMachine] Loading existing run for resumption", {
            runId,
            resourceVersionId,
            contextId: this.contextId,
        });

        // 1. Load run from database
        const existingRun = await this.loadRunFromDatabase(runId);

        // 2. Validate run state is resumable
        if (!["Paused", "Suspended"].includes(existingRun.status)) {
            throw new Error(`Cannot resume run ${runId} - status: ${existingRun.status}. Only Paused and Suspended runs can be resumed.`);
        }

        // 3. Load routine execution context from RunContextManager
        if (this.runContextManager) {
            try {
                const runContext = await this.runContextManager.getRunContext(this.contextId);

                // 4. Restore state from run context
                if (runContext) {
                    // Map database status to RunState
                    const statusToRunState: Record<string, RunState> = {
                        "Paused": RunState.PAUSED,
                        "Suspended": RunState.SUSPENDED,
                        "Running": RunState.RUNNING,
                        "Completed": RunState.COMPLETED,
                        "Failed": RunState.FAILED,
                        "Cancelled": RunState.CANCELLED,
                    };

                    this.state = statusToRunState[existingRun.status] || RunState.PAUSED;

                    // Restore resource usage from run context
                    if (runContext.resourceUsage) {
                        this.resourceUsage = {
                            creditsUsed: runContext.resourceUsage.creditsUsed,
                            durationMs: runContext.resourceUsage.durationMs,
                            memoryUsedMB: runContext.resourceUsage.memoryUsedMB,
                            stepsExecuted: runContext.resourceUsage.stepsExecuted,
                            toolCalls: 0, // Default if not tracked
                        };

                        if (runContext.resourceUsage.startTime) {
                            this.executionStartTime = new Date(runContext.resourceUsage.startTime);
                        }
                    }

                    logger.info("[RoutineStateMachine] State and context restored from run context", {
                        runId,
                        restoredState: this.state,
                        contextId: this.contextId,
                        resourceUsage: this.resourceUsage,
                    });
                }
            } catch (error) {
                logger.warn("[RoutineStateMachine] Failed to load run context for resumption", {
                    runId,
                    contextId: this.contextId,
                    error: error instanceof Error ? error.message : String(error),
                });

                // Fallback: use database status to determine state
                const statusToRunState: Record<string, RunState> = {
                    "Paused": RunState.PAUSED,
                    "Suspended": RunState.SUSPENDED,
                };
                this.state = statusToRunState[existingRun.status] || RunState.PAUSED;
            }
        } else {
            // Fallback: use database status if no RunContextManager
            const statusToRunState: Record<string, RunState> = {
                "Paused": RunState.PAUSED,
                "Suspended": RunState.SUSPENDED,
            };
            this.state = statusToRunState[existingRun.status] || RunState.PAUSED;
        }

        // 5. Resume using existing infrastructure
        await this.resume(); // This method already exists in BaseStateMachine!

        logger.info("[RoutineStateMachine] Successfully resumed existing run", {
            runId,
            resourceVersionId,
            currentState: this.getState(),
            contextId: this.contextId,
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
     * Pause the state machine (only valid during execution)
     */
    async pause(): Promise<boolean> {
        if (this.getState() !== RunState.RUNNING) {
            throw new Error(`Cannot pause from state ${this.getState()}`);
        }

        await this.transitionTo(RunState.PAUSED);
        logger.info("[RoutineStateMachine] Execution paused", {
            contextId: this.contextId,
        });
        return true;
    }

    /**
     * Resume the state machine (only valid when paused or suspended)
     */
    async resume(): Promise<boolean> {
        const currentState = this.getState();
        if (currentState !== RunState.PAUSED && currentState !== RunState.SUSPENDED) {
            throw new Error(`Cannot resume from state ${currentState}`);
        }

        await this.transitionTo(RunState.RUNNING);
        logger.info("[RoutineStateMachine] Execution resumed", {
            contextId: this.contextId,
        });
        return true;
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
                this.parentSwarmId,
            );

            await this.runContextManager.releaseToSwarm(
                this.currentAllocation.swarmId,
                this.contextId,
                this.resourceUsage,
            );
        }

        // Store failure details in routine context
        if (this.runContextManager) {
            try {
                const currentContext = await this.runContextManager.getRunContext(this.contextId);
                const updatedContext = {
                    ...currentContext,
                    lastError: error,
                    resourceUsage: {
                        ...currentContext.resourceUsage,
                        creditsUsed: this.resourceUsage.creditsUsed,
                        durationMs: this.resourceUsage.durationMs,
                        memoryUsedMB: this.resourceUsage.memoryUsedMB,
                        stepsExecuted: this.resourceUsage.stepsExecuted,
                    },
                    lastStateTransition: {
                        state: RunState.FAILED,
                        timestamp: new Date().toISOString(),
                        error,
                    },
                };
                await this.runContextManager.updateRunContext(this.contextId, updatedContext);
            } catch (contextError) {
                logger.warn("[RoutineStateMachine] Failed to update run context on failure", {
                    contextId: this.contextId,
                    error: contextError instanceof Error ? contextError.message : String(contextError),
                });
            }
        }

        // Notify swarm of routine failure (minimal update)
        if (this.swarmContextManager && this.parentSwarmId) {
            try {
                await this.swarmContextManager.updateContext(this.parentSwarmId, {
                    chatConfig: {
                        __version: "1.0",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: Date.now(),
                            lastProcessingCycleEndedAt: null,
                        },
                        records: [{
                            id: generatePK().toString(),
                            routine_id: await this.getRoutineId() || "unknown",
                            routine_name: "Failed Routine",
                            params: {},
                            output_resource_ids: [],
                            caller_bot_id: "tier2-routine-executor",
                            created_at: Date.now().toString(),
                        }],
                    },
                });
            } catch (swarmError) {
                logger.warn("[RoutineStateMachine] Failed to notify swarm of routine failure", {
                    swarmId: this.parentSwarmId,
                    error: swarmError instanceof Error ? swarmError.message : String(swarmError),
                });
            }
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

        // Process outputOperations if present (before state transition)
        await this.processOutputOperations(result);

        await this.transitionTo(RunState.COMPLETED);

        // Emit completion event and release resources
        if (this.runContextManager && this.currentAllocation) {
            await this.runContextManager.emitRunCompleted(
                this.contextId,
                result || { success: true },
                this.resourceUsage,
                this.parentSwarmId,
            );

            await this.runContextManager.releaseToSwarm(
                this.currentAllocation.swarmId,
                this.contextId,
                this.resourceUsage,
            );
        }

        // Store completion details and results in routine context
        if (this.runContextManager) {
            try {
                const currentContext = await this.runContextManager.getRunContext(this.contextId);
                const updatedContext = {
                    ...currentContext,
                    outputs: {
                        ...currentContext.outputs,
                        ...result,
                    },
                    resourceUsage: {
                        ...currentContext.resourceUsage,
                        creditsUsed: this.resourceUsage.creditsUsed,
                        durationMs: this.resourceUsage.durationMs,
                        memoryUsedMB: this.resourceUsage.memoryUsedMB,
                        stepsExecuted: this.resourceUsage.stepsExecuted,
                    },
                    progress: {
                        ...currentContext.progress,
                        percentComplete: 100,
                        currentStepId: null,
                    },
                    lastStateTransition: {
                        state: RunState.COMPLETED,
                        timestamp: new Date().toISOString(),
                        result,
                    },
                };
                await this.runContextManager.updateRunContext(this.contextId, updatedContext);
            } catch (contextError) {
                logger.warn("[RoutineStateMachine] Failed to update run context on completion", {
                    contextId: this.contextId,
                    error: contextError instanceof Error ? contextError.message : String(contextError),
                });
            }
        }

        // Notify swarm of routine completion with summary (not full execution details)
        if (this.swarmContextManager && this.parentSwarmId) {
            try {
                await this.swarmContextManager.updateContext(this.parentSwarmId, {
                    chatConfig: {
                        __version: "1.0",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: Date.now(),
                            lastProcessingCycleEndedAt: Date.now(),
                        },
                        records: [{
                            id: generatePK().toString(),
                            routine_id: await this.getRoutineId() || "unknown",
                            routine_name: "Completed Routine",
                            params: {},
                            output_resource_ids: result && Object.keys(result).length > 0 ? Object.keys(result) : [],
                            caller_bot_id: "tier2-routine-executor",
                            created_at: Date.now().toString(),
                        }],
                    },
                });

                logger.debug("[RoutineStateMachine] Notified swarm of routine completion", {
                    swarmId: this.parentSwarmId,
                    routineExecutionId: this.contextId,
                    resourcesConsumed: this.resourceUsage,
                });
            } catch (swarmError) {
                logger.warn("[RoutineStateMachine] Failed to notify swarm of routine completion", {
                    swarmId: this.parentSwarmId,
                    error: swarmError instanceof Error ? swarmError.message : String(swarmError),
                });
            }
        }

        logger.info("[RoutineStateMachine] Execution completed", {
            contextId: this.contextId,
            resourceUsage: this.resourceUsage,
        });
    }

    /**
     * Process outputOperations from agent behavior to update blackboard
     */
    private async processOutputOperations(result: any): Promise<void> {
        // Type assertion to access our custom metadata field
        const context = this.executionRequest?.context as any;
        if (!context?.outputOperationsMetadata?.outputOperations) {
            return; // No outputOperations to process
        }

        if (!this.swarmContextManager || !this.parentSwarmId) {
            logger.warn("[RoutineStateMachine] Cannot process outputOperations: missing swarm context", {
                contextId: this.contextId,
                hasSwarmManager: !!this.swarmContextManager,
                hasParentSwarmId: !!this.parentSwarmId,
            });
            return;
        }

        const outputOperations = context.outputOperationsMetadata.outputOperations as RoutineAction["outputOperations"];

        if (!outputOperations) {
            return; // Additional safety check
        }

        try {
            // Build BlackboardPatch from outputOperations
            const blackboardPatch: BlackboardPatch = {
                set: [],
                append: [],
                increment: [],
                merge: [],
                deepMerge: [],
            };

            // Process append operations
            if (outputOperations.append) {
                for (const operation of outputOperations.append) {
                    try {
                        const value = getDotNotationValue(result, operation.routineOutput);
                        if (Array.isArray(value)) {
                            blackboardPatch.append!.push({
                                id: operation.blackboardId,
                                values: value,
                            });
                        } else {
                            logger.warn("[RoutineStateMachine] Append operation: value is not an array", {
                                routineOutput: operation.routineOutput,
                                blackboardId: operation.blackboardId,
                                valueType: typeof value,
                            });
                        }
                    } catch (error) {
                        logger.warn("[RoutineStateMachine] Failed to extract value for append operation", {
                            routineOutput: operation.routineOutput,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

            // Process increment operations
            if (outputOperations.increment) {
                for (const operation of outputOperations.increment) {
                    try {
                        const value = getDotNotationValue(result, operation.routineOutput);
                        if (typeof value === "number") {
                            blackboardPatch.increment!.push({
                                id: operation.blackboardId,
                                amount: value,
                            });
                        } else {
                            logger.warn("[RoutineStateMachine] Increment operation: value is not a number", {
                                routineOutput: operation.routineOutput,
                                blackboardId: operation.blackboardId,
                                valueType: typeof value,
                            });
                        }
                    } catch (error) {
                        logger.warn("[RoutineStateMachine] Failed to extract value for increment operation", {
                            routineOutput: operation.routineOutput,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

            // Process merge operations
            if (outputOperations.merge) {
                for (const operation of outputOperations.merge) {
                    try {
                        const value = getDotNotationValue(result, operation.routineOutput);
                        if (value && typeof value === "object" && !Array.isArray(value)) {
                            blackboardPatch.merge!.push({
                                id: operation.blackboardId,
                                object: value as Record<string, unknown>,
                            });
                        } else {
                            logger.warn("[RoutineStateMachine] Merge operation: value is not an object", {
                                routineOutput: operation.routineOutput,
                                blackboardId: operation.blackboardId,
                                valueType: typeof value,
                            });
                        }
                    } catch (error) {
                        logger.warn("[RoutineStateMachine] Failed to extract value for merge operation", {
                            routineOutput: operation.routineOutput,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

            // Process deepMerge operations
            if (outputOperations.deepMerge) {
                for (const operation of outputOperations.deepMerge) {
                    try {
                        const value = getDotNotationValue(result, operation.routineOutput);
                        if (value && typeof value === "object" && !Array.isArray(value)) {
                            blackboardPatch.deepMerge!.push({
                                id: operation.blackboardId,
                                object: value as Record<string, unknown>,
                            });
                        } else {
                            logger.warn("[RoutineStateMachine] DeepMerge operation: value is not an object", {
                                routineOutput: operation.routineOutput,
                                blackboardId: operation.blackboardId,
                                valueType: typeof value,
                            });
                        }
                    } catch (error) {
                        logger.warn("[RoutineStateMachine] Failed to extract value for deepMerge operation", {
                            routineOutput: operation.routineOutput,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

            // Process set operations
            if (outputOperations.set) {
                for (const operation of outputOperations.set) {
                    try {
                        const value = getDotNotationValue(result, operation.routineOutput);
                        blackboardPatch.set!.push({
                            id: operation.blackboardId,
                            value,
                            created_at: new Date().toISOString(),
                        });
                    } catch (error) {
                        logger.warn("[RoutineStateMachine] Failed to extract value for set operation", {
                            routineOutput: operation.routineOutput,
                            error: error instanceof Error ? error.message : String(error),
                        });
                    }
                }
            }

            // Apply blackboard updates using existing swarm context infrastructure
            if (this.hasBlackboardOperations(blackboardPatch)) {
                // Get current context to access blackboard
                const currentContext = await this.swarmContextManager.getContext(this.parentSwarmId);
                if (!currentContext) {
                    throw new Error(`Swarm context not found for ID: ${this.parentSwarmId}`);
                }

                // Apply blackboard operations to current blackboard
                const updatedBlackboard = await this.applyBlackboardOperations(
                    currentContext.chatConfig.blackboard || [],
                    blackboardPatch,
                );

                // Update swarm context with modified blackboard
                await this.swarmContextManager.updateContext(this.parentSwarmId, {
                    chatConfig: {
                        ...currentContext.chatConfig,
                        blackboard: updatedBlackboard,
                    },
                }, `OutputOperations from routine ${this.contextId}`);

                logger.info("[RoutineStateMachine] OutputOperations processed successfully", {
                    contextId: this.contextId,
                    operationCounts: {
                        append: blackboardPatch.append?.length || 0,
                        increment: blackboardPatch.increment?.length || 0,
                        merge: blackboardPatch.merge?.length || 0,
                        deepMerge: blackboardPatch.deepMerge?.length || 0,
                        set: blackboardPatch.set?.length || 0,
                    },
                });
            } else {
                logger.debug("[RoutineStateMachine] No valid outputOperations to apply", {
                    contextId: this.contextId,
                });
            }

        } catch (error) {
            logger.error("[RoutineStateMachine] Failed to process outputOperations", {
                contextId: this.contextId,
                error: error instanceof Error ? error.message : String(error),
            });
            // Don't throw - outputOperations failure shouldn't crash routine completion
        }
    }

    /**
     * Check if blackboard patch has any operations to apply
     */
    private hasBlackboardOperations(patch: BlackboardPatch): boolean {
        return !!(
            (patch.set && patch.set.length > 0) ||
            (patch.append && patch.append.length > 0) ||
            (patch.increment && patch.increment.length > 0) ||
            (patch.merge && patch.merge.length > 0) ||
            (patch.deepMerge && patch.deepMerge.length > 0)
        );
    }

    /**
     * Apply blackboard operations to existing blackboard items
     * Uses the same logic as the MCP updateSwarmSharedState tool
     */
    private async applyBlackboardOperations(
        currentBlackboard: BlackboardItem[],
        patch: BlackboardPatch,
    ): Promise<BlackboardItem[]> {
        // Convert blackboard array to map for easier manipulation, preserving created_at
        const blackboardMap = new Map<string, { value: unknown; created_at: string }>();
        for (const item of currentBlackboard) {
            blackboardMap.set(item.id, { value: item.value, created_at: item.created_at });
        }

        const currentTimestamp = new Date().toISOString();

        // Apply set operations (overwrite values)
        if (patch.set) {
            for (const item of patch.set) {
                const existingItem = blackboardMap.get(item.id);
                blackboardMap.set(item.id, {
                    value: item.value,
                    created_at: existingItem?.created_at || currentTimestamp,
                });
            }
        }

        // Apply append operations (append to arrays)
        if (patch.append) {
            for (const op of patch.append) {
                const existing = blackboardMap.get(op.id);
                if (existing === undefined) {
                    blackboardMap.set(op.id, {
                        value: [...op.values],
                        created_at: currentTimestamp,
                    });
                } else if (Array.isArray(existing.value)) {
                    blackboardMap.set(op.id, {
                        value: [...existing.value, ...op.values],
                        created_at: existing.created_at,
                    });
                } else {
                    logger.warn(`[RoutineStateMachine] Blackboard item "${op.id}" is not an array. Cannot append values.`, {
                        existingType: typeof existing.value,
                        existingValue: existing.value,
                    });
                }
            }
        }

        // Apply increment operations (add to numbers)
        if (patch.increment) {
            for (const op of patch.increment) {
                const existing = blackboardMap.get(op.id);
                if (existing === undefined) {
                    blackboardMap.set(op.id, {
                        value: op.amount,
                        created_at: currentTimestamp,
                    });
                } else if (typeof existing.value === "number") {
                    blackboardMap.set(op.id, {
                        value: existing.value + op.amount,
                        created_at: existing.created_at,
                    });
                } else {
                    logger.warn(`[RoutineStateMachine] Blackboard item "${op.id}" is not a number. Cannot increment.`, {
                        existingType: typeof existing.value,
                        existingValue: existing.value,
                    });
                }
            }
        }

        // Apply merge operations (shallow merge objects)
        if (patch.merge) {
            for (const op of patch.merge) {
                const existing = blackboardMap.get(op.id);
                if (existing === undefined) {
                    blackboardMap.set(op.id, {
                        value: { ...op.object },
                        created_at: currentTimestamp,
                    });
                } else if (existing.value && typeof existing.value === "object" && !Array.isArray(existing.value)) {
                    blackboardMap.set(op.id, {
                        value: { ...existing.value as Record<string, unknown>, ...op.object },
                        created_at: existing.created_at,
                    });
                } else {
                    logger.warn(`[RoutineStateMachine] Blackboard item "${op.id}" is not an object. Cannot merge.`, {
                        existingType: typeof existing.value,
                        existingValue: existing.value,
                    });
                }
            }
        }

        // Apply deepMerge operations (recursive merge objects)
        if (patch.deepMerge) {
            for (const op of patch.deepMerge) {
                const existing = blackboardMap.get(op.id);
                if (existing === undefined) {
                    blackboardMap.set(op.id, {
                        value: this.deepMergeObjects({}, op.object),
                        created_at: currentTimestamp,
                    });
                } else if (existing.value && typeof existing.value === "object" && !Array.isArray(existing.value)) {
                    blackboardMap.set(op.id, {
                        value: this.deepMergeObjects(existing.value as Record<string, unknown>, op.object),
                        created_at: existing.created_at,
                    });
                } else {
                    logger.warn(`[RoutineStateMachine] Blackboard item "${op.id}" is not an object. Cannot deep merge.`, {
                        existingType: typeof existing.value,
                        existingValue: existing.value,
                    });
                }
            }
        }

        // Convert back to BlackboardItem array
        const updatedBlackboard: BlackboardItem[] = [];
        for (const [id, item] of blackboardMap.entries()) {
            updatedBlackboard.push({ id, value: item.value, created_at: item.created_at });
        }

        return updatedBlackboard;
    }

    /**
     * Deep merge two objects recursively
     * Uses the same logic as the MCP tool
     */
    private deepMergeObjects(target: Record<string, unknown>, source: Record<string, unknown>): Record<string, unknown> {
        const result = { ...target };

        for (const [key, value] of Object.entries(source)) {
            if (value && typeof value === "object" && !Array.isArray(value)) {
                if (result[key] && typeof result[key] === "object" && !Array.isArray(result[key])) {
                    result[key] = this.deepMergeObjects(result[key] as Record<string, unknown>, value as Record<string, unknown>);
                } else {
                    result[key] = { ...value as Record<string, unknown> };
                }
            } else {
                result[key] = value;
            }
        }

        return result;
    }

    /**
     * Get routine ID from run context
     */
    private async getRoutineId(): Promise<string | null> {
        if (!this.runContextManager) return null;

        try {
            const runContext = await this.runContextManager.getRunContext(this.contextId);
            return runContext?.routineId || null;
        } catch (error) {
            logger.debug("[RoutineStateMachine] Could not get routine ID from run context", {
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
        const current = BigInt(this.resourceUsage.creditsUsed);
        const additional = BigInt(credits);
        this.resourceUsage.creditsUsed = (current + additional).toString();
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
    public getResourceUsage(): ExecutionResourceUsage {
        this.updateResourceUsage();
        return { ...this.resourceUsage };
    }

    // ============================================
    // ManagedTaskStateMachine Implementation
    // ============================================

    /**
     * Get associated user ID for this routine execution
     */
    public getAssociatedUserId(): string | undefined {
        return this.userId;
    }

    // ============================================
    // BaseStateMachine Abstract Method Implementation
    // ============================================

    /**
     * Get event patterns this state machine should subscribe to
     * @returns Array of event patterns
     */
    protected getEventPatterns(): Array<{ pattern: string }> {
        const patterns: Array<{ pattern: string }> = [
            // Run events
            { pattern: `run/${this.contextId}/*` },
            { pattern: `step/${this.contextId}/*` },
        ];

        // Add swarm-specific patterns if part of a swarm
        if (this.swarmContextManager) {
            // Extract swarmId from context if available
            patterns.push(
                { pattern: `chat/${this.contextId}/safety/*` },
                { pattern: `swarm/${this.contextId}/safety/*` },
            );
        }

        return patterns;
    }

    /**
     * Determine if this state machine instance should handle a specific event
     * @param event - The event to check
     * @returns true if this instance should process the event
     */
    protected shouldHandleEvent(event: ServiceEvent): boolean {
        const eventData = event.data as any;

        // Check execution context
        if (eventData.executionId && eventData.executionId !== this.contextId) {
            return false;
        }
        if (eventData.runId && eventData.runId !== this.contextId) {
            return false;
        }

        // For safety events in swarm context
        if (this.swarmContextManager && event.type.includes("safety")) {
            return eventData.swarmId === this.contextId ||
                eventData.chatId === this.contextId;
        }

        return true;
    }

    /**
     * Process a single event from the event queue
     */
    protected async processEvent(event: ServiceEvent): Promise<void> {
        logger.debug("[RoutineStateMachine] Processing event", {
            contextId: this.contextId,
            eventType: event.type,
            eventId: event.id,
        });

        // Handle routine-specific events based on patterns
        if (event.type.startsWith("safety/")) {
            if (event.type.includes("emergency_stop")) {
                await this.handleEmergencyStop(event);
            }
        } else if (event.type.startsWith("run/")) {
            if (event.type.includes("/task/ready")) {
                await this.handleTaskReady(event);
            } else if (event.type.includes("/task/completed")) {
                await this.handleTaskCompleted(event);
            } else if (event.type.includes("/task/failed")) {
                await this.handleTaskFailed(event);
            }
        } else if (event.type.startsWith("step/")) {
            await this.handleStepEvent(event);
        } else if (event.type === "user/cancellation/requested") {
            await this.handleCancellationRequest(event);
        } else {
            logger.debug("[RoutineStateMachine] Unhandled event type", {
                contextId: this.contextId,
                eventType: event.type,
            });
        }
    }

    /**
     * Called when entering idle state
     */
    protected async onIdle(): Promise<void> {
        logger.debug("[RoutineStateMachine] Entered idle state", {
            contextId: this.contextId,
            runState: this.getState(),
        });

        // Check if there's work pending
        if (this.getState() === RunState.READY) {
            // Could check for pending tasks here
            logger.debug("[RoutineStateMachine] Ready to execute", {
                contextId: this.contextId,
            });
        }
    }

    /**
     * Called when pausing
     */
    protected async onPause(): Promise<void> {
        logger.info("[RoutineStateMachine] Pausing execution", {
            contextId: this.contextId,
            runState: this.getState(),
        });

        // Save pause information in run context
        if (this.runContextManager) {
            try {
                const currentContext = await this.runContextManager.getRunContext(this.contextId);
                const updatedContext = {
                    ...currentContext,
                    lastStateTransition: {
                        state: RunState.PAUSED,
                        timestamp: new Date().toISOString(),
                        reason: "user_requested",
                    },
                };
                await this.runContextManager.updateRunContext(this.contextId, updatedContext);
            } catch (error) {
                logger.warn("[RoutineStateMachine] Failed to update run context on pause", {
                    contextId: this.contextId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
    }

    /**
     * Called when resuming
     */
    protected async onResume(): Promise<void> {
        logger.info("[RoutineStateMachine] Resuming execution", {
            contextId: this.contextId,
            runState: this.getState(),
        });

        // Update resume information in run context
        if (this.runContextManager) {
            try {
                const currentContext = await this.runContextManager.getRunContext(this.contextId);
                const updatedContext = {
                    ...currentContext,
                    lastStateTransition: {
                        state: RunState.RUNNING,
                        timestamp: new Date().toISOString(),
                        reason: "resumed",
                    },
                };
                await this.runContextManager.updateRunContext(this.contextId, updatedContext);
            } catch (error) {
                logger.warn("[RoutineStateMachine] Failed to update run context on resume", {
                    contextId: this.contextId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
    }

    /**
     * Called when stopping - handles routine-specific cleanup and returns final state data
     */
    protected async onStop(mode: "graceful" | "force", reason?: string): Promise<unknown> {
        logger.info("[RoutineStateMachine] Stopping execution", {
            contextId: this.contextId,
            mode,
            reason,
            runState: this.getState(),
        });

        // Update routine context with stop details
        if (this.runContextManager) {
            try {
                const currentContext = await this.runContextManager.getRunContext(this.contextId);
                const updatedContext = {
                    ...currentContext,
                    lastStateTransition: {
                        state: RunState.CANCELLED,
                        timestamp: new Date().toISOString(),
                        reason: reason || `Manual stop (${mode})`,
                    },
                };
                await this.runContextManager.updateRunContext(this.contextId, updatedContext);
            } catch (error) {
                logger.warn("[RoutineStateMachine] Failed to update run context on stop", {
                    contextId: this.contextId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        // Notify swarm of routine cancellation (minimal update)
        if (this.swarmContextManager && this.parentSwarmId) {
            try {
                await this.swarmContextManager.updateContext(this.parentSwarmId, {
                    chatConfig: {
                        __version: "1.0",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: Date.now(),
                            lastProcessingCycleEndedAt: null,
                        },
                        records: [{
                            id: generatePK().toString(),
                            routine_id: await this.getRoutineId() || "unknown",
                            routine_name: "Cancelled Routine",
                            params: { cancelled: true, reason: reason || `Manual stop (${mode})` },
                            output_resource_ids: [],
                            caller_bot_id: "tier2-routine-executor",
                            created_at: Date.now().toString(),
                        }],
                    },
                });
            } catch (error) {
                logger.error("[RoutineStateMachine] Failed to notify swarm of cancellation", {
                    contextId: this.contextId,
                    parentSwarmId: this.parentSwarmId,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        // Clean up resources
        if (this.runContextManager && this.currentAllocation) {
            await this.runContextManager.releaseToSwarm(
                this.currentAllocation.swarmId,
                this.contextId,
                this.getResourceUsage(),
            );
        }

        // Return final state
        return {
            contextId: this.contextId,
            finalRunState: this.getState(),
            resourceUsage: this.getResourceUsage(),
            stoppedAt: new Date().toISOString(),
            stopReason: reason,
        };
    }

    /**
     * Determine if an error is fatal
     */
    protected async isErrorFatal(error: unknown, event: ServiceEvent): Promise<boolean> {
        // For now, consider all errors during RUNNING state as fatal
        // In a more sophisticated implementation, this could check error types
        if (this.getState() === RunState.RUNNING) {
            logger.error("[RoutineStateMachine] Fatal error during execution", {
                contextId: this.contextId,
                error: error instanceof Error ? error.message : String(error),
                eventType: event.type,
            });
            return true;
        }

        // Non-fatal during other states
        return false;
    }

    // ============================================
    // Event Handlers
    // ============================================

    private async handleEmergencyStop(event: ServiceEvent): Promise<void> {
        logger.warn("[RoutineStateMachine] Emergency stop requested", {
            contextId: this.contextId,
            eventData: event.data,
        });
        await this.fail("Emergency stop requested");
    }

    private async handleTaskReady(event: ServiceEvent): Promise<void> {
        logger.debug("[RoutineStateMachine] Task ready", {
            contextId: this.contextId,
            eventData: event.data,
        });
        // Implementation would start task execution
    }

    private async handleTaskCompleted(event: ServiceEvent): Promise<void> {
        logger.debug("[RoutineStateMachine] Task completed", {
            contextId: this.contextId,
            eventData: event.data,
        });
        // Implementation would update progress
    }

    private async handleTaskFailed(event: ServiceEvent): Promise<void> {
        logger.error("[RoutineStateMachine] Task failed", {
            contextId: this.contextId,
            eventData: event.data,
        });
        // Implementation would handle task failure
    }

    private async handleCancellationRequest(event: ServiceEvent): Promise<void> {
        logger.info("[RoutineStateMachine] Cancellation requested", {
            contextId: this.contextId,
            eventData: event.data,
        });
        await this.cancel();
    }

    private async handleStepEvent(event: ServiceEvent): Promise<void> {
        logger.debug("[RoutineStateMachine] Handling step event", {
            contextId: this.contextId,
            eventType: event.type,
            eventData: event.data,
        });

        // Update step count
        if (event.type.includes("/completed")) {
            this.incrementStepCount();
        }

        // Handle step-specific events as needed
        // This could be expanded to track individual step progress
    }

    /**
     * Cancel the routine execution
     */
    async cancel(): Promise<void> {
        if (await this.isTerminal()) {
            logger.debug("[RoutineStateMachine] Already in terminal state, ignoring cancel", {
                contextId: this.contextId,
                currentState: this.getState(),
            });
            return;
        }

        await this.transitionTo(RunState.CANCELLED);
        logger.info("[RoutineStateMachine] Execution cancelled", {
            contextId: this.contextId,
        });
    }
} 
