/**
 * Displays a list of nodes vertically.
 */
import { Stack } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { NodeGraphColumnProps } from '../types';
import { EndNode, LoopNode, RedirectNode, RoutineListNode, StartNode } from '../nodes';
import { NodeType } from 'graphql/generated/globalTypes';
import { NodeWidth } from '..';

export const NodeGraphColumn = ({
    id,
    scale = 1,
    columnIndex,
    nodes,
    labelVisible,
    isEditable,
    dragId,
    onDimensionsChange,
}: NodeGraphColumnProps) => {
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
        if(!nodes|| nodes.length===0) console.log('NO NODES IN COLUMN!', columnIndex);
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
        console.log('onDimensionsChange', columnIndex, { width, heights, nodeIds, tops, centers });
        onDimensionsChange(columnIndex, { width, heights, nodeIds, tops, centers });
    }, [cellHeights, nodes, onDimensionsChange, scale]);

    /**
     * Create a node component for the given node data. 
     * Each node is wrapped in a cell that accepts drag and drop. 
     */
    const nodeList = useMemo(() => nodes?.map((node, index) => {
        // Common node props
        const nodeProps = {
            key: `node-${columnIndex}-${index}`,
            node,
            scale,
            label: node.title,
            labelVisible,
            isEditable,
        }
        // Determine node to display based on node type
        switch (node.type) {
            case NodeType.End:
                return <EndNode {...nodeProps} />
            case NodeType.Loop:
                return <LoopNode {...nodeProps} />
            case NodeType.RoutineList:
                return <RoutineListNode {...nodeProps} onAdd={() => { }} onResize={onCellResize} />
            case NodeType.Redirect:
                return <RedirectNode {...nodeProps} />
            case NodeType.Start:
                return <StartNode {...nodeProps} />
            default:
                return null;
        }
    }) ?? [], [dragId, nodes, scale, labelVisible, isEditable, onCellResize]);

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
                border: '2px dashed red', // TODO: Remove
                backgroundColor: isHighlighted ? '#a2be6547' : 'transparent', 
                borderLeft: isHighlighted ? '1px solid black' : 'none',
                borderRight: isHighlighted ? '1px solid black' : 'none',
                gap: `${padding * 4}px`,
            }}
        >
            {nodeList}
        </Stack>
    )
}