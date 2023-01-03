import { Label, LabelCreateInput, LabelTranslation, LabelTranslationCreateInput, LabelTranslationUpdateInput, LabelUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, hasObjectChanged, shapeUpdate, updatePrims } from "utils";

export type LabelTranslationShape = Pick<LabelTranslation, 'id' | 'language' | 'description'>

export type LabelShape = Pick<Label, 'id' | 'label' | 'color'> & {
    organizationConnect?: string | null; // If no organization specified, assumes current user
    translations: LabelTranslationShape[];
    // Connects and disconnects of labels to other objects are handled separately, or in parent shape
}

export const shapeLabelTranslation: ShapeModel<LabelTranslationShape, LabelTranslationCreateInput, LabelTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description'))
}

export const shapeLabel: ShapeModel<LabelShape, LabelCreateInput, LabelUpdateInput> = {
    create: (item) => ({
        organizationConnect: item.organizationConnect ?? undefined,
        ...createPrims(item, 'id', 'label', 'color'),
        ...shapeCreateList(item, 'translations', shapeLabelTranslationCreate),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'label', 'color'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeLabelTranslationCreate, shapeLabelTranslationUpdate, 'id'),
    })
}