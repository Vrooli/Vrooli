import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { NodeShape } from "./node";
import { RoutineVersionShape } from "./routineVersion";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NodeEndShape = Pick<NodeEnd, "id" | "wasSuccessful" | "suggestedNextRoutineVersions"> & {
    __typename: "NodeEnd";
    node: CanConnect<Omit<NodeShape, "end">>;
    suggestedNextRoutineVersions?: CanConnect<RoutineVersionShape>[] | null;
}

export const shapeNodeEnd: ShapeModel<NodeEndShape, NodeEndCreateInput, NodeEndUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "wasSuccessful"),
        ...createRel(d, "node", ["Connect"], "one"),
        ...createRel(d, "suggestedNextRoutineVersions", ["Connect"], "many"),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "wasSuccessful"),
        ...updateRel(o, u, "suggestedNextRoutineVersions", ["Connect", "Disconnect"], "many"),
    }),
};
