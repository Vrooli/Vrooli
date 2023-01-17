import { Box, Dialog, Tooltip, useTheme } from "@mui/material";
import { GqlModelType, ProjectVersion, RoutineVersion, RunProject, RunRoutine } from "@shared/consts";
import { PlayIcon } from "@shared/icons";
import { useLocation } from "@shared/route";
import { uuidValidate } from "@shared/uuid";
import { PopoverWithArrow, RunPickerMenu, UpTransition } from "components/dialogs";
import { RunView } from "components/views";
import { useCallback, useMemo, useState } from "react";
import { getProjectVersionStatus, getRoutineVersionStatus, parseSearchParams, PubSub, setSearchParams, Status, uuidToBase36 } from "utils";
import { ColorIconButton } from "../ColorIconButton/ColorIconButton";
import { RunButtonProps } from "../types";

/**
 * Button to run a multi-step routine. 
 * If the routine is invalid, it is greyed out with a tooltip on hover or press. 
 * If the routine is incomplete, the button is available but the user must confirm an alert before running.
 */
export const RunButton = ({
    canEdit,
    handleRunAdd,
    handleRunDelete,
    isBuildGraphOpen,
    isEditing,
    runnableObject,
    session,
    zIndex,
}: RunButtonProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    // Check object status to see if it is valid and complete
    const status = useMemo<Status>(() => {
        if (!runnableObject) return Status.Invalid;
        return (runnableObject.__typename === GqlModelType.ProjectVersion ? 
            getProjectVersionStatus(runnableObject as ProjectVersion) : 
            getRoutineVersionStatus(runnableObject as RoutineVersion)).status;
    }, [runnableObject]);

    const [isRunOpen, setIsRunOpen] = useState(() => {
        const params = parseSearchParams();
        return typeof params.run === 'string' && uuidValidate(params.run);
    });
    const [selectRunAnchor, setSelectRunAnchor] = useState<any>(null);
    const handleRunSelect = useCallback((run: RunProject | RunRoutine | null) => {
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

    const startRun = useCallback((e: any) => {
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

    // Invalid message popup
    const [errorAnchorEl, setErrorAnchorEl] = useState<any | null>(null);
    const openError = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        ev.preventDefault();
        setErrorAnchorEl(ev.currentTarget ?? ev.target)
    }, []);
    const closeError = useCallback(() => {
        setErrorAnchorEl(null);
    }, []);

    const runStart = useCallback((e: any) => {
        // If invalid, don't run
        if (status === Status.Invalid) {
            openError(e);
            return;
        }
        // If incomplete, confirm user wants to run
        else if (status === Status.Incomplete) {
            PubSub.get().publishAlertDialog({
                messageKey: 'RunInvalidRoutineConfirm',
                buttons: [
                    { labelKey: 'Yes', onClick: () => { startRun(e) } },
                    { labelKey: 'Cancel', onClick: () => { } },
                ]
            });
        }
        // Otherwise, run
        else {
            startRun(e);
        }
    }, [openError, startRun, status]);

    const runStop = () => {
        setLocation(window.location.pathname, { replace: true });
        setIsRunOpen(false)
    };

    return (
        <>
            {/* Invalid routine popup */}
            <PopoverWithArrow
                anchorEl={errorAnchorEl}
                handleClose={closeError}
            >
                Routine cannot be run because it is invalid.
            </PopoverWithArrow>
            {/* Run dialog */}
            <Dialog
                fullScreen
                id="run-routine-view-dialog"
                onClose={runStop}
                open={isRunOpen}
                TransitionComponent={UpTransition}
                sx={{
                    zIndex: zIndex + 3,
                }}
            >
                {runnableObject && <RunView
                    handleClose={runStop}
                    runnableObject={runnableObject}
                    session={session}
                    zIndex={zIndex + 3}
                />}
            </Dialog>
            {/* Chooses which run to use */}
            <RunPickerMenu
                anchorEl={selectRunAnchor}
                handleClose={handleSelectRunClose}
                onAdd={handleRunAdd}
                onDelete={handleRunDelete}
                onSelect={handleRunSelect}
                runnableObject={runnableObject}
                session={session}
            />
            {/* Run button */}
            <Tooltip title="Run Routine" placement="top">
                {/* Button wrapped in div so it can be pressed when disabled */}
                <Box onClick={runStart}>
                    <ColorIconButton
                        aria-label="run-routine"
                        disabled={status === Status.Invalid}
                        background={palette.secondary.main}
                        sx={{
                            padding: 0,
                            width: '54px',
                            height: '54px',
                        }}
                    >
                        <PlayIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                </Box>
            </Tooltip>
        </>
    )
}