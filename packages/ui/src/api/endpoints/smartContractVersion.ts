import { smartContractVersionPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const smartContractVersionEndpoint = {
    findOne: toQuery('smartContractVersion', 'FindByIdInput', smartContractVersionPartial, 'full'),
    findMany: toQuery('smartContractVersions', 'SmartContractVersionSearchInput', ...toSearch(smartContractVersionPartial)),
    create: toMutation('smartContractVersionCreate', 'SmartContractVersionCreateInput', smartContractVersionPartial, 'full'),
    update: toMutation('smartContractVersionUpdate', 'SmartContractVersionUpdateInput', smartContractVersionPartial, 'full')
}