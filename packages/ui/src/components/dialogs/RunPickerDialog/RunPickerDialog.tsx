/**
 * Handles selecting a run from a list of runs.
 */

import { Box, Button, IconButton, List, ListItem, ListItemText, Menu, Typography, useTheme } from "@mui/material";
import { mutationWrapper } from "graphql/utils";
import { useCallback, useEffect, useMemo } from "react";
import { noSelect } from "styles";
import { displayDate, getTranslation, getUserLanguages } from "utils/display";
import { ListMenuItemData, RunPickerDialogProps } from "../types";
import { Close as CloseIcon } from "@mui/icons-material";
import { parseSearchParams, Pubs } from "utils";
import { useMutation } from "@apollo/client";
import { runCreate } from "graphql/generated/runCreate";
import { runCreateMutation } from "graphql/mutation";
import { Run } from "types";

export const RunPickerDialog = ({
    anchorEl,
    handleClose,
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
            console.log('on select zz', run, routine)
            onSelect(run);
            handleClose();
        }
    }, [routine, onSelect, handleClose]);

    const [runCreate] = useMutation<runCreate>(runCreateMutation);
    const createNewRun = useCallback(() => {
        console.log('CREATING NEW RUN')
        if (!routine) {
            PubSub.publish(Pubs.Snack, { message: 'Could not read routine data.', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: runCreate,
            input: {
                routineId: routine.id,
                version: routine.version,
                title: getTranslation(routine, 'title', getUserLanguages(session)),
            },
            successCondition: (response) => response.data.runCreate !== null,
            onSuccess: (response) => { console.log('onselect a'); onSelect(response.data.runCreate); handleClose(); },
            onError: () => { PubSub.publish(Pubs.Snack, { message: 'Failed to create run.', severity: 'error' }) },
        })
    }, [handleClose, onSelect, routine, runCreate, session]);

    useEffect(() => {
        if (!open) return;
        // If not logged in, open routine without creating a new run
        if (!session.id) {
            console.log('onselect b');
            onSelect(null);
        }
        // If routine has no runs, create a new one.
        else if (routine && routine.runs.length === 0) {
            console.log('createnewrun call', routine, session.id);
            createNewRun();
        }
    }, [open, routine, createNewRun, onSelect, session.id]);

    const runOptions: ListMenuItemData<Run>[] = useMemo(() => {
        if (!routine || !routine.runs) return [];
        const runs = routine.runs;
        return runs.map((run) => ({
            label: `Started: ${displayDate(run.timeStarted)} (${Math.round((run.completedComplexity as number) / routine.complexity * 100)}%)`,
            value: run as Run,
        }));
    }, [routine]);

    const items = useMemo(() => runOptions.map((data: ListMenuItemData<Run>, index) => {
        const itemText = <ListItemText primary={data.label} />;
        return (
            <ListItem button onClick={() => { console.log('onselect c'); onSelect(data.value); handleClose(); }} key={index}>
                {itemText}
            </ListItem>
        )
    }), [runOptions, onSelect, handleClose]);

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
                    background: palette.background.paper
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