import { Box, Button, Grid, IconButton, Slider, Stack, Tooltip } from '@mui/material';
import { useMemo } from 'react';
import {
    Cancel as CancelIcon,
    Pause as PauseIcon,
    PlayCircle as RunIcon,
    SkipNext as NextIcon,
    SkipPrevious as PreviousIcon,
    Update as UpdateIcon
} from '@mui/icons-material';
import { withStyles } from '@mui/styles';
import { OrchestrationRunState } from 'utils';
import { OrchestrationBottomContainerProps } from '../types';

const CustomSlider = withStyles({
    root: {
        width: 'calc(100% - 32px)',
        left: '11px', // 16px - 1/4 of thumb diameter
    },
})(Slider);

export const OrchestrationBottomContainer = ({
    canUpdate,
    canCancelUpdate,
    handleCancelRoutineUpdate,
    handleRoutineUpdate,
    handleScaleChange,
    hasPrevious,
    hasNext,
    isEditing,
    loading,
    scale,
    sliderColor,
    runState,
}: OrchestrationBottomContainerProps) => {

    const onScaleChange = (_event: any, newScale: number | number[]) => {
        handleScaleChange(newScale as number);
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
                <Stack direction="row" spacing={1}>
                    <Button
                        fullWidth
                        startIcon={<UpdateIcon />}
                        onClick={handleRoutineUpdate}
                        disabled={loading || !canUpdate}
                        sx={{ width: 'min(25vw, 150px)' }}
                    >Update</Button>
                    <Button
                        fullWidth
                        startIcon={<CancelIcon />}
                        onClick={handleCancelRoutineUpdate}
                        disabled={loading || !canCancelUpdate}
                        sx={{ width: 'min(25vw, 150px)' }}
                    >Cancel</Button>
                </Stack>
            ) :
            (
                <Stack direction="row" spacing={0}>
                    <Tooltip title={hasPrevious ? "Previous" : ''} placement="top">
                        <IconButton aria-label="show-previous-routine" size='large' disabled={!hasPrevious} >
                            <PreviousIcon sx={{ fill: hasPrevious ? '#e4efee' : '#a7a7a7' }} />
                        </IconButton>
                    </Tooltip>
                    {runState == OrchestrationRunState.Running ? (
                        <Tooltip title="Pause Routine" placement="top">
                            <IconButton aria-label="pause-routine" size='large'>
                                <PauseIcon sx={{ fill: '#e4efee', transform: 'scale(2)' }} />
                            </IconButton>
                        </Tooltip>
                    ) : (
                        <Tooltip title="Run Routine" placement="top">
                            <IconButton aria-label="run-routine" size='large'>
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
    }, [hasPrevious, hasNext, isEditing, canUpdate, canCancelUpdate, handleCancelRoutineUpdate, handleRoutineUpdate, loading, runState]);

    return (
        <Box p={2} sx={{
            background: (t) => t.palette.primary.light,
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            paddingBottom: { xs: '72px', md: '16px' },
        }}>
            {slider}
            {buttons}
        </Box>
    )
};