import { NodeCreateInput, NodeEndCreateInput, NodeEndUpdateInput, NodeRoutineListCreateInput, NodeRoutineListItemCreateInput, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput, NodeRoutineListItemUpdateInput, NodeRoutineListUpdateInput, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput } from "graphql/generated/globalTypes";
import { Node, NodeDataEnd, NodeDataRoutineList, NodeDataRoutineListItem, NodeDataRoutineListItemTranslation, NodeTranslation, ShapeWrapper } from "types";
import { hasObjectChanged, RoutineShape, shapeRoutineUpdate } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type NodeEndShape = ShapeWrapper<NodeDataEnd> & {
    id: string;
}

export const shapeNodeEndCreate = (item: NodeEndShape): NodeEndCreateInput => ({
    id: item.id,
    wasSuccessful: item.wasSuccessful,
})

export const shapeNodeEndUpdate = (
    original: NodeEndShape,
    updated: NodeEndShape
): NodeEndUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        wasSuccessful: u.wasSuccessful !== o.wasSuccessful ? u.wasSuccessful : undefined,
    }))

export type NodeRoutineListItemTranslationShape = Omit<ShapeWrapper<NodeDataRoutineListItemTranslation>, 'language'> & {
    id: string;
    language: NodeRoutineListItemTranslationCreateInput['language'];
}

export const shapeNodeRoutineListItemTranslationCreate = (item: NodeRoutineListItemTranslationShape): NodeRoutineListItemTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    title: item.title,
})

export const shapeNodeRoutineListItemTranslationUpdate = (
    original: NodeRoutineListItemTranslationShape,
    updated: NodeRoutineListItemTranslationShape
): NodeRoutineListItemTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        title: u.title !== o.title ? u.title : undefined,
    }))

export const shapeNodeRoutineListItemTranslationsCreate = (items: NodeRoutineListItemTranslationShape[] | null | undefined): {
    translationsCreate?: NodeRoutineListItemTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeNodeRoutineListItemTranslationCreate);

export const shapeNodeRoutineListItemTranslationsUpdate = (
    o: NodeRoutineListItemTranslationShape[] | null | undefined,
    u: NodeRoutineListItemTranslationShape[] | null | undefined
): {
    translationsCreate?: NodeRoutineListItemTranslationCreateInput[],
    translationsUpdate?: NodeRoutineListItemTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeRoutineListItemTranslationCreate, shapeNodeRoutineListItemTranslationUpdate)

export type NodeRoutineListItemShape = Omit<ShapeWrapper<NodeDataRoutineListItem>, 'index' | 'routine'> & {
    id: string;
    index: NodeRoutineListItemCreateInput['index'];
    isOptional: NodeRoutineListItemCreateInput['isOptional'];
    routine: RoutineShape;
    translations: NodeRoutineListItemTranslationShape[];
}

export const shapeNodeRoutineListItemCreate = (item: NodeRoutineListItemShape): NodeRoutineListItemCreateInput => ({
    id: item.id,
    index: item.index,
    isOptional: item.isOptional ?? false,
    routineConnect: item.routine.id,
    ...shapeNodeRoutineListItemTranslationsCreate(item.translations),
})

export const shapeNodeRoutineListItemUpdate = (
    original: NodeRoutineListItemShape,
    updated: NodeRoutineListItemShape
): NodeRoutineListItemUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        index: u.index !== o.index ? u.index : undefined,
        isOptional: u.isOptional !== o.isOptional ? u.isOptional : undefined,
        routineUpdate: shapeRoutineUpdate(o.routine, u.routine),
        ...shapeNodeRoutineListItemTranslationsUpdate(o.translations, u.translations),
    }))

export const shapeNodeRoutineListItemsCreate = (items: NodeRoutineListItemShape[] | null | undefined): {
    routinesCreate?: NodeRoutineListItemCreateInput[],
} => shapeCreateList(items, 'routines', shapeNodeRoutineListItemCreate);

export const shapeNodeRoutineListItemsUpdate = (
    o: NodeRoutineListItemShape[] | null | undefined,
    u: NodeRoutineListItemShape[] | null | undefined
): {
    routinesCreate?: NodeRoutineListItemCreateInput[],
    routinesUpdate?: NodeRoutineListItemUpdateInput[],
    routinesDelete?: string[],
} => shapeUpdateList(o, u, 'routines', hasObjectChanged, shapeNodeRoutineListItemCreate, shapeNodeRoutineListItemUpdate)

