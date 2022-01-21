import { NodeGraphCellProps } from '../types';
import { Box } from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import Draggable from 'react-draggable';
import Measure from 'react-measure';
import { Pubs } from 'utils';

/**
 * Wraps node with drag and drop functionality. This can be disabled for certain nodes.
 * NOTE: If a node is draggable, the draggable part must have the "handle" class
 */
export const NodeGraphCell = ({
    draggable = true,
    nodeId,
    children,
    onDragStart,
    onDrag,
    onDrop,
    onResize,
}: NodeGraphCellProps) => {

    // True if this node is being dragged
    const [isDragging, setIsDragging] = useState<boolean>(false);

    // Subscribe to drag over
    useEffect(() => {
        let snackSub = PubSub.subscribe(Pubs.NodeDrag, (_, data) => {
            if (data.nodeId === nodeId) {
                setIsDragging(true);
            } else if (isDragging) {
                setIsDragging(false);
            }
        });
        return () => { PubSub.unsubscribe(snackSub) };
    }, [isDragging])

    const handleResize = useCallback(({ bounds }: any) => {
        onResize(nodeId, { width: bounds.width, height: bounds.height });
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
        // Determine current drag position
        const { x, y } = data;
        // Pass drag event up, so other cells can determine if they are dragged over
        onDrag(nodeId, { x, y });
    }, [nodeId, onDrag]);

    const handleDrop = useCallback((e: any, data: any) => {
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
                            zIndex: isDragging ? 10 : 5,
                            opacity: isDragging ? 0.5 : 1
                        }}
                    >
                        {children}
                    </Box>
                </Draggable>
            )}
        </Measure>
    )
}