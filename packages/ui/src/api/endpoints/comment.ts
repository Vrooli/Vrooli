import { commentPartial, commentThreadPartial } from 'api/partial';
import { toMutation, toQuery } from 'api/utils';

export const commentEndpoint = {
    findOne: toQuery('comment', 'FindByIdInput', commentPartial, 'full'),
    findMany: toQuery('comments', 'CommentSearchInput', commentThreadPartial, 'full'),
    create: toMutation('commentCreate', 'CommentCreateInput', commentPartial, 'full'),
    update: toMutation('commentUpdate', 'CommentUpdateInput', commentPartial, 'full')
}