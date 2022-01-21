import { NodeGraphCellProps } from '../types';
import { Box } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import Draggable from 'react-draggable';
import Measure from 'react-measure';

/**
 * Wraps node with drag and drop functionality. This can be disabled for certain nodes.
 * NOTE: If a node is draggable, the draggable part must have the "handle" class
 */
export const NodeGraphCell = ({
    draggable = true,
    droppable = true,
    isDragging,
    dragIsOver,
    nodeId,
    children,
    onDragStart,
    onDrag,
    onDrop,
    onResize,
}: NodeGraphCellProps) => {
    // True if this node is being dragged
    const [thisNodeDragging, setThisNodeDragging] = useState<boolean>(false);

    const background = useMemo(() => {
        console.log('calculating background', isDragging);
        // If this node is being dragged, or nothing is being dragged (i.e. default)
        if (!isDragging || thisNodeDragging) return 'transparent';
        // Background if a node is being dragged, but not over this cell
        if (!dragIsOver) return "#a7ec9a";
        // Background if a node is being dragged over this cell, but dropping is disabled
        if (!droppable) return "#e93e0a";
        // Background if a node is being dragged over this cell, and dropping is enabled
        return "#3cd41f";
    }, [dragIsOver, droppable, isDragging, thisNodeDragging]);

    const handleResize = useCallback(({ width, height }: any) => {
        onResize(nodeId, { width, height });
    }, [nodeId, onResize])

    /**
     * Handles start of drag. Draggable expects either false or void, so we can't return true
     */
    const handleDragStart = useCallback((e: any, data: any) => { 
        if (draggable) {
            // Determine current drag position
            const { x, y } = data;
            onDragStart(nodeId, { x, y });
            return;
        }
        return false 
    }, [draggable]);

    const handleDrag = useCallback((e: any, data: any) => {
        if (!thisNodeDragging) setThisNodeDragging(true);
        // Determine current drag position
        const { x, y } = data;
        // Pass drag event up, so other cells can determine if they are dragged over
        onDrag(nodeId, { x, y });
    }, [thisNodeDragging, nodeId, onDrag]);

    const handleDrop = useCallback((e: any, data: any) => {
        setThisNodeDragging(false);
        // Determine current drag position
        const { x, y } = data;
        // Pass drop event up
        onDrop(nodeId, { x, y });
    }, [nodeId, onDrop]);

    return (
        <Measure
            bounds
            onResize={handleResize}
        >
            {({ measureRef }) => (
                <Draggable
                    handle=".handle"
                    defaultPosition={{ x: 0, y: 0 }}
                    scale={1}
                    onStart={handleDragStart}
                    onDrag={handleDrag}
                    onStop={handleDrop}>
                    <Box
                        ref={measureRef}
                        width={'100%'}
                        height={'100%'}
                        display={"flex"}
                        justifyContent={"center"}
                        alignItems={"center"}
                        sx={{ 
                            background,
                            zIndex: thisNodeDragging ? 10 : 5,
                        }}
                    >
                        {children}
                    </Box>
                </Draggable>
            )}
        </Measure>
    )
}