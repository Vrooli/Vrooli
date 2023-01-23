import { resourcePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const resourceEndpoint = {
    findOne: toQuery('resource', 'FindByIdInput', resourcePartial, 'full'),
    findMany: toQuery('resources', 'ResourceSearchInput', ...toSearch(resourcePartial)),
    create: toMutation('resourceCreate', 'ResourceCreateInput', resourcePartial, 'full'),
    update: toMutation('resourceUpdate', 'ResourceUpdateInput', resourcePartial, 'full')
}