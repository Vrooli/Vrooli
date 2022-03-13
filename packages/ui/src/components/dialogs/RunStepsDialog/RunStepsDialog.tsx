/**
 * Drawer to display the steps of a routine, displayed as a vertical tree
 */
import { useCallback, useState } from 'react';
import {
    AccountTree as TreeIcon,
    Close as CloseIcon,
} from '@mui/icons-material';
import {
    alpha,
    Box,
    IconButton,
    styled,
    SvgIcon,
    SwipeableDrawer,
} from '@mui/material';
import { RunStepsDialogProps } from '../types';
import { routine, routineVariables } from "graphql/generated/routine";
import { useLazyQuery } from '@apollo/client';
import { routineQuery } from 'graphql/query';
import { TreeItem, treeItemClasses, TreeView } from '@mui/lab';
import { Node, NodeDataRoutineList, RoutineListStep, RoutineStep } from 'types';
import { getTranslation, RoutineStepType } from 'utils';

function MinusSquare(props) {
    return (
        <SvgIcon fontSize="inherit" style={{ width: 14, height: 14 }} {...props}>
            {/* tslint:disable-next-line: max-line-length */}
            <path d="M22.047 22.074v0 0-20.147 0h-20.12v0 20.147 0h20.12zM22.047 24h-20.12q-.803 0-1.365-.562t-.562-1.365v-20.147q0-.776.562-1.351t1.365-.575h20.147q.776 0 1.351.575t.575 1.351v20.147q0 .803-.575 1.365t-1.378.562v0zM17.873 11.023h-11.826q-.375 0-.669.281t-.294.682v0q0 .401.294 .682t.669.281h11.826q.375 0 .669-.281t.294-.682v0q0-.401-.294-.682t-.669-.281z" />
        </SvgIcon>
    );
}

function PlusSquare(props) {
    return (
        <SvgIcon fontSize="inherit" style={{ width: 14, height: 14 }} {...props}>
            {/* tslint:disable-next-line: max-line-length */}
            <path d="M22.047 22.074v0 0-20.147 0h-20.12v0 20.147 0h20.12zM22.047 24h-20.12q-.803 0-1.365-.562t-.562-1.365v-20.147q0-.776.562-1.351t1.365-.575h20.147q.776 0 1.351.575t.575 1.351v20.147q0 .803-.575 1.365t-1.378.562v0zM17.873 12.977h-4.923v4.896q0 .401-.281.682t-.682.281v0q-.375 0-.669-.281t-.294-.682v-4.896h-4.923q-.401 0-.682-.294t-.281-.669v0q0-.401.281-.682t.682-.281h4.923v-4.896q0-.401.294-.682t.669-.281v0q.401 0 .682.281t.281.682v4.896h4.923q.401 0 .682.281t.281.682v0q0 .375-.281.669t-.682.294z" />
        </SvgIcon>
    );
}

function CloseSquare(props) {
    return (
        <SvgIcon
            className="close"
            fontSize="inherit"
            style={{ width: 14, height: 14 }}
            {...props}
        >
            {/* tslint:disable-next-line: max-line-length */}
            <path d="M17.485 17.512q-.281.281-.682.281t-.696-.268l-4.12-4.147-4.12 4.147q-.294.268-.696.268t-.682-.281-.281-.682.294-.669l4.12-4.147-4.12-4.147q-.294-.268-.294-.669t.281-.682.682-.281.696 .268l4.12 4.147 4.12-4.147q.294-.268.696-.268t.682.281 .281.669-.294.682l-4.12 4.147 4.12 4.147q.294.268 .294.669t-.281.682zM22.047 22.074v0 0-20.147 0h-20.12v0 20.147 0h20.12zM22.047 24h-20.12q-.803 0-1.365-.562t-.562-1.365v-20.147q0-.776.562-1.351t1.365-.575h20.147q.776 0 1.351.575t.575 1.351v20.147q0 .803-.575 1.365t-1.378.562v0z" />
        </SvgIcon>
    );
}

