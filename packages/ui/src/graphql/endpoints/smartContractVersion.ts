import { smartContractVersionPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const smartContractVersionEndpoint = {
    findOne: toQuery('smartContractVersion', 'FindByIdInput', smartContractVersionPartial, 'full'),
    findMany: toQuery('smartContractVersions', 'SmartContractVersionSearchInput', ...toSearch(smartContractVersionPartial)),
    create: toMutation('smartContractVersionCreate', 'SmartContractVersionCreateInput', smartContractVersionPartial, 'full'),
    update: toMutation('smartContractVersionUpdate', 'SmartContractVersionUpdateInput', smartContractVersionPartial, 'full')
}