import { statsSmartContractPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsSmartContractEndpoint = {
    findMany: toQuery('statsSmartContract', 'StatsSmartContractSearchInput', ...toSearch(statsSmartContractPartial)),
}