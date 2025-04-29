import { ModelType, RoutineVersion, RunStatus } from "../api/types.js";
import { PassableLogger } from "../consts/commonTypes.js";
import { generatePKString } from "../id/snowflake.js";
import { RoutineVersionConfig } from "../shape/configs/routine.js";
import { RunProgressConfig } from "../shape/configs/run.js";
import { getTranslation } from "../translations/translationTools.js";
import { BranchManager } from "./branch.js";
import { DEFAULT_LOOP_DELAY_MULTIPLIER, DEFAULT_MAX_LOOP_DELAY_MS, DEFAULT_MAX_RUN_CREDITS, DEFAULT_ON_BRANCH_FAILURE, LATEST_RUN_CONFIG_VERSION, MAX_MAIN_LOOP_ITERATIONS, MAX_PARALLEL_BRANCHES } from "./consts.js";
import { SubroutineContextManager } from "./context.js";
import { SubroutineExecutor } from "./executor.js";
import { RunLimitsManager } from "./limits.js";
import { RunLoader } from "./loader.js";
import { BpmnNavigator, NavigatorFactory, NavigatorRegistry } from "./navigator.js";
import { RunNotifier } from "./notifier.js";
import { PathSelectionHandler } from "./pathSelection.js";
import { RunPersistence } from "./persistence.js";
import { BranchLocationDataMap, BranchStatus, ConcurrencyMode, InitializedRunState, Location, RunConfig, RunIdentifier, RunProgress, RunRequestLimits, RunStateMachineServices, RunStateMachineState, RunStatusChangeReason, RunTriggeredBy, StateMachineStatus, SubroutineContext } from "./types.js";

/** Maps graph types to navigators */
const navigatorRegistry: NavigatorRegistry = {
    "BPMN-2.0": new BpmnNavigator(),
};

/** Factory for navigators. Allows traversal of any supported graph type. */
export const navigatorFactory = new NavigatorFactory(navigatorRegistry);

/**
 * Constructor options for the state machine.
 */
type RunStateMachineProps = {
    /** Loader for fetching routine information */
    loader: RunLoader,
    /** Logger for debugging */
    logger: PassableLogger,
    /** Finds the appropriate navigator for stepping through the current routine graph */
    navigatorFactory: NavigatorFactory,
    /** 
     * Handles emitting payloads to display the run progress.
     * 
     * Needed in the following cases:
     * - Running the state machine only in the server
     *   - Example: Automated runs
     * - Running the state machine in the server, while it's also being run in the client
     *   - Example: Following along to an automated run from the client, but haven't switched to manual mode yet
     * - Running the state machine only in the client
     *   - Examples: Manual runs, tutorial run (doesn't execute subroutines)
     * - Running the state machine for testing
     *   - Example: Unit tests
     */
    notifier: RunNotifier | null,
    /** Strategy for making path selections when there are multiple possible next steps */
    pathSelectionHandler: PathSelectionHandler,
    /** Persistence service for saving/loading run progress */
    persistence: RunPersistence,
    /** Executor for running subroutines */
    subroutineExecutor: SubroutineExecutor,
}

/**
 * High-level state machine for running a routine.
 * 
 * Orchestrates transitions between steps, handles drilling down 
 * into sub-routines, and updates the run progress.
 * 
 * Supported features:
 * - Graph-agnostic: Can run any type of routine or project (e.g. BPMN, DMN), as long as a navigator is provided
 * - Fetch-agnostic: You provide implementations for fetching and storing data, so that this works in any environment (e.g. server, browser, testing)
 * - Metrics and limits: Keeps track of how much time, credits, and steps have been used, and stops the run when limits are reached
 * - Context management: Collects context data from each step, and uses it to make decisions and generate responses
 * - Configurable decision strategies: You can choose how to handle ambiguous decisions at runtime
 * - Parallel execution of branches: If the graph supports it, you can run multiple branches in parallel
 * - Trigger support: Can register triggers to run when certain conditions are met, such as waiting a certain amount of time to continue
 * - Pause/resume support: Can pause a run and resume it later, without losing progress
 * - Bot persona/custom instructions: Can provide additional context to the bot (if applicable), such as the persona or custom instructions
 * - Logging: Provides a logger for debugging
 * 
 * NOTE: Assumes only run is active at a time. If you need to run multiple routines at once, you'll need to create multiple instances of this class.
 * 
 * To use:
 * 1) Create new state machine instance with desired configuration, including limits and stop conditions
 * 2) `const runProgress = await initNewRun(...)` or `const run = await initExistingRun(...)`
 * 3) `await runUntilDone()` or `await runOneIteration()`
 * 4) If you need the run stopped for any reason, call `stopRun()`
 */
