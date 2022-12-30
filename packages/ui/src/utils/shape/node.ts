import { NodeCreateInput, NodeEnd, NodeEndCreateInput, NodeEndUpdateInput, NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListItem, NodeRoutineListItemCreateInput, NodeRoutineListItemTranslation, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput, NodeRoutineListItemUpdateInput, NodeRoutineListUpdateInput, NodeTranslation, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged, RoutineShape, shapeRoutineUpdate } from "utils";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type NodeEndShape = OmitCalculated<NodeEnd>

export type NodeRoutineListItemTranslationShape = OmitCalculated<NodeRoutineListItemTranslation>

export type NodeRoutineListItemShape = Omit<OmitCalculated<NodeRoutineListItem>, 'index' | 'routine'> & {
    id: string;
    index: NodeRoutineListItemCreateInput['index'];
    isOptional: NodeRoutineListItemCreateInput['isOptional'];
    routine: RoutineShape;
    translations: NodeRoutineListItemTranslationShape[];
}

export type NodeRoutineListShape = Omit<OmitCalculated<NodeRoutineList>, 'routines'> & {
    id: string;
    routines: NodeRoutineListItemShape[];
}

export type NodeTranslationShape = OmitCalculated<NodeTranslation>

export type NodeShape = Omit<OmitCalculated<Node>, 'loop' | 'data' | 'translations'> & {
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

export const shapeNodeEndUpdate = (o: NodeEndShape, u: NodeEndShape): NodeEndUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'wasSuccessful'))

export const shapeNodeRoutineListItemTranslationCreate = (item: NodeRoutineListItemTranslationShape): NodeRoutineListItemTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description', 'name')

export const shapeNodeRoutineListItemTranslationUpdate = (o: NodeRoutineListItemTranslationShape, u: NodeRoutineListItemTranslationShape): NodeRoutineListItemTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description', 'name'))

export const shapeNodeRoutineListItemCreate = (item: NodeRoutineListItemShape): NodeRoutineListItemCreateInput => ({
    id: item.id,
    index: item.index,
    isOptional: item.isOptional ?? false,
    routineVersionConnect: {} as any,//TODO item.routine.id,
    ...shapeCreateList(item, 'translations', shapeNodeRoutineListItemTranslationCreate),
})

export const shapeNodeRoutineListItemUpdate = (o: NodeRoutineListItemShape, u: NodeRoutineListItemShape): NodeRoutineListItemUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'index', 'isOptional'),
        routineUpdate: shapeRoutineUpdate(o.routine, u.routine),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeRoutineListItemTranslationCreate, shapeNodeRoutineListItemTranslationUpdate, 'id'),
    })

export const shapeNodeRoutineListCreate = (item: NodeRoutineListShape): NodeRoutineListCreateInput => ({
    ...shapeCreatePrims(item, 'id', 'isOptional', 'isOrdered'),
    ...shapeCreateList(item, 'routines', shapeNodeRoutineListItemCreate)
})

export const shapeNodeRoutineListUpdate = (o: NodeRoutineListShape, u: NodeRoutineListShape): NodeRoutineListUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'isOptional', 'isOrdered'),
        ...shapeUpdateList(o, u, 'routines', hasObjectChanged, shapeNodeRoutineListItemCreate, shapeNodeRoutineListItemUpdate, 'id')
    })

export const shapeNodeTranslationCreate = (item: NodeTranslationShape): NodeTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description', 'name')

export const shapeNodeTranslationUpdate = (o: NodeTranslationShape, u: NodeTranslationShape): NodeTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description', 'name'))

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

export const shapeNodeUpdate = (o: NodeShape, u: NodeShape): NodeUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'columnIndex', 'rowIndex', 'type'),
        // ...shapeNodeLoopUpdate(o.loop, u.loop),
        nodeEndCreate: u.data?.__typename === 'NodeEnd' && !o.data ? shapeNodeEndCreate(u.data as NodeEndShape) : undefined,
        nodeEndUpdate: u.data?.__typename === 'NodeEnd' && o.data ? shapeNodeEndUpdate(o.data as NodeEndShape, u.data as NodeEndShape) : undefined,
        nodeRoutineListCreate: u.data?.__typename === 'NodeRoutineList' && !o.data ? shapeNodeRoutineListCreate(u.data as NodeRoutineListShape) : undefined,
        nodeRoutineListUpdate: u.data?.__typename === 'NodeRoutineList' && o.data ? shapeNodeRoutineListUpdate(o.data as NodeRoutineListShape, u.data as NodeRoutineListShape) : undefined,
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeTranslationCreate, shapeNodeTranslationUpdate, 'id')
    })