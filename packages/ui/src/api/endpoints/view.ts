import { viewPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const viewEndpoint = {
    views: toQuery('views', 'ViewSearchInput', ...toSearch(viewPartial)),
}