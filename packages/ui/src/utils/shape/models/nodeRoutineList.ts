import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, NodeRoutineListItemShape, shapeNodeRoutineListItem, shapeUpdate, updatePrims, updateRel } from "utils";

export type NodeRoutineListShape = Pick<NodeRoutineList, 'id' | 'isOptional' | 'isOrdered'> & {
    items: NodeRoutineListItemShape[];
    node: { id: string };
}

export const shapeNodeRoutineList: ShapeModel<NodeRoutineListShape, NodeRoutineListCreateInput, NodeRoutineListUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isOptional', 'isOrdered'),
        ...createRel(d, 'items', ['Create'], 'many', shapeNodeRoutineListItem),
        ...createRel(d, 'node', ['Connect'], 'one'),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isOptional', 'isOrdered'),
        ...updateRel(o, u, 'items', ['Create', 'Update', 'Delete'], 'many', shapeNodeRoutineListItem),
    })
}