/**
 * Handles selecting a run from a list of runs.
 */

import { Button, IconButton, List, ListItem, ListItemText, Menu, Tooltip, useTheme } from "@mui/material";
import { mutationWrapper } from "graphql/utils";
import { useCallback, useEffect, useMemo } from "react";
import { displayDate, getTranslation, getUserLanguages } from "utils/display";
import { ListMenuItemData, RunPickerMenuProps } from "../types";
import { getRunPercentComplete, parseSearchParams, PubSub } from "utils";
import { useMutation } from "@apollo/client";
import { runCreate, runCreateVariables } from "graphql/generated/runCreate";
import { deleteOneMutation, runCreateMutation } from "graphql/mutation";
import { Run } from "types";
import { deleteOne, deleteOneVariables } from "graphql/generated/deleteOne";
import { DeleteOneType } from "@shared/consts";
import { v4 as uuid } from 'uuid';
import { MenuTitle } from "../MenuTitle/MenuTitle";
import { RunStatus } from "graphql/generated/globalTypes";
import { DeleteIcon } from "@shared/icons";

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
        const searchParams = parseSearchParams(window.location.search);
        if (!searchParams.run) return
        const run = routine.runs.find(run => run.id === searchParams.run);
        if (run) {
            onSelect(run);
            handleClose();
        }
    }, [routine, onSelect, handleClose]);

    const [runCreate] = useMutation<runCreate, runCreateVariables>(runCreateMutation);
    const createNewRun = useCallback(() => {
        if (!routine) {
            PubSub.get().publishSnack({ message: 'Could not read routine data.', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: runCreate,
            input: {
                id: uuid(),
                routineId: routine.id,
                version: routine.version ?? '',
                title: getTranslation(routine, 'title', getUserLanguages(session)) ?? 'Unnamed Routine',
            },
            successCondition: (response) => response.data.runCreate !== null,
            onSuccess: (response) => {
                const newRun = response.data.runCreate;
                onAdd(newRun);
                onSelect(newRun);
                handleClose();
            },
            onError: () => { PubSub.get().publishSnack({ message: 'Failed to create run.', severity: 'error' }) },
        })
    }, [handleClose, onAdd, onSelect, routine, runCreate, session]);

    const [deleteOne] = useMutation<deleteOne, deleteOneVariables>(deleteOneMutation)
    const deleteRun = useCallback((run: Run) => {
        mutationWrapper({
            mutation: deleteOne,
            input: { id: run.id, objectType: DeleteOneType.Run },
            onSuccess: (response) => {
                if (response?.data?.deleteOne?.success) {
                    PubSub.get().publishSnack({ message: `${displayDate(run.timeStarted)} deleted.` });
                    onDelete(run);
                } else {
                    PubSub.get().publishSnack({ message: `Error deleting ${displayDate(run.timeStarted)}.`, severity: 'error' });
                }
            },
            onError: () => {
                PubSub.get().publishSnack({ message: `Failed to delete ${displayDate(run.timeStarted)}.` });
            }
        })
    }, [deleteOne, onDelete])

    useEffect(() => {
        if (!open) return;
        // If not logged in, open routine without creating a new run
        if (!session.id) {
            onSelect(null);
            handleClose();
        }
        // If routine has no runs, create a new one.
        else if (routine && routine.runs?.filter(r => r.status === RunStatus.InProgress)?.length === 0) {
            createNewRun();
        }
    }, [open, routine, createNewRun, onSelect, session.id, handleClose]);

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
                message: `Are you sure you want to delete this run from ${displayDate(run.timeStarted)} with ${getRunPercentComplete(run.completedComplexity, routine.complexity)}% completed?`,
                buttons: [
                    {
                        text: 'Yes', onClick: () => {
                            deleteRun(run);
                        }
                    },
                    { text: 'Cancel', onClick: () => { } },
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