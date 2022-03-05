import { Box, IconButton, Tooltip } from '@mui/material';
import { NodeGraph, OrchestrationBottomContainer, OrchestrationInfoContainer, RoutineInfoDialog, UnlinkedNodesDialog } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { deleteArrayIndex, formatForUpdate, OrchestrationDialogOption, OrchestrationRunState, OrchestrationStatus, Pubs, updateArray } from 'utils';
import {
    Add as AddIcon,
    DeleteForever as DeleteIcon,
} from '@mui/icons-material';
import { Node, NodeLink, Routine } from 'types';
import isEqual from 'lodash/isEqual';
import { useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { OrchestrationStatusObject } from 'components/graphs/NodeGraph/types';
import { NodeType } from 'graphql/generated/globalTypes';
import { RoutineOrchestratorPageProps } from 'pages/types';
import _ from 'lodash';

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
    const [status, setStatus] = useState<OrchestrationStatusObject>({ code: OrchestrationStatus.Incomplete, messages: ['Calculating...'] });
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    const canEdit = true;// TODO useMemo(() => Boolean(session?.id) && routine?.owner?.id === session?.id, [routine]); //TODO handle organization

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
     * 1st return - Array of nodes that are linked to the routine
     * 2nd return - Array of nodes that are not linked to the routine
     * A graph with no unlinked nodes is considered valid
     */
    const [linkedNodes, unlinkedNodes] = useMemo<[Node[], Node[]]>(() => {
        console.log('calculating linked nodes start', changedRoutine?.nodes)
        if (!changedRoutine) return [[], []];
        const linkedNodes: Node[] = [];
        const unlinkedNodes: Node[] = [];
        const statuses: [OrchestrationStatus, string][] = []; // All orchestration statuses
        // First, set all nodes that have a columnIndex and rowIndex to be linked
        for (const node of changedRoutine.nodes) {
            if (node.columnIndex !== null && node.columnIndex !== undefined && node.rowIndex !== null && node.rowIndex !== undefined) {
                linkedNodes.push(node);
            }
            else {
                unlinkedNodes.push(node);
            }
        }
        console.log('a linkednode', linkedNodes)

        // Now, perform a few checks to make sure that the columnIndexes and rowIndexes are valid
        // 1. Check that (columnIndex, rowIndex) pairs are all unique
        // 2. columnIndexes are not missing. For example, if the indexes go [0, 3, 1, 5, 6], then
        // they should be converted to [0, 2, 1, 3, 4]
        // 3. rowIndexes don't exceed the number of nodes in the largest column, minus 1
        // First check
        const points: { x: number, y: number }[] = linkedNodes.map(node => ({ x: node.columnIndex as number, y: node.rowIndex as number }));
        const uniquePoints = new Set(points);
        if (uniquePoints.size !== points.length) {
            setStatus({ code: OrchestrationStatus.Invalid, messages: ['Ran into error determining node positions'] });
            // Add all nodes to unlinkedNodes
            unlinkedNodes.push(...linkedNodes);
            console.log('failed unique points', points, uniquePoints)
            // Update changedRoutine to reflect the new state
            setChangedRoutine({
                ...changedRoutine,
                nodeLinks: [],
            });
            return [[], unlinkedNodes];
        }
        // Second check
        // Sort linkedNodes by columnIndex
        linkedNodes.sort((a, b) => (a.columnIndex as number) - (b.columnIndex as number));
        // Convert columnIndexes to be sequential. Indexes may repeat, but not be missing
        // Find number of unique indexes
        const uniqueColumnIndexes = new Set(linkedNodes.map(node => node.columnIndex as number));
        console.log('uniqueColumnIndexes check', uniqueColumnIndexes, uniqueColumnIndexes.size !== linkedNodes.length)
        // For each unique index, replace all occurrences with a sequential index
        let columnIndex: number = 0;
        for (const uniqueIndex of uniqueColumnIndexes) {
            linkedNodes.forEach(node => {
                if (node.columnIndex === uniqueIndex) {
                    node = { ...node, columnIndex };
                }
            });
            columnIndex++;
        }
        // Third check
        // Count the number of occurrences of each columnIndex
        const columnCounts: { [key: number]: number } = {};
        for (const point of points) {
            if (columnCounts[point.x]) {
                columnCounts[point.x]++;
            }
            else {
                columnCounts[point.x] = 1;
            }
        }
        // Make sure no rowIndexes exceed the number of nodes in the largest column
        const maxColumnCount = Math.max(...Object.values(columnCounts));
        const badNodes = linkedNodes.filter(node => (node.rowIndex as number) >= maxColumnCount);
        if (badNodes.length > 0) {
            // Add all nodes to unlinkedNodes
            unlinkedNodes.push(...linkedNodes);
        }

        // Now perform checks to see if the orchestration can be run
        // 1. There is only one start node
        // 2. There is only one linked node which has no incoming edges, and it is the start node
        // 3. Every node that has no outgoing edges is an end node
        // TODO validate loops and redirects
        // First check
        const startNodes = linkedNodes.filter(node => node.type === NodeType.Start);
        if (startNodes.length === 0) {
            statuses.push([OrchestrationStatus.Invalid, 'No start node found']);
        }
        else if (startNodes.length > 1) {
            statuses.push([OrchestrationStatus.Invalid, 'More than one start node found']);
        }
        // Second check
        const nodesWithoutIncomingEdges = linkedNodes.filter(node => changedRoutine.nodeLinks.every(link => link.toId !== node.id));
        if (nodesWithoutIncomingEdges.length === 0) {
            //TODO this would be fine with a redirect link
            statuses.push([OrchestrationStatus.Invalid, 'Error determining start node']);
        }
        else if (nodesWithoutIncomingEdges.length > 1) {
            statuses.push([OrchestrationStatus.Invalid, 'Nodes are not fully connected']);
        }
        // Third check
        const nodesWithoutOutgoingEdges = linkedNodes.filter(node => changedRoutine.nodeLinks.every(link => link.fromId !== node.id));
        if (nodesWithoutOutgoingEdges.length >= 0) {
            // Check that every node without outgoing edges is an end node
            if (nodesWithoutOutgoingEdges.some(node => node.type !== NodeType.End)) {
                statuses.push([OrchestrationStatus.Invalid, 'Not all paths end with an end node']);
            }
        }

        // Performs checks which make the routine incomplete, but not invalid
        // 1. There are unlinked nodes
        // First check
        if (unlinkedNodes.length > 0) {
            statuses.push([OrchestrationStatus.Incomplete, 'Some nodes are not linked']);
        }

        // Before returning, send the statuses to the status object
        if (statuses.length > 0) {
            console.log('statuses', statuses)
            // Status sent is the worst status
            let code = OrchestrationStatus.Incomplete;
            if (statuses.some(status => status[0] === OrchestrationStatus.Invalid)) code = OrchestrationStatus.Invalid;
            setStatus({ code, messages: statuses.map(status => status[1]) });
        } else {
            setStatus({ code: OrchestrationStatus.Valid, messages: ['Routine is fully connected'] });
        }

        // Remove any links which reference unlinked nodes
        const goodLinks = changedRoutine.nodeLinks.filter(link => !unlinkedNodes.some(node => node.id === link.fromId || node.id === link.toId));
        // If routine was mutated, update the routine
        const finalNodes = [...linkedNodes, ...unlinkedNodes]
        const haveNodesChanged = !_.isEqual(finalNodes, changedRoutine.nodes);
        const haveLinksChanged = !_.isEqual(goodLinks, changedRoutine.nodeLinks);
        if (haveNodesChanged || haveLinksChanged) {
            setChangedRoutine({
                ...changedRoutine,
                nodes: [...linkedNodes, ...unlinkedNodes],
                nodeLinks: goodLinks,
            })
        }

        // Return the linked and unlinked nodes
        console.log('RETURNING LINKED AND UNLINKED NODES', linkedNodes, unlinkedNodes);
        return [linkedNodes, unlinkedNodes];
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
            input: formatForUpdate(routine, changedRoutine, ['tags'], ['nodes', 'nodeLinks']),
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
        setIsEditing(false);
    }, [routine])

    /**
     * Creates a link between two nodes which already exist in the linked routine. 
     * This assumes that the link is valid.
     */
    const handleLinkCreate = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        setChangedRoutine({
            ...changedRoutine,
            nodeLinks: [...changedRoutine.nodeLinks, link]
        });
    }, [changedRoutine]);

    /**
     * Updates an existing link between two nodes
     */
    const handleLinkUpdate = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        const linkIndex = changedRoutine.nodeLinks.findIndex(l => l.id === link.id);
        if (linkIndex === -1) return;
        setChangedRoutine({
            ...changedRoutine,
            nodeLinks: updateArray(changedRoutine.nodeLinks, linkIndex, link),
        });
    }, [changedRoutine]);

    /**
     * Calculates the new set of links for an orchestration when a node is 
     * either deleted or unlinked. In certain cases, the new links can be 
     * calculated automatically.
     * @param nodeId - The ID of the node which is being deleted or unlinked
     * @param currLinks - The current set of links
     * @returns The new set of links
     */
    const calculateNewLinksList = useCallback((nodeId: string, currLinks: NodeLink[]): NodeLink[] => {
        const deletingLinks = currLinks.filter(l => l.fromId === nodeId || l.toId === nodeId);
        const newLinks: Partial<NodeLink>[] = [];
        // Find all "from" and "to" nodes in the deleting links
        const fromNodeIds = deletingLinks.map(l => l.fromId).filter(id => id !== nodeId);
        const toNodeIds = deletingLinks.map(l => l.toId).filter(id => id !== nodeId);
        console.log('deleting links', deletingLinks);
        console.log('from and to ids', fromNodeIds, toNodeIds);
        // If there is only one "from" node, create a link between it and every "to" node
        if (fromNodeIds.length === 1) {
            toNodeIds.forEach(toId => { newLinks.push({ fromId: fromNodeIds[0], toId }) });
        }
        // If there is only one "to" node, create a link between it and every "from" node
        if (toNodeIds.length === 1) {
            fromNodeIds.forEach(fromId => { newLinks.push({ fromId, toId: toNodeIds[0] }) });
        }
        // NOTE: Every other case is ambiguous, so we can't auto-create create links
        // Delete old links
        let keptLinks = currLinks.filter(l => !deletingLinks.includes(l));
        console.log('kept links', keptLinks);
        console.log('new links', newLinks);
        // Return new links combined with kept links
        return [...keptLinks, ...newLinks as any[]];
    }, []);

    /**
     * Deletes a node, and all links connected to it. 
     * Also attemps to create new links to replace the deleted links.
     */
    const handleNodeDelete = useCallback((nodeId: string) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const linksList = calculateNewLinksList(nodeId, changedRoutine.nodeLinks);
        setChangedRoutine({
            ...changedRoutine,
            nodes: deleteArrayIndex(changedRoutine.nodes, nodeIndex),
            nodeLinks: linksList,
        });
    }, [changedRoutine]);

    /**
     * Unlinks a node. This means the node is still in the routine, but every link associated with 
     * it is removed.
     */
    const handleNodeUnlink = useCallback((nodeId: string) => {
        if (!changedRoutine) return;
        const linksList = calculateNewLinksList(nodeId, changedRoutine.nodeLinks);
        setChangedRoutine({
            ...changedRoutine,
            nodeLinks: linksList,
        });
    }, [changedRoutine]);

    /**
     * Updates a node's data
     */
    const handleNodeUpdate = useCallback((node: Node) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === node.id);
        if (nodeIndex === -1) return;
        setChangedRoutine({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, node),
        });
    }, [changedRoutine]);

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
                    nodes={unlinkedNodes}
                    handleToggleOpen={toggleUnlinkedNodes}
                    handleDeleteNode={() => { }}
                />
            </Box> : null}
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
                    nodes={linkedNodes}
                    links={changedRoutine?.nodeLinks ?? []}
                    handleDialogOpen={handleDialogOpen}
                    handleLinkCreate={handleLinkCreate}
                    handleLinkUpdate={handleLinkUpdate}
                    handleNodeDelete={handleNodeDelete}
                    handleNodeUpdate={handleNodeUpdate}
                    handleNodeUnlink={handleNodeUnlink}
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