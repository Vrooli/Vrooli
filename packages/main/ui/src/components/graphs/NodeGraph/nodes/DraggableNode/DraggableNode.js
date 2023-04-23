import { jsx as _jsx } from "react/jsx-runtime";
import { Box } from "@mui/material";
import { useCallback, useRef, useState } from "react";
import Draggable from "react-draggable";
import { PubSub } from "../../../../../utils/pubsub";
export const DraggableNode = ({ canDrag = true, dragThreshold = 10, nodeId, onClick, children, ...props }) => {
    const [isDragging, setIsDragging] = useState(false);
    const dragRefs = useRef({
        graphStartScroll: null,
        nodeStartPosition: null,
    });
    const dragDistanceRef = useRef({ x: 0, y: 0 });
    const isDragPublished = useRef(false);
    const handleMouseDown = useCallback((ev) => {
        const x = ev?.clientX ?? ev?.touches[0].clientX ?? 0;
        const y = ev?.clientY ?? ev?.touches[0].clientY ?? 0;
        dragRefs.current.nodeStartPosition = { x, y };
        const graph = document.getElementById("graph-root");
        if (graph) {
            const { scrollLeft, scrollTop } = graph;
            dragRefs.current.graphStartScroll = { scrollLeft, scrollTop };
        }
    }, []);
    const handleDragStart = useCallback(() => {
        if (canDrag) {
            setIsDragging(true);
            dragDistanceRef.current = { x: 0, y: 0 };
            return;
        }
        return false;
    }, [canDrag]);
    const handleDrag = useCallback((_e, data) => {
        const { x, y } = data;
        if (!isDragPublished.current && (Math.abs(x) > dragThreshold || Math.abs(y) > dragThreshold)) {
            isDragPublished.current = true;
            PubSub.get().publishNodeDrag({ nodeId });
        }
    }, [dragThreshold, nodeId]);
    const handleDrop = useCallback((ev, data) => {
        setIsDragging(false);
        const { x, y } = data;
        const grid = document.getElementById("graph-root");
        const graphScrollDistance = {
            x: grid.scrollLeft - dragRefs.current.graphStartScroll?.scrollLeft ?? 0,
            y: grid.scrollTop - dragRefs.current.graphStartScroll?.scrollTop ?? 0,
        };
        const { x: startX, y: startY } = dragRefs.current.nodeStartPosition;
        const dropX = startX + x - graphScrollDistance.x;
        const dropY = startY + y - graphScrollDistance.y;
        dragRefs.current = {
            graphStartScroll: null,
            nodeStartPosition: null,
        };
        if (isDragPublished.current) {
            PubSub.get().publishNodeDrop({ nodeId, position: { x: dropX, y: dropY } });
            isDragPublished.current = false;
        }
        else if (onClick) {
            onClick(ev);
        }
    }, [nodeId, onClick]);
    return (_jsx(Draggable, { handle: ".handle", defaultPosition: { x: 0, y: 0 }, disabled: !canDrag, position: { x: 0, y: 0 }, scale: 1, onMouseDown: handleMouseDown, onStart: handleDragStart, onDrag: handleDrag, onStop: handleDrop, children: _jsx(Box, { id: `node-draggable-container-${nodeId}`, onTouchStart: handleMouseDown, display: "flex", justifyContent: "center", alignItems: "center", sx: {
                zIndex: isDragging ? 10000 : 100000,
                opacity: isDragging ? 0.5 : 1,
                cursor: canDrag ? "grab" : "pointer",
                "&:active": {
                    cursor: canDrag ? "grabbing" : "pointer",
                },
            }, ...props, children: children }) }));
};
//# sourceMappingURL=DraggableNode.js.map