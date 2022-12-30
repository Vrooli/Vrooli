import { Tag, TagCreateInput, TagTranslation, TagTranslationCreateInput, TagTranslationUpdateInput, TagUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged } from "./objectTools";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type TagTranslationShape = OmitCalculated<TagTranslation>

export const shapeTagTranslationCreate = (item: TagTranslationShape): TagTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description')

export const shapeTagTranslationUpdate = (o: TagTranslationShape, u: TagTranslationShape): TagTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description'))

export type TagShape = Omit<OmitCalculated<Tag>, 'tag' | 'translations'> & {
    tag: string;
    translations?: TagTranslationShape[];
}

export const shapeTagCreate = (item: TagShape): TagCreateInput => ({
    // anonymous?: boolean | null; TODO
    tag: item.tag,
    ...shapeCreateList(item, 'translations', shapeTagTranslationCreate),
})

export const shapeTagUpdate = (o: TagShape, u: TagShape): TagUpdateInput | undefined =>
    shapeUpdate(u, {
        // anonymous: TODO
        tag: o.tag,
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeTagTranslationCreate, shapeTagTranslationUpdate, 'id'),
    })