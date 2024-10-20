/**
 * Contains the nodes and links (edges) that describe a routine.
 * Nodes are displayed within columns. Dragging an unlinked node into a column
 * will add it to the routine. Links are generated automatically if possible.
 * Otherwise, a popup is displayed to allow the user to manually specify which node the link should connect to.
 */
import { Node, NodeType } from "@local/shared";
import { Box, BoxProps, styled, useTheme } from "@mui/material";
import { usePinchZoom } from "hooks/gestures";
import { TouchEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { firstString } from "utils/display/stringTools";
import { PubSub } from "utils/pubsub";
import { NodeEdge } from "../edges/NodeEdge/NodeEdge";
import { NodeColumn } from "../NodeColumn/NodeColumn";
import { NodeGraphProps } from "../types";

type DragRefs = {
    currPosition: { x: number, y: number } | null; // Current position of the cursor
    speed: number; // Current graph scroll speed, based on proximity of currPosition to the edge of the graph
    timeout: NodeJS.Timeout | null; // Timeout for scrolling the graph
}

/**
 * Checks if a point is inside a rectangle
 * @param point - The point to check
 * @param id - The ID of the rectangle to check
 * @param padding - The padding to add to the rectangle
 * @returns True if position is within the bounds of a rectangle
 */
function isInsideRectangle(point: { x: number, y: number }, id: string, padding = 25) {
    const rectangle = document.getElementById(id)?.getBoundingClientRect();
    if (!rectangle) return false;
    const zone = {
        xStart: rectangle.x - padding,
        yStart: rectangle.y - padding,
        xEnd: rectangle.x + rectangle.width + padding * 2,
        yEnd: rectangle.y + rectangle.height + padding * 2,
    };
    return Boolean(
        point.x >= zone.xStart &&
        point.x <= zone.xEnd &&
        point.y >= zone.yStart &&
        point.y <= zone.yEnd,
    );
}

// Positive modulo function
function mod(n: number, m: number) { return ((n % m) + m) % m; }

const GraphBox = styled(Box)(() => ({
    cursor: "move",
    // Disable zooming, selection, and text highlighting
    touchAction: "none",
    msTouchAction: "none",
    userSelect: "none",
    WebkitUserSelect: "none",
    MozUserSelect: "none",
    msUserSelect: "none",
    WebkitTouchCallout: "none",
    KhtmlUserSelect: "none",
    minWidth: "100%",
    // Graph fills remaining space that is not taken up by other elements (i.e. navbar). 
    height: "calc(100vh - 48px)",
    margin: 0,
    padding: 0,
    overflowX: "auto",
    overflowY: "auto",
    position: "relative",
    // Hide scrollbars
    "&::-webkit-scrollbar": {
        display: "none",
    },
}));

interface ColumnBoxProps extends BoxProps {
    scale: number;
}

const ColumnBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "scale",
})<ColumnBoxProps>(({ theme, scale }) => ({
    display: "flex",
    flexDirection: "row",
    gap: 0,
    zIndex: 5,
    width: "fit-content",
    minWidth: "100vw",
    minHeight: "100%",
    paddingLeft: "env(safe-area-inset-left)",
    paddingRight: "env(safe-area-inset-right)",
    // Create grid background pattern on stack, so it scrolls with content
    "--line-color": theme.palette.mode === "light" ? "rgba(0 0 0 / .05)" : "rgba(255 255 255 / .05)",
    "--line-thickness": "1px",
    // Minor length is 1/5 of major length, and is always between 25 and 50 pixels
    "--minor-length": `${mod(scale * 12.5, 25) + 25}px`,
    "--major-length": `${mod(scale * 62.5, 125) + 125}px`,
    "--line": "var(--line-color) 0 var(--line-thickness)",
    "--small-body": "transparent var(--line-thickness) var(--minor-length)",
    "--large-body": "transparent var(--line-thickness) var(--major-length)",

    "--small-squares": "repeating-linear-gradient(to bottom, var(--line), var(--small-body)), repeating-linear-gradient(to right, var(--line), var(--small-body))",

    "--large-squares": "repeating-linear-gradient(to bottom, var(--line), var(--large-body)), repeating-linear-gradient(to right, var(--line), var(--large-body))",
    background: "var(--small-squares), var(--large-squares)",
}));

