import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { NodeShape } from "./node";
import { NodeRoutineListItemShape, shapeNodeRoutineListItem } from "./nodeRoutineListItem";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NodeRoutineListShape = Pick<NodeRoutineList, "id" | "isOptional" | "isOrdered"> & {
    __typename: "NodeRoutineList";
    items: NodeRoutineListItemShape[];
    node: CanConnect<Omit<NodeShape, "routineList">>;
}

export const shapeNodeRoutineList: ShapeModel<NodeRoutineListShape, NodeRoutineListCreateInput, NodeRoutineListUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isOptional", "isOrdered");
        return {
            ...prims,
            ...createRel(d, "items", ["Create"], "many", shapeNodeRoutineListItem, (r) => ({ list: { id: prims.id }, ...r })),
            ...createRel(d, "node", ["Connect"], "one"),
        };
    },
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isOptional", "isOrdered"),
        ...updateRel(o, u, "items", ["Create", "Update", "Delete"], "many", shapeNodeRoutineListItem, (r, i) => ({ list: { id: i.id }, ...r })),
    }),
};
