/**
 * Handles selecting a run from a list of runs.
 */

import { Button, IconButton, List, ListItem, ListItemText, Menu, Tooltip, useTheme } from "@mui/material";
import { mutationWrapper } from "graphql/utils";
import { useCallback, useEffect, useMemo } from "react";
import { displayDate, getTranslation, getUserLanguages } from "utils/display";
import { ListMenuItemData, RunPickerMenuProps } from "../types";
import { base36ToUuid, getRunPercentComplete, parseSearchParams, PubSub } from "utils";
import { useMutation } from "@apollo/client";
import { runRoutineCreateVariables, runRoutineCreate_runRoutineCreate } from "graphql/generated/runRoutineCreate";
import { deleteOneMutation, runRoutineCreateMutation } from "graphql/mutation";
import { Run } from "types";
import { deleteOneVariables, deleteOne_deleteOne } from "graphql/generated/deleteOne";
import { DeleteType } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { MenuTitle } from "../MenuTitle/MenuTitle";
import { RunStatus } from "graphql/generated/globalTypes";
import { DeleteIcon } from "@shared/icons";
import { SnackSeverity } from "components";

const titleAria = 'run-picker-dialog-title';

export const RunPickerMenu = ({
    anchorEl,
    handleClose,
    onAdd,
    onDelete,
    onSelect,
    routine,
    session
}: RunPickerMenuProps) => {
    const { palette } = useTheme();
    const open = useMemo(() => Boolean(anchorEl), [anchorEl]);

    // If runId is in the URL, select that run automatically
    useEffect(() => {
        if (!routine) return;
        const searchParams = parseSearchParams();
        if (!searchParams.run || typeof searchParams.run !== 'string') return
        const runId = base36ToUuid(searchParams.run);
        const run = routine.runs?.find(run => run.id === runId);
        if (run) {
            onSelect(run);
            handleClose();
        }
    }, [routine, onSelect, handleClose]);

    const [runCreate] = useMutation(runRoutineCreateMutation);
    const createNewRun = useCallback(() => {
        if (!routine) {
            PubSub.get().publishSnack({ messageKey: 'CouldNotReadRoutine', severity: SnackSeverity.Error });
            return;
        }
        mutationWrapper<runRoutineCreate_runRoutineCreate, runRoutineCreateVariables>({
            mutation: runCreate,
            input: {
                // id: uuid(),
                // routineId: routine.id,
                // version: routine.version ?? '',
                // name: getTranslation(routine, getUserLanguages(session)).title ?? 'Unnamed Routine',
            } as any, //TODO will break
            successCondition: (data) => data !== null,
            onSuccess: (data) => {
                onAdd(data);
                onSelect(data);
                handleClose();
            },
            errorMessage: () => ({ key: 'FailedToCreateRun' }),
        })
    }, [handleClose, onAdd, onSelect, routine, runCreate, session]);

    const [deleteOne] = useMutation(deleteOneMutation)
    const deleteRun = useCallback((run: Run) => {
        mutationWrapper<deleteOne_deleteOne, deleteOneVariables>({
            mutation: deleteOne,
            input: { id: run.id, objectType: DeleteType.Run },
            successCondition: (data) => data.success,
            successMessage: () => ({ key: 'RunDeleted', variables: { runName: displayDate(run.timeStarted) } }),
            onSuccess: (data) => {
                onDelete(run);
            },
            errorMessage: () => ({ key: 'RunDeleteFailed', variables: { runName: displayDate(run.timeStarted) } }),
        })
    }, [deleteOne, onDelete])

    useEffect(() => {
        if (!open) return;
        // If not logged in, open routine without creating a new run
        if (!session.isLoggedIn) {
            onSelect(null);
            handleClose();
        }
        // If routine has no runs, create a new one.
        else if (routine && routine.runs?.filter(r => r.status === RunStatus.InProgress)?.length === 0) {
            createNewRun();
        }
    }, [open, routine, createNewRun, onSelect, session.isLoggedIn, handleClose]);

    const runOptions: ListMenuItemData<Run>[] = useMemo(() => {
        if (!routine || !routine.runs) return [];
        // Find incomplete runs
        const runs = routine.runs.filter(run => run.status === RunStatus.InProgress);
        return runs.map((run) => ({
            label: `Started: ${displayDate(run.timeStarted)} (${getRunPercentComplete(run.completedComplexity, routine.complexity)}%)`,
            value: run as Run,
        }));
    }, [routine]);

    const handleDelete = useCallback((event: any, run: Run) => {
        // Prevent the click from opening the menu
        event.stopPropagation();
        if (!routine) return;
        // If run has some progress, show confirmation dialog
        if (run.completedComplexity > 0) {
            PubSub.get().publishAlertDialog({
                messageKey: 'RunDeleteConfirm',
                messageVariables: { startDate: displayDate(run.timeStarted), percentComplete: getRunPercentComplete(run.completedComplexity, routine.complexity) },
                buttons: [
                    { labelKey: 'Yes', onClick: () => { deleteRun(run) } },
                    { labelKey: 'Cancel', onClick: () => { } },
                ]
            });
        } else {
            deleteRun(run);
        }
    }, [deleteRun, routine]);

    const items = useMemo(() => runOptions.map((data: ListMenuItemData<Run>, index) => {
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
        )
    }), [runOptions, palette.background.textPrimary, onSelect, handleClose, handleDelete]);

    return (
        <Menu
            id='select-run-dialog'
            aria-labelledby={titleAria}
            disableScrollLock={true}
            autoFocus={true}
            open={open}
            anchorEl={anchorEl}
            anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'center',
            }}
            transformOrigin={{
                vertical: 'top',
                horizontal: 'center',
            }}
            onClose={handleClose}
            sx={{
                '& .MuiMenu-paper': {
                    background: palette.background.default
                },
                '& .MuiMenu-list': {
                    paddingTop: '0',
                }
            }}
        >
            <MenuTitle
                ariaLabel={titleAria}
                onClose={handleClose}
                title={'Continue Existing Run?'}
            />
            <List>
                {items}
            </List>
            <Button
                color="secondary"
                onClick={createNewRun}
                sx={{
                    width: '-webkit-fill-available',
                    marginTop: 1,
                    marginBottom: 1,
                    marginLeft: 2,
                    marginRight: 2,
                }}
            >New Run</Button>
        </Menu>
    );
}