/**
 * Handles selecting a run from a list of runs.
 */

import { Box, Button, IconButton, List, ListItem, ListItemText, Menu, Tooltip, Typography, useTheme } from "@mui/material";
import { mutationWrapper } from "graphql/utils";
import { useCallback, useEffect, useMemo } from "react";
import { noSelect } from "styles";
import { displayDate, getTranslation, getUserLanguages } from "utils/display";
import { ListMenuItemData, RunPickerDialogProps } from "../types";
import {
    Close as CloseIcon,
    Delete as DeleteIcon,
} from "@mui/icons-material";
import { getRunPercentComplete, parseSearchParams, Pubs } from "utils";
import { useMutation } from "@apollo/client";
import { runCreate, runCreateVariables } from "graphql/generated/runCreate";
import { deleteOneMutation, runCreateMutation } from "graphql/mutation";
import { Run } from "types";
import { deleteOne, deleteOneVariables } from "graphql/generated/deleteOne";
import { DeleteOneType } from "@local/shared";
import { v4 as uuid } from 'uuid';

export const RunPickerDialog = ({
    anchorEl,
    handleClose,
    onAdd,
    onDelete,
    onSelect,
    routine,
    session
}: RunPickerDialogProps) => {
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
            PubSub.publish(Pubs.Snack, { message: 'Could not read routine data.', severity: 'error' });
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
            onError: () => { PubSub.publish(Pubs.Snack, { message: 'Failed to create run.', severity: 'error' }) },
        })
    }, [handleClose, onAdd, onSelect, routine, runCreate, session]);

    const [deleteOne] = useMutation<deleteOne, deleteOneVariables>(deleteOneMutation)
    const deleteRun = useCallback((run: Run) => {
        mutationWrapper({
            mutation: deleteOne,
            input: { id: run.id, objectType: DeleteOneType.Run },
            onSuccess: (response) => {
                if (response?.data?.deleteOne?.success) {
                    PubSub.publish(Pubs.Snack, { message: `${displayDate(run.timeStarted)} deleted.` });
                    onDelete(run);
                } else {
                    PubSub.publish(Pubs.Snack, { message: `Error deleting ${displayDate(run.timeStarted)}.`, severity: 'error' });
                }
            },
            onError: () => {
                PubSub.publish(Pubs.Snack, { message: `Failed to delete ${displayDate(run.timeStarted)}.` });
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
        else if (routine && routine.runs?.length === 0) {
            createNewRun();
        }
    }, [open, routine, createNewRun, onSelect, session.id, handleClose]);

    const runOptions: ListMenuItemData<Run>[] = useMemo(() => {
        if (!routine || !routine.runs) return [];
        const runs = routine.runs;
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
            PubSub.publish(Pubs.AlertDialog, {
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
                        <DeleteIcon />
                    </IconButton>
                </Tooltip>
            </ListItem>
        )
    }), [runOptions, onSelect, handleClose, handleDelete]);

    return (
        <Menu
            id='select-run-dialog'
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
            <Box
                sx={{
                    ...noSelect,
                    display: 'flex',
                    alignItems: 'center',
                    padding: 1,
                    background: palette.primary.dark
                }}
            >
                <Typography
                    variant="h6"
                    textAlign="center"
                    sx={{
                        width: '-webkit-fill-available',
                        color: palette.primary.contrastText,
                    }}
                >
                    Continue Existing Run?
                </Typography>
                <IconButton
                    edge="end"
                    onClick={handleClose}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
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