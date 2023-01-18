import { standardVersionFields as fullFields, listStandardVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const standardVersionEndpoint = {
    findOne: toQuery('standardVersion', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('standardVersions', 'StandardVersionSearchInput', toSearch(listFields)),
    create: toMutation('standardVersionCreate', 'StandardVersionCreateInput', fullFields[1]),
    update: toMutation('standardVersionUpdate', 'StandardVersionUpdateInput', fullFields[1])
}