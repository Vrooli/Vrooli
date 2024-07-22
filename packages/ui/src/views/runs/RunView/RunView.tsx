import { base36ToUuid, endpointGetProjectVersionDirectories, endpointGetRoutineVersions, endpointGetRunProject, endpointGetRunRoutine, endpointPutRunRoutine, endpointPutRunRoutineComplete, FindByIdInput, isOfType, NodeType, noop, parseSearchParams, ProjectVersionDirectory, ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult, RoutineType, RoutineVersion, RoutineVersionSearchInput, RoutineVersionSearchResult, RunProject, RunProjectStep, RunRoutine, RunRoutineCompleteInput, RunRoutineInput, RunRoutineStep, RunRoutineStepStatus, RunRoutineUpdateInput, SECONDS_1_MS, uuid, uuidValidate } from "@local/shared";
import { Box, Button, Grid, IconButton, LinearProgress, Stack, styled, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RunStepsDialog } from "components/dialogs/RunStepsDialog/RunStepsDialog";
import { SessionContext } from "contexts/SessionContext";
import { useLazyFetch } from "hooks/useLazyFetch";
import { ArrowLeftIcon, ArrowRightIcon, CloseIcon, SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, removeSearchParams, useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { DecisionStep, DirectoryStep, EndStep, MultiRoutineStep, RootStep, RoutineListStep, RunStep, SingleRoutineStep, StartStep } from "types";
import { RunStepType } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { tryOnClose } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { findStep, getNextLocation, getPreviousLocation, getRunPercentComplete, getStepComplexity, insertStep, locationArraysMatch, runInputsUpdate, siblingsAtLocation, sortStepsAndAddDecisions, stepFromLocation, stepNeedsQuerying, UnsortedSteps } from "utils/runUtils";
import { DecisionView } from "../DecisionView/DecisionView";
import { SubroutineView } from "../SubroutineView/SubroutineView";
import { RunnableProjectVersion, RunnableRoutineVersion, RunViewProps } from "../types";

const ROOT_LOCATION = [1];

/**
 * Converts a single-step routine into a Step object
 * @param routineVersion The routineVersion being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function singleRoutineToStep(
    routineVersion: RunnableRoutineVersion,
    location: number[],
    languages: string[],
): SingleRoutineStep | null {
    return {
        __type: RunStepType.SingleRoutine,
        description: getTranslation(routineVersion, languages, true).description || null,
        location,
        name: getTranslation(routineVersion, languages, true).name || "Untitled",
        routineVersion,
    };
}

/**
 * Converts a multi-step routine into a Step object
 * @param routineVersion The routineVersion being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function multiRoutineToStep(
    routineVersion: RunnableRoutineVersion,
    location: number[],
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
                    location: [...location, 1], // Sort function will correct this
                    name,
                    nodeId,
                    wasSuccessful: node.end?.wasSuccessful ?? false,
                } as EndStep;
            case NodeType.RoutineList:
                return {
                    __type: RunStepType.RoutineList,
                    description,
                    location: [...location, 1], // Sort function will correct this
                    name,
                    nodeId,
                    isOrdered: node.routineList?.isOrdered ?? false,
                    parentRoutineVersionId: routineVersion.id,
                } as RoutineListStep;
            case NodeType.Start:
                return {
                    __type: RunStepType.Start,
                    description,
                    location: [...location, 1], // Sort function will correct this
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
        location,
        name: getTranslation(routineVersion, languages, true).name || "Untitled",
        nodeLinks: routineVersion.nodeLinks || [],
        nodes: sorted,
        routineVersionId: routineVersion.id,
    };
}

/**
 * Converts a project into a Step object
 * @param projectVersion The projectVersion being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function projectToStep(
    projectVersion: RunnableProjectVersion,
    location: number[],
    languages: string[],
): DirectoryStep | null {
    // Projects are represented as root directories
    const steps: DirectoryStep[] = [];
    for (let i = 0; i < projectVersion.directories.length; i++) {
        const directory = projectVersion.directories[i];
        const directoryStep = directoryToStep(directory, location, languages);
        if (directoryStep) {
            steps.push(directoryStep);
        }
    }
    //TODO project version does not have order field, so can't order directories. Maybe change this in future
    return {
        __type: RunStepType.Directory,
        description: getTranslation(projectVersion, languages, true).description || null,
        directoryId: null,
        hasBeenQueried: true,
        isOrdered: false,
        isRoot: true,
        location,
        name: getTranslation(projectVersion, languages, true).name || "Untitled",
        projectVersionId: projectVersion.id,
        steps,
    };
}

/**
 * Converts a directory into a Step object
 * @param projectVersionDirectory The projectVersionDirectory being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function directoryToStep(
    projectVersionDirectory: ProjectVersionDirectory,
    location: number[],
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
        location,
        name: getTranslation(projectVersionDirectory, languages, true).name || "Untitled",
        projectVersionId: projectVersionDirectory.projectVersion?.id ?? "",
        steps: [],
    };
}

/**
 * Converts a runnable object into a Step object
 * @param runnableObject The projectVersion or routineVersion being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
function runnableObjectToStep(
    runnableObject: RunnableProjectVersion | RunnableRoutineVersion | null | undefined,
    location: number[],
    languages: string[],
): RootStep | null {
    if (isOfType(runnableObject, "RoutineVersion")) {
        if (runnableObject.routineType === RoutineType.MultiStep) {
            return multiRoutineToStep(runnableObject, location, languages);
        } else {
            return singleRoutineToStep(runnableObject, location, languages);
        }
    } else if (isOfType(runnableObject, "ProjectVersion")) {
        return projectToStep(runnableObject, location, languages);
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
        // Find the location we should insert the routine at
        const location = findStep(
            rootStep,
            // We use single-step routines as a placeholder before loading the full routine
            (step) => step.__type === RunStepType.SingleRoutine && (step as SingleRoutineStep).routineVersion?.id === routineVersion.id,
        )?.location;
        if (!location) {
            console.error("Could not find location to insert routine", routineVersion);
            continue;
        }
        const subroutineStep = multiRoutineToStep(routineVersion, location, languages);
        if (!subroutineStep) {
            console.error("Could not convert routine to step", routineVersion);
            continue;
        }
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
        // Find location to insert the directory at
        const location = findStep(
            rootStep,
            (step) => step.__type === RunStepType.Directory && (step as DirectoryStep).directoryId === directory.id,
        )?.location;
        if (!location) {
            console.error("Could not find location to insert directory", directory);
            continue;
        }
        const projectStep = directoryToStep(directory, location, languages);
        if (!projectStep) {
            console.error("Could not convert directory to step", directory);
            continue;
        }
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

const dialogStyle = {
    paper: {
        width: "-webkit-fill-available",
        maxWidth: "100vw",
        height: "-webkit-fill-available",
        maxHeight: "100vh",
        borderRadius: 0,
        margin: 0,
    },
} as const;
const actionBarGridStyle = {
    width: "min(100%, 700px)",
    margin: 0,
} as const;
const progressBarStyle = { height: "15px" } as const;

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
    const { runId, startingStepLocation, testMode } = useMemo(function parseUrlMemo() {
        const params = parseSearchParams();
        return {
            runId: (typeof params.run === "string" && uuidValidate(base36ToUuid(params.run))) ? base36ToUuid(params.run) : undefined,
            startingStepLocation: Array.isArray(params.step) && params.step.every(n => typeof n === "number") ? params.step as number[] : [],
            testMode: params.run === "test",
        };
    }, []);

    const lastQueriedRunId = useRef<string | null>(null);
    const [getRun, { data: runData, errors: fetchedErrors, loading: isRunLoading }] = useLazyFetch<FindByIdInput, RunProject | RunRoutine>(runnableObject?.__typename === "ProjectVersion" ? endpointGetRunProject : endpointGetRunRoutine);
    useEffect(function fetchRunEffect() {
        if (runId && runId !== lastQueriedRunId.current) {
            lastQueriedRunId.current = runId;
            getRun({ id: runId });
        }
    }, [getRun, runId]);
    const [run, setRun] = useState<RunProject | RunRoutine | undefined>(undefined);
    useEffect(function updateRunDataEffect() {
        if (runData) setRun(runData);
    }, [runData]);

    const [currStepLocation, setCurrStepLocation] = useState<number[]>(startingStepLocation);
    useEffect(function updateLocationInUrlEffect() {
        addSearchParams(setLocation, { step: currStepLocation });
    }, [currStepLocation, setLocation]);

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
        setRootStep(runnableObjectToStep(runnableObject, ROOT_LOCATION, languages));
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

    const currentStepNumber = useMemo(function currentStepNumberMemo() {
        return currStepLocation.length === 0 ? 1 : Number(currStepLocation[currStepLocation.length - 1]);
    }, [currStepLocation]);

    const stepsInCurrentNode = useMemo(function stepsInCurrentNodeMemo() {
        if (!currStepLocation || !rootStep) return 1;
        return siblingsAtLocation(currStepLocation, rootStep);
    }, [currStepLocation, rootStep]);

    const currStepRunData = useMemo<RunProjectStep | RunRoutineStep | undefined>(function currStepRunDataMemo() {
        const runStep = run?.steps?.find((step) => locationArraysMatch(step.step, currStepLocation));
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

    const intervalRef = useRef<NodeJS.Timeout | null>(null);
    const timeElapsed = useRef<number>(0);
    const contextSwitches = useRef<number>(0);
    useEffect(function trackUserBehaviorMemo() {
        if (!currStepRunData) return;
        // Start tracking time
        intervalRef.current = setInterval(() => { timeElapsed.current += 1; }, SECONDS_1_MS);
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

    const progressPercentage = useMemo(function progressPercentageMemo() {
        // Measured in complexity / total complexity * 100
        return getRunPercentComplete(completedComplexity, (run as RunProject)?.projectVersion?.complexity ?? (run as RunRoutine)?.routineVersion?.complexity);
    }, [completedComplexity, run]);

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
        if (!rootStep || currStepLocation.length === 0) {
            setCurrStepLocation(ROOT_LOCATION);
            return;
        }
        // Current step is the last step in steps list
        const currStep = stepFromLocation(currStepLocation, rootStep);
        // If current step was not found
        if (!currStep) {
            //TODO
            // // Check if this is because the top level step list is empty
            // if (currStepLocation.length === 1 && rootStep?.steps.length === 0) {
            //     // Navigate to the next step in the parent list or directory
            //     navigateToNextStepInParent();
            // } else if (currStepLocation.length > 1) {
            //     // Navigate back to the parent step
            //     navigateBackToParentStep();
            // }
            return;
        }
        // If current step is a list or directory, then redirect to first step in list or directory
        if (currStep.__type === RunStepType.RoutineList || currStep.__type === RunStepType.Directory) {
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
        if (currStep.__type === RunStepType.Directory && stepNeedsQuerying(currStep)) {
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
    }, [currStepLocation, getSubroutines, getDirectories, setCurrStepLocation, rootStep, navigateToNextStepInParent, navigateBackToParentStep]);


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

    const { instructions, name } = useMemo(function displayInfoMemo() {
        if (!currentStep) {
            return {
                instructions: "",
                name: "",
            };
        }
        const languages = getUserLanguages(session);
        return {
            instructions: currentStep.__type === RunStepType.SingleRoutine ? getTranslation((currentStep as SingleRoutineStep).routineVersion, languages, true).instructions : "",
            name: currentStep.name,
        };
    }, [currentStep, session]);

    const previousLocation = useMemo<number[] | null>(function previousLocationMemo() {
        if (!currentStep) return null;
        return getPreviousLocation(currentStep.location, rootStep);
    }, [currentStep, rootStep]);

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
        if (!rootStep) return;
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
        const currParentListStep: RoutineListStep = {} as any;//TODO stepFromLocation(currStepLocation.slice(0, currStepLocation.length - 1), steps) as RoutineListStep;
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
            subroutineId: currParentListStep.parentRoutineVersionId,
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
            sxs={dialogStyle}
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
                            {rootStep !== null && rootStep.__type !== RunStepType.SingleRoutine ? <RunStepsDialog
                                currStep={currStepLocation}
                                handleLoadSubroutine={handleLoadSubroutine}
                                handleCurrStepLocationUpdate={setCurrStepLocation}
                                history={progress}
                                percentComplete={progressPercentage}
                                rootStep={rootStep}
                            /> : <Box width="48px"></Box>}
                        </TopBar>
                        {/* Progress bar */}
                        {rootStep !== null && rootStep.__type !== RunStepType.SingleRoutine && <LinearProgress
                            color="secondary"
                            variant="determinate"
                            value={completedComplexity / ((run as RunRoutine)?.routineVersion?.complexity ?? (run as RunProject)?.projectVersion?.complexity ?? 1) * 100}
                            sx={progressBarStyle}
                        />}
                    </Stack>
                    <ContentBox>
                        {childView}
                    </ContentBox>
                    <ActionBar>
                        <Grid container spacing={2} sx={actionBarGridStyle}>
                            <Grid item xs={6} p={1}>
                                <Button
                                    disabled={!previousLocation || unsavedChanges || isRunLoading}
                                    fullWidth
                                    startIcon={<ArrowLeftIcon />}
                                    onClick={toPrevious}
                                    variant="outlined"
                                >
                                    {t("Previous")}
                                </Button>
                            </Grid>
                            <Grid item xs={6} p={1}>
                                <Button
                                    disabled={unsavedChanges || isRunLoading}
                                    fullWidth
                                    startIcon={!nextLocation && currentStep?.__type !== RunStepType.Decision ? <SuccessIcon /> : <ArrowRightIcon />}
                                    onClick={toNext} // NOTE: changes are saved on next click
                                    variant="contained"
                                >
                                    {!nextLocation && currentStep?.__type !== RunStepType.Decision ? t("Complete") : t("Next")}
                                </Button>
                            </Grid>
                        </Grid>
                    </ActionBar>
                </Box>
            </Box>
        </MaybeLargeDialog>
    );
}
