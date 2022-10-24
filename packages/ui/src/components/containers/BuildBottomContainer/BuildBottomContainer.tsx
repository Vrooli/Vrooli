import { Box, Dialog, Grid, Palette, Stack, Tooltip, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { BuildRunState, setSearchParams, uuidToBase36 } from 'utils';
import { BuildBottomContainerProps } from '../types';
import { useLocation } from '@shared/route';
import { RunPickerMenu, UpTransition } from 'components/dialogs';
import { RunView } from 'components/views';
import { Run } from 'types';
import { ColorIconButton, GridSubmitButtons } from 'components/buttons';
import { PauseIcon, PlayIcon } from '@shared/icons';

const iconButtonProps = (palette: Palette) => ({
    padding: 0, 
    width: '54px', 
    height: '54px',
})

const iconProps = (palette: Palette) => ({
    fill: palette.secondary.contrastText,
    width: '36px',
    height: '36px',
})

export const BuildBottomContainer = ({
    canSubmitMutate,
    canCancelMutate,
    errors,
    handleCancel,
    handleSubmit,
    handleRunDelete,
    handleRunAdd,
    hasPrevious,
    hasNext,
    isAdding,
    isEditing,
    loading,
    session,
    sliderColor,
    routine,
    runState,
    zIndex,
}: BuildBottomContainerProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

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
     * Display previous, play/pause, and next if not editing.
     * If editing, display update and cancel.
     */
    const buttons = useMemo(() => {
        return isEditing ?
            (
                <Grid container spacing={1} sx={{ width: 'min(100%, 600px)' }}>
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
                            <ColorIconButton aria-label="pause-routine" background={palette.secondary.main} sx={iconButtonProps(palette)}>
                                <PauseIcon {...iconProps(palette)} />
                            </ColorIconButton>
                        </Tooltip>
                    ) : (
                        <Tooltip title="Run Routine" placement="top">
                            <ColorIconButton aria-label="run-routine" onClick={runRoutine} background={palette.secondary.main} sx={iconButtonProps(palette)}>
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
            background: 'transparent',
            display: 'flex',
            justifyContent: 'center',
            position: 'absolute',
            zIndex: 2,
            bottom: 0,
            right: 0,
            paddingBottom: 'calc(16px + env(safe-area-inset-bottom))',
            paddingLeft: 'calc(16px + env(safe-area-inset-left))',
            paddingRight: 'calc(16px + env(safe-area-inset-right))',
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
            {buttons}
        </Box>
    )
};