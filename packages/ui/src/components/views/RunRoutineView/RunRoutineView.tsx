import { APP_LINKS } from "@local/shared";
import { Box, Button, IconButton, LinearProgress, Stack, Typography } from "@mui/material"
import { DecisionView, HelpButton, RunStepsDialog } from "components";
import { SubroutineView } from "components/views/SubroutineView/SubroutineView";
import { useLocation, useRoute } from "wouter";
import { RunRoutineViewProps } from "../types";
import {
    ArrowBack as PreviousIcon,
    ArrowForward as NextIcon,
    Close as CloseIcon,
    DoneAll as CompleteIcon,
} from '@mui/icons-material';
import { useCallback, useEffect, useMemo, useState } from "react";
import { getTranslation, getUserLanguages, Pubs, RoutineStepType, updateArray, useHistoryState } from "utils";
import { useLazyQuery, useMutation } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { validate as uuidValidate } from 'uuid';
import { DecisionStep, Node, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, Routine, RoutineListStep, RoutineStep, SubroutineStep } from "types";
import { parseSearchParams } from "utils/navigation/urlTools";
import { NodeType } from "graphql/generated/globalTypes";
import { routineComplete, routineCompleteVariables } from "graphql/generated/routineComplete";
import { routineCompleteMutation } from "graphql/mutation";
import { mutationWrapper } from "graphql/utils";

const TERTIARY_COLOR = '#95f3cd';

