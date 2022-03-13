import { APP_LINKS } from "@local/shared";
import { Box, Button, IconButton, LinearProgress, Stack, Typography } from "@mui/material"
import { HelpButton, RunStepsDialog } from "components";
import { SubroutineView } from "components/views/SubroutineView/SubroutineView";
import { useLocation, useRoute } from "wouter";
import { RunRoutineViewProps } from "../types";
import {
    ArrowBack as PreviousIcon,
    ArrowForward as NextIcon,
    Close as CloseIcon,
} from '@mui/icons-material';
import { useEffect, useMemo, useState } from "react";
import { getTranslation, RoutineStepType } from "utils";
import { useLazyQuery } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { validate as uuidValidate } from 'uuid';
import { Node, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, Routine, RoutineStep } from "types";

const helpText =
    `
OIFHDKSHFKHDSKJFLHDS TODO
`

const TERTIARY_COLOR = '#95f3cd';

export const RunRoutineView = ({
    handleClose,
    session
}: RunRoutineViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params1] = useRoute(`${APP_LINKS.Build}/:routineId/:subroutineId`);
    const [, params2] = useRoute(`${APP_LINKS.Run}/:routineId/:subroutineId`);
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
    // Query current subroutine, if needed. Main routine may have the data
    const [getSubroutine, { data: subroutineData, loading: subroutineLoading }] = useLazyQuery<routine, routineVariables>(routineQuery);
    const [subroutine, setSubroutine] = useState<Routine | null>(null);
    useEffect(() => {
        const subroutineId = params1?.subroutineId ?? params2?.subroutineId ?? '';
        if (uuidValidate(subroutineId)) {
            // If ID in main routine, use that
            const nodeWithSubroutine = routineData?.routine?.nodes
                ?.find((node: Node) => (node.data as NodeDataRoutineList)?.routines
                    ?.find((item: NodeDataRoutineListItem) => item.routine?.id === subroutineId));
            if (nodeWithSubroutine) {
                const item = (nodeWithSubroutine.data as NodeDataRoutineList).routines
                    .find((item: NodeDataRoutineListItem) => item.routine?.id === subroutineId);
                setSubroutine(item?.routine as any as Routine);
            } else {
                getSubroutine({ variables: { input: { id: subroutineId } } })
            }
        }
    }, [getSubroutine, params1?.subroutineId, params2?.subroutineId, routineData]);
    useEffect(() => {
        if (subroutineData?.routine) setSubroutine(subroutineData.routine);
    }, [subroutineData]);

    const { description, instructions, title } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            description: getTranslation(subroutine, 'description', languages, true),
            instructions: getTranslation(subroutine, 'instructions', languages, true),
            title: getTranslation(subroutine, 'title', languages, true) ?? 'Untitled'
        };
    }, [subroutine, session]);

    /**
     * Calculate the known subroutines. If a subroutine has a complexity > 1, then there are more subroutines to run.
     */
    const steps: RoutineStep[] = useMemo(() => {
        if (!routine || !routine.nodes || !routine.nodeLinks) return [];
        // Find all nodes that are routine lists
        let routineListNodes = routine.nodes.filter((node: Node) => Boolean((node.data as NodeDataRoutineList)?.routines));
        // Also find the start node
        const startNode = routine.nodes.find((node: Node) => node.columnIndex === 0 && node.rowIndex === 0);
        // Sort by column, then row
        routineListNodes = routineListNodes.sort((a, b) => {
            const aCol = a.columnIndex ?? 0;
            const bCol = b.columnIndex ?? 0;
            if (aCol !== bCol) return aCol - bCol;
            const aRow = a.rowIndex ?? 0;
            const bRow = b.rowIndex ?? 0;
            return aRow - bRow;
        })
        // Create result array
        let result: RoutineStep[] = [];
        // If multiple links from start node, create decision step
        const startLinks = routine.nodeLinks.filter((link: NodeLink) => link.fromId === startNode?.id);
        if (startLinks.length > 1) {
            result.push({ type: RoutineStepType.Decision, links: startLinks });
        }
        // Loop through all nodes
        for (const node of routineListNodes) {
            result.push({ type: RoutineStepType.RoutineList, node });
            const links = routine.nodeLinks.filter((link: NodeLink) => link.fromId === node.id);
            if (links.length > 1) {
                result.push({ type: RoutineStepType.Decision, links });
            }
        }
        return result;
    }, [routine]);

    //TODO
    const currentStep = 1;
    const stepsInNode = 7;
    const progress = 75;
    const hasPrevious = true;
    const hasNext = true;
    const unsavedChanges = false;
    const subroutineComplete = true;

    /**
     * Navigate to the previous subroutine
     */
    const toPrevious = () => {
        //TODO
    }

    /**
     * Navigate to the next subroutine
     */
    const toNext = () => {
        //TODO
    }

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
                        backgroundColor: (t) => t.palette.primary.light,
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
                            {(currentStep && stepsInNode) ?
                                <Typography variant="h5" component="h2">({currentStep} of {stepsInNode})</Typography>
                                : null}
                            {/* Help icon */}
                            <HelpButton markdown={helpText} sx={{ color: TERTIARY_COLOR }} />
                        </Stack>
                        {/* Steps explorer drawer */}
                        <RunStepsDialog
                            routineId={routine?.id}
                            steps={steps}
                            sxs={{ icon: { marginLeft: 1, width: '32px', height: '32px' } }}
                        />
                    </Box>
                    {/* Progress bar */}
                    <LinearProgress color="secondary" variant="determinate" value={progress} />
                </Stack>
                {/* Main content. For now, either looks like view of a basic routine, or options to select an edge */}
                <SubroutineView
                    hasPrevious={false}
                    hasNext={false}
                    session={session}
                />
                {/* Action bar */}
                <Box p={2} sx={{
                    background: (t) => t.palette.primary.light,
                    position: 'fixed',
                    bottom: 0,
                    width: '100vw',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                    paddingBottom: { md: '16px' },
                }}>
                    <Stack direction="row" spacing={1}>
                        {hasPrevious ? <Button
                            fullWidth
                            startIcon={<PreviousIcon />}
                            onClick={toPrevious}
                            disabled={unsavedChanges}
                            sx={{ width: 'min(48vw, 250px)' }}
                        >Previous</Button> : null}
                        <Button
                            fullWidth
                            startIcon={<NextIcon />}
                            onClick={toNext} // NOTE: changes are saved on next click
                            disabled={!subroutineComplete}
                            sx={{ width: 'min(48vw, 250px)' }}
                        >Next</Button>
                    </Stack>
                </Box>
            </Box>
        </Box>
    )
}