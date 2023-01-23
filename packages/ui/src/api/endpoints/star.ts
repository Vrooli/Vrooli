import { starPartial, successPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const starEndpoint = {
    stars: toQuery('stars', 'StarSearchInput', ...toSearch(starPartial)),
    star: toMutation('star', 'StarInput',successPartial, 'full'),
}