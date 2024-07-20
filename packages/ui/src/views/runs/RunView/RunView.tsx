import { base36ToUuid, endpointGetProjectVersionDirectories, endpointGetRoutineVersions, endpointPutRunRoutine, endpointPutRunRoutineComplete, exists, isOfType, NodeLink, NodeType, noop, ProjectVersionDirectory, ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult, RoutineType, RoutineVersion, RoutineVersionSearchInput, RoutineVersionSearchResult, RunProject, RunRoutine, RunRoutineCompleteInput, RunRoutineInput, RunRoutineStep, RunRoutineStepStatus, RunRoutineUpdateInput, uuid, uuidValidate } from "@local/shared";
import { Box, Button, Grid, IconButton, LinearProgress, Stack, styled, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RunStepsDialog } from "components/dialogs/RunStepsDialog/RunStepsDialog";
import { SessionContext } from "contexts/SessionContext";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useReactSearch } from "hooks/useReactSearch";
import { ArrowLeftIcon, ArrowRightIcon, CloseIcon, SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, removeSearchParams, useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { DecisionStep, DirectoryStep, EndStep, MultiRoutineStep, NodeStep, RootStep, RoutineListStep, RunStep, SingleRoutineStep, StartStep } from "types";
import { ProjectStepType, RunStepType } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { tryOnClose } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { getRunPercentComplete, locationArraysMatch, routineVersionHasSubroutines, runInputsUpdate } from "utils/runUtils";
import { DecisionView } from "../DecisionView/DecisionView";
import { SubroutineView } from "../SubroutineView/SubroutineView";
import { RunnableProjectVersion, RunnableRoutineVersion, RunViewProps } from "../types";

/**
 * Maximum routine nesting supported
 */
export const MAX_NESTING = 20;

/**
 * Inserts step data into a given RunStep, where id matches. 
 * We recursively search through the main step object to find the location to insert the step.
 * 
 * @param stepData Step to insert into the root step object. Either a MultiRoutineStep or DirectoryStep, 
 * as these are the only step types that represent full routine versions or project versions, and the reason 
 * we need to insert steps is to inject newly-fetched routine versions or project versions into the existing step object.
 * @param rootStep The root step object to insert the step into. Either a MultiRoutineStep, DirectoryStep, or SingleRoutineStep, 
 * as we can run multi-step routines, projects, and single-step routines.
 * @returns Updated root step object with the step inserted.
 */
export function insertStep(
    stepData: DirectoryStep | MultiRoutineStep,
    rootStep: RootStep,
): RootStep {
    // Helper function to perform recursive insert
    function recursiveInsert(currStep: RunStep, depth: number): RunStep {
        if (depth > MAX_NESTING) {
            console.error("Recursion depth exceeded in insertStep. No step inserted");
            return currStep;
        }
        // If the current step has a list of steps, recursively search through them
        if (currStep.__type === RunStepType.Directory || currStep.__type === RunStepType.RoutineList) {
            // If this is a directory, we may be able to insert the step here
            if (
                currStep.__type === RunStepType.Directory &&
                stepData.__type === RunStepType.Directory &&
                currStep.projectVersionId === stepData.projectVersionId
            ) {
                // Assign the new step data to the current step in a way that doesn't 
                // overwrite any fields that are missing in the new step data
                const result = currStep as DirectoryStep;
                result.description = stepData.description || currStep.description;
                result.name = stepData.name || currStep.name;
                result.directoryId = stepData.directoryId || currStep.directoryId;
                result.hasBeenQueried = stepData.hasBeenQueried ?? currStep.hasBeenQueried;
                result.isOrdered = stepData.isOrdered ?? currStep.isOrdered;
                result.isRoot = stepData.isRoot ?? currStep.isRoot;
                result.steps = stepData.steps || currStep.steps;
                return result;
            }
            // Otherwise, recursively search through the steps
            for (let i = 0; i < currStep.steps.length; i++) {
                currStep.steps[i] = recursiveInsert(currStep.steps[i], depth + 1) as (typeof currStep.steps)[number];
            }
        }
        // If the current step has a list of nodes
        else if (currStep.__type === RunStepType.MultiRoutine) {
            // If the currStep and stepData are the same routine, we can insert the step here
            if (
                stepData.__type === RunStepType.MultiRoutine &&
                currStep.routineVersionId === stepData.routineVersionId
            ) {
                // Assign the new step data to the current step in a way that doesn't
                // overwrite any fields that are missing in the new step data
                const result = currStep as MultiRoutineStep;
                result.description = stepData.description || currStep.description;
                result.name = stepData.name || currStep.name;
                result.nodeLinks = stepData.nodeLinks || currStep.nodeLinks;
                result.nodes = stepData.nodes || currStep.nodes;
                return result;
            }
            // Otherwise, recursively search through the nodes
            for (let i = 0; i < currStep.nodes.length; i++) {
                currStep.nodes[i] = recursiveInsert(currStep.nodes[i], depth + 1) as (typeof currStep.nodes)[number];
            }
        }
        // If the current step is a SingleRoutineStep (which we use as a placeholder before loading the full routine) 
        // and the stepData is a MultiRoutineStep, we may be able to insert the step here
        else if (
            currStep.__type === RunStepType.SingleRoutine &&
            stepData.__type === RunStepType.MultiRoutine &&
            currStep.routineVersion.id === stepData.routineVersionId
        ) {
            // Return stepData with location. Since these are different types, we can't 
            // do the field-by-field assignment like we did with the other step types
            const result = stepData as MultiRoutineStep;
            result.location = currStep.location;
            return result;
        }

        // For all other cases, return the current step unchanged
        return currStep;
    }

    // Start the recursive insertion
    return recursiveInsert(rootStep, 0) as RootStep;
}

/**
 * Uses a location array to find the step at a given location.
 * 
 * NOTE: Must have been queried already
 * @param location A list of natural numbers representing the location of the step in the root step
 * @param rootStep The root step object for the project or routine
 * @returns The requested step, or null if not found
 */
export function stepFromLocation(
    location: number[],
    rootStep: RootStep,
): RunStep | null {
    let currentStep: RunStep = rootStep;

    // If the location is empty, return null
    if (location.length === 0) return null;
    // If the first location is not 1, return null
    if (location[0] !== 1) return null;
    // If the location has a single number, return the root step
    if (location.length === 1) return rootStep;

    // Loop through the location array, skipping the first number
    for (let i = 1; i < location.length; i++) {
        const index = location[i];
        if (currentStep.__type === RunStepType.Directory || currentStep.__type === RunStepType.RoutineList) {
            if (index > 0 && index <= currentStep.steps.length) {
                currentStep = currentStep.steps[index - 1];
            } else {
                return null; // Invalid index
            }
        } else if (currentStep.__type === RunStepType.MultiRoutine) {
            // For MultiRoutineStep, we need to find the corresponding node
            const node = currentStep.nodes.find(node => node.location[node.location.length - 1] === index);
            if (node) {
                currentStep = node;
            } else {
                return null; // Node not found
            }
        } else if (currentStep.__type === RunStepType.Decision) {
            if (index > 0 && index <= currentStep.options.length) {
                currentStep = currentStep.options[index - 1].step;
            } else {
                return null; // Invalid index
            }
        } else {
            return null; // Cannot navigate further
        }
    }

    return currentStep;
}

/**
 * Uses a location array to find the number of sibling steps at a given location
 * 
 * NOTE: Must be queried already
 * @param location A list of natural numbers representing the location of the step in the root step
 * @param rootStep The root step object for the project or routine
 * @returns The number of sibling steps. Or in other words, the number of steps which share the same base location 
 * array (e.g. [4, 2, 1, 1], [4, 2, 1, 2], [4, 2, 1, 3], ...)
 */
export function siblingsAtLocaiton(
    location: number[],
    rootStep: RootStep,
) {
    // If there are no numbers in the location, it's invalid
    if (location.length === 0) return 0;
    // If there is one number in the location, it's the root. 
    // There can only be one root.
    if (location.length === 1) return 1;

    // Get parent
    const parentLocation = location.slice(0, -1);
    const parent = stepFromLocation(parentLocation, rootStep);
    if (!parent) {
        console.error(`Could not find parent to count siblings. Location: ${location}`);
        return 0;
    }

    // Determine siblings based on step type
    switch (parent.__type) {
        case RunStepType.Directory:
            return parent.steps.length;
        case RunStepType.MultiRoutine:
            return parent.nodes.length;
        case RunStepType.RoutineList:
            return parent.steps.length;
        // Other types can't have children, but if you're somehow here let's just return 1
        default:
            return 1;
    }
}

/**
 * Finds the previous location
 * @param location The current location
 * @returns The previous location, or null if already at the first step
 * @example getPreviousLocation([1, 2, 3]) => [1, 2, 2], getPreviousLocation([1, 2, 1]) => [1, 2]
 */
export function getPreviousLocation(location: number[]): number[] | null {
    if (location.length === 0) {
        return null; // Already at the root
    }

    const previousLocation = [...location];

    // Decrement the last number
    previousLocation[previousLocation.length - 1]--;

    // If we've decremented to 0, remove the last number
    while (previousLocation.length > 0 && previousLocation[previousLocation.length - 1] === 0) {
        previousLocation.pop();
    }

    // If we've popped all numbers, we've gone before the first step
    return previousLocation.length > 0 ? previousLocation : null;
}

//TODO update to follow new rules
/**
 * Finds the next available location. Rules (in order):
 * - If the step is a DecisionStep or EndStep, it cannot have a next location, so return null
 * // At this point, the step is valid
 * - If the step has children, go to the first child
 * - If the step has a `nextLocation` field, return that if it's not null
 * - If the step has siblings and is not the last sibling, go to the next sibling
 * // At this point, we start backtracking up the parent chain
 * - If the parent has a `nextLocation` field, return that if it's not null
 * - If the parent has siblings and is not the last sibling, go to the next sibling
 * - If all else fails, return null
 * @param location The current location
 * @param rootStep The root step object for the project or routine
 * @returns The next available location, or null if not found (either because there is not next 
 * step or because the current step is a decision)
 */
export function getNextLocation(
    location: number[],
    rootStep: RootStep | null,
): number[] | null {
    if (!rootStep) return null;

    const currentLocation = [...location];

    // If the location is empty, return the first step
    if (currentLocation.length === 0) return [1];

    // Retrieve the current step based on the location provided
    const currentStep = stepFromLocation(location, rootStep);
    if (!currentStep) return null;

    // If the first location points to a DecisionStep, return null. 
    // DecisionSteps don't have a determined next step, since the user has to choose
    const firstStep = stepFromLocation(currentLocation, rootStep);
    if (firstStep?.__type === RunStepType.Decision) {
        return null;
    }

    // Attempt to go to the first child if present
    const childLocation = [...location, 1];
    const childStep = stepFromLocation(childLocation, rootStep);
    if (childStep) return childLocation;

    // If the current step has a defined next location, return that location (even if it's null)
    if (Object.prototype.hasOwnProperty.call(currentStep, "nextLocation")) {
        const nextLocation = (currentStep as NodeStep).nextLocation;
        if (nextLocation) return nextLocation;
    }

    // Attempt to navigate to the next sibling if available
    const nextSiblingLocation = [...location];
    nextSiblingLocation[nextSiblingLocation.length - 1]++;
    const siblingStep = stepFromLocation(nextSiblingLocation, rootStep);
    if (siblingStep) return nextSiblingLocation;

    // Start backtracking to find a valid next location from parents
    while (location.length > 0) {
        location.pop(); // Move up to the parent
        console.log("backtracking...", location);
        if (location.length === 0) break; // If we've reached the root, stop

        // Check the parent's next location
        const parentStep = stepFromLocation(location, rootStep);
        console.log("parentStep", parentStep);
        if (parentStep && Object.prototype.hasOwnProperty.call(parentStep, "nextLocation")) {
            const parentNextLocation = (parentStep as NodeStep).nextLocation;
            if (parentNextLocation) return parentNextLocation;
        }

        // Check the next sibling of the parent
        const parentSiblingLocation = [...location];
        parentSiblingLocation[parentSiblingLocation.length - 1]++;
        const parentSiblingStep = stepFromLocation(parentSiblingLocation, rootStep);
        console.log("parentSiblingStep", parentSiblingStep);
        if (parentSiblingStep) {
            return parentSiblingLocation;
        }
    }

    console.log("No next location found");
    return null;
}

/**
 * Determines if a step (either subroutine or directory) needs additional queries, or if it already 
 * has enough data to render
 * @param step The step to check
 * @returns True if the step needs additional queries, false otherwise
 */
export function stepNeedsQuerying(
    step: RunStep | null | undefined,
): boolean {
    if (!step) return false;
    // If it's a subroutine, we need to query when it has its own subroutines. 
    // This works because when the data is queried, the step is replaced with a MultiRoutineStep.
    if (step.__type === RunStepType.SingleRoutine) {
        const currSubroutine: Partial<RoutineVersion> = (step as SingleRoutineStep).routineVersion;
        return routineVersionHasSubroutines(currSubroutine);
    }
    // If it's a directory, we need to query when it has not been marked as being queried. 
    // This step type has a query flag because when the data is queried, we add information to the 
    // existing step object, rather than replacing it.
    if (step.__type === RunStepType.Directory) {
        const currDirectory: Partial<DirectoryStep> = step as DirectoryStep;
        return currDirectory.hasBeenQueried === false;
    }
    return false;
}

/**
 * Calculates the complexity of a step
 * @param step The step to calculate the complexity of
 * @returns The complexity of the step
 */
export function getStepComplexity(step: RunStep): number {
    switch (step.__type) {
        // No complexity for start and end steps, since the user doesn't interact with them
        case RunStepType.End:
        case RunStepType.Start:
            return 0;
        // One decision, so one complexity
        case RunStepType.Decision:
            return 1;
        // Complexity of subroutines stored in routine data
        case RunStepType.SingleRoutine:
            return (step as SingleRoutineStep).routineVersion.complexity;
        // Complexity of a routine is the sum of its nodes' complexities
        case RunStepType.MultiRoutine:
            return (step as MultiRoutineStep).nodes.reduce((acc, curr) => acc + getStepComplexity(curr), 0);
        // Complexity of a list is the sum of its children's complexities
        case RunStepType.RoutineList:
        case RunStepType.Directory:
            return (step as RoutineListStep).steps.reduce((acc, curr) => acc + getStepComplexity(curr), 0);
        // Shouldn't reach here
        default:
            console.warn("Unknown step type in getStepComplexity");
            return 0;
    }
}

/**
 * Parses the childOrder string of a project version directory into an ordered array of child IDs
 * @param childOrder Child order string (e.g. "123,456,555,222" or "l(333,222,555),r(888,123,321)")
 * @returns Ordered array of child IDs
 */
export function parseChildOrder(childOrder: string): string[] {
    // Trim the input string
    childOrder = childOrder.trim();

    // If the input is empty, return an empty array
    if (!childOrder) return [];

    // Check if it's the root format
    const rootMatch = childOrder.match(/^l\((.*?)\)\s*,?\s*r\((.*?)\)$/);
    if (rootMatch) {
        const leftOrder = rootMatch[1].split(",").filter(Boolean);
        const rightOrder = rootMatch[2].split(",").filter(Boolean);

        // Check for nested parentheses
        if (leftOrder.some(item => item.includes("(") || item.includes(")")) ||
            rightOrder.some(item => item.includes("(") || item.includes(")"))) {
            return [];
        }

        return [...leftOrder, ...rightOrder];
    }

    // Split by comma and/or space
    const parts = childOrder.split(/[,\s]+/).filter(Boolean);

    // Make sure each part is either a UUID or alphanumeric code
    const validParts = parts.filter(part => uuidValidate(part) || /^[a-zA-Z0-9]+$/.test(part));

    // If the number of valid parts doesn't match the original number of parts, return an empty array
    return validParts.length === parts.length ? validParts : [];
}

export type UnsortedSteps = Array<EndStep | RoutineListStep | StartStep>;
export type SortedStepsWithDecisions = Array<DecisionStep | EndStep | RoutineListStep | StartStep>;

/**
 * Traverses the routine graph using Depth-First Search (DFS) and sorts steps by visitation order. 
 * Adds DecisionSteps where multiple outgoing links are found. These are used to move the run pointer 
 * to a different part of the steps array, ensuring that we can handle multiple paths and cycles 
 * when running the routine.
 * @param steps The nodes in the routine version, represented as steps
 * @param nodeLinks The links connecting the nodes in the routine graph.
 * @returns An array of nodes sorted by the order they were visited.
 */
export function sortStepsAndAddDecisions(
    steps: UnsortedSteps,
    nodeLinks: NodeLink[],
): SortedStepsWithDecisions {
    const startStep = steps.find(step => step.__type === RunStepType.Start);
    if (!startStep) {
        console.error("Routine does not have a StartStep. Cannot sort steps or generate DecisionSteps");
        return steps;
    }

    const visited: { [nodeId: string]: boolean } = {};
    let lastEndLocation = 1;
    const result: SortedStepsWithDecisions = [];

    // Get all but the last element
    const baseLocation = startStep.location.slice(0, -1);

    // Helper function to perform DFS
    function dfs(currentStep: StartStep | EndStep | RoutineListStep) {
        if (visited[currentStep.nodeId]) return;

        // Mark the current step as visited
        visited[currentStep.nodeId] = true;
        // Update location for the current step
        currentStep.location = [...baseLocation, lastEndLocation];
        lastEndLocation++;

        // Get all outgoing links from the current step's node
        const outgoingLinks = nodeLinks.filter(link => link.from?.id === currentStep.nodeId);
        // If there's one outgoing link, set the nextLocation to the next node's location
        if (outgoingLinks.length === 1) {
            const nextNode = steps.find(step => step.nodeId === outgoingLinks[0].to?.id);
            if (nextNode) {
                currentStep.nextLocation = visited[nextNode.nodeId] ? nextNode.location : [...baseLocation, lastEndLocation];
            }
        }
        // If there are multiple outgoing links, set the nextLocation to the DecisionStep we're about to create
        else if (outgoingLinks.length > 1) {
            currentStep.nextLocation = [...baseLocation, lastEndLocation];
        } else {
            currentStep.nextLocation = null;
        }

        // Add current step to result
        result.push(currentStep);

        // If there is more than one outgoing link, generate a DecisionStep
        if (outgoingLinks.length > 1) {
            const decisionStep: DecisionStep = {
                __type: RunStepType.Decision,
                description: "Select a path to continue",
                location: [...baseLocation, lastEndLocation],
                name: "Decision",
                options: outgoingLinks.map(link => {
                    const nextStep = steps.find(step => step.nodeId === link.to?.id);
                    if (
                        nextStep !== null &&
                        nextStep !== undefined &&
                        nextStep.__type !== RunStepType.Start
                    ) {
                        return {
                            link,
                            step: nextStep,
                        } as DecisionStep["options"][0];
                    }
                    return null;
                }).filter(Boolean) as unknown as DecisionStep["options"],
            };
            lastEndLocation++;
            // Add DecisionStep to result
            result.push(decisionStep);
        }
        // Traverse all outgoing links
        outgoingLinks.forEach(link => {
            const nextNode = steps.find(step => step.nodeId === link.to?.id);
            if (nextNode) {
                dfs(nextNode);
            }
        });
    }

    // Start DFS from the start node
    dfs(startStep as StartStep);

    return result;
}

/**
 * Converts a single-step routine into a Step object
 * @param routineVersion The routineVersion being run
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function singleRoutineToStep(
    routineVersion: RunnableRoutineVersion,
    languages: string[],
): SingleRoutineStep | null {
    return {
        __type: RunStepType.SingleRoutine,
        description: getTranslation(routineVersion, languages, true).description || null,
        location: asdf,
        name: getTranslation(routineVersion, languages, true).name || "Untitled",
        routineVersion,
    };
}

/**
 * Converts a multi-step routine into a Step object
 * @param routineVersion The routineVersion being run
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function multiRoutineToStep(
    routineVersion: RunnableRoutineVersion,
    languages: string[],
): MultiRoutineStep | null {
    // Convert existing nodes into steps
    const unsorted = (routineVersion.nodes || []).map(node => {
        const description = getTranslation(node, languages, true).description || null;
        const name = getTranslation(node, languages, true).name || "Untitled";
        const nodeId = node.id;

        switch (node.nodeType) {
            case NodeType.End:
                return {
                    __type: RunStepType.End,
                    description,
                    location: asdf,
                    name,
                    nodeId,
                    wasSuccessful: node.end?.wasSuccessful ?? false,
                } as EndStep;
            case NodeType.RoutineList:
                return {
                    __type: RunStepType.RoutineList,
                    description,
                    location: asdf,
                    name,
                    nodeId,
                    isOrdered: node.routineList?.isOrdered ?? false,
                    parentRoutineVersionId: routineVersion.id,
                } as RoutineListStep;
            case NodeType.Start:
                return {
                    __type: RunStepType.Start,
                    description,
                    location: asdf,
                    name,
                    nodeId,
                } as StartStep;
            default:
                return null;
        }
    }).filter(Boolean) as UnsortedSteps;
    // Sort steps by visitation order and add DecisionSteps where needed
    const sorted = sortStepsAndAddDecisions(unsorted, routineVersion.nodeLinks || []);
    return {
        __type: RunStepType.MultiRoutine,
        description: getTranslation(routineVersion, languages, true).description || null,
        location: asdf,
        name: getTranslation(routineVersion, languages, true).name || "Untitled",
        nodeLinks: routineVersion.nodeLinks || [],
        nodes: sorted,
        routineVersionId: routineVersion.id,
    };
}

/**
 * Converts a project into a Step object
 * @param projectVersion The projectVersion being run
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function projectToStep(
    projectVersion: RunnableProjectVersion,
    languages: string[],
): DirectoryStep | null {
    // Projects are represented as root directories
    const steps = projectVersion.directories?.
        map(directory => directoryToStep(directory, languages))?.
        filter(Boolean) as DirectoryStep[];
    //TODO project version does not have order field, so can't order directories. Maybe change this in future
    return {
        __type: RunStepType.Directory,
        description: getTranslation(projectVersion, languages, true).description || null,
        directoryId: null,
        hasBeenQueried: true,
        isOrdered: false,
        isRoot: true,
        location: asdf,
        name: getTranslation(projectVersion, languages, true).name || "Untitled",
        projectVersionId: projectVersion.id,
        steps,
    };
}

/**
 * Converts a directory into a Step object
 * @param projectVersionDirectory The projectVersionDirectory being run
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function directoryToStep(
    projectVersionDirectory: ProjectVersionDirectory,
    languages: string[],
): DirectoryStep | null {
    // TODO we currently only generate steps for routines in the directory. 
    // We can add more types to support other children later, and update the 
    // directory type to support nested directories.
    // const childOrder = parseChildOrder(projectVersionDirectory.childOrder || "");
    return {
        __type: RunStepType.Directory,
        description: getTranslation(projectVersionDirectory, languages, true).description || null,
        directoryId: null,
        hasBeenQueried: true,
        isOrdered: false,
        isRoot: true,
        location: asdf,
        name: getTranslation(projectVersionDirectory, languages, true).name || "Untitled",
        projectVersionId: projectVersionDirectory.projectVersion?.id ?? "",
        steps: [],
    };
}

/**
 * Converts a runnable object into a Step object
 * @param runnableObject The projectVersion or routineVersion being run
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function runnableObjectToStep(
    runnableObject: RunnableProjectVersion | RunnableRoutineVersion | null | undefined,
    languages: string[],
): RootStep | null {
    if (isOfType(runnableObject, "RoutineVersion")) {
        if (runnableObject.routineType === RoutineType.MultiStep) {
            return multiRoutineToStep(runnableObject, languages);
        } else {
            return singleRoutineToStep(runnableObject, languages);
        }
    } else if (isOfType(runnableObject, "ProjectVersion")) {
        return projectToStep(runnableObject, languages);
    }
    console.error("Invalid runnable object type in runnableObjectToStep", runnableObject);
    return null;
}

/**
 * Adds a list of subroutines to the root step object
 * @param subroutines List of subroutines to add
 * @param rootStep The root step object to add the subroutines to
 * @param languages Preferred languages to display step data in
 * @returns Updated root step object with the subroutines added
 */
export function addSubroutinesToStep(
    subroutines: RoutineVersion[],
    rootStep: RootStep,
    languages: string[],
): RootStep {
    let updatedRootStep = rootStep;
    for (const routineVersion of subroutines) {
        // We can only inject multi-step routines, as they carry all subroutines with them
        if (routineVersion.routineType !== RoutineType.MultiStep) continue;
        const subroutineStep = multiRoutineToStep(routineVersion, languages);
        if (!subroutineStep) continue;
        updatedRootStep = insertStep(subroutineStep as MultiRoutineStep, updatedRootStep);
    }
    return updatedRootStep;
}

/**
 * Adds a list of subdirectories to the root step object
 * @param subdirectories List of subdirectories to add
 * @param rootStep The root step object to add the subdirectories to
 * @param languages Preferred languages to display step data in
 * @returns Updated root step object with the subdirectories added
 */
export function addSubdirectoriesToStep(
    subdirectories: ProjectVersionDirectory[],
    rootStep: RootStep,
    languages: string[],
): RootStep {
    let updatedRootStep = rootStep;
    for (const directory of subdirectories) {
        const projectStep = directoryToStep(directory, languages);
        if (!projectStep) continue;
        updatedRootStep = insertStep(projectStep, updatedRootStep);
    }
    return updatedRootStep;
}

const TopBar = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "0.5rem",
    width: "100%",
    backgroundColor: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
}));

