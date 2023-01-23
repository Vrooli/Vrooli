import { viewPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const viewEndpoint = {
    views: toQuery('views', 'ViewSearchInput', ...toSearch(viewPartial)),
}