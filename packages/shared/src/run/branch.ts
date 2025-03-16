import { ModelType, RoutineVersion, RunStepStatus } from "../api/types.js";
import { uuid } from "../id/uuid.js";
import { RoutineVersionConfig } from "./configs/routine.js";
import { SubroutineContextManager } from "./context.js";
import { BranchLocationDataMap, BranchProgress, BranchStatus, IOMap, Id, Location, LocationStack, RunProgress, RunStateMachineServices } from "./types.js";

/**
 * Handles branch logic for the run.
 */
export class BranchManager {
    /**
     * Moves branches forward, including creating and closing branches as needed.
     * 
     * @param run The current run progress
     * @param branchLocationDataMap The location data for each branch
     * @param services The state machine's services
     */
    static async advanceBranches(
        run: RunProgress,
        branchLocationDataMap: BranchLocationDataMap,
        services: RunStateMachineServices,
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
                services.logger.error(`moveBranchesForward: Location data not found for branch ${branch.branchId}`);
                branch.status = BranchStatus.Failed;
                continue;
            }
            const { location, object, subroutine } = locationData;

            // Get the appropriate graph config, io data, and navigator for the current object
            // TODO support projects and project directories
            if (object.__typename !== ModelType.RoutineVersion) {
                services.logger.error(`moveBranchesForward: Unsupported object type: ${(object as { __typename: unknown }).__typename}`);
                branch.status = BranchStatus.Failed;
                throw new Error(`Unsupported object type: ${object.__typename}`);
            }
            const { graph: config } = RoutineVersionConfig.deserialize(object, services.logger, { useFallbacks: true });
            if (!config) {
                throw new Error("Routine has no graph type");
            }
            const navigator = services.navigatorFactory.getNavigator(config.__type);

            const subcontext = run.subcontexts[branch.subroutineInstanceId];
            if (!subcontext) {
                services.logger.error(`moveBranchesForward: Subcontext not found for branch ${branch.branchId}`);
                branch.status = BranchStatus.Failed;
                continue;
            }

            // If the branch is waiting for a multi-step subroutine, check if it's complete by seeing if all child branches are completed
            const isMultiStep = subroutine !== null && services.subroutineExecutor.isMultiStepRoutine(subroutine);
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
            const decisionKey = services.pathSelectionHandler.generateDecisionKey(branch, "advanceBranches-nextLocations");
            const nextDecision = await navigator.getAvailableNextLocations({
                config,
                decisionKey,
                location,
                runConfig: run.config,
                services,
                subcontext,
            });

            // Handle deferred decisions
            if (nextDecision.deferredDecisions) {
                // Update the decision strategy
                const updatedDecisions = services.pathSelectionHandler.updateDecisionOptions(run, nextDecision.deferredDecisions);
                // Update the run progress
                run.decisions = updatedDecisions;
                // Send a decision request to the client
                for (const decision of nextDecision.deferredDecisions) {
                    services.notifier?.sendDecisionRequest(run.runId, run.type, decision);
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
                const stepProgress = branch.stepId ? run.steps.find(s => s.id === branch.stepId) : null;
                if (stepProgress) {
                    stepProgress.completedAt = new Date(Date.now());
                    stepProgress.status = RunStepStatus.Completed;
                }
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
                        const otherStepProgress = otherBranch.stepId ? run.steps.find(s => s.id === otherBranch.stepId) : null;
                        if (otherStepProgress) {
                            otherStepProgress.completedAt = new Date(Date.now());
                            otherStepProgress.status = RunStepStatus.Completed;
                        }
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
                        services.logger.error(`advanceBranches: No parent location found for branch ${branch.branchId}.`);
                        continue;
                    }
                    const parentLocationData = await services.loader.loadLocation(parentLocation, run.config);
                    if (!parentLocationData) {
                        services.logger.error(`advanceBranches: No parent location data found for branch ${branch.branchId}.`);
                        continue;
                    }
                    const parentRoutine = parentLocationData.object as RoutineVersion;
                    // Deserialize the parent's graph configuration.
                    const { graph: parentGraph } = RoutineVersionConfig.deserialize(parentRoutine, services.logger, { useFallbacks: true });
                    if (!parentGraph) {
                        services.logger.error("advanceBranches: Invalid parent graph configuration.");
                        continue;
                    }
                    // When moving down into a multi-step subroutine, the nodeId is defined by the locationId.
                    // When moving up out of a multi-step subroutine, we need to use the last location in the parent branch's location stack.
                    const nodeId = parentBranch.locationStack[parentBranch.locationStack.length - 1]?.locationId;
                    if (!nodeId) {
                        services.logger.error(`advanceBranches: No nodeId found for branch ${branch.branchId}.`);
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
                }
            }
        }
    }

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
    static forkBranches(
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
            const newBranchId = BranchManager.generateBranchId();
            const processId = startingBranch?.processId ?? BranchManager.generateBranchProcessId(); // Shared with starting branch so we can join later

            const newBranch: BranchProgress = {
                branchId: newBranchId,
                childSubroutineInstanceId: null, // This is set on the parent branch when the new branches are created. This branch is a child, so it will never be set here.
                closedLocations: [],
                creditsSpent: BigInt(0).toString(),
                manualExecutionConfirmed: false,
                nodeStartTimeMs: null,
                processId,
                locationStack,
                status: BranchStatus.Active,
                stepId: null,
                supportsParallelExecution,
                subroutineInstanceId,
            };

            newBranches.push(newBranch);
        }

        return newBranches;
    }

    /**
     * Loads location data for all branches, and converts them into a dictionary.
     * Any branch that can't be loaded is marked as failed.
     * 
     * @param run The current run progress
     * @param services The state machine's services
     * @returns A dictionary of branch IDs to location data
     */
    static async loadBranchLocationData(
        run: RunProgress,
        services: RunStateMachineServices,
    ): Promise<BranchLocationDataMap> {
        const branchLocationDataArray = await Promise.all(
            run.branches.map(async (branch) => {
                const currentLocation = branch.locationStack[branch.locationStack.length - 1];
                if (!currentLocation) {
                    services.logger.error(`loadBranchLocationData: Branch ${branch.branchId} has no current location`);
                    branch.status = BranchStatus.Failed;
                    return null;
                }
                const currentLocationData = await services.loader.loadLocation(currentLocation, run.config);
                if (!currentLocationData) {
                    services.logger.error(`loadBranchLocationData: Location data not found for location ${currentLocation.locationId}`);
                    branch.status = BranchStatus.Failed;
                    return null;
                }
                const currentSubcontext = run.subcontexts[branch.subroutineInstanceId];
                if (!currentSubcontext) {
                    services.logger.error(`loadBranchLocationData: Subcontext not found for branch ${branch.branchId}`);
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
    * Creates a new branch ID.
    */
    static generateBranchId(): Id {
        return uuid();
    }

    /**
     * Creates a new branch process ID.
     */
    static generateBranchProcessId(): Id {
        return uuid();
    }

    /**
     * Creates a new subroutine instance ID.
     * 
     * @param subroutineId The ID of the subroutine
     * @returns A composite ID of the shape `${subroutineId}.${uniqueId}`.
     */
    static generateSubroutineInstanceId(subroutineId: Id): string {
        return `${subroutineId}.${uuid()}`;
    }
}
