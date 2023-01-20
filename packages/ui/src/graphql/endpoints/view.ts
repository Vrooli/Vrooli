import { viewPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const viewEndpoint = {
    views: toQuery('views', 'ViewSearchInput', ...toSearch(viewPartial)),
}