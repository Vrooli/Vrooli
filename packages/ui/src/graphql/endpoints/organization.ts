import { organizationPartial } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const organizationEndpoint = {
    findOne: toQuery('organization', 'FindByIdInput', organizationPartial, 'full'),
    findMany: toQuery('organizations', 'OrganizationSearchInput', ...toSearch(organizationPartial)),
    create: toMutation('organizationCreate', 'OrganizationCreateInput', organizationPartial, 'full'),
    update: toMutation('organizationUpdate', 'OrganizationUpdateInput', organizationPartial, 'full')
}