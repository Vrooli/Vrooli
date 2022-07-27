import { Node, NodeLink, Session } from 'types';
import { NodeContextMenuAction } from './NodeContextMenu/NodeContextMenu';

export interface NodeContextMenuProps {
    id: string;
    anchorEl: HTMLElement | null;
    handleClose: () => void;
    handleSelect: (option: NodeContextMenuAction) => void;
    zIndex: number;
}

export interface AddAfterLinkDialogProps {
    isOpen: boolean;
    handleClose: () => void;
    handleSelect: (selected: NodeLink) => void;
    nodeId: string;
    nodes: Node[];
    links: NodeLink[];
    session: Session;
    zIndex: number;
}

export interface AddBeforeLinkDialogProps {
    isOpen: boolean;
    handleClose: () => void;
    handleSelect: (selected: NodeLink) => void;
    nodeId: string;
    nodes: Node[];
    links: NodeLink[];
    session: Session;
    zIndex: number;
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
    handleAction: (action: BuildAction, nodeId: string, subroutineId?: string) => void;
    handleNodeDrop: (nodeId: string, columnIndex: number | null, rowIndex: number | null) => void;
    /**
     * Inserts a new routine list node along an edge
     */
    handleNodeInsert: (link: NodeLink) => void;
    /**
     * Inserts new branch
     */
    handleBranchInsert: (link: NodeLink) => void;
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
     * Updates scale
     */
    handleScaleChange: (delta: number) => void;
    language: string;
    /**
     * Dictionary of row and column pairs for every node ID on graph
     */
    nodesById: { [x: string]: Node };
    zIndex: number;
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
    handleAction: (action: BuildAction, nodeId: string, subroutineId?: string) => void;
    handleNodeUpdate: (updatedNode: Node) => void;
    language: string;
    zIndex: number;
}