import { listStarFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const starEndpoint = {
    stars: toQuery('stars', 'StarSearchInput', [listFields], toSearch(listFields)),
    star: toMutation('star', 'StarInput', [], `success`),
}