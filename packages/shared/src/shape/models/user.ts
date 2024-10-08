import { UserTranslation, UserTranslationCreateInput, UserTranslationUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { createPrims, shapeUpdate, updateTranslationPrims } from "./tools";

export type UserTranslationShape = Pick<UserTranslation, "id" | "language" | "bio"> & {
    __typename?: "UserTranslation";
}

export const shapeUserTranslation: ShapeModel<UserTranslationShape, UserTranslationCreateInput, UserTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "bio"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "bio")),
};
