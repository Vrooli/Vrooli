import { Node, NodeLink, NodeRoutineListItem, NodeType, RoutineVersion } from "@shared/consts";
import { DecisionStep, RoutineListStep, RoutineStep, SubroutineStep } from "types";
import { RoutineStepType } from "utils/consts";
import { getTranslation } from "utils/display/translationTools";
import { routineVersionHasSubroutines } from "utils/runUtils";
import { RunViewProps } from "../types";

/**
 * Maximum routine nesting supported
 */
const MAX_NESTING = 20;

/**
 * Inserts step data into a given RoutineStep, at a given location
 * @param stepData Data to insert into the steps object
 * @param steps RoutineListStep to check in
 * @returns RoutineStep for the updated step, or the original step if the step was not found
 */
const insertStep = (stepData: RoutineListStep, steps: RoutineListStep): RoutineListStep => {
    // Initialize step to be returned
    let step: RoutineListStep = steps;
    // Loop through steps
    for (let i = 0; i < step.steps.length; i++) {
        const currentStep = step.steps[i];
        // If step is a subroutine step, check if it matches the stepData
        if (currentStep.type === RoutineStepType.Subroutine) {
            // If it matches, replace with stepData
            if ((currentStep as SubroutineStep).routineVersion.id === stepData.routineVersionId) {
                step.steps[i] = stepData;
            }
        }
        // If step is a routine list step, recursively check if it matches the stepData
        else if (currentStep.type === RoutineStepType.RoutineList) {
            step.steps[i] = insertStep(stepData, currentStep as RoutineListStep);
        }
    }
    return step;
}

/**
 * Find the step array of a given nodeId
 * @param nodeId The nodeId to search for
 * @param step The current step object, since this is recursive
 * @param location The current location array, since this is recursive
 * @return The step array of the given step
 */
const locationFromNodeId = (nodeId: string, step: RoutineStep | null, location: number[] = []): number[] | null => {
    if (!step) return null;
    if (step.type === RoutineStepType.RoutineList) {
        if ((step as RoutineListStep)?.nodeId === nodeId) return location;
        const stepList = step as RoutineListStep;
        for (let i = 1; i <= stepList.steps.length; i++) {
            const currStep = stepList.steps[i - 1];
            if (currStep.type === RoutineStepType.RoutineList) {
                const currLocation = locationFromNodeId(nodeId, currStep, [...location, i]);
                if (currLocation) return currLocation;
            }
        }
    }
    return null;
}

/**
 * Find the step array of a given routineId
 * @param routineId The routineId to search for
 * @param step The current step object, since this is recursive
 * @param location The current location array, since this is recursive
 * @return The step array of the given step
 */
const locationFromRoutineId = (routineId: string, step: RoutineStep | null, location: number[] = []): number[] | null => {
    if (!step) return null;
    // If step is a subroutine, check if it matches the routineId
    if (step.type === RoutineStepType.Subroutine) {
        if ((step as SubroutineStep)?.routineVersion?.id === routineId) return [...location, 1];
    }
    // If step is a routine list, recurse over every subroutine
    else if (step.type === RoutineStepType.RoutineList) {
        const stepList = step as RoutineListStep;
        for (let i = 1; i <= stepList.steps.length; i++) {
            const currStep = stepList.steps[i - 1];
            if (currStep.type === RoutineStepType.RoutineList) {
                const currLocation = locationFromRoutineId(routineId, currStep, [...location, i]);
                if (currLocation) return currLocation;
            }
        }
    }
    return null;
}

/**
 * Uses a location array to find the step at a given location 
 * NOTE: Must have been queried already
 * @param locationArray Array of step numbers that describes nesting of requested step
 * @param steps RoutineStep for the overall routine being run
 * @returns RoutineStep for the requested step, or null if not found
 */
const stepFromLocation = (locationArray: number[], steps: RoutineStep | null): RoutineStep | null => {
    if (!steps) return null;
    let currNestedSteps: RoutineStep | null = steps;
    // If array too large, probably an error
    if (locationArray.length > MAX_NESTING) {
        console.error(`Location array too large in findStep: ${locationArray}`);
        return null;
    }
    // Loop through location array
    for (let i = 0; i < locationArray.length; i++) {
        // Can only continue if end not reached and step is a routine list (no other step type has substeps)
        if (currNestedSteps !== null && currNestedSteps.type === RoutineStepType.RoutineList) {
            currNestedSteps = currNestedSteps.steps.length > Math.max(locationArray[i] - 1, 0) ? currNestedSteps.steps[Math.max(locationArray[i] - 1, 0)] : null;
        }
    }
    return currNestedSteps;
};

