import { profilePartial, successPartial, userPartial } from 'api/partial';
import { toMutation, toQuery, toSearch } from 'api/utils';

export const userEndpoint = {
    profile: toQuery('profile', null, profilePartial, 'full'),
    findOne: toQuery('user', 'FindByIdInput', userPartial, 'full'),
    findMany: toQuery('users', 'UserSearchInput', ...toSearch(userPartial)),
    profileUpdate: toMutation('profileUpdate', 'ProfileUpdateInput', profilePartial, 'full'),
    profileEmailUpdate: toMutation('profileEmailUpdate', 'ProfileEmailUpdateInput', profilePartial, 'full'),
    deleteOne: toMutation('userDeleteOne', 'UserDeleteInput', successPartial, 'full'),
    exportData: toMutation('exportData', null),
}