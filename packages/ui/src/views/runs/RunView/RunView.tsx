import { endpointGetProjectVersionDirectories, endpointGetRoutineVersions, endpointPutRunRoutine, endpointPutRunRoutineComplete, exists, Node, NodeLink, NodeRoutineListItem, NodeType, ProjectVersion, ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult, RoutineVersion, RoutineVersionSearchInput, RoutineVersionSearchResult, RunProject, RunRoutine, RunRoutineCompleteInput, RunRoutineInput, RunRoutineStep, RunRoutineStepStatus, RunRoutineUpdateInput, uuid, uuidValidate } from "@local/shared";
import { Box, Button, Grid, IconButton, LinearProgress, Stack, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { RunStepsDialog } from "components/dialogs/RunStepsDialog/RunStepsDialog";
import { ArrowLeftIcon, ArrowRightIcon, CloseIcon, SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, removeSearchParams, useLocation } from "route";
import { DecisionStep, DirectoryStep, EndStep, ProjectStep, RoutineListStep, RoutineStep, SubroutineStep } from "types";
import { ProjectStepType, RoutineStepType } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useReactSearch } from "utils/hooks/useReactSearch";
import { base36ToUuid, tryOnClose } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { getRunPercentComplete, locationArraysMatch, routineVersionHasSubroutines, runInputsUpdate } from "utils/runUtils";
import { SessionContext } from "utils/SessionContext";
import { DecisionView } from "../DecisionView/DecisionView";
import { SubroutineView } from "../SubroutineView/SubroutineView";
import { RunViewProps } from "../types";

/**
 * Maximum routine nesting supported
 */
const MAX_NESTING = 20;

/**
 * Inserts step data into a given ProjectStep, where id matches
 * If the provided steps object is of type DirectoryStep, this function
 * will recursively traverse the nested steps and replace the matching
 * step with the provided stepData. If the steps object is of type
 * RoutineListStep, the existing logic of insertStep is used.
 * 
 * @param {RoutineListStep | DirectoryStep} stepData - Data to insert into the steps object.
 * @param {RoutineListStep | DirectoryStep} steps - ProjectStep to check in.
 * @returns {RoutineListStep | DirectoryStep} - Updated step. Returns the original step if the step was not found.
 */
const insertStep = (stepData: RoutineListStep | DirectoryStep, steps: RoutineListStep | DirectoryStep): RoutineListStep | DirectoryStep => {
    const recursiveInsert = (stepData: RoutineListStep | DirectoryStep, steps: ProjectStep): ProjectStep => {
        if (steps.type === RoutineStepType.Subroutine && "routineVersionId" in stepData && stepData.routineVersionId) {
            if ((steps as SubroutineStep).routineVersion.id === stepData.routineVersionId) {
                return stepData;
            }
        } else if (steps.type === RoutineStepType.RoutineList || steps.type === ProjectStepType.Directory) {
            for (let i = 0; i < steps.steps.length; i++) {
                steps.steps[i] = recursiveInsert(stepData, steps.steps[i]);
            }
        }
        return steps;
    };

    if (steps.type === RoutineStepType.RoutineList || steps.type === ProjectStepType.Directory) {
        return recursiveInsert(stepData, steps) as RoutineListStep | DirectoryStep;
    } else {
        return steps; // if steps is neither a RoutineListStep nor a DirectoryStep, return it as is.
    }
};

/**
 * Find the step array of a given nodeId
 * @param nodeId The nodeId to search for
 * @param step The current step object, since this is recursive
 * @param location The current location array, since this is recursive
 * @return The step array of the given step
 */
const locationFromNodeId = (nodeId: string, step: ProjectStep | null, location: number[] = []): number[] | null => {
    if (!step) return null;
    // Only routine lists and subroutines can be linked to a node
    if ((step.type === RoutineStepType.RoutineList || step.type === RoutineStepType.Subroutine) && step.nodeId === nodeId) {
        return location;
    }
    // If step is a routine list or a directory, it might contain a match.
    if ((step.type === RoutineStepType.RoutineList) || (step.type === ProjectStepType.Directory)) {
        for (let i = 0; i < step.steps.length; i++) {
            const currStep = step.steps[i];
            let currLocation: number[] | null = null;
            // Check if a subroutine is a match
            if (currStep.type === RoutineStepType.Subroutine && currStep.nodeId === nodeId) {
                return [...location, i + 1];
            }
            // Recurse on every nested routine list or directory
            if ((currStep.type === RoutineStepType.RoutineList) || (currStep.type === ProjectStepType.Directory)) {
                currLocation = locationFromNodeId(nodeId, currStep, [...location, i + 1]);
                if (currLocation) return currLocation;
            }
        }
    }
    return null;
};

