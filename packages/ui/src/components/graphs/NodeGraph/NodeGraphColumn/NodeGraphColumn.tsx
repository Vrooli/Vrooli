import { Stack } from '@mui/material';
import { useMemo } from 'react';
import { NodeGraphColumnProps } from '../types';
import { NodeType } from '@local/shared';
import { CombineNode, DecisionNode, EndNode, LoopNode, RedirectNode, RoutineListNode, StartNode } from '../nodes';

export const NodeGraphColumn = ({
    id,
    scale = 1,
    columnNumber,
    nodes,
    labelVisible,
    isEditable,
    onResize,
}: NodeGraphColumnProps) => {
    const padding = useMemo(() => `${scale * 25}px`, [scale]);

    /**
     * Create a node component for the given node data. 
     * Each node is wrapped in a cell that accepts drag and drop. 
     */
    const nodeList = useMemo(() => nodes?.map((node, index) => {
        // Common node props
        const nodeProps = {
            key: `node-${columnNumber}-${index}`,
            node,
            scale,
            label: node?.title,
            labelVisible,
            isEditable
        }
        // Determine node to display based on node type
        switch (node.type) {
            case NodeType.Combine:
                return <CombineNode {...nodeProps} />
            case NodeType.Decision:
                return <DecisionNode {...nodeProps} />
            case NodeType.End:
                return <EndNode {...nodeProps} />
            case NodeType.Loop:
                return <LoopNode {...nodeProps} />
            case NodeType.RoutineList:
                return <RoutineListNode {...nodeProps} onAdd={() => { }} onResize={onResize} />
            case NodeType.Redirect:
                return <RedirectNode {...nodeProps} />
            case NodeType.Start:
                return <StartNode {...nodeProps} />
            default:
                return null;
        }
    }) ?? [], [nodes, scale, labelVisible, isEditable, onResize]);

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
            sx={{ backgroundColor: 'transparent' }}
        >
            {nodeList}
        </Stack>
    )
}