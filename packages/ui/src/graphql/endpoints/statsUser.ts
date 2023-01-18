import { statsUserFields as listFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsUserEndpoint = {
    findMany: toQuery('statsUser', 'StatsUserSearchInput', toSearch(listFields)),
}