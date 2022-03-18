import { Box, IconButton, Tooltip } from '@mui/material';
import { LinkDialog, NodeGraph, BuildBottomContainer, BuildInfoContainer, SubroutineInfoDialog, UnlinkedNodesDialog, AddSubroutineDialog, AddAfterLinkDialog, AddBeforeLinkDialog, DeleteRoutineDialog } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineDeleteOneMutation, routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { deleteArrayIndex, formatForUpdate, BuildAction, BuildRunState, BuildStatus, Pubs, updateArray, getTranslation } from 'utils';
import {
    AddLink as AddLinkIcon,
    Compress as CleanUpIcon,
} from '@mui/icons-material';
import { Node, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, Routine } from 'types';
import isEqual from 'lodash/isEqual';
import { useLocation, useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { BuildStatusObject } from 'components/graphs/NodeGraph/types';
import { MemberRole, NodeType } from 'graphql/generated/globalTypes';
import { BuildPageProps } from 'pages/types';
import _ from 'lodash';
import { v4 as uuidv4 } from 'uuid';
import { BaseObjectAction } from 'components/dialogs/types';

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
    const [, setLocation] = useLocation();
    // Get routine ID from URL
    const [, params] = useRoute(`${APP_LINKS.Build}/:id`);
    const id: string = useMemo(() => params?.id ?? '', [params]);
    // Queries routine data
    const { data: routineData, loading: loadingRead } = useQuery<routine>(routineQuery, { variables: { input: { id } } });
    const [routine, setRoutine] = useState<Routine | null>(null);
    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null);
    useEffect(() => { console.log('got routine data. setting routine', routineData); setRoutine(routineData?.routine ?? null) }, [routineData]);
    // Routine mutators
    const [routineUpdate, { loading: loadingUpdate }] = useMutation<any>(routineUpdateMutation);
    const [routineDelete, { loading: loadingDelete }] = useMutation<any>(routineDeleteOneMutation);
    const loading = useMemo(() => loadingRead || loadingUpdate || loadingDelete, [loadingRead, loadingUpdate, loadingDelete]);
    // The routine's status (valid/invalid/incomplete)
    const [status, setStatus] = useState<BuildStatusObject>({ code: BuildStatus.Incomplete, messages: ['Calculating...'] });
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    const canEdit = useMemo<boolean>(() => [MemberRole.Admin, MemberRole.Owner].includes(routine?.role as MemberRole), [routine]);
    const language = 'en';

    // Open/close unlinked nodes drawer
    const [isUnlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState<boolean>(false);
    const toggleUnlinkedNodes = useCallback(() => setIsUnlinkedNodesOpen(curr => !curr), []);

    useEffect(() => {
        console.log('SETTING CHANGED ROUTINEEEEEEEEEE', routine);
        setChangedRoutine(routine);
    }, [routine]);

    console.log('changed routineeeeeeeeeee on renderrrrrrr', changedRoutine);

    // Add subroutine dialog
    const [addSubroutineNode, setAddSubroutineNode] = useState<string | null>(null);
    const closeAddSubroutineDialog = useCallback(() => { setAddSubroutineNode(null); }, []);

    // Edit subroutine dialog
    const [editSubroutineNode, setEditSubroutineNode] = useState<string | null>(null);
    const closeEditSubroutineDialog = useCallback(() => { setEditSubroutineNode(null); }, []);

    // "Add after" link dialog when there is more than one link (i.e. can't be done automatically)
    const [addAfterLinkNode, setAddAfterLinkNode] = useState<string | null>(null);
    const closeAddAfterLinkDialog = useCallback(() => { setAddAfterLinkNode(null); }, []);

    // "Add before" link dialog when there is more than one link (i.e. can't be done automatically)
    const [addBeforeLinkNode, setAddBeforeLinkNode] = useState<string | null>(null);
    const closeAddBeforeLinkDialog = useCallback(() => { setAddBeforeLinkNode(null); }, []);

    // Move node dialog for context menu (mainly for accessibility)
    const [moveNode, setMoveNode] = useState<string | null>(null);
    const closeMoveNodeDialog = useCallback(() => { setMoveNode(null); }, []);

    // Open boolean for delete routine confirmation
    const [deleteOpen, setDeleteOpen] = useState(false);
    const openDelete = () => setDeleteOpen(true);
    const closeDelete = () => setDeleteOpen(false);

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
            console.log('changed routine loop', node)
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
            console.log('uh oh spaghetti o', nodesWithoutIncomingEdges, nodesOnGraph)
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

    // Subroutine info drawer
    const [selectedSubroutine, setSelectedSubroutine] = useState<Routine | null>(null);
    const handleSubroutineOpen = useCallback((nodeId: string, subroutineId: string) => {
        const node = nodesById[nodeId];
        if (node) {
            const subroutine = (node.data as NodeDataRoutineList).routines.find(r => r.id === subroutineId);
            if (subroutine) {
                setSelectedSubroutine(subroutine.routine as any);
            }
        }
    }, [nodesById]);
    const closeRoutineInfo = useCallback(() => setSelectedSubroutine(null), []);

    const [isLinkDialogOpen, setIsLinkDialogOpen] = useState(false);
    const openLinkDialog = useCallback(() => setIsLinkDialogOpen(true), []);
    const handleLinkDialogClose = useCallback((link?: NodeLink) => {
        if (!changedRoutine) return;
        setIsLinkDialogOpen(false);
        // If no link data, return
        if (!link) return;
        // Upsert link
        const newLinks = [...changedRoutine.nodeLinks];
        const existingLinkIndex = newLinks.findIndex(l => l.fromId === link.fromId && l.toId === link.toId);
        if (existingLinkIndex >= 0) {
            newLinks[existingLinkIndex] = { ...link };
        } else {
            newLinks.push(link);
        }
        setChangedRoutine({
            ...changedRoutine,
            nodeLinks: newLinks,
        });
    }, [changedRoutine]);

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
        const input: any = formatForUpdate(routine, changedRoutine, ['tags', 'nodes.data.routines.routine'], ['nodes', 'nodeLinks', 'node.data.routines'])
        // If routine belongs to an organization, add organizationId to input
        if (routine?.owner?.__typename === 'Organization') {
            input.organizationId = routine.owner.id;
        };
        // If nodes have a data create/update, convert to nodeEnd or nodeRoutineList (i.e. deconstruct union)
        const relationFields = ['Create', 'Update'];
        for (const iField of relationFields) {
            if (!input.hasOwnProperty(`nodes${iField}`)) continue;
            input[`nodes${iField}`] = input[`nodes${iField}`].map(node => {
                // Find full node data
                const fullNode = changedRoutine.nodes.find(n => n.id === node.id);
                if (!fullNode) return node;
                // If end node, convert `data${jField}` to `nodeEnd${jField}`
                if (fullNode.type === NodeType.End) {
                    for (const jField of relationFields) {
                        if (!node.hasOwnProperty(`data${jField}`)) continue;
                        node[`nodeEnd${jField}`] = node[`data${jField}`];
                        delete node[`data${jField}`];
                    }
                }
                // If routine list node, convert `data${jField}` to `nodeRoutineList${jField}`
                if (fullNode.type === NodeType.RoutineList) {
                    for (const jField of relationFields) {
                        if (!node.hasOwnProperty(`data${jField}`)) continue;
                        node[`nodeRoutineList${jField}`] = node[`data${jField}`];
                        delete node[`data${jField}`];
                    }
                }
                return node;
            });
        }
        console.log('going to update routine', input)
        mutationWrapper({
            mutation: routineUpdate,
            input,
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
     * Deletes the entire routine. Assumes confirmation was already given.
     */
    const deleteRoutine = useCallback(() => {
        if (!routine) return;
        mutationWrapper({
            mutation: routineDelete,
            input: { id: routine.id },
            successMessage: () => 'Routine deleted.',
            onSuccess: () => { setLocation(APP_LINKS.Home) },
        })
    }, [routine, routineDelete])

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
     * Generates a new node object, but doens't add it to the routine
     */
    const generateNewNode = useCallback((columnIndex: number | null, rowIndex: number | null) => {
        const newNode: Partial<Node> = {
            id: uuidv4(),
            type: NodeType.RoutineList,
            rowIndex,
            columnIndex,
            data: {
                isOrdered: false,
                isOptional: false,
                routines: [],
            } as any,
        }
        return newNode;
    }, []);

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
     * Deletes a subroutine from a node
     */
    const handleSubroutineDelete = useCallback((nodeId: string, subroutineId: string) => {
        console.log('in handlesubroutine delete', nodeId, subroutineId, changedRoutine)
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        console.log('in handlesubroutine nodeIndex', nodeIndex)
        if (nodeIndex === -1) return;
        const node = changedRoutine.nodes[nodeIndex];
        const subroutineIndex = (node.data as NodeDataRoutineList).routines.findIndex((item: NodeDataRoutineListItem) => item.id === subroutineId);
        console.log('subroutineindex', subroutineIndex, node)
        if (subroutineIndex === -1) return;
        const newRoutineList = deleteArrayIndex((node.data as NodeDataRoutineList).routines, subroutineIndex);
        setChangedRoutine({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                ...node,
                data: {
                    ...node.data,
                    routines: newRoutineList,
                }
            }),
        });
    }, [changedRoutine]);

    /**
     * Drops or unlinks a node
     */
    const handleNodeDrop = useCallback((nodeId: string, columnIndex: number | null, rowIndex: number | null) => {
        console.log('HANDLE NODE DROP', nodeId, columnIndex, rowIndex);
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        // If columnIndex and rowIndex null, then it is being unlinked
        if (columnIndex === null && rowIndex === null) {
            const linksList = calculateNewLinksList(nodeId);
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
        }
        // If one or the other is null, then there must be an error
        else if (columnIndex === null || rowIndex === null) {
            PubSub.publish(Pubs.Snack, { message: 'Error: Invalid drop location.', severity: 'errror' });
        }
        // Otherwise, is a drop
        else {
            console.log('ITS A DROP', nodeIndex)
            let updatedNodes = [...changedRoutine.nodes];
            // If dropped into an existing column, shift rows in dropped column that are below the dropped node
            if (changedRoutine.nodes.some(n => n.columnIndex === columnIndex)) {
                console.log('shift rows below');
                updatedNodes = updatedNodes.map(n => {
                    if (n.columnIndex === columnIndex && n.rowIndex !== null && n.rowIndex >= rowIndex) {
                        console.log('shifting a row', n)
                        return { ...n, rowIndex: n.rowIndex + 1 }
                    }
                    return n;
                });
            }
            // If the column the node was from is now empty, then shift all columns after it
            const originalColumnIndex = changedRoutine.nodes[nodeIndex].columnIndex;
            const isRemovingColumn = originalColumnIndex !== null && changedRoutine.nodes.filter(n => n.columnIndex === originalColumnIndex).length === 1;
            console.log('original column index', originalColumnIndex);
            if (isRemovingColumn) {
                console.log('shift columns');
                updatedNodes = updatedNodes.map(n => {
                    if (n.columnIndex !== null && n.columnIndex > originalColumnIndex) {
                        return { ...n, columnIndex: n.columnIndex - 1 }
                    }
                    return n;
                });
            }
            console.log('updated nodes a', updatedNodes);
            const updated = updateArray(updatedNodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                columnIndex: (isRemovingColumn && originalColumnIndex < columnIndex) ? columnIndex - 1 : columnIndex,
                rowIndex,
            })
            console.log('updated nodes b', updated);
            console.log('testttttt', { ...changedRoutine, nodes: updated });
            // Update the routine
            setChangedRoutine({
                ...changedRoutine,
                nodes: updated,
            });
        }
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
     * Inserts a new routine list node along an edge
     */
    const handleNodeInsert = useCallback((link: NodeLink) => {
        console.log('HANDLE NODE INSERT', link);
        if (!changedRoutine) return;
        // Find link index
        const linkIndex = changedRoutine.nodeLinks.findIndex(l => l.id === link.id);
        // Delete link
        const linksList = deleteArrayIndex(changedRoutine.nodeLinks, linkIndex);
        // Find "to" node. New node will be placed in its row and column
        const toNode = changedRoutine.nodes.find(n => n.id === link.toId);
        if (!toNode) {
            PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'Error' });
            return;
        }
        const { columnIndex, rowIndex } = toNode;
        // Move every node starting from the "to" node to the right by one
        const nodesList = changedRoutine.nodes.map(n => {
            if (!_.isNil(n.columnIndex) && n.columnIndex >= (columnIndex ?? 0)) {
                return { ...n, columnIndex: n.columnIndex + 1 };
            }
            return n;
        });
        // Create new routine list node
        const newNode: Partial<Node> = generateNewNode(columnIndex, rowIndex);
        // Find every node 
        // Create two new links
        const newLinks: Partial<NodeLink>[] = [
            { fromId: link.fromId, toId: newNode.id },
            { fromId: newNode.id, toId: link.toId },
        ];
        // Insert new node and links
        const newRoutine = {
            ...changedRoutine,
            nodes: [...nodesList, newNode as any],
            nodeLinks: [...linksList, ...newLinks as any],
        };
        setChangedRoutine(newRoutine);
    }, [changedRoutine]);

    /**
     * Adds a routine list item to a routine list
     */
    const handleRoutineListItemAdd = useCallback((nodeId: string, routine: Routine) => {
        console.log('HANDLE ROUTINE LIST ITEM ADD', nodeId, routine, changedRoutine);
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        console.log('nooooode index', nodeIndex);
        if (nodeIndex === -1) return;
        const routineList: NodeDataRoutineList = changedRoutine.nodes[nodeIndex].data as NodeDataRoutineList;
        console.log('handle routine list item add a', routineList);
        let routineItem: NodeDataRoutineListItem = {
            // id: uuidv4(),
            isOptional: true,
            routine,
        } as any
        if (routineList.isOrdered) routineItem.index = routineList.routines.length
        console.log('handle routine list item add b', routineItem);
        setChangedRoutine({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                data: {
                    ...routineList,
                    routines: [...routineList.routines, routineItem],
                }
            }),
        });
    }, [changedRoutine]);

    /**
     * Add a new routine list AFTER a node
     */
    const handleAddAfter = useCallback((nodeId: string) => {
        console.log('handle add after', nodeId)
        if (!changedRoutine) return;
        // Find links where this node is the "from" node
        const links = changedRoutine.nodeLinks.filter(l => l.fromId === nodeId);
        // If multiple links, open a dialog to select which one to add after
        if (links.length > 1) {
            console.log('opening after dialog')
            setAddAfterLinkNode(nodeId);
            return;
        }
        // If only one link, add after that link
        else if (links.length === 1) {
            const link = links[0];
            console.log('inserting node on link', link)
            handleNodeInsert(link);
        }
        // If no links, create link and node
        else {
            console.log('creating new link for node');
            const newNode = generateNewNode(null, null);
            const newLink = { fromId: nodeId, toId: newNode.id };
            setChangedRoutine({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [changedRoutine]);

    /**
     * Add a new routine list BEFORE a node
     */
    const handleAddBefore = useCallback((nodeId: string) => {
        console.log('handle add before', nodeId, changedRoutine)
        if (!changedRoutine) return;
        // Find links where this node is the "to" node
        const links = changedRoutine.nodeLinks.filter(l => l.toId === nodeId);
        // If multiple links, open a dialog to select which one to add before
        if (links.length > 1) {
            console.log('opening before dialog')
            setAddBeforeLinkNode(nodeId);
            return;
        }
        // If only one link, add before that link
        else if (links.length === 1) {
            const link = links[0];
            console.log('inserting node on link', link)
            handleNodeInsert(link);
        }
        // If no links, create link and node
        else {
            console.log('creating new link for node');
            const newNode = generateNewNode(null, null);
            const newLink = { fromId: newNode.id, toId: nodeId };
            setChangedRoutine({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [changedRoutine]);

    /**
     * Updates the current selected subroutine
     */
    const handleSubroutineUpdate = useCallback((updatedSubroutine: Routine) => {
        console.log('HANDLE SUBROUTINE UPDATE', updatedSubroutine);
        if (!changedRoutine) return;
        setChangedRoutine({
            ...changedRoutine,
            nodes: changedRoutine.nodes.map((n: Node) => {
                if (n.type === NodeType.RoutineList && (n.data as NodeDataRoutineList).routines.some(r => r.routine.id === updatedSubroutine.id)) {
                    return {
                        ...n,
                        data: {
                            ...n.data,
                            routines: (n.data as NodeDataRoutineList).routines.map(r => {
                                if (r.routine.id === updatedSubroutine.id) {
                                    return { ...r, routine: updatedSubroutine };
                                }
                                return r;
                            }),
                        },
                    };
                }
                return n;
            }),
        } as any);
    }, [changedRoutine]);

    /**
     * Navigates to a subroutine's build page. Fist checks if there are unsaved changes
     */
    const handleSubroutineViewFull = useCallback(() => {
        console.log('HANDLE SUBROUTINE FULL VIEW', selectedSubroutine);
        if (!selectedSubroutine) return;
        if (!isEqual(routine, changedRoutine)) {
            PubSub.publish(Pubs.Snack, { message: 'You have unsaved changes. Please save or discard them before navigating to another routine.' });
            return;
        }
        setLocation(`${APP_LINKS.Build}/${selectedSubroutine.id}`);
    }, [selectedSubroutine, changedRoutine, routine]);

    const handleAction = useCallback((action: BuildAction, nodeId: string, subroutineId?: string) => {
        console.log('in handleaction', action, nodeId, subroutineId);
        switch (action) {
            case BuildAction.AddSubroutine:
                setAddSubroutineNode(nodeId);
                break;
            case BuildAction.DeleteNode:
                handleNodeDelete(nodeId);
                break;
            case BuildAction.DeleteSubroutine:
                handleSubroutineDelete(nodeId, subroutineId ?? '');
                break;
            case BuildAction.EditSubroutine:
                setEditSubroutineNode(nodeId);
                break;
            case BuildAction.OpenSubroutine:
                handleSubroutineOpen(nodeId, subroutineId ?? '');
                break;
            case BuildAction.UnlinkNode:
                handleNodeDrop(nodeId, null, null);
                break;
            case BuildAction.AddAfterNode:
                handleAddAfter(nodeId);
                break;
            case BuildAction.AddBeforeNode:
                handleAddBefore(nodeId);
                break;
            case BuildAction.MoveNode:
                setMoveNode(nodeId);
                break;
        }
    }, [setAddSubroutineNode, setEditSubroutineNode, handleNodeDelete, handleSubroutineDelete, handleSubroutineOpen, handleNodeDrop, handleAddAfter, handleAddBefore, setMoveNode]);

    const handleRoutineAction = useCallback((action: BaseObjectAction) => {
        switch (action) {
            case BaseObjectAction.Copy:
                //TODO
                break;
            case BaseObjectAction.Delete:
                //TODO
                break;
            case BaseObjectAction.Downvote:
                openDelete();
                break;
            case BaseObjectAction.Edit:
                //TODO
                break;
            case BaseObjectAction.Fork:
                //TODO
                break;
            case BaseObjectAction.Report:
                //TODO
                break;
            case BaseObjectAction.Share:
                //TODO
                break;
            case BaseObjectAction.Star:
                //TODO
                break;
            case BaseObjectAction.Stats:
                //TODO
                break;
            case BaseObjectAction.Unstar:
                //TODO
                break;
            case BaseObjectAction.Update:
                updateRoutine();
                break;
            case BaseObjectAction.UpdateCancel:
                setChangedRoutine(routine);
                break;
            case BaseObjectAction.Upvote:
                //TODO
                break;
        }
    }, [routine, updateRoutine, openDelete]);

    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            minHeight: '100%',
            height: '100%',
            width: '100%',
        }}>
            {/* Delete routine confirmation dialog */}
            <DeleteRoutineDialog
                isOpen={deleteOpen}
                routineName={getTranslation(changedRoutine, 'title', [language]) ?? ''}
                handleClose={closeDelete}
                handleDelete={deleteRoutine}
            />
            {/* Popup for adding new subroutines */}
            {addSubroutineNode && <AddSubroutineDialog
                handleAdd={handleRoutineListItemAdd}
                handleClose={closeAddSubroutineDialog}
                isOpen={Boolean(addSubroutineNode)}
                language={language}
                nodeId={addSubroutineNode}
                routineId={routine?.id ?? ''}
                session={session}
            />}
            {/* Popup for editing existing subroutines */}
            {/* TODO */}
            {/* Popup for "Add after" dialog */}
            {addAfterLinkNode && <AddAfterLinkDialog
                handleSelect={handleNodeInsert}
                handleClose={closeAddAfterLinkDialog}
                isOpen={Boolean(addAfterLinkNode)}
                nodes={changedRoutine?.nodes ?? []}
                links={changedRoutine?.nodeLinks ?? []}
                nodeId={addAfterLinkNode}
            />}
            {/* Popup for "Add before" dialog */}
            {addBeforeLinkNode && <AddBeforeLinkDialog
                handleSelect={handleNodeInsert}
                handleClose={closeAddBeforeLinkDialog}
                isOpen={Boolean(addBeforeLinkNode)}
                nodes={changedRoutine?.nodes ?? []}
                links={changedRoutine?.nodeLinks ?? []}
                nodeId={addBeforeLinkNode}
            />}
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
            {/* Displays routine information when you click on a routine list item*/}
            <SubroutineInfoDialog
                isEditing={isEditing}
                handleUpdate={handleSubroutineUpdate}
                handleViewFull={handleSubroutineViewFull}
                language={language}
                open={Boolean(selectedSubroutine)}
                subroutine={selectedSubroutine}
                onClose={closeRoutineInfo}
            />
            {/* Displays main routine's information and some buttons */}
            <BuildInfoContainer
                canEdit={canEdit}
                handleRoutineAction={handleRoutineAction}
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
                {/* Clean up graph */}
                <Tooltip title='Clean up graph'>
                    <IconButton
                        id="clean-graph-button"
                        edge="end"
                        onClick={cleanUpGraph}
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
                        onClick={openLinkDialog}
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
                {/* Displays unlinked nodes */}
                <UnlinkedNodesDialog
                    open={isUnlinkedNodesOpen}
                    nodes={nodesOffGraph}
                    handleNodeDelete={handleNodeDelete}
                    handleToggleOpen={toggleUnlinkedNodes}
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
                    handleAction={handleAction}
                    handleLinkCreate={handleLinkCreate}
                    handleLinkUpdate={handleLinkUpdate}
                    handleLinkDelete={handleLinkDelete}
                    handleNodeInsert={handleNodeInsert}
                    handleNodeUpdate={handleNodeUpdate}
                    handleNodeDrop={handleNodeDrop}
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
                    routineId={id}
                    scale={scale}
                    session={session}
                    sliderColor={STATUS_COLOR[status.code]}
                    runState={BuildRunState.Stopped}
                />
            </Box>
        </Box>
    )
};