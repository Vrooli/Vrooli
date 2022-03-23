import { Box, Button, Dialog, IconButton, Slider, Stack, Tooltip } from '@mui/material';
import { useMemo, useState } from 'react';
import {
    Add as AddIcon,
    Cancel as CancelIcon,
    Pause as PauseIcon,
    PlayCircle as RunIcon,
    SkipNext as NextIcon,
    SkipPrevious as PreviousIcon,
    Update as UpdateIcon
} from '@mui/icons-material';
import { withStyles } from '@mui/styles';
import { BuildRunState } from 'utils';
import { BuildBottomContainerProps } from '../types';
import { useLocation } from 'wouter';
import { UpTransition } from 'components/dialogs';
import { RunRoutineView } from 'components/views';

const CustomSlider = withStyles({
    root: {
        width: 'calc(100% - 32px)',
        left: '11px', // 16px - 1/4 of thumb diameter
    },
})(Slider);

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
        <CustomSlider
            aria-label="graph-scale"
            min={0.25}
            max={1}
            defaultValue={1}
            step={0.01}
            value={scale}
            valueLabelDisplay="auto"
            onChange={onScaleChange}
            sx={{ color: sliderColor, maxWidth: '500px', marginRight: 4 }}
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
                                fullWidth
                                startIcon={<AddIcon />}
                                onClick={handleAdd}
                                disabled={loading || !canSubmitMutate}
                                sx={{ width: 'min(25vw, 150px)' }}
                            >Create</Button>
                            <Button
                                fullWidth
                                startIcon={<CancelIcon />}
                                onClick={handleCancelAdd}
                                disabled={loading || !canCancelMutate}
                                sx={{ width: 'min(25vw, 150px)' }}
                            >Cancel</Button>
                        </Stack>
                    ) :
                    (
                        <Stack direction="row" spacing={1}>
                            <Button
                                fullWidth
                                startIcon={<UpdateIcon />}
                                onClick={handleUpdate}
                                disabled={loading || !canSubmitMutate}
                                sx={{ width: 'min(25vw, 150px)' }}
                            >Update</Button>
                            <Button
                                fullWidth
                                startIcon={<CancelIcon />}
                                onClick={handleCancelUpdate}
                                disabled={loading || !canCancelMutate}
                                sx={{ width: 'min(25vw, 150px)' }}
                            >Cancel</Button>
                        </Stack>
                    )
            ) :
            (
                <Stack direction="row" spacing={0}>
                    <Tooltip title={hasPrevious ? "Previous" : ''} placement="top">
                        <IconButton aria-label="show-previous-routine" size='large' disabled={!hasPrevious} >
                            <PreviousIcon sx={{ fill: hasPrevious ? '#e4efee' : '#a7a7a7' }} />
                        </IconButton>
                    </Tooltip>
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
                    <Tooltip title={hasNext ? "Next" : ''} placement="top">
                        <IconButton aria-label="show-next-routine" size='large' disabled={!hasNext}>
                            <NextIcon sx={{ fill: hasPrevious ? '#e4efee' : '#a7a7a7' }} />
                        </IconButton>
                    </Tooltip>
                </Stack>
            )
    }, [hasPrevious, hasNext, isEditing, isAdding, loading, canSubmitMutate, canCancelMutate, handleAdd, handleUpdate, handleCancelAdd, handleCancelUpdate, runState, runRoutine]);

    return (
        <Box p={2} sx={{
            background: (t) => t.palette.primary.light,
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            paddingBottom: { xs: '72px', md: '16px' },
        }}>
            <Dialog
                id="run-routine-view-dialog"
                fullScreen
                open={isRunOpen}
                onClose={stopRoutine}
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