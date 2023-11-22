import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { NodeLinkWhenShape, shapeNodeLinkWhen } from "./nodeLinkWhen";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NodeLinkShape = Pick<NodeLink, "id" | "operation"> & {
    __typename?: "NodeLink";
    from: { id: string };
    to: { id: string };
    routineVersion: { __typename?: "RoutineVersion", id: string };
    whens?: NodeLinkWhenShape[];
}

export const shapeNodeLink: ShapeModel<NodeLinkShape, NodeLinkCreateInput, NodeLinkUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "operation"),
        ...createRel(d, "from", ["Connect"], "one"),
        ...createRel(d, "to", ["Connect"], "one"),
        ...createRel(d, "routineVersion", ["Connect"], "one"),
        ...createRel(d, "whens", ["Create"], "many", shapeNodeLinkWhen),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "operation"),
        ...updateRel(o, u, "from", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "to", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "whens", ["Create", "Update", "Delete"], "many", shapeNodeLinkWhen),
    }, a),
};
