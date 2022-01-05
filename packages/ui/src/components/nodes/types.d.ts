import { NodeData, NodeType } from "@local/shared";
import { RoutineListNodeData, RoutineListNodeItemData } from '@local/shared';

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
export interface CombineNodeProps extends ScaleProps, LabelledProps, EditableProps {
    
}

/**
 * Props for the Decision node
 */
export interface DecisionNodeProps extends ScaleProps, LabelledProps, EditableProps {
    text?: string;
}

/**
 * Props for the End node
 */
export interface EndNodeProps extends ScaleProps, LabelledProps, EditableProps {

}

/**
 * Props for the Loop node
 */
export interface LoopNodeProps extends ScaleProps, LabelledProps, EditableProps {
}

/**
 * Props for the Node Column (a container for displaying nodes on separate branches)
 */
export interface NodeColumnProps extends ScaleProps, EditableProps {
    labelVisible?: boolean;
    columnNumber: number;
    nodes: NodeData[];
}

/**
 * Props for the Redirect node
 */
export interface RedirectNodeProps extends ScaleProps, LabelledProps, EditableProps {

}

/**
 * Props for the Routine List node
 */
export interface RoutineListNodeProps extends ScaleProps, LabelledProps, EditableProps {
    data?: RoutineListNodeData;
    onAdd: (data: RoutineListNodeItemData) => void;
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
export interface StartNodeProps extends ScaleProps, LabelledProps, EditableProps {
}