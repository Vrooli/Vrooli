/**
 * Contains the nodes and links (edges) that describe a routine orchestration.
 * Nodes are displayed within columns. Dragging an unlinked node into a column
 * will add it to the orchestration. Links are generated automatically if possible.
 * Otherwise, a popup is displayed to allow the user to manually specify which node the link should connect to.
 */
import { Box, Stack } from '@mui/material';
import { NodeGraphColumn, NodeGraphEdge } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { OrchestrationStatus, Pubs } from 'utils';
import { NodeGraphProps, NodePos } from '../types';
import { NodeWidth } from '..';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node } from 'types';

type Positions = {
    x: number;
    y: number;
}

type Dimensions = {
    width: number;
    height: number;
}

export const NodeGraphContainer = ({
    scale = 1,
    isEditable = true,
    labelVisible = true,
    nodes,
    links,
    onStatusChange,
}: NodeGraphProps) => {
    // Stores heights of node cells, used for positioning after drag and drop
    const [cellDimensions, setCellDimensions] = useState<{ [x: string]: Dimensions }>({});
    // ID of node being dragged, if any
    const [dragId, setDragId] = useState<string | null>(null);

    /**
     * Updates dimensions of a node cell
     */
    const onCellResize = useCallback((nodeId: string, { width, height }: Dimensions) => {
        setCellDimensions(dimensions => ({ ...dimensions, [nodeId]: { width, height } }));
    }, []);

    const handleDragStart = useCallback((nodeId: string) => { setDragId(nodeId) }, []);

    /**
     * Makes sure drop is valid, then updates order of nodes
     */
    const handleDragStop = useCallback((nodeId: string, { x, y }: Positions) => {
        console.log('DRAG STOPPPPPPPP START', nodeId, { x, y });
        setDragId(null);
        PubSub.publish(Pubs.NodeDrag, { nodeId: undefined });
        if (!nodes || nodes.length === 0) return;
        console.log('DRAG STOPPPPPPP A', cellDimensions)
        // Determine column that node is being dropped into
        const columnWidth = scale * 400;
        const columnIndex = Math.floor(x / columnWidth);
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
    }, [nodes, scale, cellDimensions]);

    // Set listeners for:
    // - click-and-drag background
    // - drag-and-drop nodes
    useEffect(() => {
        // Mouse drag state
        let touched = false;
        let lastPosition: { x: number, y: number } = { x: 0, y: 0 };
        // Only drag if not pressing a node or edge
        const onMouseDown = (ev: MouseEvent) => {
            const targetId = (ev.target as any)?.id;
            if (!targetId) return;
            if (targetId.startsWith('node-column') || targetId === 'graph-root' || targetId === 'graph-grid') {
                touched = true;
                lastPosition = { x: ev.clientX, y: ev.clientY };
            } else
                console.log('onmousedown fail', targetId)
        }
        const onMouseUp = (ev: MouseEvent) => {
            touched = false;
        }
        const onMouseMove = (ev: MouseEvent) => {
            if (touched) {
                const deltaX = ev.clientX - lastPosition.x;
                const deltaY = ev.clientY - lastPosition.y;
                document.getElementById('graph-root')?.scrollBy(-deltaX, -deltaY);
                lastPosition = { x: ev.clientX, y: ev.clientY };
            }
        }
        // Add event listeners
        window.addEventListener('mousedown', onMouseDown);
        window.addEventListener('mouseup', onMouseUp);
        window.addEventListener('mousemove', onMouseMove);
        // Add PubSub subscribers
        let dragStartSub = PubSub.subscribe(Pubs.NodeDrag, (_, data) => {
            handleDragStart(data.nodeId);
        });
        let dragDropSub = PubSub.subscribe(Pubs.NodeDrop, (_, data) => {
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
        return [posMap, unlinked];
    }, [nodes, links]);

    /**
     * Updates known node dimensions. 
     * Can't make this useMemo because RoutineListNodes can change size
     */
    useEffect(() => {
        let dimMap: { [x: string]: Dimensions } = {};
        if (!nodes) {
            setCellDimensions(dimMap);
            return;
        }
        for (let i = 0; i < nodes.length; i++) {
            const curr = nodes[i];
            dimMap[curr.id] = {
                width: NodeWidth[curr.type] * scale,
                // Routine Lists, unless opened, are 45 pixels tall. Every other node has the same 
                // height as its width.
                height: curr.type === NodeType.RoutineList ? 45 : NodeWidth[nodes[i].type] * scale,
            }
        }
        setCellDimensions(dimMap);
    }, [nodes, scale]);

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

    /**
     * Creates dictionary of cell positions
     */
    useEffect(() => {
        // Create map of cell positions
        let posMap: { [id: string]: Positions } = {};
        // If cell dimensions haven't been calculated yet, return empty map
        const columnItemCount = columnData.map(col => col.length).reduce((a, b) => a + b, 0);
        if (Object.keys(cellDimensions).length !== columnItemCount) return;
        // Holds x position of current column
        let x = 25; // Accounts for left padding
        // First loop through to determine column heights. This is needed so
        // column nodes can be centered vertically
        let maxHeight = 0;
        let columnHeights: number[] = [];
        for (let i = 0; i < columnData.length; i++) {
            console.log('calculating cell positions: in column', columnData[i]);
            const curr = columnData[i];
            // Calculate height of current column
            let height = 0;
            for (let j = 0; j < curr.length; j++) {
                console.log('calculating height of cell in column', curr[j].id, cellDimensions[curr[j].id]);
                // Every node besides the first has a top margin, to separate them from the previous node.
                // The margin is 10 times the Stack spacing in NodeGraphColumn. We hardcode this here, since 
                // the spacing is not configurable
                if (j > 0) height += 100;
                height += cellDimensions[curr[j].id].height;
            }
            console.log('column height', height);
            // Add height to list
            columnHeights.push(height);
            // If height is larger than current max, update max
            if (height > maxHeight) maxHeight = height;
        }
        console.log('final maxheight', maxHeight);
        // Now loop through again to calculate cell positions
        for (let i = 0; i < columnData.length; i++) {
            // Holds y position of current node
            // Since column needs to be centered vertically, the y position
            // starts halfway between the largest column height and the current column height
            let y = (maxHeight - columnHeights[i]) / 2;
            // Holds widest node in current column
            let widestWidth = 0;
            // Loop through nodes in current column
            for (let j = 0; j < columnData[i].length; j++) {
                // Populate map with node's position
                posMap[columnData[i][j].id] = { x, y };
                // Grab dimensions of current node
                const { width, height } = cellDimensions[columnData[i][j].id];
                // Update widest node width
                if (width > widestWidth) widestWidth = width;
                // Increment y position
                y += height;
            }
            // Increment x position by widest node in column + spacing between node cells
            x += widestWidth + 50;
        }
        console.log('final cell positions', posMap);
        setCellPositions(posMap);
    }, [columnData, cellDimensions]);

    /**
     * Finds the center height of a node within a column
     */
    const getNodeCenterHeight = (node: Node) => {
        // Get column number of node
        const column = nodeDataMap[node.id]?.column;
        if (!column || column < 0) return 0;
        // Get height of all nodes in column
        const columnHeight = columnHeights[column];
    }


    /**
     * Edges (links) displayed between nodes. If editing, the midpoint of an edge
     * contains an "Add Node" button
     */
    const edges = useMemo(() => {
        // If data required to render edges is not yet available
        // (i.e. no links, or number of links is more than the number of data available about nodes)
        if (!links ||
            links.length > Object.keys(nodeDataMap).length + 1 ||
            links.length > Object.keys(cellDimensions).length + 1) return [];
        console.log('calculating edges', links);
        return links?.map(link => {
            if (!nodeDataMap[link.fromId] || !nodeDataMap[link.toId]) return null;
            // Find columns
            const startColumn = nodeDataMap[link.fromId].column;
            const endColumn = nodeDataMap[link.toId].column;
            const columnWidth = 400 * scale;
            // Center of cells the edge is attached to
            const startPos: Positions = {
                x: (startColumn * columnWidth) + (columnWidth / 2),
                y: cellPositions[link.fromId].y + cellDimensions[link.fromId].height / 2,
            }
            const endPos: Positions = {
                x: (endColumn * columnWidth) + (columnWidth / 2),
                y: cellPositions[link.toId].y + cellDimensions[link.toId].height / 2,
            }
            console.log('start/end', startPos, endPos)
            return <NodeGraphEdge
                key={`edge-${link.id}`}
                start={startPos}
                end={endPos}
                isEditable={isEditable}
                onAdd={() => { }}
            />
        })
    }, [cellDimensions, isEditable, links, nodeDataMap, scale]);

    /**
     * Node column objects
     */
    const columns = useMemo(() => {
        return columnData.map((columnData, index) => <NodeGraphColumn
            key={`node-column-${index}`}
            id={`node-column-${index}`}
            columnNumber={index}
            nodes={columnData}
            isEditable={isEditable}
            dragId={dragId}
            scale={scale}
            labelVisible={labelVisible}
            onResize={onCellResize}
        />)
    }, [columnData, isEditable, scale, labelVisible, onCellResize]);

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
            {/* Nodes */}
            <Stack id="graph-grid" spacing={0} direction="row" zIndex={5} sx={{
                width: '5000px',
                height: '-webkit-fill-available',
                // Create grid background pattern
                backgroundColor: `#a8b6c3`,
                '--line-color': `rgba(0 0 0 / .05)`,
                '--line-thickness': `1px`,
                '--minor-length': `${2 * ((scale * 100 % 250) + 1)}px`,
                '--major-length': `${20 * ((scale * 100 % 250) + 1)}px`,
                '--line': `var(--line-color) 0 var(--line-thickness)`,
                '--small-body': `transparent var(--line-thickness) var(--minor-length)`,
                '--large-body': `transparent var(--line-thickness) var(--major-length)`,

                '--small-squares': `repeating-linear-gradient(to bottom, var(--line), var(--small-body)), repeating-linear-gradient(to right, var(--line), var(--small-body))`,

                '--large-squares': `repeating-linear-gradient(to bottom, var(--line), var(--large-body)), repeating-linear-gradient(to right, var(--line), var(--large-body))`,
                background: `var(--small-squares), var(--large-squares)`,
            }}>
                {columns}
            </Stack>
        </Box>
    )
};