/**
 * Find the step array of a given routineVersionId
 * @param routineVersionId The routineVersionId to search for
 * @param step The overall step object (or when recursed, the current step object)
 * @param location The current location array. Only used when recursed.
 * @return The step array of the given step
 */
const locationFromRoutineVersionId = (routineVersionId: string, step: ProjectStep | null, location: number[] = []): number[] | null => {
    if (!step) return null;
    // If step is a subroutine, it's either a single-step routine, or it is not fully-loaded. 
    // Either way, we check its routineVersion.id for a match.
    // If step is a routine list, it might be a match or it might contain a match.
    if ((step.type === RoutineStepType.Subroutine && step.routineVersion.id === routineVersionId)
        || (step.type === RoutineStepType.RoutineList && step.routineVersionId === routineVersionId)) {
        return location;
    }
    // If step is a routine list or a directory, it might contain a match.
    if ((step.type === RoutineStepType.RoutineList) || (step.type === ProjectStepType.Directory)) {
        for (let i = 0; i < step.steps.length; i++) {
            const currStep = step.steps[i];
            let currLocation: number[] | null = null;
            // Check if a subroutine is a match
            if (currStep.type === RoutineStepType.Subroutine && currStep.routineVersion.id === routineVersionId) {
                return [...location, i + 1];
            }
            // Recurse on every nested routine list or directory
            if ((currStep.type === RoutineStepType.RoutineList) || (currStep.type === ProjectStepType.Directory)) {
                currLocation = locationFromRoutineVersionId(routineVersionId, currStep, [...location, i + 1]);
                if (currLocation) return currLocation;
            }
        }
    }
    return null;
};

/**
 * Uses a location array to find the step at a given location 
 * NOTE: Must have been queried already
 * @param locationArray Array of step numbers that describes nesting of requested step
 * @param steps ProjectStep for the overall project or routine being run
 * @returns ProjectStep for the requested step, or null if not found
 */
const stepFromLocation = (locationArray: number[], steps: ProjectStep | null): ProjectStep | null => {
    if (!steps) return null;
    let currNestedSteps: ProjectStep | null = steps;
    // If array too large, probably an error
    if (locationArray.length > MAX_NESTING) {
        console.error(`Location array too large in findStep: ${locationArray}`);
        return null;
    }
    // Loop through location array
    for (let i = 0; i < locationArray.length; i++) {
        // Can only continue if end not reached and step is a routine list or directory (no other step type has substeps)
        if (
            currNestedSteps !== null &&
            (currNestedSteps.type === RoutineStepType.RoutineList || currNestedSteps.type === ProjectStepType.Directory)
        ) {
            currNestedSteps =
                currNestedSteps.steps.length > Math.max(locationArray[i] - 1, 0)
                    ? currNestedSteps.steps[Math.max(locationArray[i] - 1, 0)]
                    : null;
        }
    }
    return currNestedSteps;
};

/**
 * Determines if a step (either subroutine or directory) needs additional queries, or if it already 
 * has enough data to render
 * @param step The step to check
 * @returns True if the step needs additional queries, false otherwise
 */
const stepNeedsQuerying = (step: ProjectStep | null | undefined): boolean => {
    if (!step) return false;
    // Handle SubroutineStep
    if (step.type === RoutineStepType.Subroutine) {
        const currSubroutine: Partial<RoutineVersion> = (step as SubroutineStep).routineVersion;
        return routineVersionHasSubroutines(currSubroutine);
    }
    // Handle DirectoryStep
    if (step.type === ProjectStepType.Directory) {
        const currDirectory: Partial<DirectoryStep> = step as DirectoryStep;
        // If DirectoryStep has its own substeps, then it needs querying
        return Boolean(currDirectory.steps && currDirectory.steps.length > 0);
    }
    return false;
};

/**
 * Calculates the complexity of a step
 * @param step The step to calculate the complexity of
 * @returns The complexity of the step
 */
