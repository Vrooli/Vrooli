import { Stack } from '@mui/material';
import { useMemo } from 'react';
import { NodeGraphCellProps, NodeGraphColumnProps } from '../types';
import { NodeType } from '@local/shared';
import { CombineNode, DecisionNode, EndNode, LoopNode, RedirectNode, RoutineListNode, StartNode } from '../nodes';
import { NodeGraphCell } from '../NodeGraphCell/NodeGraphCell';
import { NodeWidth } from '..';

export const NodeGraphColumn = ({
    id,
    scale = 1,
    columnNumber,
    nodes,
    labelVisible,
    isEditable,
    graphDimensions,
    cellPositions,
    dragData,
    isDragging,
    onDragStart,
    onDrag,
    onDrop,
    onResize,
}: NodeGraphColumnProps) => {
    const padding = useMemo(() => `${scale * 25}px`, [scale]);

    /**
     * Create a node component for the given node data. 
     * Each node is wrapped in a cell that accepts drag and drop. 
     */
    const nodeList = useMemo(() => nodes?.map((node, index) => {
        // Calculate dragIsOver;
        let dragIsOver = false;
        if (graphDimensions && dragData && cellPositions && cellPositions.length > index) {
            // Find current cell position
            const cellTopLeft = cellPositions[index];
            // Half the width of the node being dragged
            const midway = NodeWidth[dragData.type] * scale / 2;
            // If node is at least halfway into the cell, it is dragIsOver
            if (dragData.x > (cellTopLeft.x - midway) && // Drag is past the left wall threshold
                dragData.x < (cellTopLeft.x + midway) && // Drag is not past the right wall threshold
                dragData.y > cellTopLeft.y && // Drag is past the top wall threshold
                (cellPositions.length > index + 1) ? dragData.y < cellPositions[index + 1].y : dragData.y < graphDimensions.height && // Drag is not past the bottom wall threshold (or overall graph height)
                dragData.x < graphDimensions.width && // Drag is not outside the overall graph width
                dragData.y > 0 && // Drag is not above the overall graph
                dragData.x > 0) { // Drag is not to the left of the overall graph
                dragIsOver = true;
            }
        }
        // Common cell props
        const cellProps: { key: string } & Omit<NodeGraphCellProps, 'children'> = {
            key: `node-${columnNumber}-${index}`,
            nodeId: node.id,
            draggable: ![NodeType.Start, NodeType.End].includes(node.type),
            droppable: columnNumber !== 0,
            isDragging,
            dragIsOver,
            onDragStart,
            onDrag,
            onDrop,
            onResize
        }
        // Common node props
        const nodeProps = {
            node,
            scale,
            label: node?.title,
            labelVisible,
            isEditable
        }
        // Determine node to display based on node type
        switch (node.type) {
            case NodeType.Combine:
                return <NodeGraphCell {...cellProps}><CombineNode {...nodeProps} dragIsOver={dragIsOver} /></NodeGraphCell>;
            case NodeType.Decision:
                return <NodeGraphCell {...cellProps}><DecisionNode {...nodeProps} dragIsOver={dragIsOver} /></NodeGraphCell>;
            case NodeType.End:
                return <NodeGraphCell {...cellProps}><EndNode {...nodeProps} /></NodeGraphCell>;
            case NodeType.Loop:
                return <NodeGraphCell {...cellProps}><LoopNode {...nodeProps} dragIsOver={dragIsOver} /></NodeGraphCell>;
            case NodeType.RoutineList:
                return <NodeGraphCell {...cellProps}><RoutineListNode {...nodeProps} dragIsOver={dragIsOver} onAdd={() => {}} /></NodeGraphCell>;
            case NodeType.Redirect:
                return <NodeGraphCell {...cellProps}><RedirectNode {...nodeProps} dragIsOver={dragIsOver} /></NodeGraphCell>;
            case NodeType.Start:
                return <NodeGraphCell {...cellProps}><StartNode {...nodeProps} /></NodeGraphCell>;
            default:
                return null;
        }
    }) ?? [], [nodes, scale, columnNumber, labelVisible, isEditable, graphDimensions, cellPositions, dragData, isDragging, onDragStart, onDrag, onDrop, onResize]);

    return (
        <Stack 
            id={id} 
            spacing={10} 
            direction="column" 
            padding={padding}
            position="relative"
            display="flex"
            justifyContent="center"
            alignItems="center"
            sx={{backgroundColor: 'transparent'}}
        >
            {nodeList}
        </Stack>
    )
}