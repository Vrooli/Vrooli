import { NodeLink } from "@local/shared";

export type Point = {
    x: number;
    y: number;
}

export type EdgePositions = {
    top: number;
    left: number;
    width: number;
    height: number;
    fromEnd: number;
    toStart: number;
    bezier: [Point, Point, Point, Point];
}

export interface BaseEdgeProps {
    /** ID of the container which points should be drawn from */
    containerId: string;
    /** ID of the first DOM node. */
    fromId: string;
    /** Determines if editing popup can be displayed. */
    isEditing: boolean;
    /** Component to display when popover is open */
    popoverComponent?: JSX.Element;
    /**
     * Time in bezier curve display popover button, from 0 to 1. 
     * Defaults to midpoint of edge (0.5).
     */
    popoverT?: number;
    /** How thick to draw the line. */
    thiccness: number;
    /** Milliseconds between each draw of the edge. */
    timeBetweenDraws: number;
    /** ID of the second DOM node. */
    toId: string;
}

export interface NodeEdgeProps {
    /**
     * True if edges should be updated quickly (e.g. moving node, collapsing node, deleting node)
     */
    fastUpdate: boolean;
    link: NodeLink;
    /**
     * Adding a node always creates a routine list node. 
     * Other node types are created automatically in other places.
     */
    handleAdd: (link: NodeLink) => void;
    /**
     * Creates a new node in the same column as the "to" node. 
     * This creates a new branch
     */
    handleBranch: (link: NodeLink) => void;
    /**
     * Deletes a link and its conditions. 
     * Does not delete any nodes
     */
    handleDelete: (link: NodeLink) => void;
    handleEdit: (link: NodeLink) => void;
    /** Determines if editing popup can be displayed. */
    isEditing: boolean;
    /** If true, puts edit button further to the right */
    isFromRoutineList: boolean;
    /** If true, puts edit button further to the left */
    isToRoutineList: boolean;
    /** Line thickness changes with scale */
    scale: number;
}
