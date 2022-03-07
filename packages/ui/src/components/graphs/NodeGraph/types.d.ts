import { BoxProps } from '@mui/material';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node, NodeLink } from 'types';
import { OrchestrationStatus } from 'utils';

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
    messages: string[],
}
export interface NodeGraphProps {
    scale?: number;
    isEditing?: boolean;
    labelVisible?: boolean;
    language: string; // Language to display/edit
    nodes: Node[];
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
    handleAdd: (link: NodeLink) => void; // Adding a node always creates a routine list node. Other node types are automatically added
    handleEdit: (link: NodeLink) => void;
    isEditing?: boolean;
    isFromRoutineList: boolean; // If true, puts edit button further right
    isToRoutineList: boolean; // If true, puts edit button further left
    dragId: string | null; // ID of node being dragged. Used to determine if 
    scale: number; // Line thickness changes with scale
}