export class RunStateMachine {
    /**
     * Configurable classes for different parts of the run.
     * Pass in different implementations depending on the run environment (e.g. server or browser).
     */
    private services: RunStateMachineServices;

    /**
     * The machine's run state (basically everything that's not services)
     */
    private state: RunStateMachineState;

    constructor({
        loader,
        logger,
        navigatorFactory,
        notifier,
        pathSelectionHandler,
        persistence,
        subroutineExecutor,
    }: RunStateMachineProps) {
        this.services = {
            limitsManager: new RunLimitsManager(logger),
            loader,
            logger,
            navigatorFactory,
            notifier,
            pathSelectionHandler,
            persistence,
            subroutineExecutor,
        };
        this.state = {
            pendingConfigUpdate: null,
            runIdentifier: null,
            status: StateMachineStatus.Idle,
            userData: null,
        };
    }

    /**
     * Makes sure the state machine has been properly initialized.
     * 
     * @param state The state of the state machine
     * @throws If the state machine does not have the data it needs to run
     */
    private assertInitialized(state: RunStateMachineState): asserts state is InitializedRunState {
        if (!state.runIdentifier || !state.userData) {
            throw new Error("State machine is not fully initialized.");
        }
    }

    /**
     * Makes sure the state machine status is one of the allowed values.
     * 
     * @param trace A string to include in the error message
     * @param acceptedStatuses The allowed statuses
     * @throws If the status is not one of the allowed values
     */
    private assertValidStatus(trace: string, ...acceptedStatuses: StateMachineStatus[]) {
        if (!acceptedStatuses.includes(this.state.status)) {
            throw new Error(`${trace}: State machine is in an invalid status: ${this.state.status}`);
        }
    }

    /**
     * Sets up a new run, but does not execute it yet.
     *
     * @param startLocations One or more starting "Location" objects
     * @param config Configuration for this run
     * @param userData Session data for the user running the routine
     */
    public async initNewRun(
        startLocations: Location[],
        config: RunConfig,
        userData: RunTriggeredBy,
    ): Promise<RunProgress> {
        this.assertValidStatus("initNewRun", StateMachineStatus.Idle);
        this.state.status = StateMachineStatus.Transitioning;

        // If no start locations are provided, we can't start the run
        const firstLocation = startLocations[0];
        if (!firstLocation) {
            throw new Error("No start locations provided");
        }

        // Make sure all start locations are in the same object
        const startingAt = { __typename: firstLocation.__typename, objectId: firstLocation.objectId };
        if (startLocations.some(loc => loc.__typename !== startingAt.__typename || loc.objectId !== startingAt.objectId)) {
            throw new Error("All start locations must be in the same object");
        }

        // Get the object for the start location
        const startLocationData = await this.services.loader.loadLocation(firstLocation, config);
        if (!startLocationData) {
            throw new Error(`Start object not found: ${firstLocation.objectId}`);
        }
        const { object: startObject } = startLocationData;

        const runId = generatePKString();
        this.services.logger.info(`Initializing new run with ID ${runId}`);
        const { description, instructions, name } = getTranslation(startObject as { translations: { language: string, name: string, description: string, instructions?: string }[] | null | undefined }, userData.languages, true);
        const subcontext: SubroutineContext = {
            ...SubroutineContextManager.initializeContext(),
            // The routine we're starting in is the current task. When we drill down into subroutines, 
            // we'll set this information as the overallTask
            currentTask: {
                description: description ?? "",
                instructions: instructions ?? "",
                name: name ?? "",
            },
        };
        const subroutineInstanceId = BranchManager.generateSubroutineInstanceId(startObject.id);
        const run: RunProgress = {
            __version: LATEST_RUN_CONFIG_VERSION,
            branches: RunProgressConfig.defaultBranches(),
            config,
            decisions: RunProgressConfig.defaultDecisions(),
            metrics: RunProgressConfig.defaultMetrics(),
            name: name ?? "",
            owner: startObject.root?.owner ?? { id: userData.id, __typename: "User" as const },
            runId,
            runOnObjectId: startObject.id,
            schedule: null,
            steps: [],
            status: RunStatus.InProgress,
            subcontexts: { [subroutineInstanceId]: subcontext },
            type: startObject.__typename === "ProjectVersion"
                ? "RunProject" as const
                : "RunRoutine" as const,
        };
        // Determine if parallel execution is allowed
        let supportsParallelExecution = false;
        if (startObject.__typename === ModelType.RoutineVersion) {
            const routineConfig = RoutineVersionConfig.deserialize(startObject as RoutineVersion, this.services.logger, { useFallbacks: true });
            const graphConfig = routineConfig.graph;
            const navigator = graphConfig ? this.services.navigatorFactory.getNavigator(graphConfig.__type) : null;
            supportsParallelExecution = navigator?.supportsParallelExecution ?? false;
        }
        // Create a separate branch for each start location
        run.branches = BranchManager.forkBranches(null, startLocations, subroutineInstanceId, supportsParallelExecution);

        // Finalize initialization
        return await this.finalizeInit(run, userData);
    }

    /**
     * Sets up an existing run, but does not execute it yet.
     *
     * @param runInfo The type and ID for this run
     * @param userData Session data for the user running the routine
     */
    public async initExistingRun(
        runInfo: RunIdentifier,
        userData: RunTriggeredBy,
    ): Promise<RunProgress> {
        this.assertValidStatus("initExistingRun", StateMachineStatus.Idle);
        this.state.status = StateMachineStatus.Transitioning;

        // Load the existing run data
        const run = await this.services.persistence.loadProgress(runInfo, userData, this.services.logger);

        // If the run doesn't have any active or waiting branches, we can't resume it
        const activeBranches = run.branches.filter(b => b.status === BranchStatus.Active);
        const waitingBranches = run.branches.filter(b => b.status === BranchStatus.Waiting);
        if (activeBranches.length === 0 && waitingBranches.length === 0) {
            throw new Error(`Run ${runInfo.runId} has no active or waiting branches`);
        }
        // If the run is already completed or failed, we can't run it
        if (run.status !== RunStatus.InProgress) {
            throw new Error(`Run ${runInfo.runId} has already finished`);
        }

        // Update the run status
        run.status = RunStatus.InProgress;

        // Finalize initialization using the helper function
        return await this.finalizeInit(run, userData);
    }

    /**
     * Finalizes the initialization process for a run by persisting the run,
     * updating related metadata, and setting the state machine's status to Initialized.
     *
     * @param run - The run progress object to finalize.
     * @param userData - Session data for the user running the routine.
     * @returns The finalized run progress object.
     */
    private async finalizeInit(run: RunProgress, userData: RunTriggeredBy): Promise<RunProgress> {
        // Update the state machine's state
        this.state.runIdentifier = { type: run.type, runId: run.runId };
        this.state.userData = userData;

        // Store the run in the database
        this.services.persistence.saveProgress(run);
        await this.services.persistence.finalizeSave(false);

        // Update path decisions
        this.services.pathSelectionHandler.updateDecisionOptions(run);

        this.state.status = StateMachineStatus.Initialized;

        return run;
    }

