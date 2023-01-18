import { postFields as fullFields, listPostFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const postEndpoint = {
    findOne: toQuery('post', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('posts', 'PostSearchInput', toSearch(listFields)),
    create: toMutation('postCreate', 'PostCreateInput', fullFields[1]),
    update: toMutation('postUpdate', 'PostUpdateInput', fullFields[1])
}