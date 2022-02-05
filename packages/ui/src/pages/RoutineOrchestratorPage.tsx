import { Box, Button, Grid, IconButton, Slider, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { NodeType, OrchestrationData } from '@local/shared';
import { NodeGraphContainer } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { Pubs } from 'utils';
import {
    Close as CloseIcon,
    Done as DoneIcon,
    Edit as EditIcon,
    Info as InfoIcon,
    Restore as RestoreIcon,
    Update as UpdateIcon
} from '@mui/icons-material';
import { Routine } from 'types';
import isEqual from 'lodash/isEqual';

//TEMP
const data: OrchestrationData = {
    title: 'Validate business idea',
    description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.',
    // Unordered list of top-level nodes
    // Page renders nodes starting at the start node
    nodes: [
        // Start node
        {
            id: '1',
            type: NodeType.Start,
            title: null,
            description: null,
            // ID of previous node
            previous: null,
            // ID of next node
            next: '2',
            // Additional data specific to the node type
            data: null
        },
        // Routine List node
        {
            id: '2',
            type: NodeType.RoutineList,
            title: 'Provide Basic Info',
            description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
            previous: '1',
            next: '3',
            data: {
                isOrdered: false,
                isOptional: false,
                routines: [
                    {
                        id: '1',
                        title: 'Provide Basic Info',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: false,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    }
                ]
            }
        },
        // Routine List node
        {
            id: '3',
            type: NodeType.RoutineList,
            title: 'Knowledge management',
            description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
            previous: '2',
            next: '4',
            data: {
                isOrdered: false,
                isOptional: false,
                routines: [
                    {
                        id: '2',
                        title: 'Create task list',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    },
                    {
                        id: '3',
                        title: 'Generate forecasts',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    },
                    {
                        id: '4',
                        title: 'List objectives',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: false,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    },
                    {
                        id: '5',
                        title: 'Create business model',
                        description: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt',
                        isOptional: true,
                        routine: {
                            id: '1',
                            title: 'Provide Basic Info',
                            description: null,
                            isAutomatable: false,
                        }
                    }
                ]
            }
        },
        // Decision node
        {
            id: '4',
            type: NodeType.Decision,
            title: 'Worth pursuing?',
            description: null,
            previous: '3',
            // Decision nodes have no definitive next node
            // Instead, next node is determined by the decision
            next: null,
            data: {
                decisions: [
                    {
                        id: '1',
                        title: 'Yes',
                        description: null,
                        next: '5',
                        // List of cases which must return true in order for this decision to be available
                        // If empty, decision is always available
                        when: []
                    },
                    {
                        id: '2',
                        title: 'No',
                        description: null,
                        next: '6',
                        when: []
                    }
                ]
            }
        },
        // Routine List node
        {
            id: '5',
            type: NodeType.RoutineList,
            title: 'todo',
            description: null,
            previous: '4',
            next: '7',
            data: {
                isOrdered: false,
                isOptional: false,
                routines: []
            },
        },
        // End node
        {
            id: '6',
            type: NodeType.End,
            title: 'Wasnt worth pursuing',
            description: 'afda',
            previous: '4',
            next: null,
            data: {
                wasSuccessful: true,
            }
        },
        // Decision node
        {
            id: '7',
            type: NodeType.Decision,
            title: 'Try again?',
            description: null,
            previous: '5',
            next: null,
            data: {
                decisions: [
                    {
                        id: '3',
                        title: 'Yes',
                        description: null,
                        next: '8',
                        // List of cases which must return true in order for this decision to be available
                        // If empty, decision is always available
                        when: []
                    },
                    {
                        id: '4',
                        title: 'No',
                        description: null,
                        next: '9',
                        when: []
                    }
                ]
            }
        },
        // Redirect node
        {
            id: '8',
            type: NodeType.Redirect,
            title: null,
            description: null,
            previous: '7',
            next: '1',
            data: null,
        },
        // End node
        {
            id: '9',
            type: NodeType.End,
            title: 'The good end',
            description: 'asdf',
            previous: '7',
            next: null,
            data: {
                wasSuccessful: true,
            }
        },
    ]
}

export const RoutineOrchestratorPage = () => {
    // Queries routine data
    const { data: routineData } = useQuery<routine>(routineQuery, { variables: { input: { id: 'TODO' } } });
    const [routine, setRoutine] = useState<Routine | null>(null);
    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null);
    // Routine mutator
    const [routineUpdate, { loading }] = useMutation<any>(routineUpdateMutation);
    // The routine's status (valid/invalid)
    const [isValid, setIsValid] = useState<boolean>(false);
    // The routine's title
    const [title, setTitle] = useState<string>('');
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    // Used for editing the title of the routine
    const [titleActive, setTitleActive] = useState<boolean>(false);
    const toggleTitle = useCallback(() => setTitleActive(a => !a), []);

    useEffect(() => {
        setTitle(routineData?.routine?.title ?? '');
        setRoutine(routineData?.routine ?? null);
    }, [routineData]);

    useEffect(() => {
        setChangedRoutine(routine);
    }, [routine]);

    const handleScaleChange = (_event: any, newScale: number | number[]) => {
        console.log('HANDLE SCALE CHANGE', newScale);
        setScale(newScale as number);
    };

    const startEditing = useCallback(() => setIsEditing(true), []);

    /**
     * Mutates routine data
     */
    const updateRoutine = useCallback(() => {
        if (!changedRoutine || isEqual(routine, changedRoutine)) {
            PubSub.publish(Pubs.Snack, { message: 'No changes detected', severity: 'error' });
            return;
        }
        if (!changedRoutine.id) {
            PubSub.publish(Pubs.Snack, { message: 'Cannot update: Invalid routine data', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: routineUpdate,
            input: {
                id: changedRoutine.id,
                title,
            },
            successMessage: () => 'Routine updated.',
        })
    }, [changedRoutine, routine, routineUpdate, title])

    const revertChanges = useCallback(() => {
        setChangedRoutine(routine);
    }, [routine])

    /**
     * Displays metadata about the routine orchestration.
     * On the left is a status indicator, which lets you know if the routine is valid or not.
     * In the middle is the title of the routine. Once clicked, the information bar converts to 
     * a text input field, which allows you to edit the title of the routine.
     * To the right is a button to switch to the metadata view/edit component. You can view/edit the 
     * title, descriptions, instructions, inputs, outputs, tags, etc.
     */
    const informationBar = useMemo(() => {
        const title = titleActive ? (
            <Stack direction="row" spacing={1} alignItems="center">
                {/* Component for editing title */}
                <TextField
                    autoFocus
                    id="title"
                    name="title"
                    autoComplete="routine-title"
                    label="Title"
                    value={routine?.title ?? ''}
                    onChange={() => { }}
                    sx={{ cursor: 'pointer' }}
                />
                {/* Buttons for saving/cancelling edit */}
                <IconButton aria-label="confirm-title-change" onClick={() => { }} color="secondary">
                    <DoneIcon />
                </IconButton>
                <IconButton aria-label="cancel-title-change" onClick={() => { }} color="secondary">
                    <CloseIcon />
                </IconButton>
            </Stack>
        ) : (
            <Stack direction="row" spacing={1} alignItems="center">
                {/* Component for viewing routine title */}
                <Typography
                    onClick={toggleTitle}
                    component="h2"
                    variant="h5"
                    textAlign="center"
                    sx={{ cursor: 'pointer' }}
                >{data.title}</Typography>
                {/* Component for editing routine title. Can also click on routine name to edit */}
                <IconButton aria-label="change-title" onClick={() => { }} color="secondary">
                    <EditIcon />
                </IconButton>
            </Stack>
        );
        return (
            <Stack direction="row" spacing={2} justifyContent="space-between" width="100%" color="white">
                {/* Status indicator */}
                <Tooltip title={isValid ? "Routine is valid" : "Routine is invalid"}>
                    <Box sx={{
                        borderRadius: 1,
                        padding: 0.5,
                        border: `2px solid ${isValid ? '#6ef04e' : '#e74242'}`,
                        color: isValid ? '#6ef04e' : '#e74242',
                        height: 'fit-content',
                        margin: 'auto 8px',
                    }}>{isValid ? 'Valid' : 'Invalid'}</Box>
                </Tooltip>
                {/* Routine title label/TextField and action buttons */}
                {title}
                {/* Switch to routine metadata page */}
                <IconButton aria-label="show-routine-metadata" onClick={() => { }} color="secondary">
                    <InfoIcon />
                </IconButton>
            </Stack>
        )
    }, [isValid, title, titleActive, toggleTitle]);

    let options = useMemo(() => (
        <Grid
            container
            sx={{
                position: 'fixed',
                bottom: { xs: '56px', md: '0px' },
                zIndex: 15,
                background: "#22334f",
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
            }}
        >
            {
                isEditing ?
                    (
                        <>
                            <Grid item xs={12} sm={6} pt={0}>
                                <Slider
                                    aria-label="graph-scale"
                                    min={0.01}
                                    max={1}
                                    defaultValue={1}
                                    step={0.01}
                                    value={scale}
                                    valueLabelDisplay="auto"
                                    onChange={handleScaleChange}
                                    sx={{paddingLeft: '12px', paddingRight: '10px', color: '#23eaed'}}
                                />
                            </Grid>
                            <Grid item xs={12} sm={3} sx={{padding: 1}}>
                                <Button
                                    fullWidth
                                    startIcon={<UpdateIcon />}
                                    onClick={updateRoutine}
                                    disabled={loading || !isEqual(routine, changedRoutine)}
                                >Update</Button>
                            </Grid>
                            <Grid item xs={12} sm={3}>
                                <Button
                                    fullWidth
                                    startIcon={<RestoreIcon />}
                                    onClick={revertChanges}
                                    disabled={loading || isEqual(routine, changedRoutine)}
                                >Revert</Button>
                            </Grid>
                        </>
                    ) :
                    (
                        <>
                            <Grid item xs={12} sm={9}>
                                <Slider
                                    aria-label="graph-scale"
                                    min={0.01}
                                    max={1}
                                    defaultValue={1}
                                    step={0.01}
                                    value={scale}
                                    valueLabelDisplay="auto"
                                    onChange={handleScaleChange} />
                            </Grid>
                            <Grid item xs={12} sm={3}>
                                <Button
                                    fullWidth
                                    startIcon={<EditIcon />}
                                    onClick={startEditing}
                                    disabled={loading}
                                >Update</Button>
                            </Grid>
                        </>
                    )
            }
        </Grid >
    ), [changedRoutine, isEditing, loading, revertChanges, routine, scale, startEditing, updateRoutine])

    return (
        <Box sx={{
            minWidth: '100%',
            minHeight: '100%',
            paddingTop: '10vh'
        }}>
            {/* Information bar */}
            <Stack
                direction="row"
                spacing={2}
                width="100%"
                height="5vh"
                justifyContent="space-around"
                sx={{ background: "#22334f" }}
            >
                {informationBar}
            </Stack>
            <Box sx={{ width: "102%", height: 'calc(80vh - 56px)', }}>
                <NodeGraphContainer
                    scale={scale}
                    isEditable={true}
                    labelVisible={true}
                    nodes={data?.nodes}
                />
            </Box>
            {options}
        </Box>
    )
};