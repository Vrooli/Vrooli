import { Comment, CommentCreateInput, CommentFor, CommentTranslation, CommentTranslationCreateInput, CommentTranslationUpdateInput, CommentUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, createRel, updateRel, updatePrims } from "utils";

export type CommentTranslationShape = Pick<CommentTranslation, 'id' | 'language' | 'text'>

export type CommentShape = Pick<Comment, 'id'> & {
    commentedOn: { type: CommentFor, id: string };
    threadId: string | null;
    translations: CommentTranslationShape[];
}

export const shapeCommentTranslation: ShapeModel<CommentTranslationShape, CommentTranslationCreateInput, CommentTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'text'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'text'))
}

export const shapeComment: ShapeModel<CommentShape, CommentCreateInput, CommentUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'threadId'),
        createdFor: d.commentedOn.type,
        forConnect: d.commentedOn.id,
        ...createRel(d, 'translations', ['Create'], 'many', shapeCommentTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id'),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeCommentTranslation),
    })
}