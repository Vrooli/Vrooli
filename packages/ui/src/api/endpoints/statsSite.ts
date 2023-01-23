import { statsSitePartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsSiteEndpoint = {
    findMany: toQuery('statsSite', 'StatsSiteSearchInput', ...toSearch(statsSitePartial)),
}