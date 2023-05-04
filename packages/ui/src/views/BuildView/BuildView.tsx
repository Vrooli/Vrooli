import { CloseIcon, exists, isEqual, keepSearchParams, Node, NodeLink, NodeRoutineList, NodeRoutineListItem, NodeType, RoutineVersion, useLocation, uuid, uuidValidate } from "@local/shared";
import { Box, IconButton, Stack, useTheme } from "@mui/material";
import { BuildEditButtons } from "components/buttons/BuildEditButtons/BuildEditButtons";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { StatusButton } from "components/buttons/StatusButton/StatusButton";
import { StatusMessageArray } from "components/buttons/types";
import { FindSubroutineDialog } from "components/dialogs/FindSubroutineDialog/FindSubroutineDialog";
import { LinkDialog } from "components/dialogs/LinkDialog/LinkDialog";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { SubroutineInfoDialog } from "components/dialogs/SubroutineInfoDialog/SubroutineInfoDialog";
import { AddAfterLinkDialog, AddBeforeLinkDialog, GraphActions, NodeGraph } from "components/graphs/NodeGraph";
import { MoveNodeMenu as MoveNodeDialog } from "components/graphs/NodeGraph/MoveNodeDialog/MoveNodeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BuildAction, Status } from "utils/consts";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { useStableObject } from "utils/hooks/useStableObject";
import { PubSub } from "utils/pubsub";
import { getRoutineVersionStatus } from "utils/runUtils";
import { deleteArrayIndex, updateArray } from "utils/shape/general";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { NodeRoutineListShape } from "utils/shape/models/nodeRoutineList";
import { NodeRoutineListItemShape } from "utils/shape/models/nodeRoutineListItem";
import { BuildViewProps } from "../types";

const helpText =
    "## What am I looking at?\nThis is the **Build** page. Here you can create and edit multi-step routines.\n\n## What is a routine?\nA *routine* is simply a process for completing a task, which takes inputs, performs some action, and outputs some results. By connecting multiple routines together, you can perform arbitrarily complex tasks.\n\nAll valid multi-step routines have a *start* node and at least one *end* node. Each node inbetween stores a list of subroutines, which can be optional or required.\n\nWhen a user runs the routine, they traverse the routine graph from left to right. Each subroutine is rendered as a page, with components such as TextFields for each input. Where the graph splits, users are given a choice of which subroutine to go to next.\n\n## How do I build a multi-step routine?\nIf you are starting from scratch, you will see a *start* node, a *routine list* node, and an *end* node.\n\nYou can press the routine list node to toggle it open/closed. The *open* stats allows you to select existing subroutines from Vrooli, or create a new one.\n\nEach link connecting nodes has a circle. Pressing this circle opens a popup menu with options to insert a node, split the graph, or delete the link.\n\nYou also have the option to *unlink* nodes. These are stored on the top status bar - along with the status indicator, a button to clean up the graph, a button to add a new link, this help button, and an info button that sets overall routine information.";

/**
 * Generates a new link object, but doesn't add it to the routine
 * @param fromId - The ID of the node the link is coming from
 * @param toId - The ID of the node the link is going to
 * @param routineVersionId - The ID of the overall routine version
 * @returns The new link object
 */
const generateNewLink = (fromId: string, toId: string, routineVersionId: string): NodeLinkShape => ({
    id: uuid(),
    from: { id: fromId },
    to: { id: toId },
    routineVersion: { id: routineVersionId },
});

// RoutineVersion with fields required for the build view
type BuildRoutineVersion = Pick<RoutineVersion, "id" | "nodes" | "nodeLinks">

