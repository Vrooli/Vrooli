import { statsApiPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsApiEndpoint = {
    findMany: toQuery('statsApi', 'StatsApiSearchInput', ...toSearch(statsApiPartial)),
}