/**
 * Determines if a subroutine step needs additional queries, or if it already 
 * has enough data to render
 * @param step The subroutine step to check
 * @returns True if the subroutine step needs additional queries, false otherwise
 */
const subroutineNeedsQuerying = (step: RoutineStep | null | undefined): boolean => {
    // Check for valid parameters
    if (!step || step.type !== RoutineStepType.Subroutine) return false;
    const currSubroutine: Partial<RoutineVersion> = (step as SubroutineStep).routineVersion;
    // If routine has its own subrotines, then it needs querying (since it would be a RoutineList 
    // if it was loaded)
    return routineVersionHasSubroutines(currSubroutine);
}

/**
 * Calculates the complexity of a step
 * @param step The step to calculate the complexity of
 * @returns The complexity of the step
 */
const getStepComplexity = (step: RoutineStep): number => {
    switch (step.type) {
        // One decision, so one complexity
        case RoutineStepType.Decision:
            return 1;
        // Complexity of subroutines stored in routine data
        case RoutineStepType.Subroutine:
            return (step as SubroutineStep).routineVersion.complexity;
        // Complexity of a list is the sum of its children's complexities
        case RoutineStepType.RoutineList:
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
    languages: string[]
): RoutineListStep | null => {
    // Check for required data to calculate steps
    if (!routineVersion || !routineVersion.nodes || !routineVersion.nodeLinks) {
        console.log('routineVersion does not have enough data to calculate steps');
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
    })
    // Create result steps array
    let resultSteps: RoutineStep[] = [];
    // If multiple links from start node, create decision step
    const startLinks = routineVersion.nodeLinks.filter((link: NodeLink) => link.from.id === startNode?.id);
    if (startLinks.length > 1) {
        resultSteps.push({
            type: RoutineStepType.Decision,
            links: startLinks,
            name: 'Decision',
            description: 'Select a subroutine to run next',
        });
    }
    // Loop through all nodes
    for (const node of routineListNodes) {
        // Find all subroutine steps, and sort by index
        const subroutineSteps: SubroutineStep[] = [...node.routineList!.items]
            .sort((r1, r2) => r1.index - r2.index)
            .map((item: NodeRoutineListItem) => ({
                type: RoutineStepType.Subroutine,
                index: item.index,
                routineVersion: item.routineVersion,
                name: getTranslation(item.routineVersion, languages, true).name ?? 'Untitled',
                description: getTranslation(item.routineVersion, languages, true).description ?? 'Description not found matching selected language',
            }))
        // Find decision step
        const links = routineVersion.nodeLinks.filter((link: NodeLink) => link.from.id === node.id);
        const decisionSteps: DecisionStep[] = links.length > 1 ? [{
            type: RoutineStepType.Decision,
            links,
            name: 'Decision',
            description: 'Select a subroutine to run next',
        }] : [];
        resultSteps.push({
            type: RoutineStepType.RoutineList,
            nodeId: node.id,
            isOrdered: node.routineList?.isOrdered ?? false,
            name: getTranslation(node, languages, true).name ?? 'Untitled',
            description: getTranslation(node, languages, true).description ?? 'Description not found matching selected language',
            steps: [...subroutineSteps, ...decisionSteps] as Array<SubroutineStep | DecisionStep>
        });
    }
    // Return result steps
    return {
        type: RoutineStepType.RoutineList,
        routineVersionId: routineVersion.id,
        isOrdered: true,
        name: getTranslation(routineVersion, languages, true).name ?? 'Untitled',
        description: getTranslation(routineVersion, languages, true).description ?? 'Description not found matching selected language',
        steps: resultSteps,
    };
}

