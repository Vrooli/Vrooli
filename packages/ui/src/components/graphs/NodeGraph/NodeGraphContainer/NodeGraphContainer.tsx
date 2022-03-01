/**
 * Contains the nodes and links (edges) that describe a routine orchestration.
 * Nodes are displayed within columns. Dragging an unlinked node into a column
 * will add it to the orchestration. Links are generated automatically if possible.
 * Otherwise, a popup is displayed to allow the user to manually specify which node the link should connect to.
 */
import { Box, Stack } from '@mui/material';
import { NodeGraphColumn, NodeGraphEdge } from 'components';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { OrchestrationStatus, Pubs, updateArray } from 'utils';
import { ColumnDimensions, NodeGraphProps, NodePos } from '../types';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node } from 'types';
import { NodeWidth } from '..';

type DragRefs = {
    lastPosition: { x: number, y: number };
    speed: number;
    left: NodeJS.Timeout | null;
    right: NodeJS.Timeout | null;
    up: NodeJS.Timeout | null;
    down: NodeJS.Timeout | null;
}

export const NodeGraphContainer = ({
    scale = 1,
    isEditable = true,
    labelVisible = true,
    nodes,
    links,
    onStatusChange,
}: NodeGraphProps) => {
    // Size and position data of each column
    const [columnDimensions, setColumnDimensions] = useState<ColumnDimensions[]>([]);
    // Dragging a node
    const [dragId, setDragId] = useState<string | null>(null);
    const handleDragStart = useCallback((nodeId: string) => { setDragId(nodeId) }, []);
    const dragRefs = useRef<DragRefs>({
        lastPosition: { x: 0, y: 0 },
        speed: 0,
        left: null,
        right: null,
        up: null,
        down: null,
    })

    /**
     * When a node is being dragged near the edge of the grid, the grid scrolls
     */
    const nodeScroll = () => {
        const gridElement = document.getElementById('graph-root');
        if (!gridElement) return;
        const { x, y } = dragRefs.current.lastPosition;
        const calculateSpeed = (useX: boolean) => {
            console.log('calculating speed', x, y)
            if (x === 0 && y === 0) {
                clearScroll();
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
        const scrollLeft = () => { gridElement.scrollBy(-dragRefs.current.speed, 0); dragRefs.current.left = setTimeout(scrollLeft, 50) };
        const scrollRight = () => { gridElement.scrollBy(dragRefs.current.speed, 0); dragRefs.current.right = setTimeout(scrollRight, 50) };
        const scrollUp = () => { gridElement.scrollBy(0, -dragRefs.current.speed); dragRefs.current.up = setTimeout(scrollUp, 50) };
        const scrollDown = () => { gridElement.scrollBy(0, dragRefs.current.speed); dragRefs.current.down = setTimeout(scrollDown, 50) };
        // If near the left edge, move the grid left. If near the right edge, move the grid right.
        let horizontalMove: boolean | null = null; // Store left right move, or no horizontal move
        if (x < (window.innerWidth * 0.15)) { calculateSpeed(true); horizontalMove = false;}
        if (x > window.innerWidth - (window.innerWidth * 0.15)) { calculateSpeed(true); horizontalMove = true;}
        // Set or clear timeout for horizontal move
        if (horizontalMove === false && !dragRefs.current.left) {
            dragRefs.current.left = setTimeout(scrollLeft, 50);
            if (dragRefs.current.right) { clearTimeout(dragRefs.current.right); dragRefs.current.right = null; }
        } else if (horizontalMove === true && !dragRefs.current.right) {
            dragRefs.current.right = setTimeout(scrollRight, 50);
            if (dragRefs.current.left) { clearTimeout(dragRefs.current.left); dragRefs.current.left = null; }
        } else if (horizontalMove === null) {
            if (dragRefs.current.left) { clearTimeout(dragRefs.current.left); dragRefs.current.left = null; }
            if (dragRefs.current.right) { clearTimeout(dragRefs.current.right); dragRefs.current.right = null; }
        }
        // If near the top edge, move the grid up. If near the bottom edge, move the grid down.
        let verticalMove: boolean | null = null; // Store up down move, or no vertical move
        if (y < (window.innerHeight * 0.15)) { calculateSpeed(false); verticalMove = false;}
        if (y > window.innerHeight - (window.innerHeight * 0.15)) { calculateSpeed(false); verticalMove = true;}
        // Set or clear timeout for vertical move
        if (verticalMove === false && !dragRefs.current.up) {
            dragRefs.current.up = setTimeout(scrollUp, 50);
            if (dragRefs.current.down) { clearTimeout(dragRefs.current.down); dragRefs.current.down = null; }
        }
        else if (verticalMove === true && !dragRefs.current.down) {
            dragRefs.current.down = setTimeout(scrollDown, 50);
            if (dragRefs.current.up) { clearTimeout(dragRefs.current.up); dragRefs.current.up = null; }
        }
        else if (verticalMove === null) {
            if (dragRefs.current.up) { clearTimeout(dragRefs.current.up); dragRefs.current.up = null; }
            if (dragRefs.current.down) { clearTimeout(dragRefs.current.down); dragRefs.current.down = null; }
        }
    }

    const clearScroll = () => {
        if (dragRefs.current.left) clearTimeout(dragRefs.current.left)
        if (dragRefs.current.right) clearTimeout(dragRefs.current.right)
        if (dragRefs.current.up) clearTimeout(dragRefs.current.up)
        if (dragRefs.current.down) clearTimeout(dragRefs.current.down)
        dragRefs.current = {
            lastPosition: { x: 0, y: 0 },
            speed: 0,
            left: null,
            right: null,
            up: null,
            down: null,
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

    useEffect(() => {
        console.log('colunn dimensions updated!!🎉', columnDimensions);
    }, [columnDimensions]);

    /**
     * Makes sure drop is valid, then updates order of nodes
     */
    const handleDragStop = useCallback((nodeId: string, { x, y }: { x: number, y: number }) => {
        console.log('DRAG STOPPPPPPPP START', nodeId, { x, y });
        setDragId(null);
        PubSub.publish(Pubs.NodeDrag, { nodeId: undefined });
        console.log('DRAG STOPPPPPPP A')
        // Determine column that node is being dropped into
        const columnPadding = scale * 25;
        let graphWidth = 0;
        let columnIndex = 0;
        for (let i = 0; i < columnDimensions.length; i++) {
            graphWidth += columnDimensions[i].width + (2 * columnPadding);
            if (x < graphWidth) {
                columnIndex = i;
                break;
            }
        }
        // If columnIndex is out of bounds (or start node column), return
        if (columnIndex < 1 || columnIndex >= columnData.length) return;
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
        let touched = false;
        // Only drag if not pressing a node or edge
        const onMouseDown = (ev: MouseEvent) => {
            const targetId = (ev.target as any)?.id;
            if (!targetId) return;
            if (targetId.startsWith('node-column') || targetId === 'graph-root' || targetId === 'graph-grid') {
                touched = true;
                dragRefs.current.lastPosition = { x: ev.clientX, y: ev.clientY };
            }
        }
        const onMouseUp = (ev: MouseEvent) => {
            touched = false;
            dragRefs.current.lastPosition = { x: 0, y: 0 };
            clearScroll();
        }
        const onMouseMove = (ev: MouseEvent) => {
            // If the grid is being dragged, move the grid
            if (touched) {
                const gridElement = document.getElementById('graph-root');
                if (!gridElement) return;
                const deltaX = ev.clientX - dragRefs.current.lastPosition.x;
                const deltaY = ev.clientY - dragRefs.current.lastPosition.y;
                gridElement.scrollBy(-deltaX, -deltaY);
            }
            dragRefs.current.lastPosition = { x: ev.clientX, y: ev.clientY };
        }
        // Add event listeners
        window.addEventListener('mousedown', onMouseDown);
        window.addEventListener('mouseup', onMouseUp);
        window.addEventListener('mousemove', onMouseMove);
        // Add PubSub subscribers
        let dragStartSub = PubSub.subscribe(Pubs.NodeDrag, (_, data) => {
            nodeScroll();
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
     * 1st return - Dictionary of node data and their columns
     * 2nd return - List of nodes which are not yet linked
     * If nodeDataMap is same length as nodes, and unlinkedList is empty, then all nodes are linked 
     * and the graph is valid
     */
    const [nodeDataMap, unlinkedList] = useMemo<[{ [id: string]: NodePos }, Node[]]>(() => {
        // Position map for calculating node positions
        let posMap: { [id: string]: NodePos } = {};
        if (!nodes || !links) return [posMap, nodes ?? []];
        let startNodeId: string | null = null;
        console.log('node data map', nodes, links);
        // First pass of raw node data, to locate start node and populate position map
        for (let i = 0; i < nodes.length; i++) {
            const currId = nodes[i].id;
            // If start node, must be in first column
            if (nodes[i].type === NodeType.Start) {
                startNodeId = currId;
                posMap[currId] = { column: 0, node: nodes[i] }
            }
        }
        // If start node was found
        if (startNodeId) {
            // Loop through links. Each loop finds every node that belongs in the next column
            // We set the max number of columns to be 100, but this is arbitrary
            for (let currColumn = 0; currColumn < 100; currColumn++) {
                // Calculate the IDs of each node in the next column TODO this should be sorted in some way so it shows the same order every time
                const nextNodes = links.filter(link => posMap[link.fromId]?.column === currColumn).map(link => nodes.find(node => node.id === link.toId)).filter(node => node) as Node[];
                // Add each node to the position map
                for (let i = 0; i < nextNodes.length; i++) {
                    const curr = nextNodes[i];
                    posMap[curr.id] = { column: currColumn + 1, node: curr };
                }
                // If not nodes left, or if all of the next nodes are end nodes, stop looping
                if (nextNodes.length === 0 || nextNodes.every(n => n.type === NodeType.End)) {
                    break;
                }
            }
        } else {
            console.error('Error: No start node found');
            onStatusChange({ code: OrchestrationStatus.Invalid, details: 'No start node found' });
        }
        // TODO check if all paths end with an end node (and account for loops)
        const unlinked = nodes.filter(node => !posMap[node.id]);
        if (unlinked.length > 0) {
            console.warn('Warning: Some nodes are not linked');
            onStatusChange({ code: OrchestrationStatus.Incomplete, details: 'Some nodes are not linked' });
        }
        if (startNodeId && unlinked.length === 0) {
            onStatusChange({ code: OrchestrationStatus.Valid, details: '' });
        }
        return [posMap, unlinked];
    }, [nodes, links]);

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
     * Calculates the position of a node in the graph
     */
    const getNodePos = useCallback((id: string): { x: number, y: number } | null => {
        // Get column number of node
        const columnIndex = nodeDataMap[id]?.column;
        if (!columnIndex || columnIndex < 0) return null;
        const columnDims = columnDimensions[columnIndex];
        const rowIndex = columnDims.nodeIds.indexOf(id) ?? 0;
        // Calculate x position by summing the previous column widths (and padding between columns)
        const columnPadding = scale * 25;
        let x = 0;
        for (let i = 0; i < columnIndex; i++) {
            x += (columnDimensions[i].width ?? (scale * NodeWidth[NodeType.Start])) + (2 * columnPadding);
        }
        x += columnPadding + (columnDims.width ?? (scale * NodeWidth[NodeType.Start])) / 2;
        // Calculate y position
        let y = columnDims.centers[rowIndex];
        // Using the largest column, reshape centers to be in the middle vertically
        const largestHeight = Math.max(...columnDimensions.map(d => Math.max(...d.heights)));
        y += (largestHeight - columnDims.heights[rowIndex]) / 2;
        return { x, y };
    }, [nodeDataMap, columnDimensions]);

    /**
     * Edges (links) displayed between nodes. If editing, the midpoint of an edge
     * contains an "Add Node" button
     */
    const edges = useMemo(() => {
        console.log('in edges start before cancel check', columnDimensions)
        // If data required to render edges is not yet available
        // (i.e. no links, or column dimensions not complete)
        if (!links ||
            Object.values(columnDimensions).length === 0 ||
            Object.values(columnDimensions).some((d, i) => d.width === 0 && i > 0)) return [];
        console.log('calculating edges', columnDimensions);
        return links?.map(link => {
            if (!nodeDataMap[link.fromId] || !nodeDataMap[link.toId]) return null;
            // Find centers of nodes relative to their columns
            const fromPos = getNodePos(link.fromId);
            const toPos = getNodePos(link.toId);
            console.log('start/end', fromPos, toPos)
            if (!fromPos || !toPos) return null;
            return <NodeGraphEdge
                key={`edge-${link.id}`}
                start={fromPos}
                end={toPos}
                isEditable={isEditable}
                onAdd={() => { }}
            />
        })
    }, [columnDimensions, isEditable, links, nodeDataMap, scale]);

    /**
     * Node column objects
     */
    const columns = useMemo(() => {
        return columnData.map((columnData, index) => <NodeGraphColumn
            key={`node-column-${index}`}
            id={`node-column-${index}`}
            columnIndex={index}
            nodes={columnData}
            isEditable={isEditable}
            dragId={dragId}
            scale={scale}
            labelVisible={labelVisible}
            onDimensionsChange={onColumnDimensionsChange}
        />)
    }, [columnData, isEditable, scale, labelVisible, onColumnDimensionsChange, dragId]);

    return (
        <Box id="graph-root" position="relative" sx={{
            cursor: 'move',
            minWidth: '100%',
            height: { xs: '80vh', md: '90vh' },
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