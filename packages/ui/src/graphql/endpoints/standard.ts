import { standardFields as fullFields, listStandardFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const standardEndpoint = {
    findOne: toQuery('standard', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('standards', 'StandardSearchInput', toSearch(listFields)),
    create: toMutation('standardCreate', 'StandardCreateInput', fullFields[1]),
    update: toMutation('standardUpdate', 'StandardUpdateInput', fullFields[1])
}