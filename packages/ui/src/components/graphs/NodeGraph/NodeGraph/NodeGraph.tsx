/**
 * Contains the nodes and links (edges) that describe a routine.
 * Nodes are displayed within columns. Dragging an unlinked node into a column
 * will add it to the routine. Links are generated automatically if possible.
 * Otherwise, a popup is displayed to allow the user to manually specify which node the link should connect to.
 */
import { Box, Stack, useTheme } from '@mui/material';
import { NodeColumn, NodeEdge } from 'components';
import { TouchEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Pubs } from 'utils';
import { NodeGraphProps } from '../types';
import { Node } from 'types';
import { NodeType } from 'graphql/generated/globalTypes';

type DragRefs = {
    currPosition: { x: number, y: number } | null; // Current position of the cursor
    speed: number; // Current graph scroll speed, based on proximity of currPosition to the edge of the graph
    timeout: NodeJS.Timeout | null; // Timeout for scrolling the graph
}

type PinchRefs = {
    currPosition: { x: number, y: number } | null; // Most recent pinch position
    lastPosition: { x: number, y: number } | null; // Pinch position when scale was last updated
    isShiftKeyPressed: boolean; // Whether shift key is pressed (for zooming with mouse and keyboard)
    timeout: NodeJS.Timeout | null;
}

