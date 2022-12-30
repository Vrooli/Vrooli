import { Comment, CommentCreateInput, CommentFor, CommentTranslation, CommentTranslationCreateInput, CommentTranslationUpdateInput, CommentUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged } from "utils";
import { shapeCreateList, shapeUpdatePrims, shapeUpdate, shapeUpdateList, shapeCreatePrims } from "./shapeTools";

export type CommentTranslationShape = Pick<CommentTranslation, 'id' | 'language' | 'text'>

export type CommentShape = Pick<Comment, 'id'> & {
    commentedOn: { __typename: CommentFor, id: string };
    translations: CommentTranslationShape[];
}

export const shapeCommentTranslationCreate = (item: CommentTranslationShape): CommentTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'text')

export const shapeCommentTranslationUpdate = (o: CommentTranslationShape, u: CommentTranslationShape): CommentTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'text'))

export const shapeCommentCreate = (item: CommentShape, threadId: string | null | undefined): CommentCreateInput => ({
    id: item.id,
    createdFor: item.commentedOn.__typename,
    forConnect: item.commentedOn.id,
    parentConnect: threadId ?? undefined,
    ...shapeCreateList(item, 'translations', shapeCommentTranslationCreate),
})

export const shapeCommentUpdate = (o: CommentShape, u: CommentShape): CommentUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeCommentTranslationCreate, shapeCommentTranslationUpdate, 'id'),
    })