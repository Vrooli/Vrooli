import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { LinkDialog, NodeGraph, BuildBottomContainer, SubroutineInfoDialog, SubroutineSelectOrCreateDialog, AddAfterLinkDialog, AddBeforeLinkDialog, DeleteDialog, EditableLabel, UnlinkedNodesDialog, BuildInfoDialog, HelpButton } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation } from '@apollo/client';
import { routineCreateMutation, routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { deleteArrayIndex, formatForUpdate, BuildAction, BuildRunState, Status, Pubs, updateArray, getTranslation, formatForCreate, getUserLanguages, parseSearchParams, stringifySearchParams, TERTIARY_COLOR } from 'utils';
import { NewObject, Node, NodeDataRoutineList, NodeDataRoutineListItem, NodeLink, Routine } from 'types';
import isEqual from 'lodash/isEqual';
import { useLocation } from 'wouter';
import { APP_LINKS, DeleteOneType } from '@local/shared';
import { NodeType } from 'graphql/generated/globalTypes';
import _ from 'lodash';
import { v4 as uuidv4 } from 'uuid';
import { BaseObjectAction } from 'components/dialogs/types';
import { owns } from 'utils/authentication';
import { BuildViewProps } from '../types';
import {
    AddLink as AddLinkIcon,
    Close as CloseIcon,
    Compress as CleanUpIcon,
    Edit as EditIcon,
} from '@mui/icons-material';
import { validate as uuidValidate } from 'uuid';
import { StatusMessageArray } from 'components/buttons/types';
import { StatusButton } from 'components/buttons';
import { routineUpdate } from 'graphql/generated/routineUpdate';
import { routineCreate } from 'graphql/generated/routineCreate';

//TODO
const helpText =
    `## What am I looking at?
Lorem ipsum dolor sit amet consectetur adipisicing elit. 


## How does it work?
Lorem ipsum dolor sit amet consectetur adipisicing elit.
`

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
    console.log('languageeeeeeee', language)

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
    const [routineCreate] = useMutation<routineCreate>(routineCreateMutation);
    const [routineUpdate] = useMutation<routineUpdate>(routineUpdateMutation);
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
            console.log('languages user routine', userLanguages, routineLanguages)
            // Find the first language in the user's languages that is also in the routine's languages
            const lang = userLanguages.find(l => routineLanguages.includes(l));
            console.log('lang', lang)
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

    // Handle delete
    const [deleteOpen, setDeleteOpen] = useState(false);
    const openDelete = useCallback(() => setDeleteOpen(true), []);
    const handleDeleteClose = useCallback((wasDeleted: boolean) => {
        if (wasDeleted) setLocation(APP_LINKS.Home);
        else setDeleteOpen(false);
    }, [setLocation])

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
        // First check
        const startNodes = changedRoutine.nodes.filter(node => node.type === NodeType.Start);
        if (startNodes.length === 0) {
            statuses.push([Status.Invalid, 'No start node found']);
        }
        else if (startNodes.length > 1) {
            statuses.push([Status.Invalid, 'More than one start node found']);
        }
        // Second check
        const nodesWithoutIncomingEdges = nodesOnGraph.filter(node => changedRoutine.nodeLinks.every(link => link.toId !== node.id));
        if (nodesWithoutIncomingEdges.length === 0) {
            //TODO this would be fine with a redirect link
            statuses.push([Status.Invalid, 'Error determining start node']);
        }
        else if (nodesWithoutIncomingEdges.length > 1) {
            statuses.push([Status.Invalid, 'Nodes are not fully connected']);
        }
        // Third check
        const nodesWithoutOutgoingEdges = nodesOnGraph.filter(node => changedRoutine.nodeLinks.every(link => link.fromId !== node.id));
        if (nodesWithoutOutgoingEdges.length >= 0) {
            // Check that every node without outgoing edges is an end node
            if (nodesWithoutOutgoingEdges.some(node => node.type !== NodeType.End)) {
                statuses.push([Status.Invalid, 'Not all paths end with an end node']);
            }
        }
        // Performs checks which make the routine incomplete, but not invalid
        // 1. There are unpositioned nodes
        // First check
        if (nodesOffGraph.length > 0) {
            statuses.push([Status.Incomplete, 'Some nodes are not linked']);
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
    const [openedSubroutine, setOpenedSubroutine] = useState<{ node: NodeDataRoutineList, routineId: string } | null>(null);
    const handleSubroutineOpen = useCallback((nodeId: string, subroutineId: string) => {
        const node = nodesById[nodeId];
        if (node) setOpenedSubroutine({ node: (node.data as NodeDataRoutineList), routineId: subroutineId });
    }, [nodesById]);
    const closeRoutineInfo = useCallback(() => {
        setOpenedSubroutine(null);
    }, []);

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
                onChange(data.routineCreate);
                removeSearchParams();
                handleClose();
            },
        })
    }, [changedRoutine, handleClose, onChange, removeSearchParams, routineCreate]);

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
            onSuccess: ({ data }) => { onChange(data.routineUpdate); },
        })
    }, [changedRoutine, onChange, routine, routineUpdate])

    /**
     * If closing with unsaved changes, prompt user to save
     */
    const onClose = useCallback(() => {
        if (isEditing && JSON.stringify(routine) !== JSON.stringify(changedRoutine)) {
            PubSub.publish(Pubs.AlertDialog, {
                message: 'There are unsaved changes. Would you like to save before exiting?',
                buttons: [
                    {
                        text: 'Save', onClick: () => {
                            updateRoutine();
                            removeSearchParams();
                            handleClose();
                        }
                    },
                    {
                        text: "Don't Save", onClick: () => {
                            removeSearchParams();
                            handleClose();
                        }
                    },
                ]
            });
        } else {
            removeSearchParams();
            handleClose();
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
        // If updating routine, revert to original routine
        if (id) {
            setChangedRoutine(routine);
            setIsEditing(false);
        }
        // If adding new routine, go back
        else window.history.back();
    }, [id, routine])

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
            // Generate unique placeholder title
            translations: [{ language, title: `Node ${(changedRoutine?.nodes?.length ?? 0) - 1}` }] as Node['translations'],
        }
        return newNode;
    }, [language, changedRoutine?.nodes]);

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
        const linkIndex = changedRoutine.nodeLinks.findIndex(l => l.fromId === link.fromId && l.toId === link.toId);
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
        if (!openedSubroutine) return;
        if (!isEqual(routine, changedRoutine)) {
            PubSub.publish(Pubs.Snack, { message: 'You have unsaved changes. Please save or discard them before navigating to another routine.' });
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

    const handleRoutineAction = useCallback((action: BaseObjectAction) => {
        switch (action) {
            case BaseObjectAction.Copy:
                //TODO
                break;
            case BaseObjectAction.Delete:
                openDelete();
                break;
            case BaseObjectAction.Downvote:
                //TODO
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

    // Open/close unlinked nodes drawer
    const [isUnlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState<boolean>(false);
    const toggleUnlinkedNodes = useCallback(() => setIsUnlinkedNodesOpen(curr => !curr), []);

    /**
     * Cleans up graph by removing empty columns and row gaps within columns.
     * Also adds end nodes to the end of each unfinished path
     */
    const cleanUpGraph = useCallback(() => {
        //TODO
    }, []);

    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            minHeight: '100%',
            height: '100%',
            width: '100%',
        }}>
            {/* Delete routine confirmation dialog */}
            <DeleteDialog
                isOpen={deleteOpen}
                objectId={routine?.id ?? ''}
                objectType={DeleteOneType.Routine}
                objectName={getTranslation(changedRoutine, 'title', [language]) ?? ''}
                handleClose={handleDeleteClose}
                zIndex={zIndex + 3}
            />
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