import { NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemTranslation, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput, NodeRoutineListItemUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { NodeRoutineListShape } from "./nodeRoutineList";
import { RoutineVersionShape, shapeRoutineVersion } from "./routineVersion";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type NodeRoutineListItemTranslationShape = Pick<NodeRoutineListItemTranslation, "id" | "language" | "description" | "name"> & {
    __typename?: "NodeRoutineListItemTranslation";
}

export type NodeRoutineListItemShape = Pick<NodeRoutineListItem, "id" | "index" | "isOptional"> & {
    __typename: "NodeRoutineListItem";
    list: CanConnect<NodeRoutineListShape>;
    routineVersion: RoutineVersionShape;
    translations: NodeRoutineListItemTranslationShape[];
}

export const shapeNodeRoutineListItemTranslation: ShapeModel<NodeRoutineListItemTranslationShape, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "name")),
};

export const shapeNodeRoutineListItem: ShapeModel<NodeRoutineListItemShape, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "index", "isOptional"),
        ...createRel(d, "list", ["Connect"], "one"),
        ...createRel(d, "routineVersion", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeNodeRoutineListItemTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "index", "isOptional"),
        ...updateRel(o, u, "routineVersion", ["Update"], "one", shapeRoutineVersion),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNodeRoutineListItemTranslation),
    }),
};
