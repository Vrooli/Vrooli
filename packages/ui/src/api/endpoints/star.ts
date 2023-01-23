import { starPartial, successPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const starEndpoint = {
    stars: toQuery('stars', 'StarSearchInput', ...toSearch(starPartial)),
    star: toMutation('star', 'StarInput',successPartial, 'full'),
}