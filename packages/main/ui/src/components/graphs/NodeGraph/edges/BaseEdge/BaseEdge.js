import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { Box, Popover, Tooltip } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
const getBezierPoint = (t, sp, cp1, cp2, ep) => {
    return {
        x: Math.pow(1 - t, 3) * sp.x + 3 * t * Math.pow(1 - t, 2) * cp1.x
            + 3 * t * t * (1 - t) * cp2.x + t * t * t * ep.x,
        y: Math.pow(1 - t, 3) * sp.y + 3 * t * Math.pow(1 - t, 2) * cp1.y
            + 3 * t * t * (1 - t) * cp2.y + t * t * t * ep.y,
    };
};
const createBezier = (p1, p2) => {
    const midX = (p1.x + p2.x) / 2;
    const mid1 = {
        x: midX,
        y: p1.y,
    };
    const mid2 = {
        x: midX,
        y: p2.y,
    };
    return [p1, mid1, mid2, p2];
};
const getPoint = (containerId, nodeId) => {
    const graph = document.getElementById(containerId);
    const node = document.getElementById(nodeId);
    if (!graph || !node) {
        console.error("Could not find node to connect to edge", nodeId);
        return null;
    }
    const nodeRect = node.getBoundingClientRect();
    const graphRect = graph.getBoundingClientRect();
    return {
        x: graph.scrollLeft + nodeRect.left + (nodeRect.width / 2) - graphRect.left,
        y: graph.scrollTop + nodeRect.top + (nodeRect.height / 2) - graphRect.top,
        startX: graph.scrollLeft + nodeRect.left - graphRect.left,
        endX: graph.scrollLeft + nodeRect.left + nodeRect.width - graphRect.left,
    };
};
export const BaseEdge = ({ containerId, fromId, isEditing, popoverComponent, popoverT, thiccness, timeBetweenDraws, toId, }) => {
    const padding = 50;
    const [dims, setDims] = useState(null);
    const [anchorEl, setAnchorEl] = useState(null);
    const toggleEdit = useCallback((event) => {
        if (Boolean(anchorEl))
            setAnchorEl(null);
        else
            setAnchorEl(event.currentTarget);
    }, [anchorEl]);
    const closeEdit = () => { setAnchorEl(null); };
    const isEditOpen = Boolean(anchorEl);
    const calculateDims = useCallback(() => {
        const fromPoint = getPoint(containerId, fromId);
        const toPoint = getPoint(containerId, toId);
        if (fromPoint && toPoint) {
            const top = Math.min(fromPoint.y, toPoint.y) - padding;
            const left = Math.min(fromPoint.x, toPoint.x) - padding;
            const width = Math.abs(fromPoint.x - toPoint.x) + (padding * 2);
            const height = Math.abs(fromPoint.y - toPoint.y) + (padding * 2);
            const from = {
                x: toPoint.x > fromPoint.x ? padding : width - padding,
                y: toPoint.y > fromPoint.y ? padding : height - padding,
            };
            const to = {
                x: toPoint.x > fromPoint.x ? width - padding : padding,
                y: toPoint.y > fromPoint.y ? height - padding : padding,
            };
            const fromEnd = (fromPoint?.endX ?? 0) - padding;
            const toStart = (toPoint?.startX ?? 0) + padding;
            const bezier = createBezier(from, to);
            setDims({ top, left, width, height, fromEnd, toStart, bezier });
        }
        else
            setDims(null);
    }, [containerId, fromId, toId]);
    const intervalRef = useRef(null);
    useEffect(() => {
        intervalRef.current = setInterval(() => { calculateDims(); }, timeBetweenDraws);
        calculateDims();
        return () => {
            if (intervalRef.current)
                clearInterval(intervalRef.current);
        };
    }, [calculateDims, timeBetweenDraws]);
    const edge = useMemo(() => {
        if (!dims)
            return null;
        return (_jsx("svg", { width: dims.width, height: dims.height, style: {
                zIndex: 0,
                position: "absolute",
                pointerEvents: "none",
                top: dims.top,
                left: dims.left,
            }, children: _jsx("path", { d: `
                        M${dims.bezier[0].x},${dims.bezier[0].y} 
                        C${dims.bezier[1].x},${dims.bezier[1].y}
                        ${dims.bezier[2].x},${dims.bezier[2].y}
                        ${dims.bezier[3].x},${dims.bezier[3].y}
                        `, fill: "none", stroke: "#9e3984", strokeWidth: thiccness / 10 }) }));
    }, [dims, thiccness]);
    const popoverButton = useMemo(() => {
        if (!isEditing || !dims)
            return null;
        const bezierPoint = getBezierPoint(popoverT ?? 0.5, ...dims.bezier);
        const diameter = isEditOpen ? thiccness : thiccness * 5 / 6;
        return (_jsxs(_Fragment, { children: [_jsx(Tooltip, { title: isEditOpen ? "" : "Edit edge or insert node", children: _jsx(Box, { onClick: toggleEdit, sx: {
                            position: "absolute",
                            top: dims.top + bezierPoint.y - (diameter / 2),
                            left: dims.left + bezierPoint.x - (diameter / 2),
                            height: `${diameter}px`,
                            width: `${diameter}px`,
                            borderRadius: "100%",
                            border: isEditOpen ? "2px solid #9e3984" : "none",
                            cursor: "pointer",
                            zIndex: 2,
                            background: isEditOpen ? "transparent" : "#9e3984",
                            "&:hover": {
                                height: `${thiccness}px`,
                                width: `${thiccness}px`,
                                top: dims.top + bezierPoint.y - (thiccness / 2),
                                left: dims.left + bezierPoint.x - (thiccness / 2),
                            },
                        } }) }), _jsx(Popover, { open: isEditOpen, anchorEl: anchorEl, onClose: closeEdit, onClick: closeEdit, anchorOrigin: {
                        vertical: "top",
                        horizontal: "center",
                    }, transformOrigin: {
                        vertical: "bottom",
                        horizontal: "center",
                    }, sx: {
                        "& .MuiPopover-paper": {
                            background: "transparent",
                            boxShadow: "none",
                            border: "none",
                            paddingBottom: 1,
                        },
                    }, children: popoverComponent })] }));
    }, [anchorEl, dims, isEditOpen, isEditing, popoverComponent, popoverT, thiccness, toggleEdit]);
    return (_jsxs(_Fragment, { children: [edge, Boolean(popoverComponent) && popoverButton] }));
};
//# sourceMappingURL=BaseEdge.js.map