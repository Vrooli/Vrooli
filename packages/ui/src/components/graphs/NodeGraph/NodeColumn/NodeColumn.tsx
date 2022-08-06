/**
 * Displays a list of nodes vertically.
 */
import { Box, Stack } from '@mui/material';
import { useMemo } from 'react';
import { NodeColumnProps } from '../types';
import { EndNode, RedirectNode, RoutineListNode, StartNode } from '../nodes';
import { NodeType } from 'graphql/generated/globalTypes';

export const NodeColumn = ({
    handleAction,
    handleNodeUpdate,
    id,
    isEditing,
    columnIndex,
    labelVisible,
    language,
    links,
    dragId,
    nodes,
    scale = 1,
    zIndex,
}: NodeColumnProps) => {
    // Padding between cells
    const padding = useMemo(() => scale * 25, [scale]);
    // Highlights column when a dragging node can be dropped on it
    const isHighlighted = useMemo(() => dragId, [dragId]);

    /**
     * Create a node component for the given node data. 
     * Each node is wrapped in a cell that accepts drag and drop. 
     */
    const nodeList = useMemo(() => {
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
                handleAction,
                isLinked: true,
                node,
                scale,
                label: node.title,
                labelVisible,
                isEditing,
                canDrag: isEditing,
                zIndex,
            }
            // Determine node to display based on node type
            switch (node.type) {
                case NodeType.End:
                    return <EndNode {...nodeProps} linksIn={links.filter(l => l.toId === node.id)} />
                case NodeType.Redirect:
                    return <RedirectNode {...nodeProps} />
                case NodeType.RoutineList:
                    return (<RoutineListNode
                        {...nodeProps}
                        canExpand={true}
                        handleUpdate={handleNodeUpdate}
                        language={language}
                        linksIn={links.filter(l => l.toId === node.id)}
                        linksOut={links.filter(l => l.fromId === node.id)}
                    />)
                case NodeType.Start:
                    return <StartNode {...nodeProps} linksOut={links.filter(l => l.fromId === node.id)} />
                default:
                    return null;
            }
        })
    }, [columnIndex, handleAction, handleNodeUpdate, isEditing, labelVisible, language, links, nodes, scale, zIndex]);

    return (
        <Stack
            id={id}
            direction="column"
            padding={`${padding}px`}
            paddingTop={'100px'}
            paddingBottom={'100px'}
            position="relative"
            display="flex"
            justifyContent="center"
            alignItems="center"
            sx={{
                backgroundColor: isHighlighted ? '#a2be6547' : 'transparent',
                borderLeft: isHighlighted ? '1px solid #71c84f' : 'none',
                borderRight: isHighlighted ? '1px solid #71c84f' : 'none',
                gap: `${padding * 4}px`,
                // Fill available if column is empty
                width: nodes.length === 0 ? '-webkit-fill-available' : 'auto',
            }}
        >
            {nodeList}
        </Stack>
    )
}