import { Box, BoxProps, styled } from "@mui/material";
import { useCallback, useState } from "react";
import Draggable, { DraggableEvent } from "react-draggable";
import { PubSub } from "utils/pubsub";
import { DraggableNodeProps } from "../types";

const DEFAULT_POSITION = { x: 0, y: 0 } as const;

interface DragBoxProps extends BoxProps {
    canDrag: boolean;
    isDragging: boolean;
}

const DragBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isDragging" && prop !== "canDrag",
})<DragBoxProps>(({ canDrag, isDragging }) => ({
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    zIndex: isDragging ? 10000 : 100000,
    opacity: isDragging ? 0.5 : 1,
    cursor: canDrag ? "grab" : "pointer",
    "&:active": {
        cursor: canDrag ? "grabbing" : "pointer",
    },
}));

/**
 * Wraps node with drag and drop functionality. This can be disabled for certain nodes.
 * NOTE: If a node is draggable, the draggable part must have the "handle" class
 */
export function DraggableNode({
    canDrag = true,
    nodeId,
    children,
    ...props
}: DraggableNodeProps) {
    // True if this node is being dragged
    const [isDragging, setIsDragging] = useState<boolean>(false);

    /**
     * Handles start of drag. Draggable expects either false or void, so we can't return true
     */
    const handleDragStart = useCallback(() => {
        if (canDrag) {
            setIsDragging(true);
        }
    }, [canDrag]);

    const handleDrop = useCallback((e: DraggableEvent) => {
        setIsDragging(false);
        // Get viewport position of drop point
        let clientX, clientY;
        // This is a bit trickier for touch events
        if (Object.prototype.hasOwnProperty.call(e, "changedTouches") && (e as TouchEvent).changedTouches.length > 0) {
            const touchEvent = e as TouchEvent;
            clientX = touchEvent.changedTouches[0].clientX;
            clientY = touchEvent.changedTouches[0].clientY;
        } else {
            const mouseEvent = e as MouseEvent;
            clientX = mouseEvent.clientX;
            clientY = mouseEvent.clientY;
        }
        // Publish drop event so that the node graph can react to it
        PubSub.get().publish("nodeDrop", { nodeId, position: { x: clientX, y: clientY } });
    }, [nodeId]);

    return (
        <Draggable
            handle=".handle"
            defaultPosition={DEFAULT_POSITION}
            disabled={!canDrag}
            position={DEFAULT_POSITION}
            scale={1}
            onStart={handleDragStart}
            onStop={handleDrop}>
            <DragBox
                id={`node-draggable-container-${nodeId}`}
                canDrag={canDrag}
                isDragging={isDragging}
                {...props}
            >
                {children}
            </DragBox>
        </Draggable>
    );
}
