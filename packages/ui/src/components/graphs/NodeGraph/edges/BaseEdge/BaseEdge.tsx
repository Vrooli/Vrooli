import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { BaseEdgeProps, EdgePositions, Point } from '../types';
import { Box, Popover, Tooltip } from '@mui/material';

/**
 * Finds the coordinate of a point on a bezier curve
 * @param t The t value of the point (0-1)
 * @param sp The start point of the curve
 * @param cp1 The first control point
 * @param cp2 The second control point
 * @param ep The end point of the curve
 * @returns The y position of the point
 */
const getBezierPoint = (t: number, sp: Point, cp1: Point, cp2: Point, ep: Point): Point => {
    return {
        x: Math.pow(1 - t, 3) * sp.x + 3 * t * Math.pow(1 - t, 2) * cp1.x
            + 3 * t * t * (1 - t) * cp2.x + t * t * t * ep.x,
        y: Math.pow(1 - t, 3) * sp.y + 3 * t * Math.pow(1 - t, 2) * cp1.y
            + 3 * t * t * (1 - t) * cp2.y + t * t * t * ep.y
    };
}

/**
 * Creates a bezier curve between two points
 * @param p1 The first point
 * @param p2 The second point
 * @returns The bezier curve, as an array of {x, y} objects
 */
const createBezier = (p1: Point, p2: Point): [Point, Point, Point, Point] => {
    const midX = (p1.x + p2.x) / 2;
    // Calculate two horizontal lines for midpoints
    const mid1 = {
        x: midX,
        y: p1.y,
    }
    const mid2 = {
        x: midX,
        y: p2.y,
    }
    return [p1, mid1, mid2, p2];
}

/**
 * Calculates absolute position of one point of an edge.
 * Edges are displayed relative to the graph, but bounding boxes are calculated relative to the viewport.
 * We account for this using the graph's bounding box
 * @param containerId ID of the container which the point should be measured relative to
 * @param nodeId ID of the node that we are finding the position of
 * @returns x and y coordinates of the center point, as well as the start and end x
 */
const getPoint = (containerId: string, nodeId: string): { x: number, y: number, startX: number, endX: number } | null => {
    // Find graph and node
    const graph = document.getElementById(containerId);
    const node = document.getElementById(nodeId);
    if (!graph || !node) {
        console.error('Could not find node to connect to edge', nodeId);
        return null;
    }
    const nodeRect = node.getBoundingClientRect();
    const graphRect = graph.getBoundingClientRect();
    return {
        x: graph.scrollLeft + nodeRect.left + (nodeRect.width / 2) - graphRect.left,
        y: graph.scrollTop + nodeRect.top + (nodeRect.height / 2) - graphRect.top,
        startX: graph.scrollLeft + nodeRect.left - graphRect.left,
        endX: graph.scrollLeft + nodeRect.left + nodeRect.width - graphRect.left,
    }
}

/**
 * Displays a bezier line between any two UX components (probably have to be in the same DOM stack, but I never checked).
 * If in editing mode, displays a clickable button to edit the link or inserting a node
 */
