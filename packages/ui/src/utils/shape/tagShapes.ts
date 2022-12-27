import { ShapeWrapper } from "types";
import { hasObjectChanged } from "./objectTools";
import { shapeCreateList, shapePrim, shapeUpdate, shapeUpdateList } from "./shapeTools";

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
        ...shapePrim(o, u, 'description'),
    }), 'id')

export type TagShape = Omit<ShapeWrapper<Tag>, 'tag' | 'translations'> & {
    tag: string;
    translations?: TagTranslationShape[];
}

export const shapeTagCreate = (item: TagShape): TagCreateInput => ({
    // anonymous?: boolean | null; TODO
    tag: item.tag,
    ...shapeCreateList(item, 'translations', shapeTagTranslationCreate),
})

export const shapeTagUpdate = (
    original: TagShape,
    updated: TagShape
): TagUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        // anonymous: TODO
        tag: o.tag,
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeTagTranslationCreate, shapeTagTranslationUpdate, 'id'),
    }), 'tag')