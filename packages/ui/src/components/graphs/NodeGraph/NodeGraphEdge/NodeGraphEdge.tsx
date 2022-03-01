import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { NodeGraphEdgeProps } from '../types';

type EdgePositions = {
    top: number;
    left: number;
    width: number;
    height: number;
    x1: number;
    y1: number;
    x2: number;
    y2: number;
}

// Displays a line between two nodes.
// If in editing mode, an "Add Node" button appears on the line. 
// This button always appears inbetween two node columns, to avoid collisions with nodes.
export const NodeGraphEdge = ({
    fromId,
    toId,
    isEditable = true,
    dragId,
    scale,
    onAdd,
}: NodeGraphEdgeProps) => {
    const [dims, setDims] = useState<EdgePositions>({ top: 0, left: 0, width: 0, height: 0, x1: 0, y1: 0, x2: 0, y2: 0 });
    const thiccness = useMemo(() => Math.ceil(scale * 3), [scale]);

    const getPoint = (id: string): { x: number, y: number } | null => {
        // Find graph and node
        const graph = document.getElementById('graph-root');
        const node = document.getElementById(`node-${id}`);
        if (!graph || !node) {
            console.error('Could not find node to connect to edge', id);
            return null;
        }
        const rect = node.getBoundingClientRect();
        console.log('BOUNDING RECT', id, rect);
        return {
            x: graph.scrollLeft + rect.left + (rect.width / 2),
            y: graph.scrollTop + rect.top + (rect.height / 2),
        }
    }

    const calculateDims = useCallback(() => {
        const start = getPoint(fromId);
        const end = getPoint(toId);
        let top = 0, left = 0, width = 0, height = 0, x1 = 0, y1 = 0, x2 = 0, y2 = 0;
        if (start && end) {
            top = Math.min(start.y, end.y);
            left = Math.min(start.x, end.x);
            width = Math.abs(start.x - end.x);
            height = Math.abs(start.y - end.y);
            x1 = end.x > start.x ? 0 : width;
            y1 = end.y > start.y ? 0 : height;
            x2 = end.x > start.x ? width : 0;
            y2 = end.y > start.y ? height : 0;
        }
        setDims({ top, left, width, height, x1, y1, x2, y2 });
    }, [fromId, toId, setDims]);

    /**
     * If dragId matches one of the edges, continually update the position of the edge
     */
    const dragRef = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        if (dragId && dragId === fromId || dragId === toId) {
            dragRef.current = setInterval(() => { calculateDims(); }, 25);
        }
        calculateDims();
        return () => {
            if (dragRef.current) clearInterval(dragRef.current);
        }
    }, [dragId, calculateDims, fromId, toId, scale]);


    return (
        <svg
            width={dims.width}
            // Extra height to make sure horizontal lines are shown
            height={dims.height + thiccness}
            style={{
                zIndex: -1, // Display behind nodes
                position: "absolute",
                pointerEvents: "none",
                top: dims.top - 15, // Not sure why top isn't right
                left: dims.left,
            }}
        >
            <line x1={dims.x1} y1={dims.y1} x2={dims.x2} y2={dims.y2} stroke="#9e3984" strokeWidth={thiccness} />
        </svg>
    )
}