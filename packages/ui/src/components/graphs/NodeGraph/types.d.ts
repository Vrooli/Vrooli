import { NodeData } from '@local/shared';

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
    node: NodeData;
}

export interface NodeContextMenuProps {
    id: string;
    anchorEl: HTMLElement | null;
    node: NodeData;
    onClose: () => void;
    onAddBefore: (node: NodeData) => void;
    onAddAfter: (node: NodeData) => void;
    onEdit: (node: NodeData) => void;
    onDelete: (node: NodeData) => void;
    onMove: (node: NodeData) => void;
}

export interface NodeGraphProps {
    scale?: number;
    isEditable?: boolean;
    labelVisible?: boolean;
    nodes?: NodeData[]
}

/**
 * Props for the Node Column (a container for displaying nodes on separate branches)
 */
export interface NodeGraphColumnProps {
    id?: string;
    scale?: number;
    isEditable?: boolean;
    labelVisible?: boolean;
    columnNumber: number;
    nodes: NodeData[];
}

export interface NodeGraphEdgeProps {
    from: NodePos,
    to: NodePos,
    isEditable?: boolean;
    scale?: number,
    onAdd: any,
}