const getStepComplexity = (step: ProjectStep): number => {
    switch (step.type) {
        // One decision, so one complexity
        case RoutineStepType.Decision:
            return 1;
        // Complexity of subroutines stored in routine data
        case RoutineStepType.Subroutine:
            return (step as SubroutineStep).routineVersion.complexity;
        // Complexity of a list is the sum of its children's complexities
        case RoutineStepType.RoutineList:
        case ProjectStepType.Directory:
            return (step as RoutineListStep).steps.reduce((acc, curr) => acc + getStepComplexity(curr), 0);
    }
};

/**
 * Converts a routine version (can be the main routine or a subroutine) into a RoutineStep
 * @param routineVersion The routineVersion to convert
 * @param languages Preferred languages to display step data in
 * @returns RoutineStep for the given routine, or null if invalid
 */
const convertRoutineVersionToStep = (
    routineVersion: RoutineVersion | null | undefined,
    languages: string[],
): RoutineListStep | null => {
    // Check for required data to calculate steps
    if (!routineVersion || !routineVersion.nodes || !routineVersion.nodeLinks) {
        console.log("routineVersion does not have enough data to calculate steps");
        return null;
    }
    // Find all nodes that are routine lists
    let routineListNodes: Node[] = routineVersion.nodes.filter(({ nodeType }) => nodeType === NodeType.RoutineList);
    // Also find the start node
    const startNode = routineVersion.nodes.find((node: Node) => node.nodeType === NodeType.Start);
    // Sort by column, then row
    routineListNodes = routineListNodes.sort((a, b) => {
        const aCol = a.columnIndex ?? 0;
        const bCol = b.columnIndex ?? 0;
        if (aCol !== bCol) return aCol - bCol;
        const aRow = a.rowIndex ?? 0;
        const bRow = b.rowIndex ?? 0;
        return aRow - bRow;
    });
    // Find all nodes that are end nodes
    const endNodes = routineVersion.nodes.filter(({ nodeType }) => nodeType === NodeType.End);
    // Create result steps arrays
    const steps: RoutineStep[] = [];
    const endSteps: EndStep[] = [];
    // If multiple links from start node, create decision step
    const startLinks = routineVersion.nodeLinks.filter((link: NodeLink) => link.from.id === startNode?.id);
    if (startLinks.length > 1) {
        steps.push({
            type: RoutineStepType.Decision,
            parentRoutineVersionId: routineVersion.id,
            links: startLinks,
            name: "Decision",
            description: "Select a subroutine to run next",
        });
    }
    // Loop through all routine list nodes
    for (const node of routineListNodes) {
        // Find all subroutine steps, and sort by index
        const subroutineSteps: SubroutineStep[] = [...node.routineList!.items]
            .sort((r1, r2) => r1.index - r2.index)
            .map((item: NodeRoutineListItem) => ({
                type: RoutineStepType.Subroutine,
                nodeId: node.id,
                parentRoutineVersionId: routineVersion.id,
                index: item.index,
                routineVersion: item.routineVersion,
                name: getTranslation(item.routineVersion, languages, true).name ?? "Untitled",
                description: getTranslation(item.routineVersion, languages, true).description ?? "Description not found matching selected language",
            }));
        // Find decision step
        const links = routineVersion.nodeLinks.filter((link: NodeLink) => link.from.id === node.id);
        const decisionSteps: DecisionStep[] = links.length > 1 ? [{
            type: RoutineStepType.Decision,
            parentRoutineVersionId: routineVersion.id,
            links,
            name: "Decision",
            description: "Select a subroutine to run next",
        }] : [];
        steps.push({
            type: RoutineStepType.RoutineList,
            nodeId: node.id,
            parentRoutineVersionId: routineVersion.id,
            isOrdered: node.routineList?.isOrdered ?? false,
            name: getTranslation(node, languages, true).name ?? "Untitled",
            description: getTranslation(node, languages, true).description ?? "Description not found matching selected language",
            steps: [...subroutineSteps, ...decisionSteps] as Array<SubroutineStep | DecisionStep>,
            endSteps: [], // N/A here
        });
    }
    // Loop through all end nodes
    for (const node of endNodes) {
        // Add end step
        endSteps.push({
            type: "End",
            name: getTranslation(node, languages, true).name ?? "Untitled",
            description: getTranslation(node, languages, true).description ?? "Description not found matching selected language",
            wasSuccessful: node.end?.wasSuccessful ?? false,
            nodeId: node.id,
        });
    }
    // Return result steps
    return {
        type: RoutineStepType.RoutineList,
        parentRoutineVersionId: routineVersion.id,
        routineVersionId: routineVersion.id,
        isOrdered: true,
        name: getTranslation(routineVersion, languages, true).name ?? "Untitled",
        description: getTranslation(routineVersion, languages, true).description ?? "Description not found matching selected language",
        steps,
        endSteps,
    };
};

