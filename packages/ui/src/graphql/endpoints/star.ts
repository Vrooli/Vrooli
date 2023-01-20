import { starPartial, successPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const starEndpoint = {
    stars: toQuery('stars', 'StarSearchInput', ...toSearch(starPartial)),
    star: toMutation('star', 'StarInput',successPartial, 'full'),
}