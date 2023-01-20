import { resourceListPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const resourceListEndpoint = {
    findOne: toQuery('resourceList', 'FindByIdInput', resourceListPartial, 'full'),
    findMany: toQuery('resourceLists', 'ResourceListSearchInput', ...toSearch(resourceListPartial)),
    create: toMutation('resourceListCreate', 'ResourceListCreateInput', resourceListPartial 'full'),
    update: toMutation('resourceListUpdate', 'ResourceListUpdateInput', resourceListPartial, 'full')
}