import { NodeCreateInput, NodeUpdateInput, RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { Node } from "types";
import { formatForUpdate, hasObjectChanged } from "utils";

/**
 * Format a node's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeNodeTranslationsCreate = (translations: { [x: string]: any }[] | null | undefined): { translationsCreate?: NodeCreateInput['translationsCreate'] } => {
    if (!translations) return {};
    const formatted: NodeCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
            id: translation.id,
            language: translation.language,
            description: translation.description,
            title: translation.title,
        });
    }
    return formatted.length > 0 ? {
        translationsCreate: formatted,
    } : {};
}

/**
 * Format a node's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeNodeTranslationsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: NodeUpdateInput['translationsCreate'],
    translationsUpdate?: NodeUpdateInput['translationsUpdate'],
    translationsDelete?: NodeUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return shapeNodeTranslationsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeNodeTranslationsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        translationsUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? formatForUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as NodeUpdateInput['translationsUpdate'],
        translationsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
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
 export const shapeNodesCreate =  (nodes: ShapeNodeCreateInput[] | null | undefined): { 
    nodesCreate?: RoutineCreateInput['nodesCreate'],
} => {
    if (!nodes) return {};
    const formatted: NodeCreateInput[] = [];
    for (const node of nodes) {
        const currShaped = shapeNodeCreate(node);
        if (currShaped) formatted.push(currShaped);
    }
    return formatted.length > 0 ? {
        nodesCreate: formatted,
    } : {};
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
    if (!updated) return { };
    if (!original || !Array.isArray(original)) {
        return shapeNodesCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeNodesCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        nodesUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? shapeNodeUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as RoutineUpdateInput['nodesUpdate'],
        nodesDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id).filter(id => id) as string[],
    } : {}
}