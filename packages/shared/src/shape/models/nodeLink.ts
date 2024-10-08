import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { NodeShape } from "./node";
import { NodeLinkWhenShape, shapeNodeLinkWhen } from "./nodeLinkWhen";
import { RoutineVersionShape } from "./routineVersion";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NodeLinkShape = Pick<NodeLink, "id" | "operation"> & {
    __typename: "NodeLink";
    from: CanConnect<NodeShape>;
    to: CanConnect<NodeShape>;
    routineVersion: CanConnect<RoutineVersionShape>;
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
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "operation"),
        ...updateRel(o, u, "from", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "to", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "whens", ["Create", "Update", "Delete"], "many", shapeNodeLinkWhen),
    }),
};
