import { Node, NodeCreateInput, NodeTranslation, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, NodeEndShape, NodeRoutineListShape, shapeUpdate, updatePrims, updateRel } from "utils";

export type NodeTranslationShape = Pick<NodeTranslation, 'id' | 'language' | 'description' | 'name'>

export type NodeShape = Pick<Node, 'id' | 'columnIndex' | 'rowIndex' | 'type'> & {
    // loop: LoopShape
    nodeEnd: NodeEndShape;
    nodeRoutineList: NodeRoutineListShape;
    routineVersionId: string;
    translations: NodeTranslationShape[];
}

export const shapeNodeTranslation: ShapeModel<NodeTranslationShape, NodeTranslationCreateInput, NodeTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeNode: ShapeModel<NodeShape, NodeCreateInput, NodeUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'columnIndex', 'rowIndex', 'type'),
        ...createRel(item, 'routineVersion', ['Connect'], 'one'),
        // ...createRel(item, 'loop', ['Create'], 'one', shapeLoop),
        ...createRel(item, 'nodeEnd', ['Create'], 'one', shapeNodeEnd),
        ...createRel(item, 'nodeRoutineList', ['Create'], 'one', shapeNodeRoutineList),
        ...createRel(item, 'translations', ['Create'], 'many', shapeNodeTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'columnIndex', 'rowIndex', 'type'),
        ...updateRel(o, u, 'routineVersion', ['Connect'], 'one'),
        // ...updateRel(o, u, 'loop', ['Create', 'Update', 'Delete'], 'one', shapeLoop),
        ...updateRel(o, u, 'nodeEnd', ['Update'], 'one', shapeNodeEnd),
        ...updateRel(o, u, 'nodeRoutineList', ['Update'], 'one', shapeNodeRoutineList),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeNodeTranslation),
    })
}