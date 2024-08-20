import { IconButton, Tooltip, Typography } from "@mui/material";
import usePress from "hooks/usePress";
import { RedirectIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { noSelect } from "styles";
import { DraggableNode, calculateNodeSize } from "../";
import { NodeWidth } from "../..";
import { nodeLabel } from "../styles";
import { RedirectNodeProps } from "../types";

export function RedirectNode({
    canDrag,
    isLinked = true,
    node,
    scale = 1,
    label = "Redirect",
    labelVisible = true,
}: RedirectNodeProps) {
    const labelObject = useMemo(() => labelVisible && scale >= 0.5 ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...nodeLabel,
                pointerEvents: "none",
            }}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, label, scale]);

    const nodeSize = useMemo(() => `${calculateNodeSize(NodeWidth.Redirect, scale)}px`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.Redirect, scale) / 5}px, 2em)`, [scale]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: EventTarget) => {
        // Ignore if not linked or editing
        if (!canDrag || !isLinked) return;
        setContextAnchor(target);
    }, [canDrag, isLinked]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onRightClick: openContext,
    });

    return (
        <DraggableNode className="handle" canDrag={canDrag} nodeId={node.id}>
            <Tooltip placement={"top"} title='Redirect'>
                <IconButton
                    id={`${isLinked ? "" : "unlinked-"}node-${node.id}`}
                    className="handle"
                    aria-owns={contextOpen ? contextId : undefined}
                    {...pressEvents}
                    sx={{
                        width: nodeSize,
                        height: nodeSize,
                        fontSize,
                        position: "relative",
                        display: "block",
                        backgroundColor: "#6daf72",
                        color: "white",
                        boxShadow: "0px 0px 12px gray",
                        "&:hover": {
                            backgroundColor: "#6daf72",
                            filter: "brightness(120%)",
                            transition: "filter 0.2s",
                        },
                    }}
                >
                    <RedirectIcon
                        id={`${isLinked ? "" : "unlinked-"}node-redirect-icon-${node.id}`}
                    // sx={{
                    //     width: '100%',
                    //     height: '100%',
                    //     color: '#00000044',
                    //     '&:hover': {
                    //         transform: 'scale(1.2)',
                    //         transition: 'scale .2s ease-in-out',
                    //     }
                    // }}
                    />
                    {labelObject}
                </IconButton>
            </Tooltip>
        </DraggableNode>
    );
}
