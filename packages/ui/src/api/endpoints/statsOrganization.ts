import { statsOrganizationPartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const statsOrganizationEndpoint = {
    findMany: toQuery('statsOrganization', 'StatsOrganizationSearchInput', ...toSearch(statsOrganizationPartial)),
}