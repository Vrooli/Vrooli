import { type Logger } from "winston";
import {
    type Run,
    type RunState,
    type RunConfig,
    type RunProgress,
    type RunContext,
    type RunEvent,
    type RunEventType,
    type Location,
    type Navigator,
    type Routine,
    type StepStatus,
    type BranchExecution,
    RunState as RunStateEnum,
    RunEventType as RunEventTypeEnum,
    generatePk,
} from "@vrooli/shared";
import { EventBus } from "../../cross-cutting/eventBus.js";
import { type IRunStateStore } from "../state/runStateStore.js";
import { NavigatorRegistry } from "../navigation/navigatorRegistry.js";
import { ContextManager } from "../context/contextManager.js";
import { BranchCoordinator } from "./branchCoordinator.js";
import { StepExecutor } from "./stepExecutor.js";
import { CheckpointManager } from "../persistence/checkpointManager.js";
import { PerformanceMonitor } from "../intelligence/performanceMonitor.js";
import { PathOptimizer } from "../intelligence/pathOptimizer.js";
import { MOISEGate } from "../validation/moiseGate.js";

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
 * TierTwoRunStateMachine - Universal routine execution engine
 * 
 * This is the heart of Vrooli's ability to execute diverse automation routines.
 * It represents a universal routine execution engine that's completely agnostic
 * to the underlying automation platform.
 * 
 * Key capabilities:
 * - Navigator-agnostic execution (BPMN, Langchain, Temporal, Native)
 * - Parallel branch coordination with resource awareness
 * - State management and checkpoint/recovery
 * - MOISE+ permission validation for every step
 * - Performance monitoring and path optimization
 * - Event-driven orchestration for external integrations
 * 
 * This creates an unprecedented universal automation ecosystem where workflows
 * from different platforms can share and execute each other's routines.
 */
export class TierTwoRunStateMachine {
    private readonly logger: Logger;
    private readonly eventBus: EventBus;
    private readonly stateStore: IRunStateStore;
    private readonly navigatorRegistry: NavigatorRegistry;
    private readonly contextManager: ContextManager;
    private readonly branchCoordinator: BranchCoordinator;
    private readonly stepExecutor: StepExecutor;
    private readonly checkpointManager: CheckpointManager;
    private readonly performanceMonitor: PerformanceMonitor;
    private readonly pathOptimizer: PathOptimizer;
    private readonly moiseGate: MOISEGate;

    // Active runs
    private readonly activeRuns: Map<string, Run> = new Map();
    private readonly runNavigators: Map<string, Navigator> = new Map();

    constructor(
        logger: Logger,
        eventBus: EventBus,
        stateStore: IRunStateStore,
    ) {
        this.logger = logger;
        this.eventBus = eventBus;
        this.stateStore = stateStore;

        // Initialize components
        this.navigatorRegistry = new NavigatorRegistry(logger);
        this.contextManager = new ContextManager(stateStore, logger);
        this.branchCoordinator = new BranchCoordinator(eventBus, logger);
        this.stepExecutor = new StepExecutor(eventBus, logger);
        this.checkpointManager = new CheckpointManager(stateStore, logger);
        this.performanceMonitor = new PerformanceMonitor(eventBus, logger);
        this.pathOptimizer = new PathOptimizer(logger);
        this.moiseGate = new MOISEGate(logger);

        // Subscribe to tier 3 events
        this.subscribeToExecutionEvents();
    }

