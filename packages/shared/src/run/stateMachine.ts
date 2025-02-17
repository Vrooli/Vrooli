import { ModelType, RoutineVersion, RunStatus } from "../api/types.js";
import { PassableLogger } from "../consts/commonTypes.js";
import { uuid } from "../id/uuid.js";
import { getTranslation } from "../translations/translationTools.js";
import { GraphConfig, RoutineVersionConfig } from "./configs/routine.js";
import { RunProgressConfig } from "./configs/run.js";
import { DEFAULT_LOOP_DELAY_MULTIPLIER, DEFAULT_MAX_LOOP_DELAY_MS, DEFAULT_MAX_RUN_CREDITS, DEFAULT_ON_BRANCH_FAILURE, LATEST_RUN_CONFIG_VERSION, MAX_MAIN_LOOP_ITERATIONS, MAX_PARALLEL_BRANCHES } from "./consts.js";
import { SubroutineContextManager } from "./context.js";
import { RunSubroutineResult, SubroutineExecutor } from "./executor.js";
import { RunLoader } from "./loader.js";
import { BpmnNavigator, NavigatorFactory, NavigatorRegistry } from "./navigator.js";
import { RunNotifier } from "./notifier.js";
import { RunPersistence } from "./persistence.js";
import { DecisionStrategy } from "./strategy.js";
import { BranchProgress, BranchStatus, ConcurrencyMode, DeferredDecisionData, IOKey, IOMap, IOValue, Id, Location, LocationData, LocationStack, NodeProgress, RunConfig, RunIdentifier, RunLimitBehavior, RunProgress, RunRequestLimits, RunStatusChangeReason, RunTriggeredBy, SubroutineContext } from "./types.js";

/** Maps graph types to navigators */
const navigatorRegistry: NavigatorRegistry = {
    "BPMN-2.0": new BpmnNavigator(),
};

/** Factory for navigators. Allows traversal of any supported graph type. */
export const navigatorFactory = new NavigatorFactory(navigatorRegistry);

type RunStateMachineConfig = {
    /** Strategy for making decisions when there are multiple possible next steps */
    decisionStrategy: DecisionStrategy,
    /** Loader for fetching routine information */
    loader: RunLoader,
    /** Logger for debugging */
    logger: PassableLogger,
    /** Finds the appropriate navigator for stepping through the current routine graph */
    navigatorFactory: NavigatorFactory,
    /** Handles emitting payloads to display the run progress */
    notifier: RunNotifier | null,
    /** Persistence service for saving/loading run progress */
    persistence: RunPersistence,
    /** Executor for running subroutines */
    subroutineExecutor: SubroutineExecutor,
}

/**
 * A map of branch IDs to their current step's location data.
 */
type BranchLocationDataMap = Record<Id, LocationData & {
    location: Location;
    subcontext: SubroutineContext;
}>;

/**
 * The result of executing a step (i.e. performing the action specified by 
 * the current node, and handling any branching logic).
 * 
 * This is required because steps may be executed in parallel, so we defer
 * updating branch and run progress until all steps have been executed.
 */
type ExecuteStepResult = {
    /** The ID of the branch the step was executed in */
    branchId: Id,
    /** The new status of the branch */
    branchStatus: BranchStatus,
    /** Stats for each node completed during this step */
    completedNodes: NodeProgress[],
    /** The complexity completed during this step */
    complexityCompleted: number,
    /** The total amount of credits spent during this step, as a stringified bigint */
    creditsSpent: string,
    /**
    * Any deferred decisions that need to be made.
    * 
    * This should pause the branch until the decisions are resolved.
    */
    deferredDecisions: DeferredDecisionData[];
    /** Any new branches that should be created as a result of this step */
    newLocations: null | {
        /** The initial subcontext for the new branches */
        initialContext: SubroutineContext,
        /** The locations to move to */
        locations: Location[],
        /** Whether the new branches support parallel execution */
        supportsParallelExecution: boolean,
    }
    /** Information about the subroutine that was run, if any */
    subroutineRun: null | {
        /** Inputs generated for every subroutine that was run */
        inputs: IOMap,
        /** Outputs generated for every subroutine that was run */
        outputs: IOMap,
    }
}

/**
 * High-level state machine for running a routine.
 * 
 * Orchestrates transitions between steps, handles drilling down 
 * into sub-routines, and updates the run progress.
 * 
 * Supported features:
 * - Graph-agnostic: Can run any type of routine or project (e.g. BPMN, DMN), as long as a navigator is provided
 * - Fetch-agnostic: You provide implementations for fetching and storing data, so that this works in any environment
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
 * 3) `await run()`
 * 4) If you need the run stopped for any reason, call `stopRun()`
 */
export class RunStateMachine {
    // Configurable classes for different parts of the run.
    // Pass in different implementations depending on the run environment (e.g. server or browser).
    private decisionStrategy: DecisionStrategy;
    private loader: RunLoader;
    private logger: PassableLogger;
    private navigatorFactory: NavigatorFactory;
    private notifier: RunNotifier | null;
    private persistence: RunPersistence;
    private subroutineExecutor: SubroutineExecutor;

    // Info for identifying the current run
    private runIdentifier: RunIdentifier | null = null;
    // Info about the user running the routine
    private userData: RunTriggeredBy | null = null;
    // Temporary config storage to update the run configuration during the run. 
    // Should be cleared after the config in RunProgress is updated.
    private pendingConfigUpdate: RunConfig | null = null;

    constructor({
        decisionStrategy,
        loader,
        logger,
        navigatorFactory,
        notifier,
        persistence,
        subroutineExecutor,
    }: RunStateMachineConfig) {
        // Set configurable classes
        this.decisionStrategy = decisionStrategy;
        this.loader = loader;
        this.logger = logger;
        this.navigatorFactory = navigatorFactory;
        this.notifier = notifier;
        this.persistence = persistence;
        this.subroutineExecutor = subroutineExecutor;
    }

    /**
     * Asserts the run identifier is set.
     * 
     * @param trace A string to include in the error message
     * @returns The run identifier
     * @throws If the run identifier is not set
     */
    private getRunIdentifier(trace: string): RunIdentifier {
        if (!this.runIdentifier) {
            throw new Error(`${trace}: Run identifier not set. Make sure to call initNewRun or initExistingRun before running the state machine.`);
        }
        return this.runIdentifier;
    }

