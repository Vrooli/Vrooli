import { ResourceCreateInput, ResourceListCreateInput, ResourceListTranslationCreateInput, ResourceListTranslationUpdateInput, ResourceListUpdateInput, ResourceListUsedFor, ResourceTranslationCreateInput, ResourceTranslationUpdateInput, ResourceUpdateInput } from "graphql/generated/globalTypes";
import { Resource, ResourceList, ResourceListTranslation, ResourceTranslation, ShapeWrapper } from "types";
import { hasObjectChanged } from "./objectTools";
import { shapeCreateList, shapePrim, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type ResourceTranslationShape = Omit<ShapeWrapper<ResourceTranslation>, 'language'> & {
    id: string;
    language: ResourceTranslationCreateInput['language'];
}

export type ResourceShape = Omit<ShapeWrapper<Resource>, 'translations' | 'link' | 'usedFor'> & {
    id: string;
    link: ResourceCreateInput['link'];
    listId: string;
    usedFor: ResourceCreateInput['usedFor'] | null;
    translations: ResourceTranslationShape[];
}

export type ResourceListTranslationShape = Omit<ShapeWrapper<ResourceListTranslation>, 'language'> & {
    id: string;
    language: ResourceListTranslationCreateInput['language'];
}

export type ResourceListShape = Omit<ShapeWrapper<ResourceList>, 'usedFor' | 'translations' | 'resources'> & {
    id: string;
    usedFor: ResourceListCreateInput['usedFor'] | null;
    resources: Omit<ResourceShape, 'listId'>[] | null;
    translations: ResourceListTranslationShape[] | null;
}

export const shapeResourceTranslationCreate = (item: ResourceTranslationShape): ResourceTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    name: item.name,
})

export const shapeResourceTranslationUpdate = (
    original: ResourceTranslationShape,
    updated: ResourceTranslationShape
): ResourceTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        ...shapePrim(o, u, 'description'),
        ...shapePrim(o, u, 'name'),
    }), 'id')

export const shapeResourceCreate = (item: ResourceShape): ResourceCreateInput => ({
    id: item.id,
    listId: item.listId,
    index: item.index,
    link: item.link,
    usedFor: item.usedFor as any,
    ...shapeCreateList(item, 'translations', shapeResourceTranslationCreate),
})

export const shapeResourceUpdate = (
    original: ResourceShape,
    updated: ResourceShape
): ResourceUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        ...shapePrim(o, u, 'index'),
        ...shapePrim(o, u, 'link'),
        ...shapePrim(o, u, 'listId'),
        ...shapePrim(o, u, 'usedFor'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeResourceTranslationCreate, shapeResourceTranslationUpdate, 'id'),
    }), 'id')

export const shapeResourceListTranslationCreate = (item: ResourceListTranslationShape): ResourceListTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    name: item.name,
})

export const shapeResourceListTranslationUpdate = (
    original: ResourceListTranslationShape,
    updated: ResourceListTranslationShape
): ResourceListTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        ...shapePrim(o, u, 'description'),
        ...shapePrim(o, u, 'name'),
    }), 'id')

export const shapeResourceListCreate = (item: ResourceListShape): ResourceListCreateInput => ({
    id: item.id,
    index: item.index,
    usedFor: item.usedFor ?? ResourceListUsedFor.Display,
    ...shapeCreateList(item, 'translations', shapeResourceListTranslationCreate),
    ...shapeCreateList({
        resources: item.resources?.map(r => ({
            ...r,
            listId: item.id,
        }))
    }, 'resources', shapeResourceCreate),
})

export const shapeResourceListUpdate = (
    original: ResourceListShape,
    updated: ResourceListShape
): ResourceListUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        ...shapePrim(o, u, 'index'),
        ...shapePrim(o, u, 'usedFor'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeResourceListTranslationCreate, shapeResourceListTranslationUpdate, 'id'),
        ...shapeUpdateList({
            resources: o.resources?.map(r => ({
                ...r,
                listId: o.id,
            }))
        }, {
            resources: u.resources?.map(r => ({
                ...r,
                listId: u.id,
            }))
        }, 'resources', hasObjectChanged, shapeResourceCreate, shapeResourceUpdate, 'id'),
    }), 'id')