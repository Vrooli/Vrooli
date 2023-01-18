import { statsStandardFields as listFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsStandardEndpoint = {
    findMany: toQuery('statsStandard', 'StatsStandardSearchInput', toSearch(listFields)),
}