    /**
     * Asserts the user data is set.
     * 
     * @param trace A string to include in the error message
     * @returns The user data
     * @throws If the user data is not set
     */
    private getUserData(trace: string): RunTriggeredBy {
        if (!this.userData) {
            throw new Error(`${trace}: User data not set. Make sure to call initNewRun or initExistingRun before running the state machine.`);
        }
        return this.userData;
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
        const startLocationData = await this.loader.loadLocation(firstLocation, config);
        if (!startLocationData) {
            throw new Error(`Start object not found: ${firstLocation.objectId}`);
        }
        const { object: startObject } = startLocationData;

        const runId = uuid();
        this.logger.info(`Initializing new run with ID ${runId}`);
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
        const subroutineInstanceId = this.generateSubroutineInstanceId(startObject.id);
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
            const routineConfig = RoutineVersionConfig.deserialize(startObject as RoutineVersion, this.logger, { useFallbacks: true });
            const graphConfig = routineConfig.graph;
            const navigator = graphConfig ? this.navigatorFactory.getNavigator(graphConfig.__type) : null;
            supportsParallelExecution = navigator?.supportsParallelExecution ?? false;
        }
        // Create a separate branch for each start location
        run.branches = this.forkBranches(null, startLocations, subroutineInstanceId, supportsParallelExecution);

        // Store the run in the database
        this.persistence.saveProgress(run);
        await this.persistence.finalizeSave(false);

        // Store the run identifer and user data
        this.runIdentifier = { type: run.type, runId: run.runId };
        this.userData = userData;

        // Store decisions in the decision strategy
        this.decisionStrategy.updateDecisionOptions(run);

        // Return the run
        return run;
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
        // Load the existing run data
        const run = await this.persistence.loadProgress(runInfo, userData, this.logger);

        // If the run isn't new and the existing run data is not found, we have a problem
        if (!run) {
            throw new Error(`Run ID ${runInfo.runId} not found in database`);
        }
        // If the run doesn't have any active or waiting branches, we can't resume it
        const activeBranches = run.branches.filter(b => b.status === BranchStatus.Active);
        const waitingBranches = run.branches.filter(b => b.status === BranchStatus.Waiting);
        if (activeBranches.length === 0 && waitingBranches.length === 0) {
            throw new Error(`Run ${runInfo.runId} has no active or waiting branches`);
        }
        // Make sure the run is in progress
        run.status = RunStatus.InProgress;

        // Store the run configuration
        this.persistence.saveProgress(run);
        await this.persistence.finalizeSave(false);

        // Store the run identifer and user data
        this.runIdentifier = { type: run.type, runId: run.runId };
        this.userData = userData;

        // Store decisions in the decision strategy
        this.decisionStrategy.updateDecisionOptions(run);

