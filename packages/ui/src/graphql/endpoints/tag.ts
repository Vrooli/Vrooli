import { tagFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const tagEndpoint = {
    findOne: toQuery('tag', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('tags', 'TagSearchInput', [fullFields], toSearch(fullFields)),
    create: toMutation('tagCreate', 'TagCreateInput', [fullFields], `...fullFields`),
    update: toMutation('tagUpdate', 'TagUpdateInput', [fullFields], `...fullFields`)
}