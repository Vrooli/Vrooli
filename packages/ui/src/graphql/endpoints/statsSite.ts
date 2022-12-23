import { statsSiteFields as listFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsSiteEndpoint = {
    findMany: toQuery('statsSite', 'StatsSiteSearchInput', [listFields], toSearch(listFields)),
}