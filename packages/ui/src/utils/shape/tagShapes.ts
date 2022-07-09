import { TagCreateInput, TagTranslationCreateInput, TagTranslationUpdateInput, TagUpdateInput } from "graphql/generated/globalTypes";
import { ShapeWrapper, Tag, TagTranslation } from "types";
import { hasObjectChanged } from "./objectTools";
import { findCreatedItems, findRemovedItems, findUpdatedItems, shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

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
    tag: string;
    translations?: TagTranslationShape[];
}

export const shapeTagCreate = (item: TagShape): TagCreateInput => ({
    // anonymous?: boolean | null; TODO
    tag: item.tag,
    ...shapeTagTranslationsCreate(item.translations),
})

export const shapeTagUpdate = (
    original: TagShape,
    updated: TagShape
): TagUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    let result: TagUpdateInput = {
        // anonymous: TODO
        tag: original.tag,
        ...shapeTagTranslationsUpdate(original.translations, updated.translations),
    }
    // Remove every value from the result that is undefined
    if (result) result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined)) as TagUpdateInput;
    // Return result if it is not empty
    return result && Object.keys(result).length > 0 ? result : undefined;
}

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
} => {
    if (!u) return {};
    // If no original items, treat all as created
    if (!o || !Array.isArray(o)) {
        return shapeCreateList(u ?? [], 'tags', shapeTagCreate);
    }
    if (Array.isArray(u) && u.length > 0) {
        return {
            tagsCreate: findCreatedItems(o, u, shapeTagCreate),
            tagsUpdate: findUpdatedItems(o, u, hasObjectChanged, shapeTagUpdate),
            tagsDelete: findRemovedItems(o, u),
        } as any;
    }
    return {};
}