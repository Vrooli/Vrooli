import { organizationPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const organizationEndpoint = {
    findOne: toQuery('organization', 'FindByIdInput', organizationPartial, 'full'),
    findMany: toQuery('organizations', 'OrganizationSearchInput', ...toSearch(organizationPartial)),
    create: toMutation('organizationCreate', 'OrganizationCreateInput', organizationPartial, 'full'),
    update: toMutation('organizationUpdate', 'OrganizationUpdateInput', organizationPartial, 'full')
}