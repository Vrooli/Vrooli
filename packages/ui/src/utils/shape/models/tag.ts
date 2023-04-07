import { Tag, TagCreateInput, TagTranslation, TagTranslationCreateInput, TagTranslationUpdateInput, TagUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type TagTranslationShape = Pick<TagTranslation, 'id' | 'language' | 'description'> & {
    __typename?: 'TagTranslation';
}

export type TagShape = Pick<Tag, 'tag'> & {
    __typename?: 'Tag';
    anonymous?: boolean | null;
    translations?: TagTranslationShape[] | null;
}

export const shapeTagTranslation: ShapeModel<TagTranslationShape, TagTranslationCreateInput, TagTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description'), a)
}

export const shapeTag: ShapeModel<TagShape, TagCreateInput, TagUpdateInput> = {
    idField: 'tag',
    create: (d) => ({
        // anonymous?: boolean | null; TODO
        tag: d.tag,
        ...createRel(d, 'translations', ['Create'], 'many', shapeTagTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        // anonymous: TODO
        tag: o.tag,
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeTagTranslation),
    }, a)
}
