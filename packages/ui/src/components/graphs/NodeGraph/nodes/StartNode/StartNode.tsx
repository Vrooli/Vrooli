import { Box, BoxProps, Tooltip, Typography, TypographyProps, styled } from "@mui/material";
import { UsePressEvent, usePress } from "hooks/gestures";
import { useCallback, useMemo, useState } from "react";
import { noSelect } from "styles";
import { BuildAction, NodeWidth } from "utils/consts";
import { NodeContextMenu } from "../../NodeContextMenu/NodeContextMenu";
import { nodeLabel } from "../styles";
import { StartNodeProps } from "../types";
import { SHOW_TITLE_ABOVE_SCALE, calculateNodeSize } from "../utils";

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
    nodeSize: string;
    statusColor: string | null;
}

const OuterCircle = styled(Box, {
    shouldForwardProp: (prop) => prop !== "nodeSize" && prop !== "statusColor",
})<OuterCircleProps>(({ nodeSize, statusColor, theme }) => ({
    boxShadow: statusColor ? `0px 0px 12px ${statusColor}` : theme.shadows[12],
    width: nodeSize,
    height: nodeSize,
    position: "relative",
    display: "block",
    backgroundColor: theme.palette.mode === "light" ? "#259a17" : "#387e30",
    color: "white",
    borderRadius: "100%",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
    "@media print": {
        border: `1px solid ${theme.palette.mode === "light" ? "#259a17" : "#387e30"}`,
        boxShadow: "none",
    },
}));

export function StartNode({
    handleAction,
    node,
    scale = 1,
    isEditing,
    label = "Start",
    labelVisible = true,
    linksOut,
}: StartNodeProps) {
    const statusColor = useMemo<string | null>(() => {
        if (linksOut.length === 0) return "red";
        return null;
    }, [linksOut.length]);

    const nodeSize = useMemo(() => `min(max(${calculateNodeSize(NodeWidth.Start, scale) * 2}px, 5vw), 150px)`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.Start, scale) / 2}px, 1.5em)`, [scale]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback(({ target }: UsePressEvent) => {
        if (!isEditing) return;
        setContextAnchor(target);
    }, [isEditing]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onRightClick: openContext,
    });

    const availableActions = useMemo(() => {
        return isEditing ?
            [BuildAction.AddOutgoingLink] :
            [];
    }, [isEditing]);
    const handleContextSelect = useCallback((action: BuildAction) => { handleAction(action as BuildAction.AddOutgoingLink, node.id); }, [handleAction, node.id]);

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
            <Tooltip placement={"top"} title={label ?? ""}>
                <OuterCircle
                    id={`node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    nodeSize={nodeSize}
                    statusColor={statusColor}
                    {...pressEvents}
                >
                    {labelVisible && <NodeTitle
                        fontSize={fontSize}
                        scale={scale}
                        variant="h6"
                    >{label}</NodeTitle>}
                </OuterCircle>
            </Tooltip>
        </>
    );
}
