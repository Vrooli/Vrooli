import { statsUserPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsUserEndpoint = {
    findMany: toQuery('statsUser', 'StatsUserSearchInput', ...toSearch(statsUserPartial)),
}