import { statsApiPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsApiEndpoint = {
    findMany: toQuery('statsApi', 'StatsApiSearchInput', ...toSearch(statsApiPartial)),
}