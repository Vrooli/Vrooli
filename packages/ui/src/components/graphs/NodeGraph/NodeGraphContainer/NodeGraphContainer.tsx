import { Box, Stack } from '@mui/material';
import { CombineNodeData, DecisionNodeData, DecisionNodeDataDecision, NodeData, NodeType } from '@local/shared';
import { NodeGraphColumn, NodeGraphEdge } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Pubs } from 'utils';
import { NodeGraphProps, NodePos } from '../types';
import { NodeWidth } from '..';

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
}: NodeGraphProps) => {
    // Stores positions and sizes of node cells, which can be used to calculate dragIsOver
    const [cellPositions, setCellPositions] = useState<{ [x: string]: Positions }>({});
    const [cellDimensions, setCellDimensions] = useState<{ [x: string]: Dimensions }>({});
    // Position that drag started
    const [draggingId, setDraggingId] = useState<string | undefined>(undefined);

    /**
     * Updates dimensions of a node cell
     */
    const onCellResize = useCallback((nodeId: string, { width, height }: Dimensions) => {
        console.log('onCellResize', nodeId, { width, height });
        setCellDimensions(dimensions => ({ ...dimensions, [nodeId]: { width, height } }));
    }, []);

    const handleDragStart = useCallback((nodeId: string) => {
        setDraggingId(nodeId);
    }, []);

    /**
     * Makes sure drop is valid, then updates order of nodes
     */
    const handleDragStop = useCallback((nodeId: string, { x, y }: Positions) => {
        setDraggingId(undefined);
        PubSub.publish(Pubs.NodeDrag, { nodeId: undefined });
        if (!nodes || nodes.length === 0) return;
        // x and y define the top left of the node that was dropped
        // We need to calculate the center of the node, to determine if it was dropped 
        // in a valid position
        const center = {
            x: x + (cellDimensions[nodeId]?.width ?? 0) / 2,
            y: y + (cellDimensions[nodeId]?.height ?? 0) / 2,
        };
        // Check if the node being dragged is over another node
        for (const node of nodes) {
            // Can't drop on itself or a start/end node
            if (nodeId !== nodeId && node.type !== NodeType.Start && node.type !== NodeType.End) {
                const position = cellPositions[node.id];
                if (center.x > position.x && // Drag is past the left wall threshold
                    center.x < (cellDimensions[node.id]?.width ?? 0) && // Drag is not past the right wall threshold
                    center.y > position.y && // Drag is past the top wall threshold
                    center.y < (cellDimensions[node.id]?.height ?? 0) // Drag is not past the bottom wall threshold
                ) {
                    // TODO reorder nodes
                    return;
                }
            }
        }
        //TODO put the node back where it was if invalid, or center if valid
    }, [nodes, cellPositions, cellDimensions]);

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
            if (targetId.startsWith('node-column') || targetId === 'graph-root') {
                touched = true;
                lastPosition = { x: ev.clientX, y: ev.clientY };
            }
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
     * Dictionary of node data and their columns
     */
    const nodeDataMap: { [id: string]: NodePos } = useMemo(() => {
        // Position map for calculating node positions
        let posMap: { [id: string]: NodePos } = {};
        if (!nodes) return posMap;
        let startNodeId: string | null = null;
        console.log('node data map', nodes);
        // First pass of raw node data, to locate start node and populate position map
        for (let i = 0; i < nodes.length; i++) {
            console.log('raw data', nodes[i]);
            const currId = nodes[i].id;
            // If start node, must be in first column
            if (nodes[i].type === NodeType.Start) {
                startNodeId = currId;
                posMap[currId] = {
                    column: 0,
                    node: nodes[i],
                }
            }
            // Otherwise, node's column is currently unknown
            else {
                posMap[currId] = {
                    column: -1,
                    node: nodes[i],
                }
            }
        }
        // If no start node was found, throw error
        if (!startNodeId) {
            PubSub.publish(Pubs.Snack, { message: 'No start node found', severity: 'error' });
            return posMap;
        }
        // Helper function for recursively updating position map
        const addNode = (nodeId: string) => {
            // Find node data
            const curr: NodePos = posMap[nodeId];
            console.log('addNode', nodeId, curr);
            // If node not found or column already calculated (exception being the start node at column 0), skip
            if (!curr || curr.column > 0) return;
            console.log('a')
            // Calculate node's column (unless it is the start node). This is the same for all node types EXCEPT for combine nodes
            if (curr.node.type === NodeType.Combine) {
                console.log('b.1')
                const previousNodes = (curr.node.data as CombineNodeData | null)?.from ?? [];
                const farthestPreviousNode = Math.max(...previousNodes.map(prev => posMap[prev].column));
                posMap[nodeId].column = farthestPreviousNode === -1 ? -1 : farthestPreviousNode + 1;
            }
            else if (curr.node.type !== NodeType.Start) {
                console.log('b.2')
                const prevNode = posMap[curr.node.previous ?? ''];
                console.log('prevNode', prevNode);
                if (!prevNode) return;
                posMap[nodeId].column = prevNode.column === -1 ? -1 : prevNode.column + 1;
                console.log('calculated column', posMap[nodeId].column);
            }
            // Call addNode on each next node. Thiss is the same for all node types EXCEPT for decision nodes
            if (curr.node.type === NodeType.Decision) {
                console.log('in decision logic', curr?.node?.data)
                const decisions: DecisionNodeDataDecision[] | undefined = (curr.node.data as DecisionNodeData | undefined)?.decisions;
                if (!decisions) return;
                for (let i = 0; i < decisions.length; i++) {
                    addNode(decisions[i].next);
                }
            }
            else {
                if (curr.node.next) addNode(curr.node.next);
            }
        }
        // Starting with the start node, search for other nodes
        addNode(startNodeId);
        return posMap;
    }, [nodes]);

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
                height: NodeWidth[nodes[i].type] * scale, //TODO: fix this
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
        let list: NodeData[][] = [];
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
        let x = 0;
        // First loop through to determine column heights. This is needed so
        // column nodes can be centered vertically
        let maxHeight = 0;
        let columnHeights: number[] = [];
        for (let i = 0; i < columnData.length; i++) {
            const curr = columnData[i];
            // Calculate height of current column
            let height = 0;
            for (let j = 0; j < curr.length; j++) {
                height += cellDimensions[curr[j].id].height;
            }
            // Add height to list
            columnHeights.push(height);
            // If height is larger than current max, update max
            if (height > maxHeight) maxHeight = height;
        }
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
        setCellPositions(posMap);
    }, [columnData, cellDimensions]);

    /**
     * Edges displayed between nodes. If editing, the midpoint of an edge
     * contains an "Add Node" button
     */
    const edges = useMemo(() => {
        // If data required to render edges is not yet available
        if (!nodes ||
            Object.keys(nodes).length !== Object.keys(cellPositions).length ||
            Object.keys(nodes).length !== Object.keys(cellDimensions).length) return [];
        return nodes.map(node => {
            console.log('in edge node loop', node)
            if (!node.previous || !node.next) return null;
            // Center of cells the edge is attached to
            const startPos: Positions = {
                x: cellPositions[node.previous].x + cellDimensions[node.previous].width / 2,
                y: cellPositions[node.previous].y + cellDimensions[node.previous].height / 2,
            }
            const endPos: Positions = {
                x: cellPositions[node.next].x + cellDimensions[node.next].width / 2,
                y: cellPositions[node.next].y + cellDimensions[node.next].height / 2,
            }
            console.log('start/end', startPos, endPos)
            return <NodeGraphEdge
                key={`edge-${node.id}`}
                start={startPos}
                end={endPos}
                isEditable={isEditable}
                onAdd={() => { }}
            />
        })
    }, [nodes, cellPositions, cellDimensions, isEditable]);

    /**
     * Highlight the areas of the graph that a node can be dropped into
     */
    const highlights = useMemo(() => {
        // If no nodes are being dragged
        if (!draggingId) return [];
        // If data required to render highlights is not yet available
        if (!nodes ||
            Object.keys(nodes).length !== Object.keys(cellPositions).length ||
            Object.keys(nodes).length !== Object.keys(cellDimensions).length) return [];
        return nodes.map(node => {
            // Cannot drop onto a start or end node
            if (node.type === NodeType.Start || node.type === NodeType.End) return null;
            return <Box
                position="absolute"
                zIndex={1}
                key={`highlight-${node.id}`}
                left={cellPositions[node.id].x}
                right={cellPositions[node.id].y}
                width={cellDimensions[node.id].width}
                height={cellDimensions[node.id].height}
                sx={{
                    background: "#6ef04e",
                    '&:hover': {
                        filter: `brightness(120%)`,
                        transition: 'filter 0.2s',
                    },
                }}
            />
        })
    }, [draggingId, nodes, cellPositions, cellDimensions]);

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
            scale={scale}
            labelVisible={labelVisible}
            onResize={onCellResize}
        />)
    }, [columnData, isEditable, scale, labelVisible, onCellResize]);

    return (
        <Box id="graph-root" position="relative" sx={{
            cursor: 'move',
            minWidth: '100%',
            minHeight: '100%',
            overflowX: 'scroll',
            overflowY: 'scroll',
        }}>
            {/* Edges */}
            {edges}
            {/* Highlighted squares to indicate valid drop areas */}
            {highlights}
            {/* Nodes */}
            <Stack spacing={0} direction="row">
                {columns}
            </Stack>
        </Box>
    )
};