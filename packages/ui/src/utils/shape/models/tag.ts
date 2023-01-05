import { Tag, TagCreateInput, TagTranslation, TagTranslationCreateInput, TagTranslationUpdateInput, TagUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, updatePrims, createRel, updateRel } from "utils";

export type TagTranslationShape = Pick<TagTranslation, 'id' | 'language' | 'description'>

export type TagShape = Pick<Tag, 'tag'> & {
    anonymous?: boolean | null;
    translations?: TagTranslationShape[];
}

export const shapeTagTranslation: ShapeModel<TagTranslationShape, TagTranslationCreateInput, TagTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description'))
}

export const shapeTag: ShapeModel<TagShape, TagCreateInput, TagUpdateInput> = {
    idField: 'tag',
    create: (d) => ({
        // anonymous?: boolean | null; TODO
        tag: d.tag,
        ...createRel(d, 'translations', ['Create'], 'many', shapeTagTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        // anonymous: TODO
        tag: o.tag,
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeTagTranslation),
    })
}
