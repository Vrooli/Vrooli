import { BoxProps } from '@mui/material';
import { NodeType } from 'graphql/generated/globalTypes';
import { Node, NodeDataRoutineListItem } from "types";
import { OrchestrationDialogOption } from 'utils';

/**
 * Props for all nodes (besides the Add node)
 */
export interface NodeDataProps {
    node: Node;
}

/**
 * Props for all scalable objects (so every component involved with routine orchestration)
 */
export interface ScaleProps {
    scale?: number;
}

/**
 * Props for all labelled node objects
 */
export interface LabelledProps {
    label?: string | null;
    labelVisible?: boolean;
}

/**
 * Props for editable node objects
 */
export interface EditableProps {
    isEditable?: boolean;
}

/**
 * Props for draggable node objects
 */
export interface DraggableProps {
    /**
    * Specified if the cell is allowed to be dragged
    */
    canDrag?: boolean;
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

}

/**
 * Props for the Routine List node
 */
export interface RoutineListNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps, DraggableProps {
    canExpand?: boolean;
    onAdd: (data: NodeDataRoutineListItem) => void;
    /**
     * Callback for cell resize
     */
    onResize: (nodeId: string, height: number) => void;
    /**
     * Prompts parent to open a specific dialog
     */
    handleDialogOpen: (nodeId: string, dialog: OrchestrationDialogOption) => void;
}

/**
 * Props for a Routine List's subroutine
 */
export interface RoutineSubnodeProps extends ScaleProps, LabelledProps, EditableProps {
    nodeId: string; //ID of parent node
    data?: NodeDataRoutineListItem;
    /**
      * Prompts parent to open a specific dialog
      */
    handleDialogOpen: (nodeId: string, dialog: OrchestrationDialogOption) => void;
}

/**
 * Props for a Start node
 */
export interface StartNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps {

}

export interface DraggableNodeProps extends BoxProps, DraggableProps {
    /**
     * ID of node in this cell. Used for drag events
     */
    nodeId: string;
    children: React.JSX;
}