export const NodeGraph = ({
    columns,
    handleAction,
    handleNodeDrop,
    handleNodeUpdate,
    handleNodeInsert,
    handleLinkCreate,
    handleLinkUpdate,
    handleLinkDelete,
    isEditing = true,
    labelVisible = true,
    language,
    links,
    nodesById,
    scale = 1,
}: NodeGraphProps) => {
    const { palette } = useTheme();

    // Stores edges
    const [edges, setEdges] = useState<JSX.Element[]>([])
    // Dragging a node
    const [dragId, setDragId] = useState<string | null>(null);
    const dragRefs = useRef<DragRefs>({
        currPosition: null,
        speed: 0,
        timeout: null,
    })
    // Pinching to zoom. Uses a timeout to reduce the amount of calls to handleScaleChange
    const [isShiftKeyPressed, setIsShiftKeyPressed] = useState(false); // Used to update cursor, not for calculations
    const pinchRefs = useRef<PinchRefs>({
        currPosition: null,
        lastPosition: null,
        isShiftKeyPressed: false,
        timeout: null,
    })

    /**
     * Add tag to head indicating that this page has custom scaling
     */
    useEffect(() => {
        const meta = document.createElement("meta");
        meta.name = "viewport";
        meta.content = "width=device-width, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0 user-scalable=0";
        document.head.appendChild(meta);
        // Remove the meta element when the component is unmounted
        return () => { document.head.removeChild(meta); }
    }, [])

    /**
     * When a node is being dragged near the edge of the grid, the grid scrolls
     */
    const nodeScroll = () => {
        const gridElement = document.getElementById('graph-root');
        if (!gridElement) return;
        if (dragRefs.current.currPosition === null) return;
        const { x, y } = dragRefs.current.currPosition;
        const calculateSpeed = (useX: boolean) => {
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
        const scrollLeft = () => { gridElement.scrollBy(-dragRefs.current.speed, 0) }
        const scrollRight = () => { gridElement.scrollBy(dragRefs.current.speed, 0) };
        const scrollUp = () => { gridElement.scrollBy(0, -dragRefs.current.speed) };
        const scrollDown = () => { gridElement.scrollBy(0, dragRefs.current.speed) };

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
    }

    const clearScroll = () => {
        if (dragRefs.current.timeout) clearTimeout(dragRefs.current.timeout);
        dragRefs.current = { currPosition: null, speed: 0, timeout: null }
    }

    /**
     * Rescales the graph when pinching
     */
    const pinch = () => {
        if (pinchRefs.current.currPosition === null) return;
        // Determine the distance between the last time this function was called and now
        const { x, y } = pinchRefs.current.currPosition;
        const { x: lastX, y: lastY } = pinchRefs.current.lastPosition || pinchRefs.current.currPosition;
        const distX = x - lastX;
        const distY = y - lastY;
        const dist = Math.sqrt(distX * distX + distY * distY);
        PubSub.publish(Pubs.Snack, { message: dist })
        // Determine if the pinch is expanding or contracting
        //TODO
        // Set last position to current position
        pinchRefs.current.lastPosition = pinchRefs.current.currPosition;
        // Set a timeout to call this function again
        if (pinchRefs.current.timeout) setTimeout(pinch, 100);
    }

    const clearPinch = () => {
        if (pinchRefs.current.timeout) clearTimeout(pinchRefs.current.timeout);
        pinchRefs.current = { currPosition: null, lastPosition: null, isShiftKeyPressed: false, timeout: null }
    }

    /**
     * Checks if a point is inside a rectangle
     * @param point - The point to check
     * @param id - The ID of the rectangle to check
     * @param padding - The padding to add to the rectangle
     * @returns True if position is within the bounds of a rectangle
     */
    const isInsideRectangle = (point: { x: number, y: number }, id: string, padding: number = 25) => {
        const rectangle = document.getElementById(id)?.getBoundingClientRect();
        if (!rectangle) return false;
        const zone = {
            xStart: rectangle.x - padding,
            yStart: rectangle.y - padding,
            xEnd: rectangle.x + rectangle.width + padding * 2,
            yEnd: rectangle.y + rectangle.height + padding * 2,
        }
        return Boolean(
            point.x >= zone.xStart &&
            point.x <= zone.xEnd &&
            point.y >= zone.yStart &&
            point.y <= zone.yEnd
        );
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
            PubSub.publish(Pubs.Snack, { message: `Dropped node ${nodeId} not found` });
            return;
        }
        // Next, check if the node was dropped into "Unlinked" container. 
        // If the drop position is within the unlinked container, unlink the node
        if (isInsideRectangle({ x, y }, 'unlinked-nodes-dialog')) {
            handleNodeDrop(nodeId, null, null);
            return;
        }

        // If the node wasn't dropped into a button or container, find row and column it was dropped into
        let columnIndex = -1;
        let rowIndex = -1;
        // Check if node dropped into each column
        for (let i = 0; i < columns.length; i++) {
            if (isInsideRectangle({ x, y }, `node-column-${i}`, 0)) {
                columnIndex = i;
                break;
            }
        }
        // If columnIndex is start node or earlier, return
        if (columnIndex < 1 || columnIndex >= columns.length) {
            PubSub.publish(Pubs.Snack, { message: 'Cannot drop node here', severity: 'error' })
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
                rowIndex = i
                break;
            }
        }
        // If not above any nodes, must be below   
        if (rowIndex === -1) rowIndex = centerYs.length;
        // If dropped into its same position, return
        if (node.columnIndex === columnIndex && node.rowIndex === rowIndex) return;
        // If dropped into its own column and no other nodes in that column, return
        const nodesInColumn = Object.values(nodesById).filter(node => node.columnIndex === columnIndex);
        if (nodesInColumn.length === 1 && nodesInColumn[0].id === nodeId) return;
        // Complete drop
        handleNodeDrop(nodeId, columnIndex, rowIndex);
    }, [nodesById, scale]);

    // Set listeners for:
    // - click-and-drag background
    // - drag-and-drop nodes
    useEffect(() => {
        // True if user touched the grid instead of a node or edge
        let touchedGrid = false;
        const onKeyDown = (ev: KeyboardEvent) => {
            // If the user pressed the shift key
            if (ev.key === 'Shift') {
                pinchRefs.current.isShiftKeyPressed = true;
                setIsShiftKeyPressed(true);
            }
        }
        const onKeyUp = (ev: KeyboardEvent) => {
            // If the user released the shift key
            if (ev.key === 'Shift') {
                pinchRefs.current.isShiftKeyPressed = false;
                setIsShiftKeyPressed(false);
            }
        }
        const onMouseDown = (ev: MouseEvent) => {
            // Find touch point
            const x = ev.clientX;
            const y = ev.clientY;
            // Find target
            const targetId = (ev as any)?.target.id;
            if (!targetId) return;
            // If pinching
            if (pinchRefs.current.isShiftKeyPressed) {
                pinchRefs.current.currPosition = { x, y };
                if (!pinchRefs.current.timeout) pinchRefs.current.timeout = setTimeout(pinch, 500);
            }
            // If touching the grid or a node
            else if (targetId.startsWith('node-') || targetId === 'graph-root' || targetId === 'graph-grid') {
                // Determine if grid was touched
                if (targetId.startsWith('node-column') || targetId.startsWith('node-placeholder') || targetId === 'graph-root' || targetId === 'graph-grid') touchedGrid = true;
                // Otherwise, set dragRef
                else dragRefs.current.currPosition = { x, y };
            }
        }
        // Detect node drag and graph pinch
        const onTouchStart = (ev: TouchEvent<any> | any) => {
            // Find touch point
            const x = ev.touches[0].clientX;
            const y = ev.touches[0].clientY;
            // Find the target
            const targetId = ev.target.id;
            if (!targetId) return;
            // If pinching
            if (ev.touches.length > 1) {
                pinchRefs.current.currPosition = { x, y };
                if (!pinchRefs.current.timeout) pinchRefs.current.timeout = setTimeout(pinch, 100);
            }
            // If touching the grid or a node
            else if (targetId.startsWith('node-') || targetId === 'graph-root' || targetId === 'graph-grid') {
                // Set dragRef
                dragRefs.current.currPosition = { x, y };
            }
        }
        const onMouseUp = (ev: MouseEvent | Event) => {
            touchedGrid = false;
            clearScroll();
            clearPinch();
        }
        const onMouseMove = (ev: MouseEvent) => {
            // Find cursor point
            const x = ev.clientX;
            const y = ev.clientY;
            // If pinching
            if (pinchRefs.current.currPosition) {
                pinchRefs.current.currPosition = { x, y };
            }
            // Otherwise, if the grid is being dragged, move the grid
            // NOTE: this is only needed for mouse movement, as touch screens do this automatically
            else if (touchedGrid && dragRefs.current.currPosition) {
                const gridElement = document.getElementById('graph-root');
                if (!gridElement) return;
                const deltaX = x - dragRefs.current.currPosition.x;
                const deltaY = y - dragRefs.current.currPosition.y;
                gridElement.scrollBy(-deltaX, -deltaY);
            }
            dragRefs.current.currPosition = { x, y };
        }
        const onTouchMove = (ev: any) => {
            // Prevent default behavior
            ev.preventDefault();
            // Find touch point
            const x = ev.touches[0].clientX;
            const y = ev.touches[0].clientY;
            // Handle pinch TODO
            if (pinchRefs.current.currPosition) {
                pinchRefs.current.currPosition = { x, y };
            }
            // drag
            dragRefs.current.currPosition = { x, y };
        }
        // Add event listeners
        window.addEventListener('keydown', onKeyDown); // Detects shift key for pinch
        window.addEventListener('keyup', onKeyUp); // Detects shift key for pinch
        window.addEventListener('mousedown', onMouseDown); // Detects if node or graph is being dragged
        window.addEventListener('mouseup', onMouseUp); // Stops dragging and pinching
        window.addEventListener('mousemove', onMouseMove); // Moves node or grid
        window.addEventListener('touchstart', onTouchStart); // Detects if node is being dragged or graph is being pinched
        window.addEventListener('touchmove', onTouchMove); // Detects if node is being dragged or graph is being pinched
        window.addEventListener('touchend', onMouseUp); // Stops dragging and pinching
        // Add PubSub subscribers
        let dragStartSub = PubSub.subscribe(Pubs.NodeDrag, (_, data) => {
            dragRefs.current.timeout = setTimeout(nodeScroll, 50);
            setDragId(data.nodeId)
        });
        let dragDropSub = PubSub.subscribe(Pubs.NodeDrop, (_, data) => {
            clearScroll();
            handleDragStop(data.nodeId, data.position);
        });
        return () => {
            // Remove event listeners
            window.removeEventListener('mousedown', onMouseDown);
            window.removeEventListener('mouseup', onMouseUp);
            window.removeEventListener('mousemove', onMouseMove);
            // Remove PubSub subscribers
            PubSub.unsubscribe(dragStartSub);
            PubSub.unsubscribe(dragDropSub);
        }
    }, [handleDragStop, setIsShiftKeyPressed]);

    /**
     * Edges (links) displayed between nodes
     */
    const calculateEdges = useCallback(() => {
        // If data required to render edges is not yet available
        // (i.e. no links, or column dimensions not complete)
        if (!links) return [];
        return links?.map(link => {
            if (!link.fromId || !link.toId) return null;
            const fromNode = nodesById[link.fromId];
            const toNode = nodesById[link.toId];
            if (!fromNode || !toNode) return null;
            return <NodeEdge
                key={`edge-${link.id ?? 'new-' + fromNode.id + '-to-' + toNode.id}`}
                link={link}
                isEditing={isEditing}
                isFromRoutineList={fromNode.type === NodeType.RoutineList}
                isToRoutineList={toNode.type === NodeType.RoutineList}
                dragId={dragId}
                scale={scale ?? 1}
                handleAdd={handleNodeInsert}
                handleDelete={handleLinkDelete}
                handleEdit={() => { }}
            />
        }).filter(edge => edge) as JSX.Element[];
    }, [dragId, isEditing, links, nodesById, scale]);

    useEffect(() => {
        setEdges(calculateEdges());
    }, [dragId, isEditing, links, nodesById, scale]);

    /**
     * Node column objects
     */
    const nodeColumns = useMemo(() => {
        return columns.map((col, index) => <NodeColumn
            key={`node-column-${index}`}
            id={`node-column-${index}`}
            columnIndex={index}
            dragId={dragId}
            handleAction={handleAction}
            handleNodeDrop={handleNodeDrop}
            handleNodeUpdate={handleNodeUpdate}
            isEditing={isEditing}
            labelVisible={labelVisible}
            language={language}
            nodes={col}
            scale={scale}
        />)
    }, [columns, dragId, isEditing, labelVisible, scale]);

    return (
        <Box id="graph-root" position="relative" sx={{
            cursor: isShiftKeyPressed ? 'nesw-resize' : 'move',
            minWidth: '100%',
            height: { xs: '90vh', md: '80vh' },
            overflowX: 'auto',
            overflowY: 'auto',
            margin: 0,
            padding: 0,
            // Customize scrollbar
            "&::-webkit-scrollbar": {
                width: 10,
            },
            "&::-webkit-scrollbar-track": {
                backgroundColor: '#dae5f0',
            },
            "&::-webkit-scrollbar-thumb": {
                borderRadius: '100px',
                backgroundColor: "#409590",
            },
        }}>
            {/* Edges */}
            {edges}
            <Stack id="graph-grid" direction="row" spacing={0} zIndex={5} sx={{
                width: 'fit-content',
                minHeight: '-webkit-fill-available',
                // Create grid background pattern
                '--line-color': palette.mode === 'light' ? `rgba(0 0 0 / .05)` : `rgba(255 255 255 / .05)`,
                '--line-thickness': `1px`,
                '--minor-length': `${80 * scale}px`,
                '--major-length': `${400 * scale}px`,
                '--line': `var(--line-color) 0 var(--line-thickness)`,
                '--small-body': `transparent var(--line-thickness) var(--minor-length)`,
                '--large-body': `transparent var(--line-thickness) var(--major-length)`,

                '--small-squares': `repeating-linear-gradient(to bottom, var(--line), var(--small-body)), repeating-linear-gradient(to right, var(--line), var(--small-body))`,

                '--large-squares': `repeating-linear-gradient(to bottom, var(--line), var(--large-body)), repeating-linear-gradient(to right, var(--line), var(--large-body))`,
                background: `var(--small-squares), var(--large-squares)`,
            }}>
                {/* Nodes */}
                {nodeColumns}
            </Stack>
        </Box>
    )
};