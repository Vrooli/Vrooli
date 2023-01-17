import { organizationFields as fullFields, listOrganizationFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const organizationEndpoint = {
    findOne: toQuery('organization', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('organizations', 'OrganizationSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('organizationCreate', 'OrganizationCreateInput', [fullFields], `...fullFields`),
    update: toMutation('organizationUpdate', 'OrganizationUpdateInput', [fullFields], `...fullFields`)
}