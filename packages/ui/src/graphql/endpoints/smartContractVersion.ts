import { smartContractVersionFields as fullFields, listSmartContractVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const smartContractVersionEndpoint = {
    findOne: toQuery('smartContractVersion', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('smartContractVersions', 'SmartContractVersionSearchInput', toSearch(listFields)),
    create: toMutation('smartContractVersionCreate', 'SmartContractVersionCreateInput', fullFields[1]),
    update: toMutation('smartContractVersionUpdate', 'SmartContractVersionUpdateInput', fullFields[1])
}