export const RunView = ({
    display = 'page',
    handleClose,
    runnableObject,
    session,
    zIndex,
}: RunViewProps) => {
    return {} as any;
    // const { palette } = useTheme();
    // const [, setLocation] = useLocation();
    // console.log('run view', zIndex)

    // // Find data in URL (e.g. current step, runID, whether or not this is a test run)
    // const params = useReactSearch(null);
    // const { runId, testMode } = useMemo(() => {
    //     return {
    //         runId: (typeof params.run === 'string' && uuidValidate(base36ToUuid(params.run, false))) ? base36ToUuid(params.run) : undefined,
    //         testMode: params.run === 'test',
    //     }
    // }, [params])
    // const [run, setRun] = useState<RunProject | RunRoutine | undefined>(undefined);
    // useEffect(() => {
    //     const run = exists(runnableObject?.you?.runs) ? (runnableObject.you?.runs as (RunProject | RunRoutine)[]).find(run => run.id === runId) : undefined;
    //     exists(run) && setRun(run);
    // }, [runId, runnableObject?.you?.runs]);

    // const [currStepLocation, setCurrStepLocation] = useState<number[]>(Array.isArray(params.step) ? params.step as number[] : [])
    // /**
    //  * Update URL when currStepLocation changes
    //  */
    // useEffect(() => {
    //     addSearchParams(setLocation, { step: currStepLocation });
    // }, [currStepLocation, params, setLocation]);

    // /**
    //  * The amount of routine completed so far, measured in complexity. 
    //  * Used with the routine's total complexity to calculate percent complete
    //  */
    // const [completedComplexity, setCompletedComplexity] = useState(0);

    // /**
    //  * Every step completed so far, as an array of arrays.
    //  * Each step is an array that describes its nesting, like they appear in the URL (e.g. [1], [1,3], [1,5,2]).
    //  */
    // const [progress, setProgress] = useState<number[][]>([]);

    // const languages = useMemo(() => getUserLanguages(session), [session]);

    // /**
    //  * Stores all queried steps, under one main step that represents the entire routine.
    //  * This can be traversed to find step data at a given location.
    //  */
    // const [steps, setSteps] = useState<RoutineListStep | null>(null);

    // /**
    //  * Converts the overall run into a tree of steps, and stores it in the steps ref.
    //  */
    // useEffect(() => {
    //     if (runnableObject.__typename === 'RoutineVersion') {
    //         setSteps(convertRoutineVersionToStep(runnableObject as RoutineVersion, languages));
    //     }
    //     else {
    //         setSteps(convertProjectVersionToStep(runnableObject as ProjectVersion, languages));
    //     }
    // }, [languages, runnableObject]);

    // /**
    //  * When run data is loaded, set completedComplexity and steps completed
    //  */
    // useEffect(() => {
    //     if (!run || !steps) return;
    //     // Set completedComplexity
    //     setCompletedComplexity(run.completedComplexity);
    //     // Calculate progress using run.steps
    //     const existingProgress: number[][] = [];
    //     for (const step of run.steps) {
    //         // If location found and is not a duplicate (a duplicate here indicates a mistake with storing run data elsewhere)
    //         if (step.step && !existingProgress.some((progress) => locationArraysMatch(progress, step.step))) {
    //             existingProgress.push(step.step);
    //         }
    //     }
    //     setProgress(existingProgress);
    // }, [run, steps]);

    // /**
    //  * Display for current step number (last in location array).
    //  */
    // const currentStepNumber = useMemo(() => {
    //     return currStepLocation.length === 0 ? -1 : Number(currStepLocation[currStepLocation.length - 1]);
    // }, [currStepLocation]);

    // /**
    //  * The number of steps in the current-level node, or -1 if not found.
    //  */
    // const stepsInCurrentNode = useMemo(() => {
    //     if (!currStepLocation || !steps) return -1;
    //     // For each step in ids array (except for the last id), find the nested step in the steps array.
    //     // If it doesn't exist, return -1;
    //     let currNestedSteps: RoutineStep = steps;
    //     for (let i = 0; i < currStepLocation.length - 1; i++) {
    //         if (currNestedSteps.type === RoutineStepType.RoutineList) {
    //             const currStepNum = Math.max(0, currStepLocation[i] - 1);
    //             const curr = currNestedSteps.steps.length > currStepNum ? currNestedSteps.steps[currStepNum] : null;
    //             if (curr) currNestedSteps = curr;
    //         }
    //     }
    //     return currNestedSteps.type === RoutineStepType.RoutineList ? (currNestedSteps as RoutineListStep).steps.length : -1;
    // }, [currStepLocation, steps]);

    // /**
    //  * Current step run data
    //  */
    // const currStepRunData = useMemo<RunRoutineStep | undefined>(() => {
    //     const runStep = run?.steps?.find((s: RunRoutineStep) => locationArraysMatch(s.step, currStepLocation));
    //     return runStep;
    // }, [run?.steps, currStepLocation]);

    // /**
    //  * Stores user inputs, which are uploaded to the run's data
    //  */
    // const currUserInputs = useRef<{ [inputId: string]: string }>({});
    // const handleUserInputsUpdate = useCallback((inputs: { [inputId: string]: string }) => {
    //     currUserInputs.current = inputs;
    // }, []);
    // useEffect(() => {
    //     if (!run?.inputs || !Array.isArray(run?.inputs)) return;
    //     const inputs: { [inputId: string]: string } = {};
    //     for (const input of run.inputs) {
    //         inputs[input.input.id] = input.data;
    //     }
    //     if (JSON.stringify(inputs) !== JSON.stringify(currUserInputs.current)) {
    //         handleUserInputsUpdate(inputs);
    //     }
    // }, [run?.inputs, handleUserInputsUpdate]);

    // // Track user behavior during step (time elapsed, context switches)
    // /**
    //  * Interval to track time spent on each step.
    //  */
    // const intervalRef = useRef<NodeJS.Timeout | null>(null);
    // const timeElapsed = useRef<number>(0);
    // const contextSwitches = useRef<number>(0);
    // useEffect(() => {
    //     if (!currStepRunData) return;
    //     // Start tracking time
    //     intervalRef.current = setInterval(() => { timeElapsed.current += 1; }, 1000);
    //     // Reset context switches
    //     contextSwitches.current = 0;
    //     return () => {
    //         if (intervalRef.current) clearInterval(intervalRef.current);
    //     }
    // }, [currStepRunData]);

    // /**
    //  * On tab change, add to contextSwitches
    //  */
    // useEffect(() => {
    //     const handleTabChange = (event: any) => {
    //         if (currStepRunData) {
    //             contextSwitches.current += 1;
    //         }
    //     }
    //     window.addEventListener('focus', handleTabChange);
    //     return () => {
    //         window.removeEventListener('focus', handleTabChange);
    //     }
    // }, [currStepRunData]);

    // /**
    //  * Calculates the percentage of routine completed so far, measured in complexity / total complexity * 100
    //  */
    // const progressPercentage = useMemo(() => getRunPercentComplete(completedComplexity, routineVersion.complexity), [completedComplexity, routineVersion]);

    // // Query current subroutine, if needed. Main routine may have the data
    // const [getSubroutine, { data: subroutine, loading: subroutineLoading }] = useCustomLazyQuery<RoutineVersion, FindVersionInput>(routineVersionFindMany, { errorPolicy: 'all' });
    // const [currentStep, setCurrentStep] = useState<RoutineStep | null>(null);
    // useEffect(() => {
    //     console.log('find step 1', currStepLocation)
    //     // If no steps, redirect to first step
    //     if (currStepLocation.length === 0) {
    //         console.log('find step 2')
    //         setCurrStepLocation([1]);
    //         return;
    //     }
    //     // Current step is the last step in steps list
    //     const currStep = stepFromLocation(currStepLocation, steps);
    //     console.log('find step 3', currStep)
    //     // If current step was not found
    //     if (!currStep) {
    //         console.log('find step 4')
    //         // Check if this is because the routine list has no steps (i.e. it is empty)
    //         //TODO
    //         // Check if this is because the subroutine data hasn't been fetched yet
    //         //TODO
    //         // Otherwise, this is an error
    //         // PubSub.get().publishSnack({ messageKey: 'StepNotFound', severity: 'Error' });
    //         return;
    //     }
    //     // If current step is a list, then redirect to first step in list
    //     if (currStep.type === RoutineStepType.RoutineList) {
    //         console.log('find step 5')
    //         // But first, make sure the list is not empty
    //         // If it is, skip to the next step
    //         if ((currStep as RoutineListStep).steps.length === 0) {
    //             console.log('find step 6')
    //             //TODO
    //             // asdfasfd
    //         }
    //         const newStepList = [...currStepLocation, 1];
    //         console.log('find step 7', newStepList)
    //         setCurrStepLocation(newStepList);
    //         return;
    //     }
    //     // If current step is a subroutine, then query if not all data is available
    //     if (subroutineNeedsQuerying(currStep)) {
    //         console.log('find step 8', currStep)
    //         getSubroutine({ variables: { id: (currStep as SubroutineStep).routineVersion.id } });
    //     } else {
    //         console.log('find step 9')
    //         setCurrentStep(currStep);
    //     }
    // }, [currStepLocation, getSubroutine, params, setCurrStepLocation, steps]);

    // /**
    //  * When new subroutine data is fetched, inject it into steps. 
    //  * This requires finding the subroutine step that matches the ID of the 
    //  * queried subroutine, then converting this into a RoutineList step with its own 
    //  * subroutines
    //  * TODO not tested and likely doesn't work
    //  */
    // useEffect(() => {
    //     if (!subroutine) return;
    //     // Convert to RoutineStep
    //     const subroutineStep = convertRoutineVersionToStep(subroutine, languages);
    //     if (!subroutineStep) return;
    //     // Inject into steps
    //     setSteps(s => s ? insertStep(subroutineStep, s) : subroutineStep);
    // }, [languages, subroutineData]);

    // const { instructions, name } = useMemo(() => {
    //     const languages = getUserLanguages(session);
    //     // Find step above current step
    //     const currStepParent = stepFromLocation(currStepLocation.slice(0, -1), steps);
    //     return {
    //         instructions: getTranslation(routineVersion, languages, true).instructions,
    //         // Ignore name if it's for the main routine (i.e. step is still loading, probably)
    //         name: (currStepParent?.name && currStepLocation.length > 1) ? currStepParent.name : '',
    //     };
    // }, [currStepLocation, routineVersion, session, steps]);

    // /**
    //  * Calculates previous step location array, or null
    //  * Examples: [2] => [1], [1] => null, [2, 2] => [2, 1], [2, 1] => [2, num in previous step]
    //  */
    // const previousStep = useMemo<number[] | null>(() => {
    //     if (currStepLocation.length === 0) return null;
    //     // Loop backwards. If curr > 1, then return curr - 1 and remove elements after
    //     for (let i = currStepLocation.length - 1; i >= 0; i--) {
    //         const currStepNumber = currStepLocation[i];
    //         if (currStepNumber > 1) return [...currStepLocation.slice(0, i), currStepNumber - 1]
    //     }
    //     return null
    // }, [currStepLocation]);

    // /**
    //  * Calculates next step location array, or null. Loops backwards through location array 
    //  * until a location can be incremented, then loops forward until a subroutine without any 
    //  * of its own subroutines is found.
    //  * Examples: [2] => [3] OR [2, 1] if at end of list
    //  */
    // const nextStep = useMemo<number[] | null>(() => {
    //     // If current step is a decision, return null. 
    //     // This is because the next step is determined by the decision, not automatically
    //     if (currentStep?.type === RoutineStepType.Decision) return null;
    //     // If no current step, default to [1]
    //     let result = currStepLocation.length === 0 ? [1] : [...currStepLocation];
    //     let listStepFound = false;
    //     // Loop backwards until a RoutineList node in stepParams can be incremented. Remove elements after that
    //     for (let i = result.length - 1; i >= 0; i--) {
    //         // Update current step
    //         const currStep: RoutineStep | null = stepFromLocation(result.slice(0, i), steps);
    //         // If step not found, we cannot continue
    //         if (!currStep) return null;
    //         // If current step is a RoutineList, check if there is another step in the list
    //         if (currStep.type === RoutineStepType.RoutineList) {
    //             if ((currStep as RoutineListStep).steps.length > result[i]) {
    //                 listStepFound = true;
    //                 result[i]++;
    //                 result = result.slice(0, i + 1);
    //                 break;
    //             }
    //         }
    //     }
    //     // If a RoutineList to increment was not found, return null
    //     if (!listStepFound) return null;
    //     // Remember, a RoutineList is not the actual step being run. We need to find the first 
    //     // subroutine within the RoutineList that doesn't have its own subroutines. This may require 
    //     // additional queries to the server. At least doing it here forces the queries to occur 
    //     // one step before they are needed, which helps with performance.
    //     let currNextStep: RoutineStep | null = stepFromLocation(result, steps);
    //     let endFound = !Boolean(currNextStep);
    //     while (!endFound) {
    //         switch (currNextStep?.type) {
    //             // Decisions cannot have any subroutines
    //             case RoutineStepType.Decision:
    //                 endFound = true;
    //                 break;
    //             // If current step is a RoutineList, check if there is another step in the list
    //             case RoutineStepType.RoutineList:
    //                 if ((currNextStep as RoutineListStep).steps.length > 0) {
    //                     result.push(1);
    //                     currNextStep = stepFromLocation(result, steps);
    //                 }
    //                 else endFound = true;
    //                 break;
    //             // If current step is a subroutine, this is either the end or more data needs to be fetched
    //             case RoutineStepType.Subroutine:
    //                 if (!routineVersionHasSubroutines((currNextStep as SubroutineStep).routineVersion)) {
    //                     endFound = true;
    //                 } else {
    //                     endFound = true;
    //                     //TODO - fetch subroutine data
    //                     // result.push(1);
    //                     // currNextStep = findStep(result, steps);
    //                     // console.log('NEXT STEP', currNextStep)
    //                 }
    //                 break;
    //         }
    //     }
    //     // Return result
    //     return result;
    // }, [currStepLocation, currentStep?.type, steps]);

    // //TODO
    // const unsavedChanges = false;
    // const subroutineComplete = true;

    // /**
    //   * Navigate to the previous subroutine
    //   */
    // const toPrevious = useCallback(() => {
    //     if (!previousStep) return;
    //     // Update current step
    //     setCurrStepLocation(previousStep);
    // }, [previousStep, setCurrStepLocation]);

    // const [logRunUpdate] = useCustomMutation<RunRoutine, RunRoutineUpdateInput>(runRoutineUpdate);
    // const [logRunComplete] = useCustomMutation<RunRoutine, RunRoutineCompleteInput>(runRoutineComplete);
    // /**
    //  * Navigate to the next subroutine, or complete the routine.
    //  * Also log progress, time elapsed, and other metrics
    //  */
    // const toNext = useCallback(() => {
    //     // Find step data
    //     const currStep = stepFromLocation(currStepLocation, steps);
    //     // Calculate new progress and percent complete
    //     let newProgress = Array.isArray(progress) ? [...progress] : [];
    //     let newlyCompletedComplexity: number = (currStep ? getStepComplexity(currStep) : 0);
    //     const alreadyComplete: boolean = Boolean(newProgress.find(p => locationArraysMatch(p, currStepLocation)));
    //     // If step was not already completed, update progress
    //     if (!alreadyComplete) {
    //         newProgress.push(currStepLocation);
    //         setProgress(newProgress);
    //         setCompletedComplexity(c => c + newlyCompletedComplexity);
    //     }
    //     // Update current step
    //     if (nextStep) setCurrStepLocation(nextStep);
    //     // If in test mode return
    //     if (testMode || !run) return
    //     // Now we can calculate data for the logs
    //     // Find parent RoutineList step, so we can get the nodeId 
    //     const currParentListStep: RoutineListStep = stepFromLocation(currStepLocation.slice(0, currStepLocation.length - 1), steps) as RoutineListStep;
    //     // Current step will be updated if it already exists in logged data, or created if not
    //     const stepsUpdate = currStepRunData ? [{
    //         id: currStepRunData.id,
    //         status: RunRoutineStepStatus.Completed,
    //         timeElapsed: (currStepRunData.timeElapsed ?? 0) + timeElapsed.current,
    //         contextSwitches: currStepRunData.contextSwitches + contextSwitches.current,
    //     }] : undefined
    //     const stepsCreate = currStepRunData ? undefined : [{
    //         id: uuid(),
    //         order: newProgress.length,
    //         name: currStep?.name ?? '',
    //         nodeId: currParentListStep.nodeId,
    //         subroutineId: currParentListStep.routineVersionId,
    //         step: currStepLocation,
    //         timeElapsed: timeElapsed.current,
    //         contextSwitches: contextSwitches.current,
    //     }];
    //     // If a next step exists, update
    //     if (nextStep) {
    //         mutationWrapper<RunRoutine, RunRoutineUpdateInput>({
    //             mutation: logRunUpdate,
    //             input: {
    //                 id: run.id,
    //                 completedComplexity: alreadyComplete ? undefined : newlyCompletedComplexity,
    //                 stepsCreate,
    //                 stepsUpdate,
    //                 ...runInputsUpdate(run?.inputs as RunRoutineInput[], currUserInputs.current),
    //             },
    //             onSuccess: (data) => {
    //                 setRun(data);
    //             }
    //         })
    //     }
    //     // Otherwise, mark as complete
    //     else {
    //         // Find node data
    //         const currNodeId = currStepRunData?.node?.id;
    //         const currNode = routineVersion.nodes?.find(n => n.id === currNodeId);
    //         const wasSuccessful = currNode?.end?.wasSuccessful ?? true;
    //         mutationWrapper<RunRoutine, RunRoutineCompleteInput>({
    //             mutation: logRunComplete,
    //             input: {
    //                 id: run.id,
    //                 exists: true,
    //                 completedComplexity: alreadyComplete ? undefined : newlyCompletedComplexity,
    //                 finalStepCreate: stepsCreate ? stepsCreate[0] : undefined,
    //                 finalStepUpdate: stepsUpdate ? stepsUpdate[0] : undefined,
    //                 name: getTranslation(routineVersion, getUserLanguages(session), true).name ?? 'Unnamed Routine',
    //                 wasSuccessful,
    //                 ...runInputsUpdate(run?.inputs as RunRoutineInput[], currUserInputs.current),
    //             },
    //             successMessage: () => ({ key: 'RoutineCompleted' }),
    //             onSuccess: () => {
    //                 PubSub.get().publishCelebration();
    //                 removeSearchParams(setLocation, ['run', 'step']);
    //                 handleClose();
    //             },
    //         })
    //     }
    // }, [currStepLocation, currStepRunData, handleClose, logRunComplete, logRunUpdate, nextStep, progress, routineVersion, run, session, setLocation, steps, testMode]);

    // /**
    //  * End routine after reaching end node using a decision step. 
    //  * Routine marked as complete if end node indicates success.
    //  * Also navigates out of run dialog.
    //  */
    // const reachedEndNode = useCallback((endNode: Node) => {
    //     // Make sure correct nodeType was passed
    //     if (endNode.nodeType !== NodeType.End) {
    //         console.error('Passed incorrect nodeType to reachedEndNode');
    //         return;
    //     }
    //     // Check if end was successfully reached
    //     const data = endNode.end!;
    //     const success = data?.wasSuccessful ?? true;
    //     // Don't actually do it if in test mode
    //     if (testMode || !run) {
    //         if (success) PubSub.get().publishCelebration();
    //         removeSearchParams(setLocation, ['run', 'step']);
    //         handleClose();
    //         return;
    //     }
    //     // Log complete. No step data because this function was called from a decision node, 
    //     // which we currently don't store data about
    //     mutationWrapper<RunRoutine, RunRoutineCompleteInput>({
    //         mutation: logRunComplete,
    //         input: {
    //             id: run.id,
    //             exists: true,
    //             name: getTranslation(routineVersion, getUserLanguages(session), true).name ?? 'Unnamed Routine',
    //             wasSuccessful: success,
    //             ...runInputsUpdate(run?.inputs as RunRoutineInput[], currUserInputs.current),
    //         },
    //         successMessage: () => ({ key: 'RoutineCompleted' }),
    //         onSuccess: () => {
    //             PubSub.get().publishCelebration();
    //             removeSearchParams(setLocation, ['run', 'step']);
    //             handleClose();
    //         },
    //     })
    // }, [testMode, run, logRunComplete, routineVersion, session, setLocation, handleClose]);

    // /**
    //  * Stores current progress, both for overall routine and the current subroutine
    //  */
    // const saveProgress = useCallback(() => {
    //     // Dont do this in test mode, or if there's no run data
    //     if (testMode || !run) return;
    //     // Find current step in run data
    //     const stepUpdate = currStepRunData ? {
    //         id: currStepRunData.id,
    //         timeElapsed: (currStepRunData.timeElapsed ?? 0) + timeElapsed.current,
    //         contextSwitches: currStepRunData.contextSwitches + contextSwitches.current,
    //     } : undefined
    //     // Send data to server
    //     mutationWrapper<RunRoutine, RunRoutineUpdateInput>({
    //         mutation: logRunUpdate,
    //         input: {
    //             id: run.id,
    //             stepsUpdate: stepUpdate ? [stepUpdate] : undefined,
    //             ...runInputsUpdate(run?.inputs as RunRoutineInput[], currUserInputs.current),
    //         },
    //         onSuccess: (data) => {
    //             setRun(data);
    //         }
    //     })
    // }, [currStepRunData, logRunUpdate, run, testMode]);

    // /**
    //  * End routine early
    //  */
    // const toFinishNotComplete = useCallback(() => {
    //     saveProgress();
    //     removeSearchParams(setLocation, ['run', 'step']);
    //     handleClose();
    // }, [handleClose, saveProgress, setLocation]);

    // /**
    //  * Navigate to selected decision
    //  */
    // const toDecision = useCallback((selectedNode: Node) => {
    //     // If end node, finish
    //     if (selectedNode.nodeType === NodeType.End) {
    //         reachedEndNode(selectedNode);
    //         return;
    //     }
    //     // Find step number of node
    //     const locationArray = locationFromNodeId(selectedNode.id, steps);
    //     if (!locationArray) return;
    //     // Navigate to current step
    //     console.log('todecision', locationArray);
    //     setCurrStepLocation(locationArray);
    // }, [reachedEndNode, setCurrStepLocation, steps]);

    // /**
    //  * Displays either a subroutine view or decision view
    //  */
    // const childView = useMemo(() => {
    //     console.log('calculating childview', currentStep, run);
    //     if (!currentStep) return null;
    //     switch (currentStep.type) {
    //         case RoutineStepType.Subroutine:
    //             return <SubroutineView
    //                 session={session}
    //                 handleUserInputsUpdate={handleUserInputsUpdate}
    //                 handleSaveProgress={saveProgress}
    //                 owner={routineVersion?.root?.owner}
    //                 routineVersion={(currentStep as SubroutineStep).routineVersion}
    //                 run={run}
    //                 loading={subroutineLoading}
    //                 zIndex={zIndex}
    //             />
    //         default:
    //             return <DecisionView
    //                 data={currentStep as DecisionStep}
    //                 handleDecisionSelect={toDecision}
    //                 nodes={routineVersion?.nodes ?? []}
    //                 session={session}
    //                 zIndex={zIndex}
    //             />
    //     }
    // }, [currentStep, handleUserInputsUpdate, routineVersion?.nodes, routineVersion?.root?.owner, run, saveProgress, session, subroutineLoading, toDecision, zIndex]);

    // return (
    //     <Box sx={{ minHeight: '100vh' }}>
    //         <Box sx={{
    //             margin: 'auto',
    //         }}>
    //             {/* Contains name bar and progress bar */}
    //             <Stack direction="column" spacing={0}>
    //                 {/* Top bar */}
    //                 <Box sx={{
    //                     display: 'flex',
    //                     alignItems: 'center',
    //                     justifyContent: 'space-between',
    //                     padding: '0.5rem',
    //                     width: '100%',
    //                     backgroundColor: palette.primary.dark,
    //                     color: palette.primary.contrastText,
    //                 }}>
    //                     {/* Close Icon */}
    //                     <IconButton
    //                         edge="end"
    //                         aria-label="close"
    //                         onClick={toFinishNotComplete}
    //                         color="inherit"
    //                     >
    //                         <CloseIcon width='32px' height='32px' />
    //                     </IconButton>
    //                     {/* Title and steps */}
    //                     <Stack direction="row" spacing={1} sx={{
    //                         display: 'flex',
    //                         alignItems: 'center',
    //                         justifyContent: 'center',
    //                     }}>
    //                         <Typography variant="h5" component="h2">{name}</Typography>
    //                         {(currentStepNumber >= 0 && stepsInCurrentNode >= 0) ?
    //                             <Typography variant="h5" component="h2">({currentStepNumber} of {stepsInCurrentNode})</Typography>
    //                             : null}
    //                         {/* Help icon */}
    //                         {instructions && <HelpButton markdown={instructions} />}
    //                     </Stack>
    //                     {/* Steps explorer drawer */}
    //                     <RunStepsDialog
    //                         currStep={currStepLocation}
    //                         handleLoadSubroutine={(id: string) => { getSubroutine({ variables: { id } }); }}
    //                         handleCurrStepLocationUpdate={setCurrStepLocation}
    //                         history={progress}
    //                         percentComplete={progressPercentage}
    //                         stepList={steps}
    //                         zIndex={zIndex + 3}
    //                     />
    //                 </Box>
    //                 {/* Progress bar */}
    //                 <LinearProgress color="secondary" variant="determinate" value={completedComplexity / (routineVersion?.complexity ?? 1) * 100} sx={{ height: '15px' }} />
    //             </Stack>
    //             {/* Main content. For now, either looks like view of a basic routine, or options to select an edge */}
    //             <Box sx={{
    //                 background: palette.mode === 'light' ? '#c2cadd' : palette.background.default,
    //                 display: 'flex',
    //                 alignItems: 'center',
    //                 justifyContent: 'center',
    //                 margin: 'auto',
    //                 marginBottom: '72px + env(safe-area-inset-bottom)',
    //                 overflowY: 'auto',
    //                 minHeight: '100vh',
    //             }}>
    //                 {childView}
    //             </Box>
    //             {/* Action bar */}
    //             <Box sx={{
    //                 position: 'fixed',
    //                 bottom: '0',
    //                 paddingBottom: 'env(safe-area-inset-bottom)',
    //                 // safe-area-inset-bottom is the iOS navigation bar
    //                 height: 'calc(56px + env(safe-area-inset-bottom))',
    //                 width: '-webkit-fill-available',
    //                 zIndex: 4,
    //                 background: palette.primary.dark,
    //                 display: 'flex',
    //                 justifyContent: 'center',
    //                 alignItems: 'center',
    //             }}>
    //                 <Grid container spacing={2} sx={{
    //                     width: 'min(100%, 700px)',
    //                     margin: 0,
    //                 }}>
    //                     {/* There are only ever 1 or 2 options shown. 
    //                     In either case, we want the buttons to be placed as 
    //                     if there are always 2 */}
    //                     <Grid item xs={6} p={1}>
    //                         {previousStep && <Button
    //                             fullWidth
    //                             startIcon={<ArrowLeftIcon />}
    //                             onClick={toPrevious}
    //                             disabled={unsavedChanges}
    //                         >
    //                             Previous
    //                         </Button>}
    //                     </Grid>
    //                     <Grid item xs={6} p={1}>
    //                         {nextStep && (<Button
    //                             fullWidth
    //                             startIcon={<ArrowRightIcon />}
    //                             onClick={toNext} // NOTE: changes are saved on next click
    //                             disabled={!subroutineComplete}
    //                         >
    //                             Next
    //                         </Button>)}
    //                         {!nextStep && currentStep?.type !== RoutineStepType.Decision && (<Button
    //                             fullWidth
    //                             startIcon={<SuccessIcon />}
    //                             onClick={toNext}
    //                         >
    //                             Complete
    //                         </Button>)}
    //                     </Grid>
    //                 </Grid>
    //             </Box>
    //             <Box p={2} sx={{
    //                 background: palette.primary.dark,
    //                 position: 'fixed',
    //                 bottom: 0,
    //                 width: '100vw',
    //                 display: 'flex',
    //                 justifyContent: 'center',
    //                 alignItems: 'center',
    //             }}>
    //             </Box>
    //         </Box>
    //     </Box>
    // )
}