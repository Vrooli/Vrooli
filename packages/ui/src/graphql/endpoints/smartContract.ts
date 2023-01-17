import { smartContractFields as fullFields, listSmartContractFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const smartContractEndpoint = {
    findOne: toQuery('smartContract', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('smartContracts', 'SmartContractSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('smartContractCreate', 'SmartContractCreateInput', [fullFields], `...fullFields`),
    update: toMutation('smartContractUpdate', 'SmartContractUpdateInput', [fullFields], `...fullFields`)
}