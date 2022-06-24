import { RunStepStatus } from "@local/shared";
import { Box, Button, Grid, IconButton, LinearProgress, Stack, Typography, useTheme } from "@mui/material"
import { DecisionView, HelpButton, RunStepsDialog } from "components";
import { SubroutineView } from "components/views/SubroutineView/SubroutineView";
import { useLocation } from "wouter";
import { RunViewProps } from "../types";
import {
    ArrowBack as PreviousIcon,
    ArrowForward as NextIcon,
    Close as CloseIcon,
    DoneAll as CompleteIcon,
} from '@mui/icons-material';
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getRunPercentComplete, getTranslation, getUserLanguages, locationArraysMatch, Pubs, routineHasSubroutines, RoutineStepType, TERTIARY_COLOR, updateArray, useReactSearch } from "utils";
import { useLazyQuery, useMutation } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { validate as uuidValidate } from 'uuid';
import { DecisionStep, Node, NodeDataEnd, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, Routine, RoutineListStep, RoutineStep, RunStep, SubroutineStep } from "types";
import { parseSearchParams, stringifySearchParams } from "utils/navigation/urlTools";
import { NodeType } from "graphql/generated/globalTypes";
import { runComplete } from "graphql/generated/runComplete";
import { runCompleteMutation, runUpdateMutation } from "graphql/mutation";
import { mutationWrapper } from "graphql/utils";
import { runUpdate, runUpdateVariables } from "graphql/generated/runUpdate";

/**
 * Maximum routine nesting supported
 */
const MAX_NESTING = 20;

/**
 * Find the step array of a given nodeId
 * @param nodeId The nodeId to search for
 * @param step The current step object, since this is recursive
 * @param location The current location array, since this is recursive
 * @return The step array of the given step
 */
const findLocationArray = (nodeId: string, step: RoutineStep, location: number[] = []): number[] | null => {
    if (step.type === RoutineStepType.RoutineList) {
        if ((step as RoutineListStep)?.nodeId === nodeId) return location;
        const stepList = step as RoutineListStep;
        for (let i = 1; i <= stepList.steps.length; i++) {
            const currStep = stepList.steps[i - 1];
            if (currStep.type === RoutineStepType.RoutineList) {
                const currLocation = findLocationArray(nodeId, currStep, [...location, i]);
                if (currLocation) return currLocation;
            }
        }
    }
    return null;
}

/**
 * Determines if a subroutine step needs additional queries, or if it already 
 * has enough data to render
 * @param step The subroutine step to check
 * @returns True if the subroutine step needs additional queries, false otherwise
 */
const subroutineNeedsQuerying = (step: RoutineStep | null | undefined): boolean => {
    // Check for valid parameters
    if (!step || step.type !== RoutineStepType.Subroutine) return false;
    const currSubroutine: Partial<Routine> = (step as SubroutineStep).routine;
    // If routine has its own subrotines, then it needs querying (since it would be a RoutineList 
    // if it was loaded)
    return routineHasSubroutines(currSubroutine);
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
            return (step as SubroutineStep).routine.complexity;
        // Complexity of a list is the sum of its children's complexities
        case RoutineStepType.RoutineList:
            return (step as RoutineListStep).steps.reduce((acc, curr) => acc + getStepComplexity(curr), 0);
    }
};

