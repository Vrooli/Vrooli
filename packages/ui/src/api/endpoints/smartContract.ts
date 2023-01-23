import { smartContractPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const smartContractEndpoint = {
    findOne: toQuery('smartContract', 'FindByIdInput', smartContractPartial, 'full'),
    findMany: toQuery('smartContracts', 'SmartContractSearchInput', ...toSearch(smartContractPartial)),
    create: toMutation('smartContractCreate', 'SmartContractCreateInput', smartContractPartial, 'full'),
    update: toMutation('smartContractUpdate', 'SmartContractUpdateInput', smartContractPartial, 'full')
}