import { Node, NodeEnd, NodeRoutineList } from "@local/shared";
import { FormProps } from "forms/types";
import { NodeShape } from "utils/shape/models/node";
import { NodeEndShape } from "utils/shape/models/nodeEnd";
import { NodeRoutineListShape } from "utils/shape/models/nodeRoutineList";
import { CrudProps } from "../types";

export type NodeWithEnd = Node & { end: NodeEnd };
export type NodeWithEndShape = NodeShape & { end: NodeEndShape };
export type NodeWithEndCrudProps = Omit<CrudProps<NodeWithEnd>, "isCreate" | "onCancel" | "onClose" | "onCompleted" | "overrideObject"> & Required<Pick<CrudProps<NodeWithEnd>, "onCancel" | "onClose" | "onCompleted" | "overrideObject">> & {
    isEditing: boolean;
    language: string;
}
export type NodeWithEndFormProps = Omit<FormProps<NodeWithEnd, NodeWithEndShape>, "onCancel" | "onClose" | "onCompleted"> & Pick<NodeWithEndCrudProps, "isEditing" | "language" | "onCancel" | "onClose" | "onCompleted">;

export type NodeWithRoutineList = Node & { routineList: NodeRoutineList };
export type NodeWithRoutineListShape = NodeShape & { routineList: NodeRoutineListShape };
export type NodeWithRoutineListCrudProps = Omit<CrudProps<NodeWithRoutineList>, "isCreate" | "onCancel" | "onClose" | "onCompleted" | "overrideObject"> & Required<Pick<CrudProps<NodeWithRoutineList>, "onCancel" | "onClose" | "onCompleted" | "overrideObject">> & {
    isEditing: boolean;
    language: string;
}
export type NodeWithRoutineListFormProps = Omit<FormProps<NodeWithRoutineList, NodeWithRoutineListShape>, "onCancel" | "onClose" | "onCompleted"> & Pick<NodeWithRoutineListCrudProps, "isEditing" | "language" | "onCancel" | "onClose" | "onCompleted">;
