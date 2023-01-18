import { tagFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const tagEndpoint = {
    findOne: toQuery('tag', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('tags', 'TagSearchInput', toSearch(fullFields)),
    create: toMutation('tagCreate', 'TagCreateInput', fullFields[1]),
    update: toMutation('tagUpdate', 'TagUpdateInput', fullFields[1])
}