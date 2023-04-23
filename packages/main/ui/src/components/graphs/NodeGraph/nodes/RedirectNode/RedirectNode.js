import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { RedirectIcon } from "@local/icons";
import { IconButton, Tooltip, Typography } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { calculateNodeSize, DraggableNode } from "..";
import { NodeWidth } from "../..";
import { noSelect } from "../../../../../styles";
import usePress from "../../../../../utils/hooks/usePress";
import { nodeLabel } from "../styles";
export const RedirectNode = ({ canDrag, isLinked = true, node, scale = 1, label = "Redirect", labelVisible = true, }) => {
    const labelObject = useMemo(() => labelVisible && scale >= 0.5 ? (_jsx(Typography, { variant: "h6", sx: {
            ...noSelect,
            ...nodeLabel,
            pointerEvents: "none",
        }, children: label })) : null, [labelVisible, label, scale]);
    const nodeSize = useMemo(() => `${calculateNodeSize(NodeWidth.Redirect, scale)}px`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.Redirect, scale) / 5}px, 2em)`, [scale]);
    const [contextAnchor, setContextAnchor] = useState(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target) => {
        if (!canDrag || !isLinked)
            return;
        setContextAnchor(target);
    }, [canDrag, isLinked]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onRightClick: openContext,
    });
    return (_jsx(DraggableNode, { className: "handle", canDrag: canDrag, nodeId: node.id, children: _jsx(Tooltip, { placement: "top", title: 'Redirect', children: _jsxs(IconButton, { id: `${isLinked ? "" : "unlinked-"}node-${node.id}`, className: "handle", "aria-owns": contextOpen ? contextId : undefined, ...pressEvents, sx: {
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
                }, children: [_jsx(RedirectIcon, { id: `${isLinked ? "" : "unlinked-"}node-redirect-icon-${node.id}` }), labelObject] }) }) }));
};
//# sourceMappingURL=RedirectNode.js.map