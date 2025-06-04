
import { type FindByIdInput, LINKS, type Location, type LocationData, type ResourceVersion, type Run, type RunCreateInput, type RunIdentifier, RunLoader, RunPersistence, type RunUpdateInput, endpointsResource, endpointsRun } from "@local/shared";
import { Box, Stack, styled } from "@mui/material";
import { useCallback, useEffect, useRef } from "react";
import { fetchData } from "../../api/fetchData.js";
import { ServerResponseParser } from "../../api/responseParser.js";
import { getCookie, setCookie } from "../../utils/localStorage.js";
import { type RunViewProps } from "./types.js";

type StepRunData = {
    contextSwitches?: number;
    elapsedTime?: number;
    // Add other properties of currentStepRunData if needed
}

export function createRunPath(runIdOrTest: string): string {
    return `${LINKS.Run}/${runIdOrTest}`;
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

/**
 * Loader for the RunStateMachine that fetches data from the server.
 * 
 * Supports caching, but is currently disabled. Not sure best way to handle caching, 
 * as we don't want stale data.
 */
class ClientRunLoader extends RunLoader {
    private cachingEnabled = false;

    constructor() {
        super();

        if (this.cachingEnabled) {
            const storedCache = getCookie("RunLoaderCache");
            Object.entries(storedCache.projects).forEach(([key, value]) => this.projectCache.set(key, value));
            Object.entries(storedCache.routines).forEach(([key, value]) => this.routineCache.set(key, value));
        }
    }

    public async fetchLocation(
        location: Location,
    ): Promise<LocationData | null> {
        const objectPromise = fetchData<FindByIdInput, ResourceVersion>({
            ...endpointsResource.findOne,
            inputs: { id: location.objectId },
        }).then(({ data, errors }) => {
            if (errors) {
                ServerResponseParser.displayErrors(errors);
                return null;
            }
            return data;
        });
        const subroutinePromise = location.subroutineId ?
            fetchData<FindByIdInput, ResourceVersion>({
                ...endpointsResource.findOne,
                inputs: { id: location.objectId },
            }).then(({ data, errors }) => {
                if (errors) {
                    ServerResponseParser.displayErrors(errors);
                    return null;
                }
                return data;
            }) : Promise.resolve(null);

        const [object, subroutine] = await Promise.all([objectPromise, subroutinePromise]);
        if (!object) return null;
        return {
            object,
            subroutine: subroutine || null,
        };
    }

    protected onCacheChange(): void {
        if (!this.cachingEnabled) return;
        setCookie("RunLoaderCache", {
            projects: Object.fromEntries(this.projectCache.entries()),
            routines: Object.fromEntries(this.routineCache.entries()),
        });
    }
}

class ClientRunPersistence extends RunPersistence {

    constructor() {
        super();
    }

    async postRun(input: RunCreateInput): Promise<Run | null> {
        const { endpoint, method } = endpointsRun.createOne;
        const result = await fetchData<RunCreateInput, Run>({
            endpoint,
            inputs: input,
            method,
        }).then(({ data, errors }) => {
            if (errors) {
                ServerResponseParser.displayErrors(errors);
                return null;
            }
            return data;
        });
        return result ?? null;
    }

    async putRun(input: RunUpdateInput): Promise<Run | null> {
        const { endpoint, method } = endpointsRun.updateOne;
        const result = await fetchData<RunUpdateInput, Run>({
            endpoint,
            inputs: input,
            method,
        }).then(({ data, errors }) => {
            if (errors) {
                ServerResponseParser.displayErrors(errors);
                return null;
            }
            return data;
        });
        return result ?? null;
    }

    async fetchRun(run: RunIdentifier): Promise<Run | null> {
        const { endpoint, method } = endpointsRun.findOne;
        const storedData = await fetchData<FindByIdInput, Run>({
            endpoint,
            inputs: { id: run.runId },
            method,
        }).then(({ data, errors }) => {
            if (errors) {
                ServerResponseParser.displayErrors(errors);
                return null;
            }
            return data;
        });
        return storedData ?? null;
    }
}

// /**
//  * Runs subroutines
//  */
// class ClientSubroutineExecutor extends SubroutineExecutor {
//     public async runSubroutine(): Promise<RunSubroutineResult> {
//         //TODO call task start endpoint
//         // fetchLazyWrapper<StartRunTaskInput, Success>({
//         //     fetch: startTask,
//         //     inputs: {
//         //         formValues,
//         //         routineVersionId: runnableObjectRef.current.id,
//         //         runId: runRef.current.id,
//         //     },
//         //     spinnerDelay: null, // Disable spinner since this is a background task
//         //     successCondition: (data) => data && data.success === true,
//         //     errorMessage: () => ({ messageKey: "ActionFailed" }),
//         //     // Socket event should update task data on success, so we don't need to do anything here
//         // });
//         return {} as any;
//     }
//     public async estimateCost(): Promise<bigint> {
//         //TODO
//         return BigInt(0);
//     }
// }

const loader = new ClientRunLoader();
const persistence = new ClientRunPersistence();
// const subroutineExecutor = new ClientSubroutineExecutor();

// const stateMachine = new RunStateMachine({
//     loader,
//     logger: console,
//     navigatorFactory,
//     notifier: null, // Only needed for sending socket events. We're receiving them, not sending them.
//     pathSelectionHandler: new AutoPickFirstSelectionHandler(),
//     persistence,
//     subroutineExecutor,
// });

export const ROOT_LOCATION = [1];

const TopBar = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "0.5rem",
    width: "100%",
    height: "64px",
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
const progressBarStyle = { height: "15px" } as const;

//TODO: implement `handleGenerateOutputs`, and soft disable (confirmation dialog) next button when there are missing required fields
//TODO: Need socket to listen to generate output results
export function RunView({
    display,
    isOpen,
    onClose,
}: RunViewProps) {
    // const session = useContext(SessionContext);
    // const { t } = useTranslation();
    // const [, setLocation] = useLocation();

    // // Find data in URL (e.g. current step, runID, whether or not this is a test run)
    // const { runId, testMode } = useMemo(function parseUrlMemo() {
    //     const runId = getLastPathnamePart({ offset: 0 });
    //     return {
    //         runId: (typeof runId === "string" && uuidValidate(base36ToUuid(runId))) ? base36ToUuid(runId) : undefined,
    //         testMode: runId === "test",
    //     } as const;
    //     // eslint-disable-next-line react-hooks/exhaustive-deps
    // }, [isOpen]);

    // const [currentLocation, setCurrentLocation] = useState<Location | null>(null);
    // const handleLocationUpdate = useCallback(function handleLocationUpdateCallback(updatedLocation: Location) {
    //     setCurrentLocation(updatedLocation);
    // }, []);

    // const languages = useMemo(() => getUserLanguages(session), [session]);

    // /**
    //  * Manual decision strategy for choosing the next step in ambiguous cases.
    //  */
    // const manualSelectionHandler = useMemo<PathSelectionStrategy>(function manualSelectionHandlerMemo() {
    //     return {
    //         __type: PathSelectionStrategy.ManualPick,
    //         async chooseOne(validNextNodeIds) {
    //             // Possibly show a modal, wait for user input...
    //             // const userChoice = await askUserWhichStep(validNextNodeIds); TODO create choiceDialog pub/sub
    //             const userChoice = validNextNodeIds[0];
    //             return userChoice.nodeId;
    //         },
    //         async chooseMultiple(validNextNodeIds) {
    //             // Possibly show a modal, wait for user input...
    //             // const userChoices = await askUserWhichSteps(validNextNodeIds); TODO create choiceDialog pub/sub
    //             const userChoices = validNextNodeIds;
    //             return userChoices.map((choice) => choice.nodeId);
    //         },
    //     };
    // }, []);
    // const changePathSelectionStrategy = useCallback(function changePathSelectionStrategyCallback(pathSelectionStrategy: PathSelectionStrategy) {
    //     let pathSelectionHandler: PathSelectionHandler;
    //     switch (pathSelectionStrategy) {
    //         case "AutoPickFirst":
    //             pathSelectionHandler = new AutoPickFirstSelectionHandler();
    //             break;
    //         case "AutoPickLLM":
    //             pathSelectionHandler = new AutoPickLLMSelectionHandler();
    //             break;
    //         case "AutoPickRandom":
    //             pathSelectionHandler = new AutoPickRandomSelectionHandler();
    //             break;
    //         case "ManualPick":
    //             pathSelectionHandler = new ManualPickSelectionHandler();
    //             break;
    //         default:
    //             console.error(`Unsupported path selecetion strategy type: ${pathSelectionStrategy}`);
    //     }
    // }, []);

    // /**
    //  * The amount of routine completed so far, measured in complexity. 
    //  * Used with the routine's total complexity to calculate percent complete
    //  */
    // const [completedComplexity, setCompletedComplexity] = useState(0);
    // /**
    //  * Ordered list of locations for every completed step in the routine.
    //  */
    // const [progress, setProgress] = useState<Location[]>([]);
    // useEffect(function calculateRunStatsEffect() {
    //     if (!run) return;
    //     // Set completedComplexity
    //     setCompletedComplexity(run.completedComplexity);
    //     // Calculate progress using run.steps
    //     const existingProgress: number[][] = [];
    //     for (const step of run.steps) {
    //         // If location found and is not a duplicate (a duplicate here indicates a mistake with storing run data elsewhere)
    //         if (step.step && !existingProgress.some((progress) => new RunStepNavigator(console).locationArraysMatch(progress, step.step))) {
    //             existingProgress.push(step.step);
    //         }
    //     }
    //     setProgress(existingProgress);
    // }, [run]);
    // const progressPercentage = useMemo(function progressPercentageMemo() {
    //     //TODO use state machine
    //     return 100;
    // }, []);

    // const currentStepRunData = useMemo<RunProjectStep | RunRoutineStep | undefined>(function currentStepRunDataMemo() {
    //     const runStep = run?.steps?.find((step) => new RunStepNavigator(console).locationArraysMatch(step.step, currentLocation));
    //     return runStep;
    // }, [run?.steps, currentLocation]);

    // /** Stores user inputs and generated outputs */
    // const formikRef = useRef<FormikProps<object>>(null);
    // useEffect(function setInputsOnRunLoad() {
    //     if (!runnableObject || runnableObject.__typename !== "RoutineVersion") return;
    //     const values = FormBuilder.generateInitialValuesFromRoutineConfig(runnableObjectonfig, run as RunRoutine);

    //     // Formik doesn't seem to like setting the values normally
    //     formikRef.current?.setValues(values);
    //     setTimeout(() => { formikRef.current?.resetForm({ values }); }, 100);
    // }, [run, currentLocation, runnableObject]);

    // const { getContextSwitches, getElapsedTime } = useStepMetrics(currentLocation, currentStepRunData);

    // // Query subdirectories and subroutines, since the whole run may not be able to load at once
    // const [getDirectories, { data: queriedSubdirectories, loading: isDirectoriesLoading }] = useLazyFetch<ProjectVersionDirectorySearchInput, ProjectVersionDirectorySearchResult>(endpointsProjectVersionDirectory.findMany);
    // const [getSubroutines, { data: queriedSubroutines, loading: isSubroutinesLoading }] = useLazyFetch<RoutineVersionSearchInput, RoutineVersionSearchResult>(endpointsRoutineVersion.findMany);

    // const currentStep = useMemo(function currentStepMemo() {
    //     if (!rootStep) return null;
    //     return new RunStepNavigator(console).stepFromLocation(currentLocation, rootStep);
    // }, [currentLocation, rootStep]);

    // const { instructions, name } = useMemo(function displayInfoMemo() {
    //     if (!currentStep) {
    //         return {
    //             instructions: "",
    //             name: "",
    //         };
    //     }
    //     const languages = getUserLanguages(session);
    //     return {
    //         instructions: currentStep.__type === RunStepType.SingleRoutine ? getTranslation((currentStep as SingleRoutineStep).routineVersion, languages, true).instructions : "",
    //         name: currentStep.name,
    //     };
    // }, [currentStep, session]);

    // //TODO
    // const unsavedChanges = false;

    // const [updateRunRoutine] = useLazyFetch<RunRoutineUpdateInput, RunRoutine>(endpointsRunRoutine.updateOne);
    // const [updateRunProject] = useLazyFetch<RunProjectUpdateInput, RunProject>(endpointsRunProject.updateOne);

    // const toNext = useCallback(function toNextCallback() {
    //     if (!rootStep || !nextLocation) return;
    //     const nextStep = new RunStepNavigator(console).stepFromLocation(nextLocation, rootStep);
    //     if (!nextStep) return;
    //     toStep(nextStep);
    // }, [nextLocation, rootStep, toStep]);

    // const toPrevious = useCallback(function toPreviousCallback() {
    //     if (!rootStep || !previousLocation) return;
    //     const previousStep = new RunStepNavigator(console).stepFromLocation(previousLocation, rootStep);
    //     if (!previousStep) return;
    //     toStep(previousStep);
    // }, [previousLocation, rootStep, toStep]);

    // const handleAutoSave = useCallback(function handleAutoSaveCallback() {
    //     if (testMode || !run || !rootStep || !runnableObject) return;

    //     // Find step data
    //     const currentStep = new RunStepNavigator(console).stepFromLocation(currentLocation, rootStep);
    //     if (!currentStep) return;

    //     function onSuccess(data: RunProject | RunRoutine) {
    //         setRun(data);
    //     }

    //     saveRunProgress({
    //         contextSwitches: Math.max(getContextSwitches(), currentStepRunData?.contextSwitches ?? 0),
    //         currentStep,
    //         currentStepOrder: (progress.findIndex((p) => new RunStepNavigator(console).locationArraysMatch(p, currentLocation)) ?? progress.length) + 1,
    //         currentStepRunData,
    //         formData: formikRef.current?.values ?? {},
    //         handleRunProjectUpdate: async function updateRun(inputs) {
    //             fetchLazyWrapper<RunProjectUpdateInput, RunProject>({
    //                 fetch: updateRunProject,
    //                 inputs,
    //                 onSuccess,
    //             });
    //         },
    //         handleRunRoutineUpdate: async function updateRun(inputs) {
    //             fetchLazyWrapper<RunRoutineUpdateInput, RunRoutine>({
    //                 fetch: updateRunRoutine,
    //                 inputs,
    //                 onSuccess,
    //             });
    //         },
    //         isStepCompleted: false,
    //         isRunCompleted: false,
    //         logger: console,
    //         run,
    //         runnableObject,
    //         timeElapsed: Math.max(getElapsedTime(), currentStepRunData?.timeElapsed ?? 0),
    //     });
    // }, [currentLocation, currentStepRunData, getContextSwitches, getElapsedTime, progress, rootStep, run, runnableObject, testMode, updateRunProject, updateRunRoutine]);

    // useAutoSave({ formikRef, handleSave: handleAutoSave });

    // useEffect(function autosaveOnUnloadEffect() {
    //     window.addEventListener("beforeunload", handleAutoSave);
    //     return () => {
    //         window.removeEventListener("beforeunload", handleAutoSave);
    //     };
    // }, [handleAutoSave]);

    // const handleLoadSubroutine = useCallback(function handleLoadSubroutineCallback(id: string) {
    //     getSubroutines({ ids: [id] });
    // }, [getSubroutines]);

    // const getFormValues = useCallback(function getFormValuesCallback() {
    //     return formikRef.current?.values ?? {};
    // }, []);
    // useSocketRun({
    //     applyRunUpdate,
    //     runId,
    // });

    // /**
    //  * Displays either a subroutine view or decision view
    //  */
    // const childView = useMemo(function childViewMemo() {
    //     if (!currentStep) return null;
    //     switch (currentStep.__type) {
    //         case RunStepType.SingleRoutine:
    //             return <SubroutineView
    //                 formikRef={formikRef}
    //                 handleGenerateOutputs={handleRunSubroutine}
    //                 isGeneratingOutputs={isGeneratingOutputs || subroutineTaskInfo?.status === "Running"}
    //                 isLoading={isDirectoriesLoading || isSubroutinesLoading}
    //                 routineVersion={(currentStep as SingleRoutineStep).routineVersion}
    //             />;
    //         case RunStepType.Directory:
    //             return null; //TODO
    //         // return <DirectoryView
    //         // />;
    //         case RunStepType.Decision:
    //             return <DecisionView
    //                 data={currentStep as DecisionStep}
    //                 handleDecisionSelect={toStep}
    //                 onClose={noop}
    //             />;
    //         // TODO come up with default view
    //         default:
    //             return null;
    //     }
    // }, [currentStep, handleRunSubroutine, isDirectoriesLoading, isGeneratingOutputs, isSubroutinesLoading, subroutineTaskInfo?.status, toStep]);

    // return (
    //     <MaybeLargeDialog
    //         display={display}
    //         id="run-dialog"
    //         isOpen={isOpen}
    //         onClose={onClose}
    //         sxs={dialogStyle}
    //     >
    //         <Box minHeight="100vh">
    //             <Box margin="auto">
    //                 {/* Contains name bar and progress bar */}
    //                 <Stack direction="column" spacing={0}>
    //                     <TopBar>
    //                         {/* Title */}
    //                         <TitleStepsStack direction="row" spacing={1}>
    //                             <Typography variant="h5" component="h2">{name}</Typography>
    //                             {rootStep !== null && rootStep.__type !== RunStepType.SingleRoutine && stepsInCurrentNode >= 0 ?
    //                                 <Typography variant="h5" component="h2">({currentLocation[currentLocation.length - 1] ?? 1} of {stepsInCurrentNode})</Typography>
    //                                 : null}
    //                             {instructions && <HelpButton markdown={instructions} />}
    //                             <AutoSaveIndicator formikRef={formikRef} />
    //                         </TitleStepsStack>
    //                         {/* Steps explorer drawer */}
    //                         {/* {rootStep !== null && rootStep.__type !== RunStepType.SingleRoutine ? <RunStepsDialog
    //                             currStep={currentLocation}
    //                             handleLoadSubroutine={handleLoadSubroutine}
    //                             handleCurrStepLocationUpdate={handleLocationUpdate}
    //                             history={progress}
    //                             percentComplete={progressPercentage}
    //                             rootStep={rootStep}
    //                         /> : <Box width="48px"></Box>} */}
    //                     </TopBar>
    //                     {/* Progress bar */}
    //                     {rootStep !== null && rootStep.__type !== RunStepType.SingleRoutine && <LinearProgress
    //                         color="secondary"
    //                         variant="determinate"
    //                         value={completedComplexity / ((run as RunRoutine)?.routineVersion?.complexity ?? (run as RunProject)?.projectVersion?.complexity ?? 1) * PERCENTS}
    //                         sx={progressBarStyle}
    //                     />}
    //                 </Stack>
    //                 <ContentBox>
    //                     {childView}
    //                 </ContentBox>
    //                 <BottomActionsGrid display={display}>
    //                     <Grid item xs={6} p={1}>
    //                         <Button
    //                             disabled={!previousLocation || unsavedChanges || isRunLoading}
    //                             fullWidth
    //                             startIcon={<ArrowLeftIcon />}
    //                             onClick={toPrevious}
    //                             variant="outlined"
    //                         >
    //                             {t("Previous")}
    //                         </Button>
    //                     </Grid>
    //                     <Grid item xs={6} p={1}>
    //                         <Button
    //                             disabled={unsavedChanges || isRunLoading}
    //                             fullWidth
    //                             startIcon={!nextLocation && currentStep?.__type !== RunStepType.Decision ? <SuccessIcon /> : <ArrowRightIcon />}
    //                             onClick={toNext}
    //                             variant="contained"
    //                         >
    //                             {!nextLocation && currentStep?.__type !== RunStepType.Decision ? t("Complete") : t("Next")}
    //                         </Button>
    //                     </Grid>
    //                 </BottomActionsGrid>
    //             </Box>
    //         </Box>
    //     </MaybeLargeDialog>
    // );
    return null;
}
