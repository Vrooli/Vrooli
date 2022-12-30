import { Label, LabelCreateInput, LabelTranslation, LabelTranslationCreateInput, LabelTranslationUpdateInput, LabelUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged } from "utils";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type LabelTranslationShape = OmitCalculated<LabelTranslation>

export type LabelShape = Pick<Label, 'id' | 'label' | 'color'> & {
    organizationConnect?: string | null; // If no organization specified, assumes current user
    translations: LabelTranslationShape[];
    // Connects and disconnects of labels to other objects are handled separately, or in parent shape
}

export const shapeLabelTranslationCreate = (item: LabelTranslationShape): LabelTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description')

export const shapeLabelTranslationUpdate = (o: LabelTranslationShape, u: LabelTranslationShape): LabelTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description'))

export const shapeLabelCreate = (item: LabelShape): LabelCreateInput => ({
    organizationConnect: item.organizationConnect ?? undefined,
    ...shapeCreatePrims(item, 'id', 'label', 'color'),
    ...shapeCreateList(item, 'translations', shapeLabelTranslationCreate),
})

export const shapeLabelUpdate = (o: LabelShape, u: LabelShape): LabelUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'label', 'color'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeLabelTranslationCreate, shapeLabelTranslationUpdate, 'id'),
    })