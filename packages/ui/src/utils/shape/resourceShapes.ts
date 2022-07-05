import { ResourceCreateInput, ResourceListCreateInput, ResourceListUpdateInput, ResourceListUsedFor, ResourceUpdateInput } from "graphql/generated/globalTypes";
import { Resource, ResourceList } from "types";
import { formatForUpdate, hasObjectChanged } from "./objectTools";
import { findCreatedItems, findRemovedItems, findUpdatedItems } from "./shapeTools";

type ResourceTranslationCreate = Partial<Resource['translations'][0]> & {
    id: string;
    language: Resource['translations'][0]['language'];
}
/**
 * Format a resource's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeResourceTranslationsCreate = (
    translations: ResourceTranslationCreate[] | null | undefined
): ResourceCreateInput['translationsCreate'] | undefined => {
    if (!translations) return undefined;
    const formatted: ResourceCreateInput['translationsCreate'] = [];
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
 * Format a resource's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeResourceTranslationsUpdate = (
    original: ResourceTranslationCreate[] | null | undefined,
    updated: ResourceTranslationCreate[] | null | undefined
): {
    translationsCreate?: ResourceUpdateInput['translationsCreate'],
    translationsUpdate?: ResourceUpdateInput['translationsUpdate'],
    translationsDelete?: ResourceUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { translationsCreate: shapeResourceTranslationsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        translationsCreate: findCreatedItems(original, updated, shapeResourceTranslationsCreate),
        translationsUpdate: findUpdatedItems(original, updated, hasObjectChanged, formatForUpdate) as ResourceUpdateInput['translationsUpdate'],
        translationsDelete: findRemovedItems(original, updated),
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
export const shapeResourceCreate = (resource: ShapeResourceCreateInput | null | undefined): ResourceCreateInput | undefined => {
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
 * @param updated Updated resource
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
export const shapeResourcesCreate = (
    resources: ShapeResourceCreateInput[] | null | undefined
): ResourceListCreateInput['resourcesCreate'] | undefined => {
    if (!resources) return undefined;
    const formatted: ResourceListCreateInput['resourcesCreate'] = [];
    for (const resource of resources) {
        const currShaped = shapeResourceCreate(resource);
        if (currShaped) formatted.push(currShaped);
    }
    return formatted.length > 0 ? formatted : undefined;
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
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { resourcesCreate: shapeResourcesCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        resourcesCreate: findCreatedItems(original, updated, shapeResourcesCreate),
        resourcesUpdate: findUpdatedItems(original, updated, hasObjectChanged, shapeResourceUpdate) as ResourceListUpdateInput['resourcesUpdate'],
        resourcesDelete: findRemovedItems(original, updated),
    } : {}
}

type ResourceListTranslationCreate = Partial<ResourceList['translations'][0]> & {
    id: string;
    language: ResourceList['translations'][0]['language'];
}
/**
 * Format a resource list's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeResourceListTranslationsCreate = (
    translations: ResourceListTranslationCreate[] | null | undefined
): ResourceListCreateInput['translationsCreate'] | undefined => {
    if (!translations) return undefined;
    const formatted: ResourceListCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
            language: translation.language,
            description: translation.description,
            title: translation.title,
        });
    }
    return formatted.length > 0 ? formatted : undefined;
}

/**
 * Format a resource list's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeResourceListTranslationsUpdate = (
    original: ResourceListTranslationCreate[] | null | undefined,
    updated: ResourceListTranslationCreate[] | null | undefined
): {
    translationsCreate?: ResourceListUpdateInput['translationsCreate'],
    translationsUpdate?: ResourceListUpdateInput['translationsUpdate'],
    translationsDelete?: ResourceListUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { translationsCreate: shapeResourceListTranslationsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        translationsCreate: findCreatedItems(original, updated, shapeResourceListTranslationsCreate),
        translationsUpdate: findUpdatedItems(original, updated, hasObjectChanged, formatForUpdate) as ResourceListUpdateInput['translationsUpdate'],
        translationsDelete: findRemovedItems(original, updated),
    } : {}
}

type ResourceListCreate = Partial<ResourceList> & {
    id: string;
    usedFor: ResourceList['usedFor'];
    resources: ResourceList['resources'];
}
/**
 * Format a resource list for create mutation.
 * @param resourceLists Resource lists to format
 * @returns Formatted resource lists
 */
export const shapeResourceListsCreate = (
    resourceLists: ResourceListCreate[] | null | undefined
): ResourceListCreateInput[] | undefined => {
    if (!resourceLists) return undefined;
    const formatted: ResourceListCreateInput[] = [];
    for (const list of resourceLists) {
        formatted.push({
            id: list.id,
            index: list.index,
            usedFor: list.usedFor ?? ResourceListUsedFor.Display,
            ...shapeResourceListTranslationsCreate(list.translations),
            ...shapeResourcesCreate(list.resources.map(r => ({
                ...r,
                listId: list.id,
            }))),
        });
    }
    return formatted.length > 0 ? formatted : undefined;
}

/**
 * Format a resource list for update mutation.
 * @param original Original resource lists
 * @param updated Updated resource lists
 * @returns Formatted resource lists
 */
export const shapeResourceListsUpdate = (
    original: ResourceListCreate[] | null | undefined,
    updated: ResourceListCreate[] | null | undefined,
): {
    resourceListsCreate?: ResourceListCreateInput[],
    resourceListsUpdate?: ResourceListUpdateInput[],
    resourceListsDelete?: string[],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return { resourceListsCreate: shapeResourceListsCreate(updated) };
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        resourceListsCreate: findCreatedItems(original, updated, shapeResourceListsCreate),
        resourceListsUpdate: findUpdatedItems(original, updated, hasObjectChanged, (o, u) => ({
            id: o.id,
            index: u.index !== o.index ? u.index : undefined,
            usedFor: u.usedFor !== o.usedFor ? u.usedFor : undefined,
            ...shapeResourceListTranslationsUpdate(o.translations, u.translations),
            ...shapeResourcesUpdate(o.resources.map(or => ({
                ...or,
                listId: o.id,
            })), u.resources.map(ur => ({
                ...ur,
                listId: u.id,
            }))),
        })) as ResourceListUpdateInput[],
        resourceListsDelete: findRemovedItems(original, updated),
    } : {}
}