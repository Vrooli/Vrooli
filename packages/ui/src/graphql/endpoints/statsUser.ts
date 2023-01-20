import { statsUserPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsUserEndpoint = {
    findMany: toQuery('statsUser', 'StatsUserSearchInput', ...toSearch(statsUserPartial)),
}