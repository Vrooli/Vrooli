import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, hasObjectChanged, NodeRoutineListItemShape, shapeUpdate, updateRel } from "utils";

export type NodeRoutineListShape = Pick<NodeRoutineList, 'id' | 'isOptional' | 'isOrdered'> & {
    items: NodeRoutineListItemShape[];
}

export const shapeNodeRoutineList: ShapeModel<NodeRoutineListShape, NodeRoutineListCreateInput, NodeRoutineListUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'isOptional', 'isOrdered'),
        ...createRel(item, 'items', ['Create'], shapeNodeRoutineListItem),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isOptional', 'isOrdered'),
        ...updateRel(item, 'items', ['Create', 'Update', 'Delete'], hasObjectChanged, shapeNodeRoutineListItem),
    })
}