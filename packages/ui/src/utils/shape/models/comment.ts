import { Comment, CommentCreateInput, CommentedOn, CommentFor, CommentTranslation, CommentTranslationCreateInput, CommentTranslationUpdateInput, CommentUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type CommentTranslationShape = Pick<CommentTranslation, "id" | "language" | "text"> & {
    __typename?: "CommentTranslation";
}

export type CommentShape = Pick<Comment, "id"> & {
    __typename: "Comment";
    commentedOn: { __typename: CommentedOn["__typename"], id: string };
    threadId?: string | null;
    translations: CommentTranslationShape[];
}

export const shapeCommentTranslation: ShapeModel<CommentTranslationShape, CommentTranslationCreateInput, CommentTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "text"), a),
};

export const shapeComment: ShapeModel<CommentShape, CommentCreateInput, CommentUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "threadId"),
        createdFor: d.commentedOn.__typename as CommentFor,
        forConnect: d.commentedOn.id,
        ...createRel(d, "translations", ["Create"], "many", shapeCommentTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeCommentTranslation),
    }, a),
};
