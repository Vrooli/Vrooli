import { commentPartial, commentThreadPartial } from 'graphql/partial';
import { toMutation, toQuery } from 'graphql/utils';

export const commentEndpoint = {
    findOne: toQuery('comment', 'FindByIdInput', commentPartial, 'full'),
    findMany: toQuery('comments', 'CommentSearchInput', commentThreadPartial, 'full'),
    create: toMutation('commentCreate', 'CommentCreateInput', commentPartial, 'full'),
    update: toMutation('commentUpdate', 'CommentUpdateInput', commentPartial, 'full')
}