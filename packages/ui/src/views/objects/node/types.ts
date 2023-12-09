import { Node, NodeEnd, NodeRoutineList } from "@local/shared";
import { FormProps } from "forms/types";
import { NodeShape } from "utils/shape/models/node";
import { NodeEndShape } from "utils/shape/models/nodeEnd";
import { NodeRoutineListShape } from "utils/shape/models/nodeRoutineList";
import { CrudProps } from "../types";

export type NodeWithEnd = Node & { end: NodeEnd };
export type NodeWithEndShape = NodeShape & { end: NodeEndShape };
export type NodeWithEndCrudProps = CrudProps<NodeWithEnd> & {
    isEditing: boolean;
    language: string;
}
export type NodeWithEndFormProps = FormProps<NodeWithEnd, NodeWithEndShape> & Pick<NodeWithEndCrudProps, "isEditing" | "language">;

export type NodeWithRoutineList = Node & { routineList: NodeRoutineList };
export type NodeWithRoutineListShape = NodeShape & { routineList: NodeRoutineListShape };
export type NodeWithRoutineListCrudProps = CrudProps<NodeWithRoutineList> & {
    isEditing: boolean;
    language: string;
}
export type NodeWithRoutineListFormProps = FormProps<NodeWithRoutineList, NodeWithRoutineListShape> & Pick<NodeWithRoutineListCrudProps, "isEditing" | "language">;