const StyledTreeItem = styled((props: any) => (
    <TreeItem {...props} />
))(({ theme }) => ({
    [`& .${treeItemClasses.iconContainer}`]: {
        '& .close': {
            opacity: 0.3,
        },
    },
    [`& .${treeItemClasses.group}`]: {
        marginLeft: 15,
        paddingLeft: 18,
        borderLeft: `1px dashed ${alpha(theme.palette.text.primary, 0.4)}`,
    },
}));

export const RunStepsDialog = ({
    routineId,
    steps,
    sxs,
}: RunStepsDialogProps) => {
    const [isOpen, setIsOpen] = useState(false);
    const toggleOpen = useCallback(() => setIsOpen(!isOpen), [isOpen]);
    const closeDialog = () => { setIsOpen(false) };

    // Query for loading a subroutine's graph
    const [getSubroutine, { data: subroutineData, loading: subroutineLoading }] = useLazyQuery<routine, routineVariables>(routineQuery);

    return (
        <>
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <TreeIcon sx={sxs?.icon} />
            </IconButton>
            <SwipeableDrawer
                anchor="right"
                open={isOpen}
                onOpen={() => { }}
                onClose={closeDialog}
                ModalProps={{
                    container: document.getElementById("run-routine-view-dialog"),
                    style: { position: "absolute" },
                }}
                sx={{
                    '& .MuiDrawer-paper': {
                        background: (t) => t.palette.background.default,
                        borderRight: `1px solid ${(t) => t.palette.text.primary}`,
                        minHeight: '100vh',
                        minWidth: '300px',
                        overflowY: 'auto',
                    }
                }}
            >
                {/* Title bar */}
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    alignItems: 'center',
                    background: (t) => t.palette.primary.dark,
                    color: (t) => t.palette.primary.contrastText,
                    padding: 1,
                }}>
                    <IconButton onClick={closeDialog} sx={{
                        color: (t) => t.palette.primary.contrastText,
                        borderRadius: 0,
                        borderBottom: `1px solid ${(t) => t.palette.primary.dark}`,
                        justifyContent: 'end',
                        flexDirection: 'row-reverse',
                        marginLeft: 'auto',
                    }}>
                        <CloseIcon />
                    </IconButton>
                </Box>
                {/* Tree display of steps */}
                <TreeView
                    aria-label="routine steps navigator"
                    defaultCollapseIcon={<MinusSquare />}
                    defaultExpandIcon={<PlusSquare />}
                    defaultEndIcon={<CloseSquare />}
                    sx={{ height: 240, flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}
                >
                    {
                        steps.map((step: RoutineStep, i) => {
                            // Decision has no substeps
                            if (step.type === RoutineStepType.Decision) {
                                return <StyledTreeItem nodeId={`${i}`} label={"Decision"} />
                            }
                            // Routine list has substeps
                            // If list is ordered, sort substeps by index
                            const list = ((step as any).node.data as NodeDataRoutineList);
                            let listItems = list.routines;
                            if (list.isOrdered) {
                                listItems = listItems.sort((a, b) => a.index - b.index);
                            }
                            // Otherwise, sort by name
                            else {
                                listItems = listItems.sort((a, b) => (getTranslation(a.routine, 'title', ['en']) ?? 'Untitled').localeCompare(getTranslation(b.routine, 'title', ['en']) ?? 'Untitled'));
                            }
                            return (
                                <StyledTreeItem nodeId={`${i}`} label={getTranslation((step as RoutineListStep).node, 'title', ['en']) ?? 'Untitled'}>
                                    {
                                        listItems.map((subroutine, j) => {
                                            // If complexity = 1, there are no further steps
                                            if (subroutine.routine.complexity === 1) {
                                                return <StyledTreeItem nodeId={`${i}-${j}`} label={getTranslation(subroutine.routine, 'title', ['en']) ?? 'Untitled'} />
                                            }
                                            // Otherwise, Add a "Loading" node that will be replaced with queried data if opened
                                            return (
                                                <StyledTreeItem nodeId={`${i}-${j}`} label={getTranslation(subroutine.routine, 'title', ['en']) ?? 'Untitled'}>
                                                    <StyledTreeItem nodeId={`${i}-${j}-loading`} label="Loading..." />
                                                </StyledTreeItem>
                                            )
                                        })
                                    }
                                </StyledTreeItem>
                            )
                        })
                    }
                </TreeView>
            </SwipeableDrawer>
        </>
    );
}