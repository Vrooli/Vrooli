import { postPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const postEndpoint = {
    findOne: toQuery('post', 'FindByIdInput', postPartial, 'full'),
    findMany: toQuery('posts', 'PostSearchInput', ...toSearch(postPartial)),
    create: toMutation('postCreate', 'PostCreateInput', postPartial, 'full'),
    update: toMutation('postUpdate', 'PostUpdateInput', postPartial, 'full')
}