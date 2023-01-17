import { smartContractVersionFields as fullFields, listSmartContractVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const smartContractVersionEndpoint = {
    findOne: toQuery('smartContractVersion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('smartContractVersions', 'SmartContractVersionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('smartContractVersionCreate', 'SmartContractVersionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('smartContractVersionUpdate', 'SmartContractVersionUpdateInput', [fullFields], `...fullFields`)
}