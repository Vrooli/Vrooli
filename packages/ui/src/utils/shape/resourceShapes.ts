import { ResourceCreateInput, ResourceListCreateInput, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput, ResourceListUpdateInput, ResourceListUsedFor, ResourceTranslationCreateInput, ResourceTranslationUpdateInput, ResourceUpdateInput } from "graphql/generated/globalTypes";
import { Resource, ResourceList, ResourceListTranslation, ResourceTranslation, ShapeWrapper } from "types";
import { hasObjectChanged } from "./objectTools";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type ResourceTranslationShape = Omit<ShapeWrapper<ResourceTranslation>, 'language'> & {
    id: string;
    language: ResourceTranslationCreateInput['language'];
}

export const shapeResourceTranslationCreate = (item: ResourceTranslationShape): ResourceTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    title: item.title,
})

export const shapeResourceTranslationUpdate = (
    original: ResourceTranslationShape,
    updated: ResourceTranslationShape
): ResourceTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        title: u.title !== o.title ? u.title : undefined,
    }))

export const shapeResourceTranslationsCreate = (items: ResourceTranslationShape[] | null | undefined): {
    translationsCreate?: ResourceTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeResourceTranslationCreate);

export const shapeResourceTranslationsUpdate = (
    o: ResourceTranslationShape[] | null | undefined,
    u: ResourceTranslationShape[] | null | undefined
): {
    translationsCreate?: ResourceTranslationCreateInput[],
    translationsUpdate?: ResourceTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeResourceTranslationCreate, shapeResourceTranslationUpdate)

export type ResourceShape = Omit<ShapeWrapper<Resource>, 'translations' | 'link' | 'usedFor'> & {
    id: string;
    link: ResourceCreateInput['link'];
    listId: string;
    usedFor: ResourceCreateInput['usedFor'] | null;
    translations: ResourceTranslationShape[];
}

export const shapeResourceCreate = (item: ResourceShape): ResourceCreateInput => ({
    id: item.id,
    listId: item.listId,
    index: item.index,
    link: item.link,
    usedFor: item.usedFor as any,
    ...shapeResourceTranslationsCreate(item.translations),
})

export const shapeResourceUpdate = (
    original: ResourceShape,
    updated: ResourceShape
): ResourceUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        index: u.index !== o.index ? u.index : undefined,
        link: u.link !== o.link ? u.link : undefined,
        listId: u.listId !== o.listId ? u.listId : undefined,
        usedFor: u.usedFor !== o.usedFor ? u.usedFor : undefined,
        ...shapeResourceTranslationsUpdate(o.translations, u.translations),
    }))

export const shapeResourcesCreate = (items: ResourceShape[] | null | undefined): {
    resourcesCreate?: ResourceCreateInput[],
} => shapeCreateList(items, 'resources', shapeResourceCreate);

export const shapeResourcesUpdate = (
    o: ResourceShape[] | null | undefined,
    u: ResourceShape[] | null | undefined
): {
    resourcesCreate?: ResourceCreateInput[],
    resourcesUpdate?: ResourceUpdateInput[],
    resourcesDelete?: string[],
} => shapeUpdateList(o, u, 'resources', hasObjectChanged, shapeResourceCreate, shapeResourceUpdate)

export type ResourceListTranslationShape = Omit<ShapeWrapper<ResourceListTranslation>, 'language'> & {
    id: string;
    language: ResourceListTranslationCreateInput['language'];
}

export const shapeResourceListTranslationCreate = (item: ResourceListTranslationShape): ResourceListTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    title: item.title,
})

export const shapeResourceListTranslationUpdate = (
    original: ResourceListTranslationShape,
    updated: ResourceListTranslationShape
): ResourceListTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        title: u.title !== o.title ? u.title : undefined,
    }))

export const shapeResourceListTranslationsCreate = (items: ResourceListTranslationShape[] | null | undefined): {
    translationsCreate?: ResourceListTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeResourceListTranslationCreate);

export const shapeResourceListTranslationsUpdate = (
    o: ResourceListTranslationShape[] | null | undefined,
    u: ResourceListTranslationShape[] | null | undefined
): {
    translationsCreate?: ResourceListTranslationCreateInput[],
    translationsUpdate?: ResourceListTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeResourceListTranslationCreate, shapeResourceListTranslationUpdate)

export type ResourceListShape = Omit<ShapeWrapper<ResourceList>, 'usedFor' | 'translations' | 'resources'> & {
    id: string;
    usedFor: ResourceListCreateInput['usedFor'] | null;
    resources: Omit<ResourceShape, 'listId'>[];
    translations: ResourceListTranslationShape[];
}

export const shapeResourceListCreate = (item: ResourceListShape): ResourceListCreateInput => ({
    id: item.id,
    index: item.index,
    usedFor: item.usedFor ?? ResourceListUsedFor.Display,
    ...shapeResourceListTranslationsCreate(item.translations),
    ...shapeResourcesCreate(item.resources.map(r => ({
        ...r,
        listId: item.id,
    }))),
})

export const shapeResourceListUpdate = (
    original: ResourceListShape,
    updated: ResourceListShape
): ResourceListUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
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
    }))

export const shapeResourceListsCreate = (items: ResourceListShape[] | null | undefined): {
    resourceListsCreate?: ResourceListCreateInput[],
} => shapeCreateList(items, 'resourceLists', shapeResourceListCreate);

export const shapeResourceListsUpdate = (
    o: ResourceListShape[] | null | undefined,
    u: ResourceListShape[] | null | undefined
): {
    resourceListsCreate?: ResourceListCreateInput[],
    resourceListsUpdate?: ResourceListUpdateInput[],
    resourceListsDelete?: string[],
} => shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate)