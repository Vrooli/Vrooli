import { statsOrganizationPartial } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const statsOrganizationEndpoint = {
    findMany: toQuery('statsOrganization', 'StatsOrganizationSearchInput', ...toSearch(statsOrganizationPartial)),
}