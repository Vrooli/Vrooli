import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { LinkDialog, NodeGraph, BuildBottomContainer, SubroutineInfoDialog, SubroutineSelectOrCreateDialog, AddAfterLinkDialog, AddBeforeLinkDialog, EditableLabel, UnlinkedNodesDialog, BuildInfoDialog, HelpButton } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation } from '@apollo/client';
import { routineCreateMutation, routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { deleteArrayIndex, BuildAction, BuildRunState, Status, updateArray, getTranslation, getUserLanguages, parseSearchParams, stringifySearchParams, TERTIARY_COLOR, shapeRoutineUpdate, shapeRoutineCreate, NodeShape, NodeLinkShape, PubSub } from 'utils';
import { Node, NodeDataEnd, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, Routine, Run } from 'types';
import { useLocation } from 'wouter';
import { APP_LINKS, isEqual, uniqBy } from '@local/shared';
import { NodeType } from 'graphql/generated/globalTypes';
import { BaseObjectAction } from 'components/dialogs/types';
import { owns } from 'utils/authentication';
import { BuildViewProps } from '../types';
import {
    AddLink as AddLinkIcon,
    Close as CloseIcon,
    Compress as CleanUpIcon,
    Edit as EditIcon,
} from '@mui/icons-material';
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { StatusMessageArray } from 'components/buttons/types';
import { StatusButton } from 'components/buttons';
import { routineUpdate, routineUpdateVariables } from 'graphql/generated/routineUpdate';
import { routineCreate, routineCreateVariables } from 'graphql/generated/routineCreate';

//TODO
const helpText =
    `## What am I looking at?
Lorem ipsum dolor sit amet consectetur adipisicing elit. 


## How does it work?
Lorem ipsum dolor sit amet consectetur adipisicing elit.
`

/**
 * Generates a new link object, but doesn't add it to the routine
 * @param fromId - The ID of the node the link is coming from
 * @param toId - The ID of the node the link is going to
 * @returns The new link object
 */
const generateNewLink = (fromId: string, toId: string): NodeLinkShape => ({
    __typename: 'NodeLink',
    id: uuid(),
    fromId,
    toId,
})

export const BuildView = ({
    handleClose,
    loading,
    onChange,
    routine,
    session,
    zIndex,
}: BuildViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const id: string = useMemo(() => routine?.id ?? '', [routine]);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    /**
     * On page load, check if editing
     */
    useEffect(() => {
        const searchParams = parseSearchParams(window.location.search);
        // If edit param is set or build param is not a valid id, set editing to true
        if (searchParams.edit || !uuidValidate(searchParams.build ? `${searchParams.build}` : '')) {
            setIsEditing(true);
        }
    }, []);

    /**
     * Before closing, remove build-related url params
     */
    const removeSearchParams = useCallback(() => {
        const params = parseSearchParams(window.location.search);
        if (params.build) delete params.build;
        if (params.edit) delete params.edit;
        setLocation(stringifySearchParams(params), { replace: true });
    }, [setLocation]);

    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null);
    // Routine mutators
    const [routineCreate] = useMutation<routineCreate, routineCreateVariables>(routineCreateMutation);
    const [routineUpdate] = useMutation<routineUpdate, routineUpdateVariables>(routineUpdateMutation);
    // The routine's status (valid/invalid/incomplete)
    const [status, setStatus] = useState<StatusMessageArray>({ status: Status.Incomplete, messages: ['Calculating...'] });
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const canEdit = useMemo<boolean>(() => owns(routine?.role), [routine?.role]);

    useEffect(() => {
        setChangedRoutine(routine);
        // Update language
        if (routine) {
            const userLanguages = getUserLanguages(session);
            const routineLanguages = routine?.translations?.map(t => t.language)?.filter(l => typeof l === 'string' && l.length > 1) ?? [];
            // Find the first language in the user's languages that is also in the routine's languages
            const lang = userLanguages.find(l => routineLanguages.includes(l));
            if (lang) setLanguage(lang);
            else if (routineLanguages.length > 0) setLanguage(routineLanguages[0]);
            else setLanguage(userLanguages[0]);
        }
    }, [routine, session]);

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
        const statuses: [Status, string][] = []; // Holds all status messages, so multiple can be displayed
        // Loop through nodes and add to appropriate array (and also populate nodesById dictionary)
        for (const node of changedRoutine.nodes) {
            if ((node.columnIndex !== null && node.columnIndex !== undefined) && (node.rowIndex !== null && node.rowIndex !== undefined)) {
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
        const uniqueDict = uniqBy(nodesOnGraph, (n) => `${n.columnIndex}-${n.rowIndex}`);
        // Check if length of removed duplicates is equal to the length of the original positions dictionary
        if (uniqueDict.length !== Object.values(nodesOnGraph).length) {
            // Push to status
            setStatus({ status: Status.Invalid, messages: ['Ran into error determining node positions'] });
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
        // Check 1
        const startNodes = changedRoutine.nodes.filter(node => node.type === NodeType.Start);
        if (startNodes.length === 0) {
            statuses.push([Status.Invalid, 'No start node found']);
        }
        else if (startNodes.length > 1) {
            statuses.push([Status.Invalid, 'More than one start node found']);
        }
        // Check 2
        const nodesWithoutIncomingEdges = nodesOnGraph.filter(node => changedRoutine.nodeLinks.every(link => link.toId !== node.id));
        if (nodesWithoutIncomingEdges.length === 0) {
            //TODO this would be fine with a redirect link
            statuses.push([Status.Invalid, 'Error determining start node']);
        }
        else if (nodesWithoutIncomingEdges.length > 1) {
            statuses.push([Status.Invalid, 'Nodes are not fully connected']);
        }
        // Check 3
        const nodesWithoutOutgoingEdges = nodesOnGraph.filter(node => changedRoutine.nodeLinks.every(link => link.fromId !== node.id));
        if (nodesWithoutOutgoingEdges.length >= 0) {
            // Check that every node without outgoing edges is an end node
            if (nodesWithoutOutgoingEdges.some(node => node.type !== NodeType.End)) {
                statuses.push([Status.Invalid, 'Not all paths end with an end node']);
            }
        }
        // Performs checks which make the routine incomplete, but not invalid
        // 1. There are unpositioned nodes
        // 2. Every routine list has at least one subroutine
        // Check 1
        if (nodesOffGraph.length > 0) {
            statuses.push([Status.Incomplete, 'Some nodes are not linked']);
        }
        // Check 2
        if (nodesOnGraph.some(node => node.type === NodeType.RoutineList && (node.data as NodeDataRoutineList).routines.length === 0)) {
            statuses.push([Status.Incomplete, 'At least one routine list is empty']);
        }
        // Before returning, send the statuses to the status object
        if (statuses.length > 0) {
            // Status sent is the worst status
            let status = Status.Incomplete;
            if (statuses.some(status => status[0] === Status.Invalid)) status = Status.Invalid;
            setStatus({ status, messages: statuses.map(status => status[1]) });
        } else {
            setStatus({ status: Status.Valid, messages: ['Routine is fully connected'] });
        }
        // Remove any links which reference unlinked nodes
        const goodLinks = changedRoutine.nodeLinks.filter(link => !nodesOffGraph.some(node => node.id === link.fromId || node.id === link.toId));
        // If routine was mutated, update the routine
        const finalNodes = [...nodesOnGraph, ...nodesOffGraph]
        const haveNodesChanged = !isEqual(finalNodes, changedRoutine.nodes);
        const haveLinksChanged = !isEqual(goodLinks, changedRoutine.nodeLinks);
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
            if ((node.columnIndex === null || node.columnIndex === undefined) || (node.rowIndex === null || node.rowIndex === undefined)) continue;
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
        // Add one empty column to the end, so nodes, can be dragged to the end of the graph
        columns.push([]);
        // Return
        return { columns, nodesOffGraph, nodesById };
    }, [changedRoutine]);

    // Subroutine info drawer
    const [openedSubroutine, setOpenedSubroutine] = useState<{ node: NodeDataRoutineList, routineItemId: string } | null>(null);
    const handleSubroutineOpen = useCallback((nodeId: string, subroutineId: string) => {
        const node = nodesById[nodeId];
        if (node) setOpenedSubroutine({ node: (node.data as NodeDataRoutineList), routineItemId: subroutineId });
    }, [nodesById]);
    const closeRoutineInfo = useCallback(() => {
        setOpenedSubroutine(null);
    }, []);

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

    const handleRunDelete = useCallback((run: Run) => {
        if (!changedRoutine) return;
        setChangedRoutine({
            ...changedRoutine,
            runs: changedRoutine.runs.filter(r => r.id !== run.id),
        });
    }, [changedRoutine]);

    const handleRunAdd = useCallback((run: Run) => {
        if (!changedRoutine) return;
        setChangedRoutine({
            ...changedRoutine,
            runs: [run, ...changedRoutine.runs],
        });
    }, [changedRoutine]);

    const startEditing = useCallback(() => setIsEditing(true), []);
    /**
     * Creates new routine
     */
    const createRoutine = useCallback(() => {
        if (!changedRoutine) {
            return;
        }
        mutationWrapper({
            mutation: routineCreate,
            input: shapeRoutineCreate({ ...changedRoutine, id: uuid() }),
            successMessage: () => 'Routine created.',
            onSuccess: ({ data }) => {
                onChange(data.routineCreate);
                removeSearchParams();
                handleClose(true);
            },
        })
    }, [changedRoutine, handleClose, onChange, removeSearchParams, routineCreate]);

    /**
     * Mutates routine data
     */
    const updateRoutine = useCallback(() => {
        console.log('original', routine);
        console.log('changed', changedRoutine);
        if (!changedRoutine || isEqual(routine, changedRoutine)) {
            PubSub.get().publishSnack({ message: 'No changes detected', severity: 'error' });
            return;
        }
        if (!routine || !changedRoutine.id) {
            PubSub.get().publishSnack({ message: 'Cannot update: Invalid routine data', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: routineUpdate,
            input: shapeRoutineUpdate(routine, changedRoutine),
            successMessage: () => 'Routine updated.',
            onSuccess: ({ data }) => {
                // Update main routine object
                onChange(data.routineUpdate);
                // Remove indication of editing from URL
                const params = parseSearchParams(window.location.search);
                if (params.edit) delete params.edit;
                setLocation(stringifySearchParams(params), { replace: true });
                // Turn off editing mode
                setIsEditing(false);
            },
        })
    }, [changedRoutine, onChange, routine, routineUpdate, setLocation])

    /**
     * If closing with unsaved changes, prompt user to save
     */
    const onClose = useCallback(() => {
        if (isEditing && JSON.stringify(routine) !== JSON.stringify(changedRoutine)) {
            PubSub.get().publishAlertDialog({
                message: 'There are unsaved changes. Would you like to save before exiting?',
                buttons: [
                    {
                        text: 'Save', onClick: () => {
                            updateRoutine();
                            removeSearchParams();
                            handleClose(true);
                        }
                    },
                    {
                        text: "Don't Save", onClick: () => {
                            removeSearchParams();
                            handleClose(false);
                        }
                    },
                ]
            });
        } else {
            removeSearchParams();
            handleClose(false);
        }
    }, [changedRoutine, handleClose, isEditing, removeSearchParams, routine, updateRoutine]);

    /**
     * On page leave, check if routine has changed. 
     * If so, prompt user to save changes
     */
    useEffect(() => {
        const handleBeforeUnload = (e: BeforeUnloadEvent) => {
            if (isEditing && JSON.stringify(routine) !== JSON.stringify(changedRoutine)) {
                e.preventDefault()
                e.returnValue = ''
            }
        };
        window.addEventListener('beforeunload', handleBeforeUnload);
        return () => window.removeEventListener('beforeunload', handleBeforeUnload);
    }, [changedRoutine, isEditing, routine, updateRoutine]);

    const updateRoutineTitle = useCallback((title: string) => {
        if (!changedRoutine) return;
        const newTranslations = [...changedRoutine.translations.map(t => {
            if (t.language === language) {
                return { ...t, title }
            }
            return { ...t }
        })];
        setChangedRoutine({
            ...changedRoutine,
            translations: newTranslations
        });
    }, [changedRoutine, language]);

    const revertChanges = useCallback(() => {
        // Confirm if changes have been made
        if (JSON.stringify(routine) !== JSON.stringify(changedRoutine)) {
            PubSub.get().publishAlertDialog({
                message: 'There are unsaved changes. Are you sure you would like to cancel?',
                buttons: [
                    {
                        text: 'Yes', onClick: () => {
                            // If updating routine, revert to original routine
                            if (id) {
                                setChangedRoutine(routine);
                                setIsEditing(false);
                            }
                            // If adding new routine, go back
                            else window.history.back();
                        }
                    },
                    {
                        text: "No", onClick: () => { }
                    },
                ]
            });
        }
    }, [changedRoutine, id, routine])

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
        const newLinks: NodeLinkShape[] = [];
        // Find all "from" and "to" nodes in the deleting links
        const fromNodeIds = deletingLinks.map(l => l.fromId).filter(id => id !== nodeId);
        const toNodeIds = deletingLinks.map(l => l.toId).filter(id => id !== nodeId);
        // If there is only one "from" node, create a link between it and every "to" node
        if (fromNodeIds.length === 1) {
            toNodeIds.forEach(toId => { newLinks.push(generateNewLink(fromNodeIds[0], toId)) });
        }
        // If there is only one "to" node, create a link between it and every "from" node
        else if (toNodeIds.length === 1) {
            fromNodeIds.forEach(fromId => { newLinks.push(generateNewLink(fromId, toNodeIds[0])) });
        }
        // NOTE: Every other case is ambiguous, so we can't auto-create create links
        // Delete old links
        let keptLinks = changedRoutine.nodeLinks.filter(l => !deletingLinks.includes(l));
        // Return new links combined with kept links
        return [...keptLinks, ...newLinks as any[]];
    }, [changedRoutine]);

    /**
     * Generates a new routine list node object, but doesn't add it to the routine
     */
    const generateNewRoutineListNode = useCallback((columnIndex: number | null, rowIndex: number | null) => {
        const newNode: Omit<NodeShape, 'routineId'> = {
            __typename: 'Node',
            id: uuid(),
            type: NodeType.RoutineList,
            rowIndex,
            columnIndex,
            data: {
                id: uuid(),
                __typename: 'NodeRoutineList',
                isOrdered: false,
                isOptional: false,
                routines: [],
            },
            // Generate unique placeholder title
            translations: [{
                __typename: 'NodeTranslation',
                id: uuid(),
                language,
                title: `Node ${(changedRoutine?.nodes?.length ?? 0) - 1}`,
                description: '',
            }],
        }
        return newNode;
    }, [language, changedRoutine?.nodes]);

    /**
     * Generates new end node object, but doesn't add it to the routine
     */
    const generateNewEndNode = useCallback((columnIndex: number | null, rowIndex: number | null) => {
        const newNode: Omit<NodeShape, 'routineId'> = {
            __typename: 'Node',
            id: uuid(),
            type: NodeType.End,
            rowIndex,
            columnIndex,
            data: {
                id: uuid(),
                wasSuccessful: true,
            },
            translations: []
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
        console.log('handleNodeDrop', nodeId, columnIndex, rowIndex);
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
            PubSub.get().publishSnack({ message: 'Error: Invalid drop location.', severity: 'errror' });
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
        const linkIndex = changedRoutine.nodeLinks.findIndex(l => l.fromId === link.fromId && l.toId === link.toId);
        // Delete link
        const linksList = deleteArrayIndex(changedRoutine.nodeLinks, linkIndex);
        // Find "to" node. New node will be placed in its row and column
        const toNode = changedRoutine.nodes.find(n => n.id === link.toId);
        if (!toNode) {
            PubSub.get().publishSnack({ message: 'Error occurred.', severity: 'Error' });
            return;
        }
        const { columnIndex, rowIndex } = toNode;
        // Move every node starting from the "to" node to the right by one
        const nodesList = changedRoutine.nodes.map(n => {
            if (n.columnIndex !== null && n.columnIndex !== undefined && n.columnIndex >= (columnIndex ?? 0)) {
                return { ...n, columnIndex: n.columnIndex + 1 };
            }
            return n;
        });
        // Create new routine list node
        const newNode: Omit<NodeShape, 'routineId'> = generateNewRoutineListNode(columnIndex, rowIndex);
        // Find every node 
        // Create two new links
        const newLinks: NodeLinkShape[] = [
            generateNewLink(link.fromId, newNode.id),
            generateNewLink(newNode.id, link.toId),
        ];
        // Insert new node and links
        const newRoutine = {
            ...changedRoutine,
            nodes: [...nodesList, newNode as any],
            nodeLinks: [...linksList, ...newLinks as any],
        };
        setChangedRoutine(newRoutine);
    }, [changedRoutine, generateNewRoutineListNode]);

    /**
     * Inserts a new routine list node, with its own branch
     */
    const handleBranchInsert = useCallback((link: NodeLink) => {
        if (!changedRoutine) return;
        // Find "to" node. New node will be placed in its column
        const toNode = changedRoutine.nodes.find(n => n.id === link.toId);
        if (!toNode) {
            PubSub.get().publishSnack({ message: 'Error occurred.', severity: 'Error' });
            return;
        }
        // Find the largest row index in the column. New node will be placed in the next row
        const maxRowIndex = changedRoutine.nodes.filter(n => n.columnIndex === toNode.columnIndex).map(n => n.rowIndex).reduce((a, b) => Math.max(a ?? 0, b ?? 0), 0);
        const newNode: Omit<NodeShape, 'routineId'> = generateNewRoutineListNode(toNode.columnIndex, (maxRowIndex ?? toNode.rowIndex ?? 0) + 1);
        // Since this is a new branch, we also need to add an end node after the new node
        const newEndNode: Omit<NodeShape, 'routineId'> = generateNewEndNode((toNode.columnIndex ?? 0) + 1, (maxRowIndex ?? toNode.rowIndex ?? 0) + 1);
        // Create new link, going from the "from" node to the new node
        const newLink: NodeLinkShape = generateNewLink(link.fromId, newNode.id);
        // Create new link, going from the new node to the end node
        const newEndLink: NodeLinkShape = generateNewLink(newNode.id, newEndNode.id);
        // Insert new nodes and links
        const newRoutine = {
            ...changedRoutine,
            nodes: [...changedRoutine.nodes, newNode as any, newEndNode as any],
            nodeLinks: [...changedRoutine.nodeLinks, newLink as any, newEndLink as any],
        };
        setChangedRoutine(newRoutine);
    }, [changedRoutine, generateNewEndNode, generateNewRoutineListNode]);

    /**
     * Adds a routine list item to a routine list
     */
    const handleRoutineListItemAdd = useCallback((nodeId: string, routine: Routine) => {
        if (!changedRoutine) return;
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const routineList: NodeDataRoutineList = changedRoutine.nodes[nodeIndex].data as NodeDataRoutineList;
        let routineItem: NodeDataRoutineListItem = {
            id: uuid(),
            index: routineList.routines.length,
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
     * Reoders a subroutine in a routine list item
     * @param nodeId The node id of the routine list item
     * @param oldIndex The old index of the subroutine
     * @param newIndex The new index of the subroutine
     */
    const handleRoutineListItemReorder = useCallback((nodeId: string, oldIndex: number, newIndex: number) => {
        // Find routines being swapped
        if (!changedRoutine) return;
        // Node containing routine list data with ID nodeId
        const nodeIndex = changedRoutine.nodes.findIndex(n => n.data?.id === nodeId);
        if (nodeIndex === -1) return;
        const routineList: NodeDataRoutineList = changedRoutine.nodes[nodeIndex].data as NodeDataRoutineList;
        const routines = [...routineList.routines];
        // Find subroutines matching old and new index
        const aIndex = routines.findIndex(r => r.index === oldIndex);
        const bIndex = routines.findIndex(r => r.index === newIndex);
        if (aIndex === -1 || bIndex === -1) return;
        // Swap the routine indexes
        routines[aIndex] = { ...routines[aIndex], index: newIndex };
        routines[bIndex] = { ...routines[bIndex], index: oldIndex };
        // Update the routine list
        setChangedRoutine({
            ...changedRoutine,
            nodes: updateArray(changedRoutine.nodes, nodeIndex, {
                ...changedRoutine.nodes[nodeIndex],
                data: {
                    ...routineList,
                    routines,
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
            const newNode: Omit<NodeShape, 'routineId'> = generateNewRoutineListNode(null, null);
            const newLink: NodeLinkShape = generateNewLink(nodeId, newNode.id);
            setChangedRoutine({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [changedRoutine, generateNewRoutineListNode, handleNodeInsert]);

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
            const newNode: Omit<NodeShape, 'routineId'> = generateNewRoutineListNode(null, null);
            const newLink: NodeLinkShape = generateNewLink(newNode.id, nodeId);
            setChangedRoutine({
                ...changedRoutine,
                nodes: [...changedRoutine.nodes, newNode as any],
                nodeLinks: [...changedRoutine.nodeLinks, newLink as any],
            });
        }
    }, [changedRoutine, generateNewRoutineListNode, handleNodeInsert]);

    /**
     * Updates the current selected subroutine
     */
    const handleSubroutineUpdate = useCallback((updatedSubroutine: NodeDataRoutineListItem) => {
        if (!changedRoutine) return;
        // Update routine
        setChangedRoutine({
            ...changedRoutine,
            nodes: changedRoutine.nodes.map((n: Node) => {
                if (n.type === NodeType.RoutineList && (n.data as NodeDataRoutineList).routines.some(r => r.id === updatedSubroutine.id)) {
                    return {
                        ...n,
                        data: {
                            ...n.data,
                            routines: (n.data as NodeDataRoutineList).routines.map(r => {
                                if (r.id === updatedSubroutine.id) {
                                    return {
                                        ...r,
                                        ...updatedSubroutine,
                                        routine: {
                                            ...r.routine,
                                            ...updatedSubroutine.routine,
                                        }
                                    };
                                }
                                return r;
                            }),
                        },
                    };
                }
                return n;
            }),
        } as any);
        // Close dialog
        closeRoutineInfo();
    }, [changedRoutine, closeRoutineInfo]);

    /**
     * Navigates to a subroutine's build page. Fist checks if there are unsaved changes
     */
    const handleSubroutineViewFull = useCallback(() => {
        if (!openedSubroutine) return;
        if (!isEqual(routine, changedRoutine)) {
            PubSub.get().publishSnack({ message: 'You have unsaved changes. Please save or discard them before navigating to another routine.' });
            return;
        }
        // TODO - buildview should have its own buildview, to recursively open subroutines
        //setLocation(`${APP_LINKS.Build}/${selectedSubroutine.id}`);
    }, [changedRoutine, openedSubroutine, routine]);

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

    const handleRoutineAction = useCallback((action: BaseObjectAction, data: any) => {
        switch (action) {
            case BaseObjectAction.Copy:
                setLocation(`${APP_LINKS.Routine}/${data.copy.routine.id}`);
                break;
            case BaseObjectAction.Delete:
                setLocation(APP_LINKS.Home);
                break;
            case BaseObjectAction.Downvote:
                if (data.vote.success) {
                    onChange({
                        ...routine,
                        isUpvoted: false,
                    } as any)
                }
                break;
            case BaseObjectAction.Edit:
                //TODO
                break;
            case BaseObjectAction.Fork:
                setLocation(`${APP_LINKS.Routine}/${data.fork.routine.id}`);
                break;
            case BaseObjectAction.Report:
                //TODO
                break;
            case BaseObjectAction.Share:
                //TODO
                break;
            case BaseObjectAction.Star:
                if (data.star.success) {
                    onChange({
                        ...routine,
                        isStarred: true,
                    } as any)
                }
                break;
            case BaseObjectAction.Stats:
                //TODO
                break;
            case BaseObjectAction.Unstar:
                if (data.star.success) {
                    onChange({
                        ...routine,
                        isStarred: false,
                    } as any)
                }
                break;
            case BaseObjectAction.Update:
                updateRoutine();
                break;
            case BaseObjectAction.UpdateCancel:
                setChangedRoutine(routine);
                break;
            case BaseObjectAction.Upvote:
                if (data.vote.success) {
                    onChange({
                        ...routine,
                        isUpvoted: true,
                    } as any)
                }
                break;
        }
    }, [setLocation, updateRoutine, routine, onChange]);

    // Open/close unlinked nodes drawer
    const [isUnlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState<boolean>(false);
    const toggleUnlinkedNodes = useCallback(() => setIsUnlinkedNodesOpen(curr => !curr), []);

    /**
     * Cleans up graph by removing empty columns and row gaps within columns.
     * Also adds end nodes to the end of each unfinished path. 
     * Also removes links that don't have both a valid fromId and toId.
     */
    const cleanUpGraph = useCallback(() => {
        if (!changedRoutine) return;
        const resultRoutine = JSON.parse(JSON.stringify(changedRoutine));
        // Loop through the columns, and remove gaps in rowIndex
        for (const column of columns) {
            // Sort nodes in column by rowIndex
            const sortedNodes = column.sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
            // If the nodes don't go from 0 to n without any gaps
            if (sortedNodes.length > 0 && sortedNodes.some((n, i) => (n.rowIndex ?? 0) !== i)) {
                // Update nodes in resultRoutine with new rowIndexes
                const newNodes = sortedNodes.map((n, i) => ({
                    ...n,
                    rowIndex: i,
                }));
                // Replace nodes in resultRoutine
                resultRoutine.nodes = resultRoutine.nodes.map(oldNode => {
                    const newNode = newNodes.find(nn => nn.id === oldNode.id);
                    if (newNode) {
                        return newNode;
                    }
                    return oldNode;
                });
            }
        }
        // Find every node that does not have a link leaving it, which is also 
        // not an end node
        for (const node of resultRoutine.nodes) {
            // If not an end node
            if (node.type !== NodeType.End) {
                // Check if any links have a "fromId" matching this node's ID
                const leavingLinks = resultRoutine.nodeLinks.filter(link => link.fromId === node.id);
                // If there are no leaving links, create a new link and end node
                if (leavingLinks.length === 0) {
                    // Generate node ID
                    const newEndNodeId = uuid();
                    // Calculate rowIndex and columnIndex
                    // Column is 1 after current column
                    const columnIndex: number = (node.columnIndex ?? 0) + 1;
                    // Node is 1 after last rowIndex in column
                    const rowIndex = (columnIndex >= 0 && columnIndex < columns.length) ? columns[columnIndex].length : 0;
                    const newLink: NodeLinkShape = generateNewLink(node.id, newEndNodeId);
                    const newEndNode: Omit<NodeShape, 'routineId'> = {
                        __typename: 'Node',
                        id: newEndNodeId,
                        type: NodeType.End,
                        rowIndex,
                        columnIndex,
                        data: {
                            wasSuccessful: false,
                        } as any,
                        translations: [],
                    }
                    // Add link and end node to resultRoutine
                    resultRoutine.nodeLinks.push(newLink as any);
                    resultRoutine.nodes.push(newEndNode as any);
                }
            }
        }
        // Remove links that don't have both a valid fromId and toId
        resultRoutine.nodeLinks = resultRoutine.nodeLinks.filter(link => {
            const fromNode = resultRoutine.nodes.find(n => n.id === link.fromId);
            const toNode = resultRoutine.nodes.find(n => n.id === link.toId);
            return Boolean(fromNode && toNode);
        });
        // Update changedRoutine with resultRoutine
        setChangedRoutine(resultRoutine);
    }, [changedRoutine, columns]);

    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            minHeight: '100%',
            height: '100%',
            width: '100%',
        }}>
            {/* Popup for adding new subroutines */}
            {addSubroutineNode && <SubroutineSelectOrCreateDialog
                handleAdd={handleRoutineListItemAdd}
                handleClose={closeAddSubroutineDialog}
                isOpen={Boolean(addSubroutineNode)}
                nodeId={addSubroutineNode}
                routineId={routine?.id ?? ''}
                session={session}
                zIndex={zIndex + 3}
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
                zIndex={zIndex + 3}
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
                zIndex={zIndex + 3}
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
                zIndex={zIndex + 3}
            // partial={ }
            /> : null}
            {/* Displays routine information when you click on a routine list item*/}
            <SubroutineInfoDialog
                data={openedSubroutine}
                defaultLanguage={language}
                isEditing={isEditing}
                handleUpdate={handleSubroutineUpdate}
                handleReorder={handleRoutineListItemReorder}
                handleViewFull={handleSubroutineViewFull}
                open={Boolean(openedSubroutine)}
                session={session}
                onClose={closeRoutineInfo}
                zIndex={zIndex + 3}
            />
            {/* Display top navbars */}
            {/* First contains close icon and title */}
            <Stack
                id="routine-title-and-language"
                direction="row"
                sx={{
                    zIndex: 2,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    height: '64px',
                }}>
                {/* Title */}
                <EditableLabel
                    canEdit={isEditing}
                    handleUpdate={updateRoutineTitle}
                    placeholder={loading ? 'Loading...' : 'Enter title...'}
                    renderLabel={(t) => (
                        <Typography
                            component="h2"
                            variant="h5"
                            textAlign="center"
                            sx={{
                                fontSize: { xs: '1em', sm: '1.25em', md: '1.5em' },
                            }}
                        >{t ?? (loading ? 'Loading...' : 'Enter title')}</Typography>
                    )}
                    text={getTranslation(changedRoutine, 'title', [language], false) ?? ''}
                    sxs={{
                        stack: { marginLeft: 'auto' }
                    }}
                />
                {/* Close Icon */}
                <IconButton
                    edge="start"
                    aria-label="close"
                    onClick={onClose}
                    color="inherit"
                    sx={{
                        marginLeft: 'auto',
                    }}
                >
                    <CloseIcon sx={{
                        width: '32px',
                        height: '32px',
                    }} />
                </IconButton>
            </Stack>
            {/* Second contains additional info and icons */}
            <Stack
                id="build-routine-information-bar"
                direction="row"
                spacing={2}
                width="100%"
                justifyContent="space-between"
                sx={{
                    zIndex: 2,
                    height: '48px',
                    background: palette.primary.light,
                    color: palette.primary.contrastText,
                }}
            >
                <StatusButton status={status.status} messages={status.messages} sx={{
                    marginTop: 'auto',
                    marginBottom: 'auto',
                    marginLeft: 2,
                }} />
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    {/* Clean up graph */}
                    {isEditing && <Tooltip title='Clean up graph'>
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
                    </Tooltip>}
                    {/* Add new links to the routine */}
                    {isEditing && <Tooltip title='Add new link'>
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
                    </Tooltip>}
                    {/* Displays unlinked nodes */}
                    {isEditing && <UnlinkedNodesDialog
                        handleNodeDelete={handleNodeDelete}
                        handleToggleOpen={toggleUnlinkedNodes}
                        language={language}
                        nodes={nodesOffGraph}
                        open={isUnlinkedNodesOpen}
                        zIndex={zIndex + 3}
                    />}
                    {/* Edit button */}
                    {canEdit && !isEditing ? (
                        <IconButton aria-label="confirm-title-change" onClick={startEditing} >
                            <EditIcon sx={{ fill: TERTIARY_COLOR }} />
                        </IconButton>
                    ) : null}
                    {/* Help button */}
                    <HelpButton markdown={helpText} sxRoot={{ margin: "auto", marginRight: 1 }} sx={{ color: TERTIARY_COLOR }} />
                    {/* Display routine description, insturctions, etc. */}
                    <BuildInfoDialog
                        handleAction={handleRoutineAction}
                        handleLanguageChange={setLanguage}
                        handleUpdate={(updated: Routine) => { setChangedRoutine(updated); }}
                        isEditing={isEditing}
                        language={language}
                        loading={loading}
                        routine={changedRoutine}
                        session={session}
                        sxs={{ icon: { fill: TERTIARY_COLOR, marginRight: 1 } }}
                        zIndex={zIndex + 1}
                    />
                </Box>
            </Stack>
            {/* Displays main routine's information and some buttons */}
            <Box sx={{
                background: palette.background.default,
                bottom: '0',
                display: 'flex',
                flexDirection: 'column',
                position: 'fixed',
                width: '100%',
            }}>
                <NodeGraph
                    columns={columns}
                    handleAction={handleAction}
                    handleBranchInsert={handleBranchInsert}
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
                    zIndex={zIndex}
                />
                <BuildBottomContainer
                    canCancelMutate={!loading}
                    canSubmitMutate={!loading && !isEqual(routine, changedRoutine)}
                    handleCancelAdd={() => { window.history.back(); }}
                    handleCancelUpdate={revertChanges}
                    handleAdd={createRoutine}
                    handleUpdate={updateRoutine}
                    handleScaleChange={handleScaleChange}
                    handleRunDelete={handleRunDelete}
                    handleRunAdd={handleRunAdd}
                    hasNext={false}
                    hasPrevious={false}
                    isAdding={!uuidValidate(id)}
                    isEditing={isEditing}
                    loading={loading}
                    scale={scale}
                    session={session}
                    sliderColor={palette.secondary.light}
                    routine={routine}
                    runState={BuildRunState.Stopped}
                    zIndex={zIndex}
                />
            </Box>
        </Box>
    )
};