import { Comment, CommentCreateInput, CommentFor, CommentTranslation, CommentTranslationCreateInput, CommentTranslationUpdateInput, CommentUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeCreateList, createPrims, shapeUpdate, shapeUpdateList, shapeUpdatePrims } from "utils";

export type CommentTranslationShape = Pick<CommentTranslation, 'id' | 'language' | 'text'>

export type CommentShape = Pick<Comment, 'id'> & {
    commentedOn: { __typename: CommentFor, id: string };
    translations: CommentTranslationShape[];
}

export const shapeCommentTranslation: ShapeModel<CommentTranslationShape, CommentTranslationCreateInput, CommentTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'text'),
    update: (o, u) => shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'text'))
}

export const shapeComment: ShapeModel<CommentShape, CommentCreateInput, CommentUpdateInput> = {
    create: (item) => ({
        id: item.id,
        createdFor: item.commentedOn.__typename,
        forConnect: item.commentedOn.id,
        parentConnect: threadId ?? undefined,
        ...shapeCreateList(item, 'translations', shapeCommentTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id'),
        ...shapeUpdateList(o, u, 'translations', shapeCommentTranslation, 'id'),
    })
}