/**
 * Parses the childOrder string of a project version directory into an ordered array of child IDs
 * @param childOrder Child order string (e.g. "123,456,555,222" or "l(333,222,555),r(888,123,321)")
 * @returns Ordered array of child IDs
 */
const parseChildOrder = (childOrder: string): string[] => {
    // If it's the root format, get the left and right orders and combine them
    const match = childOrder.match(/^l\((.*?)\),r\((.*?)\)$/);
    if (match) {
        const leftOrder = match[1].split(",");
        const rightOrder = match[2].split(",");
        return [...leftOrder, ...rightOrder];
    }
    // Otherwise, split by comma
    else {
        return childOrder.split(",");
    }
};

/**
 * Converts a project version into a ProjectStep
 * @param projectVersion The projectVersion to convert
 * @param languages Preferred languages to display step data in
 * @returns ProjectStep for the given project, or null if invalid
 */
const convertProjectVersionToStep = (
    projectVersion: Pick<ProjectVersion, "directories" | "translations"> | null | undefined,
    languages: string[],
): DirectoryStep | null => {
    // Check if the projectVersion object and its directories array are not null or undefined
    if (!projectVersion || !projectVersion.directories) {
        console.log("projectVersion does not have enough data to calculate steps");
        return null;
    }
    let directories = [...projectVersion.directories];
    // Find the root directory in the directories array
    const rootDirectory = directories.find(directory => directory.isRoot);
    if (rootDirectory) {
        // If a root directory is found, parse its childOrder string into an array of child IDs
        const rootChildOrder = parseChildOrder(rootDirectory.childOrder || "");
        // Sort the directories array based on the order of child IDs
        directories = directories.sort((a, b) => rootChildOrder.indexOf(a.id) - rootChildOrder.indexOf(b.id));
    }
    // Initialize an array to store the steps for the entire project
    const resultSteps: ProjectStep[] = [];
    // Loop through each directory
    for (const directory of directories) {
        // Parse the childOrder string of the directory into an array of child IDs
        const childOrder = parseChildOrder(directory.childOrder || "");
        // Sort the childRoutineVersions array of the directory based on the order of child IDs
        const sortedChildRoutines = directory.childRoutineVersions.sort((a, b) => childOrder.indexOf(a.id) - childOrder.indexOf(b.id));
        // Convert each child routine into a step and store them in an array
        const directorySteps: RoutineListStep[] = sortedChildRoutines.map(routineVersion => convertRoutineVersionToStep(routineVersion, languages) as RoutineListStep);
        // Create a ProjectStep object for the directory and push it into the resultSteps array
        resultSteps.push({
            type: ProjectStepType.Directory,
            directoryId: directory.id,
            isOrdered: true,
            isRoot: false,
            name: getTranslation(directory, languages, true).name ?? "Untitled",
            description: getTranslation(directory, languages, true).description ?? "Description not found matching selected language",
            steps: directorySteps,
        });
    }
    // Return a ProjectListStep object for the entire project, which includes all the steps calculated above
    return {
        type: ProjectStepType.Directory,
        isOrdered: true,
        isRoot: true,
        name: getTranslation(projectVersion, languages, true).name ?? "Untitled",
        description: getTranslation(projectVersion, languages, true).description ?? "Description not found matching selected language",
        steps: resultSteps,
    };
};

