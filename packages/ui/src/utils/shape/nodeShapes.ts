import { NodeCreateInput, NodeEndCreateInput, NodeEndUpdateInput, NodeRoutineListCreateInput, NodeRoutineListItemCreateInput, NodeRoutineListItemTranslationCreateInput, NodeRoutineListItemTranslationUpdateInput, NodeRoutineListItemUpdateInput, NodeRoutineListUpdateInput, NodeTranslationCreateInput, NodeTranslationUpdateInput, NodeUpdateInput } from "graphql/generated/globalTypes";
import { Node, NodeDataEnd, NodeDataRoutineList, NodeDataRoutineListItem, NodeDataRoutineListItemTranslation, NodeTranslation } from "types";
import { formatForUpdate, hasObjectChanged, shapeRoutineUpdate } from "utils";
import { shapeCreateList, shapeUpdateList, ShapeWrapper } from "./shapeTools";

type NodeEndCreate = ShapeWrapper<NodeDataEnd>;
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

interface NodeEndUpdate extends NodeEndCreate { id: string };
/**
 * Format a node end for update mutation
 * @param original The original node end's information
 * @param updated The updated node end's information
 * @returns Node end shaped for update mutation
 */
export const shapeNodeEndUpdate = (original: NodeEndUpdate | null | undefined, updated: NodeEndUpdate | null | undefined): NodeEndUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id };
    return {
        id: original.id,
        wasSuccessful: updated.wasSuccessful !== original.wasSuccessful ? updated.wasSuccessful : undefined,
    };
}

type NodeRoutineListItemTranslationCreate = ShapeWrapper<NodeDataRoutineListItemTranslation> &
    Pick<NodeDataRoutineListItemTranslation, 'language'>;
/**
 * Format a node routine list item's translations for create mutation.
 * @param translations The translation list
 * @returns Translations shaped for create mutation
 */
export const shapeNodeRoutineListItemTranslationsCreate = (translations: NodeRoutineListItemTranslationCreate[] | null | undefined): NodeRoutineListItemTranslationCreateInput[] | undefined => {
    return shapeCreateList(translations, (translation) => ({
        id: translation.id,
        language: translation.language,
        description: translation.description,
        title: translation.title,
    }))
}

interface NodeRoutineListItemTranslationUpdate extends NodeRoutineListItemTranslationCreate { id: string };
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

type NodeRoutineListItemCreate = ShapeWrapper<NodeDataRoutineListItem> &
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

interface NodeRoutineListItemUpdate extends NodeRoutineListItemCreate { id: string };
/**
 * Format a node routine list item for update mutation
 * @param original The original routine list item's information
 * @param updated The updated routine list item's information
 * @returns Node routine list item shaped for update mutation
 */
export const shapeNodeRoutineListItemUpdate = (original: NodeRoutineListItemUpdate, updated: NodeRoutineListItemUpdate): NodeRoutineListItemUpdateInput | undefined => {
    return {
        id: original.id,
        index: updated.index !== original.index ? updated.index : undefined,
        isOptional: updated.isOptional !== original.isOptional ? updated.isOptional : undefined,
        ...shapeRoutineUpdate(original.routine, updated.routine),
        ...shapeNodeRoutineListItemTranslationsUpdate(original.translations, updated.translations),
    };
}

/**
 * Format an array of node routine list items for create mutation.
 * @param items The items to format
 * @returns Items shaped for create mutation
 */
 export const shapeNodeRoutineListItemsCreate = (
    items: NodeRoutineListItemCreate[] | null | undefined
): NodeRoutineListItemCreateInput[] | undefined => {
    return shapeCreateList(items, shapeNodeRoutineListItemCreate)
}

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

type NodeRoutineListCreate = ShapeWrapper<NodeDataRoutineList>;
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

interface NodeRoutineListUpdate extends NodeRoutineListCreate { id: string };
/**
 * Format a node routine list for update mutation
 * @param original The original routine list's information
 * @param updated The updated routine list's information
 * @returns Node routine list shaped for update mutation
 */
export const shapeNodeRoutineListUpdate = (original: NodeRoutineListUpdate | null | undefined, updated: NodeRoutineListUpdate | null | undefined): NodeRoutineListUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id };
    return {
        id: original.id,
        isOptional: updated.isOptional !== original.isOptional ? updated.isOptional : undefined,
        isOrdered: updated.isOrdered !== original.isOrdered ? updated.isOrdered : undefined,
        ...shapeNodeRoutineListItemsUpdate(original.routines, updated.routines),
    };
}

type NodeTranslationCreate = ShapeWrapper<NodeTranslation> &
    Pick<NodeTranslationCreateInput, 'language' | 'title'>;
/**
 * Format a node's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeNodeTranslationsCreate = (
    translations: NodeTranslationCreate[] | null | undefined
): NodeTranslationCreate[] | undefined => {
    return shapeCreateList(translations, (translation) => ({
        id: translation.id,
        language: translation.language,
        description: translation.description,
        title: translation.title,
    }))
}

interface NodeTranslationUpdate extends NodeTranslationCreate { id: string };
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

type NodeCreate = ShapeWrapper<Node>;
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

interface NodeUpdate extends NodeCreate { id: string };
/**
 * Format a node for update mutation
 * @param original The original node's information
 * @param updated The updated node's information
 * @returns Node shaped for update mutation
 */
export const shapeNodeUpdate = (original: NodeUpdate | null | undefined, updated: NodeUpdate | null | undefined): NodeUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id };
    return {
        id: original.id,
        columnIndex: updated.columnIndex !== original.columnIndex ? updated.columnIndex : undefined,
        rowIndex: updated.rowIndex !== original.rowIndex ? updated.rowIndex : undefined,
        type: updated.type !== original.type ? updated.type : undefined,
        // ...shapeNodeLoopUpdate(original.loop, updated.loop),
        ...shapeNodeEndUpdate(original.data?.__typename === 'NodeEnd' ? original.data : undefined, updated.data?.__typename === 'NodeEnd' ? updated.data : undefined),
        ...shapeNodeRoutineListUpdate(original.data?.__typename === 'NodeRoutineList' ? original.data : undefined, updated.data?.__typename === 'NodeRoutineList' ? updated.data : undefined),
        ...shapeNodeTranslationsUpdate(original.translations, updated.translations),
    };
}

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