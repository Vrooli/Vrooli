import { BoxProps } from '@mui/material';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node, NodeLink } from 'types';
import { OrchestrationStatus } from 'utils';

/**
 * Describes the data and general position of a node in the graph. 
 * A completely linear graph would have all nodes at the same level (i.e. one per column, each at row 0).
 * A column with a decision node adds ONE row to the following column FOR EACH possible decision. 
 * A column with a combine node removes ONE row from the following column FOR EACH possible combination, AND 
 * places itself in the vertical center of each combined node
 */
export type NodePos = {
    column: number; // column in which node is displayed
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

export type OrchestrationStatusObject = {
    code: OrchestrationStatus,
    details: string,
}
export interface NodeGraphProps {
    scale?: number;
    isEditing?: boolean;
    labelVisible?: boolean;
    nodeDataMap: { [id: string]: NodePos };
    links: NodeLink[];
    /**
      * Prompts parent to open a specific dialog
      */
    handleDialogOpen: (nodeId: string, dialog: OrchestrationDialogOption) => void;
    /**
     * Moves a node to the unlinked container
     */
    handleNodeUnlink: (nodeId: string) => void;
    /**
     * Deletes a node permanently
     */
    handleNodeDelete: (nodeId: string) => void;
    /**
     * Updates a node's data
     */
    handleNodeUpdate: (node: Node) => void;
    /**
     * Create a link between two nodes
     */
    handleLinkCreate: (link: NodeLink) => void;
    /**
     * Updates a link between two nodes
     */
    handleLinkUpdate: (link: NodeLink, data: any) => void;
}

export type ColumnDimensions = {
    width: number; // Max width of node in column
    heights: number[]; // Height of each node in column, from top to bottom
    nodeIds: string[]; // Node IDs in column, from top to bottom
    tops: number[]; // Top y of each node in column, from top to bottom
    centers: number[]; // Center y of each node in column, from top to bottom
}

/**
 * Props for the Node Column (a container for displaying nodes on separate branches)
 */
export interface NodeColumnProps {
    id?: string;
    scale: number;
    isEditing: boolean;
    dragId: string | null; // ID of node being dragged. Used to display valid drop locations
    labelVisible: boolean;
    columnIndex: number;
    nodes: Node[];
    onDimensionsChange: (columnIndex: number, dimensions: ColumnDimensions) => void;
    /**
      * Prompts parent to open a specific dialog
      */
    handleDialogOpen: (nodeId: string, dialog: OrchestrationDialogOption) => void;
}

export interface NodeEdgeProps {
    link: NodeLink;
    handleAdd: (link: NodeLink, nodeType: NodeType) => void;
    handleEdit: (link: NodeLink) => void;
    isEditing?: boolean;
    isFromRoutineList: boolean; // If true, puts edit button further right
    isToRoutineList: boolean; // If true, puts edit button further left
    dragId: string | null; // ID of node being dragged. Used to determine if 
    scale: number; // Line thickness changes with scale
}