export const RunView = ({
    display = "page",
    onClose,
    runnableObject,
    zIndex = 400,
}: RunViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    console.log("run view", zIndex);

    // Find data in URL (e.g. current step, runID, whether or not this is a test run)
    const params = useReactSearch(null);
    const { runId, testMode } = useMemo(() => {
        return {
            runId: (typeof params.run === "string" && uuidValidate(base36ToUuid(params.run, false))) ? base36ToUuid(params.run) : undefined,
            testMode: params.run === "test",
        };
    }, [params]);
    const [run, setRun] = useState<RunProject | RunRoutine | undefined>(undefined);
    useEffect(() => {
        const run = exists(runnableObject?.you?.runs) ? (runnableObject.you?.runs as (RunProject | RunRoutine)[]).find(run => run.id === runId) : undefined;
        exists(run) && setRun(run);
    }, [runId, runnableObject?.you?.runs]);

    const [currStepLocation, setCurrStepLocation] = useState<number[]>(Array.isArray(params.step) ? params.step as number[] : []);
    /**
     * Update URL when currStepLocation changes
     */
    useEffect(() => {
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
     * Stores all queried steps, under one main step that represents the entire routine.
     * This can be traversed to find step data at a given location.
     */
    const [steps, setSteps] = useState<RoutineListStep | DirectoryStep | null>(null);

    /**
     * Converts the overall run into a tree of steps, and stores it in the steps ref.
     */
    useEffect(() => {
        if (runnableObject.__typename === "RoutineVersion") {
            setSteps(convertRoutineVersionToStep(runnableObject as RoutineVersion, languages));
        }
        else {
            setSteps(convertProjectVersionToStep(runnableObject as ProjectVersion, languages));
        }
    }, [languages, runnableObject]);

    /**
     * When run data is loaded, set completedComplexity and steps completed
     */
    useEffect(() => {
        if (!run || !steps) return;
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
    }, [run, steps]);

    /**
     * Display for current step number (last in location array).
     */
    const currentStepNumber = useMemo(() => {
        return currStepLocation.length === 0 ? -1 : Number(currStepLocation[currStepLocation.length - 1]);
    }, [currStepLocation]);

    /**
     * The number of steps in the current-level node, or -1 if not found.
     */
    const stepsInCurrentNode = useMemo(() => {
        if (!currStepLocation || !steps) return -1;
        // For each step in ids array (except for the last id), find the nested step in the steps array.
        // If it doesn't exist, return -1;
        let currNestedSteps: RoutineStep | DirectoryStep = steps;
        for (let i = 0; i < currStepLocation.length - 1; i++) {
            if (currNestedSteps.type === RoutineStepType.RoutineList) {
                const currStepNum = Math.max(0, currStepLocation[i] - 1);
                const curr = currNestedSteps.steps.length > currStepNum ? currNestedSteps.steps[currStepNum] : null;
                if (curr && "type" in curr) currNestedSteps = curr;
            } else if (currNestedSteps.type === ProjectStepType.Directory) {
                const currStepNum = Math.max(0, currStepLocation[i] - 1);
                const curr = currNestedSteps.steps.length > currStepNum ? currNestedSteps.steps[currStepNum] : null;
                if (curr && "type" in curr) currNestedSteps = curr;
            }
        }
        // Return number of steps in current node
        if (currNestedSteps.type === RoutineStepType.RoutineList) {
            return (currNestedSteps as RoutineListStep).steps.length;
        } else if (currNestedSteps.type === ProjectStepType.Directory) {
            return (currNestedSteps as DirectoryStep).steps.length;
        } else {
            return -1;
        }
    }, [currStepLocation, steps]);


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

    /**
     * On tab change, add to contextSwitches
     */
    useEffect(() => {
        const handleTabChange = (event: any) => {
            if (currStepRunData) {
                contextSwitches.current += 1;
            }
        };
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
    const [currentStep, setCurrentStep] = useState<ProjectStep | null>(null);
    useEffect(() => {
        // If no steps, redirect to first step
        if (currStepLocation.length === 0) {
            setCurrStepLocation([1]);
            return;
        }
        // Current step is the last step in steps list
        const currStep = stepFromLocation(currStepLocation, steps);
        // If current step was not found
        if (!currStep) {
            // Check if this is because the top level step list is empty
            if (currStepLocation.length === 1 && steps?.steps.length === 0) {
                // Navigate to the next step in the parent list or directory
                navigateToNextStepInParent();
            } else if (currStepLocation.length > 1) {
                // Navigate back to the parent step
                navigateBackToParentStep();
            }
            return;
        }
        // If current step is a list or directory, then redirect to first step in list or directory
        if (currStep.type === RoutineStepType.RoutineList || currStep.type === ProjectStepType.Directory) {
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
        if (currStep.type === ProjectStepType.Directory && stepNeedsQuerying(currStep)) {
            getDirectories({ parentDirectoryId: currStep.directoryId });
            return;
        }
        // If current step is a Subroutine and needs querying, then query the data
        if (currStep.type === RoutineStepType.Subroutine && stepNeedsQuerying(currStep)) {
            getSubroutines({ ids: [currStep.routineVersion.id] });
            return;
        }
        // Finally, if we don't need to query, we set the current step.
        setCurrentStep(currStep);
    }, [currStepLocation, getSubroutines, getDirectories, params, setCurrStepLocation, steps, navigateToNextStepInParent, navigateBackToParentStep]);


    /**
     * When new subroutine data is fetched, inject it into steps. 
     */
    useEffect(() => {
        if (!queriedSubroutines) return;
        // Convert to RoutineStep
        const queriedData = queriedSubroutines.edges.map(e => e.node);
        let updatedSteps = steps;
        for (const queriedRoutine of queriedData) {
            const subroutineStep = convertRoutineVersionToStep(queriedRoutine, languages);
            if (!subroutineStep) continue;
            // Inject into steps
            updatedSteps = updatedSteps ? insertStep(subroutineStep, updatedSteps) : subroutineStep;
        }
        setSteps(updatedSteps);
    }, [languages, queriedSubroutines, steps]);

    /**
     * When new subproject is fetched, inject it into steps.
     */
    useEffect(() => {
        if (!queriedSubdirectories) return;
        // Convert to ProjectStep
        const queriedData = queriedSubdirectories.edges.map(e => e.node);
        const subdirectoryStep = convertProjectVersionToStep({ directories: queriedData, translations: [] }, languages);
        if (!subdirectoryStep) return;
        // Inject into steps
        setSteps(s => s ? insertStep(subdirectoryStep, s) : subdirectoryStep);
    }, [languages, queriedSubdirectories]);

    const { instructions, name } = useMemo(() => {
        const languages = getUserLanguages(session);
        // Find step above current step
        const currStepParent = stepFromLocation(currStepLocation.slice(0, -1), steps);
        return {
            instructions: (run && run?.__typename === "RunRoutine" && run.routineVersion) ? getTranslation(run.routineVersion, languages, true).instructions : "",
            // Ignore name if it's for the main routine (i.e. step is still loading, probably)
            name: (currStepParent?.name && currStepLocation.length > 1) ? currStepParent.name : "",
        };
    }, [currStepLocation, run, session, steps]);

    /**
     * Calculates previous step location array, or null
     * Examples: [2] => [1], [1] => null, [2, 2] => [2, 1], [2, 1] => [2, num in previous step]
     */
    const previousStep = useMemo<number[] | null>(() => {
        if (currStepLocation.length === 0) return null;
        // Loop backwards. If curr > 1, then return curr - 1 and remove elements after
        for (let i = currStepLocation.length - 1; i >= 0; i--) {
            const currStepNumber = currStepLocation[i];
            if (currStepNumber > 1) return [...currStepLocation.slice(0, i), currStepNumber - 1];
        }
        return null;
    }, [currStepLocation]);

    /**
     * Calculates next step location array, or null. Loops backwards through location array 
     * until a location can be incremented, then loops forward until a subroutine without any 
     * of its own subroutines is found.
     * Examples: [2] => [3] OR [2, 1] if at end of list
     */
    const nextStep = useMemo<number[] | null>(() => {
        // If current step is a decision, return null. 
        // This is because the next step is determined by the decision, not automatically
        if (currentStep?.type === RoutineStepType.Decision || currentStep?.type === ProjectStepType.Directory) return null;
        // If no current step, default to [1]
        let result = currStepLocation.length === 0 ? [1] : [...currStepLocation];
        let listStepFound = false;
        // Loop backwards until a RoutineList node or Directory node in stepParams can be incremented. Remove elements after that
        for (let i = result.length - 1; i >= 0; i--) {
            // Update current step
            const currStep: ProjectStep | null = stepFromLocation(result.slice(0, i), steps);
            // If step not found, we cannot continue
            if (!currStep) return null;
            // If current step is a RoutineList or Directory, check if there is another step in the list
            if (currStep.type === RoutineStepType.RoutineList || currStep.type === ProjectStepType.Directory) {
                if (currStep.steps.length > result[i]) {
                    listStepFound = true;
                    result[i]++;
                    result = result.slice(0, i + 1);
                    break;
                }
            }
        }
        // If a RoutineList or Directory to increment was not found, return null
        if (!listStepFound) return null;
        // Continue finding the next step which is not a list or directory.
        let currNextStep: ProjectStep | null = stepFromLocation(result, steps);
        let endFound = !currNextStep;
        while (!endFound) {
            switch (currNextStep?.type) {
                // Decisions cannot have any subroutines
                case RoutineStepType.Decision:
                    endFound = true;
                    break;

                // If current step is a RoutineList or Directory, check if there is another step in the list
                case RoutineStepType.RoutineList:
                case ProjectStepType.Directory:
                    if (currNextStep.steps.length > 0) {
                        result.push(1);
                        currNextStep = stepFromLocation(result, steps);
                    }
                    else endFound = true;
                    break;

                // If current step is a subroutine, mark as found. Subroutine data (if any) fetching 
                // should be handled elsewhere.
                case RoutineStepType.Subroutine:
                    endFound = true;
                    break;
            }
        }
        // Return result
        return result;
    }, [currStepLocation, currentStep?.type, steps]);


    //TODO
    const unsavedChanges = false;
    const subroutineComplete = true;

    /**
      * Navigate to the previous subroutine
      */
    const toPrevious = useCallback(() => {
        if (!previousStep) return;
        // Update current step
        setCurrStepLocation(previousStep);
    }, [previousStep, setCurrStepLocation]);

    const [logRunUpdate] = useLazyFetch<RunRoutineUpdateInput, RunRoutine>(endpointPutRunRoutine);
    const [logRunComplete] = useLazyFetch<RunRoutineCompleteInput, RunRoutine>(endpointPutRunRoutineComplete);
    /**
     * Navigate to the next subroutine, or complete the routine.
     * Also log progress, time elapsed, and other metrics
     */
    const toNext = useCallback(() => {
        // Find step data
        const currStep = stepFromLocation(currStepLocation, steps);
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
        if (nextStep) setCurrStepLocation(nextStep);
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
        if (nextStep) {
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
                    PubSub.get().publishCelebration();
                    removeSearchParams(setLocation, ["run", "step"]);
                    tryOnClose(onClose, setLocation);
                },
            });
        }
    }, [currStepLocation, currStepRunData, onClose, logRunComplete, logRunUpdate, nextStep, progress, run, runnableObject, setLocation, steps, testMode]);

    /**
     * End run after reaching an EndStep
     * TODO if run is a project, this is not the end. Just the end of a routine in the project
     */
    const reachedEndStep = useCallback((step: EndStep) => {
        // Check if end was successfully reached
        const success = step.wasSuccessful ?? true;
        // Don't actually do it if in test mode
        if (testMode || !run) {
            if (success) PubSub.get().publishCelebration();
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
                PubSub.get().publishCelebration();
                removeSearchParams(setLocation, ["run", "step"]);
                tryOnClose(onClose, setLocation);
            },
        });
    }, [testMode, run, runnableObject, logRunComplete, setLocation, onClose]);

    /**
     * Stores current progress, both for overall routine and the current subroutine
     */
    const saveProgress = useCallback(() => {
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
    const toFinishNotComplete = useCallback(() => {
        saveProgress();
        removeSearchParams(setLocation, ["run", "step"]);
        tryOnClose(onClose, setLocation);
    }, [onClose, saveProgress, setLocation]);

    /**
     * Navigate to selected decision
     */
    const toDecision = useCallback((step: RoutineStep | EndStep) => {
        // If end node, finish
        if (step.type === "End") {
            reachedEndStep(step);
            return;
        }
        // Find step number of node
        // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
        const locationArray = locationFromNodeId((step as unknown as SubroutineStep | RoutineListStep).nodeId!, steps);
        if (!locationArray) return;
        // Navigate to current step
        console.log("todecision", locationArray);
        setCurrStepLocation(locationArray);
    }, [reachedEndStep, setCurrStepLocation, steps]);

    /**
     * Displays either a subroutine view or decision view
     */
    const childView = useMemo(() => {
        console.log("calculating childview", currentStep, run);
        if (!currentStep) return null;
        switch (currentStep.type) {
            case RoutineStepType.Subroutine:
                return <SubroutineView
                    handleUserInputsUpdate={handleUserInputsUpdate}
                    handleSaveProgress={saveProgress}
                    // eslint-disable-next-line @typescript-eslint/no-empty-function
                    onClose={() => { }}
                    owner={run?.organization ?? run?.user} //TODO not right, but also unused right now
                    routineVersion={(currentStep as SubroutineStep).routineVersion}
                    run={run as RunRoutine}
                    loading={subroutinesLoading}
                    zIndex={zIndex}
                />;
            case ProjectStepType.Directory:
                return null; //TODO
            // return <DirectoryView
            // />;
            default:
                return <DecisionView
                    data={currentStep as DecisionStep}
                    handleDecisionSelect={toDecision}
                    routineList={stepFromLocation(locationFromRoutineVersionId(currentStep.parentRoutineVersionId, steps) ?? [], steps) as unknown as RoutineListStep}
                    // eslint-disable-next-line @typescript-eslint/no-empty-function
                    onClose={() => { }}
                    zIndex={zIndex}
                />;
        }
    }, [currentStep, handleUserInputsUpdate, run, saveProgress, steps, subroutinesLoading, toDecision, zIndex]);

    return (
        <Box sx={{ minHeight: "100vh" }}>
            <Box sx={{
                margin: "auto",
            }}>
                {/* Contains name bar and progress bar */}
                <Stack direction="column" spacing={0}>
                    {/* Top bar */}
                    <Box sx={{
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "space-between",
                        padding: "0.5rem",
                        width: "100%",
                        backgroundColor: palette.primary.dark,
                        color: palette.primary.contrastText,
                    }}>
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
                        <Stack direction="row" spacing={1} sx={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                        }}>
                            <Typography variant="h5" component="h2">{name}</Typography>
                            {(currentStepNumber >= 0 && stepsInCurrentNode >= 0) ?
                                <Typography variant="h5" component="h2">({currentStepNumber} of {stepsInCurrentNode})</Typography>
                                : null}
                            {/* Help icon */}
                            {instructions && <HelpButton markdown={instructions} zIndex={zIndex} />}
                        </Stack>
                        {/* Steps explorer drawer */}
                        <RunStepsDialog
                            currStep={currStepLocation}
                            handleLoadSubroutine={(id: string) => { getSubroutines({ ids: [id] }); }}
                            handleCurrStepLocationUpdate={setCurrStepLocation}
                            history={progress}
                            percentComplete={progressPercentage}
                            rootStep={steps}
                            zIndex={zIndex + 3}
                        />
                    </Box>
                    {/* Progress bar */}
                    <LinearProgress color="secondary" variant="determinate" value={completedComplexity / ((run as RunRoutine)?.routineVersion?.complexity ?? (run as RunProject)?.projectVersion?.complexity ?? 1) * 100} sx={{ height: "15px" }} />
                </Stack>
                {/* Main content. For now, either looks like view of a basic routine, or options to select an edge */}
                <Box sx={{
                    background: palette.mode === "light" ? "#c2cadd" : palette.background.default,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    margin: "auto",
                    marginBottom: "72px + env(safe-area-inset-bottom)",
                    overflowY: "auto",
                    minHeight: "100vh",
                }}>
                    {childView}
                </Box>
                {/* Action bar */}
                <Box sx={{
                    position: "fixed",
                    bottom: "0",
                    paddingBottom: "env(safe-area-inset-bottom)",
                    // safe-area-inset-bottom is the iOS navigation bar
                    height: "calc(56px + env(safe-area-inset-bottom))",
                    width: "-webkit-fill-available",
                    zIndex: 4,
                    background: palette.primary.dark,
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                }}>
                    <Grid container spacing={2} sx={{
                        width: "min(100%, 700px)",
                        margin: 0,
                    }}>
                        {/* There are only ever 1 or 2 options shown. 
                        In either case, we want the buttons to be placed as 
                        if there are always 2 */}
                        <Grid item xs={6} p={1}>
                            {previousStep && <Button
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
                            {nextStep && (<Button
                                fullWidth
                                startIcon={<ArrowRightIcon />}
                                onClick={toNext} // NOTE: changes are saved on next click
                                disabled={!subroutineComplete}
                                variant="contained"
                            >
                                {t("Next")}
                            </Button>)}
                            {!nextStep && currentStep?.type !== RoutineStepType.Decision && (<Button
                                fullWidth
                                startIcon={<SuccessIcon />}
                                onClick={toNext}
                                variant="contained"
                            >
                                {t("Complete")}
                            </Button>)}
                        </Grid>
                    </Grid>
                </Box>
                <Box p={2} sx={{
                    background: palette.primary.dark,
                    position: "fixed",
                    bottom: 0,
                    width: "100vw",
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                }}>
                </Box>
            </Box>
        </Box>
    );
};
