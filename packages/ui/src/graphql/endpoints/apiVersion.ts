import { apiVersionFields as fullFields, listApiVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const apiVersionEndpoint = {
    findOne: toQuery('apiVersion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('apiVersions', 'ApiVersionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('apiVersionCreate', 'ApiVersionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('apiVersionUpdate', 'ApiVersionUpdateInput', [fullFields], `...fullFields`)
}