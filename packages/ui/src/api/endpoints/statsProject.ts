import { statsProjectPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsProjectEndpoint = {
    findMany: toQuery('statsProject', 'StatsProjectSearchInput', ...toSearch(statsProjectPartial)),
}