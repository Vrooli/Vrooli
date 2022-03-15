import { BoxProps } from '@mui/material';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node, NodeDataRoutineListItem } from "types";
import { BuildDialogOption } from 'utils';
import { MouseEvent } from 'react';

/**
 * Props for all nodes (besides the Add node)
 */
export interface NodeDataProps {
    node: Node;
}

/**
 * Props for all scalable objects
 */
export interface ScaleProps {
    scale: number;
}

/**
 * Props for all labelled node objects
 */
export interface LabelledProps {
    label?: string | null;
    labelVisible: boolean;
}

/**
 * Props for editable node objects
 */
export interface EditableProps {
    isEditing: boolean;
}

/**
 * Props for draggable node objects
 */
export interface DraggableProps {
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
export interface EndNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps, DraggableProps {
    handleContextItemSelect: (nodeId: string, option: NodeContextMenuOptions) => void;
}

/**
 * Props for the Loop node
 */
export interface LoopNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps, DraggableProps {

}

/**
 * Props for the Redirect node
 */
export interface RedirectNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps, DraggableProps {
    handleContextItemSelect: (nodeId: string, option: NodeContextMenuOptions) => void;
}

/**
 * Props for the Routine List node
 */
export interface RoutineListNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps, DraggableProps {
    canExpand: boolean;
    handleContextItemSelect: (nodeId: string, option: NodeContextMenuOptions) => void;
    handleNodeUnlink: (nodeId: string) => void;
    handleNodeDelete: (nodeId: string) => void;
    handleSubroutineOpen: (nodeId: string, subroutineId: string) => void;
    /**
     * Prompts parent to open a specific dialog
     */
    handleDialogOpen: (nodeId: string, dialog: BuildDialogOption) => void;
}

/**
 * Props for a Routine List's subroutine
 */
export interface RoutineSubnodeProps extends ScaleProps, LabelledProps, EditableProps {
    nodeId: string; //ID of parent node
    data: NodeDataRoutineListItem;
    handleSubroutineOpen: (nodeId: string, subroutineId: string) => void;
}

/**
 * Props for a Start node
 */
export interface StartNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps {

}

export interface DraggableNodeProps extends BoxProps, Omit<DraggableProps, 'isLinked'> {
    /**
     * ID of node in this cell. Used for drag events
     */
    nodeId: string;
    children: React.JSX;
}