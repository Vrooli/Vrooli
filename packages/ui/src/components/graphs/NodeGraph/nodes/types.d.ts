import { NodeData, NodeType } from "@local/shared";
import { NodeData, RoutineListNodeItemData } from '@local/shared';
import { BoxProps } from '@mui/material';

/**
 * Props for all nodes (besides the Add node)
 */
export interface NodeDataProps {
    node: NodeData;
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
 * Props for the "Add Node" button (has node in its name, but not actually a node)
 */
export interface AddNodeProps extends ScaleProps, EditableProps {
    options: NodeType[];
    onAdd: (type: NodeType) => void;
}

/**
 * Props for the Combine node
 */
export interface CombineNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps, DraggableProps {

}

/**
 * Props for the Decision node
 */
export interface DecisionNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps, DraggableProps {
    text?: string;
}

/**
 * Props for the End node
 */
export interface EndNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps {

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
    onAdd: (data: RoutineListNodeItemData) => void;
    /**
     * Callback for cell resize
     */
     onResize: (nodeId: string, dimensions: { width: number, height: number }) => void;
}

/**
 * Props for a Routine List's subroutine
 */
export interface RoutineSubnodeProps extends ScaleProps, LabelledProps, EditableProps {
    data?: RoutineListNodeItemData;
}

/**
 * Props for a Start node
 */
export interface StartNodeProps extends NodeDataProps, ScaleProps, LabelledProps, EditableProps {

}

export interface DraggableNodeProps extends BoxProps {
    /**
     * Specified if the cell is allowed to be dragged
     */
    draggable?: boolean;
    /**
     * ID of node in this cell. Used for drag events
     */
    nodeId: string;
    children: React.JSX;
}