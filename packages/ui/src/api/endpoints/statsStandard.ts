import { statsStandardPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsStandardEndpoint = {
    findMany: toQuery('statsStandard', 'StatsStandardSearchInput', ...toSearch(statsStandardPartial)),
}