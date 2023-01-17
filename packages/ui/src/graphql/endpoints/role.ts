import { roleFields as fullFields, listRoleFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const roleEndpoint = {
    findOne: toQuery('role', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('roles', 'RoleSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('roleCreate', 'RoleCreateInput', [fullFields], `...fullFields`),
    update: toMutation('roleUpdate', 'RoleUpdateInput', [fullFields], `...fullFields`)
}