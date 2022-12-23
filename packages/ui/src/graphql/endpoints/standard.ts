import { standardFields as fullFields, listStandardFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const standardEndpoint = {
    findOne: toQuery('standard', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('standards', 'StandardSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('standardCreate', 'StandardCreateInput', [fullFields], `...fullFields`),
    update: toMutation('standardUpdate', 'StandardUpdateInput', [fullFields], `...fullFields`)
}