import { BoxProps } from '@mui/material';
import { Node, NodeDataRoutineListItem, NodeEnd, NodeLink, NodeLoop, NodeRoutineList } from "types";
import { BuildAction } from 'utils';
import { MouseEvent } from 'react';

/**
 * Props for all scalable objects
 */
interface ScaleProps {
    scale: number;
}

/**
 * Props for all labelled node objects
 */
interface LabelledProps {
    label?: string | null;
    labelVisible: boolean;
}

/**
 * Props for editable node objects
 */
interface EditableProps {
    isEditing: boolean;
}

/**
 * Props for draggable node objects
 */
interface DraggableProps {
    isLinked: boolean; // True if node is connected to routine graph
    /**
    * Specified if the cell is allowed to be dragged
    */
    canDrag: boolean;
}

/**
 * Props for the "Add Node" button (has node in its name, but not actually a node)
 */
export interface AddNodeProps extends ScaleProps, EditableProps {
    options: NodeType[];
    onAdd: (type: NodeType) => void;
}

/**
 * Props for the End node
 */
export interface EndNodeProps extends ScaleProps, LabelledProps, EditableProps, DraggableProps {
    handleAction: (action: BuildAction, nodeId: string) => void;
    handleUpdate: (updatedNode: NodeEnd) => void; 
    language: string;
    linksIn: NodeLink[];
    node: NodeEnd;
    zIndex: number;
}

/**
 * Props for the Loop node
 */
export interface LoopNodeProps extends ScaleProps, LabelledProps, EditableProps, DraggableProps {
    node: NodeLoop;
    zIndex: number;
}

/**
 * Props for the Redirect node
 */
export interface RedirectNodeProps extends ScaleProps, LabelledProps, EditableProps, DraggableProps {
    handleAction: (action: NodeContextMenuAction, nodeId: string) => void;
    node: Node;// NodeRedirect;
    zIndex: number;
}

/**
 * Props for the Routine List node
 */
export interface RoutineListNodeProps extends ScaleProps, LabelledProps, EditableProps, DraggableProps {
    canExpand: boolean;
    handleAction: (action: BuildAction, nodeId: string, subroutineId?: string) => void;
    handleUpdate: (updatedNode: NodeRoutineList) => void; 
    language: string;
    linksIn: NodeLink[];
    linksOut: NodeLink[];
    node: NodeRoutineList;
    zIndex: number;
}

/**
 * Props for a Routine List's subroutine
 */
export interface SubroutineNodeProps extends ScaleProps, LabelledProps, EditableProps {
    data: NodeDataRoutineListItem;
    isOpen: boolean;
    handleAction: (action: BuildAction.OpenSubroutine | BuildAction.EditSubroutine | BuildAction.DeleteSubroutine, subroutineId: string) => void;
    handleUpdate: (subroutineId: string, updatedSubroutine: NodeDataRoutineListItem) => void; 
    language: string;
    zIndex: number;
}

/**
 * Props for a Start node
 */
export interface StartNodeProps extends ScaleProps, LabelledProps, EditableProps {
    handleAction: (action: BuildAction.AddOutgoingLink, subroutineId: string) => void;
    linksOut: NodeLink[];
    node: Node;
    zIndex: number;
}

export interface DraggableNodeProps extends BoxProps, Omit<DraggableProps, 'isLinked'> {
    /**
     * ID of node in this cell. Used for drag events
     */
    nodeId: string;
    children: React.JSX;
    /**
     * Threshold for dragging to start
     */
    dragThreshold?: number;
    /**
     * Callback when the node is clicked, but not dragged
     */
    onClick?: (event: MouseEvent) => void;
}