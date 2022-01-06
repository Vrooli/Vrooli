import { makeStyles } from '@mui/styles';
import { Button, Grid, Slider, Stack, Theme, Typography } from '@mui/material';
import { CombineNodeData, DecisionNodeData, DecisionNodeDataDecision, NodeData, NodeType, OrchestrationData } from '@local/shared';
import { NodeGraph, NodeGraphColumn } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { PUBS } from 'utils';
import {
    Done as DoneIcon,
    Edit as EditIcon,
    Restore as RestoreIcon,
    Update as UpdateIcon
} from '@mui/icons-material';
import { RoutineDeep } from 'types';
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

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        minWidth: '100%',
        minHeight: '100%',
        paddingTop: '10vh'
    },
    title: {
        textAlign: 'center',
    },
    // Horizontal scrolling container
    graphContainer: {
        height: '75vh',
        border: '1px solid red',
        // display: 'flex',
        // flexDirection: 'row',
        // flexWrap: 'nowrap',
        // justifyContent: 'flex-start',
        // alignItems: 'center',
        // alignContent: 'center',
    },
    optionsContainer: {
        height: '15vh',
        padding: theme.spacing(2),
    },
}));

export const RoutineOrchestratorPage = () => {
    const classes = useStyles();
    // Queries routine data
    const { data: routineData } = useQuery<routine>(routineQuery, { variables: { input: { id: 'TODO' } } });
    const [routine, setRoutine] = useState<RoutineDeep | null>(null);
    const [changedRoutine, setChangedRoutine] = useState<RoutineDeep | null>(null);
    // Routine mutator
    const [routineUpdate, { loading }] = useMutation<any>(routineUpdateMutation);
    // The routine's title
    const [title, setTitle] = useState<string>('');
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const [isEditing, setIsEditing] = useState<boolean>(false);

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
            PubSub.publish(PUBS.Snack, { message: 'No changes detected', severity: 'error' });
            return;
        }
        if (!changedRoutine.id) {
            PubSub.publish(PUBS.Snack, { message: 'Cannot update: Invalid routine data', severity: 'error' });
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

    let options = useMemo(() => (
        <Grid className={classes.optionsContainer} container spacing={1}>
            {
                isEditing ?
                    (
                        <>
                            <Grid item xs={12} sm={6}>
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
    ), [changedRoutine, classes.optionsContainer, isEditing, loading, revertChanges, routine, scale, startEditing, updateRoutine])

    return (
        <div className={classes.root}>
            <Typography component="h2" variant="h4" className={classes.title}>{data.title}</Typography>
            <div className={classes.graphContainer}>
                <NodeGraph
                    scale={scale}
                    isEditable={true}
                    labelVisible={true}
                    nodes={data?.nodes}
                />
            </div>
            {options}
        </div>
    )
};