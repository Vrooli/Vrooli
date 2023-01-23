import { apiPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const apiEndpoint = {
    findOne: toQuery('api', 'FindByIdInput', apiPartial, 'full'),
    findMany: toQuery('apis', 'ApiSearchInput', ...toSearch(apiPartial)),
    create: toMutation('apiCreate', 'ApiCreateInput', apiPartial, 'full'),
    update: toMutation('apiUpdate', 'ApiUpdateInput', apiPartial, 'full')
}