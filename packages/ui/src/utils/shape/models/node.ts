import { Node, NodeCreateInput, NodeTranslation, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { NodeEndShape, shapeNodeEnd } from "./nodeEnd";
import { NodeRoutineListShape, shapeNodeRoutineList } from "./nodeRoutineList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type NodeTranslationShape = Pick<NodeTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "NodeTranslation";
}

export type NodeShape = Pick<Node, "id" | "columnIndex" | "rowIndex" | "nodeType"> & {
    __typename: "Node";
    // loop?: LoopShape | null
    end?: NodeEndShape | null;
    routineList?: NodeRoutineListShape | null;
    routineVersion: { __typename: "RoutineVersion", id: string };
    translations: NodeTranslationShape[];
}

export const shapeNodeTranslation: ShapeModel<NodeTranslationShape, NodeTranslationCreateInput, NodeTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name"), a),
};

export const shapeNode: ShapeModel<NodeShape, NodeCreateInput, NodeUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "columnIndex", "nodeType", "rowIndex");
        return {
            ...prims,
            ...createRel(d, "routineVersion", ["Connect"], "one"),
            // ...createRel(d, "loop", ['Create'], "one", shapeLoop, (n) => ({ node: { id: prims.id }, ...n })),
            ...createRel(d, "end", ["Create"], "one", shapeNodeEnd, (n) => ({ node: { id: prims.id }, ...n })),
            ...createRel(d, "routineList", ["Create"], "one", shapeNodeRoutineList, (n) => ({ node: { id: prims.id }, ...n })),
            ...createRel(d, "translations", ["Create"], "many", shapeNodeTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "columnIndex", "nodeType", "rowIndex"),
        ...updateRel(o, u, "routineVersion", ["Connect"], "one"),
        // ...updateRel(o, u, "loop", ['Create', 'Update', 'Delete'], "one", shapeLoop, (n, i) => ({ node: { id: i.id }, ...n })),
        ...updateRel(o, u, "end", ["Create", "Update"], "one", shapeNodeEnd, (n, i) => ({ node: { id: i.id }, ...n })),
        ...updateRel(o, u, "routineList", ["Create", "Update"], "one", shapeNodeRoutineList, (n, i) => ({ node: { id: i.id }, ...n })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNodeTranslation),
    }, a),
};
