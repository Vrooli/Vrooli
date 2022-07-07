import { NodeCreateInput, NodeEndCreateInput, NodeEndUpdateInput, NodeRoutineListCreateInput, NodeRoutineListItemCreateInput, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput, NodeRoutineListItemUpdateInput, NodeRoutineListUpdateInput, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput } from "graphql/generated/globalTypes";
import { Node, NodeDataEnd, NodeDataRoutineList, NodeDataRoutineListItem, NodeDataRoutineListItemTranslation, NodeTranslation, ShapeWrapper } from "types";
import { formatForUpdate, hasObjectChanged, shapeRoutineUpdate } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type NodeEndCreate = ShapeWrapper<NodeDataEnd>;
/**
 * Format a node end for create mutation.
 * @param end The node end's information
 * @returns Node end shaped for create mutation
 */
export const shapeNodeEndCreate = (end: NodeEndCreate | null | undefined): NodeEndCreateInput | undefined => {
    if (!end) return undefined;
    return {
        id: end.id,
        wasSuccessful: end.wasSuccessful,
    };
}

export interface NodeEndUpdate extends NodeEndCreate { id: string };
/**
 * Format a node end for update mutation
 * @param original The original node end's information
 * @param updated The updated node end's information
 * @returns Node end shaped for update mutation
 */
export const shapeNodeEndUpdate = (
    original: NodeEndUpdate,
    updated: NodeEndUpdate | null | undefined
): NodeEndUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    wasSuccessful: u.wasSuccessful !== o.wasSuccessful ? u.wasSuccessful : undefined,
}))

export type NodeRoutineListItemTranslationCreate = ShapeWrapper<NodeDataRoutineListItemTranslation> &
    Pick<NodeDataRoutineListItemTranslation, 'language'>;
/**
 * Format a node routine list item's translations for create mutation.
 * @param translations The translation list
 * @returns Translations shaped for create mutation
 */
export const shapeNodeRoutineListItemTranslationsCreate = (
    translations: NodeRoutineListItemTranslationCreate[] | null | undefined
): NodeRoutineListItemTranslationCreateInput[] | undefined => shapeCreateList(translations, (translation) => ({
    id: translation.id,
    language: translation.language,
    description: translation.description,
    title: translation.title,
}))

export interface NodeRoutineListItemTranslationUpdate extends NodeRoutineListItemTranslationCreate { id: string };
/**
 * Format a node routine list item translation for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeNodeRoutineListItemTranslationsUpdate = (
    original: NodeRoutineListItemTranslationUpdate[] | null | undefined,
    updated: NodeRoutineListItemTranslationUpdate[] | null | undefined
): {
    translationsCreate?: NodeRoutineListItemTranslationCreateInput[],
    translationsUpdate?: NodeRoutineListItemTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'translations',
    shapeNodeRoutineListItemTranslationsCreate,
    hasObjectChanged,
    formatForUpdate as (original: NodeRoutineListItemTranslationUpdate, updated: NodeRoutineListItemTranslationUpdate) => NodeRoutineListItemTranslationUpdateInput | undefined,
)

export type NodeRoutineListItemCreate = ShapeWrapper<NodeDataRoutineListItem> &
    Pick<NodeDataRoutineListItem, 'index'> & {
        routine: Partial<NodeDataRoutineListItem['routine'] & { id: string }>;
    }
/**
 * Format a node routine list item for create mutation.
 * @param item The node routine list item's information
 * @returns Node routine list item shaped for create mutation
 */
export const shapeNodeRoutineListItemCreate = (item: NodeRoutineListItemCreate | null | undefined): NodeRoutineListItemCreateInput | undefined => {
    if (!item) return undefined;
    return {
        id: item.id,
        index: item.index,
        isOptional: item.isOptional,
        routineConnect: item.routine.id,
        ...shapeNodeRoutineListItemTranslationsCreate(item.translations),
    };
}

export interface NodeRoutineListItemUpdate extends NodeRoutineListItemCreate { id: string };
/**
 * Format a node routine list item for update mutation
 * @param original The original routine list item's information
 * @param updated The updated routine list item's information
 * @returns Node routine list item shaped for update mutation
 */
export const shapeNodeRoutineListItemUpdate = (
    original: NodeRoutineListItemUpdate,
    updated: NodeRoutineListItemUpdate
): NodeRoutineListItemUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    index: u.index !== o.index ? u.index : undefined,
    isOptional: u.isOptional !== o.isOptional ? u.isOptional : undefined,
    ...shapeRoutineUpdate(o.routine, u.routine),
    ...shapeNodeRoutineListItemTranslationsUpdate(o.translations, u.translations),
}))

/**
 * Format an array of node routine list items for create mutation.
 * @param items The items to format
 * @returns Items shaped for create mutation
 */
export const shapeNodeRoutineListItemsCreate = (
    items: NodeRoutineListItemCreate[] | null | undefined
): NodeRoutineListItemCreateInput[] | undefined => shapeCreateList(items, shapeNodeRoutineListItemCreate)

/**
 * Format an array of node routine list items for update mutation.
 * @param original Original items list
 * @param updated Updated items list
 * @returns Formatted items
 */