    /**
     * Performs a single iteration of the main loop.
     * 
     * This means executing and advancing active branches, checking limits, 
     * and updating the run progress.
     * 
     * @param run The run to run an iteration of
     * @param startTimeMs The start time of the run
     * @returns The run status change reason, if the run should stop. Otherwise, undefined.
     */
    private async runIteration(
        run: RunProgress,
        startTimeMs: number,
    ): Promise<RunStatusChangeReason | undefined> {
        this.assertInitialized(this.state);

        // Apply any configuration changes
        this.applyConfigUpdate(run);

        const { limits, loopConfig, onBranchFailure } = run.config;

        const delayMs = loopConfig.currentLoopDelayMs ?? loopConfig.loopDelayMs ?? 0;
        const delayMultiplier = loopConfig.loopDelayMultiplier ?? DEFAULT_LOOP_DELAY_MULTIPLIER;
        const maxLoopDelayMs = loopConfig.maxLoopDelayMs ?? DEFAULT_MAX_LOOP_DELAY_MS;

        // Check time/credits/steps limits (this also updates the run status if limits are reached)
        let runStatusChangeReason = this.services.limitsManager.checkLimits(run, limits, startTimeMs);
        if (runStatusChangeReason) {
            return runStatusChangeReason;
        }

        // Load branch location data as a dictionary for easier access
        const branchLocationDataMap = await BranchManager.loadBranchLocationData(run, this.services);

        // Find active and waiting branches
        const activeBranches = run.branches.filter((b) => b.status === BranchStatus.Active);
        const waitingBranches = run.branches.filter((b) => b.status === BranchStatus.Waiting);

        // If no active branches or waiting branches, we're done
        if (activeBranches.length === 0 && waitingBranches.length === 0) {
            this.services.logger.info("runUntilDone: No active or waiting branches. Setting status to Completed.");
            run.status = RunStatus.Completed;
            return RunStatusChangeReason.Completed;
        }
        // If all branches are waiting, increase the delay to avoid busy-waiting
        if (activeBranches.length === 0 && waitingBranches.length > 0) {
            const newDelay = Math.min(delayMs * delayMultiplier, maxLoopDelayMs);
            if (newDelay !== loopConfig.currentLoopDelayMs) {
                this.services.logger.info("runUntilDone: All branches are waiting. Increasing loop delay.");
                loopConfig.currentLoopDelayMs = newDelay;
            }
        }
        // Otherwise, reset the delay
        else {
            loopConfig.currentLoopDelayMs = loopConfig.loopDelayMs;
        }

        // Decide concurrency mode
        const mode = await this.getConcurrencyMode(run, branchLocationDataMap, limits);

        // Run branches either sequentially or in parallel. 
        if (mode === "Sequential") {
            for (const branch of activeBranches) {
                // Run the branch
                const stepResult = await this.services.subroutineExecutor.executeStep(
                    run,
                    branch,
                    branchLocationDataMap,
                    this.services,
                    this.state,
                );
                // Update the branch and run data
                this.services.subroutineExecutor.updateRunAfterStep(
                    run,
                    branch,
                    stepResult,
                    this.services,
                    this.state,
                );
                // Check limits
                runStatusChangeReason = this.services.limitsManager.checkLimits(run, limits, startTimeMs);
                if (runStatusChangeReason) {
                    return runStatusChangeReason;
                }
            }
            // Update the run progress
            this.services.persistence.saveProgress(run);
        } else {
            const batchSize = MAX_PARALLEL_BRANCHES;
            const totalBranches = activeBranches.length;

            // Process in batches
            for (let i = 0; i < totalBranches; i += batchSize) {
                // Get the current batch of branches to process
                const batch = activeBranches.slice(i, i + batchSize);

                // Execute the current batch in parallel
                const stepResults = await Promise.all(batch.map((branch) => {
                    this.assertInitialized(this.state);
                    return this.services.subroutineExecutor.executeStep(
                        run,
                        branch,
                        branchLocationDataMap,
                        this.services,
                        this.state,
                    );
                }));

                // Aggregate results
                for (const stepResult of stepResults) {
                    const branch = activeBranches.find(b => b.branchId === stepResult.branchId);
                    if (!branch) {
                        this.services.logger.error(`runUntilDone: Branch ${stepResult.branchId} not found in run ${run.runId}`);
                        continue;
                    }
                    // Update the branch and run data
                    this.services.subroutineExecutor.updateRunAfterStep(
                        run,
                        branch,
                        stepResult,
                        this.services,
                        this.state,
                    );
                }

                // Check limits after each batch
                runStatusChangeReason = this.services.limitsManager.checkLimits(run, limits, startTimeMs);
                if (runStatusChangeReason) {
                    return runStatusChangeReason;
                }
            }

            // Update the run progress
            this.services.persistence.saveProgress(run);
        }

        // Advance the branches to the next location(s), including taking branches out of waiting state if necessary
        await BranchManager.advanceBranches(
            run,
            branchLocationDataMap,
            this.services,
        );

        // Check if any branches have failed
        const hasFailedBranch = run.branches.some(b => b.status === BranchStatus.Failed);
        if (hasFailedBranch) {
            // Handle failure based on the configuration
            const behavior = onBranchFailure ?? DEFAULT_ON_BRANCH_FAILURE;
            switch (behavior) {
                case "Continue": {
                    this.services.logger.error("runUntilDone: Some branches failed, but continuing.");
                    break;
                }
                case "Pause": {
                    this.services.logger.error("runUntilDone: Some branches failed, pausing run.");
                    run.status = RunStatus.Paused;
                    return RunStatusChangeReason.AutomaticPause;
                }
                case "Stop": {
                    this.services.logger.error("runUntilDone: Some branches failed, stopping run.");
                    run.status = RunStatus.Failed;
                    return RunStatusChangeReason.AutomaticStop;
                }
            }
        }

        // Remove failed branches so they are no longer processed
        run.branches = run.branches.filter(b => {
            return b.status === BranchStatus.Active || b.status === BranchStatus.Completed || b.status === BranchStatus.Waiting;
        });

        // Remove completed branches only if every branch with the same subroutineInstanceId is completed.
        // This allows us to wait for all branches of a subroutine instance to complete, so that we can do things
        // like merge the subcontext with the parent context.
        //
        // If we did this after every individual branch completed, there might be unexpected behavior.
        // For example, a routine might be set up to override a context variable later in the routine.
        // If we pass the original value to the parent too early, bad things might happen.
        const subroutineInstanceIds = new Set(run.branches.map(b => b.subroutineInstanceId));
        for (const subroutineInstanceId of subroutineInstanceIds) {
            const branchesInSubroutine = run.branches.filter(b => b.subroutineInstanceId === subroutineInstanceId);
            const allBranchesCompleted = branchesInSubroutine.every(b => b.status === BranchStatus.Completed);
            if (allBranchesCompleted) {
                run.branches = run.branches.filter(b => b.subroutineInstanceId !== subroutineInstanceId);
            }
        }

        // If that removal leaves us with zero branches, the run is effectively done
        if (run.branches.length === 0) {
            if (hasFailedBranch) {
                this.services.logger.error("runUntilDone: All branches failed. Setting status to Failed.");
                run.status = RunStatus.Failed;
                return RunStatusChangeReason.Error;
            } else {
                this.services.logger.info("runUntilDone: All branches completed. Setting status to Completed.");
                run.status = RunStatus.Completed;
                return RunStatusChangeReason.Completed;
            }
        }

        // If every remaining branch is waiting, we need to put the run in a waiting state
        if (run.branches.every(b => b.status === BranchStatus.Waiting)) {
            await this.stopRun(RunStatus.Paused);
            return RunStatusChangeReason.AutomaticPause;
        }

        // Let listeners know the run has been updated
        this.services.notifier?.sendProgressUpdate(run, runStatusChangeReason);

        // Apply delay if provided
        if (delayMs && delayMs > 0) {
            await new Promise(resolve => setTimeout(resolve, delayMs));
        }

        // If we made it here, the run can continue
        return undefined;
    }

