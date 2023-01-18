import { resourceFields as fullFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const resourceEndpoint = {
    findOne: toQuery('resource', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('resources', 'ResourceSearchInput', toSearch(fullFields)),
    create: toMutation('resourceCreate', 'ResourceCreateInput', fullFields[1]),
    update: toMutation('resourceUpdate', 'ResourceUpdateInput', fullFields[1])
}