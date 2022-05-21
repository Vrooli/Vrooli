import { Box } from '@mui/material';
import { LinkDialog, NodeGraph, BuildBottomContainer, BuildInfoContainer, SubroutineInfoDialog, AddSubroutineDialog, AddAfterLinkDialog, AddBeforeLinkDialog, DeleteRoutineDialog } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useLazyQuery } from '@apollo/client';
import { routineCreateMutation, routineDeleteOneMutation, routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { routine, routineVariables } from 'graphql/generated/routine';
import { deleteArrayIndex, formatForUpdate, BuildAction, BuildRunState, BuildStatus, Pubs, updateArray, getTranslation, formatForCreate, getUserLanguages } from 'utils';
import { NewObject, Node, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, Routine } from 'types';
import isEqual from 'lodash/isEqual';
import { useLocation, useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';
import { BuildStatusObject } from 'components/graphs/NodeGraph/types';
import { NodeType } from 'graphql/generated/globalTypes';
import { BuildPageProps } from 'pages/types';
import _ from 'lodash';
import { v4 as uuidv4 } from 'uuid';
import { BaseObjectAction } from 'components/dialogs/types';
import { owns } from 'utils/authentication';

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
    const [isEditing, setIsEditing] = useState<boolean>(false);

    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => { setLanguage(getUserLanguages(session)[0]) }, [session]);
    const handleLanguageUpdate = useCallback((language: string) => { setLanguage(language); }, []);

    // Queries routine data
    const [getData, { data: routineData, loading: loadingRead }] = useLazyQuery<routine, routineVariables>(routineQuery);
    useEffect(() => {
        // If not add, fetch data
        if (id && id !== 'add') getData({ variables: { input: { id } } });
        // If add, set initial data
        else {
            const startNode: Node = {
                id: uuidv4(),
                type: NodeType.Start,
                columnIndex: 0,
                rowIndex: 0,
            } as Node
            const endNode: Node = {
                id: uuidv4(),
                type: NodeType.End,
                columnIndex: 1,
                rowIndex: 0,
            } as Node
            const link: NodeLink = {
                id: uuidv4(),
                fromId: startNode.id,
                toId: endNode.id,
            } as NodeLink
            setRoutine({
                nodes: [startNode, endNode],
                nodeLinks: [link],
                translations: [{
                    language,
                    title: 'New Routine',
                    instructions: 'Enter instructions here',
                }]
            } as Routine)
            setIsEditing(true);
        }
    }, [id, getData, language]);
    const [routine, setRoutine] = useState<Routine | null>(null);
    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null);
    useEffect(() => { if (routineData?.routine) setRoutine(routineData?.routine ?? null) }, [routineData]);
    // Routine mutators
    const [routineCreate, { loading: loadingCreate }] = useMutation<any>(routineCreateMutation);
    const [routineUpdate, { loading: loadingUpdate }] = useMutation<any>(routineUpdateMutation);
    const [routineDelete, { loading: loadingDelete }] = useMutation<any>(routineDeleteOneMutation);
    const loading = useMemo(() => loadingRead || loadingCreate || loadingUpdate || loadingDelete, [loadingRead, loadingCreate, loadingUpdate, loadingDelete]);
    // The routine's status (valid/invalid/incomplete)
    const [status, setStatus] = useState<BuildStatusObject>({ code: BuildStatus.Incomplete, messages: ['Calculating...'] });
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const canEdit = useMemo<boolean>(() => owns(routine?.role), [routine]);
    
    useEffect(() => {
        setChangedRoutine(routine);
    }, [routine]);

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
    const openDelete = useCallback(() => setDeleteOpen(true), []);
    const closeDelete = useCallback(() => setDeleteOpen(false), []);

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
            setIsDragging(false);
        });
        return () => {
            // Remove PubSub subscribers
            PubSub.unsubscribe(dragStartSub);
            PubSub.unsubscribe(dragDropSub);
        }
    }, []);

    /**
     * Calculates:
     * - 2D array of positioned nodes data (to represent columns and rows)
     * - 1D array of unpositioned nodes data
     * - dictionary of positioned node IDs to their data
     * Also sets the status of the routine (valid/invalid/incomplete)
     */
    const { columns, nodesOffGraph, nodesById } = useMemo(() => {
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
    const handleLinkDialogClose = useCallback((link?: NewObject<NodeLink>) => {
        if (!changedRoutine) return;
        setIsLinkDialogOpen(false);
        // If no link data, return
        if (!link) return;
        // Upsert link
        const newLinks = [...changedRoutine.nodeLinks];
        const existingLinkIndex = newLinks.findIndex(l => l.fromId === link.fromId && l.toId === link.toId);
        if (existingLinkIndex >= 0) {
            newLinks[existingLinkIndex] = { ...link } as NodeLink;
        } else {
            newLinks.push(link as NodeLink);
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
     * Creates new routine
     */
    const createRoutine = useCallback(() => {
        if (!changedRoutine) {
            return;
        }
        const input: any = formatForCreate(changedRoutine, ['nodes', 'nodeLinks', 'node.data.routines'])
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
        mutationWrapper({
            mutation: routineCreate,
            input,
            successMessage: () => 'Routine created.',
            onSuccess: ({ data }) => { 
                setRoutine(data.routineCreate);
                setLocation(`${APP_LINKS.Build}/${data.routineCreate.id}`); 
            },
        })
    }, [changedRoutine, routineCreate, setLocation]);

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
                { language, title },
            ]
        } as any);
    }, [changedRoutine, language]);

    const revertChanges = useCallback(() => {
        // If updating routine, revert to original routine
        if (id) {
            setChangedRoutine(routine);
            setIsEditing(false);
        }
        // If adding new routine, go back
        else window.history.back();
    }, [id, routine])

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
    }, [routine, routineDelete, setLocation])

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
        // Return new links combined with kept links
        return [...keptLinks, ...newLinks as any[]];
    }, [changedRoutine]);

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
    }, [calculateNewLinksList, changedRoutine]);

    /**
     * Deletes a subroutine from a node
     */
    const handleSubroutineDelete = useCallback((nodeId: string, subroutineId: string) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const node = changedRoutine.nodes[nodeIndex];
        const subroutineIndex = (node.data as NodeDataRoutineList).routines.findIndex((item: NodeDataRoutineListItem) => item.id === subroutineId);
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
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        // If columnIndex and rowIndex null, then it is being unlinked
        if (columnIndex === null && rowIndex === null) {
            const linksList = calculateNewLinksList(nodeId);
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
            let updatedNodes = [...changedRoutine.nodes];
            // If dropped into an existing column, shift rows in dropped column that are below the dropped node
            if (changedRoutine.nodes.some(n => n.columnIndex === columnIndex)) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.columnIndex === columnIndex && n.rowIndex !== null && n.rowIndex >= rowIndex) {
                        return { ...n, rowIndex: n.rowIndex + 1 }
                    }
                    return n;
                });
            }
            // If the column the node was from is now empty, then shift all columns after it
            const originalColumnIndex = changedRoutine.nodes[nodeIndex].columnIndex;
            const isRemovingColumn = originalColumnIndex !== null && changedRoutine.nodes.filter(n => n.columnIndex === originalColumnIndex).length === 1;
            if (isRemovingColumn) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.columnIndex !== null && n.columnIndex > originalColumnIndex) {
                        return { ...n, columnIndex: n.columnIndex - 1 }
                    }
                    return n;
                });
            }
            const updated = updateArray(updatedNodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                columnIndex: (isRemovingColumn && originalColumnIndex < columnIndex) ? columnIndex - 1 : columnIndex,
                rowIndex,
            })
            // Update the routine
            setChangedRoutine({
                ...changedRoutine,
                nodes: updated,
            });
        }
    }, [calculateNewLinksList, changedRoutine]);

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
    }, [changedRoutine, generateNewNode]);

    /**
     * Adds a routine list item to a routine list
     */
    const handleRoutineListItemAdd = useCallback((nodeId: string, routine: Routine) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const routineList: NodeDataRoutineList = changedRoutine.nodes[nodeIndex].data as NodeDataRoutineList;
        let routineItem: NodeDataRoutineListItem = {
            id: uuidv4(),
            isOptional: true,
            routine,
        } as any
        if (routineList.isOrdered) routineItem.index = routineList.routines.length
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
        if (!changedRoutine) return;
        // Find links where this node is the "from" node
        const links = changedRoutine.nodeLinks.filter(l => l.fromId === nodeId);
        // If multiple links, open a dialog to select which one to add after
        if (links.length > 1) {
            setAddAfterLinkNode(nodeId);
            return;
        }
        // If only one link, add after that link
        else if (links.length === 1) {
            const link = links[0];
            handleNodeInsert(link);
        }
        // If no links, create link and node
        else {
            const newNode = generateNewNode(null, null);
            const newLink = { fromId: nodeId, toId: newNode.id };
            setChangedRoutine({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [changedRoutine, generateNewNode, handleNodeInsert]);

    /**
     * Add a new routine list BEFORE a node
     */
    const handleAddBefore = useCallback((nodeId: string) => {
        if (!changedRoutine) return;
        // Find links where this node is the "to" node
        const links = changedRoutine.nodeLinks.filter(l => l.toId === nodeId);
        // If multiple links, open a dialog to select which one to add before
        if (links.length > 1) {
            setAddBeforeLinkNode(nodeId);
            return;
        }
        // If only one link, add before that link
        else if (links.length === 1) {
            const link = links[0];
            handleNodeInsert(link);
        }
        // If no links, create link and node
        else {
            const newNode = generateNewNode(null, null);
            const newLink = { fromId: newNode.id, toId: nodeId };
            setChangedRoutine({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [changedRoutine, generateNewNode, handleNodeInsert]);

    /**
     * Updates the current selected subroutine
     */
    const handleSubroutineUpdate = useCallback((updatedSubroutine: Routine) => {
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
        if (!selectedSubroutine) return;
        if (!isEqual(routine, changedRoutine)) {
            PubSub.publish(Pubs.Snack, { message: 'You have unsaved changes. Please save or discard them before navigating to another routine.' });
            return;
        }
        setLocation(`${APP_LINKS.Build}/${selectedSubroutine.id}`);
    }, [selectedSubroutine, routine, changedRoutine, setLocation]);

    const handleAction = useCallback((action: BuildAction, nodeId: string, subroutineId?: string) => {
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
                session={session}
            />}
            {/* Popup for "Add before" dialog */}
            {addBeforeLinkNode && <AddBeforeLinkDialog
                handleSelect={handleNodeInsert}
                handleClose={closeAddBeforeLinkDialog}
                isOpen={Boolean(addBeforeLinkNode)}
                nodes={changedRoutine?.nodes ?? []}
                links={changedRoutine?.nodeLinks ?? []}
                nodeId={addBeforeLinkNode}
                session={session}
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
                session={session}
                subroutine={selectedSubroutine}
                onClose={closeRoutineInfo}
            />
            {/* Displays main routine's information and some buttons */}
            <BuildInfoContainer
                canEdit={canEdit}
                handleAddLink={openLinkDialog}
                handleLanguageUpdate={handleLanguageUpdate}
                handleNodeDelete={handleNodeDelete}
                handleRoutineAction={handleRoutineAction}
                handleRoutineUpdate={updateRoutine}
                handleStartEdit={startEditing}
                handleTitleUpdate={updateRoutineTitle}
                isEditing={isEditing}
                language={language}
                loading={loadingRead}
                nodesOffGraph={nodesOffGraph}
                routine={changedRoutine}
                session={session}
                status={status}
            />
            <Box sx={{
                bottom: '0',
                display: 'flex',
                flexDirection: 'column',
                position: 'fixed',
                width: '100%',
            }}>
                <NodeGraph
                    columns={columns}
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
                    nodesById={nodesById}
                    scale={scale}
                />
                <BuildBottomContainer
                    canCancelMutate={!loading}
                    canSubmitMutate={!loading && !isEqual(routine, changedRoutine)}
                    handleCancelAdd={() => { window.history.back(); }}
                    handleCancelUpdate={revertChanges}
                    handleAdd={createRoutine}
                    handleUpdate={updateRoutine}
                    handleScaleChange={handleScaleChange}
                    hasNext={false}
                    hasPrevious={false}
                    isAdding={id === 'add'}
                    isEditing={isEditing}
                    loading={loading}
                    scale={scale}
                    session={session}
                    sliderColor={STATUS_COLOR[status.code]}
                    routine={routine}
                    runState={BuildRunState.Stopped}
                />
            </Box>
        </Box>
    )
};