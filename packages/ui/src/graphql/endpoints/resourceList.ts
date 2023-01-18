import { resourceListFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const resourceListEndpoint = {
    findOne: toQuery('resourceList', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('resourceLists', 'ResourceListSearchInput', toSearch(fullFields)),
    create: toMutation('resourceListCreate', 'ResourceListCreateInput', fullFields[1]),
    update: toMutation('resourceListUpdate', 'ResourceListUpdateInput', fullFields[1])
}