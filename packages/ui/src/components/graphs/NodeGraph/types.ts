/* c8 ignore start */
import { RoutineVersion } from "@local/shared";
import { BuildAction } from "../../../utils/consts.js";

type Node = any;
type NodeLink = any;

export interface AddAfterLinkDialogProps {
    isOpen: boolean;
    handleClose: () => unknown;
    handleSelect: (selected: NodeLink) => unknown;
    nodeId: string;
    nodes: Node[];
    links: NodeLink[];
}

export interface AddBeforeLinkDialogProps {
    isOpen: boolean;
    handleClose: () => unknown;
    handleSelect: (selected: NodeLink) => unknown;
    nodeId: string;
    nodes: Node[];
    links: NodeLink[];
}

export interface GraphActionsProps {
    canRedo: boolean;
    canUndo: boolean;
    handleCleanUpGraph: () => unknown;
    handleNodeDelete: (nodeId: string) => unknown;
    handleOpenLinkDialog: () => unknown;
    handleRedo: () => unknown;
    handleUndo: () => unknown;
    isEditing: boolean;
    language: string;
    nodesOffGraph: Node[];
}

export interface MoveNodeMenuProps {
    handleClose: (newPosition?: { columnIndex: number, rowIndex: number }) => unknown;
    isOpen: boolean;
    language: string; // Language to display/edit
    node?: Node | null; // Node to be moved
    routineVersion: RoutineVersion;
}

export interface NodeGraphProps {
    /** 2D array of nodes, by column then row */
    columns: Node[][];
    isEditing?: boolean;
    labelVisible?: boolean;
    language: string; // Language to display/edit
    links: NodeLink[];
    handleAction: (action: BuildAction, nodeId: string, subroutineId?: string) => unknown;
    handleNodeDrop: (nodeId: string, columnIndex: number | null, rowIndex: number | null) => unknown;
    /** Inserts a new routine list node along an edge */
    handleNodeInsert: (link: NodeLink) => unknown;
    /** Inserts new branch */
    handleBranchInsert: (link: NodeLink) => unknown;
    /** Updates a node's data */
    handleNodeUpdate: (node: Node) => unknown;
    /** Create a link between two nodes */
    handleLinkCreate: (link: NodeLink) => unknown;
    /** Updates a link between two nodes */
    handleLinkUpdate: (link: NodeLink, data: any) => unknown;
    /** Delete a link between two nodes */
    handleLinkDelete: (link: NodeLink) => unknown;
    handleScaleChange: (delta: number) => unknown;
    /** Dictionary of row and column pairs for every node ID on graph */
    nodesById: { [x: string]: Node };
    scale: number;
}
