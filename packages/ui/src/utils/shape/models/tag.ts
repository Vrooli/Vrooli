import { Tag, TagCreateInput, TagTranslation, TagTranslationCreateInput, TagTranslationUpdateInput, TagUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { hasObjectChanged, shapeCreateList, createPrims, shapeUpdate, shapeUpdateList, updatePrims } from "utils";

export type TagTranslationShape = Pick<TagTranslation, 'id' | 'language' | 'description'>

export type TagShape = Pick<Tag, 'tag'> & {
    anonymous?: boolean | null;
    translations?: TagTranslationShape[];
}

export const shapeTagTranslation: ShapeModel<TagTranslationShape, TagTranslationCreateInput, TagTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description'))
}

export const shapeTag: ShapeModel<TagShape, TagCreateInput, TagUpdateInput> = {
    create: (item) => ({
        // anonymous?: boolean | null; TODO
        tag: item.tag,
        ...shapeCreateList(item, 'translations', shapeTagTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        // anonymous: TODO
        tag: o.tag,
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeTagTranslation, 'id'),
    })
}
