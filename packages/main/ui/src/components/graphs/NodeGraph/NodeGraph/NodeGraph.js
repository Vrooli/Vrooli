import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { NodeType } from "@local/consts";
import { Box, Stack, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { firstString } from "../../../../utils/display/stringTools";
import { usePinchZoom } from "../../../../utils/hooks/usePinchZoom";
import { PubSub } from "../../../../utils/pubsub";
import { NodeEdge } from "../edges";
import { NodeColumn } from "../NodeColumn/NodeColumn";
export const NodeGraph = ({ columns, handleAction, handleBranchInsert, handleNodeDrop, handleNodeUpdate, handleNodeInsert, handleLinkCreate, handleLinkUpdate, handleLinkDelete, isEditing = true, labelVisible = true, language, links, nodesById, zIndex, }) => {
    const { palette } = useTheme();
    const [scale, setScale] = useState(0);
    const handleScaleChange = useCallback((delta, point) => {
        PubSub.get().publishFastUpdate({ duration: 1000 });
        setScale(s => Math.min(Math.max(s + delta, -5), 2));
    }, []);
    const [edges, setEdges] = useState([]);
    const [dragId, setDragId] = useState(null);
    const dragRefs = useRef({
        currPosition: null,
        speed: 0,
        timeout: null,
    });
    const [fastUpdate, setFastUpdate] = useState(false);
    const fastUpdateTimeout = useRef(null);
    useEffect(() => {
        setFastUpdate(Boolean(dragId));
    }, [dragId]);
    usePinchZoom({
        onScaleChange: handleScaleChange,
        validTargetIds: ["node-", "graph-", "subroutine"],
    });
    useEffect(() => {
        const handleContextMenu = (ev) => { ev.preventDefault(); };
        document.addEventListener("contextmenu", handleContextMenu);
        return () => {
            document.removeEventListener("contextmenu", handleContextMenu);
        };
    }, []);
    const nodeScroll = useCallback(() => {
        const gridElement = document.getElementById("graph-root");
        if (!gridElement)
            return;
        if (dragRefs.current.currPosition === null)
            return;
        const { x, y } = dragRefs.current.currPosition;
        const calculateSpeed = (useX) => {
            if (dragRefs.current.currPosition === null) {
                dragRefs.current.speed = 0;
                return;
            }
            const sideLength = useX ? window.innerWidth : window.innerHeight;
            const distToEdge = useX ?
                Math.min(Math.abs(sideLength - x), Math.abs(0 - x)) :
                Math.min(Math.abs(sideLength - y), Math.abs(0 - y));
            const maxSpeed = 25;
            const minSpeed = 5;
            const percent = 1 - (distToEdge) / (sideLength * 0.15);
            dragRefs.current.speed = (maxSpeed - minSpeed) * percent + minSpeed;
        };
        const scrollLeft = () => { gridElement.scrollBy(-dragRefs.current.speed, 0); };
        const scrollRight = () => { gridElement.scrollBy(dragRefs.current.speed, 0); };
        const scrollUp = () => { gridElement.scrollBy(0, -dragRefs.current.speed); };
        const scrollDown = () => { gridElement.scrollBy(0, dragRefs.current.speed); };
        let horizontalMove = null;
        if (x < (window.innerWidth * 0.15)) {
            calculateSpeed(true);
            horizontalMove = false;
        }
        if (x > window.innerWidth - (window.innerWidth * 0.15)) {
            calculateSpeed(true);
            horizontalMove = true;
        }
        if (horizontalMove === false)
            scrollLeft();
        else if (horizontalMove === true)
            scrollRight();
        let verticalMove = null;
        if (y < (window.innerHeight * 0.15)) {
            calculateSpeed(false);
            verticalMove = false;
        }
        if (y > window.innerHeight - (window.innerHeight * 0.15)) {
            calculateSpeed(false);
            verticalMove = true;
        }
        if (verticalMove === false)
            scrollUp();
        else if (verticalMove === true)
            scrollDown();
        dragRefs.current.timeout = setTimeout(nodeScroll, 50);
    }, []);
    const clearScroll = () => {
        if (dragRefs.current.timeout)
            clearTimeout(dragRefs.current.timeout);
        dragRefs.current = { currPosition: null, speed: 0, timeout: null };
    };
    const isInsideRectangle = (point, id, padding = 25) => {
        const rectangle = document.getElementById(id)?.getBoundingClientRect();
        if (!rectangle)
            return false;
        const zone = {
            xStart: rectangle.x - padding,
            yStart: rectangle.y - padding,
            xEnd: rectangle.x + rectangle.width + padding * 2,
            yEnd: rectangle.y + rectangle.height + padding * 2,
        };
        return Boolean(point.x >= zone.xStart &&
            point.x <= zone.xEnd &&
            point.y >= zone.yStart &&
            point.y <= zone.yEnd);
    };
    const handleDragStop = useCallback((nodeId, { x, y }) => {
        setDragId(null);
        const node = nodesById[nodeId];
        if (!node) {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error" });
            return;
        }
        if (isInsideRectangle({ x, y }, "unlinked-nodes-dialog")) {
            handleNodeDrop(nodeId, null, null);
            return;
        }
        let columnIndex = -1;
        let rowIndex = -1;
        for (let i = 0; i < columns.length; i++) {
            if (isInsideRectangle({ x, y }, `node-column-${i}`, 0)) {
                columnIndex = i;
                break;
            }
        }
        if (columnIndex < 0 || columnIndex >= columns.length) {
            PubSub.get().publishSnack({ messageKey: "CannotDropNodeHere", severity: "Error" });
            return;
        }
        const rowRects = columns[columnIndex].map(node => document.getElementById(`node-${node.id}`)?.getBoundingClientRect());
        if (rowRects.some(rect => !rect))
            return -1;
        const centerYs = rowRects.map((rect) => rect.y);
        for (let i = 0; i < centerYs.length; i++) {
            if (y < centerYs[i]) {
                rowIndex = i;
                break;
            }
        }
        if (rowIndex === -1)
            rowIndex = centerYs.length;
        handleNodeDrop(nodeId, columnIndex, rowIndex);
    }, [columns, handleNodeDrop, nodesById]);
    useEffect(() => {
        let touchedGrid = false;
        const handleStart = (x, y, targetId) => {
            if (!targetId)
                return;
            else if (targetId.startsWith("node-") || targetId === "graph-root" || targetId === "graph-grid") {
                if (targetId.startsWith("node-column") || targetId.startsWith("node-placeholder") || targetId === "graph-root" || targetId === "graph-grid")
                    touchedGrid = true;
                else
                    dragRefs.current.currPosition = { x, y };
            }
        };
        const handleMove = (x, y) => {
            if (touchedGrid && dragRefs.current.currPosition) {
                const gridElement = document.getElementById("graph-root");
                if (!gridElement)
                    return;
                const deltaX = x - dragRefs.current.currPosition.x;
                const deltaY = y - dragRefs.current.currPosition.y;
                gridElement.scrollBy(-deltaX, -deltaY);
            }
            dragRefs.current.currPosition = { x, y };
        };
        const handleEnd = () => {
            touchedGrid = false;
            clearScroll();
        };
        const onMouseDown = (ev) => handleStart(ev.clientX, ev.clientY, ev?.target.id);
        const onTouchStart = (ev) => handleStart(ev.touches[0].clientX, ev.touches[0].clientY, ev?.target.id);
        const onMouseUp = handleEnd;
        const onTouchEnd = handleEnd;
        const onMouseMove = (ev) => handleMove(ev.clientX, ev.clientY);
        const onTouchMove = (ev) => handleMove(ev.touches[0].clientX, ev.touches[0].clientY);
        window.addEventListener("mousedown", onMouseDown);
        window.addEventListener("mouseup", onMouseUp);
        window.addEventListener("mousemove", onMouseMove);
        window.addEventListener("touchstart", onTouchStart);
        window.addEventListener("touchmove", onTouchMove);
        window.addEventListener("touchend", onTouchEnd);
        const dragStartSub = PubSub.get().subscribeNodeDrag((data) => {
            dragRefs.current.timeout = setTimeout(nodeScroll, 50);
            setDragId(data.nodeId);
        });
        const dragDropSub = PubSub.get().subscribeNodeDrop((data) => {
            clearScroll();
            handleDragStop(data.nodeId, data.position);
        });
        const fastUpdateSub = PubSub.get().subscribeFastUpdate((data) => {
            setFastUpdate(data.on);
            fastUpdateTimeout.current = setTimeout(() => { setFastUpdate(false); }, data.duration);
        });
        return () => {
            window.removeEventListener("mousedown", onMouseDown);
            window.removeEventListener("mouseup", onMouseUp);
            window.removeEventListener("mousemove", onMouseMove);
            window.removeEventListener("touchstart", onTouchStart);
            window.removeEventListener("touchmove", onTouchMove);
            window.removeEventListener("touchend", onTouchEnd);
            PubSub.get().unsubscribe(dragStartSub);
            PubSub.get().unsubscribe(dragDropSub);
            PubSub.get().unsubscribe(fastUpdateSub);
        };
    }, [handleDragStop, nodeScroll]);
    const calculateEdges = useCallback(() => {
        if (!links)
            return [];
        return links?.map(link => {
            if (!link.from.id || !link.to.id)
                return null;
            const fromNode = nodesById[link.from.id];
            const toNode = nodesById[link.to.id];
            if (!fromNode || !toNode)
                return null;
            return _jsx(NodeEdge, { fastUpdate: fastUpdate, link: link, isEditing: isEditing, isFromRoutineList: fromNode.nodeType === NodeType.RoutineList, isToRoutineList: toNode.nodeType === NodeType.RoutineList, scale: scale, handleAdd: handleNodeInsert, handleBranch: handleBranchInsert, handleDelete: handleLinkDelete, handleEdit: () => { } }, `edge-${firstString(link.id, "new-") + fromNode.id + "-to-" + toNode.id}`);
        }).filter(edge => edge);
    }, [fastUpdate, handleBranchInsert, handleLinkDelete, handleNodeInsert, isEditing, links, nodesById, scale]);
    useEffect(() => {
        setEdges(calculateEdges());
    }, [calculateEdges, dragId, isEditing, links, nodesById, scale]);
    const nodeColumns = useMemo(() => {
        return columns.map((col, index) => _jsx(NodeColumn, { id: `node-column-${index}`, columnIndex: index, handleAction: handleAction, handleNodeUpdate: handleNodeUpdate, isEditing: isEditing, labelVisible: labelVisible, language: language, links: links, nodes: col, scale: scale, zIndex: zIndex }, `node-column-${index}`));
    }, [columns, handleAction, handleNodeUpdate, isEditing, labelVisible, language, links, scale, zIndex]);
    const mod = (n, m) => ((n % m) + m) % m;
    return (_jsxs(Box, { id: "graph-root", position: "relative", sx: {
            cursor: "move",
            touchAction: "none",
            msTouchAction: "none",
            userSelect: "none",
            WebkitUserSelect: "none",
            MozUserSelect: "none",
            msUserSelect: "none",
            WebkitTouchCallout: "none",
            KhtmlUserSelect: "none",
            minWidth: "100%",
            height: "calc(100vh - 48px)",
            margin: 0,
            padding: 0,
            overflowX: "auto",
            overflowY: "auto",
            "&::-webkit-scrollbar": {
                display: "none",
            },
        }, children: [edges, _jsx(Stack, { id: "graph-grid", direction: "row", spacing: 0, zIndex: 5, sx: {
                    width: "fit-content",
                    minWidth: "100vw",
                    minHeight: "100%",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    "--line-color": palette.mode === "light" ? "rgba(0 0 0 / .05)" : "rgba(255 255 255 / .05)",
                    "--line-thickness": "1px",
                    "--minor-length": `${mod(scale * 12.5, 25) + 25}px`,
                    "--major-length": `${mod(scale * 62.5, 125) + 125}px`,
                    "--line": "var(--line-color) 0 var(--line-thickness)",
                    "--small-body": "transparent var(--line-thickness) var(--minor-length)",
                    "--large-body": "transparent var(--line-thickness) var(--major-length)",
                    "--small-squares": "repeating-linear-gradient(to bottom, var(--line), var(--small-body)), repeating-linear-gradient(to right, var(--line), var(--small-body))",
                    "--large-squares": "repeating-linear-gradient(to bottom, var(--line), var(--large-body)), repeating-linear-gradient(to right, var(--line), var(--large-body))",
                    background: "var(--small-squares), var(--large-squares)",
                }, children: nodeColumns })] }));
};
//# sourceMappingURL=NodeGraph.js.map