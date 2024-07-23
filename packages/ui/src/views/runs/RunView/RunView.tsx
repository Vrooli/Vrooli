import { base36ToUuid, endpointGetProjectVersionDirectories, endpointGetRoutineVersions, endpointGetRunProject, endpointGetRunRoutine, endpointPutRunRoutine, endpointPutRunRoutineComplete, FindByIdInput, noop, parseSearchParams, ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult, RoutineVersionSearchInput, RoutineVersionSearchResult, RunProject, RunProjectStep, RunRoutine, RunRoutineCompleteInput, RunRoutineInput, RunRoutineStep, RunRoutineStepStatus, RunRoutineUpdateInput, SECONDS_1_MS, uuid, uuidValidate } from "@local/shared";
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
import { RunStepType } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { tryOnClose } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { addSubdirectoriesToStep, addSubroutinesToStep, DecisionStep, detectSubstepLoad, DetectSubstepLoadResult, EndStep, getNextLocation, getPreviousLocation, getRunPercentComplete, getStepComplexity, locationArraysMatch, MultiRoutineStep, RootStep, RoutineListStep, runInputsUpdate, runnableObjectToStep, siblingsAtLocation, SingleRoutineStep, stepFromLocation } from "utils/runUtils";
import { DecisionView } from "../DecisionView/DecisionView";
import { SubroutineView } from "../SubroutineView/SubroutineView";
import { RunViewProps } from "../types";

