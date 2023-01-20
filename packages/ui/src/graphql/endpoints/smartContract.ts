import { smartContractPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const smartContractEndpoint = {
    findOne: toQuery('smartContract', 'FindByIdInput', smartContractPartial, 'full'),
    findMany: toQuery('smartContracts', 'SmartContractSearchInput', ...toSearch(smartContractPartial)),
    create: toMutation('smartContractCreate', 'SmartContractCreateInput', smartContractPartial, 'full'),
    update: toMutation('smartContractUpdate', 'SmartContractUpdateInput', smartContractPartial, 'full')
}