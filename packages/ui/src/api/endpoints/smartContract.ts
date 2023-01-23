import { smartContractPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const smartContractEndpoint = {
    findOne: toQuery('smartContract', 'FindByIdInput', smartContractPartial, 'full'),
    findMany: toQuery('smartContracts', 'SmartContractSearchInput', ...toSearch(smartContractPartial)),
    create: toMutation('smartContractCreate', 'SmartContractCreateInput', smartContractPartial, 'full'),
    update: toMutation('smartContractUpdate', 'SmartContractUpdateInput', smartContractPartial, 'full')
}