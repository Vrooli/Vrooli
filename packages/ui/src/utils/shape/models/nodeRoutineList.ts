import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { NodeRoutineListItemShape, shapeNodeRoutineListItem } from "./nodeRoutineListItem";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NodeRoutineListShape = Pick<NodeRoutineList, 'id' | 'isOptional' | 'isOrdered'> & {
    __typename?: 'NodeRoutineList';
    items: NodeRoutineListItemShape[];
    node: { id: string };
}

export const shapeNodeRoutineList: ShapeModel<NodeRoutineListShape, NodeRoutineListCreateInput, NodeRoutineListUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isOptional', 'isOrdered'),
        ...createRel(d, 'items', ['Create'], 'many', shapeNodeRoutineListItem),
        ...createRel(d, 'node', ['Connect'], 'one'),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isOptional', 'isOrdered'),
        ...updateRel(o, u, 'items', ['Create', 'Update', 'Delete'], 'many', shapeNodeRoutineListItem),
    }, a)
}