    /**
     * Creates and initializes a new run
     */
    async createRun(params: RunInitParams): Promise<Run> {
        const runId = generatePk();
        
        this.logger.info(`[RunStateMachine] Creating new run`, {
            runId,
            routineId: params.routineId,
            userId: params.userId,
        });

        try {
            // Load routine
            const routine = await this.loadRoutine(params.routineId);
            
            // Select appropriate navigator
            const navigator = this.navigatorRegistry.getNavigator(routine.type);
            if (!navigator.canNavigate(routine.definition)) {
                throw new Error(`Navigator ${routine.type} cannot handle routine ${routine.id}`);
            }

            // Create run configuration
            const config: RunConfig = {
                maxSteps: 1000,
                maxDepth: 10,
                maxTime: 3600000, // 1 hour
                maxCost: 100,
                parallelization: true,
                checkpointInterval: 300000, // 5 minutes
                recoveryStrategy: "retry",
                ...params.config,
            };

            // Initialize progress
            const startLocation = navigator.getStartLocation(routine.definition);
            const progress: RunProgress = {
                totalSteps: this.estimateTotalSteps(routine, navigator),
                completedSteps: 0,
                failedSteps: 0,
                skippedSteps: 0,
                currentLocation: startLocation,
                locationStack: {
                    locations: [startLocation],
                    depth: 1,
                },
                branches: [],
            };

            // Initialize context
            const context = await this.contextManager.createContext({
                variables: {},
                blackboard: {},
                scopes: [{
                    id: "root",
                    name: "Root Scope",
                    variables: params.inputs,
                }],
            });

            // Create run
            const run: Run = {
                id: runId,
                routineId: params.routineId,
                state: RunStateEnum.UNINITIALIZED,
                config,
                progress,
                context,
                startedAt: new Date(),
            };

            // Store in state store
            await this.stateStore.createRun(runId, {
                id: runId,
                routineId: params.routineId,
                userId: params.userId,
                inputs: params.inputs,
                metadata: {
                    parentRunId: params.parentRunId,
                    swarmId: params.swarmId,
                },
                createdAt: new Date(),
                updatedAt: new Date(),
            });

            // Cache run and navigator
            this.activeRuns.set(runId, run);
            this.runNavigators.set(runId, navigator);

            // Emit creation event
            await this.emitRunEvent({
                type: RunEventTypeEnum.RUN_STARTED,
                timestamp: new Date(),
                runId,
                metadata: {
                    routineId: params.routineId,
                    routineType: routine.type,
                },
            });

            // Transition to LOADING state
            await this.transitionState(run, RunStateEnum.LOADING);

            return run;

        } catch (error) {
            this.logger.error(`[RunStateMachine] Failed to create run`, {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Starts execution of a run
     */
    async startRun(runId: string): Promise<void> {
        const run = this.activeRuns.get(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        this.logger.info(`[RunStateMachine] Starting run`, { runId });

        try {
            // Validate run can be started
            if (![RunStateEnum.READY, RunStateEnum.PAUSED].includes(run.state)) {
                throw new Error(`Cannot start run in state ${run.state}`);
            }

            // Transition to RUNNING
            await this.transitionState(run, RunStateEnum.RUNNING);

            // Start execution loop
            this.executeRunLoop(run).catch(error => {
                this.logger.error(`[RunStateMachine] Run loop error`, {
                    runId,
                    error: error instanceof Error ? error.message : String(error),
                });
                this.handleRunFailure(run, error);
            });

        } catch (error) {
            this.logger.error(`[RunStateMachine] Failed to start run`, {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Pauses a running run
     */
    async pauseRun(runId: string): Promise<void> {
        const run = this.activeRuns.get(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        if (run.state !== RunStateEnum.RUNNING) {
            throw new Error(`Cannot pause run in state ${run.state}`);
        }

        this.logger.info(`[RunStateMachine] Pausing run`, { runId });

        await this.transitionState(run, RunStateEnum.PAUSED);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_PAUSED,
            timestamp: new Date(),
            runId,
        });
    }

    /**
     * Resumes a paused run
     */
    async resumeRun(runId: string): Promise<void> {
        const run = this.activeRuns.get(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        if (run.state !== RunStateEnum.PAUSED) {
            throw new Error(`Cannot resume run in state ${run.state}`);
        }

        this.logger.info(`[RunStateMachine] Resuming run`, { runId });

        await this.transitionState(run, RunStateEnum.RUNNING);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_RESUMED,
            timestamp: new Date(),
            runId,
        });

        // Resume execution loop
        this.executeRunLoop(run).catch(error => {
            this.logger.error(`[RunStateMachine] Run loop error on resume`, {
                runId,
                error: error instanceof Error ? error.message : String(error),
            });
            this.handleRunFailure(run, error);
        });
    }

    /**
     * Cancels a run
     */
    async cancelRun(runId: string): Promise<void> {
        const run = this.activeRuns.get(runId);
        if (!run) {
            throw new Error(`Run ${runId} not found`);
        }

        this.logger.info(`[RunStateMachine] Cancelling run`, { runId });

        await this.transitionState(run, RunStateEnum.CANCELLED);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_CANCELLED,
            timestamp: new Date(),
            runId,
        });

        this.cleanupRun(run);
    }

    /**
     * Main execution loop for a run
     */
    private async executeRunLoop(run: Run): Promise<void> {
        const navigator = this.runNavigators.get(run.id);
        if (!navigator) {
            throw new Error(`Navigator not found for run ${run.id}`);
        }

        while (run.state === RunStateEnum.RUNNING) {
            try {
                // Check resource limits
                if (await this.checkResourceLimits(run)) {
                    this.logger.warn(`[RunStateMachine] Resource limits exceeded`, {
                        runId: run.id,
                    });
                    await this.suspendRun(run, "Resource limits exceeded");
                    break;
                }

                // Create checkpoint if needed
                if (await this.shouldCreateCheckpoint(run)) {
                    await this.checkpointManager.createCheckpoint(run);
                }

                // Get current location
                const currentLocation = run.progress.currentLocation;

                // Check if at end location
                if (navigator.isEndLocation(currentLocation)) {
                    await this.completeRun(run);
                    break;
                }

                // Get next locations (could be multiple for parallel branches)
                const nextLocations = navigator.getNextLocations(
                    currentLocation,
                    run.context.variables,
                );

                if (nextLocations.length === 0) {
                    // Dead end - complete run
                    await this.completeRun(run);
                    break;
                }

                // Handle branching
                if (nextLocations.length > 1) {
                    await this.executeBranches(run, nextLocations, navigator);
                } else {
                    await this.executeStep(run, nextLocations[0], navigator);
                }

                // Check if paused
                if (run.state !== RunStateEnum.RUNNING) {
                    break;
                }

            } catch (error) {
                this.logger.error(`[RunStateMachine] Error in run loop`, {
                    runId: run.id,
                    error: error instanceof Error ? error.message : String(error),
                });

                if (run.config.recoveryStrategy === "retry") {
                    // Retry from last checkpoint
                    const recovered = await this.recoverFromCheckpoint(run);
                    if (!recovered) {
                        await this.handleRunFailure(run, error);
                        break;
                    }
                } else {
                    await this.handleRunFailure(run, error);
                    break;
                }
            }
        }
    }

    /**
     * Executes a single step
     */
    private async executeStep(
        run: Run,
        location: Location,
        navigator: Navigator,
    ): Promise<void> {
        const stepInfo = navigator.getStepInfo(location);
        const stepId = stepInfo.id;

        this.logger.debug(`[RunStateMachine] Executing step`, {
            runId: run.id,
            stepId,
            location,
        });

        // Check MOISE+ permissions
        const permitted = await this.moiseGate.checkPermission(
            run.id,
            stepId,
            run.context,
        );

        if (!permitted) {
            this.logger.warn(`[RunStateMachine] Step not permitted by MOISE+`, {
                runId: run.id,
                stepId,
            });
            await this.skipStep(run, location, "Permission denied");
            return;
        }

        // Emit step start event
        await this.emitRunEvent({
            type: RunEventTypeEnum.STEP_STARTED,
            timestamp: new Date(),
            runId: run.id,
            stepId,
            metadata: { location, stepInfo },
        });

        try {
            // Execute step via Tier 3
            const result = await this.stepExecutor.executeStep({
                runId: run.id,
                stepId,
                stepInfo,
                context: run.context,
                location,
            });

            // Update context with results
            if (result.outputs) {
                await this.contextManager.updateVariables(
                    run.id,
                    result.outputs,
                );
            }

            // Update progress
            run.progress.completedSteps++;
            run.progress.currentLocation = location;

            // Update state store
            await this.stateStore.recordStepExecution(run.id, {
                stepId,
                state: "completed",
                startedAt: new Date(),
                completedAt: new Date(),
                result: result.outputs,
            });

            // Emit completion event
            await this.emitRunEvent({
                type: RunEventTypeEnum.STEP_COMPLETED,
                timestamp: new Date(),
                runId: run.id,
                stepId,
                metadata: { result },
            });

        } catch (error) {
            this.logger.error(`[RunStateMachine] Step execution failed`, {
                runId: run.id,
                stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            // Update progress
            run.progress.failedSteps++;

            // Record failure
            await this.stateStore.recordStepExecution(run.id, {
                stepId,
                state: "failed",
                startedAt: new Date(),
                completedAt: new Date(),
                error: error instanceof Error ? error.message : String(error),
            });

            // Emit failure event
            await this.emitRunEvent({
                type: RunEventTypeEnum.STEP_FAILED,
                timestamp: new Date(),
                runId: run.id,
                stepId,
                metadata: { error: error instanceof Error ? error.message : String(error) },
            });

            // Handle based on recovery strategy
            if (run.config.recoveryStrategy === "skip") {
                await this.skipStep(run, location, "Step failed");
            } else {
                throw error;
            }
        }
    }

    /**
     * Executes parallel branches
     */
    private async executeBranches(
        run: Run,
        locations: Location[],
        navigator: Navigator,
    ): Promise<void> {
        this.logger.debug(`[RunStateMachine] Executing ${locations.length} branches`, {
            runId: run.id,
        });

        const branches = await this.branchCoordinator.createBranches(
            run.id,
            locations,
            run.config.parallelization,
        );

        // Execute branches
        const results = await this.branchCoordinator.executeBranches(
            run,
            branches,
            navigator,
            this.stepExecutor,
        );

        // Merge results
        const mergedContext = await this.branchCoordinator.mergeBranchResults(
            run.context,
            results,
        );

        // Update run context
        run.context = mergedContext;
        await this.contextManager.updateContext(run.id, mergedContext);

        // Update progress
        for (const result of results) {
            run.progress.completedSteps += result.completedSteps;
            run.progress.failedSteps += result.failedSteps;
            run.progress.skippedSteps += result.skippedSteps;
        }

        // Continue from convergence point
        const convergenceLocation = this.findConvergenceLocation(locations, navigator);
        if (convergenceLocation) {
            run.progress.currentLocation = convergenceLocation;
        }
    }

    /**
     * Helper methods
     */
    private async transitionState(run: Run, newState: RunState): Promise<void> {
        const oldState = run.state;
        run.state = newState;
        
        await this.stateStore.updateRunState(run.id, newState);
        
        this.logger.info(`[RunStateMachine] State transition`, {
            runId: run.id,
            oldState,
            newState,
        });
    }

    private async loadRoutine(routineId: string): Promise<Routine> {
        // TODO: Load from routine store
        return {
            id: routineId,
            type: "native",
            version: "1.0.0",
            name: "Test Routine",
            definition: {},
            metadata: {
                author: "system",
                createdAt: new Date(),
                updatedAt: new Date(),
                tags: [],
                complexity: "simple",
            },
        };
    }

    private estimateTotalSteps(routine: Routine, navigator: Navigator): number {
        // TODO: Implement step counting based on routine definition
        return 10;
    }

    private async checkResourceLimits(run: Run): Promise<boolean> {
        // Check step limit
        if (run.config.maxSteps && run.progress.completedSteps >= run.config.maxSteps) {
            return true;
        }

        // Check time limit
        if (run.config.maxTime && run.startedAt) {
            const elapsed = Date.now() - run.startedAt.getTime();
            if (elapsed >= run.config.maxTime) {
                return true;
            }
        }

        // Check cost limit
        // TODO: Implement cost tracking

        return false;
    }

    private async shouldCreateCheckpoint(run: Run): Promise<boolean> {
        if (!run.config.checkpointInterval) {
            return false;
        }

        const lastCheckpoint = await this.checkpointManager.getLastCheckpoint(run.id);
        if (!lastCheckpoint) {
            return true;
        }

        const elapsed = Date.now() - lastCheckpoint.timestamp.getTime();
        return elapsed >= run.config.checkpointInterval;
    }

    private async suspendRun(run: Run, reason: string): Promise<void> {
        await this.transitionState(run, RunStateEnum.SUSPENDED);
        run.error = reason;
        
        this.logger.warn(`[RunStateMachine] Run suspended`, {
            runId: run.id,
            reason,
        });
    }

    private async completeRun(run: Run): Promise<void> {
        run.completedAt = new Date();
        await this.transitionState(run, RunStateEnum.COMPLETED);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_COMPLETED,
            timestamp: new Date(),
            runId: run.id,
            metadata: {
                duration: run.completedAt.getTime() - (run.startedAt?.getTime() || 0),
                stepsCompleted: run.progress.completedSteps,
            },
        });

        this.cleanupRun(run);
    }

    private async handleRunFailure(run: Run, error: unknown): Promise<void> {
        run.completedAt = new Date();
        run.error = error instanceof Error ? error.message : String(error);
        await this.transitionState(run, RunStateEnum.FAILED);

        await this.emitRunEvent({
            type: RunEventTypeEnum.RUN_FAILED,
            timestamp: new Date(),
            runId: run.id,
            metadata: {
                error: run.error,
                failedSteps: run.progress.failedSteps,
            },
        });

        this.cleanupRun(run);
    }

    private async recoverFromCheckpoint(run: Run): Promise<boolean> {
        try {
            const checkpoint = await this.checkpointManager.getLastCheckpoint(run.id);
            if (!checkpoint) {
                return false;
            }

            await this.checkpointManager.restoreCheckpoint(run, checkpoint);
            return true;
        } catch (error) {
            this.logger.error(`[RunStateMachine] Failed to recover from checkpoint`, {
                runId: run.id,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    private async skipStep(run: Run, location: Location, reason: string): Promise<void> {
        run.progress.skippedSteps++;
        
        await this.emitRunEvent({
            type: RunEventTypeEnum.STEP_SKIPPED,
            timestamp: new Date(),
            runId: run.id,
            stepId: location.nodeId,
            metadata: { reason },
        });
    }

    private findConvergenceLocation(
        branches: Location[],
        navigator: Navigator,
    ): Location | null {
        // TODO: Implement branch convergence detection
        return branches[0];
    }

    private cleanupRun(run: Run): void {
        this.activeRuns.delete(run.id);
        this.runNavigators.delete(run.id);
    }

    private async emitRunEvent(event: RunEvent): Promise<void> {
        await this.eventBus.publish("run.events", event);
    }

    private subscribeToExecutionEvents(): void {
        // Subscribe to Tier 3 execution results
        this.eventBus.subscribe("execution.outputs", async (event) => {
            // Handle step completion from Tier 3
            this.logger.debug(`[RunStateMachine] Received execution output`, event);
        });

        // Subscribe to performance events
        this.eventBus.subscribe("telemetry.perf", async (event) => {
            // Feed to performance monitor
            await this.performanceMonitor.recordMetric(event);
        });
    }
}