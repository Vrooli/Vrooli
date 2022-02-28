/**
 * Displays a list of nodes vertically.
 */
import { Stack } from '@mui/material';
import { useMemo } from 'react';
import { NodeGraphColumnProps } from '../types';
import { EndNode, LoopNode, RedirectNode, RoutineListNode, StartNode } from '../nodes';
import { NodeType } from 'graphql/generated/globalTypes';

export const NodeGraphColumn = ({
    id,
    scale = 1,
    columnNumber,
    nodes,
    labelVisible,
    isEditable,
    dragId,
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
            label: node.title,
            labelVisible,
            isEditable,
            // Cannot drop onto itself or a start or end node
            isHighlighted: Boolean(dragId && dragId !== node.id && node.type !== NodeType.Start && node.type !== NodeType.End),
        }
        // Determine node to display based on node type
        switch (node.type) {
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
            // Spacing is handled by nodes, to allow their whole background to highlight on hover
            spacing={0}
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