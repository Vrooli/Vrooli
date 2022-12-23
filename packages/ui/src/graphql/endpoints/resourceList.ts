import { resourceListFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const resourceListEndpoint = {
    findOne: toQuery('resourceList', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('resourceLists', 'ResourceListSearchInput', [fullFields], toSearch(fullFields)),
    create: toMutation('resourceListCreate', 'ResourceListCreateInput', [fullFields], `...fullFields`),
    update: toMutation('resourceListUpdate', 'ResourceListUpdateInput', [fullFields], `...fullFields`)
}