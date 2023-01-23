import { statsSitePartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsSiteEndpoint = {
    findMany: toQuery('statsSite', 'StatsSiteSearchInput', ...toSearch(statsSitePartial)),
}