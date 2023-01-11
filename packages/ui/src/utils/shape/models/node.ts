import { Node, NodeCreateInput, NodeTranslation, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, NodeEndShape, NodeRoutineListShape, shapeNodeEnd, shapeNodeRoutineList, shapeUpdate, updatePrims, updateRel } from "utils";

export type NodeTranslationShape = Pick<NodeTranslation, 'id' | 'language' | 'description' | 'name'>

export type NodeShape = Pick<Node, 'id' | 'columnIndex' | 'rowIndex' | 'nodeType'> & {
    // loop?: LoopShape | null
    end?: NodeEndShape | null;
    routineList?: NodeRoutineListShape | null;
    routineVersion: { id: string };
    translations: NodeTranslationShape[];
}

export const shapeNodeTranslation: ShapeModel<NodeTranslationShape, NodeTranslationCreateInput, NodeTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeNode: ShapeModel<NodeShape, NodeCreateInput, NodeUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'columnIndex', 'nodeType', 'rowIndex'),
        ...createRel(d, 'routineVersion', ['Connect'], 'one'),
        // ...createRel(d, 'loop', ['Create'], 'one', shapeLoop),
        ...createRel(d, 'end', ['Create'], 'one', shapeNodeEnd),
        ...createRel(d, 'routineList', ['Create'], 'one', shapeNodeRoutineList),
        ...createRel(d, 'translations', ['Create'], 'many', shapeNodeTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'columnIndex', 'nodeType', 'rowIndex'),
        ...updateRel(o, u, 'routineVersion', ['Connect'], 'one'),
        // ...updateRel(o, u, 'loop', ['Create', 'Update', 'Delete'], 'one', shapeLoop),
        ...updateRel(o, u, 'end', ['Update'], 'one', shapeNodeEnd),
        ...updateRel(o, u, 'routineList', ['Update'], 'one', shapeNodeRoutineList),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeNodeTranslation),
    })
}