export const BuildView = ({
    display = "dialog",
    handleCancel,
    handleClose,
    handleSubmit,
    isEditing,
    loading,
    routineVersion,
    translationData,
    zIndex = 200,
}: BuildViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const id: string = useMemo(() => routineVersion?.id ?? "", [routineVersion]);

    const stableRoutineVersion = useStableObject(routineVersion);
    const [changedRoutineVersion, setChangedRoutineVersion] = useState<BuildRoutineVersion>(routineVersion);
    // The routineVersion's status (valid/invalid/incomplete)
    const [status, setStatus] = useState<StatusMessageArray>({ status: Status.Incomplete, messages: ["Calculating..."] });

    // Stores previous routineVersion states for undo/redo
    const [changeStack, setChangeStack] = useState<BuildRoutineVersion[]>([]);
    const [changeStackIndex, setChangeStackIndex] = useState<number>(0);
    const clearChangeStack = useCallback(() => {
        setChangeStack(stableRoutineVersion ? [stableRoutineVersion] : []);
        setChangeStackIndex(stableRoutineVersion ? 0 : -1);
        PubSub.get().publishFastUpdate({ duration: 1000 });
        setChangedRoutineVersion(stableRoutineVersion);
    }, [stableRoutineVersion]);
    useEffect(() => {
        clearChangeStack();
    }, [clearChangeStack]);
    /**
     * Moves back one in the change stack
     */
    const undo = useCallback(() => {
        if (changeStackIndex > 0) {
            setChangeStackIndex(changeStackIndex - 1);
            PubSub.get().publishFastUpdate({ duration: 1000 });
            console.log("undoing");
            setChangedRoutineVersion(changeStack[changeStackIndex - 1]);
        }
    }, [changeStackIndex, changeStack, setChangedRoutineVersion]);
    const canUndo = useMemo(() => changeStackIndex > 0 && changeStack.length > 0, [changeStackIndex, changeStack]);
    /**
     * Moves forward one in the change stack
     */
    const redo = useCallback(() => {
        if (changeStackIndex < changeStack.length - 1) {
            setChangeStackIndex(changeStackIndex + 1);
            PubSub.get().publishFastUpdate({ duration: 1000 });
            console.log("redoing");
            setChangedRoutineVersion(changeStack[changeStackIndex + 1]);
        }
    }, [changeStackIndex, changeStack, setChangedRoutineVersion]);
    const canRedo = useMemo(() => changeStackIndex < changeStack.length - 1 && changeStack.length > 0, [changeStackIndex, changeStack]);
    /**
     * Adds, to change stack, and removes anything from the change stack after the current index
     */
    const addToChangeStack = useCallback((changedRoutine: BuildRoutineVersion) => {
        const newChangeStack = [...changeStack];
        newChangeStack.splice(changeStackIndex + 1, newChangeStack.length - changeStackIndex - 1);
        newChangeStack.push(changedRoutine);
        setChangeStack(newChangeStack);
        setChangeStackIndex(newChangeStack.length - 1);
        PubSub.get().publishFastUpdate({ duration: 1000 });
        console.log("adding to change stack", changedRoutine);
        setChangedRoutineVersion(changedRoutine);
    }, [changeStack, changeStackIndex, setChangeStack, setChangeStackIndex, setChangedRoutineVersion]);

    // Handle undo and redo keys
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // CTRL + Y or CTRL + SHIFT + Z = redo
            if (e.ctrlKey && (e.key === "y" || e.key === "Z")) { redo(); }
            // CTRL + Z = undo
            else if (e.ctrlKey && e.key === "z") { undo(); }
        };
        // Attach the event listener
        document.addEventListener("keydown", handleKeyDown);
        // Remove the event listener
        return () => { document.removeEventListener("keydown", handleKeyDown); };
    }, [redo, undo]);

    usePromptBeforeUnload({ shouldPrompt: isEditing && changeStack.length > 1 });

    /**
     * Calculates:
     * - 2D array of positioned nodes data (to represent columns and rows)
     * - 1D array of unpositioned nodes data
     * - dictionary of positioned node IDs to their data
     * Also sets the status of the routineVersion (valid/invalid/incomplete)
     */
    const { columns, nodesOffGraph, nodesById } = useMemo(() => {
        if (!changedRoutineVersion) return { columns: [], nodesOffGraph: [], nodesById: {} };
        const { messages, nodesById, nodesOnGraph, nodesOffGraph, status } = getRoutineVersionStatus(changedRoutineVersion);
        // Check for critical errors
        if (messages.includes("No node or link data found")) {
            return { columns: [], nodesOffGraph: [], nodesById: {} };
        }
        if (messages.includes("Ran into error determining node positions")) {
            // Remove all node positions and links
            console.log("calculate memo setting changed routine 1");
            setChangedRoutineVersion({
                ...changedRoutineVersion,
                nodes: changedRoutineVersion.nodes.map(n => ({ ...n, columnIndex: null, rowIndex: null })),
                nodeLinks: [],
            });
            return { columns: [], nodesOffGraph: changedRoutineVersion.nodes, nodesById: {} };
        }
        // Update status
        setStatus({ status, messages });
        // Remove any links which reference unlinked nodes
        const goodLinks = changedRoutineVersion.nodeLinks.filter(link => !nodesOffGraph.some(node => node.id === link.from.id || node.id === link.to.id));
        // If routineVersion was mutated, update the routineVersion
        const finalNodes = [...nodesOnGraph, ...nodesOffGraph];
        const haveNodesChanged = !isEqual(finalNodes, changedRoutineVersion.nodes);
        const haveLinksChanged = !isEqual(goodLinks, changedRoutineVersion.nodeLinks);
        if (haveNodesChanged || haveLinksChanged) {
            console.log("calculate memo setting changed routine 2");
            setChangedRoutineVersion({
                ...changedRoutineVersion,
                nodes: finalNodes,
                nodeLinks: goodLinks,
            });
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
    }, [changedRoutineVersion]);

    // Add subroutine dialog
    const [addSubroutineNode, setAddSubroutineNode] = useState<string | null>(null);
    const closeAddSubroutineDialog = useCallback(() => { setAddSubroutineNode(null); }, []);
    // "Add after" link dialog when there is more than one link (i.e. can't be done automatically)
    const [addAfterLinkNode, setAddAfterLinkNode] = useState<string | null>(null);
    const closeAddAfterLinkDialog = useCallback(() => { setAddAfterLinkNode(null); }, []);
    // "Add before" link dialog when there is more than one link (i.e. can't be done automatically)
    const [addBeforeLinkNode, setAddBeforeLinkNode] = useState<string | null>(null);
    const closeAddBeforeLinkDialog = useCallback(() => { setAddBeforeLinkNode(null); }, []);

    // Subroutine info drawer
    const [openedSubroutine, setOpenedSubroutine] = useState<{ node: Node & { routineList: NodeRoutineList }, routineItemId: string } | null>(null);
    const handleSubroutineOpen = useCallback((nodeId: string, subroutineId: string) => {
        const node = nodesById[nodeId];
        if (node && node.routineList) setOpenedSubroutine({ node: node as Node & { routineList: NodeRoutineList }, routineItemId: subroutineId });
    }, [nodesById]);
    const closeRoutineInfo = useCallback(() => {
        setOpenedSubroutine(null);
    }, []);

    const [isLinkDialogOpen, setIsLinkDialogOpen] = useState(false);
    const [linkDialogFrom, setLinkDialogFrom] = useState<Node | null>(null);
    const [linkDialogTo, setLinkDialogTo] = useState<Node | null>(null);
    const openLinkDialog = useCallback(() => setIsLinkDialogOpen(true), []);
    const handleLinkDialogClose = useCallback((link?: NodeLink) => {
        setLinkDialogFrom(null);
        setLinkDialogTo(null);
        setIsLinkDialogOpen(false);
        // If no link data, return
        if (!link) return;
        // Upsert link
        const newLinks = [...changedRoutineVersion.nodeLinks];
        const existingLinkIndex = newLinks.findIndex(l => l.from.id === link.from.id && l.to.id === link.to.id);
        if (existingLinkIndex >= 0) {
            newLinks[existingLinkIndex] = { ...link } as NodeLink;
        } else {
            newLinks.push(link as NodeLink);
        }
        addToChangeStack({
            ...changedRoutineVersion,
            nodeLinks: newLinks,
        });
    }, [addToChangeStack, changedRoutineVersion]);

    /**
     * Deletes a link, without deleting any nodes.
     */
    const handleLinkDelete = useCallback((link: NodeLink) => {
        addToChangeStack({
            ...changedRoutineVersion,
            nodeLinks: changedRoutineVersion.nodeLinks.filter(l => l.id !== link.id),
        });
    }, [addToChangeStack, changedRoutineVersion]);

    const revertChanges = useCallback(() => {
        // Helper function to revert changes
        const revert = () => {
            // If updating routineVersion, revert to original routineVersion
            if (id) {
                clearChangeStack();
                handleCancel();
            }
            // If adding new routineVersion, go back
            else window.history.back();
        };
        // Confirm if changes have been made
        if (changeStack.length > 1) {
            PubSub.get().publishAlertDialog({
                messageKey: "UnsavedChangesBeforeCancel",
                buttons: [
                    { labelKey: "Yes", onClick: () => { revert(); } },
                    { labelKey: "No" },
                ],
            });
        } else {
            revert();
        }
    }, [changeStack.length, clearChangeStack, handleCancel, id]);

    /**
     * If closing with unsaved changes, prompt user to save
     */
    const onClose = useCallback(() => {
        if (isEditing) {
            revertChanges();
        } else {
            keepSearchParams(setLocation, []);
            if (!uuidValidate(id)) window.history.back();
            else handleClose();
        }
    }, [handleClose, id, isEditing, setLocation, revertChanges]);

    /**
     * Calculates the new set of links for an routineVersion when a node is 
     * either deleted or unlinked. In certain cases, the new links can be 
     * calculated automatically.
     * @param nodeId - The ID of the node which is being deleted or unlinked
     * @param currLinks - The current set of links
     * @returns The new set of links
     */
    const calculateLinksAfterNodeRemove = useCallback((nodeId: string): NodeLink[] => {
        const deletingLinks = changedRoutineVersion.nodeLinks.filter(l => l.from.id === nodeId || l.to.id === nodeId);
        const newLinks: NodeLinkShape[] = [];
        // Find all "from" and "to" nodes in the deleting links
        const fromNodeIds = deletingLinks.map(l => l.from.id).filter(id => id !== nodeId);
        const toNodeIds = deletingLinks.map(l => l.to.id).filter(id => id !== nodeId);
        // If there is only one "from" node, create a link between it and every "to" node
        if (fromNodeIds.length === 1) {
            toNodeIds.forEach(toId => { newLinks.push(generateNewLink(fromNodeIds[0], toId, changedRoutineVersion.id)); });
        }
        // If there is only one "to" node, create a link between it and every "from" node
        else if (toNodeIds.length === 1) {
            fromNodeIds.forEach(fromId => { newLinks.push(generateNewLink(fromId, toNodeIds[0], changedRoutineVersion.id)); });
        }
        // NOTE: Every other case is ambiguous, so we can't auto-create create links
        // Delete old links
        const keptLinks = changedRoutineVersion.nodeLinks.filter(l => !deletingLinks.includes(l));
        // Return new links combined with kept links
        return [...keptLinks, ...newLinks as any[]];
    }, [changedRoutineVersion]);

    /**
     * Finds the closest node position available to the given position
     * @param column - The preferred column
     * @param row - The preferred row
     * @returns a node position in the same column, with the first available row starting at the given row
     */
    const closestOpenPosition = useCallback((
        column: number | null | undefined,
        row: number | null | undefined,
    ): { columnIndex: number, rowIndex: number } => {
        if (column === null || column === undefined || row === null || row === undefined) return { columnIndex: -1, rowIndex: -1 };
        const columnNodes = changedRoutineVersion.nodes?.filter(n => n.columnIndex === column) ?? [];
        let rowIndex: number = row;
        // eslint-disable-next-line no-loop-func
        while (columnNodes.some(n => n.rowIndex !== null && n.rowIndex === rowIndex) && rowIndex <= 100) {
            rowIndex++;
        }
        if (rowIndex > 100) return { columnIndex: -1, rowIndex: -1 };
        return { columnIndex: column, rowIndex };
    }, [changedRoutineVersion.nodes]);

    /**
     * Generates a new routine list node object, but doesn't add it to the routine
     * @param column Suggested column for the node
     * @param row Suggested row for the node
     */
    const createRoutineListNode = useCallback((column: number | null | undefined, row: number | null | undefined) => {
        const { columnIndex, rowIndex } = closestOpenPosition(column, row);
        const newNodeId = uuid();
        const newNode: Omit<NodeShape, "routineVersion"> = {
            id: newNodeId,
            nodeType: NodeType.RoutineList,
            rowIndex,
            columnIndex,
            routineList: {
                id: uuid(),
                isOrdered: false,
                isOptional: false,
                items: [],
                node: { id: newNodeId },
            },
            // Generate unique placeholder name
            translations: [{
                id: uuid(),
                language: translationData.language,
                name: `Node ${(changedRoutineVersion.nodes?.length ?? 0) - 1}`,
                description: "",
            }],
        };
        return newNode;
    }, [closestOpenPosition, translationData.language, changedRoutineVersion.nodes?.length]);

    /**
     * Generates new end node object, but doesn't add it to the routineVersion
     * @param column Suggested column for the node
     * @param row Suggested row for the node
     */
    const createEndNode = useCallback((column: number | null, row: number | null) => {
        const { columnIndex, rowIndex } = closestOpenPosition(column, row);
        const newNodeId = uuid();
        const newNode: Omit<NodeShape, "routineVersion"> = {
            id: newNodeId,
            nodeType: NodeType.End,
            rowIndex,
            columnIndex,
            end: {
                id: uuid(),
                node: { id: newNodeId },
                suggestedNextRoutineVersions: [],
                wasSuccessful: true,
            },
            translations: [],
        };
        return newNode;
    }, [closestOpenPosition]);

    /**
     * Creates a link between two nodes which already exist in the linked routineVersion. 
     * This assumes that the link is valid.
     */
    const handleLinkCreate = useCallback((link: NodeLink) => {
        addToChangeStack({
            ...changedRoutineVersion,
            nodeLinks: [...changedRoutineVersion.nodeLinks, link],
        });
    }, [addToChangeStack, changedRoutineVersion]);

    /**
     * Updates an existing link between two nodes
     */
    const handleLinkUpdate = useCallback((link: NodeLink) => {
        const linkIndex = changedRoutineVersion.nodeLinks.findIndex(l => l.id === link.id);
        if (linkIndex === -1) return;
        addToChangeStack({
            ...changedRoutineVersion,
            nodeLinks: updateArray(changedRoutineVersion.nodeLinks, linkIndex, link),
        });
    }, [addToChangeStack, changedRoutineVersion]);

    /**
     * Deletes a node, and all links connected to it. 
     * Also attemps to create new links to replace the deleted links.
     */
    const handleNodeDelete = useCallback((nodeId: string) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const linksList = calculateLinksAfterNodeRemove(nodeId);
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: deleteArrayIndex(changedRoutineVersion.nodes, nodeIndex),
            nodeLinks: linksList,
        });
    }, [addToChangeStack, calculateLinksAfterNodeRemove, changedRoutineVersion]);

    /**
     * Deletes a subroutine from a node
     */
    const handleSubroutineDelete = useCallback((nodeId: string, subroutineId: string) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const node = changedRoutineVersion.nodes[nodeIndex];
        const subroutineIndex = node.routineList!.items.findIndex((item: NodeRoutineListItem) => item.id === subroutineId);
        if (subroutineIndex === -1) return;
        const newRoutineList = deleteArrayIndex(node.routineList!.items, subroutineIndex);
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updateArray(changedRoutineVersion.nodes, nodeIndex, {
                ...node,
                routineList: {
                    ...node.routineList,
                    items: newRoutineList,
                } as any,
            }),
        });
    }, [addToChangeStack, changedRoutineVersion]);

    /**
     * Drops or unlinks a node
     */
    const handleNodeDrop = useCallback((nodeId: string, columnIndex: number | null, rowIndex: number | null) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        // If columnIndex and rowIndex null, then it is being unlinked
        if (columnIndex === null && rowIndex === null) {
            const linksList = calculateLinksAfterNodeRemove(nodeId);
            addToChangeStack({
                ...changedRoutineVersion,
                nodes: updateArray(changedRoutineVersion.nodes, nodeIndex, {
                    ...changedRoutineVersion.nodes[nodeIndex],
                    rowIndex: null,
                    columnIndex: null,
                }),
                nodeLinks: linksList,
            });
            return;
        }
        // If one or the other is null, then there must be an error
        if (columnIndex === null || rowIndex === null) {
            PubSub.get().publishSnack({ messageKey: "InvalidDropLocation", severity: "Error" });
            return;
        }
        // Otherwise, is a drop
        let updatedNodes = [...changedRoutineVersion.nodes];
        // If dropped into the first column, then shift everything that's not the start node to the right
        if (columnIndex === 0) {
            updatedNodes = updatedNodes.map(n => {
                if (n.rowIndex === null || n.columnIndex === null || n.columnIndex === 0) return n;
                return {
                    ...n,
                    columnIndex: (n.columnIndex ?? 0) + 1,
                };
            });
            // Update dropped node
            updatedNodes = updateArray(updatedNodes, nodeIndex, {
                ...changedRoutineVersion.nodes[nodeIndex],
                columnIndex: 1,
                rowIndex,
            });
        }
        // If dropped into the same column the node started in, either shift or swap
        else if (columnIndex === changedRoutineVersion.nodes[nodeIndex].columnIndex) {
            // Find and order nodes in the same column, which are above (or at the same position as) the dropped node
            const nodesAbove = changedRoutineVersion.nodes.filter(n =>
                n.columnIndex === columnIndex &&
                exists(n.rowIndex) &&
                n.rowIndex <= rowIndex,
            ).sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
            // If no nodes above, then shift everything in the column down by 1
            if (nodesAbove.length === 0) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.rowIndex === null || n.columnIndex !== columnIndex) return n;
                    return {
                        ...n,
                        rowIndex: (n.rowIndex ?? 0) + 1,
                    };
                });
            }
            // Otherwise, swap with the last node in the above list
            else {
                updatedNodes = updatedNodes.map(n => {
                    if (n.rowIndex === null || n.columnIndex !== columnIndex) return n;
                    if (n.id === nodeId) return {
                        ...n,
                        rowIndex: nodesAbove[nodesAbove.length - 1].rowIndex,
                    };
                    if (n.rowIndex === nodesAbove[nodesAbove.length - 1].rowIndex) return {
                        ...n,
                        rowIndex: changedRoutineVersion.nodes[nodeIndex].rowIndex,
                    };
                    return n;
                });
            }
        }
        // Otherwise, treat as a normal drop
        else {
            // If dropped into an existing column, shift rows in dropped column that are below the dropped node
            if (changedRoutineVersion.nodes.some(n => n.columnIndex === columnIndex)) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.columnIndex === columnIndex && exists(n.rowIndex) && n.rowIndex >= rowIndex) {
                        return { ...n, rowIndex: n.rowIndex + 1 };
                    }
                    return n;
                });
            }
            // If the column the node was from is now empty, then shift all columns after it.
            const originalColumnIndex = changedRoutineVersion.nodes[nodeIndex].columnIndex;
            const isRemovingColumn = exists(originalColumnIndex) && changedRoutineVersion.nodes.filter(n => n.columnIndex === originalColumnIndex).length === 1;
            if (isRemovingColumn) {
                updatedNodes = updatedNodes.map(n => {
                    if (exists(n.columnIndex) && n.columnIndex > originalColumnIndex) {
                        return { ...n, columnIndex: n.columnIndex - 1 };
                    }
                    return n;
                });
            }
            updatedNodes = updateArray(updatedNodes, nodeIndex, {
                ...changedRoutineVersion.nodes[nodeIndex],
                columnIndex: (isRemovingColumn && originalColumnIndex < columnIndex) ?
                    columnIndex - 1 :
                    columnIndex,
                rowIndex,
            });
        }
        // Update the routineVersion
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updatedNodes,
        });
    }, [addToChangeStack, calculateLinksAfterNodeRemove, changedRoutineVersion]);

    // Move node dialog for context menu (mainly for accessibility)
    const [moveNode, setMoveNode] = useState<Node | null>(null);
    const closeMoveNodeDialog = useCallback((newPosition?: { columnIndex: number, rowIndex: number }) => {
        if (newPosition && moveNode) {
            handleNodeDrop(moveNode.id, newPosition.columnIndex, newPosition.rowIndex);
        }
        setMoveNode(null);
    }, [handleNodeDrop, moveNode]);

    /**
     * Updates a node's data
     */
    const handleNodeUpdate = useCallback((node: Node) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === node.id);
        if (nodeIndex === -1) return;
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updateArray(changedRoutineVersion.nodes, nodeIndex, node),
        });
    }, [addToChangeStack, changedRoutineVersion]);

    /**
     * Inserts a new routine list node along an edge
     */
    const handleNodeInsert = useCallback((link: NodeLink) => {
        // Find link index
        const linkIndex = changedRoutineVersion.nodeLinks.findIndex(l => l.from.id === link.from.id && l.to.id === link.to.id);
        // Delete link
        const linksList = deleteArrayIndex(changedRoutineVersion.nodeLinks, linkIndex);
        // Find "to" node. New node will be placed in its row and column
        const toNode = changedRoutineVersion.nodes.find(n => n.id === link.to.id);
        if (!toNode) {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error" });
            return;
        }
        const { columnIndex, rowIndex } = toNode;
        // Move every node starting from the "to" node to the right by one
        const nodesList = changedRoutineVersion.nodes.map(n => {
            if (exists(n.columnIndex) && n.columnIndex >= (columnIndex ?? 0)) {
                return { ...n, columnIndex: n.columnIndex + 1 };
            }
            return n;
        });
        // Create new routine list node
        const newNode: Omit<NodeShape, "routineVersion"> = createRoutineListNode(columnIndex, rowIndex);
        // Find every node 
        // Create two new links
        const newLinks: NodeLinkShape[] = [
            generateNewLink(link.from.id, newNode.id, changedRoutineVersion.id),
            generateNewLink(newNode.id, link.to.id, changedRoutineVersion.id),
        ];
        // Insert new node and links
        const newRoutine = {
            ...changedRoutineVersion,
            nodes: [...nodesList, newNode as any],
            nodeLinks: [...linksList, ...newLinks as any],
        };
        addToChangeStack(newRoutine);
    }, [addToChangeStack, changedRoutineVersion, createRoutineListNode]);

    /**
     * Inserts a new routine list node, with its own branch
     */
    const handleBranchInsert = useCallback((link: NodeLink) => {
        // Find "to" node. New node will be placed in its column
        const toNode = changedRoutineVersion.nodes.find(n => n.id === link.to.id);
        if (!toNode) {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error" });
            return;
        }
        // Find the largest row index in the column. New node will be placed in the next row
        const maxRowIndex = changedRoutineVersion.nodes.filter(n => n.columnIndex === toNode.columnIndex).map(n => n.rowIndex).reduce((a, b) => Math.max(a ?? 0, b ?? 0), 0);
        const newNode: Omit<NodeShape, "routineVersion"> = createRoutineListNode(toNode.columnIndex, (maxRowIndex ?? toNode.rowIndex ?? 0) + 1);
        // Since this is a new branch, we also need to add an end node after the new node
        const newEndNode: Omit<NodeShape, "routineVersion"> = createEndNode((toNode.columnIndex ?? 0) + 1, (maxRowIndex ?? toNode.rowIndex ?? 0) + 1);
        // Create new link, going from the "from" node to the new node
        const newLink: NodeLinkShape = generateNewLink(link.from.id, newNode.id, changedRoutineVersion.id);
        // Create new link, going from the new node to the end node
        const newEndLink: NodeLinkShape = generateNewLink(newNode.id, newEndNode.id, changedRoutineVersion.id);
        // Insert new nodes and links
        const newRoutine = {
            ...changedRoutineVersion,
            nodes: [...changedRoutineVersion.nodes, newNode as any, newEndNode as any],
            nodeLinks: [...changedRoutineVersion.nodeLinks, newLink as any, newEndLink as any],
        };
        addToChangeStack(newRoutine);
    }, [addToChangeStack, changedRoutineVersion, createEndNode, createRoutineListNode]);

    /**
     * Adds a subroutine routine list
     */
    const handleSubroutineAdd = useCallback((nodeId: string, routineVersion: RoutineVersion) => {
        // Find the node with changes
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        console.log("handleSubroutineAdd", nodeId, routineVersion, nodeIndex, changedRoutineVersion.nodes);
        if (nodeIndex === -1) return;
        // Find the subroutine info in the node
        const routineList: NodeRoutineList = changedRoutineVersion.nodes[nodeIndex].routineList!;
        // Create a routine list item, which wraps the added routine version with some extra info
        const routineItem: NodeRoutineListItem = {
            id: uuid(),
            index: routineList.items.length,
            isOptional: true,
            routineVersion,
        } as NodeRoutineListItem;
        if (routineList.isOrdered) routineItem.index = routineList.items.length;
        // Add the new data to the change stack
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updateArray(changedRoutineVersion.nodes, nodeIndex, {
                ...changedRoutineVersion.nodes[nodeIndex],
                routineList: {
                    ...routineList,
                    items: [...routineList.items, routineItem],
                },
            }),
        });
        // Close dialog
        closeAddSubroutineDialog();
    }, [addToChangeStack, changedRoutineVersion]);

    /**
     * Reoders a subroutine in a routine list item
     * @param nodeId The node id of the routine list item
     * @param oldIndex The old index of the subroutine
     * @param newIndex The new index of the subroutine
     */
    const handleSubroutineReorder = useCallback((nodeId: string, oldIndex: number, newIndex: number) => {
        // Find items being swapped
        // Node containing routine list data with ID nodeId
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1) return;
        const routineList: NodeRoutineListShape = changedRoutineVersion.nodes[nodeIndex].routineList! as NodeRoutineListShape;
        const items = [...routineList.items];
        // Find subroutines matching old and new index
        const aIndex = items.findIndex(r => r.index === oldIndex);
        const bIndex = items.findIndex(r => r.index === newIndex);
        if (aIndex === -1 || bIndex === -1) return;
        // Swap the item indexes
        items[aIndex] = { ...items[aIndex], index: newIndex };
        items[bIndex] = { ...items[bIndex], index: oldIndex };
        // Update the routine list
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updateArray(changedRoutineVersion.nodes, nodeIndex, {
                ...changedRoutineVersion.nodes[nodeIndex],
                routineList: {
                    ...routineList,
                    __typename: "NodeRoutineList",
                    items,
                } as NodeRoutineList,
            }),
        });
    }, [addToChangeStack, changedRoutineVersion]);

    /**
     * Add a new end node AFTER a node
     */
    const handleAddEndAfter = useCallback((nodeId: string) => {
        // Find links where this node is the "from" node
        const links = changedRoutineVersion.nodeLinks.filter(l => l.from.id === nodeId);
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
            const node = changedRoutineVersion.nodes.find(n => n.id === nodeId);
            if (!node) return;
            const newNode: Omit<NodeShape, "routineVersion"> = createEndNode((node.columnIndex ?? 1) + 1, (node.rowIndex ?? 0));
            const newLink: NodeLink = generateNewLink(nodeId, newNode.id, changedRoutineVersion.id) as NodeLink;
            addToChangeStack({
                ...changedRoutineVersion,
                nodes: [...changedRoutineVersion.nodes, newNode as any],
                nodeLinks: [...changedRoutineVersion.nodeLinks, newLink as any],
            });
        }
    }, [addToChangeStack, changedRoutineVersion, createEndNode, handleNodeInsert]);

    /**
     * Add a new routine list AFTER a node
     */
    const handleAddListAfter = useCallback((nodeId: string) => {
        // Find links where this node is the "from" node
        const links = changedRoutineVersion.nodeLinks.filter(l => l.from.id === nodeId);
        // If multiple links, open a dialog to select which one to add after
        if (links.length > 1) {
            setAddAfterLinkNode(nodeId);
            return;
        }
        // If only one link, add after that link
        else if (links.length === 1) {
            handleNodeInsert(links[0]);
        }
        // If no links, create link and node
        else {
            const node = changedRoutineVersion.nodes.find(n => n.id === nodeId);
            if (!node) return;
            const newNode: Omit<NodeShape, "routineVersion"> = createRoutineListNode((node.columnIndex ?? 1) + 1, (node.rowIndex ?? 0));
            const newLink: NodeLink = generateNewLink(nodeId, newNode.id, changedRoutineVersion.id) as NodeLink;
            addToChangeStack({
                ...changedRoutineVersion,
                nodes: [...changedRoutineVersion.nodes, newNode as any],
                nodeLinks: [...changedRoutineVersion.nodeLinks, newLink as any],
            });
        }
    }, [addToChangeStack, changedRoutineVersion, createRoutineListNode, handleNodeInsert]);

    /**
     * Add a new routine list BEFORE a node
     */
    const handleAddListBefore = useCallback((nodeId: string) => {
        // Find links where this node is the "to" node
        const links = changedRoutineVersion.nodeLinks.filter(l => l.to.id === nodeId);
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
            const node = changedRoutineVersion.nodes.find(n => n.id === nodeId);
            if (!node) return;
            const newNode: Omit<NodeShape, "routineVersion"> = createRoutineListNode((node.columnIndex ?? 1) - 1, (node.rowIndex ?? 0));
            const newLink: NodeLink = generateNewLink(newNode.id, nodeId, changedRoutineVersion.id) as NodeLink;
            addToChangeStack({
                ...changedRoutineVersion,
                nodes: [...changedRoutineVersion.nodes, newNode as any],
                nodeLinks: [...changedRoutineVersion.nodeLinks, newLink as any],
            });
        }
    }, [addToChangeStack, changedRoutineVersion, createRoutineListNode, handleNodeInsert]);

    /**
     * Updates the current selected subroutine
     */
    const handleSubroutineUpdate = useCallback((updatedSubroutine: NodeRoutineListItemShape) => {
        // Update routine
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: changedRoutineVersion.nodes.map((n: Node) => {
                if (n.nodeType === NodeType.RoutineList && n.routineList!.items.some(r => r.id === updatedSubroutine.id)) {
                    return {
                        ...n,
                        routineList: {
                            ...n.routineList,
                            items: n.routineList!.items.map(r => {
                                if (r.id === updatedSubroutine.id) {
                                    return {
                                        ...r,
                                        ...updatedSubroutine,
                                        routineVersion: {
                                            ...r.routineVersion,
                                            ...updatedSubroutine.routineVersion,
                                        },
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
    }, [addToChangeStack, changedRoutineVersion, closeRoutineInfo]);

    /**
     * Navigates to a subroutine's build page. Fist checks if there are unsaved changes
     */
    const handleSubroutineViewFull = useCallback(() => {
        if (!openedSubroutine) return;
        if (!isEqual(routineVersion, changedRoutineVersion)) {
            PubSub.get().publishSnack({ messageKey: "SaveChangesBeforeLeaving", severity: "Error" });
            return;
        }
        // TODO - buildview should have its own buildview, to recursively open subroutines
        //setLocation(`${LINKS.Build}/${selectedSubroutine.id}`);
    }, [changedRoutineVersion, openedSubroutine, routineVersion]);

    const handleAction = useCallback((action: BuildAction, nodeId: string, subroutineId?: string) => {
        const node = changedRoutineVersion.nodes?.find(n => n.id === nodeId);
        switch (action) {
            case BuildAction.AddIncomingLink:
                setLinkDialogTo(node ?? null);
                setIsLinkDialogOpen(true);
                break;
            case BuildAction.AddOutgoingLink:
                setLinkDialogFrom(node ?? null);
                setIsLinkDialogOpen(true);
                break;
            case BuildAction.AddSubroutine:
                setAddSubroutineNode(nodeId);
                break;
            case BuildAction.DeleteNode:
                handleNodeDelete(nodeId);
                break;
            case BuildAction.DeleteSubroutine:
                handleSubroutineDelete(nodeId, subroutineId ?? "");
                break;
            case BuildAction.EditSubroutine:
                handleSubroutineOpen(nodeId, subroutineId ?? "");
                break;
            case BuildAction.OpenSubroutine:
                handleSubroutineOpen(nodeId, subroutineId ?? "");
                break;
            case BuildAction.UnlinkNode:
                handleNodeDrop(nodeId, null, null);
                break;
            case BuildAction.AddEndAfterNode:
                handleAddEndAfter(nodeId);
                break;
            case BuildAction.AddListAfterNode:
                handleAddListAfter(nodeId);
                break;
            case BuildAction.AddListBeforeNode:
                handleAddListBefore(nodeId);
                break;
            case BuildAction.MoveNode:
                if (node) setMoveNode(node);
                break;
        }
    }, [changedRoutineVersion.nodes, handleNodeDelete, handleSubroutineDelete, handleSubroutineOpen, handleNodeDrop, handleAddEndAfter, handleAddListAfter, handleAddListBefore]);

    /**
     * Cleans up graph by removing empty columns and row gaps within columns.
     * Also adds end nodes to the end of each unfinished path. 
     * Also removes links that don't have both a valid from.id and to.id.
     */
    const cleanUpGraph = useCallback(() => {
        const resultRoutine = JSON.parse(JSON.stringify(changedRoutineVersion));
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
            if (node.nodeType !== NodeType.End) {
                // Check if any links have a "from.id" matching this node's ID
                const leavingLinks = resultRoutine.nodeLinks.filter(link => link.from.id === node.id);
                // If there are no leaving links, create a new link and end node
                if (leavingLinks.length === 0) {
                    // Generate node ID
                    const newEndNodeId = uuid();
                    // Calculate rowIndex and columnIndex
                    // Column is 1 after current column
                    const columnIndex: number = (node.columnIndex ?? 0) + 1;
                    // Node is 1 after last rowIndex in column
                    const rowIndex = (columnIndex >= 0 && columnIndex < columns.length) ? columns[columnIndex].length : 0;
                    const newLink: NodeLink = generateNewLink(node.id, newEndNodeId, changedRoutineVersion.id) as NodeLink;
                    const newEndNode: Omit<NodeShape, "routineVersion"> = {
                        id: newEndNodeId,
                        nodeType: NodeType.End,
                        rowIndex,
                        columnIndex,
                        end: {
                            id: uuid(),
                            node: { id: newEndNodeId },
                            suggestedNextRoutineVersions: [],
                            wasSuccessful: false,
                        },
                        translations: [],
                    };
                    // Add link and end node to resultRoutine
                    resultRoutine.nodeLinks.push(newLink as any);
                    resultRoutine.nodes.push(newEndNode as any);
                }
            }
        }
        // Remove links that don't have both a valid from.id and to.id
        resultRoutine.nodeLinks = resultRoutine.nodeLinks.filter(link => {
            const fromNode = resultRoutine.nodes.find(n => n.id === link.from.id);
            const toNode = resultRoutine.nodes.find(n => n.id === link.to.id);
            return Boolean(fromNode && toNode);
        });
        // Update changedRoutine with resultRoutine
        addToChangeStack(resultRoutine);
    }, [addToChangeStack, changedRoutineVersion, columns]);

    const languageComponent = useMemo(() => {
        if (isEditing) return (
            <LanguageInput
                currentLanguage={translationData.language}
                handleAdd={translationData.handleAddLanguage}
                handleDelete={translationData.handleDeleteLanguage}
                handleCurrent={translationData.setLanguage}
                languages={translationData.languages}
                zIndex={zIndex}
            />
        );
        return (
            <SelectLanguageMenu
                currentLanguage={translationData.language}
                handleCurrent={translationData.setLanguage}
                languages={translationData.languages}
                zIndex={zIndex}
            />
        );
    }, [translationData, isEditing, zIndex]);

    return (
        <Box sx={{
            display: "flex",
            flexDirection: "column",
            minHeight: "100%",
            height: "100%",
            width: "100%",
        }}>
            {/* Popup for adding new subroutines */}
            {addSubroutineNode && <FindSubroutineDialog
                handleCancel={closeAddSubroutineDialog}
                handleComplete={handleSubroutineAdd}
                isOpen={Boolean(addSubroutineNode)}
                nodeId={addSubroutineNode}
                routineVersionId={routineVersion?.id}
                zIndex={zIndex + 3}
            />}
            {/* Popup for "Add after" dialog */}
            {addAfterLinkNode && <AddAfterLinkDialog
                handleSelect={handleNodeInsert}
                handleClose={closeAddAfterLinkDialog}
                isOpen={Boolean(addAfterLinkNode)}
                nodes={changedRoutineVersion.nodes}
                links={changedRoutineVersion.nodeLinks}
                nodeId={addAfterLinkNode}
                zIndex={zIndex + 3}
            />}
            {/* Popup for "Add before" dialog */}
            {addBeforeLinkNode && <AddBeforeLinkDialog
                handleSelect={handleNodeInsert}
                handleClose={closeAddBeforeLinkDialog}
                isOpen={Boolean(addBeforeLinkNode)}
                nodes={changedRoutineVersion.nodes}
                links={changedRoutineVersion.nodeLinks}
                nodeId={addBeforeLinkNode}
                zIndex={zIndex + 3}
            />}
            {/* Popup for creating new links */}
            {changedRoutineVersion ? <LinkDialog
                handleClose={handleLinkDialogClose as any}
                handleDelete={handleLinkDelete as any}
                isAdd={true}
                isOpen={isLinkDialogOpen}
                language={translationData.language}
                link={undefined}
                nodeFrom={linkDialogFrom as NodeShape}
                nodeTo={linkDialogTo as NodeShape}
                routineVersion={changedRoutineVersion}
                zIndex={zIndex + 3}
            // partial={ }
            /> : null}
            {/* Popup for moving nodes */}
            {moveNode && <MoveNodeDialog
                handleClose={closeMoveNodeDialog}
                isOpen={Boolean(moveNode)}
                language={translationData.language}
                node={moveNode}
                routineVersion={changedRoutineVersion as RoutineVersion}
                zIndex={zIndex + 3}
            />}
            {/* Displays routine information when you click on a routine list item*/}
            <SubroutineInfoDialog
                data={openedSubroutine}
                defaultLanguage={translationData.language}
                isEditing={isEditing}
                handleUpdate={handleSubroutineUpdate as any}
                handleReorder={handleSubroutineReorder}
                handleViewFull={handleSubroutineViewFull}
                open={Boolean(openedSubroutine)}
                onClose={closeRoutineInfo}
                zIndex={zIndex + 3}
            />
            {/* Navbar */}
            <Stack
                id="build-routine-information-bar"
                direction="row"
                spacing={1}
                width="100%"
                display="flex"
                alignItems="center"
                justifyContent="flex-start"
                sx={{
                    zIndex: 2,
                    height: "48px",
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    paddingLeft: "calc(8px + env(safe-area-inset-left))",
                    paddingRight: "calc(8px + env(safe-area-inset-right))",
                }}
            >
                <StatusButton status={status.status} messages={status.messages} />
                {/* Language */}
                {languageComponent}
                {/* Help button */}
                <HelpButton markdown={helpText} sx={{ fill: palette.secondary.light }} />
                {/* Close Icon */}
                <IconButton
                    edge="start"
                    aria-label="close"
                    onClick={onClose}
                    color="inherit"
                    sx={{
                        position: "absolute",
                        right: "env(safe-area-inset-right)",
                    }}
                >
                    <CloseIcon width='32px' height='32px' />
                </IconButton>
            </Stack>
            {/* Buttons displayed when editing (except for submit/cancel) */}
            <GraphActions
                canRedo={canRedo}
                canUndo={canUndo}
                handleCleanUpGraph={cleanUpGraph}
                handleNodeDelete={handleNodeDelete}
                handleOpenLinkDialog={openLinkDialog}
                handleRedo={redo}
                handleUndo={undo}
                isEditing={isEditing}
                language={translationData.language}
                nodesOffGraph={nodesOffGraph}
                zIndex={zIndex}
            />
            <Box sx={{
                background: palette.background.default,
                bottom: "0",
                display: "flex",
                flexDirection: "column",
                position: "fixed",
                width: "100%",
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
                    language={translationData.language}
                    links={changedRoutineVersion.nodeLinks}
                    nodesById={nodesById}
                    zIndex={zIndex}
                />
            </Box>
            <BuildEditButtons
                canCancelMutate={!loading}
                canSubmitMutate={!loading && !isEqual(routineVersion, changedRoutineVersion)}
                errors={{
                    "graph": status.status !== Status.Valid ? status.messages : null,
                    "unchanged": isEqual(routineVersion, changedRoutineVersion) ? "No changes made" : null,
                }}
                handleCancel={revertChanges}
                handleSubmit={() => { handleSubmit(changedRoutineVersion); }}
                isAdding={!uuidValidate(id)}
                isEditing={isEditing}
                loading={loading}
            />
        </Box>
    );
};
