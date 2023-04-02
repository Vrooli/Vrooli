import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NodeEndShape = Pick<NodeEnd, 'id' | 'wasSuccessful' | 'suggestedNextRoutineVersions'> & {
    __typename?: 'NodeEnd';
    node: { __typename?: 'Node', id: string };
    suggestedNextRoutineVersions?: { id: string }[] | null;
}

export const shapeNodeEnd: ShapeModel<NodeEndShape, NodeEndCreateInput, NodeEndUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'wasSuccessful'),
        ...createRel(d, 'node', ['Connect'], 'one'),
        ...createRel(d, 'suggestedNextRoutineVersions', ['Connect'], 'many'),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'wasSuccessful'),
        ...updateRel(o, u, 'suggestedNextRoutineVersions', ['Connect', 'Disconnect'], 'many'),
    }, a)
}