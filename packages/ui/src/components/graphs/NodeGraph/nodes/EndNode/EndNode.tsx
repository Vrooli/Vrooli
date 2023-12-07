import { Box, Tooltip, Typography } from "@mui/material";
import usePress from "hooks/usePress";
import { useCallback, useMemo, useState } from "react";
import { noSelect } from "styles";
import { BuildAction } from "utils/consts";
import { firstString } from "utils/display/stringTools";
import { NodeWithEndCrud } from "views/objects/node";
import { NodeWithEnd } from "views/objects/node/types";
import { DraggableNode, NodeContextMenu, NodeWidth, calculateNodeSize } from "../..";
import { nodeLabel } from "../styles";
import { EndNodeProps } from "../types";

/**
 * Distance before a click is considered a drag
 */
const DRAG_THRESHOLD = 10;

export const EndNode = ({
    canDrag = true,
    handleAction,
    handleDelete,
    handleUpdate,
    isEditing,
    isLinked = true,
    label = "End",
    labelVisible = true,
    language,
    linksIn,
    node,
    scale = 1,
}: EndNodeProps) => {

    /**
     * Border color indicates status of node.
     * Red for not fully connected (missing in links)
     */
    const borderColor = useMemo<string | null>(() => {
        if (!isLinked) return null;
        if (linksIn.length === 0) return "red";
        return null;
    }, [isLinked, linksIn.length]);

    const outerCircleSize = useMemo(() => `min(max(${calculateNodeSize(NodeWidth.End, scale) * 2}px, 5vw), 150px)`, [scale]);
    const innerCircleSize = useMemo(() => `min(max(${calculateNodeSize(NodeWidth.End, scale)}px, 2.5vw), 75px)`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.End, scale) / 2}px, 1.5em)`, [scale]);

    const labelObject = useMemo(() => labelVisible && scale > -2 ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...nodeLabel,
                pointerEvents: "none",
                fontSize,
            }}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, scale, fontSize, label]);

    // Normal click edit menu (title, wasSuccessful, etc.)
    const [editDialogOpen, setEditDialogOpen] = useState<boolean>(false);
    const openEditDialog = useCallback(() => {
        if (isLinked) {
            setEditDialogOpen(!editDialogOpen);
        }
    }, [isLinked, editDialogOpen]);
    const closeEditDialog = useCallback(() => { setEditDialogOpen(false); }, []);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: EventTarget) => {
        // Ignore if not linked or not editing
        if (!canDrag || !isLinked || !isEditing) return;
        setContextAnchor(target);
    }, [canDrag, isLinked, isEditing]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: openEditDialog,
        onRightClick: openContext,
    });

    return (
        <>
            {/* Right-click context menu */}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={[BuildAction.AddListBeforeNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.DeleteNode]}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option, node.id); }}
            />
            {/* Normal-click menu */}
            <NodeWithEndCrud
                display="dialog"
                onCancel={closeEditDialog}
                onClose={closeEditDialog}
                onCompleted={handleUpdate}
                onDeleted={handleDelete as ((data: NodeWithEnd) => unknown)}
                isCreate={false}
                isEditing={isEditing}
                isOpen={editDialogOpen}
                language={language}
                overrideObject={node as NodeWithEnd}
            />
            <DraggableNode
                className="handle"
                nodeId={node.id}
                canDrag={canDrag}
                dragThreshold={DRAG_THRESHOLD}
            >
                <Tooltip placement={"top"} title={isEditing ? `Edit "${firstString(label, "End")}"` : firstString(label, "End")}>
                    <Box
                        id={`${isLinked ? "" : "unlinked-"}node-${node.id}`}
                        aria-owns={contextOpen ? contextId : undefined}
                        {...pressEvents}
                        sx={{
                            width: outerCircleSize,
                            height: outerCircleSize,
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
                            "@media print": {
                                border: `1px solid ${node.end?.wasSuccessful === false ? "#e97691" : "#9ce793"}`,
                                boxShadow: "none",
                            },
                        }}
                    >
                        <Box
                            id={`${isLinked ? "" : "unlinked-"}node-end-inner-circle-${node.id}`}
                            sx={{
                                width: innerCircleSize,
                                height: innerCircleSize,
                                position: "absolute",
                                display: "block",
                                margin: "0",
                                top: "50%",
                                left: "50%",
                                transform: "translate(-50%, -50%)",
                                borderRadius: "100%",
                                border: `2px solid ${node.end?.wasSuccessful === false ? "#e97691" : "#9ce793"}`,
                            } as const}
                        >
                        </Box>
                        {labelObject}
                    </Box>
                </Tooltip>
            </DraggableNode>
        </>
    );
};
