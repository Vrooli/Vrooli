/**
 * Handles selecting a run from a list of runs.
 */
import { DeleteIcon, DeleteOneInput, DeleteType, endpointPostDeleteOne, endpointPostRunProject, endpointPostRunRoutine, parseSearchParams, ProjectVersion, RoutineVersion, RunProject, RunProjectCreateInput, RunRoutine, RunRoutineCreateInput, RunStatus, Success, uuid } from "@local/shared";
import { Button, IconButton, List, ListItem, ListItemText, Menu, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { displayDate } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { base36ToUuid } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { getRunPercentComplete } from "utils/runUtils";
import { SessionContext } from "utils/SessionContext";
import { MenuTitle } from "../MenuTitle/MenuTitle";
import { ListMenuItemData, RunPickerMenuProps } from "../types";

const titleId = "run-picker-dialog-title";

export const RunPickerMenu = ({
    anchorEl,
    handleClose,
    onAdd,
    onDelete,
    onSelect,
    runnableObject,
    zIndex,
}: RunPickerMenuProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const open = useMemo(() => Boolean(anchorEl), [anchorEl]);

    // If runId is in the URL, select that run automatically
    useEffect(() => {
        if (!runnableObject) return;
        const searchParams = parseSearchParams();
        if (!searchParams.run || typeof searchParams.run !== "string") return;
        const runId = base36ToUuid(searchParams.run);
        const run = (runnableObject.you?.runs as (RunRoutine | RunProject)[])?.find((run: RunProject | RunRoutine) => run.id === runId);
        if (run) {
            onSelect(run);
            handleClose();
        }
    }, [runnableObject, onSelect, handleClose]);

    const [createRunProject] = useLazyFetch<RunProjectCreateInput, RunProject>(endpointPostRunProject);
    const [createRunRoutine] = useLazyFetch<RunRoutineCreateInput, RunRoutine>(endpointPostRunRoutine);
    const createNewRun = useCallback(() => {
        if (!runnableObject) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (runnableObject.__typename === "ProjectVersion") {
            fetchLazyWrapper<RunProjectCreateInput, RunProject>({
                fetch: createRunProject,
                inputs: {
                    id: uuid(),
                    name: getTranslation(runnableObject as ProjectVersion, getUserLanguages(session)).name ?? "Unnamed Project",
                    projectVersionConnect: runnableObject.id,
                    status: RunStatus.InProgress,
                },
                successCondition: (data) => data !== null,
                onSuccess: (data) => {
                    onAdd(data);
                    onSelect(data);
                    handleClose();
                },
                errorMessage: () => ({ messageKey: "FailedToCreateRun" }),
            });
        }
        else {
            console.log("creating run routine");
            fetchLazyWrapper<RunRoutineCreateInput, RunRoutine>({
                fetch: createRunRoutine,
                inputs: {
                    id: uuid(),
                    name: getTranslation(runnableObject as RoutineVersion, getUserLanguages(session)).name ?? "Unnamed Routine",
                    routineVersionConnect: runnableObject.id,
                    status: RunStatus.InProgress,
                },
                successCondition: (data) => data !== null,
                onSuccess: (data) => {
                    onAdd(data);
                    onSelect(data);
                    handleClose();
                },
                errorMessage: () => ({ messageKey: "FailedToCreateRun" }),
            });
        }
    }, [handleClose, onAdd, onSelect, runnableObject, createRunProject, createRunRoutine, session]);

    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const deleteRun = useCallback((run: RunProject | RunRoutine) => {
        fetchLazyWrapper<DeleteOneInput, Success>({
            fetch: deleteOne,
            inputs: { id: run.id, objectType: run.__typename as DeleteType },
            successCondition: (data) => data.success,
            successMessage: () => ({ messageKey: "RunDeleted", messageVariables: { runName: displayDate(run.startedAt) } }),
            onSuccess: (data) => {
                onDelete(run);
            },
            errorMessage: () => ({ messageKey: "RunDeleteFailed", messageVariables: { runName: displayDate(run.startedAt) } }),
        });
    }, [deleteOne, onDelete]);

    useEffect(() => {
        if (!open) return;
        // If not logged in, open without creating a new run
        if (session?.isLoggedIn !== true) {
            onSelect(null);
            handleClose();
        }
        // If object has no runs, create a new one.
        else if (runnableObject && (runnableObject.you?.runs as (RunRoutine | RunProject)[])?.filter(r => r.status === RunStatus.InProgress)?.length === 0) {
            createNewRun();
        }
    }, [open, runnableObject, createNewRun, onSelect, session?.isLoggedIn, handleClose]);

    const runOptions: ListMenuItemData<RunProject | RunRoutine>[] = useMemo(() => {
        if (!runnableObject || !runnableObject.you.runs) return [];
        // Find incomplete runs
        const runs = (runnableObject.you?.runs as (RunRoutine | RunProject)[]).filter(run => run.status === RunStatus.InProgress);
        return runs.map((run) => ({
            label: `Started: ${displayDate(run.startedAt)} (${getRunPercentComplete(run.completedComplexity, runnableObject.complexity)}%)`,
            value: run as RunProject | RunRoutine,
        }));
    }, [runnableObject]);

    const handleDelete = useCallback((event: any, run: RunProject | RunRoutine) => {
        // Prevent the click from opening the menu
        event.stopPropagation();
        if (!runnableObject) return;
        // If run has some progress, show confirmation dialog
        if (run.completedComplexity > 0) {
            PubSub.get().publishAlertDialog({
                messageKey: "RunDeleteConfirm",
                messageVariables: { startDate: displayDate(run.startedAt), percentComplete: getRunPercentComplete(run.completedComplexity, runnableObject.complexity) },
                buttons: [
                    { labelKey: "Yes", onClick: () => { deleteRun(run); } },
                    { labelKey: "Cancel" },
                ],
            });
        } else {
            deleteRun(run);
        }
    }, [deleteRun, runnableObject]);

    const items = useMemo(() => runOptions.map((data: ListMenuItemData<RunProject | RunRoutine>, index) => {
        const itemText = <ListItemText primary={data.label} />;
        return (
            <ListItem button onClick={() => { onSelect(data.value); handleClose(); }} key={index}>
                {itemText}
                <Tooltip title="Delete" placement="right">
                    <IconButton edge="end" onClick={(event: any) => handleDelete(event, data.value)}>
                        <DeleteIcon fill={palette.background.textPrimary} />
                    </IconButton>
                </Tooltip>
            </ListItem>
        );
    }), [runOptions, palette.background.textPrimary, onSelect, handleClose, handleDelete]);

    return (
        <Menu
            id='select-run-dialog'
            aria-labelledby={titleId}
            disableScrollLock={true}
            autoFocus={true}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={{
                vertical: "bottom",
                horizontal: "center",
            }}
            transformOrigin={{
                vertical: "top",
                horizontal: "center",
            }}
            onClose={handleClose}
            sx={{
                "& .MuiMenu-paper": {
                    background: palette.background.default,
                },
                "& .MuiMenu-list": {
                    paddingTop: "0",
                },
            }}
        >
            <MenuTitle
                ariaLabel={titleId}
                onClose={handleClose}
                title={"Continue Existing Run?"}
                zIndex={zIndex}
            />
            <List>
                {items}
            </List>
            <Button
                color="secondary"
                onClick={createNewRun}
                sx={{
                    width: "-webkit-fill-available",
                    marginTop: 1,
                    marginBottom: 1,
                    marginLeft: 2,
                    marginRight: 2,
                }}
                variant="contained"
            >New Run</Button>
        </Menu>
    );
};
