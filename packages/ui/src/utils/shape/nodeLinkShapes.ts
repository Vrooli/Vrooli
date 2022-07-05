import { NodeLinkCreateInput, NodeLinkUpdateInput, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput, RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { NodeLink } from "types";
import { formatForUpdate, hasObjectChanged } from "utils";
import { findCreatedItems, findRemovedItems, findUpdatedItems } from "./shapeTools";

type NodeLinkWhenCreate = Partial<NodeLink['whens'][0]['translations'][0]> & {
    id: string;
    language: NodeLink['whens'][0]['translations'][0]['language'];
    title: NodeLink['whens'][0]['translations'][0]['title'];
}
/**
 * Format a nodeLink's when's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeNodeLinkWhenTranslationsCreate = (
    translations: NodeLinkWhenCreate[] | null | undefined
): NodeLinkWhenCreateInput['translationsCreate'] | undefined => {
    if (!translations) return undefined;
    const formatted: NodeLinkWhenCreateInput['translationsCreate'] = [];
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
 * Format a nodeLink's when's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeNodeLinkWhenTranslationsUpdate = (
    original: NodeLinkWhenCreate[] | null | undefined,
    updated: NodeLinkWhenCreate[] | null | undefined
): {
    translationsCreate?: NodeLinkWhenUpdateInput['translationsCreate'],
    translationsUpdate?: NodeLinkWhenUpdateInput['translationsUpdate'],
    translationsDelete?: NodeLinkWhenUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { translationsCreate: shapeNodeLinkWhenTranslationsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        translationsCreate: findCreatedItems(original, updated, shapeNodeLinkWhenTranslationsCreate),
        translationsUpdate: findUpdatedItems(original, updated, hasObjectChanged, formatForUpdate) as NodeLinkWhenUpdateInput['translationsUpdate'],
        translationsDelete: findRemovedItems(original, updated),
    } : {}
}

type ShapeNodeLinkWhenCreateInput = Partial<NodeLink['whens'][0]> & {
    id: NodeLink['whens'][0]['id'],
    toId?: NodeLink['id'],
}
/**
 * Format a nodeLink when for create mutation.
 * @param when The nodeLink when's information
 * @returns NodeLink when shaped for create mutation
 */
export const shapeNodeLinkWhenCreate = (when: ShapeNodeLinkWhenCreateInput | null | undefined): NodeLinkWhenCreateInput | undefined => {
    if (!when) return undefined;
    return {
        id: when.id,
        toId: when.toId,
        condition: when.condition ?? '',
        ...shapeNodeLinkWhenTranslationsCreate(when.translations),
    };
}

type ShapeNodeLinkWhenUpdateInput = ShapeNodeLinkWhenCreateInput & {

};
/**
 * Format a nodeLink when for update mutation
 * @param original The original nodeLink when's information
 * @param updated The updated nodeLink when's information
 * @returns NodeLink when shaped for update mutation
 */
export const shapeNodeLinkWhenUpdate = (original: ShapeNodeLinkWhenUpdateInput | null | undefined, updated: ShapeNodeLinkWhenUpdateInput | null | undefined): NodeLinkWhenUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id } as ShapeNodeLinkWhenUpdateInput;
    return {
        id: original.id,
        toId: updated.toId !== original.toId ? updated.toId : undefined,
        condition: updated.condition !== original.condition ? updated.condition : undefined,
        ...shapeNodeLinkWhenTranslationsUpdate(original.translations, updated.translations),
    };
}

/**
 * Format an array of nodeLink whens for create mutation.
 * @param whens The nodeLink whens to format
 * @returns NodeLink whens shaped for create mutation
 */
export const shapeNodeLinkWhensCreate = (
    whens: ShapeNodeLinkWhenCreateInput[] | null | undefined
): NodeLinkWhenCreateInput[] | undefined => {
    if (!whens) return undefined;
    const formatted: NodeLinkWhenCreateInput[] = [];
    for (const when of whens) {
        const currShaped = shapeNodeLinkWhenCreate(when);
        if (currShaped) formatted.push(currShaped);
    }
    return formatted.length > 0 ? formatted : undefined;
}

/**
 * Format an array of nodeLink whenss for update mutation.
 * @param original Original nodeLink whens list
 * @param updated Updated nodeLink whens list
 * @returns Formatted nodeLink whens
 */
export const shapeNodeLinkWhensUpdate = (
    original: ShapeNodeLinkWhenUpdateInput[] | null | undefined,
    updated: ShapeNodeLinkWhenUpdateInput[] | null | undefined
): {
    whensCreate?: NodeLinkCreateInput['whens']
    whensUpdate?: NodeLinkUpdateInput['whensUpdate'],
    whensDelete?: NodeLinkUpdateInput['whensDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { whensCreate: shapeNodeLinkWhensCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        whensCreate: findCreatedItems(original, updated, shapeNodeLinkWhensCreate),
        whensUpdate: findUpdatedItems(original, updated, hasObjectChanged, shapeNodeLinkWhenUpdate) as NodeLinkUpdateInput['whensUpdate'],
        whensDelete: findRemovedItems(original, updated),
    } : {}
}

type ShapeNodeLinkCreateInput = Partial<NodeLink> & {
    id: NodeLink['id'],
    fromId: NodeLink['fromId'],
    toId: NodeLink['toId'],
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
        operation: nodeLink.operation,
        fromId: nodeLink.fromId,
        toId: nodeLink.toId,
        ...shapeNodeLinkWhensCreate(nodeLink.whens),
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
    if (!original) original = { id: updated.id } as ShapeNodeLinkUpdateInput;
    return {
        id: original.id,
        operation: updated.operation !== original.operation ? updated.operation : undefined,
        fromId: updated.fromId !== original.fromId ? updated.fromId : undefined,
        toId: updated.toId !== original.toId ? updated.toId : undefined,
        ...shapeNodeLinkWhensUpdate(original.whens, updated.whens),
    };
}

/**
 * Format an array nodeLinks for create mutation.
 * @param nodeLinks The nodeLinks to format
 * @returns NodeLinks shaped for create mutation
 */
export const shapeNodeLinksCreate = (
    nodeLinks: ShapeNodeLinkCreateInput[] | null | undefined
): RoutineCreateInput['nodeLinksCreate'] | undefined => {
    if (!nodeLinks) return undefined;
    const formatted: NodeLinkCreateInput[] = [];
    for (const nodeLink of nodeLinks) {
        const currShaped = shapeNodeLinkCreate(nodeLink);
        if (currShaped) formatted.push(currShaped);
    }
    return formatted.length > 0 ? formatted : undefined;
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
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { nodeLinksCreate: shapeNodeLinksCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        nodeLinksCreate: findCreatedItems(original, updated, shapeNodeLinksCreate),
        nodeLinksUpdate: findUpdatedItems(original, updated, hasObjectChanged, shapeNodeLinkUpdate) as RoutineUpdateInput['nodeLinksUpdate'],
        nodeLinksDelete: findRemovedItems(original, updated),
    } : {}
}