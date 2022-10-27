import { CommentCreateInput, CommentTranslationCreateInput, CommentTranslationUpdateInput, CommentFor, CommentUpdateInput } from "graphql/generated/globalTypes";
import { Comment, CommentTranslation, ShapeWrapper } from "types";
import { hasObjectChanged } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type CommentTranslationShape = Omit<ShapeWrapper<CommentTranslation>, 'language' | 'description'> & {
    id: string;
    language: CommentTranslationCreateInput['language'];
    text: CommentTranslationCreateInput['text'];
}

export type CommentShape = Omit<ShapeWrapper<Comment>, 'commentedOn' | 'creator' | 'translations'> & {
    id: string;
    commentedOn: { __typename: CommentFor, id: string };
    translations: CommentTranslationShape[];
}

export const shapeCommentTranslationCreate = (item: CommentTranslationShape): CommentTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    text: item.text,
})

export const shapeCommentTranslationUpdate = (
    original: CommentTranslationShape,
    updated: CommentTranslationShape
): CommentTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        text: u.text !== o.text ? u.text : undefined,
    }), 'id')

export const shapeCommentCreate = (item: CommentShape, threadId: string | null | undefined): CommentCreateInput => {
    return {
        id: item.id,
        createdFor: item.commentedOn.__typename as CommentFor,
        forId: item.commentedOn.id,
        parentId: threadId ?? undefined,
        ...shapeCreateList(item, 'translations', shapeCommentTranslationCreate),
    }
}

export const shapeCommentUpdate = (
    original: CommentShape,
    updated: CommentShape
): CommentUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => {
        return {
            id: o.id,
            ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeCommentTranslationCreate, shapeCommentTranslationUpdate, 'id'),
        }
    }, 'id')