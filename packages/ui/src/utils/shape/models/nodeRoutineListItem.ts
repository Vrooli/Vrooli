import { NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemTranslation, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput, NodeRoutineListItemUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, hasObjectChanged, RoutineVersionShape, shapeRoutineUpdate, shapeUpdate, updatePrims } from "utils";

export type NodeRoutineListItemTranslationShape = Pick<NodeRoutineListItemTranslation, 'id' | 'language' | 'description' | 'name'>

export type NodeRoutineListItemShape = Pick<NodeRoutineListItem, 'id' | 'index' | 'isOptional'> & {
    list: { id: string };
    routineVersion: RoutineVersionShape;
    translations: NodeRoutineListItemTranslationShape[];
}

export const shapeNodeRoutineListItemTranslation: ShapeModel<NodeRoutineListItemTranslationShape, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeNodeRoutineListItem: ShapeModel<NodeRoutineListItemShape, NodeRoutineListItemCreateInput, NodeRoutineListItemUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'index', 'isOptional'),
        listConnect: item.list.id,
        routineVersionConnect: item.routineVersion.id,
        ...shapeCreateList(item, 'translations', shapeNodeRoutineListItemTranslationCreate),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'index', 'isOptional'),
        routineUpdate: shapeRoutineUpdate(o.routineVersion, u.routineVersion),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeRoutineListItemTranslationCreate, shapeNodeRoutineListItemTranslationUpdate, 'id'),
    })
}