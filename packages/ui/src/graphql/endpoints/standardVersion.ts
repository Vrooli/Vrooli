import { standardVersionFields as fullFields, listStandardVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const standardVersionEndpoint = {
    findOne: toQuery('standardVersion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('standardVersions', 'StandardVersionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('standardVersionCreate', 'StandardVersionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('standardVersionUpdate', 'StandardVersionUpdateInput', [fullFields], `...fullFields`)
}