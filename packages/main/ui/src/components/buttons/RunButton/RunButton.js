import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { GqlModelType } from "@local/consts";
import { PlayIcon } from "@local/icons";
import { uuidValidate } from "@local/uuid";
import { Box, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { Status } from "../../../utils/consts";
import { uuidToBase36 } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { parseSearchParams, setSearchParams, useLocation } from "../../../utils/route";
import { getProjectVersionStatus, getRoutineVersionStatus } from "../../../utils/runUtils";
import { RunView } from "../../../views/runs";
import { LargeDialog } from "../../dialogs/LargeDialog/LargeDialog";
import { PopoverWithArrow } from "../../dialogs/PopoverWithArrow/PopoverWithArrow";
import { RunPickerMenu } from "../../dialogs/RunPickerMenu/RunPickerMenu";
import { ColorIconButton } from "../ColorIconButton/ColorIconButton";
export const RunButton = ({ canUpdate, handleRunAdd, handleRunDelete, isBuildGraphOpen, isEditing, runnableObject, zIndex, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const status = useMemo(() => {
        if (!runnableObject)
            return Status.Invalid;
        return (runnableObject.__typename === GqlModelType.ProjectVersion ?
            getProjectVersionStatus(runnableObject) :
            getRoutineVersionStatus(runnableObject)).status;
    }, [runnableObject]);
    const [isRunOpen, setIsRunOpen] = useState(() => {
        const params = parseSearchParams();
        return typeof params.run === "string" && uuidValidate(params.run);
    });
    const [selectRunAnchor, setSelectRunAnchor] = useState(null);
    const handleRunSelect = useCallback((run) => {
        if (!run) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1],
            });
        }
        else {
            setSearchParams(setLocation, {
                run: uuidToBase36(run.id),
                step: run.steps.length > 0 ? run.steps[run.steps.length - 1].step : undefined,
            });
        }
        setIsRunOpen(true);
    }, [setLocation]);
    const handleSelectRunClose = useCallback(() => setSelectRunAnchor(null), []);
    const startRun = useCallback((e) => {
        if (isEditing) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1],
            });
            setIsRunOpen(true);
        }
        else {
            setSelectRunAnchor(e.currentTarget);
        }
    }, [isEditing, setLocation]);
    const [errorAnchorEl, setErrorAnchorEl] = useState(null);
    const openError = useCallback((ev) => {
        ev.preventDefault();
        setErrorAnchorEl(ev.currentTarget ?? ev.target);
    }, []);
    const closeError = useCallback(() => {
        setErrorAnchorEl(null);
    }, []);
    const runStart = useCallback((e) => {
        if (status === Status.Invalid) {
            openError(e);
            return;
        }
        else if (status === Status.Incomplete) {
            PubSub.get().publishAlertDialog({
                messageKey: "RunInvalidRoutineConfirm",
                buttons: [
                    { labelKey: "Yes", onClick: () => { startRun(e); } },
                    { labelKey: "Cancel", onClick: () => { } },
                ],
            });
        }
        else {
            startRun(e);
        }
    }, [openError, startRun, status]);
    const runStop = () => {
        setLocation(window.location.pathname, { replace: true });
        setIsRunOpen(false);
    };
    return (_jsxs(_Fragment, { children: [_jsx(PopoverWithArrow, { anchorEl: errorAnchorEl, handleClose: closeError, children: "Routine cannot be run because it is invalid." }), _jsx(LargeDialog, { id: "run-routine-view-dialog", onClose: runStop, isOpen: isRunOpen, titleId: "", zIndex: zIndex + 3, children: runnableObject && _jsx(RunView, { handleClose: runStop, runnableObject: runnableObject, zIndex: zIndex + 3 }) }), _jsx(RunPickerMenu, { anchorEl: selectRunAnchor, handleClose: handleSelectRunClose, onAdd: handleRunAdd, onDelete: handleRunDelete, onSelect: handleRunSelect, runnableObject: runnableObject }), _jsx(Tooltip, { title: "Run Routine", placement: "top", children: _jsx(Box, { onClick: runStart, children: _jsx(ColorIconButton, { "aria-label": "run-routine", disabled: status === Status.Invalid, background: palette.secondary.main, sx: {
                            padding: 0,
                            width: "54px",
                            height: "54px",
                        }, children: _jsx(PlayIcon, { fill: palette.secondary.contrastText, width: '36px', height: '36px' }) }) }) })] }));
};
//# sourceMappingURL=RunButton.js.map