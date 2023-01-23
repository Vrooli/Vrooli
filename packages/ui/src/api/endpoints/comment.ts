import { commentPartial, commentThreadPartial } from '../partial';
import { toMutation, toQuery } from '../utils';

export const commentEndpoint = {
    findOne: toQuery('comment', 'FindByIdInput', commentPartial, 'full'),
    findMany: toQuery('comments', 'CommentSearchInput', commentThreadPartial, 'full'),
    create: toMutation('commentCreate', 'CommentCreateInput', commentPartial, 'full'),
    update: toMutation('commentUpdate', 'CommentUpdateInput', commentPartial, 'full')
}