const TitleStepsStack = styled(Stack)(() => ({
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
}));

const ContentBox = styled(Box)(({ theme }) => ({
    background: theme.palette.mode === "light" ? "#c2cadd" : theme.palette.background.default,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    margin: "auto",
    marginBottom: "72px + env(safe-area-inset-bottom)",
    overflowY: "auto",
    minHeight: "100vh",
}));

const ActionBar = styled(Box)(({ theme }) => ({
    position: "fixed",
    bottom: "0",
    paddingBottom: "env(safe-area-inset-bottom)",
    height: pagePaddingBottom,
    width: "-webkit-fill-available",
    zIndex: 4,
    background: theme.palette.primary.dark,
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
}));

const actionBarGridStyle = {
    width: "min(100%, 700px)",
    margin: 0,
} as const;

export function RunView({
    display,
    isOpen,
    onClose,
    runnableObject,
}: RunViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // Find data in URL (e.g. current step, runID, whether or not this is a test run)
    const params = useReactSearch(null);
    const { runId, testMode } = useMemo(() => {
        return {
            runId: (typeof params.run === "string" && uuidValidate(base36ToUuid(params.run))) ? base36ToUuid(params.run) : undefined,
            testMode: params.run === "test",
        };
    }, [params]);
    const [run, setRun] = useState<RunProject | RunRoutine | undefined>(undefined);
    useEffect(() => {
        const run = exists(runnableObject?.you?.runs) ? (runnableObject.you?.runs as (RunProject | RunRoutine)[]).find(run => run.id === runId) : undefined;
        exists(run) && setRun(run);
    }, [runId, runnableObject?.you?.runs]);

    const [currStepLocation, setCurrStepLocation] = useState<number[]>(Array.isArray(params.step) ? params.step as number[] : []);
    useEffect(function updateLocationInUrlEffect() {
        addSearchParams(setLocation, { step: currStepLocation });
    }, [currStepLocation, params, setLocation]);

    /**
     * The amount of routine completed so far, measured in complexity. 
     * Used with the routine's total complexity to calculate percent complete
     */
    const [completedComplexity, setCompletedComplexity] = useState(0);

    /**
     * Every step completed so far, as an array of arrays.
     * Each step is an array that describes its nesting, like they appear in the URL (e.g. [1], [1,3], [1,5,2]).
     */
    const [progress, setProgress] = useState<number[][]>([]);

    const languages = useMemo(() => getUserLanguages(session), [session]);

    /**
     * Stores all queried steps, under one root step that represents the entire project/routine.
     * This can be traversed to find step data at a given location.
     */
    const [rootStep, setRootStep] = useState<RootStep | null>(null);

    useEffect(function convertRunnableObjectToStepTreeEffect() {
        setRootStep(runnableObjectToStep(runnableObject, languages));
    }, [languages, runnableObject]);

    useEffect(function calculateRunStatsEffect() {
        if (!run) return;
        // Set completedComplexity
        setCompletedComplexity(run.completedComplexity);
        // Calculate progress using run.steps
        const existingProgress: number[][] = [];
        for (const step of run.steps) {
            // If location found and is not a duplicate (a duplicate here indicates a mistake with storing run data elsewhere)
            if (step.step && !existingProgress.some((progress) => locationArraysMatch(progress, step.step))) {
                existingProgress.push(step.step);
            }
        }
        setProgress(existingProgress);
    }, [run]);

    /**
     * Display for current step number (last in location array).
     */
    const currentStepNumber = useMemo(() => {
        return currStepLocation.length === 0 ? -1 : Number(currStepLocation[currStepLocation.length - 1]);
    }, [currStepLocation]);

    const stepsInCurrentNode = useMemo(function stepsInCurrentNodeMemo() {
        if (!currStepLocation || !rootStep) return 1;
        return siblingsAtLocaiton(currStepLocation, rootStep);
    }, [currStepLocation, rootStep]);

    /**
     * Current step run data
     */
    const currStepRunData = useMemo<RunRoutineStep | undefined>(() => {
        const runStep = (run?.steps as any[])?.find((s: RunRoutineStep) => locationArraysMatch(s.step, currStepLocation));
        return runStep;
    }, [run?.steps, currStepLocation]);

    /**
     * Stores user inputs, which are uploaded to the run's data
     */
    const currUserInputs = useRef<{ [inputId: string]: string }>({});
    const handleUserInputsUpdate = useCallback((inputs: { [inputId: string]: string }) => {
        currUserInputs.current = inputs;
    }, []);
    useEffect(() => {
        const inputs: { [inputId: string]: string } = {};
        // Only set inputs for RunRoutine
        if (run && run.__typename === "RunRoutine" && Array.isArray(run.inputs)) {
            for (const input of run.inputs) {
                inputs[input.input.id] = input.data;
            }
        }
        if (JSON.stringify(inputs) !== JSON.stringify(currUserInputs.current)) {
            handleUserInputsUpdate(inputs);
        }
    }, [run, handleUserInputsUpdate]);

    // Track user behavior during step (time elapsed, context switches)
    /**
     * Interval to track time spent on each step.
     */
    const intervalRef = useRef<NodeJS.Timeout | null>(null);
    const timeElapsed = useRef<number>(0);
    const contextSwitches = useRef<number>(0);
    useEffect(() => {
        if (!currStepRunData) return;
        // Start tracking time
        intervalRef.current = setInterval(() => { timeElapsed.current += 1; }, 1000);
        // Reset context switches
        contextSwitches.current = 0;
        return () => {
            if (intervalRef.current) clearInterval(intervalRef.current);
        };
    }, [currStepRunData]);

    /** On tab change, add to contextSwitches */
    useEffect(function contextSwitchesMemo() {
        function handleTabChange() {
            if (currStepRunData) {
                contextSwitches.current += 1;
            }
        }
        window.addEventListener("focus", handleTabChange);
        return () => {
            window.removeEventListener("focus", handleTabChange);
        };
    }, [currStepRunData]);

    /**
     * Calculates the percentage of routine completed so far, measured in complexity / total complexity * 100
     */
    const progressPercentage = useMemo(() => getRunPercentComplete(completedComplexity, (run as RunProject)?.projectVersion?.complexity ?? (run as RunRoutine)?.routineVersion?.complexity), [completedComplexity, run]);

    const navigateToNextStepInParent = useCallback(() => {
        // make sure there are steps in the current location
        if (currStepLocation.length > 0) {
            const newCurrStepLocation = [...currStepLocation];
            newCurrStepLocation[newCurrStepLocation.length - 1] += 1; // increment the last index
            setCurrStepLocation(newCurrStepLocation);
        }
    }, [currStepLocation]);

    const navigateBackToParentStep = useCallback(() => {
        // make sure there are steps in the current location
        if (currStepLocation.length > 0) {
            const newCurrStepLocation = currStepLocation.slice(0, currStepLocation.length - 1); // remove the last index
            setCurrStepLocation(newCurrStepLocation);
        }
    }, [currStepLocation]);

    // Query current subroutine or directory, if needed. Main routine may have the data
    const [getSubroutines, { data: queriedSubroutines, loading: subroutinesLoading }] = useLazyFetch<RoutineVersionSearchInput, RoutineVersionSearchResult>(endpointGetRoutineVersions);
    const [getDirectories, { data: queriedSubdirectories, loading: directoriesLoading }] = useLazyFetch<ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult>(endpointGetProjectVersionDirectories);
    const [currentStep, setCurrentStep] = useState<RunStep | null>(null);
    useEffect(() => {
        // If no steps, redirect to first step
        if (currStepLocation.length === 0) {
            setCurrStepLocation([1]);
            return;
        }
        // Current step is the last step in steps list
        const currStep = stepFromLocation(currStepLocation, rootStep);
        // If current step was not found
        if (!currStep) {
            // Check if this is because the top level step list is empty
            if (currStepLocation.length === 1 && rootStep?.steps.length === 0) {
                // Navigate to the next step in the parent list or directory
                navigateToNextStepInParent();
            } else if (currStepLocation.length > 1) {
                // Navigate back to the parent step
                navigateBackToParentStep();
            }
            return;
        }
        // If current step is a list or directory, then redirect to first step in list or directory
        if (currStep.__type === RunStepType.RoutineList || currStep.__type === ProjectStepType.Directory) {
            // But first, make sure the list or directory is not empty
            // If it is, skip to the next step
            if (currStep.steps.length === 0) {
                // Navigate to the next step in the parent list or directory
                navigateToNextStepInParent();
                return;
            } else {
                const newStepList = [...currStepLocation, 1];
                setCurrStepLocation(newStepList);
            }
        }
        // If current step is a Directory and needs querying, then query the data
        if (currStep.__type === ProjectStepType.Directory && stepNeedsQuerying(currStep)) {
            getDirectories({ parentDirectoryId: currStep.directoryId });
            return;
        }
        // If current step is a Subroutine and needs querying, then query the data
        if (currStep.__type === RunStepType.SingleRoutine && stepNeedsQuerying(currStep)) {
            getSubroutines({ ids: [currStep.routineVersion.id] });
            return;
        }
        // Finally, if we don't need to query, we set the current step.
        setCurrentStep(currStep);
    }, [currStepLocation, getSubroutines, getDirectories, params, setCurrStepLocation, rootStep, navigateToNextStepInParent, navigateBackToParentStep]);


    useEffect(function addQueriedSubroutinesToRootEffect() {
        if (!queriedSubroutines) return;
        const subroutines = queriedSubroutines.edges.map(e => e.node);
        setRootStep(root => root ? addSubroutinesToStep(subroutines, root, languages) : null);
    }, [languages, queriedSubroutines]);

    useEffect(function addQueriedSubdirectoriesToRootEffect() {
        if (!queriedSubdirectories) return;
        const subdirectories = queriedSubdirectories.edges.map(e => e.node);
        setRootStep(root => root ? addSubdirectoriesToStep(subdirectories, root, languages) : null);
    }, [languages, queriedSubdirectories]);

    const { instructions, name } = useMemo(function parentDisplayInfoMemo() {
        const languages = getUserLanguages(session);
        // Find step above current step
        const currStepParent = rootStep ? stepFromLocation(currStepLocation.slice(0, -1), rootStep) : null;
        return {
            instructions: (run && run?.__typename === "RunRoutine" && run.routineVersion) ? getTranslation(run.routineVersion, languages, true).instructions : "",
            // Ignore name if it's for the main routine (i.e. step is still loading, probably)
            name: (currStepParent?.name && currStepLocation.length > 1) ? currStepParent.name : "",
        };
    }, [currStepLocation, run, session, rootStep]);

    const previousLocation = useMemo<number[] | null>(function previousLocationMemo() {
        if (!currentStep) return null;
        return getPreviousLocation(currentStep.location);
    }, [currentStep]);

    const nextLocation = useMemo<number[] | null>(function nextLocationMemo() {
        if (!currentStep) return null;
        return getNextLocation(currentStep.location, rootStep);
    }, [currentStep, rootStep]);

    //TODO
    const unsavedChanges = false;
    const subroutineComplete = true;

    /**
      * Navigate to the previous subroutine
      */
    const toPrevious = useCallback(function toPreviousCallback() {
        if (!previousLocation) return;
        setCurrStepLocation(previousLocation);
    }, [previousLocation, setCurrStepLocation]);

    const [logRunUpdate] = useLazyFetch<RunRoutineUpdateInput, RunRoutine>(endpointPutRunRoutine);
    const [logRunComplete] = useLazyFetch<RunRoutineCompleteInput, RunRoutine>(endpointPutRunRoutineComplete);
    /**
     * Navigate to the next subroutine, or complete the routine.
     * Also log progress, time elapsed, and other metrics
     */
    const toNext = useCallback(function toNextCallback() {
        // Find step data
        const currStep = stepFromLocation(currStepLocation, rootStep);
        // Calculate new progress and percent complete
        const newProgress = Array.isArray(progress) ? [...progress] : [];
        const newlyCompletedComplexity: number = (currStep ? getStepComplexity(currStep) : 0);
        const alreadyComplete = Boolean(newProgress.find(p => locationArraysMatch(p, currStepLocation)));
        // If step was not already completed, update progress
        if (!alreadyComplete) {
            newProgress.push(currStepLocation);
            setProgress(newProgress);
            setCompletedComplexity(c => c + newlyCompletedComplexity);
        }
        // Update current step
        if (nextLocation) setCurrStepLocation(nextLocation);
        // If in test mode return
        if (testMode || !run) return;
        // Now we can calculate data for the logs
        // Find parent RoutineList step, so we can get the nodeId 
        const currParentListStep: RoutineListStep = stepFromLocation(currStepLocation.slice(0, currStepLocation.length - 1), steps) as RoutineListStep;
        // Current step will be updated if it already exists in logged data, or created if not
        const stepsUpdate = currStepRunData ? [{
            id: currStepRunData.id,
            status: RunRoutineStepStatus.Completed,
            timeElapsed: (currStepRunData.timeElapsed ?? 0) + timeElapsed.current,
            contextSwitches: currStepRunData.contextSwitches + contextSwitches.current,
        }] : undefined;
        const stepsCreate = currStepRunData ? undefined : [{
            id: uuid(),
            order: newProgress.length,
            name: currStep?.name ?? "",
            nodeId: currParentListStep.nodeId,
            subroutineId: currParentListStep.routineVersionId,
            step: currStepLocation,
            timeElapsed: timeElapsed.current,
            contextSwitches: contextSwitches.current,
        }];
        // If a next step exists, update
        if (nextLocation) {
            fetchLazyWrapper<RunRoutineUpdateInput, RunRoutine>({
                fetch: logRunUpdate,
                inputs: {
                    id: run.id,
                    completedComplexity: alreadyComplete ? undefined : newlyCompletedComplexity,
                    stepsCreate,
                    stepsUpdate,
                    ...runInputsUpdate((run as any)?.inputs as RunRoutineInput[], currUserInputs.current), //TODO
                },
                onSuccess: (data) => {
                    setRun(data);
                },
            });
        }
        // Otherwise, mark as complete
        // TODO if running a project, the overall run may not be complete - just one routine in the project
        else {
            // Find node data
            const currNodeId = currStepRunData?.node?.id;
            // const currNode = routineVersion.nodes?.find(n => n.id === currNodeId);
            // const wasSuccessful = currNode?.end?.wasSuccessful ?? true;
            const wasSuccessful = true; //TODO
            fetchLazyWrapper<RunRoutineCompleteInput, RunRoutine>({
                fetch: logRunComplete,
                inputs: {
                    id: run.id,
                    exists: true,
                    completedComplexity: alreadyComplete ? undefined : newlyCompletedComplexity,
                    finalStepCreate: stepsCreate ? stepsCreate[0] : undefined,
                    finalStepUpdate: stepsUpdate ? stepsUpdate[0] : undefined,
                    name: run.name ?? getDisplay(runnableObject).title ?? "Unnamed Routine",
                    wasSuccessful,
                    ...runInputsUpdate((run as RunRoutine)?.inputs as RunRoutineInput[], currUserInputs.current),
                },
                successMessage: () => ({ messageKey: "RoutineCompleted" }),
                onSuccess: () => {
                    PubSub.get().publish("celebration");
                    removeSearchParams(setLocation, ["run", "step"]);
                    tryOnClose(onClose, setLocation);
                },
            });
        }
    }, [currStepLocation, rootStep, progress, nextLocation, testMode, run, currStepRunData, logRunUpdate, logRunComplete, runnableObject, setLocation, onClose]);

    /**
     * End run after reaching an EndStep
     * TODO if run is a project, this is not the end. Just the end of a routine in the project
     */
    const reachedEndStep = useCallback(function reachedEndStepCallback(step: EndStep) {
        // Check if end was successfully reached
        const success = step.wasSuccessful ?? true;
        // Don't actually do it if in test mode
        if (testMode || !run) {
            if (success) PubSub.get().publish("celebration");
            removeSearchParams(setLocation, ["run", "step"]);
            tryOnClose(onClose, setLocation);
            return;
        }
        // Log complete. No step data because this function was called from a decision node, 
        // which we currently don't store data about
        fetchLazyWrapper<RunRoutineCompleteInput, RunRoutine>({
            fetch: logRunComplete,
            inputs: {
                id: run.id,
                exists: true,
                name: run.name ?? getDisplay(runnableObject).title ?? "Unnamed Routine",
                wasSuccessful: success,
                ...runInputsUpdate((run as RunRoutine)?.inputs as RunRoutineInput[], currUserInputs.current),
            },
            successMessage: () => ({ messageKey: "RoutineCompleted" }),
            onSuccess: () => {
                PubSub.get().publish("celebration");
                removeSearchParams(setLocation, ["run", "step"]);
                tryOnClose(onClose, setLocation);
            },
        });
    }, [testMode, run, runnableObject, logRunComplete, setLocation, onClose]);

    /**
     * Stores current progress, both for overall routine and the current subroutine
     */
    const saveProgress = useCallback(function saveProgressCallback() {
        // Dont do this in test mode, or if there's no run data
        if (testMode || !run) return;
        // Find current step in run data
        const stepUpdate = currStepRunData ? {
            id: currStepRunData.id,
            timeElapsed: (currStepRunData.timeElapsed ?? 0) + timeElapsed.current,
            contextSwitches: currStepRunData.contextSwitches + contextSwitches.current,
        } : undefined;
        // Send data to server
        //TODO support run projects
        fetchLazyWrapper<RunRoutineUpdateInput, RunRoutine>({
            fetch: logRunUpdate,
            inputs: {
                id: run.id,
                stepsUpdate: stepUpdate ? [stepUpdate] : undefined,
                ...runInputsUpdate((run as RunRoutine)?.inputs as RunRoutineInput[], currUserInputs.current),
            },
            onSuccess: (data) => {
                setRun(data);
            },
        });
    }, [currStepRunData, logRunUpdate, run, testMode]);

    /**
     * End routine early
     */
    const toFinishNotComplete = useCallback(function toFinishNotCompleteCallback() {
        saveProgress();
        removeSearchParams(setLocation, ["run", "step"]);
        tryOnClose(onClose, setLocation);
    }, [onClose, saveProgress, setLocation]);

    /**
     * Navigate to selected decision
     */
    const toDecision = useCallback(function toDecisionCallback(step: MultiRoutineStep["nodes"][0]) {
        // If end node, finish
        if (step.__type === RunStepType.End) {
            reachedEndStep(step);
            return;
        }
        // Navigate to current step
        setCurrStepLocation(step.location);
    }, [reachedEndStep, setCurrStepLocation]);

    const handleLoadSubroutine = useCallback(function handleLoadSubroutineCallback(id: string) {
        getSubroutines({ ids: [id] });
    }, [getSubroutines]);

    /**
     * Displays either a subroutine view or decision view
     */
    const childView = useMemo(function childViewMemo() {
        console.log("calculating childview", currentStep, run);
        if (!currentStep) return null;
        switch (currentStep.__type) {
            case RunStepType.SingleRoutine:
                return <SubroutineView
                    handleUserInputsUpdate={handleUserInputsUpdate}
                    handleSaveProgress={saveProgress}
                    onClose={noop}
                    owner={run?.team ?? run?.user} //TODO not right, but also unused right now
                    routineVersion={(currentStep as SingleRoutineStep).routineVersion}
                    run={run as RunRoutine}
                    loading={subroutinesLoading}
                />;
            case RunStepType.Directory:
                return null; //TODO
            // return <DirectoryView
            // />;
            case RunStepType.Decision:
                return <DecisionView
                    data={currentStep as DecisionStep}
                    handleDecisionSelect={toDecision}
                    onClose={noop}
                />;
            // TODO come up with default view
            default:
                return null;
        }
    }, [currentStep, handleUserInputsUpdate, run, saveProgress, subroutinesLoading, toDecision]);

    return (
        <MaybeLargeDialog
            display={display}
            id="run-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <Box minHeight="100vh">
                <Box margin="auto">
                    {/* Contains name bar and progress bar */}
                    <Stack direction="column" spacing={0}>
                        <TopBar>
                            {/* Close Icon */}
                            <IconButton
                                edge="end"
                                aria-label="close"
                                onClick={toFinishNotComplete}
                                color="inherit"
                            >
                                <CloseIcon width='32px' height='32px' />
                            </IconButton>
                            {/* Title and steps */}
                            <TitleStepsStack direction="row" spacing={1}>
                                <Typography variant="h5" component="h2">{name}</Typography>
                                {(currentStepNumber >= 0 && stepsInCurrentNode >= 0) ?
                                    <Typography variant="h5" component="h2">({currentStepNumber} of {stepsInCurrentNode})</Typography>
                                    : null}
                                {/* Help icon */}
                                {instructions && <HelpButton markdown={instructions} />}
                            </TitleStepsStack>
                            {/* Steps explorer drawer */}
                            {rootStep !== null && rootStep.__type !== RunStepType.SingleRoutine && <RunStepsDialog
                                currStep={currStepLocation}
                                handleLoadSubroutine={handleLoadSubroutine}
                                handleCurrStepLocationUpdate={setCurrStepLocation}
                                history={progress}
                                percentComplete={progressPercentage}
                                rootStep={rootStep}
                            />}
                        </TopBar>
                        {/* Progress bar */}
                        <LinearProgress color="secondary" variant="determinate" value={completedComplexity / ((run as RunRoutine)?.routineVersion?.complexity ?? (run as RunProject)?.projectVersion?.complexity ?? 1) * 100} sx={{ height: "15px" }} />
                    </Stack>
                    <ContentBox>
                        {childView}
                    </ContentBox>
                    <ActionBar>
                        <Grid container spacing={2} sx={actionBarGridStyle}>
                            {/* There are only ever 1 or 2 options shown. 
                        In either case, we want the buttons to be placed as 
                        if there are always 2 */}
                            <Grid item xs={6} p={1}>
                                {previousLocation && <Button
                                    fullWidth
                                    startIcon={<ArrowLeftIcon />}
                                    onClick={toPrevious}
                                    disabled={unsavedChanges}
                                    variant="outlined"
                                >
                                    {t("Previous")}
                                </Button>}
                            </Grid>
                            <Grid item xs={6} p={1}>
                                {nextLocation && (<Button
                                    fullWidth
                                    startIcon={<ArrowRightIcon />}
                                    onClick={toNext} // NOTE: changes are saved on next click
                                    disabled={!subroutineComplete}
                                    variant="contained"
                                >
                                    {t("Next")}
                                </Button>)}
                                {!nextLocation && currentStep?.__type !== RunStepType.Decision && (<Button
                                    fullWidth
                                    startIcon={<SuccessIcon />}
                                    onClick={toNext}
                                    variant="contained"
                                >
                                    {t("Complete")}
                                </Button>)}
                            </Grid>
                        </Grid>
                    </ActionBar>
                </Box>
            </Box>
        </MaybeLargeDialog>
    );
}
