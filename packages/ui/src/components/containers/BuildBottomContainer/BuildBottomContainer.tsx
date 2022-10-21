import { Box, Dialog, Grid, Palette, Slider, Stack, Tooltip, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { BuildRunState, setSearchParams, uuidToBase36 } from 'utils';
import { BuildBottomContainerProps } from '../types';
import { useLocation } from '@shared/route';
import { RunPickerMenu, UpTransition } from 'components/dialogs';
import { RunView } from 'components/views';
import { Run } from 'types';
import { ColorIconButton, GridSubmitButtons } from 'components/buttons';
import { PauseIcon, PlayIcon } from '@shared/icons';

const iconButtonProps = {
    padding: 0, 
    width: '48px', 
    height: '48px',
}

const iconProps = (palette: Palette) => ({
    fill: palette.primary.dark,
    width: '30px',
    height: '30px',
})

export const BuildBottomContainer = ({
    canSubmitMutate,
    canCancelMutate,
    errors,
    handleCancel,
    handleSubmit,
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
                run: uuidToBase36(run.id),
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
                <Grid container spacing={1} sx={{
                    display: { xs: 'contents', md: 'flex' },
                    width: { xs: '100%', md: '300px' },
                }}>
                    <GridSubmitButtons
                        disabledCancel={loading || !canCancelMutate}
                        disabledSubmit={loading || !canSubmitMutate}
                        errors={errors}
                        isCreate={isAdding}
                        onCancel={handleCancel}
                        onSubmit={handleSubmit}
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
                            <ColorIconButton aria-label="pause-routine" background='#e4efee' sx={iconButtonProps}>
                                <PauseIcon {...iconProps(palette)} />
                            </ColorIconButton>
                        </Tooltip>
                    ) : (
                        <Tooltip title="Run Routine" placement="top">
                            <ColorIconButton aria-label="run-routine" onClick={runRoutine} background='#e4efee' sx={iconButtonProps}>
                                <PlayIcon {...iconProps(palette)} />
                            </ColorIconButton>
                        </Tooltip>
                    )}
                    {/* <Tooltip title={hasNext ? "Next" : ''} placement="top">
                        <IconButton aria-label="show-next-routine" size='large' disabled={!hasNext}>
                            <NextIcon sx={{ fill: hasPrevious ? '#e4efee' : '#a7a7a7' }} />
                        </IconButton>
                    </Tooltip> */}
                </Stack>
            )
    }, [canCancelMutate, canSubmitMutate, errors, handleCancel, handleSubmit, isAdding, isEditing, loading, palette, runRoutine, runState]);

    return (
        <Box sx={{
            alignItems: 'center',
            background: palette.primary.dark,
            display: 'flex',
            justifyContent: 'center',
            bottom: 0,
            paddingTop: 1,
            paddingLeft: { xs: 1, sm: 4 },
            paddingRight: { xs: 1, sm: 4 },
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