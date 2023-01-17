import { commentFields, commentThreadFields } from 'graphql/partial';
import { toMutation, toQuery } from 'graphql/utils';

export const commentEndpoint = {
    findOne: toQuery('comment', 'FindByIdInput', [commentFields], `...commentFields`),
    findMany: toQuery('comments', 'CommentSearchInput', [commentThreadFields], `...commentThreadFields`),
    create: toMutation('commentCreate', 'CommentCreateInput', [commentFields], `...commentFields`),
    update: toMutation('commentUpdate', 'CommentUpdateInput', [commentFields], `...commentFields`)
}