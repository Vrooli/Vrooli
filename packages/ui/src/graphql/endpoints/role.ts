import { roleFields as fullFields, listRoleFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const roleEndpoint = {
    findOne: toQuery('role', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('roles', 'RoleSearchInput', toSearch(listFields)),
    create: toMutation('roleCreate', 'RoleCreateInput', fullFields[1]),
    update: toMutation('roleUpdate', 'RoleUpdateInput', fullFields[1])
}