import { statsSmartContractPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsSmartContractEndpoint = {
    findMany: toQuery('statsSmartContract', 'StatsSmartContractSearchInput', ...toSearch(statsSmartContractPartial)),
}