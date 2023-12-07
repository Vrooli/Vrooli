import { Node, NodeLink, NodeLoop, NodeRoutineListItem, NodeType } from "@local/shared";
import { BoxProps } from "@mui/material";
import { MouseEvent, ReactNode } from "react";
import { BuildAction } from "utils/consts";
import { NodeShape } from "utils/shape/models/node";
import { NodeWithEndShape, NodeWithRoutineListShape } from "views/objects/node/types";

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
    onAdd: (nodeType: NodeType) => unknown;
}

/**
 * Props for the End node
 */
export interface EndNodeProps extends ScaleProps, LabelledProps, EditableProps, DraggableProps {
    handleAction: (action: BuildAction, nodeId: string) => unknown;
    handleDelete: (node: NodeShape) => unknown;
    handleUpdate: (updatedNode: NodeWithEndShape) => unknown;
    language: string;
    linksIn: NodeLink[];
    node: NodeWithEndShape;
}

/**
 * Props for the Loop node
 */
export interface LoopNodeProps extends ScaleProps, LabelledProps, EditableProps, DraggableProps {
    node: Node & { loop: NodeLoop };
}

/**
 * Props for the Redirect node
 */
export interface RedirectNodeProps extends ScaleProps, LabelledProps, EditableProps, DraggableProps {
    handleAction: (action: BuildAction, nodeId: string) => unknown;
    node: NodeShape;// & { redirect: NodeRedirectShape }; TODO
}

/**
 * Props for the Routine List node
 */
export interface RoutineListNodeProps extends ScaleProps, LabelledProps, EditableProps, DraggableProps {
    canExpand: boolean;
    handleAction: (action: BuildAction, nodeId: string, subroutineId?: string) => unknown;
    handleDelete: (node: NodeShape) => unknown;
    handleUpdate: (updatedNode: NodeWithRoutineListShape) => unknown;
    language: string;
    linksIn: NodeLink[];
    linksOut: NodeLink[];
    node: NodeWithRoutineListShape;
}

/**
 * Props for a Routine List's subroutine
 */
export interface SubroutineNodeProps extends ScaleProps, LabelledProps, EditableProps {
    data: NodeRoutineListItem;
    isOpen: boolean;
    handleAction: (action: BuildAction.OpenSubroutine | BuildAction.EditSubroutine | BuildAction.DeleteSubroutine, subroutineId: string) => unknown;
    handleUpdate: (subroutineId: string, updatedItem: NodeRoutineListItem) => unknown;
    language: string;
}

/**
 * Props for a Start node
 */
export interface StartNodeProps extends ScaleProps, LabelledProps, EditableProps {
    handleAction: (action: BuildAction.AddOutgoingLink, subroutineId: string) => unknown;
    linksOut: NodeLink[];
    node: NodeShape;
}

export interface DraggableNodeProps extends BoxProps, Omit<DraggableProps, "isLinked"> {
    /** ID of node in this cell. Used for drag events */
    nodeId: string;
    children: ReactNode;
    /** Threshold for dragging to start */
    dragThreshold?: number;
    /** Callback when the node is clicked, but not dragged */
    onClick?: (event: MouseEvent) => unknown;
}
