import { Box, BoxProps, Tooltip, Typography, TypographyProps, styled } from "@mui/material";
import { UsePressEvent, usePress } from "hooks/gestures";
import { useCallback, useMemo, useState } from "react";
import { noSelect } from "styles";
import { BuildAction } from "utils/consts";
import { firstString } from "utils/display/stringTools";
import { NodeWithEndCrud } from "views/objects/node";
import { NodeWithEnd, NodeWithEndShape } from "views/objects/node/types";
import { DRAG_THRESHOLD, DraggableNode, NodeContextMenu, NodeWidth, SHOW_TITLE_ABOVE_SCALE, calculateNodeSize } from "../..";
import { nodeLabel } from "../styles";
import { EndNodeProps } from "../types";

interface NodeTitleProps extends TypographyProps {
    fontSize: string;
    scale: number;
}

const NodeTitle = styled(Typography, {
    shouldForwardProp: (prop) => prop !== "fontSize" && prop !== "scale",
})<NodeTitleProps>(({ fontSize, scale }) => ({
    ...noSelect,
    ...nodeLabel,
    pointerEvents: "none",
    fontSize,
    display: scale < SHOW_TITLE_ABOVE_SCALE ? "none" : "block",
} as any));

interface OuterCircleProps extends BoxProps {
    node: NodeWithEndShape;
    scale: number;
    statusColor: string | null;
}

const OuterCircle = styled(Box, {
    shouldForwardProp: (prop) => prop !== "node" && prop !== "scale" && prop !== "statusColor",
})<OuterCircleProps>(({ node, scale, statusColor, theme }) => {
    const outerCircleSize = `min(max(${calculateNodeSize(NodeWidth.End, scale) * 2}px, 5vw), 150px)`;

    return {
        width: outerCircleSize,
        height: outerCircleSize,
        position: "relative",
        display: "block",
        backgroundColor: node.end?.wasSuccessful === false ? "#7c262a" : "#387e30",
        color: "white",
        borderRadius: "100%",
        boxShadow: statusColor ? `0px 0px 12px ${statusColor}` : theme.shadows[12],
        "&:hover": {
            filter: "brightness(120%)",
            transition: "all 0.2s",
        },
        "@media print": {
            border: `1px solid ${node.end?.wasSuccessful === false ? "#e97691" : "#9ce793"}`,
            boxShadow: "none",
        },
    };
});

interface InnerCircleProps extends BoxProps {
    node: NodeWithEndShape;
    scale: number;
}

const InnerCircle = styled(Box, {
    shouldForwardProp: (prop) => prop !== "node" && prop !== "scale",
})<InnerCircleProps>(({ node, scale }) => {
    const innerCircleSize = `min(max(${calculateNodeSize(NodeWidth.End, scale)}px, 2.5vw), 75px)`;

    return {
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
    };
});

export function EndNode({
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
}: EndNodeProps) {

    const statusColor = useMemo<string | null>(() => {
        if (!isLinked) return null;
        if (linksIn.length === 0) return "red";
        return null;
    }, [isLinked, linksIn.length]);

    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.End, scale) / 2}px, 1.5em)`, [scale]);

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
    const openContext = useCallback(({ target }: UsePressEvent) => {
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

    const availableActions = useMemo(() => {
        return isEditing ?
            [BuildAction.AddListBeforeNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.DeleteNode] :
            [];
    }, [isEditing]);
    const handleContextSelect = useCallback((action: BuildAction) => { handleAction(action, node.id); }, [handleAction, node.id]);

    return (
        <>
            {/* Right-click context menu */}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={availableActions}
                handleClose={closeContext}
                handleSelect={handleContextSelect}
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
                    <OuterCircle
                        id={`${isLinked ? "" : "unlinked-"}node-${node.id}`}
                        aria-owns={contextOpen ? contextId : undefined}
                        node={node}
                        scale={scale}
                        statusColor={statusColor}
                        {...pressEvents}
                    >
                        <InnerCircle
                            id={`${isLinked ? "" : "unlinked-"}node-end-inner-circle-${node.id}`}
                            node={node}
                            scale={scale}
                        >
                        </InnerCircle>
                        {labelVisible && <NodeTitle
                            fontSize={fontSize}
                            scale={scale}
                            variant="h6"
                        >{label}</NodeTitle>}
                    </OuterCircle>
                </Tooltip>
            </DraggableNode>
        </>
    );
}
