import { statsUserPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsUserEndpoint = {
    findMany: toQuery('statsUser', 'StatsUserSearchInput', ...toSearch(statsUserPartial)),
}