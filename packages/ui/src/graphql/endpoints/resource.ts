import { resourceFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const resourceEndpoint = {
    findOne: toQuery('resource', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('resources', 'ResourceSearchInput', [fullFields], toSearch(fullFields)),
    create: toMutation('resourceCreate', 'ResourceCreateInput', [fullFields], `...fullFields`),
    update: toMutation('resourceUpdate', 'ResourceUpdateInput', [fullFields], `...fullFields`)
}