    //TODO need to handle boundary events during node execution. Should include setting up polling if time-based boundary events exist, setting up event listeners for message-based boundary events, etc.
    /**
     * A top-level loop that runs until the run is complete (or otherwise 
     * cannot progress). It enforces:
     * - a loop limit (to avoid infinite loops),
     * - concurrency limits,
     * - time/credits checks,
     * - dynamic switch between parallel and sequential modes based on 
     *   remaining credits/tokens.
     * 
     * NOTE: Must initialize the run before calling this function.
     */
    public async runUntilDone(): Promise<void> {
        this.assertInitialized(this.state);

        this.assertValidStatus("runUntilDone", StateMachineStatus.Initialized, StateMachineStatus.Paused);
        this.state.status = StateMachineStatus.Running;

        // Get the run
        const run = await this.services.persistence.loadProgress(this.state.runIdentifier, this.state.userData, this.services.logger);

        // Get the start time
        const startTimeMs = this.getOrInitializeStartTime(run);

        // Let listeners know the run has started
        this.services.notifier?.sendProgressUpdate(run, undefined);

        // Main loop
        let loopCount = 0; // Track the number of iterations to avoid infinite loops
        let runStatusChangeReason: RunStatusChangeReason | undefined; // Track the reason for status changes that stop the run
        while (loopCount < MAX_MAIN_LOOP_ITERATIONS && run.status === RunStatus.InProgress) {
            runStatusChangeReason = await this.runIteration(run, startTimeMs);
            if (runStatusChangeReason) {
                break;
            }
            loopCount++;
        }

        // 9. If we exited because of the loop limit
        if (loopCount >= MAX_MAIN_LOOP_ITERATIONS) {
            this.services.logger.error("runUntilDone: Reached max loop iterations. Potential infinite loop?");
            runStatusChangeReason = RunStatusChangeReason.MaxLoops;
            run.status = RunStatus.Failed;
        }

        // Update total time elapsed
        run.metrics.timeElapsed += Date.now() - startTimeMs;

        this.services.logger.info(`runUntilDone: Run ${run.runId} ended with status: ${run.status}`);
        this.services.persistence.saveProgress(run);
        await this.services.persistence.finalizeSave(true); // Clear cache after saving

        // Let listeners know the run has ended
        this.services.notifier?.sendProgressUpdate(run, runStatusChangeReason);
        this.services.notifier?.finalizeSend(); // Send immediately instead of waiting for the throttle

        // Update the state machine status
        this.state.status = run.status === RunStatus.Completed ? StateMachineStatus.Completed : StateMachineStatus.Failed;
    }

