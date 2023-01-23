import { resourceListPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const resourceListEndpoint = {
    findOne: toQuery('resourceList', 'FindByIdInput', resourceListPartial, 'full'),
    findMany: toQuery('resourceLists', 'ResourceListSearchInput', ...toSearch(resourceListPartial)),
    create: toMutation('resourceListCreate', 'ResourceListCreateInput', resourceListPartial, 'full'),
    update: toMutation('resourceListUpdate', 'ResourceListUpdateInput', resourceListPartial, 'full')
}