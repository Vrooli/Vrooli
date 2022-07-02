import { NodeLinkCreateInput, NodeLinkUpdateInput, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput, RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { NodeLink } from "types";
import { formatForUpdate, hasObjectChanged } from "utils";

/**
 * Format a nodeLink's when's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeNodeLinkWhenTranslationsCreate = (translations: { [x: string]: any }[] | null | undefined): { translationsCreate?: NodeLinkCreateInput['translationsCreate'] } => {
    if (!translations) return {};
    const formatted: NodeLinkWhenCreateInput['translationsCreate'] = [];
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
 * Format a nodeLink's when's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeNodeLinkWhenTranslationsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: NodeLinkWhenUpdateInput['translationsCreate'],
    translationsUpdate?: NodeLinkWhenUpdateInput['translationsUpdate'],
    translationsDelete?: NodeLinkWhenUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return shapeNodeLinkWhenTranslationsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeNodeLinkWhenTranslationsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        translationsUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? formatForUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as NodeLinkWhenUpdateInput['translationsUpdate'],
        translationsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
    } : {}
}

type ShapeNodeLinkCreateInput = Partial<NodeLink> & {
    id: NodeLink['id'],
}
/**
 * Format a nodeLink for create mutation.
 * @param nodeLink The nodeLink's information
 * @returns NodeLink shaped for create mutation
 */
export const shapeNodeLinkCreate = (nodeLink: ShapeNodeLinkCreateInput | null | undefined): NodeLinkCreateInput | undefined => {
    if (!nodeLink) return undefined;
    return {
        id: nodeLink.id,
        columnIndex: nodeLink.columnIndex,
        rowIndex: nodeLink.rowIndex,
        type: nodeLink.type,
        // ...shapeNodeLinkLoopCreate(nodeLink.loop),
        ...shapeNodeLinkEndCreate(nodeLink.end),
        ...shapeNodeLinkRoutineListCreate(nodeLink.routineList),
        ...shapeNodeLinkTranslationsCreate(nodeLink.translations),
    };
}

type ShapeNodeLinkUpdateInput = ShapeNodeLinkCreateInput & {

};
/**
 * Format a nodeLink for update mutation
 * @param original The original nodeLink's information
 * @param updated The updated nodeLink's information
 * @returns NodeLink shaped for update mutation
 */
export const shapeNodeLinkUpdate = (original: ShapeNodeLinkUpdateInput | null | undefined, updated: ShapeNodeLinkUpdateInput | null | undefined): NodeLinkUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id };
    return {
        id: original.id,
        columnIndex: updated.columnIndex !== original.columnIndex ? updated.columnIndex : undefined,
        rowIndex: updated.rowIndex !== original.rowIndex ? updated.rowIndex : undefined,
        type: updated.type !== original.type ? updated.type : undefined,   
        // ...shapeNodeLinkLoopUpdate(original.loop, updated.loop),
        ...shapeNodeLinkEndUpdate(original.end, updated.end),
        ...shapeNodeLinkRoutineListUpdate(original.routineList, updated.routineList),
        ...shapeNodeLinkTranslationsCreate(original.translations, updated.translations),
    };
}

/**
 * Format an array nodeLinks for create mutation.
 * @param nodeLinks The nodeLinks to format
 * @returns NodeLinks shaped for create mutation
 */
 export const shapeNodeLinksCreate =  (nodeLinks: ShapeNodeLinkCreateInput[] | null | undefined): { 
    nodeLinksCreate?: RoutineCreateInput['nodeLinksCreate'],
} => {
    if (!nodeLinks) return {};
    const formatted: NodeLinkCreateInput[] = [];
    for (const nodeLink of nodeLinks) {
        const currShaped = shapeNodeLinkCreate(nodeLink);
        if (currShaped) formatted.push(currShaped);
    }
    return formatted.length > 0 ? {
        nodeLinksCreate: formatted,
    } : {};
}

/**
 * Format an array of nodeLinks for update mutation.
 * @param original Original nodeLinks list
 * @param updated Updated nodeLinks list
 * @returns Formatted nodeLinks
 */
 export const shapeNodeLinksUpdate = (
    original: ShapeNodeLinkUpdateInput[] | null | undefined,
    updated: ShapeNodeLinkUpdateInput[] | null | undefined
): {
    nodeLinksCreate?: RoutineUpdateInput['nodeLinksCreate'],
    nodeLinksUpdate?: RoutineUpdateInput['nodeLinksUpdate'],
    nodeLinksDelete?: RoutineUpdateInput['nodeLinksDelete'],
} => {
    if (!updated) return { };
    if (!original || !Array.isArray(original)) {
        return shapeNodeLinksCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeNodeLinksCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        nodeLinksUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? shapeNodeLinkUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as RoutineUpdateInput['nodeLinksUpdate'],
        nodeLinksDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id).filter(id => id) as string[],
    } : {}
}