    /**
     * Performs a single iteration of the main loop.
     */
    public async runOneIteration(): Promise<void> {
        this.assertInitialized(this.state);

        this.assertValidStatus("runUntilDone", StateMachineStatus.Initialized, StateMachineStatus.Paused);
        this.state.status = StateMachineStatus.Running;

        // Get the run
        const run = await this.services.persistence.loadProgress(this.state.runIdentifier, this.state.userData, this.services.logger);

        // Get the start time
        const startTimeMs = this.getOrInitializeStartTime(run);

        // Run one iteration
        const runStatusChangeReason = await this.runIteration(run, startTimeMs);

        // Update total time elapsed
        run.metrics.timeElapsed += Date.now() - startTimeMs;

        // Pause the run if it's still in progress
        if (run.status === RunStatus.InProgress) {
            await this.stopRun(RunStatus.Paused);
        }

        // Save the run
        this.services.persistence.saveProgress(run);

        // Notify listeners
        this.services.notifier?.sendProgressUpdate(run, runStatusChangeReason);

        // Update the state machine status
        this.state.status = run.status === RunStatus.Paused
            ? StateMachineStatus.Paused
            : run.status === RunStatus.Completed
                ? StateMachineStatus.Completed
                : StateMachineStatus.Failed;
    }

    /**
     * Retrieves the start time (in ms) for the run from run.metrics.startedAt.
     * If it isn’t set, uses the current time and sets it.
     *
     * @param run The current run progress.
     * @returns The start time in milliseconds.
     */
    private getOrInitializeStartTime(run: RunProgress): number {
        if (run.metrics.startedAt) {
            return new Date(run.metrics.startedAt).getTime();
        } else {
            const now = Date.now();
            run.metrics.startedAt = new Date(now);
            return now;
        }
    }


    /**
     * Determines which concurrency mode to use based on the current run state 
     * and available resources.
     * 
     * @param run The current run
     * @param branchLocationDataMap The location data for each branch
     * @param limits The limits for this run
     * @param config The run configuration
     * @returns The concurrency mode to use
     */
    private async getConcurrencyMode(
        run: RunProgress,
        branchLocationDataMap: BranchLocationDataMap,
        limits: RunRequestLimits,
    ): Promise<ConcurrencyMode> {
        // If some of the branches don't support parallel execution, switch to sequential mode
        const parallelExecutionDisabled = run.branches.some(b => b.status === BranchStatus.Active && !b.supportsParallelExecution);
        if (parallelExecutionDisabled) {
            return "Sequential";
        }

        // Collect estimated max credits for each branch
        const costArray = await Promise.all(run.branches.map(async (branch) => {
            const currentObject = branchLocationDataMap[branch.branchId]?.object;
            if (!currentObject || currentObject.__typename !== ModelType.RoutineVersion) {
                return BigInt(0);
            }

            const subcontext = run.subcontexts[branch.subroutineInstanceId];
            if (!subcontext) {
                // Return a large number to ensure sequential mode is used 
                // when we can't correctly estimate the cost
                return BigInt(Number.MAX_SAFE_INTEGER);
            }

            const cost = await this.services.subroutineExecutor.estimateCost(
                branch.subroutineInstanceId,
                currentObject,
                run.config,
            );
            return BigInt(cost);
        }));

        // Sum the costs
        const maxCreditsGeneratedThisIteration = costArray.reduce((acc, cost) => acc + cost, BigInt(0));

        // If the estimated total for this iteration exceeds the remaining credits, switch to sequential mode
        const totalCreditsSoFar = BigInt(run.metrics.creditsSpent);
        const maxCreditsLimit = BigInt(limits.maxCredits ?? DEFAULT_MAX_RUN_CREDITS);
        if ((totalCreditsSoFar + maxCreditsGeneratedThisIteration) > maxCreditsLimit) {
            return "Sequential";
        }

        // Otherwise, use parallel mode
        return "Parallel";
    }

