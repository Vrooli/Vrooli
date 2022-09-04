import { Box, Dialog, Grid, IconButton, Slider, Stack, Tooltip, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import {
    Pause as PauseIcon,
    PlayCircle as RunIcon,
} from '@mui/icons-material';
import { BuildRunState, setSearchParams } from 'utils';
import { BuildBottomContainerProps } from '../types';
import { useLocation } from '@shared/route';
import { RunPickerMenu, UpTransition } from 'components/dialogs';
import { RunView } from 'components/views';
import { Run } from 'types';
import { GridSubmitButtons } from 'components/buttons';

export const BuildBottomContainer = ({
    canSubmitMutate,
    canCancelMutate,
    handleCancelAdd,
    handleCancelUpdate,
    handleAdd,
    handleUpdate,
    handleRunDelete,
    handleRunAdd,
    handleScaleChange,
    hasPrevious,
    hasNext,
    isAdding,
    isEditing,
    loading,
    scale,
    session,
    sliderColor,
    routine,
    runState,
    zIndex,
}: BuildBottomContainerProps) => {
    console.log('build bottom container', zIndex);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const onScaleChange = useCallback((_event: any, newScale: number | number[]) => {
        handleScaleChange(newScale as number);
    }, [handleScaleChange]);

    const [isRunOpen, setIsRunOpen] = useState(false)
    const [selectRunAnchor, setSelectRunAnchor] = useState<any>(null);
    const handleRunSelect = useCallback((run: Run | null) => {
        // If run is null, it means the routine will be opened without a run
        if (!run) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1]
            });
        }
        // Otherwise, open routine where last left off in run
        else {
            setSearchParams(setLocation, {
                run: run.id,
                step: run.steps.length > 0 ? run.steps[run.steps.length - 1].step : undefined,
            });
        }
        setIsRunOpen(true);
    }, [setLocation]);
    const handleSelectRunClose = useCallback(() => setSelectRunAnchor(null), []);

    const runRoutine = useCallback((e: any) => {
        // If editing, don't use a real run
        if (isEditing) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1]
            });
            setIsRunOpen(true);
        }
        else {
            setSelectRunAnchor(e.currentTarget);
        }
    }, [isEditing, setLocation]);
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
                <Grid container spacing={1} sx={{ width: 'min(100vw, 350px)' }}>
                    <GridSubmitButtons
                        disabledCancel={loading || !canCancelMutate}
                        disabledSubmit={loading || !canSubmitMutate}
                        isCreate={isAdding}
                        onCancel={handleCancelAdd}
                        onSubmit={handleAdd}
                    />
                </Grid>
            ) :
            (
                <Stack direction="row" spacing={0}>
                    {/* <Tooltip title={hasPrevious ? "Previous" : ''} placement="top">
                        <IconButton aria-label="show-previous-routine" size='large' disabled={!hasPrevious} >
                            <PreviousIcon sx={{ fill: hasPrevious ? '#e4efee' : '#a7a7a7' }} />
                        </IconButton>
                    </Tooltip> */}
                    {runState === BuildRunState.Running ? (
                        <Tooltip title="Pause Routine" placement="top">
                            <IconButton aria-label="pause-routine" size='large' sx={{ padding: 0, width: '48px', height: '48px' }}>
                                <PauseIcon sx={{ fill: '#e4efee', marginBottom: 'auto', width: '48px', height: '48px' }} />
                            </IconButton>
                        </Tooltip>
                    ) : (
                        <Tooltip title="Run Routine" placement="top">
                            <IconButton aria-label="run-routine" size='large' onClick={runRoutine} sx={{ padding: 0, width: '48px', height: '48px' }}>
                                <RunIcon sx={{ fill: '#e4efee', marginBottom: 'auto', width: '48px', height: '48px' }} />
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
    }, [canCancelMutate, canSubmitMutate, handleAdd, handleCancelAdd, isAdding, isEditing, loading, runRoutine, runState]);

    return (
        <Box sx={{
            alignItems: 'center',
            background: palette.primary.dark,
            display: 'flex',
            justifyContent: 'center',
            bottom: 0,
            paddingTop: 1,
            paddingLeft: 4,
            paddingRight: 4,
            paddingBottom: 'calc(8px + env(safe-area-inset-bottom))',
            // safe-area-inset-bottom is the iOS navigation bar
            height: 'calc(64px + env(safe-area-inset-bottom))',
        }}>
            {/* Chooses which run to use */}
            <RunPickerMenu
                anchorEl={selectRunAnchor}
                handleClose={handleSelectRunClose}
                onAdd={handleRunAdd}
                onDelete={handleRunDelete}
                onSelect={handleRunSelect}
                routine={routine}
                session={session}
            />
            <Dialog
                fullScreen
                id="run-routine-view-dialog"
                onClose={stopRoutine}
                open={isRunOpen}
                TransitionComponent={UpTransition}
                sx={{
                    zIndex: zIndex + 1,
                }}
            >
                {routine && <RunView
                    handleClose={stopRoutine}
                    routine={routine}
                    session={session}
                    zIndex={zIndex + 1}
                />}
            </Dialog>
            {slider}
            {buttons}
        </Box>
    )
};