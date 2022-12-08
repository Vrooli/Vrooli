import { NodeCreateInput, NodeEndCreateInput, NodeEndUpdateInput, NodeRoutineListCreateInput, NodeRoutineListItemCreateInput, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput, NodeRoutineListItemUpdateInput, NodeRoutineListUpdateInput, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput } from "graphql/generated/globalTypes";
import { Node, NodeDataEnd, NodeDataRoutineList, NodeDataRoutineListItem, NodeDataRoutineListItemTranslation, NodeTranslation, ShapeWrapper } from "types";
import { hasObjectChanged, RoutineShape, shapeRoutineUpdate } from "utils";
import { shapeCreateList, shapePrim, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type NodeEndShape = ShapeWrapper<NodeDataEnd> & {
    id: string;
}

export type NodeRoutineListItemTranslationShape = Omit<ShapeWrapper<NodeDataRoutineListItemTranslation>, 'language'> & {
    id: string;
    language: NodeRoutineListItemTranslationCreateInput['language'];
}

export type NodeRoutineListItemShape = Omit<ShapeWrapper<NodeDataRoutineListItem>, 'index' | 'routine'> & {
    id: string;
    index: NodeRoutineListItemCreateInput['index'];
    isOptional: NodeRoutineListItemCreateInput['isOptional'];
    routine: RoutineShape;
    translations: NodeRoutineListItemTranslationShape[];
}

export type NodeRoutineListShape = Omit<ShapeWrapper<NodeDataRoutineList>, 'routines'> & {
    id: string;
    routines: NodeRoutineListItemShape[];
}

export type NodeTranslationShape = Omit<ShapeWrapper<NodeTranslation>, 'language' | 'name'> & {
    id: string;
    language: NodeTranslationCreateInput['language'];
    name: NodeTranslationCreateInput['name'];
}

export type NodeShape = Omit<ShapeWrapper<Node>, 'loop' | 'data' | 'translations'> & {
    id: string;
    routineId: string;
    // loop
    data?: NodeEndShape | NodeRoutineListShape | null;
    translations: NodeTranslationShape[];
}

export const shapeNodeEndCreate = (item: NodeEndShape): NodeEndCreateInput => ({
    id: item.id,
    wasSuccessful: item.wasSuccessful,
})

export const shapeNodeEndUpdate = (
    original: NodeEndShape,
    updated: NodeEndShape
): NodeEndUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    ...shapePrim(o, u, 'wasSuccessful'),
}), 'id')

export const shapeNodeRoutineListItemTranslationCreate = (item: NodeRoutineListItemTranslationShape): NodeRoutineListItemTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    name: item.name,
})

export const shapeNodeRoutineListItemTranslationUpdate = (
    original: NodeRoutineListItemTranslationShape,
    updated: NodeRoutineListItemTranslationShape
): NodeRoutineListItemTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        ...shapePrim(o, u, 'description'),
        ...shapePrim(o, u, 'name'),
    }), 'id')

export const shapeNodeRoutineListItemCreate = (item: NodeRoutineListItemShape): NodeRoutineListItemCreateInput => ({
    id: item.id,
    index: item.index,
    isOptional: item.isOptional ?? false,
    routineVersionConnect: {} as any ,//TODO item.routine.id,
    ...shapeCreateList(item, 'translations', shapeNodeRoutineListItemTranslationCreate),
})

export const shapeNodeRoutineListItemUpdate = (
    original: NodeRoutineListItemShape,
    updated: NodeRoutineListItemShape
): NodeRoutineListItemUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        ...shapePrim(o, u, 'index'),
        ...shapePrim(o, u, 'isOptional'),
        routineUpdate: shapeRoutineUpdate(o.routine, u.routine),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeRoutineListItemTranslationCreate, shapeNodeRoutineListItemTranslationUpdate, 'id'),
    }), 'id')

export const shapeNodeRoutineListCreate = (item: NodeRoutineListShape): NodeRoutineListCreateInput => ({
    id: item.id,
    isOptional: item.isOptional,
    isOrdered: item.isOrdered,
    ...shapeCreateList(item, 'routines', shapeNodeRoutineListItemCreate)
})

export const shapeNodeRoutineListUpdate = (
    original: NodeRoutineListShape,
    updated: NodeRoutineListShape
): NodeRoutineListUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        ...shapePrim(o, u, 'isOptional'),
        ...shapePrim(o, u, 'isOrdered'),
        ...shapeUpdateList(o, u, 'routines', hasObjectChanged, shapeNodeRoutineListItemCreate, shapeNodeRoutineListItemUpdate, 'id')
    }), 'id')

export const shapeNodeTranslationCreate = (item: NodeTranslationShape): NodeTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description ?? undefined,
    name: item.name,
})

export const shapeNodeTranslationUpdate = (
    original: NodeTranslationShape,
    updated: NodeTranslationShape
): NodeTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        ...shapePrim(o, u, 'description'),
        ...shapePrim(o, u, 'name'),
    }), 'id')

export const shapeNodeCreate = (item: NodeShape): NodeCreateInput => ({
    id: item.id,
    columnIndex: item.columnIndex,
    rowIndex: item.rowIndex,
    type: {} as any,//TODO item.type,
    routineVersionId: {} as any,//TODO
    // loopCreate: shapeNodeLoopCreate(node.loop),
    nodeEndCreate: item.data?.__typename === 'NodeEnd' ? shapeNodeEndCreate(item.data as NodeEndShape) : undefined,
    nodeRoutineListCreate: item.data?.__typename === 'NodeRoutineList' ? shapeNodeRoutineListCreate(item.data as NodeRoutineListShape) : undefined,
    ...shapeCreateList(item, 'translations', shapeNodeTranslationCreate)
})

export const shapeNodeUpdate = (
    original: NodeShape,
    updated: NodeShape
): NodeUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    ...shapePrim(o, u, 'columnIndex'),
    ...shapePrim(o, u, 'rowIndex'),
    ...shapePrim(o, u, 'type'),
    // ...shapeNodeLoopUpdate(o.loop, u.loop),
    nodeEndCreate: u.data?.__typename === 'NodeEnd' && !o.data ? shapeNodeEndCreate(u.data as NodeEndShape) : undefined,
    nodeEndUpdate: u.data?.__typename === 'NodeEnd' && o.data ? shapeNodeEndUpdate(o.data as NodeEndShape, u.data as NodeEndShape) : undefined,
    nodeRoutineListCreate: u.data?.__typename === 'NodeRoutineList' && !o.data ? shapeNodeRoutineListCreate(u.data as NodeRoutineListShape) : undefined,
    nodeRoutineListUpdate: u.data?.__typename === 'NodeRoutineList' && o.data ? shapeNodeRoutineListUpdate(o.data as NodeRoutineListShape, u.data as NodeRoutineListShape) : undefined,
    ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeTranslationCreate, shapeNodeTranslationUpdate, 'id')
}), 'id')