export const BaseEdge = ({
    containerId,
    fromId,
    isEditing,
    popoverComponent,
    popoverT,
    thiccness,
    timeBetweenDraws,
    toId,
}: BaseEdgeProps) => {

    /**
     * Padding ensures that the line is doesn't get cut off
     */
    const padding = 50;
    // Store dimensions of edge
    const [dims, setDims] = useState<EdgePositions | null>(null);

    // Handle opening/closing of the edit popover menu
    const [anchorEl, setAnchorEl] = useState(null);
    const toggleEdit = useCallback((event) => {
        if (Boolean(anchorEl)) setAnchorEl(null);
        else setAnchorEl(event.currentTarget);
    }, [anchorEl]);
    const closeEdit = () => { setAnchorEl(null) };
    const isEditOpen = Boolean(anchorEl);

    /**
     * Calculate dimensions for edge svg.
     * Also calculate dimensions for popover display button.
     */
    const calculateDims = useCallback(() => {
        const fromPoint = getPoint(containerId, fromId);
        const toPoint = getPoint(containerId, toId);
        if (fromPoint && toPoint) {
            const top = Math.min(fromPoint.y, toPoint.y) - padding;
            const left = Math.min(fromPoint.x, toPoint.x) - padding;
            const width = Math.abs(fromPoint.x - toPoint.x) + (padding * 2);
            const height = Math.abs(fromPoint.y - toPoint.y) + (padding * 2);
            const from = {
                x: toPoint.x > fromPoint.x ? padding : width - padding,
                y: toPoint.y > fromPoint.y ? padding : height - padding,
            }
            const to = {
                x: toPoint.x > fromPoint.x ? width - padding : padding,
                y: toPoint.y > fromPoint.y ? height - padding : padding,
            }
            const fromEnd = (fromPoint?.endX ?? 0) - padding;
            const toStart = (toPoint?.startX ?? 0) + padding;
            let bezier = createBezier(from, to);
            setDims({ top, left, width, height, fromEnd, toStart, bezier });
        }
        else setDims(null);
    }, [containerId, fromId, toId]);

    /**
     * If dragId matches one of the edges, continually update the position of the edge
     */
    const intervalRef = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        intervalRef.current = setInterval(() => { calculateDims(); }, timeBetweenDraws);
        calculateDims();
        return () => {
            if (intervalRef.current) clearInterval(intervalRef.current);
        }
    }, [calculateDims, timeBetweenDraws]);

    /**
     * Edge component. Can either be straight or bezier
     */
    const edge = useMemo(() => {
        if (!dims) return null;
        return (
            <svg
                width={dims.width}
                height={dims.height}
                style={{
                    zIndex: 0, // Display behind nodes
                    position: "absolute",
                    pointerEvents: "none",
                    top: dims.top,
                    left: dims.left,
                }}
            >
                <path
                    d={`
                        M${dims.bezier[0].x},${dims.bezier[0].y} 
                        C${dims.bezier[1].x},${dims.bezier[1].y}
                        ${dims.bezier[2].x},${dims.bezier[2].y}
                        ${dims.bezier[3].x},${dims.bezier[3].y}
                        `}
                    fill="none"
                    stroke="#9e3984"
                    strokeWidth={thiccness / 10}
                />
            </svg>
        )
    }, [dims, thiccness])

    /**
     * If isEditable, displays a clickable button for displaying a popover menu
     */
    const popoverButton = useMemo(() => {
        if (!isEditing || !dims) return null;
        // If from and to are both routine lists (or both NOT routine lists), then use bezier midpoint.
        // If one is a routine list and the other is not, use a point further from the routine list
        const bezierPoint = getBezierPoint(popoverT ?? 0.5, ...dims.bezier);
        const diameter: number = isEditOpen ? thiccness : thiccness * 5 / 6
        return (
            <>
                <Tooltip title={isEditOpen ? '' : "Edit edge or insert node"}>
                    <Box onClick={toggleEdit} sx={{
                        position: "absolute",
                        top: dims.top + bezierPoint.y - (diameter / 2),
                        left: dims.left + bezierPoint.x - (diameter / 2),
                        height: `${diameter}px`,
                        width: `${diameter}px`,
                        borderRadius: '100%',
                        border: isEditOpen ? `2px solid #9e3984` : `none`,
                        cursor: 'pointer',
                        zIndex: 2,
                        background: isEditOpen ? 'transparent' : '#9e3984',
                        '&:hover': {
                            height: `${thiccness}px`,
                            width: `${thiccness}px`,
                            top: dims.top + bezierPoint.y - (thiccness / 2),
                            left: dims.left + bezierPoint.x - (thiccness / 2),
                        }
                    }} />
                </Tooltip >
                <Popover
                    open={isEditOpen}
                    anchorEl={anchorEl}
                    onClose={closeEdit}
                    onClick={closeEdit}
                    anchorOrigin={{
                        vertical: 'top',
                        horizontal: 'center',
                    }}
                    transformOrigin={{
                        vertical: 'bottom',
                        horizontal: 'center',
                    }}
                    sx={{
                        '& .MuiPopover-paper': {
                            background: 'transparent',
                            boxShadow: 'none',
                            border: 'none',
                            paddingBottom: 1,
                        }
                    }}
                >
                    {popoverComponent}
                </Popover>
            </>
        );
    }, [anchorEl, dims, isEditOpen, isEditing, popoverComponent, popoverT, thiccness, toggleEdit]);

    return (
        <>
            {edge}
            {Boolean(popoverComponent) && popoverButton}
        </>
    )
}