    /**
     * Switch the path decision handler at runtime
     * 
     * @param updatedHandler The new path decision handler to use
     */
    public async setPathSelectionHandler(updatedHandler: PathSelectionHandler) {
        this.assertInitialized(this.state);

        // Update the handler
        this.services.pathSelectionHandler = updatedHandler;

        // Get the run
        const run = await this.services.persistence.loadProgress(this.state.runIdentifier, this.state.userData, this.services.logger);

        // Set the available decisions to the new handler
        this.services.pathSelectionHandler.updateDecisionOptions(run, run.decisions);
    }

    /**
     * Gets the current path selection handler.
     * 
     * @returns The current path selection handler
     */
    public getPathSelectionHandler(): PathSelectionHandler {
        return this.services.pathSelectionHandler;
    }

    /**
     * Initializes a run configuration update. 
     * On the next iteration of the run loop, the run configuration will be updated.
     * 
     * @param config The new configuration for this run
     */
    public updateRunConfig(config: RunConfig): void {
        this.state.pendingConfigUpdate = config;
    }

    /**
     * Applies configuration updates to the run.
     * Does not persist the changes to the database.
     * 
     * @param run The current run progress, containing the configuration to update
     */
    private applyConfigUpdate(run: RunProgress): void {
        if (this.state.pendingConfigUpdate) {
            run.config = this.state.pendingConfigUpdate;
            this.state.pendingConfigUpdate = null;
        }
    }

    /**
     * Stop the run.
     * 
     * The running loop in `runUntilDone` or `runOneIteration` will detect this change 
     * and handle storing the run and updating the state machine status.
     *
     * @param status The new status for the run
     */
    public async stopRun(status: RunStatus.Cancelled | RunStatus.Paused): Promise<void> {
        this.assertInitialized(this.state);

        this.assertValidStatus("stopRun", StateMachineStatus.Running);
        this.state.status = StateMachineStatus.Transitioning;

        const run = await this.services.persistence.loadProgress(this.state.runIdentifier, this.state.userData, this.services.logger);

        if (run.status !== RunStatus.InProgress) {
            this.services.logger.error(`pauseRun: Cannot pause run ${this.state.runIdentifier.runId} because it is not in progress.`);
            return;
        }

        // Update the run status
        run.status = status;
        // NOTE: Don't call saveProgress here. Let the main loop handle it.
        this.services.logger.info(`pauseRun: Run ${this.state.runIdentifier.runId} has been paused.`);
    }
}

//TODO context building for LLM, better project handling, triggers
//TODO support roles. For example, should be possible to assume one role, and let bots assume other roles. This could be useful in a scenario where a bot handles the steps related to researching a topic, and the user does the steps where they read the report and choose what to do next.
//TODO add permissions config for setting up subroutine-level permissions like if objects can be created without confirmation, can be created with a delay (to give you time to cancel), etc.
//TODO add safety config for setting up safety checks like input sanitization, output sanitization, etc.
//TODO need to make sure the user can never go over their max credits, even if they run multiple routines at once.
//TODO would be cool if instead of having an LLM generate inputs, they could sometimes search for inputs instead. For example, if an input is JavaScript code, they could check for existing code that's public, owned by the user/team, etc. and use that instead.
//TODO db updates: data field for chats, data field for chat_messages
//TODO LinkItem input type needs to be found using search instead of AI generation
