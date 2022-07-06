import { ResourceCreateInput, ResourceListCreateInput, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput, ResourceListUpdateInput, ResourceListUsedFor, ResourceTranslationCreateInput, ResourceTranslationUpdateInput, ResourceUpdateInput } from "graphql/generated/globalTypes";
import { Resource, ResourceList, ResourceListTranslation, ResourceTranslation } from "types";
import { formatForUpdate, hasObjectChanged } from "./objectTools";
import { shapeCreateList, shapeUpdate, shapeUpdateList, ShapeWrapper } from "./shapeTools";

type ResourceTranslationCreate = ShapeWrapper<ResourceTranslation> &
    Pick<ResourceTranslation, 'language'>;
/**
 * Format a resource's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeResourceTranslationsCreate = (
    translations: ResourceTranslationCreate[] | null | undefined
): ResourceTranslationCreateInput[] | undefined => shapeCreateList(translations, (translation) => ({
    id: translation.id,
    language: translation.language,
    description: translation.description,
    title: translation.title,
}))

interface ResourceTranslationUpdate extends ResourceTranslationCreate { id: string };
/**
 * Format a resource's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeResourceTranslationsUpdate = (
    original: ResourceTranslationUpdate[] | null | undefined,
    updated: ResourceTranslationUpdate[] | null | undefined
): {
    translationsCreate?: ResourceTranslationCreateInput[],
    translationsUpdate?: ResourceTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'translations',
    shapeResourceTranslationsCreate,
    hasObjectChanged,
    formatForUpdate as (original: ResourceTranslationUpdate, updated: ResourceTranslationUpdate) => ResourceTranslationUpdateInput | undefined,
)

type ResourceCreate = ShapeWrapper<Resource> &
    Pick<Resource, 'link' | 'usedFor'> & { listId: string }
/**
 * Format a resource for create mutation.
 * @param resource The resource's information
 * @returns Resource shaped for create mutation
 */
export const shapeResourceCreate = (resource: ResourceCreate | null | undefined): ResourceCreateInput | undefined => {
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

interface ResourceUpdate extends ResourceCreate { id: string };
/**
 * Format a resource for update mutation.
 * @param original Original resource
 * @param updated Updated resource
 * @returns Formatted resource
 */
export const shapeResourceUpdate = (
    original: ResourceUpdate,
    updated: ResourceUpdate | null | undefined
): ResourceUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    index: u.index !== o.index ? u.index : undefined,
    link: u.link !== o.link ? u.link : undefined,
    listId: u.listId !== o.listId ? u.listId : undefined,
    usedFor: u.usedFor !== o.usedFor ? u.usedFor : undefined,
    ...shapeResourceTranslationsUpdate(o.translations, u.translations),
}))

/**
 * Format an array resources (not a resource list object) for create mutation.
 * @param resources The resources to format
 * @returns Resources shaped for create mutation
 */
export const shapeResourcesCreate = (
    resources: ResourceCreate[] | null | undefined
): ResourceCreateInput[] | undefined => shapeCreateList(resources, shapeResourceCreate)

/**
 * Format an array of resources for update mutation.
 * @param original Original resource list
 * @param updated Updated resource list
 * @returns Formatted resource list
 */
export const shapeResourcesUpdate = (
    original: ResourceUpdate[] | null | undefined,
    updated: ResourceUpdate[] | null | undefined
): {
    resourcesCreate?: ResourceCreateInput[],
    resourcesUpdate?: ResourceUpdateInput[],
    resourcesDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'resources',
    shapeResourcesCreate,
    hasObjectChanged,
    shapeResourceUpdate,
)

type ResourceListTranslationCreate = ShapeWrapper<ResourceListTranslation> &
    Pick<ResourceListTranslation, 'language'>;
/**
 * Format a resource list's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeResourceListTranslationsCreate = (
    translations: ResourceListTranslationCreate[] | null | undefined
): ResourceListTranslationCreateInput[] | undefined => shapeCreateList(translations, (translation) => ({
    language: translation.language,
    description: translation.description,
    title: translation.title,
}))

interface ResourceListTranslationUpdate extends ResourceListTranslationCreate { id: string };
/**
 * Format a resource list's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeResourceListTranslationsUpdate = (
    original: ResourceListTranslationUpdate[] | null | undefined,
    updated: ResourceListTranslationUpdate[] | null | undefined
): {
    translationsCreate?: ResourceListTranslationCreateInput[],
    translationsUpdate?: ResourceListTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'translations',
    shapeResourceListTranslationsCreate,
    hasObjectChanged,
    formatForUpdate as (original: ResourceListTranslationUpdate, updated: ResourceListTranslationUpdate) => ResourceTranslationUpdateInput | undefined,
)

type ResourceListCreate = ShapeWrapper<ResourceList> &
    Pick<ResourceList, 'id' | 'usedFor' | 'resources'>;
/**
 * Format a resource list for create mutation.
 * @param resourceLists Resource lists to format
 * @returns Formatted resource lists
 */
export const shapeResourceListsCreate = (
    resourceLists: ResourceListCreate[] | null | undefined
): ResourceListCreateInput[] | undefined => shapeCreateList(resourceLists, (list) => ({
    id: list.id,
    index: list.index,
    usedFor: list.usedFor ?? ResourceListUsedFor.Display,
    ...shapeResourceListTranslationsCreate(list.translations),
    ...shapeResourcesCreate(list.resources.map(r => ({
        ...r,
        listId: list.id,
    }))),
}))

interface ResourceListUpdate extends ResourceListCreate { id: string };
/**
 * Format a resource list for update mutation.
 * @param original Original resource lists
 * @param updated Updated resource lists
 * @returns Formatted resource lists
 */
export const shapeResourceListsUpdate = (
    original: ResourceListUpdate[] | null | undefined,
    updated: ResourceListUpdate[] | null | undefined,
): {
    resourceListsCreate?: ResourceListCreateInput[],
    resourceListsUpdate?: ResourceListUpdateInput[],
    resourceListsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'resourceLists',
    shapeResourceListsCreate,
    hasObjectChanged,
    (o, u) => ({
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
    })
)