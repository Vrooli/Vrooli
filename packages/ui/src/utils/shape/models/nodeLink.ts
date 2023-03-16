import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { NodeLinkWhenShape, createPrims, shapeNodeLinkWhen, shapeUpdate, updatePrims, updateRel, createRel } from "utils";

export type NodeLinkShape = Pick<NodeLink, 'id' | 'operation'> & {
    __typename?: 'NodeLink';
    from: { id: string };
    to: { id: string };
    routineVersion: { id: string };
    whens?: NodeLinkWhenShape[];
}

export const shapeNodeLink: ShapeModel<NodeLinkShape, NodeLinkCreateInput, NodeLinkUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'operation'),
        ...createRel(d, 'from', ['Connect'], 'one'),
        ...createRel(d, 'to', ['Connect'], 'one'),
        ...createRel(d, 'routineVersion', ['Connect'], 'one'),
        ...createRel(d, 'whens', ['Create'], 'many', shapeNodeLinkWhen),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'operation'),
        ...updateRel(o, u, 'from', ['Connect', 'Disconnect'], 'one'),
        ...updateRel(o, u, 'to', ['Connect', 'Disconnect'], 'one'),
        ...updateRel(o, u, 'whens', ['Create', 'Update', 'Delete'], 'many', shapeNodeLinkWhen),
    }, a)
}