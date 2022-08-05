/**
 * Drawer to display the steps of a routine, displayed as a vertical tree
 */
import { useCallback, useState } from 'react';
import {
    AccountTree as TreeIcon,
    Close as CloseIcon,
    Launch as OpenStepIcon,
} from '@mui/icons-material';
import {
    alpha,
    Box,
    Checkbox,
    IconButton,
    styled,
    SvgIcon,
    SwipeableDrawer,
    Typography,
    useTheme,
} from '@mui/material';
import { RunStepsDialogProps } from '../types';
import { TreeItem, treeItemClasses, TreeView } from '@mui/lab';
import { RoutineStep } from 'types';
import { locationArraysMatch, parseSearchParams, routineHasSubroutines, RoutineStepType, stringifySearchParams } from 'utils';
import { useLocation } from 'wouter';

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

const StyledTreeItem = styled((props: any) => {
    const {
        label,
        isComplete,
        isSelected,
        onOpen,
        onToStep,
        ...other
    } = props;
    return (
        <TreeItem
            label={
                <Box sx={{ display: 'flex', alignItems: 'center', p: 0.5, pr: 0 }} onClick={onOpen}>
                    <Typography variant="body2" sx={{ fontWeight: 'inherit', flexGrow: 1 }}>
                        {label}
                    </Typography>
                    {/* Indicator for completeness */}
                    {isComplete && <Checkbox
                        size="small"
                        color='secondary'
                        checked={true}
                    />}
                    {/* Redirects to step */}
                    {onToStep && <IconButton size="small" onClick={onToStep}>
                        <OpenStepIcon />
                    </IconButton>}
                </Box>
            }
            {...other}
        />
    )
})(({ theme }) => ({
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
    [`& .${treeItemClasses.label}`]: {
        fontSize: '1.2rem',
        '& p': {
            paddingTop: '12px',
            paddingBottom: '12px',
        }
    },
}));

export const RunStepsDialog = ({
    handleLoadSubroutine,
    handleCurrStepLocationUpdate,
    history,
    percentComplete,
    stepList,
    sxs,
    zIndex,
}: RunStepsDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const [isOpen, setIsOpen] = useState(false);
    const toggleOpen = useCallback(() => setIsOpen(!isOpen), [isOpen]);
    const closeDialog = () => { setIsOpen(false) };

    /**
     * Checks if a routine is complete. If it is a subroutine,
     * recursively checks all subroutine steps.
     * @param step RoutineStep
     * @param location Array indicating step location
     */
    const isComplete = useCallback((step: RoutineStep, location: number[]) => {
        switch (step.type) {
            // If RoutineList, check all child steps
            case RoutineStepType.RoutineList:
                return step.steps.every((childStep, index) => isComplete(childStep, [...location, index + 1]));
            // If Subroutine, check if subroutine is loaded
            case RoutineStepType.Subroutine:
                // Not loaded if subroutine has its own subroutines (since it would be converted to a RoutineList
                // if it was loaded)
                if (routineHasSubroutines(step.routine)) {
                    // We can't know if it is complete until it is loaded
                    return false;
                }
                // If simple routine
                return history.some(h => locationArraysMatch(h, location));
            // If decision
            case RoutineStepType.Decision:
                return history.some(h => locationArraysMatch(h, location));
        }
    }, [history]);

    /**
     * Generate a tree of the subroutine's steps
     */
    const getTreeItem = useCallback((step: RoutineStep, location: number[] = [1]) => {
        // Ignore first number in location array, as it only exists to group the tree items
        const realLocation = location.slice(1);
        // Determine if step is complete
        const completed = isComplete(step, realLocation);
        const locationLabel = location.join('.');
        const realLocationLabel = realLocation.join('.');
        // Helper function for navigating to step
        const toLocation = () => {
            // Update URL
            const searchParams = parseSearchParams(window.location.search);
            setLocation(stringifySearchParams({
                run: searchParams.run,
                step: realLocation
            }), { replace: true });
            // Update current step location
            handleCurrStepLocationUpdate(realLocation);
            // Close dialog
            closeDialog();
        }
        // Item displayed differently depending on its type
        switch (step.type) {
            // A decision step never has children
            case RoutineStepType.Decision:
                return <StyledTreeItem nodeId={locationLabel} label={`${realLocationLabel}. Decision`} onToStep={toLocation} isComplete={completed} />
            // A subroutine may have children, but they may not be loaded
            case RoutineStepType.Subroutine:
                // If there are further steps, add a "Loaging" node
                if (routineHasSubroutines(step.routine)) {
                    return (
                        <StyledTreeItem nodeId={locationLabel} label={`${realLocationLabel}. ${step.title}`} onToStep={toLocation}>
                            <StyledTreeItem nodeId={`${locationLabel}-loading`} label="Loading..." onOpen={() => { handleLoadSubroutine(step.routine.id) }} />
                        </StyledTreeItem>
                    )
                }
                return <StyledTreeItem nodeId={locationLabel} label={`${realLocationLabel}. ${step.title}`} onToStep={toLocation} isComplete={completed} />

            // A routine list always has children
            default:
                let stepItems = step.steps;
                // Don't wrap in a tree item if location is one element long (i.e. the root)
                if (location.length === 1) return stepItems.map((substep, i) => getTreeItem(substep, [...location, i + 1]))
                return (
                    <StyledTreeItem nodeId={locationLabel} label={`${realLocationLabel}. ${step.title}`} isComplete={completed}>
                        {stepItems.map((substep, i) => getTreeItem(substep, [...location, i + 1]))}
                    </StyledTreeItem>
                )
        }
    }, [handleLoadSubroutine, handleCurrStepLocationUpdate, isComplete, setLocation]);

    return (
        <>
            {/* Icon for opening/closing dialog */}
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <TreeIcon sx={sxs?.icon} />
            </IconButton>
            {/* The dialog */}
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
                    zIndex: zIndex + 1,
                    '& .MuiDrawer-paper': {
                        background: palette.background.default,
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
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    padding: 1,
                }}>
                    {/* Title */}
                    <Typography variant="h6" sx={{
                        flexGrow: 1,
                        color: palette.primary.contrastText,
                    }}>
                        {`Steps (${Math.floor(percentComplete)}% Complete)`}
                    </Typography>
                    <IconButton onClick={closeDialog} sx={{
                        color: palette.primary.contrastText,
                        borderRadius: 0,
                        borderBottom: `1px solid ${palette.primary.dark}`,
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
                    sx={{
                        height: 240,
                        flexGrow: 1,
                        maxWidth: 400,
                        overflowY: 'auto',
                        // Add padding to bottom to account for iOS navbar (safe-area-inset-bottom not working for some reason)
                        paddingBottom: '80px',
                    }}
                >
                    {stepList && getTreeItem(stepList)}
                </TreeView>
            </SwipeableDrawer>
        </>
    );
}