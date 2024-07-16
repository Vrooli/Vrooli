import { GqlModelType, ProjectVersion, RoutineVersion, RunProject, RunRoutine, parseSearchParams, uuidToBase36, uuidValidate } from "@local/shared";
import { Box, Tooltip, useTheme } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { RunPickerMenu } from "components/dialogs/RunPickerMenu/RunPickerMenu";
import { usePopover } from "hooks/usePopover";
import { PlayIcon } from "icons";
import React, { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { setSearchParams, useLocation } from "route";
import { SideActionsButton } from "styles";
import { Status } from "utils/consts";
import { PubSub } from "utils/pubsub";
import { getProjectVersionStatus, getRoutineVersionStatus } from "utils/runUtils";
import { RunView } from "views/runs";
import { RunButtonProps } from "../types";

/**
 * Button to run a multi-step routine. 
 * If the routine is invalid, it is greyed out with a tooltip on hover or press. 
 * If the routine is incomplete, the button is available but the user must confirm an alert before running.
 */
export function RunButton({
    canUpdate,
    handleRunAdd,
    handleRunDelete,
    isBuildGraphOpen,
    isEditing,
    runnableObject,
}: RunButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
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
        return typeof params.run === "string" && uuidValidate(params.run);
    });
    const [selectRunAnchor, openSelectRunDialog, closeSelectRunDialog] = usePopover();
    const handleRunSelect = useCallback((run: RunProject | RunRoutine | null) => {
        // If run is null, it means the routine will be opened without a run
        if (!run) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1],
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

    const startRun = useCallback((event: React.MouseEvent<HTMLElement>) => {
        // If editing, don't use a real run
        if (isEditing) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1],
            });
            setIsRunOpen(true);
        }
        else {
            openSelectRunDialog(event);
        }
    }, [isEditing, openSelectRunDialog, setLocation]);

    // Invalid message popup
    const [errorAnchorEl, openErrorPopup, closeErrorPopup] = usePopover();
    const openError = useCallback((ev: React.MouseEvent<HTMLElement>) => {
        ev.preventDefault();
        openErrorPopup(ev);
    }, [openErrorPopup]);

    const runStart = useCallback((event: React.MouseEvent<HTMLElement>) => {
        // If invalid, don't run
        if (status === Status.Invalid) {
            openError(event);
            return;
        }
        // If incomplete, confirm user wants to run
        else if (status === Status.Incomplete) {
            PubSub.get().publish("alertDialog", {
                messageKey: "RunInvalidRoutineConfirm",
                buttons: [
                    { labelKey: "Yes", onClick: () => { startRun(event); } },
                    { labelKey: "Cancel" },
                ],
            });
        }
        // Otherwise, run
        else {
            startRun(event);
        }
    }, [openError, startRun, status]);

    const runStop = useCallback(function runStopCallback() {
        setLocation(window.location.pathname, { replace: true });
        setIsRunOpen(false);
    }, [setLocation]);

    return (
        <>
            {/* Invalid routine popup */}
            <PopoverWithArrow
                anchorEl={errorAnchorEl}
                handleClose={closeErrorPopup}
            >{t("RoutineCannotRunInvalid", { ns: "error" })}</PopoverWithArrow>
            {/* Run dialog */}
            {runnableObject && <RunView
                display="dialog"
                isOpen={isRunOpen}
                onClose={runStop}
                runnableObject={runnableObject}
            />}
            {/* Chooses which run to use */}
            <RunPickerMenu
                anchorEl={selectRunAnchor}
                handleClose={closeSelectRunDialog}
                onAdd={handleRunAdd}
                onDelete={handleRunDelete}
                onSelect={handleRunSelect}
                runnableObject={runnableObject}
            />
            {/* Run button */}
            <Tooltip title="Run Routine" placement="top">
                {/* Button wrapped in div so it can be pressed when disabled */}
                <Box onClick={runStart}>
                    <SideActionsButton
                        aria-label="run-routine"
                        disabled={status === Status.Invalid}
                    >
                        <PlayIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </SideActionsButton>
                </Box>
            </Tooltip>
        </>
    );
}
