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
    // How far the node has been dragged
    const dragDistanceRef = useRef({ x: 0, y: 0 });
    // Stores if drag was already published
    const isDragPublished = useRef(false);

    /**
     * Handles mouse down event. Sets absolute position reference, since drag start is relative to the 
     * start position we set on Draggable, but we don't have the real start position at that point. 
     * Worse, this is called before dragStart on click, but after on touch
     */
    const handleMouseDown = useCallback((ev: any) => {
        const x = (ev as MouseEvent)?.clientX ?? (ev as TouchEvent)?.touches[0].clientX ?? 0;
        const y = (ev as MouseEvent)?.clientY ?? (ev as TouchEvent)?.touches[0].clientY ?? 0;
        console.log('DRAGGABLE NODE handleMouseDown', x, y);
        dragStartRef.current = { x, y };
    }, []);

    /**
     * Handles start of drag. Draggable expects either false or void, so we can't return true
     */
    const handleDragStart = useCallback(() => {
        console.log('DRAGGABLE NODE handleDragStart');
        if (canDrag) {
            setIsDragging(true);
            // Initialize drag distance
            dragDistanceRef.current = { x: 0, y: 0 };
            return;
        }
        return false
    }, [canDrag]);

    const handleDrag = useCallback((_e: any, data: any) => {
        // Determine current drag distance
        const { x, y } = data;
        // If dragged a little bit (to distinguish from a click), set isDragging to true
        if (!isDragPublished.current && (Math.abs(x - dragDistanceRef.current.x) > 5 || Math.abs(y - dragDistanceRef.current.y) > 5)) {
            isDragPublished.current = true;
            // Alert other components that this node is being dragged (to turn on highlights, for example)
            PubSub.publish(Pubs.NodeDrag, { nodeId });
        }
    }, [nodeId]);

    const handleDrop = useCallback((_e: any, data: any) => {
        // Reset drag data
        isDragPublished.current = false;
        setIsDragging(false);
        // Determine dropped distance. Add with absolute start position to get absolute drop position
        const { x: distX, y: distY } = data;
        const { x: startX, y: startY } = dragStartRef.current;
        const dropX = startX + distX;
        const dropY = startY + distY;
        // Reset drag distance and start position
        dragDistanceRef.current = { x: 0, y: 0 };
        dragStartRef.current = { x: 0, y: 0 };
        // Send drop event to parent
        PubSub.publish(Pubs.NodeDrop, { nodeId, position: { x: dropX, y: dropY } });
    }, [nodeId]);

    return (
        <Draggable
            handle=".handle"
            defaultPosition={{ x: 0, y: 0 }}
            disabled={!canDrag}
            position={isDragging ? undefined : { x: 0, y: 0 }}
            scale={1}
            onMouseDown={handleMouseDown}
            onStart={handleDragStart}
            onDrag={handleDrag}
            onStop={handleDrop}>
            <Box
                id={`node-draggable-container-${nodeId}`}
                onTouchStart={handleMouseDown}
                display={"flex"}
                justifyContent={"center"}
                alignItems={"center"}
                sx={{
                    zIndex: isDragging ? 10000 : 100000,
                    opacity: isDragging ? 0.5 : 1,
                    cursor: canDrag ? 'grab' : 'pointer',
                    '&:active': {
                        cursor: canDrag ? 'grabbing' : 'pointer',
                    },
                }}
                {...props}
            >
                {children}
            </Box>
        </Draggable>
    )
}