import { Box, Button, Dialog, IconButton, Slider, Stack, Tooltip } from '@mui/material';
import { useMemo, useState } from 'react';
import {
    Add as AddIcon,
    Cancel as CancelIcon,
    Pause as PauseIcon,
    PlayCircle as RunIcon,
    Update as UpdateIcon
} from '@mui/icons-material';
import { BuildRunState } from 'utils';
import { BuildBottomContainerProps } from '../types';
import { useLocation } from 'wouter';
import { UpTransition } from 'components/dialogs';
import { RunRoutineView } from 'components/views';

export const BuildBottomContainer = ({
    canSubmitMutate,
    canCancelMutate,
    handleCancelAdd,
    handleCancelUpdate,
    handleAdd,
    handleUpdate,
    handleScaleChange,
    hasPrevious,
    hasNext,
    isAdding,
    isEditing,
    loading,
    scale,
    session,
    sliderColor,
    runState,
}: BuildBottomContainerProps) => {
    const [, setLocation] = useLocation();

    const onScaleChange = (_event: any, newScale: number | number[]) => {
        handleScaleChange(newScale as number);
    };

    const [isRunOpen, setIsRunOpen] = useState(false)
    const runRoutine = () => {
        setLocation(`?step=1`, { replace: true });
        setIsRunOpen(true)
    };
    const stopRoutine = () => {
        setLocation(window.location.pathname, { replace: true });
        setIsRunOpen(false)
    };

    /**
     * Slider for scaling the graph
     */
    const slider = useMemo(() => (
        <Slider
            aria-label="graph-scale"
            defaultValue={1}
            max={1}
            min={0.25}
            onChange={onScaleChange}
            step={0.01}
            value={scale}
            valueLabelDisplay="auto"
            sx={{ 
                color: sliderColor, 
                maxWidth: '500px', 
                marginRight: 2,
            }}
        />
    ), [scale, sliderColor, onScaleChange]);

    /**
     * Display previous, play/pause, and next if not editing.
     * If editing, display update and cancel.
     */
    const buttons = useMemo(() => {
        return isEditing ?
            (
                isAdding ?
                    (
                        <Stack direction="row" spacing={1}>
                            <Button
                                disabled={loading || !canSubmitMutate}
                                fullWidth
                                onClick={handleAdd}
                                startIcon={<AddIcon />}
                                sx={{ width: 'min(25vw, 150px)' }}
                            >Create</Button>
                            <Button
                                disabled={loading || !canCancelMutate}
                                fullWidth
                                onClick={handleCancelAdd}
                                startIcon={<CancelIcon />}
                                sx={{ width: 'min(25vw, 150px)' }}
                            >Cancel</Button>
                        </Stack>
                    ) :
                    (
                        <Stack direction="row" spacing={1}>
                            <Button
                                disabled={loading || !canSubmitMutate}
                                fullWidth
                                onClick={handleUpdate}
                                startIcon={<UpdateIcon />}
                                sx={{ width: 'min(25vw, 150px)' }}
                            >Update</Button>
                            <Button
                                disabled={loading || !canCancelMutate}
                                fullWidth
                                onClick={handleCancelUpdate}
                                startIcon={<CancelIcon />}
                                sx={{ width: 'min(25vw, 150px)' }}
                            >Cancel</Button>
                        </Stack>
                    )
            ) :
            (
                <Stack direction="row" spacing={0}>
                    {/* <Tooltip title={hasPrevious ? "Previous" : ''} placement="top">
                        <IconButton aria-label="show-previous-routine" size='large' disabled={!hasPrevious} >
                            <PreviousIcon sx={{ fill: hasPrevious ? '#e4efee' : '#a7a7a7' }} />
                        </IconButton>
                    </Tooltip> */}
                    {runState == BuildRunState.Running ? (
                        <Tooltip title="Pause Routine" placement="top">
                            <IconButton aria-label="pause-routine" size='large'>
                                <PauseIcon sx={{ fill: '#e4efee', transform: 'scale(2)' }} />
                            </IconButton>
                        </Tooltip>
                    ) : (
                        <Tooltip title="Run Routine" placement="top">
                            <IconButton aria-label="run-routine" size='large' onClick={runRoutine}>
                                <RunIcon sx={{ fill: '#e4efee', transform: 'scale(2)' }} />
                            </IconButton>
                        </Tooltip>
                    )}
                    {/* <Tooltip title={hasNext ? "Next" : ''} placement="top">
                        <IconButton aria-label="show-next-routine" size='large' disabled={!hasNext}>
                            <NextIcon sx={{ fill: hasPrevious ? '#e4efee' : '#a7a7a7' }} />
                        </IconButton>
                    </Tooltip> */}
                </Stack>
            )
    }, [canCancelMutate, canSubmitMutate, handleAdd, handleCancelAdd, handleCancelUpdate, handleUpdate, hasNext, hasPrevious, isAdding, isEditing, loading, runRoutine, runState]);

    return (
        <Box p={2} sx={{
            alignItems: 'center',
            background: (t) => t.palette.primary.light,
            display: 'flex',
            justifyContent: 'center',
            paddingBottom: { xs: '72px', md: '16px' },
        }}>
            <Dialog
                fullScreen
                id="run-routine-view-dialog"
                onClose={stopRoutine}
                open={isRunOpen}
                TransitionComponent={UpTransition}
            >
                <RunRoutineView
                    handleClose={stopRoutine}
                    session={session}
                />
            </Dialog>
            {slider}
            {buttons}
        </Box>
    )
};