import { StandardCreateInput, StandardTranslationCreateInput, StandardTranslationUpdateInput, StandardUpdateInput } from "graphql/generated/globalTypes";
import { ShapeWrapper, Standard, StandardTranslation } from "types";
import { hasObjectChanged, ResourceListShape, shapeResourceListsCreate, shapeResourceListsUpdate, shapeTagsCreate, shapeTagsUpdate, TagShape } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type StandardTranslationShape = Omit<ShapeWrapper<StandardTranslation>, 'language' | 'jsonVariable'> & {
    id: string;
    language: StandardTranslationCreateInput['language'];
    jsonVariable: StandardTranslationCreateInput['jsonVariable'];
}

export const shapeStandardTranslationCreate = (item: StandardTranslationShape): StandardTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    jsonVariable: item.jsonVariable,
})

export const shapeStandardTranslationUpdate = (
    original: StandardTranslationShape,
    updated: StandardTranslationShape
): StandardTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        jsonVariable: u.jsonVariable !== o.jsonVariable ? u.jsonVariable : undefined,
    }))

export const shapeStandardTranslationsCreate = (items: StandardTranslationShape[] | null | undefined): {
    translationsCreate?: StandardTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeStandardTranslationCreate);

export const shapeStandardTranslationsUpdate = (
    o: StandardTranslationShape[] | null | undefined,
    u: StandardTranslationShape[] | null | undefined
): {
    translationsCreate?: StandardTranslationCreateInput[],
    translationsUpdate?: StandardTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeStandardTranslationCreate, shapeStandardTranslationUpdate)

export type StandardShape = Omit<ShapeWrapper<Standard>, 'props' | 'type' | 'name' | 'resourceLists' | 'tags' | 'translations' | 'creator'> & {
    id: string;
    props: StandardCreateInput['props'];
    type: StandardCreateInput['type'];
    name: StandardCreateInput['name'];
    resourceLists?: ResourceListShape[];
    tags?: TagShape[];
    translations: StandardTranslationShape[];
    creator?: {
        __typename: 'User' | 'Organization';
        id: string;
    };
}

export const shapeStandardCreate = (item: StandardShape): StandardCreateInput => ({
    id: item.id,
    default: item.default,
    isInternal: item.isInternal,
    name: item.name,
    props: item.props,
    yup: item.yup,
    type: item.type,
    version: item.version,
    ...shapeStandardTranslationsCreate(item.translations),
    ...shapeResourceListsCreate(item.resourceLists),
    ...shapeTagsCreate(item.tags ?? []),
})

export const shapeStandardUpdate = (
    original: StandardShape,
    updated: StandardShape
): StandardUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        // makingAnonymous: updated.makingAnonymous, TODO
        ...shapeStandardTranslationsUpdate(o.translations, u.translations),
        ...shapeResourceListsUpdate(o.resourceLists, u.resourceLists),
        ...shapeTagsUpdate(o.tags, u.tags),
    }))