const ROOT_LOCATION = [1];

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
            startingStepLocation:
                Array.isArray(params.step) &&
                    params.step.length > 0 &&
                    params.step.every(n => typeof n === "number")
                    ? params.step as number[]
                    : [...ROOT_LOCATION],
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

    const [currentLocation, setCurrentLocation] = useState<number[]>([...startingStepLocation]);
    const handleLocationUpdate = useCallback(function handleLocationUpdateCallback(step: number[]) {
        setCurrentLocation(step);
    }, []);
    useEffect(function updateLocationInUrlEffect() {
        addSearchParams(setLocation, { step: currentLocation });
    }, [currentLocation, setLocation]);

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
        setRootStep(runnableObjectToStep(runnableObject, [...ROOT_LOCATION], languages));
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

    const stepsInCurrentNode = useMemo(function stepsInCurrentNodeMemo() {
        if (!currentLocation || !rootStep) return 1;
        return siblingsAtLocation(currentLocation, rootStep);
    }, [currentLocation, rootStep]);

    const currentStepRunData = useMemo<RunProjectStep | RunRoutineStep | undefined>(function currentStepRunDataMemo() {
        const runStep = run?.steps?.find((step) => locationArraysMatch(step.step, currentLocation));
        return runStep;
    }, [run?.steps, currentLocation]);

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
        if (!currentStepRunData) return;
        // Start tracking time
        intervalRef.current = setInterval(() => { timeElapsed.current += 1; }, SECONDS_1_MS);
        // Reset context switches
        contextSwitches.current = 0;
        return () => {
            if (intervalRef.current) clearInterval(intervalRef.current);
        };
    }, [currentStepRunData]);

    /** On tab change, add to contextSwitches */
    useEffect(function contextSwitchesMemo() {
        function handleTabChange() {
            if (currentStepRunData) {
                contextSwitches.current += 1;
            }
        }
        window.addEventListener("focus", handleTabChange);
        return () => {
            window.removeEventListener("focus", handleTabChange);
        };
    }, [currentStepRunData]);

    const progressPercentage = useMemo(function progressPercentageMemo() {
        // Measured in complexity / total complexity * 100
        return getRunPercentComplete(completedComplexity, (run as RunProject)?.projectVersion?.complexity ?? (run as RunRoutine)?.routineVersion?.complexity);
    }, [completedComplexity, run]);

    // Query subroutines and subdirectories, since the whole run may not be able to load at once
    const [getSubroutines, { data: queriedSubroutines, loading: subroutinesLoading }] = useLazyFetch<RoutineVersionSearchInput, RoutineVersionSearchResult>(endpointGetRoutineVersions);
    const [getDirectories, { data: queriedSubdirectories, loading: directoriesLoading }] = useLazyFetch<ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult>(endpointGetProjectVersionDirectories);

    // Store queried IDs to make sure we don't query the same thing multiple times
    const queriedIdsRef = useRef<{ [key: string]: boolean }>({});
    useEffect(function clearQueriedIdsEffect() {
        queriedIdsRef.current = {};
    }, [runId]);

    // Store result of `detectSubstepLoad` to determine loading behavior
    const substepLoadResult = useRef<DetectSubstepLoadResult | null>(null);

    const loadSubsteps = useCallback(function loadSubstepsCallback() {
        // Helper function to query without duplicates
        function safeQuery(ids: string[], fetch: ((input: ProjectVersionDirectorySearchInput) => unknown) | ((input: RoutineVersionSearchInput) => unknown)) {
            const newIds = ids.filter(id => !queriedIdsRef.current[id]);
            newIds.forEach(id => {
                queriedIdsRef.current[id] = true;
            });
            if (newIds.length > 0) {
                fetch({ ids: newIds });
            }
        }
        const result = detectSubstepLoad(
            currentLocation,
            rootStep,
            (input) => safeQuery(input.ids || [], getDirectories),
            (input) => safeQuery(input.ids || [], getSubroutines),
        );
        substepLoadResult.current = result;
    }, [currentLocation, getDirectories, getSubroutines, rootStep]);
    // Load substeps on initial render
    useEffect(function loadSubstepsEffect() {
        loadSubsteps();
    }, [loadSubsteps]);

    const currentStep = useMemo(function currentStepMemo() {
        if (!rootStep) return null;
        return stepFromLocation(currentLocation, rootStep);
    }, [currentLocation, rootStep]);

    useEffect(function addQueriedSubroutinesToRootEffect() {
        if (!queriedSubroutines) return;
        const subroutines = queriedSubroutines.edges.map(e => e.node);
        setRootStep(root => root ? addSubroutinesToStep(subroutines, root, languages) : null);
        if (substepLoadResult.current && substepLoadResult.current.needsFurtherQuerying) {
            loadSubsteps();
        }
    }, [languages, loadSubsteps, queriedSubroutines]);

    useEffect(function addQueriedSubdirectoriesToRootEffect() {
        if (!queriedSubdirectories) return;
        const subdirectories = queriedSubdirectories.edges.map(e => e.node);
        setRootStep(root => root ? addSubdirectoriesToStep(subdirectories, root, languages) : null);
        if (substepLoadResult.current && substepLoadResult.current.needsFurtherQuerying) {
            loadSubsteps();
        }
    }, [languages, loadSubsteps, queriedSubdirectories]);

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
        handleLocationUpdate(previousLocation);
    }, [previousLocation, handleLocationUpdate]);

    const [logRunUpdate] = useLazyFetch<RunRoutineUpdateInput, RunRoutine>(endpointPutRunRoutine);
    const [logRunComplete] = useLazyFetch<RunRoutineCompleteInput, RunRoutine>(endpointPutRunRoutineComplete);
    /**
     * Navigate to the next subroutine, or complete the routine.
     * Also log progress, time elapsed, and other metrics
     */
    const toNext = useCallback(function toNextCallback() {
        if (!rootStep) return;
        // Find step data
        const currStep = stepFromLocation(currentLocation, rootStep);
        // Calculate new progress and percent complete
        const newProgress = Array.isArray(progress) ? [...progress] : [];
        const newlyCompletedComplexity: number = (currStep ? getStepComplexity(currStep) : 0);
        const alreadyComplete = Boolean(newProgress.find(p => locationArraysMatch(p, currentLocation)));
        // If step was not already completed, update progress
        if (!alreadyComplete) {
            newProgress.push(currentLocation);
            setProgress(newProgress);
            setCompletedComplexity(c => c + newlyCompletedComplexity);
        }
        // Update current step
        if (nextLocation) {
            handleLocationUpdate(nextLocation);
        }
        // If in test mode return
        if (testMode || !run) return;
        // Now we can calculate data for the logs
        // Find parent RoutineList step, so we can get the nodeId 
        const currParentListStep: RoutineListStep = {} as any;//TODO stepFromLocation(currentLocation.slice(0, currentLocation.length - 1), steps) as RoutineListStep;
        // Current step will be updated if it already exists in logged data, or created if not
        const stepsUpdate = currentStepRunData ? [{
            id: currentStepRunData.id,
            status: RunRoutineStepStatus.Completed,
            timeElapsed: (currentStepRunData.timeElapsed ?? 0) + timeElapsed.current,
            contextSwitches: currentStepRunData.contextSwitches + contextSwitches.current,
        }] : undefined;
        const stepsCreate = currentStepRunData ? undefined : [{
            id: uuid(),
            order: newProgress.length,
            name: currStep?.name ?? "",
            nodeId: currParentListStep.nodeId,
            subroutineId: currParentListStep.parentRoutineVersionId,
            step: currentLocation,
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
            const currNodeId = currentStepRunData?.node?.id;
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
    }, [rootStep, currentLocation, progress, nextLocation, testMode, run, currentStepRunData, handleLocationUpdate, logRunUpdate, logRunComplete, runnableObject, setLocation, onClose]);

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
        const stepUpdate = currentStepRunData ? {
            id: currentStepRunData.id,
            timeElapsed: (currentStepRunData.timeElapsed ?? 0) + timeElapsed.current,
            contextSwitches: currentStepRunData.contextSwitches + contextSwitches.current,
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
    }, [currentStepRunData, logRunUpdate, run, testMode]);

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
        handleLocationUpdate(step.location);
    }, [reachedEndStep, handleLocationUpdate]);

    const handleLoadSubroutine = useCallback(function handleLoadSubroutineCallback(id: string) {
        getSubroutines({ ids: [id] });
    }, [getSubroutines]);

    /**
     * Displays either a subroutine view or decision view
     */
    const childView = useMemo(function childViewMemo() {
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
                                {stepsInCurrentNode >= 0 ?
                                    <Typography variant="h5" component="h2">({currentLocation[currentLocation.length - 1] ?? 1} of {stepsInCurrentNode})</Typography>
                                    : null}
                                {/* Help icon */}
                                {instructions && <HelpButton markdown={instructions} />}
                            </TitleStepsStack>
                            {/* Steps explorer drawer */}
                            {rootStep !== null && rootStep.__type !== RunStepType.SingleRoutine ? <RunStepsDialog
                                currStep={currentLocation}
                                handleLoadSubroutine={handleLoadSubroutine}
                                handleCurrStepLocationUpdate={handleLocationUpdate}
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