export const RunRoutineView = ({
    handleClose,
    session
}: RunRoutineViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [stepParams, setStepParams] = useState<number[]>(() => {
        const stepUrl = (parseSearchParams(window.location.search).step ?? '').split('.');
        if (Array.isArray(stepUrl)) return stepUrl.map(Number)
        return []
    });
    const [, params1] = useRoute(`${APP_LINKS.Build}/:routineId`);
    const [, params2] = useRoute(`${APP_LINKS.Run}/:routineId`);
    // Query main routine being run. This should not change for the entire orchestration, 
    // no matter how deep we go.
    const [getRoutine, { data: routineData, loading: routineLoading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    const [routine, setRoutine] = useState<Routine | null>(null);
    useEffect(() => {
        const routineId = params1?.routineId ?? params2?.routineId ?? ''
        if (uuidValidate(routineId)) {
            getRoutine({ variables: { input: { id: routineId } } })
        }
    }, [getRoutine, params1?.routineId, params2?.routineId]);
    useEffect(() => {
        if (routineData?.routine) setRoutine(routineData.routine);
    }, [routineData]);

    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [stepList, setStepList] = useState<RoutineStep | null>(null);
    /**
     * Calculate the known subroutines. If a subroutine has a complexity > 1, then there are more subroutines to run.
     */
    useEffect(() => {
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
                description: getTranslation(item.routine, 'description', languages, true),
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
                description: getTranslation(node, 'description', languages, true),
                steps: [...subroutineSteps, ...decisionSteps] as Array<SubroutineStep | DecisionStep>
            });
        }
        // Main routine acts like routine list
        setStepList({
            type: RoutineStepType.RoutineList,
            nodeId: null,
            isOrdered: true,
            title: getTranslation(routine, 'title', languages, true) ?? 'Untitled',
            description: getTranslation(routine, 'description', languages, true),
            steps: resultSteps,
        });
    }, [routine]);

    /**
     * Returns the requested step
     * @param locationArray Array of step numbers that describes nesting of requested step
     */
    const findStep = useCallback((locationArray: number[]): RoutineStep | null => {
        if (!stepList) return null;
        let currNestedSteps: RoutineStep | null = stepList;
        // If array too large, probably an error
        if (locationArray.length > 20) return null;
        for (let i = 0; i < locationArray.length; i++) {
            if (currNestedSteps !== null && currNestedSteps.type === RoutineStepType.RoutineList) {
                currNestedSteps = currNestedSteps.steps.length > Math.max(locationArray[i] - 1, 0) ? currNestedSteps.steps[Math.max(locationArray[i] - 1, 0)] : null;
            }
        }
        return currNestedSteps;
    }, [stepList]);

    const currentStepNumber = useMemo(() => {
        return stepParams.length === 0 ? -1 : Number(stepParams[stepParams.length - 1]);
    }, [stepParams]);

    const stepsInCurrentNode = useMemo(() => {
        if (!stepParams || !stepList) return -1;
        // For each step in ids array (except for the last id), find the nested step in the steps array.
        // If it doesn't exist, return -1;
        let currNestedSteps: RoutineStep = stepList;
        for (let i = 0; i < stepParams.length - 1; i++) {
            if (currNestedSteps.type === RoutineStepType.RoutineList) {
                const curr = currNestedSteps.steps.length > stepParams[i] ? currNestedSteps.steps[stepParams[i]] : null;
                if (curr) currNestedSteps = curr;
            }
        }
        return currNestedSteps.type === RoutineStepType.RoutineList ? (currNestedSteps as RoutineListStep).steps.length : -1;
    }, [stepParams, stepList]);

    /**
     * Every step completed so far. 
     * Steps are stored as an array that describes their nesting, like they appear in the URL (e.g. [1], [1,3], [1,5,2]).
     * TODO History key should be combination of routineId and updated_at, so history is reset when routine is updated.
     */
    const [progress, setProgress] = useHistoryState(params1?.routineId ?? params2?.routineId ?? '', [])

    /**
     * Calculates the complexity of a step
     */
    const getStepComplexity = useCallback((step: RoutineStep): number => {
        switch (step.type) {
            case RoutineStepType.Decision:
                return 1;
            case RoutineStepType.Subroutine:
                return (step as SubroutineStep).routine.complexity;
            case RoutineStepType.RoutineList:
                return (step as RoutineListStep).steps.reduce((acc, curr) => acc + getStepComplexity(curr), 0);
        }
    }, []);

    /**
     * Calculates progress percentage, as complexity of all completed steps / complexity of all steps
     */
    const progressPercentage = useMemo(() => {
        if (!stepList || !(stepList as RoutineListStep).steps.length || !progress || !progress.length || !routine) return 0;
        // Add the complexity of all steps in progress
        let completedComplexity = 0;
        for (const completedStep of progress) {
            const currStep = findStep(completedStep);
            if (currStep) completedComplexity += getStepComplexity(currStep);
        }
        // Find the total complexity
        const totalComplexity = routine.complexity;
        return completedComplexity / totalComplexity * 100;
    }, [progress, stepList, routine, findStep, getStepComplexity]);

    // Query current subroutine, if needed. Main routine may have the data
    const [getSubroutine, { data: subroutineData, loading: subroutineLoading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    const [currentStep, setCurrentStep] = useState<RoutineStep | null>(null);
    useEffect(() => {
        // If no steps, redirect to first step
        if (stepParams.length === 0) {
            setLocation(`?step=1`, { replace: true });
            setStepParams([1]);
            return;
        }
        // Current step is the last step in steps list
        const currStep = findStep(stepParams);
        if (!currStep) {
            // TODO might need to fetch subroutines multiple times to get to current step, so this shouldn't be an error
            return;
        }
        // If current step is a list, then redirect to first step in list
        if (currStep.type === RoutineStepType.RoutineList) {
            const newStepList = [...stepParams, 1];
            setLocation(`?step=${newStepList.join('.')}`, { replace: true });
            setStepParams(newStepList);
            return;
        }
        // If current step is a subroutine, then query if needed (i.e. complexity > 1 and not already queried)
        if (currStep.type === RoutineStepType.Subroutine) {
            const currSubroutine = (currStep as SubroutineStep).routine;
            if (currSubroutine.complexity > 1 && (!currSubroutine.nodes || currSubroutine.nodes.length === 0)) {
                getSubroutine({ variables: { input: { id: currSubroutine.id } } });
            } else {
                setCurrentStep(currStep);
            }
        } else {
            setCurrentStep(currStep);
        }
    }, [stepParams, getSubroutine, stepList, findStep, setLocation]);
    // Add subroutine data to stepList when new data is fetched
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
            description: (subroutineStep as SubroutineStep).description ?? getTranslation(subroutine, 'description', languages, true),
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
        const languages = session?.languages ?? navigator.languages;
        // Find step above current step
        const currStepParent = findStep(stepParams.slice(0, -1));
        return {
            instructions: getTranslation(routine, 'instructions', languages, true),
            title: currStepParent?.title ?? 'Untitled',
        };
    }, [routine, session, stepParams, findStep]);

    /**
     * Calculates previous step params, or null
     * Examples: [2] => [1], [1] => null, [2, 2] => [2, 1], [2, 1] => [2, num in previous step]
     */
    const previousStep = useMemo<number[] | null>(() => {
        if (stepParams.length === 0) return null;
        // Loop backwards. If curr > 1, then return curr - 1 and remove elements after
        for (let i = stepParams.length - 1; i >= 0; i--) {
            const currStepNumber = stepParams[i];
            if (currStepNumber > 1) return [...stepParams.slice(0, i), currStepNumber - 1]
        }
        return null
    }, [stepParams]);

    /**
     * Calculates next step params, or null
     * Examples: [2] => [3] OR [2, 1] if at end of list
     */
    const nextStep = useMemo<number[] | null>(() => {
        // If current step is a decision, return null
        if (currentStep?.type === RoutineStepType.Decision) return null;
        if (stepParams.length === 0) return [1];
        let result = [...stepParams];
        // Loop backwards until a number in stepParams can be incremented. Remove elements after that
        for (let i = result.length - 1; i >= 0; i--) {
            const currStep = findStep(result.slice(0, i));
            if (!currStep) return null;
            if (currStep.type === RoutineStepType.RoutineList) {
                if ((currStep as RoutineListStep).steps.length > result[i]) {
                    result[i]++;
                    return result.slice(0, i + 1)
                }
            }
        }
        return null;
    }, [stepParams, currentStep, findStep]);

    //TODO
    const unsavedChanges = false;
    const subroutineComplete = true;

    /**
      * Navigate to the previous subroutine
      */
    const toPrevious = useCallback(() => {
        if (!previousStep) return;
        // Update current step
        setLocation(`?step=${previousStep.join('.')}`, { replace: true });
        setStepParams(previousStep);
    }, [previousStep, progress, stepParams, setLocation]);

    /**
     * Navigate to the next subroutine
     */
    const toNext = useCallback(() => {
        if (!nextStep) return;
        // Update progress
        let newProgress = Array.isArray(progress) ? [...progress] : []
        const alreadyComplete = newProgress.find(p => p.length === stepParams.length && p.every((val, index) => val === stepParams[index]))
        if (!alreadyComplete) newProgress.push(stepParams);
        setProgress(newProgress);
        // Update current step
        setLocation(`?step=${nextStep.join('.')}`, { replace: true });
        setStepParams(nextStep);
    }, [nextStep, progress, stepParams, setLocation, setProgress]);

    const [logRoutineComplete] = useMutation<routineComplete>(routineCompleteMutation);
    /**
     * Mark routine as complete and navigate
     */
    const toComplete = useCallback(() => {
        // Log complete
        mutationWrapper({
            mutation: logRoutineComplete,
            input: { id: routine?.id ?? '', standalone: true },
            successMessage: () => 'Routine completed!🎉',
            onSuccess: () => { 
                PubSub.publish(Pubs.Celebration);
                handleClose() 
            },
        })
    }, [logRoutineComplete, handleClose, mutationWrapper, routine]);

    /**
     * End routine early
     */
    const toFinishNotComplete = () => {
        console.log('tofinishnotcomplete');
        //TODO
        // Log incomplete
        logRoutineComplete({ variables: { input: { id: routine?.id ?? '' } } })
        handleClose();
    }

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
     * Navigate to selected decision
     */
    const toDecision = useCallback((selectedNode: Node) => {
        // If end node, finish
        if (selectedNode.type === NodeType.End) {
            toFinishNotComplete();
            return;
        }
        // Find step number of node
        if (!stepList) return;
        const locationArray = findLocationArray(selectedNode.id, stepList);
        if (!locationArray) return;
        // Navigate to current step
        setLocation(`?step=${locationArray.join('.')}`, { replace: true });
        setStepParams(locationArray);
    }, [stepList]);

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
                    loading={subroutineLoading}
                />
            default:
                return <DecisionView
                    data={currentStep as DecisionStep}
                    handleDecisionSelect={toDecision}
                    nodes={routine?.nodes ?? []}
                    session={session}
                />
        }
    }, [currentStep, subroutineLoading]);

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
                        backgroundColor: (t) => t.palette.primary.dark,
                        color: (t) => t.palette.primary.contrastText,
                    }}>
                        {/* Close Icon */}
                        <IconButton
                            edge="end"
                            aria-label="close"
                            onClick={handleClose}
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
                            handleStepParamsUpdate={setStepParams}
                            history={progress}
                            percentComplete={progressPercentage}
                            routineId={routine?.id}
                            stepList={stepList}
                            sxs={{ icon: { marginLeft: 1, width: '32px', height: '32px' } }}
                        />
                    </Box>
                    {/* Progress bar */}
                    <LinearProgress color="secondary" variant="determinate" value={progressPercentage} sx={{ height: '15px' }} />
                </Stack>
                {/* Main content. For now, either looks like view of a basic routine, or options to select an edge */}
                <Box sx={{
                    background: 'fixed radial-gradient(circle, rgba(208,213,226,1) 7%, rgba(179,191,217,1) 66%, rgba(160,188,249,1) 94%)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    margin: 'auto',
                    overflowY: 'auto',
                    minHeight: '88vh',
                }}>
                    {childView}
                </Box>
                {/* Action bar */}
                <Box p={2} sx={{
                    background: (t) => t.palette.primary.dark,
                    position: 'fixed',
                    bottom: 0,
                    width: '100vw',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                    paddingBottom: { md: '16px' },
                }}>
                    <Stack direction="row" spacing={1}>
                        {previousStep && <Button
                            fullWidth
                            startIcon={<PreviousIcon />}
                            onClick={toPrevious}
                            disabled={unsavedChanges}
                            sx={{ width: 'min(48vw, 250px)' }}
                        >Previous</Button>}
                        {nextStep && (<Button
                            fullWidth
                            startIcon={<NextIcon />}
                            onClick={toNext} // NOTE: changes are saved on next click
                            disabled={!subroutineComplete}
                            sx={{ width: 'min(48vw, 250px)' }}
                        >Next</Button>)}
                        {!nextStep && currentStep?.type !== RoutineStepType.Decision && (<Button
                            fullWidth
                            startIcon={<CompleteIcon />}
                            onClick={toComplete}
                            sx={{ width: 'min(48vw, 250px)' }}
                        >Complete</Button>)}
                    </Stack>
                </Box>
            </Box>
        </Box>
    )
}