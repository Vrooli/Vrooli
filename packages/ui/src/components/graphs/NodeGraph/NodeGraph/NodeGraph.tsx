/**
 * Contains the nodes and links (edges) that describe a routine orchestration.
 * Nodes are displayed within columns. Dragging an unlinked node into a column
 * will add it to the orchestration. Links are generated automatically if possible.
 * Otherwise, a popup is displayed to allow the user to manually specify which node the link should connect to.
 */
import { Box, Stack } from '@mui/material';
import { NodeColumn, NodeEdge } from 'components';
import { TouchEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Pubs, updateArray } from 'utils';
import { ColumnDimensions, NodeGraphProps } from '../types';
import { Node } from 'types';
import { NodeType } from 'graphql/generated/globalTypes';

type DragRefs = {
    lastPosition: { x: number, y: number };
    speed: number;
    timeout: NodeJS.Timeout | null;
}

export const NodeGraph = ({
    scale = 1,
    isEditing = true,
    labelVisible = true,
    nodeDataMap,
    links,
    handleDialogOpen,
    handleNodeUnlink,
    handleNodeDelete,
    handleNodeUpdate,
    handleLinkCreate,
    handleLinkUpdate,
}: NodeGraphProps) => {
    // Size and position data of each column
    const [columnDimensions, setColumnDimensions] = useState<ColumnDimensions[]>([]);
    // Stores edges
    const [edges, setEdges] = useState<JSX.Element[]>([])
    // Dragging a node
    const [dragId, setDragId] = useState<string | null>(null);
    const handleDragStart = useCallback((nodeId: string) => { setDragId(nodeId) }, []);
    const dragRefs = useRef<DragRefs>({
        lastPosition: { x: 0, y: 0 },
        speed: 0,
        timeout: null,
    })

    /**
     * When a node is being dragged near the edge of the grid, the grid scrolls
     */
    const nodeScroll = () => {
        const gridElement = document.getElementById('graph-root');
        if (!gridElement) return;
        const { x, y } = dragRefs.current.lastPosition;
        if (x === 0 && y === 0) return;
        const calculateSpeed = (useX: boolean) => {
            if (x === 0 && y === 0) {
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
            const temp = (maxSpeed - minSpeed) * percent + minSpeed;
            console.log('i am speed', temp, percent, distToEdge, sideLength);
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
        dragRefs.current = {
            lastPosition: { x: 0, y: 0 },
            speed: 0,
            timeout: null,
        }
    }

    /**
     * Updates dimensions of a column
     */
    const onColumnDimensionsChange = useCallback((columnIndex: number, dimensions: ColumnDimensions) => {
        if (columnDimensions.length === 0) return;
        console.log('onColumnDimensionsChange', columnIndex, dimensions);
        if (columnIndex < 0 || columnIndex >= columnDimensions.length) {
            console.error(`Invalid column index: ${columnIndex}`);
            return;
        }
        setColumnDimensions(updateArray(columnDimensions, columnIndex, dimensions));
    }, [columnDimensions]);

    /**
     * Checks if a point is inside a rectangle
     * @param point - The point to check
     * @param rectange - The rectangle to check
     * @param padding - The padding to add to the rectangle
     * @returns True if position is within the bounds of a rectangle
     */
    const isInsideRectangle = (point: { x: number, y: number }, rectangle: DOMRect, padding: number = 75) => {
        console.log('isInsideRectangle check', point, rectangle);
        const zone = {
            x: rectangle.x - padding,
            y: rectangle.y - padding,
            width: rectangle.width + padding * 2,
            height: rectangle.height + padding * 2,
        }
        return (
            point.x > zone.x &&
            point.x < zone.x + zone.width &&
            point.y > zone.y &&
            point.y < zone.y + zone.height
        );
    }

    /**
     * Makes sure drop is valid, then updates order of nodes
     */
    const handleDragStop = useCallback((nodeId: string, { x, y }: { x: number, y: number }) => {
        setDragId(null);
        // First, find the node being dropped
        const node = nodeDataMap[nodeId];
        if (!node) {
            console.error(`Dropped node ${nodeId} not found`);
            return;
        }
        // Next, check if the node was dropped into the "Delete" button or "Unlinked" container. 
        // Find delete button and unlinked container
        const deleteButton = document.getElementById('delete-node-button');
        const unlinkedContainer = document.getElementById('unlinked-nodes-dialog');
        // Get their bounding rectangles
        const deleteButtonRect = deleteButton ? deleteButton.getBoundingClientRect() : null;
        const unlinkedContainerRect = unlinkedContainer ? unlinkedContainer.getBoundingClientRect() : null;
        // If the drop position is within the delete button, delete the node
        if (deleteButtonRect && isInsideRectangle({ x, y }, deleteButtonRect)) {
            // If this is a routine list node, prompt for confirmation
            if (node.node.type === NodeType.RoutineList) {
                PubSub.publish(Pubs.AlertDialog, {
                    message: `Are you sure you want to delete the routine list "${node.node.title}"?`,
                    buttons: [
                        { text: 'Yes', onClick: () => { handleNodeDelete(nodeId); } },
                        { text: 'Cancel' },
                    ]
                });
            }
            // Otherwise, just delete the node
            else handleNodeDelete(nodeId);
            return;
        }
        // If the drop position is within the unlinked container, unlink the node
        if (unlinkedContainerRect && isInsideRectangle({ x, y }, unlinkedContainerRect)) {
            handleNodeUnlink(nodeId);
            return;
        }

        // If the node wasn't dropped into a button or container, find the position it was dropped in
        // Calculate column using column dimensions and padding
        const columnPadding = scale * 25;
        let graphWidth = 0;
        let columnIndex = 0;
        //TODO bug causes first index to have 0 width. We know this is the start node, so this can be calculated
        graphWidth += (100 * scale) + (2 * columnPadding);
        for (let i = 1; i < columnDimensions.length; i++) {
            graphWidth += columnDimensions[i].width + (2 * columnPadding);
            if (x < graphWidth) {
                columnIndex = i;
                break;
            }
        }
        // If columnIndex is greater than max, set to max
        if (columnIndex > columnDimensions.length - 1) columnIndex = columnDimensions.length - 1;
        // If columnIndex is start node or earlier, return
        if (columnIndex < 1 || columnIndex >= columnData.length) {
            PubSub.publish(Pubs.Snack, { message: 'Cannot drop node here', severity: 'error' })
            return;
        }
        console.log('DROPPED COLUMN INDEX', columnIndex, columnDimensions);
        // Determine if node can be dropped without conflicts
        //TODO
        // Otherwise, determine if node can be dropped with conflicts
        //TODO
        // If the node was dropped, center it in the column
        //TODO
        // If the node was not dropped, but it back to its original position
        //TODO
    }, [scale, columnDimensions]);

    // Set listeners for:
    // - click-and-drag background
    // - drag-and-drop nodes
    useEffect(() => {
        // Mouse drag state
        let touchedGrid = false;
        // Only drag if not pressing a node or edge
        const onMouseDown = (ev: MouseEvent | TouchEvent<any> | Event) => {
            // Find first ID in event path
            const targetId = (ev as any)?.target.id;
            if (!targetId) return;
            if (targetId.startsWith('node-') || targetId === 'graph-root' || targetId === 'graph-grid') {
                const x = (ev as MouseEvent)?.clientX ?? (ev as TouchEvent)?.touches[0].clientX ?? 0;
                const y = (ev as MouseEvent)?.clientY ?? (ev as TouchEvent)?.touches[0].clientY ?? 0;
                dragRefs.current.lastPosition = { x, y };
                if (targetId.startsWith('node-column') || targetId === 'graph-root' || targetId === 'graph-grid') touchedGrid = true;
            }
        }
        const onMouseUp = (ev: MouseEvent | Event) => {
            touchedGrid = false;
            dragRefs.current.lastPosition = { x: 0, y: 0 };
            clearScroll();
        }
        const onMouseMove = (ev: MouseEvent) => {
            // If the grid is being dragged, move the grid
            // NOTE: this is only needed for mouse movement, as touch screens do this automatically
            if (touchedGrid) {
                const gridElement = document.getElementById('graph-root');
                if (!gridElement) return;
                const deltaX = ev.clientX - dragRefs.current.lastPosition.x;
                const deltaY = ev.clientY - dragRefs.current.lastPosition.y;
                gridElement.scrollBy(-deltaX, -deltaY);
            }
            dragRefs.current.lastPosition = { x: ev.clientX, y: ev.clientY };
        }
        const onTouchMove = (ev: any) => {
            dragRefs.current.lastPosition = { x: ev.touches[0].clientX ?? 0, y: ev.touches[0].clientY ?? 0 };
        }
        // Add event listeners
        window.addEventListener('mousedown', onMouseDown);
        window.addEventListener('mouseup', onMouseUp);
        window.addEventListener('mousemove', onMouseMove);
        window.addEventListener('touchstart', onMouseDown);
        window.addEventListener('touchmove', onTouchMove);
        window.addEventListener('touchend', onMouseUp);
        // Add PubSub subscribers
        let dragStartSub = PubSub.subscribe(Pubs.NodeDrag, (_, data) => {
            dragRefs.current.timeout = setTimeout(nodeScroll, 50);
            handleDragStart(data.nodeId);
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
    }, [handleDragStart, handleDragStop]);

    /**
     * Node data map converted to a 2D array of columns
     */
    const columnData = useMemo(() => {
        // 2D node data array, ordered by column. 
        // Each column is ordered in a consistent way, so that the nodes in a column are always in the same order
        let list: Node[][] = [];
        // Iterate through node data map
        for (const value of Object.values(nodeDataMap)) {
            // Skips nodes that did not receive a column
            if (value.column < 0) continue;
            // Add new column(s) if necessary
            while (list.length <= value.column) {
                list.push([]);
            }
            // Add node to column
            list[value.column].push(value.node);
        }
        // Sort each column
        // TODO
        return list;
    }, [nodeDataMap]);

    useEffect(() => {
        setColumnDimensions(Array(columnData.length).fill({ width: 0, heights: [], nodeIds: [], tops: [], centers: [] }));
    }, [columnData]);

    /**
     * Edges (links) displayed between nodes
     */
    const calculateEdges = useCallback(() => {
        console.log('in edges start before cancel check', scale)
        // If data required to render edges is not yet available
        // (i.e. no links, or column dimensions not complete)
        if (!links) return [];
        console.log('calculating edges');
        return links?.map(link => {
            if (!link.fromId || !link.toId) return null;
            const fromNode = nodeDataMap[link.fromId];
            const toNode = nodeDataMap[link.toId];
            if (!fromNode || !toNode) return null;
            return <NodeEdge
                key={`edge-${link.id ?? 'new-'+fromNode.node.id+'-to-'+toNode.node.id}`}
                link={link}
                isEditing={isEditing}
                isFromRoutineList={fromNode.node.type === NodeType.RoutineList}
                isToRoutineList={toNode.node.type === NodeType.RoutineList}
                dragId={dragId}
                scale={scale ?? 1}
                handleAdd={() => { }}
                handleEdit={() => { }}
            />
        }).filter(edge => edge) as JSX.Element[];
    }, [dragId, nodeDataMap, isEditing, links, scale]);

    useEffect(() => {
        setEdges(calculateEdges());
    }, [columnDimensions, dragId, nodeDataMap, isEditing, links, scale]);

    /**
     * Node column objects
     */
    const columns = useMemo(() => {
        return columnData.map((columnData, index) => <NodeColumn
            key={`node-column-${index}`}
            id={`node-column-${index}`}
            columnIndex={index}
            nodes={columnData}
            isEditing={isEditing}
            dragId={dragId}
            scale={scale}
            labelVisible={labelVisible}
            onDimensionsChange={onColumnDimensionsChange}
            handleDialogOpen={handleDialogOpen}
        />)
    }, [columnData, isEditing, scale, labelVisible, onColumnDimensionsChange, dragId]);

    return (
        <Box id="graph-root" position="relative" sx={{
            cursor: 'move',
            minWidth: '100%',
            height: { xs: '70vh', md: '80vh' },
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
                height: '-webkit-fill-available',
                // Create grid background pattern
                backgroundColor: `#a8b6c3`,
                '--line-color': `rgba(0 0 0 / .05)`,
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
                {columns}
            </Stack>
        </Box>
    )
};