        // Return the run
        return run;
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
    public async run(): Promise<void> {
        // Get run identifier and user data
        const runIdentifier = this.getRunIdentifier("run");
        const userData = this.getUserData("run");

        // Get the run
        const run = await this.persistence.loadProgress(runIdentifier, userData, this.logger);
        if (!run) {
            throw new Error(`Run ID ${runIdentifier.runId} not found in database`);
        }

        const startTimeMs = Date.now();

        // Let listeners know the run has started
        this.notifier?.sendProgressUpdate(run, undefined);

        /**
         * Helper function to update the branch and run progress after a branch is stepped.
         * 
         * We use an arrow function to get access to the state machine's methods.
         */
        const updateRunAfterStep = (
            run: RunProgress,
            branch: BranchProgress,
            stepResult: ExecuteStepResult,
        ): void => {
            // Update status
            branch.status = stepResult.branchStatus;
            // Update metrics
            run.metrics.complexityCompleted += stepResult.complexityCompleted;
            run.metrics.creditsSpent = (BigInt(run.metrics.creditsSpent) + BigInt(stepResult.creditsSpent)).toString();
            run.metrics.stepsRun++;
            // Add deferred decisions
            if (stepResult.deferredDecisions && stepResult.deferredDecisions.length > 0) {
                // Update the decision strategy
                const updatedDecisions = this.decisionStrategy.updateDecisionOptions(run, stepResult.deferredDecisions);
                // Update the run progress
                run.decisions = updatedDecisions;
                // Send a decision request to the client
                for (const decision of stepResult.deferredDecisions) {
                    this.notifier?.sendDecisionRequest(runIdentifier.runId, run.type, decision);
                }
                // If the branch is still active, put it in a waiting state
                if (branch.status === BranchStatus.Active) {
                    branch.status = BranchStatus.Waiting;
                }
            }
            // Add new branches (should only happen when a multi-step subroutine is encountered)
            if (stepResult.newLocations && stepResult.newLocations.locations.length > 0) {
                const { initialContext, locations, supportsParallelExecution } = stepResult.newLocations;
                // Find the ID of the multi-step subroutine all of the new branches are part of
                const firstLocation = locations[0];
                if (!firstLocation) {
                    throw new Error("No location found in step result");
                }
                const subroutineId = firstLocation.objectId;
                // Use a new subroutine instance ID for the new branches
                const subroutineInstanceId = this.generateSubroutineInstanceId(subroutineId);
                // Add it to the current branch, so we can link the new branches to it
                branch.childSubroutineInstanceId = subroutineInstanceId;
                // Initialize the subcontext for the new branches
                run.subcontexts[subroutineInstanceId] = initialContext;
                // Create the new branches
                const newBranches = this.forkBranches(branch, locations, subroutineInstanceId, supportsParallelExecution, branch.locationStack);
                // Add them to the run branches
                run.branches.push(...newBranches);
                // Put the current branch in a waiting state until the new branches are completed
                branch.status = BranchStatus.Waiting;
            }
            // Update subcontext (should only happen when a single-step subroutine is encountered)
            if (stepResult.subroutineRun) {
                const { inputs, outputs } = stepResult.subroutineRun;
                const existingSubcontext = run.subcontexts[branch.subroutineInstanceId];
                run.subcontexts[branch.subroutineInstanceId] = SubroutineContextManager.updateContext(existingSubcontext, inputs, outputs);
            }
        };

        // Main loop
        let loopCount = 0; // Track the number of iterations to avoid infinite loops
        let runStatusChangeReason: RunStatusChangeReason | undefined; // Track the reason for status changes that stop the run
        mainLoop: // Label for breaking out of the loop when needed
        while (loopCount < MAX_MAIN_LOOP_ITERATIONS && run.status === RunStatus.InProgress) {
            // Apply any configuration changes
            this.applyConfigUpdate(run);

            const { limits, loopConfig, onBranchFailure } = run.config;

            const delayMs = loopConfig.currentLoopDelayMs ?? loopConfig.loopDelayMs ?? 0;
            const delayMultiplier = loopConfig.loopDelayMultiplier ?? DEFAULT_LOOP_DELAY_MULTIPLIER;
            const maxLoopDelayMs = loopConfig.maxLoopDelayMs ?? DEFAULT_MAX_LOOP_DELAY_MS;

            // Check time/credits/steps limits (this also updates the run status if limits are reached)
            runStatusChangeReason = this.checkLimits(run, limits, startTimeMs);
            if (runStatusChangeReason) {
                break mainLoop; // Exit the loop
            }

            // Load branch location data as a dictionary for easier access
            const branchLocationDataMap = await this.loadBranchLocationData(run);

            // Find active and waiting branches
            const activeBranches = run.branches.filter((b) => b.status === BranchStatus.Active);
            const waitingBranches = run.branches.filter((b) => b.status === BranchStatus.Waiting);

            // If no active branches or waiting branches, we're done
            if (activeBranches.length === 0 && waitingBranches.length === 0) {
                this.logger.info("runUntilDone: No active or waiting branches. Setting status to Completed.");
                run.status = RunStatus.Completed;
                break mainLoop; // Exit the loop
            }
            // If all branches are waiting, increase the delay to avoid busy-waiting
            if (activeBranches.length === 0 && waitingBranches.length > 0) {
                const newDelay = Math.min(delayMs * delayMultiplier, maxLoopDelayMs);
                if (newDelay !== loopConfig.currentLoopDelayMs) {
                    this.logger.info("runUntilDone: All branches are waiting. Increasing loop delay.");
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
                    const stepResult = await this.executeStep(branch, branchLocationDataMap[branch.branchId], run.config);
                    // Update the branch and run data
                    updateRunAfterStep(run, branch, stepResult);
                    // Check limits
                    runStatusChangeReason = this.checkLimits(run, limits, startTimeMs);
                    if (runStatusChangeReason) {
                        break mainLoop; // Break out of main loop. If we didn't use a label, we'd only break out of the inner loop
                    }
                }
                // Update the run progress
                this.persistence.saveProgress(run);
            } else {
                const batchSize = MAX_PARALLEL_BRANCHES;
                const totalBranches = activeBranches.length;

                // Process in batches
                for (let i = 0; i < totalBranches; i += batchSize) {
                    // Get the current batch of branches to process
                    const batch = activeBranches.slice(i, i + batchSize);

                    // Execute the current batch in parallel
                    const stepResults = await Promise.all(batch.map((branch) =>
                        this.executeStep(branch, branchLocationDataMap[branch.branchId], run.config),
                    ));

                    // Aggregate results
                    for (const stepResult of stepResults) {
                        const branch = activeBranches.find(b => b.branchId === stepResult.branchId);
                        if (!branch) {
                            this.logger.error(`runUntilDone: Branch ${stepResult.branchId} not found in run ${run.runId}`);
                            continue;
                        }
                        // Update the branch and run data
                        updateRunAfterStep(run, branch, stepResult);
                    }

                    // Check limits after each batch
                    runStatusChangeReason = this.checkLimits(run, limits, startTimeMs);
                    if (runStatusChangeReason) {
                        break mainLoop; // Break out of main loop. If we didn't use a label, we'd only break out of the inner loop
                    }
                }

                // Update the run progress
                this.persistence.saveProgress(run);
            }

            // Advance the branches to the next location(s), including taking branches out of waiting state if necessary
            await this.advanceBranches(run, branchLocationDataMap);

            // Check if any branches have failed
            const hasFailedBranch = run.branches.some(b => b.status === BranchStatus.Failed);
            if (hasFailedBranch) {
                // Handle failure based on the configuration
                const behavior = onBranchFailure ?? DEFAULT_ON_BRANCH_FAILURE;
                switch (behavior) {
                    case "Continue":
                        this.logger.error("runUntilDone: Some branches failed, but continuing.");
                        break;
                    case "Pause":
                        this.logger.error("runUntilDone: Some branches failed, pausing run.");
                        run.status = RunStatus.Paused;
                        break mainLoop;
                    case "Stop":
                        this.logger.error("runUntilDone: Some branches failed, stopping run.");
                        run.status = RunStatus.Failed;
                        break mainLoop;
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
                run.status = hasFailedBranch ? RunStatus.Failed : RunStatus.Completed;
                if (hasFailedBranch) {
                    this.logger.error("runUntilDone: All branches failed. Setting status to Failed.");
                } else {
                    this.logger.info("runUntilDone: All branches completed. Setting status to Completed.");
                }
                break mainLoop;
            }

            // If every remaining branch is waiting, we need to put the run in a waiting state
            if (run.branches.every(b => b.status === BranchStatus.Waiting)) {
                await this.stopRun(RunStatus.Paused);
                break mainLoop;
            }

            // Let listeners know the run has been updated
            this.notifier?.sendProgressUpdate(run, runStatusChangeReason);

            // Apply delay if provided
            if (delayMs && delayMs > 0) {
                await new Promise(resolve => setTimeout(resolve, delayMs));
            }

            loopCount++;
        }

        // 9. If we exited because of the loop limit
        if (loopCount >= MAX_MAIN_LOOP_ITERATIONS) {
            this.logger.error("runUntilDone: Reached max loop iterations. Potential infinite loop?");
            runStatusChangeReason = RunStatusChangeReason.MaxLoops;
            run.status = RunStatus.Failed;
        }

        // Calculate total time elapsed
        const endTimeMs = Date.now();
        run.metrics.timeElapsed += endTimeMs - startTimeMs;

        this.logger.info(`runUntilDone: Run ${run.runId} ended with status: ${run.status}`);
        this.persistence.saveProgress(run);
        await this.persistence.finalizeSave(true); // Clear cache after saving

        // Let listeners know the run has ended
        this.notifier?.sendProgressUpdate(run, runStatusChangeReason);
        this.notifier?.finalizeSend(); // Send immediately instead of waiting for the throttle
    }

    //TODO need method to run one branch at a time for the UI.

    /**
     * Loads location data for all branches, and converts them into a dictionary.
     * Any branch that can't be loaded is marked as failed.
     * 
     * @param run The current run progress
     * @returns A dictionary of branch IDs to location data
     */
    private async loadBranchLocationData(
        run: RunProgress,
    ): Promise<BranchLocationDataMap> {
        const branchLocationDataArray = await Promise.all(
            run.branches.map(async (branch) => {
                const currentLocation = branch.locationStack[branch.locationStack.length - 1];
                if (!currentLocation) {
                    this.logger.error(`loadBranchLocationData: Branch ${branch.branchId} has no current location`);
                    branch.status = BranchStatus.Failed;
                    return null;
                }
                const currentLocationData = await this.loader.loadLocation(currentLocation, run.config);
                if (!currentLocationData) {
                    this.logger.error(`loadBranchLocationData: Location data not found for location ${currentLocation.locationId}`);
                    branch.status = BranchStatus.Failed;
                    return null;
                }
                const currentSubcontext = run.subcontexts[branch.subroutineInstanceId];
                if (!currentSubcontext) {
                    this.logger.error(`loadBranchLocationData: Subcontext not found for branch ${branch.branchId}`);
                    branch.status = BranchStatus.Failed;
                    return null;
                }

                return {
                    branchId: branch.branchId,
                    currentLocationData: {
                        location: currentLocation,
                        subcontext: currentSubcontext,
                        ...currentLocationData,
                    },
                };
            }),
        );
        return branchLocationDataArray.reduce(
            (acc, curr) => {
                if (curr) {
                    acc[curr.branchId] = curr.currentLocationData;
                }
                return acc;
            }, {} as BranchLocationDataMap,
        );
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

            const cost = await this.subroutineExecutor.estimateCost(
                currentObject.routineType,
                subcontext,
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
     * Check if we have reached any time/credits/steps limit.
     * 
     * @param runProgress The current run
     * @param runLimits The limits for this run
     * @param startTime The time when the run started
     * @returns A reason for the status change, if limits have been reached. 
     * Otherwise, undefined.
     */
    private checkLimits(
        run: RunProgress,
        runLimits: RunRequestLimits,
        startTime: number,
    ): RunStatusChangeReason | undefined {
        const now = Date.now();
        const elapsed = now - startTime;

        /**
         * Helper function to determine new status if we've hit a limit.
         */
        function getNextStatus(behavior: RunLimitBehavior | undefined) {
            return behavior === "Pause" ? RunStatus.Paused : RunStatus.Failed;
        }

        // Time limit check
        if (runLimits.maxTime && elapsed >= runLimits.maxTime) {
            this.logger.error(`checkLimits: Run ${run.runId} reached time limit.`);
            run.status = getNextStatus(runLimits.onMaxTime);
            return RunStatusChangeReason.MaxTime;
        }

        // Credits check
        if (runLimits.maxCredits) {
            const spent = BigInt(run.metrics.creditsSpent || "0");
            const limit = BigInt(runLimits.maxCredits);
            if (spent >= limit) {
                this.logger.error(`checkLimits: Run ${run.runId} reached credits limit.`);
                run.status = getNextStatus(runLimits.onMaxCredits);
                return RunStatusChangeReason.MaxCredits;
            }
        }

        // Steps check
        if (runLimits.maxSteps !== undefined) {
            if (run.metrics.stepsRun >= runLimits.maxSteps) {
                this.logger.error(`checkLimits: Run ${run.runId} reached steps limit.`);
                run.status = getNextStatus(runLimits.onMaxSteps);
                return RunStatusChangeReason.MaxSteps;
            }
        }

        return undefined;
    }

    // TODO need to create new run_routine_io records for each new input and output (that doesn't reference an existing one), and way to find inputs/outputs by following references. Also need to add run_routien_step record each time we execute a step. not sure if elapsed time should be calculated including waiting or what. And need way to move time tracking outside of the state machine, so that the UI can pause the timer when not on the tab, and provide context switches
    /**
     * Execute the next step in a branch
     * 
     * NOTE 1: We can't update the run progress in here, since multiple 
     * branches might be executing in parallel (leading to race conditions).
     * Instead, we have to return the updated branch and run progress,
     * and let the caller (runUntilDone) handle data updates.
     * 
     * NOTE 2: This should not be used to step through the graph. The only 
     * new branches that should be created are when a multi-step routine 
     * starts.
     * 
     * @param branch The branch to execute
     * @param branchLocationDataMap The location data for each branch
     * @param config The configuration for this run
     * @returns The updated branch and run data
     */
    private async executeStep(
        branch: BranchProgress,
        locationData: BranchLocationDataMap["string"] | undefined,
        config: RunConfig,
    ): Promise<ExecuteStepResult> {
        const result: ExecuteStepResult = {
            branchId: branch.branchId,
            branchStatus: branch.status,
            complexityCompleted: 0,
            completedNodes: [], //TODO this is tricky. Single-step routines will complete right away, but multi-step routines will only complete when all child branches have completed
            deferredDecisions: [],
            creditsSpent: BigInt(0).toString(),
            newLocations: null,
            subroutineRun: null,
        };

        // Skip if branch is not active
        if (branch.status !== BranchStatus.Active) {
            return result;
        }

        // Skip if we don't have location data
        if (!locationData) {
            this.logger.error(`executeStep: Object not found for branch ${branch.branchId}`);
            result.branchStatus = BranchStatus.Failed;
            return result;
        }
        const { location, object, subroutine, subcontext: parentContext } = locationData;
        const nodeId = location.locationId;

        // Run the appropriate action based on the object type
        if (object.__typename === ModelType.RoutineVersion) {
            // Action only needed if there is a subroutine
            if (!subroutine) {
                this.logger.info(`executeStep: No subroutine found at ${branch.locationStack[branch.locationStack.length - 1]?.locationId}`);
                return result;
            }
            // If it's a single-step routine, run it
            const isSingleStep = this.subroutineExecutor.isSingleStepRoutine(subroutine);
            if (isSingleStep) {
                // Build context to pass into subroutine
                const initialChildContextValues = await this.prepareSubroutineContextBase(object, nodeId, parentContext);
                if (!initialChildContextValues) {
                    this.logger.error(`executeStep: Child context could not be prepared for single-step subroutine ${subroutine.id}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                const childContext = this.buildSubroutineContext(initialChildContextValues, subroutine, parentContext);

                // Execute the subroutine
                const subroutineResult = await this.subroutineExecutor.run(subroutine, childContext, config);

                // Parse the result
                const properlyKeyedResult = this.convertKeysInRunResultToComposite(subroutineResult, nodeId, object);
                if (!properlyKeyedResult) {
                    this.logger.error(`executeStep: Inputs or outputs could not be converted to composite keys for single-step subroutine ${subroutine.id}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                result.complexityCompleted = subroutine.complexity;
                result.subroutineRun = {
                    inputs: properlyKeyedResult.inputs,
                    outputs: properlyKeyedResult.outputs,
                };
                result.creditsSpent = (BigInt(result.creditsSpent) + BigInt(subroutineResult.cost)).toString();
                //TODO update timeElapsed somewhere, either for branch and overall run, or just overall run
                // result.completedNodes.push({
                //     fdsfdsafdafdasf,
                // }); TODO
            }
            // If it's a multi-step routine, we need to call getAvailableStartLocations with the subroutine config and create new branches
            const isMultiStep = this.subroutineExecutor.isMultiStepRoutine(subroutine);
            if (isMultiStep) {
                // Deserialize the child's graph configuration.
                const { graph: childGraph } = RoutineVersionConfig.deserialize(subroutine, this.logger, { useFallbacks: true });
                if (!childGraph) {
                    this.logger.error("executeStep: Invalid child graph configuration.");
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                // Get the child's navigator
                const childNavigator = this.navigatorFactory.getNavigator(childGraph.__type);
                if (!childNavigator) {
                    this.logger.error(`executeStep: No navigator found for child graph type ${childGraph.__type}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }

                // Build context to pass into subroutine
                const subroutineInputNameToParentContextValueMap = await this.prepareSubroutineContextBase(object, nodeId, parentContext);
                if (!subroutineInputNameToParentContextValueMap) {
                    this.logger.error(`executeStep: subroutineInputNameToParentContextValueMap could not be prepared for multi-step subroutine ${subroutine.id}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                const initialChildContextValues = await this.prepareMultiStepSubroutineContext(subroutineInputNameToParentContextValueMap, childGraph);
                if (!initialChildContextValues) {
                    this.logger.error(`executeStep: initialChildContextValues could not be prepared for multi-step subroutine ${subroutine.id}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                const childContext = this.buildSubroutineContext(initialChildContextValues, subroutine, parentContext);

                // Ask the subroutine's navigator for the available start locations.
                const decisionKey = this.decisionStrategy.generateDecisionKey(branch, "executeStep-startLocations");
                const decision = await childNavigator.getAvailableStartLocations({
                    config: childGraph,
                    decisionKey,
                    decisionStrategy: this.decisionStrategy,
                    logger: this.logger,
                    subroutine,
                    subcontext: childContext,
                });
                if (decision.deferredDecisions) {
                    result.deferredDecisions = decision.deferredDecisions;
                    result.branchStatus = BranchStatus.Waiting;
                    return result;
                }
                if (decision.triggerBranchFailure) {
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                // Assign the new start locations to the result.
                result.newLocations = {
                    initialContext: childContext,
                    locations: decision.nextLocations,
                    supportsParallelExecution: childNavigator.supportsParallelExecution,
                };
                // When the result is processed (see updateRunAfterStep), new branches will be created
                // for each location in decision.nextLocations, and the current branch will be marked as waiting.
                return result;
            }
            return result;
        } else if (object.__typename === ModelType.ProjectVersion) {
            // Projects are for navigation only, so no action needed
            return result;
        } else {
            this.logger.error(`executeStep: Unsupported object type: ${(object as { __typename: unknown }).__typename}`);
            result.branchStatus = BranchStatus.Failed;
            return result;
        }
    }

    /**
     * Helper method to prepare a subroutine context.
     * 
     * This method takes the parent routine configuration and subroutine (child)
     * configuration, and uses the parent's mapping configuration to determine which
     * IO values (with composite keys) should be passed into the subroutine.
     * 
     * NOTE: The result of this method cannot be used directly. Further processing is 
     * needed to create a SubroutineContext object. We stop here because the remaining 
     * logic is dependent on whether the subroutine is single-step or multi-step.
     * 
     * @param parentRoutine The parent routine (from the current location’s object)
     * @param nodeId The ID of the current node in the parent graph (i.e. where the subroutine is located in the parent graph)
     * @param parentSubcontext The existing subcontext (I/O values) from the parent branch
     * @returns An object with keys that are the subroutine's input names, and values that are the parent's IO values.
     */
    private async prepareSubroutineContextBase(
        parentRoutine: RoutineVersion,
        nodeId: string,
        parentSubcontext: SubroutineContext,
    ): Promise<IOMap | null> {
        // Deserialize the parent's graph configuration.
        const { graph: parentGraph } = RoutineVersionConfig.deserialize(parentRoutine, this.logger, { useFallbacks: true });
        if (!parentGraph) {
            this.logger.error("prepareSubroutineContextBase: Invalid parent graph configuration.");
            return null;
        }

        // Retrieve the parent's navigator to obtain input mapping information.
        const parentNavigator = this.navigatorFactory.getNavigator(parentGraph.__type);
        if (!parentNavigator) {
            this.logger.error(`prepareSubroutineContextBase: No navigator found for parent graph type ${parentGraph.__type}`);
            return null;
        }

        // 1. Find which inputs for the child node are linked to io values in the parent graph. 
        //    The result should be an object that maps the child node's input names (as they appear in the parent graph) 
        //    to the parent graph's io name composite keys (i.e. `root.${ioName}` or `${nodeId}.${ioName}`).
        //
        //    In BPMN graphs, the node would list inputs like this:
        //    <bpmn:callActivity id="callActivityA" name="Call A" calledElement="A">
        //    //...
        //    <vrooli:input name="inputA" fromContext="callActivityB.inputC" />
        //    //...
        //    
        //    Which should give us an object like this:
        //    {
        //        inputA: "callActivityB.inputC",
        //        //...
        //    }
        const nodeInputNameToParentContextKeyMap = await parentNavigator.getIONamesPassedIntoNode({
            config: parentGraph,
            logger: this.logger,
            nodeId,
        });

        // 2. Create an object that maps the keys from step 1 to the subroutine's input names as they appear in the subroutine's formInput.
        //    
        //    For BPMN, we can get this easily by using the `activityMap` that accompanies the parent graph's config.
        //    {
        //        callActivityA: {
        //            //...
        //            inputMap: {
        //                inputA: "subroutineInputA",
        //            },
        //        },
        //    }
        //
        //    Which would give us an object like this:
        //    {
        //        inputA: "subroutineInputA",
        //    }
        const nodeInputNameToSubroutineInputNameMap = parentGraph.getIONamesToSubroutineInputNames(nodeId);

        // 3. Using these two objects, we can go from nodeInputName -> parentGraphInputName, and nodeInputName -> subroutineInputName.
        //    Now we'll create a new object that goes from subroutineInputName -> parentGraphInputVALUE.
        // 
        //    It should look like this:
        //    {
        //        subroutineInputA: parentSubcontext["callActivityB.inputC"],
        //    }
        const subroutineInputNameToParentContextValueMap = Object.fromEntries(
            Object.entries(nodeInputNameToSubroutineInputNameMap)
                .map(([nodeInputName, subroutineInputName]) => {
                    const parentContextKey = nodeInputNameToParentContextKeyMap[nodeInputName];
                    if (!parentContextKey) {
                        return null;
                    }
                    const parentContextValue = parentSubcontext.allInputsMap[parentContextKey] ?? parentSubcontext.allOutputsMap[parentContextKey];
                    if (!parentContextValue) {
                        return null;
                    }
                    return [subroutineInputName, parentContextValue];
                })
                .filter(Boolean) as [IOKey, IOValue][],
        );

        // Return this object for further processing by routine type-specific methods.
        return subroutineInputNameToParentContextValueMap;
    }

    /**
     * Helper method to prepare the subroutine context for a multi-step subroutine.
     * 
     * This method takes the result of `prepareSubroutineContextBase` and uses it to
     * create the initial subcontext values for a multi-step subroutine. This entails 
     * converting the keys from subroutine input names to child context keys, using the 
     * child's graph config.
     * 
     * @param subroutineInputNameToParentContextValueMap The result of `prepareSubroutineContextBase`
     * @param childGraph The graph config for the child subroutine
     * @returns An object that can be used to initialize the subroutine context for a multi-step subroutine
     */
    private async prepareMultiStepSubroutineContext(
        subroutineInputNameToParentContextValueMap: IOMap,
        childGraph: GraphConfig,
    ): Promise<IOMap | null> {
        // `prepareSubroutineContext` completed the first 3 steps...
        // 4. Now using the CHILD's graph config, create a map of the subroutine's starting input names to its routine's input names.
        //
        //    For BPMN, we can get this easily by using the `rootContext` that accompanies the child's graph config.
        //    {
        //        inputMap: {
        //            inputA: "subroutineInputA",
        //        },
        //    }
        //
        //    Which would give us an object like this:
        //    {
        //        root.inputA: "subroutineInputA",
        //    }
        //
        const childContextKeyToSubroutineInputNameMap = childGraph.getRootIONamesToRoutineInputNames();

        // 5. Now we'll create a new object with the same keys as step 4, but with the values from the parent context. We'll use the object from 
        //    step 3 to find the correct parent context key. It should look like this:
        //
        //    {
        //        root.inputA: parentContext["callActivityB.inputC"],
        //    }
        const initialSubcontextValues: IOMap = {};
        for (const [childContextKey, subroutineInputName] of Object.entries(childContextKeyToSubroutineInputNameMap)) {
            const parentContextValue = subroutineInputNameToParentContextValueMap[subroutineInputName];
            if (!parentContextValue) {
                continue;
            }
            initialSubcontextValues[childContextKey] = parentContextValue;
        }

        // Return the initial subcontext values
        return initialSubcontextValues;
    }

    /**
     * Builds a subroutine context from a set of initial values.
     * 
     * @param initialValues The initial values for the subroutine context
     * @param subroutine The subroutine to build the context for
     * @param parentContext The parent context for the subroutine
     * @returns A new subroutine context
     */
    private buildSubroutineContext(
        initialValues: IOMap,
        subroutine: RoutineVersion,
        parentContext: SubroutineContext,
    ): SubroutineContext {
        const userData = this.getUserData("buildSubroutineContext");

        // Get information about the subroutine and use it to build the currentTask
        const { description, instructions, name } = getTranslation(subroutine, userData.languages, true);
        const currentTask: SubroutineContext["currentTask"] = {
            description: description ?? "",
            instructions: instructions ?? "",
            name: name ?? "",
        };

        // Get the overall task for the subroutine
        const overallTask: SubroutineContext["overallTask"] = parentContext.overallTask ?? parentContext.currentTask;

        // Build the subroutine context
        const subroutineContext: SubroutineContext = {
            ...SubroutineContextManager.initializeContext(initialValues),
            currentTask,
            overallTask,
        };

        // Return the subroutine context
        return subroutineContext;
    }

    /**
     * Helper method to convert inputs and outputs returned by `SubroutineExecutor.run` to the correct format - 
     * a composite key of the form `${nodeId}.${ioName}`.
     * 
     * Functionally, this is similar to `prepareMultiStepSubroutineContext`. That method goes from an object keyed by 
     * subroutine input names to and object keyed by child context keys. Here, we go from an object keyed by subroutine input/output 
     * names to an object keyed by PARENT context keys.
     * 
     * @param subroutineResult The result of `SubroutineExecutor.run`, containing inputs and outputs
     * @param nodeId The ID of the node that the inputs belong to
     * @param parentRoutine The parent routine (from the current location’s object)
     * @returns The converted inputs or outputs
     */
    private convertKeysInRunResultToComposite(
        subroutineResult: Pick<RunSubroutineResult, "inputs" | "outputs">,
        nodeId: string,
        parentRoutine: RoutineVersion,
    ): Pick<RunSubroutineResult, "inputs" | "outputs"> | null {
        // Deserialize the parent's graph configuration.
        const { graph: parentGraph } = RoutineVersionConfig.deserialize(parentRoutine, this.logger, { useFallbacks: true });
        if (!parentGraph) {
            this.logger.error("convertKeysInRunResultToComposite: Invalid parent graph configuration.");
            return null;
        }

        // Get the node's maps for node input -> subroutine input and node output -> subroutine output
        const nodeInputNameToSubroutineInputNameMap = parentGraph.getIONamesToSubroutineInputNames(nodeId);
        const nodeOutputNameToSubroutineInputNameMap = parentGraph.getIONamesToSubroutineOutputNames(nodeId);

        // Reverse the maps so we can go from subroutine input/output names to node input/output names
        const subroutineInputNameToNodeInputNameMap = Object.fromEntries(
            Object.entries(nodeInputNameToSubroutineInputNameMap).map(([nodeInputName, subroutineInputName]) => [subroutineInputName, nodeInputName]),
        );
        const subroutineOutputNameToNodeOutputNameMap = Object.fromEntries(
            Object.entries(nodeOutputNameToSubroutineInputNameMap).map(([nodeOutputName, subroutineOutputName]) => [subroutineOutputName, nodeOutputName]),
        );

        // Initialize the result
        const result: Pick<RunSubroutineResult, "inputs" | "outputs"> = {
            inputs: {},
            outputs: {},
        };

        // Convert the inputs
        for (const [subroutineInputName, value] of Object.entries(subroutineResult.inputs)) {
            const nodeInputName = subroutineInputNameToNodeInputNameMap[subroutineInputName];
            if (!nodeInputName) {
                continue;
            }
            // Add the value to the result as a composite key
            result.inputs[`${nodeId}.${nodeInputName}`] = value;
        }

        // Convert the outputs
        for (const [subroutineOutputName, value] of Object.entries(subroutineResult.outputs)) {
            const nodeOutputName = subroutineOutputNameToNodeOutputNameMap[subroutineOutputName];
            if (!nodeOutputName) {
                continue;
            }
            // Add the value to the result as a composite key
            result.outputs[`${nodeId}.${nodeOutputName}`] = value;
        }

        // Return the converted inputs and outputs
        return result;
    }

    /**
     * Moves branches forward, including creating and closing branches as needed.
     * 
     * @param run The current run progress
     * @param branchLocationDataMap The location data for each branch
     */
    private async advanceBranches(
        run: RunProgress,
        branchLocationDataMap: BranchLocationDataMap,
    ): Promise<void> {
        // Loop through each branch
        for (const branch of run.branches) {
            // Skip if branch is not active or waiting
            if (branch.status !== BranchStatus.Active && branch.status !== BranchStatus.Waiting) {
                continue;
            }

            // Get the location data
            const locationData = branchLocationDataMap[branch.branchId];
            // Mark as completed if we can't find the data
            if (!locationData) {
                this.logger.error(`moveBranchesForward: Location data not found for branch ${branch.branchId}`);
                branch.status = BranchStatus.Failed;
                continue;
            }
            const { location, object, subroutine } = locationData;

            // Get the appropriate graph config, io data, and navigator for the current object
            // TODO support projects and project directories
            if (object.__typename !== ModelType.RoutineVersion) {
                this.logger.error(`moveBranchesForward: Unsupported object type: ${(object as { __typename: unknown }).__typename}`);
                branch.status = BranchStatus.Failed;
                throw new Error(`Unsupported object type: ${object.__typename}`);
            }
            const { graph: config } = RoutineVersionConfig.deserialize(object, this.logger, { useFallbacks: true });
            if (!config) {
                throw new Error("Routine has no graph type");
            }
            const navigator = this.navigatorFactory.getNavigator(config.__type);

            const subcontext = run.subcontexts[branch.subroutineInstanceId];
            if (!subcontext) {
                this.logger.error(`moveBranchesForward: Subcontext not found for branch ${branch.branchId}`);
                branch.status = BranchStatus.Failed;
                continue;
            }

            // If the branch is waiting for a multi-step subroutine, check if it's complete by seeing if all child branches are completed
            const isMultiStep = subroutine !== null && this.subroutineExecutor.isMultiStepRoutine(subroutine);
            if (isMultiStep && branch.status === BranchStatus.Waiting) {
                const childBranches = run.branches.filter(b => b.subroutineInstanceId === branch.childSubroutineInstanceId);
                const acceptedBranchStates = run.config.onBranchFailure === "Continue" ? [BranchStatus.Completed, BranchStatus.Failed] : [BranchStatus.Completed];
                const allChildBranchesCompleted = childBranches.every(b => acceptedBranchStates.includes(b.status));
                // Don't continue if the multi-step subroutine is not complete
                if (!allChildBranchesCompleted) {
                    continue;
                }
            }
            // Determine which node(s) to go to next
            const decisionKey = this.decisionStrategy.generateDecisionKey(branch, "advanceBranches-nextLocations");
            const nextDecision = await navigator.getAvailableNextLocations({
                config,
                decisionKey,
                decisionStrategy: this.decisionStrategy,
                location,
                logger: this.logger,
                runConfig: run.config,
                subcontext,
            });

            // Handle deferred decisions
            if (nextDecision.deferredDecisions) {
                // Update the decision strategy
                const updatedDecisions = this.decisionStrategy.updateDecisionOptions(run, nextDecision.deferredDecisions);
                // Update the run progress
                run.decisions = updatedDecisions;
                // Send a decision request to the client
                for (const decision of nextDecision.deferredDecisions) {
                    this.notifier?.sendDecisionRequest(run.runId, run.type, decision);
                }
                // Put the branch in a waiting state
                branch.status = BranchStatus.Waiting;
                continue;
            }
            // Handle failures
            if (nextDecision.triggerBranchFailure) {
                branch.status = BranchStatus.Failed;
                continue;
            }
            // Spawn new branches if needed
            if (nextDecision.nextLocations.length > 0) {
                // Filter out any locations that are already closed
                const filteredNextLocations = nextDecision.nextLocations.filter(loc => !branch.closedLocations.includes(loc));
                this.forkBranches(branch, filteredNextLocations, branch.subroutineInstanceId, branch.supportsParallelExecution);
            }
            // Handle blocking certain branches from spawning again
            if (nextDecision.closedLocations) {
                branch.closedLocations.push(...nextDecision.closedLocations);
            }
            // If the node is still active (i.e. we're not moving to a new location), mark the branch as waiting
            if (nextDecision.isNodeStillActive) {
                branch.status = BranchStatus.Waiting;
            }
            // If the node is not active, mark the branch as completed
            else {
                branch.status = BranchStatus.Completed;
                // Ensure every other active branch with the same process ID at this location 
                // is also marked as completed (handles joins)
                const activeBranchesOnProcess = run.branches.filter(b => b.status === BranchStatus.Active && b.processId === branch.processId);
                for (const otherBranch of activeBranchesOnProcess) {
                    const currLocationData = branchLocationDataMap[otherBranch.branchId];
                    if (!currLocationData) {
                        continue;
                    }
                    if (currLocationData.location.locationId === location.locationId) {
                        otherBranch.status = BranchStatus.Completed;
                    }
                }
                // If there are no more active branches remaining in this subroutine instance, merge the contexts 
                // of the completed branches' subroutine instance with the parent routine that started it, 
                // and remove the subroutineInstanceId from subcontexts.
                //
                // The way we do this is tricky, since we don't want to merge every value in the subcontext. 
                // Rather, we only want to merge values the subroutine defines as outputs, and rename them to match 
                // what the parent expects to receive.
                // 
                // This approach ensures that the parent routine understands the outputs its receives, and can limit the amount of context 
                // that is kept in memory.
                const hasRemainingActiveBranches = run.branches.some(b => b.status === BranchStatus.Active && b.subroutineInstanceId === branch.subroutineInstanceId);
                const parentBranch = hasRemainingActiveBranches
                    ? null
                    : run.branches.find(b => b.childSubroutineInstanceId === branch.subroutineInstanceId);
                const parentSubroutineInstanceId = parentBranch?.subroutineInstanceId;
                const parentSubcontext = parentSubroutineInstanceId ? run.subcontexts[parentSubroutineInstanceId] : null;
                if (parentBranch && parentSubroutineInstanceId && parentSubcontext) {
                    // Load the parent routine
                    const parentLocation = branch.locationStack[branch.locationStack.length - 2];
                    if (!parentLocation) {
                        this.logger.error(`advanceBranches: No parent location found for branch ${branch.branchId}.`);
                        continue;
                    }
                    const parentLocationData = await this.loader.loadLocation(parentLocation, run.config);
                    if (!parentLocationData) {
                        this.logger.error(`advanceBranches: No parent location data found for branch ${branch.branchId}.`);
                        continue;
                    }
                    const parentRoutine = parentLocationData.object as RoutineVersion;
                    // Deserialize the parent's graph configuration.
                    const { graph: parentGraph } = RoutineVersionConfig.deserialize(parentRoutine, this.logger, { useFallbacks: true });
                    if (!parentGraph) {
                        this.logger.error("advanceBranches: Invalid parent graph configuration.");
                        continue;
                    }
                    // When moving down into a multi-step subroutine, the nodeId is defined by the locationId.
                    // When moving up out of a multi-step subroutine, we need to use the last location in the parent branch's location stack.
                    const nodeId = parentBranch.locationStack[parentBranch.locationStack.length - 1]?.locationId;
                    if (!nodeId) {
                        this.logger.error(`advanceBranches: No nodeId found for branch ${branch.branchId}.`);
                        continue;
                    }

                    // 1. Get the child's output map to determine which values to keep from the child's subcontext.
                    //
                    //    For BPMN, we can get this easily by using the `rootContext` that accompanies the child's graph config.
                    //    {
                    //        outputMap: {
                    //            outputA: "subroutineOutputA",
                    //        },
                    //    }
                    //
                    //    Which would give us an object like this:
                    //    {
                    //        root.outputA: "subroutineOutputA",
                    //    }
                    //
                    //    Note that they keys in this object should match the keys in the child's subcontext.
                    const childContextKeyToSubroutineOutputNameMap = config.getRootIONamesToRoutineOutputNames();
                    // 2. Get the output map of the corresponding node in the parent graph config.
                    //    
                    //    For BPMN, we can get this easily by using the `activityMap` that accompanies the parent graph's config.
                    //    {
                    //        callActivityA: {
                    //            //...
                    //            outputMap: {
                    //                outputA: "subroutineOutputA",
                    //            },
                    //        },
                    //    }
                    //
                    //    Which would give us an object like this:
                    //    {
                    //        outputA: "subroutineOutputA",
                    //    }
                    const nodeOutputNameToSubroutineOutputNameMap = parentGraph.getIONamesToSubroutineOutputNames(nodeId);
                    // 3. Reverse the nodeOutputNameToSubroutineOutputNameMap and convert keys to composite keys.
                    //    This will give us an object that goes from subroutineOutputName -> parentContextKey.
                    const subroutineOutputNameToParentContextKey = Object.fromEntries(
                        Object.entries(nodeOutputNameToSubroutineOutputNameMap).map(([nodeOutputName, subroutineOutputName]) => [subroutineOutputName, `${nodeId}.${nodeOutputName}`]),
                    );
                    // 4. Loop through the child's output map and create a new object that goes from parentContextKey -> value.
                    const outputsToAddToParentContext: IOMap = {};
                    for (const [childContextKey, subroutineOutputName] of Object.entries(childContextKeyToSubroutineOutputNameMap)) {
                        const parentContextKey = subroutineOutputNameToParentContextKey[subroutineOutputName];
                        if (!parentContextKey) {
                            continue;
                        }
                        outputsToAddToParentContext[parentContextKey] = subcontext.allInputsMap[childContextKey] ?? subcontext.allOutputsMap[childContextKey];
                    }

                    run.subcontexts[parentSubroutineInstanceId] = SubroutineContextManager.updateContext(parentSubcontext, {}, outputsToAddToParentContext);
                    delete run.subcontexts[branch.subroutineInstanceId];

                    // Also add 1 complexity point to the run's metrics
                    run.metrics.complexityCompleted++;
                }
                //TODO probably also accumulate time elapsed here
            }
        }
    }

    /**
     * Switch how decisions are made at runtime.
     * 
     * @param newStrategy The new decision strategy to use
     */
    public async setDecisionStrategy(newStrategy: DecisionStrategy) {
        // Update the decision strategy
        this.decisionStrategy = newStrategy;

        // Get the run
        const runIdentifier = this.getRunIdentifier("setDecisionStrategy");
        const userData = this.getUserData("setDecisionStrategy");
        const run = await this.persistence.loadProgress(runIdentifier, userData, this.logger);
        if (!run) {
            throw new Error(`Run ID ${runIdentifier.runId} not found in database`);
        }

        // Set the available decisions to the new strategy
        this.decisionStrategy.updateDecisionOptions(run, run.decisions);
    }

    /**
     * Gets the current decision strategy.
     * 
     * @returns The current decision strategy
     */
    public getDecisionStrategy(): DecisionStrategy {
        return this.decisionStrategy;
    }

    /**
     * Initializes a run configuration update. 
     * On the next iteration of the run loop, the run configuration will be updated.
     * 
     * @param config The new configuration for this run
     */
    public updateRunConfig(config: RunConfig): void {
        this.pendingConfigUpdate = config;
    }

    /**
     * Applies configuration updates to the run.
     * Does not persist the changes to the database.
     * 
     * @param run The current run progress, containing the configuration to update
     */
    private applyConfigUpdate(run: RunProgress): void {
        if (this.pendingConfigUpdate) {
            run.config = this.pendingConfigUpdate;
            this.pendingConfigUpdate = null;
        }
    }

    /**
     * Stop the run.
     * The running loop in runUntilDone will detect this change 
     * and exit on the next iteration.
     *
     * @param status The new status for the run
     */
    public async stopRun(status: RunStatus.Cancelled | RunStatus.Paused): Promise<void> {
        // Get run identifier and user data
        const runIdentifier = this.getRunIdentifier("stopRun");
        const userData = this.getUserData("stopRun");

        const run = await this.persistence.loadProgress(runIdentifier, userData, this.logger);
        if (!run) {
            throw new Error(`pauseRun: Run not found for ID: ${runIdentifier.runId}`);
        }

        if (run.status !== RunStatus.InProgress) {
            this.logger.error(`pauseRun: Cannot pause run ${runIdentifier.runId} because it is not in progress.`);
            return;
        }

        // Update the run status
        run.status = status;
        // NOTE: Don't call saveProgress here. Let the main loop handle it.
        this.logger.info(`pauseRun: Run ${runIdentifier.runId} has been paused.`);
    }

    /**
     * Creates a new branch ID.
     */
    private generateBranchId(): Id {
        return uuid();
    }

    /**
     * Creates a new branch process ID.
     */
    private generateProcessId(): Id {
        return uuid();
    }

    /**
     * Creates a new subroutine instance ID.
     * 
     * @param subroutineId The ID of the subroutine
     * @returns A composite ID of the shape `${subroutineId}.${uniqueId}`.
     */
    private generateSubroutineInstanceId(subroutineId: Id): string {
        return `${subroutineId}.${uuid()}`;
    }

    // /**
    //  * Spawns multiple parallel branches from a single “base” branch. 
    //  * You might call this when you detect a parallel-split gateway (or multiple next nodes) 
    //  * that must run in parallel.
    //  * 
    //  * NOTE: We can't update the run progress in here, since multiple 
    //  * branches might be executing it in parallel. 
    //  * Instead, we have to return the updated branch and run progress,
    //  *
    //  * @param baseBranch The branch to fork from
    //  * @param nextLocations All possible locations for the new branches
    //  * @returns The new branches
    //  */
    /**
     * Spawns multiple parallel branches.
     * 
     * Called in two cases:
     * 1. When starting a multi-step subroutine
     * 2. When there is a parallel-split gateway
     * 
     * NOTE: We can't update the run progress in here, since multiple 
     * branches might be executing it in parallel. 
     * Instead, we have to return the updated branch and run progress,
     *
     * @param startingBranch The branch to fork from if there is a parallel-split gateway. 
     * Pass in null if starting a new multi-step subroutine.
     * @param nextLocations The locations to create new branches for
     * @param supportsParallelExecution Whether the branches support parallel execution
     * @param subroutineInstanceId The subroutine instance ID to use for the new branches
     * @param baseLocationStack What to append to the new branches' location stacks, if startingBranch is null
     * @returns The new branches
     */
    public forkBranches(
        startingBranch: BranchProgress | null,
        nextLocations: Location[],
        subroutineInstanceId: Id,
        supportsParallelExecution: boolean,
        baseLocationStack: LocationStack | null = null,
    ): BranchProgress[] {
        const newBranches: BranchProgress[] = []; // Shared with starting branch so we can share context

        // Spawn a new branch for each location
        for (const newLoc of nextLocations) {

            const locationStack = [...(baseLocationStack ?? startingBranch?.locationStack.slice(0, -1) ?? []), newLoc];
            const newBranchId = this.generateBranchId();
            const processId = startingBranch?.processId ?? this.generateProcessId(); // Shared with starting branch so we can join later

            const newBranch: BranchProgress = {
                branchId: newBranchId,
                childSubroutineInstanceId: null, // This is set on the parent branch when the new branches are created. This branch is a child, so it will never be set here.
                closedLocations: [],
                creditsSpent: BigInt(0).toString(),
                nodeStartTimeMs: null,
                processId,
                locationStack,
                status: BranchStatus.Active,
                supportsParallelExecution,
                subroutineInstanceId,
            };

            newBranches.push(newBranch);
        }

        return newBranches;
    }
}

//TODO context building for LLM, better project handling, triggers
//TODO support roles. For example, should be possible to assume one role, and let bots assume other roles. This could be useful in a scenario where a bot handles the steps related to researching a topic, and the user does the steps where they read the report and choose what to do next.
