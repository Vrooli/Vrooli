import { standardVersionPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const standardVersionEndpoint = {
    findOne: toQuery('standardVersion', 'FindByIdInput', standardVersionPartial, 'full'),
    findMany: toQuery('standardVersions', 'StandardVersionSearchInput', ...toSearch(standardVersionPartial)),
    create: toMutation('standardVersionCreate', 'StandardVersionCreateInput', standardVersionPartial, 'full'),
    update: toMutation('standardVersionUpdate', 'StandardVersionUpdateInput', standardVersionPartial, 'full')
}