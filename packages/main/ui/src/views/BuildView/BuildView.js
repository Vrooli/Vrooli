import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { NodeType } from "@local/consts";
import { CloseIcon } from "@local/icons";
import { exists, isEqual } from "@local/utils";
import { uuid, uuidValidate } from "@local/uuid";
import { Box, IconButton, Stack, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BuildEditButtons } from "../../components/buttons/BuildEditButtons/BuildEditButtons";
import { HelpButton } from "../../components/buttons/HelpButton/HelpButton";
import { StatusButton } from "../../components/buttons/StatusButton/StatusButton";
import { FindSubroutineDialog } from "../../components/dialogs/FindSubroutineDialog/FindSubroutineDialog";
import { LinkDialog } from "../../components/dialogs/LinkDialog/LinkDialog";
import { SelectLanguageMenu } from "../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { SubroutineInfoDialog } from "../../components/dialogs/SubroutineInfoDialog/SubroutineInfoDialog";
import { AddAfterLinkDialog, AddBeforeLinkDialog, GraphActions, NodeGraph } from "../../components/graphs/NodeGraph";
import { MoveNodeMenu as MoveNodeDialog } from "../../components/graphs/NodeGraph/MoveNodeDialog/MoveNodeDialog";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { BuildAction, Status } from "../../utils/consts";
import { usePromptBeforeUnload } from "../../utils/hooks/usePromptBeforeUnload";
import { PubSub } from "../../utils/pubsub";
import { keepSearchParams, useLocation } from "../../utils/route";
import { getRoutineVersionStatus } from "../../utils/runUtils";
import { deleteArrayIndex, updateArray } from "../../utils/shape/general";
const helpText = "## What am I looking at?\nThis is the **Build** page. Here you can create and edit multi-step routines.\n\n## What is a routine?\nA *routine* is simply a process for completing a task, which takes inputs, performs some action, and outputs some results. By connecting multiple routines together, you can perform arbitrarily complex tasks.\n\nAll valid multi-step routines have a *start* node and at least one *end* node. Each node inbetween stores a list of subroutines, which can be optional or required.\n\nWhen a user runs the routine, they traverse the routine graph from left to right. Each subroutine is rendered as a page, with components such as TextFields for each input. Where the graph splits, users are given a choice of which subroutine to go to next.\n\n## How do I build a multi-step routine?\nIf you are starting from scratch, you will see a *start* node, a *routine list* node, and an *end* node.\n\nYou can press the routine list node to toggle it open/closed. The *open* stats allows you to select existing subroutines from Vrooli, or create a new one.\n\nEach link connecting nodes has a circle. Pressing this circle opens a popup menu with options to insert a node, split the graph, or delete the link.\n\nYou also have the option to *unlink* nodes. These are stored on the top status bar - along with the status indicator, a button to clean up the graph, a button to add a new link, this help button, and an info button that sets overall routine information.";
const generateNewLink = (fromId, toId, routineVersionId) => ({
    id: uuid(),
    from: { id: fromId },
    to: { id: toId },
    routineVersion: { id: routineVersionId },
});
export const BuildView = ({ display = "dialog", handleCancel, handleClose, handleSubmit, isEditing, loading, routineVersion, translationData, zIndex = 200, }) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const id = useMemo(() => routineVersion?.id ?? "", [routineVersion]);
    const [changedRoutineVersion, setChangedRoutineVersion] = useState(routineVersion);
    const [status, setStatus] = useState({ status: Status.Incomplete, messages: ["Calculating..."] });
    const [changeStack, setChangeStack] = useState([]);
    const [changeStackIndex, setChangeStackIndex] = useState(0);
    const clearChangeStack = useCallback(() => {
        setChangeStack(routineVersion ? [routineVersion] : []);
        setChangeStackIndex(routineVersion ? 0 : -1);
        PubSub.get().publishFastUpdate({ duration: 1000 });
        console.log("clearing change stack");
        setChangedRoutineVersion(routineVersion);
    }, [routineVersion]);
    const undo = useCallback(() => {
        if (changeStackIndex > 0) {
            setChangeStackIndex(changeStackIndex - 1);
            PubSub.get().publishFastUpdate({ duration: 1000 });
            console.log("undoing");
            setChangedRoutineVersion(changeStack[changeStackIndex - 1]);
        }
    }, [changeStackIndex, changeStack, setChangedRoutineVersion]);
    const canUndo = useMemo(() => changeStackIndex > 0 && changeStack.length > 0, [changeStackIndex, changeStack]);
    const redo = useCallback(() => {
        if (changeStackIndex < changeStack.length - 1) {
            setChangeStackIndex(changeStackIndex + 1);
            PubSub.get().publishFastUpdate({ duration: 1000 });
            console.log("redoing");
            setChangedRoutineVersion(changeStack[changeStackIndex + 1]);
        }
    }, [changeStackIndex, changeStack, setChangedRoutineVersion]);
    const canRedo = useMemo(() => changeStackIndex < changeStack.length - 1 && changeStack.length > 0, [changeStackIndex, changeStack]);
    const addToChangeStack = useCallback((changedRoutine) => {
        const newChangeStack = [...changeStack];
        newChangeStack.splice(changeStackIndex + 1, newChangeStack.length - changeStackIndex - 1);
        newChangeStack.push(changedRoutine);
        setChangeStack(newChangeStack);
        setChangeStackIndex(newChangeStack.length - 1);
        PubSub.get().publishFastUpdate({ duration: 1000 });
        console.log("adding to change stack", changedRoutine);
        setChangedRoutineVersion(changedRoutine);
    }, [changeStack, changeStackIndex, setChangeStack, setChangeStackIndex, setChangedRoutineVersion]);
    useEffect(() => {
        const handleKeyDown = (e) => {
            if (e.ctrlKey && (e.key === "y" || e.key === "Z")) {
                redo();
            }
            else if (e.ctrlKey && e.key === "z") {
                undo();
            }
        };
        document.addEventListener("keydown", handleKeyDown);
        return () => { document.removeEventListener("keydown", handleKeyDown); };
    }, [redo, undo]);
    useEffect(() => {
        clearChangeStack();
    }, [clearChangeStack, routineVersion]);
    usePromptBeforeUnload({ shouldPrompt: isEditing && changeStack.length > 1 });
    const { columns, nodesOffGraph, nodesById } = useMemo(() => {
        if (!changedRoutineVersion)
            return { columns: [], nodesOffGraph: [], nodesById: {} };
        const { messages, nodesById, nodesOnGraph, nodesOffGraph, status } = getRoutineVersionStatus(changedRoutineVersion);
        if (messages.includes("No node or link data found")) {
            return { columns: [], nodesOffGraph: [], nodesById: {} };
        }
        if (messages.includes("Ran into error determining node positions")) {
            console.log("calculate memo setting changed routine 1");
            setChangedRoutineVersion({
                ...changedRoutineVersion,
                nodes: changedRoutineVersion.nodes.map(n => ({ ...n, columnIndex: null, rowIndex: null })),
                nodeLinks: [],
            });
            return { columns: [], nodesOffGraph: changedRoutineVersion.nodes, nodesById: {} };
        }
        setStatus({ status, messages });
        const goodLinks = changedRoutineVersion.nodeLinks.filter(link => !nodesOffGraph.some(node => node.id === link.from.id || node.id === link.to.id));
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
        const columns = [];
        for (const node of nodesOnGraph) {
            if ((node.columnIndex === null || node.columnIndex === undefined) || (node.rowIndex === null || node.rowIndex === undefined))
                continue;
            while (columns.length <= node.columnIndex) {
                columns.push([]);
            }
            columns[node.columnIndex].push(node);
        }
        for (const column of columns) {
            column.sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
        }
        columns.push([]);
        return { columns, nodesOffGraph, nodesById };
    }, [changedRoutineVersion]);
    const [addSubroutineNode, setAddSubroutineNode] = useState(null);
    const closeAddSubroutineDialog = useCallback(() => { setAddSubroutineNode(null); }, []);
    const [addAfterLinkNode, setAddAfterLinkNode] = useState(null);
    const closeAddAfterLinkDialog = useCallback(() => { setAddAfterLinkNode(null); }, []);
    const [addBeforeLinkNode, setAddBeforeLinkNode] = useState(null);
    const closeAddBeforeLinkDialog = useCallback(() => { setAddBeforeLinkNode(null); }, []);
    const [openedSubroutine, setOpenedSubroutine] = useState(null);
    const handleSubroutineOpen = useCallback((nodeId, subroutineId) => {
        const node = nodesById[nodeId];
        if (node && node.routineList)
            setOpenedSubroutine({ node: node, routineItemId: subroutineId });
    }, [nodesById]);
    const closeRoutineInfo = useCallback(() => {
        setOpenedSubroutine(null);
    }, []);
    const [isLinkDialogOpen, setIsLinkDialogOpen] = useState(false);
    const [linkDialogFrom, setLinkDialogFrom] = useState(null);
    const [linkDialogTo, setLinkDialogTo] = useState(null);
    const openLinkDialog = useCallback(() => setIsLinkDialogOpen(true), []);
    const handleLinkDialogClose = useCallback((link) => {
        setLinkDialogFrom(null);
        setLinkDialogTo(null);
        setIsLinkDialogOpen(false);
        if (!link)
            return;
        const newLinks = [...changedRoutineVersion.nodeLinks];
        const existingLinkIndex = newLinks.findIndex(l => l.from.id === link.from.id && l.to.id === link.to.id);
        if (existingLinkIndex >= 0) {
            newLinks[existingLinkIndex] = { ...link };
        }
        else {
            newLinks.push(link);
        }
        addToChangeStack({
            ...changedRoutineVersion,
            nodeLinks: newLinks,
        });
    }, [addToChangeStack, changedRoutineVersion]);
    const handleLinkDelete = useCallback((link) => {
        addToChangeStack({
            ...changedRoutineVersion,
            nodeLinks: changedRoutineVersion.nodeLinks.filter(l => l.id !== link.id),
        });
    }, [addToChangeStack, changedRoutineVersion]);
    const revertChanges = useCallback(() => {
        const revert = () => {
            if (id) {
                clearChangeStack();
                handleCancel();
            }
            else
                window.history.back();
        };
        if (changeStack.length > 1) {
            PubSub.get().publishAlertDialog({
                messageKey: "UnsavedChangesBeforeCancel",
                buttons: [
                    { labelKey: "Yes", onClick: () => { revert(); } },
                    { labelKey: "No" },
                ],
            });
        }
        else {
            revert();
        }
    }, [changeStack.length, clearChangeStack, handleCancel, id]);
    const onClose = useCallback(() => {
        if (isEditing) {
            revertChanges();
        }
        else {
            keepSearchParams(setLocation, []);
            if (!uuidValidate(id))
                window.history.back();
            else
                handleClose();
        }
    }, [handleClose, id, isEditing, setLocation, revertChanges]);
    const calculateLinksAfterNodeRemove = useCallback((nodeId) => {
        const deletingLinks = changedRoutineVersion.nodeLinks.filter(l => l.from.id === nodeId || l.to.id === nodeId);
        const newLinks = [];
        const fromNodeIds = deletingLinks.map(l => l.from.id).filter(id => id !== nodeId);
        const toNodeIds = deletingLinks.map(l => l.to.id).filter(id => id !== nodeId);
        if (fromNodeIds.length === 1) {
            toNodeIds.forEach(toId => { newLinks.push(generateNewLink(fromNodeIds[0], toId, changedRoutineVersion.id)); });
        }
        else if (toNodeIds.length === 1) {
            fromNodeIds.forEach(fromId => { newLinks.push(generateNewLink(fromId, toNodeIds[0], changedRoutineVersion.id)); });
        }
        const keptLinks = changedRoutineVersion.nodeLinks.filter(l => !deletingLinks.includes(l));
        return [...keptLinks, ...newLinks];
    }, [changedRoutineVersion]);
    const closestOpenPosition = useCallback((column, row) => {
        if (column === null || column === undefined || row === null || row === undefined)
            return { columnIndex: -1, rowIndex: -1 };
        const columnNodes = changedRoutineVersion.nodes?.filter(n => n.columnIndex === column) ?? [];
        let rowIndex = row;
        while (columnNodes.some(n => n.rowIndex !== null && n.rowIndex === rowIndex) && rowIndex <= 100) {
            rowIndex++;
        }
        if (rowIndex > 100)
            return { columnIndex: -1, rowIndex: -1 };
        return { columnIndex: column, rowIndex };
    }, [changedRoutineVersion.nodes]);
    const createRoutineListNode = useCallback((column, row) => {
        const { columnIndex, rowIndex } = closestOpenPosition(column, row);
        const newNodeId = uuid();
        const newNode = {
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
            translations: [{
                    id: uuid(),
                    language: translationData.language,
                    name: `Node ${(changedRoutineVersion.nodes?.length ?? 0) - 1}`,
                    description: "",
                }],
        };
        return newNode;
    }, [closestOpenPosition, translationData.language, changedRoutineVersion.nodes?.length]);
    const createEndNode = useCallback((column, row) => {
        const { columnIndex, rowIndex } = closestOpenPosition(column, row);
        const newNodeId = uuid();
        const newNode = {
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
    const handleLinkCreate = useCallback((link) => {
        addToChangeStack({
            ...changedRoutineVersion,
            nodeLinks: [...changedRoutineVersion.nodeLinks, link],
        });
    }, [addToChangeStack, changedRoutineVersion]);
    const handleLinkUpdate = useCallback((link) => {
        const linkIndex = changedRoutineVersion.nodeLinks.findIndex(l => l.id === link.id);
        if (linkIndex === -1)
            return;
        addToChangeStack({
            ...changedRoutineVersion,
            nodeLinks: updateArray(changedRoutineVersion.nodeLinks, linkIndex, link),
        });
    }, [addToChangeStack, changedRoutineVersion]);
    const handleNodeDelete = useCallback((nodeId) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1)
            return;
        const linksList = calculateLinksAfterNodeRemove(nodeId);
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: deleteArrayIndex(changedRoutineVersion.nodes, nodeIndex),
            nodeLinks: linksList,
        });
    }, [addToChangeStack, calculateLinksAfterNodeRemove, changedRoutineVersion]);
    const handleSubroutineDelete = useCallback((nodeId, subroutineId) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1)
            return;
        const node = changedRoutineVersion.nodes[nodeIndex];
        const subroutineIndex = node.routineList.items.findIndex((item) => item.id === subroutineId);
        if (subroutineIndex === -1)
            return;
        const newRoutineList = deleteArrayIndex(node.routineList.items, subroutineIndex);
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updateArray(changedRoutineVersion.nodes, nodeIndex, {
                ...node,
                routineList: {
                    ...node.routineList,
                    items: newRoutineList,
                },
            }),
        });
    }, [addToChangeStack, changedRoutineVersion]);
    const handleNodeDrop = useCallback((nodeId, columnIndex, rowIndex) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1)
            return;
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
        if (columnIndex === null || rowIndex === null) {
            PubSub.get().publishSnack({ messageKey: "InvalidDropLocation", severity: "Error" });
            return;
        }
        let updatedNodes = [...changedRoutineVersion.nodes];
        if (columnIndex === 0) {
            updatedNodes = updatedNodes.map(n => {
                if (n.rowIndex === null || n.columnIndex === null || n.columnIndex === 0)
                    return n;
                return {
                    ...n,
                    columnIndex: (n.columnIndex ?? 0) + 1,
                };
            });
            updatedNodes = updateArray(updatedNodes, nodeIndex, {
                ...changedRoutineVersion.nodes[nodeIndex],
                columnIndex: 1,
                rowIndex,
            });
        }
        else if (columnIndex === changedRoutineVersion.nodes[nodeIndex].columnIndex) {
            const nodesAbove = changedRoutineVersion.nodes.filter(n => n.columnIndex === columnIndex &&
                exists(n.rowIndex) &&
                n.rowIndex <= rowIndex).sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
            if (nodesAbove.length === 0) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.rowIndex === null || n.columnIndex !== columnIndex)
                        return n;
                    return {
                        ...n,
                        rowIndex: (n.rowIndex ?? 0) + 1,
                    };
                });
            }
            else {
                updatedNodes = updatedNodes.map(n => {
                    if (n.rowIndex === null || n.columnIndex !== columnIndex)
                        return n;
                    if (n.id === nodeId)
                        return {
                            ...n,
                            rowIndex: nodesAbove[nodesAbove.length - 1].rowIndex,
                        };
                    if (n.rowIndex === nodesAbove[nodesAbove.length - 1].rowIndex)
                        return {
                            ...n,
                            rowIndex: changedRoutineVersion.nodes[nodeIndex].rowIndex,
                        };
                    return n;
                });
            }
        }
        else {
            if (changedRoutineVersion.nodes.some(n => n.columnIndex === columnIndex)) {
                updatedNodes = updatedNodes.map(n => {
                    if (n.columnIndex === columnIndex && exists(n.rowIndex) && n.rowIndex >= rowIndex) {
                        return { ...n, rowIndex: n.rowIndex + 1 };
                    }
                    return n;
                });
            }
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
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updatedNodes,
        });
    }, [addToChangeStack, calculateLinksAfterNodeRemove, changedRoutineVersion]);
    const [moveNode, setMoveNode] = useState(null);
    const closeMoveNodeDialog = useCallback((newPosition) => {
        if (newPosition && moveNode) {
            handleNodeDrop(moveNode.id, newPosition.columnIndex, newPosition.rowIndex);
        }
        setMoveNode(null);
    }, [handleNodeDrop, moveNode]);
    const handleNodeUpdate = useCallback((node) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === node.id);
        if (nodeIndex === -1)
            return;
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updateArray(changedRoutineVersion.nodes, nodeIndex, node),
        });
    }, [addToChangeStack, changedRoutineVersion]);
    const handleNodeInsert = useCallback((link) => {
        const linkIndex = changedRoutineVersion.nodeLinks.findIndex(l => l.from.id === link.from.id && l.to.id === link.to.id);
        const linksList = deleteArrayIndex(changedRoutineVersion.nodeLinks, linkIndex);
        const toNode = changedRoutineVersion.nodes.find(n => n.id === link.to.id);
        if (!toNode) {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error" });
            return;
        }
        const { columnIndex, rowIndex } = toNode;
        const nodesList = changedRoutineVersion.nodes.map(n => {
            if (exists(n.columnIndex) && n.columnIndex >= (columnIndex ?? 0)) {
                return { ...n, columnIndex: n.columnIndex + 1 };
            }
            return n;
        });
        const newNode = createRoutineListNode(columnIndex, rowIndex);
        const newLinks = [
            generateNewLink(link.from.id, newNode.id, changedRoutineVersion.id),
            generateNewLink(newNode.id, link.to.id, changedRoutineVersion.id),
        ];
        const newRoutine = {
            ...changedRoutineVersion,
            nodes: [...nodesList, newNode],
            nodeLinks: [...linksList, ...newLinks],
        };
        addToChangeStack(newRoutine);
    }, [addToChangeStack, changedRoutineVersion, createRoutineListNode]);
    const handleBranchInsert = useCallback((link) => {
        const toNode = changedRoutineVersion.nodes.find(n => n.id === link.to.id);
        if (!toNode) {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error" });
            return;
        }
        const maxRowIndex = changedRoutineVersion.nodes.filter(n => n.columnIndex === toNode.columnIndex).map(n => n.rowIndex).reduce((a, b) => Math.max(a ?? 0, b ?? 0), 0);
        const newNode = createRoutineListNode(toNode.columnIndex, (maxRowIndex ?? toNode.rowIndex ?? 0) + 1);
        const newEndNode = createEndNode((toNode.columnIndex ?? 0) + 1, (maxRowIndex ?? toNode.rowIndex ?? 0) + 1);
        const newLink = generateNewLink(link.from.id, newNode.id, changedRoutineVersion.id);
        const newEndLink = generateNewLink(newNode.id, newEndNode.id, changedRoutineVersion.id);
        const newRoutine = {
            ...changedRoutineVersion,
            nodes: [...changedRoutineVersion.nodes, newNode, newEndNode],
            nodeLinks: [...changedRoutineVersion.nodeLinks, newLink, newEndLink],
        };
        addToChangeStack(newRoutine);
    }, [addToChangeStack, changedRoutineVersion, createEndNode, createRoutineListNode]);
    const handleSubroutineAdd = useCallback((nodeId, routineVersion) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        console.log("handleSubroutineAdd", nodeId, routineVersion, nodeIndex, changedRoutineVersion.nodes);
        if (nodeIndex === -1)
            return;
        const routineList = changedRoutineVersion.nodes[nodeIndex].routineList;
        const routineItem = {
            id: uuid(),
            index: routineList.items.length,
            isOptional: true,
            routineVersion,
        };
        if (routineList.isOrdered)
            routineItem.index = routineList.items.length;
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
    }, [addToChangeStack, changedRoutineVersion]);
    const handleSubroutineReorder = useCallback((nodeId, oldIndex, newIndex) => {
        const nodeIndex = changedRoutineVersion.nodes.findIndex(n => n.id === nodeId);
        if (nodeIndex === -1)
            return;
        const routineList = changedRoutineVersion.nodes[nodeIndex].routineList;
        const items = [...routineList.items];
        const aIndex = items.findIndex(r => r.index === oldIndex);
        const bIndex = items.findIndex(r => r.index === newIndex);
        if (aIndex === -1 || bIndex === -1)
            return;
        items[aIndex] = { ...items[aIndex], index: newIndex };
        items[bIndex] = { ...items[bIndex], index: oldIndex };
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: updateArray(changedRoutineVersion.nodes, nodeIndex, {
                ...changedRoutineVersion.nodes[nodeIndex],
                routineList: {
                    ...routineList,
                    __typename: "NodeRoutineList",
                    items,
                },
            }),
        });
    }, [addToChangeStack, changedRoutineVersion]);
    const handleAddEndAfter = useCallback((nodeId) => {
        const links = changedRoutineVersion.nodeLinks.filter(l => l.from.id === nodeId);
        if (links.length > 1) {
            setAddAfterLinkNode(nodeId);
            return;
        }
        else if (links.length === 1) {
            const link = links[0];
            handleNodeInsert(link);
        }
        else {
            const node = changedRoutineVersion.nodes.find(n => n.id === nodeId);
            if (!node)
                return;
            const newNode = createEndNode((node.columnIndex ?? 1) + 1, (node.rowIndex ?? 0));
            const newLink = generateNewLink(nodeId, newNode.id, changedRoutineVersion.id);
            addToChangeStack({
                ...changedRoutineVersion,
                nodes: [...changedRoutineVersion.nodes, newNode],
                nodeLinks: [...changedRoutineVersion.nodeLinks, newLink],
            });
        }
    }, [addToChangeStack, changedRoutineVersion, createEndNode, handleNodeInsert]);
    const handleAddListAfter = useCallback((nodeId) => {
        const links = changedRoutineVersion.nodeLinks.filter(l => l.from.id === nodeId);
        if (links.length > 1) {
            setAddAfterLinkNode(nodeId);
            return;
        }
        else if (links.length === 1) {
            handleNodeInsert(links[0]);
        }
        else {
            const node = changedRoutineVersion.nodes.find(n => n.id === nodeId);
            if (!node)
                return;
            const newNode = createRoutineListNode((node.columnIndex ?? 1) + 1, (node.rowIndex ?? 0));
            const newLink = generateNewLink(nodeId, newNode.id, changedRoutineVersion.id);
            addToChangeStack({
                ...changedRoutineVersion,
                nodes: [...changedRoutineVersion.nodes, newNode],
                nodeLinks: [...changedRoutineVersion.nodeLinks, newLink],
            });
        }
    }, [addToChangeStack, changedRoutineVersion, createRoutineListNode, handleNodeInsert]);
    const handleAddListBefore = useCallback((nodeId) => {
        const links = changedRoutineVersion.nodeLinks.filter(l => l.to.id === nodeId);
        if (links.length > 1) {
            setAddBeforeLinkNode(nodeId);
            return;
        }
        else if (links.length === 1) {
            const link = links[0];
            handleNodeInsert(link);
        }
        else {
            const node = changedRoutineVersion.nodes.find(n => n.id === nodeId);
            if (!node)
                return;
            const newNode = createRoutineListNode((node.columnIndex ?? 1) - 1, (node.rowIndex ?? 0));
            const newLink = generateNewLink(newNode.id, nodeId, changedRoutineVersion.id);
            addToChangeStack({
                ...changedRoutineVersion,
                nodes: [...changedRoutineVersion.nodes, newNode],
                nodeLinks: [...changedRoutineVersion.nodeLinks, newLink],
            });
        }
    }, [addToChangeStack, changedRoutineVersion, createRoutineListNode, handleNodeInsert]);
    const handleSubroutineUpdate = useCallback((updatedSubroutine) => {
        addToChangeStack({
            ...changedRoutineVersion,
            nodes: changedRoutineVersion.nodes.map((n) => {
                if (n.nodeType === NodeType.RoutineList && n.routineList.items.some(r => r.id === updatedSubroutine.id)) {
                    return {
                        ...n,
                        routineList: {
                            ...n.routineList,
                            items: n.routineList.items.map(r => {
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
        });
        closeRoutineInfo();
    }, [addToChangeStack, changedRoutineVersion, closeRoutineInfo]);
    const handleSubroutineViewFull = useCallback(() => {
        if (!openedSubroutine)
            return;
        if (!isEqual(routineVersion, changedRoutineVersion)) {
            PubSub.get().publishSnack({ messageKey: "SaveChangesBeforeLeaving", severity: "Error" });
            return;
        }
    }, [changedRoutineVersion, openedSubroutine, routineVersion]);
    const handleAction = useCallback((action, nodeId, subroutineId) => {
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
                if (node)
                    setMoveNode(node);
                break;
        }
    }, [changedRoutineVersion.nodes, handleNodeDelete, handleSubroutineDelete, handleSubroutineOpen, handleNodeDrop, handleAddEndAfter, handleAddListAfter, handleAddListBefore]);
    const cleanUpGraph = useCallback(() => {
        const resultRoutine = JSON.parse(JSON.stringify(changedRoutineVersion));
        for (const column of columns) {
            const sortedNodes = column.sort((a, b) => (a.rowIndex ?? 0) - (b.rowIndex ?? 0));
            if (sortedNodes.length > 0 && sortedNodes.some((n, i) => (n.rowIndex ?? 0) !== i)) {
                const newNodes = sortedNodes.map((n, i) => ({
                    ...n,
                    rowIndex: i,
                }));
                resultRoutine.nodes = resultRoutine.nodes.map(oldNode => {
                    const newNode = newNodes.find(nn => nn.id === oldNode.id);
                    if (newNode) {
                        return newNode;
                    }
                    return oldNode;
                });
            }
        }
        for (const node of resultRoutine.nodes) {
            if (node.nodeType !== NodeType.End) {
                const leavingLinks = resultRoutine.nodeLinks.filter(link => link.from.id === node.id);
                if (leavingLinks.length === 0) {
                    const newEndNodeId = uuid();
                    const columnIndex = (node.columnIndex ?? 0) + 1;
                    const rowIndex = (columnIndex >= 0 && columnIndex < columns.length) ? columns[columnIndex].length : 0;
                    const newLink = generateNewLink(node.id, newEndNodeId, changedRoutineVersion.id);
                    const newEndNode = {
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
                    resultRoutine.nodeLinks.push(newLink);
                    resultRoutine.nodes.push(newEndNode);
                }
            }
        }
        resultRoutine.nodeLinks = resultRoutine.nodeLinks.filter(link => {
            const fromNode = resultRoutine.nodes.find(n => n.id === link.from.id);
            const toNode = resultRoutine.nodes.find(n => n.id === link.to.id);
            return Boolean(fromNode && toNode);
        });
        addToChangeStack(resultRoutine);
    }, [addToChangeStack, changedRoutineVersion, columns]);
    const languageComponent = useMemo(() => {
        if (isEditing)
            return (_jsx(LanguageInput, { currentLanguage: translationData.language, handleAdd: translationData.handleAddLanguage, handleDelete: translationData.handleDeleteLanguage, handleCurrent: translationData.setLanguage, languages: translationData.languages, zIndex: zIndex }));
        return (_jsx(SelectLanguageMenu, { currentLanguage: translationData.language, handleCurrent: translationData.setLanguage, languages: translationData.languages, zIndex: zIndex }));
    }, [translationData, isEditing, zIndex]);
    return (_jsxs(Box, { sx: {
            display: "flex",
            flexDirection: "column",
            minHeight: "100%",
            height: "100%",
            width: "100%",
        }, children: [addSubroutineNode && _jsx(FindSubroutineDialog, { handleCancel: closeAddSubroutineDialog, handleComplete: handleSubroutineAdd, isOpen: Boolean(addSubroutineNode), nodeId: addSubroutineNode, routineVersionId: routineVersion?.id, zIndex: zIndex + 3 }), addAfterLinkNode && _jsx(AddAfterLinkDialog, { handleSelect: handleNodeInsert, handleClose: closeAddAfterLinkDialog, isOpen: Boolean(addAfterLinkNode), nodes: changedRoutineVersion.nodes, links: changedRoutineVersion.nodeLinks, nodeId: addAfterLinkNode, zIndex: zIndex + 3 }), addBeforeLinkNode && _jsx(AddBeforeLinkDialog, { handleSelect: handleNodeInsert, handleClose: closeAddBeforeLinkDialog, isOpen: Boolean(addBeforeLinkNode), nodes: changedRoutineVersion.nodes, links: changedRoutineVersion.nodeLinks, nodeId: addBeforeLinkNode, zIndex: zIndex + 3 }), changedRoutineVersion ? _jsx(LinkDialog, { handleClose: handleLinkDialogClose, handleDelete: handleLinkDelete, isAdd: true, isOpen: isLinkDialogOpen, language: translationData.language, link: undefined, nodeFrom: linkDialogFrom, nodeTo: linkDialogTo, routineVersion: changedRoutineVersion, zIndex: zIndex + 3 }) : null, moveNode && _jsx(MoveNodeDialog, { handleClose: closeMoveNodeDialog, isOpen: Boolean(moveNode), language: translationData.language, node: moveNode, routineVersion: changedRoutineVersion, zIndex: zIndex + 3 }), _jsx(SubroutineInfoDialog, { data: openedSubroutine, defaultLanguage: translationData.language, isEditing: isEditing, handleUpdate: handleSubroutineUpdate, handleReorder: handleSubroutineReorder, handleViewFull: handleSubroutineViewFull, open: Boolean(openedSubroutine), onClose: closeRoutineInfo, zIndex: zIndex + 3 }), _jsxs(Stack, { id: "build-routine-information-bar", direction: "row", spacing: 1, width: "100%", display: "flex", alignItems: "center", justifyContent: "flex-start", sx: {
                    zIndex: 2,
                    height: "48px",
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    paddingLeft: "calc(8px + env(safe-area-inset-left))",
                    paddingRight: "calc(8px + env(safe-area-inset-right))",
                }, children: [_jsx(StatusButton, { status: status.status, messages: status.messages }), languageComponent, _jsx(HelpButton, { markdown: helpText, sx: { fill: palette.secondary.light } }), _jsx(IconButton, { edge: "start", "aria-label": "close", onClick: onClose, color: "inherit", sx: {
                            position: "absolute",
                            right: "env(safe-area-inset-right)",
                        }, children: _jsx(CloseIcon, { width: '32px', height: '32px' }) })] }), _jsx(GraphActions, { canRedo: canRedo, canUndo: canUndo, handleCleanUpGraph: cleanUpGraph, handleNodeDelete: handleNodeDelete, handleOpenLinkDialog: openLinkDialog, handleRedo: redo, handleUndo: undo, isEditing: isEditing, language: translationData.language, nodesOffGraph: nodesOffGraph, zIndex: zIndex }), _jsx(Box, { sx: {
                    background: palette.background.default,
                    bottom: "0",
                    display: "flex",
                    flexDirection: "column",
                    position: "fixed",
                    width: "100%",
                }, children: _jsx(NodeGraph, { columns: columns, handleAction: handleAction, handleBranchInsert: handleBranchInsert, handleLinkCreate: handleLinkCreate, handleLinkUpdate: handleLinkUpdate, handleLinkDelete: handleLinkDelete, handleNodeInsert: handleNodeInsert, handleNodeUpdate: handleNodeUpdate, handleNodeDrop: handleNodeDrop, isEditing: isEditing, labelVisible: true, language: translationData.language, links: changedRoutineVersion.nodeLinks, nodesById: nodesById, zIndex: zIndex }) }), _jsx(BuildEditButtons, { canCancelMutate: !loading, canSubmitMutate: !loading && !isEqual(routineVersion, changedRoutineVersion), errors: {
                    "graph": status.status !== Status.Valid ? status.messages : null,
                    "unchanged": isEqual(routineVersion, changedRoutineVersion) ? "No changes made" : null,
                }, handleCancel: revertChanges, handleSubmit: () => { handleSubmit(changedRoutineVersion); }, isAdding: !uuidValidate(id), isEditing: isEditing, loading: loading })] }));
};
//# sourceMappingURL=BuildView.js.map