import { Box, Button, Grid, IconButton, Slider, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { HelpButton, NodeGraphContainer, OrchestrationInfoDialog, RoutineInfoDialog, UnlinkedNodesDialog } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { OrchestrationDialogOption, OrchestrationStatus, Pubs } from 'utils';
import {
    Add as AddIcon,
    Close as CloseIcon,
    Done as DoneIcon,
    Edit as EditIcon,
    Info as InfoIcon,
    Restore as RestoreIcon,
    Update as UpdateIcon
} from '@mui/icons-material';
import { Node, Routine } from 'types';
import isEqual from 'lodash/isEqual';
import { withStyles } from '@mui/styles';
import helpMarkdown from './OrchestratorHelp.md';
import { useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { NodePos, OrchestrationStatusObject } from 'components/graphs/NodeGraph/types';
import { NodeType } from 'graphql/generated/globalTypes';

/**
 * Status indicator and slider change color to represent orchestration's status
 */
const STATUS_COLOR = {
    [OrchestrationStatus.Incomplete]: '#cde22c', // Yellow
    [OrchestrationStatus.Invalid]: '#ff6a6a', // Red
    [OrchestrationStatus.Valid]: '#00d51e', // Green
}
const STATUS_LABEL = {
    [OrchestrationStatus.Incomplete]: 'Incomplete',
    [OrchestrationStatus.Invalid]: 'Invalid',
    [OrchestrationStatus.Valid]: 'Valid',
}

const TERTIARY_COLOR = '#95f3cd';

const CustomSlider = withStyles({
    root: {
        width: 'calc(100% - 32px)',
        left: '11px', // 16px - 1/4 of thumb diameter
    },
})(Slider);

export const RoutineOrchestratorPage = () => {
    // Get routine ID from URL
    const [, params] = useRoute(`${APP_LINKS.Orchestrate}/:id`);
    const id: string = useMemo(() => params?.id ?? '', [params]);
    // Queries routine data
    const { data: routineData } = useQuery<routine>(routineQuery, { variables: { input: { id } } });
    const [routine, setRoutine] = useState<Routine | null>(null);
    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null);
    useEffect(() => { setRoutine(routineData?.routine ?? null) }, [routineData]);
    // Routine mutator
    const [routineUpdate, { loading }] = useMutation<any>(routineUpdateMutation);
    // The routine's status (valid/invalid/incomplete)
    const [status, setStatus] = useState<OrchestrationStatusObject>({ code: OrchestrationStatus.Incomplete, details: 'TODO' });
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    // Used for editing the title of the routine
    const [titleActive, setTitleActive] = useState<boolean>(false);
    const toggleTitle = useCallback(() => setTitleActive(a => !a), []);
    const saveTitle = useCallback(() => {
        //TODO
        setTitleActive(false);
    }, []);
    const cancelTitle = useCallback(() => {
        //TODO
        setTitleActive(false);
    }, []);

    // Open/close routine info drawer
    const [isRoutineInfoOpen, setIsRoutineInfoOpen] = useState<boolean>(false);
    const closeRoutineInfo = useCallback(() => setIsRoutineInfoOpen(false), []);
    // Open/close unlinked nodes drawer
    const [isUnlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState<boolean>(false);
    const toggleUnlinkedNodes = useCallback(() => setIsUnlinkedNodesOpen(curr => !curr), []);

    useEffect(() => {
        setChangedRoutine(routine);
    }, [routine]);

    /**
     * 1st return - Dictionary of node data and their columns
     * 2nd return - List of nodes which are not yet linked
     * If nodeDataMap is same length as nodes, and unlinkedList is empty, then all nodes are linked 
     * and the graph is valid
     */
    const [nodeDataMap, unlinkedList] = useMemo<[{ [id: string]: NodePos }, Node[]]>(() => {
        // Position map for calculating node positions
        let posMap: { [id: string]: NodePos } = {};
        const nodes = changedRoutine?.nodes ?? [];
        const links = changedRoutine?.nodeLinks ?? [];
        if (!nodes || !links) return [posMap, nodes ?? []];
        let startNodeId: string | null = null;
        console.log('node data map', nodes, links);
        // First pass of raw node data, to locate start node and populate position map
        for (let i = 0; i < nodes.length; i++) {
            const currId = nodes[i].id;
            // If start node, must be in first column
            if (nodes[i].type === NodeType.Start) {
                startNodeId = currId;
                posMap[currId] = { column: 0, node: nodes[i] }
            }
        }
        // If start node was found
        if (startNodeId) {
            // Loop through links. Each loop finds every node that belongs in the next column
            // We set the max number of columns to be 100, but this is arbitrary
            for (let currColumn = 0; currColumn < 100; currColumn++) {
                // Calculate the IDs of each node in the next column TODO this should be sorted in some way so it shows the same order every time
                const nextNodes = links
                    .filter(link => posMap[link.fromId]?.column === currColumn)
                    .map(link => nodes.find(node => node.id === link.toId))
                    .filter(node => node) as Node[];
                // Add each node to the position map
                for (let i = 0; i < nextNodes.length; i++) {
                    const curr = nextNodes[i];
                    posMap[curr.id] = { column: currColumn + 1, node: curr };
                }
                // If not nodes left, or if all of the next nodes are end nodes, stop looping
                if (nextNodes.length === 0 || nextNodes.every(n => n.type === NodeType.End)) {
                    break;
                }
            }
        } else {
            console.error('Error: No start node found');
            setStatus({ code: OrchestrationStatus.Invalid, details: 'No start node found' });
        }
        // TODO check if all paths end with an end node (and account for loops)
        const unlinked = nodes.filter(node => !posMap[node.id]);
        if (unlinked.length > 0) {
            console.warn('Warning: Some nodes are not linked');
            setStatus({ code: OrchestrationStatus.Incomplete, details: 'Some nodes are not linked' });
        }
        if (startNodeId && unlinked.length === 0) {
            setStatus({ code: OrchestrationStatus.Valid, details: '' });
        }
        return [posMap, unlinked];
    }, [changedRoutine]);

    const handleDialogOpen = useCallback((nodeId: string, dialog: OrchestrationDialogOption) => {
        switch (dialog) {
            case OrchestrationDialogOption.AddRoutineItem:
                break;
            case OrchestrationDialogOption.ViewRoutineItem:
                setIsRoutineInfoOpen(true);
                break;
        }
    }, []);

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
            input: changedRoutine,
            successMessage: () => 'Routine updated.',
            onSuccess: ({ data }) => { setRoutine(data.routineUpdate); },
        })
    }, [changedRoutine, routine, routineUpdate])

    const revertChanges = useCallback(() => {
        setChangedRoutine(routine);
    }, [routine])

    // Parse markdown from .md file
    const [helpText, setHelpText] = useState<string>('');
    useEffect(() => {
        fetch(helpMarkdown).then((r) => r.text()).then((text) => { setHelpText(text) });
    }, []);

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
                    variant="filled"
                    id="title"
                    name="title"
                    autoComplete="routine-title"
                    label="Title"
                    value={changedRoutine?.title ?? ''}
                    onChange={() => { }}
                    sx={{
                        marginTop: 1,
                        marginBottom: 1,
                        '& .MuiInputLabel-root': {
                            display: 'none',
                        },
                        '& .MuiInputBase-root': {
                            borderBottom: 'none',
                            borderRadius: '32px',
                            border: `2px solid green`,//TODO titleValid ? green : red
                            overflow: 'overlay',
                        },
                        '& .MuiInputBase-input': {
                            position: 'relative',
                            backgroundColor: '#ffffff94',
                            border: '1px solid #ced4da',
                            fontSize: 16,
                            width: 'auto',
                            padding: '8px 8px',
                        }
                    }}
                />
                {/* Buttons for saving/cancelling edit */}
                <IconButton aria-label="confirm-title-change" onClick={saveTitle}>
                    <DoneIcon sx={{ fill: '#40dd43' }} />
                </IconButton>
                <IconButton aria-label="cancel-title-change" onClick={cancelTitle} color="secondary">
                    <CloseIcon sx={{ fill: '#ff2a2a' }} />
                </IconButton>
            </Stack>
        ) : (
            <Box sx={{ display: 'flex', alignItems: 'center', cursor: 'pointer', paddingTop: 1, paddingBottom: 1 }} onClick={toggleTitle}>
                <Typography
                    component="h2"
                    variant="h5"
                    textAlign="center"
                >{changedRoutine?.title ?? 'Loading...'}</Typography>
            </Box>
        );
        return (
            <Stack
                direction="row"
                spacing={2}
                width="100%"
                justifyContent="space-between"
                sx={{
                    zIndex: 2,
                    background: (t) => t.palette.primary.light,
                    color: (t) => t.palette.primary.contrastText,
                }}
            >
                {/* Status indicator */}
                <Tooltip title={status.details}>
                    <Box sx={{
                        borderRadius: 1,
                        border: `2px solid ${STATUS_COLOR[status.code]}`,
                        color: STATUS_COLOR[status.code],
                        height: 'fit-content',
                        fontWeight: 'bold',
                        fontSize: 'larger',
                        padding: 0.5,
                        marginTop: 'auto',
                        marginBottom: 'auto',
                        marginLeft: 2,
                    }}>{STATUS_LABEL[status.code]}</Box>
                </Tooltip>
                {/* Routine title label/TextField and action buttons */}
                {title}
                <Stack direction="row" spacing={1}>
                    {/* Help button */}
                    <HelpButton markdown={helpText} sxRoot={{ margin: "auto" }} sx={{ color: TERTIARY_COLOR }} />
                    {/* Switch to routine metadata page */}
                    <OrchestrationInfoDialog
                        sxs={{ icon: { fill: TERTIARY_COLOR, marginRight: 1 } }}
                        routineInfo={changedRoutine}
                        onUpdate={updateRoutine}
                        onCancel={() => { setRoutine(changedRoutine); }}
                    />
                </Stack>
            </Stack>
        )
    }, [status, titleActive, toggleTitle]);

    let optionsBar = useMemo(() => (
        <Grid
            container
            p={2}
            sx={{
                background: (t) => t.palette.primary.light,
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                paddingBottom: { xs: '72px', md: '16px' },
            }}
        >
            {
                isEditing ?
                    (
                        <>
                            <Grid item xs={12} sm={6} pt={0}>
                                <CustomSlider
                                    aria-label="graph-scale"
                                    min={0.25}
                                    max={1}
                                    defaultValue={1}
                                    step={0.01}
                                    value={scale}
                                    valueLabelDisplay="auto"
                                    onChange={handleScaleChange}
                                />
                            </Grid>
                            <Grid item xs={12} sm={3} sx={{ padding: 1 }}>
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
                                <CustomSlider
                                    aria-label="graph-scale"
                                    min={0.01}
                                    max={1}
                                    defaultValue={1}
                                    step={0.01}
                                    value={scale}
                                    valueLabelDisplay="auto"
                                    valueLabelFormat={value => <div>{`Scale: ${Math.floor(value * 100)}`}</div>}
                                    onChange={handleScaleChange}
                                    sx={{
                                        color: STATUS_COLOR[status.code],
                                    }}
                                />
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
    ), [changedRoutine, isEditing, loading, revertChanges, routine, scale, status, startEditing, updateRoutine])

    return (
        <Box sx={{
            paddingTop: '10vh',
            display: 'flex',
            flexDirection: 'column',
            minHeight: '100%',
            height: '100%',
            width: '100%',
        }}>
            {/* Displays routine information when you click on a routine list item*/}
            <RoutineInfoDialog
                open={isRoutineInfoOpen}
                routineInfo={changedRoutine}
                onClose={closeRoutineInfo}
            />
            {/* Displays orchestration information and some buttons */}
            {informationBar}
            <Box sx={{
                display: 'flex',
                alignItems: isUnlinkedNodesOpen ? 'baseline' : 'center',
                alignSelf: 'flex-end',
                marginTop: 1,
                marginLeft: 1,
                marginRight: 1,
                zIndex: 2,
            }}>
                {/* Add new nodes to the orchestration */}
                <Tooltip title='Add new node'>
                    <IconButton edge="end" onClick={() => { }} aria-label='Add node' sx={{
                        background: (t) => t.palette.secondary.main,
                        marginRight: 1,
                        transition: 'brightness 0.2s ease-in-out',
                        '&:hover': {
                            filter: `brightness(105%)`,
                            background: (t) => t.palette.secondary.main,
                        },
                    }}>
                        <AddIcon sx={{ fill: 'white' }} />
                    </IconButton>
                </Tooltip>
                {/* Displays unlinked nodes */}
                <UnlinkedNodesDialog
                    open={isUnlinkedNodesOpen}
                    nodes={changedRoutine?.nodes ?? []} //TODO change to unlinkedNodes. This is only like this for testing
                    handleToggleOpen={toggleUnlinkedNodes}
                    handleDeleteNode={() => { }}
                />
            </Box>
            <Box sx={{
                width: '100%',
                display: 'flex',
                flexDirection: 'column',
                position: 'fixed',
                bottom: '0',
            }}>
                <NodeGraphContainer
                    scale={scale}
                    isEditable={true}
                    labelVisible={true}
                    nodeDataMap={nodeDataMap}
                    links={changedRoutine?.nodeLinks ?? []}
                    handleDialogOpen={handleDialogOpen}
                />
                {optionsBar}
            </Box>
        </Box>
    )
};