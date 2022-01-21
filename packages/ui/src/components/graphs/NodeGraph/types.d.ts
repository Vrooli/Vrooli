import { NodeData } from '@local/shared';
import { BoxProps } from '@mui/material';

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
    /**
     * Dimensions of the entire graph, for determining if a node has been dragged too far.
     */
    graphDimensions?: { width: number, height: number };
    /**
     * Top-left position of each cell. Used with dragPos to determine if a cell should be highlighted
     */
    cellPositions?: { x: number, y: number }[];
    /**
     * Data of current node being dragged, if any
     */
    dragData?: { x: number, y: number, type: NodeData['type'] };
    /**
     * Indicates if ANY node is currently being dragged, not necessarily in this column
     */
    isDragging: boolean;
    /**
     * Callback for the start of a node drag
     */
     onDragStart: (nodeId: string, position: { x: number, y: number }) => void;
    /**
     * Callback for dragging node
     */
    onDrag: (nodeId: string, position: { x: number, y: number }) => void;
    /**
     * Callback for dropped node
     */
    onDrop: (nodeId: string, position: { x: number, y: number }) => void;
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

export interface NodeGraphCellProps extends BoxProps {
    /**
     * Specified if the cell is allowed to be dragged
     */
    draggable?: boolean;
    /**
     * Specifies if the cell accepts drop events
     */
    droppable?: boolean;
    /**
     * Indicates if ANY node is currently being dragged, not necessarily in this cell
     */
     isDragging: boolean;
    /**
     * Specifies if a dragged node is over this cell
     */
    dragIsOver: boolean;
    /**
     * ID of node in this cell. Used for drag events
     */
    nodeId: string;
    /**
     * Callback for the start of a node drag
     */
     onDragStart: (nodeId: string, position: { x: number, y: number }) => void;
    /**
     * Callback for dragging node
     */
    onDrag: (nodeId: string, position: { x: number, y: number }) => void;
    /**
     * Callback for dropped node
     */
    onDrop: (nodeId: string, position: { x: number, y: number }) => void;
    /**
     * Callback for cell resize
     */
    onResize: (nodeId: string, dimensions: { width: number, height: number }) => void;
    /**
     * The node
     */
    children: React.JSX;
}