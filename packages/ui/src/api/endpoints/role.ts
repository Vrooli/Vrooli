import { rolePartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const roleEndpoint = {
    findOne: toQuery('role', 'FindByIdInput', rolePartial, 'full'),
    findMany: toQuery('roles', 'RoleSearchInput', ...toSearch(rolePartial)),
    create: toMutation('roleCreate', 'RoleCreateInput', rolePartial, 'full'),
    update: toMutation('roleUpdate', 'RoleUpdateInput', rolePartial, 'full')
}