import { apiVersionPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const apiVersionEndpoint = {
    findOne: toQuery('apiVersion', 'FindByIdInput', apiVersionPartial, 'full'),
    findMany: toQuery('apiVersions', 'ApiVersionSearchInput', ...toSearch(apiVersionPartial)),
    create: toMutation('apiVersionCreate', 'ApiVersionCreateInput', apiVersionPartial, 'full'),
    update: toMutation('apiVersionUpdate', 'ApiVersionUpdateInput', apiVersionPartial, 'full')
}