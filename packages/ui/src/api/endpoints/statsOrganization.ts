import { statsOrganizationPartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const statsOrganizationEndpoint = {
    findMany: toQuery('statsOrganization', 'StatsOrganizationSearchInput', ...toSearch(statsOrganizationPartial)),
}