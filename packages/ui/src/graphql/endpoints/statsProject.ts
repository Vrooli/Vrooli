import { statsProjectPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsProjectEndpoint = {
    findMany: toQuery('statsProject', 'StatsProjectSearchInput', ...toSearch(statsProjectPartial)),
}