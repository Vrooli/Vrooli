import { Tag, TagCreateInput, TagTranslation, TagTranslationCreateInput, TagTranslationUpdateInput, TagUpdateInput } from "../../api/generated/graphqlTypes";
import { ShapeModel } from "../../consts/commonTypes";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type TagTranslationShape = Pick<TagTranslation, "id" | "language" | "description"> & {
    __typename?: "TagTranslation";
}

export type TagShape = Pick<Tag, "id" | "tag"> & {
    __typename: "Tag";
    anonymous?: boolean | null;
    translations?: TagTranslationShape[] | null;
    you?: Tag["you"] | null;
}

export const shapeTagTranslation: ShapeModel<TagTranslationShape, TagTranslationCreateInput, TagTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description")),
};

export const shapeTag: ShapeModel<TagShape, TagCreateInput, TagUpdateInput> = {
    idField: "tag",
    create: (d) => ({
        ...createPrims(d, "id", "tag"),
        ...createRel(d, "translations", ["Create"], "many", shapeTagTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "tag"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeTagTranslation),
    }),
};
