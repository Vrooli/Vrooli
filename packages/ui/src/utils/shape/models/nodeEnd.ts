import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NodeEndShape = Pick<NodeEnd, 'id' | 'wasSuccessful' | 'suggestedNextRoutineVersions'> & {
    node: { id: string };
    suggestedNextRoutineVersions: { id: string }[];
}

export const shapeNodeEnd: ShapeModel<NodeEndShape, NodeEndCreateInput, NodeEndUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'wasSuccessful'),
        ...createRel(item, 'node', ['Connect'], 'one'),
        ...createRel(item, 'suggestedNextRoutineVersions', ['Connect'], 'many'),
    }),
    update: (o, u) => shapeUpdate(u, ({
        ...updatePrims(o, u, 'id', 'wasSuccessful'),
        ...updateRel(o, u, 'suggestedNextRoutineVersions', ['Connect', 'Disconnect'], 'many'),
    }))
}