export type NodeRoutineListShape = Omit<ShapeWrapper<NodeDataRoutineList>, 'routines'> & {
    id: string;
    routines: NodeRoutineListItemShape[];
}

export const shapeNodeRoutineListCreate = (item: NodeRoutineListShape): NodeRoutineListCreateInput => ({
    id: item.id,
    isOptional: item.isOptional,
    isOrdered: item.isOrdered,
    ...shapeNodeRoutineListItemsCreate(item.routines),
})

export const shapeNodeRoutineListUpdate = (
    original: NodeRoutineListShape,
    updated: NodeRoutineListShape
): NodeRoutineListUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        isOptional: u.isOptional !== o.isOptional ? u.isOptional : undefined,
        isOrdered: u.isOrdered !== o.isOrdered ? u.isOrdered : undefined,
        ...shapeNodeRoutineListItemsUpdate(o.routines, u.routines),
    }))

export type NodeTranslationShape = Omit<ShapeWrapper<NodeTranslation>, 'language' | 'title'> & {
    id: string;
    language: NodeTranslationCreateInput['language'];
    title: NodeTranslationCreateInput['title'];
}

export const shapeNodeTranslationCreate = (item: NodeTranslationShape): NodeTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    title: item.title,
})

export const shapeNodeTranslationUpdate = (
    original: NodeTranslationShape,
    updated: NodeTranslationShape
): NodeTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        title: u.title !== o.title ? u.title : undefined,
    }))

export const shapeNodeTranslationsCreate = (items: NodeTranslationShape[] | null | undefined): {
    translationsCreate?: NodeTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeNodeTranslationCreate);

export const shapeNodeTranslationsUpdate = (
    o: NodeTranslationShape[] | null | undefined,
    u: NodeTranslationShape[] | null | undefined
): {
    translationsCreate?: NodeTranslationCreateInput[],
    translationsUpdate?: NodeTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeTranslationCreate, shapeNodeTranslationUpdate)

export type NodeShape = Omit<ShapeWrapper<Node>, 'loop' | 'data' | 'translations'> & {
    id: string;
    routineId: string;
    // loop
    data?: NodeEndShape | NodeRoutineListShape | null;
    translations: NodeTranslationShape[];
}

export const shapeNodeCreate = (item: NodeShape): NodeCreateInput => ({
    id: item.id,
    columnIndex: item.columnIndex,
    rowIndex: item.rowIndex,
    type: item.type,
    // loopCreate: shapeNodeLoopCreate(node.loop),
    nodeEndCreate: item.data?.__typename === 'NodeEnd' ? shapeNodeEndCreate(item.data as NodeEndShape) : undefined,
    nodeRoutineListCreate: item.data?.__typename === 'NodeRoutineList' ? shapeNodeRoutineListCreate(item.data as NodeRoutineListShape) : undefined,
    ...shapeNodeTranslationsCreate(item.translations),
})

export const shapeNodeUpdate = (
    original: NodeShape,
    updated: NodeShape
): NodeUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        columnIndex: u.columnIndex !== o.columnIndex ? u.columnIndex : undefined,
        rowIndex: u.rowIndex !== o.rowIndex ? u.rowIndex : undefined,
        type: u.type !== o.type ? u.type : undefined,
        // ...shapeNodeLoopUpdate(o.loop, u.loop),
        nodeEndUpdate: o.data?.__typename === 'NodeEnd' ? shapeNodeEndUpdate(o.data as NodeEndShape, u.data as NodeEndShape) : undefined,
        nodeRoutineListUpdate: o.data?.__typename === 'NodeRoutineList' ? shapeNodeRoutineListUpdate(o.data as NodeRoutineListShape, u.data as NodeRoutineListShape) : undefined,
        ...shapeNodeTranslationsUpdate(o.translations, u.translations),
    }))

export const shapeNodesCreate = (items: NodeShape[] | null | undefined): {
    nodesCreate?: NodeCreateInput[],
} => shapeCreateList(items, 'nodes', shapeNodeCreate);

export const shapeNodesUpdate = (
    o: NodeShape[] | null | undefined,
    u: NodeShape[] | null | undefined
): {
    nodesCreate?: NodeCreateInput[],
    nodesUpdate?: NodeUpdateInput[],
    nodesDelete?: string[],
} => shapeUpdateList(o, u, 'nodes', hasObjectChanged, shapeNodeCreate, shapeNodeUpdate)