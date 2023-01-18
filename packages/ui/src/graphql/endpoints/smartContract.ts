import { smartContractFields as fullFields, listSmartContractFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const smartContractEndpoint = {
    findOne: toQuery('smartContract', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('smartContracts', 'SmartContractSearchInput', toSearch(listFields)),
    create: toMutation('smartContractCreate', 'SmartContractCreateInput', fullFields[1]),
    update: toMutation('smartContractUpdate', 'SmartContractUpdateInput', fullFields[1])
}