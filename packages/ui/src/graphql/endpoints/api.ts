import { apiFields as fullFields, listApiFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const apiEndpoint = {
    findOne: toQuery('api', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('apis', 'ApiSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('apiCreate', 'ApiCreateInput', [fullFields], `...fullFields`),
    update: toMutation('apiUpdate', 'ApiUpdateInput', [fullFields], `...fullFields`)
}