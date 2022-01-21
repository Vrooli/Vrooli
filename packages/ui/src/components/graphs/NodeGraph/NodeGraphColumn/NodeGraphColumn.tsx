import { Stack } from '@mui/material';
import { useMemo } from 'react';
import { NodeGraphCellProps, NodeGraphColumnProps } from '../types';
import { NodeType } from '@local/shared';
import { CombineNode, DecisionNode, EndNode, LoopNode, RedirectNode, RoutineListNode, StartNode } from '../nodes';
import { NodeGraphCell } from '../NodeGraphCell/NodeGraphCell';

export const NodeGraphColumn = ({
    id,
    scale = 1,
    columnNumber,
    nodes,
    labelVisible,
    isEditable,
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
        // Common cell props
        const cellProps: { key: string } & Omit<NodeGraphCellProps, 'children'> = {
            key: `node-${columnNumber}-${index}`,
            nodeId: node.id,
            draggable: ![NodeType.Start, NodeType.End].includes(node.type),
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
                return <NodeGraphCell {...cellProps}><CombineNode {...nodeProps} /></NodeGraphCell>;
            case NodeType.Decision:
                return <NodeGraphCell {...cellProps}><DecisionNode {...nodeProps} /></NodeGraphCell>;
            case NodeType.End:
                return <NodeGraphCell {...cellProps}><EndNode {...nodeProps} /></NodeGraphCell>;
            case NodeType.Loop:
                return <NodeGraphCell {...cellProps}><LoopNode {...nodeProps} /></NodeGraphCell>;
            case NodeType.RoutineList:
                return <NodeGraphCell {...cellProps}><RoutineListNode {...nodeProps} onAdd={() => {}} /></NodeGraphCell>;
            case NodeType.Redirect:
                return <NodeGraphCell {...cellProps}><RedirectNode {...nodeProps} /></NodeGraphCell>;
            case NodeType.Start:
                return <NodeGraphCell {...cellProps}><StartNode {...nodeProps} /></NodeGraphCell>;
            default:
                return null;
        }
    }) ?? [], [nodes, scale, labelVisible, isEditable]);

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