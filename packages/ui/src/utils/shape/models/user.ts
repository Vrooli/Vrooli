import { UserTranslation, UserTranslationCreateInput, UserTranslationUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, updateTranslationPrims } from "./tools";

export type UserTranslationShape = Pick<UserTranslation, "id" | "language" | "bio"> & {
    __typename?: "UserTranslation";
}

export const shapeUserTranslation: ShapeModel<UserTranslationShape, UserTranslationCreateInput, UserTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "bio"), a),
};
