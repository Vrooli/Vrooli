import { Label, LabelCreateInput, LabelTranslation, LabelTranslationCreateInput, LabelTranslationUpdateInput, LabelUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "utils";

export type LabelTranslationShape = Pick<LabelTranslation, 'id' | 'language' | 'description'>

export type LabelShape = Pick<Label, 'id' | 'label' | 'color'> & {
    organization?: { id: string } | null; // If no organization specified, assumes current user
    translations: LabelTranslationShape[];
    // Connects and disconnects of labels to other objects are handled separately, or in parent shape
}

export const shapeLabelTranslation: ShapeModel<LabelTranslationShape, LabelTranslationCreateInput, LabelTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description'))
}

export const shapeLabel: ShapeModel<LabelShape, LabelCreateInput, LabelUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'label', 'color'),
        ...createRel(d, 'organization', ['Connect'], 'one'),
        ...createRel(d, 'translations', ['Create'], 'many', shapeLabelTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'label', 'color'),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeLabelTranslation),
    })
}