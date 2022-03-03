import { Box, IconButton, Tooltip } from '@mui/material';
import { NodeGraph, OrchestrationBottomContainer, OrchestrationInfoContainer, RoutineInfoDialog, UnlinkedNodesDialog } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { OrchestrationDialogOption, OrchestrationRunState, OrchestrationStatus, Pubs } from 'utils';
import {
    Add as AddIcon,
    DeleteForever as DeleteIcon,
} from '@mui/icons-material';
import { Node, Routine } from 'types';
import isEqual from 'lodash/isEqual';
import { useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { NodePos, OrchestrationStatusObject } from 'components/graphs/NodeGraph/types';
import { NodeType } from 'graphql/generated/globalTypes';
import { RoutineOrchestratorPageProps } from 'pages/types';

/**
 * Status indicator and slider change color to represent orchestration's status
 */
const STATUS_COLOR = {
    [OrchestrationStatus.Incomplete]: '#cde22c', // Yellow
    [OrchestrationStatus.Invalid]: '#ff6a6a', // Red
    [OrchestrationStatus.Valid]: '#00d51e', // Green
}

export const RoutineOrchestratorPage = ({
    session,
}: RoutineOrchestratorPageProps) => {
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
    const canEdit = useMemo(() => Boolean(session?.id) && routine?.owner?.id === session?.id, [routine]); //TODO handle organization

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
     * Hacky way to display dragging nodes over over elements. Disables z-index when dragging
     */
    const [isDragging, setIsDragging] = useState<boolean>(false);
    useEffect(() => {
        // Add PubSub subscribers
        let dragStartSub = PubSub.subscribe(Pubs.NodeDrag, (_, data) => {
            setIsDragging(true);
        });
        let dragDropSub = PubSub.subscribe(Pubs.NodeDrop, (_, data) => {
            console.log('IN DA DROPPPPPPPPPP')
            setIsDragging(false);
        });
        return () => {
            // Remove PubSub subscribers
            PubSub.unsubscribe(dragStartSub);
            PubSub.unsubscribe(dragDropSub);
        }
    }, []);

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

    const handleScaleChange = (newScale: number) => {
        setScale(newScale);
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

    const updateRoutineTitle = useCallback((title: string) => {
        if (!changedRoutine) return;
        setChangedRoutine({ ...changedRoutine, title });
    }, [changedRoutine]);

    const revertChanges = useCallback(() => {
        setChangedRoutine(routine);
    }, [routine])

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
            <OrchestrationInfoContainer
                canEdit={canEdit}
                handleStartEdit={startEditing}
                isEditing={isEditing}
                status={status}
                routine={changedRoutine}
                handleRoutineUpdate={updateRoutine}
                handleTitleUpdate={updateRoutineTitle}
            />
            {/* Components shown when editing */}
            {isEditing ? <Box sx={{
                display: 'flex',
                alignItems: isUnlinkedNodesOpen ? 'baseline' : 'center',
                // alignSelf: 'flex-end',
                marginTop: 1,
                marginLeft: 1,
                marginRight: 1,
                zIndex: isDragging ? 'unset' : 2,
            }}>
                {/* Delete node (or whole orchestration) */}
                <Tooltip title='Delete a node'>
                    <IconButton
                        id="delete-node-button"
                        edge="start"
                        size="large"
                        onClick={() => { }}
                        aria-label='Delete node'
                        sx={{
                            marginLeft: 1,
                            marginRight: 'auto',
                            '&:hover': {
                                background: 'transparent',
                            },
                        }}
                    >
                        <DeleteIcon id="delete-node-button-icon" sx={{
                            fill: '#8f969b',
                            transform: 'scale(1.5)',
                            '&:hover': {
                                fill: (t) => t.palette.error.main,
                            }
                        }} />
                    </IconButton>
                </Tooltip>
                {/* Add new nodes to the orchestration */}
                <Tooltip title='Add new node'>
                    <IconButton
                        id="add-node-button"
                        edge="end"
                        onClick={() => { }}
                        aria-label='Add node'
                        sx={{
                            background: (t) => t.palette.secondary.main,
                            marginLeft: 'auto',
                            marginRight: 1,
                            transition: 'brightness 0.2s ease-in-out',
                            '&:hover': {
                                filter: `brightness(105%)`,
                                background: (t) => t.palette.secondary.main,
                            },
                        }}
                    >
                        <AddIcon id="add-node-button-icon" sx={{ fill: 'white' }} />
                    </IconButton>
                </Tooltip>
                {/* Displays unlinked nodes */}
                <UnlinkedNodesDialog
                    open={isUnlinkedNodesOpen}
                    nodes={changedRoutine?.nodes ?? []} //TODO change to unlinkedNodes. This is only like this for testing
                    handleToggleOpen={toggleUnlinkedNodes}
                    handleDeleteNode={() => { }}
                />
            </Box> : null }
            <Box sx={{
                width: '100%',
                display: 'flex',
                flexDirection: 'column',
                position: 'fixed',
                bottom: '0',
            }}>
                <NodeGraph
                    scale={scale}
                    isEditing={isEditing}
                    labelVisible={true}
                    nodeDataMap={nodeDataMap}
                    links={changedRoutine?.nodeLinks ?? []}
                    handleDialogOpen={handleDialogOpen}
                />
                <OrchestrationBottomContainer
                    canCancelUpdate={!loading}
                    canUpdate={!loading && !isEqual(routine, changedRoutine)}
                    handleCancelRoutineUpdate={revertChanges}
                    handleRoutineUpdate={updateRoutine}
                    handleScaleChange={handleScaleChange}
                    hasPrevious={false}
                    hasNext={false}
                    isEditing={isEditing}
                    loading={loading}
                    scale={scale}
                    sliderColor={STATUS_COLOR[status.code]}
                    runState={OrchestrationRunState.Stopped}
                />
            </Box>
        </Box>
    )
};