export const RunView = ({
    handleClose,
    routine,
    session,
    zIndex,
}: RunViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    // Find data in URL (e.g. current step, runID, whether or not this is a test run)
    const params = useReactSearch();
    const { currStepLocation, runId, testMode } = useMemo(() => {
        return {
            currStepLocation: Array.isArray(params.step) ? params.step as number[] : [],
            runId: typeof params.run === 'string' && uuidValidate(params.run) ? params.run : undefined,
            testMode: params.run === 'test',
        }
    }, [params])
    const run = useMemo(() => {
        if (!routine) return undefined;
        return routine.runs.find(run => run.id === runId);
    }, [routine, runId]);

    /**
     * Updates step location in the URL
     * @param newLocation The new step params, as an array
     */
    const setCurrStepLocation = useCallback((newLocation: number[]) => {
        setLocation(stringifySearchParams({
            ...params,
            step: newLocation,
        }), { replace: true });
    }, [params, setLocation]);

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
     * Stores the structure of known steps in the routine. This includes subroutines, decisions, and any other step types added later. 
     * Further nesting may require additional queries to get the data.
     */
    const [stepList, setStepList] = useState<RoutineStep | null>(null);
    /**
     * Calculate the known subroutines. 
     */
    useEffect(() => {
        // Check for required data to calculate steps
        if (!routine || !routine.nodes || !routine.nodeLinks) {
            setStepList(null)
            return;
        }
        // Find all nodes that are routine lists
        let routineListNodes = routine.nodes.filter((node: Node) => Boolean((node.data as NodeDataRoutineList)?.routines));
        // Also find the start node
        const startNode = routine.nodes.find((node: Node) => node.type === NodeType.Start);
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
        const startLinks = routine.nodeLinks.filter((link: NodeLink) => link.fromId === startNode?.id);
        if (startLinks.length > 1) {
            resultSteps.push({
                type: RoutineStepType.Decision,
                links: startLinks,
                title: 'Decision',
                description: 'Select a subroutine to run next',
            });
        }
        // Loop through all nodes
        for (const node of routineListNodes) {
            // Find all subroutine steps
            let subroutineSteps: SubroutineStep[] = (node.data as NodeDataRoutineList).routines.map((item: NodeDataRoutineListItem) => ({
                type: RoutineStepType.Subroutine,
                index: item.index,
                routine: item.routine as any,
                title: getTranslation(item.routine, 'title', languages, true) ?? 'Untitled',
                description: getTranslation(item.routine, 'description', languages, true) ?? 'Description not found matching selected language',
            }));
            // Sort subroutine steps
            // If list is ordered, sort by index
            if ((node.data as NodeDataRoutineList).isOrdered) {
                // If the step is a routine step, sort by its index. 
                // Otherwise, step is a decision. This goes at the end of the list.
                subroutineSteps = subroutineSteps.sort((a: SubroutineStep, b: SubroutineStep) => a.index - b.index);
            }
            // Otherwise, sort by name
            else {
                subroutineSteps = subroutineSteps.sort((a: SubroutineStep, b: SubroutineStep) => (a.title.localeCompare(b.title)));
            }
            // Find decision step
            const links = routine.nodeLinks.filter((link: NodeLink) => link.fromId === node.id);
            const decisionSteps: DecisionStep[] = links.length > 1 ? [{
                type: RoutineStepType.Decision,
                links,
                title: 'Decision',
                description: 'Select a subroutine to run next',
            }] : [];
            resultSteps.push({
                type: RoutineStepType.RoutineList,
                nodeId: node.id,
                isOrdered: (node.data as NodeDataRoutineList).isOrdered ?? false,
                title: getTranslation(node, 'title', languages, true) ?? 'Untitled',
                description: getTranslation(node, 'description', languages, true) ?? 'Description not found matching selected language',
                steps: [...subroutineSteps, ...decisionSteps] as Array<SubroutineStep | DecisionStep>
            });
        }
        // Main routine acts like routine list
        setStepList({
            type: RoutineStepType.RoutineList,
            nodeId: '',
            isOrdered: true,
            title: getTranslation(routine, 'title', languages, true) ?? 'Untitled',
            description: getTranslation(routine, 'description', languages, true) ?? 'Description not found matching selected language',
            steps: resultSteps,
        });
    }, [languages, routine]);

    /**
     * Returns the requested step from stepList. 
     * NOTE: Must have been queried already.
     * @param locationArray Array of step numbers that describes nesting of requested step
     */
    const findStep = useCallback((locationArray: number[]): RoutineStep | null => {
        if (!stepList) return null;
        let currNestedSteps: RoutineStep | null = stepList;
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
    }, [stepList]);

    /**
     * When run data is loaded, set completedComplexity and steps completed
     */
    useEffect(() => {
        if (!run || !stepList) return;
        // Set completedComplexity
        setCompletedComplexity(run.completedComplexity);
        // Calculate progress using run.steps
        const existingProgress: number[][] = [];
        console.log('going to calculate existing progress:', run.steps, stepList);
        for (const step of run.steps) {
            console.log('in calculating xisting progress step:', step);
            console.log('current location:', step.step);
            // If location found and is not a duplicate (a duplicate here indicates a mistake with storing run data elsewhere)
            if (step.step && !existingProgress.some((progress) => locationArraysMatch(progress, step.step))) {
                existingProgress.push(step.step);
            }
        }
        console.log('setting existing progress:', existingProgress);
        setProgress(existingProgress);
    }, [run, stepList]);

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
        if (!currStepLocation || !stepList) return -1;
        // For each step in ids array (except for the last id), find the nested step in the steps array.
        // If it doesn't exist, return -1;
        let currNestedSteps: RoutineStep = stepList;
        for (let i = 0; i < currStepLocation.length - 1; i++) {
            if (currNestedSteps.type === RoutineStepType.RoutineList) {
                const curr = currNestedSteps.steps.length > currStepLocation[i] ? currNestedSteps.steps[currStepLocation[i]] : null;
                if (curr) currNestedSteps = curr;
            }
        }
        return currNestedSteps.type === RoutineStepType.RoutineList ? (currNestedSteps as RoutineListStep).steps.length : -1;
    }, [currStepLocation, stepList]);

    /**
     * Current step run data
     */
    const currStepRunData = useMemo(() => {
        const runStep = run?.steps?.find((s: RunStep) => locationArraysMatch(s.step, currStepLocation));
        return runStep;
    }, [run?.steps, currStepLocation]);

    /**
     * Interval to track time spent on each step.
     */
    const intervalRef = useRef<NodeJS.Timeout | null>(null);
    const [timeElapsed, setTimeElapsed] = useState<number>(0);
    useEffect(() => {
        if (!currStepRunData) return;
        intervalRef.current = setInterval(() => { setTimeElapsed(t => t + 1); }, 1000);
        return () => {
            if (intervalRef.current) clearInterval(intervalRef.current);
        }
    }, [currStepRunData]);

    /**
     * Calculates the percentage of routine completed so far, measured in complexity / total complexity * 100
     */
    const progressPercentage = useMemo(() => getRunPercentComplete(completedComplexity, routine.complexity), [completedComplexity, routine]);

    // Query current subroutine, if needed. Main routine may have the data
    const [getSubroutine, { data: subroutineData, loading: subroutineLoading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    const [currentStep, setCurrentStep] = useState<RoutineStep | null>(null);
    useEffect(() => {
        // If no steps, redirect to first step
        if (currStepLocation.length === 0) {
            console.log('setting step params because no steps')
            setCurrStepLocation([1]);
            return;
        }
        // Current step is the last step in steps list
        const currStep = findStep(currStepLocation);
        console.log('CURRENT STEP HERE', currStep);
        // If current step was not found
        if (!currStep) {
            // Check if this is because the routine list has no steps (i.e. it is empty)
            //TODO
            // Check if this is because the subroutine data hasn't been fetched yet
            //TODO
            // Otherwise, this is an error
            PubSub.publish(Pubs.Snack, { message: 'Error: Step not found', severity: 'error' });
            return;
        }
        // If current step is a list, then redirect to first step in list
        if (currStep.type === RoutineStepType.RoutineList) {
            // But first, make sure the list is not empty
            // If it is, skip to the next step
            if ((currStep as RoutineListStep).steps.length === 0) {
                //TODO
                // asdfasfd
            }
            const newStepList = [...currStepLocation, 1];
            console.log('setting step params because current step is a list', newStepList)
            setCurrStepLocation(newStepList);
            return;
        }
        // If current step is a subroutine, then query if not all data is available
        if (subroutineNeedsQuerying(currStep)) {
            getSubroutine({ variables: { input: { id: (currStep as SubroutineStep).routine.id } } });
        } else {
            setCurrentStep(currStep);
        }
    }, [currStepLocation, findStep, getSubroutine, params, setCurrStepLocation]);

    /**
     * When new subroutine data is fetched, inject it into stepList. 
     * This requires finding the subroutine step that matches the ID of the 
     * queried subroutine, then converting this into a RoutineList step with its own 
     * subroutines
     */
    useEffect(() => {
        const subroutine = subroutineData?.routine;
        if (!stepList || !subroutine) return;
        // Helper function to recursively find the subroutine location array in the step list
        const indexArrayInStep = (stepList: RoutineStep, id: string): number[] | null => {
            if (stepList.type === RoutineStepType.Subroutine && (stepList as SubroutineStep).routine.id === id) {
                return [0];
            } else if (stepList.type === RoutineStepType.RoutineList) {
                for (let i = 0; i < (stepList as RoutineListStep).steps.length; i++) {
                    const currStep = (stepList as RoutineListStep).steps[i];
                    const currIndex = indexArrayInStep(currStep, id);
                    if (currIndex) {
                        return [i, ...currIndex];
                    }
                }
            }
            return null;
        }
        // If subroutine is found, update it in the step list
        const indexArray = indexArrayInStep(stepList, subroutine.id);
        if (!indexArray) return;
        // Helper function to find a step in stepList by index array
        const findStepByIndex = (stepList: RoutineStep, indexArray: number[]): RoutineStep | null => {
            if (indexArray.length === 0) return stepList;
            if (stepList.type === RoutineStepType.RoutineList) {
                const currStep = (stepList as RoutineListStep).steps[indexArray[0]];
                if (currStep) return findStepByIndex(currStep, indexArray.slice(1));
            }
            return null;
        }
        // Find nested step
        const subroutineStep = findStepByIndex(stepList, indexArray);
        if (!subroutineStep) return;
        // Initialize new step list for loop
        let updatedStepList: RoutineStep | null = {
            type: RoutineStepType.Subroutine,
            routine: subroutine,
            index: (subroutineStep as SubroutineStep).index,
            title: (subroutineStep as SubroutineStep).title ?? getTranslation(subroutine, 'title', languages, true) ?? 'Untitled',
            description: (subroutineStep as SubroutineStep).description ?? getTranslation(subroutine, 'description', languages, true) ?? 'Description not found matching selected language',
        };
        // If loop needed
        if (indexArray.length > 1) {
            // Loop backwards to update the subroutine in the step list
            for (let i = indexArray.length - 2; i >= 0; i--) {
                // Find nested step
                const currStep = findStepByIndex(stepList, indexArray.slice(0, i));
                if (!currStep) break;
                // Every step found in this loop should be a list
                if (currStep.type !== RoutineStepType.RoutineList) break;
                // Add step to updated step list
                updatedStepList = {
                    type: RoutineStepType.RoutineList,
                    nodeId: (currStep as RoutineListStep).nodeId,
                    steps: updateArray(currStep.steps, indexArray[i + 1], updatedStepList),
                    isOrdered: (currStep as RoutineListStep).isOrdered,
                    title: (currStep as RoutineListStep).title,
                    description: (currStep as RoutineListStep).description,
                };
            }
        }
        // Update step list
        setStepList(updatedStepList);
    }, [languages, subroutineData, stepList]);

    const { instructions, title } = useMemo(() => {
        console.log('calculating title', currStepLocation, currStepLocation.slice(0, -1), findStep(currStepLocation.slice(0, -1)))
        const languages = session?.languages ?? navigator.languages;
        // Find step above current step
        const currStepParent = findStep(currStepLocation.slice(0, -1));
        return {
            instructions: getTranslation(routine, 'instructions', languages, true),
            title: currStepParent?.title ?? 'Untitled',
        };
    }, [currStepLocation, findStep, routine, session]);

    /**
     * Calculates previous step params, or null
     * Examples: [2] => [1], [1] => null, [2, 2] => [2, 1], [2, 1] => [2, num in previous step]
     */
    const previousStep = useMemo<number[] | null>(() => {
        if (currStepLocation.length === 0) return null;
        // Loop backwards. If curr > 1, then return curr - 1 and remove elements after
        for (let i = currStepLocation.length - 1; i >= 0; i--) {
            const currStepNumber = currStepLocation[i];
            if (currStepNumber > 1) return [...currStepLocation.slice(0, i), currStepNumber - 1]
        }
        return null
    }, [currStepLocation]);

    /**
     * Calculates next step params, or null
     * Examples: [2] => [3] OR [2, 1] if at end of list
     */
    const nextStep = useMemo<number[] | null>(() => {
        console.log('calculating next step start - currentstep', currentStep)
        // If current step is a decision, return null. 
        // This is because the next step is determined by the decision, not automatically
        if (currentStep?.type === RoutineStepType.Decision) return null;
        // If no current step, default to [1]
        if (currStepLocation.length === 0) return [1];
        let result = [...currStepLocation];
        let listStepFound = false;
        // Loop backwards until a RoutineList node in stepParams can be incremented. Remove elements after that
        for (let i = result.length - 1; i >= 0; i--) {
            // Update current step
            const currStep: RoutineStep | null = findStep(result.slice(0, i));
            // If step not found, we cannot continue
            if (!currStep) return null;
            // If current step is a RoutineList, check if there is another step in the list
            if (currStep.type === RoutineStepType.RoutineList) {
                if ((currStep as RoutineListStep).steps.length > result[i]) {
                    listStepFound = true;
                    result[i]++;
                    result = result.slice(0, i + 1);
                    break;
                }
            }
        }
        // If a RoutineList to increment was not found, return null
        if (!listStepFound) return null;
        // Remember, a RoutineList is not the actual step being run. We need to find the first 
        // subroutine within the RoutineList that doesn't have its own subroutines. This may require 
        // additional queries to the server. At least doing it here forces the queries to occur 
        // one step before they are needed, which helps with performance.
        let currNextStep: RoutineStep | null = findStep(result);
        console.log('NEXT STEP currNextStep', currNextStep)
        let endFound = !Boolean(currNextStep);
        while (!endFound) {
            console.log('not end found loop', currNextStep)
            // If current step is a decision, we're done
            if (currNextStep?.type === RoutineStepType.Decision) {
                endFound = true;
                break;
            }
            // If current step is a subroutine, see if we need to query more steps
            //TODO
            // If subroutine has no nodes, we're done
            //TODO
            // If 
            // temp
            endFound = true;
        }
        // Return result
        console.log('NEXT STEP RESULT', result)
        return result;
    }, [currStepLocation, currentStep, findStep]);

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

    const [logRunUpdate] = useMutation<runUpdate, runUpdateVariables>(runUpdateMutation);
    /**
     * Navigate to the next subroutine. 
     * Also log progress, time elapsed, and other metrics
     */
    const toNext = useCallback(() => {
        // Don't calculate if no next step, since we initialize run data for the next step. 
        // This has the side effect is not logging decisions. If these should be logged, 
        // then the logic will have to change a bit
        if (!nextStep) return;
        console.log('TO NEXT', nextStep);//TODO nextstep only gets next node (I think), not the actual subroutine which will be run. This messes up the stepCreate log
        // Find step data
        const currStep = findStep(currStepLocation);
        // Calculate new progress and percent complete
        let newProgress = Array.isArray(progress) ? [...progress] : [];
        let newComplexity: number = completedComplexity;
        const alreadyComplete = newProgress.find(p => locationArraysMatch(p, currStepLocation));
        // If step was not already completed, update progress
        if (!alreadyComplete) {
            newProgress.push(currStepLocation);
            newComplexity += (currStep ? getStepComplexity(currStep) : 0);
            setProgress(newProgress);
            setCompletedComplexity(newComplexity);
        }
        // Update current step
        setCurrStepLocation(nextStep);
        // If in test mode return
        if (testMode || !run) return
        // Now we can calculate data for the logs
        // Find next step
        const nextData = findStep(nextStep);
        if (nextData?.type === RoutineStepType.RoutineList) {
            // Find data to update current step
            const existingCurrStepData = run.steps.find(s => s.node?.id === (currStep as RoutineListStep)?.nodeId)
            const stepUpdate = existingCurrStepData ? {
                id: existingCurrStepData.id,
                status: RunStepStatus.Completed,
                timeElapsed: (existingCurrStepData.timeElapsed ?? 0) + timeElapsed,
                pickups: existingCurrStepData.pickups + 1,
            } : undefined
            // Make sure next step hasn't been logged already (meaning you've worked on it before)
            const existingNextStepData = run.steps.find(s => s.node?.id === nextData.nodeId);
            if (!existingNextStepData) {
                logRunUpdate({
                    variables: {
                        input: {
                            id: run.id,
                            completedComplexity: newComplexity,
                            stepsCreate: [{
                                order: progress.length,
                                title: (nextData as RoutineListStep).title,
                                nodeId: (nextData as RoutineListStep).nodeId,
                                step: nextStep,
                            }],
                            stepsUpdate: stepUpdate ? [stepUpdate] : undefined,
                        }
                    }
                });
            }
        }
    }, [completedComplexity, currStepLocation, findStep, logRunUpdate, nextStep, progress, run, setCurrStepLocation, testMode, timeElapsed]);

    /**
     * Clears run-specific search params from URL
     */
    const clearSearchParams = useCallback(() => {
        console.log('clearing search params...')
        const params = parseSearchParams(window.location.search);
        delete params.run;
        delete params.step;
        console.log('params after clearing', stringifySearchParams(params))
        setLocation(`${window.location.pathname}${stringifySearchParams(params)}`, { replace: true });
    }, [setLocation]);

    const [logRunComplete] = useMutation<runComplete>(runCompleteMutation);
    /**
     * End routine after reaching end node. 
     * Routine marked as complete if end node indicates success.
     * Also navigates out of run dialog.
     */
    const reachedEndNode = useCallback((endNode: Node) => {
        // Make sure correct node type was passed
        if (endNode.type !== NodeType.End) {
            console.error('Passed incorrect node type to reachedEndNode');
            return;
        }
        // Check if end was successfully reached
        const data = endNode.data as NodeDataEnd;
        const success = data.wasSuccessful;
        // Don't actually do it if in test mode
        if (testMode || !run) {
            if (success) PubSub.publish(Pubs.Celebration);
            clearSearchParams();
            handleClose();
            return;
        }
        // Update logs for current step (time elapsed, pickups, etc.)
        const currentStepRunData = run.steps.find(s => s.node?.id === (currentStep as RoutineListStep)?.nodeId);
        const stepUpdate = currentStepRunData ? {
            id: currentStepRunData.id,
            timeElapsed: (currentStepRunData.timeElapsed ?? 0) + timeElapsed,
            pickups: currentStepRunData.pickups + 1,
        } : undefined
        // Log complete
        mutationWrapper({
            mutation: logRunComplete,
            input: {
                id: run.id,
                standalone: true,
                completedComplexity: run.completedComplexity,
                timeElapsed: (run.timeElapsed ?? 0) + timeElapsed,
                title: getTranslation(routine, 'title', getUserLanguages(session), true),
                pickups: run.pickups + 1,
                stepsUpdate: stepUpdate ? [stepUpdate] : undefined,
                version: routine.version,
                wasSuccessful: success,
            },
            successMessage: () => 'Routine completed!🎉',
            onSuccess: () => {
                PubSub.publish(Pubs.Celebration);
                clearSearchParams();
                handleClose();
            },
        })
    }, [testMode, run, timeElapsed, logRunComplete, routine, session, clearSearchParams, handleClose, currentStep]);
    /**
     * Complete routine after reaching end node.
     */
    const toComplete = useCallback(() => {
        // Find current step
        const currStep = findStep(currStepLocation);
        if (!currStep || currStep.type !== RoutineStepType.Subroutine) {
            console.error('Could not find current step, or current step is not a subroutine', currStep);
            return;
        }
        // const currStepData = currStep as SubroutineStep;
        // // Find current node
        // if (step.type === RoutineStepType.Subroutine
        // console.log('tocomplete', currStep)
    }, [currStepLocation, findStep]);

    /**
     * Stores current progress, both for overall routine and the current subroutine
     */
    const saveProgress = useCallback(() => {
        // Dont do this in test mode, or if there's no run data
        if (testMode || !run) return;
        // Find current step in run data
        const currentStepRunData = run.steps.find(s => s.node?.id === (currentStep as RoutineListStep)?.nodeId);
        const stepUpdate = currentStepRunData ? {
            id: currentStepRunData.id,
            timeElapsed: (currentStepRunData.timeElapsed ?? 0) + timeElapsed,
            pickups: currentStepRunData.pickups + 1,
        } : undefined
        // Send data to server
        logRunUpdate({
            variables: {
                input: {
                    id: run.id,
                    timeElapsed: (run.timeElapsed ?? 0) + timeElapsed,
                    pickups: run.pickups + 1,
                    stepsUpdate: stepUpdate ? [stepUpdate] : undefined,
                }
            }
        });
    }, [currentStep, logRunUpdate, run, testMode, timeElapsed]);

    /**
     * End routine early
     */
    const toFinishNotComplete = useCallback(() => {
        saveProgress();
        clearSearchParams();
        handleClose();
    }, [clearSearchParams, handleClose, saveProgress]);

    /**
     * Navigate to selected decision
     */
    const toDecision = useCallback((selectedNode: Node) => {
        // If end node, finish
        if (selectedNode.type === NodeType.End) {
            reachedEndNode(selectedNode);
            return;
        }
        // Find step number of node
        if (!stepList) return;
        const locationArray = findLocationArray(selectedNode.id, stepList);
        if (!locationArray) return;
        // Navigate to current step
        setCurrStepLocation(locationArray);
    }, [reachedEndNode, setCurrStepLocation, stepList]);

    /**
     * Displays either a subroutine view or decision view
     */
    const childView = useMemo(() => {
        if (!currentStep) return null;
        switch (currentStep.type) {
            case RoutineStepType.Subroutine:
                return <SubroutineView
                    session={session}
                    data={(currentStep as SubroutineStep).routine}
                    handleSaveProgress={saveProgress}
                    owner={routine.owner}
                    loading={subroutineLoading}
                    zIndex={zIndex}
                />
            default:
                return <DecisionView
                    data={currentStep as DecisionStep}
                    handleDecisionSelect={toDecision}
                    nodes={routine?.nodes ?? []}
                    session={session}
                    zIndex={zIndex}
                />
        }
    }, [currentStep, routine?.nodes, routine.owner, saveProgress, session, subroutineLoading, toDecision, zIndex]);

    return (
        <Box sx={{ minHeight: '100vh' }}>
            <Box sx={{
                margin: 'auto',
            }}>
                {/* Contains title bar and progress bar */}
                <Stack direction="column" spacing={0}>
                    {/* Top bar */}
                    <Box sx={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'space-between',
                        padding: '0.5rem',
                        width: '100%',
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
                            <CloseIcon sx={{
                                width: '32px',
                                height: '32px',
                            }} />
                        </IconButton>
                        {/* Title and steps */}
                        <Stack direction="row" spacing={1} sx={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                        }}>
                            <Typography variant="h5" component="h2">{title}</Typography>
                            {(currentStepNumber >= 0 && stepsInCurrentNode >= 0) ?
                                <Typography variant="h5" component="h2">({currentStepNumber} of {stepsInCurrentNode})</Typography>
                                : null}
                            {/* Help icon */}
                            {instructions && <HelpButton markdown={instructions} sx={{ color: TERTIARY_COLOR }} />}
                        </Stack>
                        {/* Steps explorer drawer */}
                        <RunStepsDialog
                            handleLoadSubroutine={(id: string) => { getSubroutine({ variables: { input: { id } } }); }}
                            handleStepParamsUpdate={setCurrStepLocation}
                            history={progress}
                            percentComplete={progressPercentage}
                            stepList={stepList}
                            sxs={{ icon: { marginLeft: 1, width: '32px', height: '32px' } }}
                            zIndex={zIndex + 1}
                        />
                    </Box>
                    {/* Progress bar */}
                    <LinearProgress color="secondary" variant="determinate" value={completedComplexity / (routine?.complexity ?? 1) * 100} sx={{ height: '15px' }} />
                </Stack>
                {/* Main content. For now, either looks like view of a basic routine, or options to select an edge */}
                <Box sx={{
                    background: palette.mode === 'light' ? '#c2cadd' : palette.background.default,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    margin: 'auto',
                    overflowY: 'auto',
                    minHeight: '87vh',
                }}>
                    {childView}
                </Box>
                {/* Action bar */}
                <Box sx={{
                    position: 'fixed',
                    bottom: '0',
                    paddingBottom: 'env(safe-area-inset-bottom)',
                    // safe-area-inset-bottom is the iOS navigation bar
                    height: 'calc(56px + env(safe-area-inset-bottom))',
                    width: '-webkit-fill-available',
                    zIndex: 4,
                    background: palette.primary.dark,
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    <Grid container spacing={2} sx={{
                        maxWidth: '100%',
                        margin: 0,
                    }}>
                        {/* There are only ever 1 or 2 options shown. 
                        In either case, we want the buttons to be placed as 
                        if there are always 2 */}
                        <Grid item xs={6} sx={{
                            padding: '8px 4px 8px 4px',
                            display: 'flex',
                            justifyContent: 'center',
                        }}>
                            {previousStep && <Button
                                fullWidth
                                startIcon={<PreviousIcon />}
                                onClick={toPrevious}
                                disabled={unsavedChanges}
                                sx={{
                                    width: 'min(48vw, 250px)',
                                }}
                            >
                                <Box sx={{ display: { xs: 'none', sm: 'block' } }}>
                                    Previous
                                </Box>
                            </Button>}
                        </Grid>
                        <Grid item xs={6} p={1} sx={{
                            padding: '8px 4px 8px 4px',
                            display: 'flex',
                            justifyContent: 'center',
                        }}>
                            {nextStep && (<Button
                                fullWidth
                                startIcon={<NextIcon />}
                                onClick={toNext} // NOTE: changes are saved on next click
                                disabled={!subroutineComplete}
                                sx={{ width: 'min(48vw, 250px)' }}
                            >
                                <Box sx={{ display: { xs: 'none', sm: 'block' } }}>
                                    Next
                                </Box>
                            </Button>)}
                            {!nextStep && currentStep?.type !== RoutineStepType.Decision && (<Button
                                fullWidth
                                startIcon={<CompleteIcon />}
                                onClick={toComplete}
                                sx={{ width: 'min(48vw, 250px)' }}
                            >
                                <Box sx={{ display: { xs: 'none', sm: 'block' } }}>
                                    Complete
                                </Box>
                            </Button>)}
                        </Grid>
                    </Grid>
                </Box>
                <Box p={2} sx={{
                    background: palette.primary.dark,
                    position: 'fixed',
                    bottom: 0,
                    width: '100vw',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                </Box>
            </Box>
        </Box>
    )
}