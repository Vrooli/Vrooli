import { DraggableNodeProps } from '../types';
import { Box } from '@mui/material';
import { useCallback, useRef, useState } from 'react';
import Draggable from 'react-draggable';
import { Pubs } from 'utils';

type DragRefs = {
    graphStartScroll: { scrollLeft: number, scrollTop: number } | null;
    nodeStartPosition: { x: number, y: number } | null;
}

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
    // Position refs
    const dragRefs = useRef<DragRefs>({
        graphStartScroll: null,
        nodeStartPosition: null,
    });
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
        // Find absolute position of cursor
        const x = (ev as MouseEvent)?.clientX ?? (ev as TouchEvent)?.touches[0].clientX ?? 0;
        const y = (ev as MouseEvent)?.clientY ?? (ev as TouchEvent)?.touches[0].clientY ?? 0;
        dragRefs.current.nodeStartPosition = { x, y };
        // Find current scroll position of graph
        const graph = document.getElementById('graph-root');
        if (graph) {
            const { scrollLeft, scrollTop } = graph;
            dragRefs.current.graphStartScroll = { scrollLeft, scrollTop };
        }
    }, []);

    /**
     * Handles start of drag. Draggable expects either false or void, so we can't return true
     */
    const handleDragStart = useCallback(() => {
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
        if (!isDragPublished.current && (Math.abs(x) > 5 || Math.abs(y) > 5)) {
            isDragPublished.current = true;
            // Alert other components that this node is being dragged (to turn on highlights, for example)
            PubSub.publish(Pubs.NodeDrag, { nodeId });
        }
    }, [nodeId]);

    const handleDrop = useCallback((_e: any, data: any) => { //TODO dragging along grid messes up the position. maybe store grid's scroll position on drag start?
        setIsDragging(false);
        // Get dropped distance from data
        const { x, y } = data;
        // Calculate scroll distance since drag start
        const grid = document.getElementById('graph-root') as HTMLElement;
        const graphScrollDistance = {
            x: grid.scrollLeft - (dragRefs.current.graphStartScroll as any)?.scrollLeft ?? 0,
            y: grid.scrollTop - (dragRefs.current.graphStartScroll as any)?.scrollTop ?? 0,
        };
        // Add drag distance to current position
        const { x: startX, y: startY } = dragRefs.current.nodeStartPosition as any;
        const dropX = startX + x - graphScrollDistance.x
        const dropY = startY + y - graphScrollDistance.y;
        // Reset drag distance and start position
        dragRefs.current = {
            graphStartScroll: null,
            nodeStartPosition: null,
        }
        // Send drop event to parent
        if (isDragPublished.current) {
            PubSub.publish(Pubs.NodeDrop, { nodeId, position: { x: dropX, y: dropY } });
            isDragPublished.current = false;
        }
    }, [nodeId]);

    return (
        <Draggable
            handle=".handle"
            defaultPosition={{ x: 0, y: 0 }}
            disabled={!canDrag}
            position={{ x: 0, y: 0 }}
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