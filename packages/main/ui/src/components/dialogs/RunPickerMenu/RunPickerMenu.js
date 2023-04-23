import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { RunStatus } from "@local/consts";
import { DeleteIcon } from "@local/icons";
import { uuid } from "@local/uuid";
import { Button, IconButton, List, ListItem, ListItemText, Menu, Tooltip, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { deleteOneOrManyDeleteOne } from "../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { runProjectCreate } from "../../../api/generated/endpoints/runProject_create";
import { runRoutineCreate } from "../../../api/generated/endpoints/runRoutine_create";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { displayDate } from "../../../utils/display/stringTools";
import { getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { base36ToUuid } from "../../../utils/navigation/urlTools";
import { PubSub } from "../../../utils/pubsub";
import { parseSearchParams } from "../../../utils/route";
import { getRunPercentComplete } from "../../../utils/runUtils";
import { SessionContext } from "../../../utils/SessionContext";
import { MenuTitle } from "../MenuTitle/MenuTitle";
const titleId = "run-picker-dialog-title";
export const RunPickerMenu = ({ anchorEl, handleClose, onAdd, onDelete, onSelect, runnableObject, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const open = useMemo(() => Boolean(anchorEl), [anchorEl]);
    useEffect(() => {
        if (!runnableObject)
            return;
        const searchParams = parseSearchParams();
        if (!searchParams.run || typeof searchParams.run !== "string")
            return;
        const runId = base36ToUuid(searchParams.run);
        const run = runnableObject.you?.runs?.find((run) => run.id === runId);
        if (run) {
            onSelect(run);
            handleClose();
        }
    }, [runnableObject, onSelect, handleClose]);
    const [createRunProject] = useCustomMutation(runProjectCreate);
    const [createRunRoutine] = useCustomMutation(runRoutineCreate);
    const createNewRun = useCallback(() => {
        if (!runnableObject) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (runnableObject.__typename === "ProjectVersion") {
            mutationWrapper({
                mutation: createRunProject,
                input: {
                    id: uuid(),
                    name: getTranslation(runnableObject, getUserLanguages(session)).name ?? "Unnamed Project",
                    projectVersionConnect: runnableObject.id,
                    status: RunStatus.InProgress,
                },
                successCondition: (data) => data !== null,
                onSuccess: (data) => {
                    onAdd(data);
                    onSelect(data);
                    handleClose();
                },
                errorMessage: () => ({ key: "FailedToCreateRun" }),
            });
        }
        else {
            mutationWrapper({
                mutation: createRunRoutine,
                input: {
                    id: uuid(),
                    name: getTranslation(runnableObject, getUserLanguages(session)).name ?? "Unnamed Routine",
                    routineVersionConnect: runnableObject.id,
                    status: RunStatus.InProgress,
                },
                successCondition: (data) => data !== null,
                onSuccess: (data) => {
                    onAdd(data);
                    onSelect(data);
                    handleClose();
                },
                errorMessage: () => ({ key: "FailedToCreateRun" }),
            });
        }
    }, [handleClose, onAdd, onSelect, runnableObject, createRunProject, createRunRoutine, session]);
    const [deleteOne] = useCustomMutation(deleteOneOrManyDeleteOne);
    const deleteRun = useCallback((run) => {
        mutationWrapper({
            mutation: deleteOne,
            input: { id: run.id, objectType: run.__typename },
            successCondition: (data) => data.success,
            successMessage: () => ({ key: "RunDeleted", variables: { runName: displayDate(run.startedAt) } }),
            onSuccess: (data) => {
                onDelete(run);
            },
            errorMessage: () => ({ key: "RunDeleteFailed", variables: { runName: displayDate(run.startedAt) } }),
        });
    }, [deleteOne, onDelete]);
    useEffect(() => {
        if (!open)
            return;
        if (session?.isLoggedIn !== true) {
            onSelect(null);
            handleClose();
        }
        else if (runnableObject && runnableObject.you?.runs?.filter(r => r.status === RunStatus.InProgress)?.length === 0) {
            createNewRun();
        }
    }, [open, runnableObject, createNewRun, onSelect, session?.isLoggedIn, handleClose]);
    const runOptions = useMemo(() => {
        if (!runnableObject || !runnableObject.you.runs)
            return [];
        const runs = (runnableObject.you?.runs).filter(run => run.status === RunStatus.InProgress);
        return runs.map((run) => ({
            label: `Started: ${displayDate(run.startedAt)} (${getRunPercentComplete(run.completedComplexity, runnableObject.complexity)}%)`,
            value: run,
        }));
    }, [runnableObject]);
    const handleDelete = useCallback((event, run) => {
        event.stopPropagation();
        if (!runnableObject)
            return;
        if (run.completedComplexity > 0) {
            PubSub.get().publishAlertDialog({
                messageKey: "RunDeleteConfirm",
                messageVariables: { startDate: displayDate(run.startedAt), percentComplete: getRunPercentComplete(run.completedComplexity, runnableObject.complexity) },
                buttons: [
                    { labelKey: "Yes", onClick: () => { deleteRun(run); } },
                    { labelKey: "Cancel", onClick: () => { } },
                ],
            });
        }
        else {
            deleteRun(run);
        }
    }, [deleteRun, runnableObject]);
    const items = useMemo(() => runOptions.map((data, index) => {
        const itemText = _jsx(ListItemText, { primary: data.label });
        return (_jsxs(ListItem, { button: true, onClick: () => { onSelect(data.value); handleClose(); }, children: [itemText, _jsx(Tooltip, { title: "Delete", placement: "right", children: _jsx(IconButton, { edge: "end", onClick: (event) => handleDelete(event, data.value), children: _jsx(DeleteIcon, { fill: palette.background.textPrimary }) }) })] }, index));
    }), [runOptions, palette.background.textPrimary, onSelect, handleClose, handleDelete]);
    return (_jsxs(Menu, { id: 'select-run-dialog', "aria-labelledby": titleId, disableScrollLock: true, autoFocus: true, open: open, anchorEl: anchorEl, anchorOrigin: {
            vertical: "bottom",
            horizontal: "center",
        }, transformOrigin: {
            vertical: "top",
            horizontal: "center",
        }, onClose: handleClose, sx: {
            "& .MuiMenu-paper": {
                background: palette.background.default,
            },
            "& .MuiMenu-list": {
                paddingTop: "0",
            },
        }, children: [_jsx(MenuTitle, { ariaLabel: titleId, onClose: handleClose, title: "Continue Existing Run?" }), _jsx(List, { children: items }), _jsx(Button, { color: "secondary", onClick: createNewRun, sx: {
                    width: "-webkit-fill-available",
                    marginTop: 1,
                    marginBottom: 1,
                    marginLeft: 2,
                    marginRight: 2,
                }, children: "New Run" })] }));
};
//# sourceMappingURL=RunPickerMenu.js.map