export const shapeNodeRoutineListItemsUpdate = (
    original: NodeRoutineListItemUpdate[] | null | undefined,
    updated: NodeRoutineListItemUpdate[] | null | undefined
): {
    routinesCreate?: NodeRoutineListItemCreateInput[],
    routinesUpdate?: NodeRoutineListItemUpdateInput[],
    routinesDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'routines',
    shapeNodeRoutineListItemsCreate,
    hasObjectChanged,
    shapeNodeRoutineListItemUpdate,
)

export type NodeRoutineListCreate = ShapeWrapper<NodeDataRoutineList>;
/**
 * Format a node routine list for create mutation.
 * @param routineList The node routine list's information
 * @returns Node routine list shaped for create mutation
 */
export const shapeNodeRoutineListCreate = (routineList: NodeRoutineListCreate | null | undefined): NodeRoutineListCreateInput | undefined => {
    if (!routineList) return undefined;
    return {
        id: routineList.id,
        isOptional: routineList.isOptional,
        isOrdered: routineList.isOrdered,
        ...shapeNodeRoutineListItemsCreate(routineList.routines),
    };
}

export interface NodeRoutineListUpdate extends NodeRoutineListCreate { id: string };
/**
 * Format a node routine list for update mutation
 * @param original The original routine list's information
 * @param updated The updated routine list's information
 * @returns Node routine list shaped for update mutation
 */
export const shapeNodeRoutineListUpdate = (
    original: NodeRoutineListUpdate,
    updated: NodeRoutineListUpdate | null | undefined
): NodeRoutineListUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    isOptional: u.isOptional !== o.isOptional ? u.isOptional : undefined,
    isOrdered: u.isOrdered !== o.isOrdered ? u.isOrdered : undefined,
    ...shapeNodeRoutineListItemsUpdate(o.routines, u.routines),
}))

type NodeTranslationCreate = ShapeWrapper<NodeTranslation> &
    Pick<NodeTranslationCreateInput, 'language' | 'title'>;
/**
 * Format a node's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeNodeTranslationsCreate = (
    translations: NodeTranslationCreate[] | null | undefined
): NodeTranslationCreate[] | undefined => shapeCreateList(translations, (translation) => ({
    id: translation.id,
    language: translation.language,
    description: translation.description,
    title: translation.title,
}))

export interface NodeTranslationUpdate extends NodeTranslationCreate { id: string };
/**
 * Format a node's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeNodeTranslationsUpdate = (
    original: NodeTranslationUpdate[] | null | undefined,
    updated: NodeTranslationUpdate[] | null | undefined
): {
    translationsCreate?: NodeTranslationCreateInput[],
    translationsUpdate?: NodeTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'translations',
    shapeNodeTranslationsCreate,
    hasObjectChanged,
    formatForUpdate as (original: NodeTranslationCreate, updated: NodeTranslationCreate) => NodeTranslationUpdateInput | undefined,
)

export type NodeCreate = ShapeWrapper<Node>;
/**
 * Format a node for create mutation.
 * @param node The node's information
 * @returns Node shaped for create mutation
 */
export const shapeNodeCreate = (node: NodeCreate | null | undefined): NodeCreateInput | undefined => {
    if (!node) return undefined;
    return {
        id: node.id,
        columnIndex: node.columnIndex,
        rowIndex: node.rowIndex,
        type: node.type,
        // ...shapeNodeLoopCreate(node.loop),
        ...shapeNodeEndCreate(node.data?.__typename === 'NodeEnd' ? node.data : undefined),
        ...shapeNodeRoutineListCreate(node.data?.__typename === 'NodeRoutineList' ? node.data : undefined),
        ...shapeNodeTranslationsCreate(node.translations),
    };
}

export interface NodeUpdate extends NodeCreate { id: string };
/**
 * Format a node for update mutation
 * @param original The original node's information
 * @param updated The updated node's information
 * @returns Node shaped for update mutation
 */
export const shapeNodeUpdate = (
    original: NodeUpdate,
    updated: NodeUpdate | null | undefined
): NodeUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    columnIndex: u.columnIndex !== o.columnIndex ? u.columnIndex : undefined,
    rowIndex: u.rowIndex !== o.rowIndex ? u.rowIndex : undefined,
    type: u.type !== o.type ? u.type : undefined,
    // ...shapeNodeLoopUpdate(o.loop, u.loop),
    nodeEndUpdate: o.data?.__typename === 'NodeEnd' ? shapeNodeEndUpdate(o.data, u.data) : undefined,
    nodeRoutineListUpdate: o.data?.__typename === 'NodeRoutineList' ? shapeNodeRoutineListUpdate(o.data, u.data) : undefined,
    ...shapeNodeTranslationsUpdate(o.translations, u.translations),
}))

/**
 * Format an array of nodes for create mutation.
 * @param nodes The nodes to format
 * @returns Nodes shaped for create mutation
 */
export const shapeNodesCreate = (
    nodes: NodeCreate[] | null | undefined
): NodeCreateInput[] | undefined => {
    return shapeCreateList(nodes, shapeNodeCreate)
}

/**
 * Format an array of nodes for update mutation.
 * @param original Original nodes list
 * @param updated Updated nodes list
 * @returns Formatted nodes
 */
export const shapeNodesUpdate = (
    original: NodeUpdate[] | null | undefined,
    updated: NodeUpdate[] | null | undefined
): {
    nodesCreate?: NodeCreateInput[],
    nodesUpdate?: NodeUpdateInput[],
    nodesDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'nodes',
    shapeNodesCreate,
    hasObjectChanged,
    shapeNodeUpdate,
)