import { NodeGraphCellProps } from '../types';
import { Box } from '@mui/material';
import { useCallback, useMemo } from 'react';
import Draggable from 'react-draggable';

/**
 * Wraps node with drag and drop functionality. This can be disabled for certain nodes.
 * NOTE: If a node is draggable, the draggable part must have the "handle" class
 */
export const NodeGraphCell = ({
    draggable = true,
    droppable = true,
    dragIsOver,
    nodeId,
    children,
    onDrag,
    onDrop,
}: NodeGraphCellProps) => {
    const background = useMemo(() => dragIsOver ? '#f5f5f5' : 'transparent', [dragIsOver]);

    /**
     * Handles start of drag. Draggable expects either false or void, so we can't return true
     */
    const handleDragStart = useCallback(() => {if (draggable) return; return false}, [draggable]);

    const handleDrag = useCallback((e: any, data: any) => {
        // Determine current drag position
        const { x, y } = data;
        // Pass drag event up, so other cells can determine if they are dragged over
        onDrag(nodeId, {x, y});
    }, [nodeId, onDrag]);

    const handleDrop = useCallback((e: any, data: any) => {
        // Determine current drag position
        const { x, y } = data;
        // Pass drop event up
        onDrop(nodeId, {x, y});
    }, [nodeId, onDrop]);

    return (
        <Draggable
            handle=".handle"
            defaultPosition={{ x: 0, y: 0 }}
            scale={1}
            onStart={handleDragStart}
            onDrag={handleDrag}
            onStop={handleDrop}>
            <Box
                width={'100%'}
                height={'100%'}
                display={"flex"}
                justifyContent={"center"}
                alignItems={"center"}
                sx={{ background }}
            >
                {children}
            </Box>
        </Draggable>
    )
}