import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { NodeEdgeProps } from '../types';
import {
    Add as AddIcon,
    Edit as EditIcon,
    Delete as DeleteIcon,
} from '@mui/icons-material';
import { Box, IconButton, Popover, Stack, Tooltip } from '@mui/material';
import { NodeType } from 'graphql/generated/globalTypes';
import { ListMenuItemData } from 'components/dialogs/types';
import { ListMenu } from 'components';

type Point = {
    x: number;
    y: number;
}

type EdgePositions = {
    top: number;
    left: number;
    width: number;
    height: number;
    fromEnd: number;
    toStart: number;
    bezier: [Point, Point, Point, Point];
}

/**
 * Displays a line between two nodes.
 * If in editing mode, displays a clickable button to edit the link or inserting a node
 */
export const NodeEdge = ({
    handleAdd,
    handleDelete,
    handleEdit,
    isEditing,
    isFromRoutineList,
    isToRoutineList,
    link,
    dragId,
    scale,
}: NodeEdgeProps) => {
    const padding = 100; // Adds padding to make sure the line doesn't get cut off
    // Store dimensions of edge
    const [dims, setDims] = useState<EdgePositions | null>(null);
    const thiccness = useMemo(() => Math.ceil(scale * 3), [scale]);

    // Handle opening/closing of the edit popover menu
    const [anchorEl, setAnchorEl] = useState(null);
    const toggleEdit = useCallback((event) => {
        if (Boolean(anchorEl)) setAnchorEl(null);
        else setAnchorEl(event.currentTarget);
    }, []);
    const closeEdit = () => { setAnchorEl(null) };
    const isEditOpen = Boolean(anchorEl);

    // Triggers edit menu in parent. This is needed because the link's from and to nodes
    // can be updated, and the edge doesn't have this information
    const handleEditClick = useCallback(() => {
        handleEdit(link);
    }, [handleEdit, link]);

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
     * @returns x and y coordinates of the center point, as well as the start and end x
     */
    const getPoint = (id: string): { x: number, y: number, startX: number, endX: number } | null => {
        // Find graph and node
        const graph = document.getElementById('graph-root');
        const node = document.getElementById(`node-${id}`);
        if (!graph || !node) {
            console.error('Could not find node to connect to edge', id);
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
     * Calculate dimensions for edge svg.
     * Also calculate dimensions for "Add Node" button.
     */
    const calculateDims = useCallback(() => {
        const fromPoint = getPoint(link.fromId);
        const toPoint = getPoint(link.toId);
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
    }, [link.fromId, link.toId, setDims]);

    /**
     * If dragId matches one of the edges, continually update the position of the edge
     */
    const dragRef = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        // Update edge quickly when dragging, and slowly when not dragging
        // Updates are needed when not dragging to handle adding/removing nodes
        let delta = 1000; // Milliseconds
        if (dragId && dragId === link.fromId || dragId === link.toId) {
            delta = 15;
        }
        dragRef.current = setInterval(() => { calculateDims(); }, delta);
        calculateDims();
        return () => {
            if (dragRef.current) clearInterval(dragRef.current);
        }
    }, [dragId, calculateDims, link.fromId, link.toId, scale]);

    /**
     * Edge component
     */
    const edge = useMemo(() => {
        if (!dims) return null;
        return (
            <svg
                width={dims.width}
                height={dims.height}
                style={{
                    zIndex: -1, // Display behind nodes
                    position: "absolute",
                    pointerEvents: "none",
                    top: dims.top,
                    left: dims.left,
                }}
            >
                {/* Bezier curve line */}
                <path
                    d={`
                    M${dims.bezier[0].x},${dims.bezier[0].y} 
                    C${dims.bezier[1].x},${dims.bezier[1].y}
                    ${dims.bezier[2].x},${dims.bezier[2].y}
                    ${dims.bezier[3].x},${dims.bezier[3].y}
                `}
                    fill="none"
                    stroke="#9e3984"
                    strokeWidth={thiccness}
                />
                {/* Straight line */}
                {/* <line x1={dims.x1} y1={dims.y1} x2={dims.x2} y2={dims.y2} stroke="#9e3984" strokeWidth={thiccness} /> */}
            </svg>
        )
    }, [dims, thiccness])

    /**
     * If isEditable, displays a clickable button for editing the edge or inserting a node
     */
    const editButton = useMemo(() => {
        if (!isEditing || !dims) return null;
        // If from and to are both routine lists (or both NOT routine lists), then use bezier midpoint.
        // If one is a routine list and the other is not, use a point further from the routine list
        let t = 0.5;
        if (isFromRoutineList && !isToRoutineList) t = 0.77;
        else if (!isFromRoutineList && isToRoutineList) t = 0.23;
        const bezierPoint = getBezierPoint(t, ...dims.bezier);
        const diameter: number = isEditOpen ? scale * 30 : scale * 25
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
                            height: `${scale * 30}px`,
                            width: `${scale * 30}px`,
                            top: dims.top + bezierPoint.y - (scale * 30 / 2),
                            left: dims.left + bezierPoint.x - (scale * 30 / 2),
                        }
                    }} />
                </Tooltip >
                <Popover
                    open={isEditOpen}
                    anchorEl={anchorEl}
                    onClose={closeEdit}
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
                    <Stack direction="row" spacing={1}>
                        {/* Insert Node */}
                        <Tooltip title='Insert node'>
                            <IconButton
                                id="insert-node-on-edge-button"
                                size="small"
                                onClick={() => { handleAdd(link) }}
                                aria-label='Insert node on edge'
                                sx={{
                                    background: (t) => t.palette.secondary.main,
                                    transition: 'brightness 0.2s ease-in-out',
                                    '&:hover': {
                                        filter: `brightness(105%)`,
                                        background: (t) => t.palette.secondary.main,
                                    },
                                }}
                            >
                                <AddIcon id="insert-node-on-edge-button-icon" sx={{ fill: 'white' }} />
                            </IconButton>
                        </Tooltip>
                        {/* Edit Link */}
                        <Tooltip title='Edit link'>
                            <IconButton
                                id="edit-edge-button"
                                size="small"
                                onClick={handleEditClick}
                                aria-label='Edit link'
                                sx={{
                                    background: '#c5ab17',
                                    transition: 'brightness 0.2s ease-in-out',
                                    '&:hover': {
                                        filter: `brightness(105%)`,
                                        background: '#c5ab17',
                                    },
                                }}
                            >
                                <EditIcon id="insert-node-on-edge-button-icon" sx={{ fill: 'white' }} />
                            </IconButton>
                        </Tooltip>
                        {/* Delete link */}
                        <Tooltip title='Delete link'>
                            <IconButton
                                id="delete-link-on-edge-button"
                                size="small"
                                onClick={() => { handleDelete(link) }}
                                aria-label='Delete link button'
                                sx={{
                                    background: (t) => t.palette.error.main,
                                    transition: 'brightness 0.2s ease-in-out',
                                    '&:hover': {
                                        filter: `brightness(105%)`,
                                        background: (t) => t.palette.error.main,
                                    },
                                }}
                            >
                                <DeleteIcon id="delete-link-on-edge-button-icon" sx={{ fill: 'white' }} />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                </Popover>
            </>
        );
    }, [isEditOpen, isEditing, isFromRoutineList, isToRoutineList, dims, scale]);

    return (
        <>
            {edge}
            {editButton}
        </>
    )
}