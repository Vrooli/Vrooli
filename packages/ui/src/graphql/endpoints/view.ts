import { listViewFields as listFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const viewEndpoint = {
    views: toQuery('views', 'ViewSearchInput', toSearch(listFields)),
}