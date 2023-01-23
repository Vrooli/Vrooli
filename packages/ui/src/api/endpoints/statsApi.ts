import { statsApiPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsApiEndpoint = {
    findMany: toQuery('statsApi', 'StatsApiSearchInput', ...toSearch(statsApiPartial)),
}