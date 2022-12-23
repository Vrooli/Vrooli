import { postFields as fullFields, listPostFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const postEndpoint = {
    findOne: toQuery('post', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('posts', 'PostSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('postCreate', 'PostCreateInput', [fullFields], `...fullFields`),
    update: toMutation('postUpdate', 'PostUpdateInput', [fullFields], `...fullFields`)
}