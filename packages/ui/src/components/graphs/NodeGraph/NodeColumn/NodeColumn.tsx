/**
 * Displays a list of nodes vertically.
 */
import { Box, Stack } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { NodeColumnProps } from '../types';
import { EndNode, RedirectNode, RoutineListNode, StartNode } from '../nodes';
import { NodeType } from 'graphql/generated/globalTypes';
import { NodeWidth } from '..';

export const NodeColumn = ({
    id,
    scale = 1,
    columnIndex,
    nodes,
    labelVisible,
    isEditing,
    dragId,
    onDimensionsChange,
    handleDialogOpen,
}: NodeColumnProps) => {
    // Stores heights of node cells, used for positioning after drag and drop
    const [cellHeights, setCellHeights] = useState<{ [x: string]: number }>({});
    // Padding between cells
    const padding = useMemo(() => scale * 25, [scale]);
    // Highlights column when a dragging node can be dropped on it
    const isHighlighted = useMemo(() => columnIndex > 0 && dragId && nodes.every(node => node.id !== dragId), [columnIndex, dragId, nodes]);

    /**
     * Updates dimensions of a node cell
     */
    const onCellResize = useCallback((nodeId: string, height: number) => {
        setCellHeights(dimensions => ({ ...dimensions, [nodeId]: height }));
    }, []);

    /**
     * Calculates positions of nodes in column
     */
    useEffect(() => {
        // Calculate column width as largest node width
        const width = Math.max(...nodes.map(node => NodeWidth[node.type] * scale));
        const heights = nodes.map(node => {
            if (node.type === NodeType.RoutineList) return cellHeights[node.id] || (45 * scale);
            return NodeWidth[node.type] * scale
        })
        const nodeIds = nodes.map(node => node.id);
        const tops: number[] = [];
        const centers: number[] = [];
        let y = 0;
        for (let i = 0; i < nodes.length; i++) {
            y += padding; // Top padding
            tops.push(y);
            centers.push(y + heights[i] / 2);
            y += heights[i] + padding; // Node height + Bottom padding
        }
        onDimensionsChange(columnIndex, { width, heights, nodeIds, tops, centers });
    }, [cellHeights, nodes, onDimensionsChange, scale]);

    /**
     * Create a node component for the given node data. 
     * Each node is wrapped in a cell that accepts drag and drop. 
     */
    const nodeList = useMemo(() => {
        console.log('in column nodelist', nodes);
        // Sort nodes by their row index
        if (nodes.length === 0) return null;
        // There may be gaps between the nodes. For each missing rowIndex,
        // add a placeholder node to fill the gap.
        // Create an array that's the length of the largest rowIndex + 1
        const nodesWithGaps = Array(Math.max(...nodes.map(node => (node.rowIndex ?? 0))) + 1).fill(null);
        // Loop through the nodes and add them to the array
        nodes.forEach(node => {
            if (node.rowIndex === undefined || node.rowIndex === null) return;
            nodesWithGaps[node.rowIndex as number] = node;
        })
        // Now that we have a complete array, create a list of nodes
        return nodesWithGaps.map((node, index) => {
            // If a placeholder, return a placeholder node
            if (node === null) {
                return (
                    <Box key={`node-placeholder-${columnIndex}-${index}`} sx={{
                        width: `${100 * scale}px`,
                        height: `${350 * scale}px`,
                    }} />
                )
            }
            // Otherwise, return correct node
            // Common node props
            const nodeProps = {
                key: `node-${columnIndex}-${index}`,
                isLinked: true,
                node,
                scale,
                label: node.title,
                labelVisible,
                isEditing,
                canDrag: isEditing,
            }
            // Determine node to display based on node type
            switch (node.type) {
                case NodeType.End:
                    return <EndNode {...nodeProps} />
                case NodeType.Redirect:
                    return <RedirectNode {...nodeProps} />
                case NodeType.RoutineList:
                    return <RoutineListNode {...nodeProps} canExpand={true} onAdd={() => { }} onResize={onCellResize} handleDialogOpen={handleDialogOpen} />
                case NodeType.Start:
                    return <StartNode {...nodeProps} />
                default:
                    return null;
            }
        })
    }, [dragId, nodes, scale, labelVisible, isEditing, onCellResize]);

    return (
        <Stack
            id={id}
            direction="column"
            padding={`${padding}px`}
            position="relative"
            display="flex"
            justifyContent="center"
            alignItems="center"
            sx={{
                backgroundColor: isHighlighted ? '#a2be6547' : 'transparent',
                borderLeft: isHighlighted ? '1px solid #71c84f' : 'none',
                borderRight: isHighlighted ? '1px solid #71c84f' : 'none',
                gap: `${padding * 4}px`,
            }}
        >
            {nodeList}
        </Stack>
    )
}