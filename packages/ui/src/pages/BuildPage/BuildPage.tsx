import { Box, IconButton, Tooltip } from '@mui/material';
import { LinkDialog, NodeGraph, BuildBottomContainer, BuildInfoContainer, RoutineInfoDialog, UnlinkedNodesDialog } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { deleteArrayIndex, formatForUpdate, BuildDialogOption, BuildRunState, BuildStatus, Pubs, updateArray } from 'utils';
import {
    Add as AddIcon,
    AddLink as AddLinkIcon,
    Compress as CleanUpIcon,
    DeleteForever as DeleteIcon,
} from '@mui/icons-material';
import { Node, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, Routine } from 'types';
import isEqual from 'lodash/isEqual';
import { useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { BuildStatusObject } from 'components/graphs/NodeGraph/types';
import { MemberRole, NodeType } from 'graphql/generated/globalTypes';
import { BuildPageProps } from 'pages/types';
import _ from 'lodash';

/**
 * Status indicator and slider change color to represent routine's status
 */
const STATUS_COLOR = {
    [BuildStatus.Incomplete]: '#cde22c', // Yellow
    [BuildStatus.Invalid]: '#ff6a6a', // Red
    [BuildStatus.Valid]: '#00d51e', // Green
}

export const BuildPage = ({
    session,
}: BuildPageProps) => {
    // Get routine ID from URL
    const [, params] = useRoute(`${APP_LINKS.Build}/:id`);
    const id: string = useMemo(() => params?.id ?? '', [params]);
    // Queries routine data
    const { data: routineData } = useQuery<routine>(routineQuery, { variables: { input: { id } } });
    const [routine, setRoutine] = useState<Routine | null>(null);
    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null);
    useEffect(() => { setRoutine(routineData?.routine ?? null) }, [routineData]);
    // Routine mutator
    const [routineUpdate, { loading }] = useMutation<any>(routineUpdateMutation);
    // The routine's status (valid/invalid/incomplete)
    const [status, setStatus] = useState<BuildStatusObject>({ code: BuildStatus.Incomplete, messages: ['Calculating...'] });
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    const canEdit = useMemo<boolean>(() => [MemberRole.Admin, MemberRole.Owner].includes(routine?.role as MemberRole), [routine]);
    const language = 'en';

    // Open/close routine info drawer
    const [isRoutineInfoOpen, setIsRoutineInfoOpen] = useState<boolean>(false);
    const closeRoutineInfo = useCallback(() => setIsRoutineInfoOpen(false), []);
    // Open/close unlinked nodes drawer
    const [isUnlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState<boolean>(false);
    const toggleUnlinkedNodes = useCallback(() => setIsUnlinkedNodesOpen(curr => !curr), []);

    useEffect(() => {
        console.log('SETTING CHANGED ROUTINEEEEEEEEEE', routine);
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
     * Cleans up graph by removing empty columns and row gaps within columns.
     * Also adds end nodes to the end of each unfinished path
     */
    const cleanUpGraph = useCallback(() => {
        //TODO
    }, []);

    /**
     * Calculates:
     * - 2D array of positioned nodes data (to represent columns and rows)
     * - 1D array of unpositioned nodes data
     * - dictionary of positioned node IDs to their data
     * Also sets the status of the routine (valid/invalid/incomplete)
     */
    const { columns, nodesOffGraph, nodesById } = useMemo(() => {
        console.log('CALCULATING COLUMNS NODESOFFGRAPH NODESBYID', changedRoutine)
        if (!changedRoutine) return { columns: [], nodesOffGraph: [], nodesById: {} };
        const nodesOnGraph: Node[] = [];
        const nodesOffGraph: Node[] = [];
        const nodesById: { [id: string]: Node } = {};
        const statuses: [BuildStatus, string][] = []; // Holds all status messages, so multiple can be displayed
        // Loop through nodes and add to appropriate array (and also populate nodesById dictionary)
        for (const node of changedRoutine.nodes) {
            if (!_.isNil(node.columnIndex) && !_.isNil(node.rowIndex)) {
                nodesOnGraph.push(node);
            } else {
                nodesOffGraph.push(node);
            }
            nodesById[node.id] = node;
        }
        console.log('NODES OFF GRAPH', nodesOffGraph)
        // Now, perform a few checks to make sure that the columnIndexes and rowIndexes are valid
        // 1. Check that (columnIndex, rowIndex) pairs are all unique
        // First check
        // Remove duplicate values from positions dictionary
        const uniqueDict = _.uniqBy(nodesOnGraph, (n) => `${n.columnIndex}-${n.rowIndex}`);
        // Check if length of removed duplicates is equal to the length of the original positions dictionary
        if (uniqueDict.length !== Object.values(nodesOnGraph).length) {
            // Push to status
            setStatus({ code: BuildStatus.Invalid, messages: ['Ran into error determining node positions'] });
            // This is a critical error, so we'll remove all node positions and links
            setChangedRoutine({
                ...changedRoutine,
                nodes: changedRoutine.nodes.map(n => ({ ...n, columnIndex: null, rowIndex: null })),
                nodeLinks: [],
            })
            return { columns: [], nodesOffGraph: changedRoutine.nodes, nodesById: {} };
        }
        // Now perform checks to see if the routine can be run
        // 1. There is only one start node
        // 2. There is only one linked node which has no incoming edges, and it is the start node
        // 3. Every node that has no outgoing edges is an end node
        // 4. Validate loop TODO
        // 5. Validate redirects TODO
        // First check
        const startNodes = changedRoutine.nodes.filter(node => node.type === NodeType.Start);
        if (startNodes.length === 0) {
            statuses.push([BuildStatus.Invalid, 'No start node found']);
        }
        else if (startNodes.length > 1) {
            statuses.push([BuildStatus.Invalid, 'More than one start node found']);
        }
        // Second check
        const nodesWithoutIncomingEdges = nodesOnGraph.filter(node => changedRoutine.nodeLinks.every(link => link.toId !== node.id));
        if (nodesWithoutIncomingEdges.length === 0) {
            //TODO this would be fine with a redirect link
            statuses.push([BuildStatus.Invalid, 'Error determining start node']);
        }
        else if (nodesWithoutIncomingEdges.length > 1) {
            statuses.push([BuildStatus.Invalid, 'Nodes are not fully connected']);
        }
        // Third check
        const nodesWithoutOutgoingEdges = nodesOnGraph.filter(node => changedRoutine.nodeLinks.every(link => link.fromId !== node.id));
        if (nodesWithoutOutgoingEdges.length >= 0) {
            // Check that every node without outgoing edges is an end node
            if (nodesWithoutOutgoingEdges.some(node => node.type !== NodeType.End)) {
                statuses.push([BuildStatus.Invalid, 'Not all paths end with an end node']);
            }
        }
        // Performs checks which make the routine incomplete, but not invalid
        // 1. There are unpositioned nodes
        // First check
        if (nodesOffGraph.length > 0) {
            statuses.push([BuildStatus.Incomplete, 'Some nodes are not linked']);
        }
        // Before returning, send the statuses to the status object
        if (statuses.length > 0) {
            console.log('statuses', statuses)
            // Status sent is the worst status
            let code = BuildStatus.Incomplete;
            if (statuses.some(status => status[0] === BuildStatus.Invalid)) code = BuildStatus.Invalid;
            setStatus({ code, messages: statuses.map(status => status[1]) });
        } else {
            setStatus({ code: BuildStatus.Valid, messages: ['Routine is fully connected'] });
        }
        // Remove any links which reference unlinked nodes
        const goodLinks = changedRoutine.nodeLinks.filter(link => !nodesOffGraph.some(node => node.id === link.fromId || node.id === link.toId));
        // If routine was mutated, update the routine
        const finalNodes = [...nodesOnGraph, ...nodesOffGraph]
        const haveNodesChanged = !_.isEqual(finalNodes, changedRoutine.nodes);
        const haveLinksChanged = !_.isEqual(goodLinks, changedRoutine.nodeLinks);
        if (haveNodesChanged || haveLinksChanged) {
            setChangedRoutine({
                ...changedRoutine,
                nodes: finalNodes,
                nodeLinks: goodLinks,
            })
        }
        // Create 2D node data array, ordered by column. Each column is ordered by row index
        const columns: Node[][] = [];
        // Loop through positioned nodes
        for (const node of nodesOnGraph) {
            // Skips nodes without a columnIndex or rowIndex
            if (_.isNil(node.columnIndex) || _.isNil(node.rowIndex)) continue;
            // Add new column(s) if necessary
            while (columns.length <= node.columnIndex) {
                columns.push([]);
            }
            // Add node to column
            columns[node.columnIndex].push(node);
        }
        // Now sort each column by row index
        for (const column of columns) {
            column.sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
        }
        // Return
        console.log('COLUMNSSS', columns)
        return { columns, nodesOffGraph, nodesById };
    }, [changedRoutine]);

    const handleDialogOpen = useCallback((nodeId: string, dialog: BuildDialogOption) => {
        switch (dialog) {
            case BuildDialogOption.AddRoutineItem:
                break;
            case BuildDialogOption.ViewRoutineItem:
                setIsRoutineInfoOpen(true);
                break;
        }
    }, []);
    const handleAddLinkDialogOpen = useCallback((fromId?: string, toId?: string) => {
    }, []);

    const [isLinkDialogOpen, setIsLinkDialogOpen] = useState(false);
    const handleLinkDialogClose = useCallback(() => {
    }, []);

    /**
     * Deletes a link, without deleting any nodes. This may make the graph invalid.
     */
    const handleLinkDelete = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        setChangedRoutine({
            ...changedRoutine,
            nodeLinks: changedRoutine.nodeLinks.filter(l => l.id !== link.id),
        });
    }, [changedRoutine]);

    const handleScaleChange = (newScale: number) => { setScale(newScale) };

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
        setChangedRoutine({
            ...changedRoutine, translations: [
                { language: 'en', title },
            ]
        } as any);
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
     * Calculates the new set of links for an routine when a node is 
     * either deleted or unlinked. In certain cases, the new links can be 
     * calculated automatically.
     * @param nodeId - The ID of the node which is being deleted or unlinked
     * @param currLinks - The current set of links
     * @returns The new set of links
     */
    const calculateNewLinksList = useCallback((nodeId: string): NodeLink[] => {
        if (!changedRoutine) return [];
        const deletingLinks = changedRoutine.nodeLinks.filter(l => l.fromId === nodeId || l.toId === nodeId);
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
        else if (toNodeIds.length === 1) {
            fromNodeIds.forEach(fromId => { newLinks.push({ fromId, toId: toNodeIds[0] }) });
        }
        // NOTE: Every other case is ambiguous, so we can't auto-create create links
        // Delete old links
        let keptLinks = changedRoutine.nodeLinks.filter(l => !deletingLinks.includes(l));
        console.log('kept links', keptLinks);
        console.log('new links', newLinks);
        // Return new links combined with kept links
        return [...keptLinks, ...newLinks as any[]];
    }, [changedRoutine?.nodeLinks]);

    /**
     * Deletes a node, and all links connected to it. 
     * Also attemps to create new links to replace the deleted links.
     */
    const handleNodeDelete = useCallback((nodeId: string) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const linksList = calculateNewLinksList(nodeId);
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
        console.log('HANDLE NODE UNLINK', nodeId, changedRoutine?.nodeLinks.length);
        if (!changedRoutine) return;
        const linksList = calculateNewLinksList(nodeId);
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        console.log('node index', nodeIndex);
        console.log('links list', linksList);
        setChangedRoutine({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                rowIndex: null,
                columnIndex: null,
            }),
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

    /**
     * Adds a routine list item to a routine list
     */
    const handleRoutineListItemAdd = useCallback((nodeId: string, data: NodeDataRoutineListItem) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const routineList: NodeDataRoutineList = changedRoutine.nodes[nodeIndex].data as NodeDataRoutineList;
        setChangedRoutine({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                data: {
                    ...routineList,
                    routines: [...routineList.routines, data],
                }
            }),
        });
    }, [changedRoutine]);

    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            minHeight: '100%',
            height: '100%',
            width: '100%',
        }}>
            {/* Popup for creating new links */}
            {changedRoutine ? <LinkDialog
                handleClose={handleLinkDialogClose}
                handleDelete={handleLinkDelete}
                isAdd={true}
                isOpen={isLinkDialogOpen}
                language={language}
                link={undefined}
                routine={changedRoutine}
            // partial={ }
            /> : null}
            {/* Popup for deleting nodes (or the entire routine) */}
            {/* <DeleteNodeDialog
            /> */}
            {/* Displays routine information when you click on a routine list item*/}
            <RoutineInfoDialog
                open={isRoutineInfoOpen}
                routineInfo={changedRoutine}
                onClose={closeRoutineInfo}
            />
            {/* Displays main routine's information and some buttons */}
            <BuildInfoContainer
                canEdit={canEdit}
                handleRoutineUpdate={updateRoutine}
                handleStartEdit={startEditing}
                handleTitleUpdate={updateRoutineTitle}
                isEditing={isEditing}
                language={language}
                routine={changedRoutine}
                session={session}
                status={status}
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
                {/* Delete node (or whole routine) */}
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
                {/* Clean up graph */}
                <Tooltip title='Clean up graph'>
                    <IconButton
                        id="clean-graph-button"
                        edge="end"
                        onClick={() => { }}
                        aria-label='Clean up graph'
                        sx={{
                            background: '#ab9074',
                            marginLeft: 'auto',
                            marginRight: 1,
                            transition: 'brightness 0.2s ease-in-out',
                            '&:hover': {
                                filter: `brightness(105%)`,
                                background: '#ab9074',
                            },
                        }}
                    >
                        <CleanUpIcon id="clean-up-button-icon" sx={{ fill: 'white' }} />
                    </IconButton>
                </Tooltip>
                {/* Add new links to the routine */}
                <Tooltip title='Add new link'>
                    <IconButton
                        id="add-link-button"
                        edge="end"
                        onClick={() => { }}
                        aria-label='Add link'
                        sx={{
                            background: '#9e3984',
                            marginRight: 1,
                            transition: 'brightness 0.2s ease-in-out',
                            '&:hover': {
                                filter: `brightness(105%)`,
                                background: '#9e3984',
                            },
                        }}
                    >
                        <AddLinkIcon id="add-link-button-icon" sx={{ fill: 'white' }} />
                    </IconButton>
                </Tooltip>
                {/* Add new nodes to the routine */}
                <Tooltip title='Add new node'>
                    <IconButton
                        id="add-node-button"
                        edge="end"
                        onClick={() => { }}
                        aria-label='Add node'
                        sx={{
                            background: (t) => t.palette.secondary.main,
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
                    nodes={nodesOffGraph}
                    handleToggleOpen={toggleUnlinkedNodes}
                    handleNodeDelete={handleNodeDelete}
                    handleDialogOpen={handleDialogOpen}
                    handleNodeUnlink={handleNodeUnlink}
                    handleRoutineListItemAdd={handleRoutineListItemAdd}
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
                    handleDialogOpen={handleDialogOpen}
                    handleLinkCreate={handleLinkCreate}
                    handleLinkUpdate={handleLinkUpdate}
                    handleLinkDelete={handleLinkDelete}
                    handleNodeDelete={handleNodeDelete}
                    handleNodeUpdate={handleNodeUpdate}
                    handleNodeUnlink={handleNodeUnlink}
                    handleRoutineListItemAdd={handleRoutineListItemAdd}
                    isEditing={isEditing}
                    labelVisible={true}
                    language={language}
                    links={changedRoutine?.nodeLinks ?? []}
                    columns={columns}
                    nodesById={nodesById}
                    scale={scale}
                />
                <BuildBottomContainer
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
                    runState={BuildRunState.Stopped}
                />
            </Box>
        </Box>
    )
};