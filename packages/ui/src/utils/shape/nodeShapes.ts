import { NodeCreateInput, NodeUpdateInput, RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { Node } from "types";
import { formatForUpdate, hasObjectChanged } from "utils";
import { findCreatedItems, findRemovedItems, findUpdatedItems } from "./shapeTools";

type NodeTranslationCreate = Partial<Node['translations'][0]> & {
    id: string;
    language: Node['translations'][0]['language'];
    title: Node['translations'][0]['title'];
}
/**
 * Format a node's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeNodeTranslationsCreate = (
    translations: NodeTranslationCreate[] | null | undefined
): NodeCreateInput['translationsCreate'] | undefined => {
    if (!translations) return undefined;
    const formatted: NodeCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
            id: translation.id,
            language: translation.language,
            description: translation.description,
            title: translation.title,
        });
    }
    return formatted.length > 0 ? formatted : undefined;
}

/**
 * Format a node's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeNodeTranslationsUpdate = (
    original: NodeTranslationCreate[] | null | undefined,
    updated: NodeTranslationCreate[] | null | undefined
): {
    translationsCreate?: NodeUpdateInput['translationsCreate'],
    translationsUpdate?: NodeUpdateInput['translationsUpdate'],
    translationsDelete?: NodeUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { translationsCreate: shapeNodeTranslationsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        translationsCreate: findCreatedItems(original, updated, shapeNodeTranslationsCreate),
        translationsUpdate: findUpdatedItems(original, updated, hasObjectChanged, formatForUpdate) as NodeUpdateInput['translationsUpdate'],
        translationsDelete: findRemovedItems(original, updated),
    } : {}
}

type ShapeNodeCreateInput = Partial<Node> & {
    id: Node['id'],
}
/**
 * Format a node for create mutation.
 * @param node The node's information
 * @returns Node shaped for create mutation
 */
export const shapeNodeCreate = (node: ShapeNodeCreateInput | null | undefined): NodeCreateInput | undefined => {
    if (!node) return undefined;
    return {
        id: node.id,
        columnIndex: node.columnIndex,
        rowIndex: node.rowIndex,
        type: node.type,
        // ...shapeNodeLoopCreate(node.loop),
        ...shapeNodeEndCreate(node.end),
        ...shapeNodeRoutineListCreate(node.routineList),
        ...shapeNodeTranslationsCreate(node.translations),
    };
}

type ShapeNodeUpdateInput = ShapeNodeCreateInput & {

};
/**
 * Format a node for update mutation
 * @param original The original node's information
 * @param updated The updated node's information
 * @returns Node shaped for update mutation
 */
export const shapeNodeUpdate = (original: ShapeNodeUpdateInput | null | undefined, updated: ShapeNodeUpdateInput | null | undefined): NodeUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id };
    return {
        id: original.id,
        columnIndex: updated.columnIndex !== original.columnIndex ? updated.columnIndex : undefined,
        rowIndex: updated.rowIndex !== original.rowIndex ? updated.rowIndex : undefined,
        type: updated.type !== original.type ? updated.type : undefined,
        // ...shapeNodeLoopUpdate(original.loop, updated.loop),
        ...shapeNodeEndUpdate(original.end, updated.end),
        ...shapeNodeRoutineListUpdate(original.routineList, updated.routineList),
        ...shapeNodeTranslationsCreate(original.translations, updated.translations),
    };
}

/**
 * Format an array nodes for create mutation.
 * @param nodes The nodes to format
 * @returns Nodes shaped for create mutation
 */
export const shapeNodesCreate = (
    nodes: ShapeNodeCreateInput[] | null | undefined
): RoutineCreateInput['nodesCreate'] | undefined => {
    if (!nodes) return undefined;
    const formatted: NodeCreateInput[] = [];
    for (const node of nodes) {
        const currShaped = shapeNodeCreate(node);
        if (currShaped) formatted.push(currShaped);
    }
    return formatted.length > 0 ? formatted : undefined;
}

/**
 * Format an array of nodes for update mutation.
 * @param original Original nodes list
 * @param updated Updated nodes list
 * @returns Formatted nodes
 */
export const shapeNodesUpdate = (
    original: ShapeNodeUpdateInput[] | null | undefined,
    updated: ShapeNodeUpdateInput[] | null | undefined
): {
    nodesCreate?: RoutineUpdateInput['nodesCreate'],
    nodesUpdate?: RoutineUpdateInput['nodesUpdate'],
    nodesDelete?: RoutineUpdateInput['nodesDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { nodesCreate: shapeNodesCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        nodesCreate: findCreatedItems(original, updated, shapeNodesCreate),
        nodesUpdate: findUpdatedItems(original, updated, hasObjectChanged, shapeNodeUpdate) as RoutineUpdateInput['nodesUpdate'],
        nodesDelete: findRemovedItems(original, updated),
    } : {}
}