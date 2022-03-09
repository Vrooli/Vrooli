import { BoxProps } from '@mui/material';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node, NodeLink } from 'types';
import { BuildDialogOption, BuildStatus } from 'utils';

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

export type BuildStatusObject = {
    code: BuildStatus,
    messages: string[],
}
export interface NodeGraphProps {
    /**
     * 2D array of nodes, by column then row
     */
    columns: Node[][];
    scale?: number;
    isEditing?: boolean;
    labelVisible?: boolean;
    language: string; // Language to display/edit
    links: NodeLink[];
    /**
      * Prompts parent to open a specific dialog
      */
    handleDialogOpen: (nodeId: string, dialog: BuildDialogOption) => void;
    /**
     * Moves a node to the unlinked container
     */
    handleNodeUnlink: (nodeId: string) => void;
    /**
     * Deletes a node permanently
     */
    handleNodeDelete: (nodeId: string) => void;
    /**
     * Inserts a new routine list node along an edge
     */
    handleNodeInsert: (link: NodeLink) => void;
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
    /**
     * Delete a link between two nodes
     */
    handleLinkDelete: (link: NodeLink) => void;
    /**
     * Adds a routine item to a routine list
     */
    handleRoutineListItemAdd: (nodeId: string, data: NodeDataRoutineListItem) => void;
    /**
     * Dictionary of row and column pairs for every node ID on graph
     */
    nodesById: { [x: string]: Node };
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
    /**
      * Prompts parent to open a specific dialog
      */
    handleDialogOpen: (nodeId: string, dialog: BuildDialogOption) => void;
    handleNodeDelete: (nodeId: string) => void;
    handleNodeUnlink: (nodeId: string) => void;
    handleRoutineListItemAdd: (nodeId: string, data: NodeDataRoutineListItem) => void;
}

export interface NodeEdgeProps {
    link: NodeLink;
    handleAdd: (link: NodeLink) => void; // Adding a node always creates a routine list node. Other node types are automatically added
    handleDelete: (link: NodeLink) => void;
    handleEdit: (link: NodeLink) => void;
    isEditing?: boolean;
    isFromRoutineList: boolean; // If true, puts edit button further right
    isToRoutineList: boolean; // If true, puts edit button further left
    dragId: string | null; // ID of node being dragged. Used to determine if 
    scale: number; // Line thickness changes with scale
}