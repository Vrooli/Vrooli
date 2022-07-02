import { ResourceCreateInput, ResourceListCreateInput, ResourceListUpdateInput, ResourceUpdateInput } from "graphql/generated/globalTypes";
import { Resource } from "types";
import { formatForUpdate, hasObjectChanged } from "./objectTools";

/**
 * Format a resource's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeResourceTranslationsCreate =  (translations: { [x: string]: any }[] | null | undefined): { translationsCreate?: ResourceCreateInput['translationsCreate'] } => {
    if (!translations) return {};
    const formatted: ResourceCreateInput['translationsCreate'] = [];
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
 * Format a resource's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
 export const shapeResourceTranslationsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: ResourceUpdateInput['translationsCreate'],
    translationsUpdate?: ResourceUpdateInput['translationsUpdate'],
    translationsDelete?: ResourceUpdateInput['translationsDelete'],
} => {
    if (!updated) return { };
    if (!original || !Array.isArray(original)) {
        return shapeResourceTranslationsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeResourceTranslationsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        translationsUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? formatForUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as ResourceUpdateInput['translationsUpdate'],
        translationsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
    } : {}
}

type ShapeResourceCreateInput = Partial<Resource> & {
    link: Resource['link'],
    listId: string;
    usedFor: Resource['usedFor'],
}
/**
 * Format a resource for create mutation.
 * @param resource The resource's information
 * @returns Resource shaped for create mutation
 */
export const shapeResourceCreate =  (resource: ShapeResourceCreateInput | null | undefined): ResourceCreateInput | undefined => {
    if (!resource) return undefined;
    return {
        id: resource.id,
        listId: resource.listId,
        index: resource.index,
        link: resource.link,
        usedFor: resource.usedFor as any,
        ...shapeResourceTranslationsCreate(resource.translations),
    }
}

type ShapeResourceUpdateInput = ShapeResourceCreateInput & {
    id: Resource['id'],
};
/**
 * Format a resource for update mutation.
 * @param original Original resource
 * @param updatedResource Updated resource
 * @returns Formatted resource
 */
 export const shapeResourceUpdate = (
    original: ShapeResourceUpdateInput | null | undefined,
    updated: ShapeResourceUpdateInput | null | undefined
): ResourceUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    const orig = original ? original : { id: updated.id } as any;
    return {
        id: orig.id,
        index: updated.index !== orig.index ? updated.index : undefined,
        link: updated.link !== orig.link ? updated.link : undefined,
        listId: updated.listId !== orig.listId ? updated.listId : undefined,
        usedFor: updated.usedFor !== orig.usedFor ? updated.usedFor : undefined,
        ...shapeResourceTranslationsUpdate(orig.translations, updated.translations),
    }
}

/**
 * Format an array resources (not a resource list object) for create mutation.
 * @param resources The resources to format
 * @returns Resources shaped for create mutation
 */
export const shapeResourcesCreate =  (resources: ShapeResourceCreateInput[] | null | undefined): { 
    resourcesCreate?: ResourceListCreateInput['resourcesCreate'] 
} => {
    if (!resources) return {};
    const formatted: ResourceListCreateInput['resourcesCreate'] = [];
    for (const resource of resources) {
        const currShaped = shapeResourceCreate(resource);
        if (currShaped) formatted.push(currShaped);
    }
    return formatted.length > 0 ? {
        resourcesCreate: formatted,
    } : {};
}

/**
 * Format an array of resources for update mutation.
 * @param original Original resource list
 * @param updated Updated resource list
 * @returns Formatted resource list
 */
 export const shapeResourcesUpdate = (
    original: ShapeResourceUpdateInput[] | null | undefined,
    updated: ShapeResourceUpdateInput[] | null | undefined
): {
    resourcesCreate?: ResourceListUpdateInput['resourcesCreate'],
    resourcesUpdate?: ResourceListUpdateInput['resourcesUpdate'],
    resourcesDelete?: ResourceListUpdateInput['resourcesDelete'],
} => {
    if (!updated) return { };
    if (!original || !Array.isArray(original)) {
        return shapeResourcesCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeResourcesCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        resourcesUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? shapeResourceUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as ResourceListUpdateInput['resourcesUpdate'],
        resourcesDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id).filter(id => id) as string[],
    } : {}
}

/**
 * Format a resource list's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeResourceListTranslationsCreate =  (translations: { [x: string]: any }[] | null | undefined): { translationsCreate?: ResourceListCreateInput['translationsCreate'] } => {
    if (!translations) return {};
    const formatted: ResourceListCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
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
 * Format a resource list's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
 export const shapeResourceListTranslationsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: ResourceListUpdateInput['translationsCreate'],
    translationsUpdate?: ResourceListUpdateInput['translationsUpdate'],
    translationsDelete?: ResourceListUpdateInput['translationsDelete'],
} => {
    if (!updated) return { };
    if (!original || !Array.isArray(original)) {
        return shapeResourceListTranslationsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeResourceListTranslationsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        translationsUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? formatForUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as ResourceListUpdateInput['translationsUpdate'],
        translationsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
    } : {}
}

/**
 * Format a resource list for create mutation.
 * @param resourceLists Resource lists to format
 * @returns Formatted resource lists
 */
export const shapeResourceListsCreate =  (resourceLists: { [x: string]: any }[] | null | undefined): { resourceListsCreate?: ResourceListCreateInput[] } => {
    if (!resourceLists) return {};
    const formatted: ResourceListCreateInput[] = [];
    for (const list of resourceLists) {
        formatted.push({
            id: list.id,
            index: list.index,
            usedFor: list.usedFor,
            ...shapeResourceListTranslationsCreate(list.translations),
            ...shapeResourcesCreate(list.resources),
        });
    }
    return formatted.length > 0 ? {
        resourceListsCreate: formatted,
    } : {};
}

/**
 * Format a resource list for update mutation.
 * @param original Original resource lists
 * @param updated Updated resource lists
 * @returns Formatted resource lists
 */
export const shapeResourceListsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined,
): {
    resourceListsCreate?: ResourceListCreateInput[],
    resourceListsUpdate?: ResourceListUpdateInput[],
    resourceListsDelete?: string[],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return shapeResourceListsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeResourceListsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        resourceListsUpdate: updated.map(t => {
            const ol = original.find(o => o.id === t.id);
            return (ol && hasObjectChanged(ol, t)) ? {
                id: ol.id,
                index: t.index !== ol.index ? t.index : undefined,
                usedFor: t.usedFor !== ol.usedFor ? t.usedFor : undefined,
                ...shapeResourceListTranslationsUpdate(ol.translations, t.translations),
                ...shapeResourcesUpdate(ol.resources.map(or => ({
                    ...or,
                    listId: ol.id,
                })), t.resources.map(tr => ({
                    ...tr,
                    listId: t.id,
                }))),
            } : undefined;
        }).filter(t => Boolean(t)) as ResourceListUpdateInput[],
        resourceListsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
    } : {}
}