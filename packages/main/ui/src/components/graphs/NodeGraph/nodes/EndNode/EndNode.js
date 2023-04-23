import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { Box, Tooltip, Typography } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { calculateNodeSize, DraggableNode, NodeContextMenu, NodeEndDialog, NodeWidth } from "../..";
import { noSelect } from "../../../../../styles";
import { BuildAction } from "../../../../../utils/consts";
import { firstString } from "../../../../../utils/display/stringTools";
import usePress from "../../../../../utils/hooks/usePress";
import { nodeLabel } from "../styles";
const DRAG_THRESHOLD = 10;
export const EndNode = ({ canDrag = true, handleAction, handleUpdate, isEditing, isLinked = true, label = "End", labelVisible = true, language, linksIn, node, scale = 1, zIndex, }) => {
    const borderColor = useMemo(() => {
        if (!isLinked)
            return null;
        if (linksIn.length === 0)
            return "red";
        return null;
    }, [isLinked, linksIn.length]);
    const { innerCircleSize, outerCircleSize } = useMemo(() => {
        const nodeSize = calculateNodeSize(NodeWidth.End, scale);
        return {
            innerCircleSize: nodeSize / 2,
            outerCircleSize: nodeSize,
        };
    }, [scale]);
    const labelObject = useMemo(() => labelVisible && outerCircleSize > 75 ? (_jsx(Typography, { variant: "h6", sx: {
            ...noSelect,
            ...nodeLabel,
            pointerEvents: "none",
            fontSize: `min(${outerCircleSize / 5}px, 2em)`,
        }, children: label })) : null, [labelVisible, outerCircleSize, label]);
    const [editDialogOpen, setEditDialogOpen] = useState(false);
    const openEditDialog = useCallback((event) => {
        if (isLinked) {
            setEditDialogOpen(!editDialogOpen);
        }
    }, [isLinked, editDialogOpen]);
    const handleEditDialogClose = useCallback((updatedNode) => {
        if (updatedNode)
            handleUpdate(updatedNode);
        setEditDialogOpen(false);
    }, [handleUpdate]);
    const [contextAnchor, setContextAnchor] = useState(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target) => {
        if (!canDrag || !isLinked || !isEditing)
            return;
        setContextAnchor(target);
    }, [canDrag, isLinked, isEditing]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: openEditDialog,
        onRightClick: openContext,
    });
    return (_jsxs(_Fragment, { children: [_jsx(NodeContextMenu, { id: contextId, anchorEl: contextAnchor, availableActions: [BuildAction.AddListBeforeNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.DeleteNode], handleClose: closeContext, handleSelect: (option) => { handleAction(option, node.id); }, zIndex: zIndex + 1 }), _jsx(NodeEndDialog, { handleClose: handleEditDialogClose, isEditing: isEditing, isOpen: editDialogOpen, language: language, node: node, zIndex: zIndex + 1 }), _jsx(DraggableNode, { className: "handle", nodeId: node.id, canDrag: canDrag, dragThreshold: DRAG_THRESHOLD, children: _jsx(Tooltip, { placement: "top", title: isEditing ? `Edit "${firstString(label, "End")}"` : firstString(label, "End"), children: _jsxs(Box, { id: `${isLinked ? "" : "unlinked-"}node-${node.id}`, "aria-owns": contextOpen ? contextId : undefined, ...pressEvents, sx: {
                            width: `max(${outerCircleSize}px, 48px)`,
                            height: `max(${outerCircleSize}px, 48px)`,
                            position: "relative",
                            display: "block",
                            backgroundColor: node.end?.wasSuccessful === false ? "#7c262a" : "#387e30",
                            color: "white",
                            borderRadius: "100%",
                            boxShadow: borderColor ? `0px 0px 12px ${borderColor}` : 12,
                            "&:hover": {
                                filter: "brightness(120%)",
                                transform: "scale(1.1)",
                                transition: "all 0.2s",
                            },
                        }, children: [_jsx(Box, { id: `${isLinked ? "" : "unlinked-"}node-end-inner-circle-${node.id}`, sx: {
                                    width: `max(${innerCircleSize}px, 32px)`,
                                    height: `max(${innerCircleSize}px, 32px)`,
                                    position: "absolute",
                                    display: "block",
                                    margin: "0",
                                    top: "50%",
                                    left: "50%",
                                    transform: "translate(-50%, -50%)",
                                    borderRadius: "100%",
                                    border: `2px solid ${node.end?.wasSuccessful === false ? "#e97691" : "#9ce793"}`,
                                } }), labelObject] }) }) })] }));
};
//# sourceMappingURL=EndNode.js.map