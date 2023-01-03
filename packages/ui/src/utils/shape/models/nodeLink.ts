import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { NodeLinkWhenShape, createPrims, shapeNodeLinkWhen, shapeUpdate, updatePrims, updateRel, createRel } from "utils";

export type NodeLinkShape = Pick<NodeLink, 'id' | 'operation'> & {
    from: { id: string };
    to: { id: string };
    routineVersion: { id: string };
    whens?: NodeLinkWhenShape[];
}

export const shapeNodeLink: ShapeModel<NodeLinkShape, NodeLinkCreateInput, NodeLinkUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'operation'),
        ...createRel(item, 'from', ['Connect'], 'one'),
        ...createRel(item, 'to', ['Connect'], 'one'),
        ...createRel(item, 'routineVersion', ['Connect'], 'one'),
        ...createRel(item, 'whens', ['Create'], shapeNodeLinkWhen),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'operation'),
        ...updateRel(o, u, 'from', ['Connect', 'Disconnect'], 'one'),
        ...updateRel(o, u, 'to', ['Connect', 'Disconnect'], 'one'),
        ...updateRel(o, u, 'whens', ['Create', 'Update', 'Delete'], shapeNodeLinkWhen),
    })
}