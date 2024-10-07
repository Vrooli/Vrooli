import { DecisionStep, DetectSubstepLoadResult, FindByIdInput, ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult, RootStep, RoutineVersionSearchInput, RoutineVersionSearchResult, RunProject, RunProjectStep, RunProjectUpdateInput, RunRoutine, RunRoutineStep, RunRoutineUpdateInput, RunStatus, RunStep, RunStepType, SingleRoutineStep, addSubdirectoriesToStep, addSubroutinesToStep, base36ToUuid, detectSubstepLoad, endpointGetProjectVersionDirectories, endpointGetRoutineVersions, endpointGetRunProject, endpointGetRunRoutine, endpointPutRunProject, endpointPutRunRoutine, generateRoutineInitialValues, getNextLocation, getPreviousLocation, getRunPercentComplete, getStepComplexity, getTranslation, locationArraysMatch, noop, parseSchemaInput, parseSchemaOutput, parseSearchParams, runnableObjectToStep, saveRunProgress, siblingsAtLocation, stepFromLocation, uuidValidate } from "@local/shared";
import { Box, Button, Grid, IconButton, LinearProgress, Stack, Typography, styled } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { AutoSaveIndicator } from "components/AutoSaveIndicator/AutoSaveIndicator";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RunStepsDialog } from "components/dialogs/RunStepsDialog/RunStepsDialog";
import { SessionContext } from "contexts";
import { FormikProps } from "formik";
import { useAutoSave } from "hooks/useAutoSave";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSocketRun } from "hooks/useSocketRun";
import { ArrowLeftIcon, ArrowRightIcon, CloseIcon, SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, removeSearchParams, useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { getUserLanguages } from "utils/display/translationTools";
import { tryOnClose } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { DecisionView } from "../DecisionView/DecisionView";
import { SubroutineView } from "../SubroutineView/SubroutineView";
import { RunViewProps } from "../types";

interface StepRunData {
    contextSwitches?: number;
    elapsedTime?: number;
    // Add other properties of currentStepRunData if needed
}

export function useStepMetrics(
    currentLocation: number[], // Only provided to reinitialize metrics when location changes
    currentStepRunData: StepRunData | null | undefined,
) {
    const contextSwitches = useRef<number>(0);
    const elapsedTimeRef = useRef<number>(0);
    const lastFocusTime = useRef<number>(0);
    const isTabFocused = useRef<boolean>(false);

    const updateElapsedTime = useCallback(() => {
        if (isTabFocused.current) {
            const now = Date.now();
            elapsedTimeRef.current += now - lastFocusTime.current;
            lastFocusTime.current = now;
        }
    }, []);

    const getElapsedTime = useCallback(() => {
        updateElapsedTime();
        return elapsedTimeRef.current;
    }, [updateElapsedTime]);

    useEffect(function initializeStepMetricsEffect() {
        contextSwitches.current = currentStepRunData?.contextSwitches ?? 0;
        elapsedTimeRef.current = currentStepRunData?.elapsedTime ?? 0;
        lastFocusTime.current = Date.now();
        isTabFocused.current = document.hasFocus();
    }, [currentLocation, currentStepRunData]);

    useEffect(() => {
        function handleVisibilityChange() {
            if (document.hidden) {
                updateElapsedTime();
                isTabFocused.current = false;
            } else {
                lastFocusTime.current = Date.now();
                isTabFocused.current = true;
                contextSwitches.current += 1;  // Increment context switches when tab gains focus
            }
        }

        function handleBeforeUnload() {
            updateElapsedTime();
        }

        document.addEventListener("visibilitychange", handleVisibilityChange);
        window.addEventListener("beforeunload", handleBeforeUnload);

        if (document.hasFocus()) {
            isTabFocused.current = true;
            lastFocusTime.current = Date.now();
        }

        return () => {
            document.removeEventListener("visibilitychange", handleVisibilityChange);
            window.removeEventListener("beforeunload", handleBeforeUnload);
        };
    }, [updateElapsedTime]);

    return {
        getElapsedTime,
        getContextSwitches: useCallback(() => contextSwitches.current, []),
    };
}

const ROOT_LOCATION = [1];
const PERCENTS = 100;

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
    background: theme.palette.background.default,
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

//TODO: implement `handleGenerateOutputs`, and soft disable (confirmation dialog) next button when there are missing required fields
//TODO: Need socket to listen to generate output results
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
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [isOpen]);

    const lastQueriedRunId = useRef<string | null>(null);
    const [getRun, { data: runData, loading: isRunLoading }] = useLazyFetch<FindByIdInput, RunProject | RunRoutine>(runnableObject?.__typename === "ProjectVersion" ? endpointGetRunProject : endpointGetRunRoutine);
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
        if (isOpen && currentLocation.length > 0) {
            addSearchParams(setLocation, { step: currentLocation });
        } else {
            removeSearchParams(setLocation, ["step"]);
        }
    }, [currentLocation, isOpen, setLocation]);

    const languages = useMemo(() => getUserLanguages(session), [session]);

    /**
     * Stores all queried steps, under one root step that represents the entire project/routine.
     * This can be traversed to find step data at a given location.
     */
    const [rootStep, setRootStep] = useState<RootStep | null>(null);
    useEffect(function convertRunnableObjectToStepTreeEffect() {
        setRootStep(runnableObjectToStep(runnableObject, [...ROOT_LOCATION], languages, console));
    }, [languages, runnableObject]);

    /**
     * The amount of routine completed so far, measured in complexity. 
     * Used with the routine's total complexity to calculate percent complete
     */
    const [completedComplexity, setCompletedComplexity] = useState(0);
    /**
     * Ordered list of location arrays for every completed step in the routine.
     */
    const [progress, setProgress] = useState<number[][]>([]);
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
        return siblingsAtLocation(currentLocation, rootStep, console);
    }, [currentLocation, rootStep]);

    const currentStepRunData = useMemo<RunProjectStep | RunRoutineStep | undefined>(function currentStepRunDataMemo() {
        const runStep = run?.steps?.find((step) => locationArraysMatch(step.step, currentLocation));
        return runStep;
    }, [run?.steps, currentLocation]);

    /** Stores user inputs and generated outputs */
    const formikRef = useRef<FormikProps<object>>(null);
    useEffect(function setInputsOnRunLoad() {
        if (!runnableObject || runnableObject.__typename !== "RoutineVersion") return;
        const values = generateRoutineInitialValues({
            configFormInput: parseSchemaInput(runnableObject.configFormInput, runnableObject.routineType, console),
            configFormOutput: parseSchemaOutput(runnableObject.configFormOutput, runnableObject.routineType, console),
            logger: console,
            runInputs: (run as RunRoutine)?.inputs,
            runOutputs: (run as RunRoutine)?.outputs,
        });

        // Formik doesn't seem to like setting the values normally
        formikRef.current?.setValues(values);
        setTimeout(() => { formikRef.current?.resetForm({ values }); }, 100);
    }, [run, currentLocation, runnableObject]);

    const { getContextSwitches, getElapsedTime } = useStepMetrics(currentLocation, currentStepRunData);

    const progressPercentage = useMemo(function progressPercentageMemo() {
        // Measured in complexity / total complexity * 100
        return getRunPercentComplete(completedComplexity, (run as RunProject)?.projectVersion?.complexity ?? (run as RunRoutine)?.routineVersion?.complexity);
    }, [completedComplexity, run]);

    // Query subdirectories and subroutines, since the whole run may not be able to load at once
    const [getDirectories, { data: queriedSubdirectories, loading: isDirectoriesLoading }] = useLazyFetch<ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult>(endpointGetProjectVersionDirectories);
    const [getSubroutines, { data: queriedSubroutines, loading: isSubroutinesLoading }] = useLazyFetch<RoutineVersionSearchInput, RoutineVersionSearchResult>(endpointGetRoutineVersions);

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
            console,
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
        setRootStep(root => root ? addSubroutinesToStep(subroutines, root, languages, console) : null);
        if (substepLoadResult.current && substepLoadResult.current.needsFurtherQuerying) {
            loadSubsteps();
        }
    }, [languages, loadSubsteps, queriedSubroutines]);

    useEffect(function addQueriedSubdirectoriesToRootEffect() {
        if (!queriedSubdirectories) return;
        const subdirectories = queriedSubdirectories.edges.map(e => e.node);
        setRootStep(root => root ? addSubdirectoriesToStep(subdirectories, root, languages, console) : null);
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

    const [updateRunRoutine] = useLazyFetch<RunRoutineUpdateInput, RunRoutine>(endpointPutRunRoutine);
    const [updateRunProject] = useLazyFetch<RunProjectUpdateInput, RunProject>(endpointPutRunProject);

    /**
     * Navigate to a step
     * @param nextStep The step to navigate to, or null if the run is completed
     */
    const toStep = useCallback(function toStepCallback(nextStep: RunStep) {
        if (!rootStep) return;

        // Find step data
        const currentStep = stepFromLocation(currentLocation, rootStep);
        if (!currentStep) return;

        const isRunCompleted = !nextStep || nextStep.__type === RunStepType.End;

        function onSuccess(data: RunProject | RunRoutine) {
            if (isRunCompleted) {
                if (data.status === RunStatus.Completed) PubSub.get().publish("celebration");
                removeSearchParams(setLocation, ["run", "step"]);
                tryOnClose(onClose, setLocation);
            } else {
                setRun(data);
            }
        }

        const isStepAlreadyCompleted = currentStepRunData?.status === "Completed";
        const newProgress = [...(progress ?? [])];
        // If step was not already marked as completed, update progress and complexity
        if (!isStepAlreadyCompleted && currentStep) {
            if (!newProgress.some((p) => locationArraysMatch(p, currentLocation))) {
                newProgress.push(currentLocation);
            }
            setProgress(newProgress);
            setCompletedComplexity(c => c + getStepComplexity(currentStep, console));
        }

        // Update run data
        if (!testMode && run) {
            saveRunProgress({
                contextSwitches: Math.max(getContextSwitches(), currentStepRunData?.contextSwitches ?? 0),
                currentStep,
                currentStepOrder: (newProgress.findIndex((p) => locationArraysMatch(p, currentLocation)) ?? newProgress.length) + 1,
                currentStepRunData,
                formData: formikRef.current?.values ?? {},
                handleRunProjectUpdate: async function updateRun(inputs) {
                    fetchLazyWrapper<RunProjectUpdateInput, RunProject>({
                        fetch: updateRunProject,
                        inputs,
                        onSuccess,
                    });
                },
                handleRunRoutineUpdate: async function updateRun(inputs) {
                    fetchLazyWrapper<RunRoutineUpdateInput, RunRoutine>({
                        fetch: updateRunRoutine,
                        inputs,
                        onSuccess,
                    });
                },
                logger: console,
                isStepCompleted: true, //TODO shouldn't always be true
                isRunCompleted,
                run,
                runnableObject,
                timeElapsed: Math.max(getElapsedTime(), currentStepRunData?.timeElapsed ?? 0),
            });
        }
    }, [currentLocation, currentStepRunData, getContextSwitches, getElapsedTime, onClose, progress, rootStep, run, runnableObject, setLocation, testMode, updateRunProject, updateRunRoutine]);

    const toNext = useCallback(function toNextCallback() {
        if (!rootStep || !nextLocation) return;
        const nextStep = stepFromLocation(nextLocation, rootStep);
        if (!nextStep) return;
        toStep(nextStep);
    }, [nextLocation, rootStep, toStep]);

    const toPrevious = useCallback(function toPreviousCallback() {
        if (!rootStep || !previousLocation) return;
        const previousStep = stepFromLocation(previousLocation, rootStep);
        if (!previousStep) return;
        toStep(previousStep);
    }, [previousLocation, rootStep, toStep]);

    const handleAutoSave = useCallback(function handleAutoSaveCallback() {
        if (testMode || !run || !rootStep) return;

        // Find step data
        const currentStep = stepFromLocation(currentLocation, rootStep);
        if (!currentStep) return;

        function onSuccess(data: RunProject | RunRoutine) {
            setRun(data);
        }

        saveRunProgress({
            contextSwitches: Math.max(getContextSwitches(), currentStepRunData?.contextSwitches ?? 0),
            currentStep,
            currentStepOrder: (progress.findIndex((p) => locationArraysMatch(p, currentLocation)) ?? progress.length) + 1,
            currentStepRunData,
            formData: formikRef.current?.values ?? {},
            handleRunProjectUpdate: async function updateRun(inputs) {
                fetchLazyWrapper<RunProjectUpdateInput, RunProject>({
                    fetch: updateRunProject,
                    inputs,
                    onSuccess,
                });
            },
            handleRunRoutineUpdate: async function updateRun(inputs) {
                fetchLazyWrapper<RunRoutineUpdateInput, RunRoutine>({
                    fetch: updateRunRoutine,
                    inputs,
                    onSuccess,
                });
            },
            isStepCompleted: false,
            isRunCompleted: false,
            logger: console,
            run,
            runnableObject,
            timeElapsed: Math.max(getElapsedTime(), currentStepRunData?.timeElapsed ?? 0),
        });
    }, [currentLocation, currentStepRunData, getContextSwitches, getElapsedTime, progress, rootStep, run, runnableObject, testMode, updateRunProject, updateRunRoutine]);

    useAutoSave({ formikRef, handleSave: handleAutoSave });

    /**
     * End routine early
     */
    const toFinishNotComplete = useCallback(function toFinishNotCompleteCallback() {
        handleAutoSave();
        removeSearchParams(setLocation, ["run", "step"]);
        tryOnClose(onClose, setLocation);
    }, [handleAutoSave, onClose, setLocation]);

    const handleLoadSubroutine = useCallback(function handleLoadSubroutineCallback(id: string) {
        getSubroutines({ ids: [id] });
    }, [getSubroutines]);

    const {
        handleRunSubroutine,
        isGeneratingOutputs,
        subroutineTaskInfo,
    } = useSocketRun({
        formikRef,
        handleRunUpdate: setRun,
        run,
        runnableObject,
    });

    /**
     * Displays either a subroutine view or decision view
     */
    const childView = useMemo(function childViewMemo() {
        if (!currentStep) return null;
        switch (currentStep.__type) {
            case RunStepType.SingleRoutine:
                return <SubroutineView
                    formikRef={formikRef}
                    handleGenerateOutputs={handleRunSubroutine}
                    isGeneratingOutputs={isGeneratingOutputs || subroutineTaskInfo?.status === "Running"}
                    isLoading={isDirectoriesLoading || isSubroutinesLoading}
                    routineVersion={(currentStep as SingleRoutineStep).routineVersion}
                />;
            case RunStepType.Directory:
                return null; //TODO
            // return <DirectoryView
            // />;
            case RunStepType.Decision:
                return <DecisionView
                    data={currentStep as DecisionStep}
                    handleDecisionSelect={toStep}
                    onClose={noop}
                />;
            // TODO come up with default view
            default:
                return null;
        }
    }, [currentStep, handleRunSubroutine, isDirectoriesLoading, isGeneratingOutputs, isSubroutinesLoading, subroutineTaskInfo?.status, toStep]);

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
                            {/* Title */}
                            <TitleStepsStack direction="row" spacing={1}>
                                <Typography variant="h5" component="h2">{name}</Typography>
                                {rootStep !== null && rootStep.__type !== RunStepType.SingleRoutine && stepsInCurrentNode >= 0 ?
                                    <Typography variant="h5" component="h2">({currentLocation[currentLocation.length - 1] ?? 1} of {stepsInCurrentNode})</Typography>
                                    : null}
                                {instructions && <HelpButton markdown={instructions} />}
                                <AutoSaveIndicator formikRef={formikRef} />
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
                            value={completedComplexity / ((run as RunRoutine)?.routineVersion?.complexity ?? (run as RunProject)?.projectVersion?.complexity ?? 1) * PERCENTS}
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
                                    onClick={toNext}
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
