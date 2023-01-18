import { statsSmartContractFields as listFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsSmartContractEndpoint = {
    findMany: toQuery('statsSmartContract', 'StatsSmartContractSearchInput', toSearch(listFields)),
}