export function NodeGraph({
    columns,
    handleAction,
    handleBranchInsert,
    handleNodeDrop,
    handleNodeUpdate,
    handleNodeInsert,
    handleLinkCreate,
    handleLinkUpdate,
    handleLinkDelete,
    handleScaleChange,
    isEditing = true,
    labelVisible = true,
    language,
    links,
    nodesById,
    scale,
}: NodeGraphProps) {
    const { palette } = useTheme();

    // Stores edges
    const [edges, setEdges] = useState<JSX.Element[]>([]);
    // Dragging a node
    const [dragId, setDragId] = useState<string | null>(null);
    const dragRefs = useRef<DragRefs>({
        currPosition: null,
        speed: 0,
        timeout: null,
    });
    // Determines if links should be re-rendered quickly or slowly
    const [fastUpdate, setFastUpdate] = useState(false);
    const fastUpdateTimeout = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        setFastUpdate(Boolean(dragId));
    }, [dragId]);

    usePinchZoom({
        onScaleChange: handleScaleChange,
        validTargetIds: ["node-", "graph-", "subroutine"],
    });

    // Add context menu event listener
    useEffect(() => {
        // Disable context menu for whole page
        const handleContextMenu = (ev: any) => { ev.preventDefault(); };
        document.addEventListener("contextmenu", handleContextMenu);
        return () => {
            document.removeEventListener("contextmenu", handleContextMenu);
        };
    }, []);

    /**
     * When a node is being dragged near the edge of the grid, the grid scrolls
     */
    const nodeScroll = useCallback(function nodeScrollCallback() {
        const gridElement = document.getElementById("graph-root");
        if (!gridElement) return;
        if (dragRefs.current.currPosition === null) return;
        const { x, y } = dragRefs.current.currPosition;
        function calculateSpeed(useX: boolean) {
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
        }
        function scrollLeft() { gridElement?.scrollBy(-dragRefs.current.speed, 0); }
        function scrollRight() { gridElement?.scrollBy(dragRefs.current.speed, 0); }
        function scrollUp() { gridElement?.scrollBy(0, -dragRefs.current.speed); }
        function scrollDown() { gridElement?.scrollBy(0, dragRefs.current.speed); }

        // If near the left edge, move the grid left. If near the right edge, move the grid right.
        let horizontalMove: boolean | null = null; // Store left right move, or no horizontal move
        if (x < (window.innerWidth * 0.15)) { calculateSpeed(true); horizontalMove = false; }
        if (x > window.innerWidth - (window.innerWidth * 0.15)) { calculateSpeed(true); horizontalMove = true; }
        // Set or clear timeout for horizontal move
        if (horizontalMove === false) scrollLeft();
        else if (horizontalMove === true) scrollRight();

        // If near the top edge, move the grid up. If near the bottom edge, move the grid down.
        let verticalMove: boolean | null = null; // Store up down move, or no vertical move
        if (y < (window.innerHeight * 0.15)) { calculateSpeed(false); verticalMove = false; }
        if (y > window.innerHeight - (window.innerHeight * 0.15)) { calculateSpeed(false); verticalMove = true; }
        // Set or clear timeout for vertical move
        if (verticalMove === false) scrollUp();
        else if (verticalMove === true) scrollDown();
        dragRefs.current.timeout = setTimeout(nodeScroll, 50);
    }, []);

    function clearScroll() {
        if (dragRefs.current.timeout) clearTimeout(dragRefs.current.timeout);
        dragRefs.current = { currPosition: null, speed: 0, timeout: null };
    }

    /**
     * Makes sure drop is valid, then updates order of nodes
     * @param nodeId - The id of the node to drop
     * @param {x, y} - Absolute drop position of the node
     */
    const handleDragStop = useCallback((nodeId: string, { x, y }: { x: number, y: number }) => {
        setDragId(null);
        // First, find the node being dropped
        const node: Node = nodesById[nodeId];
        if (!node) {
            PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error" });
            return;
        }
        // Next, check if the node was dropped into "Unlinked" container. 
        // If the drop position is within the unlinked container, unlink the node
        if (isInsideRectangle({ x, y }, "unlinked-nodes-dialog")) {
            handleNodeDrop(nodeId, null, null);
            return;
        }
        // If the node wasn't dropped into a button or container, find the column and row it was dropped into
        let columnIndex = -1;
        let rowIndex = -1;
        // Loop through each column to determine the closest one
        let closestDistance = Infinity;
        let closestColumnIndex = -1;
        for (let i = 0; i < columns.length; i++) {
            const rect = document.getElementById(`node-column-${i}`)?.getBoundingClientRect();
            if (!rect) continue;

            const columnCenterX = rect.x + rect.width / 2;
            const distanceToDropPoint = Math.abs(x - columnCenterX);

            if (distanceToDropPoint < closestDistance) {
                closestDistance = distanceToDropPoint;
                closestColumnIndex = i;
            }
        }
        // If we found a closest column, use that.
        if (closestColumnIndex !== -1) {
            columnIndex = closestColumnIndex;
        } else {
            // No closest column found (or some error in the logic).
            PubSub.get().publish("snack", { messageKey: "CannotDropNodeHere", severity: "Error" });
            return;
        }
        // Get the drop row
        const rowRects = columns[columnIndex].map(node => document.getElementById(`node-${node.id}`)?.getBoundingClientRect());
        if (rowRects.some(rect => !rect)) return -1;
        // Calculate the absolute center Y of each node
        const centerYs = rowRects.map((rect: any) => rect.y);
        // Check if the drop is above each node
        for (let i = 0; i < centerYs.length; i++) {
            if (y < centerYs[i]) {
                rowIndex = i;
                break;
            }
        }
        // If not above any nodes, must be below   
        if (rowIndex === -1) rowIndex = centerYs.length;
        // Complete drop
        handleNodeDrop(nodeId, columnIndex, rowIndex);
    }, [columns, handleNodeDrop, nodesById]);

    // Set listeners for:
    // - click-and-drag grid
    // - drag-and-drop nodes
    useEffect(function dragAndDropListenerEffect() {
        // True if user touched the grid instead of a node or edge
        let touchedGrid = false;
        function handleStart(x: number, y: number, targetId: string | null) {
            // Find target
            if (!targetId) return;
            // If touching the grid or a node
            else if (targetId.startsWith("node-") || targetId === "graph-root" || targetId === "graph-grid") {
                // Determine if grid was touched
                if (targetId.startsWith("node-column") || targetId.startsWith("node-placeholder") || targetId === "graph-root" || targetId === "graph-grid") touchedGrid = true;
                // Otherwise, set dragRef
                else dragRefs.current.currPosition = { x, y };
            }
        }
        function handleMove(x: number, y: number) {
            // I the grid is being dragged, move the grid
            if (touchedGrid && dragRefs.current.currPosition) {
                const gridElement = document.getElementById("graph-root");
                if (!gridElement) return;
                const deltaX = x - dragRefs.current.currPosition.x;
                const deltaY = y - dragRefs.current.currPosition.y;
                gridElement.scrollBy(-deltaX, -deltaY);
            }
            // drag
            dragRefs.current.currPosition = { x, y };
        }
        function handleEnd() {
            touchedGrid = false;
            clearScroll();
        }
        function onMouseDown(ev: MouseEvent) { return handleStart(ev.clientX, ev.clientY, (ev as any)?.target.id); }
        function onTouchStart(ev: TouchEvent<any> | any) { return handleStart(ev.touches[0].clientX, ev.touches[0].clientY, (ev as any)?.target.id); }
        function onMouseUp() { return handleEnd(); }
        function onTouchEnd() { return handleEnd(); }
        function onMouseMove(ev: MouseEvent) { return handleMove(ev.clientX, ev.clientY); }
        function onTouchMove(ev: any) { return handleMove(ev.touches[0].clientX, ev.touches[0].clientY); }
        // Add event listeners
        window.addEventListener("mousedown", onMouseDown); // Detects if node or graph is being dragged
        window.addEventListener("mouseup", onMouseUp); // Stops dragging and pinching
        window.addEventListener("mousemove", onMouseMove); // Moves node or grid
        window.addEventListener("touchstart", onTouchStart); // Detects if node is being dragged
        window.addEventListener("touchmove", onTouchMove); // Detects if node is being dragged
        window.addEventListener("touchend", onTouchEnd); // Stops dragging and pinching
        // window.addEventListener('wheel', onMouseWheel); // Detects mouse scroll for zoom
        // Add PubSub subscribers
        const dragStartSub = PubSub.get().subscribe("nodeDrag", function nodeDragSub(data) {
            dragRefs.current.timeout = setTimeout(nodeScroll, 50);
            setDragId(data.nodeId);
        });
        const dragDropSub = PubSub.get().subscribe("nodeDrop", function nodeDropSub(data) {
            clearScroll();
            handleDragStop(data.nodeId, data.position);
        });
        const fastUpdateSub = PubSub.get().subscribe("fastUpdate", function fastUpdateSub(data) {
            setFastUpdate(data.on ?? false);
            fastUpdateTimeout.current = setTimeout(() => { setFastUpdate(false); }, data.duration);
        });
        return () => {
            // Remove event listeners
            window.removeEventListener("mousedown", onMouseDown);
            window.removeEventListener("mouseup", onMouseUp);
            window.removeEventListener("mousemove", onMouseMove);
            window.removeEventListener("touchstart", onTouchStart);
            window.removeEventListener("touchmove", onTouchMove);
            window.removeEventListener("touchend", onTouchEnd);
            // Cleanup pubsub subscribers
            dragStartSub();
            dragDropSub();
            fastUpdateSub();
        };
    }, [handleDragStop, nodeScroll]);

    /**
     * Edges (links) displayed between nodes
     */
    const calculateEdges = useCallback(() => {
        // If data required to render edges is not yet available
        // (i.e. no links, or column dimensions not complete)
        if (!links) return [];
        return links?.map(link => {
            if (!link.from.id || !link.to.id) return null;
            const fromNode = nodesById[link.from.id];
            const toNode = nodesById[link.to.id];
            if (!fromNode || !toNode) return null;
            return <NodeEdge
                key={`edge-${firstString(link.id, "new-") + fromNode.id + "-to-" + toNode.id}`}
                fastUpdate={fastUpdate}
                link={link}
                isEditing={isEditing}
                isFromRoutineList={fromNode.nodeType === NodeType.RoutineList}
                isToRoutineList={toNode.nodeType === NodeType.RoutineList}
                scale={scale}
                handleAdd={handleNodeInsert}
                handleBranch={handleBranchInsert}
                handleDelete={handleLinkDelete}
                handleEdit={() => { }}
            />;
        }).filter(edge => edge) as JSX.Element[];
    }, [fastUpdate, handleBranchInsert, handleLinkDelete, handleNodeInsert, isEditing, links, nodesById, scale]);

    useEffect(() => {
        setEdges(calculateEdges());
    }, [calculateEdges, dragId, isEditing, links, nodesById, scale]);

    /**
     * Node column objects
     */
    const nodeColumns = useMemo(() => {
        return columns.map((col, index) => <NodeColumn
            key={`node-column-${index}`}
            id={`node-column-${index}`}
            columnIndex={index}
            columnsLength={columns.length}
            handleAction={handleAction}
            handleNodeUpdate={handleNodeUpdate}
            isEditing={isEditing}
            labelVisible={labelVisible}
            language={language}
            links={links}
            nodes={col}
            scale={scale}
        />);
    }, [columns, handleAction, handleNodeUpdate, isEditing, labelVisible, language, links, scale]);

    return (
        <GraphBox id="graph-root">
            {/* Edges */}
            {edges}
            <ColumnBox id="graph-grid" scale={scale}>
                {/* Nodes */}
                {nodeColumns}
            </ColumnBox>
        </GraphBox>
    );
}
