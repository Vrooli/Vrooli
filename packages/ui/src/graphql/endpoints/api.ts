import { apiFields as fullFields, listApiFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const apiEndpoint = {
    findOne: toQuery('api', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('apis', 'ApiSearchInput', toSearch(listFields)),
    create: toMutation('apiCreate', 'ApiCreateInput', fullFields[1]),
    update: toMutation('apiUpdate', 'ApiUpdateInput', fullFields[1])
}