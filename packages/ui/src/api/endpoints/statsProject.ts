import { statsProjectPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsProjectEndpoint = {
    findMany: toQuery('statsProject', 'StatsProjectSearchInput', ...toSearch(statsProjectPartial)),
}