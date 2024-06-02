import { Label, LabelCreateInput, LabelTranslation, LabelTranslationCreateInput, LabelTranslationUpdateInput, LabelUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { TeamShape } from "./team";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type LabelTranslationShape = Pick<LabelTranslation, "id" | "language" | "description"> & {
    __typename?: "LabelTranslation";
}

export type LabelShape = Pick<Label, "id" | "label" | "color"> & {
    __typename: "Label";
    team?: CanConnect<TeamShape> | null; // If no team specified, assumes current user
    translations: LabelTranslationShape[];
    // Connects and disconnects of labels to other objects are handled separately, or in parent shape
}

export const shapeLabelTranslation: ShapeModel<LabelTranslationShape, LabelTranslationCreateInput, LabelTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description"), a),
};

export const shapeLabel: ShapeModel<LabelShape, LabelCreateInput, LabelUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "label", "color"),
        ...createRel(d, "team", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeLabelTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "label", "color"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeLabelTranslation),
    }, a),
};
