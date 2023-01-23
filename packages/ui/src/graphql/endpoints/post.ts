import { postPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const postEndpoint = {
    findOne: toQuery('post', 'FindByIdInput', postPartial, 'full'),
    findMany: toQuery('posts', 'PostSearchInput', ...toSearch(postPartial)),
    create: toMutation('postCreate', 'PostCreateInput', postPartial, 'full'),
    update: toMutation('postUpdate', 'PostUpdateInput', postPartial, 'full')
}