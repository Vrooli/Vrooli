import { organizationFields as fullFields, listOrganizationFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const organizationEndpoint = {
    findOne: toQuery('organization', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('organizations', 'OrganizationSearchInput', toSearch(listFields)),
    create: toMutation('organizationCreate', 'OrganizationCreateInput', fullFields[1]),
    update: toMutation('organizationUpdate', 'OrganizationUpdateInput', fullFields[1])
}