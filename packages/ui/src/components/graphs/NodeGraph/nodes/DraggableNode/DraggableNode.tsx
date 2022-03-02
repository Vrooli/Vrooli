import { DraggableNodeProps } from '../types';
import { Box } from '@mui/material';
import { useCallback, useRef, useState } from 'react';
import Draggable from 'react-draggable';
import { Pubs } from 'utils';

/**
 * Wraps node with drag and drop functionality. This can be disabled for certain nodes.
 * NOTE: If a node is draggable, the draggable part must have the "handle" class
 */
export const DraggableNode = ({
    canDrag = true,
    nodeId,
    children,
    ...props
}: DraggableNodeProps) => {

    // True if this node is being dragged
    const [isDragging, setIsDragging] = useState<boolean>(false);
    // Where the node started dragging from
    const dragStartRef = useRef({ x: 0, y: 0 });

    /**
     * Handles start of drag. Draggable expects either false or void, so we can't return true
     */
    const handleDragStart = useCallback((e: any, data: any) => {
        if (canDrag) {
            setIsDragging(true);
            // Determine current drag position
            const { x, y } = data;
            // Set drag start
            dragStartRef.current = { x, y };
            return;
        }
        return false
    }, [canDrag]);

    const handleDrag = useCallback((e: any, data: any) => {
        // Determine current drag position
        const { x, y } = data;
        // If dragged a little bit (to distinguish from a click), set isDragging to true
        if (Math.abs(x - dragStartRef.current.x) > 5 || Math.abs(y - dragStartRef.current.y) > 5) {
            setIsDragging(true);
            // Alert other components that this node is being dragged (to turn on highlights, for example)
            PubSub.publish(Pubs.NodeDrag, { nodeId });
        }
    }, [nodeId]);

    const handleDrop = useCallback((e: any, data: any) => {
        // Reset drag data
        setIsDragging(false);
        dragStartRef.current = { x: 0, y: 0 };
        // Determine dropped position
        const { x, y } = data;
        // Send drop event to parent
        PubSub.publish(Pubs.NodeDrop, { nodeId, position: {x, y} });
        // Reset node position. If the drop is accepted, the updated column data will move the node
        //TODO
    }, [nodeId]);

    return (
        <Draggable
            handle=".handle"
            defaultPosition={{ x: 0, y: 0 }}
            disabled={!canDrag}
            position={isDragging ? undefined : { x: 0, y: 0 }}
            scale={1}
            onStart={handleDragStart}
            onDrag={handleDrag}
            onStop={handleDrop}>
            <Box
                display={"flex"}
                justifyContent={"center"}
                alignItems={"center"}
                sx={{
                    zIndex: isDragging ? 10 : 5,
                    opacity: isDragging ? 0.5 : 1
                }}
                {...props}
            >
                {children}
            </Box>
        </Draggable>
    )
}