import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { NodeEdgeProps } from '../types';
import {
    Add as AddIcon
} from '@mui/icons-material';
import { IconButton, Tooltip } from '@mui/material';
import { NodeType } from 'graphql/generated/globalTypes';
import { ListMenuItemData } from 'components/dialogs/types';
import { ListMenu } from 'components';

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

// NOTE: Since adding along an edge means that there will always be a node 
// after the new node, an "End" node cannot be selected
const addListOptionsMap: Array<[NodeType, string, string]> = [
    [NodeType.Loop, 'Loop', 'Repeat a section of the orchestration'],
    [NodeType.Redirect, 'Redirect', 'Redirect to an earlier point or another branch'],
    [NodeType.RoutineList, 'Routine List', 'Add one or more subroutines'],
]

const addListOptions: ListMenuItemData<NodeType>[] = addListOptionsMap.map(o => ({
    value: o[0],
    label: o[1],
    helpData: { markdown: o[2] },
}));

// Displays a line between two nodes.
// If in editing mode, an "Add Node" button appears on the line. 
// This button always appears inbetween two node columns, to avoid collisions with nodes.
export const NodeEdge = ({
    fromId,
    toId,
    handleAdd,
    isEditing,
    dragId,
    scale,
}: NodeEdgeProps) => {
    // Store dimensions of edge
    const [dims, setDims] = useState<EdgePositions>({ top: 0, left: 0, width: 0, height: 0, x1: 0, y1: 0, x2: 0, y2: 0 });
    // Store dimensions required to position the "Add Node" button (i.e. right edge of "from" node and left edge of "to" node)
    const [addDims, setAddDims] = useState<{fromEnd: number, toStart: number}>({ fromEnd: 0, toStart: 0 });
    const thiccness = useMemo(() => Math.ceil(scale * 3), [scale]);

    // Add node menu
    const [anchorEl, setAnchorEl] = useState<any>(null);
    const openAddMenu = useCallback((ev: any) => {
        setAnchorEl(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeAddMenu = useCallback(() => {
        setAnchorEl(null);
    }, []);
    const handleAddSelect = useCallback((selected: NodeType) => {
        closeAddMenu();
        handleAdd(fromId, toId, selected);
    }, [closeAddMenu, handleAdd, fromId, toId]);

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
        const fromPoint = getPoint(fromId);
        const toPoint = getPoint(toId);
        let top = 0, left = 0, width = 0, height = 0, x1 = 0, y1 = 0, x2 = 0, y2 = 0;
        if (fromPoint && toPoint) {
            top = Math.min(fromPoint.y, toPoint.y);
            left = Math.min(fromPoint.x, toPoint.x);
            width = Math.abs(fromPoint.x - toPoint.x);
            height = Math.abs(fromPoint.y - toPoint.y);
            x1 = toPoint.x > fromPoint.x ? 0 : width;
            y1 = toPoint.y > fromPoint.y ? 0 : height;
            x2 = toPoint.x > fromPoint.x ? width : 0;
            y2 = toPoint.y > fromPoint.y ? height : 0;
        }
        setDims({ top, left, width, height, x1, y1, x2, y2 });
        setAddDims({ fromEnd: fromPoint?.endX ?? 0, toStart: toPoint?.startX ?? 0 });
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

    /**
     * Edge component
     */
    const edge = useMemo(() => (
        <svg
            width={dims.width}
            // Extra height to make sure horizontal lines are shown
            height={dims.height + thiccness}
            style={{
                zIndex: -1, // Display behind nodes
                position: "absolute",
                pointerEvents: "none",
                top: dims.top,
                left: dims.left,
            }}
        >
            <line x1={dims.x1} y1={dims.y1} x2={dims.x2} y2={dims.y2} stroke="#9e3984" strokeWidth={thiccness} />
        </svg>
    ), [dims, thiccness])

    /**
     * If isEditable, display the "Add Node" button at the midpoint of the edge
     */
    const addButton = useMemo(() => {
        if (!isEditing) return null;
        return (
            <Tooltip title='Add node here'>
                <IconButton
                    id="add-node-on-edge-button"
                    onClick={openAddMenu}
                    aria-label='Add node on edge'
                    sx={{
                        position: "absolute",
                        top: dims.top + (dims.height / 2) - (scale * 35 / 2),
                        left: (addDims.fromEnd + addDims.toStart) / 2 - (scale * 35 / 2),
                        width: `${scale * 35}px`,
                        height: `${scale * 35}px`,
                        cursor: 'pointer',
                        zIndex: 2,
                        background: (t) => t.palette.secondary.main,
                        marginLeft: 'auto',
                        marginRight: 1,
                        transition: 'brightness 0.2s ease-in-out',
                        '&:hover': {
                            filter: `brightness(105%)`,
                            background: (t) => t.palette.secondary.main,
                        },
                    }}
                >
                    <AddIcon id="add-node-on-edge-button-icon" sx={{ fill: 'white' }} />
                </IconButton>
            </Tooltip>
        )
    }, [isEditing, addDims, dims]);

    /**
     * Menu for adding a new node
     */
    const addMenu = useMemo(() => {
        if (!isEditing) return null;
        return (
            <ListMenu
                id={'add-node-on-edge-menu'}
                anchorEl={anchorEl}
                title='Resource Options'
                data={addListOptions}
                onSelect={handleAddSelect}
                onClose={closeAddMenu}
            />
        )
    }, [anchorEl, handleAddSelect, isEditing, closeAddMenu]);


    return (
        <>
            {edge}
            {addButton}
            {addMenu}
        </>
    )
}