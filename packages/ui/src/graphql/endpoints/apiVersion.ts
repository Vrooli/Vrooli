import { apiVersionFields as fullFields, listApiVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const apiVersionEndpoint = {
    findOne: toQuery('apiVersion', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('apiVersions', 'ApiVersionSearchInput', toSearch(listFields)),
    create: toMutation('apiVersionCreate', 'ApiVersionCreateInput', fullFields[1]),
    update: toMutation('apiVersionUpdate', 'ApiVersionUpdateInput', fullFields[1])
}