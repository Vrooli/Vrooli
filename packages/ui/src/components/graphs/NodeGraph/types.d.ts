import { BoxProps } from '@mui/material';
import { Node, NodeLink } from 'types';

/**
 * Describes the data and general position of a node in the graph. 
 * A completely linear graph would have all nodes at the same level (i.e. one per column, each at row 0).
 * A column with a decision node adds ONE row to the following column FOR EACH possible decision. 
 * A column with a combine node removes ONE row from the following column FOR EACH possible combination, AND 
 * places itself in the vertical center of each combined node
 */
export type NodePos = {
    column: number; // column in which node is displayed
    // rows: number; // number of rows in the column the node is displayed in
    // pos: number; // relative position in column. 0 is top, 1 is bottom
    node: Node;
}

export interface NodeContextMenuProps {
    id: string;
    anchorEl: HTMLElement | null;
    node: Node;
    onClose: () => void;
    onAddBefore: (node: Node) => void;
    onAddAfter: (node: Node) => void;
    onEdit: (node: Node) => void;
    onDelete: (node: Node) => void;
    onMove: (node: Node) => void;
}

export interface NodeGraphProps {
    scale?: number;
    isEditable?: boolean;
    labelVisible?: boolean;
    nodes: Node[]
    links: NodeLink[];
}

/**
 * Props for the Node Column (a container for displaying nodes on separate branches)
 */
export interface NodeGraphColumnProps {
    id?: string;
    scale?: number;
    isEditable?: boolean;
    dragId: string | null; // ID of node being dragged. Used to display valid drop locations
    labelVisible?: boolean;
    columnNumber: number;
    nodes: Node[];
    /**
    * Callback for cell resize
    */
    onResize: (nodeId: string, dimensions: { width: number, height: number }) => void;
}

export interface NodeGraphEdgeProps {
    start: { x: number, y: number };
    end: { x: number, y: number };
    isEditable?: boolean;
    onAdd: any,
}