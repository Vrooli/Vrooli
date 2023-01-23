import { standardVersionPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const standardVersionEndpoint = {
    findOne: toQuery('standardVersion', 'FindByIdInput', standardVersionPartial, 'full'),
    findMany: toQuery('standardVersions', 'StandardVersionSearchInput', ...toSearch(standardVersionPartial)),
    create: toMutation('standardVersionCreate', 'StandardVersionCreateInput', standardVersionPartial, 'full'),
    update: toMutation('standardVersionUpdate', 'StandardVersionUpdateInput', standardVersionPartial, 'full')
}