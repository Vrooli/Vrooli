import { commentFields, commentThreadFields } from 'graphql/partial';
import { toMutation, toQuery } from 'graphql/utils';

export const commentEndpoint = {
    findOne: toQuery('comment', 'FindByIdInput', commentFields[1]),
    findMany: toQuery('comments', 'CommentSearchInput', commentThreadFields[1]),
    create: toMutation('commentCreate', 'CommentCreateInput', commentFields[1]),
    update: toMutation('commentUpdate', 'CommentUpdateInput', commentFields[1])
}