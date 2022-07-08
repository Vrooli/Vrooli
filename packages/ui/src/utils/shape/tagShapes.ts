import { TagCreateInput, TagTranslationCreateInput, TagTranslationUpdateInput, TagUpdateInput } from "graphql/generated/globalTypes";
import { ShapeWrapper, Tag, TagTranslation } from "types";
import { hasObjectChanged } from "./objectTools";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type TagTranslationShape = Omit<ShapeWrapper<TagTranslation>, 'language'> & {
    id: string;
    language: TagTranslationCreateInput['language'];
}

export const shapeTagTranslationCreate = (item: TagTranslationShape): TagTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
})

export const shapeTagTranslationUpdate = (
    original: TagTranslationShape,
    updated: TagTranslationShape
): TagTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
    }))

export const shapeTagTranslationsCreate = (items: TagTranslationShape[] | null | undefined): {
    translationsCreate?: TagTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeTagTranslationCreate);

export const shapeTagTranslationsUpdate = (
    o: TagTranslationShape[] | null | undefined,
    u: TagTranslationShape[] | null | undefined
): {
    translationsCreate?: TagTranslationCreateInput[],
    translationsUpdate?: TagTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeTagTranslationCreate, shapeTagTranslationUpdate)

export type TagShape = Omit<ShapeWrapper<Tag>, 'tag' | 'translations'> & {
    id: string;
    tag: string;
    translations: TagTranslationShape[];
}

export const shapeTagCreate = (item: TagShape): TagCreateInput => ({
    id: item.id,
    // anonymous?: boolean | null; TODO
    tag: item.tag,
    ...shapeTagTranslationsCreate(item.translations),
})

export const shapeTagUpdate = (
    original: TagShape,
    updated: TagShape
): TagUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        // anonymous: TODO
        // tag: u.tag !== o.tag ? u.tag : undefined, probably shouldn't be able to update tag
        ...shapeTagTranslationsUpdate(o.translations, u.translations),
    }))

export const shapeTagsCreate = (items: TagShape[] | null | undefined): {
    tagsCreate?: TagCreateInput[],
} => shapeCreateList(items, 'tags', shapeTagCreate);

export const shapeTagsUpdate = (
    o: TagShape[] | null | undefined,
    u: TagShape[] | null | undefined
): {
    tagsCreate?: TagCreateInput[],
    tagsUpdate?: TagUpdateInput[],
    tagsDelete?: string[],
} => shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate)