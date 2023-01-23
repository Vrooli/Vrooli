import { statsStandardPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsStandardEndpoint = {
    findMany: toQuery('statsStandard', 'StatsStandardSearchInput', ...toSearch(statsStandardPartial)),
}